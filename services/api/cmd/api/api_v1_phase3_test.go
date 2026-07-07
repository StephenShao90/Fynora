package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAPIV1RefreshTokenRotationAndLogout(t *testing.T) {
	a := newAPITestApp(t)
	created := registerAPITestUser(t, a, "sessions@example.com")
	var first authTestResponse
	login := performAPIRequest(a, http.MethodPost, "/api/v1/auth/login", "", "req_login_session", []byte(`{"email":"sessions@example.com","password":"password123"}`))
	if login.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d: %s", login.Code, login.Body.String())
	}
	if err := json.NewDecoder(login.Body).Decode(&first); err != nil {
		t.Fatal(err)
	}
	if first.AccessToken == "" || first.RefreshToken == "" || first.Token == "" {
		t.Fatalf("expected access and refresh tokens: %#v", first)
	}
	refreshBody := []byte(`{"refreshToken":"` + first.RefreshToken + `"}`)
	refresh := performAPIRequestWithHeaders(a, http.MethodPost, "/api/v1/auth/refresh", "", "req_refresh", refreshBody, map[string]string{"X-Forwarded-For": "10.0.0.2"})
	if refresh.Code != http.StatusOK {
		t.Fatalf("expected refresh 200, got %d: %s", refresh.Code, refresh.Body.String())
	}
	var rotated struct {
		RefreshToken string `json:"refreshToken"`
		AccessToken  string `json:"accessToken"`
	}
	if err := json.NewDecoder(refresh.Body).Decode(&rotated); err != nil {
		t.Fatal(err)
	}
	if rotated.RefreshToken == "" || rotated.RefreshToken == first.RefreshToken || rotated.AccessToken == "" {
		t.Fatalf("expected rotated refresh token: %#v", rotated)
	}
	reuse := performAPIRequestWithHeaders(a, http.MethodPost, "/api/v1/auth/refresh", "", "req_refresh_reuse", refreshBody, map[string]string{"X-Forwarded-For": "10.0.0.3"})
	if reuse.Code != http.StatusUnauthorized {
		t.Fatalf("expected old refresh token to fail, got %d: %s", reuse.Code, reuse.Body.String())
	}
	logout := performAPIRequestWithHeaders(a, http.MethodPost, "/api/v1/auth/logout", "", "req_logout", []byte(`{"refreshToken":"`+rotated.RefreshToken+`"}`), map[string]string{"X-Forwarded-For": "10.0.0.4"})
	if logout.Code != http.StatusNoContent {
		t.Fatalf("expected logout 204, got %d: %s", logout.Code, logout.Body.String())
	}
	revoked := performAPIRequestWithHeaders(a, http.MethodPost, "/api/v1/auth/refresh", "", "req_revoked", []byte(`{"refreshToken":"`+rotated.RefreshToken+`"}`), map[string]string{"X-Forwarded-For": "10.0.0.5"})
	if revoked.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked refresh token to fail, got %d: %s", revoked.Code, revoked.Body.String())
	}
	sessions := performAPIRequest(a, http.MethodGet, "/api/v1/auth/sessions", created.Token, "req_sessions", nil)
	if sessions.Code != http.StatusOK {
		t.Fatalf("expected sessions 200, got %d: %s", sessions.Code, sessions.Body.String())
	}
}

func TestAPIV1AuthRateLimit(t *testing.T) {
	a := newAPITestApp(t)
	for i := 0; i < 5; i++ {
		rec := performAPIRequest(a, http.MethodPost, "/api/v1/auth/login", "", "req_rate", []byte(`{"email":"nobody@example.com","password":"password123"}`))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected auth failure before limit, got %d", rec.Code)
		}
	}
	limited := performAPIRequest(a, http.MethodPost, "/api/v1/auth/login", "", "req_rate_limited", []byte(`{"email":"nobody@example.com","password":"password123"}`))
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", limited.Code, limited.Body.String())
	}
	if limited.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
}

func TestAPIV1IdempotentJobEnqueueAndAuditLogs(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "jobs@example.com")
	orgID := owner.Organizations[0].ID
	target := "/api/v1/reconciliation-runs?organizationId=" + orgID
	first := performAPIRequestWithHeaders(a, http.MethodPost, target, owner.Token, "req_job_1", []byte(`{"source":"manual"}`), map[string]string{"Idempotency-Key": "job-key-1"})
	if first.Code != http.StatusAccepted {
		t.Fatalf("expected job enqueue 202, got %d: %s", first.Code, first.Body.String())
	}
	var payload struct {
		JobID  string `json:"jobId"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(first.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	second := performAPIRequestWithHeaders(a, http.MethodPost, target, owner.Token, "req_job_2", []byte(`{"source":"manual"}`), map[string]string{"Idempotency-Key": "job-key-1"})
	if second.Code != http.StatusAccepted || second.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatalf("expected idempotency replay 202, got %d headers=%v body=%s", second.Code, second.Header(), second.Body.String())
	}
	conflict := performAPIRequestWithHeaders(a, http.MethodPost, target, owner.Token, "req_job_3", []byte(`{"source":"different"}`), map[string]string{"Idempotency-Key": "job-key-1"})
	if conflict.Code != http.StatusConflict {
		t.Fatalf("expected idempotency conflict 409, got %d: %s", conflict.Code, conflict.Body.String())
	}
	jobs := performAPIRequest(a, http.MethodGet, "/api/v1/jobs?organizationId="+orgID, owner.Token, "req_jobs", nil)
	if jobs.Code != http.StatusOK {
		t.Fatalf("expected jobs 200, got %d: %s", jobs.Code, jobs.Body.String())
	}
	oneJob := performAPIRequest(a, http.MethodGet, "/api/v1/jobs/"+payload.JobID+"?organizationId="+orgID, owner.Token, "req_job_get", nil)
	if oneJob.Code != http.StatusOK {
		t.Fatalf("expected job get 200, got %d: %s", oneJob.Code, oneJob.Body.String())
	}
	a.writeAudit(nil, httptestRequest("GET", "/audit", owner.Token), orgID, owner.User.ID, "test.audit", "job", payload.JobID, "{}")
	audit := performAPIRequest(a, http.MethodGet, "/api/v1/audit-logs?organizationId="+orgID, owner.Token, "req_audit", nil)
	if audit.Code != http.StatusOK {
		t.Fatalf("expected audit logs 200, got %d: %s", audit.Code, audit.Body.String())
	}
}

func TestAPIV1ViewerCannotReadAuditLogs(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "audit-owner@example.com")
	viewer := registerAPITestUser(t, a, "audit-viewer@example.com")
	orgID := owner.Organizations[0].ID
	add := performAPIRequest(a, http.MethodPost, "/api/v1/organizations/"+orgID+"/members", owner.Token, "req_add_viewer", []byte(`{"email":"audit-viewer@example.com","role":"viewer"}`))
	if add.Code != http.StatusCreated {
		t.Fatalf("expected add viewer 201, got %d: %s", add.Code, add.Body.String())
	}
	audit := performAPIRequest(a, http.MethodGet, "/api/v1/audit-logs?organizationId="+orgID, viewer.Token, "req_viewer_audit", nil)
	if audit.Code != http.StatusForbidden {
		t.Fatalf("expected viewer audit 403, got %d: %s", audit.Code, audit.Body.String())
	}
}
