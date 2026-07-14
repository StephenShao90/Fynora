package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/advisor"
	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/authz"
	"github.com/StephenShao90/Fynora/services/api/internal/config"
	"github.com/StephenShao90/Fynora/services/api/internal/db"
	"github.com/StephenShao90/Fynora/services/api/internal/httpapi"
	"github.com/StephenShao90/Fynora/services/api/internal/idempotency"
	"github.com/StephenShao90/Fynora/services/api/internal/logger"
	"github.com/StephenShao90/Fynora/services/api/internal/marketdata"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
	"github.com/StephenShao90/Fynora/services/api/internal/observability"
	"github.com/StephenShao90/Fynora/services/api/internal/plaid"
	"github.com/StephenShao90/Fynora/services/api/internal/portfolio"
	"github.com/StephenShao90/Fynora/services/api/internal/ratelimit"
	"github.com/StephenShao90/Fynora/services/api/internal/redisstore"
	"github.com/StephenShao90/Fynora/services/api/internal/repository"
	"github.com/StephenShao90/Fynora/services/api/internal/storage"
	"github.com/StephenShao90/Fynora/services/api/internal/validation"
)

type app struct {
	cfg              config.Config
	log              logger.Logger
	store            *memoryStore
	raw              storage.RawEventStore
	market           marketdata.Provider
	plaid            plaid.Client
	cfRepo           *repository.ClearflowRepository
	rateLimiter      ratelimit.Limiter
	idempotencyLocks idempotency.LockStore
	tracer           observability.Tracer
}

type memoryStore struct {
	mu                       sync.RWMutex
	users                    map[string]models.User
	usersByEmail             map[string]string
	profiles                 map[string]models.AdvisorProfile
	imports                  map[string]models.RawImport
	importErrors             map[string]models.ImportError
	transactions             map[string]models.Transaction
	accounts                 map[string]models.BrokerageAccount
	holdings                 map[string]models.Holding
	portfolioTransactions    map[string]models.PortfolioTransaction
	plaidConnections         map[string]models.PlaidConnection
	plaidItemOrganizations   map[string]string
	organizations            map[string]models.Organization
	organizationMembers      map[string]models.OrganizationMember
	customers                map[string]models.Customer
	invoices                 map[string]models.Invoice
	payments                 map[string]models.Payment
	refunds                  map[string]models.Refund
	fees                     map[string]models.Fee
	payouts                  map[string]models.Payout
	bankTransactions         map[string]models.BankTransaction
	reconciliationRuns       map[string]models.ReconciliationRun
	reconciliationMatches    map[string]models.ReconciliationMatch
	reconciliationExceptions map[string]models.ReconciliationException
	exceptionNotes           map[string]models.ExceptionNote
	organizationSetup        map[string]models.OrganizationSetup
	auditLogs                map[string]models.AuditLog
	refreshSessions          map[string]models.RefreshSession
	refreshTokensByHash      map[string]string
	jobs                     map[string]models.Job
	idempotencyRecords       map[string]models.IdempotencyRecord
	rateLimits               map[string]rateLimitBucket
	outboxEvents             map[string]models.OutboxEvent
	plaidWebhookEvents       map[string]models.WebhookEvent
	processorWebhookEvents   map[string]models.WebhookEvent
	providerConnections      map[string]models.ProviderConnection
	oauthStates              map[string]models.OAuthState
	metrics                  opsMetrics
}

func newStore() *memoryStore {
	return &memoryStore{
		users:                    map[string]models.User{},
		usersByEmail:             map[string]string{},
		profiles:                 map[string]models.AdvisorProfile{},
		imports:                  map[string]models.RawImport{},
		importErrors:             map[string]models.ImportError{},
		transactions:             map[string]models.Transaction{},
		accounts:                 map[string]models.BrokerageAccount{},
		holdings:                 map[string]models.Holding{},
		portfolioTransactions:    map[string]models.PortfolioTransaction{},
		plaidConnections:         map[string]models.PlaidConnection{},
		plaidItemOrganizations:   map[string]string{},
		organizations:            map[string]models.Organization{},
		organizationMembers:      map[string]models.OrganizationMember{},
		customers:                map[string]models.Customer{},
		invoices:                 map[string]models.Invoice{},
		payments:                 map[string]models.Payment{},
		refunds:                  map[string]models.Refund{},
		fees:                     map[string]models.Fee{},
		payouts:                  map[string]models.Payout{},
		bankTransactions:         map[string]models.BankTransaction{},
		reconciliationRuns:       map[string]models.ReconciliationRun{},
		reconciliationMatches:    map[string]models.ReconciliationMatch{},
		reconciliationExceptions: map[string]models.ReconciliationException{},
		exceptionNotes:           map[string]models.ExceptionNote{},
		organizationSetup:        map[string]models.OrganizationSetup{},
		auditLogs:                map[string]models.AuditLog{},
		refreshSessions:          map[string]models.RefreshSession{},
		refreshTokensByHash:      map[string]string{},
		jobs:                     map[string]models.Job{},
		idempotencyRecords:       map[string]models.IdempotencyRecord{},
		rateLimits:               map[string]rateLimitBucket{},
		outboxEvents:             map[string]models.OutboxEvent{},
		plaidWebhookEvents:       map[string]models.WebhookEvent{},
		processorWebhookEvents:   map[string]models.WebhookEvent{},
		providerConnections:      map[string]models.ProviderConnection{},
		oauthStates:              map[string]models.OAuthState{},
	}
}

func main() {
	cfg := config.Load()
	if err := cfg.ValidateProduction(); err != nil {
		log.Fatal(err)
	}
	tracer, err := observability.NewWithConfig(context.Background(), observability.Config{
		Enabled:     observability.Enabled(cfg.OTELEnabled),
		ServiceName: cfg.OTELServiceName,
		Environment: cfg.OTELEnvironment,
		Endpoint:    cfg.OTELExporterOTLPEndpoint,
		Protocol:    cfg.OTELExporterOTLPProtocol,
		Headers:     cfg.OTELExporterOTLPHeaders,
		SampleRatio: observability.SampleRatio(cfg.OTELSampleRatio),
	})
	if err != nil {
		if cfg.AppEnv == "production" {
			log.Fatalf("otel exporter setup failed: %v", err)
		}
		log.Printf("otel exporter disabled after setup failure: %v", err)
		tracer = observability.New(false, cfg.OTELServiceName, cfg.OTELEnvironment)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracer.Shutdown(ctx); err != nil {
			log.Printf("otel shutdown failed: %v", err)
		}
	}()
	a := &app{
		cfg:              cfg,
		log:              logger.New(),
		store:            newStore(),
		raw:              storage.NewLocalStore(cfg.LocalStorageDir),
		market:           marketdata.MockProvider{},
		plaid:            plaid.Client{ClientID: cfg.PlaidClientID, Secret: cfg.PlaidSecret, Env: cfg.PlaidEnv},
		rateLimiter:      ratelimit.NewMemoryLimiter(),
		idempotencyLocks: idempotency.NewMemoryLockStore(),
		tracer:           tracer,
	}
	if redisEnabled(cfg.RedisEnabled) {
		client, err := redisstore.New(cfg.RedisURL, redisEnabled(cfg.RedisTLS))
		if err == nil {
			err = client.Ping(context.Background())
		}
		if err != nil {
			if cfg.AppEnv == "production" {
				log.Fatalf("redis is enabled but unavailable: %v", err)
			}
			a.log.Error("redis.connect_failed", map[string]interface{}{"error": err.Error(), "fallback": "memory"})
		} else {
			a.rateLimiter = ratelimit.NewRedisLimiter(client)
			a.idempotencyLocks = idempotency.NewRedisLockStore(client)
			a.log.Info("redis.connected", map[string]interface{}{"rate_limit": true, "idempotency_locks": true})
		}
	}
	if conn, err := db.Open(context.Background(), cfg.DatabaseURL); err != nil {
		if cfg.AppEnv == "production" {
			log.Fatalf("postgres is required in production: %v", err)
		}
		a.log.Error("database.connect_failed", map[string]interface{}{"error": err.Error(), "fallback": "clearflow_in_memory"})
	} else {
		a.cfRepo = repository.NewClearflow(conn)
		a.log.Info("database.connected", map[string]interface{}{"driver": "pgx", "clearflow_storage": "postgres"})
	}
	if err := a.loadPlaidConnections(); err != nil {
		a.log.Error("load_plaid_connections_failed", map[string]interface{}{"error": err.Error()})
	}
	mux := http.NewServeMux()
	a.routes(mux)
	handler := a.recover(a.requestLog(a.tracer.Middleware(a.securityHeaders(a.bodyLimit(a.withCORS(mux))), func() {
		a.incrementMetric(func(m *opsMetrics) { m.OTELTracesStartedTotal++ })
	})))
	log.Printf("Clearflow API listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatal(err)
	}
}

