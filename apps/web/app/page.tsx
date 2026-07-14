"use client";

import { useRouter } from "next/navigation";
import { GuideMarker } from "@/components/GuideMarker";
import { api, setToken } from "@/lib/api";

export default function Home() {
  const router = useRouter();
  async function tryDemo() {
    const res = await api<{ token: string }>("/auth/demo-token", { method: "POST", body: "{}" });
    setToken(res.token);
    router.push("/onboarding");
  }
  return (
    <main className="min-h-screen bg-[#f4f6f2] text-ink">
      <section className="mx-auto grid min-h-screen max-w-7xl items-center gap-10 px-6 py-10 lg:grid-cols-[.9fr_1.1fr]">
        <div>
          <div className="mb-3 flex justify-start"><GuideMarker guide={{ number: 1, title: "Product entry", body: "Read the product promise, then click Try Demo to enter the guided Clearflow workflow with demo auth." }} /></div>
          <p className="text-sm font-semibold uppercase tracking-wide text-moss">Stripe-to-bank reconciliation for small operators</p>
          <h1 className="mt-4 max-w-4xl text-5xl font-semibold leading-tight tracking-normal text-ink sm:text-6xl">Clearflow</h1>
          <p className="mt-5 max-w-2xl text-lg leading-8 text-ink/70">
            Connect processor and bank data, explain every payout, resolve cash breaks with an audit trail, and forecast operating cash before month-end gets messy.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <button onClick={tryDemo} className="rounded-md bg-ink px-5 py-3 text-sm font-semibold text-white">Try Demo</button>
            <a href="/login" className="rounded-md border border-ink/15 bg-white px-5 py-3 text-sm font-semibold text-ink">Log in</a>
          </div>
          <div className="mt-8 grid gap-3 text-sm leading-6 text-ink/65">
            <p><span className="font-semibold text-ink">Built for:</span> student organizations, creator shops, nonprofits, and small SaaS teams that use Stripe and need bank-level cash confidence.</p>
            <p><span className="font-semibold text-ink">Backend story:</span> Go API, Postgres, Redis-ready jobs, idempotent financial writes, provider webhooks, audit logs, and OpenAPI docs.</p>
            <p><span className="font-semibold text-ink">Boundary:</span> Clearflow does not move money or store bank credentials. Plaid/Stripe handle authorization.</p>
          </div>
        </div>

        <div className="rounded-md border border-ink/10 bg-white p-4 shadow-panel">
          <div className="mb-3 flex justify-end"><GuideMarker guide={{ number: 2, title: "Product preview", body: "This preview shows the core workflow: cash metrics, exceptions, provider pipeline, and cash forecast before entering the app." }} /></div>
          <div className="border-b border-ink/10 pb-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Operations dashboard</p>
                <p className="mt-1 text-xl font-semibold">Payout settlement monitor</p>
              </div>
              <span className="rounded bg-mint px-2 py-1 text-xs font-semibold text-moss">live demo</span>
            </div>
          </div>
          <div className="mt-4 grid gap-3 md:grid-cols-3">
            <PreviewMetric label="Operating cash" value="$12,780" detail="posted bank cash" />
            <PreviewMetric label="Match rate" value="92%" detail="payouts reconciled" />
            <PreviewMetric label="Open breaks" value="2" detail="needs review" tone="warn" />
          </div>
          <div className="mt-4 grid gap-4 lg:grid-cols-[1fr_.85fr]">
            <div className="rounded-md border border-ink/10 p-4">
              <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Exception workbench</p>
              <div className="mt-3 grid gap-3">
                <PreviewBreak title="Likely payout amount mismatch" detail="Stripe payout is close to a bank deposit but differs by $11.00." />
                <PreviewBreak title="Unmatched bank deposit" detail="A bank credit has no known processor payout." />
              </div>
            </div>
            <div className="rounded-md border border-ink/10 p-4">
              <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Provider pipeline</p>
              <div className="mt-4 grid gap-2 text-sm">
                <PipelineRow label="Stripe sync" status="completed" />
                <PipelineRow label="Plaid bank sync" status="completed" />
                <PipelineRow label="Reconciliation job" status="queued" />
                <PipelineRow label="Audit log" status="written" />
              </div>
            </div>
          </div>
          <div className="mt-4 rounded-md bg-[#f7faf6] p-4">
            <p className="text-sm font-semibold">Cash forecast</p>
            <div className="mt-4 grid h-28 grid-cols-8 items-end gap-2 border-b border-l border-ink/15 pl-2">
              {[42, 52, 47, 58, 62, 55, 67, 71].map((height, index) => (
                <div key={index} className="rounded-t bg-moss/80" style={{ height: `${height}%` }} />
              ))}
            </div>
            <div className="mt-2 flex justify-between text-xs text-ink/45"><span>Today</span><span>30 days</span></div>
          </div>
        </div>
      </section>
    </main>
  );
}

function PreviewMetric({ label, value, detail, tone = "neutral" }: { label: string; value: string; detail: string; tone?: "neutral" | "warn" }) {
  return (
    <div className="rounded-md border border-ink/10 bg-[#fbfcf8] p-4">
      <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">{label}</p>
      <p className={`mt-2 text-2xl font-semibold ${tone === "warn" ? "text-coral" : "text-ink"}`}>{value}</p>
      <p className="mt-1 text-xs text-ink/50">{detail}</p>
    </div>
  );
}

function PreviewBreak({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="rounded-md border border-coral/25 bg-coral/5 p-3">
      <p className="text-sm font-semibold">{title}</p>
      <p className="mt-1 text-xs leading-5 text-ink/55">{detail}</p>
    </div>
  );
}

function PipelineRow({ label, status }: { label: string; status: string }) {
  return <div className="flex items-center justify-between border-b border-ink/10 py-2 last:border-0"><span>{label}</span><span className="rounded bg-mint px-2 py-1 text-xs font-semibold text-moss">{status}</span></div>;
}
