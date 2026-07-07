"use client";

import { Card } from "@/components/Shell";
import { Empty } from "@/components/Common";
import type { CashRecommendation } from "@/lib/api";

export function CashRecommendations({ recommendations, loading, error }: { recommendations: CashRecommendation[]; loading: boolean; error: string }) {
  return (
    <Card title="Cash recommendations">
      <p className="mb-4 text-sm leading-6 text-ink/60">Operational cash-management guidance only. This is not investment, tax, or lending advice.</p>
      {loading ? <div className="h-44 animate-pulse rounded-md bg-ink/[0.04]" /> : error ? <Empty text={`Could not load recommendations: ${error}`} /> : recommendations.length ? (
        <div className="grid gap-3">
          {recommendations.map((item) => (
            <article key={`${item.type}:${item.title}`} className="rounded-md border border-ink/10 p-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <div className="flex flex-wrap items-center gap-2">
                    <Priority value={item.priority} />
                    <span className="text-xs uppercase tracking-wide text-ink/40">{item.type.replaceAll("_", " ")}</span>
                  </div>
                  <p className="mt-2 font-semibold">{item.title}</p>
                  <p className="mt-1 text-sm leading-6 text-ink/60">{item.description}</p>
                </div>
                {item.amountMinor ? <p className="text-lg font-semibold text-ink">{moneyMinor(item.amountMinor, item.currency)}</p> : null}
              </div>
            </article>
          ))}
        </div>
      ) : <Empty text="No cash recommendations right now." />}
    </Card>
  );
}

function Priority({ value }: { value: string }) {
  const classes = value === "critical" || value === "high" ? "bg-coral/10 text-coral" : value === "medium" ? "bg-gold/20 text-[#755817]" : "bg-mint text-moss";
  return <span className={`rounded px-2 py-1 text-xs font-semibold uppercase tracking-wide ${classes}`}>{value}</span>;
}

function moneyMinor(value: number, currency: string) {
  return new Intl.NumberFormat("en-US", { style: "currency", currency, maximumFractionDigits: 0 }).format(value / 100);
}