func (a *app) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("GET /ready", a.ready)
	mux.HandleFunc("GET /api/v1/health", a.health)
	mux.HandleFunc("GET /api/v1/ready", a.ready)
	mux.HandleFunc("POST /auth/register", a.register)
	mux.HandleFunc("POST /auth/login", a.login)
	mux.HandleFunc("POST /auth/demo-token", a.demoToken)
	mux.HandleFunc("POST /api/v1/auth/register", a.authRateLimited(a.register))
	mux.HandleFunc("POST /api/v1/auth/login", a.authRateLimited(a.login))
	mux.HandleFunc("POST /api/v1/auth/demo-token", a.demoToken)
	mux.HandleFunc("POST /api/v1/auth/refresh", a.authRateLimited(a.refreshToken))
	mux.HandleFunc("POST /api/v1/auth/logout", a.authRateLimited(a.logout))
	mux.HandleFunc("GET /api/v1/auth/sessions", a.authed(a.listSessions))
	mux.HandleFunc("DELETE /api/v1/auth/sessions/{sessionId}", a.authed(a.revokeSession))
	mux.HandleFunc("GET /me", a.authed(a.me))
	mux.HandleFunc("GET /api/v1/me", a.authed(a.meV1))
	mux.HandleFunc("GET /me/advisor-profile", a.authed(a.getProfile))
	mux.HandleFunc("PUT /me/advisor-profile", a.authed(a.putProfile))
	mux.HandleFunc("POST /imports/transactions-csv", a.authed(a.importTransactionsCSV))
	mux.HandleFunc("GET /imports", a.authed(a.listImports))
	mux.HandleFunc("GET /imports/{id}", a.authed(a.getImport))
	mux.HandleFunc("POST /transactions", a.authed(a.createTransaction))
	mux.HandleFunc("GET /transactions", a.authed(a.listTransactions))
	mux.HandleFunc("GET /transactions/{id}", a.authed(a.getTransaction))
	mux.HandleFunc("PATCH /transactions/{id}/category", a.authed(a.patchTransactionCategory))
	mux.HandleFunc("DELETE /transactions/{id}", a.authed(a.deleteTransaction))
	mux.HandleFunc("GET /insights/monthly-summary", a.authed(a.monthlySummary))
	mux.HandleFunc("GET /insights/categories", a.authed(a.categories))
	mux.HandleFunc("GET /insights/merchants", a.authed(a.merchants))
	mux.HandleFunc("GET /insights/subscriptions", a.authed(a.subscriptions))
	mux.HandleFunc("GET /insights/anomalies", a.authed(a.anomalies))
	mux.HandleFunc("GET /insights/duplicate-charges", a.authed(a.duplicates))
	mux.HandleFunc("GET /insights/cash-flow", a.authed(a.cashFlow))
	mux.HandleFunc("GET /advisor/plan", a.authed(a.advisorPlan))
	mux.HandleFunc("GET /advisor/emergency-fund", a.authed(a.emergencyFund))
	mux.HandleFunc("GET /advisor/account-priority", a.authed(a.accountPriority))
	mux.HandleFunc("POST /advisor/investment-projection", a.authed(a.investmentProjection))
	mux.HandleFunc("POST /advisor/chat", a.authed(a.chat))
	mux.HandleFunc("POST /advisor/monthly-summary", a.authed(a.monthlyAdvisorSummary))
	mux.HandleFunc("POST /portfolio/accounts", a.authed(a.createAccount))
	mux.HandleFunc("GET /portfolio/accounts", a.authed(a.listAccounts))
	mux.HandleFunc("GET /portfolio/accounts/{id}", a.authed(a.getAccount))
	mux.HandleFunc("DELETE /portfolio/accounts/{id}", a.authed(a.deleteAccount))
	mux.HandleFunc("GET /connections", a.authed(a.listConnections))
	mux.HandleFunc("DELETE /connections/{id}", a.authed(a.deleteConnection))
	mux.HandleFunc("POST /connections/plaid/link-token", a.authed(a.heavyRateLimited(a.createPlaidLinkToken)))
	mux.HandleFunc("POST /connections/plaid/sandbox-connect", a.authed(a.heavyRateLimited(a.createPlaidSandboxConnection)))
	mux.HandleFunc("POST /connections/plaid/exchange-public-token", a.authed(a.heavyRateLimited(a.exchangePlaidPublicToken)))
	mux.HandleFunc("POST /connections/plaid/sync-transactions", a.authed(a.heavyRateLimited(a.syncPlaidTransactions)))
	mux.HandleFunc("POST /connections/plaid/sync-investments", a.authed(a.heavyRateLimited(a.syncPlaidInvestments)))
	mux.HandleFunc("POST /organizations", a.authed(a.createOrganization))
	mux.HandleFunc("GET /organizations", a.authed(a.listOrganizations))
	mux.HandleFunc("POST /api/v1/organizations", a.authed(a.createOrganizationV1))
	mux.HandleFunc("GET /api/v1/organizations", a.authed(a.listOrganizationsV1))
	mux.HandleFunc("GET /api/v1/organizations/{organizationId}/members", a.authed(a.listOrganizationMembersV1))
	mux.HandleFunc("POST /api/v1/organizations/{organizationId}/members", a.authed(a.addOrganizationMemberV1))
	mux.HandleFunc("PATCH /api/v1/organizations/{organizationId}/members/{userId}", a.authed(a.updateOrganizationMemberV1))
	mux.HandleFunc("DELETE /api/v1/organizations/{organizationId}/members/{userId}", a.authed(a.deleteOrganizationMemberV1))
	mux.HandleFunc("GET /payments", a.authed(a.listPayments))
	mux.HandleFunc("GET /api/v1/payments", a.authed(a.listPaymentsV1))
	mux.HandleFunc("GET /payouts", a.authed(a.listPayouts))
	mux.HandleFunc("GET /api/v1/payouts", a.authed(a.listPayoutsV1))
	mux.HandleFunc("GET /api/v1/payouts/{payoutId}/explanation", a.authed(a.payoutExplanationV1))
	mux.HandleFunc("GET /api/v1/payouts/{id}/breakdown", a.authed(a.payoutExplanationV1))
	mux.HandleFunc("GET /payouts/{id}/breakdown", a.authed(a.getPayoutBreakdown))
	mux.HandleFunc("GET /bank-transactions", a.authed(a.listBankTransactions))
	mux.HandleFunc("GET /api/v1/bank-transactions", a.authed(a.listBankTransactionsV1))
	mux.HandleFunc("POST /sync/stripe", a.authed(a.syncStripeMock))
	mux.HandleFunc("POST /api/v1/sync/stripe", a.authed(a.heavyRateLimited(a.syncStripeMockV1)))
	mux.HandleFunc("POST /sync/bank", a.authed(a.syncBankMock))
	mux.HandleFunc("POST /api/v1/sync/bank", a.authed(a.heavyRateLimited(a.syncBankMockV1)))
	mux.HandleFunc("POST /reconciliation/runs", a.authed(a.createReconciliationRun))
	mux.HandleFunc("POST /api/v1/reconciliation-runs", a.authed(a.heavyRateLimited(a.createReconciliationRunV1)))
	mux.HandleFunc("GET /reconciliation/runs", a.authed(a.listReconciliationRuns))
	mux.HandleFunc("GET /api/v1/reconciliation-runs", a.authed(a.listReconciliationRunsV1))
	mux.HandleFunc("GET /reconciliation/runs/{id}", a.authed(a.getReconciliationRun))
	mux.HandleFunc("GET /reconciliation/exceptions", a.authed(a.listReconciliationExceptions))
	mux.HandleFunc("PATCH /reconciliation/exceptions/{id}", a.authed(a.patchReconciliationException))
	mux.HandleFunc("GET /reconciliation/exceptions/{id}/notes", a.authed(a.listExceptionNotes))
	mux.HandleFunc("POST /reconciliation/exceptions/{id}/notes", a.authed(a.addExceptionNote))
	mux.HandleFunc("GET /cash-flow/summary", a.authed(a.clearflowCashSummary))
	mux.HandleFunc("GET /api/v1/cash-flow/summary", a.authed(a.clearflowCashSummaryV1))
	mux.HandleFunc("GET /cash-flow/forecast", a.authed(a.clearflowCashForecast))
	mux.HandleFunc("GET /api/v1/cash-flow/forecast", a.authed(a.clearflowCashForecastV1))
	mux.HandleFunc("GET /api/v1/cashflow/forecast", a.authed(a.cashflowForecastV1))
	mux.HandleFunc("GET /api/v1/insights/anomalies", a.authed(a.anomaliesV1))
	mux.HandleFunc("GET /api/v1/insights/spending", a.authed(a.spendingInsightsV1))
	mux.HandleFunc("GET /api/v1/recommendations/cash", a.authed(a.cashRecommendationsV1))
	mux.HandleFunc("GET /api/v1/reconciliation-runs/{runId}/matches", a.authed(a.reconciliationMatchesV1))
	mux.HandleFunc("GET /api/v1/integrations/stripe/connect-url", a.authed(a.stripeConnectURLV1))
	mux.HandleFunc("GET /api/v1/integrations/stripe/callback", a.stripeCallbackV1)
	mux.HandleFunc("GET /api/v1/integrations/stripe/status", a.authed(a.stripeStatusV1))
	mux.HandleFunc("DELETE /api/v1/integrations/stripe", a.authed(a.stripeDisconnectV1))
	mux.HandleFunc("GET /api/v1/jobs", a.authed(a.listJobsV1))
	mux.HandleFunc("GET /api/v1/jobs/{jobId}", a.authed(a.getJobV1))
	mux.HandleFunc("POST /api/v1/jobs/{jobId}/cancel", a.authed(a.cancelJobV1))
	mux.HandleFunc("POST /api/v1/jobs/{jobId}/retry", a.authed(a.retryJobV1))
	mux.HandleFunc("GET /api/v1/jobs/dead", a.authed(a.listDeadJobsV1))
	mux.HandleFunc("GET /api/v1/audit-logs", a.authed(a.listAuditLogsV1))
	mux.HandleFunc("POST /api/v1/webhooks/plaid", a.webhookRateLimited(a.handlePlaidWebhook))
	mux.HandleFunc("POST /api/v1/webhooks/processors/{provider}", a.webhookRateLimited(a.handleProcessorWebhook))
	mux.HandleFunc("GET /api/v1/ops/metrics", a.authed(a.opsMetricsV1))
	mux.HandleFunc("GET /api/v1/onboarding/status", a.authed(a.onboardingStatusV1))
	mux.HandleFunc("PUT /api/v1/onboarding/status", a.authed(a.updateOnboardingStatusV1))
	mux.HandleFunc("GET /reports/monthly", a.authed(a.clearflowMonthlyReport))
	mux.HandleFunc("GET /debug/clearflow", a.authed(a.debugClearflowState))
	mux.HandleFunc("POST /debug/clearflow/reset-demo", a.authed(a.resetClearflowDemo))
	mux.HandleFunc("POST /portfolio/import/holdings-csv", a.authed(a.importHoldingsCSV))
	mux.HandleFunc("POST /portfolio/import/transactions-csv", a.authed(a.importPortfolioTransactionsCSV))
	mux.HandleFunc("GET /portfolio/imports", a.authed(a.listPortfolioImports))
	mux.HandleFunc("GET /portfolio/imports/{id}/errors", a.authed(a.listPortfolioImportErrors))
	mux.HandleFunc("POST /portfolio/holdings", a.authed(a.createHolding))
	mux.HandleFunc("GET /portfolio/holdings", a.authed(a.listHoldings))
	mux.HandleFunc("GET /portfolio/holdings/{id}", a.authed(a.getHolding))
	mux.HandleFunc("PATCH /portfolio/holdings/{id}", a.authed(a.patchHolding))
	mux.HandleFunc("DELETE /portfolio/holdings/{id}", a.authed(a.deleteHolding))
	mux.HandleFunc("GET /portfolio/transactions", a.authed(a.listPortfolioTransactions))
	mux.HandleFunc("GET /portfolio/summary", a.authed(a.portfolioSummary))
	mux.HandleFunc("GET /portfolio/allocation", a.authed(a.portfolioAllocation))
	mux.HandleFunc("GET /portfolio/performance", a.authed(a.portfolioPerformance))
	mux.HandleFunc("GET /portfolio/risk", a.authed(a.portfolioRisk))
	mux.HandleFunc("GET /portfolio/concentration", a.authed(a.portfolioRisk))
	mux.HandleFunc("GET /portfolio/rebalance-suggestions", a.authed(a.rebalance))
	mux.HandleFunc("GET /portfolio/projected-growth", a.authed(a.projectedGrowth))
	mux.HandleFunc("GET /market/quote/{symbol}", a.authed(a.quote))
	mux.HandleFunc("POST /market/quotes", a.authed(a.quotes))
}

func redisEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "enabled":
		return true
	default:
		return false
	}
}

func (a *app) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "clearflow-api"})
}

func (a *app) ready(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "degraded", "storage": "memory"})
		return
	}
	if err := a.cfRepo.Ping(r.Context()); err != nil {
		errorJSON(w, r, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "postgres is not reachable")
		return
	}
	if err := a.cfRepo.Readiness(r.Context()); err != nil {
		errorJSON(w, r, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "critical tables are not ready")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready", "storage": "postgres"})
}

func (a *app) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		Name             string `json:"name"`
		OrganizationName string `json:"organization_name"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := validation.Email(req.Email); err != nil {
		errorJSON(w, r, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := validation.Password(req.Password); err != nil {
		errorJSON(w, r, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		errorJSON(w, r, 500, "INTERNAL", "could not hash password")
		return
	}
	if a.cfRepo != nil {
		orgName := strings.TrimSpace(req.OrganizationName)
		if orgName == "" {
			orgName = strings.TrimSpace(req.Name)
		}
		u, memberships, err := a.cfRepo.CreateUserWithDefaultOrganization(r.Context(), req.Email, hash, orgName)
		if err == repository.ErrDuplicateEmail {
			errorJSON(w, r, 409, "CONFLICT", "email is already registered")
			return
		}
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not create user")
			return
		}
		a.issueAuthTokens(w, r, http.StatusCreated, u, memberships)
		return
	}
	a.store.mu.Lock()
	if _, ok := a.store.usersByEmail[strings.ToLower(req.Email)]; ok {
		a.store.mu.Unlock()
		errorJSON(w, r, 400, "VALIDATION_ERROR", "email is already registered")
		return
	}
	u := models.User{ID: auth.NewID(), Email: strings.ToLower(req.Email), PasswordHash: hash, CreatedAt: time.Now().UTC()}
	a.store.users[u.ID] = u
	a.store.usersByEmail[u.Email] = u.ID
	a.store.profiles[u.ID] = defaultProfile(u.ID)
	org := a.ensureOrganizationLocked(u.ID, strings.TrimSpace(req.OrganizationName))
	a.addOrganizationMemberLocked(org.ID, u.ID, authz.RoleOwner)
	memberships := a.userMembershipsLocked(u.ID)
	a.store.mu.Unlock()
	a.issueAuthTokens(w, r, http.StatusCreated, u, memberships)
}

func (a *app) login(w http.ResponseWriter, r *http.Request) {
	var req struct{ Email, Password string }
	if !decode(w, r, &req) {
		return
	}
	if a.cfRepo != nil {
		u, err := a.cfRepo.GetUserByEmail(r.Context(), req.Email)
		if err != nil || !auth.CheckPassword(u.PasswordHash, req.Password) {
			errorJSON(w, r, 401, "UNAUTHORIZED", "invalid email or password")
			return
		}
		memberships, err := a.cfRepo.ListUserMemberships(r.Context(), u.ID)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not load organizations")
			return
		}
		a.issueAuthTokens(w, r, http.StatusOK, u, memberships)
		return
	}
	a.store.mu.RLock()
	id, ok := a.store.usersByEmail[strings.ToLower(req.Email)]
	u := a.store.users[id]
	a.store.mu.RUnlock()
	if !ok || !auth.CheckPassword(u.PasswordHash, req.Password) {
		errorJSON(w, r, 401, "UNAUTHORIZED", "invalid email or password")
		return
	}
	a.issueAuthTokens(w, r, http.StatusOK, u, a.userMembershipsLocked(u.ID))
}

func (a *app) demoToken(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		u, err := a.cfRepo.GetUserByEmail(r.Context(), "demo@clearflow.dev")
		if err != nil {
			hash, hashErr := auth.HashPassword("demo-password")
			if hashErr != nil {
				errorJSON(w, r, 500, "INTERNAL", "could not prepare demo user")
				return
			}
			var memberships []models.OrganizationMember
			u, memberships, err = a.cfRepo.CreateUserWithDefaultOrganization(r.Context(), "demo@clearflow.dev", hash, "Clearflow Demo Organization")
			if err != nil {
				errorJSON(w, r, 500, "DATABASE_ERROR", "could not create demo user")
				return
			}
			token, _ := auth.SignJWT(a.cfg.JWTSecret, u.ID, u.Email, 24*time.Hour)
			writeJSON(w, 200, map[string]interface{}{"token": token, "user": u, "organizations": membershipOrganizations(memberships)})
			return
		}
		memberships, _ := a.cfRepo.ListUserMemberships(r.Context(), u.ID)
		token, _ := auth.SignJWT(a.cfg.JWTSecret, u.ID, u.Email, 24*time.Hour)
		writeJSON(w, 200, map[string]interface{}{"token": token, "user": u, "organizations": membershipOrganizations(memberships)})
		return
	}
	u := a.seedDemo()
	token, _ := auth.SignJWT(a.cfg.JWTSecret, u.ID, u.Email, 24*time.Hour)
	writeJSON(w, 200, map[string]interface{}{"token": token, "user": u})
}

func (a *app) me(w http.ResponseWriter, r *http.Request) {
	u, _ := a.currentUser(r)
	writeJSON(w, 200, u)
}

func (a *app) meV1(w http.ResponseWriter, r *http.Request) {
	u, ok := a.currentUser(r)
	if !ok {
		errorJSON(w, r, 404, "NOT_FOUND", "user not found")
		return
	}
	memberships, err := a.userMemberships(r.Context(), u.ID)
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not load organizations")
		return
	}
	writeJSON(w, 200, map[string]interface{}{"user": publicUser(u), "organizations": membershipOrganizations(memberships)})
}
func (a *app) getProfile(w http.ResponseWriter, r *http.Request) {
	p := a.profile(userID(r))
	writeJSON(w, 200, p)
}
func (a *app) putProfile(w http.ResponseWriter, r *http.Request) {
	var p models.AdvisorProfile
	if !decode(w, r, &p) {
		return
	}
	p.UserID = userID(r)
	p.RiskTolerance = validation.RiskTolerance(p.RiskTolerance)
	if p.ID == "" {
		p.ID = auth.NewID()
		p.CreatedAt = time.Now().UTC()
	}
	p.UpdatedAt = time.Now().UTC()
	a.store.mu.Lock()
	a.store.profiles[p.UserID] = p
	a.store.mu.Unlock()
	writeJSON(w, 200, p)
}

func (a *app) importTransactionsCSV(w http.ResponseWriter, r *http.Request) {
	file, header, ok := upload(w, r, "file")
	if !ok {
		return
	}
	defer file.Close()
	raw, _ := io.ReadAll(file)
	key := "transactions/" + auth.NewID() + "-" + cleanName(header.Filename)
	_ = a.raw.Put(r.Context(), key, raw)
	rows, failed := parseTransactionsCSV(userID(r), string(raw))
	imp := models.RawImport{ID: auth.NewID(), UserID: userID(r), ImportType: "transactions", OriginalFilename: header.Filename, RawStorageKey: key, RowCount: len(rows) + failed, ImportedCount: len(rows), FailedCount: failed, CreatedAt: time.Now().UTC()}
	a.store.mu.Lock()
	a.store.imports[imp.ID] = imp
	for i := range rows {
		rows[i].ImportID = imp.ID
		a.store.transactions[rows[i].ID] = rows[i]
	}
	a.store.mu.Unlock()
	writeJSON(w, 201, map[string]interface{}{"import": imp, "transactions": rows})
}

func (a *app) listImports(w http.ResponseWriter, r *http.Request) {
	out := []models.RawImport{}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	for _, imp := range a.store.imports {
		if imp.UserID == userID(r) {
			out = append(out, imp)
		}
	}
	writeJSON(w, 200, out)
}
func (a *app) getImport(w http.ResponseWriter, r *http.Request) {
	a.store.mu.RLock()
	imp, ok := a.store.imports[r.PathValue("id")]
	a.store.mu.RUnlock()
	if !ok || imp.UserID != userID(r) {
		errorJSON(w, r, 404, "NOT_FOUND", "import not found")
		return
	}
	writeJSON(w, 200, imp)
}

func (a *app) createTransaction(w http.ResponseWriter, r *http.Request) {
	var t models.Transaction
	if !decode(w, r, &t) {
		return
	}
	t.ID = auth.NewID()
	t.UserID = userID(r)
	t.CreatedAt = time.Now().UTC()
	normalizeTransaction(&t)
	a.store.mu.Lock()
	a.store.transactions[t.ID] = t
	a.store.mu.Unlock()
	writeJSON(w, 201, t)
}
func (a *app) listTransactions(w http.ResponseWriter, r *http.Request) {
	rows := a.transactions(userID(r))
	q := r.URL.Query()
	filtered := rows[:0]
	for _, t := range rows {
		if q.Get("category") != "" && !strings.EqualFold(t.Category, q.Get("category")) {
			continue
		}
		if q.Get("merchant") != "" && !strings.Contains(strings.ToLower(t.NormalizedMerchant), strings.ToLower(q.Get("merchant"))) {
			continue
		}
		if q.Get("direction") != "" && t.Direction != q.Get("direction") {
			continue
		}
		filtered = append(filtered, t)
	}
	writeJSON(w, 200, filtered)
}
func (a *app) getTransaction(w http.ResponseWriter, r *http.Request) {
	a.store.mu.RLock()
	t, ok := a.store.transactions[r.PathValue("id")]
	a.store.mu.RUnlock()
	if !ok || t.UserID != userID(r) {
		errorJSON(w, r, 404, "NOT_FOUND", "transaction not found")
		return
	}
	writeJSON(w, 200, t)
}
func (a *app) patchTransactionCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Category string `json:"category"`
	}
	if !decode(w, r, &req) {
		return
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	t, ok := a.store.transactions[r.PathValue("id")]
	if !ok || t.UserID != userID(r) {
		errorJSON(w, r, 404, "NOT_FOUND", "transaction not found")
		return
	}
	t.Category = req.Category
	a.store.transactions[t.ID] = t
	writeJSON(w, 200, t)
}
func (a *app) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	t, ok := a.store.transactions[r.PathValue("id")]
	if !ok || t.UserID != userID(r) {
		errorJSON(w, r, 404, "NOT_FOUND", "transaction not found")
		return
	}
	delete(a.store.transactions, t.ID)
	w.WriteHeader(204)
}

func (a *app) monthlySummary(w http.ResponseWriter, r *http.Request) {
	rows := a.transactions(userID(r))
	writeJSON(w, 200, map[string]interface{}{"cash_flow": advisor.CashFlow(rows), "top_categories": advisor.CategoryBreakdown(rows), "top_merchants": advisor.MerchantBreakdown(rows), "subscriptions": advisor.DetectSubscriptions(rows), "unusual_transactions": advisor.DetectAnomalies(rows)})
}
func (a *app) categories(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, advisor.CategoryBreakdown(a.transactions(userID(r))))
}
func (a *app) merchants(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, advisor.MerchantBreakdown(a.transactions(userID(r))))
}
func (a *app) subscriptions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, advisor.DetectSubscriptions(a.transactions(userID(r))))
}
func (a *app) anomalies(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, advisor.DetectAnomalies(a.transactions(userID(r))))
}
func (a *app) duplicates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, advisor.DetectDuplicateCharges(a.transactions(userID(r))))
}
func (a *app) cashFlow(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, advisor.CashFlow(a.transactions(userID(r))))
}
func (a *app) advisorPlan(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	p := a.profile(uid)
	rows := a.transactions(uid)
	writeJSON(w, 200, map[string]interface{}{"average_net_cash_flow": advisor.CashFlow(rows).AverageNetCashFlow, "recommended_allocation": advisor.MonthlyAllocation(p, rows), "emergency_fund": advisor.EmergencyFund(p, rows), "account_priority": advisor.AccountPriority(p), "disclaimer": "Educational estimate only, not financial advice."})
}
func (a *app) emergencyFund(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, advisor.EmergencyFund(a.profile(userID(r)), a.transactions(userID(r))))
}
func (a *app) accountPriority(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, advisor.AccountPriority(a.profile(userID(r))))
}
func (a *app) investmentProjection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MonthlyContribution float64 `json:"monthly_contribution"`
		InitialBalance      float64 `json:"initial_balance"`
		Years               int     `json:"years"`
		RiskTolerance       string  `json:"risk_tolerance"`
	}
	if !decode(w, r, &req) {
		return
	}
	writeJSON(w, 200, advisor.InvestmentProjection(req.MonthlyContribution, req.InitialBalance, req.Years, req.RiskTolerance))
}
func (a *app) chat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
	}
	if !decode(w, r, &req) {
		return
	}
	uid := userID(r)
	summary := a.summary(uid)
	risks := portfolio.ConcentrationRisk(a.holdings(uid), a.profile(uid))
	riskText := []string{}
	for _, f := range risks {
		riskText = append(riskText, f.Explanation)
	}
	writeJSON(w, 200, map[string]string{"answer": advisor.RuleBasedChat(req.Message, a.profile(uid), a.transactions(uid), summary.TotalMarketValue, riskText), "mode": "deterministic"})
}
func (a *app) monthlyAdvisorSummary(w http.ResponseWriter, r *http.Request) { a.advisorPlan(w, r) }

func (a *app) createAccount(w http.ResponseWriter, r *http.Request) {
	var acct models.BrokerageAccount
	if !decode(w, r, &acct) {
		return
	}
	acct.ID = auth.NewID()
	acct.UserID = userID(r)
	acct.CreatedAt = time.Now().UTC()
	acct.UpdatedAt = acct.CreatedAt
	if acct.Provider == "" {
		acct.Provider = "manual"
	}
	if acct.ConnectionStatus == "" {
		acct.ConnectionStatus = "manual"
	}
	if a.cfRepo != nil {
		if err := a.cfRepo.EnsureUser(r.Context(), a.currentClearflowUser(r)); err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not prepare portfolio user")
			return
		}
		created, err := a.cfRepo.CreateBrokerageAccount(r.Context(), acct)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not create brokerage account")
			return
		}
		writeJSON(w, 201, created)
		return
	}
	a.store.mu.Lock()
	a.store.accounts[acct.ID] = acct
	a.store.mu.Unlock()
	writeJSON(w, 201, acct)
}
func (a *app) listAccounts(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		rows, err := a.cfRepo.ListBrokerageAccounts(r.Context(), userID(r))
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not list brokerage accounts")
			return
		}
		writeJSON(w, 200, rows)
		return
	}
	writeJSON(w, 200, a.accounts(userID(r)))
}
func (a *app) getAccount(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		acct, err := a.cfRepo.GetBrokerageAccount(r.Context(), userID(r), r.PathValue("id"))
		if err != nil {
			errorJSON(w, r, 404, "NOT_FOUND", "account not found")
			return
		}
		writeJSON(w, 200, acct)
		return
	}
	a.store.mu.RLock()
	acct, ok := a.store.accounts[r.PathValue("id")]
	a.store.mu.RUnlock()
	if !ok || acct.UserID != userID(r) {
		errorJSON(w, r, 404, "NOT_FOUND", "account not found")
		return
	}
	writeJSON(w, 200, acct)
}
func (a *app) deleteAccount(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		if err := a.cfRepo.DeleteBrokerageAccount(r.Context(), userID(r), r.PathValue("id")); err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not delete brokerage account")
			return
		}
		w.WriteHeader(204)
		return
	}
	a.store.mu.Lock()
	delete(a.store.accounts, r.PathValue("id"))
	a.store.mu.Unlock()
	w.WriteHeader(204)
}
func (a *app) listConnections(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.PlaidConnection{}
	for _, c := range a.store.plaidConnections {
		if c.UserID == uid {
			out = append(out, c)
		}
	}
	writeJSON(w, 200, out)
}
func (a *app) deleteConnection(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	id := r.PathValue("id")
	a.store.mu.Lock()
	c, ok := a.store.plaidConnections[id]
	if !ok || c.UserID != uid {
		a.store.mu.Unlock()
		errorJSON(w, r, 404, "NOT_FOUND", "connection not found")
		return
	}
	delete(a.store.plaidConnections, id)
	a.store.mu.Unlock()
	if err := a.persistPlaidConnections(); err != nil {
		a.log.Error("persist_plaid_connections_failed", map[string]interface{}{"error": err.Error()})
	}
	w.WriteHeader(204)
}
func (a *app) createPlaidLinkToken(w http.ResponseWriter, r *http.Request) {
	if !a.plaid.Ready() {
		errorJSON(w, r, 400, "PLAID_NOT_CONFIGURED", "PLAID_CLIENT_ID and PLAID_SECRET must be set in the API environment")
		return
	}
	link, err := a.plaid.CreateLinkToken(r.Context(), userID(r), a.cfg.PlaidProducts, a.cfg.PlaidCountries)
	if err != nil {
		errorJSON(w, r, 502, "PLAID_ERROR", err.Error())
		return
	}
	writeJSON(w, 200, link)
}

func (a *app) createPlaidSandboxConnection(w http.ResponseWriter, r *http.Request) {
	if strings.ToLower(a.cfg.PlaidEnv) != "sandbox" {
		errorJSON(w, r, 400, "PLAID_ENV_NOT_SANDBOX", "sandbox test connections require PLAID_ENV=sandbox")
		return
	}
	if !a.plaid.Ready() {
		errorJSON(w, r, 400, "PLAID_NOT_CONFIGURED", "PLAID_CLIENT_ID and PLAID_SECRET must be set in the API environment")
		return
	}
	var req struct {
		InstitutionID string `json:"institution_id"`
		Username      string `json:"username"`
		Password      string `json:"password"`
	}
	decodeOptional(r, &req)
	token, err := a.plaid.CreateSandboxPublicToken(r.Context(), req.InstitutionID, a.cfg.PlaidProducts, req.Username, req.Password)
	if err != nil {
		errorJSON(w, r, 502, "PLAID_ERROR", err.Error())
		return
	}
	exchange, err := a.storePlaidConnectionFromPublicToken(r, token.PublicToken)
	if err != nil {
		errorJSON(w, r, 502, "PLAID_ERROR", err.Error())
		return
	}
	imported, err := a.syncOnePlaidConnection(r.Context(), exchange)
	if err != nil {
		errorJSON(w, r, 502, "PLAID_ERROR", err.Error())
		return
	}
	writeJSON(w, 201, map[string]interface{}{"connection": exchange, "imported_count": imported, "sandbox": true})
}

func (a *app) exchangePlaidPublicToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PublicToken string `json:"public_token"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.PublicToken == "" {
		errorJSON(w, r, 400, "VALIDATION_ERROR", "public_token is required")
		return
	}
	conn, err := a.storePlaidConnectionFromPublicToken(r, req.PublicToken)
	if err != nil {
		errorJSON(w, r, 502, "PLAID_ERROR", err.Error())
		return
	}
	writeJSON(w, 201, map[string]interface{}{"connection": conn})
}

