"use client";

import { useEffect, useState } from "react";
import { Card, Shell } from "@/components/layout";
import { Header } from "@/components/layout";
import { useToast } from "@/components/layout";
import { api, upload } from "@/lib/api";

const uploads = [
  ["Bank activity CSV", "/imports/transactions-csv", "sample_transactions.csv"]
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

type PlaidLinkTokenResponse = {
  link_token?: string;
  token?: string;
  expiration?: string;
  demo_unavailable?: boolean;
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
  const { pushToast } = useToast();
  const [result, setResult] = useState("");
  const [connections, setConnections] = useState<PlaidConnection[]>([]);
  const [plaidStatus, setPlaidStatus] = useState<"loading" | "ready" | "error">("loading");
  const [busy, setBusy] = useState("");

  useEffect(() => {
    refreshConnections();
    loadPlaidScript()
      .then(() => setPlaidStatus("ready"))
      .catch((err) => {
        console.error("[plaid-link:load-error]", err);
        setPlaidStatus("error");
        setResult("Plaid Link could not load. Check your browser content blocker/network, then refresh this page.");
        pushToast({ tone: "error", title: "Plaid Link failed to load", detail: "Check content blockers or network access to cdn.plaid.com." });
      });
  }, [pushToast]);

  async function send(path: string, file?: File) {
    if (!file) return;
    try {
      const res = await upload(path, file);
      console.info("[clearflow-import]", res);
      setResult(`${file.name} imported. Reconciliation can now use these bank transactions.`);
      pushToast({ tone: "success", title: "CSV imported", detail: file.name });
    } catch (err) {
      pushToast({ tone: "error", title: "CSV import failed", detail: (err as Error).message });
    }
  }

  async function refreshConnections() {
    try {
      setConnections(await api<PlaidConnection[]>("/connections"));
    } catch {
      setConnections([]);
    }
  }

  async function connectPlaid() {
    if (plaidStatus === "loading") {
      setResult("Plaid Link is loading. Try again in a moment.");
      pushToast({ tone: "info", title: "Plaid Link is still loading", detail: "Try again in a moment." });
      return;
    }
    if (plaidStatus === "error" || !window.Plaid) {
      setResult("Plaid Link did not load. Refresh the page or disable browser content blockers for cdn.plaid.com.");
      pushToast({ tone: "error", title: "Plaid Link is unavailable", detail: "Refresh or disable browser content blockers for cdn.plaid.com." });
      return;
    }
    setBusy("Creating secure Plaid link...");
    try {
      const link = await api<PlaidLinkTokenResponse>("/connections/plaid/link-token", { method: "POST", body: "{}" });
      const linkToken = link.link_token || link.token || "";
      if (!linkToken) {
        setBusy("");
        setResult(link.demo_unavailable
          ? "Plaid Link requires the local API to be running with Plaid credentials. Start `make api`, then refresh this page."
          : "Plaid did not return a link token. Check the API logs for /connections/plaid/link-token.");
        return;
      }
      const handler = window.Plaid.create({
        token: linkToken,
        onSuccess: async (publicToken) => {
          setBusy("Exchanging token and syncing transactions...");
          const exchange = await api("/connections/plaid/exchange-public-token", {
            method: "POST",
            body: JSON.stringify({ public_token: publicToken })
          });
          const sync = await api("/connections/plaid/sync-transactions", { method: "POST", body: "{}" });
          console.info("[clearflow-plaid:connected]", { exchange, sync });
          setResult("Bank connected and transaction sync completed. Open Reconcile payouts to match deposits against processor payouts.");
          await refreshConnections();
          setBusy("");
          pushToast({ tone: "success", title: "Bank connected", detail: "Plaid token exchanged and transaction sync completed." });
        },
        onExit: (error) => {
          setBusy("");
          if (error) {
            console.warn("[clearflow-plaid:exit]", error);
            setResult("Plaid connection was not completed. Try again or use the sandbox connection for local testing.");
          }
        }
      });
      handler.open();
      setBusy("");
    } catch (err) {
      setBusy("");
      setResult((err as Error).message);
      pushToast({ tone: "error", title: "Plaid connection failed", detail: (err as Error).message });
    }
  }

  async function syncPlaid() {
    setBusy("Syncing Plaid transactions...");
    try {
      const sync = await api<Record<string, unknown>>("/connections/plaid/sync-transactions", { method: "POST", body: "{}" });
      console.info("[clearflow-plaid:sync]", sync);
      setResult(sync.message ? String(sync.message) : `Bank sync complete. Imported ${Number(sync.imported_count || 0).toLocaleString()} new transaction(s).`);
      await refreshConnections();
      pushToast({ tone: "success", title: "Bank sync complete", detail: "Connected bank transactions were refreshed." });
    } catch (err) {
      setResult((err as Error).message);
      pushToast({ tone: "error", title: "Bank sync failed", detail: (err as Error).message });
    } finally {
      setBusy("");
    }
  }

  async function createSandboxConnection() {
    setBusy("Creating Plaid Sandbox test connection...");
    try {
      const sandbox = await api<Record<string, unknown>>("/connections/plaid/sandbox-connect", {
        method: "POST",
        body: JSON.stringify({
          institution_id: "ins_109508",
          username: "user_transactions_dynamic",
          password: "pass_good"
        })
      });
      console.info("[clearflow-plaid:sandbox]", sandbox);
      setResult(`Sandbox bank connected. Imported ${Number(sandbox.imported_count || 0).toLocaleString()} transaction(s) for the demo workflow.`);
      await refreshConnections();
      pushToast({ tone: "success", title: "Sandbox bank connected", detail: "A Plaid Sandbox connection was created for local testing." });
    } catch (err) {
      setResult((err as Error).message);
      pushToast({ tone: "error", title: "Sandbox connection failed", detail: (err as Error).message });
    } finally {
      setBusy("");
    }
  }

  return (
    <Shell>
      <Header title="Connect data" subtitle="Bring in the two sources Clearflow needs: processor payouts and posted bank activity." />
      <div className="mb-5 grid gap-4 lg:grid-cols-[1.15fr_.85fr]">
        <Card title="Step 1: Bank activity" guide={{ number: 1, title: "Bank activity", body: "Start here to connect Plaid or create a sandbox bank. This supplies bank deposits and debits for reconciliation." }}>
          <p className="text-sm leading-6 text-ink/65">
            Plaid handles bank login and MFA. Clearflow receives authorized transaction data and stores Plaid tokens on the backend only.
          </p>
          <div className="mt-4 flex flex-wrap gap-3">
            <button
              onClick={connectPlaid}
              disabled={plaidStatus === "loading"}
              className="rounded-md bg-ink px-4 py-2 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-ink/40"
            >
              Connect bank with Plaid
            </button>
            <button onClick={syncPlaid} className="rounded-md border border-ink/15 px-4 py-2 text-sm font-semibold text-ink">
              Sync connected banks
            </button>
            <button onClick={createSandboxConnection} className="rounded-md border border-moss/30 bg-mint/70 px-4 py-2 text-sm font-semibold text-moss">
              Create sandbox test connection
            </button>
          </div>
          <p className={`mt-3 text-sm ${plaidStatus === "ready" ? "text-moss" : plaidStatus === "error" ? "text-coral" : "text-ink/50"}`}>
            Plaid Link: {plaidStatus === "ready" ? "ready" : plaidStatus === "error" ? "failed to load" : "loading"}
          </p>
          <p className="mt-2 text-xs leading-5 text-ink/50">
            For manual Sandbox Link testing, use username <span className="font-mono">user_good</span> and password <span className="font-mono">pass_good</span>. The sandbox button skips the bank login screen and creates a test connection directly.
          </p>
          {busy ? <p className="mt-3 text-sm text-moss">{busy}</p> : null}
        </Card>
        <Card title="Connection status" guide={{ number: 2, title: "Connection status", body: "This confirms which institutions are connected and whether they have synced. If empty, bank-based reconciliation will be incomplete." }}>
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
      <div className="grid gap-4 lg:grid-cols-[1fr_1fr]">
        {uploads.map(([label, path, sample], index) => (
          <Card key={path} title={label} guide={{ number: index + 3, title: label, body: `Use this CSV fallback when live provider data is unavailable. Download ${sample}, inspect the format, then upload a matching export.` }}>
            <input type="file" accept=".csv" onChange={(e) => send(path, e.target.files?.[0])} className="w-full text-sm" />
            <a className="mt-4 block text-sm text-moss" href={`/sample-data/${sample}`}>Download {sample}</a>
          </Card>
        ))}
        <Card title="Step 2: Processor payouts" guide={{ number: 4, title: "Processor payouts", body: "Processor payouts currently come from Stripe Connect or the mock Stripe sync. Use Reconciliation or Provider Health to run that sync." }}>
          <p className="text-sm leading-6 text-ink/60">
            Stripe-style payments, refunds, fees, and payouts are loaded from the provider sync path instead of CSV upload. That keeps the main workflow close to how a payment operations team would work.
          </p>
          <a className="mt-4 inline-flex rounded-md bg-ink px-4 py-2 text-sm font-semibold text-white" href="/reconciliation">Run reconciliation workflow</a>
        </Card>
      </div>
      {result ? (
        <div className="mt-5 rounded-md border border-moss/25 bg-mint/60 p-4 text-sm leading-6 text-moss">
          <p className="font-semibold">Latest data action</p>
          <p className="mt-1 text-ink/70">{result}</p>
        </div>
      ) : null}
    </Shell>
  );
}

function loadPlaidScript() {
  return new Promise<void>((resolve, reject) => {
    if (typeof window === "undefined") return resolve();
    if (window.Plaid) return resolve();

    const existing = document.querySelector<HTMLScriptElement>('script[src="https://cdn.plaid.com/link/v2/stable/link-initialize.js"]');
    const script = existing || document.createElement("script");
    let timeout: number;

    const complete = () => {
      window.clearTimeout(timeout);
      if (window.Plaid) resolve();
      else reject(new Error("Plaid script loaded but window.Plaid was not initialized"));
    };

    const fail = () => {
      window.clearTimeout(timeout);
      reject(new Error("Plaid script failed to load"));
    };

    script.addEventListener("load", complete, { once: true });
    script.addEventListener("error", fail, { once: true });

    if (!existing) {
      script.src = "https://cdn.plaid.com/link/v2/stable/link-initialize.js";
      script.async = true;
      document.head.appendChild(script);
    }

    timeout = window.setTimeout(() => {
      if (window.Plaid) resolve();
      else reject(new Error("Timed out waiting for Plaid Link"));
    }, 8000);

    if (window.Plaid) complete();
  });
}
