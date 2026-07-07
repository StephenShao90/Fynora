package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

func (r *ClearflowRepository) SavePlaidWebhookEvent(ctx context.Context, event models.WebhookEvent, payload string) (models.WebhookEvent, bool, error) {
	if event.ID == "" {
		event.ID = auth.NewID()
	}
	event.Status = fallback(event.Status, "received")
	event.CreatedAt = zeroTime(event.CreatedAt, time.Now().UTC())
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO plaid_webhook_events (id, organization_id, webhook_type, webhook_code, item_id, environment, payload_json, dedupe_key, status, created_at)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7::jsonb, $8, $9, $10)
		ON CONFLICT (dedupe_key) DO NOTHING
	`, event.ID, event.OrganizationID, event.Type, event.Code, event.ItemID, event.Provider, payload, event.DedupeKey, event.Status, event.CreatedAt)
	if err != nil {
		return models.WebhookEvent{}, false, err
	}
	rows, _ := res.RowsAffected()
	return event, rows > 0, nil
}

func (r *ClearflowRepository) SaveProcessorWebhookEvent(ctx context.Context, event models.WebhookEvent, payload string) (models.WebhookEvent, bool, error) {
	if event.ID == "" {
		event.ID = auth.NewID()
	}
	event.Status = fallback(event.Status, "received")
	event.CreatedAt = zeroTime(event.CreatedAt, time.Now().UTC())
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO processor_webhook_events (id, organization_id, provider, event_type, external_event_id, payload_json, dedupe_key, status, created_at)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6::jsonb, $7, $8, $9)
		ON CONFLICT (dedupe_key) DO NOTHING
	`, event.ID, event.OrganizationID, event.Provider, event.Type, event.ItemID, payload, event.DedupeKey, event.Status, event.CreatedAt)
	if err != nil {
		return models.WebhookEvent{}, false, err
	}
	rows, _ := res.RowsAffected()
	return event, rows > 0, nil
}

func (r *ClearflowRepository) FindOrganizationByPlaidItemID(ctx context.Context, itemID string) (string, error) {
	var orgID string
	err := r.db.QueryRowContext(ctx, `SELECT organization_id::text FROM plaid_items WHERE item_id = $1 LIMIT 1`, itemID).Scan(&orgID)
	return orgID, err
}

func (r *ClearflowRepository) EmitOutboxEvent(ctx context.Context, event models.OutboxEvent) (models.OutboxEvent, error) {
	now := time.Now().UTC()
	if event.ID == "" {
		event.ID = auth.NewID()
	}
	event.Status = fallback(event.Status, "pending")
	event.PayloadJSON = fallback(event.PayloadJSON, "{}")
	event.MaxAttempts = maxInt(event.MaxAttempts, 3)
	event.AvailableAt = zeroTime(event.AvailableAt, now)
	event.CreatedAt = zeroTime(event.CreatedAt, now)
	event.UpdatedAt = now
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO outbox_events (id, organization_id, event_type, aggregate_type, aggregate_id, payload_json, status, attempts, max_attempts, available_at, created_at, updated_at)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11, $12)
	`, event.ID, event.OrganizationID, event.EventType, event.AggregateType, event.AggregateID, event.PayloadJSON, event.Status, event.Attempts, event.MaxAttempts, event.AvailableAt, event.CreatedAt, event.UpdatedAt)
	return event, err
}

func (r *ClearflowRepository) ListPendingOutboxEvents(ctx context.Context, limit int) ([]models.OutboxEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(organization_id::text, ''), event_type, COALESCE(aggregate_type, ''), COALESCE(aggregate_id, ''), payload_json::text, status, attempts, max_attempts, available_at, published_at, COALESCE(error, ''), created_at, updated_at
		FROM outbox_events WHERE status IN ('pending', 'failed') AND available_at <= now() ORDER BY created_at LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOutbox(rows)
}

func (r *ClearflowRepository) MarkOutboxPublished(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE outbox_events SET status = 'published', published_at = now(), updated_at = now() WHERE id = $1`, id)
	return err
}

func (r *ClearflowRepository) MarkOutboxFailed(ctx context.Context, id, message string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE outbox_events
		SET attempts = attempts + 1,
		    status = CASE WHEN attempts + 1 >= max_attempts THEN 'dead' ELSE 'failed' END,
		    error = $2,
		    available_at = now() + interval '30 seconds',
		    updated_at = now()
		WHERE id = $1
	`, id, message)
	return err
}

func (r *ClearflowRepository) CancelJob(ctx context.Context, orgID, jobID string) (models.Job, error) {
	job, err := r.GetJob(ctx, orgID, jobID)
	if err != nil {
		return models.Job{}, err
	}
	if job.Status == "completed" {
		return models.Job{}, ErrIdempotencyConflict
	}
	status := "cancelled"
	if job.Status == "running" {
		status = "cancel_requested"
	}
	_, err = r.db.ExecContext(ctx, `UPDATE sync_jobs SET status = $1, updated_at = now() WHERE id = $2 AND organization_id = $3`, status, jobID, orgID)
	if err != nil {
		return models.Job{}, err
	}
	return r.GetJob(ctx, orgID, jobID)
}

func (r *ClearflowRepository) RetryJob(ctx context.Context, orgID, jobID string) (models.Job, error) {
	_, err := r.db.ExecContext(ctx, `UPDATE sync_jobs SET status = 'queued', run_after = now(), error = NULL, updated_at = now() WHERE id = $1 AND organization_id = $2 AND status IN ('failed', 'dead', 'cancelled')`, jobID, orgID)
	if err != nil {
		return models.Job{}, err
	}
	return r.GetJob(ctx, orgID, jobID)
}

func (r *ClearflowRepository) ListDeadJobs(ctx context.Context, orgID string, limit, offset int) ([]models.Job, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, organization_id, COALESCE(user_id::text, ''), COALESCE(type, source), status, payload_json::text, attempts, max_attempts, run_after,
		       locked_at, COALESCE(locked_by, ''), started_at, completed_at, failed_at, COALESCE(error, ''), created_at, updated_at
		FROM sync_jobs WHERE organization_id = $1 AND status = 'dead' ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

func scanOutbox(rows *sql.Rows) ([]models.OutboxEvent, error) {
	out := []models.OutboxEvent{}
	for rows.Next() {
		var event models.OutboxEvent
		var published sql.NullTime
		if err := rows.Scan(&event.ID, &event.OrganizationID, &event.EventType, &event.AggregateType, &event.AggregateID, &event.PayloadJSON, &event.Status, &event.Attempts, &event.MaxAttempts, &event.AvailableAt, &published, &event.Error, &event.CreatedAt, &event.UpdatedAt); err != nil {
			return nil, err
		}
		if published.Valid {
			event.PublishedAt = published.Time
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (r *ClearflowRepository) Readiness(ctx context.Context) error {
	for _, table := range []string{"users", "organizations", "organization_members", "sync_jobs", "audit_logs", "refresh_tokens", "outbox_events"} {
		if _, err := r.db.ExecContext(ctx, `SELECT 1 FROM `+table+` LIMIT 1`); err != nil {
			return err
		}
	}
	return nil
}
