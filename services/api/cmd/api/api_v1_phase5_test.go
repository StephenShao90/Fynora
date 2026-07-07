package main

import (
	"net/http"
	"testing"
)

func TestAPIV1ViewerCanReadInsightsAndCrossOrgForbidden(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "intel-owner@example.com")
	viewer := registerAPITestUser(t, a, "intel-viewer@example.com")
	outsider := registerAPITestUser(t, a, "intel-outsider@example.com")
	orgID := owner.Organizations[0].ID
	add := performAPIRequest(a, http.MethodPost, "/api/v1/organizations/"+orgID+"/members", owner.Token, "req_intel_add", []byte(`{"email":"intel-viewer@example.com","role":"viewer"}`))
	if add.Code != http.StatusCreated {
		t.Fatalf("expected add viewer 201, got %d: %s", add.Code, add.Body.String())
	}
	ok := performAPIRequest(a, http.MethodGet, "/api/v1/insights/anomalies?organizationId="+orgID, viewer.Token, "req_intel_view", nil)
	if ok.Code != http.StatusOK {
		t.Fatalf("expected viewer insight read 200, got %d: %s", ok.Code, ok.Body.String())
	}
	forbidden := performAPIRequest(a, http.MethodGet, "/api/v1/recommendations/cash?organizationId="+orgID, outsider.Token, "req_intel_forbidden", nil)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("expected cross-org forbidden, got %d: %s", forbidden.Code, forbidden.Body.String())
	}
}

func TestAPIV1ForecastAndPayoutExplanation(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "intel-payout@example.com")
	orgID := owner.Organizations[0].ID
	_ = performAPIRequest(a, http.MethodPost, "/sync/stripe", owner.Token, "req_sync_stripe_legacy", nil)
	forecast := performAPIRequest(a, http.MethodGet, "/api/v1/cashflow/forecast?organizationId="+orgID+"&horizonDays=7", owner.Token, "req_forecast", nil)
	if forecast.Code != http.StatusOK {
		t.Fatalf("expected forecast 200, got %d: %s", forecast.Code, forecast.Body.String())
	}
	a.store.mu.RLock()
	var payoutID string
	for _, payout := range a.store.payouts {
		if payout.OrganizationID == orgID {
			payoutID = payout.ID
			break
		}
	}
	a.store.mu.RUnlock()
	explanation := performAPIRequest(a, http.MethodGet, "/api/v1/payouts/"+payoutID+"/explanation?organizationId="+orgID, owner.Token, "req_payout_explain", nil)
	if explanation.Code != http.StatusOK {
		t.Fatalf("expected payout explanation 200, got %d: %s", explanation.Code, explanation.Body.String())
	}
}
