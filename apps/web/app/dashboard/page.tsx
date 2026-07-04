"use client";

import { Bar, BarChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { Card, Metric, Shell, money } from "@/components/Shell";
import { Empty, Header } from "@/components/Common";
import { useApi } from "@/hooks/useApi";
import type { CashFlow, NamedAmount, Summary, RiskFinding } from "@/types";

export default function Dashboard() {
  const cash = useApi<CashFlow>("/insights/cash-flow", {} as CashFlow);
  const categories = useApi<NamedAmount[]>("/insights/categories", []);
  const portfolio = useApi<Summary>("/portfolio/summary", {} as Summary);
  const risks = useApi<RiskFinding[]>("/portfolio/risk", []);
  return (
    <Shell>
      <Header title="Dashboard" subtitle="Cash flow, spending leaks, and portfolio health in one view." />
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Metric label="Monthly income" value={money(cash.data.average_monthly_income)} />
        <Metric label="Monthly expenses" value={money((cash.data.average_fixed_expenses || 0) + (cash.data.average_variable_expenses || 0))} />
        <Metric label="Net cash flow" value={money(cash.data.average_net_cash_flow)} tone="good" />
        <Metric label="Portfolio value" value={money(portfolio.data.total_market_value)} />
      </div>
      <div className="mt-5 grid gap-5 xl:grid-cols-[1.2fr_.8fr]">
        <Card title="Top categories">
          <div className="h-72">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={categories.data.slice(0, 8)}><XAxis dataKey="name" /><YAxis /><Tooltip /><Bar dataKey="amount" fill="#315846" radius={[4, 4, 0, 0]} /></BarChart>
            </ResponsiveContainer>
          </div>
        </Card>
        <Card title="Portfolio warnings">
          <div className="grid gap-3">
            {risks.data.length ? risks.data.map((r) => <div key={r.title} className="rounded-md bg-coral/10 p-3"><p className="font-medium">{r.title}</p><p className="text-sm text-ink/65">{r.explanation}</p></div>) : <Empty text="No major concentration warnings detected." />}
          </div>
        </Card>
      </div>
    </Shell>
  );
}
