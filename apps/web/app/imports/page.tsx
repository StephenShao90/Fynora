"use client";

import Script from "next/script";
import { useEffect, useState } from "react";
import { Card, Shell } from "@/components/Shell";
import { Header } from "@/components/Common";
import { api, upload } from "@/lib/api";

const uploads = [
  ["Transactions CSV", "/imports/transactions-csv", "sample_transactions.csv"],
  ["Holdings CSV", "/portfolio/import/holdings-csv", "sample_holdings.csv"],
  ["Portfolio transactions CSV", "/portfolio/import/transactions-csv", "sample_portfolio_transactions.csv"]
];

type PlaidHandler = {
  open: () => void;
  exit: () => void;
};

type PlaidConnection = {
  id: string;
  institution_name: string;
  last_synced_at?: string;
  created_at: string;
};

declare global {
  interface Window {
    Plaid?: {
      create: (config: {
        token: string;
        onSuccess: (publicToken: string, metadata: unknown) => void;
        onExit?: (error: unknown, metadata: unknown) => void;
      }) => PlaidHandler;
    };
  }
}

export default function Imports() {
  const [result, setResult] = useState("");
  const [connections, setConnections] = useState<PlaidConnection[]>([]);
  const [plaidReady, setPlaidReady] = useState(false);
  const [busy, setBusy] = useState("");

  useEffect(() => {
    refreshConnections();
  }, []);

  async function send(path: string, file?: File) {
    if (!file) return;
    const res = await upload(path, file);
    setResult(JSON.stringify(res, null, 2));
  }

  async function refreshConnections() {
    try {
      setConnections(await api<PlaidConnection[]>("/connections"));
    } catch {
      setConnections([]);
    }
  }

  async function connectPlaid() {
    if (!window.Plaid || !plaidReady) {
      setResult("Plaid Link is still loading. Try again in a moment.");
      return;
    }
    setBusy("Creating secure Plaid link...");
    try {
      const link = await api<{ link_token: string }>("/connections/plaid/link-token", { method: "POST", body: "{}" });
      const handler = window.Plaid.create({
        token: link.link_token,
        onSuccess: async (publicToken) => {
          setBusy("Exchanging token and syncing transactions...");
          const exchange = await api("/connections/plaid/exchange-public-token", {
            method: "POST",
            body: JSON.stringify({ public_token: publicToken })
          });
          const sync = await api("/connections/plaid/sync-transactions", { method: "POST", body: "{}" });
          setResult(JSON.stringify({ exchange, sync }, null, 2));
          await refreshConnections();
          setBusy("");
        },
        onExit: (error) => {
          setBusy("");
          if (error) setResult(JSON.stringify(error, null, 2));
        }
      });
      handler.open();
    } catch (err) {
      setBusy("");
      setResult((err as Error).message);
    }
  }

  async function syncPlaid() {
    setBusy("Syncing Plaid transactions...");
    try {
      const sync = await api("/connections/plaid/sync-transactions", { method: "POST", body: "{}" });
      setResult(JSON.stringify(sync, null, 2));
      await refreshConnections();
    } catch (err) {
      setResult((err as Error).message);
    } finally {
      setBusy("");
    }
  }

  return (
    <Shell>
      <Script src="https://cdn.plaid.com/link/v2/stable/link-initialize.js" onLoad={() => setPlaidReady(true)} />
      <Header title="Imports" subtitle="Upload bank transactions and brokerage CSV exports without sharing credentials." />
      <div className="mb-5 grid gap-4 lg:grid-cols-[1.15fr_.85fr]">
        <Card title="Secure bank connection">
          <p className="text-sm leading-6 text-ink/65">
            Plaid handles bank login and MFA. Fynora only receives authorized transaction data and stores the Plaid access token on the backend.
          </p>
          <div className="mt-4 flex flex-wrap gap-3">
            <button onClick={connectPlaid} className="rounded-md bg-ink px-4 py-2 text-sm font-semibold text-white">
              Connect bank with Plaid
            </button>
            <button onClick={syncPlaid} className="rounded-md border border-ink/15 px-4 py-2 text-sm font-semibold text-ink">
              Sync connected banks
            </button>
          </div>
          {busy ? <p className="mt-3 text-sm text-moss">{busy}</p> : null}
        </Card>
        <Card title="Connections">
          {connections.length ? (
            <div className="grid gap-2">
              {connections.map((connection) => (
                <div key={connection.id} className="rounded-md bg-mint/60 p-3 text-sm">
                  <p className="font-medium">{connection.institution_name}</p>
                  <p className="text-ink/55">
                    {connection.last_synced_at ? `Last synced ${connection.last_synced_at.slice(0, 10)}` : "Connected, not synced yet"}
                  </p>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-ink/55">No bank connections yet.</p>
          )}
        </Card>
      </div>
      <div className="grid gap-4 lg:grid-cols-3">
        {uploads.map(([label, path, sample]) => (
          <Card key={path} title={label}>
            <input type="file" accept=".csv" onChange={(e) => send(path, e.target.files?.[0])} className="w-full text-sm" />
            <a className="mt-4 block text-sm text-moss" href={`/sample-data/${sample}`}>Download {sample}</a>
          </Card>
        ))}
      </div>
      {result ? <pre className="mt-5 overflow-auto rounded-lg bg-ink p-4 text-xs text-white">{result}</pre> : null}
    </Shell>
  );
}
