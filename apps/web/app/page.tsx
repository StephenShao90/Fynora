"use client";

import { useRouter } from "next/navigation";
import { api, setToken } from "@/lib/api";

export default function Home() {
  const router = useRouter();
  async function tryDemo() {
    const res = await api<{ token: string }>("/auth/demo-token", { method: "POST", body: "{}" });
    setToken(res.token);
    router.push("/dashboard");
  }
  return (
    <main className="min-h-screen bg-[radial-gradient(circle_at_20%_0%,#dff5e8,transparent_35%),linear-gradient(135deg,#f7faf6,#dceefa)]">
      <section className="mx-auto grid min-h-screen max-w-7xl items-center gap-10 px-6 py-12 lg:grid-cols-[1.05fr_.95fr]">
        <div>
          <p className="text-sm font-semibold uppercase tracking-wide text-moss">Educational finance operating system</p>
          <h1 className="mt-4 max-w-4xl text-5xl font-semibold leading-tight text-ink sm:text-6xl">Fynora</h1>
          <p className="mt-5 max-w-2xl text-lg leading-8 text-ink/70">
            Spending intelligence, cash-flow planning, portfolio analytics, and grounded AI guidance for students and young professionals.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <button onClick={tryDemo} className="rounded-md bg-ink px-5 py-3 text-sm font-semibold text-white">Try Demo</button>
            <a href="/login" className="rounded-md border border-ink/15 bg-white px-5 py-3 text-sm font-semibold text-ink">Log in</a>
          </div>
          <p className="mt-6 max-w-2xl text-sm text-ink/55">Fynora is not a registered financial advisor. Projections are hypothetical and educational, not financial advice.</p>
        </div>
        <div className="rounded-lg border border-white/70 bg-white/75 p-5 shadow-panel backdrop-blur">
          <div className="grid gap-4">
            {[
              ["Net cash flow", "$742/mo", "Savings capacity after fixed and variable spending"],
              ["Portfolio value", "$21,024", "Holdings, allocation, and concentration risk"],
              ["Advisor plan", "$250 investing", "Grounded in cash flow and emergency fund gap"]
            ].map(([label, value, copy]) => (
              <div key={label} className="rounded-lg border border-ink/10 bg-white p-5">
                <p className="text-sm text-ink/60">{label}</p>
                <p className="mt-2 text-3xl font-semibold text-ink">{value}</p>
                <p className="mt-2 text-sm text-ink/55">{copy}</p>
              </div>
            ))}
          </div>
        </div>
      </section>
    </main>
  );
}
