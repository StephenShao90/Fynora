"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { api } from "@/lib/api";
import { useToast } from "@/components/layout";

type StepStatus = "idle" | "running" | "passed" | "failed";
type DemoStep = {
  id: string;
  label: string;
  purpose: string;
  run: () => Promise<unknown>;
};

export function DemoPilot({ compact = false }: { compact?: boolean }) {
  const { pushToast } = useToast();
  const [running, setRunning] = useState(false);
  const [statuses, setStatuses] = useState<Record<string, StepStatus>>({});
  const [lastRun, setLastRun] = useState("");
  const [error, setError] = useState("");

  const steps = useMemo<DemoStep[]>(() => [
    {
      id: "onboarding",
      label: "Save onboarding state",
      purpose: "Persists the workspace/product setup checklist.",
      run: () => api("/api/v1/onboarding/status", {
        method: "PUT",
        body: JSON.stringify({
          selected_scenario: "student_org",
          business_type: "student_organization",
          checklist: {
            workspace_created: true,
            business_profile_chosen: true,
            demo_walkthrough_started: true
          }
        })
      })
    },
    {
      id: "stripe",
      label: "Sync processor data",
      purpose: "Loads Stripe-style payments, refunds, fees, and payouts.",
      run: () => api("/sync/stripe", {
        method: "POST",
        headers: { "Idempotency-Key": `demo-pilot-stripe-${Date.now()}` },
        body: "{}"
      })
    },
    {
      id: "bank",
      label: "Sync bank data",
      purpose: "Loads Plaid/CSV-style deposits and operating debits.",
      run: () => api("/sync/bank", {
        method: "POST",
        headers: { "Idempotency-Key": `demo-pilot-bank-${Date.now()}` },
        body: "{}"
      })
    },
    {
      id: "reconciliation",
      label: "Run reconciliation",
      purpose: "Matches processor payouts to bank deposits and creates breaks.",
      run: () => api("/reconciliation/runs", {
        method: "POST",
        headers: { "Idempotency-Key": `demo-pilot-recon-${Date.now()}` },
        body: "{}"
      })
    }
  ], []);

  async function runDemo() {
    setRunning(true);
    setError("");
    setLastRun("");
    setStatuses(Object.fromEntries(steps.map((step) => [step.id, "idle"])));
    console.info("[clearflow-demo-pilot:start]", { steps: steps.map((step) => step.id) });

    try {
      for (const step of steps) {
        setStatuses((current) => ({ ...current, [step.id]: "running" }));
        console.info("[clearflow-demo-pilot:step]", { id: step.id, label: step.label });
        await step.run();
        setStatuses((current) => ({ ...current, [step.id]: "passed" }));
      }
      const completedAt = new Date().toISOString();
      setLastRun(completedAt);
      console.info("[clearflow-demo-pilot:complete]", { completedAt });
      pushToast({ tone: "success", title: "Sample close ready", detail: "Home base, reconciliation, and cash forecast now have fresh data." });
    } catch (err) {
      const message = (err as Error).message;
      setError(message);
      console.error("[clearflow-demo-pilot:error]", { message });
      setStatuses((current) => {
        const next = { ...current };
        const runningStep = Object.entries(next).find(([, status]) => status === "running");
        if (runningStep) next[runningStep[0]] = "failed";
        return next;
      });
      pushToast({ tone: "error", title: "Demo workflow stopped", detail: message });
    } finally {
      setRunning(false);
    }
  }

  return (
    <section className="rounded-md border border-ink/10 bg-white p-5 shadow-sm">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Sample close runner</p>
          <h2 className="mt-1 text-lg font-semibold text-ink">Prepare a complete payout close</h2>
          <p className="mt-1 max-w-2xl text-sm leading-6 text-ink/55">
            Loads the sample workspace in the same order an operator would close cash: team setup, processor payouts, bank deposits, then payout matching.
          </p>
        </div>
        <button
          type="button"
          onClick={runDemo}
          disabled={running}
          className="rounded-md bg-ink px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
        >
          {running ? "Running..." : "Prepare sample close"}
        </button>
      </div>

      <div className={`mt-4 grid gap-2 ${compact ? "md:grid-cols-1" : "md:grid-cols-4"}`}>
        {steps.map((step) => (
          <div key={step.id} className="rounded-md border border-ink/10 p-3">
            <div className="flex items-center justify-between gap-2">
              <p className="text-sm font-semibold text-ink">{step.label}</p>
              <StatusBadge status={statuses[step.id] || "idle"} />
            </div>
            <p className="mt-2 text-xs leading-5 text-ink/55">{step.purpose}</p>
          </div>
        ))}
      </div>

      {error ? <p className="mt-3 rounded-md border border-coral/30 bg-coral/10 p-3 text-sm text-coral">{error}</p> : null}
      {lastRun ? (
        <div className="mt-3 flex flex-wrap items-center gap-3 rounded-md bg-mint/60 p-3 text-sm text-moss">
          <span className="font-semibold">Ready at {new Date(lastRun).toLocaleTimeString()}</span>
          <Link href="/dashboard" className="underline">Open home base</Link>
          <Link href="/reconciliation" className="underline">Review breaks</Link>
        </div>
      ) : null}
    </section>
  );
}

function StatusBadge({ status }: { status: StepStatus }) {
  const classes: Record<StepStatus, string> = {
    idle: "bg-ink/5 text-ink/45",
    running: "bg-gold/20 text-ink",
    passed: "bg-mint text-moss",
    failed: "bg-coral/10 text-coral"
  };
  const label = status === "idle" ? "ready" : status;
  return <span className={`rounded px-2 py-1 text-xs font-semibold ${classes[status]}`}>{label}</span>;
}
