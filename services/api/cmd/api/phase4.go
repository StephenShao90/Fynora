package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/authz"
	"github.com/StephenShao90/Fynora/services/api/internal/httpapi"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
	"github.com/StephenShao90/Fynora/services/api/internal/plaid"
	"github.com/StephenShao90/Fynora/services/api/internal/processors"
	"github.com/StephenShao90/Fynora/services/api/internal/repository"
)

var processorProviderRE = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

type opsMetrics struct {
	HTTPRequestsTotal                     int64 `json:"http_requests_total"`
	HTTPErrorsTotal                       int64 `json:"http_errors_total"`
	HTTPRequestDurationMS                 int64 `json:"http_request_duration_ms"`
	JobsQueuedTotal                       int64 `json:"jobs_queued_total"`
	JobsCompletedTotal                    int64 `json:"jobs_completed_total"`
	JobsFailedTotal                       int64 `json:"jobs_failed_total"`
	JobsDeadTotal                         int64 `json:"jobs_dead_total"`
	JobQueueDepth                         int64 `json:"job_queue_depth"`
	PlaidWebhooksReceivedTotal            int64 `json:"plaid_webhooks_received_total"`
	PlaidSyncFailuresTotal                int64 `json:"plaid_sync_failures_total"`
	ReconciliationRunsTotal               int64 `json:"reconciliation_runs_total"`
	ReconciliationExceptions              int64 `json:"reconciliation_exceptions_total"`
	IdempotencyReplaysTotal               int64 `json:"idempotency_replays_total"`
	RateLimitedRequestsTotal              int64 `json:"rate_limited_requests_total"`
	RedisRateLimitHitsTotal               int64 `json:"redis_rate_limit_hits_total"`
	RedisIdempotencyLocksTotal            int64 `json:"redis_idempotency_locks_total"`
	StripeWebhookEventsTotal              int64 `json:"stripe_webhook_events_total"`
	StripeWebhookFailuresTotal            int64 `json:"stripe_webhook_failures_total"`
	StripeOAuthExchangeFailuresTotal      int64 `json:"stripe_oauth_exchange_failures_total"`
	PlaidWebhookVerificationFailuresTotal int64 `json:"plaid_webhook_verification_failures_total"`
	OTELTracesStartedTotal                int64 `json:"otel_traces_started_total"`
	JobRetryTotal                         int64 `json:"job_retry_total"`
	JobCancelTotal                        int64 `json:"job_cancel_total"`
	JobDeadTotal                          int64 `json:"job_dead_total"`
}

func (a *app) webhookRateLimited(next http.HandlerFunc) http.HandlerFunc {
	return a.rateLimited("webhook", 60, time.Minute, func(r *http.Request) string { return clientIP(r) }, next)
}

