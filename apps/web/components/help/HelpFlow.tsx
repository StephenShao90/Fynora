"use client";

import { useEffect, useMemo, useState } from "react";

export type HelpNode = {
  label: string;
  x: number;
  y: number;
  tone?: "primary" | "success" | "warn" | "muted";
};

export type HelpSlide = {
  title: string;
  body: string;
  nodes: HelpNode[];
  note?: string;
};

const defaultSlides: HelpSlide[] = [
  {
    title: "Start with the close question",
    body: "Clearflow is built around one operating question: did processor activity become trusted bank cash?",
    nodes: [
      { label: "Processor payouts", x: 18, y: 42, tone: "primary" },
      { label: "Bank deposits", x: 50, y: 42, tone: "success" },
      { label: "Open breaks", x: 82, y: 42, tone: "warn" }
    ],
    note: "Read left to right: source data, cash evidence, then exceptions."
  },
  {
    title: "Use the page action",
    body: "Each page has one main job: load data, reconcile payouts, forecast cash, or prove operational control.",
    nodes: [
      { label: "Click primary action", x: 24, y: 34, tone: "primary" },
      { label: "Watch status", x: 50, y: 58, tone: "muted" },
      { label: "Review output", x: 76, y: 34, tone: "success" }
    ],
    note: "If something fails, copy the visible request ID or terminal log."
  }
];

