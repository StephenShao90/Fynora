"use client";

import { useEffect, useState } from "react";
import { Header } from "@/components/Common";
import { GuideMarker } from "@/components/GuideMarker";
import { Shell } from "@/components/Shell";
import { AnomalyList } from "@/components/insights/AnomalyList";
import { CashRecommendations } from "@/components/insights/CashRecommendations";
import { CashflowForecastCard } from "@/components/insights/CashflowForecastCard";
import { SpendingInsights } from "@/components/insights/SpendingInsights";
import { ReconciliationMatches } from "@/components/reconciliation/ReconciliationMatches";
import {
  getAnomalies,
  getCashRecommendations,
  getCashflowForecast,
  getReconciliationMatches,
  getSpendingInsights,
  type AnomalyInsight,
  type CashRecommendation,
  type CashflowForecast,
  type ReconciliationMatch,
  type SpendingInsights as SpendingInsightsType
} from "@/lib/api";
import { useApi } from "@/hooks/useApi";

type Run = { id: string; status: string; matched_count: number; exception_count: number; started_at: string };
type LoadState<T> = { data?: T; loading: boolean; error: string };

export default function Insights() {
  const [horizon, setHorizon] = useState(30);
  const [forecast, setForecast] = useState<LoadState<CashflowForecast>>({ loading: true, error: "" });
  const [anomalies, setAnomalies] = useState<LoadState<AnomalyInsight[]>>({ data: [], loading: true, error: "" });
  const [recommendations, setRecommendations] = useState<LoadState<CashRecommendation[]>>({ data: [], loading: true, error: "" });
  const [spending, setSpending] = useState<LoadState<SpendingInsightsType>>({ loading: true, error: "" });
  const [matches, setMatches] = useState<LoadState<ReconciliationMatch[]>>({ data: [], loading: true, error: "" });
  const runs = useApi<Run[]>("/reconciliation/runs", []);
  const latestRunId = runs.data[0]?.id || "latest";

  useEffect(() => {
    let cancelled = false;
    setForecast((current) => ({ ...current, loading: true, error: "" }));
    getCashflowForecast(horizon)
      .then((data) => !cancelled && setForecast({ data, loading: false, error: "" }))
      .catch((err) => !cancelled && setForecast({ loading: false, error: (err as Error).message }));
    return () => { cancelled = true; };
  }, [horizon]);

  useEffect(() => {
    let cancelled = false;
    setAnomalies((current) => ({ ...current, loading: true, error: "" }));
    getAnomalies()
      .then((data) => !cancelled && setAnomalies({ data, loading: false, error: "" }))
      .catch((err) => !cancelled && setAnomalies({ data: [], loading: false, error: (err as Error).message }));
    return () => { cancelled = true; };
  }, []);

  useEffect(() => {
    let cancelled = false;
    setRecommendations((current) => ({ ...current, loading: true, error: "" }));
    getCashRecommendations()
      .then((data) => !cancelled && setRecommendations({ data, loading: false, error: "" }))
      .catch((err) => !cancelled && setRecommendations({ data: [], loading: false, error: (err as Error).message }));
    return () => { cancelled = true; };
  }, []);

  useEffect(() => {
    let cancelled = false;
    setSpending((current) => ({ ...current, loading: true, error: "" }));
    getSpendingInsights()
      .then((data) => !cancelled && setSpending({ data, loading: false, error: "" }))
      .catch((err) => !cancelled && setSpending({ loading: false, error: (err as Error).message }));
    return () => { cancelled = true; };
  }, []);

  useEffect(() => {
    let cancelled = false;
    setMatches((current) => ({ ...current, loading: true, error: "" }));
    getReconciliationMatches(latestRunId)
      .then((data) => !cancelled && setMatches({ data, loading: false, error: "" }))
      .catch((err) => !cancelled && setMatches({ data: [], loading: false, error: (err as Error).message }));
    return () => { cancelled = true; };
  }, [latestRunId]);

  return (
    <Shell>
      <Header title="Financial intelligence" subtitle="Forecast cash, explain reconciliation outcomes, and surface payment operations issues before they become reporting problems." />

      <div className="grid gap-4 xl:grid-cols-[1.2fr_.8fr]">
        <div className="relative"><div className="absolute right-4 top-4 z-10"><GuideMarker guide={{ number: 1, title: "Cash-flow forecast", body: "Change the horizon to project cash over time. Use the chart and assumptions to explain where cash may tighten." }} /></div><CashflowForecastCard forecast={forecast.data} horizon={horizon} onHorizonChange={setHorizon} loading={forecast.loading} error={forecast.error} /></div>
        <div className="relative"><div className="absolute right-4 top-4 z-10"><GuideMarker guide={{ number: 2, title: "Cash recommendations", body: "Prioritized operational suggestions based on cash position, reserves, and upcoming pressure." }} /></div><CashRecommendations recommendations={recommendations.data || []} loading={recommendations.loading} error={recommendations.error} /></div>
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-[.95fr_1.05fr]">
        <div className="relative"><div className="absolute right-4 top-4 z-10"><GuideMarker guide={{ number: 3, title: "Anomalies", body: "Review unusual or missing financial events. High-severity anomalies should usually send you back to Reconciliation." }} /></div><AnomalyList anomalies={anomalies.data || []} loading={anomalies.loading} error={anomalies.error} /></div>
        <div className="relative"><div className="absolute right-4 top-4 z-10"><GuideMarker guide={{ number: 4, title: "Match intelligence", body: "Explains reconciliation confidence and reasons. Use this to justify why a payout and bank deposit were matched or flagged." }} /></div><ReconciliationMatches matches={matches.data || []} loading={matches.loading || runs.loading} error={matches.error || runs.error} /></div>
      </div>

      <div className="mt-4">
        <div className="relative"><div className="absolute right-4 top-4 z-10"><GuideMarker guide={{ number: 5, title: "Spending insights", body: "Shows categorized spending patterns from normalized transactions. Use it to understand operating cost drivers." }} /></div><SpendingInsights spending={spending.data} loading={spending.loading} error={spending.error} /></div>
      </div>
    </Shell>
  );
}
