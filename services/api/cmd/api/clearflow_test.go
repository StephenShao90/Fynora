package main

import (
	"testing"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

func TestReconcileOrganizationMatchesPayoutToBankDeposit(t *testing.T) {
	a := &app{store: newStore()}
	uid := "user-1"
	org := models.Organization{ID: auth.NewID(), UserID: uid, Name: "Demo Org", Type: "student_organization", Currency: "USD", CreatedAt: time.Now()}
	payout := models.Payout{ID: auth.NewID(), OrganizationID: org.ID, Processor: "stripe", ProcessorPayoutID: "po_test", Amount: 100, Currency: "USD", Status: "paid", ExpectedArrivalAt: time.Now()}
	bank := models.BankTransaction{ID: auth.NewID(), OrganizationID: org.ID, Source: "test", ExternalID: "bank_test", Amount: 100, Direction: "credit", Currency: "USD", Description: "STRIPE PAYOUT", PostedAt: time.Now()}
	a.store.organizations[org.ID] = org
	a.store.payouts[payout.ID] = payout
	a.store.bankTransactions[bank.ID] = bank

	run := a.reconcileOrganization(org.ID, uid)
	if run.MatchedCount != 1 {
		t.Fatalf("expected 1 match, got %#v", run)
	}
	if run.ExceptionCount != 0 {
		t.Fatalf("expected no exceptions, got %#v", run)
	}
}

func TestReconcileOrganizationFlagsUnmatchedDeposit(t *testing.T) {
	a := &app{store: newStore()}
	uid := "user-1"
	org := models.Organization{ID: auth.NewID(), UserID: uid, Name: "Demo Org", Type: "student_organization", Currency: "USD", CreatedAt: time.Now()}
	bank := models.BankTransaction{ID: auth.NewID(), OrganizationID: org.ID, Source: "test", ExternalID: "bank_unknown", Amount: 212.45, Direction: "credit", Currency: "USD", Description: "Unknown deposit", PostedAt: time.Now()}
	a.store.organizations[org.ID] = org
	a.store.bankTransactions[bank.ID] = bank

	run := a.reconcileOrganization(org.ID, uid)
	if run.ExceptionCount != 1 {
		t.Fatalf("expected 1 exception, got %#v", run)
	}
}
