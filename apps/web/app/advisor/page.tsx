"use client";

import { FormEvent, useState } from "react";
import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { Card, Shell, money } from "@/components/Shell";
import { Header } from "@/components/Common";
import { useToast } from "@/components/ToastProvider";
import { api } from "@/lib/api";
import { useApi } from "@/hooks/useApi";

export default function Advisor() {
  const { pushToast } = useToast();
  const plan = useApi<any>("/advisor/plan", {});
  const [projection, setProjection] = useState<any>(null);
  const [answer, setAnswer] = useState("");
  async function project(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    try {
      setProjection(await api("/advisor/investment-projection", { method: "POST", body: JSON.stringify({ monthly_contribution: Number(f.get("monthly")), initial_balance: Number(f.get("initial")), years: Number(f.get("years")), risk_tolerance: f.get("risk") }) }));
      pushToast({ tone: "success", title: "Projection updated", detail: "The chart now reflects your contribution assumptions." });
    } catch (err) {
      pushToast({ tone: "error", title: "Projection failed", detail: (err as Error).message });
    }
  }
  async function chat(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    try {
      const res = await api<{ answer: string }>("/advisor/chat", { method: "POST", body: JSON.stringify({ message: f.get("message") }) });
      setAnswer(res.answer);
      pushToast({ tone: "success", title: "Advisor answered", detail: "The response is shown below the question." });
    } catch (err) {
      pushToast({ tone: "error", title: "Advisor request failed", detail: (err as Error).message });
    }
  }
  return (
    <Shell>
      <Header title="Advisor" subtitle="Rule-based guidance grounded in computed cash-flow and portfolio data." />
      <div className="grid gap-5 xl:grid-cols-2">
        <Card title="Monthly allocation" guide={{ number: 1, title: "Monthly allocation", body: "Shows rule-based allocation guidance from the current cash/portfolio context. It is educational and not personalized financial advice." }}><pre className="text-sm">{JSON.stringify(plan.data.recommended_allocation || {}, null, 2)}</pre><p className="mt-3 text-sm text-ink/55">Educational estimate only. Clearflow does not provide individualized securities advice.</p></Card>
        <Card title="Emergency fund" guide={{ number: 2, title: "Emergency fund", body: "Shows estimated reserve gap. Use this before projections so users understand cash safety first." }}><p className="text-3xl font-semibold">{money(plan.data.emergency_fund?.gap)}</p><p className="mt-2 text-sm text-ink/60">{plan.data.emergency_fund?.explanation}</p></Card>
        <Card title="Investment projection" guide={{ number: 3, title: "Investment projection", body: "Enter monthly contribution, initial balance, years, and risk level, then simulate to see expected/lower/upper projection bands." }}>
          <form onSubmit={project} className="grid gap-3 sm:grid-cols-4"><input name="monthly" defaultValue="300" className="rounded-md border px-3 py-2" /><input name="initial" defaultValue="1000" className="rounded-md border px-3 py-2" /><input name="years" defaultValue="30" className="rounded-md border px-3 py-2" /><select name="risk" className="rounded-md border px-3 py-2"><option>moderate</option><option>conservative</option><option>aggressive</option></select><button className="rounded-md bg-ink px-4 py-2 text-white sm:col-span-4">Simulate</button></form>
          {projection ? <div className="mt-4 h-72"><ResponsiveContainer><LineChart data={projection.points}><XAxis dataKey="year" /><YAxis /><Tooltip /><Line dataKey="expected" stroke="#315846" /><Line dataKey="lower" stroke="#f07b63" /><Line dataKey="upper" stroke="#d6a53a" /></LineChart></ResponsiveContainer></div> : null}
        </Card>
        <Card title="Advisor chat" guide={{ number: 4, title: "Advisor chat", body: "Ask educational cash-flow or portfolio questions. The response is rule-based and grounded in available app data." }}>
          <form onSubmit={chat} className="flex gap-2"><input name="message" className="min-w-0 flex-1 rounded-md border px-3 py-2" defaultValue="How much can I invest each month without hurting my cash flow?" /><button className="rounded-md bg-moss px-4 py-2 text-white">Ask</button></form>
          {answer ? <p className="mt-4 rounded-md bg-mint p-4 text-sm leading-6">{answer}</p> : null}
        </Card>
      </div>
    </Shell>
  );
}
