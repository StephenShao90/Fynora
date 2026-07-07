"use client";

import { useEffect, useMemo, useState } from "react";
import { Card, Shell, money } from "@/components/Shell";
import { Header, Empty } from "@/components/Common";
import { PayoutExplanationPanel } from "@/components/payouts/PayoutExplanationPanel";
import { ReconciliationMatches } from "@/components/reconciliation/ReconciliationMatches";
import { api, getPayoutExplanation, getReconciliationMatches, type PayoutExplanation, type ReconciliationMatch } from "@/lib/api";
import { useApi } from "@/hooks/useApi";

type Run = { id: string; status: string; matched_count: number; exception_count: number; started_at: string };
type Exception = { id: string; severity: string; title: string; explanation: string; status: string; created_at: string };
type Payment = { id: string; processor_payment_id: string; amount: number; status: string; description: string; occurred_at: string; customer_email?: string };
type Payout = { id: string; processor_payout_id: string; amount: number; status: string; expected_arrival_at: string };
type Activity = { id: string; label: string; detail: string; status: "ok" | "error" | "running"; at: string };

export default function ReconciliationPage() {
  const [activity, setActivity] = useState<Activity[]>([]);
  const [reloadKey, setReloadKey] = useState(0);
  const [selectedPayoutId, setSelectedPayoutId] = useState("");
  const [explanation, setExplanation] = useState<{ data?: PayoutExplanation; loading: boolean; error: string }>({ loading: false, error: "" });
  const [matches, setMatches] = useState<{ data: ReconciliationMatch[]; loading: boolean; error: string }>({ data: [], loading: false, error: "" });
  const runs = useApi<Run[]>(`/reconciliation/runs?reload=${reloadKey}`, []);
  const exceptions = useApi<Exception[]>(`/reconciliation/exceptions?reload=${reloadKey}`, []);
  const payments = useApi<Payment[]>(`/payments?reload=${reloadKey}`, []);
  const payouts = useApi<Payout[]>(`/payouts?reload=${reloadKey}`, []);
  const cash = useApi<Record<string, number>>(`/cash-flow/summary?reload=${reloadKey}`, {});

  const openExceptions = useMemo(() => exceptions.data.filter((item) => item.status === "open"), [exceptions.data]);
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

  async function action(label: string, path: string) {
    const id = crypto.randomUUID();
    setActivity((items) => [{ id, label, detail: "Running", status: "running" as const, at: new Date().toISOString() }, ...items].slice(0, 8));
    try {
      const result = await api<Record<string, unknown>>(path, { method: "POST", body: "{}" });
      setActivity((items) => items.map((item) => item.id === id ? { ...item, status: "ok" as const, detail: summarizeResult(path, result), at: new Date().toISOString() } : item));
      setReloadKey((value) => value + 1);
    } catch (err) {
      setActivity((items) => items.map((item) => item.id === id ? { ...item, status: "error" as const, detail: (err as Error).message, at: new Date().toISOString() } : item));
    }
  }

  async function resolveException(id: string) {
    await api(`/reconciliation/exceptions/${id}`, { method: "PATCH", body: JSON.stringify({ status: "resolved" }) });
    setActivity((items) => [{ id: crypto.randomUUID(), label: "Resolved exception", detail: id.slice(0, 8), status: "ok" as const, at: new Date().toISOString() }, ...items].slice(0, 8));
    setReloadKey((value) => value + 1);
  }

  return (
    <Shell>
      <Header title="Reconciliation" subtitle="Processor payouts, bank deposits, exceptions, and operator workflow." />

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
        <Kpi label="Cash" value={money(cash.data.cash_balance)} sub="posted bank balance" />
        <Kpi label="Match rate" value={`${matchRate}%`} sub={latestRun ? `${latestRun.matched_count} matched / ${latestRun.exception_count} breaks` : "no run yet"} tone={matchRate >= 80 ? "good" : "warn"} />
        <Kpi label="Open breaks" value={`${openExceptions.length}`} sub="needs review" tone={openExceptions.length ? "warn" : "good"} />
        <Kpi label="Fees" value={money(cash.data.fees)} sub="processor cost" tone="warn" />
        <Kpi label="Refunds" value={money(cash.data.refunds)} sub="customer returns" tone="warn" />
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-[360px_1fr]">
        <Card title="Runbook">
          <div className="grid gap-2">
            <ActionButton label="1. Sync processor" detail="Load Stripe-style charges, fees, refunds, payout" onClick={() => action("Processor sync", "/sync/stripe")} />
            <ActionButton label="2. Sync bank" detail="Load bank deposits and operating debits" onClick={() => action("Bank sync", "/sync/bank")} />
            <ActionButton label="3. Reconcile" detail="Match payouts to bank deposits" onClick={() => action("Reconciliation run", "/reconciliation/runs")} primary />
          </div>
          <div className="mt-5 border-t border-ink/10 pt-4">
            <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Activity</p>
            <div className="mt-3 grid gap-2">
              {activity.length ? activity.map((item) => <ActivityRow key={item.id} item={item} />) : <p className="text-sm text-ink/50">Actions and API outcomes will appear here. Browser console also logs request IDs.</p>}
            </div>
          </div>
        </Card>

        <Card title="Latest reconciliation runs">
          {runs.data.length ? (
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
        <Card title="Exception queue">
          {exceptions.data.length ? (
            <div className="grid gap-2">
              {exceptions.data.slice(0, 8).map((item) => (
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
                    {item.status === "open" ? <button onClick={() => resolveException(item.id)} className="shrink-0 rounded-md border border-ink/15 px-3 py-1.5 text-xs font-medium hover:bg-white">Resolve</button> : null}
                  </div>
                </div>
              ))}
            </div>
          ) : <Empty text="No exceptions yet." />}
        </Card>

        <Card title="Payout ledger">
          {payouts.data.length ? (
            <div className="overflow-x-auto">
              <table className="min-w-full text-left text-sm">
                <thead className="border-b border-ink/10 text-xs uppercase tracking-wide text-ink/45"><tr><th className="py-2 pr-4">Payout</th><th className="pr-4">Arrival</th><th className="pr-4">Status</th><th className="pr-4 text-right">Net</th><th className="text-right">Explain</th></tr></thead>
                <tbody>{payouts.data.slice(0, 8).map((payout) => <tr key={payout.id} className="border-b border-ink/10 last:border-0"><td className="py-3 pr-4 font-mono text-xs">{payout.processor_payout_id}</td><td className="pr-4">{formatDate(payout.expected_arrival_at)}</td><td className="pr-4"><StatusChip label={payout.status} tone="good" /></td><td className="pr-4 text-right font-medium">{money(payout.amount)}</td><td className="text-right"><button onClick={() => setSelectedPayoutId(payout.id)} className={`rounded-md border px-2.5 py-1.5 text-xs font-medium ${selectedPayoutId === payout.id ? "border-moss bg-mint text-moss" : "border-ink/15 hover:bg-ink/[0.03]"}`}>View explanation</button></td></tr>)}</tbody>
              </table>
            </div>
          ) : <Empty text="No payouts yet." />}
        </Card>
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-[.95fr_1.05fr]">
        <PayoutExplanationPanel explanation={explanation.data} loading={explanation.loading} error={explanation.error} />
        <ReconciliationMatches matches={matches.data} loading={matches.loading} error={matches.error} />
      </div>

      <div className="mt-4">
        <Card title="Recent processor payments">
          {payments.data.length ? (
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

function ActionButton({ label, detail, onClick, primary = false }: { label: string; detail: string; onClick: () => void; primary?: boolean }) {
  return (
    <button onClick={onClick} className={`rounded-md border px-3 py-3 text-left transition ${primary ? "border-gold/50 bg-gold/20 hover:bg-gold/30" : "border-ink/10 bg-white hover:bg-ink/[0.03]"}`}>
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

function summarizeResult(path: string, result: Record<string, unknown>) {
  if (path.includes("stripe")) return `${result.payments || 0} payments, ${result.fees || 0} fees, ${result.refunds || 0} refunds`;
  if (path.includes("bank")) return `${result.bank_transactions || 0} bank transactions loaded`;
  if (path.includes("reconciliation")) return `${result.matched_count || 0} matches, ${result.exception_count || 0} breaks`;
  return "Completed";
}

function formatDate(value?: string) {
  if (!value) return "—";
  return new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }).format(new Date(value));
}
