"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { Header } from "@/components/layout";
import { DemoPilot } from "@/components/demo";
import { GuideMarker } from "@/components/help";
import { Card, Shell } from "@/components/layout";
import { useToast } from "@/components/layout";
import { activeDemoScenario, api, setDemoScenario } from "@/lib/api";
import { useApi } from "@/hooks/useApi";
import type { OnboardingStatus } from "@/types";

type Organization = { id: string; name: string; type?: string; currency?: string; role?: string };
type StripeStatus = { connected: boolean; displayName?: string };
type PlaidConnection = { id: string; institution_name: string; last_synced_at?: string };

const businessTypes = [
  ["small_business", "Small business"],
  ["creator", "Creator / online seller"],
  ["student_organization", "Student organization"],
  ["nonprofit", "Nonprofit"],
  ["saas", "Small SaaS"]
];

export default function OnboardingPage() {
  const { pushToast } = useToast();
  const scenario = activeDemoScenario();
  const [name, setName] = useState(scenario.name);
  const [type, setType] = useState(scenario.type);
  const [currency, setCurrency] = useState(scenario.currency);
  const [busy, setBusy] = useState("");
  const orgs = useApi<Organization[]>("/organizations", []);
  const stripe = useApi<StripeStatus>("/api/v1/integrations/stripe/status", { connected: false });
  const plaid = useApi<PlaidConnection[]>("/connections", []);
  const payouts = useApi<unknown[]>("/payouts", []);
  const bank = useApi<unknown[]>("/bank-transactions", []);
  const setup = useApi<OnboardingStatus>("/api/v1/onboarding/status", { organization_id: "", selected_scenario: scenario.id, business_type: scenario.type, checklist: {} });

  const checklist = useMemo(() => {
    const readiness = setup.data.provider_readiness || setup.data.checklist || {};
    return [
      { label: "Workspace created", done: Boolean(readiness.workspace_created) || orgs.data.length > 0, href: "/settings", detail: orgs.data[0]?.name || "Create a company workspace" },
      { label: "Business profile chosen", done: Boolean(type && currency), href: "/onboarding", detail: `${labelForType(type)} · ${currency}` },
      { label: "Processor connected", done: Boolean(readiness.stripe_connected || readiness.processor_data_ready) || stripe.data.connected || payouts.data.length > 0, href: "/integrations", detail: stripe.data.connected ? stripe.data.displayName || "Stripe connected" : "Connect Stripe or run mock sync" },
      { label: "Bank data connected", done: Boolean(readiness.plaid_connected || readiness.bank_data_ready) || plaid.data.length > 0 || bank.data.length > 0, href: "/imports", detail: plaid.data[0]?.institution_name || "Connect Plaid or import bank CSV" },
      { label: "First reconciliation run", done: Boolean(readiness.reconciliation_ready), href: "/reconciliation", detail: "Match processor payouts to bank deposits" }
    ];
  }, [bank.data.length, currency, orgs.data, plaid.data, payouts.data.length, setup.data, stripe.data, type]);

  async function createWorkspace() {
    setBusy("Creating workspace...");
    try {
      setDemoScenario(typeToScenario(type), { name, type, currency });
      const created = await api<Organization>("/api/v1/organizations", {
        method: "POST",
        body: JSON.stringify({ name, type, currency })
      });
      await api<OnboardingStatus>("/api/v1/onboarding/status", {
        method: "PUT",
        body: JSON.stringify({
          selected_scenario: typeToScenario(type),
          business_type: type,
          checklist: {
            workspace_created: true,
            business_profile_chosen: true
          }
        })
      });
      pushToast({ tone: "success", title: "Workspace created", detail: created.name });
      window.location.reload();
    } catch (err) {
      pushToast({ tone: "error", title: "Could not create workspace", detail: (err as Error).message });
    } finally {
      setBusy("");
    }
  }

  async function switchScenario(id: string) {
    setDemoScenario(id);
    await api<OnboardingStatus>("/api/v1/onboarding/status", {
      method: "PUT",
      body: JSON.stringify({ selected_scenario: id, business_type: activeDemoScenario().type, checklist: setup.data.checklist || {} })
    }).catch(() => undefined);
    pushToast({ tone: "success", title: "Scenario switched", detail: "Demo data will refresh around the selected company profile." });
    window.location.reload();
  }

  return (
    <Shell>
      <Header title="Onboarding" subtitle="Set up the workspace, connect provider data, and reach a reliable reconciliation-ready state." />

      <div className="grid gap-4 xl:grid-cols-[.95fr_1.05fr]">
        <Card title="Workspace profile" guide={{ number: 1, title: "Workspace profile", body: "Set the organization name, business type, and currency. This is the customer/account context for all cash and reconciliation data." }}>
          <div className="grid gap-3">
            <label className="grid gap-1 text-sm font-medium text-ink">
              Organization name
              <input value={name} onChange={(event) => setName(event.target.value)} className="rounded-md border border-ink/15 px-3 py-2 font-normal" />
            </label>
            <label className="grid gap-1 text-sm font-medium text-ink">
              Business type
              <select value={type} onChange={(event) => setType(event.target.value)} className="rounded-md border border-ink/15 px-3 py-2 font-normal">
                {businessTypes.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
              </select>
            </label>
            <label className="grid gap-1 text-sm font-medium text-ink">
              Currency
              <select value={currency} onChange={(event) => setCurrency(event.target.value)} className="rounded-md border border-ink/15 px-3 py-2 font-normal">
                <option value="USD">USD</option>
                <option value="CAD">CAD</option>
              </select>
            </label>
          </div>
          <button onClick={createWorkspace} disabled={Boolean(busy)} className="mt-4 rounded-md bg-ink px-4 py-2 text-sm font-semibold text-white disabled:opacity-50">
            {busy || "Save workspace"}
          </button>
          <p className="mt-3 text-sm leading-6 text-ink/55">This creates a real organization when the API is running. In demo fallback mode it switches the sample company profile locally.</p>
        </Card>

        <Card title="Setup checklist" guide={{ number: 2, title: "Setup checklist", body: "Use this to see what is missing before Clearflow can reconcile reliably: workspace, processor data, bank data, and a first reconciliation run." }}>
          {setup.loading ? <p className="mb-3 text-sm text-ink/45">Loading saved setup status...</p> : null}
          <div className="grid gap-2">
            {checklist.map((item) => (
              <Link key={item.label} href={item.href} className={`rounded-md border p-3 transition hover:bg-ink/[0.03] ${item.done ? "border-moss/25 bg-mint/60" : "border-gold/35 bg-gold/10"}`}>
                <div className="flex items-center justify-between gap-3">
                  <p className="text-sm font-semibold">{item.label}</p>
                  <span className={`rounded px-2 py-1 text-xs font-semibold ${item.done ? "bg-white text-moss" : "bg-white text-ink/55"}`}>{item.done ? "done" : "needed"}</span>
                </div>
                <p className="mt-1 text-xs leading-5 text-ink/55">{item.detail}</p>
              </Link>
            ))}
          </div>
          <div className="mt-4 rounded-md bg-ink/[0.03] p-3 text-sm leading-6 text-ink/60">
            Setup status is persisted through the API when the local backend is running. Provider readiness is derived from actual Stripe/Plaid connections, imported processor data, bank records, and team membership.
          </div>
        </Card>
      </div>

      <div className="mt-4">
        <div className="mb-2 flex justify-end">
          <GuideMarker guide={{ number: 3, title: "Guided demo setup", body: "Click Run full demo setup to prepare the whole walkthrough: onboarding state, processor data, bank data, and reconciliation." }} />
        </div>
        <DemoPilot compact />
      </div>

      <div className="mt-4">
        <Card title="Demo company switcher" guide={{ number: 4, title: "Demo company switcher", body: "Switch the sample company profile so you can show Clearflow for student orgs, creators, SaaS teams, or nonprofits." }}>
          <div className="grid gap-3 md:grid-cols-4">
            <ScenarioButton active={scenario.id === "student_org"} title="Student org" detail="Dues, event tickets, sponsor payments, venue deposits." onClick={() => switchScenario("student_org")} />
            <ScenarioButton active={scenario.id === "creator"} title="Creator shop" detail="Stripe storefront payouts, refunds, platform tools." onClick={() => switchScenario("creator")} />
            <ScenarioButton active={scenario.id === "saas"} title="Small SaaS" detail="Subscription revenue, churn/refunds, software costs." onClick={() => switchScenario("saas")} />
            <ScenarioButton active={scenario.id === "nonprofit"} title="Nonprofit" detail="Donations, grant deposits, program spend." onClick={() => switchScenario("nonprofit")} />
          </div>
        </Card>
      </div>
    </Shell>
  );
}

function ScenarioButton({ active, title, detail, onClick }: { active: boolean; title: string; detail: string; onClick: () => void }) {
  return (
    <button onClick={onClick} className={`rounded-md border p-4 text-left transition hover:bg-ink/[0.03] ${active ? "border-moss bg-mint/70" : "border-ink/10 bg-white"}`}>
      <p className="text-sm font-semibold">{title}</p>
      <p className="mt-2 text-xs leading-5 text-ink/55">{detail}</p>
    </button>
  );
}

function labelForType(value: string) {
  return businessTypes.find(([id]) => id === value)?.[1] || value;
}

function typeToScenario(value: string) {
  if (value === "creator") return "creator";
  if (value === "saas") return "saas";
  if (value === "nonprofit") return "nonprofit";
  return "student_org";
}
