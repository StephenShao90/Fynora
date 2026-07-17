"use client";

import { useEffect, useMemo, useState } from "react";
import { Card, Shell, SkeletonBlock, money } from "@/components/layout";
import { Header, Empty } from "@/components/layout";
import { GuideMarker } from "@/components/help";
import { PayoutExplanationPanel } from "@/components/payouts/PayoutExplanationPanel";
import { ReconciliationMatches } from "@/components/reconciliation/ReconciliationMatches";
import { useToast } from "@/components/layout";
import { api, getPayoutExplanation, getReconciliationMatches, type PayoutExplanation, type ReconciliationMatch } from "@/lib/api";
import { useApi } from "@/hooks/useApi";
import type { ExceptionNote } from "@/types";

type Run = { id: string; status: string; matched_count: number; exception_count: number; started_at: string };
type Exception = { id: string; severity: string; title: string; explanation: string; status: string; created_at: string };
type Payment = { id: string; processor_payment_id: string; amount: number; status: string; description: string; occurred_at: string; customer_email?: string };
type Payout = { id: string; processor_payout_id: string; amount: number; status: string; expected_arrival_at: string };
type BankTransaction = { id: string; external_id?: string; amount: number; direction: string; description: string; posted_at: string };
type Activity = { id: string; label: string; detail: string; status: "ok" | "error" | "running"; at: string };
type Organization = { id: string; name: string };
type Job = { id: string; type: string; status: string };

