package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/authz"
	"github.com/StephenShao90/Fynora/services/api/internal/httpapi"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
	"github.com/StephenShao90/Fynora/services/api/internal/repository"
	"github.com/StephenShao90/Fynora/services/api/internal/validation"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
)

type rateLimitBucket struct {
	Count     int
	ResetAt   time.Time
	RetryTime time.Time
}

func (a *app) authRateLimited(next http.HandlerFunc) http.HandlerFunc {
	return a.rateLimited("auth", 5, time.Minute, func(r *http.Request) string { return clientIP(r) }, next)
}

func (a *app) heavyRateLimited(next http.HandlerFunc) http.HandlerFunc {
	return a.rateLimited("heavy", 10, time.Minute, func(r *http.Request) string {
		key := userID(r)
		if orgID := organizationIDFromRequest(r); orgID != "" {
			key += ":" + orgID
		}
		return key
	}, next)
}

func (a *app) rateLimited(scope string, limit int, window time.Duration, keyFn func(*http.Request) string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := scope + ":" + keyFn(r)
		if a.rateLimiter != nil {
			decision, err := a.rateLimiter.Allow(r.Context(), key, limit, window)
			if err == nil {
				if !decision.Allowed {
					a.incrementMetric(func(m *opsMetrics) {
						m.RateLimitedRequestsTotal++
						if redisEnabled(a.cfg.RedisEnabled) {
							m.RedisRateLimitHitsTotal++
						}
					})
					retry := int(decision.RetryAfter.Seconds())
					if retry < 1 {
						retry = 1
					}
					w.Header().Set("Retry-After", strconvItoa(retry))
					errorJSON(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
					return
				}
				next(w, r)
				return
			}
			if a.cfg.AppEnv == "production" && redisEnabled(a.cfg.RedisEnabled) {
				errorJSON(w, r, http.StatusServiceUnavailable, "INTERNAL_ERROR", "rate limiter unavailable")
				return
			}
		}
		now := time.Now().UTC()
		a.store.mu.Lock()
		bucket := a.store.rateLimits[key]
		if bucket.ResetAt.IsZero() || now.After(bucket.ResetAt) {
			bucket = rateLimitBucket{ResetAt: now.Add(window)}
		}
		bucket.Count++
		limited := bucket.Count > limit
		retry := int(time.Until(bucket.ResetAt).Seconds())
		if retry < 1 {
			retry = 1
		}
		a.store.rateLimits[key] = bucket
		a.store.mu.Unlock()
		if limited {
			a.incrementMetric(func(m *opsMetrics) { m.RateLimitedRequestsTotal++ })
			w.Header().Set("Retry-After", strconvItoa(retry))
			errorJSON(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
			return
		}
		next(w, r)
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func strconvItoa(v int) string {
	return strconv.FormatInt(int64(v), 10)
}

func (a *app) issueAuthTokens(w http.ResponseWriter, r *http.Request, status int, user models.User, memberships []models.OrganizationMember) {
	access, _ := auth.SignJWT(a.cfg.JWTSecret, user.ID, user.Email, accessTokenTTL)
	refresh, session, err := a.createRefreshSession(r.Context(), user.ID, r.UserAgent(), clientIP(r), "")
	if err != nil {
		errorJSON(w, r, 500, "INTERNAL_ERROR", "could not create session")
		return
	}
	a.writeAudit(r.Context(), r, "", user.ID, "auth.login", "session", session.ID, "{}")
	writeAuthJSON(w, status, map[string]interface{}{
		"token":         access,
		"accessToken":   access,
		"refreshToken":  refresh,
		"user":          user,
		"organizations": membershipOrganizations(memberships),
	})
}

func (a *app) createRefreshSession(ctx context.Context, userID, userAgent, ip, rotatedFromID string) (string, models.RefreshSession, error) {
	token := auth.NewID() + "." + auth.NewID()
	hash := hashToken(token)
	expiresAt := time.Now().UTC().Add(refreshTokenTTL)
	if a.cfRepo != nil {
		session, err := a.cfRepo.CreateRefreshSession(ctx, userID, hash, expiresAt, userAgent, ip, rotatedFromID)
		return token, session, err
	}
	session := models.RefreshSession{ID: auth.NewID(), UserID: userID, TokenHash: hash, ExpiresAt: expiresAt, RotatedFromID: rotatedFromID, CreatedAt: time.Now().UTC(), UserAgent: userAgent, IPAddress: ip}
	a.store.mu.Lock()
	a.store.refreshSessions[session.ID] = session
	a.store.refreshTokensByHash[hash] = session.ID
	a.store.mu.Unlock()
	return token, session, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (a *app) refreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if !decode(w, r, &req) {
		return
	}
	session, err := a.refreshSessionByToken(r.Context(), req.RefreshToken)
	if err != nil || session.RevokedAt != nil || time.Now().UTC().After(session.ExpiresAt) {
		errorJSON(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token")
		return
	}
	user, err := a.userByID(r.Context(), session.UserID)
	if err != nil {
		errorJSON(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token")
		return
	}
	if err := a.revokeRefreshSession(r.Context(), session.ID); err != nil {
		errorJSON(w, r, 500, "INTERNAL_ERROR", "could not rotate refresh token")
		return
	}
	access, _ := auth.SignJWT(a.cfg.JWTSecret, user.ID, user.Email, accessTokenTTL)
	refresh, nextSession, err := a.createRefreshSession(r.Context(), user.ID, r.UserAgent(), clientIP(r), session.ID)
	if err != nil {
		errorJSON(w, r, 500, "INTERNAL_ERROR", "could not rotate refresh token")
		return
	}
	_ = a.markRefreshSessionUsed(r.Context(), nextSession.ID)
	a.writeAudit(r.Context(), r, "", user.ID, "auth.refresh_rotated", "session", nextSession.ID, "{}")
	writeAuthJSON(w, 200, map[string]interface{}{"accessToken": access, "token": access, "refreshToken": refresh})
}

func (a *app) logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if !decode(w, r, &req) {
		return
	}
	session, err := a.refreshSessionByToken(r.Context(), req.RefreshToken)
	if err == nil {
		_ = a.revokeRefreshSession(r.Context(), session.ID)
		a.writeAudit(r.Context(), r, "", session.UserID, "auth.logout", "session", session.ID, "{}")
	}
	setAuthNoStore(w)
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) listSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := a.sessionsForUser(r.Context(), userID(r))
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not list sessions")
		return
	}
	writeJSON(w, 200, sessions)
}

func (a *app) revokeSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")
	if err := validation.UUID(sessionID, "sessionId"); err != nil {
		errorJSON(w, r, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	revoked, err := a.revokeRefreshSessionForUser(r.Context(), userID(r), sessionID)
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not revoke session")
		return
	}
	if !revoked {
		errorJSON(w, r, http.StatusNotFound, "NOT_FOUND", "session not found")
		return
	}
	a.writeAudit(r.Context(), r, "", userID(r), "auth.session_revoked", "session", sessionID, "{}")
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) refreshSessionByToken(ctx context.Context, token string) (models.RefreshSession, error) {
	hash := hashToken(token)
	if a.cfRepo != nil {
		return a.cfRepo.GetRefreshSessionByHash(ctx, hash)
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	id, ok := a.store.refreshTokensByHash[hash]
	if !ok {
		return models.RefreshSession{}, repository.ErrNotFound
	}
	return a.store.refreshSessions[id], nil
}

func (a *app) revokeRefreshSession(ctx context.Context, sessionID string) error {
	if a.cfRepo != nil {
		return a.cfRepo.RevokeRefreshSession(ctx, sessionID)
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	s := a.store.refreshSessions[sessionID]
	now := time.Now().UTC()
	s.RevokedAt = &now
	a.store.refreshSessions[sessionID] = s
	return nil
}

func (a *app) revokeRefreshSessionForUser(ctx context.Context, uid, sessionID string) (bool, error) {
	if a.cfRepo != nil {
		return a.cfRepo.RevokeRefreshSessionForUser(ctx, uid, sessionID)
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	s, ok := a.store.refreshSessions[sessionID]
	if !ok || s.UserID != uid {
		return false, nil
	}
	now := time.Now().UTC()
	s.RevokedAt = &now
	a.store.refreshSessions[sessionID] = s
	return true, nil
}

func (a *app) markRefreshSessionUsed(ctx context.Context, sessionID string) error {
	if a.cfRepo != nil {
		return a.cfRepo.MarkRefreshSessionUsed(ctx, sessionID)
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	s := a.store.refreshSessions[sessionID]
	now := time.Now().UTC()
	s.LastUsedAt = &now
	a.store.refreshSessions[sessionID] = s
	return nil
}

func (a *app) sessionsForUser(ctx context.Context, uid string) ([]models.RefreshSession, error) {
	if a.cfRepo != nil {
		return a.cfRepo.ListRefreshSessions(ctx, uid)
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.RefreshSession{}
	for _, s := range a.store.refreshSessions {
		if s.UserID == uid {
			out = append(out, s)
		}
	}
	return out, nil
}

func (a *app) userByID(ctx context.Context, uid string) (models.User, error) {
	if a.cfRepo != nil {
		return a.cfRepo.GetUserByID(ctx, uid)
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	u, ok := a.store.users[uid]
	if !ok {
		return models.User{}, repository.ErrNotFound
	}
	return u, nil
}

func (a *app) enqueueFinancialJob(w http.ResponseWriter, r *http.Request, jobType string) {
	ctx, span := a.tracer.Start(r.Context(), "job.enqueue.request", map[string]string{"job.type": jobType})
	defer span.End()
	r = r.WithContext(ctx)
	r, ok := a.withV1Organization(w, r, true, authz.CanWriteFinancialData)
	if !ok {
		return
	}
	org := r.Context().Value(clearflowOrgContextKey{}).(models.Organization)
	body, _ := readAndRestoreBody(r)
	if replayed, ok := a.handleIdempotencyReplay(w, r, org.ID, body); ok {
		if replayed {
			return
		}
	}
	job, err := a.enqueueJob(r.Context(), org.ID, userID(r), jobType, string(body))
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not enqueue job")
		return
	}
	payload := map[string]interface{}{"jobId": job.ID, "status": job.Status}
	a.saveIdempotencyResponse(r, org.ID, body, http.StatusAccepted, payload)
	a.writeAudit(r.Context(), r, org.ID, userID(r), jobType+".queued", "job", job.ID, "{}")
	a.emitOutbox(r.Context(), org.ID, "job.queued", "job", job.ID, `{"type":"`+jobType+`"}`)
	a.incrementMetric(func(m *opsMetrics) {
		m.JobsQueuedTotal++
		m.JobQueueDepth++
	})
	writeJSON(w, http.StatusAccepted, payload)
}

func readAndRestoreBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return []byte("{}"), nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		body = []byte("{}")
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func (a *app) handleIdempotencyReplay(w http.ResponseWriter, r *http.Request, orgID string, body []byte) (bool, bool) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		return false, false
	}
	requestHash := requestBodyHash(r, orgID, body)
	scopedKey := orgID + ":" + key
	if a.cfRepo != nil {
		status, stored, ok, err := a.cfRepo.ReadIdempotency(r.Context(), userID(r), scopedKey, requestHash)
		if err == repository.ErrIdempotencyConflict {
			errorJSON(w, r, http.StatusConflict, "CONFLICT", "idempotency key reused with a different request body")
			return true, true
		}
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not read idempotency key")
			return true, true
		}
		if ok {
			a.incrementMetric(func(m *opsMetrics) { m.IdempotencyReplaysTotal++ })
			w.Header().Set("Idempotency-Replayed", "true")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write(stored)
			return true, true
		}
		if a.idempotencyLocks != nil && redisEnabled(a.cfg.RedisEnabled) {
			acquired, err := a.idempotencyLocks.Acquire(r.Context(), "idem:"+userID(r)+":"+scopedKey, requestHash, time.Minute)
			if err == nil && !acquired {
				errorJSON(w, r, http.StatusConflict, "CONFLICT", "idempotency key is already in flight")
				return true, true
			}
			if err == nil && acquired {
				a.incrementMetric(func(m *opsMetrics) { m.RedisIdempotencyLocksTotal++ })
			}
		}
		return false, false
	}
	recordKey := userID(r) + ":" + orgID + ":" + key
	a.store.mu.RLock()
	record, ok := a.store.idempotencyRecords[recordKey]
	a.store.mu.RUnlock()
	if !ok {
		if a.idempotencyLocks != nil && redisEnabled(a.cfg.RedisEnabled) {
			acquired, err := a.idempotencyLocks.Acquire(r.Context(), "idem:"+recordKey, requestHash, time.Minute)
			if err == nil && !acquired {
				errorJSON(w, r, http.StatusConflict, "CONFLICT", "idempotency key is already in flight")
				return true, true
			}
			if err == nil && acquired {
				a.incrementMetric(func(m *opsMetrics) { m.RedisIdempotencyLocksTotal++ })
			}
		}
		return false, false
	}
	if record.RequestHash != requestHash {
		errorJSON(w, r, http.StatusConflict, "CONFLICT", "idempotency key reused with a different request body")
		return true, true
	}
	w.Header().Set("Idempotency-Replayed", "true")
	a.incrementMetric(func(m *opsMetrics) { m.IdempotencyReplaysTotal++ })
	writeJSON(w, record.StatusCode, json.RawMessage(record.ResponseBody))
	return true, true
}

func (a *app) saveIdempotencyResponse(r *http.Request, orgID string, body []byte, status int, payload interface{}) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		return
	}
	encoded, _ := json.Marshal(payload)
	requestHash := requestBodyHash(r, orgID, body)
	scopedKey := orgID + ":" + key
	if a.cfRepo != nil {
		_ = a.cfRepo.SaveIdempotency(r.Context(), userID(r), orgID, scopedKey, requestHash, status, encoded)
		if a.idempotencyLocks != nil && redisEnabled(a.cfg.RedisEnabled) {
			_ = a.idempotencyLocks.Release(r.Context(), "idem:"+userID(r)+":"+scopedKey)
		}
		return
	}
	recordKey := userID(r) + ":" + orgID + ":" + key
	a.store.mu.Lock()
	a.store.idempotencyRecords[recordKey] = models.IdempotencyRecord{Key: key, UserID: userID(r), OrganizationID: orgID, RequestHash: requestHash, StatusCode: status, ResponseBody: string(encoded), CreatedAt: time.Now().UTC()}
	a.store.mu.Unlock()
	if a.idempotencyLocks != nil && redisEnabled(a.cfg.RedisEnabled) {
		_ = a.idempotencyLocks.Release(r.Context(), "idem:"+recordKey)
	}
}

func requestBodyHash(r *http.Request, orgID string, body []byte) string {
	sum := sha256.Sum256([]byte(r.Method + "|" + r.URL.Path + "|" + orgID + "|" + string(body)))
	return hex.EncodeToString(sum[:])
}

func (a *app) enqueueJob(ctx context.Context, orgID, uid, jobType, payload string) (models.Job, error) {
	ctx, span := a.tracer.Start(ctx, "job.enqueue", map[string]string{"job.type": jobType, "organization.id": orgID})
	defer span.End()
	payload = a.payloadWithTrace(ctx, payload)
	if a.cfRepo != nil {
		return a.cfRepo.EnqueueJob(ctx, orgID, uid, jobType, payload)
	}
	now := time.Now().UTC()
	if payload == "" {
		payload = "{}"
	}
	job := models.Job{ID: auth.NewID(), OrganizationID: orgID, UserID: uid, Type: jobType, Status: "queued", PayloadJSON: payload, MaxAttempts: 3, RunAfter: now, CreatedAt: now, UpdatedAt: now}
	a.store.mu.Lock()
	a.store.jobs[job.ID] = job
	a.store.mu.Unlock()
	return job, nil
}

func (a *app) payloadWithTrace(ctx context.Context, payload string) string {
	if !a.tracer.Enabled {
		return payload
	}
	if strings.TrimSpace(payload) == "" {
		payload = "{}"
	}
	var object map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &object); err != nil {
		return payload
	}
	headers := map[string]string{}
	a.tracer.Inject(ctx, headers)
	if len(headers) == 0 {
		return payload
	}
	object["_trace"] = headers
	encoded, err := json.Marshal(object)
	if err != nil {
		return payload
	}
	return string(encoded)
}

func (a *app) getJobV1(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireQueryOrganizationID(w, r)
	if !ok {
		return
	}
	if _, ok := a.requireOrgRole(w, r, orgID, authz.CanRead); !ok {
		return
	}
	job, err := a.jobByID(r.Context(), orgID, r.PathValue("jobId"))
	if err != nil {
		errorJSON(w, r, 404, "NOT_FOUND", "job not found")
		return
	}
	writeJSON(w, 200, job)
}

func (a *app) listJobsV1(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireQueryOrganizationID(w, r)
	if !ok {
		return
	}
	if _, ok := a.requireOrgRole(w, r, orgID, authz.CanRead); !ok {
		return
	}
	query, err := httpapi.ParseListQuery(r)
	if err != nil {
		errorJSON(w, r, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	jobs, err := a.jobsForOrg(r.Context(), orgID, query.Limit, query.Offset)
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not list jobs")
		return
	}
	writeJSON(w, 200, httpapi.Paginated(jobs, query))
}

func (a *app) listAuditLogsV1(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireQueryOrganizationID(w, r)
	if !ok {
		return
	}
	if _, ok := a.requireOrgRole(w, r, orgID, authz.CanManageMembers); !ok {
		return
	}
	query, err := httpapi.ParseListQuery(r)
	if err != nil {
		errorJSON(w, r, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	logs, err := a.auditLogsForOrg(r.Context(), orgID, r.URL.Query().Get("action"), r.URL.Query().Get("actorId"), query.Limit, query.Offset)
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not list audit logs")
		return
	}
	writeJSON(w, 200, httpapi.Paginated(logs, query))
}

func requireQueryOrganizationID(w http.ResponseWriter, r *http.Request) (string, bool) {
	orgID := organizationIDFromRequest(r)
	if orgID == "" {
		errorJSON(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "organizationId is required")
		return "", false
	}
	if err := validation.UUID(orgID, "organizationId"); err != nil {
		errorJSON(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return "", false
	}
	return orgID, true
}

func (a *app) jobByID(ctx context.Context, orgID, jobID string) (models.Job, error) {
	if a.cfRepo != nil {
		return a.cfRepo.GetJob(ctx, orgID, jobID)
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	job, ok := a.store.jobs[jobID]
	if !ok || job.OrganizationID != orgID {
		return models.Job{}, repository.ErrNotFound
	}
	return job, nil
}

func (a *app) jobsForOrg(ctx context.Context, orgID string, limit, offset int) ([]models.Job, error) {
	if a.cfRepo != nil {
		return a.cfRepo.ListJobs(ctx, orgID, limit, offset)
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	all := []models.Job{}
	for _, job := range a.store.jobs {
		if job.OrganizationID == orgID {
			all = append(all, job)
		}
	}
	return httpapi.Page(all, httpapi.ListQuery{Limit: limit, Offset: offset}), nil
}

func (a *app) writeAudit(ctx context.Context, r *http.Request, orgID, uid, action, targetType, targetID, metadata string) {
	if metadata == "" {
		metadata = "{}"
	}
	log := models.AuditLog{ID: auth.NewID(), OrganizationID: orgID, UserID: uid, Action: action, TargetType: targetType, TargetID: targetID, RequestID: r.Header.Get("X-Request-ID"), IPAddress: clientIP(r), UserAgent: r.UserAgent(), Metadata: metadata, CreatedAt: time.Now().UTC()}
	if a.cfRepo != nil {
		_ = a.cfRepo.WriteAudit(ctx, log)
		return
	}
	a.store.mu.Lock()
	a.store.auditLogs[log.ID] = log
	a.store.mu.Unlock()
}

func (a *app) auditLogsForOrg(ctx context.Context, orgID, action, actorID string, limit, offset int) ([]models.AuditLog, error) {
	if a.cfRepo != nil {
		return a.cfRepo.ListAuditLogs(ctx, orgID, action, actorID, limit, offset)
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	all := []models.AuditLog{}
	for _, log := range a.store.auditLogs {
		if log.OrganizationID == orgID && (action == "" || log.Action == action) && (actorID == "" || log.UserID == actorID) {
			all = append(all, log)
		}
	}
	return httpapi.Page(all, httpapi.ListQuery{Limit: limit, Offset: offset}), nil
}
