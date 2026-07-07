package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/authz"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
	"github.com/StephenShao90/Fynora/services/api/internal/processors"
	"github.com/StephenShao90/Fynora/services/api/internal/repository"
	"github.com/StephenShao90/Fynora/services/api/internal/security"
)

func (a *app) stripeConnectURLV1(w http.ResponseWriter, r *http.Request) {
	r, ok := a.withV1Organization(w, r, false, authz.CanManageMembers)
	if !ok {
		return
	}
	org := r.Context().Value(clearflowOrgContextKey{}).(models.Organization)
	state, err := randomState()
	if err != nil {
		errorJSON(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create OAuth state")
		return
	}
	row := models.OAuthState{ID: auth.NewID(), OrganizationID: org.ID, UserID: userID(r), Provider: "stripe", StateHash: hashOAuthState(state), RedirectURI: a.cfg.StripeRedirectURL, ExpiresAt: time.Now().UTC().Add(10 * time.Minute), CreatedAt: time.Now().UTC()}
	if err := a.saveOAuthState(r.Context(), row); err != nil {
		errorJSON(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "could not persist OAuth state")
		return
	}
	client := a.stripeOAuthClient()
	writeJSON(w, http.StatusOK, map[string]string{"url": client.ConnectURL(a.cfg.StripeClientID, a.cfg.StripeRedirectURL, state), "state": state})
}

func (a *app) stripeCallbackV1(w http.ResponseWriter, r *http.Request) {
	r, ok := a.withV1Organization(w, r, false, authz.CanManageMembers)
	if !ok {
		return
	}
	org := r.Context().Value(clearflowOrgContextKey{}).(models.Organization)
	if providerErr := r.URL.Query().Get("error"); providerErr != "" {
		a.writeAudit(r.Context(), r, org.ID, userID(r), "stripe.connect_failed", "provider_connection", "stripe", `{"error":"provider_error"}`)
		errorJSON(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Stripe OAuth returned an error")
		return
	}
	code, state := r.URL.Query().Get("code"), r.URL.Query().Get("state")
	if code == "" || state == "" {
		errorJSON(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "code and state are required")
		return
	}
	oauthState, err := a.consumeOAuthState(r.Context(), "stripe", hashOAuthState(state))
	if err != nil {
		a.writeAudit(r.Context(), r, org.ID, userID(r), "stripe.connect_failed", "provider_connection", "stripe", `{"error":"invalid_state"}`)
		errorJSON(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}
	if oauthState.OrganizationID != org.ID || oauthState.UserID != userID(r) {
		errorJSON(w, r, http.StatusForbidden, "FORBIDDEN", "OAuth state does not match this organization")
		return
	}
	client := a.stripeOAuthClient()
	account, err := client.ExchangeCode(r.Context(), code)
	if err != nil {
		a.incrementMetric(func(m *opsMetrics) { m.StripeOAuthExchangeFailuresTotal++ })
		a.writeAudit(r.Context(), r, org.ID, userID(r), "stripe.connect_failed", "provider_connection", "stripe", `{"error":"exchange_failed"}`)
		errorJSON(w, r, http.StatusBadGateway, "INTERNAL_ERROR", "could not exchange Stripe authorization code")
		return
	}
	protector, err := security.NewTokenProtector(a.cfg.AppEnv, a.cfg.ProviderTokenEncryptionKey)
	if err != nil {
		errorJSON(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "provider token encryption is not configured")
		return
	}
	accessCipher, err := protector.Protect(r.Context(), account.AccessToken)
	if err != nil {
		errorJSON(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "could not protect provider token")
		return
	}
	refreshCipher, err := protector.Protect(r.Context(), account.RefreshToken)
	if err != nil {
		errorJSON(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "could not protect provider token")
		return
	}
	conn := models.ProviderConnection{ID: auth.NewID(), OrganizationID: org.ID, Provider: "stripe", ExternalAccountID: account.AccountID, DisplayName: account.DisplayName, Status: "connected", AccessTokenCiphertext: accessCipher, RefreshTokenCiphertext: refreshCipher, Scopes: account.Scope, ConnectedByUserID: userID(r), ConnectedAt: time.Now().UTC()}
	if err := a.upsertProviderConnection(r.Context(), conn); err != nil {
		errorJSON(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "could not persist Stripe connection")
		return
	}
	a.writeAudit(r.Context(), r, org.ID, userID(r), "stripe.connected", "provider_connection", conn.ExternalAccountID, "{}")
	a.emitOutbox(r.Context(), org.ID, "stripe.account_connected", "provider_connection", conn.ExternalAccountID, "{}")
	writeJSON(w, http.StatusOK, stripeStatusResponse(conn))
}

func (a *app) stripeStatusV1(w http.ResponseWriter, r *http.Request) {
	r, ok := a.withV1Organization(w, r, false, authz.CanRead)
	if !ok {
		return
	}
	org := r.Context().Value(clearflowOrgContextKey{}).(models.Organization)
	conn, err := a.getProviderConnection(r.Context(), org.ID, "stripe")
	if err != nil || conn.Status != "connected" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"connected": false, "provider": "stripe"})
		return
	}
	writeJSON(w, http.StatusOK, stripeStatusResponse(conn))
}