func (a *app) handlePlaidWebhook(w http.ResponseWriter, r *http.Request) {
	ctx, span := a.tracer.Start(r.Context(), "webhook.plaid.handle", map[string]string{"provider": "plaid"})
	defer span.End()
	r = r.WithContext(ctx)
	body, err := readLimitedBody(r, 2<<20)
	if err != nil {
		errorJSON(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid webhook body")
		return
	}
	var payload struct {
		WebhookType string `json:"webhook_type"`
		WebhookCode string `json:"webhook_code"`
		ItemID      string `json:"item_id"`
		Environment string `json:"environment"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.WebhookType == "" || payload.WebhookCode == "" {
		errorJSON(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid Plaid webhook payload")
		return
	}
	if !a.verifyPlaidWebhook(r, body) {
		a.incrementMetric(func(m *opsMetrics) { m.PlaidWebhookVerificationFailuresTotal++ })
		errorJSON(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid Plaid webhook signature")
		return
	}
	orgID := a.organizationForPlaidItem(r.Context(), payload.ItemID)
	event := models.WebhookEvent{ID: auth.NewID(), OrganizationID: orgID, Type: payload.WebhookType, Code: payload.WebhookCode, ItemID: payload.ItemID, Provider: payload.Environment, DedupeKey: plaidWebhookDedupeKey(payload.ItemID, payload.WebhookType, payload.WebhookCode, body), Status: "received", CreatedAt: time.Now().UTC()}
	created, err := a.savePlaidWebhookEvent(r.Context(), event, string(body))
	if err != nil {
		errorJSON(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "could not persist Plaid webhook")
		return
	}
	if created {
		a.incrementMetric(func(m *opsMetrics) { m.PlaidWebhooksReceivedTotal++ })
		a.emitOutbox(r.Context(), orgID, "plaid.webhook_received", "plaid_item", payload.ItemID, string(body))
		a.writeAudit(r.Context(), r, orgID, "", "plaid.webhook_received", "plaid_item", payload.ItemID, `{"webhook_type":"`+payload.WebhookType+`","webhook_code":"`+payload.WebhookCode+`"}`)
		if plaidShouldQueueSync(payload.WebhookType, payload.WebhookCode) && orgID != "" {
			job, err := a.enqueueJob(r.Context(), orgID, "", "plaid.transactions.sync", string(body))
			if err == nil {
				a.emitOutbox(r.Context(), orgID, "job.queued", "job", job.ID, `{"type":"plaid.transactions.sync"}`)
			}
		}
	}
	writeJSON(w, http.StatusAccepted, map[string]interface{}{"status": "accepted", "deduped": !created})
}

func (a *app) verifyPlaidWebhook(r *http.Request, body []byte) bool {
	verifier := plaid.ConfigurableWebhookVerifier{
		Enabled: plaid.WebhookVerificationEnabled(a.cfg.PlaidWebhookVerification),
		AppEnv:  a.cfg.AppEnv,
	}
	return verifier.Verify(r.Context(), body, r.Header) == nil
}

func plaidWebhookDedupeKey(itemID, webhookType, webhookCode string, body []byte) string {
	sum := sha256.Sum256([]byte(itemID + "|" + webhookType + "|" + webhookCode + "|" + canonicalPayloadHash(body)))
	return hex.EncodeToString(sum[:])
}

func canonicalPayloadHash(body []byte) string {
	var value interface{}
	if err := json.Unmarshal(body, &value); err != nil {
		sum := sha256.Sum256(body)
		return hex.EncodeToString(sum[:])
	}
	canonical, _ := json.Marshal(value)
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}

func (a *app) handleProcessorWebhook(w http.ResponseWriter, r *http.Request) {
	providerName := r.PathValue("provider")
	ctx, span := a.tracer.Start(r.Context(), "webhook.processor.handle", map[string]string{"provider": providerName})
	defer span.End()
	r = r.WithContext(ctx)
	body, err := readLimitedBody(r, 2<<20)
	if err != nil || providerName == "" || !processorProviderRE.MatchString(strings.ToLower(providerName)) {
		errorJSON(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid processor webhook")
		return
	}
	orgID := organizationIDFromRequest(r)
	if a.cfg.AppEnv == "production" && orgID == "" {
		errorJSON(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "organizationId is required for production processor webhooks")
		return
	}
	var provider processors.PaymentProcessorProvider
	if strings.EqualFold(providerName, "stripe") {
		provider = processors.StripeWebhookProvider{Verifier: processors.StripeWebhookVerifier{Secret: a.cfg.StripeWebhookSecret, AppEnv: a.cfg.AppEnv}}
	} else {
		if a.cfg.AppEnv == "production" {
			errorJSON(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "unsupported processor provider")
			return
		}
		provider = processors.MockProvider{Name: providerName}
	}
	result, err := provider.HandleWebhook(r.Context(), body, r.Header)
	if err != nil {
		if strings.EqualFold(providerName, "stripe") {
			a.incrementMetric(func(m *opsMetrics) { m.StripeWebhookFailuresTotal++ })
		}
		errorJSON(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid processor webhook")
		return
	}
	event := models.WebhookEvent{ID: auth.NewID(), OrganizationID: orgID, Type: result.EventType, Code: result.EventType, ItemID: result.ExternalEventID, Provider: provider.ProviderName(), DedupeKey: processorWebhookDedupeKey(provider.ProviderName(), result.ExternalEventID, body), Status: "received", CreatedAt: time.Now().UTC()}
	created, err := a.saveProcessorWebhookEvent(r.Context(), event, string(body))
	if err != nil {
		errorJSON(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "could not persist processor webhook")
		return
	}
	if created {
		if strings.EqualFold(provider.ProviderName(), "stripe") {
			a.incrementMetric(func(m *opsMetrics) { m.StripeWebhookEventsTotal++ })
		}
		a.writeAudit(r.Context(), r, orgID, "", provider.ProviderName()+".webhook_received", "processor_event", result.ExternalEventID, `{"event_type":"`+result.EventType+`"}`)
		a.emitOutbox(r.Context(), orgID, "processor.webhook_received", "processor_event", result.ExternalEventID, string(body))
		if result.ShouldSync && orgID != "" {
			_, _ = a.enqueueJob(r.Context(), orgID, "", provider.ProviderName()+".sync", string(body))
		}
	}
	writeJSON(w, http.StatusAccepted, map[string]interface{}{"status": "accepted", "deduped": !created})
}

func plaidShouldQueueSync(webhookType, webhookCode string) bool {
	if !strings.EqualFold(webhookType, "TRANSACTIONS") {
		return false
	}
	switch webhookCode {
	case "SYNC_UPDATES_AVAILABLE", "DEFAULT_UPDATE", "HISTORICAL_UPDATE", "TRANSACTIONS_REMOVED", "INITIAL_UPDATE":
		return true
	default:
		return false
	}
}

func processorWebhookDedupeKey(provider, externalID string, body []byte) string {
	if externalID == "" {
		externalID = canonicalPayloadHash(body)
	}
	sum := sha256.Sum256([]byte(provider + "|" + externalID))
	return hex.EncodeToString(sum[:])
}

func readLimitedBody(r *http.Request, max int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > max {
		return nil, errors.New("request body too large")
	}
	return body, nil
}

func (a *app) organizationForPlaidItem(ctx context.Context, itemID string) string {
	if itemID == "" {
		return ""
	}
	if a.cfRepo != nil {
		orgID, err := a.cfRepo.FindOrganizationByPlaidItemID(ctx, itemID)
		if err == nil {
			return orgID
		}
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	if orgID := a.store.plaidItemOrganizations[itemID]; orgID != "" {
		return orgID
	}
	return ""
}

func (a *app) savePlaidWebhookEvent(ctx context.Context, event models.WebhookEvent, payload string) (bool, error) {
	if a.cfRepo != nil {
		_, created, err := a.cfRepo.SavePlaidWebhookEvent(ctx, event, payload)
		return created, err
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	if _, ok := a.store.plaidWebhookEvents[event.DedupeKey]; ok {
		return false, nil
	}
	a.store.plaidWebhookEvents[event.DedupeKey] = event
	return true, nil
}

func (a *app) saveProcessorWebhookEvent(ctx context.Context, event models.WebhookEvent, payload string) (bool, error) {
	if a.cfRepo != nil {
		_, created, err := a.cfRepo.SaveProcessorWebhookEvent(ctx, event, payload)
		return created, err
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	if _, ok := a.store.processorWebhookEvents[event.DedupeKey]; ok {
		return false, nil
	}
	a.store.processorWebhookEvents[event.DedupeKey] = event
	return true, nil
}

func (a *app) emitOutbox(ctx context.Context, orgID, eventType, aggregateType, aggregateID, payload string) {
	if payload == "" {
		payload = "{}"
	}
	event := models.OutboxEvent{ID: auth.NewID(), OrganizationID: orgID, EventType: eventType, AggregateType: aggregateType, AggregateID: aggregateID, PayloadJSON: payload, Status: "pending", MaxAttempts: 3, AvailableAt: time.Now().UTC(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if a.cfRepo != nil {
		_, _ = a.cfRepo.EmitOutboxEvent(ctx, event)
		return
	}
	a.store.mu.Lock()
	a.store.outboxEvents[event.ID] = event
	a.store.mu.Unlock()
}

func (a *app) cancelJobV1(w http.ResponseWriter, r *http.Request) {
	a.operateJob(w, r, "cancel")
}

func (a *app) retryJobV1(w http.ResponseWriter, r *http.Request) {
	a.operateJob(w, r, "retry")
}

func (a *app) operateJob(w http.ResponseWriter, r *http.Request, op string) {
	orgID, ok := requireQueryOrganizationID(w, r)
	if !ok {
		return
	}
	if _, ok := a.requireOrgRole(w, r, orgID, authz.CanManageMembers); !ok {
		return
	}
	var job models.Job
	var err error
	if op == "cancel" {
		job, err = a.cancelJob(r.Context(), orgID, r.PathValue("jobId"))
	} else {
		job, err = a.retryJob(r.Context(), orgID, r.PathValue("jobId"))
	}
	if err == repository.ErrIdempotencyConflict {
		errorJSON(w, r, http.StatusConflict, "CONFLICT", "completed jobs cannot be cancelled")
		return
	}
	if err != nil {
		errorJSON(w, r, http.StatusNotFound, "NOT_FOUND", "job not found")
		return
	}
	a.incrementMetric(func(m *opsMetrics) {
		if op == "cancel" {
			m.JobCancelTotal++
		} else {
			m.JobRetryTotal++
		}
	})
	a.writeAudit(r.Context(), r, orgID, userID(r), "job."+op+"led", "job", job.ID, "{}")
	a.emitOutbox(r.Context(), orgID, "job."+op+"led", "job", job.ID, "{}")
	writeJSON(w, 200, map[string]interface{}{"job": job, "status": job.Status})
}

func (a *app) listDeadJobsV1(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireQueryOrganizationID(w, r)
	if !ok {
		return
	}
	if _, ok := a.requireOrgRole(w, r, orgID, authz.CanManageMembers); !ok {
		return
	}
	query, err := httpapi.ParseListQuery(r)
	if err != nil {
		errorJSON(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	jobs, err := a.deadJobs(r.Context(), orgID, query.Limit, query.Offset)
	if err != nil {
		errorJSON(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list dead jobs")
		return
	}
	writeJSON(w, 200, httpapi.Paginated(jobs, query))
}

func (a *app) cancelJob(ctx context.Context, orgID, jobID string) (models.Job, error) {
	if a.cfRepo != nil {
		return a.cfRepo.CancelJob(ctx, orgID, jobID)
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	job, ok := a.store.jobs[jobID]
	if !ok || job.OrganizationID != orgID {
		return models.Job{}, repository.ErrNotFound
	}
	if job.Status == "completed" {
		return models.Job{}, repository.ErrIdempotencyConflict
	}
	if job.Status == "running" {
		job.Status = "cancel_requested"
	} else {
		job.Status = "cancelled"
	}
	job.UpdatedAt = time.Now().UTC()
	a.store.jobs[jobID] = job
	return job, nil
}

func (a *app) retryJob(ctx context.Context, orgID, jobID string) (models.Job, error) {
	if a.cfRepo != nil {
		return a.cfRepo.RetryJob(ctx, orgID, jobID)
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	job, ok := a.store.jobs[jobID]
	if !ok || job.OrganizationID != orgID {
		return models.Job{}, repository.ErrNotFound
	}
	if job.Status != "failed" && job.Status != "dead" && job.Status != "cancelled" {
		return models.Job{}, repository.ErrNotFound
	}
	job.Status = "queued"
	job.Error = ""
	job.RunAfter = time.Now().UTC()
	job.UpdatedAt = job.RunAfter
	a.store.jobs[jobID] = job
	return job, nil
}

func (a *app) deadJobs(ctx context.Context, orgID string, limit, offset int) ([]models.Job, error) {
	if a.cfRepo != nil {
		return a.cfRepo.ListDeadJobs(ctx, orgID, limit, offset)
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.Job{}
	for _, job := range a.store.jobs {
		if job.OrganizationID == orgID && job.Status == "dead" {
			out = append(out, job)
		}
	}
	return httpapi.Page(out, httpapi.ListQuery{Limit: limit, Offset: offset}), nil
}

func (a *app) opsMetricsV1(w http.ResponseWriter, r *http.Request) {
	a.store.mu.RLock()
	metrics := a.store.metrics
	for _, job := range a.store.jobs {
		applyJobStatusToMetrics(&metrics, job.Status)
	}
	a.store.mu.RUnlock()
	if a.cfRepo != nil {
		counts, err := a.cfRepo.JobStatusCounts(r.Context())
		if err != nil {
			errorJSON(w, r, http.StatusInternalServerError, "DATABASE_ERROR", "could not load job metrics")
			return
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
	writeJSON(w, 200, metrics)
}

func applyJobStatusToMetrics(metrics *opsMetrics, status string) {
	switch status {
	case "queued", "running":
		metrics.JobQueueDepth++
	case "completed":
		metrics.JobsCompletedTotal++
	case "failed":
		metrics.JobsFailedTotal++
	case "dead":
		metrics.JobsDeadTotal++
	}
}

func (a *app) incrementMetric(fn func(*opsMetrics)) {
	a.store.mu.Lock()
	fn(&a.store.metrics)
	a.store.mu.Unlock()
}
