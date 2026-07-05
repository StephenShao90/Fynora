"use client";

import { useState } from "react";
import { Card, Metric, Shell, money } from "@/components/Shell";
import { Header, Empty } from "@/components/Common";
import { api } from "@/lib/api";
import { useApi } from "@/hooks/useApi";

type Run = {
  id: string;
  status: string;
  matched_count: number;
  exception_count: number;
  started_at: string;
};

type Exception = {
  id: string;
  severity: string;
  title: string;
  explanation: string;
  status: string;
  created_at: string;
};

type Payment = {
  id: string;
  processor_payment_id: string;
  amount: number;
  status: string;
  description: string;
  occurred_at: string;
};

type Payout = {
  id: string;
  processor_payout_id: string;
  amount: number;
  status: string;
  expected_arrival_at: string;
};

export default function ReconciliationPage() {
  const [message, setMessage] = useState("");
  const [reloadKey, setReloadKey] = useState(0);
  const runs = useApi<Run[]>(`/reconciliation/runs?reload=${reloadKey}`, []);
  const exceptions = useApi<Exception[]>(`/reconciliation/exceptions?reload=${reloadKey}`, []);
  const payments = useApi<Payment[]>(`/payments?reload=${reloadKey}`, []);
  const payouts = useApi<Payout[]>(`/payouts?reload=${reloadKey}`, []);
  const cash = useApi<Record<string, number>>(`/cash-flow/summary?reload=${reloadKey}`, {});

  async function action(label: string, path: string) {
    setMessage(`${label}...`);
    try {
      const result = await api(path, { method: "POST", body: "{}" });
      setMessage(JSON.stringify(result, null, 2));
      setReloadKey((value) => value + 1);
    } catch (err) {
      setMessage((err as Error).message);
    }
  }

  async function resolveException(id: string) {
    await api(`/reconciliation/exceptions/${id}`, { method: "PATCH", body: JSON.stringify({ status: "resolved" }) });
    setReloadKey((value) => value + 1);
  }

  return (
    <Shell>
      <Header title="Reconciliation" subtitle="Match processor payouts to bank deposits and surface operational exceptions." />
      <div className="grid gap-4 md:grid-cols-4">
        <Metric label="Cash balance" value={money(cash.data.cash_balance)} />
        <Metric label="Processor fees" value={money(cash.data.fees)} tone="warn" />
        <Metric label="Refunds" value={money(cash.data.refunds)} tone="warn" />
        <Metric label="Open exceptions" value={`${exceptions.data.filter((item) => item.status === "open").length}`} tone={exceptions.data.length ? "warn" : "good"} />
      </div>

      <div className="mt-5 grid gap-5 lg:grid-cols-[.9fr_1.1fr]">
        <Card title="Workflow">
          <div className="flex flex-wrap gap-3">
            <button onClick={() => action("Syncing Stripe sample data", "/sync/stripe")} className="rounded-md bg-ink px-4 py-2 text-sm font-semibold text-white">
              Sync Stripe sample
            </button>
            <button onClick={() => action("Syncing bank sample data", "/sync/bank")} className="rounded-md bg-moss px-4 py-2 text-sm font-semibold text-white">
              Sync bank sample
            </button>
            <button onClick={() => action("Running reconciliation", "/reconciliation/runs")} className="rounded-md bg-gold px-4 py-2 text-sm font-semibold text-ink">
              Run reconciliation
            </button>
          </div>
          {message ? <pre className="mt-4 max-h-72 overflow-auto rounded-md bg-ink p-3 text-xs text-white">{message}</pre> : <p className="mt-4 text-sm text-ink/55">Start with Stripe sample, bank sample, then run reconciliation.</p>}
        </Card>

        <Card title="Latest runs">
          {runs.data.length ? (
            <table className="w-full text-left text-sm">
              <thead className="text-ink/50"><tr><th className="py-2">Started</th><th>Status</th><th>Matches</th><th>Exceptions</th></tr></thead>
              <tbody>
                {runs.data.slice(0, 5).map((run) => (
                  <tr key={run.id} className="border-t border-ink/10">
                    <td className="py-3">{run.started_at?.slice(0, 19).replace("T", " ")}</td>
                    <td>{run.status}</td>
                    <td>{run.matched_count}</td>
                    <td>{run.exception_count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : <Empty text="No reconciliation runs yet." />}
        </Card>
      </div>

      <div className="mt-5 grid gap-5 xl:grid-cols-2">
        <Card title="Open exceptions">
          {exceptions.data.length ? (
            <div className="grid gap-3">
              {exceptions.data.map((item) => (
                <div key={item.id} className="rounded-md border border-coral/30 bg-coral/10 p-3">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="font-medium">{item.title}</p>
                      <p className="mt-1 text-sm text-ink/65">{item.explanation}</p>
                      <p className="mt-2 text-xs uppercase text-ink/45">{item.severity} · {item.status}</p>
                    </div>
                    {item.status === "open" ? <button onClick={() => resolveException(item.id)} className="rounded-md border border-ink/15 px-3 py-1 text-xs">Resolve</button> : null}
                  </div>
                </div>
              ))}
            </div>
          ) : <Empty text="No exceptions yet." />}
        </Card>

        <Card title="Payouts">
          {payouts.data.length ? (
            <table className="w-full text-left text-sm">
              <tbody>{payouts.data.map((payout) => <tr key={payout.id} className="border-t border-ink/10"><td className="py-3">{payout.processor_payout_id}</td><td>{payout.status}</td><td className="text-right">{money(payout.amount)}</td></tr>)}</tbody>
            </table>
          ) : <Empty text="No payouts yet." />}
        </Card>

        <Card title="Payments">
          {payments.data.length ? (
            <table className="w-full text-left text-sm">
              <tbody>{payments.data.slice(0, 8).map((payment) => <tr key={payment.id} className="border-t border-ink/10"><td className="py-3">{payment.description}</td><td>{payment.status}</td><td className="text-right">{money(payment.amount)}</td></tr>)}</tbody>
            </table>
          ) : <Empty text="No payments yet." />}
        </Card>
      </div>
    </Shell>
  );
}
