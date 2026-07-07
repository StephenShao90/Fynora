package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

type authTestResponse struct {
	Token        string `json:"token"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
	Organizations []struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	} `json:"organizations"`
}

func registerAPITestUser(t *testing.T, a *app, email string) authTestResponse {
	t.Helper()
	body := []byte(`{"email":"` + email + `","password":"password123","organization_name":"Test Org"}`)
	rec := performAPIRequest(a, http.MethodPost, "/api/v1/auth/register", "", "req_register", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload authTestResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Token == "" || payload.User.ID == "" || len(payload.Organizations) == 0 {
		t.Fatalf("unexpected register payload: %#v", payload)
	}
	return payload
}

func TestAPIV1RegisterLoginAndMeReturnMemberships(t *testing.T) {
	a := newAPITestApp(t)
	created := registerAPITestUser(t, a, "owner@example.com")
	if created.Organizations[0].Role != "owner" {
		t.Fatalf("expected registered user to own default org, got %#v", created.Organizations)
	}
	login := performAPIRequest(a, http.MethodPost, "/api/v1/auth/login", "", "req_login", []byte(`{"email":"owner@example.com","password":"password123"}`))
	if login.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d: %s", login.Code, login.Body.String())
	}
	me := performAPIRequest(a, http.MethodGet, "/api/v1/me", created.Token, "req_me", nil)
	if me.Code != http.StatusOK {
		t.Fatalf("expected me status 200, got %d: %s", me.Code, me.Body.String())
	}
	var payload struct {
		User struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"user"`
		Organizations []struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"organizations"`
	}
	if err := json.NewDecoder(me.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.User.Email != "owner@example.com" || payload.User.Name == "" || len(payload.Organizations) != 1 {
		t.Fatalf("unexpected me payload: %#v", payload)
	}
}

func TestAPIV1MemberManagementAndRBAC(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "owner2@example.com")
	viewer := registerAPITestUser(t, a, "viewer@example.com")
	orgID := owner.Organizations[0].ID

	addBody := []byte(`{"email":"viewer@example.com","role":"viewer"}`)
	add := performAPIRequest(a, http.MethodPost, "/api/v1/organizations/"+orgID+"/members", owner.Token, "req_add_member", addBody)
	if add.Code != http.StatusCreated {
		t.Fatalf("expected add member status 201, got %d: %s", add.Code, add.Body.String())
	}
	duplicate := performAPIRequest(a, http.MethodPost, "/api/v1/organizations/"+orgID+"/members", owner.Token, "req_dup_member", addBody)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("expected duplicate member status 409, got %d: %s", duplicate.Code, duplicate.Body.String())
	}

	read := performAPIRequest(a, http.MethodGet, "/api/v1/payments?organizationId="+orgID, viewer.Token, "req_viewer_read", nil)
	if read.Code != http.StatusOK {
		t.Fatalf("expected viewer read status 200, got %d: %s", read.Code, read.Body.String())
	}
	runAsViewer := performAPIRequest(a, http.MethodPost, "/api/v1/reconciliation-runs?organizationId="+orgID, viewer.Token, "req_viewer_run", nil)
	if runAsViewer.Code != http.StatusForbidden {
		t.Fatalf("expected viewer reconciliation status 403, got %d: %s", runAsViewer.Code, runAsViewer.Body.String())
	}
	patch := performAPIRequest(a, http.MethodPatch, "/api/v1/organizations/"+orgID+"/members/"+viewer.User.ID, owner.Token, "req_patch_member", []byte(`{"role":"analyst"}`))
	if patch.Code != http.StatusOK {
		t.Fatalf("expected patch member status 200, got %d: %s", patch.Code, patch.Body.String())
	}
	runAsAnalyst := performAPIRequest(a, http.MethodPost, "/api/v1/reconciliation-runs?organizationId="+orgID, viewer.Token, "req_analyst_run", nil)
	if runAsAnalyst.Code != http.StatusAccepted {
		t.Fatalf("expected analyst reconciliation status 202, got %d: %s", runAsAnalyst.Code, runAsAnalyst.Body.String())
	}
}

func TestAPIV1OrganizationWriteRequiresOrganizationID(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "missing-org@example.com")
	rec := performAPIRequest(a, http.MethodPost, "/api/v1/reconciliation-runs", owner.Token, "req_missing_org", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected missing org status 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %#v", payload)
	}
}

func TestAPIV1BlocksCrossOrganizationAccess(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "cross-owner@example.com")
	outsider := registerAPITestUser(t, a, "outsider@example.com")
	orgID := owner.Organizations[0].ID
	rec := performAPIRequest(a, http.MethodGet, "/api/v1/payments?organizationId="+orgID, outsider.Token, "req_cross_org", nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected cross-org status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPIV1CannotRemoveLastOwner(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "last-owner@example.com")
	orgID := owner.Organizations[0].ID
	rec := performAPIRequest(a, http.MethodDelete, "/api/v1/organizations/"+orgID+"/members/"+owner.User.ID, owner.Token, "req_last_owner", nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected last owner status 409, got %d: %s", rec.Code, rec.Body.String())
	}
}
