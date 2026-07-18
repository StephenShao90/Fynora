package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/authz"
	"github.com/StephenShao90/Fynora/services/api/internal/httpapi"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
	"github.com/StephenShao90/Fynora/services/api/internal/repository"
	"github.com/StephenShao90/Fynora/services/api/internal/validation"
)

type clearflowOrgContextKey struct{}

func (a *app) createOrganization(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Currency string `json:"currency"`
	}
	if !decode(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		req.Name = "Clearflow Demo Organization"
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.Type == "" {
		req.Type = "student_organization"
	}
	if a.cfRepo != nil {
		org, err := a.cfRepo.CreateOrganization(r.Context(), a.currentClearflowUser(r), models.Organization{Name: req.Name, Type: req.Type, Currency: req.Currency})
		if err != nil {
			a.logOperation(r, "organization.create_failed", "", map[string]interface{}{"error": err.Error(), "latency_ms": time.Since(start).Milliseconds()})
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not create organization")
			return
		}
		a.logOperation(r, "organization.created", org.ID, map[string]interface{}{"organization_id": org.ID, "name": org.Name, "storage": "postgres", "latency_ms": time.Since(start).Milliseconds()})
		writeJSON(w, 201, org)
		return
	}
	now := time.Now().UTC()
	org := models.Organization{ID: auth.NewID(), UserID: userID(r), Name: req.Name, Type: req.Type, Currency: req.Currency, CreatedAt: now, UpdatedAt: now}
	a.store.mu.Lock()
	a.store.organizations[org.ID] = org
	a.auditLocked(org.ID, userID(r), "organization.created", "organization", org.ID)
	a.store.mu.Unlock()
	a.logOperation(r, "organization.created", org.ID, map[string]interface{}{"organization_id": org.ID, "name": org.Name, "latency_ms": time.Since(start).Milliseconds()})
	writeJSON(w, 201, org)
}

func (a *app) listOrganizations(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		orgs, err := a.cfRepo.ListOrganizations(r.Context(), userID(r))
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not list organizations")
			return
		}
		writeJSON(w, 200, orgs)
		return
	}
	writeJSON(w, 200, a.userOrganizations(userID(r)))
}

func (a *app) listPayments(w http.ResponseWriter, r *http.Request) {
	rows, _, ok := a.paymentsForRequest(w, r)
	if !ok {
		return
	}
	writeJSON(w, 200, rows)
}

func (a *app) listPaymentsV1(w http.ResponseWriter, r *http.Request) {
	if r, ok := a.withV1Organization(w, r, false, authz.CanRead); ok {
		a.listPaymentsV1Response(w, r)
	}
}

func (a *app) listPaymentsV1Response(w http.ResponseWriter, r *http.Request) {
	rows, query, ok := a.paymentsForRequest(w, r)
	if !ok {
		return
	}
	writeJSON(w, 200, httpapi.Paginated(rows, query))
}

func (a *app) paymentsForRequest(w http.ResponseWriter, r *http.Request) ([]models.Payment, httpapi.ListQuery, bool) {
	query, ok := parseClearflowListQuery(w, r)
	if !ok {
		return nil, query, false
	}
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return nil, query, false
		}
		rows, err := a.cfRepo.ListPayments(r.Context(), org.ID)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not list payments")
			return nil, query, false
		}
		return httpapi.Page(filterPaymentsByDate(rows, query), query), query, true
	}
	orgID := a.ensureOrganization(userID(r)).ID
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.Payment{}
	for _, row := range a.store.payments {
		if row.OrganizationID == orgID {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OccurredAt.After(out[j].OccurredAt) })
	return httpapi.Page(filterPaymentsByDate(out, query), query), query, true
}

func (a *app) listPayouts(w http.ResponseWriter, r *http.Request) {
	rows, _, ok := a.payoutsForRequest(w, r)
	if !ok {
		return
	}
	writeJSON(w, 200, rows)
}

func (a *app) listPayoutsV1(w http.ResponseWriter, r *http.Request) {
	if r, ok := a.withV1Organization(w, r, false, authz.CanRead); ok {
		a.listPayoutsV1Response(w, r)
	}
}

func (a *app) listPayoutsV1Response(w http.ResponseWriter, r *http.Request) {
	rows, query, ok := a.payoutsForRequest(w, r)
	if !ok {
		return
	}
	writeJSON(w, 200, httpapi.Paginated(rows, query))
}

func (a *app) payoutsForRequest(w http.ResponseWriter, r *http.Request) ([]models.Payout, httpapi.ListQuery, bool) {
	query, ok := parseClearflowListQuery(w, r)
	if !ok {
		return nil, query, false
	}
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return nil, query, false
		}
		rows, err := a.cfRepo.ListPayouts(r.Context(), org.ID)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not list payouts")
			return nil, query, false
		}
		return httpapi.Page(filterPayoutsByDate(rows, query), query), query, true
	}
	orgID := a.ensureOrganization(userID(r)).ID
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.Payout{}
	for _, row := range a.store.payouts {
		if row.OrganizationID == orgID {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ExpectedArrivalAt.After(out[j].ExpectedArrivalAt) })
	return httpapi.Page(filterPayoutsByDate(out, query), query), query, true
}

func (a *app) getPayoutBreakdown(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return
		}
		payload, err := a.cfRepo.PayoutBreakdown(r.Context(), org.ID, r.PathValue("id"))
		if err != nil {
			errorJSON(w, r, 404, "NOT_FOUND", "payout not found")
			return
		}
		writeJSON(w, 200, payload)
		return
	}
	orgID := a.ensureOrganization(userID(r)).ID
	payoutID := r.PathValue("id")
	a.store.mu.RLock()
	payout, ok := a.store.payouts[payoutID]
	items := []models.PayoutItem{}
	var gross, fees, refunds float64
	for _, payment := range a.store.payments {
		if payment.OrganizationID == orgID {
			gross += payment.Amount
			items = append(items, models.PayoutItem{ID: payment.ID, OrganizationID: orgID, PayoutID: payoutID, SourceType: "payment", SourceID: payment.ProcessorPaymentID, Amount: payment.Amount, Currency: payment.Currency, Description: payment.Description, CreatedAt: payment.CreatedAt})
		}
	}
	for _, fee := range a.store.fees {
		if fee.OrganizationID == orgID {
			fees += fee.Amount
			items = append(items, models.PayoutItem{ID: fee.ID, OrganizationID: orgID, PayoutID: payoutID, SourceType: "fee", SourceID: fee.ProcessorFeeID, Amount: -fee.Amount, Currency: fee.Currency, Description: fee.Description, CreatedAt: fee.OccurredAt})
		}
	}
	for _, refund := range a.store.refunds {
		if refund.OrganizationID == orgID {
			refunds += refund.Amount
			items = append(items, models.PayoutItem{ID: refund.ID, OrganizationID: orgID, PayoutID: payoutID, SourceType: "refund", SourceID: refund.ProcessorRefundID, Amount: -refund.Amount, Currency: refund.Currency, Description: "Refund", CreatedAt: refund.OccurredAt})
		}
	}
	a.store.mu.RUnlock()
	if !ok || payout.OrganizationID != orgID {
		errorJSON(w, r, 404, "NOT_FOUND", "payout not found")
		return
	}
	writeJSON(w, 200, map[string]interface{}{"payout": payout, "items": items, "gross_payments": round2(gross), "fees": round2(fees), "refunds": round2(refunds), "net_payout": payout.Amount})
}

func (a *app) listBankTransactions(w http.ResponseWriter, r *http.Request) {
	rows, _, ok := a.bankTransactionsForRequest(w, r)
	if !ok {
		return
	}
	writeJSON(w, 200, rows)
}

func (a *app) listBankTransactionsV1(w http.ResponseWriter, r *http.Request) {
	if r, ok := a.withV1Organization(w, r, false, authz.CanRead); ok {
		a.listBankTransactionsV1Response(w, r)
	}
}

