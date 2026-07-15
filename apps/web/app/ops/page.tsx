"use client";

import { useEffect, useMemo, useState } from "react";
import { Empty, Header } from "@/components/layout";
import { GuideMarker } from "@/components/help";
import { Card, Shell } from "@/components/layout";
import { api } from "@/lib/api";
import { useApi } from "@/hooks/useApi";

type Organization = { id: string; name: string };
type Job = { id: string; type: string; status: string; attempts: number; max_attempts: number; created_at: string; updated_at: string };
type AuditLog = { id: string; action: string; target_type: string; target_id: string; created_at: string };
type Paginated<T> = { data: T[] };
type Metrics = Record<string, number>;
type LoadState<T> = { data: T; loading: boolean; error: string };

export default function OpsPage() {
  const orgs = useApi<Organization[]>("/organizations", []);
  const orgID = orgs.data[0]?.id || "";
  const [jobs, setJobs] = useState<LoadState<Job[]>>({ data: [], loading: true, error: "" });
  const [audit, setAudit] = useState<LoadState<AuditLog[]>>({ data: [], loading: true, error: "" });
  const [metrics, setMetrics] = useState<LoadState<Metrics>>({ data: {}, loading: true, error: "" });

  useEffect(() => {
    if (!orgID) return;
    let cancelled = false;
    async function load() {
      setJobs((current) => ({ ...current, loading: true, error: "" }));
      setAudit((current) => ({ ...current, loading: true, error: "" }));
      setMetrics((current) => ({ ...current, loading: true, error: "" }));

      api<Paginated<Job>>(`/api/v1/jobs?organizationId=${orgID}`)
        .then((result) => !cancelled && setJobs({ data: result.data || [], loading: false, error: "" }))
        .catch((err) => !cancelled && setJobs({ data: [], loading: false, error: (err as Error).message }));

      api<Paginated<AuditLog>>(`/api/v1/audit-logs?organizationId=${orgID}`)
        .then((result) => !cancelled && setAudit({ data: result.data || [], loading: false, error: "" }))
        .catch((err) => !cancelled && setAudit({ data: [], loading: false, error: (err as Error).message }));

      api<Metrics>("/api/v1/ops/metrics")
        .then((result) => !cancelled && setMetrics({ data: result, loading: false, error: "" }))
        .catch((err) => !cancelled && setMetrics({ data: {}, loading: false, error: (err as Error).message }));
    }
    load();
    const interval = window.setInterval(load, 10000);

    return () => {
      cancelled = true;
      window.clearInterval(interval);
    };
  }, [orgID]);

  const metricCards = useMemo(() => [
    ["HTTP requests", metrics.data.http_requests_total],
    ["Jobs queued", metrics.data.jobs_queued_total],
    ["Jobs completed", metrics.data.jobs_completed_total],
    ["Queue depth", metrics.data.job_queue_depth],
    ["Webhook events", (metrics.data.stripe_webhook_events_total || 0) + (metrics.data.plaid_webhooks_received_total || 0)],
    ["Idempotency replays", metrics.data.idempotency_replays_total]
  ], [metrics.data]);
  const controls = useMemo(() => {
    const queueDepth = Number(metrics.data.job_queue_depth || 0);
    const completedJobs = Number(metrics.data.jobs_completed_total || 0);
    const failedJobs = jobs.data.filter((job) => ["failed", "dead", "cancelled"].includes(job.status)).length;
    const auditCount = audit.data.length;
    const idempotencyReplays = Number(metrics.data.idempotency_replays_total || 0);
    const webhookEvents = Number(metrics.data.stripe_webhook_events_total || 0) + Number(metrics.data.plaid_webhooks_received_total || 0);
    return [
      {
        name: "Worker health",
        status: queueDepth === 0 && failedJobs === 0 ? "pass" : queueDepth > 0 ? "watch" : "fail",
        evidence: queueDepth === 0 ? "Queue depth is 0" : `${queueDepth} job(s) still queued`,
        detail: "Shows async sync/reconciliation work is not stuck."
      },
      {
        name: "Financial write safety",
        status: idempotencyReplays > 0 ? "pass" : "watch",
        evidence: `${idempotencyReplays.toLocaleString()} idempotency replay(s) observed`,
        detail: "Protects retries from duplicating money movement records."
      },
      {
        name: "Auditability",
        status: auditCount > 0 ? "pass" : "watch",
        evidence: `${auditCount.toLocaleString()} recent audit event(s) loaded`,
        detail: "Supports investigation, compliance review, and debugging."
      },
      {
        name: "Provider event handling",
        status: webhookEvents > 0 ? "pass" : "watch",
        evidence: `${webhookEvents.toLocaleString()} webhook event(s) observed`,
        detail: "Verifies provider callbacks are visible to operations."
      },
      {
        name: "Job durability",
        status: completedJobs > 0 && failedJobs === 0 ? "pass" : failedJobs > 0 ? "fail" : "watch",
        evidence: `${completedJobs.toLocaleString()} completed, ${failedJobs.toLocaleString()} failed/dead`,
        detail: "Demonstrates background jobs finish and expose failures."
      }
    ];
  }, [audit.data.length, jobs.data, metrics.data]);
  const controlScore = Math.round((controls.filter((control) => control.status === "pass").length / controls.length) * 100);

  return (
    <Shell>
      <Header title="Operations" subtitle="Async jobs, audit trail, metrics, and debugging surfaces for financial operations." />

      <div className="mb-2 flex items-center justify-between"><p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Worker status</p><GuideMarker guide={{ number: 1, title: "Worker status", body: "Check this first after running sync or reconciliation. Queue depth should eventually return to zero when the worker is healthy." }} /></div>
      <div className={`mb-4 rounded-md border px-4 py-3 text-sm ${Number(metrics.data.job_queue_depth || 0) > 0 ? "border-gold/40 bg-gold/15 text-ink" : "border-moss/25 bg-mint/60 text-moss"}`}>
        {Number(metrics.data.job_queue_depth || 0) > 0
          ? `Worker queue has ${Number(metrics.data.job_queue_depth).toLocaleString()} pending job(s). Keep make worker running until this returns to 0.`
          : "Worker queue is clear. If you just ran Reconciliation, completed jobs should appear below after the next refresh."}
      </div>

      <div className="mb-2 flex items-center justify-between"><p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Operational metrics</p><GuideMarker guide={{ number: 2, title: "Operational metrics", body: "Use these counters to prove API traffic, background jobs, webhooks, and idempotency replays are being recorded." }} /></div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-6">
        {metricCards.map(([label, value]) => (
          <section key={label} className="rounded-md border border-ink/10 bg-white px-4 py-3 shadow-sm">
            <p className="text-xs font-medium uppercase tracking-wide text-ink/45">{label}</p>
            <p className="mt-1 text-2xl font-semibold text-ink">{Number(value || 0).toLocaleString()}</p>
          </section>
        ))}
      </div>

      <div className="mt-4">
        <Card title="Bank-grade control evidence" guide={{ number: 3, title: "Control evidence", body: "This converts raw logs and metrics into an interview-ready controls view: worker health, idempotency, auditability, provider events, and job durability." }}>
          <div className="grid gap-4 xl:grid-cols-[240px_1fr]">
            <div className="rounded-md border border-ink/10 bg-[#f7faf6] p-4">
              <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Readiness score</p>
              <p className={`mt-2 text-5xl font-semibold ${controlScore >= 80 ? "text-moss" : controlScore >= 60 ? "text-ink" : "text-coral"}`}>{controlScore}%</p>
              <p className="mt-2 text-sm leading-6 text-ink/55">Based on live operational evidence loaded from jobs, audit logs, metrics, webhooks, and idempotency counters.</p>
            </div>
            <div className="grid gap-2">
              {controls.map((control) => (
                <div key={control.name} className="grid gap-3 rounded-md border border-ink/10 p-3 md:grid-cols-[180px_120px_1fr]">
                  <p className="text-sm font-semibold text-ink">{control.name}</p>
                  <ControlStatus value={control.status} />
                  <div>
                    <p className="text-sm text-ink/70">{control.evidence}</p>
                    <p className="mt-1 text-xs leading-5 text-ink/45">{control.detail}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </Card>
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-[1fr_1fr]">
        <Card title="Recent jobs" guide={{ number: 4, title: "Recent jobs", body: "Shows queued, running, completed, or failed async work. Sync and reconciliation should appear here after using the workflow buttons." }}>
          {jobs.loading ? <Skeleton /> : jobs.error ? <Empty text={`Could not load jobs: ${jobs.error}`} /> : jobs.data.length ? (
            <table className="w-full text-left text-sm">
              <thead className="border-b border-ink/10 text-xs uppercase tracking-wide text-ink/45">
                <tr><th className="py-2 pr-3">Job</th><th className="pr-3">Type</th><th className="pr-3">Status</th><th className="text-right">Attempts</th></tr>
              </thead>
              <tbody>
                {jobs.data.slice(0, 8).map((job) => (
                  <tr key={job.id} className="border-b border-ink/10 last:border-0">
                    <td className="py-3 pr-3 font-mono text-xs text-ink/60">{job.id.slice(0, 12)}</td>
                    <td className="pr-3">{job.type}</td>
                    <td className="pr-3"><Status value={job.status} /></td>
                    <td className="text-right">{job.attempts}/{job.max_attempts}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : <Empty text="No jobs found for this organization yet. Run the Reconciliation workflow with make worker running, then this table will fill with sync and reconciliation jobs." />}
        </Card>

        <Card title="Audit trail" guide={{ number: 5, title: "Audit trail", body: "Audit logs explain who did what and when. Use these entries with request IDs/logs to debug financial workflow issues." }}>
          {audit.loading ? <Skeleton /> : audit.error ? <Empty text={`Could not load audit logs: ${audit.error}`} /> : audit.data.length ? (
            <div className="grid gap-2">
              {audit.data.slice(0, 8).map((entry) => (
                <div key={entry.id} className="rounded-md border border-ink/10 p-3">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="font-medium">{entry.action}</p>
                      <p className="mt-1 font-mono text-xs text-ink/45">{entry.target_type}:{entry.target_id}</p>
                    </div>
                    <span className="text-xs text-ink/45">{formatDate(entry.created_at)}</span>
                  </div>
                </div>
              ))}
            </div>
          ) : <Empty text="No audit logs found." />}
        </Card>
      </div>

      <div className="mt-4">
        <Card title="Operational readiness" guide={{ number: 6, title: "Operational readiness", body: "This summarizes production backend concepts recruiters care about: idempotency, async workers, Redis locks/rate limits, and tracing." }}>
          <div className="grid gap-3 text-sm leading-6 text-ink/60 md:grid-cols-3">
            <p>Financial writes use idempotency keys so retries do not duplicate payouts, deposits, or reconciliation jobs.</p>
            <p>The worker processes async sync and reconciliation jobs separately from API request handling.</p>
            <p>Redis and OpenTelemetry are optional locally but ready for production rate limiting, locks, and trace export.</p>
          </div>
        </Card>
      </div>
    </Shell>
  );
}

function Status({ value }: { value: string }) {
  const good = value === "completed";
  return <span className={`rounded px-2 py-1 text-xs font-medium ${good ? "bg-mint text-moss" : "bg-ink/5 text-ink/55"}`}>{value}</span>;
}

function ControlStatus({ value }: { value: string }) {
  const classes = value === "pass" ? "bg-mint text-moss" : value === "fail" ? "bg-coral/10 text-coral" : "bg-gold/20 text-ink";
  const label = value === "pass" ? "passing" : value === "fail" ? "failing" : "watch";
  return <span className={`w-fit rounded px-2 py-1 text-xs font-semibold uppercase tracking-wide ${classes}`}>{label}</span>;
}

function Skeleton() {
  return <div className="h-48 animate-pulse rounded-md bg-ink/[0.04]" />;
}

function formatDate(value?: string) {
  if (!value) return "Not available";
  return new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }).format(new Date(value));
}
