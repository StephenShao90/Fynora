"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { api, setToken } from "@/lib/api";

export default function Login() {
  const router = useRouter();
  const [error, setError] = useState("");
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = new FormData(e.currentTarget);
    try {
      const res = await api<{ token: string }>("/auth/login", { method: "POST", body: JSON.stringify({ email: form.get("email"), password: form.get("password") }) });
      setToken(res.token); router.push("/dashboard");
    } catch (err) { setError((err as Error).message); }
  }
  return <AuthForm title="Log in" error={error} onSubmit={submit} button="Log in" />;
}

function AuthForm({ title, error, onSubmit, button }: { title: string; error: string; onSubmit: (e: FormEvent<HTMLFormElement>) => void; button: string }) {
  return (
    <main className="grid min-h-screen place-items-center bg-sky px-4">
      <form onSubmit={onSubmit} className="w-full max-w-md rounded-lg border border-ink/10 bg-white p-6 shadow-panel">
        <h1 className="text-2xl font-semibold">{title}</h1>
        <input name="email" type="email" placeholder="Email" className="mt-6 w-full rounded-md border border-ink/15 px-3 py-2" required />
        <input name="password" type="password" placeholder="Password" className="mt-3 w-full rounded-md border border-ink/15 px-3 py-2" required />
        {error ? <p className="mt-3 text-sm text-coral">{error}</p> : null}
        <button className="mt-5 w-full rounded-md bg-ink px-4 py-2 text-white">{button}</button>
      </form>
    </main>
  );
}