func (a *app) dashboardSummaryV1(w http.ResponseWriter, r *http.Request) {
	r, ok := a.withV1Organization(w, r, false, authz.CanRead)
	if !ok {
		return
	}
	org := r.Context().Value(clearflowOrgContextKey{}).(models.Organization)

	cash, err := a.dashboardCashSummary(r, org.ID)
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not build dashboard cash summary")
		return
	}
	forecast, err := a.dashboardCashForecast(r, org.ID)
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not build dashboard cash forecast")
		return
	}
	payments, err := a.dashboardPayments(r, org.ID)
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not load dashboard payments")
		return
	}
	payouts, err := a.dashboardPayouts(r, org.ID)
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not load dashboard payouts")
		return
	}
	bank, err := a.dashboardBankTransactions(r, org.ID)
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not load dashboard bank transactions")
		return
	}
	exceptions, err := a.dashboardExceptions(r, org.ID)
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not load dashboard exceptions")
		return
	}
	metrics, err := a.dashboardMetrics(r)
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not load dashboard metrics")
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"cash":              cash,
		"forecast":          forecast,
		"exceptions":        firstN(exceptions, 8),
		"payouts":           firstN(payouts, 8),
		"payments":          firstN(payments, 8),
		"bank_transactions": firstN(bank, 8),
		"connections":       a.userPlaidConnections(userID(r)),
		"metrics":           metrics,
	})
}

func (a *app) dashboardCashSummary(r *http.Request, orgID string) (map[string]float64, error) {
	if a.cfRepo != nil {
		return a.cfRepo.CashSummary(r.Context(), orgID)
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	var cash, income, expenses, pendingPayouts, fees, refunds float64
	for _, row := range a.store.bankTransactions {
		if row.OrganizationID != orgID {
			continue
		}
		if row.Direction == "credit" {
			cash += row.Amount
			income += row.Amount
		} else {
			cash -= row.Amount
			expenses += row.Amount
		}
	}
	for _, payout := range a.store.payouts {
		if payout.OrganizationID == orgID && payout.Status != "paid" {
			pendingPayouts += payout.Amount
		}
	}
	for _, fee := range a.store.fees {
		if fee.OrganizationID == orgID {
			fees += fee.Amount
		}
	}
	for _, refund := range a.store.refunds {
		if refund.OrganizationID == orgID {
			refunds += refund.Amount
		}
	}
	return map[string]float64{"cash_balance": round2(cash), "income": round2(income), "expenses": round2(expenses), "pending_payouts": round2(pendingPayouts), "fees": round2(fees), "refunds": round2(refunds), "net_cash_flow": round2(income - expenses - fees - refunds)}, nil
}

func (a *app) dashboardCashForecast(r *http.Request, orgID string) ([]map[string]interface{}, error) {
	if a.cfRepo != nil {
		return a.cfRepo.CashForecast(r.Context(), orgID)
	}
	summary := a.cashSnapshot(orgID)
	points := []map[string]interface{}{}
	for _, days := range []int{7, 30, 60} {
		expectedPayouts := a.expectedPayouts(orgID, days)
		expectedExpenses := 0.0
		if days >= 30 {
			expectedExpenses = 450
		}
		points = append(points, map[string]interface{}{"days": days, "projected_cash": round2(summary + expectedPayouts - expectedExpenses), "expected_payouts": round2(expectedPayouts), "expected_expenses": expectedExpenses})
	}
	return points, nil
}

func (a *app) dashboardPayments(r *http.Request, orgID string) ([]models.Payment, error) {
	if a.cfRepo != nil {
		rows, err := a.cfRepo.ListPayments(r.Context(), orgID)
		if err != nil {
			return nil, err
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].OccurredAt.After(rows[j].OccurredAt) })
		return rows, nil
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	rows := []models.Payment{}
	for _, row := range a.store.payments {
		if row.OrganizationID == orgID {
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].OccurredAt.After(rows[j].OccurredAt) })
	return rows, nil
}

func (a *app) dashboardPayouts(r *http.Request, orgID string) ([]models.Payout, error) {
	if a.cfRepo != nil {
		rows, err := a.cfRepo.ListPayouts(r.Context(), orgID)
		if err != nil {
			return nil, err
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].ExpectedArrivalAt.After(rows[j].ExpectedArrivalAt) })
		return rows, nil
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	rows := []models.Payout{}
	for _, row := range a.store.payouts {
		if row.OrganizationID == orgID {
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ExpectedArrivalAt.After(rows[j].ExpectedArrivalAt) })
	return rows, nil
}

func (a *app) dashboardBankTransactions(r *http.Request, orgID string) ([]models.BankTransaction, error) {
	if a.cfRepo != nil {
		rows, err := a.cfRepo.ListBankTransactions(r.Context(), orgID)
		if err != nil {
			return nil, err
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].PostedAt.After(rows[j].PostedAt) })
		return rows, nil
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	rows := []models.BankTransaction{}
	for _, row := range a.store.bankTransactions {
		if row.OrganizationID == orgID {
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].PostedAt.After(rows[j].PostedAt) })
	return rows, nil
}

func (a *app) dashboardExceptions(r *http.Request, orgID string) ([]models.ReconciliationException, error) {
	if a.cfRepo != nil {
		return a.cfRepo.ListExceptions(r.Context(), orgID)
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	rows := []models.ReconciliationException{}
	for _, row := range a.store.reconciliationExceptions {
		if row.OrganizationID == orgID {
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].CreatedAt.After(rows[j].CreatedAt) })
	return rows, nil
}

func (a *app) dashboardMetrics(r *http.Request) (opsMetrics, error) {
	a.store.mu.RLock()
	metrics := a.store.metrics
	for _, job := range a.store.jobs {
		applyJobStatusToMetrics(&metrics, job.Status)
	}
	a.store.mu.RUnlock()
	if a.cfRepo != nil {
		counts, err := a.cfRepo.JobStatusCounts(r.Context())
		if err != nil {
			return metrics, err
		}
		metrics.JobQueueDepth = 0
		metrics.JobsCompletedTotal = counts["completed"]
		metrics.JobsFailedTotal = counts["failed"]
		metrics.JobsDeadTotal = counts["dead"]
		for status, count := range counts {
			if status == "queued" || status == "running" {
				metrics.JobQueueDepth += count
			}
		}
	}
	return metrics, nil
}

func firstN[T any](rows []T, n int) []T {
	if len(rows) == 0 {
		return []T{}
	}
	if len(rows) <= n {
		return rows
	}
	return rows[:n]
}

func (a *app) listBankTransactionsV1Response(w http.ResponseWriter, r *http.Request) {
	rows, query, ok := a.bankTransactionsForRequest(w, r)
	if !ok {
		return
	}
	writeJSON(w, 200, httpapi.Paginated(rows, query))
}

func (a *app) bankTransactionsForRequest(w http.ResponseWriter, r *http.Request) ([]models.BankTransaction, httpapi.ListQuery, bool) {
	query, ok := parseClearflowListQuery(w, r)
	if !ok {
		return nil, query, false
	}
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return nil, query, false
		}
		rows, err := a.cfRepo.ListBankTransactions(r.Context(), org.ID)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not list bank transactions")
			return nil, query, false
		}
		return httpapi.Page(filterBankTransactionsByDate(rows, query), query), query, true
	}
	orgID := a.ensureOrganization(userID(r)).ID
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.BankTransaction{}
	for _, row := range a.store.bankTransactions {
		if row.OrganizationID == orgID {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PostedAt.After(out[j].PostedAt) })
	return httpapi.Page(filterBankTransactionsByDate(out, query), query), query, true
}

