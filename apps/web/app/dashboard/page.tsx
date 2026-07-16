"use client";

import Link from "next/link";
import dynamic from "next/dynamic";
import { Card, Shell, SkeletonBlock, money } from "@/components/layout";
import { Header, Empty } from "@/components/layout";
import { DemoPilot } from "@/components/demo";
import { GuideMarker } from "@/components/help";
import { activeDemoScenario } from "@/lib/api";
import { useApi } from "@/hooks/useApi";

type Payout = { id: string; processor_payout_id: string; amount: number; status: string; expected_arrival_at: string };
type Payment = { id: string; amount: number; status: string; description: string; occurred_at: string };
type BankTransaction = { id: string; amount: number; direction: string; description: string; posted_at: string };
type Exception = { id: string; title: string; explanation: string; status: string; severity: string };
type Connection = { id: string; institution_name: string };

const CashForecastMiniChart = dynamic(() => import("@/components/charts/DashboardCharts").then((mod) => mod.CashForecastMiniChart), {
  ssr: false,
  loading: () => <SkeletonBlock className="h-full" />
});
const PayoutVolumeChart = dynamic(() => import("@/components/charts/DashboardCharts").then((mod) => mod.PayoutVolumeChart), {
  ssr: false,
  loading: () => <SkeletonBlock className="h-full" />
});

