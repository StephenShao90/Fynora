"use client";

import { Bar, BarChart, CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { Card, Shell, money } from "@/components/Shell";
import { Header, Empty } from "@/components/Common";
import { useApi } from "@/hooks/useApi";

type Payout = { id: string; processor_payout_id: string; amount: number; status: string; expected_arrival_at: string };
type Payment = { id: string; amount: number; status: string; description: string; occurred_at: string };
type BankTransaction = { id: string; amount: number; direction: string; description: string; posted_at: string };
type Exception = { id: string; title: string; explanation: string; status: string; severity: string };

export default function Dashboard() {
  const cash = useApi<Record<string, number>>("/cash-flow/summary", {});
  const forecast = useApi<Array<Record<string, number>>>("/cash-flow/forecast", []);
  const exceptions = useApi<Exception[]>("/reconciliation/exceptions", []);
  const payouts = useApi<Payout[]>("/payouts", []);
  const payments = useApi<Payment[]>("/payments", []);
  const bank = useApi<BankTransaction[]>("/bank-transactions", []);
  const openBreaks = exceptions.data.filter((item) => item.status === "open");

  return (
    <Shell>
      <Header title="Operations dashboard" subtitle="A real-time view of payout settlement, bank cash, open breaks, and forecasted operating runway." />

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
        <Kpi label="Operating cash" value={money(cash.data.cash_balance)} detail="posted bank cash" />
        <Kpi label="Net flow" value={money(cash.data.net_cash_flow)} detail="income minus debits, fees, refunds" tone={(cash.data.net_cash_flow || 0) >= 0 ? "good" : "warn"} />
        <Kpi label="Processor cost" value={money(cash.data.fees)} detail="fees this period" tone="warn" />
        <Kpi label="Refunds" value={money(cash.data.refunds)} detail="returned volume" tone="warn" />
        <Kpi label="Open breaks" value={`${openBreaks.length}`} detail="requires review" tone={openBreaks.length ? "warn" : "good"} />
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-[1.15fr_.85fr]">
        <Card title="Cash forecast">
          <div className="mb-3 flex flex-wrap items-center justify-between gap-2 text-xs text-ink/45">
            <span>X-axis: days from today</span>
            <span>Y-axis: projected cash amount</span>
          </div>
          <div className="h-72">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={forecast.data}>
                <CartesianGrid stroke="#dfe5dc" strokeDasharray="3 3" />
                <XAxis dataKey="days" tickLine={false} axisLine={false} label={{ value: "Days ahead", position: "insideBottom", offset: -4 }} />
                <YAxis tickLine={false} axisLine={false} tickFormatter={(value) => `$${Number(value).toLocaleString()}`} label={{ value: "Projected cash", angle: -90, position: "insideLeft" }} />
                <Tooltip formatter={(value) => money(Number(value))} />
                <Line type="monotone" dataKey="projected_cash" stroke="#17211b" strokeWidth={3} dot={{ r: 4 }} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </Card>

        <Card title="Settlement health">
          <div className="grid gap-3">
            <HealthRow label="Payouts imported" value={payouts.data.length} />
            <HealthRow label="Bank transactions" value={bank.data.length} />
            <HealthRow label="Processor payments" value={payments.data.length} />
            <HealthRow label="Open exceptions" value={openBreaks.length} tone={openBreaks.length ? "warn" : "good"} />
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
        <Card title="Payout volume">
          <div className="h-72">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={payouts.data.slice(0, 8)}>
                <XAxis dataKey="processor_payout_id" tickLine={false} axisLine={false} />
                <YAxis tickLine={false} axisLine={false} />
                <Tooltip formatter={(value) => money(Number(value))} />
                <Bar dataKey="amount" fill="#315846" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </Card>

        <Card title="Exception queue">
          {exceptions.data.length ? (
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
        <Ledger title="Recent processor payments" rows={payments.data.map((row) => ({ id: row.id, label: row.description, detail: row.status, date: row.occurred_at, amount: row.amount }))} />
        <Ledger title="Recent bank activity" rows={bank.data.map((row) => ({ id: row.id, label: row.description, detail: row.direction, date: row.posted_at, amount: row.direction === "credit" ? row.amount : -row.amount }))} />
      </div>
    </Shell>
  );
}

function Kpi({ label, value, detail, tone = "neutral" }: { label: string; value: string; detail: string; tone?: "neutral" | "good" | "warn" }) {
  const color = tone === "good" ? "text-moss" : tone === "warn" ? "text-coral" : "text-ink";
  return (
    <section className="rounded-md border border-ink/10 bg-white px-4 py-3 shadow-sm">
      <p className="text-xs font-medium uppercase tracking-wide text-ink/45">{label}</p>
      <p className={`mt-1 text-2xl font-semibold ${color}`}>{value}</p>
      <p className="mt-1 text-xs text-ink/45">{detail}</p>
    </section>
  );
}

function HealthRow({ label, value, tone = "neutral" }: { label: string; value: number; tone?: "neutral" | "good" | "warn" }) {
  const color = tone === "good" ? "text-moss" : tone === "warn" ? "text-coral" : "text-ink";
  return <div className="flex items-center justify-between border-b border-ink/10 py-2 last:border-0"><span className="text-sm text-ink/60">{label}</span><span className={`text-sm font-semibold ${color}`}>{value}</span></div>;
}

function Ledger({ title, rows }: { title: string; rows: Array<{ id: string; label: string; detail: string; date: string; amount: number }> }) {
  return (
    <Card title={title}>
      {rows.length ? <table className="w-full text-left text-sm"><tbody>{rows.slice(0, 8).map((row) => <tr key={row.id} className="border-b border-ink/10 last:border-0"><td className="py-3"><p className="font-medium">{row.label}</p><p className="text-xs text-ink/45">{row.detail} · {formatDate(row.date)}</p></td><td className={`text-right font-medium ${row.amount < 0 ? "text-coral" : "text-ink"}`}>{money(row.amount)}</td></tr>)}</tbody></table> : <Empty text="No ledger rows yet." />}
    </Card>
  );
}

function formatDate(value?: string) {
  if (!value) return "—";
  return new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric" }).format(new Date(value));
}