func (a *app) syncStripeMock(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return
		}
		requestHash := a.idempotencyHash(r, org.ID)
		if a.replayIdempotentResponse(w, r, org.ID, requestHash) {
			return
		}
		payload, err := a.cfRepo.SyncStripeDemo(r.Context(), org, userID(r))
		if err != nil {
			a.logOperation(r, "stripe.sync.failed", org.ID, map[string]interface{}{"error": err.Error(), "storage": "postgres", "latency_ms": time.Since(start).Milliseconds()})
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not sync Stripe sample data")
			return
		}
		a.writeIdempotentJSON(w, r, org.ID, requestHash, 200, payload)
		a.logOperation(r, "stripe.sync.completed", org.ID, map[string]interface{}{"organization_id": org.ID, "storage": "postgres", "latency_ms": time.Since(start).Milliseconds()})
		return
	}
	org := a.ensureOrganization(userID(r))
	now := time.Now().UTC()
	payments := []models.Payment{
		{ID: auth.NewID(), OrganizationID: org.ID, Processor: "stripe", ProcessorPaymentID: "ch_hoodie_001", CustomerEmail: "buyer1@example.com", Amount: 48, Currency: org.Currency, Status: "succeeded", OccurredAt: now.AddDate(0, 0, -7), Description: "Hoodie order", CreatedAt: now},
		{ID: auth.NewID(), OrganizationID: org.ID, Processor: "stripe", ProcessorPaymentID: "ch_hoodie_002", CustomerEmail: "buyer2@example.com", Amount: 48, Currency: org.Currency, Status: "succeeded", OccurredAt: now.AddDate(0, 0, -7), Description: "Hoodie order", CreatedAt: now},
		{ID: auth.NewID(), OrganizationID: org.ID, Processor: "stripe", ProcessorPaymentID: "ch_ticket_001", CustomerEmail: "guest@example.com", Amount: 35, Currency: org.Currency, Status: "succeeded", OccurredAt: now.AddDate(0, 0, -6), Description: "Event ticket", CreatedAt: now},
		{ID: auth.NewID(), OrganizationID: org.ID, Processor: "stripe", ProcessorPaymentID: "ch_dues_001", CustomerEmail: "member@example.com", Amount: 120, Currency: org.Currency, Status: "succeeded", OccurredAt: now.AddDate(0, 0, -5), Description: "Semester dues", CreatedAt: now},
		{ID: auth.NewID(), OrganizationID: org.ID, Processor: "stripe", ProcessorPaymentID: "ch_sponsor_001", CustomerEmail: "sponsor@example.com", Amount: 1500, Currency: org.Currency, Status: "succeeded", OccurredAt: now.AddDate(0, 0, -4), Description: "Event sponsorship", CreatedAt: now},
	}
	refund := models.Refund{ID: auth.NewID(), OrganizationID: org.ID, ProcessorRefundID: "re_ticket_001", PaymentID: payments[2].ID, Amount: 35, Currency: org.Currency, OccurredAt: now.AddDate(0, 0, -3)}
	fees := []models.Fee{
		{ID: auth.NewID(), OrganizationID: org.ID, ProcessorFeeID: "fee_001", PaymentID: payments[0].ID, Amount: 1.69, Currency: org.Currency, OccurredAt: payments[0].OccurredAt, Description: "Stripe processing fee"},
		{ID: auth.NewID(), OrganizationID: org.ID, ProcessorFeeID: "fee_002", PaymentID: payments[1].ID, Amount: 1.69, Currency: org.Currency, OccurredAt: payments[1].OccurredAt, Description: "Stripe processing fee"},
		{ID: auth.NewID(), OrganizationID: org.ID, ProcessorFeeID: "fee_003", PaymentID: payments[2].ID, Amount: 1.32, Currency: org.Currency, OccurredAt: payments[2].OccurredAt, Description: "Stripe processing fee"},
		{ID: auth.NewID(), OrganizationID: org.ID, ProcessorFeeID: "fee_004", PaymentID: payments[3].ID, Amount: 3.78, Currency: org.Currency, OccurredAt: payments[3].OccurredAt, Description: "Stripe processing fee"},
		{ID: auth.NewID(), OrganizationID: org.ID, ProcessorFeeID: "fee_005", PaymentID: payments[4].ID, Amount: 43.80, Currency: org.Currency, OccurredAt: payments[4].OccurredAt, Description: "Stripe processing fee"},
	}
	net := 0.0
	for _, payment := range payments {
		net += payment.Amount
	}
	net -= refund.Amount
	for _, fee := range fees {
		net -= fee.Amount
	}
	payout := models.Payout{ID: auth.NewID(), OrganizationID: org.ID, Processor: "stripe", ProcessorPayoutID: "po_demo_001", Amount: round2(net), Currency: org.Currency, Status: "paid", ExpectedArrivalAt: now.AddDate(0, 0, -2), CreatedAt: now.AddDate(0, 0, -2)}
	a.store.mu.Lock()
	for _, payment := range payments {
		a.upsertPaymentLocked(payment)
	}
	a.store.refunds[refund.ID] = refund
	for _, fee := range fees {
		a.store.fees[fee.ID] = fee
	}
	a.upsertPayoutLocked(payout)
	a.auditLocked(org.ID, userID(r), "stripe.mock_synced", "payout", payout.ID)
	a.store.mu.Unlock()
	a.logOperation(r, "stripe.sync.completed", org.ID, map[string]interface{}{"organization_id": org.ID, "payments": len(payments), "refunds": 1, "fees": len(fees), "payout_id": payout.ID, "payout_amount": payout.Amount, "latency_ms": time.Since(start).Milliseconds()})
	writeJSON(w, 200, map[string]interface{}{"payments": len(payments), "refunds": 1, "fees": len(fees), "payout": payout})
}

func (a *app) syncBankMock(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return
		}
		requestHash := a.idempotencyHash(r, org.ID)
		if a.replayIdempotentResponse(w, r, org.ID, requestHash) {
			return
		}
		payload, err := a.cfRepo.SyncBankDemo(r.Context(), org, userID(r))
		if err != nil {
			a.logOperation(r, "bank.sync.failed", org.ID, map[string]interface{}{"error": err.Error(), "storage": "postgres", "latency_ms": time.Since(start).Milliseconds()})
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not sync bank sample data")
			return
		}
		a.writeIdempotentJSON(w, r, org.ID, requestHash, 200, payload)
		a.logOperation(r, "bank.sync.completed", org.ID, map[string]interface{}{"organization_id": org.ID, "storage": "postgres", "latency_ms": time.Since(start).Milliseconds()})
		return
	}
	org := a.ensureOrganization(userID(r))
	now := time.Now().UTC()
	payoutAmount := 0.0
	a.store.mu.RLock()
	for _, payout := range a.store.payouts {
		if payout.OrganizationID == org.ID && payoutAmount == 0 {
			payoutAmount = payout.Amount
		}
	}
	a.store.mu.RUnlock()
	if payoutAmount == 0 {
		payoutAmount = 1665.72
	}
	rows := []models.BankTransaction{
		{ID: auth.NewID(), OrganizationID: org.ID, Source: "plaid_or_csv", ExternalID: "bank_stripe_demo_001", Amount: round2(payoutAmount), Direction: "credit", Currency: org.Currency, Description: "STRIPE PAYOUT", PostedAt: now.AddDate(0, 0, -2), CreatedAt: now},
		{ID: auth.NewID(), OrganizationID: org.ID, Source: "plaid_or_csv", ExternalID: "bank_venue_001", Amount: 300, Direction: "debit", Currency: org.Currency, Description: "Venue deposit", PostedAt: now.AddDate(0, 0, -1), CreatedAt: now},
		{ID: auth.NewID(), OrganizationID: org.ID, Source: "plaid_or_csv", ExternalID: "bank_unmatched_001", Amount: 212.45, Direction: "credit", Currency: org.Currency, Description: "Unknown deposit", PostedAt: now.AddDate(0, 0, -1), CreatedAt: now},
	}
	a.store.mu.Lock()
	for _, row := range rows {
		a.upsertBankTransactionLocked(row)
	}
	a.auditLocked(org.ID, userID(r), "bank.mock_synced", "bank_transaction", rows[0].ID)
	a.store.mu.Unlock()
	a.logOperation(r, "bank.sync.completed", org.ID, map[string]interface{}{"organization_id": org.ID, "bank_transactions": len(rows), "stripe_deposit_amount": rows[0].Amount, "latency_ms": time.Since(start).Milliseconds()})
	writeJSON(w, 200, map[string]interface{}{"bank_transactions": len(rows)})
}

func (a *app) syncStripeMockV1(w http.ResponseWriter, r *http.Request) {
	a.enqueueFinancialJob(w, r, "stripe.sync")
}

func (a *app) syncBankMockV1(w http.ResponseWriter, r *http.Request) {
	a.enqueueFinancialJob(w, r, "bank.sync")
}

