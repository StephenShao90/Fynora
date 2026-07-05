"use client";

import { Bar, BarChart, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { Card, Metric, Shell, money } from "@/components/Shell";
import { Empty, Header } from "@/components/Common";
import { useApi } from "@/hooks/useApi";
import type { NamedAmount, RiskFinding } from "@/types";

export default function Dashboard() {
  const cash = useApi<Record<string, number>>("/cash-flow/summary", {});
  const forecast = useApi<Array<Record<string, number>>>("/cash-flow/forecast", []);
  const exceptions = useApi<RiskFinding[]>("/reconciliation/exceptions", []);
  const payouts = useApi<NamedAmount[]>("/payouts", []);
  return (
    <Shell>
      <Header title="Dashboard" subtitle="Payment operations, reconciliation health, and cash visibility in one view." />
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Metric label="Cash balance" value={money(cash.data.cash_balance)} />
        <Metric label="Income" value={money(cash.data.income)} tone="good" />
        <Metric label="Fees + refunds" value={money((cash.data.fees || 0) + (cash.data.refunds || 0))} tone="warn" />
        <Metric label="Open exceptions" value={`${exceptions.data.length || 0}`} tone={exceptions.data.length ? "warn" : "good"} />
      </div>
      <div className="mt-5 grid gap-5 xl:grid-cols-[1.2fr_.8fr]">
        <Card title="Cash forecast">
          <div className="h-72">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={forecast.data}><XAxis dataKey="days" /><YAxis /><Tooltip /><Line type="monotone" dataKey="projected_cash" stroke="#315846" strokeWidth={3} /></LineChart>
            </ResponsiveContainer>
          </div>
        </Card>
        <Card title="Reconciliation exceptions">
          <div className="grid gap-3">
            {exceptions.data.length ? exceptions.data.slice(0, 4).map((r) => <div key={r.title} className="rounded-md bg-coral/10 p-3"><p className="font-medium">{r.title}</p><p className="text-sm text-ink/65">{r.explanation}</p></div>) : <Empty text="No reconciliation exceptions yet. Sync sample data and run reconciliation." />}
          </div>
        </Card>
        <Card title="Payouts">
          <div className="h-72">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={payouts.data.slice(0, 8)}><XAxis dataKey="processor_payout_id" /><YAxis /><Tooltip /><Bar dataKey="amount" fill="#d6a53a" radius={[4, 4, 0, 0]} /></BarChart>
            </ResponsiveContainer>
          </div>
        </Card>
      </div>
    </Shell>
  );
}
