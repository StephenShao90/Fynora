package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/config"
	"github.com/StephenShao90/Fynora/services/api/internal/db"
	"github.com/StephenShao90/Fynora/services/api/internal/logger"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
	"github.com/StephenShao90/Fynora/services/api/internal/observability"
	"github.com/StephenShao90/Fynora/services/api/internal/repository"
)

func main() {
	cfg := config.Load()
	if err := cfg.ValidateProduction(); err != nil {
		log.Fatal(err)
	}
	l := logger.New()
	tracer, err := observability.NewWithConfig(context.Background(), observability.Config{
		Enabled:     observability.Enabled(cfg.OTELEnabled),
		ServiceName: cfg.OTELServiceName + "-worker",
		Environment: cfg.OTELEnvironment,
		Endpoint:    cfg.OTELExporterOTLPEndpoint,
		Protocol:    cfg.OTELExporterOTLPProtocol,
		Headers:     cfg.OTELExporterOTLPHeaders,
		SampleRatio: observability.SampleRatio(cfg.OTELSampleRatio),
	})
	if err != nil {
		if cfg.AppEnv == "production" {
			log.Fatalf("otel exporter setup failed: %v", err)
		}
		l.Error("otel.setup_failed", map[string]interface{}{"error": err.Error(), "fallback": "disabled"})
		tracer = observability.New(false, cfg.OTELServiceName+"-worker", cfg.OTELEnvironment)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracer.Shutdown(ctx); err != nil {
			l.Error("otel.shutdown_failed", map[string]interface{}{"error": err.Error()})
		}
	}()
	conn, err := db.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	repo := repository.NewClearflow(conn)
	workerID := cfg.WorkerID
	if workerID == "" {
		host, _ := os.Hostname()
		workerID = host
	}
	poll := time.Duration(cfg.WorkerPollMS) * time.Millisecond
	if poll <= 0 {
		poll = 10 * time.Second
	}
	l.Info("worker.started", map[string]interface{}{"queue": "sync_jobs", "worker_id": workerID, "poll_ms": poll.Milliseconds()})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	for {
		if err := claimAndRunJob(ctx, repo, l, tracer, workerID); err != nil {
			l.Error("worker.tick_failed", map[string]interface{}{"error": err.Error()})
		}
		select {
		case <-ctx.Done():
			l.Info("worker.stopped", map[string]interface{}{"worker_id": workerID})
			return
		case <-ticker.C:
		}
	}
}

func claimAndRunJob(ctx context.Context, repo *repository.ClearflowRepository, l logger.Logger, tracer observability.Tracer, workerID string) error {
	job, ok, err := repo.ClaimJob(ctx, workerID)
	if err != nil || !ok {
		return err
	}
	ctx = tracer.Extract(ctx, traceHeadersFromPayload(job.PayloadJSON))
	ctx, span := tracer.Start(ctx, "job.process", map[string]string{"job.id": job.ID, "job.type": job.Type, "organization.id": job.OrganizationID})
	defer span.End()
	logFields := map[string]interface{}{"job_id": job.ID, "organization_id": job.OrganizationID, "type": job.Type, "attempts": job.Attempts, "worker": workerID, "trace_id": observability.TraceID(ctx), "span_id": observability.SpanID(ctx)}
	l.Info("worker.job.started", logFields)
	if err := runJob(ctx, repo, job); err != nil {
		_ = repo.FailJob(ctx, job, err.Error())
		_ = repo.WriteAudit(ctx, models.AuditLog{ID: auth.NewID(), OrganizationID: job.OrganizationID, UserID: job.UserID, Action: "job.failed", TargetType: "job", TargetID: job.ID, Metadata: `{"type":"` + job.Type + `"}`, CreatedAt: time.Now().UTC()})
		_, _ = repo.EmitOutboxEvent(ctx, models.OutboxEvent{OrganizationID: job.OrganizationID, EventType: "job.failed", AggregateType: "job", AggregateID: job.ID, PayloadJSON: `{"type":"` + job.Type + `"}`})
		logFields["error"] = err.Error()
		l.Error("worker.job.failed", logFields)
		return err
	}
	if err := repo.CompleteJob(ctx, job.ID); err != nil {
		return err
	}
	_ = repo.WriteAudit(ctx, models.AuditLog{ID: auth.NewID(), OrganizationID: job.OrganizationID, UserID: job.UserID, Action: "job.completed", TargetType: "job", TargetID: job.ID, Metadata: `{"type":"` + job.Type + `"}`, CreatedAt: time.Now().UTC()})
	_, _ = repo.EmitOutboxEvent(ctx, models.OutboxEvent{OrganizationID: job.OrganizationID, EventType: "job.completed", AggregateType: "job", AggregateID: job.ID, PayloadJSON: `{"type":"` + job.Type + `"}`})
	l.Info("worker.job.completed", logFields)
	return nil
}

func traceHeadersFromPayload(payload string) map[string]string {
	var object map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &object); err != nil {
		return nil
	}
	raw, ok := object["_trace"].(map[string]interface{})
	if !ok {
		return nil
	}
	headers := map[string]string{}
	for key, value := range raw {
		if text, ok := value.(string); ok {
			headers[key] = text
		}
	}
	return headers
}

func runJob(ctx context.Context, repo *repository.ClearflowRepository, job models.Job) error {
	user := models.User{ID: job.UserID, Email: job.UserID + "@worker.local", CreatedAt: time.Now().UTC()}
	org := models.Organization{ID: job.OrganizationID, UserID: job.UserID, Name: "Worker Organization", Type: "small_business", Currency: "USD"}
	switch job.Type {
	case "stripe.sync":
		_, err := repo.SyncStripeDemo(ctx, org, job.UserID)
		return err
	case "bank.sync":
		_, err := repo.SyncBankDemo(ctx, org, job.UserID)
		return err
	case "reconciliation.run":
		_, err := repo.Reconcile(ctx, job.OrganizationID, job.UserID)
		return err
	case "plaid.transactions.sync":
		return fmt.Errorf("plaid worker sync requires configured Plaid client and is deferred")
	default:
		_ = user
		return fmt.Errorf("unsupported job type %s", job.Type)
	}
}