func (a *app) createReconciliationRun(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return
		}
		requestHash := a.idempotencyHash(r, org.ID)
		if a.replayIdempotentResponse(w, r, org.ID, requestHash) {
			return
		}
		run, err := a.cfRepo.Reconcile(r.Context(), org.ID, userID(r))
		if err != nil {
			a.logOperation(r, "reconciliation.run.failed", org.ID, map[string]interface{}{"error": err.Error(), "storage": "postgres", "latency_ms": time.Since(start).Milliseconds()})
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not run reconciliation")
			return
		}
		a.writeIdempotentJSON(w, r, org.ID, requestHash, 201, run)
		a.logOperation(r, "reconciliation.run.created", org.ID, map[string]interface{}{"organization_id": org.ID, "run_id": run.ID, "matched_count": run.MatchedCount, "exception_count": run.ExceptionCount, "storage": "postgres", "latency_ms": time.Since(start).Milliseconds()})
		return
	}
	org := a.ensureOrganization(userID(r))
	run := a.reconcileOrganization(org.ID, userID(r))
	a.logOperation(r, "reconciliation.run.created", org.ID, map[string]interface{}{"organization_id": org.ID, "run_id": run.ID, "matched_count": run.MatchedCount, "exception_count": run.ExceptionCount, "latency_ms": time.Since(start).Milliseconds()})
	writeJSON(w, 201, run)
}

func (a *app) createReconciliationRunV1(w http.ResponseWriter, r *http.Request) {
	r, ok := a.withV1Organization(w, r, true, authz.CanRunReconciliation)
	if !ok {
		return
	}
	a.enqueueFinancialJob(w, r, "reconciliation.run")
}

func (a *app) listReconciliationRuns(w http.ResponseWriter, r *http.Request) {
	rows, _, ok := a.reconciliationRunsForRequest(w, r)
	if !ok {
		return
	}
	writeJSON(w, 200, rows)
}

func (a *app) listReconciliationRunsV1(w http.ResponseWriter, r *http.Request) {
	if r, ok := a.withV1Organization(w, r, false, authz.CanRead); ok {
		a.listReconciliationRunsV1Response(w, r)
	}
}

func (a *app) listReconciliationRunsV1Response(w http.ResponseWriter, r *http.Request) {
	rows, query, ok := a.reconciliationRunsForRequest(w, r)
	if !ok {
		return
	}
	writeJSON(w, 200, httpapi.Paginated(rows, query))
}

func (a *app) reconciliationRunsForRequest(w http.ResponseWriter, r *http.Request) ([]models.ReconciliationRun, httpapi.ListQuery, bool) {
	query, ok := parseClearflowListQuery(w, r)
	if !ok {
		return nil, query, false
	}
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return nil, query, false
		}
		rows, err := a.cfRepo.ListReconciliationRuns(r.Context(), org.ID)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not list reconciliation runs")
			return nil, query, false
		}
		return httpapi.Page(filterReconciliationRunsByDate(rows, query), query), query, true
	}
	orgID := a.ensureOrganization(userID(r)).ID
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.ReconciliationRun{}
	for _, row := range a.store.reconciliationRuns {
		if row.OrganizationID == orgID {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	return httpapi.Page(filterReconciliationRunsByDate(out, query), query), query, true
}

func (a *app) getReconciliationRun(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return
		}
		payload, err := a.cfRepo.GetReconciliationRun(r.Context(), org.ID, r.PathValue("id"))
		if err != nil {
			errorJSON(w, r, 404, "NOT_FOUND", "reconciliation run not found")
			return
		}
		writeJSON(w, 200, payload)
		return
	}
	orgID := a.ensureOrganization(userID(r)).ID
	id := r.PathValue("id")
	a.store.mu.RLock()
	run, ok := a.store.reconciliationRuns[id]
	matches := []models.ReconciliationMatch{}
	exceptions := []models.ReconciliationException{}
	for _, row := range a.store.reconciliationMatches {
		if row.RunID == id {
			matches = append(matches, row)
		}
	}
	for _, row := range a.store.reconciliationExceptions {
		if row.RunID == id {
			exceptions = append(exceptions, row)
		}
	}
	a.store.mu.RUnlock()
	if !ok || run.OrganizationID != orgID {
		errorJSON(w, r, 404, "NOT_FOUND", "reconciliation run not found")
		return
	}
	writeJSON(w, 200, map[string]interface{}{"run": run, "matches": matches, "exceptions": exceptions})
}

func (a *app) listReconciliationExceptions(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return
		}
		rows, err := a.cfRepo.ListExceptions(r.Context(), org.ID)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not list exceptions")
			return
		}
		writeJSON(w, 200, rows)
		return
	}
	orgID := a.ensureOrganization(userID(r)).ID
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.ReconciliationException{}
	for _, row := range a.store.reconciliationExceptions {
		if row.OrganizationID == orgID {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	writeJSON(w, 200, out)
}

func (a *app) patchReconciliationException(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req struct {
		Status                   string `json:"status"`
		Note                     string `json:"note"`
		MatchedBankTransactionID string `json:"matched_bank_transaction_id"`
	}
	if !decode(w, r, &req) {
		return
	}
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return
		}
		row, err := a.cfRepo.UpdateException(r.Context(), org.ID, userID(r), r.PathValue("id"), req.Status)
		if err != nil {
			errorJSON(w, r, 404, "NOT_FOUND", "exception not found")
			return
		}
		if strings.TrimSpace(req.Note) != "" {
			noteBody := strings.TrimSpace(req.Note)
			if req.MatchedBankTransactionID != "" {
				noteBody += " (manual bank match: " + req.MatchedBankTransactionID + ")"
			}
			if _, err := a.cfRepo.AddExceptionNote(r.Context(), org.ID, userID(r), row.ID, noteBody); err != nil {
				errorJSON(w, r, 500, "DATABASE_ERROR", "could not save exception note")
				return
			}
		}
		a.logOperation(r, "reconciliation_exception.updated", org.ID, map[string]interface{}{"organization_id": org.ID, "exception_id": row.ID, "status": row.Status, "storage": "postgres", "latency_ms": time.Since(start).Milliseconds()})
		writeJSON(w, 200, row)
		return
	}
	orgID := a.ensureOrganization(userID(r)).ID
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	row, ok := a.store.reconciliationExceptions[r.PathValue("id")]
	if !ok || row.OrganizationID != orgID {
		errorJSON(w, r, 404, "NOT_FOUND", "exception not found")
		return
	}
	if req.Status == "" {
		req.Status = "resolved"
	}
	row.Status = req.Status
	a.store.reconciliationExceptions[row.ID] = row
	if strings.TrimSpace(req.Note) != "" {
		body := strings.TrimSpace(req.Note)
		if req.MatchedBankTransactionID != "" {
			body += " (manual bank match: " + req.MatchedBankTransactionID + ")"
		}
		note := models.ExceptionNote{ID: auth.NewID(), OrganizationID: orgID, ExceptionID: row.ID, UserID: userID(r), Body: body, CreatedAt: time.Now().UTC()}
		a.store.exceptionNotes[note.ID] = note
	}
	a.auditLocked(orgID, userID(r), "reconciliation_exception.updated", "reconciliation_exception", row.ID)
	a.logOperation(r, "reconciliation_exception.updated", orgID, map[string]interface{}{"organization_id": orgID, "exception_id": row.ID, "status": row.Status, "latency_ms": time.Since(start).Milliseconds()})
	writeJSON(w, 200, row)
}

