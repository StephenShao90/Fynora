"use client";

import { Bar, BarChart, Pie, PieChart, ResponsiveContainer, Tooltip, XAxis, YAxis, Cell } from "recharts";
import { Card, Metric, Shell, money } from "@/components/Shell";
import { Empty, Header } from "@/components/Common";
import { useApi } from "@/hooks/useApi";
import type { Holding, NamedAmount, RiskFinding, Summary } from "@/types";

export default function PortfolioPage() {
  const summary = useApi<Summary>("/portfolio/summary", {} as Summary);
  const allocation = useApi<{ by_security_type: NamedAmount[]; by_symbol: NamedAmount[] }>("/portfolio/allocation", { by_security_type: [], by_symbol: [] });
  const holdings = useApi<Holding[]>("/portfolio/holdings", []);
  const risk = useApi<RiskFinding[]>("/portfolio/risk", []);
  return (
    <Shell>
      <Header title="Portfolio" subtitle="Track holdings, allocation, performance, and concentration risk without brokerage credentials." />
      <div className="grid gap-4 md:grid-cols-3">
        <Metric label="Market value" value={money(summary.data.total_market_value)} />
        <Metric label="Unrealized gain/loss" value={money(summary.data.unrealized_gain_loss)} tone={(summary.data.unrealized_gain_loss || 0) >= 0 ? "good" : "warn"} />
        <Metric label="Cash" value={money(summary.data.cash_value)} />
      </div>
      <div className="mt-5 grid gap-5 xl:grid-cols-2">
        <Card title="Allocation by security type"><div className="h-72"><ResponsiveContainer><PieChart><Pie data={allocation.data.by_security_type} dataKey="value" nameKey="name">{allocation.data.by_security_type.map((_, i) => <Cell key={i} fill={["#315846", "#f07b63", "#d6a53a", "#5a8bb0"][i % 4]} />)}</Pie><Tooltip /></PieChart></ResponsiveContainer></div></Card>
        <Card title="Top holdings"><div className="h-72"><ResponsiveContainer><BarChart data={allocation.data.by_symbol?.slice(0, 8)}><XAxis dataKey="name" /><YAxis /><Tooltip /><Bar dataKey="value" fill="#315846" /></BarChart></ResponsiveContainer></div></Card>
      </div>
      <div className="mt-5 grid gap-5 xl:grid-cols-[1.4fr_.6fr]">
        <Card title="Holdings">{holdings.data.length ? <table className="w-full text-left text-sm"><tbody>{holdings.data.map((h) => <tr key={h.id} className="border-t border-ink/10"><td className="py-3 font-medium">{h.symbol}</td><td>{h.security_type}</td><td>{h.quantity}</td><td className="text-right">{money(h.market_value)}</td></tr>)}</tbody></table> : <Empty text="No holdings yet. Import holdings CSV to populate this view." />}</Card>
        <Card title="Risk">{risk.data.length ? risk.data.map((r) => <div key={r.title} className="mb-3 rounded-md bg-gold/15 p-3 text-sm"><p className="font-medium">{r.title}</p><p className="text-ink/65">{r.explanation}</p></div>) : <Empty text="No concentration warnings." />}</Card>
      </div>
    </Shell>
  );
}
