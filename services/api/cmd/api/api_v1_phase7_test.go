package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/models"
	"github.com/StephenShao90/Fynora/services/api/internal/processors"
)

func TestAPIV1StripeOAuthConnectCallbackStatusAndDisconnect(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "stripe-owner@example.com")
	orgID := owner.Organizations[0].ID

	connect := performAPIRequest(a, http.MethodGet, "/api/v1/integrations/stripe/connect-url?organizationId="+orgID, owner.Token, "req_stripe_connect_url", nil)
	if connect.Code != http.StatusOK {
		t.Fatalf("expected connect-url 200, got %d: %s", connect.Code, connect.Body.String())
	}
	var connectPayload struct {
		URL   string `json:"url"`
		State string `json:"state"`
	}
	if err := json.NewDecoder(connect.Body).Decode(&connectPayload); err != nil {
		t.Fatal(err)
	}
	if connectPayload.State == "" || !strings.Contains(connectPayload.URL, "connect.stripe.com/oauth/authorize") {
		t.Fatalf("unexpected connect payload: %#v", connectPayload)
	}

	callback := performAPIRequest(a, http.MethodGet, "/api/v1/integrations/stripe/callback?code=ac_test_123&state="+connectPayload.State, "", "req_stripe_callback", nil)
	if callback.Code != http.StatusSeeOther {
		t.Fatalf("expected callback redirect, got %d: %s", callback.Code, callback.Body.String())
	}
	if location := callback.Header().Get("Location"); !strings.Contains(location, "/integrations?") || !strings.Contains(location, "stripe=connected") {
		t.Fatalf("expected redirect back to integrations, got %q", location)
	}
	var status struct {
		Connected bool   `json:"connected"`
		AccountID string `json:"accountId"`
	}
	statusRec := performAPIRequest(a, http.MethodGet, "/api/v1/integrations/stripe/status?organizationId="+orgID, owner.Token, "req_stripe_status", nil)
	if statusRec.Code != http.StatusOK || strings.Contains(statusRec.Body.String(), "sk_mock") {
		t.Fatalf("unexpected status response %d: %s", statusRec.Code, statusRec.Body.String())
	}
	if err := json.NewDecoder(statusRec.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if !status.Connected || status.AccountID == "" {
		t.Fatalf("expected connected status without tokens, got %#v body=%s", status, statusRec.Body.String())
	}

	reuse := performAPIRequest(a, http.MethodGet, "/api/v1/integrations/stripe/callback?code=ac_test_456&state="+connectPayload.State, "", "req_stripe_callback_reuse", nil)
	if reuse.Code != http.StatusSeeOther || !strings.Contains(reuse.Header().Get("Location"), "stripe=error") {
		t.Fatalf("expected reused state error redirect, got %d location=%q body=%s", reuse.Code, reuse.Header().Get("Location"), reuse.Body.String())
	}

	disconnect := performAPIRequest(a, http.MethodDelete, "/api/v1/integrations/stripe?organizationId="+orgID, owner.Token, "req_stripe_disconnect", nil)
	if disconnect.Code != http.StatusOK {
		t.Fatalf("expected disconnect 200, got %d: %s", disconnect.Code, disconnect.Body.String())
	}
}

func TestAPIV1StripeOAuthExpiredStateRejected(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "stripe-expired@example.com")
	orgID := owner.Organizations[0].ID
	state := "expired_state"
	if err := a.saveOAuthState(nil, oauthStateForTest(orgID, owner.User.ID, state, time.Now().UTC().Add(-time.Minute))); err != nil {
		t.Fatal(err)
	}
	rec := performAPIRequest(a, http.MethodGet, "/api/v1/integrations/stripe/callback?code=ac_test&state="+state, "", "req_stripe_expired", nil)
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "stripe=error") {
		t.Fatalf("expected expired state error redirect, got %d location=%q body=%s", rec.Code, rec.Header().Get("Location"), rec.Body.String())
	}
}

func TestAPIV1StripeOAuthCallbackRejectsOrganizationMismatch(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "stripe-mismatch@example.com")
	orgID := owner.Organizations[0].ID
	state := "mismatch_state"
	if err := a.saveOAuthState(nil, oauthStateForTest(orgID, owner.User.ID, state, time.Now().UTC().Add(time.Minute))); err != nil {
		t.Fatal(err)
	}
	rec := performAPIRequest(a, http.MethodGet, "/api/v1/integrations/stripe/callback?organizationId=00000000-0000-0000-0000-000000000000&code=ac_test&state="+state, "", "req_stripe_mismatch", nil)
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "stripe=error") {
		t.Fatalf("expected mismatched organization error redirect, got %d location=%q body=%s", rec.Code, rec.Header().Get("Location"), rec.Body.String())
	}
}

