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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/advisor"
	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/config"
	"github.com/StephenShao90/Fynora/services/api/internal/logger"
	"github.com/StephenShao90/Fynora/services/api/internal/marketdata"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
	"github.com/StephenShao90/Fynora/services/api/internal/plaid"
	"github.com/StephenShao90/Fynora/services/api/internal/portfolio"
	"github.com/StephenShao90/Fynora/services/api/internal/storage"
	"github.com/StephenShao90/Fynora/services/api/internal/validation"
)

type app struct {
	cfg    config.Config
	log    logger.Logger
	store  *memoryStore
	raw    storage.RawEventStore
	market marketdata.Provider
	plaid  plaid.Client
}

type memoryStore struct {
	mu                    sync.RWMutex
	users                 map[string]models.User
	usersByEmail          map[string]string
	profiles              map[string]models.AdvisorProfile
	imports               map[string]models.RawImport
	transactions          map[string]models.Transaction
	accounts              map[string]models.BrokerageAccount
	holdings              map[string]models.Holding
	portfolioTransactions map[string]models.PortfolioTransaction
	plaidConnections      map[string]models.PlaidConnection
}

func newStore() *memoryStore {
	return &memoryStore{
		users: map[string]models.User{}, usersByEmail: map[string]string{},
		profiles: map[string]models.AdvisorProfile{}, imports: map[string]models.RawImport{},
		transactions: map[string]models.Transaction{}, accounts: map[string]models.BrokerageAccount{},
		holdings: map[string]models.Holding{}, portfolioTransactions: map[string]models.PortfolioTransaction{},
		plaidConnections: map[string]models.PlaidConnection{},
	}
}

func main() {
	cfg := config.Load()
	a := &app{
		cfg:    cfg,
		log:    logger.New(),
		store:  newStore(),
		raw:    storage.NewLocalStore(cfg.LocalStorageDir),
		market: marketdata.MockProvider{},
		plaid:  plaid.Client{ClientID: cfg.PlaidClientID, Secret: cfg.PlaidSecret, Env: cfg.PlaidEnv},
	}
	if err := a.loadPlaidConnections(); err != nil {
		a.log.Error("load_plaid_connections_failed", map[string]interface{}{"error": err.Error()})
	}
	mux := http.NewServeMux()
	a.routes(mux)
	handler := a.recover(a.requestLog(a.withCORS(mux)))
	log.Printf("Fynora API listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatal(err)
	}
}

func (a *app) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("POST /auth/register", a.register)
	mux.HandleFunc("POST /auth/login", a.login)
	mux.HandleFunc("POST /auth/demo-token", a.demoToken)
	mux.HandleFunc("GET /me", a.authed(a.me))
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
	mux.HandleFunc("POST /connections/plaid/link-token", a.authed(a.createPlaidLinkToken))
	mux.HandleFunc("POST /connections/plaid/exchange-public-token", a.authed(a.exchangePlaidPublicToken))
	mux.HandleFunc("POST /connections/plaid/sync-transactions", a.authed(a.syncPlaidTransactions))
	mux.HandleFunc("POST /portfolio/import/holdings-csv", a.authed(a.importHoldingsCSV))
	mux.HandleFunc("POST /portfolio/import/transactions-csv", a.authed(a.importPortfolioTransactionsCSV))
	mux.HandleFunc("POST /portfolio/holdings", a.authed(a.createHolding))
	mux.HandleFunc("GET /portfolio/holdings", a.authed(a.listHoldings))
	mux.HandleFunc("GET /portfolio/holdings/{id}", a.authed(a.getHolding))
	mux.HandleFunc("PATCH /portfolio/holdings/{id}", a.authed(a.patchHolding))
	mux.HandleFunc("DELETE /portfolio/holdings/{id}", a.authed(a.deleteHolding))
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

func (a *app) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "fynora-api"})
}