export default function Dashboard() {
  const fallback = dashboardFallback();
  const cash = useApi<Record<string, number>>("/cash-flow/summary", fallback.cash, { instant: true });
  const forecast = useApi<Array<Record<string, number>>>("/cash-flow/forecast", fallback.forecast, { instant: true });
  const exceptions = useApi<Exception[]>("/reconciliation/exceptions", fallback.exceptions, { instant: true });
  const payouts = useApi<Payout[]>("/payouts", fallback.payouts, { instant: true });
  const payments = useApi<Payment[]>("/payments", fallback.payments, { instant: true });
  const bank = useApi<BankTransaction[]>("/bank-transactions", fallback.bank, { instant: true });
  const connections = useApi<Connection[]>("/connections", fallback.connections, { instant: true });
  const metrics = useApi<Record<string, number>>("/api/v1/ops/metrics", fallback.metrics, { instant: true });
  const openBreaks = exceptions.data.filter((item) => item.status === "open");
  const completedJobs = metrics.data.jobs_completed_total || 0;
  const metricsLoading = cash.loading || exceptions.loading;
  const nextAction = openBreaks.length
    ? { label: "Review open breaks", href: "/reconciliation", detail: `${openBreaks.length} payout/deposit issue(s) need operator review.` }
    : payouts.data.length === 0 || bank.data.length === 0
      ? { label: "Load settlement data", href: "/reconciliation", detail: "Run processor and bank sync before trusting the dashboard." }
      : { label: "Open control evidence", href: "/ops", detail: "Reconciliation is clear. Verify jobs, audit logs, webhooks, and idempotency." };

  return (
    <Shell>
      <Header title="Today's close" subtitle="One workflow for small-business payment ops: load Stripe payouts, match deposits, resolve breaks, and prove cash is reliable." />

      <section className="mb-5 rounded-md border border-ink/10 bg-[#17211b] p-5 text-white shadow-sm">
        <div className="grid gap-5 xl:grid-cols-[1.1fr_.9fr]">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-white/45">Current operating question</p>
            <h2 className="mt-2 max-w-3xl text-2xl font-semibold">Can we explain every Stripe payout that hit the bank?</h2>
            <p className="mt-2 max-w-3xl text-sm leading-6 text-white/65">
              Clearflow is focused on payout reconciliation: processor payments, refunds, and fees roll into payouts; bank deposits prove settlement; exceptions become an operator queue with audit history.
            </p>
            <div className="mt-4 flex flex-wrap gap-2">
              <StepLink number="1" label="Load data" href="/imports" />
              <StepLink number="2" label="Run reconciliation" href="/reconciliation" />
              <StepLink number="3" label="Resolve breaks" href="/reconciliation" />
              <StepLink number="4" label="Prove controls" href="/ops" />
            </div>
          </div>
          <div className="rounded-md border border-white/10 bg-white/[0.06] p-4">
            <p className="text-xs font-semibold uppercase tracking-wide text-white/45">Next best action</p>
            <p className="mt-2 text-xl font-semibold">{nextAction.label}</p>
            <p className="mt-2 text-sm leading-6 text-white/65">{nextAction.detail}</p>
            <Link href={nextAction.href} className="mt-4 inline-flex rounded-md bg-white px-4 py-2 text-sm font-semibold text-ink">Continue workflow</Link>
          </div>
        </div>
      </section>

      <div className="mb-4">
        <div className="mb-2 flex justify-end"><GuideMarker guide={{ number: 1, title: "Guided demo setup", body: "Start here when testing locally. It prepares onboarding, processor data, bank data, and reconciliation in the correct order." }} /></div>
        <DemoPilot />
      </div>

      <div className="mb-2 flex items-center justify-between"><p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Key operating metrics</p><GuideMarker guide={{ number: 2, title: "Operating metrics", body: "Read these cards left to right: available cash, net operating movement after costs/refunds, processor cost, refunds, then open reconciliation breaks." }} /></div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
        <Kpi label="Operating cash" value={money(cash.data.cash_balance)} detail="posted bank cash" loading={cash.loading} />
        <Kpi label="Net flow" value={money(cash.data.net_cash_flow)} detail="cash credits minus debits, fees, refunds" tone={(cash.data.net_cash_flow || 0) >= 0 ? "good" : "warn"} loading={cash.loading} />
        <Kpi label="Processor cost" value={money(cash.data.fees)} detail="fees this period" tone="warn" loading={cash.loading} />
        <Kpi label="Refunds" value={money(cash.data.refunds)} detail="returned volume" tone="warn" loading={cash.loading} />
        <Kpi label="Open breaks" value={`${openBreaks.length}`} detail="requires review" tone={openBreaks.length ? "warn" : "good"} loading={metricsLoading} />
      </div>

      <div className="mt-4">
        <Card title="Operator checklist" guide={{ number: 3, title: "Operator checklist", body: "Use this as the daily workflow map. Any card marked next tells you which page to open to finish setup or fix active issues." }}>
          <div className="grid gap-3 md:grid-cols-4">
            <ChecklistItem label="Bank connection" done={connections.data.length > 0} detail={connections.data.length ? "Plaid connection available" : "Connect Plaid or create sandbox bank"} href="/imports" loading={connections.loading} />
            <ChecklistItem label="Processor data" done={payouts.data.length > 0 && payments.data.length > 0} detail={payouts.data.length ? "Payouts and payments loaded" : "Run processor sync"} href="/reconciliation" loading={payouts.loading || payments.loading} />
            <ChecklistItem label="Worker jobs" done={completedJobs > 0} detail={completedJobs ? `${completedJobs} completed job(s)` : "Run full reconciliation with worker on"} href="/ops" loading={metrics.loading} />
            <ChecklistItem label="Open breaks" done={openBreaks.length === 0} detail={openBreaks.length ? `${openBreaks.length} break(s) need review` : "No active breaks"} href="/reconciliation" loading={exceptions.loading} />
          </div>
        </Card>
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-[1.15fr_.85fr]">
        <Card title="Cash forecast" guide={{ number: 4, title: "Cash-flow forecast", body: "This chart projects cash over time. The x-axis is days from today and the y-axis is projected cash amount. Use it to spot future cash pressure." }}>
          <div className="mb-3 flex flex-wrap items-center justify-between gap-2 text-xs text-ink/45">
            <span>X-axis: days from today</span>
            <span>Y-axis: projected cash amount</span>
          </div>
          <div className="h-72">
            <CashForecastMiniChart data={forecast.data} />
          </div>
        </Card>

        <Card title="Settlement health" guide={{ number: 5, title: "Settlement health", body: "This summarizes whether enough processor and bank data exists to trust reconciliation. If open exceptions are nonzero, review breaks next." }}>
          <div className="grid gap-3">
            <HealthRow label="Payouts imported" value={payouts.data.length} loading={payouts.loading} />
            <HealthRow label="Bank transactions" value={bank.data.length} loading={bank.loading} />
            <HealthRow label="Processor payments" value={payments.data.length} loading={payments.loading} />
            <HealthRow label="Open exceptions" value={openBreaks.length} tone={openBreaks.length ? "warn" : "good"} loading={exceptions.loading} />
          </div>
          <div className="mt-5 rounded-md bg-ink/[0.03] p-4">
            <p className="text-sm font-medium">Next operator action</p>
            <p className="mt-1 text-sm leading-6 text-ink/55">
              Review unmatched deposits on the Reconciliation page, then resolve or annotate each exception before month-end reporting.
            </p>
          </div>
        </Card>
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-[.9fr_1.1fr]">
        <Card title="Payout volume" guide={{ number: 6, title: "Payout volume", body: "Shows recent processor payouts by amount so you can quickly spot unusually large or small settlement batches." }}>
          <div className="h-72">
            <PayoutVolumeChart data={payouts.data.slice(0, 8)} />
          </div>
        </Card>

        <Card title="Exception queue" guide={{ number: 7, title: "Exception queue", body: "These are reconciliation breaks. Open items need operator review before you trust cash reporting or month-end close." }}>
          {exceptions.loading ? <SkeletonBlock className="h-56" /> : exceptions.data.length ? (
            <div className="grid gap-2">
              {exceptions.data.slice(0, 5).map((item) => (
                <div key={item.id} className="rounded-md border border-ink/10 p-3">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="font-medium">{item.title}</p>
                      <p className="mt-1 text-sm leading-6 text-ink/60">{item.explanation}</p>
                    </div>
                    <span className={`rounded px-2 py-1 text-xs font-medium ${item.status === "open" ? "bg-coral/10 text-coral" : "bg-mint text-moss"}`}>{item.status}</span>
                  </div>
                </div>
              ))}
            </div>
          ) : <Empty text="No exceptions yet." />}
        </Card>
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-2">
        <Ledger title="Recent processor payments" guide={{ number: 8, title: "Processor payments", body: "These are the gross payment records that roll up into Stripe-style payouts, fees, and refunds." }} rows={payments.data.map((row) => ({ id: row.id, label: row.description, detail: row.status, date: row.occurred_at, amount: row.amount }))} />
        <Ledger title="Recent bank activity" guide={{ number: 9, title: "Bank activity", body: "These are posted bank credits and debits. Reconciliation compares processor payouts against these deposits." }} rows={bank.data.map((row) => ({ id: row.id, label: row.description, detail: row.direction, date: row.posted_at, amount: row.direction === "credit" ? row.amount : -row.amount }))} />
      </div>
    </Shell>
  );
}