func (a *app) listExceptionNotes(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return
		}
		rows, err := a.cfRepo.ListExceptionNotes(r.Context(), org.ID, r.PathValue("id"))
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not list exception notes")
			return
		}
		writeJSON(w, 200, rows)
		return
	}
	orgID := a.ensureOrganization(userID(r)).ID
	exceptionID := r.PathValue("id")
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.ExceptionNote{}
	for _, note := range a.store.exceptionNotes {
		if note.OrganizationID == orgID && note.ExceptionID == exceptionID {
			out = append(out, note)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	writeJSON(w, 200, out)
}

func (a *app) addExceptionNote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Body string `json:"body"`
	}
	if !decode(w, r, &req) {
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		errorJSON(w, r, 400, "VALIDATION_ERROR", "body is required")
		return
	}
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return
		}
		note, err := a.cfRepo.AddExceptionNote(r.Context(), org.ID, userID(r), r.PathValue("id"), req.Body)
		if err != nil {
			errorJSON(w, r, 404, "NOT_FOUND", "exception not found")
			return
		}
		writeJSON(w, 201, note)
		return
	}
	orgID := a.ensureOrganization(userID(r)).ID
	exceptionID := r.PathValue("id")
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	ex, ok := a.store.reconciliationExceptions[exceptionID]
	if !ok || ex.OrganizationID != orgID {
		errorJSON(w, r, 404, "NOT_FOUND", "exception not found")
		return
	}
	note := models.ExceptionNote{ID: auth.NewID(), OrganizationID: orgID, ExceptionID: exceptionID, UserID: userID(r), Body: req.Body, CreatedAt: time.Now().UTC()}
	a.store.exceptionNotes[note.ID] = note
	a.auditLocked(orgID, userID(r), "reconciliation_exception.note_added", "reconciliation_exception", exceptionID)
	writeJSON(w, 201, note)
}

func (a *app) onboardingStatusV1(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return
		}
		status, err := a.cfRepo.OnboardingStatus(r.Context(), org.ID)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not load onboarding status")
			return
		}
		if status.BusinessType == "" {
			status.BusinessType = org.Type
		}
		writeJSON(w, 200, status)
		return
	}
	org := a.ensureOrganization(userID(r))
	a.store.mu.RLock()
	status, ok := a.store.organizationSetup[org.ID]
	payouts, bank, team, openBreaks := 0, 0, 0, 0
	for _, row := range a.store.payouts {
		if row.OrganizationID == org.ID {
			payouts++
		}
	}
	for _, row := range a.store.bankTransactions {
		if row.OrganizationID == org.ID {
			bank++
		}
	}
	for _, row := range a.store.organizationMembers {
		if row.OrganizationID == org.ID {
			team++
		}
	}
	for _, row := range a.store.reconciliationExceptions {
		if row.OrganizationID == org.ID && row.Status == "open" {
			openBreaks++
		}
	}
	a.store.mu.RUnlock()
	if !ok {
		status = models.OrganizationSetup{OrganizationID: org.ID, BusinessType: org.Type, Checklist: map[string]interface{}{}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	}
	status.ProviderReadiness = map[string]interface{}{"workspace_created": true, "processor_data_ready": payouts > 0, "bank_data_ready": bank > 0, "team_ready": team > 0, "open_breaks": openBreaks}
	if status.Checklist == nil {
		status.Checklist = map[string]interface{}{}
	}
	for key, value := range status.ProviderReadiness {
		status.Checklist[key] = value
	}
	writeJSON(w, 200, status)
}

func (a *app) updateOnboardingStatusV1(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SelectedScenario string                 `json:"selected_scenario"`
		BusinessType     string                 `json:"business_type"`
		Checklist        map[string]interface{} `json:"checklist"`
		Completed        bool                   `json:"completed"`
	}
	if !decode(w, r, &req) {
		return
	}
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return
		}
		status, err := a.cfRepo.UpsertOnboardingStatus(r.Context(), org.ID, req.SelectedScenario, req.BusinessType, req.Checklist, req.Completed)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not update onboarding status")
			return
		}
		a.writeAudit(r.Context(), r, org.ID, userID(r), "onboarding.updated", "organization", org.ID, "{}")
		writeJSON(w, 200, status)
		return
	}
	org := a.ensureOrganization(userID(r))
	now := time.Now().UTC()
	a.store.mu.Lock()
	status := a.store.organizationSetup[org.ID]
	if status.OrganizationID == "" {
		status = models.OrganizationSetup{OrganizationID: org.ID, CreatedAt: now}
	}
	status.SelectedScenario = req.SelectedScenario
	status.BusinessType = fallback(req.BusinessType, org.Type)
	status.Checklist = req.Checklist
	if status.Checklist == nil {
		status.Checklist = map[string]interface{}{}
	}
	if req.Completed {
		status.CompletedAt = now
	}
	status.UpdatedAt = now
	a.store.organizationSetup[org.ID] = status
	a.auditLocked(org.ID, userID(r), "onboarding.updated", "organization", org.ID)
	a.store.mu.Unlock()
	writeJSON(w, 200, status)
}

func (a *app) clearflowCashSummary(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return
		}
		payload, err := a.cfRepo.CashSummary(r.Context(), org.ID)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not build cash summary")
			return
		}
		writeJSON(w, 200, payload)
		return
	}
	orgID := a.ensureOrganization(userID(r)).ID
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	var cash, income, expenses, pendingPayouts, fees, refunds float64
	for _, row := range a.store.bankTransactions {
		if row.OrganizationID != orgID {
			continue
		}
		if row.Direction == "credit" {
			cash += row.Amount
			income += row.Amount
		} else {
			cash -= row.Amount
			expenses += row.Amount
		}
	}
	for _, payout := range a.store.payouts {
		if payout.OrganizationID == orgID && payout.Status != "paid" {
			pendingPayouts += payout.Amount
		}
	}
	for _, fee := range a.store.fees {
		if fee.OrganizationID == orgID {
			fees += fee.Amount
		}
	}
	for _, refund := range a.store.refunds {
		if refund.OrganizationID == orgID {
			refunds += refund.Amount
		}
	}
	writeJSON(w, 200, map[string]float64{"cash_balance": round2(cash), "income": round2(income), "expenses": round2(expenses), "pending_payouts": round2(pendingPayouts), "fees": round2(fees), "refunds": round2(refunds), "net_cash_flow": round2(income - expenses - fees - refunds)})
}

func (a *app) clearflowCashSummaryV1(w http.ResponseWriter, r *http.Request) {
	if r, ok := a.withV1Organization(w, r, false, authz.CanRead); ok {
		a.clearflowCashSummary(w, r)
	}
}

func (a *app) clearflowCashForecast(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return
		}
		payload, err := a.cfRepo.CashForecast(r.Context(), org.ID)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not build cash forecast")
			return
		}
		writeJSON(w, 200, payload)
		return
	}
	orgID := a.ensureOrganization(userID(r)).ID
	summary := a.cashSnapshot(orgID)
	points := []map[string]interface{}{}
	for _, days := range []int{7, 30, 60} {
		expectedPayouts := a.expectedPayouts(orgID, days)
		expectedExpenses := 0.0
		if days >= 30 {
			expectedExpenses = 450
		}
		points = append(points, map[string]interface{}{"days": days, "projected_cash": round2(summary + expectedPayouts - expectedExpenses), "expected_payouts": round2(expectedPayouts), "expected_expenses": expectedExpenses})
	}
	writeJSON(w, 200, points)
}

func (a *app) clearflowCashForecastV1(w http.ResponseWriter, r *http.Request) {
	if r, ok := a.withV1Organization(w, r, false, authz.CanRead); ok {
		a.clearflowCashForecast(w, r)
	}
}

func (a *app) clearflowMonthlyReport(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return
		}
		payload, err := a.cfRepo.MonthlyReport(r.Context(), org.ID)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not build monthly report")
			return
		}
		writeJSON(w, 200, payload)
		return
	}
	orgID := a.ensureOrganization(userID(r)).ID
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	var gross, refunds, fees, payouts float64
	for _, row := range a.store.payments {
		if row.OrganizationID == orgID && row.Status == "succeeded" {
			gross += row.Amount
		}
	}
	for _, row := range a.store.refunds {
		if row.OrganizationID == orgID {
			refunds += row.Amount
		}
	}
	for _, row := range a.store.fees {
		if row.OrganizationID == orgID {
			fees += row.Amount
		}
	}
	for _, row := range a.store.payouts {
		if row.OrganizationID == orgID {
			payouts += row.Amount
		}
	}
	writeJSON(w, 200, map[string]interface{}{"gross_payments": round2(gross), "refunds": round2(refunds), "fees": round2(fees), "net_processor_activity": round2(gross - refunds - fees), "payouts": round2(payouts), "month": time.Now().UTC().Format("2006-01")})
}

