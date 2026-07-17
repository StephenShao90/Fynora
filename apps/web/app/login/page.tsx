"use client";

import { FormEvent, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { GuideMarker } from "@/components/help";
import { api, setToken } from "@/lib/api";

export default function Login() {
  const router = useRouter();
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setBusy("Signing in...");
    setError("");
    const form = new FormData(e.currentTarget);
    try {
      const res = await api<{ token: string }>("/auth/login", { method: "POST", body: JSON.stringify({ email: form.get("email"), password: form.get("password") }) });
      setToken(res.token); router.push("/dashboard");
    } catch (err) {
      setError((err as Error).message);
      setBusy("");
    }
  }
  async function tryDemo() {
    setBusy("Opening demo...");
    setError("");
    try {
      const res = await api<{ token: string }>("/auth/demo-token", { method: "POST", body: "{}" });
      setToken(res.token);
      router.push("/dashboard");
    } catch (err) {
      setError((err as Error).message);
      setBusy("");
    }
  }
  return <AuthForm title="Log in" error={error} busy={busy} onSubmit={submit} onDemo={tryDemo} button="Log in" />;
}

function AuthForm({ title, error, busy, onSubmit, onDemo, button }: { title: string; error: string; busy: string; onSubmit: (e: FormEvent<HTMLFormElement>) => void; onDemo: () => void; button: string }) {
  return (
    <main className="min-h-screen bg-[#f4f6f2] px-4 py-8 text-ink">
      <div className="mx-auto grid min-h-[calc(100vh-4rem)] max-w-5xl items-center gap-8 lg:grid-cols-[.9fr_1.1fr]">
        <section>
          <Link href="/" className="text-xl font-semibold">Clearflow</Link>
          <p className="mt-4 text-sm font-semibold uppercase tracking-wide text-moss">Secure workspace access</p>
          <h1 className="mt-3 max-w-xl text-4xl font-semibold leading-tight">{title} to reconcile payouts</h1>
          <p className="mt-4 max-w-xl text-sm leading-6 text-ink/60">
            Open your close workspace to review payouts, confirm bank deposits, resolve breaks, and decide what cash is safe to use.
          </p>
          <div className="mt-6 grid gap-3 rounded-md border border-ink/10 bg-white p-4 text-sm text-ink/65">
            <p><span className="font-semibold text-ink">Returning operator:</span> log in and continue the latest close checklist.</p>
            <p><span className="font-semibold text-ink">Just exploring:</span> use the demo path to see a complete payout-to-bank story with sample data.</p>
          </div>
        </section>

        <form onSubmit={onSubmit} className="w-full rounded-md border border-ink/10 bg-white p-6 shadow-panel">
          <div className="flex items-center justify-between gap-3">
            <div>
              <h2 className="text-2xl font-semibold">{title}</h2>
              <p className="mt-1 text-sm text-ink/50">Continue to your cash close workspace.</p>
            </div>
            <GuideMarker guide={{ number: 1, title: "Authentication", body: "Enter an existing email and password to open the workspace. Use the sample workspace when you only want to explore the flow." }} />
          </div>
          <label className="mt-6 grid gap-1 text-sm font-medium">
            Email
            <input name="email" type="email" placeholder="you@company.com" className="rounded-md border border-ink/15 px-3 py-2 font-normal" required />
          </label>
          <label className="mt-3 grid gap-1 text-sm font-medium">
            Password
            <input name="password" type="password" placeholder="Your password" className="rounded-md border border-ink/15 px-3 py-2 font-normal" required />
          </label>
          {error ? <p className="mt-3 rounded-md border border-coral/25 bg-coral/5 p-3 text-sm text-coral">{error}</p> : null}
          <button disabled={Boolean(busy)} className="mt-5 w-full rounded-md bg-ink px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50">{busy || button}</button>
          <button type="button" onClick={onDemo} disabled={Boolean(busy)} className="mt-3 w-full rounded-md border border-ink/15 px-4 py-2.5 text-sm font-semibold text-ink disabled:opacity-50">Open sample workspace</button>
          <p className="mt-5 text-center text-sm text-ink/55">
            New to Clearflow? <Link href="/register" className="font-semibold text-moss">Create an account</Link>
          </p>
        </form>
        </div>
    </main>
  );
}
