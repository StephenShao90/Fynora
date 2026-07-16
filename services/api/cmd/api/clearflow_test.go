package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestPortfolioCSVParsersAcceptBrokerStyleHeaders(t *testing.T) {
	holdingsCSV := `Ticker,Security,Asset Class,Shares,Cost Basis,Current Price,Current Value,CCY
AAPL,Apple Inc.,Equity,"10","$1,200.00","$200.00","$2,000.00",USD
CASH,Cash,Money Market,500,500,1,500,USD`
	holdings, importErrors := parseHoldingsCSV("user-1", holdingsCSV, "acct-1")
	if len(importErrors) != 0 || len(holdings) != 2 {
		t.Fatalf("expected 2 imported holdings and 0 failures, got holdings=%d failed=%d", len(holdings), len(importErrors))
	}
	if holdings[0].Symbol != "AAPL" || holdings[0].SecurityType != "stock" || holdings[0].AverageCost != 120 || holdings[0].MarketValue != 2000 {
		t.Fatalf("unexpected parsed holding: %#v", holdings[0])
	}
	if holdings[1].SecurityType != "cash" {
		t.Fatalf("expected money market to normalize to cash, got %#v", holdings[1])
	}

	transactionsCSV := `Trade Date,Activity,Ticker,Shares,Trade Price,Net Amount,Commission,CCY,Details
07/01/2026,Buy,AAPL,3,190,"(570.00)",1.50,USD,Buy Apple
07/02/2026,Dividend,AAPL,0,0,12.42,0,USD,Dividend received`
	txs, importErrors := parsePortfolioTransactionsCSV("user-1", transactionsCSV, "acct-1")
	if len(importErrors) != 0 || len(txs) != 2 {
		t.Fatalf("expected 2 imported transactions and 0 failures, got txs=%d failed=%d", len(txs), len(importErrors))
	}
	if txs[0].TransactionType != "buy" || txs[0].Amount != -570 || txs[0].Fees != 1.5 {
		t.Fatalf("unexpected parsed transaction: %#v", txs[0])
	}
	if txs[1].TransactionType != "dividend" {
		t.Fatalf("expected dividend normalization, got %#v", txs[1])
	}
}

func TestPortfolioCSVParsersReturnRowLevelErrors(t *testing.T) {
	holdingsCSV := `Ticker,Shares,Current Value
AAPL,10,2000
,bad,100`
	holdings, importErrors := parseHoldingsCSV("user-1", holdingsCSV, "acct-1")
	if len(holdings) != 1 || len(importErrors) != 1 {
		t.Fatalf("expected 1 holding and 1 error, got holdings=%d errors=%d", len(holdings), len(importErrors))
	}
	if importErrors[0].RowNumber != 3 || importErrors[0].Code != "invalid_quantity" {
		t.Fatalf("unexpected import error: %#v", importErrors[0])
	}
}

func TestPortfolioReviewEndpointsReturnImportsAndTransactions(t *testing.T) {
	a := &app{store: newStore()}
	uid := "user-1"
	now := time.Now().UTC()
	a.store.imports["imp-1"] = models.RawImport{ID: "imp-1", UserID: uid, ImportType: "holdings", OriginalFilename: "holdings.csv", RowCount: 2, ImportedCount: 2, CreatedAt: now}
	a.store.imports["imp-2"] = models.RawImport{ID: "imp-2", UserID: uid, ImportType: "transactions", OriginalFilename: "bank.csv", RowCount: 1, ImportedCount: 1, CreatedAt: now}
	a.store.portfolioTransactions["ptx-1"] = models.PortfolioTransaction{ID: "ptx-1", UserID: uid, Symbol: "AAPL", TransactionType: "buy", Amount: -570, Currency: "USD", OccurredAt: now}
	a.store.importErrors["err-1"] = models.ImportError{ID: "err-1", ImportID: "imp-1", UserID: uid, RowNumber: 4, Field: "symbol", Code: "missing_symbol", Message: "symbol/ticker is required", CreatedAt: now}

	req := httptest.NewRequest(http.MethodGet, "/portfolio/imports", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", uid))
	rec := httptest.NewRecorder()
	a.listPortfolioImports(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected imports status 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "holdings.csv") || strings.Contains(body, "bank.csv") {
		t.Fatalf("unexpected imports body: %s", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/portfolio/transactions", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", uid))
	rec = httptest.NewRecorder()
	a.listPortfolioTransactions(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected transactions status 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "AAPL") || !strings.Contains(body, "buy") {
		t.Fatalf("unexpected transactions body: %s", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/portfolio/imports/imp-1/errors", nil)
	req.SetPathValue("id", "imp-1")
	req = req.WithContext(context.WithValue(req.Context(), "user_id", uid))
	rec = httptest.NewRecorder()
	a.listPortfolioImportErrors(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected import errors status 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "missing_symbol") {
		t.Fatalf("unexpected import errors body: %s", body)
	}
}

func TestPlaidInvestmentsMockSyncPopulatesPortfolio(t *testing.T) {
	a := &app{store: newStore()}
	uid := "user-1"
	req := httptest.NewRequest(http.MethodPost, "/connections/plaid/sync-investments", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", uid))
	rec := httptest.NewRecorder()

	a.syncPlaidInvestments(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(a.store.holdings) != 3 {
		t.Fatalf("expected 3 holdings from mock investments sync, got %d", len(a.store.holdings))
	}
	if len(a.store.portfolioTransactions) != 3 {
		t.Fatalf("expected 3 portfolio transactions from mock investments sync, got %d", len(a.store.portfolioTransactions))
	}
}

func TestPlaidTransactionSyncRemovesUndecryptableLocalConnection(t *testing.T) {
	a := newAPITestApp(t)
	uid := "user-1"
	a.cfg.LocalStorageDir = t.TempDir() + "/raw"
	a.store.plaidConnections["stale"] = models.PlaidConnection{
		ID:                    "stale",
		UserID:                uid,
		ItemID:                "item_stale",
		InstitutionName:       "Old Sandbox Bank",
		AccessTokenCiphertext: "not-valid-ciphertext",
		CreatedAt:             time.Now().UTC(),
		UpdatedAt:             time.Now().UTC(),
	}

	req := httptest.NewRequest(http.MethodPost, "/connections/plaid/sync-transactions", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", uid))
	rec := httptest.NewRecorder()
	a.syncPlaidTransactions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected stale Plaid connection to be handled as recoverable, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		ImportedCount          int    `json:"imported_count"`
		ConnectionCount        int    `json:"connection_count"`
		InvalidConnectionCount int    `json:"invalid_connection_count"`
		Message                string `json:"message"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.InvalidConnectionCount != 1 || payload.ConnectionCount != 0 || payload.Message == "" {
		t.Fatalf("unexpected stale connection payload: %#v", payload)
	}
	if _, ok := a.store.plaidConnections["stale"]; ok {
		t.Fatal("expected stale Plaid connection to be removed")
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
