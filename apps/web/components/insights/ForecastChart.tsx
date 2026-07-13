"use client";

import { Area, AreaChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import type { ForecastPoint } from "@/lib/api";

export function ForecastChart({ points, currency }: { points: ForecastPoint[]; currency: string }) {
  const data = points.map((point) => ({
    ...point,
    balance: point.projectedBalanceMinor / 100,
    inflows: point.expectedInflowsMinor / 100,
    outflows: point.expectedOutflowsMinor / 100
  }));

  return (
    <div>
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2 text-xs text-ink/45">
        <span>X-axis: calendar day in forecast window</span>
        <span>Y-axis: cash amount</span>
      </div>
      <div className="h-72">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 8, right: 12, left: 0, bottom: 18 }}>
            <CartesianGrid stroke="#dfe5dc" strokeDasharray="3 3" />
            <XAxis dataKey="date" tickLine={false} axisLine={false} tickFormatter={(value) => shortDate(value)} minTickGap={28} label={{ value: "Forecast date", position: "insideBottom", offset: -12 }} />
            <YAxis tickLine={false} axisLine={false} tickFormatter={(value) => compactMoney(Number(value), currency)} width={64} label={{ value: "Cash amount", angle: -90, position: "insideLeft" }} />
            <Tooltip
              formatter={(value, name) => [moneyMinor(Number(value) * 100, currency), labelFor(name as string)]}
              labelFormatter={(value) => shortDate(String(value))}
            />
            <Legend verticalAlign="top" height={28} />
            <Area type="monotone" dataKey="balance" name="Projected cash" stroke="#315846" fill="#dcefe3" strokeWidth={3} />
            <Area type="monotone" dataKey="inflows" name="Expected inflows" stroke="#5a8bb0" fill="#d9e8f4" strokeWidth={2} />
            <Area type="monotone" dataKey="outflows" name="Expected outflows" stroke="#f07b63" fill="#f7d8d1" strokeWidth={2} />
          </AreaChart>
        </ResponsiveContainer>
      </div>
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
