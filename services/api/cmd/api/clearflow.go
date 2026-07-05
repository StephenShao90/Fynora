package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

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
	writeJSON(w, 200, a.userOrganizations(userID(r)))
}

func (a *app) listPayments(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, 200, out)
}

func (a *app) listPayouts(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, 200, out)
}

func (a *app) listBankTransactions(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, 200, out)
}

func (a *app) syncStripeMock(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
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

func (a *app) createReconciliationRun(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	org := a.ensureOrganization(userID(r))
	run := a.reconcileOrganization(org.ID, userID(r))
	a.logOperation(r, "reconciliation.run.created", org.ID, map[string]interface{}{"organization_id": org.ID, "run_id": run.ID, "matched_count": run.MatchedCount, "exception_count": run.ExceptionCount, "latency_ms": time.Since(start).Milliseconds()})
	writeJSON(w, 201, run)
}

func (a *app) listReconciliationRuns(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, 200, out)
}

func (a *app) getReconciliationRun(w http.ResponseWriter, r *http.Request) {
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
		Status string `json:"status"`
	}
	if !decode(w, r, &req) {
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
	a.auditLocked(orgID, userID(r), "reconciliation_exception.updated", "reconciliation_exception", row.ID)
	a.logOperation(r, "reconciliation_exception.updated", orgID, map[string]interface{}{"organization_id": orgID, "exception_id": row.ID, "status": row.Status, "latency_ms": time.Since(start).Milliseconds()})
	writeJSON(w, 200, row)
}

func (a *app) clearflowCashSummary(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, 200, map[string]float64{"cash_balance": round2(cash), "income": round2(income), "expenses": round2(expenses), "pending_payouts": round2(pendingPayouts), "fees": round2(fees), "refunds": round2(refunds), "net_cash_flow": round2(income - expenses)})
}

func (a *app) clearflowCashForecast(w http.ResponseWriter, r *http.Request) {
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

func (a *app) clearflowMonthlyReport(w http.ResponseWriter, r *http.Request) {
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
	now := time.Now().UTC()
	org := models.Organization{ID: auth.NewID(), UserID: uid, Name: "Clearflow Demo Organization", Type: "student_organization", Currency: "USD", CreatedAt: now, UpdatedAt: now}
	a.store.mu.Lock()
	a.store.organizations[org.ID] = org
	a.auditLocked(org.ID, uid, "organization.created", "organization", org.ID)
	a.store.mu.Unlock()
	return org
}

func (a *app) seedClearflowDemo(uid string) {
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
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.Organization{}
	for _, org := range a.store.organizations {
		if org.UserID == uid {
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

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func decodeOptional(r *http.Request, dst interface{}) {
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(dst)
	}
}
