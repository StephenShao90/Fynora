"use client";

import { Bar, BarChart, CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { money } from "@/components/layout";

type ForecastPoint = Record<string, number>;
type PayoutPoint = { processor_payout_id: string; amount: number };

export function CashForecastMiniChart({ data }: { data: ForecastPoint[] }) {
  return (
    <ResponsiveContainer width="100%" height="100%">
      <LineChart data={data}>
        <CartesianGrid stroke="#dfe5dc" strokeDasharray="3 3" />
        <XAxis dataKey="days" tickLine={false} axisLine={false} label={{ value: "Days ahead", position: "insideBottom", offset: -4 }} />
        <YAxis tickLine={false} axisLine={false} tickFormatter={(value) => `$${Number(value).toLocaleString()}`} label={{ value: "Projected cash", angle: -90, position: "insideLeft" }} />
        <Tooltip formatter={(value) => money(Number(value))} />
        <Line type="monotone" dataKey="projected_cash" stroke="#17211b" strokeWidth={3} dot={{ r: 4 }} />
      </LineChart>
    </ResponsiveContainer>
  );
}

export function PayoutVolumeChart({ data }: { data: PayoutPoint[] }) {
  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart data={data}>
        <XAxis dataKey="processor_payout_id" tickLine={false} axisLine={false} />
        <YAxis tickLine={false} axisLine={false} />
        <Tooltip formatter={(value) => money(Number(value))} />
        <Bar dataKey="amount" fill="#315846" radius={[4, 4, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}
