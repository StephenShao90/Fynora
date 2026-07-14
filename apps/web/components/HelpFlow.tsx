"use client";

import { useMemo, useState } from "react";

type Slide = {
  title: string;
  body: string;
  steps: string[];
};

const flows: Record<string, Slide[]> = {
  "Onboarding": [
    { title: "Create the workspace", body: "Start by naming the company, picking the business type, and saving the workspace profile.", steps: ["Choose the closest operating model.", "Confirm currency.", "Save workspace to create or switch the active company."] },
    { title: "Run a full demo setup", body: "The guided demo runner prepares the whole product story without making you remember endpoint order.", steps: ["Click Run full demo setup.", "Wait for every step to show passed.", "Open Reconciliation, Ops, and Portfolio from the ready links."] },
    { title: "Finish the data checklist", body: "The checklist turns setup into concrete steps so you know exactly what data is missing.", steps: ["Connect or sync processor data.", "Connect Plaid or import bank activity.", "Upload portfolio data only if it matters for the use case."] },
    { title: "Switch demo companies", body: "The scenario switcher changes the sample company so Clearflow demos across student orgs, creators, SaaS, and nonprofits.", steps: ["Pick a scenario.", "Open Dashboard.", "Verify cash and workflow language fit the company."] }
  ],
  "Operations dashboard": [
    { title: "Prepare demo data", body: "Use the guided demo runner when you want a clean local walkthrough without terminal commands.", steps: ["Click Run full demo setup.", "Confirm every step passes.", "Use the checklist and KPI cards to explain what changed."] },
    { title: "Read cash health first", body: "The top cards summarize bank cash, net operating movement, processor costs, refunds, and unresolved breaks.", steps: ["Operating cash is posted bank cash.", "Net flow subtracts debits, fees, and refunds.", "Open breaks are reconciliation issues that still need action."] },
    { title: "Use the forecast", body: "The chart plots projected cash by day so you can see whether expected activity will pressure operating cash.", steps: ["X-axis means days ahead.", "Y-axis means projected dollar amount.", "Hover points to inspect projected cash, payouts, and expenses."] },
    { title: "Move to exceptions", body: "If open breaks are nonzero, go to Reconciliation and resolve or explain the exception before reporting.", steps: ["Sync processor.", "Sync bank.", "Run reconciliation.", "Review and resolve open breaks."] }
  ],
  "Reconciliation": [
    { title: "Run the workflow", body: "The runbook queues production-style jobs so the worker does the financial work outside the request.", steps: ["Use Run full reconciliation for the normal path.", "Or run processor sync, bank sync, then reconcile manually.", "Keep make worker running until each job completes."] },
    { title: "Review exceptions", body: "Open exceptions are active operational breaks. Resolving one removes it from the active queue and updates open-break counts.", steps: ["Read the exception explanation.", "Confirm it is expected or handled.", "Click Resolve."] },
    { title: "Explain payouts", body: "The payout ledger lets you inspect how a net bank deposit was formed.", steps: ["Click View explanation.", "Review gross payments, fees, refunds, and bank deposit match.", "Use warnings to decide whether follow-up is needed."] }
  ],
  "Data connections": [
    { title: "Connect bank data", body: "Plaid handles login and MFA. Clearflow receives authorized transaction data and stores provider tokens only on the backend.", steps: ["Use Connect bank with Plaid for the full Link flow.", "Use Create sandbox test connection for fast local testing.", "Use Sync connected banks to refresh transactions."] },
    { title: "Use CSV backups", body: "CSV import is there for manual exports and demo data when provider access is unavailable.", steps: ["Download the sample CSV.", "Upload it in the matching card.", "Confirm the JSON result shows imported records."] }
  ],
  "Integrations": [
    { title: "Stripe connection", body: "Stripe Connect authorizes processor data access without storing your Stripe password.", steps: ["Click Connect Stripe.", "Complete or cancel Stripe onboarding.", "You return to this page with connection status."] },
    { title: "Provider status", body: "The cards show whether Stripe and Plaid are connected and when data last synced.", steps: ["Connected means tokens exist server-side.", "Last sync shows freshness.", "Last error tells you whether provider sync failed."] },
    { title: "Run syncs manually", body: "Provider sync controls let you trigger ingestion without leaving the integration page.", steps: ["Sync Stripe for processor data.", "Sync Plaid bank for bank transactions.", "Sync investments sample to populate Portfolio."] }
  ],
  "Operations": [
    { title: "Watch the worker", body: "Async jobs show whether background sync and reconciliation are actually being processed.", steps: ["Run Reconciliation workflow.", "Open Ops.", "Recent jobs should show queued/running/completed entries."] },
    { title: "Use audit logs", body: "Audit trail records important actions for debugging and compliance-style review.", steps: ["Look for job.completed.", "Look for reconciliation actions.", "Copy timestamps and request IDs when reporting bugs."] }
  ],
  "Financial intelligence": [
    { title: "Change the horizon", body: "Forecast buttons rerun the cash model at different time windows.", steps: ["Click 7d for near-term cash.", "Click 30d/60d/90d for planning.", "Review assumptions below the chart."] },
    { title: "Read recommendations", body: "Recommendations and anomalies prioritize issues before they become reporting problems.", steps: ["Check priority labels.", "Read recommended action.", "Go back to Reconciliation if the issue is a break."] }
  ],
  "Portfolio": [
    { title: "Inspect holdings", body: "Portfolio views are for imported/manual holdings, not brokerage scraping.", steps: ["Market value summarizes holdings.", "Allocation charts show concentration.", "Risk panel flags overweight positions."] }
  ],
  "Advisor": [
    { title: "Use rule-based guidance", body: "Advisor features turn cash and portfolio data into educational estimates.", steps: ["Simulate projection with contribution inputs.", "Ask a cash-flow question.", "Treat responses as educational, not personalized financial advice."] }
  ],
  "Transactions": [
    { title: "Review normalized records", body: "Transactions show categorized spending and merchant normalization.", steps: ["Check date and merchant.", "Confirm category/direction.", "Use Imports if rows are missing."] }
  ],
  "Settings": [
    { title: "Manage local demo state", body: "Settings gives you a quick way to recover a known-good local scenario after lots of manual testing.", steps: ["Use Reset demo data to reseed payments, payouts, bank deposits, and reconciliation breaks.", "Use Log out to clear the browser token.", "Never enter real brokerage credentials in this MVP."] },
    { title: "Manage team access", body: "The Team panel uses organization membership APIs so you can demonstrate RBAC, invites, role changes, and removal.", steps: ["Add a teammate by email.", "Choose viewer, analyst, or admin.", "Owners are protected from accidental removal."] },
    { title: "Prepare deployment", body: "Production readiness notes keep frontend, API, worker, database, Redis, and provider secrets separated.", steps: ["Frontend goes to Vercel.", "API and worker go to a backend host.", "Provider secrets stay backend-only."] }
  ]
};