function StepLink({ number, label, href }: { number: string; label: string; href: string }) {
  return (
    <Link href={href} className="inline-flex items-center gap-2 rounded-md border border-white/10 bg-white/[0.06] px-3 py-2 text-sm font-semibold text-white hover:bg-white/[0.1]">
      <span className="grid h-6 w-6 place-items-center rounded-full bg-[#83c5ff] text-xs font-bold text-ink">{number}</span>
      {label}
    </Link>
  );
}

function ChecklistItem({ label, done, detail, href, loading = false }: { label: string; done: boolean; detail: string; href: string; loading?: boolean }) {
  if (loading) return <SkeletonBlock className="h-28" />;
  return (
    <Link href={href} className={`rounded-md border p-3 transition hover:bg-ink/[0.03] ${done ? "border-moss/25 bg-mint/60" : "border-gold/35 bg-gold/10"}`}>
      <span className={`inline-flex rounded px-2 py-1 text-xs font-semibold ${done ? "bg-white text-moss" : "bg-white text-ink/65"}`}>{done ? "done" : "next"}</span>
      <p className="mt-3 text-sm font-semibold text-ink">{label}</p>
      <p className="mt-1 text-xs leading-5 text-ink/55">{detail}</p>
    </Link>
  );
}

function Kpi({ label, value, detail, tone = "neutral", loading = false }: { label: string; value: string; detail: string; tone?: "neutral" | "good" | "warn"; loading?: boolean }) {
  const color = tone === "good" ? "text-moss" : tone === "warn" ? "text-coral" : "text-ink";
  return (
    <section className="rounded-md border border-ink/10 bg-white px-4 py-3 shadow-sm">
      <p className="text-xs font-medium uppercase tracking-wide text-ink/45">{label}</p>
      {loading ? <SkeletonBlock className="mt-2 h-8 w-28" /> : <p className={`mt-1 text-2xl font-semibold ${color}`}>{value}</p>}
      {loading ? <SkeletonBlock className="mt-2 h-3 w-24" /> : <p className="mt-1 text-xs text-ink/45">{detail}</p>}
    </section>
  );
}

