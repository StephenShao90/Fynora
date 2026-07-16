"use client";

import { Bar, BarChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

type SpendingChartPoint = {
  category: string;
  amount: number;
};

export function SpendingChart({ data, currency }: { data: SpendingChartPoint[]; currency: string }) {
  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart data={data}>
        <XAxis dataKey="category" tickLine={false} axisLine={false} />
        <YAxis tickLine={false} axisLine={false} tickFormatter={(value) => money(Number(value), currency)} width={64} />
        <Tooltip formatter={(value) => money(Number(value), currency)} />
        <Bar dataKey="amount" fill="#315846" radius={[4, 4, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}

function money(value: number, currency: string) {
  return new Intl.NumberFormat("en-US", { style: "currency", currency, maximumFractionDigits: 0 }).format(value);
}
