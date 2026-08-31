package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/config"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
	"github.com/StephenShao90/Fynora/services/api/internal/processors"
)

func TestAPIV1PlaidWebhookPersistsDedupesAndQueuesJob(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "webhook-owner@example.com")
	orgID := owner.Organizations[0].ID
	if err := a.storePlaidItemForTest(orgID, "item_test"); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"webhook_type":"TRANSACTIONS","webhook_code":"SYNC_UPDATES_AVAILABLE","item_id":"item_test","environment":"sandbox"}`)
	first := performAPIRequest(a, http.MethodPost, "/api/v1/webhooks/plaid", "", "req_plaid_hook_1", body)
	if first.Code != http.StatusAccepted {
		t.Fatalf("expected webhook 202, got %d: %s", first.Code, first.Body.String())
	}
	second := performAPIRequest(a, http.MethodPost, "/api/v1/webhooks/plaid", "", "req_plaid_hook_2", body)
	if second.Code != http.StatusAccepted {
		t.Fatalf("expected duplicate webhook 202, got %d: %s", second.Code, second.Body.String())
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	if len(a.store.plaidWebhookEvents) != 1 {
		t.Fatalf("expected one webhook event, got %d", len(a.store.plaidWebhookEvents))
	}
	jobs := 0
	for _, job := range a.store.jobs {
		if job.OrganizationID == orgID && job.Type == "plaid.transactions.sync" {
			jobs++
		}
	}
	if jobs != 1 {
		t.Fatalf("expected one Plaid sync job, got %d", jobs)
	}
	if len(a.store.outboxEvents) == 0 {
		t.Fatal("expected outbox event")
	}
}

func TestAPIV1PlaidWebhookUnsupportedCodePersistsWithoutQueue(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "plaid-unsupported@example.com")
	orgID := owner.Organizations[0].ID
	if err := a.storePlaidItemForTest(orgID, "item_unsupported"); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"webhook_type":"TRANSACTIONS","webhook_code":"UNKNOWN_CODE","item_id":"item_unsupported","environment":"sandbox"}`)
	rec := performAPIRequest(a, http.MethodPost, "/api/v1/webhooks/plaid", "", "req_plaid_unknown_code", body)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected webhook 202, got %d: %s", rec.Code, rec.Body.String())
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	if len(a.store.plaidWebhookEvents) != 1 {
		t.Fatalf("expected persisted webhook, got %d", len(a.store.plaidWebhookEvents))
	}
	for _, job := range a.store.jobs {
		if job.Type == "plaid.transactions.sync" {
			t.Fatalf("did not expect sync job for unsupported code: %#v", job)
		}
	}
}