function HealthRow({ label, value, tone = "neutral", loading = false }: { label: string; value: number; tone?: "neutral" | "good" | "warn"; loading?: boolean }) {
  const color = tone === "good" ? "text-moss" : tone === "warn" ? "text-coral" : "text-ink";
  return <div className="flex items-center justify-between border-b border-ink/10 py-2 last:border-0"><span className="text-sm text-ink/60">{label}</span>{loading ? <SkeletonBlock className="h-4 w-8" /> : <span className={`text-sm font-semibold ${color}`}>{value}</span>}</div>;
}

function Ledger({ title, rows, guide }: { title: string; rows: Array<{ id: string; label: string; detail: string; date: string; amount: number }>; guide?: { number: number; title: string; body: string } }) {
  return (
    <Card title={title} guide={guide}>
      {rows.length ? <table className="w-full text-left text-sm"><tbody>{rows.slice(0, 8).map((row) => <tr key={row.id} className="border-b border-ink/10 last:border-0"><td className="py-3"><p className="font-medium">{row.label}</p><p className="text-xs text-ink/45">{row.detail} · {formatDate(row.date)}</p></td><td className={`text-right font-medium ${row.amount < 0 ? "text-coral" : "text-ink"}`}>{money(row.amount)}</td></tr>)}</tbody></table> : <Empty text="No ledger rows yet." />}
    </Card>
  );
}

function formatDate(value?: string) {
  if (!value) return "—";
  return new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric" }).format(new Date(value));
}

function dashboardFallback() {
  const scenario = activeDemoScenario();
  const cash = {
    cash_balance: scenario.cashBalance,
    income: scenario.income,
    expenses: scenario.expenses,
    pending_payouts: 0,
    fees: scenario.fees,
    refunds: scenario.refunds,
    net_cash_flow: scenario.income - scenario.expenses - scenario.fees - scenario.refunds
  };
  return {
    cash,
    forecast: [
      { days: 7, projected_cash: scenario.cashBalance, expected_payouts: 0, expected_expenses: 0 },
      { days: 30, projected_cash: scenario.cashBalance - scenario.expenses * 0.9, expected_payouts: 0, expected_expenses: scenario.expenses * 0.9 },
      { days: 60, projected_cash: scenario.cashBalance - scenario.expenses * 1.7, expected_payouts: 0, expected_expenses: scenario.expenses * 1.7 }
    ],
    exceptions: [
      { id: "fallback-ex-1", title: "Likely payout amount mismatch", explanation: "Processor payout is close to a bank deposit but differs by expected fees or reserves.", status: "open", severity: "medium" }
    ],
    payouts: [
      { id: "fallback-po-1", processor_payout_id: "po_demo_001", amount: Math.max(0, scenario.income - scenario.fees - scenario.refunds), status: "paid", expected_arrival_at: new Date().toISOString() }
    ],
    payments: [
      { id: "fallback-pay-1", amount: Math.round(scenario.income * 0.55), status: "succeeded", description: "Customer payments batch", occurred_at: new Date().toISOString() },
      { id: "fallback-pay-2", amount: Math.round(scenario.income * 0.45), status: "succeeded", description: "Sponsor or subscription payments", occurred_at: new Date().toISOString() }
    ],
    bank: [
      { id: "fallback-bank-1", amount: scenario.cashBalance, direction: "credit", description: "Stripe payout deposit", posted_at: new Date().toISOString() },
      { id: "fallback-bank-2", amount: scenario.expenses, direction: "debit", description: "Operating expenses", posted_at: new Date().toISOString() }
    ],
    connections: [{ id: "fallback-conn-1", institution_name: "Connected bank" }],
    metrics: { jobs_completed_total: 3, job_queue_depth: 0, http_requests_total: 0 }
  };
}
