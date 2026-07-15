"use client";

import { Bar, BarChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { Card } from "@/components/layout";
import { Empty } from "@/components/layout";
import type { SpendingInsights as SpendingInsightsType } from "@/lib/api";

export function SpendingInsights({ spending, loading, error }: { spending?: SpendingInsightsType; loading: boolean; error: string }) {
  const chart = spending?.categories.map((item) => ({ ...item, amount: item.amountMinor / 100 })) || [];
  return (
    <Card title="Spending insights">
      {loading ? <div className="h-64 animate-pulse rounded-md bg-ink/[0.04]" /> : error ? <Empty text={`Could not load spending insights: ${error}`} /> : spending && spending.totalSpendMinor > 0 ? (
        <div className="grid gap-5 lg:grid-cols-[1fr_320px]">
          <div>
            <p className="text-sm text-ink/55">Total spend</p>
            <p className="mt-1 text-3xl font-semibold">{moneyMinor(spending.totalSpendMinor, spending.currency)}</p>
            <div className="mt-4 h-64">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={chart}>
                  <XAxis dataKey="category" tickLine={false} axisLine={false} />
                  <YAxis tickLine={false} axisLine={false} tickFormatter={(value) => money(Number(value), spending.currency)} width={64} />
                  <Tooltip formatter={(value) => money(Number(value), spending.currency)} />
                  <Bar dataKey="amount" fill="#315846" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>
          <div className="grid content-start gap-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Category mix</p>
              <div className="mt-2 grid gap-2">
                {spending.categories.map((item) => (
                  <div key={item.category} className="rounded-md bg-ink/[0.03] p-3">
                    <div className="flex items-center justify-between gap-3">
                      <p className="font-medium capitalize">{item.category}</p>
                      <p className="text-sm text-ink/55">{item.percentage.toFixed(1)}%</p>
                    </div>
                    <p className="mt-1 text-sm text-ink/55">{moneyMinor(item.amountMinor, spending.currency)} · {signedPercent(item.changeVsPreviousPeriod)} vs previous</p>
                  </div>
                ))}
              </div>
            </div>
            {spending.topMerchants.length ? (
              <div>
                <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Top merchants</p>
                <div className="mt-2 grid gap-2 text-sm">
                  {spending.topMerchants.map((item) => (
                    <div key={item.merchant} className="flex items-center justify-between border-b border-ink/10 pb-2 last:border-0">
                      <span className="text-ink/65">{item.merchant}</span>
                      <span className="font-medium">{moneyMinor(item.amountMinor, spending.currency)}</span>
                    </div>
                  ))}
                </div>
              </div>
            ) : null}
            {spending.notes.length ? <p className="rounded-md bg-mint px-3 py-2 text-sm leading-6 text-moss">{spending.notes[0]}</p> : null}
          </div>
        </div>
      ) : <Empty text="No debit spending found for this period." />}
    </Card>
  );
}

function signedPercent(value: number) {
  if (value === 0) return "0.0%";
  return `${value > 0 ? "+" : ""}${value.toFixed(1)}%`;
}

function money(value: number, currency: string) {
  return new Intl.NumberFormat("en-US", { style: "currency", currency, maximumFractionDigits: 0 }).format(value);
}

function moneyMinor(value: number, currency: string) {
  return money(value / 100, currency);
}