const pageSlides: Record<string, HelpSlide[]> = {
  "Clearflow": [
    {
      title: "Understand the product in one flow",
      body: "Clearflow turns processor payouts, bank deposits, fees, refunds, and unexplained differences into one daily close workflow.",
      nodes: [
        { label: "Connect data", x: 16, y: 42, tone: "primary" },
        { label: "Match payouts", x: 40, y: 62, tone: "muted" },
        { label: "Resolve breaks", x: 64, y: 38, tone: "warn" },
        { label: "Trust cash", x: 86, y: 58, tone: "success" }
      ],
      note: "Use the guided demo when you want sample data without connecting real providers."
    },
    {
      title: "Know who it helps",
      body: "The product is for operators who need a fast answer to whether card sales, delivery payouts, and bank deposits line up.",
      nodes: [
        { label: "Owner", x: 18, y: 40, tone: "primary" },
        { label: "Manager", x: 44, y: 60, tone: "muted" },
        { label: "Accountant", x: 70, y: 40, tone: "success" }
      ],
      note: "The app is intentionally about cash close, not general bookkeeping."
    }
  ],
  "Daily revenue close": [
    {
      title: "Read the owner answer first",
      body: "This dashboard tells an owner whether yesterday's sales, delivery payouts, refunds, fees, and deposits line up.",
      nodes: [
        { label: "Sales", x: 14, y: 35, tone: "primary" },
        { label: "Fees + refunds", x: 38, y: 58, tone: "warn" },
        { label: "Bank cash", x: 62, y: 35, tone: "success" },
        { label: "Close verdict", x: 86, y: 58, tone: "primary" }
      ],
      note: "Start here before jumping into detailed ledgers."
    },
    {
      title: "Follow the daily workflow",
      body: "Load provider data, run reconciliation, resolve breaks, then review forecasted cash.",
      nodes: [
        { label: "1. Load data", x: 16, y: 46, tone: "primary" },
        { label: "2. Reconcile", x: 40, y: 46, tone: "primary" },
        { label: "3. Resolve", x: 64, y: 46, tone: "warn" },
        { label: "4. Forecast", x: 88, y: 46, tone: "success" }
      ],
      note: "The dashboard is the summary; Reconcile is where operator action happens."
    },
    {
      title: "Use forecast as the decision layer",
      body: "Cash forecast shows where today's close puts the business over the next 30 days.",
      nodes: [
        { label: "Today", x: 18, y: 70, tone: "muted" },
        { label: "Expected inflows", x: 42, y: 38, tone: "success" },
        { label: "Known costs", x: 62, y: 62, tone: "warn" },
        { label: "30-day cash", x: 84, y: 28, tone: "primary" }
      ],
      note: "X-axis is days; Y-axis is projected cash."
    }
  ],
  "Home base": [
    {
      title: "Read the close health",
      body: "Home base summarizes connected data, open breaks, cash, and the next best action.",
      nodes: [
        { label: "Cash", x: 20, y: 35, tone: "success" },
        { label: "Data health", x: 45, y: 62, tone: "muted" },
        { label: "Open breaks", x: 70, y: 35, tone: "warn" },
        { label: "Next action", x: 86, y: 64, tone: "primary" }
      ]
    }
  ],
  "Connect data": [
    {
      title: "Connect bank data safely",
      body: "Plaid handles bank login. Clearflow receives authorized transaction data and stores provider tokens only on the backend.",
      nodes: [
        { label: "User bank", x: 14, y: 50, tone: "muted" },
        { label: "Plaid Link", x: 38, y: 32, tone: "primary" },
        { label: "Backend token vault", x: 62, y: 60, tone: "warn" },
        { label: "Bank ledger", x: 86, y: 42, tone: "success" }
      ],
      note: "Clearflow should never ask for bank usernames or passwords."
    },
    {
      title: "Use CSV as a fallback",
      body: "CSV imports let teams validate the reconciliation flow before live providers are approved.",
      nodes: [
        { label: "Download sample", x: 22, y: 38, tone: "primary" },
        { label: "Upload export", x: 50, y: 62, tone: "muted" },
        { label: "Normalize rows", x: 78, y: 38, tone: "success" }
      ]
    }
  ],
  "Payout reconciliation": [
    {
      title: "Match expected cash to posted cash",
      body: "Reconciliation compares processor payouts against bank deposits by amount, date, and explanation.",
      nodes: [
        { label: "Stripe payout", x: 18, y: 36, tone: "primary" },
        { label: "Match engine", x: 50, y: 58, tone: "muted" },
        { label: "Bank deposit", x: 82, y: 36, tone: "success" }
      ],
      note: "A good match becomes evidence; a bad match becomes a break."
    },
    {
      title: "Resolve breaks with evidence",
      body: "Open exceptions are not just errors. They are investigation tasks with notes, status, and audit trail.",
      nodes: [
        { label: "Open break", x: 18, y: 45, tone: "warn" },
        { label: "Add note", x: 42, y: 62, tone: "primary" },
        { label: "Attach bank record", x: 66, y: 34, tone: "muted" },
        { label: "Resolved", x: 86, y: 56, tone: "success" }
      ]
    },
    {
      title: "Explain every payout",
      body: "The payout explanation breaks gross payments into fees, refunds, net payout, linked bank deposit, and warnings.",
      nodes: [
        { label: "Gross", x: 16, y: 38, tone: "primary" },
        { label: "Fees", x: 38, y: 62, tone: "warn" },
        { label: "Refunds", x: 60, y: 62, tone: "warn" },
        { label: "Net deposit", x: 84, y: 38, tone: "success" }
      ]
    }
  ],
  "Cash forecast": [
    {
      title: "Forecast operating runway",
      body: "The forecast projects cash forward from posted balances, expected inflows, known outflows, and recent settlement behavior.",
      nodes: [
        { label: "Starting cash", x: 14, y: 66, tone: "success" },
        { label: "Inflows", x: 38, y: 36, tone: "success" },
        { label: "Outflows", x: 62, y: 58, tone: "warn" },
        { label: "Ending cash", x: 86, y: 28, tone: "primary" }
      ],
      note: "X-axis means days. Y-axis means projected cash amount."
    },
    {
      title: "Act on anomalies and recommendations",
      body: "Use anomalies and recommendations to decide whether to review reconciliation, reduce spend, or preserve cash.",
      nodes: [
        { label: "Anomaly", x: 20, y: 42, tone: "warn" },
        { label: "Recommendation", x: 50, y: 62, tone: "primary" },
        { label: "Operator action", x: 80, y: 42, tone: "success" }
      ]
    }
  ],
  "Transaction ledger": [
    {
      title: "Inspect normalized activity",
      body: "The ledger is the searchable source record for merchants, categories, credits, debits, and descriptions.",
      nodes: [
        { label: "Search", x: 18, y: 38, tone: "primary" },
        { label: "Filter", x: 42, y: 60, tone: "muted" },
        { label: "Inspect row", x: 66, y: 38, tone: "success" },
        { label: "Categorize", x: 84, y: 62, tone: "primary" }
      ]
    }
  ],
  "Provider health": [
    {
      title: "Verify external connections",
      body: "Provider health shows whether Stripe and Plaid are connected, synced, and safe to use for reconciliation.",
      nodes: [
        { label: "Stripe", x: 20, y: 36, tone: "primary" },
        { label: "Plaid", x: 44, y: 60, tone: "success" },
        { label: "Webhook tester", x: 68, y: 36, tone: "warn" },
        { label: "Sync status", x: 86, y: 62, tone: "muted" }
      ],
      note: "Sandbox testers are local-only; production uses signed provider webhooks."
    }
  ],
  "Control center": [
    {
      title: "Prove the system worked",
      body: "Control center turns backend behavior into evidence: jobs, audit logs, metrics, idempotency, and webhook events.",
      nodes: [
        { label: "HTTP request", x: 14, y: 44, tone: "primary" },
        { label: "Queued job", x: 38, y: 64, tone: "muted" },
        { label: "Worker", x: 62, y: 36, tone: "success" },
        { label: "Audit log", x: 86, y: 58, tone: "warn" }
      ]
    },
    {
      title: "Use logs for debugging",
      body: "When you send logs back, request IDs and job IDs let us trace a user action across API, worker, and database state.",
      nodes: [
        { label: "Request ID", x: 22, y: 34, tone: "primary" },
        { label: "Job ID", x: 50, y: 62, tone: "muted" },
        { label: "Trace ID", x: 78, y: 34, tone: "success" }
      ]
    }
  ],
  "Team settings": [
    {
      title: "Manage access and production readiness",
      body: "Settings contains workspace controls, team roles, sessions, and launch checklist evidence.",
      nodes: [
        { label: "Workspace", x: 16, y: 36, tone: "primary" },
        { label: "Roles", x: 40, y: 62, tone: "muted" },
        { label: "Sessions", x: 64, y: 36, tone: "warn" },
        { label: "Readiness", x: 86, y: 62, tone: "success" }
      ]
    }
  ],
  "Portfolio": [
    {
      title: "Import investor context",
      body: "Portfolio tools normalize holdings and brokerage activity so Clearflow can show allocation, concentration, and cash exposure alongside operating cash.",
      nodes: [
        { label: "Holdings CSV", x: 16, y: 44, tone: "primary" },
        { label: "Brokerage activity", x: 42, y: 62, tone: "muted" },
        { label: "Allocation", x: 66, y: 38, tone: "success" },
        { label: "Risk view", x: 86, y: 58, tone: "warn" }
      ],
      note: "This is secondary to the revenue-close story, but useful for founders and operators tracking treasury exposure."
    }
  ],
  "Advisor": [
    {
      title: "Use computed guidance carefully",
      body: "Advisor turns cash-flow, reconciliation, and portfolio signals into rule-based recommendations an operator can review.",
      nodes: [
        { label: "Cash signal", x: 18, y: 38, tone: "success" },
        { label: "Break signal", x: 42, y: 62, tone: "warn" },
        { label: "Rule engine", x: 66, y: 38, tone: "muted" },
        { label: "Recommendation", x: 86, y: 62, tone: "primary" }
      ],
      note: "It explains what to inspect; it does not replace accounting judgment."
    }
  ],
  "Setup": [
    {
      title: "Pick a realistic workspace",
      body: "Onboarding sets the organization type, currency, and checklist state so the demo feels like an actual operator workflow.",
      nodes: [
        { label: "Business type", x: 22, y: 38, tone: "primary" },
        { label: "Readiness", x: 50, y: 62, tone: "warn" },
        { label: "Sample close", x: 78, y: 38, tone: "success" }
      ]
    }
  ],
  "Log in": [
    {
      title: "Choose real auth or sample workspace",
      body: "Login opens an existing workspace. Demo mode gives a safe sample path without production credentials.",
      nodes: [
        { label: "Credentials", x: 24, y: 42, tone: "primary" },
        { label: "Demo session", x: 50, y: 62, tone: "muted" },
        { label: "Workspace", x: 76, y: 42, tone: "success" }
      ]
    }
  ],
  "Create account": [
    {
      title: "Create the owner workspace",
      body: "Registration creates the owner user, default organization, membership, and authenticated session.",
      nodes: [
        { label: "Owner", x: 20, y: 38, tone: "primary" },
        { label: "Organization", x: 50, y: 62, tone: "muted" },
        { label: "Session", x: 80, y: 38, tone: "success" }
      ]
    }
  ]
};