func (a *app) debugClearflowState(w http.ResponseWriter, r *http.Request) {
	if a.cfRepo != nil {
		org, ok := a.clearflowOrganizationForRequest(w, r)
		if !ok {
			return
		}
		payload, err := a.cfRepo.DebugState(r.Context(), org)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not build debug state")
			return
		}
		writeJSON(w, 200, payload)
		return
	}
	org := a.ensureOrganization(userID(r))
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	counts := map[string]int{
		"payments":     0,
		"refunds":      0,
		"fees":         0,
		"payouts":      0,
		"bank_tx":      0,
		"runs":         0,
		"matches":      0,
		"exceptions":   0,
		"open_breaks":  0,
		"audit_events": 0,
	}
	for _, row := range a.store.payments {
		if row.OrganizationID == org.ID {
			counts["payments"]++
		}
	}
	for _, row := range a.store.refunds {
		if row.OrganizationID == org.ID {
			counts["refunds"]++
		}
	}
	for _, row := range a.store.fees {
		if row.OrganizationID == org.ID {
			counts["fees"]++
		}
	}
	for _, row := range a.store.payouts {
		if row.OrganizationID == org.ID {
			counts["payouts"]++
		}
	}
	for _, row := range a.store.bankTransactions {
		if row.OrganizationID == org.ID {
			counts["bank_tx"]++
		}
	}
	for _, row := range a.store.reconciliationRuns {
		if row.OrganizationID == org.ID {
			counts["runs"]++
		}
	}
	for _, row := range a.store.reconciliationMatches {
		if row.OrganizationID == org.ID {
			counts["matches"]++
		}
	}
	for _, row := range a.store.reconciliationExceptions {
		if row.OrganizationID == org.ID {
			counts["exceptions"]++
			if row.Status == "open" {
				counts["open_breaks"]++
			}
		}
	}
	for _, row := range a.store.auditLogs {
		if row.OrganizationID == org.ID {
			counts["audit_events"]++
		}
	}
	writeJSON(w, 200, map[string]interface{}{"organization": org, "counts": counts, "debug_note": "No secrets are included. Pair this with terminal JSON logs when reporting bugs."})
}

func (a *app) resetClearflowDemo(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	if a.cfRepo != nil {
		a.store.mu.RLock()
		user := a.store.users[uid]
		a.store.mu.RUnlock()
		if user.ID == "" {
			user = models.User{ID: uid, Email: uid + "@clearflow.local", CreatedAt: time.Now().UTC()}
		}
		payload, err := a.cfRepo.ResetDemo(r.Context(), user)
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not reset demo data")
			return
		}
		writeJSON(w, 200, payload)
		return
	}
	org := a.ensureOrganization(uid)
	a.store.mu.Lock()
	for id, row := range a.store.payments {
		if row.OrganizationID == org.ID {
			delete(a.store.payments, id)
		}
	}
	for id, row := range a.store.refunds {
		if row.OrganizationID == org.ID {
			delete(a.store.refunds, id)
		}
	}
	for id, row := range a.store.fees {
		if row.OrganizationID == org.ID {
			delete(a.store.fees, id)
		}
	}
	for id, row := range a.store.payouts {
		if row.OrganizationID == org.ID {
			delete(a.store.payouts, id)
		}
	}
	for id, row := range a.store.bankTransactions {
		if row.OrganizationID == org.ID {
			delete(a.store.bankTransactions, id)
		}
	}
	for id, row := range a.store.reconciliationRuns {
		if row.OrganizationID == org.ID {
			delete(a.store.reconciliationRuns, id)
		}
	}
	for id, row := range a.store.reconciliationMatches {
		if row.OrganizationID == org.ID {
			delete(a.store.reconciliationMatches, id)
		}
	}
	for id, row := range a.store.reconciliationExceptions {
		if row.OrganizationID == org.ID {
			delete(a.store.reconciliationExceptions, id)
		}
	}
	for id, row := range a.store.jobs {
		if row.OrganizationID == org.ID {
			delete(a.store.jobs, id)
		}
	}
	for key, row := range a.store.idempotencyRecords {
		if row.OrganizationID == org.ID {
			delete(a.store.idempotencyRecords, key)
		}
	}
	a.store.mu.Unlock()
	a.seedClearflowDemo(uid)
	writeJSON(w, 200, map[string]interface{}{"organization_id": org.ID, "status": "reset"})
}

func (a *app) reconcileOrganization(orgID, uid string) models.ReconciliationRun {
	now := time.Now().UTC()
	run := models.ReconciliationRun{ID: auth.NewID(), OrganizationID: orgID, Status: "completed", StartedAt: now}
	evaluatedPayouts := 0
	evaluatedDeposits := 0
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	matchedBanks := map[string]bool{}
	for _, payout := range a.store.payouts {
		if payout.OrganizationID != orgID {
			continue
		}
		evaluatedPayouts++
		best := models.BankTransaction{}
		bestScore := 0.0
		for _, bank := range a.store.bankTransactions {
			if bank.OrganizationID != orgID || bank.Direction != "credit" || matchedBanks[bank.ID] {
				continue
			}
			evaluatedDeposits++
			amountScore := 1 - math.Min(1, math.Abs(bank.Amount-payout.Amount)/math.Max(payout.Amount, 1))
			dateDistance := math.Abs(bank.PostedAt.Sub(payout.ExpectedArrivalAt).Hours()) / 24
			dateScore := math.Max(0, 1-dateDistance/5)
			descriptionScore := 0.0
			if strings.Contains(strings.ToLower(bank.Description), strings.ToLower(payout.Processor)) {
				descriptionScore = 0.2
			}
			score := amountScore*0.7 + dateScore*0.2 + descriptionScore
			if score > bestScore {
				bestScore = score
				best = bank
			}
		}
		if best.ID != "" && bestScore >= 0.88 {
			match := models.ReconciliationMatch{ID: auth.NewID(), OrganizationID: orgID, RunID: run.ID, PayoutID: payout.ID, BankTransactionID: best.ID, Amount: payout.Amount, Confidence: round2(bestScore), Explanation: fmt.Sprintf("Matched %s payout %s to bank deposit %s by amount/date/description.", payout.Processor, payout.ProcessorPayoutID, best.ExternalID), CreatedAt: now}
			a.store.reconciliationMatches[match.ID] = match
			matchedBanks[best.ID] = true
			run.MatchedCount++
		} else {
			a.addExceptionLocked(orgID, run.ID, "unmatched_payout", "high", "Unmatched payout", fmt.Sprintf("Payout %s for $%.2f did not match any bank deposit.", payout.ProcessorPayoutID, payout.Amount), payout.ID)
			run.ExceptionCount++
		}
	}
	for _, bank := range a.store.bankTransactions {
		if bank.OrganizationID == orgID && bank.Direction == "credit" && !matchedBanks[bank.ID] {
			a.addExceptionLocked(orgID, run.ID, "unmatched_deposit", "medium", "Unmatched bank deposit", fmt.Sprintf("Bank deposit %s for $%.2f is not tied to a known payout.", bank.Description, bank.Amount), bank.ID)
			run.ExceptionCount++
		}
	}
	run.CompletedAt = time.Now().UTC()
	a.store.reconciliationRuns[run.ID] = run
	a.auditLocked(orgID, uid, "reconciliation.run_completed", "reconciliation_run", run.ID)
	a.log.Info("reconciliation.engine.completed", map[string]interface{}{"organization_id": orgID, "run_id": run.ID, "evaluated_payouts": evaluatedPayouts, "evaluated_deposit_candidates": evaluatedDeposits, "matched_count": run.MatchedCount, "exception_count": run.ExceptionCount, "duration_ms": run.CompletedAt.Sub(run.StartedAt).Milliseconds()})
	return run
}

func (a *app) ensureOrganization(uid string) models.Organization {
	orgs := a.userOrganizations(uid)
	if len(orgs) > 0 {
		return orgs[0]
	}
	a.store.mu.Lock()
	org := a.ensureOrganizationLocked(uid, "")
	a.addOrganizationMemberLocked(org.ID, uid, authz.RoleOwner)
	a.store.mu.Unlock()
	return org
}

