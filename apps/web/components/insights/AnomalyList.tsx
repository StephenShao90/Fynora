"use client";

import { Card } from "@/components/layout";
import { Empty } from "@/components/layout";
import type { AnomalyInsight } from "@/lib/api";

export function AnomalyList({ anomalies, loading, error }: { anomalies: AnomalyInsight[]; loading: boolean; error: string }) {
  return (
    <Card title="Active anomalies">
      {loading ? <div className="h-40 animate-pulse rounded-md bg-ink/[0.04]" /> : error ? <Empty text={`Could not load anomalies: ${error}`} /> : anomalies.length ? (
        <div className="grid gap-3">
          {anomalies.map((item) => (
            <article key={item.id} className="rounded-md border border-ink/10 p-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <div className="flex flex-wrap items-center gap-2">
                    <Severity value={item.severity} />
                    <p className="font-semibold">{item.title}</p>
                  </div>
                  <p className="mt-2 text-sm leading-6 text-ink/62">{item.description}</p>
                </div>
                <span className="rounded bg-ink/[0.04] px-2 py-1 text-xs font-medium text-ink/50">{item.resourceType}</span>
              </div>
              <div className="mt-3 grid gap-2 border-t border-ink/10 pt-3 text-sm text-ink/60">
                <p><span className="font-medium text-ink/70">Action:</span> {item.recommendedAction}</p>
                <p className="text-xs text-ink/40">Detected {formatDate(item.detectedAt)}</p>
              </div>
            </article>
          ))}
        </div>
      ) : <Empty text="No active payment issues found." />}
    </Card>
  );
}

function Severity({ value }: { value: string }) {
  const classes = value === "critical" || value === "high" ? "bg-coral/10 text-coral" : value === "medium" ? "bg-gold/20 text-[#755817]" : "bg-mint text-moss";
  return <span className={`rounded px-2 py-1 text-xs font-semibold uppercase tracking-wide ${classes}`}>{value}</span>;
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }).format(new Date(value));
}
