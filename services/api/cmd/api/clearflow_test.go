package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestPayoutBreakdownExplainsPaymentsFeesAndRefunds(t *testing.T) {
	a := &app{store: newStore()}
	uid := "user-1"
	now := time.Now().UTC()
	org := models.Organization{ID: auth.NewID(), UserID: uid, Name: "Demo Org", Type: "student_organization", Currency: "USD", CreatedAt: now}
	payment := models.Payment{ID: auth.NewID(), OrganizationID: org.ID, Processor: "stripe", ProcessorPaymentID: "ch_test", Amount: 100, Currency: "USD", Status: "succeeded", OccurredAt: now, Description: "Ticket", CreatedAt: now}
	fee := models.Fee{ID: auth.NewID(), OrganizationID: org.ID, ProcessorFeeID: "fee_test", PaymentID: payment.ID, Amount: 3.2, Currency: "USD", OccurredAt: now, Description: "Stripe fee"}
	refund := models.Refund{ID: auth.NewID(), OrganizationID: org.ID, ProcessorRefundID: "re_test", PaymentID: payment.ID, Amount: 20, Currency: "USD", OccurredAt: now}
	payout := models.Payout{ID: auth.NewID(), OrganizationID: org.ID, Processor: "stripe", ProcessorPayoutID: "po_test", Amount: 76.8, Currency: "USD", Status: "paid", ExpectedArrivalAt: now, CreatedAt: now}
	a.store.organizations[org.ID] = org
	a.store.payments[payment.ID] = payment
	a.store.fees[fee.ID] = fee
	a.store.refunds[refund.ID] = refund
	a.store.payouts[payout.ID] = payout

	req := httptest.NewRequest(http.MethodGet, "/payouts/"+payout.ID+"/breakdown", nil)
	req.SetPathValue("id", payout.ID)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", uid))
	rec := httptest.NewRecorder()

	a.getPayoutBreakdown(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		GrossPayments float64             `json:"gross_payments"`
		Fees          float64             `json:"fees"`
		Refunds       float64             `json:"refunds"`
		NetPayout     float64             `json:"net_payout"`
		Items         []models.PayoutItem `json:"items"`
		Payout        models.Payout       `json:"payout"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.GrossPayments != 100 || payload.Fees != 3.2 || payload.Refunds != 20 || payload.NetPayout != 76.8 {
		t.Fatalf("unexpected payout breakdown: %#v", payload)
	}
	if len(payload.Items) != 3 {
		t.Fatalf("expected 3 payout items, got %d", len(payload.Items))
	}
}