func (a *app) seedClearflowDemo(uid string) {
	if a.cfRepo != nil {
		u := models.User{ID: uid, Email: uid + "@clearflow.local", CreatedAt: time.Now().UTC()}
		a.store.mu.RLock()
		if existing, ok := a.store.users[uid]; ok {
			u = existing
		}
		a.store.mu.RUnlock()
		if err := a.cfRepo.SeedDemo(context.Background(), u); err != nil {
			a.log.Error("clearflow.demo_seed_failed", map[string]interface{}{"user_id": uid, "error": err.Error(), "storage": "postgres"})
		}
		return
	}
	org := a.ensureOrganization(uid)
	a.store.mu.RLock()
	hasData := false
	for _, payout := range a.store.payouts {
		if payout.OrganizationID == org.ID {
			hasData = true
			break
		}
	}
	a.store.mu.RUnlock()
	if hasData {
		return
	}
	now := time.Now().UTC()
	payments := []models.Payment{
		{ID: auth.NewID(), OrganizationID: org.ID, Processor: "stripe", ProcessorPaymentID: "ch_demo_membership", CustomerEmail: "member@example.com", Amount: 120, Currency: org.Currency, Status: "succeeded", OccurredAt: now.AddDate(0, 0, -8), Description: "Membership dues", CreatedAt: now},
		{ID: auth.NewID(), OrganizationID: org.ID, Processor: "stripe", ProcessorPaymentID: "ch_demo_sponsor", CustomerEmail: "sponsor@example.com", Amount: 1500, Currency: org.Currency, Status: "succeeded", OccurredAt: now.AddDate(0, 0, -7), Description: "Event sponsorship", CreatedAt: now},
		{ID: auth.NewID(), OrganizationID: org.ID, Processor: "stripe", ProcessorPaymentID: "ch_demo_ticket", CustomerEmail: "guest@example.com", Amount: 35, Currency: org.Currency, Status: "succeeded", OccurredAt: now.AddDate(0, 0, -7), Description: "Event ticket", CreatedAt: now},
	}
	fees := []models.Fee{
		{ID: auth.NewID(), OrganizationID: org.ID, ProcessorFeeID: "fee_demo_membership", PaymentID: payments[0].ID, Amount: 3.78, Currency: org.Currency, OccurredAt: payments[0].OccurredAt, Description: "Stripe processing fee"},
		{ID: auth.NewID(), OrganizationID: org.ID, ProcessorFeeID: "fee_demo_sponsor", PaymentID: payments[1].ID, Amount: 43.80, Currency: org.Currency, OccurredAt: payments[1].OccurredAt, Description: "Stripe processing fee"},
		{ID: auth.NewID(), OrganizationID: org.ID, ProcessorFeeID: "fee_demo_ticket", PaymentID: payments[2].ID, Amount: 1.32, Currency: org.Currency, OccurredAt: payments[2].OccurredAt, Description: "Stripe processing fee"},
	}
	refund := models.Refund{ID: auth.NewID(), OrganizationID: org.ID, ProcessorRefundID: "re_demo_ticket", PaymentID: payments[2].ID, Amount: 35, Currency: org.Currency, OccurredAt: now.AddDate(0, 0, -6)}
	payoutAmount := round2(payments[0].Amount + payments[1].Amount + payments[2].Amount - fees[0].Amount - fees[1].Amount - fees[2].Amount - refund.Amount)
	payout := models.Payout{ID: auth.NewID(), OrganizationID: org.ID, Processor: "stripe", ProcessorPayoutID: "po_demo_seed", Amount: payoutAmount, Currency: org.Currency, Status: "paid", ExpectedArrivalAt: now.AddDate(0, 0, -4), CreatedAt: now.AddDate(0, 0, -4)}
	bank := []models.BankTransaction{
		{ID: auth.NewID(), OrganizationID: org.ID, Source: "plaid_or_csv", ExternalID: "bank_demo_stripe", Amount: payoutAmount, Direction: "credit", Currency: org.Currency, Description: "STRIPE PAYOUT", PostedAt: now.AddDate(0, 0, -4), CreatedAt: now},
		{ID: auth.NewID(), OrganizationID: org.ID, Source: "plaid_or_csv", ExternalID: "bank_demo_unmatched", Amount: 212.45, Direction: "credit", Currency: org.Currency, Description: "Unknown deposit", PostedAt: now.AddDate(0, 0, -2), CreatedAt: now},
		{ID: auth.NewID(), OrganizationID: org.ID, Source: "plaid_or_csv", ExternalID: "bank_demo_venue", Amount: 300, Direction: "debit", Currency: org.Currency, Description: "Venue deposit", PostedAt: now.AddDate(0, 0, -1), CreatedAt: now},
	}
	a.store.mu.Lock()
	for _, payment := range payments {
		a.store.payments[payment.ID] = payment
	}
	for _, fee := range fees {
		a.store.fees[fee.ID] = fee
	}
	a.store.refunds[refund.ID] = refund
	a.store.payouts[payout.ID] = payout
	for _, row := range bank {
		a.store.bankTransactions[row.ID] = row
	}
	a.store.mu.Unlock()
	a.reconcileOrganization(org.ID, uid)
}