const fallbackSlides: Slide[] = [
  { title: "How to test this page", body: "Use the primary controls, confirm the page updates, and copy console/API logs if anything fails.", steps: ["Click the main action.", "Watch for updated cards or tables.", "Check browser console for clearflow-api logs."] }
];

export function HelpFlow({ page }: { page: string }) {
  const [open, setOpen] = useState(false);
  const [index, setIndex] = useState(0);
  const slides = useMemo(() => flows[page] || fallbackSlides, [page]);
  const slide = slides[index] || slides[0];

  function close() {
    setOpen(false);
    setIndex(0);
  }

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="inline-flex h-9 w-9 items-center justify-center rounded-full border border-ink/15 bg-white text-sm font-semibold text-ink/70 shadow-sm hover:bg-ink/[0.03]"
        aria-label={`Help for ${page}`}
        title={`Help for ${page}`}
      >
        ?
      </button>
      {open ? (
        <div className="fixed inset-0 z-50 grid place-items-center bg-ink/35 px-4">
          <section className="w-full max-w-2xl rounded-lg border border-ink/10 bg-white p-5 shadow-panel">
            <div className="flex items-start justify-between gap-4">
              <div>
                <p className="text-xs font-semibold uppercase tracking-wide text-moss">{page} walkthrough</p>
                <h2 className="mt-2 text-2xl font-semibold text-ink">{slide.title}</h2>
              </div>
              <button onClick={close} className="rounded-md border border-ink/15 px-3 py-1.5 text-sm text-ink/65 hover:bg-ink/[0.03]">Close</button>
            </div>
            <div className="mt-5 rounded-md border border-ink/10 bg-[#f7faf6] p-4">
              <div className="mb-4 h-28 rounded-md border border-dashed border-moss/30 bg-white p-3">
                <div className="h-3 w-28 rounded bg-moss/25" />
                <div className="mt-3 grid grid-cols-3 gap-2">
                  <div className="h-14 rounded bg-mint" />
                  <div className="h-14 rounded bg-gold/25" />
                  <div className="h-14 rounded bg-coral/15" />
                </div>
              </div>
              <p className="text-sm leading-6 text-ink/65">{slide.body}</p>
              <ol className="mt-4 grid gap-2 text-sm text-ink/70">
                {slide.steps.map((step, stepIndex) => (
                  <li key={step} className="flex gap-2">
                    <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-ink text-xs text-white">{stepIndex + 1}</span>
                    <span>{step}</span>
                  </li>
                ))}
              </ol>
            </div>
            <div className="mt-5 flex items-center justify-between">
              <p className="text-sm text-ink/45">{index + 1} of {slides.length}</p>
              <div className="flex gap-2">
                <button onClick={() => setIndex((value) => Math.max(0, value - 1))} disabled={index === 0} className="rounded-md border border-ink/15 px-3 py-2 text-sm disabled:opacity-40">Previous</button>
                <button onClick={() => setIndex((value) => Math.min(slides.length - 1, value + 1))} disabled={index === slides.length - 1} className="rounded-md bg-ink px-3 py-2 text-sm text-white disabled:opacity-40">Next</button>
              </div>
            </div>
          </section>
        </div>
      ) : null}
    </>
  );
}
