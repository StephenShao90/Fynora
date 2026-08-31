package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

var ErrSessionRevoked = errors.New("refresh session is revoked")

func (r *ClearflowRepository) CreateRefreshSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time, userAgent, ip string, rotatedFromID string) (models.RefreshSession, error) {
	session := models.RefreshSession{ID: auth.NewID(), UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt, RotatedFromID: rotatedFromID, CreatedAt: time.Now().UTC(), UserAgent: userAgent, IPAddress: ip}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, rotated_from_id, created_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::uuid, $6, $7, $8)
	`, session.ID, session.UserID, session.TokenHash, session.ExpiresAt, session.RotatedFromID, session.CreatedAt, session.UserAgent, session.IPAddress)
	return session, err
}

func (r *ClearflowRepository) GetRefreshSessionByHash(ctx context.Context, tokenHash string) (models.RefreshSession, error) {
	var s models.RefreshSession
	var revokedAt, lastUsedAt sql.NullTime
	var rotatedFrom sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at, rotated_from_id::text, created_at, last_used_at, COALESCE(user_agent, ''), COALESCE(ip_address, '')
		FROM refresh_tokens WHERE token_hash = $1
	`, tokenHash).Scan(&s.ID, &s.UserID, &s.TokenHash, &s.ExpiresAt, &revokedAt, &rotatedFrom, &s.CreatedAt, &lastUsedAt, &s.UserAgent, &s.IPAddress)
	if revokedAt.Valid {
		s.RevokedAt = &revokedAt.Time
	}
	if lastUsedAt.Valid {
		s.LastUsedAt = &lastUsedAt.Time
	}
	if rotatedFrom.Valid {
		s.RotatedFromID = rotatedFrom.String
	}
	return s, err
}

func (r *ClearflowRepository) MarkRefreshSessionUsed(ctx context.Context, sessionID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE refresh_tokens SET last_used_at = now() WHERE id = $1`, sessionID)
	return err
}

func (r *ClearflowRepository) RevokeRefreshSession(ctx context.Context, sessionID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE refresh_tokens SET revoked_at = COALESCE(revoked_at, now()) WHERE id = $1`, sessionID)
	return err
}

func (r *ClearflowRepository) RevokeRefreshSessionForUser(ctx context.Context, userID, sessionID string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE refresh_tokens SET revoked_at = COALESCE(revoked_at, now()) WHERE id = $1 AND user_id = $2`, sessionID, userID)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	return rows > 0, err
}

func (r *ClearflowRepository) ListRefreshSessions(ctx context.Context, userID string) ([]models.RefreshSession, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at, rotated_from_id::text, created_at, last_used_at, COALESCE(user_agent, ''), COALESCE(ip_address, '')
		FROM refresh_tokens WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.RefreshSession{}
	for rows.Next() {
		var s models.RefreshSession
		var revokedAt, lastUsedAt sql.NullTime
		var rotatedFrom sql.NullString
		if err := rows.Scan(&s.ID, &s.UserID, &s.TokenHash, &s.ExpiresAt, &revokedAt, &rotatedFrom, &s.CreatedAt, &lastUsedAt, &s.UserAgent, &s.IPAddress); err != nil {
			return nil, err
		}
		if revokedAt.Valid {
			s.RevokedAt = &revokedAt.Time
		}
		if lastUsedAt.Valid {
			s.LastUsedAt = &lastUsedAt.Time
		}
		if rotatedFrom.Valid {
			s.RotatedFromID = rotatedFrom.String
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ClearflowRepository) EnqueueJob(ctx context.Context, orgID, userID, jobType string, payload string) (models.Job, error) {
	now := time.Now().UTC()
	job := models.Job{ID: auth.NewID(), OrganizationID: orgID, UserID: userID, Type: jobType, Status: "queued", PayloadJSON: fallback(payload, "{}"), Attempts: 0, MaxAttempts: 3, RunAfter: now, CreatedAt: now, UpdatedAt: now}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sync_jobs (id, organization_id, user_id, source, type, status, payload_json, attempts, max_attempts, run_after, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4, 'queued', $5::jsonb, 0, 3, $6, $6, $6)
	`, job.ID, job.OrganizationID, job.UserID, job.Type, job.PayloadJSON, now)
	return job, err
}

