package main

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/authz"
	"github.com/StephenShao90/Fynora/services/api/internal/cashflow"
	"github.com/StephenShao90/Fynora/services/api/internal/insights"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
	"github.com/StephenShao90/Fynora/services/api/internal/payouts"
	"github.com/StephenShao90/Fynora/services/api/internal/recommendations"
	"github.com/StephenShao90/Fynora/services/api/internal/reconciliation"
)

type financialSnapshot struct {
	Organization models.Organization
	Payments     []models.Payment
	Payouts      []models.Payout
	Bank         []models.BankTransaction
	Fees         []models.Fee
	Refunds      []models.Refund
}

func (a *app) payoutExplanationV1(w http.ResponseWriter, r *http.Request) {
	payoutID := r.PathValue("payoutId")
	if payoutID == "" {
		payoutID = r.PathValue("id")
	}
	ctx, span := a.tracer.Start(r.Context(), "payout.explain", map[string]string{"payout.id": payoutID})
	defer span.End()
	r = r.WithContext(ctx)
	r, snap, ok := a.intelligenceSnapshot(w, r)
	if !ok {
		return
	}
	for _, payout := range snap.Payouts {
		if payout.ID == payoutID {
			result := payouts.Explain(payout, snap.Payments, snap.Fees, snap.Refunds, snap.Bank)
			a.writeAudit(r.Context(), r, snap.Organization.ID, userID(r), "payout.explanation_viewed", "payout", payout.ID, "{}")
			a.emitOutbox(r.Context(), snap.Organization.ID, "payout.explanation_viewed", "payout", payout.ID, "{}")
			writeJSON(w, http.StatusOK, result)
			return
		}
	}
	errorJSON(w, r, http.StatusNotFound, "NOT_FOUND", "payout not found")
}

func (a *app) cashflowForecastV1(w http.ResponseWriter, r *http.Request) {
	ctx, span := a.tracer.Start(r.Context(), "cashflow.forecast", map[string]string{"horizon_days": r.URL.Query().Get("horizonDays")})
	defer span.End()
	r = r.WithContext(ctx)
	r, snap, ok := a.intelligenceSnapshot(w, r)
	if !ok {
		return
	}
	horizon, _ := strconv.Atoi(r.URL.Query().Get("horizonDays"))
	result := cashflow.Build(snap.Organization.ID, horizon, snap.Organization.Currency, snap.Bank, snap.Payouts)
	a.writeAudit(r.Context(), r, snap.Organization.ID, userID(r), "intelligence.forecast_generated", "organization", snap.Organization.ID, `{"horizonDays":`+strconv.Itoa(result.HorizonDays)+`}`)
	a.emitOutbox(r.Context(), snap.Organization.ID, "intelligence.forecast_generated", "organization", snap.Organization.ID, `{"horizonDays":`+strconv.Itoa(result.HorizonDays)+`}`)
	writeJSON(w, http.StatusOK, result)
}

func (a *app) anomaliesV1(w http.ResponseWriter, r *http.Request) {
	r, snap, ok := a.intelligenceSnapshot(w, r)
	if !ok {
		return
	}
	data := insights.DetectAnomalies(snap.Payouts, snap.Bank, snap.Fees, snap.Refunds)
	a.writeAudit(r.Context(), r, snap.Organization.ID, userID(r), "intelligence.anomalies_viewed", "organization", snap.Organization.ID, "{}")
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": data})
}

func (a *app) spendingInsightsV1(w http.ResponseWriter, r *http.Request) {
	r, snap, ok := a.intelligenceSnapshot(w, r)
	if !ok {
		return
	}
	from, _ := parseOptionalDate(r.URL.Query().Get("from"))
	to, _ := parseOptionalDate(r.URL.Query().Get("to"))
	result := insights.Spending(snap.Bank, from, to, snap.Organization.Currency)
	writeJSON(w, http.StatusOK, result)
}

