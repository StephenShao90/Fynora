"use client";

import { Area, AreaChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import type { ForecastPoint } from "@/lib/api";

export function ForecastChart({ points, currency }: { points: ForecastPoint[]; currency: string }) {
  const data = points.map((point) => ({
    ...point,
    balance: point.projectedBalanceMinor / 100,
    inflows: point.expectedInflowsMinor / 100,
    outflows: point.expectedOutflowsMinor / 100
  }));

  return (
    <div className="h-72">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
          <XAxis dataKey="date" tickLine={false} axisLine={false} tickFormatter={(value) => shortDate(value)} minTickGap={28} />
          <YAxis tickLine={false} axisLine={false} tickFormatter={(value) => compactMoney(Number(value), currency)} width={64} />
          <Tooltip
            formatter={(value, name) => [moneyMinor(Number(value) * 100, currency), labelFor(name as string)]}
            labelFormatter={(value) => shortDate(String(value))}
          />
          <Area type="monotone" dataKey="balance" stroke="#315846" fill="#dcefe3" strokeWidth={3} />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

function labelFor(value: string) {
  if (value === "balance") return "Projected balance";
  if (value === "inflows") return "Expected inflows";
  if (value === "outflows") return "Expected outflows";
  return value;
}

function shortDate(value: string) {
  const raw = String(value);
  const date = new Date(raw.includes("T") ? raw : `${raw}T00:00:00Z`);
  if (Number.isNaN(date.getTime())) return raw;
  return new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric" }).format(date);
}

function compactMoney(value: number, currency: string) {
  return new Intl.NumberFormat("en-US", { style: "currency", currency, notation: "compact", maximumFractionDigits: 1 }).format(value);
}

function moneyMinor(value: number, currency: string) {
  return new Intl.NumberFormat("en-US", { style: "currency", currency, maximumFractionDigits: 0 }).format(value / 100);
}
