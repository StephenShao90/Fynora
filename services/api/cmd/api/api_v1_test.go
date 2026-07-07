package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/StephenShao90/Fynora/services/api/internal/config"
	"github.com/StephenShao90/Fynora/services/api/internal/logger"
	"github.com/StephenShao90/Fynora/services/api/internal/marketdata"
	"github.com/StephenShao90/Fynora/services/api/internal/plaid"
	"github.com/StephenShao90/Fynora/services/api/internal/storage"
)

func newAPITestApp(t *testing.T) *app {
	t.Helper()
	return &app{
		cfg:    config.Config{JWTSecret: "test-secret"},
		log:    logger.Logger{},
		store:  newStore(),
		raw:    storage.NewLocalStore(t.TempDir()),
		market: marketdata.MockProvider{},
		plaid:  plaid.Client{Env: "sandbox"},
	}
}

func performAPIRequest(a *app, method, target, token, requestID string, body []byte) *httptest.ResponseRecorder {
	return performAPIRequestWithHeaders(a, method, target, token, requestID, body, nil)
}

func performAPIRequestWithHeaders(a *app, method, target, token, requestID string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	a.routes(mux)
	handler := a.recover(a.requestLog(a.securityHeaders(a.bodyLimit(a.withCORS(mux)))))
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func httptestRequest(method, target, token string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("X-Request-ID", "req_test")
	return req
}

func demoTokenForTest(t *testing.T, a *app) string {
	t.Helper()
	rec := performAPIRequest(a, http.MethodPost, "/api/v1/auth/demo-token", "", "req_demo", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected demo token status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Token == "" {
		t.Fatal("expected demo token")
	}
	return payload.Token
}

func TestAPIV1Health(t *testing.T) {
	a := newAPITestApp(t)
	rec := performAPIRequest(a, http.MethodGet, "/api/v1/health", "", "req_health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Request-ID"); got != "req_health" {
		t.Fatalf("expected X-Request-ID req_health, got %q", got)
	}
}

func TestAPIV1ReadyReturnsJSON(t *testing.T) {
	a := newAPITestApp(t)
	rec := performAPIRequest(a, http.MethodGet, "/api/v1/ready", "", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if payload["status"] == "" {
		t.Fatalf("expected readiness status, got %#v", payload)
	}
}

func TestAPIV1DemoTokenWorks(t *testing.T) {
	a := newAPITestApp(t)
	_ = demoTokenForTest(t, a)
}

func TestAPIV1PaymentsRejectsMissingAuth(t *testing.T) {
	a := newAPITestApp(t)
	rec := performAPIRequest(a, http.MethodGet, "/api/v1/payments", "", "req_no_auth", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Error struct {
			Code      string `json:"code"`
			RequestID string `json:"requestId"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected UNAUTHORIZED, got %#v", payload)
	}
	if payload.Error.RequestID != "req_no_auth" {
		t.Fatalf("expected requestId req_no_auth, got %q", payload.Error.RequestID)
	}
}

func TestAPIV1PaymentsRejectsInvalidLimit(t *testing.T) {
	a := newAPITestApp(t)
	token := demoTokenForTest(t, a)
	rec := performAPIRequest(a, http.MethodGet, "/api/v1/payments?limit=999", token, "req_bad_limit", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Error struct {
			Code      string `json:"code"`
			RequestID string `json:"requestId"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %#v", payload)
	}
	if payload.Error.RequestID != "req_bad_limit" {
		t.Fatalf("expected requestId req_bad_limit, got %q", payload.Error.RequestID)
	}
}

func TestAPIV1PaymentsValidPagination(t *testing.T) {
	a := newAPITestApp(t)
	token := demoTokenForTest(t, a)
	rec := performAPIRequest(a, http.MethodGet, "/api/v1/payments?limit=10&offset=0", token, "req_valid_page", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Data       []map[string]interface{} `json:"data"`
		Pagination struct {
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		} `json:"pagination"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Pagination.Limit != 10 || payload.Pagination.Offset != 0 {
		t.Fatalf("unexpected pagination: %#v", payload.Pagination)
	}
	if len(payload.Data) > 10 {
		t.Fatalf("expected at most 10 payments, got %d", len(payload.Data))
	}
}

func TestLegacyPaymentsStillReturnsArray(t *testing.T) {
	a := newAPITestApp(t)
	token := demoTokenForTest(t, a)
	rec := performAPIRequest(a, http.MethodGet, "/payments?limit=10&offset=0", token, "req_legacy", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("expected legacy array response: %v", err)
	}
	if len(payload) > 10 {
		t.Fatalf("expected at most 10 payments, got %d", len(payload))
	}
}
