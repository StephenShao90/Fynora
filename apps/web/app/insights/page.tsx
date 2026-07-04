"use client";

import { Pie, PieChart, ResponsiveContainer, Cell, Tooltip } from "recharts";
import { Card, Shell } from "@/components/Shell";
import { Empty, Header } from "@/components/Common";
import { useApi } from "@/hooks/useApi";
import type { NamedAmount, RiskFinding } from "@/types";

const colors = ["#315846", "#f07b63", "#d6a53a", "#5a8bb0", "#7f6aaa", "#94b447"];

export default function Insights() {
  const categories = useApi<NamedAmount[]>("/insights/categories", []);
  const subscriptions = useApi<NamedAmount[]>("/insights/subscriptions", []);
  const anomalies = useApi<RiskFinding[]>("/insights/anomalies", []);
  return (
    <Shell>
      <Header title="Insights" subtitle="Recurring charges, anomalies, category mix, and merchant behavior." />
      <div className="grid gap-5 lg:grid-cols-2">
        <Card title="Category breakdown">
          <div className="h-80"><ResponsiveContainer><PieChart><Pie data={categories.data} dataKey="amount" nameKey="name">{categories.data.map((_, i) => <Cell key={i} fill={colors[i % colors.length]} />)}</Pie><Tooltip /></PieChart></ResponsiveContainer></div>
        </Card>
        <Card title="Subscriptions">
          {subscriptions.data.length ? <pre className="text-sm">{JSON.stringify(subscriptions.data, null, 2)}</pre> : <Empty text="No recurring subscriptions detected yet." />}
        </Card>
        <Card title="Anomalies">
          {anomalies.data.length ? <pre className="text-sm">{JSON.stringify(anomalies.data, null, 2)}</pre> : <Empty text="No unusual charges detected." />}
        </Card>
      </div>
    </Shell>
  );
}