func (r *ClearflowRepository) GetJob(ctx context.Context, orgID, jobID string) (models.Job, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, organization_id, COALESCE(user_id::text, ''), COALESCE(type, source), status, payload_json::text, attempts, max_attempts, run_after,
		       locked_at, COALESCE(locked_by, ''), started_at, completed_at, failed_at, COALESCE(error, ''), created_at, updated_at
		FROM sync_jobs WHERE organization_id = $1 AND id = $2
	`, orgID, jobID)
	if err != nil {
		return models.Job{}, err
	}
	defer rows.Close()
	jobs, err := scanJobs(rows)
	if err != nil {
		return models.Job{}, err
	}
	if len(jobs) == 0 {
		return models.Job{}, sql.ErrNoRows
	}
	return jobs[0], nil
}

func (r *ClearflowRepository) ListJobs(ctx context.Context, orgID string, limit, offset int) ([]models.Job, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, organization_id, COALESCE(user_id::text, ''), COALESCE(type, source), status, payload_json::text, attempts, max_attempts, run_after,
		       locked_at, COALESCE(locked_by, ''), started_at, completed_at, failed_at, COALESCE(error, ''), created_at, updated_at
		FROM sync_jobs WHERE organization_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

func (r *ClearflowRepository) JobStatusCounts(ctx context.Context) (map[string]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM sync_jobs GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int64{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

func (r *ClearflowRepository) ClaimJob(ctx context.Context, worker string) (models.Job, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Job{}, false, err
	}
	defer rollback(tx)
	var job models.Job
	err = tx.QueryRowContext(ctx, `
		SELECT id, organization_id, COALESCE(user_id::text, ''), COALESCE(type, source), status, payload_json::text, attempts, max_attempts, run_after, created_at, updated_at
		FROM sync_jobs
		WHERE status IN ('queued', 'failed') AND run_after <= now() AND attempts < max_attempts
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`).Scan(&job.ID, &job.OrganizationID, &job.UserID, &job.Type, &job.Status, &job.PayloadJSON, &job.Attempts, &job.MaxAttempts, &job.RunAfter, &job.CreatedAt, &job.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Job{}, false, nil
	}
	if err != nil {
		return models.Job{}, false, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE sync_jobs SET status = 'running', locked_at = now(), locked_by = $2, started_at = COALESCE(started_at, now()), attempts = attempts + 1, updated_at = now()
		WHERE id = $1
	`, job.ID, worker); err != nil {
		return models.Job{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return models.Job{}, false, err
	}
	job.Status = "running"
	job.LockedBy = worker
	job.Attempts++
	return job, true, nil
}

func (r *ClearflowRepository) CompleteJob(ctx context.Context, jobID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE sync_jobs SET status = 'completed', completed_at = now(), updated_at = now() WHERE id = $1`, jobID)
	return err
}

func (r *ClearflowRepository) FailJob(ctx context.Context, job models.Job, message string) error {
	status := "failed"
	if job.Attempts >= job.MaxAttempts {
		status = "dead"
	}
	_, err := r.db.ExecContext(ctx, `UPDATE sync_jobs SET status = $2, failed_at = now(), error = $3, run_after = now() + interval '30 seconds', updated_at = now() WHERE id = $1`, job.ID, status, message)
	return err
}

func (r *ClearflowRepository) WriteAudit(ctx context.Context, log models.AuditLog) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, organization_id, user_id, action, target_type, target_id, request_id, ip_address, user_agent, metadata_json, created_at)
		VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, COALESCE(NULLIF($10, ''), '{}')::jsonb, $11)
	`, fallback(log.ID, auth.NewID()), log.OrganizationID, log.UserID, log.Action, log.TargetType, log.TargetID, log.RequestID, log.IPAddress, log.UserAgent, fallback(log.Metadata, "{}"), zeroTime(log.CreatedAt, time.Now().UTC()))
	return err
}

func (r *ClearflowRepository) ListAuditLogs(ctx context.Context, orgID, action, actorID string, limit, offset int) ([]models.AuditLog, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(organization_id::text, ''), COALESCE(user_id::text, ''), action, COALESCE(target_type, ''), COALESCE(target_id, ''), COALESCE(request_id, ''), COALESCE(ip_address, ''), COALESCE(user_agent, ''), metadata_json::text, created_at
		FROM audit_logs
		WHERE organization_id = $1
		  AND ($2 = '' OR action = $2)
		  AND ($3 = '' OR user_id = NULLIF($3, '')::uuid)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`, orgID, action, actorID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.AuditLog{}
	for rows.Next() {
		var log models.AuditLog
		if err := rows.Scan(&log.ID, &log.OrganizationID, &log.UserID, &log.Action, &log.TargetType, &log.TargetID, &log.RequestID, &log.IPAddress, &log.UserAgent, &log.Metadata, &log.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, log)
	}
	return out, rows.Err()
}

func (r *ClearflowRepository) SavePlaidItem(ctx context.Context, orgID, userID, itemID, tokenCiphertext, institutionID, institutionName, cursor string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO plaid_items (id, organization_id, user_id, item_id, access_token_ciphertext, institution_id, institution_name, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'active', now(), now())
		ON CONFLICT (organization_id, item_id) DO UPDATE
		SET access_token_ciphertext = EXCLUDED.access_token_ciphertext, institution_id = EXCLUDED.institution_id, institution_name = EXCLUDED.institution_name, updated_at = now()
	`, auth.NewID(), orgID, userID, itemID, tokenCiphertext, institutionID, institutionName)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO plaid_sync_state (id, organization_id, plaid_item_id, cursor, created_at, updated_at)
		SELECT $1, organization_id, id, $2, now(), now() FROM plaid_items WHERE organization_id = $3 AND item_id = $4
		ON CONFLICT (plaid_item_id) DO UPDATE SET cursor = EXCLUDED.cursor, updated_at = now()
	`, auth.NewID(), cursor, orgID, itemID)
	return err
}

func scanJobs(rows *sql.Rows) ([]models.Job, error) {
	out := []models.Job{}
	for rows.Next() {
		var job models.Job
		var lockedAt, startedAt, completedAt, failedAt sql.NullTime
		if err := rows.Scan(&job.ID, &job.OrganizationID, &job.UserID, &job.Type, &job.Status, &job.PayloadJSON, &job.Attempts, &job.MaxAttempts, &job.RunAfter, &lockedAt, &job.LockedBy, &startedAt, &completedAt, &failedAt, &job.Error, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		if lockedAt.Valid {
			job.LockedAt = lockedAt.Time
		}
		if startedAt.Valid {
			job.StartedAt = startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = completedAt.Time
		}
		if failedAt.Valid {
			job.FailedAt = failedAt.Time
		}
		out = append(out, job)
	}
	return out, rows.Err()
}
