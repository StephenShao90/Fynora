package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

func (r *ClearflowRepository) CreateOAuthState(ctx context.Context, state models.OAuthState) (models.OAuthState, error) {
	now := time.Now().UTC()
	if state.ID == "" {
		state.ID = auth.NewID()
	}
	state.CreatedAt = zeroTime(state.CreatedAt, now)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO oauth_states (id, organization_id, user_id, provider, state_hash, redirect_uri, expires_at, used_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, state.ID, state.OrganizationID, state.UserID, state.Provider, state.StateHash, state.RedirectURI, state.ExpiresAt, nullableTime(state.UsedAt), state.CreatedAt)
	return state, err
}

func (r *ClearflowRepository) GetOAuthStateByHash(ctx context.Context, provider, stateHash string) (models.OAuthState, error) {
	var state models.OAuthState
	var used sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, organization_id::text, user_id::text, provider, state_hash, redirect_uri, expires_at, used_at, created_at
		FROM oauth_states WHERE provider = $1 AND state_hash = $2
	`, provider, stateHash).Scan(&state.ID, &state.OrganizationID, &state.UserID, &state.Provider, &state.StateHash, &state.RedirectURI, &state.ExpiresAt, &used, &state.CreatedAt)
	if used.Valid {
		state.UsedAt = used.Time
	}
	return state, err
}

func (r *ClearflowRepository) MarkOAuthStateUsed(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE oauth_states SET used_at = now() WHERE id = $1 AND used_at IS NULL`, id)
	return err
}

func (r *ClearflowRepository) UpsertProviderConnection(ctx context.Context, conn models.ProviderConnection) (models.ProviderConnection, error) {
	now := time.Now().UTC()
	if conn.ID == "" {
		conn.ID = auth.NewID()
	}
	conn.Status = fallback(conn.Status, "connected")
	conn.ConnectedAt = zeroTime(conn.ConnectedAt, now)
	conn.CreatedAt = zeroTime(conn.CreatedAt, now)
	conn.UpdatedAt = now
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO provider_connections (id, organization_id, provider, external_account_id, display_name, status, access_token_ciphertext, refresh_token_ciphertext, scopes, connected_by_user_id, connected_at, disconnected_at, last_sync_at, last_error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10, '')::uuid, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (organization_id, provider) DO UPDATE
		SET external_account_id = EXCLUDED.external_account_id,
		    display_name = EXCLUDED.display_name,
		    status = EXCLUDED.status,
		    access_token_ciphertext = EXCLUDED.access_token_ciphertext,
		    refresh_token_ciphertext = EXCLUDED.refresh_token_ciphertext,
		    scopes = EXCLUDED.scopes,
		    connected_by_user_id = EXCLUDED.connected_by_user_id,
		    connected_at = EXCLUDED.connected_at,
		    disconnected_at = NULL,
		    last_error = NULL,
		    updated_at = EXCLUDED.updated_at
	`, conn.ID, conn.OrganizationID, conn.Provider, conn.ExternalAccountID, conn.DisplayName, conn.Status, conn.AccessTokenCiphertext, conn.RefreshTokenCiphertext, conn.Scopes, conn.ConnectedByUserID, conn.ConnectedAt, nullableTime(conn.DisconnectedAt), nullableTime(conn.LastSyncAt), conn.LastError, conn.CreatedAt, conn.UpdatedAt)
	return conn, err
}

func (r *ClearflowRepository) GetProviderConnection(ctx context.Context, orgID, provider string) (models.ProviderConnection, error) {
	var conn models.ProviderConnection
	var connectedAt, disconnectedAt, lastSyncAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, organization_id::text, provider, external_account_id, display_name, status, access_token_ciphertext, refresh_token_ciphertext, COALESCE(scopes, ''), COALESCE(connected_by_user_id::text, ''), connected_at, disconnected_at, last_sync_at, COALESCE(last_error, ''), created_at, updated_at
		FROM provider_connections WHERE organization_id = $1 AND provider = $2
	`, orgID, provider).Scan(&conn.ID, &conn.OrganizationID, &conn.Provider, &conn.ExternalAccountID, &conn.DisplayName, &conn.Status, &conn.AccessTokenCiphertext, &conn.RefreshTokenCiphertext, &conn.Scopes, &conn.ConnectedByUserID, &connectedAt, &disconnectedAt, &lastSyncAt, &conn.LastError, &conn.CreatedAt, &conn.UpdatedAt)
	if connectedAt.Valid {
		conn.ConnectedAt = connectedAt.Time
	}
	if disconnectedAt.Valid {
		conn.DisconnectedAt = disconnectedAt.Time
	}
	if lastSyncAt.Valid {
		conn.LastSyncAt = lastSyncAt.Time
	}
	return conn, err
}

func (r *ClearflowRepository) DisconnectProviderConnection(ctx context.Context, orgID, provider, lastError string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE provider_connections SET status = 'disconnected', disconnected_at = now(), last_error = NULLIF($3, ''), updated_at = now() WHERE organization_id = $1 AND provider = $2`, orgID, provider, lastError)
	return err
}

func nullableTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}