func (a *app) userOrganizations(uid string) []models.Organization {
	if a.cfRepo != nil {
		orgs, err := a.cfRepo.ListOrganizations(context.Background(), uid)
		if err == nil {
			return orgs
		}
		a.log.Error("clearflow.organizations_list_failed", map[string]interface{}{"user_id": uid, "error": err.Error(), "storage": "postgres"})
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.Organization{}
	seen := map[string]bool{}
	for _, member := range a.store.organizationMembers {
		if member.UserID == uid {
			if org, ok := a.store.organizations[member.OrganizationID]; ok {
				out = append(out, org)
				seen[org.ID] = true
			}
		}
	}
	for _, org := range a.store.organizations {
		if org.UserID == uid && !seen[org.ID] {
			out = append(out, org)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (a *app) cashSnapshot(orgID string) float64 {
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	total := 0.0
	for _, row := range a.store.bankTransactions {
		if row.OrganizationID != orgID {
			continue
		}
		if row.Direction == "credit" {
			total += row.Amount
		} else {
			total -= row.Amount
		}
	}
	return total
}

func (a *app) expectedPayouts(orgID string, days int) float64 {
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	cutoff := time.Now().UTC().AddDate(0, 0, days)
	total := 0.0
	for _, row := range a.store.payouts {
		if row.OrganizationID == orgID && row.Status != "paid" && row.ExpectedArrivalAt.Before(cutoff) {
			total += row.Amount
		}
	}
	return total
}

func (a *app) upsertPaymentLocked(row models.Payment) {
	for id, existing := range a.store.payments {
		if existing.OrganizationID == row.OrganizationID && existing.ProcessorPaymentID == row.ProcessorPaymentID {
			row.ID = id
			a.store.payments[id] = row
			return
		}
	}
	a.store.payments[row.ID] = row
}

func (a *app) upsertPayoutLocked(row models.Payout) {
	for id, existing := range a.store.payouts {
		if existing.OrganizationID == row.OrganizationID && existing.ProcessorPayoutID == row.ProcessorPayoutID {
			row.ID = id
			a.store.payouts[id] = row
			return
		}
	}
	a.store.payouts[row.ID] = row
}

func (a *app) upsertBankTransactionLocked(row models.BankTransaction) {
	for id, existing := range a.store.bankTransactions {
		if existing.OrganizationID == row.OrganizationID && existing.ExternalID == row.ExternalID {
			row.ID = id
			a.store.bankTransactions[id] = row
			return
		}
	}
	a.store.bankTransactions[row.ID] = row
}

func (a *app) addExceptionLocked(orgID, runID, kind, severity, title, explanation, referenceID string) {
	exception := models.ReconciliationException{ID: auth.NewID(), OrganizationID: orgID, RunID: runID, Type: kind, Severity: severity, Title: title, Explanation: explanation, Status: "open", ReferenceID: referenceID, CreatedAt: time.Now().UTC()}
	a.store.reconciliationExceptions[exception.ID] = exception
}

func (a *app) auditLocked(orgID, uid, action, targetType, targetID string) {
	log := models.AuditLog{ID: auth.NewID(), OrganizationID: orgID, UserID: uid, Action: action, TargetType: targetType, TargetID: targetID, CreatedAt: time.Now().UTC()}
	a.store.auditLogs[log.ID] = log
}

func (a *app) logOperation(r *http.Request, event, orgID string, fields map[string]interface{}) {
	if fields == nil {
		fields = map[string]interface{}{}
	}
	fields["event"] = event
	fields["organization_id"] = orgID
	fields["request_id"] = r.Header.Get("X-Request-ID")
	fields["user_id"] = userID(r)
	fields["path"] = r.URL.Path
	a.log.Info("clearflow.operation", fields)
}

func parseClearflowListQuery(w http.ResponseWriter, r *http.Request) (httpapi.ListQuery, bool) {
	query, err := httpapi.ParseListQuery(r)
	if err != nil {
		errorJSON(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return query, false
	}
	return query, true
}

func withinDateWindow(value time.Time, query httpapi.ListQuery) bool {
	if query.From != nil && value.Before(*query.From) {
		return false
	}
	if query.To != nil && value.After(query.To.Add(24*time.Hour)) {
		return false
	}
	return true
}

func filterPaymentsByDate(rows []models.Payment, query httpapi.ListQuery) []models.Payment {
	if query.From == nil && query.To == nil {
		return rows
	}
	out := make([]models.Payment, 0, len(rows))
	for _, row := range rows {
		if withinDateWindow(row.OccurredAt, query) {
			out = append(out, row)
		}
	}
	return out
}

func filterPayoutsByDate(rows []models.Payout, query httpapi.ListQuery) []models.Payout {
	if query.From == nil && query.To == nil {
		return rows
	}
	out := make([]models.Payout, 0, len(rows))
	for _, row := range rows {
		if withinDateWindow(row.ExpectedArrivalAt, query) {
			out = append(out, row)
		}
	}
	return out
}

func filterBankTransactionsByDate(rows []models.BankTransaction, query httpapi.ListQuery) []models.BankTransaction {
	if query.From == nil && query.To == nil {
		return rows
	}
	out := make([]models.BankTransaction, 0, len(rows))
	for _, row := range rows {
		if withinDateWindow(row.PostedAt, query) {
			out = append(out, row)
		}
	}
	return out
}

func filterReconciliationRunsByDate(rows []models.ReconciliationRun, query httpapi.ListQuery) []models.ReconciliationRun {
	if query.From == nil && query.To == nil {
		return rows
	}
	out := make([]models.ReconciliationRun, 0, len(rows))
	for _, row := range rows {
		if withinDateWindow(row.StartedAt, query) {
			out = append(out, row)
		}
	}
	return out
}

func organizationIDFromRequest(r *http.Request) string {
	if id := strings.TrimSpace(r.Header.Get("X-Organization-ID")); id != "" {
		return id
	}
	if id := strings.TrimSpace(r.URL.Query().Get("organizationId")); id != "" {
		return id
	}
	return strings.TrimSpace(r.URL.Query().Get("organization_id"))
}

func (a *app) withV1Organization(w http.ResponseWriter, r *http.Request, requireExplicit bool, allowed func(string) bool) (*http.Request, bool) {
	orgID := organizationIDFromRequest(r)
	if orgID == "" && requireExplicit {
		errorJSON(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "organizationId is required")
		return r, false
	}
	if orgID == "" {
		orgs := a.userOrganizations(userID(r))
		if len(orgs) == 0 {
			errorJSON(w, r, http.StatusNotFound, "NOT_FOUND", "organization not found")
			return r, false
		}
		orgID = orgs[0].ID
	}
	if err := validation.UUID(orgID, "organizationId"); err != nil {
		errorJSON(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return r, false
	}
	member, ok := a.requireOrgRole(w, r, orgID, allowed)
	if !ok {
		return r, false
	}
	org := models.Organization{ID: member.OrganizationID, UserID: userID(r), Name: member.OrganizationName, Type: member.OrganizationType, Currency: member.Currency}
	if org.Name == "" {
		for _, candidate := range a.userOrganizations(userID(r)) {
			if candidate.ID == orgID {
				org = candidate
				break
			}
		}
	}
	ctx := context.WithValue(r.Context(), clearflowOrgContextKey{}, org)
	return r.WithContext(ctx), true
}

func (a *app) requireOrgRole(w http.ResponseWriter, r *http.Request, orgID string, allowed func(string) bool) (models.OrganizationMember, bool) {
	member, err := a.membership(r.Context(), userID(r), orgID)
	if err != nil {
		errorJSON(w, r, http.StatusForbidden, "FORBIDDEN", "you do not have access to this organization")
		return models.OrganizationMember{}, false
	}
	if !allowed(member.Role) {
		errorJSON(w, r, http.StatusForbidden, "FORBIDDEN", "your role cannot perform this action")
		return models.OrganizationMember{}, false
	}
	return member, true
}

func (a *app) currentClearflowUser(r *http.Request) models.User {
	if u, ok := a.currentUser(r); ok {
		return u
	}
	return models.User{ID: userID(r), Email: userID(r) + "@clearflow.local", CreatedAt: time.Now().UTC()}
}

func (a *app) clearflowOrganizationForRequest(w http.ResponseWriter, r *http.Request) (models.Organization, bool) {
	if org, ok := r.Context().Value(clearflowOrgContextKey{}).(models.Organization); ok && org.ID != "" {
		return org, true
	}
	u := a.currentClearflowUser(r)
	if organizationIDFromRequest(r) == "" {
		org, err := a.cfRepo.EnsureOrganization(r.Context(), u)
		if err != nil {
			a.logOperation(r, "organization.ensure_failed", "", map[string]interface{}{"error": err.Error(), "storage": "postgres"})
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not load organization")
			return models.Organization{}, false
		}
		return org, true
	}
	orgID := organizationIDFromRequest(r)
	orgs, err := a.cfRepo.ListOrganizations(r.Context(), u.ID)
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not load organizations")
		return models.Organization{}, false
	}
	for _, org := range orgs {
		if org.ID == orgID {
			return org, true
		}
	}
	errorJSON(w, r, 403, "FORBIDDEN", "user does not have access to this organization")
	return models.Organization{}, false
}

func (a *app) idempotencyHash(r *http.Request, orgID string) string {
	sum := sha256.Sum256([]byte(r.Method + "|" + r.URL.Path + "|" + r.URL.RawQuery + "|" + orgID))
	return fmt.Sprintf("%x", sum[:])
}

func (a *app) replayIdempotentResponse(w http.ResponseWriter, r *http.Request, orgID, requestHash string) bool {
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		return false
	}
	status, body, ok, err := a.cfRepo.ReadIdempotency(r.Context(), userID(r), key, requestHash)
	if err == repository.ErrIdempotencyConflict {
		errorJSON(w, r, 409, "IDEMPOTENCY_CONFLICT", "idempotency key was already used for a different request")
		return true
	}
	if err != nil {
		a.logOperation(r, "idempotency.read_failed", orgID, map[string]interface{}{"error": err.Error()})
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not read idempotency key")
		return true
	}
	if !ok {
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Idempotency-Replayed", "true")
	w.WriteHeader(status)
	_, _ = w.Write(body)
	a.logOperation(r, "idempotency.replayed", orgID, map[string]interface{}{"idempotency_key": key, "status": status})
	return true
}

func (a *app) writeIdempotentJSON(w http.ResponseWriter, r *http.Request, orgID, requestHash string, status int, payload interface{}) {
	body := repository.EncodeBody(payload)
	if key := r.Header.Get("Idempotency-Key"); key != "" {
		if err := a.cfRepo.SaveIdempotency(r.Context(), userID(r), orgID, key, requestHash, status, body); err != nil {
			a.logOperation(r, "idempotency.save_failed", orgID, map[string]interface{}{"error": err.Error(), "idempotency_key": key})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func decodeOptional(r *http.Request, dst interface{}) {
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(dst)
	}
}