func (a *app) register(w http.ResponseWriter, r *http.Request) {
	var req struct{ Email, Password string }
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
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	if _, ok := a.store.usersByEmail[strings.ToLower(req.Email)]; ok {
		errorJSON(w, r, 400, "VALIDATION_ERROR", "email is already registered")
		return
	}
	u := models.User{ID: auth.NewID(), Email: strings.ToLower(req.Email), PasswordHash: hash, CreatedAt: time.Now().UTC()}
	a.store.users[u.ID] = u
	a.store.usersByEmail[u.Email] = u.ID
	a.store.profiles[u.ID] = defaultProfile(u.ID)
	token, _ := auth.SignJWT(a.cfg.JWTSecret, u.ID, u.Email, 24*time.Hour)
	writeJSON(w, 201, map[string]interface{}{"token": token, "user": u})
}

func (a *app) login(w http.ResponseWriter, r *http.Request) {
	var req struct{ Email, Password string }
	if !decode(w, r, &req) {
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
	token, _ := auth.SignJWT(a.cfg.JWTSecret, u.ID, u.Email, 24*time.Hour)
	writeJSON(w, 200, map[string]interface{}{"token": token, "user": u})
}

func (a *app) demoToken(w http.ResponseWriter, r *http.Request) {
	u := a.seedDemo()
	token, _ := auth.SignJWT(a.cfg.JWTSecret, u.ID, u.Email, 24*time.Hour)
	writeJSON(w, 200, map[string]interface{}{"token": token, "user": u})
}

func (a *app) me(w http.ResponseWriter, r *http.Request) {
	u, _ := a.currentUser(r)
	writeJSON(w, 200, u)
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
	a.store.mu.Lock()
	a.store.accounts[acct.ID] = acct
	a.store.mu.Unlock()
	writeJSON(w, 201, acct)
}
func (a *app) listAccounts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, a.accounts(userID(r)))
}
func (a *app) getAccount(w http.ResponseWriter, r *http.Request) {
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
	exchange, err := a.plaid.ExchangePublicToken(r.Context(), req.PublicToken)
	if err != nil {
		errorJSON(w, r, 502, "PLAID_ERROR", err.Error())
		return
	}
	item, err := a.plaid.GetItem(r.Context(), exchange.AccessToken)
	if err != nil {
		errorJSON(w, r, 502, "PLAID_ERROR", err.Error())
		return
	}
	ciphertext, err := a.encryptToken(exchange.AccessToken)
	if err != nil {
		errorJSON(w, r, 500, "INTERNAL", "could not secure Plaid token")
		return
	}
	now := time.Now().UTC()
	name := item.Institution.Name
	if name == "" {
		name = "Plaid institution"
	}
	conn := models.PlaidConnection{ID: auth.NewID(), UserID: userID(r), ItemID: exchange.ItemID, InstitutionName: name, AccessTokenCiphertext: ciphertext, Products: splitCSV(a.cfg.PlaidProducts), CreatedAt: now, UpdatedAt: now}
	a.store.mu.Lock()
	a.store.plaidConnections[conn.ID] = conn
	a.store.mu.Unlock()
	if err := a.persistPlaidConnections(); err != nil {
		errorJSON(w, r, 500, "INTERNAL", "could not persist Plaid connection")
		return
	}
	writeJSON(w, 201, map[string]interface{}{"connection": conn})
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
func (a *app) importHoldingsCSV(w http.ResponseWriter, r *http.Request) {
	file, header, ok := upload(w, r, "file")
	if !ok {
		return
	}
	defer file.Close()
	raw, _ := io.ReadAll(file)
	key := "holdings/" + auth.NewID() + "-" + cleanName(header.Filename)
	_ = a.raw.Put(r.Context(), key, raw)
	rows, failed := parseHoldingsCSV(userID(r), string(raw), a.ensureDefaultAccount(userID(r)))
	imp := models.RawImport{ID: auth.NewID(), UserID: userID(r), ImportType: "holdings", OriginalFilename: header.Filename, RawStorageKey: key, RowCount: len(rows) + failed, ImportedCount: len(rows), FailedCount: failed, CreatedAt: time.Now().UTC()}
	a.store.mu.Lock()
	a.store.imports[imp.ID] = imp
	for _, h := range rows {
		a.store.holdings[h.ID] = h
	}
	a.store.mu.Unlock()
	writeJSON(w, 201, map[string]interface{}{"import": imp, "holdings": rows})
}
func (a *app) importPortfolioTransactionsCSV(w http.ResponseWriter, r *http.Request) {
	file, header, ok := upload(w, r, "file")
	if !ok {
		return
	}
	defer file.Close()
	raw, _ := io.ReadAll(file)
	key := "portfolio-transactions/" + auth.NewID() + "-" + cleanName(header.Filename)
	_ = a.raw.Put(r.Context(), key, raw)
	rows, failed := parsePortfolioTransactionsCSV(userID(r), string(raw), a.ensureDefaultAccount(userID(r)))
	imp := models.RawImport{ID: auth.NewID(), UserID: userID(r), ImportType: "portfolio_transactions", OriginalFilename: header.Filename, RawStorageKey: key, RowCount: len(rows) + failed, ImportedCount: len(rows), FailedCount: failed, CreatedAt: time.Now().UTC()}
	a.store.mu.Lock()
	a.store.imports[imp.ID] = imp
	for _, row := range rows {
		row.ImportID = imp.ID
		a.store.portfolioTransactions[row.ID] = row
	}
	a.store.mu.Unlock()
	writeJSON(w, 201, map[string]interface{}{"import": imp, "portfolio_transactions": rows})
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
		h.BrokerageAccountID = a.ensureDefaultAccount(userID(r))
	}
	h = portfolio.PriceHoldings(r.Context(), a.market, []models.Holding{h})[0]
	a.store.mu.Lock()
	a.store.holdings[h.ID] = h
	a.store.mu.Unlock()
	writeJSON(w, 201, h)
}
func (a *app) listHoldings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, portfolio.PriceHoldings(r.Context(), a.market, a.holdings(userID(r))))
}
func (a *app) getHolding(w http.ResponseWriter, r *http.Request) {
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
	a.store.mu.Lock()
	delete(a.store.holdings, r.PathValue("id"))
	a.store.mu.Unlock()
	w.WriteHeader(204)
}
func (a *app) portfolioSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, a.summary(userID(r)))
}
func (a *app) portfolioAllocation(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	writeJSON(w, 200, portfolio.BuildAllocation(a.holdings(uid), a.accounts(uid)))
}
func (a *app) portfolioPerformance(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]interface{}{"summary": a.summary(userID(r)), "cash_flows": a.portfolioTxs(userID(r)), "method": "simple unrealized return plus transaction cash-flow history"})
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
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	u, ok := a.store.users[userID(r)]
	return u, ok
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
func parseHoldingsCSV(uid, raw, accountID string) ([]models.Holding, int) {
	records, err := csv.NewReader(strings.NewReader(raw)).ReadAll()
	if err != nil || len(records) < 2 {
		return nil, 1
	}
	idx := headerIndex(records[0])
	var out []models.Holding
	failed := 0
	for _, row := range records[1:] {
		qty, err := strconv.ParseFloat(cell(row, idx, "quantity"), 64)
		if err != nil {
			failed++
			continue
		}
		avg := parseFloat(cell(row, idx, "average_cost"))
		mv := parseFloat(cell(row, idx, "market_value"))
		price := parseFloat(fallback(cell(row, idx, "market_price"), cell(row, idx, "last_price")))
		h := models.Holding{ID: auth.NewID(), UserID: uid, BrokerageAccountID: accountID, Symbol: strings.ToUpper(cell(row, idx, "symbol")), SecurityName: fallback(cell(row, idx, "name"), cell(row, idx, "security_name")), SecurityType: fallback(cell(row, idx, "security_type"), "etf"), Quantity: qty, AverageCost: avg, Currency: fallback(cell(row, idx, "currency"), "USD"), MarketValue: mv, LastPrice: price, PriceAsOf: time.Now().UTC(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
		if h.MarketValue == 0 && h.LastPrice > 0 {
			h.MarketValue = h.Quantity * h.LastPrice
		}
		out = append(out, h)
	}
	return out, failed
}
func parsePortfolioTransactionsCSV(uid, raw, accountID string) ([]models.PortfolioTransaction, int) {
	records, err := csv.NewReader(strings.NewReader(raw)).ReadAll()
	if err != nil || len(records) < 2 {
		return nil, 1
	}
	idx := headerIndex(records[0])
	var out []models.PortfolioTransaction
	failed := 0
	for _, row := range records[1:] {
		date, err := parseDate(cell(row, idx, "date"))
		if err != nil {
			failed++
			continue
		}
		out = append(out, models.PortfolioTransaction{ID: auth.NewID(), UserID: uid, BrokerageAccountID: accountID, Symbol: strings.ToUpper(cell(row, idx, "symbol")), TransactionType: strings.ToLower(fallback(cell(row, idx, "action"), cell(row, idx, "transaction_type"))), Quantity: parseFloat(cell(row, idx, "quantity")), Price: parseFloat(cell(row, idx, "price")), Amount: parseFloat(cell(row, idx, "amount")), Fees: parseFloat(cell(row, idx, "fees")), Currency: fallback(cell(row, idx, "currency"), "USD"), OccurredAt: date, Description: cell(row, idx, "description"), CreatedAt: time.Now().UTC()})
	}
	return out, failed
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
		m[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return m
}
func cell(row []string, idx map[string]int, key string) string {
	if i, ok := idx[key]; ok && i < len(row) {
		return strings.TrimSpace(row[i])
	}
	return ""
}
func parseDate(v string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02", "01/02/2006", "2006/01/02", time.RFC3339} {
		if t, err := time.Parse(layout, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid date")
}
func parseFloat(v string) float64 {
	f, _ := strconv.ParseFloat(strings.ReplaceAll(v, "$", ""), 64)
	return f
}
func fallback(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return b
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
func errorJSON(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, map[string]interface{}{"error": map[string]string{"code": code, "message": message, "request_id": r.Header.Get("X-Request-ID")}})
}
func (a *app) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
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
		next.ServeHTTP(w, r)
		a.log.Info("request", map[string]interface{}{"request_id": requestID, "method": r.Method, "path": r.URL.Path, "latency_ms": time.Since(start).Milliseconds(), "user_id": userID(r)})
	})
}
func (a *app) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				a.log.Error("panic", map[string]interface{}{"error": fmt.Sprint(err)})
				errorJSON(w, r, 500, "INTERNAL", "internal server error")
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
	if id, ok := a.store.usersByEmail["demo@fynora.dev"]; ok {
		u := a.store.users[id]
		a.store.mu.Unlock()
		return u
	}
	hash, _ := auth.HashPassword("demo-password")
	u := models.User{ID: auth.NewID(), Email: "demo@fynora.dev", PasswordHash: hash, CreatedAt: time.Now().UTC()}
	a.store.users[u.ID] = u
	a.store.usersByEmail[u.Email] = u.ID
	a.store.profiles[u.ID] = defaultProfile(u.ID)
	a.store.mu.Unlock()
	accountID := a.ensureDefaultAccount(u.ID)
	txs, _ := parseTransactionsCSV(u.ID, demoTransactionsCSV())
	holdings, _ := parseHoldingsCSV(u.ID, demoHoldingsCSV(), accountID)
	ptxs, _ := parsePortfolioTransactionsCSV(u.ID, demoPortfolioTransactionsCSV(), accountID)
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for _, t := range txs {
		a.store.transactions[t.ID] = t
	}
	for _, h := range holdings {
		a.store.holdings[h.ID] = h
	}
	for _, tx := range ptxs {
		a.store.portfolioTransactions[tx.ID] = tx
	}
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
