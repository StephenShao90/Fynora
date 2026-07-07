"use client";

import { useEffect, useState } from "react";
import { Empty, Header } from "@/components/Common";
import { Card, Shell } from "@/components/Shell";
import { disconnectStripe, getStripeConnectUrl, getStripeStatus, type StripeIntegrationStatus } from "@/lib/api";
import { useApi } from "@/hooks/useApi";

type PlaidConnection = { id: string; institution_name: string; products: string[]; last_synced_at?: string };

export default function IntegrationsPage() {
  const [stripe, setStripe] = useState<{ data?: StripeIntegrationStatus; loading: boolean; error: string }>({ loading: true, error: "" });
  const [busy, setBusy] = useState("");
  const plaid = useApi<PlaidConnection[]>("/connections", []);

  async function loadStripe() {
    setStripe((current) => ({ ...current, loading: true, error: "" }));
    try {
      setStripe({ data: await getStripeStatus(), loading: false, error: "" });
    } catch (err) {
      setStripe({ loading: false, error: (err as Error).message });
    }
  }

  useEffect(() => {
    loadStripe();
  }, []);

  async function connectStripe() {
    setBusy("Opening Stripe Connect...");
    try {
      const result = await getStripeConnectUrl();
      window.location.href = result.url;
    } catch (err) {
      setStripe((current) => ({ ...current, error: (err as Error).message }));
    } finally {
      setBusy("");
    }
  }

  async function disconnect() {
    setBusy("Disconnecting Stripe...");
    try {
      const result = await disconnectStripe();
      setStripe({ data: result, loading: false, error: "" });
    } catch (err) {
      setStripe((current) => ({ ...current, error: (err as Error).message }));
    } finally {
      setBusy("");
    }
  }

  return (
    <Shell>
      <Header title="Integrations" subtitle="Connect payment and bank providers, verify connection health, and keep external sync state visible." />

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
