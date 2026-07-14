"use client";

import { useEffect, useState } from "react";
import { Empty, Header } from "@/components/Common";
import { Card, Shell } from "@/components/Shell";
import { useToast } from "@/components/ToastProvider";
import { api, disconnectStripe, getStripeConnectUrl, getStripeStatus, type StripeIntegrationStatus } from "@/lib/api";
import { useApi } from "@/hooks/useApi";

type PlaidConnection = { id: string; institution_name: string; products: string[]; last_synced_at?: string };
type Organization = { id: string; name: string };

export default function IntegrationsPage() {
  const { pushToast } = useToast();
  const [stripe, setStripe] = useState<{ data?: StripeIntegrationStatus; loading: boolean; error: string }>({ loading: true, error: "" });
  const [busy, setBusy] = useState("");
  const [returnStatus, setReturnStatus] = useState<{ status: string; message: string }>({ status: "", message: "" });
  const [syncResult, setSyncResult] = useState("");
  const plaid = useApi<PlaidConnection[]>("/connections", []);
  const organizations = useApi<Organization[]>("/organizations", []);
  const orgId = organizations.data[0]?.id || "";

  async function loadStripe() {
    setStripe((current) => ({ ...current, loading: true, error: "" }));
    try {
      setStripe({ data: await getStripeStatus(), loading: false, error: "" });
    } catch (err) {
      setStripe({ loading: false, error: (err as Error).message });
    }
  }

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setReturnStatus({ status: params.get("stripe") || "", message: params.get("message") || "" });
    if (params.get("stripe") === "connected") {
      pushToast({ tone: "success", title: "Stripe connected", detail: "Processor connection status is refreshed below." });
    }
    if (params.get("stripe") === "error") {
      pushToast({ tone: "error", title: "Stripe connection failed", detail: params.get("message") || "Stripe returned without a usable connection." });
    }
    loadStripe();
  }, [pushToast]);

  async function connectStripe() {
    setBusy("Opening Stripe Connect...");
    try {
      const result = await getStripeConnectUrl();
      pushToast({ tone: "info", title: "Opening Stripe Connect", detail: "You will return to Integrations when Stripe finishes." });
      window.location.href = result.url;
    } catch (err) {
      setStripe((current) => ({ ...current, error: (err as Error).message }));
      pushToast({ tone: "error", title: "Could not open Stripe", detail: (err as Error).message });
    } finally {
      setBusy("");
    }
  }

  async function disconnect() {
    setBusy("Disconnecting Stripe...");
    try {
      const result = await disconnectStripe();
      setStripe({ data: result, loading: false, error: "" });
      pushToast({ tone: "success", title: "Stripe disconnected", detail: "Local provider connection state was cleared." });
    } catch (err) {
      setStripe((current) => ({ ...current, error: (err as Error).message }));
      pushToast({ tone: "error", title: "Could not disconnect Stripe", detail: (err as Error).message });
    } finally {
      setBusy("");
    }
  }

  async function syncProvider(label: string, path: string) {
    setBusy(`${label} sync running...`);
    try {
      const suffix = path.startsWith("/api/v1/") && orgId ? `?organizationId=${encodeURIComponent(orgId)}` : "";
      const result = await api<Record<string, unknown>>(`${path}${suffix}`, {
        method: "POST",
        headers: { "Idempotency-Key": `${label.toLowerCase().replace(/[^a-z0-9]+/g, "-")}-${Date.now()}` },
        body: "{}"
      });
      setSyncResult(JSON.stringify(result, null, 2));
      pushToast({ tone: "success", title: `${label} sync started`, detail: "Check Operations for background job status." });
    } catch (err) {
      setSyncResult((err as Error).message);
      pushToast({ tone: "error", title: `${label} sync failed`, detail: (err as Error).message });
    } finally {
      setBusy("");
    }
  }

  async function sendWebhook(label: string, path: string, body: Record<string, unknown>) {
    setBusy(`${label} webhook sending...`);
    try {
      const suffix = orgId ? `?organizationId=${encodeURIComponent(orgId)}` : "";
      const result = await api<Record<string, unknown>>(`${path}${suffix}`, {
        method: "POST",
        body: JSON.stringify(body)
      });
      setSyncResult(JSON.stringify(result, null, 2));
      pushToast({ tone: "success", title: `${label} webhook accepted`, detail: "Open Ops to verify webhook metrics, queued jobs, and audit logs." });
    } catch (err) {
      setSyncResult((err as Error).message);
      pushToast({ tone: "error", title: `${label} webhook failed`, detail: (err as Error).message });
    } finally {
      setBusy("");
    }
  }

  return (
    <Shell>
      <Header title="Integrations" subtitle="Connect payment and bank providers, verify connection health, and keep external sync state visible." />

      {returnStatus.status ? (
        <div className={`mb-4 rounded-md border px-4 py-3 text-sm ${returnStatus.status === "connected" ? "border-moss/30 bg-mint text-moss" : "border-coral/30 bg-coral/5 text-coral"}`}>
          {returnStatus.status === "connected" ? "Stripe connected. Connection status is refreshed below." : `Stripe connection failed: ${returnStatus.message || "unknown error"}`}
        </div>
      ) : null}

      <div className="grid gap-4 xl:grid-cols-2">
        <Card title="Stripe">
          {stripe.loading ? <div className="h-36 animate-pulse rounded-md bg-ink/[0.04]" /> : stripe.error ? <Empty text={`Could not load Stripe status: ${stripe.error}`} /> : (
            <div>
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div>
                  <Status connected={Boolean(stripe.data?.connected)} />
                  <p className="mt-3 text-lg font-semibold">{stripe.data?.displayName || "Stripe not connected"}</p>
                  <p className="mt-1 text-sm text-ink/55">{stripe.data?.accountId || "Use Stripe Connect to authorize processor data access."}</p>
                </div>
                {stripe.data?.connected ? (
                  <button onClick={disconnect} disabled={Boolean(busy)} className="rounded-md border border-coral/30 px-3 py-2 text-sm font-medium text-coral hover:bg-coral/5 disabled:opacity-50">Disconnect Stripe</button>
                ) : (
                  <button onClick={connectStripe} disabled={Boolean(busy)} className="rounded-md bg-ink px-3 py-2 text-sm font-medium text-white hover:bg-ink/90 disabled:opacity-50">Connect Stripe</button>
                )}
              </div>
              <div className="mt-5 grid gap-2 rounded-md bg-ink/[0.03] p-3 text-sm text-ink/60">
                <p><span className="font-medium text-ink/70">Connected:</span> {formatDate(stripe.data?.connectedAt)}</p>
                <p><span className="font-medium text-ink/70">Last sync:</span> {formatDate(stripe.data?.lastSyncAt)}</p>
                <p><span className="font-medium text-ink/70">Last error:</span> {stripe.data?.lastError || "None"}</p>
              </div>
              {busy ? <p className="mt-3 text-sm text-ink/50">{busy}</p> : null}
            </div>
          )}
        </Card>

        <Card title="Plaid">
          {plaid.loading ? <div className="h-36 animate-pulse rounded-md bg-ink/[0.04]" /> : plaid.error ? <Empty text={`Could not load Plaid connections: ${plaid.error}`} /> : plaid.data.length ? (
            <div className="grid gap-3">
              {plaid.data.map((conn) => (
                <div key={conn.id} className="rounded-md border border-ink/10 p-3">
                  <Status connected />
                  <p className="mt-2 font-semibold">{conn.institution_name || "Plaid institution"}</p>
                  <p className="mt-1 text-sm text-ink/55">{conn.products?.join(", ") || "transactions"} · Last sync {formatDate(conn.last_synced_at)}</p>
                </div>
              ))}
            </div>
          ) : <Empty text="No Plaid bank connection found. Connect a bank from Imports." />}
        </Card>
      </div>

      <div className="mt-4">
        <Card title="Provider sync controls">
          <div className="grid gap-3 md:grid-cols-3">
            <button onClick={() => syncProvider("Stripe", "/api/v1/sync/stripe")} className="rounded-md border border-ink/15 px-4 py-3 text-left text-sm font-semibold hover:bg-ink/[0.03]">
              Sync Stripe data
              <span className="mt-1 block text-xs font-normal text-ink/50">Queues processor payments, fees, refunds, and payouts.</span>
            </button>
            <button onClick={() => syncProvider("Plaid bank", "/connections/plaid/sync-transactions")} className="rounded-md border border-ink/15 px-4 py-3 text-left text-sm font-semibold hover:bg-ink/[0.03]">
              Sync Plaid bank
              <span className="mt-1 block text-xs font-normal text-ink/50">Refreshes authorized bank transactions.</span>
            </button>
            <button onClick={() => syncProvider("Plaid Investments", "/connections/plaid/sync-investments")} className="rounded-md border border-ink/15 px-4 py-3 text-left text-sm font-semibold hover:bg-ink/[0.03]">
              Sync investments sample
              <span className="mt-1 block text-xs font-normal text-ink/50">Imports mock holdings/activity into Portfolio.</span>
            </button>
          </div>
          {syncResult ? <pre className="mt-4 max-h-60 overflow-auto rounded-md bg-ink p-4 text-xs text-white">{syncResult}</pre> : null}
        </Card>
      </div>

      <div className="mt-4">
        <Card title="Sandbox webhook tester">
          <div className="grid gap-3 md:grid-cols-2">
            <button onClick={() => sendWebhook("Stripe", "/api/v1/webhooks/processors/stripe", { id: `evt_demo_${Date.now()}`, type: "payout.paid", data: { object: { id: "po_demo_webhook" } } })} className="rounded-md border border-ink/15 px-4 py-3 text-left text-sm font-semibold hover:bg-ink/[0.03]">
              Send Stripe payout webhook
              <span className="mt-1 block text-xs font-normal text-ink/50">Exercises webhook persistence, dedupe, metrics, outbox, and job queueing in dev.</span>
            </button>
            <button onClick={() => sendWebhook("Plaid", "/api/v1/webhooks/plaid", { webhook_type: "TRANSACTIONS", webhook_code: "SYNC_UPDATES_AVAILABLE", item_id: "item_demo_webhook", environment: "sandbox" })} className="rounded-md border border-ink/15 px-4 py-3 text-left text-sm font-semibold hover:bg-ink/[0.03]">
              Send Plaid transactions webhook
              <span className="mt-1 block text-xs font-normal text-ink/50">Exercises Plaid webhook handling and transaction-sync job queueing in local mode.</span>
            </button>
          </div>
          <p className="mt-3 text-xs leading-5 text-ink/50">
            If production webhook verification is enabled, use provider dashboards or signed webhook tooling instead of these local test buttons.
          </p>
        </Card>
      </div>

      <div className="mt-4">
        <Card title="Integration security">
          <div className="grid gap-3 text-sm leading-6 text-ink/60 md:grid-cols-3">
            <p>Stripe webhooks are verified with `Stripe-Signature` when `STRIPE_WEBHOOK_SECRET` is configured.</p>
            <p>Plaid webhook verification can be required with `PLAID_WEBHOOK_VERIFICATION=true`; local mock bypass is development-only.</p>
            <p>Provider tokens are protected server-side and never returned by status endpoints.</p>
          </div>
        </Card>
      </div>
    </Shell>
  );
}

function Status({ connected }: { connected: boolean }) {
  return <span className={`rounded px-2 py-1 text-xs font-semibold uppercase tracking-wide ${connected ? "bg-mint text-moss" : "bg-ink/5 text-ink/50"}`}>{connected ? "connected" : "not connected"}</span>;
}

function formatDate(value?: string) {
  if (!value || value.startsWith("0001-")) return "Not available";
  return new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }).format(new Date(value));
}