func TestAPIV1StripeWebhookSignatureAndDedupe(t *testing.T) {
	a := newAPITestApp(t)
	a.cfg.StripeWebhookSecret = "whsec_test"
	body := []byte(`{"id":"evt_sig_1","type":"payout.paid"}`)
	headers := map[string]string{
		"Stripe-Signature": processors.BuildStripeTestSignature("whsec_test", time.Now(), body),
	}
	first := performAPIRequestWithHeaders(a, http.MethodPost, "/api/v1/webhooks/processors/stripe?organizationId=00000000-0000-0000-0000-000000000001", "", "req_stripe_hook_1", body, headers)
	if first.Code != http.StatusAccepted {
		t.Fatalf("expected valid webhook 202, got %d: %s", first.Code, first.Body.String())
	}
	second := performAPIRequestWithHeaders(a, http.MethodPost, "/api/v1/webhooks/processors/stripe?organizationId=00000000-0000-0000-0000-000000000001", "", "req_stripe_hook_2", body, headers)
	if second.Code != http.StatusAccepted {
		t.Fatalf("expected duplicate webhook 202, got %d: %s", second.Code, second.Body.String())
	}
	bad := performAPIRequestWithHeaders(a, http.MethodPost, "/api/v1/webhooks/processors/stripe", "", "req_stripe_hook_bad", body, map[string]string{"Stripe-Signature": "t=1,v1=bad"})
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid signature rejection, got %d: %s", bad.Code, bad.Body.String())
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	if len(a.store.processorWebhookEvents) != 1 {
		t.Fatalf("expected one deduped Stripe webhook, got %d", len(a.store.processorWebhookEvents))
	}
}

func TestAPIV1PlaidWebhookVerificationMockAndInvalid(t *testing.T) {
	a := newAPITestApp(t)
	a.cfg.PlaidWebhookVerification = "true"
	body := []byte(`{"webhook_type":"TRANSACTIONS","webhook_code":"SYNC_UPDATES_AVAILABLE","item_id":"item_test","environment":"sandbox"}`)
	bad := performAPIRequest(a, http.MethodPost, "/api/v1/webhooks/plaid", "", "req_plaid_bad_sig", body)
	if bad.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid Plaid verification 401, got %d: %s", bad.Code, bad.Body.String())
	}
	good := performAPIRequestWithHeaders(a, http.MethodPost, "/api/v1/webhooks/plaid", "", "req_plaid_mock_sig", body, map[string]string{"X-Plaid-Mock-Webhook": "true"})
	if good.Code != http.StatusAccepted {
		t.Fatalf("expected Plaid mock verification 202, got %d: %s", good.Code, good.Body.String())
	}
}

func TestAPIV1WebhookMetricsCounters(t *testing.T) {
	a := newAPITestApp(t)
	a.cfg.StripeWebhookSecret = "whsec_test"
	body := []byte(`{"id":"evt_metrics_1","type":"balance.available"}`)
	headers := map[string]string{"Stripe-Signature": processors.BuildStripeTestSignature("whsec_test", time.Now(), body)}
	rec := performAPIRequestWithHeaders(a, http.MethodPost, "/api/v1/webhooks/processors/stripe", "", "req_metrics_stripe", body, headers)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected stripe webhook 202, got %d: %s", rec.Code, rec.Body.String())
	}
	owner := registerAPITestUser(t, a, "metrics-phase8@example.com")
	metrics := performAPIRequest(a, http.MethodGet, "/api/v1/ops/metrics", owner.Token, "req_metrics_phase8", nil)
	if metrics.Code != http.StatusOK {
		t.Fatalf("expected metrics 200, got %d: %s", metrics.Code, metrics.Body.String())
	}
	if !strings.Contains(metrics.Body.String(), "stripe_webhook_events_total") {
		t.Fatalf("expected stripe metric in response: %s", metrics.Body.String())
	}
}

func oauthStateForTest(orgID, userID, state string, expiresAt time.Time) models.OAuthState {
	return models.OAuthState{ID: "state-test", OrganizationID: orgID, UserID: userID, Provider: "stripe", StateHash: hashOAuthState(state), RedirectURI: "http://localhost/callback", ExpiresAt: expiresAt, CreatedAt: time.Now().UTC()}
}