func (a *app) cashRecommendationsV1(w http.ResponseWriter, r *http.Request) {
	r, snap, ok := a.intelligenceSnapshot(w, r)
	if !ok {
		return
	}
	anoms := insights.DetectAnomalies(snap.Payouts, snap.Bank, snap.Fees, snap.Refunds)
	data := recommendations.Cash(snap.Bank, anoms, snap.Fees, snap.Payouts, snap.Organization.Currency)
	a.writeAudit(r.Context(), r, snap.Organization.ID, userID(r), "recommendations.cash_generated", "organization", snap.Organization.ID, "{}")
	a.emitOutbox(r.Context(), snap.Organization.ID, "recommendations.cash_generated", "organization", snap.Organization.ID, "{}")
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": data})
}

func (a *app) reconciliationMatchesV1(w http.ResponseWriter, r *http.Request) {
	ctx, span := a.tracer.Start(r.Context(), "reconciliation.score", map[string]string{"run.id": r.PathValue("runId")})
	defer span.End()
	r = r.WithContext(ctx)
	r, snap, ok := a.intelligenceSnapshot(w, r)
	if !ok {
		return
	}
	data := reconciliation.ScoreAll(snap.Payouts, snap.Bank, snap.Fees, snap.Refunds)
	a.writeAudit(r.Context(), r, snap.Organization.ID, userID(r), "reconciliation.scored", "reconciliation_run", r.PathValue("runId"), "{}")
	a.emitOutbox(r.Context(), snap.Organization.ID, "reconciliation.scored", "reconciliation_run", r.PathValue("runId"), "{}")
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": data})
}

func (a *app) intelligenceSnapshot(w http.ResponseWriter, r *http.Request) (*http.Request, financialSnapshot, bool) {
	r, ok := a.withV1Organization(w, r, false, authz.CanRead)
	if !ok {
		return r, financialSnapshot{}, false
	}
	org := r.Context().Value(clearflowOrgContextKey{}).(models.Organization)
	snap, err := a.loadFinancialSnapshot(r.Context(), org)
	if err != nil {
		errorJSON(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load financial intelligence data")
		return r, financialSnapshot{}, false
	}
	return r, snap, true
}

func (a *app) loadFinancialSnapshot(ctx context.Context, org models.Organization) (financialSnapshot, error) {
	snap := financialSnapshot{Organization: org}
	if a.cfRepo != nil {
		var err error
		if snap.Payments, err = a.cfRepo.ListPayments(ctx, org.ID); err != nil {
			return snap, err
		}
		if snap.Payouts, err = a.cfRepo.ListPayouts(ctx, org.ID); err != nil {
			return snap, err
		}
		if snap.Bank, err = a.cfRepo.ListBankTransactions(ctx, org.ID); err != nil {
			return snap, err
		}
		if snap.Fees, err = a.cfRepo.ListFees(ctx, org.ID); err != nil {
			return snap, err
		}
		if snap.Refunds, err = a.cfRepo.ListRefunds(ctx, org.ID); err != nil {
			return snap, err
		}
		return snap, nil
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	for _, row := range a.store.payments {
		if row.OrganizationID == org.ID {
			snap.Payments = append(snap.Payments, row)
		}
	}
	for _, row := range a.store.payouts {
		if row.OrganizationID == org.ID {
			snap.Payouts = append(snap.Payouts, row)
		}
	}
	for _, row := range a.store.bankTransactions {
		if row.OrganizationID == org.ID {
			snap.Bank = append(snap.Bank, row)
		}
	}
	for _, row := range a.store.fees {
		if row.OrganizationID == org.ID {
			snap.Fees = append(snap.Fees, row)
		}
	}
	for _, row := range a.store.refunds {
		if row.OrganizationID == org.ID {
			snap.Refunds = append(snap.Refunds, row)
		}
	}
	return snap, nil
}

func parseOptionalDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, value)
}

func (a *app) legacyCashflowForecastAlias(w http.ResponseWriter, r *http.Request) {
	a.cashflowForecastV1(w, r)
}