const pageAliases: Record<string, string> = {
  "Start daily revenue close": "Setup",
  "Reconcile payouts": "Payout reconciliation",
  "Integrations": "Provider health",
  "Operations": "Control center",
  "Settings": "Team settings",
  "Transactions": "Transaction ledger",
  "Portfolio": "Portfolio",
  "Advisor": "Advisor"
};

function slidesForPage(page: string) {
  return pageSlides[pageAliases[page] || page] || defaultSlides;
}

export function HelpFlow({ page }: { page: string }) {
  const slides = useMemo(() => slidesForPage(page), [page]);
  const [open, setOpen] = useState(false);
  const [index, setIndex] = useState(0);
  const current = slides[Math.min(index, slides.length - 1)] || defaultSlides[0];

  useEffect(() => {
    setOpen(false);
    setIndex(0);
  }, [page]);

  useEffect(() => {
    if (!open) return;
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") setOpen(false);
      if (event.key === "ArrowRight") setIndex((value) => Math.min(slides.length - 1, value + 1));
      if (event.key === "ArrowLeft") setIndex((value) => Math.max(0, value - 1));
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [open, slides.length]);

  function next() {
    setIndex((value) => Math.min(slides.length - 1, value + 1));
  }

  function previous() {
    setIndex((value) => Math.max(0, value - 1));
  }

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-ink/15 bg-white text-sm font-semibold text-ink/70 shadow-sm hover:bg-ink/[0.03] focus:outline-none focus:ring-2 focus:ring-[#5ea8ff]"
        aria-label={`Open ${page} help slideshow`}
        title={`Open ${page} help slideshow`}
      >
        ?
      </button>

      {open ? (
        <div className="fixed inset-0 z-50 overflow-y-auto bg-ink/55 px-4 py-4 backdrop-blur-sm sm:grid sm:place-items-center sm:py-6" role="dialog" aria-modal="true" aria-label={`${page} help slideshow`}>
          <div className="mx-auto w-full max-w-4xl rounded-md border border-ink/10 bg-[#fbfcf8] shadow-panel sm:max-h-[calc(100vh-2rem)] sm:overflow-y-auto">
            <div className="flex items-start justify-between gap-4 border-b border-ink/10 px-5 py-4">
              <div>
                <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">{page} guide</p>
                <h2 className="mt-1 text-xl font-semibold text-ink">{current.title}</h2>
              </div>
              <button
                type="button"
                onClick={() => setOpen(false)}
                className="rounded-md border border-ink/15 bg-white px-3 py-2 text-sm font-semibold text-ink/65 hover:bg-ink/[0.03]"
                aria-label="Close help slideshow"
              >
                Close
              </button>
            </div>

            <div className="grid gap-5 p-5 lg:grid-cols-[1.2fr_.8fr]">
              <SlideDiagram slide={current} />
              <div className="flex min-h-0 flex-col justify-between lg:min-h-80">
                <div>
                  <p className="text-base leading-7 text-ink/70">{current.body}</p>
                  {current.note ? <p className="mt-4 rounded-md border border-[#5ea8ff]/30 bg-[#e8f4ff] p-3 text-sm leading-6 text-ink/70">{current.note}</p> : null}
                  <div className="mt-5 grid gap-2">
                    {current.nodes.map((node, nodeIndex) => (
                      <div key={`${node.label}-${nodeIndex}`} className="flex items-center gap-3 rounded-md border border-ink/10 bg-white px-3 py-2">
                        <span className={`h-3 w-3 rounded-full ${nodeColor(node.tone)}`} />
                        <span className="text-sm font-medium text-ink">{node.label}</span>
                      </div>
                    ))}
                  </div>
                </div>
                <div className="mt-6">
                  <div className="mb-3 flex items-center justify-between text-xs text-ink/45">
                    <span>Slide {index + 1} of {slides.length}</span>
                    <span>Use arrows or buttons</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={previous}
                      disabled={index === 0}
                      className="rounded-md border border-ink/15 bg-white px-4 py-2 text-sm font-semibold text-ink disabled:cursor-not-allowed disabled:opacity-40"
                    >
                      Back
                    </button>
                    <button
                      type="button"
                      onClick={next}
                      disabled={index === slides.length - 1}
                      className="rounded-md bg-ink px-4 py-2 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:opacity-40"
                    >
                      Next
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
}

function SlideDiagram({ slide }: { slide: HelpSlide }) {
  return (
    <div className="relative min-h-72 overflow-hidden rounded-md border border-ink/10 bg-white sm:min-h-80">
      <div className="absolute inset-0 bg-[linear-gradient(rgba(23,33,27,.06)_1px,transparent_1px),linear-gradient(90deg,rgba(23,33,27,.06)_1px,transparent_1px)] bg-[size:44px_44px]" />
      <svg className="absolute inset-0 h-full w-full" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
        {slide.nodes.slice(1).map((node, index) => {
          const previous = slide.nodes[index];
          return <line key={`${previous.label}-${node.label}`} x1={previous.x} y1={previous.y} x2={node.x} y2={node.y} stroke="rgba(23,33,27,.22)" strokeWidth="0.8" strokeDasharray="3 2" />;
        })}
      </svg>
      {slide.nodes.map((node, index) => (
        <div
          key={`${node.label}-${index}`}
          className="absolute -translate-x-1/2 -translate-y-1/2"
          style={{ left: `${node.x}%`, top: `${node.y}%` }}
        >
          <div className={`grid h-16 w-16 place-items-center rounded-full border-4 border-white text-center text-xs font-bold leading-tight text-ink shadow-panel sm:h-20 sm:w-20 ${nodeColor(node.tone)}`}>
            {index + 1}
          </div>
          <div className="absolute left-1/2 top-full mt-2 w-24 -translate-x-1/2 rounded-md border border-ink/10 bg-[#fbfcf8] px-2 py-1 text-center text-[11px] font-semibold leading-tight text-ink shadow-sm sm:w-32 sm:text-xs">
            {node.label}
          </div>
        </div>
      ))}
    </div>
  );
}

function nodeColor(tone: HelpNode["tone"] = "primary") {
  if (tone === "success") return "bg-mint";
  if (tone === "warn") return "bg-gold";
  if (tone === "muted") return "bg-ink/10";
  return "bg-[#83c5ff]";
}
