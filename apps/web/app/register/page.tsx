"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { GuideMarker } from "@/components/GuideMarker";
import { api, setToken } from "@/lib/api";

export default function Register() {
  const router = useRouter();
  const [error, setError] = useState("");
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = new FormData(e.currentTarget);
    try {
      const res = await api<{ token: string }>("/auth/register", { method: "POST", body: JSON.stringify({ email: form.get("email"), password: form.get("password") }) });
      setToken(res.token); router.push("/dashboard");
    } catch (err) { setError((err as Error).message); }
  }
  return (
    <main className="grid min-h-screen place-items-center bg-mint px-4">
      <form onSubmit={submit} className="w-full max-w-md rounded-lg border border-ink/10 bg-white p-6 shadow-panel">
        <div className="flex items-center justify-between gap-3">
          <h1 className="text-2xl font-semibold">Create account</h1>
          <GuideMarker guide={{ number: 1, title: "Create account", body: "Register a local user, receive a JWT, and start with a default organization workspace." }} />
        </div>
        <input name="email" type="email" placeholder="Email" className="mt-6 w-full rounded-md border border-ink/15 px-3 py-2" required />
        <input name="password" type="password" placeholder="Password" className="mt-3 w-full rounded-md border border-ink/15 px-3 py-2" required />
        {error ? <p className="mt-3 text-sm text-coral">{error}</p> : null}
        <button className="mt-5 w-full rounded-md bg-ink px-4 py-2 text-white">Register</button>
      </form>
    </main>
  );
}