func (a *app) stripeDisconnectV1(w http.ResponseWriter, r *http.Request) {
	r, ok := a.withV1Organization(w, r, false, authz.CanManageMembers)
	if !ok {
		return
	}
	org := r.Context().Value(clearflowOrgContextKey{}).(models.Organization)
	conn, _ := a.getProviderConnection(r.Context(), org.ID, "stripe")
	if conn.Status == "connected" && conn.AccessTokenCiphertext != "" {
		if protector, err := security.NewTokenProtector(a.cfg.AppEnv, a.cfg.ProviderTokenEncryptionKey); err == nil {
			if token, err := protector.Unprotect(r.Context(), conn.AccessTokenCiphertext); err == nil {
				if err := a.stripeOAuthClient().Revoke(r.Context(), conn.ExternalAccountID, token); err != nil {
					conn.LastError = "remote Stripe revocation deferred: " + err.Error()
				}
			}
		}
	}
	if err := a.disconnectProviderConnection(r.Context(), org.ID, "stripe", conn.LastError); err != nil {
		errorJSON(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "could not disconnect Stripe")
		return
	}
	a.writeAudit(r.Context(), r, org.ID, userID(r), "stripe.disconnected", "provider_connection", conn.ExternalAccountID, "{}")
	a.emitOutbox(r.Context(), org.ID, "stripe.account_disconnected", "provider_connection", conn.ExternalAccountID, "{}")
	writeJSON(w, http.StatusOK, map[string]interface{}{"connected": false, "provider": "stripe"})
}

func (a *app) stripeOAuthClient() processors.StripeOAuthClient {
	if a.cfg.StripeClientID != "" && a.cfg.StripeSecretKey != "" {
		return processors.HTTPStripeOAuthClient{ClientID: a.cfg.StripeClientID, SecretKey: a.cfg.StripeSecretKey, RedirectURI: a.cfg.StripeRedirectURL}
	}
	return processors.MockStripeOAuthClient{}
}

func stripeStatusResponse(conn models.ProviderConnection) map[string]interface{} {
	return map[string]interface{}{
		"connected":   conn.Status == "connected",
		"provider":    conn.Provider,
		"accountId":   conn.ExternalAccountID,
		"displayName": conn.DisplayName,
		"connectedAt": conn.ConnectedAt,
		"lastSyncAt":  conn.LastSyncAt,
		"lastError":   conn.LastError,
	}
}

func randomState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashOAuthState(state string) string {
	sum := sha256.Sum256([]byte(state))
	return hex.EncodeToString(sum[:])
}

func (a *app) saveOAuthState(ctx context.Context, state models.OAuthState) error {
	if a.cfRepo != nil {
		_, err := a.cfRepo.CreateOAuthState(ctx, state)
		return err
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	a.store.oauthStates[state.StateHash] = state
	return nil
}

func (a *app) consumeOAuthState(ctx context.Context, provider, stateHash string) (models.OAuthState, error) {
	var state models.OAuthState
	var err error
	if a.cfRepo != nil {
		state, err = a.cfRepo.GetOAuthStateByHash(ctx, provider, stateHash)
		if err != nil {
			return state, errors.New("invalid OAuth state")
		}
		if !state.UsedAt.IsZero() {
			return state, errors.New("OAuth state was already used")
		}
		if time.Now().UTC().After(state.ExpiresAt) {
			return state, errors.New("OAuth state expired")
		}
		return state, a.cfRepo.MarkOAuthStateUsed(ctx, state.ID)
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	state, ok := a.store.oauthStates[stateHash]
	if !ok || state.Provider != provider {
		return state, errors.New("invalid OAuth state")
	}
	if !state.UsedAt.IsZero() {
		return state, errors.New("OAuth state was already used")
	}
	if time.Now().UTC().After(state.ExpiresAt) {
		return state, errors.New("OAuth state expired")
	}
	state.UsedAt = time.Now().UTC()
	a.store.oauthStates[stateHash] = state
	return state, nil
}

func (a *app) upsertProviderConnection(ctx context.Context, conn models.ProviderConnection) error {
	if a.cfRepo != nil {
		_, err := a.cfRepo.UpsertProviderConnection(ctx, conn)
		return err
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	a.store.providerConnections[conn.OrganizationID+":"+conn.Provider] = conn
	return nil
}

func (a *app) getProviderConnection(ctx context.Context, orgID, provider string) (models.ProviderConnection, error) {
	if a.cfRepo != nil {
		return a.cfRepo.GetProviderConnection(ctx, orgID, provider)
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	conn, ok := a.store.providerConnections[orgID+":"+provider]
	if !ok {
		return models.ProviderConnection{}, repository.ErrNotFound
	}
	return conn, nil
}

func (a *app) disconnectProviderConnection(ctx context.Context, orgID, provider, lastError string) error {
	if a.cfRepo != nil {
		return a.cfRepo.DisconnectProviderConnection(ctx, orgID, provider, lastError)
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	key := orgID + ":" + provider
	conn := a.store.providerConnections[key]
	conn.Status = "disconnected"
	conn.LastError = lastError
	conn.DisconnectedAt = time.Now().UTC()
	a.store.providerConnections[key] = conn
	return nil
}