func (a *app) storePlaidConnectionFromPublicToken(r *http.Request, publicToken string) (models.PlaidConnection, error) {
	exchange, err := a.plaid.ExchangePublicToken(r.Context(), publicToken)
	if err != nil {
		return models.PlaidConnection{}, err
	}
	item, err := a.plaid.GetItem(r.Context(), exchange.AccessToken)
	if err != nil {
		return models.PlaidConnection{}, err
	}
	ciphertext, err := a.encryptToken(exchange.AccessToken)
	if err != nil {
		return models.PlaidConnection{}, fmt.Errorf("could not secure Plaid token")
	}
	now := time.Now().UTC()
	name := item.Institution.Name
	if name == "" {
		name = "Plaid institution"
	}
	conn := models.PlaidConnection{ID: auth.NewID(), UserID: userID(r), ItemID: exchange.ItemID, InstitutionName: name, AccessTokenCiphertext: ciphertext, Products: splitCSV(a.cfg.PlaidProducts), CreatedAt: now, UpdatedAt: now}
	if a.cfRepo != nil {
		orgs := a.userOrganizations(userID(r))
		if len(orgs) > 0 {
			if err := a.cfRepo.SavePlaidItem(r.Context(), orgs[0].ID, userID(r), conn.ItemID, conn.AccessTokenCiphertext, item.Item.InstitutionID, conn.InstitutionName, conn.Cursor); err != nil {
				return models.PlaidConnection{}, fmt.Errorf("could not persist Plaid item")
			}
			a.writeAudit(r.Context(), r, orgs[0].ID, userID(r), "plaid.item_connected", "plaid_item", conn.ItemID, "{}")
		}
	}
	a.store.mu.Lock()
	a.store.plaidConnections[conn.ID] = conn
	a.store.mu.Unlock()
	if err := a.persistPlaidConnections(); err != nil {
		return models.PlaidConnection{}, fmt.Errorf("could not persist Plaid connection")
	}
	return conn, nil
}
func (a *app) syncPlaidTransactions(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConnectionID string `json:"connection_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	uid := userID(r)
	connections := a.userPlaidConnections(uid)
	if req.ConnectionID != "" {
		connections = nil
		a.store.mu.RLock()
		c, ok := a.store.plaidConnections[req.ConnectionID]
		a.store.mu.RUnlock()
		if !ok || c.UserID != uid {
			errorJSON(w, r, 404, "NOT_FOUND", "connection not found")
			return
		}
		connections = []models.PlaidConnection{c}
	}
	imported := 0
	for _, conn := range connections {
		n, err := a.syncOnePlaidConnection(r.Context(), conn)
		if err != nil {
			errorJSON(w, r, 502, "PLAID_ERROR", err.Error())
			return
		}
		imported += n
	}
	writeJSON(w, 200, map[string]interface{}{"imported_count": imported, "connection_count": len(connections)})
}

func (a *app) syncPlaidInvestments(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	accountID, ok := a.defaultPortfolioAccountID(w, r)
	if !ok {
		return
	}
	holdings, parseErrors := parseHoldingsCSV(userID(r), demoPlaidInvestmentHoldingsCSV(), accountID)
	txs, txErrors := parsePortfolioTransactionsCSV(userID(r), demoPlaidInvestmentTransactionsCSV(), accountID)
	parseErrors = append(parseErrors, txErrors...)
	imp := models.RawImport{ID: auth.NewID(), UserID: userID(r), ImportType: "holdings", OriginalFilename: "plaid_investments_mock", RawStorageKey: "plaid/investments/mock", RowCount: len(holdings) + len(txs) + len(parseErrors), ImportedCount: len(holdings) + len(txs), FailedCount: len(parseErrors), CreatedAt: time.Now().UTC()}
	if a.cfRepo != nil {
		saved, err := a.cfRepo.SavePortfolioImport(r.Context(), imp, holdings, txs, parseErrors)
		if err != nil {
			a.logOperation(r, "plaid.investments_sync_failed", "", map[string]interface{}{"error": err.Error(), "latency_ms": time.Since(start).Milliseconds()})
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not save Plaid investments sync")
			return
		}
		a.logOperation(r, "plaid.investments_synced", "", map[string]interface{}{"mode": "mock", "holdings": len(holdings), "transactions": len(txs), "errors": len(parseErrors), "storage": "postgres", "latency_ms": time.Since(start).Milliseconds()})
		writeJSON(w, 200, map[string]interface{}{"mode": "mock", "import": saved, "holdings": holdings, "portfolio_transactions": txs, "errors": parseErrors})
		return
	}
	a.store.mu.Lock()
	a.store.imports[imp.ID] = imp
	for i := range parseErrors {
		parseErrors[i].ImportID = imp.ID
		a.store.importErrors[parseErrors[i].ID] = parseErrors[i]
	}
	for _, h := range holdings {
		a.store.holdings[h.ID] = h
	}
	for _, tx := range txs {
		tx.ImportID = imp.ID
		a.store.portfolioTransactions[tx.ID] = tx
	}
	a.store.mu.Unlock()
	a.logOperation(r, "plaid.investments_synced", "", map[string]interface{}{"mode": "mock", "holdings": len(holdings), "transactions": len(txs), "errors": len(parseErrors), "latency_ms": time.Since(start).Milliseconds()})
	writeJSON(w, 200, map[string]interface{}{"mode": "mock", "import": imp, "holdings": holdings, "portfolio_transactions": txs, "errors": parseErrors})
}

func (a *app) importHoldingsCSV(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	file, header, ok := upload(w, r, "file")
	if !ok {
		return
	}
	defer file.Close()
	raw, _ := io.ReadAll(file)
	key := "holdings/" + auth.NewID() + "-" + cleanName(header.Filename)
	_ = a.raw.Put(r.Context(), key, raw)
	accountID, ok := a.defaultPortfolioAccountID(w, r)
	if !ok {
		return
	}
	rows, importErrors := parseHoldingsCSV(userID(r), string(raw), accountID)
	imp := models.RawImport{ID: auth.NewID(), UserID: userID(r), ImportType: "holdings", OriginalFilename: header.Filename, RawStorageKey: key, RowCount: len(rows) + len(importErrors), ImportedCount: len(rows), FailedCount: len(importErrors), CreatedAt: time.Now().UTC()}
	if a.cfRepo != nil {
		saved, err := a.cfRepo.SavePortfolioImport(r.Context(), imp, rows, nil, importErrors)
		if err != nil {
			a.logOperation(r, "portfolio.holdings_import_failed", "", map[string]interface{}{"filename": header.Filename, "error": err.Error(), "latency_ms": time.Since(start).Milliseconds()})
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not save holdings import")
			return
		}
		a.logOperation(r, "portfolio.holdings_imported", "", map[string]interface{}{"filename": header.Filename, "rows": saved.RowCount, "imported": saved.ImportedCount, "failed": saved.FailedCount, "storage": "postgres", "latency_ms": time.Since(start).Milliseconds()})
		writeJSON(w, 201, map[string]interface{}{"import": saved, "holdings": rows, "errors": importErrors})
		return
	}
	a.store.mu.Lock()
	a.store.imports[imp.ID] = imp
	for i := range importErrors {
		importErrors[i].ImportID = imp.ID
		a.store.importErrors[importErrors[i].ID] = importErrors[i]
	}
	for _, h := range rows {
		a.store.holdings[h.ID] = h
	}
	a.store.mu.Unlock()
	a.logOperation(r, "portfolio.holdings_imported", "", map[string]interface{}{"filename": header.Filename, "rows": imp.RowCount, "imported": imp.ImportedCount, "failed": imp.FailedCount, "latency_ms": time.Since(start).Milliseconds()})
	writeJSON(w, 201, map[string]interface{}{"import": imp, "holdings": rows, "errors": importErrors})
}
func (a *app) importPortfolioTransactionsCSV(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	file, header, ok := upload(w, r, "file")
	if !ok {
		return
	}
	defer file.Close()
	raw, _ := io.ReadAll(file)
	key := "portfolio-transactions/" + auth.NewID() + "-" + cleanName(header.Filename)
	_ = a.raw.Put(r.Context(), key, raw)
	accountID, ok := a.defaultPortfolioAccountID(w, r)
	if !ok {
		return
	}
	rows, importErrors := parsePortfolioTransactionsCSV(userID(r), string(raw), accountID)
	imp := models.RawImport{ID: auth.NewID(), UserID: userID(r), ImportType: "portfolio_transactions", OriginalFilename: header.Filename, RawStorageKey: key, RowCount: len(rows) + len(importErrors), ImportedCount: len(rows), FailedCount: len(importErrors), CreatedAt: time.Now().UTC()}
	if a.cfRepo != nil {
		saved, err := a.cfRepo.SavePortfolioImport(r.Context(), imp, nil, rows, importErrors)
		if err != nil {
			a.logOperation(r, "portfolio.transactions_import_failed", "", map[string]interface{}{"filename": header.Filename, "error": err.Error(), "latency_ms": time.Since(start).Milliseconds()})
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not save portfolio transaction import")
			return
		}
		a.logOperation(r, "portfolio.transactions_imported", "", map[string]interface{}{"filename": header.Filename, "rows": saved.RowCount, "imported": saved.ImportedCount, "failed": saved.FailedCount, "storage": "postgres", "latency_ms": time.Since(start).Milliseconds()})
		writeJSON(w, 201, map[string]interface{}{"import": saved, "portfolio_transactions": rows, "errors": importErrors})
		return
	}
	a.store.mu.Lock()
	a.store.imports[imp.ID] = imp
	for i := range importErrors {
		importErrors[i].ImportID = imp.ID
		a.store.importErrors[importErrors[i].ID] = importErrors[i]
	}
	for _, row := range rows {
		row.ImportID = imp.ID
		a.store.portfolioTransactions[row.ID] = row
	}
	a.store.mu.Unlock()
	a.logOperation(r, "portfolio.transactions_imported", "", map[string]interface{}{"filename": header.Filename, "rows": imp.RowCount, "imported": imp.ImportedCount, "failed": imp.FailedCount, "latency_ms": time.Since(start).Milliseconds()})
	writeJSON(w, 201, map[string]interface{}{"import": imp, "portfolio_transactions": rows, "errors": importErrors})
}
func (a *app) createHolding(w http.ResponseWriter, r *http.Request) {
	var h models.Holding
	if !decode(w, r, &h) {
		return
	}
	h.ID = auth.NewID()
	h.UserID = userID(r)
	h.CreatedAt = time.Now().UTC()
	h.UpdatedAt = h.CreatedAt
	if h.BrokerageAccountID == "" {
		accountID, ok := a.defaultPortfolioAccountID(w, r)
		if !ok {
			return
		}
		h.BrokerageAccountID = accountID
	}
	h = portfolio.PriceHoldings(r.Context(), a.market, []models.Holding{h})[0]
	if a.cfRepo != nil {
		created, err := a.cfRepo.CreateHolding(r.Context(), h)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not create holding")
			return
		}
		writeJSON(w, 201, created)
		return
	}
	a.store.mu.Lock()
	a.store.holdings[h.ID] = h
	a.store.mu.Unlock()
	writeJSON(w, 201, h)
}
func (a *app) listHoldings(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		rows, err := a.cfRepo.ListHoldings(r.Context(), userID(r))
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not list holdings")
			return
		}
		writeJSON(w, 200, portfolio.PriceHoldings(r.Context(), a.market, rows))
		return
	}
	writeJSON(w, 200, portfolio.PriceHoldings(r.Context(), a.market, a.holdings(userID(r))))
}
func (a *app) getHolding(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		h, err := a.cfRepo.GetHolding(r.Context(), userID(r), r.PathValue("id"))
		if err != nil {
			errorJSON(w, r, 404, "NOT_FOUND", "holding not found")
			return
		}
		writeJSON(w, 200, h)
		return
	}
	a.store.mu.RLock()
	h, ok := a.store.holdings[r.PathValue("id")]
	a.store.mu.RUnlock()
	if !ok || h.UserID != userID(r) {
		errorJSON(w, r, 404, "NOT_FOUND", "holding not found")
		return
	}
	writeJSON(w, 200, h)
}
func (a *app) patchHolding(w http.ResponseWriter, r *http.Request) {
	var patch models.Holding
	if !decode(w, r, &patch) {
		return
	}
	if a.cfRepo != nil {
		h, err := a.cfRepo.GetHolding(r.Context(), userID(r), r.PathValue("id"))
		if err != nil {
			errorJSON(w, r, 404, "NOT_FOUND", "holding not found")
			return
		}
		if patch.Quantity != 0 {
			h.Quantity = patch.Quantity
		}
		if patch.AverageCost != 0 {
			h.AverageCost = patch.AverageCost
		}
		if patch.MarketValue != 0 {
			h.MarketValue = patch.MarketValue
		}
		updated, err := a.cfRepo.UpdateHolding(r.Context(), h)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not update holding")
			return
		}
		writeJSON(w, 200, updated)
		return
	}
	a.store.mu.Lock()
	h, ok := a.store.holdings[r.PathValue("id")]
	if !ok || h.UserID != userID(r) {
		a.store.mu.Unlock()
		errorJSON(w, r, 404, "NOT_FOUND", "holding not found")
		return
	}
	if patch.Quantity != 0 {
		h.Quantity = patch.Quantity
	}
	if patch.AverageCost != 0 {
		h.AverageCost = patch.AverageCost
	}
	if patch.MarketValue != 0 {
		h.MarketValue = patch.MarketValue
	}
	h.UpdatedAt = time.Now().UTC()
	a.store.holdings[h.ID] = h
	a.store.mu.Unlock()
	writeJSON(w, 200, h)
}
func (a *app) deleteHolding(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		if err := a.cfRepo.DeleteHolding(r.Context(), userID(r), r.PathValue("id")); err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not delete holding")
			return
		}
		w.WriteHeader(204)
		return
	}
	a.store.mu.Lock()
	delete(a.store.holdings, r.PathValue("id"))
	a.store.mu.Unlock()
	w.WriteHeader(204)
}
func (a *app) listPortfolioTransactions(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		rows, err := a.cfRepo.ListPortfolioTransactions(r.Context(), userID(r))
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not list portfolio transactions")
			return
		}
		writeJSON(w, 200, rows)
		return
	}
	rows := a.portfolioTxs(userID(r))
	sort.Slice(rows, func(i, j int) bool { return rows[i].OccurredAt.After(rows[j].OccurredAt) })
	writeJSON(w, 200, rows)
}
func (a *app) listPortfolioImports(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	if a.cfRepo != nil {
		rows, err := a.cfRepo.ListPortfolioImports(r.Context(), uid)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not list portfolio imports")
			return
		}
		writeJSON(w, 200, rows)
		return
	}
	out := []models.RawImport{}
	a.store.mu.RLock()
	for _, imp := range a.store.imports {
		if imp.UserID == uid && (imp.ImportType == "holdings" || imp.ImportType == "portfolio_transactions") {
			out = append(out, imp)
		}
	}
	a.store.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	writeJSON(w, 200, out)
}
func (a *app) listPortfolioImportErrors(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	importID := r.PathValue("id")
	if a.cfRepo != nil {
		rows, err := a.cfRepo.ListImportErrors(r.Context(), uid, importID)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not list import errors")
			return
		}
		writeJSON(w, 200, rows)
		return
	}
	out := []models.ImportError{}
	a.store.mu.RLock()
	for _, row := range a.store.importErrors {
		if row.UserID == uid && row.ImportID == importID {
			out = append(out, row)
		}
	}
	a.store.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].RowNumber < out[j].RowNumber })
	writeJSON(w, 200, out)
}
func (a *app) portfolioSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, a.summary(userID(r)))
}
func (a *app) portfolioAllocation(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	writeJSON(w, 200, portfolio.BuildAllocation(a.holdings(uid), a.accounts(uid)))
}
func (a *app) portfolioPerformance(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	writeJSON(w, 200, map[string]interface{}{"summary": a.summary(uid), "performance": portfolio.BuildPerformance(a.holdings(uid), a.portfolioTxs(uid), a.accounts(uid)), "cash_flows": a.portfolioTxs(uid), "method": "cash-flow adjusted estimate using imported holdings and portfolio transactions"})
}
func (a *app) portfolioRisk(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, portfolio.ConcentrationRisk(a.holdings(userID(r)), a.profile(userID(r))))
}
func (a *app) rebalance(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, portfolio.RebalanceSuggestions(portfolio.ConcentrationRisk(a.holdings(userID(r)), a.profile(userID(r)))))
}
func (a *app) projectedGrowth(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	cf := advisor.CashFlow(a.transactions(uid))
	summary := a.summary(uid)
	writeJSON(w, 200, advisor.InvestmentProjection(mathMax(0, cf.AverageNetCashFlow*0.5), summary.TotalMarketValue, 30, a.profile(uid).RiskTolerance))
}
func (a *app) quote(w http.ResponseWriter, r *http.Request) {
	q, _ := a.market.GetQuote(r.Context(), r.PathValue("symbol"))
	writeJSON(w, 200, q)
}
func (a *app) quotes(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Symbols []string `json:"symbols"`
	}
	if !decode(w, r, &req) {
		return
	}
	q, _ := a.market.GetQuotes(r.Context(), req.Symbols)
	writeJSON(w, 200, q)
}

func (a *app) authed(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			errorJSON(w, r, 401, "UNAUTHORIZED", "missing bearer token")
			return
		}
		claims, err := auth.VerifyJWT(a.cfg.JWTSecret, strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			errorJSON(w, r, 401, "UNAUTHORIZED", "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		next(w, r.WithContext(ctx))
	}
}
func userID(r *http.Request) string { v, _ := r.Context().Value("user_id").(string); return v }
func (a *app) currentUser(r *http.Request) (models.User, bool) {
	if a.cfRepo != nil {
		u, err := a.cfRepo.GetUserByID(r.Context(), userID(r))
		if err == nil {
			return u, true
		}
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	u, ok := a.store.users[userID(r)]
	return u, ok
}

func (a *app) defaultPortfolioAccountID(w http.ResponseWriter, r *http.Request) (string, bool) {
	if a.cfRepo != nil {
		if err := a.cfRepo.EnsureUser(r.Context(), a.currentClearflowUser(r)); err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not prepare portfolio user")
			return "", false
		}
		accountID, err := a.cfRepo.EnsureDefaultBrokerageAccount(r.Context(), userID(r))
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not prepare brokerage account")
			return "", false
		}
		return accountID, true
	}
	return a.ensureDefaultAccount(userID(r)), true
}

func (a *app) transactions(uid string) []models.Transaction {
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.Transaction{}
	for _, t := range a.store.transactions {
		if t.UserID == uid {
			out = append(out, t)
		}
	}
	return out
}
func (a *app) holdings(uid string) []models.Holding {
	if a.cfRepo != nil {
		rows, err := a.cfRepo.ListHoldings(context.Background(), uid)
		if err == nil {
			return portfolio.PriceHoldings(context.Background(), a.market, rows)
		}
		a.log.Error("portfolio.holdings_load_failed", map[string]interface{}{"error": err.Error(), "user_id": uid})
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.Holding{}
	for _, h := range a.store.holdings {
		if h.UserID == uid {
			out = append(out, h)
		}
	}
	return portfolio.PriceHoldings(context.Background(), a.market, out)
}
func (a *app) accounts(uid string) []models.BrokerageAccount {
	if a.cfRepo != nil {
		rows, err := a.cfRepo.ListBrokerageAccounts(context.Background(), uid)
		if err == nil {
			return rows
		}
		a.log.Error("portfolio.accounts_load_failed", map[string]interface{}{"error": err.Error(), "user_id": uid})
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.BrokerageAccount{}
	for _, acct := range a.store.accounts {
		if acct.UserID == uid {
			out = append(out, acct)
		}
	}
	return out
}
func (a *app) portfolioTxs(uid string) []models.PortfolioTransaction {
	if a.cfRepo != nil {
		rows, err := a.cfRepo.ListPortfolioTransactions(context.Background(), uid)
		if err == nil {
			return rows
		}
		a.log.Error("portfolio.transactions_load_failed", map[string]interface{}{"error": err.Error(), "user_id": uid})
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.PortfolioTransaction{}
	for _, tx := range a.store.portfolioTransactions {
		if tx.UserID == uid {
			out = append(out, tx)
		}
	}
	return out
}
func (a *app) profile(uid string) models.AdvisorProfile {
	a.store.mu.RLock()
	p, ok := a.store.profiles[uid]
	a.store.mu.RUnlock()
	if ok {
		return p
	}
	return defaultProfile(uid)
}
func (a *app) summary(uid string) portfolio.Summary {
	return portfolio.BuildSummary(a.holdings(uid), a.accounts(uid))
}
func (a *app) userPlaidConnections(uid string) []models.PlaidConnection {
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.PlaidConnection{}
	for _, c := range a.store.plaidConnections {
		if c.UserID == uid {
			out = append(out, c)
		}
	}
	return out
}
func (a *app) syncOnePlaidConnection(ctx context.Context, conn models.PlaidConnection) (int, error) {
	accessToken, err := a.decryptToken(conn.AccessTokenCiphertext)
	if err != nil {
		return 0, fmt.Errorf("could not decrypt Plaid access token")
	}
	imported := 0
	cursor := conn.Cursor
	for {
		resp, err := a.plaid.SyncTransactions(ctx, accessToken, cursor)
		if err != nil {
			return imported, err
		}
		a.store.mu.Lock()
		for _, tx := range resp.Added {
			t := plaidTransactionToModel(conn.UserID, tx)
			if a.hasRawTransactionLocked(conn.UserID, t.RawEventKey) {
				continue
			}
			a.store.transactions[t.ID] = t
			imported++
		}
		for _, tx := range resp.Modified {
			t := plaidTransactionToModel(conn.UserID, tx)
			a.upsertRawTransactionLocked(t)
		}
		for _, removed := range resp.Removed {
			a.deleteRawTransactionLocked(conn.UserID, "plaid:"+removed.TransactionID)
		}
		conn.Cursor = resp.NextCursor
		conn.UpdatedAt = time.Now().UTC()
		conn.LastSyncedAt = conn.UpdatedAt
		a.store.plaidConnections[conn.ID] = conn
		a.store.mu.Unlock()
		if a.cfRepo != nil {
			orgs := a.userOrganizations(conn.UserID)
			if len(orgs) > 0 {
				_ = a.cfRepo.SavePlaidItem(ctx, orgs[0].ID, conn.UserID, conn.ItemID, conn.AccessTokenCiphertext, "", conn.InstitutionName, conn.Cursor)
			}
		}
		cursor = resp.NextCursor
		if !resp.HasMore {
			break
		}
	}
	if err := a.persistPlaidConnections(); err != nil {
		return imported, err
	}
	return imported, nil
}
func (a *app) hasRawTransactionLocked(uid, rawKey string) bool {
	for _, existing := range a.store.transactions {
		if existing.UserID == uid && existing.RawEventKey == rawKey {
			return true
		}
	}
	return false
}
func (a *app) upsertRawTransactionLocked(t models.Transaction) {
	for id, existing := range a.store.transactions {
		if existing.UserID == t.UserID && existing.RawEventKey == t.RawEventKey {
			t.ID = id
			t.CreatedAt = existing.CreatedAt
			a.store.transactions[id] = t
			return
		}
	}
	a.store.transactions[t.ID] = t
}
func (a *app) deleteRawTransactionLocked(uid, rawKey string) {
	for id, existing := range a.store.transactions {
		if existing.UserID == uid && existing.RawEventKey == rawKey {
			delete(a.store.transactions, id)
		}
	}
}
func plaidTransactionToModel(uid string, tx plaid.Transaction) models.Transaction {
	occurredAt, err := parseDate(tx.Date)
	if err != nil {
		occurredAt = time.Now().UTC()
	}
	amount := tx.Amount
	direction := "expense"
	if amount < 0 {
		direction = "income"
		amount = abs(amount)
	}
	merchant := tx.MerchantName
	if merchant == "" {
		merchant = tx.Name
	}
	category := mapPlaidCategory(tx.PersonalFinanceCategory.Primary, tx.Category)
	t := models.Transaction{ID: auth.NewID(), UserID: uid, AccountID: tx.AccountID, Amount: amount, Direction: direction, Currency: fallback(tx.ISOCurrencyCode, "USD"), Merchant: merchant, Description: tx.Name, Category: category, OccurredAt: occurredAt, RawEventKey: "plaid:" + tx.TransactionID, CreatedAt: time.Now().UTC(), Metadata: map[string]interface{}{"source": "plaid", "plaid_transaction_id": tx.TransactionID}}
	normalizeTransaction(&t)
	if category != "" {
		t.Category = category
	}
	return t
}
func mapPlaidCategory(primary string, legacy []string) string {
	p := strings.ToUpper(primary)
	switch {
	case strings.Contains(p, "INCOME"):
		return "Income"
	case strings.Contains(p, "RENT") || strings.Contains(p, "HOME"):
		return "Housing"
	case strings.Contains(p, "GROCER"):
		return "Groceries"
	case strings.Contains(p, "FOOD") || strings.Contains(p, "RESTAURANT"):
		return "Food"
	case strings.Contains(p, "TRANSPORT"):
		return "Transportation"
	case strings.Contains(p, "ENTERTAINMENT") || strings.Contains(p, "SUBSCRIPTION"):
		return "Subscriptions"
	case strings.Contains(p, "SHOP"):
		return "Shopping"
	}
	if len(legacy) > 0 {
		return legacy[len(legacy)-1]
	}
	return "Other"
}
func (a *app) plaidConnectionsPath() string {
	return filepath.Join(filepath.Dir(a.cfg.LocalStorageDir), "plaid-connections.json")
}
func (a *app) loadPlaidConnections() error {
	path := a.plaidConnectionsPath()
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var rows []models.PlaidConnection
	if err := json.Unmarshal(raw, &rows); err != nil {
		return err
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for _, row := range rows {
		a.store.plaidConnections[row.ID] = row
	}
	return nil
}
func (a *app) persistPlaidConnections() error {
	a.store.mu.RLock()
	rows := make([]models.PlaidConnection, 0, len(a.store.plaidConnections))
	for _, row := range a.store.plaidConnections {
		rows = append(rows, row)
	}
	a.store.mu.RUnlock()
	raw, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	path := a.plaidConnectionsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}
func (a *app) encryptToken(token string) (string, error) {
	block, err := aes.NewCipher(a.encryptionKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := crand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(token), nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}
func (a *app) decryptToken(ciphertext string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(a.encryptionKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext is too short")
	}
	nonce, payload := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
func (a *app) encryptionKey() []byte {
	sum := sha256.Sum256([]byte(a.cfg.JWTSecret + ":plaid-access-token"))
	return sum[:]
}

func parseTransactionsCSV(uid, raw string) ([]models.Transaction, int) {
	records, err := csv.NewReader(strings.NewReader(raw)).ReadAll()
	if err != nil || len(records) < 2 {
		return nil, 1
	}
	idx := headerIndex(records[0])
	var out []models.Transaction
	failed := 0
	for _, row := range records[1:] {
		amount, err := strconv.ParseFloat(cell(row, idx, "amount"), 64)
		if err != nil {
			failed++
			continue
		}
		date, err := parseDate(cell(row, idx, "date"))
		if err != nil {
			failed++
			continue
		}
		t := models.Transaction{ID: auth.NewID(), UserID: uid, Amount: abs(amount), Direction: "income", Currency: fallback(cell(row, idx, "currency"), "USD"), Merchant: cell(row, idx, "merchant"), Description: cell(row, idx, "description"), Category: cell(row, idx, "category"), OccurredAt: date, CreatedAt: time.Now().UTC()}
		if amount < 0 {
			t.Direction = "expense"
		}
		normalizeTransaction(&t)
		out = append(out, t)
	}
	return out, failed
}
func parseHoldingsCSV(uid, raw, accountID string) ([]models.Holding, []models.ImportError) {
	records, err := csv.NewReader(strings.NewReader(raw)).ReadAll()
	if err != nil || len(records) < 2 {
		return nil, []models.ImportError{newImportError(uid, "", 1, "", "invalid_csv", "CSV is empty or could not be parsed", nil)}
	}
	idx := headerIndex(records[0])
	var out []models.Holding
	var importErrors []models.ImportError
	for rowIndex, row := range records[1:] {
		rowNumber := rowIndex + 2
		qty, err := parseFloatRequired(cellAny(row, idx, "quantity", "qty", "shares", "units"))
		if err != nil {
			importErrors = append(importErrors, newImportError(uid, "", rowNumber, "quantity", "invalid_quantity", "quantity/shares is required and must be numeric", row))
			continue
		}
		avg := parseFloat(firstNonEmpty(cellAny(row, idx, "average_cost", "avg_cost", "cost_basis_per_share", "average_price"), perShareCost(cellAny(row, idx, "cost_basis", "book_cost", "total_cost"), qty)))
		mv := parseFloat(cellAny(row, idx, "market_value", "value", "current_value"))
		price := parseFloat(firstNonEmpty(cellAny(row, idx, "market_price", "last_price", "price", "current_price"), perShareCost(cellAny(row, idx, "market_value", "value", "current_value"), qty)))
		symbol := strings.ToUpper(cellAny(row, idx, "symbol", "ticker", "security_symbol"))
		if symbol == "" {
			importErrors = append(importErrors, newImportError(uid, "", rowNumber, "symbol", "missing_symbol", "symbol/ticker is required", row))
			continue
		}
		h := models.Holding{ID: auth.NewID(), UserID: uid, BrokerageAccountID: accountID, Symbol: symbol, SecurityName: fallback(cellAny(row, idx, "name", "security_name", "description", "security"), symbol), SecurityType: normalizeSecurityType(cellAny(row, idx, "security_type", "type", "asset_class")), Quantity: qty, AverageCost: avg, Currency: fallback(cellAny(row, idx, "currency", "ccy"), "USD"), MarketValue: mv, LastPrice: price, PriceAsOf: time.Now().UTC(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
		if h.MarketValue == 0 && h.LastPrice > 0 {
			h.MarketValue = h.Quantity * h.LastPrice
		}
		if h.AverageCost == 0 && h.MarketValue > 0 && h.Quantity > 0 {
			h.AverageCost = h.MarketValue / h.Quantity
		}
		out = append(out, h)
	}
	return out, importErrors
}
func parsePortfolioTransactionsCSV(uid, raw, accountID string) ([]models.PortfolioTransaction, []models.ImportError) {
	records, err := csv.NewReader(strings.NewReader(raw)).ReadAll()
	if err != nil || len(records) < 2 {
		return nil, []models.ImportError{newImportError(uid, "", 1, "", "invalid_csv", "CSV is empty or could not be parsed", nil)}
	}
	idx := headerIndex(records[0])
	var out []models.PortfolioTransaction
	var importErrors []models.ImportError
	for rowIndex, row := range records[1:] {
		rowNumber := rowIndex + 2
		date, err := parseDate(cellAny(row, idx, "date", "trade_date", "transaction_date", "settlement_date", "occurred_at"))
		if err != nil {
			importErrors = append(importErrors, newImportError(uid, "", rowNumber, "date", "invalid_date", "date/trade_date is required and must be a supported date format", row))
			continue
		}
		txType := normalizePortfolioTransactionType(firstNonEmpty(cellAny(row, idx, "action", "transaction_type", "type", "activity"), cellAny(row, idx, "description")))
		if txType == "" {
			importErrors = append(importErrors, newImportError(uid, "", rowNumber, "transaction_type", "invalid_transaction_type", "action/activity/transaction_type is required", row))
			continue
		}
		amount := parseFloat(cellAny(row, idx, "amount", "net_amount", "total", "value"))
		fees := parseFloat(cellAny(row, idx, "fees", "fee", "commission"))
		out = append(out, models.PortfolioTransaction{ID: auth.NewID(), UserID: uid, BrokerageAccountID: accountID, Symbol: strings.ToUpper(cellAny(row, idx, "symbol", "ticker", "security_symbol")), TransactionType: txType, Quantity: parseFloat(cellAny(row, idx, "quantity", "qty", "shares", "units")), Price: parseFloat(cellAny(row, idx, "price", "trade_price", "unit_price")), Amount: amount, Fees: fees, Currency: fallback(cellAny(row, idx, "currency", "ccy"), "USD"), OccurredAt: date, Description: firstNonEmpty(cellAny(row, idx, "description", "memo", "details"), txType), CreatedAt: time.Now().UTC()})
	}
	return out, importErrors
}
func newImportError(uid, importID string, rowNumber int, field, code, message string, rawRow []string) models.ImportError {
	return models.ImportError{ID: auth.NewID(), ImportID: importID, UserID: uid, RowNumber: rowNumber, Field: field, Code: code, Message: message, RawRow: rawRow, CreatedAt: time.Now().UTC()}
}
func normalizeTransaction(t *models.Transaction) {
	if t.Currency == "" {
		t.Currency = "USD"
	}
	if t.OccurredAt.IsZero() {
		t.OccurredAt = time.Now().UTC()
	}
	t.NormalizedMerchant = advisor.NormalizeMerchant(t.Description, t.Merchant)
	if t.Category == "" {
		t.Category = advisor.Categorize(*t)
	}
	if t.Direction == "" {
		if t.Amount < 0 {
			t.Direction = "expense"
			t.Amount = abs(t.Amount)
		} else {
			t.Direction = "income"
		}
	}
}
func headerIndex(header []string) map[string]int {
	m := map[string]int{}
	for i, h := range header {
		m[normalizeHeader(h)] = i
	}
	return m
}
func cell(row []string, idx map[string]int, key string) string {
	if i, ok := idx[normalizeHeader(key)]; ok && i < len(row) {
		return strings.TrimSpace(row[i])
	}
	return ""
}
func cellAny(row []string, idx map[string]int, keys ...string) string {
	for _, key := range keys {
		if value := cell(row, idx, key); value != "" {
			return value
		}
	}
	return ""
}
func normalizeHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "_", "-", "_", "/", "_", ".", "", "(", "", ")", "", "%", "pct")
	value = replacer.Replace(value)
	for strings.Contains(value, "__") {
		value = strings.ReplaceAll(value, "__", "_")
	}
	return strings.Trim(value, "_")
}
func parseDate(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	for _, layout := range []string{"2006-01-02", "01/02/2006", "1/2/2006", "2006/01/02", "Jan 2 2006", "January 2 2006", time.RFC3339} {
		if t, err := time.Parse(layout, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid date")
}
func parseFloat(v string) float64 {
	f, _ := parseFloatRequired(v)
	return f
}
func parseFloatRequired(v string) (float64, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, errors.New("missing number")
	}
	negative := strings.HasPrefix(v, "(") && strings.HasSuffix(v, ")")
	v = strings.Trim(v, "()")
	replacer := strings.NewReplacer("$", "", "C$", "", "CA$", "", "US$", "", ",", "", "%", "")
	v = strings.TrimSpace(replacer.Replace(v))
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, err
	}
	if negative {
		f = -f
	}
	return f, nil
}
func fallback(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return b
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
func perShareCost(totalValue string, quantity float64) string {
	total := parseFloat(totalValue)
	if total == 0 || quantity == 0 {
		return ""
	}
	return fmt.Sprintf("%f", total/quantity)
}
func normalizeSecurityType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "stock", "equity", "common_stock", "common stock":
		return "stock"
	case "etf", "exchange_traded_fund", "exchange traded fund":
		return "etf"
	case "mutual_fund", "mutual fund", "fund":
		return "mutual_fund"
	case "crypto", "cryptocurrency":
		return "crypto"
	case "cash", "money_market", "money market":
		return "cash"
	case "":
		return "etf"
	default:
		return "other"
	}
}
func normalizePortfolioTransactionType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch {
	case value == "":
		return ""
	case strings.Contains(value, "buy") || strings.Contains(value, "purchase"):
		return "buy"
	case strings.Contains(value, "sell"):
		return "sell"
	case strings.Contains(value, "dividend") || strings.Contains(value, "distribution"):
		return "dividend"
	case strings.Contains(value, "deposit") || strings.Contains(value, "contribution"):
		return "deposit"
	case strings.Contains(value, "withdraw"):
		return "withdrawal"
	case strings.Contains(value, "fee") || strings.Contains(value, "commission"):
		return "fee"
	case strings.Contains(value, "transfer"):
		return "transfer"
	case strings.Contains(value, "split"):
		return "split"
	default:
		return "other"
	}
}
func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
func mathMax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
func splitCSV(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func upload(w http.ResponseWriter, r *http.Request, name string) (multipart.File, *multipart.FileHeader, bool) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		errorJSON(w, r, 400, "VALIDATION_ERROR", "multipart file is required")
		return nil, nil, false
	}
	file, header, err := r.FormFile(name)
	if err != nil {
		errorJSON(w, r, 400, "VALIDATION_ERROR", "file field is required")
		return nil, nil, false
	}
	return file, header, true
}
func cleanName(v string) string { return strings.NewReplacer("/", "-", "\\", "-", " ", "_").Replace(v) }
func decode(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		errorJSON(w, r, 400, "VALIDATION_ERROR", "invalid JSON body")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	httpapi.WriteJSON(w, status, payload)
}
func errorJSON(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	httpapi.Error(w, r, status, code, message)
}
func (a *app) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := a.allowedOrigin(origin)
		if allowed != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowed)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, Idempotency-Key")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, Idempotency-Replayed")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == "OPTIONS" {
			if origin != "" && allowed == "" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(204)
			return
		}
		if origin != "" && allowed == "" {
			errorJSON(w, r, http.StatusForbidden, "FORBIDDEN", "origin is not allowed")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *app) allowedOrigin(origin string) string {
	if a.cfg.AppEnv != "production" && (a.cfg.AllowedOrigins == "" || a.cfg.AllowedOrigins == "*") {
		return "*"
	}
	for _, allowed := range strings.Split(a.cfg.AllowedOrigins, ",") {
		if strings.TrimSpace(allowed) == origin {
			return origin
		}
	}
	return ""
}

func (a *app) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func (a *app) bodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			limit := a.cfg.MaxBodyBytes
			if limit <= 0 {
				limit = 1 << 20
			}
			if strings.Contains(r.URL.Path, "/webhooks/") {
				limit = 2 << 20
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}

func (a *app) requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = auth.NewID()
			r.Header.Set("X-Request-ID", requestID)
		}
		w.Header().Set("X-Request-ID", requestID)
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		latency := time.Since(start).Milliseconds()
		a.incrementMetric(func(m *opsMetrics) {
			m.HTTPRequestsTotal++
			m.HTTPRequestDurationMS += latency
			if recorder.status >= 400 {
				m.HTTPErrorsTotal++
			}
		})
		a.log.Info("http.request", map[string]interface{}{"request_id": requestID, "trace_id": observability.TraceID(r.Context()), "span_id": observability.SpanID(r.Context()), "method": r.Method, "path": r.URL.Path, "query": r.URL.RawQuery, "status": recorder.status, "bytes": recorder.bytes, "latency_ms": latency, "user_id": userID(r)})
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(payload []byte) (int, error) {
	n, err := r.ResponseWriter.Write(payload)
	r.bytes += n
	return n, err
}
func (a *app) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				a.log.Error("panic", map[string]interface{}{"error": fmt.Sprint(err)})
				errorJSON(w, r, 500, "INTERNAL_ERROR", "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func defaultProfile(uid string) models.AdvisorProfile {
	now := time.Now().UTC()
	return models.AdvisorProfile{ID: auth.NewID(), UserID: uid, Country: "CA", Age: 23, MonthlyIncomeEstimate: 4200, RiskTolerance: "moderate", EmergencyFundMonthsTarget: 6, CurrentEmergencyFund: 1800, HasHighInterestDebt: false, HasEmployerMatch: false, RetirementAccountAccess: "TFSA, FHSA, RRSP", PrimaryGoal: "Build emergency fund and invest consistently", InvestmentTimeHorizonYears: 30, CreatedAt: now, UpdatedAt: now}
}

func (a *app) ensureDefaultAccount(uid string) string {
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for _, acct := range a.store.accounts {
		if acct.UserID == uid {
			return acct.ID
		}
	}
	acct := models.BrokerageAccount{ID: auth.NewID(), UserID: uid, Provider: "manual", AccountName: "Demo TFSA", AccountType: "TFSA", Currency: "CAD", InstitutionName: "Manual Import", ConnectionStatus: "manual", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	a.store.accounts[acct.ID] = acct
	return acct.ID
}

func (a *app) seedDemo() models.User {
	a.store.mu.Lock()
	if id, ok := a.store.usersByEmail["demo@clearflow.dev"]; ok {
		u := a.store.users[id]
		a.store.mu.Unlock()
		return u
	}
	hash, _ := auth.HashPassword("demo-password")
	u := models.User{ID: auth.NewID(), Email: "demo@clearflow.dev", PasswordHash: hash, CreatedAt: time.Now().UTC()}
	a.store.users[u.ID] = u
	a.store.usersByEmail[u.Email] = u.ID
	a.store.profiles[u.ID] = defaultProfile(u.ID)
	a.store.mu.Unlock()
	accountID := a.ensureDefaultAccount(u.ID)
	txs, _ := parseTransactionsCSV(u.ID, demoTransactionsCSV())
	holdings, _ := parseHoldingsCSV(u.ID, demoHoldingsCSV(), accountID)
	ptxs, _ := parsePortfolioTransactionsCSV(u.ID, demoPortfolioTransactionsCSV(), accountID)
	a.store.mu.Lock()
	for _, t := range txs {
		a.store.transactions[t.ID] = t
	}
	for _, h := range holdings {
		a.store.holdings[h.ID] = h
	}
	for _, tx := range ptxs {
		a.store.portfolioTransactions[tx.ID] = tx
	}
	a.store.mu.Unlock()
	a.seedClearflowDemo(u.ID)
	return u
}

func demoTransactionsCSV() string {
	b, err := os.ReadFile("../../sample-data/sample_transactions.csv")
	if err == nil {
		return string(b)
	}
	return "date,description,merchant,amount,currency,category\n2026-04-01,Payroll deposit,Payroll,3200,USD,Income\n2026-04-02,Rent,Rent,-1200,USD,Housing\n2026-04-05,Netflix,Netflix,-18.99,USD,Subscriptions\n2026-05-05,Netflix,Netflix,-18.99,USD,Subscriptions\n2026-06-05,Netflix,Netflix,-18.99,USD,Subscriptions\n"
}
func demoHoldingsCSV() string {
	b, err := os.ReadFile("../../sample-data/sample_holdings.csv")
	if err == nil {
		return string(b)
	}
	return "account,account_type,symbol,name,security_type,quantity,average_cost,market_price,market_value,currency\nDemo TFSA,TFSA,VFV.TO,Vanguard S&P 500 Index ETF,etf,45,112,148.35,6675.75,CAD\n"
}
func demoPortfolioTransactionsCSV() string {
	b, err := os.ReadFile("../../sample-data/sample_portfolio_transactions.csv")
	if err == nil {
		return string(b)
	}
	return "date,account,symbol,action,quantity,price,amount,fees,currency,description\n2026-04-10,Demo TFSA,VFV.TO,buy,10,140,1400,0,CAD,Initial buy\n"
}

func demoPlaidInvestmentHoldingsCSV() string {
	return "ticker,security,asset_class,shares,cost_basis,current_price,current_value,ccy\nVOO,Vanguard S&P 500 ETF,ETF,12,4680,510,6120,USD\nBND,Vanguard Total Bond Market ETF,ETF,20,1440,73,1460,USD\nCASH,Cash,Money Market,850,850,1,850,USD\n"
}

func demoPlaidInvestmentTransactionsCSV() string {
	return "trade_date,activity,ticker,shares,trade_price,net_amount,commission,ccy,details\n2026-06-03,Contribution,CASH,0,0,1500,0,USD,Plaid investment cash contribution\n2026-06-04,Buy,VOO,2,500,1000,0,USD,Plaid investment buy\n2026-06-20,Dividend,VOO,0,0,18.42,0,USD,Plaid dividend\n"
}
