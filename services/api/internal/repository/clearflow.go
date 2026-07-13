package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

var ErrIdempotencyConflict = errors.New("idempotency key reused with a different request")

type ClearflowRepository struct {
	db *sql.DB
}

func NewClearflow(db *sql.DB) *ClearflowRepository {
	return &ClearflowRepository{db: db}
}

func (r *ClearflowRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *ClearflowRepository) EnsureUser(ctx context.Context, u models.User) error {
	if u.ID == "" {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE
		SET email = EXCLUDED.email,
		    password_hash = COALESCE(NULLIF(EXCLUDED.password_hash, ''), users.password_hash)
	`, u.ID, u.Email, fallback(u.PasswordHash, "external-auth-user"), zeroTime(u.CreatedAt, time.Now().UTC()))
	if err != nil && strings.Contains(err.Error(), "users_email_key") {
		_, err = r.db.ExecContext(ctx, `
			INSERT INTO users (id, email, password_hash, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO NOTHING
		`, u.ID, u.ID+"@clearflow.local", fallback(u.PasswordHash, "external-auth-user"), zeroTime(u.CreatedAt, time.Now().UTC()))
	}
	return err
}

func (r *ClearflowRepository) CreateOrganization(ctx context.Context, user models.User, req models.Organization) (models.Organization, error) {
	if err := r.EnsureUser(ctx, user); err != nil {
		return models.Organization{}, err
	}
	now := time.Now().UTC()
	org := models.Organization{
		ID:        fallback(req.ID, auth.NewID()),
		UserID:    user.ID,
		Name:      fallback(req.Name, "Clearflow Demo Organization"),
		Type:      fallback(req.Type, "student_organization"),
		Currency:  fallback(req.Currency, "USD"),
		CreatedAt: now,
		UpdatedAt: now,
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Organization{}, err
	}
	defer rollback(tx)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO organizations (id, user_id, name, type, currency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, type = EXCLUDED.type, currency = EXCLUDED.currency, updated_at = EXCLUDED.updated_at
	`, org.ID, org.UserID, org.Name, org.Type, org.Currency, org.CreatedAt, org.UpdatedAt); err != nil {
		return models.Organization{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO organization_members (id, organization_id, user_id, role, created_at)
		VALUES ($1, $2, $3, 'owner', $4)
		ON CONFLICT (organization_id, user_id) DO UPDATE SET role = organization_members.role
	`, auth.NewID(), org.ID, user.ID, now); err != nil {
		return models.Organization{}, err
	}
	if err := insertAudit(ctx, tx, org.ID, user.ID, "organization.created", "organization", org.ID); err != nil {
		return models.Organization{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.Organization{}, err
	}
	return org, nil
}

func (r *ClearflowRepository) ListOrganizations(ctx context.Context, userID string) ([]models.Organization, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT o.id, o.user_id, o.name, o.type, o.currency, o.created_at, o.updated_at
		FROM organizations o
		JOIN organization_members m ON m.organization_id = o.id
		WHERE m.user_id = $1
		ORDER BY o.created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Organization
	for rows.Next() {
		var org models.Organization
		if err := rows.Scan(&org.ID, &org.UserID, &org.Name, &org.Type, &org.Currency, &org.CreatedAt, &org.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, org)
	}
	return out, rows.Err()
}

func (r *ClearflowRepository) EnsureOrganization(ctx context.Context, user models.User) (models.Organization, error) {
	orgs, err := r.ListOrganizations(ctx, user.ID)
	if err != nil {
		return models.Organization{}, err
	}
	if len(orgs) > 0 {
		return orgs[0], nil
	}
	return r.CreateOrganization(ctx, user, models.Organization{})
}

func (r *ClearflowRepository) UserCanAccessOrg(ctx context.Context, userID, orgID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM organization_members WHERE user_id = $1 AND organization_id = $2)`, userID, orgID).Scan(&exists)
	return exists, err
}

func (r *ClearflowRepository) ListPayments(ctx context.Context, orgID string) ([]models.Payment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, organization_id, processor, processor_payment_id, customer_email, amount_minor, currency, status, occurred_at, description, created_at
		FROM payments WHERE organization_id = $1 ORDER BY occurred_at DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Payment
	for rows.Next() {
		var row models.Payment
		var amount int64
		if err := rows.Scan(&row.ID, &row.OrganizationID, &row.Processor, &row.ProcessorPaymentID, &row.CustomerEmail, &amount, &row.Currency, &row.Status, &row.OccurredAt, &row.Description, &row.CreatedAt); err != nil {
			return nil, err
		}
		row.Amount = fromMinor(amount)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *ClearflowRepository) ListPayouts(ctx context.Context, orgID string) ([]models.Payout, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, organization_id, processor, processor_payout_id, amount_minor, currency, status, expected_arrival_at, created_at
		FROM payouts WHERE organization_id = $1 ORDER BY expected_arrival_at DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Payout
	for rows.Next() {
		var row models.Payout
		var amount int64
		if err := rows.Scan(&row.ID, &row.OrganizationID, &row.Processor, &row.ProcessorPayoutID, &amount, &row.Currency, &row.Status, &row.ExpectedArrivalAt, &row.CreatedAt); err != nil {
			return nil, err
		}
		row.Amount = fromMinor(amount)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *ClearflowRepository) ListBankTransactions(ctx context.Context, orgID string) ([]models.BankTransaction, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, organization_id, source, external_id, amount_minor, direction, currency, description, posted_at, created_at
		FROM bank_transactions WHERE organization_id = $1 ORDER BY posted_at DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.BankTransaction
	for rows.Next() {
		var row models.BankTransaction
		var amount int64
		if err := rows.Scan(&row.ID, &row.OrganizationID, &row.Source, &row.ExternalID, &amount, &row.Direction, &row.Currency, &row.Description, &row.PostedAt, &row.CreatedAt); err != nil {
			return nil, err
		}
		row.Amount = fromMinor(amount)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *ClearflowRepository) ListFees(ctx context.Context, orgID string) ([]models.Fee, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, organization_id, processor_fee_id, COALESCE(payment_id::text, ''), amount_minor, currency, occurred_at, COALESCE(description, '')
		FROM fees WHERE organization_id = $1 ORDER BY occurred_at DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Fee{}
	for rows.Next() {
		var row models.Fee
		var amount int64
		if err := rows.Scan(&row.ID, &row.OrganizationID, &row.ProcessorFeeID, &row.PaymentID, &amount, &row.Currency, &row.OccurredAt, &row.Description); err != nil {
			return nil, err
		}
		row.Amount = fromMinor(amount)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *ClearflowRepository) ListRefunds(ctx context.Context, orgID string) ([]models.Refund, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, organization_id, processor_refund_id, COALESCE(payment_id::text, ''), amount_minor, currency, occurred_at
		FROM refunds WHERE organization_id = $1 ORDER BY occurred_at DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Refund{}
	for rows.Next() {
		var row models.Refund
		var amount int64
		if err := rows.Scan(&row.ID, &row.OrganizationID, &row.ProcessorRefundID, &row.PaymentID, &amount, &row.Currency, &row.OccurredAt); err != nil {
			return nil, err
		}
		row.Amount = fromMinor(amount)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *ClearflowRepository) SyncStripeDemo(ctx context.Context, org models.Organization, userID string) (map[string]interface{}, error) {
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
	netMinor := int64(0)
	for _, payment := range payments {
		netMinor += toMinor(payment.Amount)
	}
	netMinor -= toMinor(refund.Amount)
	for _, fee := range fees {
		netMinor -= toMinor(fee.Amount)
	}
	payout := models.Payout{ID: auth.NewID(), OrganizationID: org.ID, Processor: "stripe", ProcessorPayoutID: "po_demo_001", Amount: fromMinor(netMinor), Currency: org.Currency, Status: "paid", ExpectedArrivalAt: now.AddDate(0, 0, -2), CreatedAt: now.AddDate(0, 0, -2)}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	paymentIDs := map[string]string{}
	for _, payment := range payments {
		id, err := upsertPayment(ctx, tx, payment)
		if err != nil {
			return nil, err
		}
		paymentIDs[payment.ProcessorPaymentID] = id
	}
	refund.PaymentID = paymentIDs["ch_ticket_001"]
	if err := upsertRefund(ctx, tx, refund); err != nil {
		return nil, err
	}
	for i := range fees {
		fees[i].PaymentID = paymentIDs[payments[i].ProcessorPaymentID]
		if err := upsertFee(ctx, tx, fees[i]); err != nil {
			return nil, err
		}
	}
	payoutID, err := upsertPayout(ctx, tx, payout)
	if err != nil {
		return nil, err
	}
	payout.ID = payoutID
	if err := replacePayoutItems(ctx, tx, org.ID, payout.ID, payments, fees, refund); err != nil {
		return nil, err
	}
	if err := insertAudit(ctx, tx, org.ID, userID, "stripe.mock_synced", "payout", payout.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]interface{}{"payments": len(payments), "refunds": 1, "fees": len(fees), "payout": payout}, nil
}

func (r *ClearflowRepository) SyncBankDemo(ctx context.Context, org models.Organization, userID string) (map[string]interface{}, error) {
	payouts, err := r.ListPayouts(ctx, org.ID)
	if err != nil {
		return nil, err
	}
	payoutAmount := 1665.72
	if len(payouts) > 0 {
		payoutAmount = payouts[len(payouts)-1].Amount
	}
	now := time.Now().UTC()
	rows := []models.BankTransaction{
		{ID: auth.NewID(), OrganizationID: org.ID, Source: "plaid_or_csv", ExternalID: "bank_stripe_demo_001", Amount: payoutAmount, Direction: "credit", Currency: org.Currency, Description: "STRIPE PAYOUT", PostedAt: now.AddDate(0, 0, -2), CreatedAt: now},
		{ID: auth.NewID(), OrganizationID: org.ID, Source: "plaid_or_csv", ExternalID: "bank_venue_001", Amount: 300, Direction: "debit", Currency: org.Currency, Description: "Venue deposit", PostedAt: now.AddDate(0, 0, -1), CreatedAt: now},
		{ID: auth.NewID(), OrganizationID: org.ID, Source: "plaid_or_csv", ExternalID: "bank_unmatched_001", Amount: 212.45, Direction: "credit", Currency: org.Currency, Description: "Unknown deposit", PostedAt: now.AddDate(0, 0, -1), CreatedAt: now},
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	for _, row := range rows {
		if err := upsertBankTransaction(ctx, tx, row); err != nil {
			return nil, err
		}
	}
	if err := insertAudit(ctx, tx, org.ID, userID, "bank.mock_synced", "bank_transaction", rows[0].ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]interface{}{"bank_transactions": len(rows)}, nil
}

func (r *ClearflowRepository) SeedDemo(ctx context.Context, user models.User) error {
	org, err := r.EnsureOrganization(ctx, user)
	if err != nil {
		return err
	}
	payouts, err := r.ListPayouts(ctx, org.ID)
	if err != nil {
		return err
	}
	if len(payouts) > 0 {
		return nil
	}
	if _, err := r.SyncStripeDemo(ctx, org, user.ID); err != nil {
		return err
	}
	if _, err := r.SyncBankDemo(ctx, org, user.ID); err != nil {
		return err
	}
	_, err = r.Reconcile(ctx, org.ID, user.ID)
	return err
}

func (r *ClearflowRepository) Reconcile(ctx context.Context, orgID, userID string) (models.ReconciliationRun, error) {
	payouts, err := r.ListPayouts(ctx, orgID)
	if err != nil {
		return models.ReconciliationRun{}, err
	}
	banks, err := r.ListBankTransactions(ctx, orgID)
	if err != nil {
		return models.ReconciliationRun{}, err
	}
	now := time.Now().UTC()
	run := models.ReconciliationRun{ID: auth.NewID(), OrganizationID: orgID, Status: "completed", StartedAt: now}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return models.ReconciliationRun{}, err
	}
	defer rollback(tx)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO reconciliation_runs (id, organization_id, status, matched_count, exception_count, started_at)
		VALUES ($1, $2, 'running', 0, 0, $3)
	`, run.ID, run.OrganizationID, run.StartedAt); err != nil {
		return models.ReconciliationRun{}, err
	}
	matchedBanks := map[string]bool{}
	for _, payout := range payouts {
		best := models.BankTransaction{}
		bestScore := 0.0
		for _, bank := range banks {
			if bank.Direction != "credit" || matchedBanks[bank.ID] {
				continue
			}
			score := reconciliationScore(payout, bank)
			if score > bestScore {
				bestScore = score
				best = bank
			}
		}
		if best.ID != "" && bestScore >= 0.88 {
			match := models.ReconciliationMatch{ID: auth.NewID(), OrganizationID: orgID, RunID: run.ID, PayoutID: payout.ID, BankTransactionID: best.ID, Amount: payout.Amount, Confidence: round2(bestScore), Explanation: fmt.Sprintf("Matched %s payout %s to bank deposit %s by amount/date/description.", payout.Processor, payout.ProcessorPayoutID, best.ExternalID), CreatedAt: now}
			if err := insertMatch(ctx, tx, match); err != nil {
				return models.ReconciliationRun{}, err
			}
			matchedBanks[best.ID] = true
			run.MatchedCount++
		} else {
			ex := models.ReconciliationException{ID: auth.NewID(), OrganizationID: orgID, RunID: run.ID, Type: "unmatched_payout", Severity: "high", Title: "Unmatched payout", Explanation: fmt.Sprintf("Payout %s for $%.2f did not match any bank deposit.", payout.ProcessorPayoutID, payout.Amount), Status: "open", ReferenceID: payout.ID, CreatedAt: now}
			if err := insertException(ctx, tx, ex); err != nil {
				return models.ReconciliationRun{}, err
			}
			run.ExceptionCount++
		}
	}
	for _, bank := range banks {
		if bank.Direction == "credit" && !matchedBanks[bank.ID] {
			ex := models.ReconciliationException{ID: auth.NewID(), OrganizationID: orgID, RunID: run.ID, Type: "unmatched_deposit", Severity: "medium", Title: "Unmatched bank deposit", Explanation: fmt.Sprintf("Bank deposit %s for $%.2f is not tied to a known payout.", bank.Description, bank.Amount), Status: "open", ReferenceID: bank.ID, CreatedAt: now}
			if err := insertException(ctx, tx, ex); err != nil {
				return models.ReconciliationRun{}, err
			}
			run.ExceptionCount++
		}
	}
	run.CompletedAt = time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `
		UPDATE reconciliation_runs
		SET status = $1, matched_count = $2, exception_count = $3, completed_at = $4
		WHERE id = $5 AND organization_id = $6
	`, run.Status, run.MatchedCount, run.ExceptionCount, run.CompletedAt, run.ID, run.OrganizationID); err != nil {
		return models.ReconciliationRun{}, err
	}
	if err := insertAudit(ctx, tx, orgID, userID, "reconciliation.run_completed", "reconciliation_run", run.ID); err != nil {
		return models.ReconciliationRun{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.ReconciliationRun{}, err
	}
	return run, nil
}

func (r *ClearflowRepository) ListReconciliationRuns(ctx context.Context, orgID string) ([]models.ReconciliationRun, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, organization_id, status, matched_count, exception_count, started_at, completed_at FROM reconciliation_runs WHERE organization_id = $1 ORDER BY started_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ReconciliationRun
	for rows.Next() {
		var row models.ReconciliationRun
		if err := rows.Scan(&row.ID, &row.OrganizationID, &row.Status, &row.MatchedCount, &row.ExceptionCount, &row.StartedAt, &row.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *ClearflowRepository) GetReconciliationRun(ctx context.Context, orgID, runID string) (map[string]interface{}, error) {
	var run models.ReconciliationRun
	if err := r.db.QueryRowContext(ctx, `SELECT id, organization_id, status, matched_count, exception_count, started_at, completed_at FROM reconciliation_runs WHERE id = $1 AND organization_id = $2`, runID, orgID).Scan(&run.ID, &run.OrganizationID, &run.Status, &run.MatchedCount, &run.ExceptionCount, &run.StartedAt, &run.CompletedAt); err != nil {
		return nil, err
	}
	matches, err := r.listMatches(ctx, orgID, runID)
	if err != nil {
		return nil, err
	}
	exceptions, err := r.ListExceptions(ctx, orgID)
	if err != nil {
		return nil, err
	}
	runExceptions := []models.ReconciliationException{}
	for _, ex := range exceptions {
		if ex.RunID == runID {
			runExceptions = append(runExceptions, ex)
		}
	}
	return map[string]interface{}{"run": run, "matches": matches, "exceptions": runExceptions}, nil
}

func (r *ClearflowRepository) listMatches(ctx context.Context, orgID, runID string) ([]models.ReconciliationMatch, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, organization_id, run_id, payout_id, bank_transaction_id, amount_minor, confidence, explanation, created_at FROM reconciliation_matches WHERE organization_id = $1 AND run_id = $2`, orgID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ReconciliationMatch
	for rows.Next() {
		var row models.ReconciliationMatch
		var amount int64
		if err := rows.Scan(&row.ID, &row.OrganizationID, &row.RunID, &row.PayoutID, &row.BankTransactionID, &amount, &row.Confidence, &row.Explanation, &row.CreatedAt); err != nil {
			return nil, err
		}
		row.Amount = fromMinor(amount)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *ClearflowRepository) ListExceptions(ctx context.Context, orgID string) ([]models.ReconciliationException, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, organization_id, run_id, type, severity, title, explanation, status, reference_id, created_at FROM reconciliation_exceptions WHERE organization_id = $1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ReconciliationException
	for rows.Next() {
		var row models.ReconciliationException
		if err := rows.Scan(&row.ID, &row.OrganizationID, &row.RunID, &row.Type, &row.Severity, &row.Title, &row.Explanation, &row.Status, &row.ReferenceID, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *ClearflowRepository) UpdateException(ctx context.Context, orgID, userID, exceptionID, status string) (models.ReconciliationException, error) {
	if status == "" {
		status = "resolved"
	}
	var row models.ReconciliationException
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return row, err
	}
	defer rollback(tx)
	if err := tx.QueryRowContext(ctx, `
		UPDATE reconciliation_exceptions SET status = $1
		WHERE id = $2 AND organization_id = $3
		RETURNING id, organization_id, run_id, type, severity, title, explanation, status, reference_id, created_at
	`, status, exceptionID, orgID).Scan(&row.ID, &row.OrganizationID, &row.RunID, &row.Type, &row.Severity, &row.Title, &row.Explanation, &row.Status, &row.ReferenceID, &row.CreatedAt); err != nil {
		return row, err
	}
	if row.RunID != "" {
		if _, err := tx.ExecContext(ctx, `
			UPDATE reconciliation_runs
			SET exception_count = (
				SELECT COUNT(*) FROM reconciliation_exceptions
				WHERE organization_id = $1 AND run_id = $2 AND status = 'open'
			)
			WHERE organization_id = $1 AND id = $2
		`, orgID, row.RunID); err != nil {
			return row, err
		}
	}
	if err := insertAudit(ctx, tx, orgID, userID, "reconciliation_exception.updated", "reconciliation_exception", row.ID); err != nil {
		return row, err
	}
	if err := tx.Commit(); err != nil {
		return row, err
	}
	return row, nil
}

func (r *ClearflowRepository) PayoutBreakdown(ctx context.Context, orgID, payoutID string) (map[string]interface{}, error) {
	var payout models.Payout
	var amount int64
	if err := r.db.QueryRowContext(ctx, `SELECT id, organization_id, processor, processor_payout_id, amount_minor, currency, status, expected_arrival_at, created_at FROM payouts WHERE id = $1 AND organization_id = $2`, payoutID, orgID).Scan(&payout.ID, &payout.OrganizationID, &payout.Processor, &payout.ProcessorPayoutID, &amount, &payout.Currency, &payout.Status, &payout.ExpectedArrivalAt, &payout.CreatedAt); err != nil {
		return nil, err
	}
	payout.Amount = fromMinor(amount)
	rows, err := r.db.QueryContext(ctx, `SELECT id, organization_id, payout_id, source_type, source_id, amount_minor, currency, description, created_at FROM payout_items WHERE organization_id = $1 AND payout_id = $2 ORDER BY created_at`, orgID, payoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.PayoutItem{}
	grossMinor, feeMinor, refundMinor := int64(0), int64(0), int64(0)
	for rows.Next() {
		var item models.PayoutItem
		var minor int64
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.PayoutID, &item.SourceType, &item.SourceID, &minor, &item.Currency, &item.Description, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Amount = fromMinor(minor)
		switch item.SourceType {
		case "payment":
			grossMinor += minor
		case "fee":
			feeMinor += abs64(minor)
		case "refund":
			refundMinor += abs64(minor)
		}
		items = append(items, item)
	}
	return map[string]interface{}{"payout": payout, "items": items, "gross_payments": fromMinor(grossMinor), "fees": fromMinor(feeMinor), "refunds": fromMinor(refundMinor), "net_payout": payout.Amount}, rows.Err()
}

func (r *ClearflowRepository) CashSummary(ctx context.Context, orgID string) (map[string]float64, error) {
	var cashMinor, incomeMinor, expensesMinor, pendingMinor, feesMinor, refundsMinor int64
	rows, err := r.db.QueryContext(ctx, `SELECT amount_minor, direction FROM bank_transactions WHERE organization_id = $1`, orgID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var amount int64
		var direction string
		if err := rows.Scan(&amount, &direction); err != nil {
			rows.Close()
			return nil, err
		}
		if direction == "credit" {
			cashMinor += amount
			incomeMinor += amount
		} else {
			cashMinor -= amount
			expensesMinor += amount
		}
	}
	rows.Close()
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_minor), 0) FROM payouts WHERE organization_id = $1 AND status <> 'paid'`, orgID).Scan(&pendingMinor); err != nil {
		return nil, err
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_minor), 0) FROM fees WHERE organization_id = $1`, orgID).Scan(&feesMinor); err != nil {
		return nil, err
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_minor), 0) FROM refunds WHERE organization_id = $1`, orgID).Scan(&refundsMinor); err != nil {
		return nil, err
	}
	return map[string]float64{"cash_balance": fromMinor(cashMinor), "income": fromMinor(incomeMinor), "expenses": fromMinor(expensesMinor), "pending_payouts": fromMinor(pendingMinor), "fees": fromMinor(feesMinor), "refunds": fromMinor(refundsMinor), "net_cash_flow": fromMinor(incomeMinor - expensesMinor - feesMinor - refundsMinor)}, nil
}

func (r *ClearflowRepository) CashForecast(ctx context.Context, orgID string) ([]map[string]interface{}, error) {
	summary, err := r.CashSummary(ctx, orgID)
	if err != nil {
		return nil, err
	}
	points := []map[string]interface{}{}
	for _, days := range []int{7, 30, 60} {
		cutoff := time.Now().UTC().AddDate(0, 0, days)
		var payoutMinor int64
		if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_minor), 0) FROM payouts WHERE organization_id = $1 AND status <> 'paid' AND expected_arrival_at <= $2`, orgID, cutoff).Scan(&payoutMinor); err != nil {
			return nil, err
		}
		expenses := 0.0
		if days >= 30 {
			expenses = 450
		}
		points = append(points, map[string]interface{}{"days": days, "projected_cash": round2(summary["cash_balance"] + fromMinor(payoutMinor) - expenses), "expected_payouts": fromMinor(payoutMinor), "expected_expenses": expenses})
	}
	return points, nil
}

func (r *ClearflowRepository) MonthlyReport(ctx context.Context, orgID string) (map[string]interface{}, error) {
	month := time.Now().UTC().Format("2006-01")
	var gross, refunds, fees, payouts int64
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_minor), 0) FROM payments WHERE organization_id = $1 AND status = 'succeeded'`, orgID).Scan(&gross); err != nil {
		return nil, err
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_minor), 0) FROM refunds WHERE organization_id = $1`, orgID).Scan(&refunds); err != nil {
		return nil, err
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_minor), 0) FROM fees WHERE organization_id = $1`, orgID).Scan(&fees); err != nil {
		return nil, err
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_minor), 0) FROM payouts WHERE organization_id = $1`, orgID).Scan(&payouts); err != nil {
		return nil, err
	}
	return map[string]interface{}{"gross_payments": fromMinor(gross), "refunds": fromMinor(refunds), "fees": fromMinor(fees), "net_processor_activity": fromMinor(gross - refunds - fees), "payouts": fromMinor(payouts), "month": month}, nil
}

