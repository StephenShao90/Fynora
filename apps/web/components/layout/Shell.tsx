"use client";

import Link from "next/link";
import { useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";
import { GuideMarker, type Guide } from "@/components/help";
import { activeDemoScenario, isDemoFallbackMode, setDemoScenario } from "@/lib/api";

const primaryNav = [
  ["/dashboard", "Home base"],
  ["/imports", "Connect data"],
  ["/reconciliation", "Reconcile payouts"],
  ["/insights", "Cash forecast"],
  ["/transactions", "Transaction ledger"]
];

const advancedNav = [
  ["/integrations", "Provider health"],
  ["/ops", "Control center"],
  ["/settings", "Team settings"]
];

const nav = [...primaryNav, ...advancedNav];

export function Shell({ children }: { children: React.ReactNode }) {
  const path = usePathname();
  const router = useRouter();
  const scenario = activeDemoScenario();
  useEffect(() => {
    nav.forEach(([href]) => router.prefetch(href));
  }, [router]);
  function signOut() {
    localStorage.removeItem("clearflow_token");
    localStorage.removeItem("fynora_token");
    router.replace("/");
  }
  return (
    <div className="min-h-screen bg-[#f4f6f2]">
      <aside className="fixed inset-y-0 left-0 hidden w-72 border-r border-ink/10 bg-[#fbfcf8] p-6 lg:block">
        <Link href="/dashboard" className="text-2xl font-semibold text-ink">Clearflow</Link>
        <p className="mt-2 max-w-52 text-sm leading-6 text-ink/55">A simple close checklist for Stripe payouts, bank deposits, cash breaks, and proof.</p>
        <div className="mt-6 rounded-md border border-ink/10 bg-white p-3">
          <p className="text-xs uppercase tracking-wide text-ink/40">Workspace</p>
          <p className="mt-1 truncate text-sm font-medium text-ink">{scenario.name}</p>
          {isDemoFallbackMode() ? <p className="mt-2 inline-flex rounded bg-gold/25 px-2 py-1 text-xs font-medium text-ink/70">Demo mode · sample data</p> : null}
          <label className="mt-3 block text-xs font-medium uppercase tracking-wide text-ink/40" htmlFor="demo-scenario">Scenario</label>
          <select
            id="demo-scenario"
            defaultValue={scenario.id}
            onChange={(event) => setDemoScenario(event.target.value)}
            className="mt-1 w-full rounded-md border border-ink/15 bg-white px-2 py-2 text-sm text-ink"
          >
            <option value="student_org">Student org</option>
            <option value="creator">Creator shop</option>
            <option value="saas">Small SaaS</option>
            <option value="nonprofit">Nonprofit</option>
          </select>
        </div>
        <nav className="mt-6 grid max-h-[calc(100vh-17rem)] gap-1 overflow-y-auto pr-1">
          <p className="px-3 pb-1 text-xs font-semibold uppercase tracking-wide text-ink/35">Daily workflow</p>
          {primaryNav.map(([href, label]) => (
            <Link key={href} href={href} className={`rounded-md px-3 py-2.5 text-sm font-medium ${path === href ? "bg-ink text-white" : "text-ink/65 hover:bg-ink/5 hover:text-ink"}`}>
              {label}
            </Link>
          ))}
          <p className="mt-4 px-3 pb-1 text-xs font-semibold uppercase tracking-wide text-ink/35">Advanced</p>
          {advancedNav.map(([href, label]) => (
            <Link key={href} href={href} className={`rounded-md px-3 py-2.5 text-sm font-medium ${path === href ? "bg-ink text-white" : "text-ink/55 hover:bg-ink/5 hover:text-ink"}`}>
              {label}
            </Link>
          ))}
        </nav>
        <button onClick={signOut} className="absolute bottom-6 left-6 rounded-md border border-ink/15 bg-white px-3 py-2 text-sm text-ink/65 hover:bg-ink/[0.03]">Log out</button>
      </aside>
      <main className="lg:pl-72">
        <header className="sticky top-0 z-30 border-b border-ink/10 bg-[#fbfcf8]/95 px-4 py-3 backdrop-blur lg:hidden">
          <div className="flex items-center justify-between gap-3">
            <Link href="/dashboard" className="text-lg font-semibold text-ink">Clearflow</Link>
            <button onClick={signOut} className="rounded-md border border-ink/15 bg-white px-3 py-2 text-xs text-ink/65">Log out</button>
          </div>
          <div className="mt-3 grid gap-2 sm:grid-cols-2">
            <label className="grid gap-1 text-xs font-medium uppercase tracking-wide text-ink/45" htmlFor="mobile-nav">
              Page
              <select
                id="mobile-nav"
                value={path}
                onChange={(event) => router.push(event.target.value)}
                className="rounded-md border border-ink/15 bg-white px-2 py-2 text-sm normal-case tracking-normal text-ink"
              >
                <optgroup label="Daily workflow">
                  {primaryNav.map(([href, label]) => <option key={href} value={href}>{label}</option>)}
                </optgroup>
                <optgroup label="Advanced">
                  {advancedNav.map(([href, label]) => <option key={href} value={href}>{label}</option>)}
                </optgroup>
              </select>
            </label>
            <label className="grid gap-1 text-xs font-medium uppercase tracking-wide text-ink/45" htmlFor="mobile-scenario">
              Scenario
              <select
                id="mobile-scenario"
                defaultValue={scenario.id}
                onChange={(event) => setDemoScenario(event.target.value)}
                className="rounded-md border border-ink/15 bg-white px-2 py-2 text-sm normal-case tracking-normal text-ink"
              >
                <option value="student_org">Student org</option>
                <option value="creator">Creator shop</option>
                <option value="saas">Small SaaS</option>
                <option value="nonprofit">Nonprofit</option>
              </select>
            </label>
          </div>
        </header>
        <div key={path} className="app-page-surface mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-10">{children}</div>
      </main>
    </div>
  );
}

export function Card({ title, children, guide }: { title?: string; children: React.ReactNode; guide?: Guide }) {
  return (
    <section className="rounded-md border border-ink/10 bg-white p-5 shadow-sm">
      {title || guide ? (
        <div className="mb-4 flex items-center justify-between gap-3">
          {title ? <h2 className="text-xs font-semibold uppercase tracking-wide text-ink/45">{title}</h2> : <span />}
          {guide ? <GuideMarker guide={guide} /> : null}
        </div>
      ) : null}
      {children}
    </section>
  );
}

export function Metric({ label, value, tone = "neutral" }: { label: string; value: string; tone?: "neutral" | "good" | "warn" }) {
  const color = tone === "good" ? "text-moss" : tone === "warn" ? "text-coral" : "text-ink";
  return (
    <section className="rounded-md border border-ink/10 bg-white px-4 py-3 shadow-sm">
      <p className="text-sm text-ink/60">{label}</p>
      <p className={`mt-2 text-2xl font-semibold ${color}`}>{value}</p>
    </section>
  );
}

export function money(value = 0) {
  return new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", maximumFractionDigits: 0 }).format(value);
}
