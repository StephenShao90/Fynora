"use client";

import { FormEvent, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { GuideMarker } from "@/components/help";
import { api, setToken } from "@/lib/api";

export default function Register() {
  const router = useRouter();
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setBusy("Creating account...");
    setError("");
    const form = new FormData(e.currentTarget);
    try {
      const res = await api<{ token: string }>("/auth/register", { method: "POST", body: JSON.stringify({ email: form.get("email"), password: form.get("password") }) });
      setToken(res.token); router.push("/onboarding");
    } catch (err) {
      setError((err as Error).message);
      setBusy("");
    }
  }
  return (
    <main className="min-h-screen bg-[#f4f6f2] px-4 py-8 text-ink">
      <div className="mx-auto grid min-h-[calc(100vh-4rem)] max-w-5xl items-center gap-8 lg:grid-cols-[.9fr_1.1fr]">
        <section>
          <Link href="/" className="text-xl font-semibold">Clearflow</Link>
          <p className="mt-4 text-sm font-semibold uppercase tracking-wide text-moss">Create a reconciliation workspace</p>
          <h1 className="mt-3 max-w-xl text-4xl font-semibold leading-tight">Start with secure login, then connect provider data</h1>
          <p className="mt-4 max-w-xl text-sm leading-6 text-ink/60">
            A new account creates local authenticated access so you can set up an organization, sync Stripe-style payouts, connect bank data, and resolve settlement breaks.
          </p>
          <div className="mt-6 rounded-md border border-ink/10 bg-white p-4">
            <p className="text-xs font-semibold uppercase tracking-wide text-ink/45">After signup</p>
            <div className="mt-3 grid gap-2 text-sm text-ink/65">
              <p>1. Save the workspace profile.</p>
              <p>2. Connect Stripe/Plaid or run the local demo sync.</p>
              <p>3. Run payout reconciliation and review exceptions.</p>
            </div>
          </div>
        </section>

        <form onSubmit={submit} className="w-full rounded-md border border-ink/10 bg-white p-6 shadow-panel">
          <div className="flex items-center justify-between gap-3">
            <div>
              <h2 className="text-2xl font-semibold">Create account</h2>
              <p className="mt-1 text-sm text-ink/50">Register a local Clearflow user.</p>
            </div>
            <GuideMarker guide={{ number: 1, title: "Create account", body: "Register a local user, receive a JWT, and continue to onboarding so the workspace can be configured." }} />
          </div>
          <label className="mt-6 grid gap-1 text-sm font-medium">
            Email
            <input name="email" type="email" placeholder="you@company.com" className="rounded-md border border-ink/15 px-3 py-2 font-normal" required />
          </label>
          <label className="mt-3 grid gap-1 text-sm font-medium">
            Password
            <input name="password" type="password" placeholder="Use at least 8 characters" minLength={8} className="rounded-md border border-ink/15 px-3 py-2 font-normal" required />
          </label>
          {error ? <p className="mt-3 rounded-md border border-coral/25 bg-coral/5 p-3 text-sm text-coral">{error}</p> : null}
          <button disabled={Boolean(busy)} className="mt-5 w-full rounded-md bg-ink px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50">{busy || "Create account"}</button>
          <p className="mt-5 text-center text-sm text-ink/55">
            Already have an account? <Link href="/login" className="font-semibold text-moss">Log in</Link>
          </p>
        </form>
      </div>
    </main>
  );
}