func TestAPIV1ProcessorWebhookPersistsAndDedupes(t *testing.T) {
	a := newAPITestApp(t)
	body := []byte(`{"id":"evt_1","type":"payout.paid"}`)
	first := performAPIRequest(a, http.MethodPost, "/api/v1/webhooks/processors/stripe", "", "req_processor_hook_1", body)
	if first.Code != http.StatusAccepted {
		t.Fatalf("expected processor webhook 202, got %d: %s", first.Code, first.Body.String())
	}
	second := performAPIRequest(a, http.MethodPost, "/api/v1/webhooks/processors/stripe", "", "req_processor_hook_2", body)
	if second.Code != http.StatusAccepted {
		t.Fatalf("expected duplicate processor webhook 202, got %d: %s", second.Code, second.Body.String())
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	if len(a.store.processorWebhookEvents) != 1 {
		t.Fatalf("expected one processor webhook event, got %d", len(a.store.processorWebhookEvents))
	}
}

func TestAPIV1MetricsSecurityHeadersAndBodyLimit(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "metrics@example.com")
	rec := performAPIRequest(a, http.MethodGet, "/api/v1/ops/metrics", owner.Token, "req_metrics", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected metrics 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("expected security headers, got %#v", rec.Header())
	}
	var payload struct {
		HTTPRequestsTotal int64 `json:"http_requests_total"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.HTTPRequestsTotal == 0 {
		t.Fatalf("expected request metrics, got %#v", payload)
	}
	a.cfg.MaxBodyBytes = 8
	tooLarge := performAPIRequest(a, http.MethodPost, "/api/v1/auth/login", "", "req_body_limit", []byte(`{"email":"too-large@example.com","password":"password123"}`))
	if tooLarge.Code != http.StatusBadRequest && tooLarge.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected oversized body rejection, got %d: %s", tooLarge.Code, tooLarge.Body.String())
	}
}

func TestAPIV1JobCancelRetryAndDeadInspection(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "ops-owner@example.com")
	orgID := owner.Organizations[0].ID
	job, err := a.enqueueJob(nil, orgID, owner.User.ID, "bank.sync", "{}")
	if err != nil {
		t.Fatal(err)
	}
	cancel := performAPIRequest(a, http.MethodPost, "/api/v1/jobs/"+job.ID+"/cancel?organizationId="+orgID, owner.Token, "req_cancel_job", nil)
	if cancel.Code != http.StatusOK {
		t.Fatalf("expected cancel 200, got %d: %s", cancel.Code, cancel.Body.String())
	}
	retry := performAPIRequest(a, http.MethodPost, "/api/v1/jobs/"+job.ID+"/retry?organizationId="+orgID, owner.Token, "req_retry_job", nil)
	if retry.Code != http.StatusOK {
		t.Fatalf("expected retry 200, got %d: %s", retry.Code, retry.Body.String())
	}
	a.store.mu.Lock()
	job = a.store.jobs[job.ID]
	job.Status = "dead"
	a.store.jobs[job.ID] = job
	a.store.mu.Unlock()
	dead := performAPIRequest(a, http.MethodGet, "/api/v1/jobs/dead?organizationId="+orgID, owner.Token, "req_dead_jobs", nil)
	if dead.Code != http.StatusOK {
		t.Fatalf("expected dead jobs 200, got %d: %s", dead.Code, dead.Body.String())
	}
}

func TestProductionConfigRejectsUnsafeValues(t *testing.T) {
	cfg := config.Config{AppEnv: "production", JWTSecret: "dev-secret", DatabaseURL: "postgres://example", AllowedOrigins: "*"}
	if err := cfg.ValidateProduction(); err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected JWT_SECRET production error, got %v", err)
	}
	cfg.JWTSecret = "real-secret-with-at-least-thirty-two-bytes"
	if err := cfg.ValidateProduction(); err == nil || !strings.Contains(err.Error(), "ALLOWED_ORIGINS") {
		t.Fatalf("expected ALLOWED_ORIGINS production error, got %v", err)
	}
	cfg.AllowedOrigins = "https://app.example.com"
	cfg.EnableDemoAuth = "true"
	if err := cfg.ValidateProduction(); err == nil || !strings.Contains(err.Error(), "ENABLE_DEMO_AUTH") {
		t.Fatalf("expected demo auth production error, got %v", err)
	}
	cfg.EnableDemoAuth = "false"
	if err := cfg.ValidateProduction(); err == nil || !strings.Contains(err.Error(), "PLAID_CLIENT_ID") {
		t.Fatalf("expected Plaid production error, got %v", err)
	}
	cfg.PlaidClientID = "plaid-client"
	cfg.PlaidSecret = "plaid-secret"
	cfg.PlaidEnv = "production"
	cfg.PlaidWebhookVerification = "true"
	if err := cfg.ValidateProduction(); err == nil || !strings.Contains(err.Error(), "STRIPE_CLIENT_ID") {
		t.Fatalf("expected Stripe production error, got %v", err)
	}
	cfg.StripeClientID = "ca_live_test"
	cfg.StripeSecretKey = "stripe-secret-fixture"
	cfg.StripeWebhookSecret = "stripe-webhook-secret-fixture"
	cfg.StripeRedirectURL = "https://api.example.com/api/v1/integrations/stripe/callback"
	cfg.FrontendURL = "https://app.example.com"
	if err := cfg.ValidateProduction(); err == nil || !strings.Contains(err.Error(), "PROVIDER_TOKEN_ENCRYPTION_KEY") {
		t.Fatalf("expected provider token encryption production error, got %v", err)
	}
	cfg.ProviderTokenEncryptionKey = "test-key-with-at-least-thirty-two-bytes"
	cfg.RedisEnabled = "true"
	cfg.RedisURL = ""
	if err := cfg.ValidateProduction(); err == nil || !strings.Contains(err.Error(), "REDIS_URL") {
		t.Fatalf("expected REDIS_URL production error, got %v", err)
	}
}

func TestProductionConfigAcceptsCustomerReadyValues(t *testing.T) {
	cfg := config.Config{
		AppEnv:                     "production",
		JWTSecret:                  "real-secret-with-at-least-thirty-two-bytes",
		DatabaseURL:                "postgres://example",
		AllowedOrigins:             "https://app.example.com",
		EnableDemoAuth:             "false",
		PlaidClientID:              "plaid-client",
		PlaidSecret:                "plaid-secret",
		PlaidEnv:                   "production",
		PlaidWebhookVerification:   "true",
		StripeClientID:             "ca_live_test",
		StripeSecretKey:            "stripe-secret-fixture",
		StripeWebhookSecret:        "stripe-webhook-secret-fixture",
		StripeRedirectURL:          "https://api.example.com/api/v1/integrations/stripe/callback",
		FrontendURL:                "https://app.example.com",
		ProviderTokenEncryptionKey: "test-key-with-at-least-thirty-two-bytes",
		RedisEnabled:               "false",
	}
	if err := cfg.ValidateProduction(); err != nil {
		t.Fatalf("expected production config to pass, got %v", err)
	}
}

func TestProductionCORSRejectsDisallowedOrigin(t *testing.T) {
	a := newAPITestApp(t)
	a.cfg.AppEnv = "production"
	a.cfg.AllowedOrigins = "https://app.example.com"
	rec := performAPIRequestWithHeaders(a, http.MethodGet, "/api/v1/health", "", "req_cors", nil, map[string]string{"Origin": "https://evil.example.com"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected disallowed origin 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProductionRejectsUnsignedMockProcessorWebhook(t *testing.T) {
	a := newAPITestApp(t)
	a.cfg.AppEnv = "production"
	body := []byte(`{"id":"evt_mock_1","type":"payout.paid"}`)
	rec := performAPIRequest(a, http.MethodPost, "/api/v1/webhooks/processors/mock?organizationId=00000000-0000-0000-0000-000000000001", "", "req_prod_mock_webhook", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected production mock webhook 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProductionStripeWebhookRequiresOrganization(t *testing.T) {
	a := newAPITestApp(t)
	a.cfg.AppEnv = "production"
	a.cfg.StripeWebhookSecret = "stripe-webhook-secret-fixture"
	body := []byte(`{"id":"evt_missing_org","type":"payout.paid"}`)
	headers := map[string]string{"Stripe-Signature": processors.BuildStripeTestSignature("stripe-webhook-secret-fixture", time.Now(), body)}
	rec := performAPIRequestWithHeaders(a, http.MethodPost, "/api/v1/webhooks/processors/stripe", "", "req_prod_stripe_missing_org", body, headers)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected missing organization 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebhookBodyLimitRejectsOversizedBody(t *testing.T) {
	req := httptestRequest(http.MethodPost, "/api/v1/webhooks/processors/stripe", "")
	req.Body = io.NopCloser(strings.NewReader(strings.Repeat("a", (2<<20)+1)))
	if _, err := readLimitedBody(req, 2<<20); err == nil {
		t.Fatal("expected oversized webhook body error")
	}
}

func (a *app) storePlaidItemForTest(orgID, itemID string) error {
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	a.store.plaidConnections[itemID] = models.PlaidConnection{ID: itemID, UserID: "test", ItemID: itemID, InstitutionName: "Test Bank"}
	a.store.plaidItemOrganizations[itemID] = orgID
	return nil
}
