"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { logout } from "@/lib/api";

const nav = [
  ["/dashboard", "Dashboard"],
  ["/reconciliation", "Reconciliation"],
  ["/imports", "Imports"],
  ["/transactions", "Transactions"],
  ["/insights", "Cash Flow"],
  ["/settings", "Settings"]
];

export function Shell({ children }: { children: React.ReactNode }) {
  const path = usePathname();
  return (
    <div className="min-h-screen">
      <aside className="fixed inset-y-0 left-0 hidden w-64 border-r border-ink/10 bg-white/90 p-5 lg:block">
        <Link href="/dashboard" className="text-2xl font-semibold text-ink">Clearflow</Link>
        <p className="mt-2 text-sm text-ink/60">Payment reconciliation and cash-flow intelligence.</p>
        <nav className="mt-8 grid gap-1">
          {nav.map(([href, label]) => (
            <Link key={href} href={href} className={`rounded-md px-3 py-2 text-sm ${path === href ? "bg-mint text-moss" : "text-ink/70 hover:bg-ink/5"}`}>
              {label}
            </Link>
          ))}
        </nav>
        <button onClick={logout} className="absolute bottom-5 left-5 rounded-md border border-ink/15 px-3 py-2 text-sm text-ink/70">Log out</button>
      </aside>
      <main className="lg:pl-64">
        <div className="mx-auto max-w-7xl px-4 py-5 sm:px-6 lg:px-8">{children}</div>
      </main>
    </div>
  );
}

export function Card({ title, children }: { title?: string; children: React.ReactNode }) {
  return (
    <section className="rounded-lg border border-ink/10 bg-white p-5 shadow-panel">
      {title ? <h2 className="mb-4 text-sm font-semibold uppercase tracking-wide text-ink/60">{title}</h2> : null}
      {children}
    </section>
  );
}

export function Metric({ label, value, tone = "neutral" }: { label: string; value: string; tone?: "neutral" | "good" | "warn" }) {
  const color = tone === "good" ? "text-moss" : tone === "warn" ? "text-coral" : "text-ink";
  return (
    <Card>
      <p className="text-sm text-ink/60">{label}</p>
      <p className={`mt-2 text-3xl font-semibold ${color}`}>{value}</p>
    </Card>
  );
}

export function money(value = 0) {
  return new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", maximumFractionDigits: 0 }).format(value);
}
