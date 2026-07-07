"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { isDemoFallbackMode, logout } from "@/lib/api";

const nav = [
  ["/dashboard", "Dashboard"],
  ["/reconciliation", "Reconciliation"],
  ["/imports", "Imports"],
  ["/integrations", "Integrations"],
  ["/ops", "Ops"],
  ["/transactions", "Transactions"],
  ["/insights", "Cash Flow"],
  ["/settings", "Settings"]
];

export function Shell({ children }: { children: React.ReactNode }) {
  const path = usePathname();
  return (
    <div className="min-h-screen bg-[#f4f6f2]">
      <aside className="fixed inset-y-0 left-0 hidden w-72 border-r border-ink/10 bg-[#fbfcf8] p-6 lg:block">
        <Link href="/dashboard" className="text-2xl font-semibold text-ink">Clearflow</Link>
        <p className="mt-2 max-w-52 text-sm leading-6 text-ink/55">Payment reconciliation and cash visibility for small teams.</p>
        <div className="mt-6 rounded-md border border-ink/10 bg-white p-3">
          <p className="text-xs uppercase tracking-wide text-ink/40">Workspace</p>
          <p className="mt-1 truncate text-sm font-medium text-ink">Demo Organization</p>
          {isDemoFallbackMode() ? <p className="mt-2 inline-flex rounded bg-gold/25 px-2 py-1 text-xs font-medium text-ink/70">Demo mode · sample data</p> : null}
        </div>
        <nav className="mt-6 grid gap-1">
          {nav.map(([href, label]) => (
            <Link key={href} href={href} className={`rounded-md px-3 py-2.5 text-sm font-medium ${path === href ? "bg-ink text-white" : "text-ink/65 hover:bg-ink/5 hover:text-ink"}`}>
              {label}
            </Link>
          ))}
        </nav>
        <button onClick={logout} className="absolute bottom-6 left-6 rounded-md border border-ink/15 bg-white px-3 py-2 text-sm text-ink/65 hover:bg-ink/[0.03]">Log out</button>
      </aside>
      <main className="lg:pl-72">
        <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-10">{children}</div>
      </main>
    </div>
  );
}

export function Card({ title, children }: { title?: string; children: React.ReactNode }) {
  return (
    <section className="rounded-md border border-ink/10 bg-white p-5 shadow-sm">
      {title ? <h2 className="mb-4 text-xs font-semibold uppercase tracking-wide text-ink/45">{title}</h2> : null}
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