func (r *ClearflowRepository) DebugState(ctx context.Context, org models.Organization) (map[string]interface{}, error) {
	counts := map[string]int{}
	for name, query := range map[string]string{
		"payments":     `SELECT COUNT(*) FROM payments WHERE organization_id = $1`,
		"refunds":      `SELECT COUNT(*) FROM refunds WHERE organization_id = $1`,
		"fees":         `SELECT COUNT(*) FROM fees WHERE organization_id = $1`,
		"payouts":      `SELECT COUNT(*) FROM payouts WHERE organization_id = $1`,
		"bank_tx":      `SELECT COUNT(*) FROM bank_transactions WHERE organization_id = $1`,
		"runs":         `SELECT COUNT(*) FROM reconciliation_runs WHERE organization_id = $1`,
		"matches":      `SELECT COUNT(*) FROM reconciliation_matches WHERE organization_id = $1`,
		"exceptions":   `SELECT COUNT(*) FROM reconciliation_exceptions WHERE organization_id = $1`,
		"open_breaks":  `SELECT COUNT(*) FROM reconciliation_exceptions WHERE organization_id = $1 AND status = 'open'`,
		"audit_events": `SELECT COUNT(*) FROM audit_logs WHERE organization_id = $1`,
	} {
		var count int
		if err := r.db.QueryRowContext(ctx, query, org.ID).Scan(&count); err != nil {
			return nil, err
		}
		counts[name] = count
	}
	return map[string]interface{}{"organization": org, "counts": counts, "storage": "postgres", "debug_note": "No secrets are included. Pair this with terminal JSON logs when reporting bugs."}, nil
}

