"use client";

import Link from "next/link";
import dynamic from "next/dynamic";
import { Card, Shell, SkeletonBlock, money } from "@/components/layout";
import { Header, Empty } from "@/components/layout";
import { DemoPilot } from "@/components/demo";
import { GuideMarker } from "@/components/help";
import { dashboardFallback, normalizeDashboardSummary, type DashboardSummary } from "@/components/dashboard/data";
import { useApi } from "@/hooks/useApi";
import { activeDemoScenario } from "@/lib/api";

const CashForecastMiniChart = dynamic(() => import("@/components/charts/DashboardCharts").then((mod) => mod.CashForecastMiniChart), {
  ssr: false,
  loading: () => <SkeletonBlock className="h-full" />
});
const PayoutVolumeChart = dynamic(() => import("@/components/charts/DashboardCharts").then((mod) => mod.PayoutVolumeChart), {
  ssr: false,
  loading: () => <SkeletonBlock className="h-full" />
});

export default function Dashboard() {
  const scenario = activeDemoScenario();
  const fallback = dashboardFallback();
  const summary = useApi<DashboardSummary>("/api/v1/dashboard/summary", fallback, { instant: true });
  const data = normalizeDashboardSummary(summary.data, fallback);
  const { cash, forecast, exceptions, payouts, payments, bank_transactions: bank, connections, metrics } = data;
  const openBreaks = exceptions.filter((item) => item.status === "open");
  const completedJobs = metrics.jobs_completed_total || 0;
  const expectedDeposits = payouts.reduce((sum, payout) => sum + (payout.amount || 0), 0);
  const postedCredits = bank.filter((row) => row.direction === "credit").reduce((sum, row) => sum + (row.amount || 0), 0);
  const depositGap = expectedDeposits - postedCredits;
  const restaurantMode = scenario.type === "restaurant";
  const nextAction = openBreaks.length
    ? { label: "Review open breaks", href: "/reconciliation", detail: `${openBreaks.length} payout/deposit issue(s) need operator review.` }
    : payouts.length === 0 || bank.length === 0
      ? { label: "Connect settlement data", href: "/imports", detail: "Load processor payouts and bank deposits before trusting the dashboard." }
      : { label: "Review cash forecast", href: "/insights", detail: "Payouts are explained. Check whether the next 30 days of cash still look healthy." };

  return (
    <Shell>
      <Header title={restaurantMode ? "Daily revenue close" : "Home base"} subtitle={restaurantMode ? "Verify POS sales, delivery payouts, refunds, fees, cash deposits, and bank settlement before trusting today's cash." : "A daily close view for small teams: explain payouts, fix breaks, and know what cash is safe to use."} />

      <section className="mb-5 rounded-md border border-ink/10 bg-[#17211b] p-5 text-white shadow-sm">
        <div className="grid gap-5 xl:grid-cols-[1.1fr_.9fr]">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-white/45">Customer question</p>
            <h2 className="mt-2 max-w-3xl text-2xl font-semibold">{restaurantMode ? "Did yesterday's sales actually become bank cash?" : "Can we explain every Stripe payout that hit the bank?"}</h2>
            <p className="mt-2 max-w-3xl text-sm leading-6 text-white/65">
              {restaurantMode
                ? "Clearflow compares POS batches, delivery settlements, processor fees, refunds, and bank deposits so an owner can close the day with evidence."
                : "Clearflow turns processor payouts and bank deposits into a simple close checklist: connected data, matched payouts, resolved breaks, and a cash forecast you can act on."}
            </p>
            <div className="mt-4 flex flex-wrap gap-2">
              <StepLink number="1" label={restaurantMode ? "Load POS + bank data" : "Load data"} href="/imports" />
              <StepLink number="2" label="Run reconciliation" href="/reconciliation" />
              <StepLink number="3" label={restaurantMode ? "Explain exceptions" : "Resolve breaks"} href="/reconciliation" />
              <StepLink number="4" label="Forecast cash" href="/insights" />
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
        <Kpi label="Operating cash" value={money(cash.cash_balance)} detail="posted bank cash" />
        <Kpi label="Net flow" value={money(cash.net_cash_flow)} detail="cash credits minus debits, fees, refunds" tone={(cash.net_cash_flow || 0) >= 0 ? "good" : "warn"} />
        <Kpi label="Processor cost" value={money(cash.fees)} detail="fees this period" tone="warn" />
        <Kpi label="Refunds" value={money(cash.refunds)} detail="returned volume" tone="warn" />
        <Kpi label="Open breaks" value={`${openBreaks.length}`} detail="requires review" tone={openBreaks.length ? "warn" : "good"} />
      </div>

      {restaurantMode ? (
        <div className="mt-4">
          <Card title="Restaurant close verdict" guide={{ number: 3, title: "Restaurant close verdict", body: "This is the owner-facing answer: expected payouts versus posted deposits, open exceptions, and whether the day can be marked complete." }}>
            <div className="grid gap-4 xl:grid-cols-[.9fr_1.1fr]">
              <div className={`rounded-md border p-4 ${openBreaks.length ? "border-gold/40 bg-gold/10" : "border-moss/25 bg-mint/60"}`}>
                <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Close status</p>
                <p className="mt-2 text-2xl font-semibold">{openBreaks.length ? "Review required" : "Ready to close"}</p>
                <p className="mt-2 text-sm leading-6 text-ink/60">
                  {openBreaks.length
                    ? `${openBreaks.length} exception(s) need evidence before yesterday's revenue can be trusted.`
                    : "Expected payouts and bank deposits are explained for this close window."}
                </p>
              </div>
              <div className="grid gap-3 md:grid-cols-3">
                <MiniMetric label="Expected deposits" value={money(expectedDeposits)} detail="processor + delivery payouts" />
                <MiniMetric label="Posted credits" value={money(postedCredits)} detail="bank deposits found" />
                <MiniMetric label="Unexplained gap" value={money(depositGap)} detail="positive means expected cash is missing" tone={Math.abs(depositGap) > 1 ? "warn" : "good"} />
              </div>
            </div>
          </Card>
        </div>
      ) : null}

      <div className="mt-4">
        <Card title={restaurantMode ? "Daily close checklist" : "Operator checklist"} guide={{ number: restaurantMode ? 4 : 3, title: "Operator checklist", body: "Use this as the daily workflow map. Any card marked next tells you which page to open to finish setup or fix active issues." }}>
          <div className="grid gap-3 md:grid-cols-4">
            <ChecklistItem label="Bank connection" done={connections.length > 0} detail={connections.length ? "Plaid connection available" : "Connect Plaid or create sandbox bank"} href="/imports" />
            <ChecklistItem label={restaurantMode ? "POS + delivery data" : "Processor data"} done={payouts.length > 0 && payments.length > 0} detail={payouts.length ? "Payouts and payments loaded" : "Run processor sync"} href="/reconciliation" />
            <ChecklistItem label="Close run completed" done={completedJobs > 0} detail={completedJobs ? `${completedJobs} completed background step(s)` : "Run the close checklist"} href="/reconciliation" />
            <ChecklistItem label="Open breaks" done={openBreaks.length === 0} detail={openBreaks.length ? `${openBreaks.length} break(s) need review` : "No active breaks"} href="/reconciliation" />
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
            <CashForecastMiniChart data={forecast} />
          </div>
        </Card>

        <Card title="Settlement health" guide={{ number: 5, title: "Settlement health", body: "This summarizes whether enough processor and bank data exists to trust reconciliation. If open exceptions are nonzero, review breaks next." }}>
          <div className="grid gap-3">
            <HealthRow label="Payouts imported" value={payouts.length} />
            <HealthRow label="Bank transactions" value={bank.length} />
            <HealthRow label="Processor payments" value={payments.length} />
            <HealthRow label="Open exceptions" value={openBreaks.length} tone={openBreaks.length ? "warn" : "good"} />
          </div>
          <div className="mt-5 rounded-md bg-ink/[0.03] p-4">
            <p className="text-sm font-medium">Next operator action</p>
            <p className="mt-1 text-sm leading-6 text-ink/55">
              {restaurantMode ? "Review missing delivery payouts, short deposits, and cash deposits on the Reconciliation page before marking the daily close complete." : "Review unmatched deposits on the Reconciliation page, then resolve or annotate each exception before month-end reporting."}
            </p>
          </div>
        </Card>
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-[.9fr_1.1fr]">
        <Card title="Payout volume" guide={{ number: 6, title: "Payout volume", body: "Shows recent processor payouts by amount so you can quickly spot unusually large or small settlement batches." }}>
          <div className="h-72">
            <PayoutVolumeChart data={payouts.slice(0, 8)} />
          </div>
        </Card>

        <Card title="Exception queue" guide={{ number: 7, title: "Exception queue", body: "These are reconciliation breaks. Open items need operator review before you trust cash reporting or month-end close." }}>
          {exceptions.length ? (
            <div className="grid gap-2">
              {exceptions.slice(0, 5).map((item) => (
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
        <Ledger title="Recent processor payments" guide={{ number: 8, title: "Processor payments", body: "These are the gross payment records that roll up into Stripe-style payouts, fees, and refunds." }} rows={payments.map((row) => ({ id: row.id, label: row.description, detail: row.status, date: row.occurred_at, amount: row.amount }))} />
        <Ledger title="Recent bank activity" guide={{ number: 9, title: "Bank activity", body: "These are posted bank credits and debits. Reconciliation compares processor payouts against these deposits." }} rows={bank.map((row) => ({ id: row.id, label: row.description, detail: row.direction, date: row.posted_at, amount: row.direction === "credit" ? row.amount : -row.amount }))} />
      </div>
    </Shell>
  );
}

function MiniMetric({ label, value, detail, tone = "neutral" }: { label: string; value: string; detail: string; tone?: "neutral" | "good" | "warn" }) {
  const color = tone === "good" ? "text-moss" : tone === "warn" ? "text-coral" : "text-ink";
  return (
    <div className="rounded-md border border-ink/10 bg-white p-4">
      <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">{label}</p>
      <p className={`mt-2 text-xl font-semibold ${color}`}>{value}</p>
      <p className="mt-1 text-xs leading-5 text-ink/50">{detail}</p>
    </div>
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