export default function ReconciliationPage() {
  const { pushToast } = useToast();
  const [activity, setActivity] = useState<Activity[]>([]);
  const [workflowRunning, setWorkflowRunning] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);
  const [selectedPayoutId, setSelectedPayoutId] = useState("");
  const [resolvedIds, setResolvedIds] = useState<string[]>([]);
  const [selectedException, setSelectedException] = useState<Exception | undefined>();
  const [resolutionNote, setResolutionNote] = useState("");
  const [manualMatchId, setManualMatchId] = useState("");
  const [notes, setNotes] = useState<ExceptionNote[]>([]);
  const [noteDraft, setNoteDraft] = useState("");
  const [explanation, setExplanation] = useState<{ data?: PayoutExplanation; loading: boolean; error: string }>({ loading: false, error: "" });
  const [matches, setMatches] = useState<{ data: ReconciliationMatch[]; loading: boolean; error: string }>({ data: [], loading: false, error: "" });
  const runs = useApi<Run[]>(`/reconciliation/runs?reload=${reloadKey}`, []);
  const exceptions = useApi<Exception[]>(`/reconciliation/exceptions?reload=${reloadKey}`, []);
  const payments = useApi<Payment[]>(`/payments?reload=${reloadKey}`, []);
  const payouts = useApi<Payout[]>(`/payouts?reload=${reloadKey}`, []);
  const bank = useApi<BankTransaction[]>(`/bank-transactions?reload=${reloadKey}`, []);
  const cash = useApi<Record<string, number>>(`/cash-flow/summary?reload=${reloadKey}`, {});
  const organizations = useApi<Organization[]>("/organizations", []);
  const orgId = organizations.data[0]?.id || "";

  const openExceptions = useMemo(() => exceptions.data.filter((item) => item.status === "open" && !resolvedIds.includes(item.id)), [exceptions.data, resolvedIds]);
  const latestRun = runs.data[0];
  const matchRate = latestRun ? Math.round((latestRun.matched_count / Math.max(1, latestRun.matched_count + latestRun.exception_count)) * 100) : 0;

  useEffect(() => {
    if (!selectedPayoutId && payouts.data[0]?.id) setSelectedPayoutId(payouts.data[0].id);
  }, [payouts.data, selectedPayoutId]);

  useEffect(() => {
    if (!selectedPayoutId) return;
    let cancelled = false;
    setExplanation({ loading: true, error: "" });
    getPayoutExplanation(selectedPayoutId)
      .then((data) => !cancelled && setExplanation({ data, loading: false, error: "" }))
      .catch((err) => !cancelled && setExplanation({ loading: false, error: (err as Error).message }));
    return () => { cancelled = true; };
  }, [selectedPayoutId, reloadKey]);

  useEffect(() => {
    const runId = latestRun?.id || "latest";
    let cancelled = false;
    setMatches((current) => ({ ...current, loading: true, error: "" }));
    getReconciliationMatches(runId)
      .then((data) => !cancelled && setMatches({ data, loading: false, error: "" }))
      .catch((err) => !cancelled && setMatches({ data: [], loading: false, error: (err as Error).message }));
    return () => { cancelled = true; };
  }, [latestRun?.id, reloadKey]);

  useEffect(() => {
    if (!selectedException?.id) {
      setNotes([]);
      return;
    }
    let cancelled = false;
    api<ExceptionNote[]>(`/reconciliation/exceptions/${selectedException.id}/notes`)
      .then((rows) => !cancelled && setNotes(rows))
      .catch(() => !cancelled && setNotes([]));
    return () => { cancelled = true; };
  }, [selectedException?.id, reloadKey]);

  async function action(label: string, jobPath: string) {
    if (!orgId) {
      setActivity((items) => [{ id: crypto.randomUUID(), label, detail: "Organization is still loading", status: "error" as const, at: new Date().toISOString() }, ...items].slice(0, 8));
      pushToast({ tone: "error", title: `${label} could not start`, detail: "Organization is still loading." });
      return false;
    }
    const id = crypto.randomUUID();
    setActivity((items) => [{ id, label, detail: "Queued for worker", status: "running" as const, at: new Date().toISOString() }, ...items].slice(0, 8));
    try {
      const result = await api<{ jobId?: string; status?: string }>(`${jobPath}?organizationId=${encodeURIComponent(orgId)}`, {
        method: "POST",
        headers: { "Idempotency-Key": `${label.toLowerCase().replace(/[^a-z0-9]+/g, "-")}-${Date.now()}` },
        body: "{}"
      });
      if (result.jobId) {
        await waitForJob(result.jobId, id, label, orgId);
      } else {
        setActivity((items) => items.map((item) => item.id === id ? { ...item, status: "ok" as const, detail: "Completed", at: new Date().toISOString() } : item));
      }
      setReloadKey((value) => value + 1);
      pushToast({ tone: "success", title: `${label} completed`, detail: "The page data has refreshed." });
      return true;
    } catch (err) {
      setActivity((items) => items.map((item) => item.id === id ? { ...item, status: "error" as const, detail: (err as Error).message, at: new Date().toISOString() } : item));
      pushToast({ tone: "error", title: `${label} failed`, detail: (err as Error).message });
      return false;
    }
  }

  async function runFullWorkflow() {
    if (workflowRunning) return;
    setWorkflowRunning(true);
    pushToast({ tone: "info", title: "Full reconciliation started", detail: "Processor sync, bank sync, and reconciliation will run in order." });
    try {
      const steps: Array<[string, string]> = [
        ["Processor sync", "/api/v1/sync/stripe"],
        ["Bank sync", "/api/v1/sync/bank"],
        ["Reconciliation run", "/api/v1/reconciliation-runs"]
      ];
      for (const [label, path] of steps) {
        const ok = await action(label, path);
        if (!ok) return;
      }
      pushToast({ tone: "success", title: "Full reconciliation complete", detail: "Processor data, bank data, matches, breaks, jobs, and audit logs are updated." });
    } finally {
      setWorkflowRunning(false);
    }
  }

  async function waitForJob(jobId: string, activityId: string, label: string, organizationId: string) {
    const deadline = Date.now() + 45000;
    while (Date.now() < deadline) {
      const job = await api<Job>(`/api/v1/jobs/${jobId}?organizationId=${encodeURIComponent(organizationId)}`);
      setActivity((items) => items.map((item) => item.id === activityId ? { ...item, detail: `${job.type} ${job.status}`, at: new Date().toISOString() } : item));
      if (job.status === "completed") {
        setActivity((items) => items.map((item) => item.id === activityId ? { ...item, status: "ok" as const, detail: `${label} completed`, at: new Date().toISOString() } : item));
        return;
      }
      if (["failed", "dead", "cancelled"].includes(job.status)) {
        throw new Error(`${label} job ${job.status}`);
      }
      await new Promise((resolve) => setTimeout(resolve, 2000));
    }
    throw new Error(`${label} job did not complete. Make sure make worker is running.`);
  }

  async function resolveException(item: Exception) {
    try {
      await api(`/reconciliation/exceptions/${item.id}`, { method: "PATCH", body: JSON.stringify({ status: "resolved", note: resolutionNote, matched_bank_transaction_id: manualMatchId }) });
      setResolvedIds((ids) => Array.from(new Set([...ids, item.id])));
      setActivity((items) => [{ id: crypto.randomUUID(), label: "Resolved exception", detail: `${item.id.slice(0, 8)} · ${resolutionNote || "No note"}`, status: "ok" as const, at: new Date().toISOString() }, ...items].slice(0, 8));
      setSelectedException(undefined);
      setResolutionNote("");
      setManualMatchId("");
      setReloadKey((value) => value + 1);
      pushToast({ tone: "success", title: "Break resolved", detail: "The active exception queue and open-break count will refresh." });
    } catch (err) {
      pushToast({ tone: "error", title: "Could not resolve break", detail: (err as Error).message });
    }
  }

  async function addNote(item: Exception) {
    if (!noteDraft.trim()) return;
    try {
      const note = await api<ExceptionNote>(`/reconciliation/exceptions/${item.id}/notes`, {
        method: "POST",
        body: JSON.stringify({ body: noteDraft.trim() })
      });
      setNotes((rows) => [note, ...rows]);
      setNoteDraft("");
      pushToast({ tone: "success", title: "Note added", detail: "Exception history was updated." });
    } catch (err) {
      pushToast({ tone: "error", title: "Could not add note", detail: (err as Error).message });
    }
  }

  return (
    <Shell>
      <Header title="Reconcile payouts" subtitle="Close the loop between processor payouts and bank deposits, then resolve anything that does not explain cleanly." />

      <section className="mb-5 grid gap-3 rounded-md border border-ink/10 bg-white p-4 shadow-sm md:grid-cols-4">
        <WorkflowStep number="1" title="Load processor payouts" body="Pull payments, fees, refunds, and net payouts." />
        <WorkflowStep number="2" title="Load bank deposits" body="Use Plaid or CSV bank activity as settlement proof." />
        <WorkflowStep number="3" title="Match payouts" body="Compare amount, date, and memo evidence." />
        <WorkflowStep number="4" title="Close breaks" body="Explain or resolve anything unmatched." />
      </section>

      <div className="mb-2 flex items-center justify-between"><p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Reconciliation summary</p><GuideMarker guide={{ number: 1, title: "Summary metrics", body: "Start here to understand whether cash and payouts are reconciled. Match rate and open breaks tell you if the workflow needs review." }} /></div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
        <Kpi label="Cash" value={money(cash.data.cash_balance)} sub="posted bank balance" />
        <Kpi label="Match rate" value={`${matchRate}%`} sub={latestRun ? `${latestRun.matched_count} matched / ${latestRun.exception_count} breaks` : "no run yet"} tone={matchRate >= 80 ? "good" : "warn"} />
        <Kpi label="Open breaks" value={`${openExceptions.length}`} sub="needs review" tone={openExceptions.length ? "warn" : "good"} />
        <Kpi label="Fees" value={money(cash.data.fees)} sub="processor cost" tone="warn" />
        <Kpi label="Refunds" value={money(cash.data.refunds)} sub="customer returns" tone="warn" />
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-[360px_1fr]">
        <Card title="Close checklist" guide={{ number: 2, title: "Close checklist", body: "Use Run full reconciliation for the normal path. It queues processor sync, bank sync, then matching through the worker." }}>
          <button onClick={runFullWorkflow} disabled={workflowRunning || !orgId} className="mb-3 w-full rounded-md bg-ink px-3 py-3 text-left text-sm font-semibold text-white hover:bg-ink/90 disabled:cursor-not-allowed disabled:bg-ink/40">
            Run close checklist
            <span className="mt-1 block text-xs font-normal text-white/70">Loads data, matches payouts, and refreshes breaks.</span>
          </button>
          <div className="grid gap-2">
            <ActionButton label="1. Sync processor" detail="Queue Stripe-style charges, fees, refunds, payout" onClick={() => action("Processor sync", "/api/v1/sync/stripe")} disabled={workflowRunning} />
            <ActionButton label="2. Sync bank" detail="Queue bank deposits and operating debits" onClick={() => action("Bank sync", "/api/v1/sync/bank")} disabled={workflowRunning} />
            <ActionButton label="3. Reconcile" detail="Queue payout-to-deposit matching" onClick={() => action("Reconciliation run", "/api/v1/reconciliation-runs")} primary disabled={workflowRunning} />
          </div>
          <div className="mt-5 border-t border-ink/10 pt-4">
            <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Activity</p>
            <div className="mt-3 grid gap-2">
              {activity.length ? activity.map((item) => <ActivityRow key={item.id} item={item} />) : <p className="text-sm text-ink/50">Run the checklist to see each step complete here.</p>}
            </div>
          </div>
        </Card>

        <Card title="Close history" guide={{ number: 3, title: "Close history", body: "Each run records matching results. Use this table to prove reconciliation executed and to compare matches versus breaks over time." }}>
          {runs.loading ? <SkeletonBlock className="h-64" /> : runs.data.length ? (
            <div className="overflow-x-auto">
              <table className="min-w-full text-left text-sm">
                <thead className="border-b border-ink/10 text-xs uppercase tracking-wide text-ink/45">
                  <tr><th className="py-2 pr-4">Run</th><th className="pr-4">Started</th><th className="pr-4">Status</th><th className="pr-4 text-right">Matches</th><th className="text-right">Breaks</th></tr>
                </thead>
                <tbody>
                  {runs.data.slice(0, 7).map((run) => (
                    <tr key={run.id} className="border-b border-ink/10 last:border-0">
                      <td className="py-3 pr-4 font-mono text-xs text-ink/65">{run.id.slice(0, 8)}</td>
                      <td className="pr-4">{formatDate(run.started_at)}</td>
                      <td className="pr-4"><StatusChip label={run.status} tone="good" /></td>
                      <td className="pr-4 text-right font-medium">{run.matched_count}</td>
                      <td className="text-right font-medium">{run.exception_count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : <Empty text="No reconciliation runs yet." />}
        </Card>
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-[1.05fr_.95fr]">
        <Card title="Exception queue" guide={{ number: 4, title: "Exception queue", body: "Review these breaks one by one. Click Review to add notes, choose a matching bank record, or resolve the issue." }}>
          {exceptions.loading ? <SkeletonBlock className="h-64" /> : openExceptions.length ? (
            <div className="grid gap-2">
              {openExceptions.slice(0, 8).map((item) => (
                <div key={item.id} className={`rounded-md border p-3 ${item.status === "open" ? "border-coral/30 bg-coral/5" : "border-ink/10 bg-ink/[0.02]"}`}>
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <div className="flex flex-wrap items-center gap-2">
                        <p className="font-medium">{item.title}</p>
                        <StatusChip label={item.status} tone={item.status === "open" ? "warn" : "neutral"} />
                        <span className="text-xs uppercase text-ink/40">{item.severity}</span>
                      </div>
                      <p className="mt-1 text-sm leading-6 text-ink/65">{item.explanation}</p>
                    </div>
                    {item.status === "open" ? <button onClick={() => { setSelectedException(item); setResolutionNote(""); setManualMatchId(""); setNoteDraft(""); }} className="shrink-0 rounded-md border border-ink/15 px-3 py-1.5 text-xs font-medium hover:bg-white">Review</button> : null}
                  </div>
                </div>
              ))}
            </div>
          ) : <Empty text="No open exceptions." />}
        </Card>

        <Card title="Payout ledger" guide={{ number: 5, title: "Payout ledger", body: "Processor payouts are the deposits you expect to see in the bank. Click View explanation to inspect gross payments, fees, refunds, and net amount." }}>
          {payouts.loading ? <SkeletonBlock className="h-64" /> : payouts.data.length ? (
            <div className="overflow-x-auto">
              <table className="min-w-full text-left text-sm">
                <thead className="border-b border-ink/10 text-xs uppercase tracking-wide text-ink/45"><tr><th className="py-2 pr-4">Payout</th><th className="pr-4">Arrival</th><th className="pr-4">Status</th><th className="pr-4 text-right">Net</th><th className="text-right">Explain</th></tr></thead>
                <tbody>{payouts.data.slice(0, 8).map((payout) => <tr key={payout.id} className="border-b border-ink/10 last:border-0"><td className="py-3 pr-4 font-mono text-xs">{payout.processor_payout_id}</td><td className="pr-4">{formatDate(payout.expected_arrival_at)}</td><td className="pr-4"><StatusChip label={payout.status} tone="good" /></td><td className="pr-4 text-right font-medium">{money(payout.amount)}</td><td className="text-right"><button onClick={() => setSelectedPayoutId(payout.id)} className={`rounded-md border px-2.5 py-1.5 text-xs font-medium ${selectedPayoutId === payout.id ? "border-moss bg-mint text-moss" : "border-ink/15 hover:bg-ink/[0.03]"}`}>View explanation</button></td></tr>)}</tbody>
              </table>
            </div>
          ) : <Empty text="No payouts yet." />}
        </Card>
      </div>

      {selectedException ? (
        <div className="mt-4">
          <Card title="Exception workbench" guide={{ number: 6, title: "Exception workbench", body: "This is the operator action area. Add an investigation note, optionally associate a bank record, then resolve the break when it is explained." }}>
            <div className="grid gap-4 xl:grid-cols-[1fr_1fr]">
              <div className="rounded-md border border-coral/25 bg-coral/5 p-4">
                <p className="text-sm font-semibold text-coral">{selectedException.title}</p>
                <p className="mt-2 text-sm leading-6 text-ink/65">{selectedException.explanation}</p>
                <p className="mt-3 text-xs uppercase tracking-wide text-ink/45">{selectedException.severity} · {formatDate(selectedException.created_at)}</p>
              </div>
              <div className="grid gap-3">
                <label className="grid gap-1 text-sm font-medium text-ink">
                  Optional matching bank record
                  <select value={manualMatchId} onChange={(event) => setManualMatchId(event.target.value)} className="rounded-md border border-ink/15 px-3 py-2 font-normal">
                    <option value="">No manual match</option>
                    {bank.data.map((row) => <option key={row.id} value={row.id}>{row.description} · {money(row.amount)} · {formatDate(row.posted_at)}</option>)}
                  </select>
                </label>
                <label className="grid gap-1 text-sm font-medium text-ink">
                  Resolution note
                  <textarea value={resolutionNote} onChange={(event) => setResolutionNote(event.target.value)} rows={3} placeholder="Example: verified with bank memo, amount gap is processor reserve release." className="rounded-md border border-ink/15 px-3 py-2 font-normal" />
                </label>
                <div className="flex flex-wrap gap-2">
                  <button onClick={() => resolveException(selectedException)} className="rounded-md bg-ink px-4 py-2 text-sm font-semibold text-white">Resolve break</button>
                  <button onClick={() => setSelectedException(undefined)} className="rounded-md border border-ink/15 px-4 py-2 text-sm font-semibold text-ink">Cancel</button>
                </div>
                <div className="rounded-md border border-ink/10 p-3">
                  <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Note history</p>
                  <div className="mt-3 grid gap-2">
                    {notes.length ? notes.map((note) => (
                      <div key={note.id} className="rounded-md bg-ink/[0.03] p-2 text-sm">
                        <p className="text-ink/75">{note.body}</p>
                        <p className="mt-1 text-xs text-ink/40">{formatDate(note.created_at)}</p>
                      </div>
                    )) : <p className="text-sm text-ink/50">No notes yet.</p>}
                  </div>
                  <div className="mt-3 grid gap-2 md:grid-cols-[1fr_auto]">
                    <input value={noteDraft} onChange={(event) => setNoteDraft(event.target.value)} placeholder="Add investigation note without resolving" className="rounded-md border border-ink/15 px-3 py-2 text-sm" />
                    <button onClick={() => addNote(selectedException)} className="rounded-md border border-ink/15 px-3 py-2 text-sm font-semibold">Add note</button>
                  </div>
                </div>
              </div>
            </div>
          </Card>
        </div>
      ) : null}

      <div className="mt-4 grid gap-4 xl:grid-cols-[.95fr_1.05fr]">
        <div className="relative"><div className="absolute right-4 top-4 z-10"><GuideMarker guide={{ number: 7, title: "Payout explanation", body: "Use this to explain how gross processor activity becomes the net bank deposit, including fees, refunds, and warnings." }} /></div><PayoutExplanationPanel explanation={explanation.data} loading={explanation.loading} error={explanation.error} /></div>
        <div className="relative"><div className="absolute right-4 top-4 z-10"><GuideMarker guide={{ number: 8, title: "Match scoring", body: "This explains why Clearflow thinks a payout and bank deposit match, including confidence score and amount/date differences." }} /></div><ReconciliationMatches matches={matches.data} loading={matches.loading} error={matches.error} /></div>
      </div>

      <div className="mt-4">
        <Card title="Recent processor payments" guide={{ number: 9, title: "Processor payments", body: "These payment rows are the source activity behind fees, refunds, payout items, and payout explanations." }}>
          {payments.loading ? <SkeletonBlock className="h-72" /> : payments.data.length ? (
            <div className="overflow-x-auto">
              <table className="min-w-full text-left text-sm">
                <thead className="border-b border-ink/10 text-xs uppercase tracking-wide text-ink/45"><tr><th className="py-2 pr-4">Payment</th><th className="pr-4">Customer</th><th className="pr-4">Date</th><th className="pr-4">Status</th><th className="text-right">Gross</th></tr></thead>
                <tbody>{payments.data.slice(0, 10).map((payment) => <tr key={payment.id} className="border-b border-ink/10 last:border-0"><td className="py-3 pr-4"><p className="font-medium">{payment.description}</p><p className="font-mono text-xs text-ink/40">{payment.processor_payment_id}</p></td><td className="pr-4 text-ink/65">{payment.customer_email || "Unknown"}</td><td className="pr-4">{formatDate(payment.occurred_at)}</td><td className="pr-4"><StatusChip label={payment.status} tone="good" /></td><td className="text-right font-medium">{money(payment.amount)}</td></tr>)}</tbody>
              </table>
            </div>
          ) : <Empty text="No payments yet." />}
        </Card>
      </div>
    </Shell>
  );
}

function Kpi({ label, value, sub, tone = "neutral" }: { label: string; value: string; sub: string; tone?: "neutral" | "good" | "warn" }) {
  const valueColor = tone === "good" ? "text-moss" : tone === "warn" ? "text-coral" : "text-ink";
  return (
    <section className="rounded-md border border-ink/10 bg-white px-4 py-3 shadow-sm">
      <p className="text-xs font-medium uppercase tracking-wide text-ink/45">{label}</p>
      <p className={`mt-1 text-2xl font-semibold ${valueColor}`}>{value}</p>
      <p className="mt-1 text-xs text-ink/45">{sub}</p>
    </section>
  );
}

function WorkflowStep({ number, title, body }: { number: string; title: string; body: string }) {
  return (
    <div className="flex gap-3 rounded-md bg-ink/[0.025] p-3">
      <span className="grid h-7 w-7 shrink-0 place-items-center rounded-full bg-[#83c5ff] text-xs font-bold text-ink">{number}</span>
      <div>
        <p className="text-sm font-semibold text-ink">{title}</p>
        <p className="mt-1 text-xs leading-5 text-ink/55">{body}</p>
      </div>
    </div>
  );
}

function ActionButton({ label, detail, onClick, primary = false, disabled = false }: { label: string; detail: string; onClick: () => void; primary?: boolean; disabled?: boolean }) {
  return (
    <button onClick={onClick} disabled={disabled} className={`rounded-md border px-3 py-3 text-left transition disabled:cursor-not-allowed disabled:opacity-50 ${primary ? "border-gold/50 bg-gold/20 hover:bg-gold/30" : "border-ink/10 bg-white hover:bg-ink/[0.03]"}`}>
      <span className="block text-sm font-semibold text-ink">{label}</span>
      <span className="mt-1 block text-xs leading-5 text-ink/50">{detail}</span>
    </button>
  );
}

function ActivityRow({ item }: { item: Activity }) {
  return (
    <div className="rounded-md bg-ink/[0.03] px-3 py-2 text-sm">
      <div className="flex items-center justify-between gap-2">
        <p className="font-medium">{item.label}</p>
        <StatusChip label={item.status} tone={item.status === "ok" ? "good" : item.status === "error" ? "warn" : "neutral"} />
      </div>
      <p className="mt-1 text-xs text-ink/50">{item.detail}</p>
    </div>
  );
}

function StatusChip({ label, tone }: { label: string; tone: "good" | "warn" | "neutral" }) {
  const classes = tone === "good" ? "bg-mint text-moss" : tone === "warn" ? "bg-coral/10 text-coral" : "bg-ink/5 text-ink/55";
  return <span className={`inline-flex rounded px-2 py-0.5 text-xs font-medium ${classes}`}>{label}</span>;
}

function formatDate(value?: string) {
  if (!value) return "—";
  return new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }).format(new Date(value));
}
