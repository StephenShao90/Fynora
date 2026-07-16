"use client";

import { Card, Empty, SkeletonBlock } from "@/components/layout";
import type { PayoutExplanation } from "@/lib/api";

export function PayoutExplanationPanel({ explanation, loading, error }: { explanation?: PayoutExplanation; loading: boolean; error: string }) {
  return (
    <Card title="Payout explanation">
      {loading ? <SkeletonBlock className="h-48" /> : error ? <Empty text={`Could not load payout explanation: ${error}`} /> : explanation ? (
        <div>
          <p className="text-sm leading-6 text-ink/62">{explanation.summary}</p>
          <div className="mt-4 grid gap-3 sm:grid-cols-4">
            <Amount label="Gross" value={explanation.grossAmountMinor} currency={explanation.currency} />
            <Amount label="Fees" value={-explanation.feesMinor} currency={explanation.currency} tone="warn" />
            <Amount label="Refunds" value={-explanation.refundsMinor} currency={explanation.currency} tone="warn" />
            <Amount label="Net deposit" value={explanation.netAmountMinor} currency={explanation.currency} tone="good" />
          </div>
          <div className="mt-4 rounded-md bg-ink/[0.03] p-3 text-sm">
            <p className="font-medium">Matching bank deposit</p>
            {explanation.bankDeposit ? (
              <p className="mt-1 text-ink/60">{moneyMinor(explanation.bankDeposit.amountMinor, explanation.currency)} posted {formatDate(explanation.bankDeposit.postedAt)} · {explanation.bankDeposit.id.slice(0, 8)}</p>
            ) : <p className="mt-1 text-coral">No matching bank deposit found.</p>}
          </div>
          {explanation.warnings.length ? (
            <div className="mt-4 rounded-md border border-coral/20 bg-coral/5 p-3">
              <p className="text-xs font-semibold uppercase tracking-wide text-coral">Warnings</p>
              <ul className="mt-2 grid gap-1 text-sm text-ink/65">
                {explanation.warnings.map((item) => <li key={item}>{item}</li>)}
              </ul>
            </div>
          ) : null}
          {explanation.lineItems.length ? <pre className="mt-4 overflow-auto rounded-md bg-ink p-3 text-xs text-white">{JSON.stringify(explanation.lineItems, null, 2)}</pre> : null}
        </div>
      ) : <Empty text="Select a payout to view its explanation." />}
    </Card>
  );
}

function Amount({ label, value, currency, tone = "neutral" }: { label: string; value: number; currency: string; tone?: "neutral" | "good" | "warn" }) {
  const color = tone === "good" ? "text-moss" : tone === "warn" ? "text-coral" : "text-ink";
  return (
    <div className="rounded-md border border-ink/10 p-3">
      <p className="text-xs font-medium uppercase tracking-wide text-ink/40">{label}</p>
      <p className={`mt-1 text-xl font-semibold ${color}`}>{moneyMinor(value, currency)}</p>
    </div>
  );
}

function moneyMinor(value: number, currency: string) {
  return new Intl.NumberFormat("en-US", { style: "currency", currency, maximumFractionDigits: 0 }).format(value / 100);
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric" }).format(new Date(value));
}