func (r *ClearflowRepository) ReadIdempotency(ctx context.Context, userID, key, requestHash string) (int, []byte, bool, error) {
	if key == "" {
		return 0, nil, false, nil
	}
	var status int
	var body string
	var storedHash string
	err := r.db.QueryRowContext(ctx, `SELECT request_hash, status_code, response_body FROM idempotency_keys WHERE user_id = $1 AND key = $2`, userID, key).Scan(&storedHash, &status, &body)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, false, nil
	}
	if err != nil {
		return 0, nil, false, err
	}
	if storedHash != requestHash {
		return 0, nil, false, ErrIdempotencyConflict
	}
	return status, []byte(body), true, nil
}

func (r *ClearflowRepository) SaveIdempotency(ctx context.Context, userID, orgID, key, requestHash string, status int, body []byte) error {
	if key == "" {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO idempotency_keys (key, user_id, organization_id, request_hash, status_code, response_body, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, key) DO NOTHING
	`, key, userID, orgID, requestHash, status, string(body), time.Now().UTC())
	return err
}

func upsertPayment(ctx context.Context, tx *sql.Tx, row models.Payment) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `
		INSERT INTO payments (id, organization_id, processor, processor_payment_id, customer_email, amount_minor, currency, status, occurred_at, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (organization_id, processor_payment_id) DO UPDATE
		SET customer_email = EXCLUDED.customer_email, amount_minor = EXCLUDED.amount_minor, currency = EXCLUDED.currency, status = EXCLUDED.status, occurred_at = EXCLUDED.occurred_at, description = EXCLUDED.description
		RETURNING id
	`, row.ID, row.OrganizationID, row.Processor, row.ProcessorPaymentID, row.CustomerEmail, toMinor(row.Amount), row.Currency, row.Status, row.OccurredAt, row.Description, row.CreatedAt).Scan(&id)
	return id, err
}

func upsertRefund(ctx context.Context, tx *sql.Tx, row models.Refund) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO refunds (id, organization_id, processor_refund_id, payment_id, amount_minor, currency, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (organization_id, processor_refund_id) DO UPDATE
		SET payment_id = EXCLUDED.payment_id, amount_minor = EXCLUDED.amount_minor, currency = EXCLUDED.currency, occurred_at = EXCLUDED.occurred_at
	`, row.ID, row.OrganizationID, row.ProcessorRefundID, nullableUUID(row.PaymentID), toMinor(row.Amount), row.Currency, row.OccurredAt)
	return err
}

func upsertFee(ctx context.Context, tx *sql.Tx, row models.Fee) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO fees (id, organization_id, processor_fee_id, payment_id, amount_minor, currency, occurred_at, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (organization_id, processor_fee_id) DO UPDATE
		SET payment_id = EXCLUDED.payment_id, amount_minor = EXCLUDED.amount_minor, currency = EXCLUDED.currency, occurred_at = EXCLUDED.occurred_at, description = EXCLUDED.description
	`, row.ID, row.OrganizationID, row.ProcessorFeeID, nullableUUID(row.PaymentID), toMinor(row.Amount), row.Currency, row.OccurredAt, row.Description)
	return err
}

func upsertPayout(ctx context.Context, tx *sql.Tx, row models.Payout) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `
		INSERT INTO payouts (id, organization_id, processor, processor_payout_id, amount_minor, currency, status, expected_arrival_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (organization_id, processor_payout_id) DO UPDATE
		SET amount_minor = EXCLUDED.amount_minor, currency = EXCLUDED.currency, status = EXCLUDED.status, expected_arrival_at = EXCLUDED.expected_arrival_at
		RETURNING id
	`, row.ID, row.OrganizationID, row.Processor, row.ProcessorPayoutID, toMinor(row.Amount), row.Currency, row.Status, row.ExpectedArrivalAt, row.CreatedAt).Scan(&id)
	return id, err
}

func upsertBankTransaction(ctx context.Context, tx *sql.Tx, row models.BankTransaction) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO bank_transactions (id, organization_id, source, external_id, amount_minor, direction, currency, description, posted_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (organization_id, external_id) DO UPDATE
		SET source = EXCLUDED.source, amount_minor = EXCLUDED.amount_minor, direction = EXCLUDED.direction, currency = EXCLUDED.currency, description = EXCLUDED.description, posted_at = EXCLUDED.posted_at
	`, row.ID, row.OrganizationID, row.Source, row.ExternalID, toMinor(row.Amount), row.Direction, row.Currency, row.Description, row.PostedAt, row.CreatedAt)
	return err
}

func replacePayoutItems(ctx context.Context, tx *sql.Tx, orgID, payoutID string, payments []models.Payment, fees []models.Fee, refund models.Refund) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM payout_items WHERE payout_id = $1 AND organization_id = $2`, payoutID, orgID); err != nil {
		return err
	}
	for _, payment := range payments {
		if err := insertPayoutItem(ctx, tx, orgID, payoutID, "payment", payment.ProcessorPaymentID, toMinor(payment.Amount), payment.Currency, payment.Description); err != nil {
			return err
		}
	}
	for _, fee := range fees {
		if err := insertPayoutItem(ctx, tx, orgID, payoutID, "fee", fee.ProcessorFeeID, -toMinor(fee.Amount), fee.Currency, fee.Description); err != nil {
			return err
		}
	}
	return insertPayoutItem(ctx, tx, orgID, payoutID, "refund", refund.ProcessorRefundID, -toMinor(refund.Amount), refund.Currency, "Refund")
}

func insertPayoutItem(ctx context.Context, tx *sql.Tx, orgID, payoutID, sourceType, sourceID string, amountMinor int64, currency, description string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO payout_items (id, organization_id, payout_id, source_type, source_id, amount_minor, currency, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, auth.NewID(), orgID, payoutID, sourceType, sourceID, amountMinor, currency, description, time.Now().UTC())
	return err
}

func insertMatch(ctx context.Context, tx *sql.Tx, row models.ReconciliationMatch) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO reconciliation_matches (id, organization_id, run_id, payout_id, bank_transaction_id, amount_minor, confidence, explanation, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, row.ID, row.OrganizationID, row.RunID, nullableUUID(row.PayoutID), nullableUUID(row.BankTransactionID), toMinor(row.Amount), row.Confidence, row.Explanation, row.CreatedAt)
	return err
}

func insertException(ctx context.Context, tx *sql.Tx, row models.ReconciliationException) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO reconciliation_exceptions (id, organization_id, run_id, type, severity, title, explanation, status, reference_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, row.ID, row.OrganizationID, nullableUUID(row.RunID), row.Type, row.Severity, row.Title, row.Explanation, row.Status, row.ReferenceID, row.CreatedAt)
	return err
}

func insertAudit(ctx context.Context, tx *sql.Tx, orgID, userID, action, targetType, targetID string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO audit_logs (id, organization_id, user_id, action, target_type, target_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, auth.NewID(), orgID, nullableUUID(userID), action, targetType, targetID, time.Now().UTC())
	return err
}

func reconciliationScore(payout models.Payout, bank models.BankTransaction) float64 {
	amountScore := 1 - math.Min(1, math.Abs(bank.Amount-payout.Amount)/math.Max(payout.Amount, 1))
	dateDistance := math.Abs(bank.PostedAt.Sub(payout.ExpectedArrivalAt).Hours()) / 24
	dateScore := math.Max(0, 1-dateDistance/5)
	descriptionScore := 0.0
	if containsFold(bank.Description, payout.Processor) {
		descriptionScore = 0.2
	}
	return amountScore*0.7 + dateScore*0.2 + descriptionScore
}

func EncodeBody(payload interface{}) []byte {
	body, _ := json.Marshal(payload)
	return body
}

func SortPayoutItems(items []models.PayoutItem) {
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
}

func toMinor(v float64) int64 {
	return int64(math.Round(v * 100))
}

func fromMinor(v int64) float64 {
	return math.Round(float64(v)) / 100
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func zeroTime(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value
}

func fallback(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func nullableUUID(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func containsFold(haystack, needle string) bool {
	return len(needle) == 0 || len(haystack) >= len(needle) && (stringContainsFold(haystack, needle))
}

func stringContainsFold(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if len(haystack[i:i+len(needle)]) == len(needle) && equalFoldASCII(haystack[i:i+len(needle)], needle) {
			return true
		}
	}
	return false
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ac, bc := a[i], b[i]
		if ac >= 'A' && ac <= 'Z' {
			ac += 'a' - 'A'
		}
		if bc >= 'A' && bc <= 'Z' {
			bc += 'a' - 'A'
		}
		if ac != bc {
			return false
		}
	}
	return true
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func rollback(tx *sql.Tx) {
	_ = tx.Rollback()
}
