"use client";

import { Card, Empty, SkeletonBlock } from "@/components/layout";
import type { ReconciliationMatch } from "@/lib/api";

export function ReconciliationMatches({ matches, loading, error }: { matches: ReconciliationMatch[]; loading: boolean; error: string }) {
  return (
    <Card title="Match intelligence">
      {loading ? <SkeletonBlock className="h-48" /> : error ? <Empty text={`Could not load match intelligence: ${error}`} /> : matches.length ? (
        <div className="grid gap-3">
          {matches.slice(0, 8).map((item) => (
            <article key={item.id} className="rounded-md border border-ink/10 p-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <div className="flex flex-wrap items-center gap-2">
                    <Status value={item.status} />
                    <p className="font-semibold">{Math.round(item.confidenceScore * 100)}% confidence</p>
                  </div>
                  <p className="mt-2 text-sm leading-6 text-ink/62">{item.explanation}</p>
                </div>
                <p className="text-sm font-medium text-ink/65">{moneyMinor(item.amountDifferenceMinor, item.currency)} difference</p>
              </div>
              <div className="mt-3 flex flex-wrap gap-2">
                {item.reasons.map((reason) => <span key={reason} className="rounded bg-ink/[0.04] px-2 py-1 text-xs font-medium text-ink/55">{reason}</span>)}
              </div>
              <p className="mt-3 text-xs text-ink/40">Payout {item.processorPayoutId?.slice(0, 8) || "missing"} · Bank {item.bankDepositId?.slice(0, 8) || "missing"}</p>
            </article>
          ))}
        </div>
      ) : <Empty text="No reconciliation match intelligence available yet." />}
    </Card>
  );
}

function Status({ value }: { value: string }) {
  const good = value === "matched" || value === "likely_match";
  const classes = good ? "bg-mint text-moss" : "bg-coral/10 text-coral";
  return <span className={`rounded px-2 py-1 text-xs font-semibold uppercase tracking-wide ${classes}`}>{value.replaceAll("_", " ")}</span>;
}

function moneyMinor(value: number, currency: string) {
  return new Intl.NumberFormat("en-US", { style: "currency", currency, maximumFractionDigits: 0 }).format(value / 100);
}
