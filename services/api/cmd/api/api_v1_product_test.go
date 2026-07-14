package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAPIV1OnboardingStatusPersistsChecklist(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "onboarding-owner@example.com")

	save := performAPIRequest(a, http.MethodPut, "/api/v1/onboarding/status", owner.Token, "req_onboarding_put", []byte(`{
		"selected_scenario":"creator",
		"business_type":"creator",
		"checklist":{"workspace_created":true,"business_profile_chosen":true},
		"completed":true
	}`))
	if save.Code != http.StatusOK {
		t.Fatalf("expected onboarding save 200, got %d: %s", save.Code, save.Body.String())
	}

	get := performAPIRequest(a, http.MethodGet, "/api/v1/onboarding/status", owner.Token, "req_onboarding_get", nil)
	if get.Code != http.StatusOK {
		t.Fatalf("expected onboarding get 200, got %d: %s", get.Code, get.Body.String())
	}
	var payload struct {
		SelectedScenario  string                 `json:"selected_scenario"`
		BusinessType      string                 `json:"business_type"`
		Checklist         map[string]interface{} `json:"checklist"`
		ProviderReadiness map[string]interface{} `json:"provider_readiness"`
	}
	if err := json.NewDecoder(get.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.SelectedScenario != "creator" || payload.BusinessType != "creator" {
		t.Fatalf("unexpected onboarding payload: %#v", payload)
	}
	if payload.Checklist["workspace_created"] != true || payload.ProviderReadiness["workspace_created"] != true {
		t.Fatalf("expected checklist/readiness flags, got %#v", payload)
	}
}

func TestExceptionNotesCanBeAddedListedAndSavedOnResolve(t *testing.T) {
	a := newAPITestApp(t)
	owner := registerAPITestUser(t, a, "notes-owner@example.com")

	_ = performAPIRequest(a, http.MethodPost, "/sync/stripe", owner.Token, "req_notes_stripe", nil)
	_ = performAPIRequest(a, http.MethodPost, "/sync/bank", owner.Token, "req_notes_bank", nil)
	reconcile := performAPIRequest(a, http.MethodPost, "/reconciliation/runs", owner.Token, "req_notes_reconcile", nil)
	if reconcile.Code != http.StatusCreated {
		t.Fatalf("expected reconciliation 201, got %d: %s", reconcile.Code, reconcile.Body.String())
	}

	exceptions := performAPIRequest(a, http.MethodGet, "/reconciliation/exceptions", owner.Token, "req_notes_exceptions", nil)
	if exceptions.Code != http.StatusOK {
		t.Fatalf("expected exceptions 200, got %d: %s", exceptions.Code, exceptions.Body.String())
	}
	var rows []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(exceptions.Body).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("expected at least one reconciliation exception")
	}
	exceptionID := rows[0].ID

	add := performAPIRequest(a, http.MethodPost, "/reconciliation/exceptions/"+exceptionID+"/notes", owner.Token, "req_notes_add", []byte(`{"body":"Called bank and confirmed deposit memo."}`))
	if add.Code != http.StatusCreated {
		t.Fatalf("expected note create 201, got %d: %s", add.Code, add.Body.String())
	}

	resolve := performAPIRequest(a, http.MethodPatch, "/reconciliation/exceptions/"+exceptionID, owner.Token, "req_notes_resolve", []byte(`{"status":"resolved","note":"Resolved after processor reserve cleared."}`))
	if resolve.Code != http.StatusOK {
		t.Fatalf("expected resolve 200, got %d: %s", resolve.Code, resolve.Body.String())
	}

	list := performAPIRequest(a, http.MethodGet, "/reconciliation/exceptions/"+exceptionID+"/notes", owner.Token, "req_notes_list", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("expected notes list 200, got %d: %s", list.Code, list.Body.String())
	}
	body := list.Body.String()
	if !strings.Contains(body, "Called bank") || !strings.Contains(body, "processor reserve") {
		t.Fatalf("expected both notes in history, got %s", body)
	}
}
