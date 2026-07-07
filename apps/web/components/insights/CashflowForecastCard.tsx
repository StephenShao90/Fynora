"use client";

import { Card } from "@/components/Shell";
import { Empty } from "@/components/Common";
import type { CashflowForecast } from "@/lib/api";
import { ForecastChart } from "./ForecastChart";

const horizons = [7, 30, 60, 90];

export function CashflowForecastCard({
  forecast,
  horizon,
  onHorizonChange,
  loading,
  error
}: {
  forecast?: CashflowForecast;
  horizon: number;
  onHorizonChange: (value: number) => void;
  loading: boolean;
  error: string;
}) {
  return (
    <Card title="Cash-flow forecast">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-sm leading-6 text-ink/60">Projected cash position based on recent deposits, operating debits, and known pending payouts.</p>
          {forecast ? (
            <div className="mt-4 grid gap-3 sm:grid-cols-3">
              <Metric label="Starting balance" value={moneyMinor(forecast.startingBalanceMinor, forecast.currency)} />
              <Metric label="Projected ending" value={moneyMinor(forecast.projectedEndingBalanceMinor, forecast.currency)} tone={forecast.projectedEndingBalanceMinor < forecast.startingBalanceMinor ? "warn" : "good"} />
              <Metric label="Confidence" value={forecast.confidence} tone={forecast.confidence === "low" ? "warn" : "good"} />
            </div>
          ) : null}
        </div>
        <div className="inline-flex rounded-md border border-ink/10 bg-ink/[0.03] p-1">
          {horizons.map((value) => (
            <button
              key={value}
              onClick={() => onHorizonChange(value)}
              className={`rounded px-3 py-1.5 text-sm font-medium ${horizon === value ? "bg-white text-ink shadow-sm" : "text-ink/55 hover:text-ink"}`}
            >
              {value}d
            </button>
          ))}
        </div>
      </div>

      <div className="mt-5">
        {loading ? <Skeleton /> : error ? <Empty text={`Could not load forecast: ${error}`} /> : forecast?.series.length ? <ForecastChart points={forecast.series} currency={forecast.currency} /> : <Empty text="No forecast data available yet." />}
      </div>

      {forecast?.assumptions.length ? (
        <div className="mt-4 rounded-md bg-ink/[0.03] p-4">
          <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Assumptions</p>
          <ul className="mt-2 grid gap-1 text-sm leading-6 text-ink/60">
            {forecast.assumptions.map((item) => <li key={item}>{item}</li>)}
          </ul>
        </div>
      ) : null}
    </Card>
  );
}

function Metric({ label, value, tone = "neutral" }: { label: string; value: string; tone?: "neutral" | "good" | "warn" }) {
  const color = tone === "good" ? "text-moss" : tone === "warn" ? "text-coral" : "text-ink";
  return (
    <div>
      <p className="text-xs font-medium uppercase tracking-wide text-ink/40">{label}</p>
      <p className={`mt-1 text-xl font-semibold ${color}`}>{value}</p>
    </div>
  );
}

function Skeleton() {
  return <div className="h-72 animate-pulse rounded-md bg-ink/[0.04]" />;
}

function moneyMinor(value: number, currency: string) {
  return new Intl.NumberFormat("en-US", { style: "currency", currency, maximumFractionDigits: 0 }).format(value / 100);
}
