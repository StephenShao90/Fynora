"use client";

const CONFIGURED_API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL;
const API_BASE = CONFIGURED_API_BASE || "http://localhost:8080";
const DEMO_FALLBACK_ENABLED = !CONFIGURED_API_BASE;
const DEMO_TOKEN = "clearflow-demo-token";
const TOKEN_KEY = "clearflow_token";
const LEGACY_TOKEN_KEY = "fynora_token";
const DEMO_SCENARIO_KEY = "clearflow_demo_scenario";

export type DemoScenario = {
  id: string;
  name: string;
  type: string;
  currency: string;
  description: string;
  cashBalance: number;
  income: number;
  expenses: number;
  fees: number;
  refunds: number;
};

const demoScenarios: Record<string, DemoScenario> = {
  student_org: { id: "student_org", name: "Northside Student Association", type: "student_organization", currency: "USD", description: "Dues, events, sponsor payments, and venue deposits.", cashBalance: 2967.27, income: 3179.72, expenses: 512.45, fees: 258.12, refunds: 175 },
  creator: { id: "creator", name: "Maple Street Studio", type: "creator", currency: "USD", description: "Digital products, merch drops, refunds, and creator tools.", cashBalance: 8420.44, income: 12180.9, expenses: 2488.2, fees: 612.45, refunds: 660 },
  saas: { id: "saas", name: "LedgerLoop SaaS", type: "saas", currency: "USD", description: "Subscription payouts, churn refunds, cloud software costs.", cashBalance: 18440.12, income: 32200, expenses: 11840.2, fees: 1188.52, refunds: 1430 },
  nonprofit: { id: "nonprofit", name: "River Fund Collective", type: "nonprofit", currency: "USD", description: "Donations, grants, program expenses, and sponsor deposits.", cashBalance: 12780.65, income: 18800, expenses: 6990.4, fees: 420.75, refunds: 80 }
};

export function isDemoFallbackMode() {
  return DEMO_FALLBACK_ENABLED;
}

export function activeDemoScenario(): DemoScenario {
  if (typeof window === "undefined") return demoScenarios.student_org;
  const stored = localStorage.getItem(DEMO_SCENARIO_KEY) || "student_org";
  try {
    const parsed = JSON.parse(stored) as Partial<DemoScenario>;
    if (parsed?.id && parsed.name) return { ...demoScenarios[parsed.id] || demoScenarios.student_org, ...parsed };
  } catch {
    // Stored value may be an old plain scenario id.
  }
  return demoScenarios[stored] || demoScenarios.student_org;
}

export function setDemoScenario(id: string, override?: Partial<DemoScenario>) {
  const base = demoScenarios[id] || demoScenarios.student_org;
  if (typeof window !== "undefined") {
    localStorage.setItem(DEMO_SCENARIO_KEY, JSON.stringify({ ...base, ...override, id: base.id }));
  }
}

export function token() {
  if (typeof window === "undefined") return "";
  return localStorage.getItem(TOKEN_KEY) || localStorage.getItem(LEGACY_TOKEN_KEY) || "";
}

export function setToken(value: string) {
  localStorage.setItem(TOKEN_KEY, value);
  localStorage.removeItem(LEGACY_TOKEN_KEY);
}

export function logout() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(LEGACY_TOKEN_KEY);
  window.location.href = "/";
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const started = performance.now();
  const requestId = crypto.randomUUID();
  const jwt = token();
  const instantDemo = DEMO_FALLBACK_ENABLED && (path === "/auth/demo-token" || jwt === DEMO_TOKEN);
  if (instantDemo) {
    const fallback = demoResponse<T>(path, init);
    if (fallback !== undefined) {
      console.info("[clearflow-api:instant-demo]", { path, method: init.method || "GET", requestId });
      return fallback;
    }
  }
  const headers = new Headers(init.headers);
  headers.set("Content-Type", headers.get("Content-Type") || "application/json");
  headers.set("X-Request-ID", requestId);
  if (jwt) headers.set("Authorization", `Bearer ${jwt}`);
  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, { ...init, headers });
  } catch (err) {
    const fallback = DEMO_FALLBACK_ENABLED ? demoResponse<T>(path, init) : undefined;
    if (fallback !== undefined) {
      console.info("[clearflow-api:demo-fallback]", { path, method: init.method || "GET", requestId, reason: (err as Error).message });
      return fallback;
    }
    throw err;
  }
  const durationMs = Math.round(performance.now() - started);
  const responseRequestId = res.headers.get("X-Request-ID") || requestId;
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    const fallback = DEMO_FALLBACK_ENABLED ? demoResponse<T>(path, init) : undefined;
    if (fallback !== undefined && [404, 501, 503].includes(res.status)) {
      console.info("[clearflow-api:demo-fallback]", { path, method: init.method || "GET", requestId: responseRequestId, status: res.status, reason: body?.error?.message || "endpoint unavailable" });
      return fallback;
    }
    console.warn("[clearflow-api:error]", { path, status: res.status, durationMs, requestId: responseRequestId, message: body?.error?.message });
    throw new Error(body?.error?.message || `Request failed: ${res.status}`);
  }
  console.info("[clearflow-api]", { path, method: init.method || "GET", status: res.status, durationMs, requestId: responseRequestId });
  if (res.status === 204) return undefined as T;
  return res.json();
}

export async function upload<T>(path: string, file: File): Promise<T> {
  const started = performance.now();
  const requestId = crypto.randomUUID();
  const form = new FormData();
  form.append("file", file);
  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers: token() ? { Authorization: `Bearer ${token()}`, "X-Request-ID": requestId } : { "X-Request-ID": requestId },
    body: form
  });
  const durationMs = Math.round(performance.now() - started);
  const responseRequestId = res.headers.get("X-Request-ID") || requestId;
  if (!res.ok) {
    console.warn("[clearflow-api:error]", { path, status: res.status, durationMs, requestId: responseRequestId });
    throw new Error("Upload failed");
  }
  console.info("[clearflow-api]", { path, method: "POST", status: res.status, durationMs, requestId: responseRequestId });
  return res.json();
}

export type ForecastPoint = {
  date: string;
  projectedBalanceMinor: number;
  expectedInflowsMinor: number;
  expectedOutflowsMinor: number;
};

export type CashflowForecast = {
  organizationId: string;
  horizonDays: number;
  startingBalanceMinor: number;
  projectedEndingBalanceMinor: number;
  currency: string;
  series: ForecastPoint[];
  assumptions: string[];
  confidence: "low" | "medium" | "high" | string;
};

export type AnomalyInsight = {
  id: string;
  type: string;
  severity: "critical" | "high" | "medium" | "low" | string;
  title: string;
  description: string;
  resourceType: string;
  resourceId?: string;
  detectedAt: string;
  recommendedAction: string;
};

export type CashRecommendation = {
  type: string;
  priority: "critical" | "high" | "medium" | "low" | string;
  title: string;
  description: string;
  amountMinor?: number;
  currency: string;
};

export type SpendingCategory = {
  category: string;
  amountMinor: number;
  percentage: number;
  changeVsPreviousPeriod: number;
};

export type MerchantSpend = {
  merchant: string;
  amountMinor: number;
};

export type SpendingInsights = {
  totalSpendMinor: number;
  currency: string;
  categories: SpendingCategory[];
  topMerchants: MerchantSpend[];
  notes: string[];
};

export type PayoutExplanation = {
  payoutId: string;
  processor: string;
  grossAmountMinor: number;
  feesMinor: number;
  refundsMinor: number;
  netAmountMinor: number;
  currency: string;
  bankDeposit?: {
    id: string;
    amountMinor: number;
    postedAt: string;
  };
  summary: string;
  lineItems: unknown[];
  warnings: string[];
};

export type ReconciliationMatch = {
  id: string;
  processorPayoutId?: string;
  bankDepositId?: string;
  status: string;
  confidenceScore: number;
  amountDifferenceMinor: number;
  currency: string;
  reasons: string[];
  explanation: string;
};

export type StripeConnectUrlResponse = {
  url: string;
  state: string;
};

export type StripeIntegrationStatus = {
  connected: boolean;
  provider: "stripe" | string;
  accountId?: string;
  displayName?: string;
  connectedAt?: string;
  lastSyncAt?: string;
  lastError?: string;
};

export function getPayoutExplanation(payoutId: string) {
  return api<PayoutExplanation>(`/api/v1/payouts/${payoutId}/explanation`);
}

export function getCashflowForecast(horizonDays = 30) {
  return api<CashflowForecast>(`/api/v1/cashflow/forecast?horizonDays=${horizonDays}`);
}

export async function getAnomalies() {
  const response = await api<{ data: AnomalyInsight[] }>("/api/v1/insights/anomalies");
  return response.data;
}

export function getSpendingInsights(from?: string, to?: string) {
  const params = new URLSearchParams();
  if (from) params.set("from", from);
  if (to) params.set("to", to);
  const suffix = params.toString() ? `?${params}` : "";
  return api<SpendingInsights>(`/api/v1/insights/spending${suffix}`);
}

export async function getCashRecommendations() {
  const response = await api<{ data: CashRecommendation[] }>("/api/v1/recommendations/cash");
  return response.data;
}

export async function getReconciliationMatches(runId: string) {
  const response = await api<{ data: ReconciliationMatch[] }>(`/api/v1/reconciliation-runs/${runId}/matches`);
  return response.data;
}

export function getStripeConnectUrl() {
  return api<StripeConnectUrlResponse>("/api/v1/integrations/stripe/connect-url");
}

export function getStripeStatus() {
  return api<StripeIntegrationStatus>("/api/v1/integrations/stripe/status");
}

export function disconnectStripe() {
  return api<StripeIntegrationStatus>("/api/v1/integrations/stripe", { method: "DELETE" });
}

function demoResponse<T>(path: string, init: RequestInit = {}): T | undefined {
  const method = init.method || "GET";
  const scenario = activeDemoScenario();
  if (path === "/auth/demo-token") return { token: DEMO_TOKEN, user: { id: "demo-user", email: "demo@clearflow.local" } } as T;
  if (path === "/me") return { id: "demo-user", email: "demo@clearflow.local" } as T;
  if (path === "/api/v1/me") return { user: { id: "demo-user", email: "demo@clearflow.local", name: "Demo Operator" }, organizations: [{ id: "demo-org", name: scenario.name, type: scenario.type, currency: scenario.currency, role: "owner" }] } as T;
  if (path.startsWith("/api/v1/auth/sessions") && method === "GET") return [{ id: "session-demo-1", created_at: "2026-07-06T18:00:00Z", expires_at: "2026-07-07T18:00:00Z", user_agent: "Demo browser" }] as T;
  if (path.startsWith("/api/v1/auth/sessions") && method === "DELETE") return undefined as T;
  if (path.startsWith("/api/v1/organizations") && method === "POST") return { id: "demo-org", name: scenario.name, type: scenario.type, currency: scenario.currency, role: "owner" } as T;
  if (path.startsWith("/api/v1/onboarding/status") && method === "GET") return demoOnboardingStatus(scenario) as T;
  if (path.startsWith("/api/v1/onboarding/status") && method === "PUT") return demoOnboardingStatus(scenario) as T;
  if (path.includes("/reconciliation/exceptions/") && path.includes("/notes") && method === "GET") return demoExceptionNotes as T;
  if (path.includes("/reconciliation/exceptions/") && path.includes("/notes") && method === "POST") return { id: crypto.randomUUID(), organization_id: "demo-org", exception_id: "ex-1", user_id: "demo-user", body: "Demo note saved.", created_at: new Date().toISOString() } as T;
  if (path.match(/^\/api\/v1\/organizations\/[^/]+\/members$/) && method === "GET") return demoMembers as T;
  if (path.match(/^\/api\/v1\/organizations\/[^/]+\/members$/) && method === "POST") return { id: crypto.randomUUID(), organization_id: "demo-org", user_id: crypto.randomUUID(), user_email: "invited@example.com", user_name: "Invited User", role: "viewer", created_at: new Date().toISOString() } as T;
  if (path.includes("/members/") && method === "PATCH") return { id: "member-demo-2", organization_id: "demo-org", user_id: "member-demo-2", user_email: "analyst@clearflow.local", user_name: "Analyst", role: "admin", created_at: "2026-07-06T18:00:00Z" } as T;
  if (path.includes("/members/") && method === "DELETE") return undefined as T;
  if (path === "/organizations" || path === "/api/v1/organizations") return [{ id: "demo-org", name: scenario.name, type: scenario.type, currency: scenario.currency, role: "owner" }] as T;
  if (path.startsWith("/api/v1/dashboard/summary")) return demoDashboardSummary(scenario) as T;
  if (path.startsWith("/cash-flow/summary")) return { cash_balance: scenario.cashBalance, income: scenario.income, expenses: scenario.expenses, pending_payouts: 0, fees: scenario.fees, refunds: scenario.refunds, net_cash_flow: scenario.income - scenario.expenses - scenario.fees - scenario.refunds } as T;
  if (path.startsWith("/cash-flow/forecast")) return [{ days: 7, projected_cash: scenario.cashBalance, expected_payouts: 0, expected_expenses: 0 }, { days: 30, projected_cash: scenario.cashBalance - scenario.expenses * 0.9, expected_payouts: 0, expected_expenses: scenario.expenses * 0.9 }, { days: 60, projected_cash: scenario.cashBalance - scenario.expenses * 1.7, expected_payouts: 0, expected_expenses: scenario.expenses * 1.7 }] as T;
  if (path.startsWith("/payments")) return demoPayments as T;
  if (path.startsWith("/payouts")) return demoPayouts as T;
  if (path.startsWith("/bank-transactions")) return demoBankTransactions as T;
  if (path.startsWith("/reconciliation/runs") && method === "GET") return demoRuns as T;
  if (path.startsWith("/reconciliation/runs") && method === "POST") return demoRuns[0] as T;
  if (path.startsWith("/reconciliation/exceptions") && method === "GET") return demoExceptions as T;
  if (path.startsWith("/reconciliation/exceptions") && method === "PATCH") return { ...demoExceptions[0], status: "resolved" } as T;
  if (path.startsWith("/sync/stripe")) return { payments: 5, refunds: 1, fees: 5, payout: demoPayouts[0] } as T;
  if (path.startsWith("/sync/bank")) return { bank_transactions: 3 } as T;
  if (path.startsWith("/api/v1/payouts/") && (path.includes("/explanation") || path.includes("/breakdown"))) return demoPayoutExplanation as T;
  if (path.startsWith("/api/v1/cashflow/forecast")) return demoIntelligenceForecast(scenario) as T;
  if (path.startsWith("/api/v1/insights/anomalies")) return { data: demoAnomalies } as T;
  if (path.startsWith("/api/v1/insights/spending")) return demoSpendingInsights as T;
  if (path.startsWith("/api/v1/recommendations/cash")) return { data: demoCashRecommendations } as T;
  if (path.startsWith("/api/v1/reconciliation-runs/") && path.includes("/matches")) return { data: demoReconciliationMatches } as T;
  if (path.startsWith("/api/v1/integrations/stripe/connect-url")) return { url: "https://connect.stripe.com/oauth/authorize?response_type=code&client_id=ca_mock_clearflow&state=demo-state", state: "demo-state" } as T;
  if (path.startsWith("/api/v1/integrations/stripe/status")) return demoStripeStatus as T;
  if (path.startsWith("/api/v1/integrations/stripe") && method === "DELETE") return { connected: false, provider: "stripe" } as T;
  if (path.startsWith("/api/v1/webhooks/processors/stripe")) return { status: "accepted", provider: "stripe", queued: true } as T;
  if (path.startsWith("/api/v1/webhooks/plaid")) return { status: "accepted", provider: "plaid", queued: true } as T;
  if (path.startsWith("/api/v1/jobs/dead")) return { data: [], pagination: demoPagination } as T;
  if (path.startsWith("/api/v1/jobs")) return { data: demoJobs, pagination: demoPagination } as T;
  if (path.startsWith("/api/v1/audit-logs")) return { data: demoAuditLogs, pagination: demoPagination } as T;
  if (path.startsWith("/api/v1/ops/metrics")) return demoMetrics as T;
  if (path.startsWith("/debug/clearflow/reset-demo")) return { status: "reset", organization_id: "demo-org" } as T;
  if (path.startsWith("/portfolio/summary")) return demoPortfolioSummary as T;
  if (path.startsWith("/portfolio/allocation")) return demoPortfolioAllocation as T;
  if (path.startsWith("/portfolio/holdings")) return demoPortfolioHoldings as T;
  if (path.includes("/portfolio/imports/") && path.includes("/errors")) return [] as T;
  if (path.startsWith("/portfolio/transactions")) return demoPortfolioTransactions as T;
  if (path.startsWith("/portfolio/imports")) return demoPortfolioImports as T;
  if (path.startsWith("/portfolio/risk")) return demoPortfolioRisk as T;
  if (path.startsWith("/connections/plaid/link-token")) return { link_token: "", demo_unavailable: true } as T;
  if (path.startsWith("/connections/plaid/sandbox-connect")) return { demo_unavailable: true, message: "Start the local API with Plaid Sandbox credentials to create a test connection." } as T;
  if (path.startsWith("/connections/plaid/sync-transactions")) return { imported_count: 0, connection_count: 0 } as T;
  if (path.startsWith("/connections/plaid/sync-investments")) return { mode: "mock", import: demoPortfolioImports[0], holdings: demoPortfolioHoldings, portfolio_transactions: demoPortfolioTransactions, errors: [] } as T;
  if (path.startsWith("/connections")) return demoPlaidConnections as T;
  if (path.startsWith("/transactions")) return [] as T;
  if (path.startsWith("/insights")) return [] as T;
  return undefined;
}

function demoDashboardSummary(scenario: DemoScenario) {
  return {
    cash: { cash_balance: scenario.cashBalance, income: scenario.income, expenses: scenario.expenses, pending_payouts: 0, fees: scenario.fees, refunds: scenario.refunds, net_cash_flow: scenario.income - scenario.expenses - scenario.fees - scenario.refunds },
    forecast: [{ days: 7, projected_cash: scenario.cashBalance, expected_payouts: 0, expected_expenses: 0 }, { days: 30, projected_cash: scenario.cashBalance - scenario.expenses * 0.9, expected_payouts: 0, expected_expenses: scenario.expenses * 0.9 }, { days: 60, projected_cash: scenario.cashBalance - scenario.expenses * 1.7, expected_payouts: 0, expected_expenses: scenario.expenses * 1.7 }],
    exceptions: demoExceptions,
    payouts: demoPayouts,
    payments: demoPayments,
    bank_transactions: demoBankTransactions,
    connections: demoPlaidConnections,
    metrics: demoMetrics
  };
}

const demoMembers = [
  { id: "member-demo-1", organization_id: "demo-org", user_id: "demo-user", user_email: "demo@clearflow.local", user_name: "Demo Operator", role: "owner", created_at: "2026-07-06T18:00:00Z" },
  { id: "member-demo-2", organization_id: "demo-org", user_id: "analyst-demo", user_email: "analyst@clearflow.local", user_name: "Analyst", role: "viewer", created_at: "2026-07-06T18:10:00Z" }
];

function demoOnboardingStatus(scenario: DemoScenario) {
  return {
    organization_id: "demo-org",
    selected_scenario: scenario.id,
    business_type: scenario.type,
    checklist: {
      workspace_created: true,
      stripe_connected: true,
      plaid_connected: true,
      processor_data_ready: true,
      bank_data_ready: true,
      team_ready: true,
      open_breaks: 2
    },
    provider_readiness: {
      workspace_created: true,
      stripe_connected: true,
      plaid_connected: true,
      processor_data_ready: true,
      bank_data_ready: true,
      team_ready: true,
      open_breaks: 2
    },
    created_at: "2026-07-06T18:00:00Z",
    updated_at: "2026-07-06T20:00:00Z"
  };
}

const demoExceptionNotes = [
  { id: "note-demo-1", organization_id: "demo-org", exception_id: "ex-1", user_id: "demo-user", body: "Verified likely processor fee reserve timing during demo review.", created_at: "2026-07-06T20:12:00Z" }
];

const demoPayments = [
  { id: "pay-1", processor_payment_id: "ch_hoodie_001", customer_email: "buyer1@example.com", amount: 48, status: "succeeded", description: "Hoodie order", occurred_at: "2026-07-01T15:30:00Z" },
  { id: "pay-2", processor_payment_id: "ch_dues_001", customer_email: "member@example.com", amount: 120, status: "succeeded", description: "Semester dues", occurred_at: "2026-07-02T16:10:00Z" },
  { id: "pay-3", processor_payment_id: "ch_sponsor_001", customer_email: "sponsor@example.com", amount: 1500, status: "succeeded", description: "Event sponsorship", occurred_at: "2026-07-03T13:20:00Z" },
  { id: "pay-4", processor_payment_id: "ch_ticket_001", customer_email: "guest@example.com", amount: 35, status: "refunded", description: "Event ticket", occurred_at: "2026-07-03T18:45:00Z" }
];

const demoPayouts = [
  { id: "po-1", processor_payout_id: "po_demo_001", amount: 1665.72, status: "paid", expected_arrival_at: "2026-07-04T12:00:00Z" },
  { id: "po-2", processor_payout_id: "po_demo_002", amount: 1301.55, status: "paid", expected_arrival_at: "2026-07-05T12:00:00Z" },
  { id: "po-3", processor_payout_id: "po_demo_003", amount: 720, status: "paid", expected_arrival_at: "2026-07-06T12:00:00Z" }
];

const demoBankTransactions = [
  { id: "bank-1", external_id: "bank_stripe_001", amount: 1665.72, direction: "credit", description: "STRIPE PAYOUT", posted_at: "2026-07-04T12:00:00Z" },
  { id: "bank-2", external_id: "bank_stripe_002", amount: 1290.55, direction: "credit", description: "STRIPE PAYOUT POSSIBLE", posted_at: "2026-07-05T12:00:00Z" },
  { id: "bank-3", external_id: "bank_unknown_001", amount: 212.45, direction: "credit", description: "Unknown deposit", posted_at: "2026-07-05T14:15:00Z" },
  { id: "bank-4", external_id: "bank_venue_001", amount: 300, direction: "debit", description: "Venue deposit", posted_at: "2026-07-05T18:00:00Z" }
];

const demoRuns = [
  { id: "run-001", status: "completed", matched_count: 2, exception_count: 1, started_at: "2026-07-05T03:07:04Z" },
  { id: "run-002", status: "completed", matched_count: 2, exception_count: 1, started_at: "2026-07-04T21:20:00Z" }
];

const demoExceptions = [
  { id: "ex-1", severity: "medium", title: "Likely payout amount mismatch", explanation: "Stripe payout po_demo_002 is close to a bank deposit but differs by $11.00.", status: "open", created_at: "2026-07-05T03:07:04Z" },
  { id: "ex-2", severity: "high", title: "Missing payout deposit", explanation: "Stripe payout po_demo_003 was expected on Jul 6 and has no matching bank deposit.", status: "open", created_at: "2026-07-06T15:30:00Z" },
  { id: "ex-3", severity: "medium", title: "Unmatched bank deposit", explanation: "Bank deposit Unknown deposit for $212.45 is not tied to a known payout.", status: "open", created_at: "2026-07-06T15:32:00Z" }
];

function demoIntelligenceForecast(scenario: DemoScenario): CashflowForecast {
  const starting = Math.round(scenario.cashBalance * 100);
  const dailyBurn = Math.round((scenario.expenses / 30) * 100);
  const weeklyInflow = Math.round((scenario.income / 4) * 100);
  return {
    organizationId: "demo-org",
    horizonDays: 30,
    startingBalanceMinor: starting,
    projectedEndingBalanceMinor: starting + weeklyInflow * 4 - dailyBurn * 30,
    currency: scenario.currency,
    confidence: "medium",
    assumptions: [
      `Scenario: ${scenario.name}.`,
      "Used recent bank credits and debits as the operating baseline.",
      "Included expected weekly processor deposits and recurring operating costs."
    ],
    series: Array.from({ length: 30 }, (_, index) => {
      const day = index + 1;
      const date = new Date(Date.UTC(2026, 6, 7 + day));
      const inflow = day % 7 === 0 ? weeklyInflow : 0;
      const outflow = dailyBurn + (day % 10 === 0 ? Math.round(scenario.fees * 100) : 0);
      return {
        date: date.toISOString().slice(0, 10),
        projectedBalanceMinor: starting + Math.floor(day / 7) * weeklyInflow - day * dailyBurn - Math.floor(day / 10) * Math.round(scenario.fees * 100),
        expectedInflowsMinor: inflow,
        expectedOutflowsMinor: outflow
      };
    })
  };
}

const demoAnomalies: AnomalyInsight[] = [
  {
    id: "missing_payout:po-demo-3",
    type: "missing_payout",
    severity: "high",
    title: "Expected payout has not reached the bank",
    description: "Stripe payout po_demo_003 was expected yesterday and has no matching bank deposit.",
    resourceType: "processor_payout",
    resourceId: "po-demo-3",
    detectedAt: "2026-07-06T15:30:00Z",
    recommendedAction: "Check the processor payout status and confirm bank settlement timing."
  },
  {
    id: "unmatched_deposit:bank-2",
    type: "unmatched_deposit",
    severity: "medium",
    title: "Unmatched bank deposit",
    description: "A $212.45 bank credit is not tied to a known processor payout.",
    resourceType: "bank_transaction",
    resourceId: "bank-2",
    detectedAt: "2026-07-06T15:32:00Z",
    recommendedAction: "Review bank memo details and annotate the deposit source."
  },
  {
    id: "high_processor_fee",
    type: "high_processor_fee",
    severity: "medium",
    title: "Processor fees look high",
    description: "Fees are running above 5% of payout volume for the current sample period.",
    resourceType: "fees",
    detectedAt: "2026-07-06T15:35:00Z",
    recommendedAction: "Compare card mix, pricing tier, and refund-related fee leakage."
  }
];

const demoSpendingInsights: SpendingInsights = {
  totalSpendMinor: 51245,
  currency: "USD",
  categories: [
    { category: "facilities", amountMinor: 30000, percentage: 58.5, changeVsPreviousPeriod: 12.4 },
    { category: "software", amountMinor: 12900, percentage: 25.2, changeVsPreviousPeriod: -4.1 },
    { category: "food", amountMinor: 8345, percentage: 16.3, changeVsPreviousPeriod: 3.7 }
  ],
  topMerchants: [
    { merchant: "Venue deposit", amountMinor: 30000 },
    { merchant: "Design software", amountMinor: 12900 },
    { merchant: "Event food vendor", amountMinor: 8345 }
  ],
  notes: ["Facilities is the largest spending category in this period.", "Software spend is trending lower than the previous period."]
};

const demoCashRecommendations: CashRecommendation[] = [
  { type: "reserve", priority: "high", title: "Keep at least 30 days of operating cash", description: "Maintain a reserve before approving new event spend.", amountMinor: 51245, currency: "USD" },
  { type: "follow_up_missing_payout", priority: "high", title: "Follow up on missing payouts", description: "Confirm payout status before relying on expected processor cash.", currency: "USD" },
  { type: "reduce_fees", priority: "medium", title: "Review processor fees", description: "Fees are elevated relative to payout volume. Check payment method mix and pricing tier.", currency: "USD" }
];

const demoPayoutExplanation: PayoutExplanation = {
  payoutId: "po-1",
  processor: "stripe",
  grossAmountMinor: 209872,
  feesMinor: 25812,
  refundsMinor: 17500,
  netAmountMinor: 166572,
  currency: "USD",
  bankDeposit: { id: "bank-1", amountMinor: 166572, postedAt: "2026-07-04T12:00:00Z" },
  summary: "This payout represents $2,098.72 in gross payments minus $258.12 in fees and $175.00 in refunds, resulting in a $1,665.72 bank deposit.",
  lineItems: [],
  warnings: ["processor fees are elevated for the current sample period"]
};

const demoReconciliationMatches: ReconciliationMatch[] = [
  {
    id: "po-1:bank-1",
    processorPayoutId: "po-1",
    bankDepositId: "bank-1",
    status: "matched",
    confidenceScore: 0.98,
    amountDifferenceMinor: 0,
    currency: "USD",
    reasons: ["same_currency", "exact_amount", "date_within_window", "payout_reference_match"],
    explanation: "This deposit matches the processor payout because the amount, currency, and deposit timing align."
  },
  {
    id: "po-2:bank-2",
    processorPayoutId: "po-2",
    bankDepositId: "bank-2",
    status: "likely_match",
    confidenceScore: 0.72,
    amountDifferenceMinor: 1100,
    currency: "USD",
    reasons: ["same_currency", "date_within_window", "amount_off_by_fee"],
    explanation: "This bank deposit is likely tied to the payout, but the $11.00 amount gap needs operator review."
  },
  {
    id: "missing:po-3",
    processorPayoutId: "po-3",
    status: "missing_payout",
    confidenceScore: 0,
    amountDifferenceMinor: 72000,
    currency: "USD",
    reasons: ["expected_arrival_passed", "no_bank_deposit"],
    explanation: "This processor payout was expected but has no matching bank deposit."
  },
  {
    id: "unmatched:bank-3",
    bankDepositId: "bank-3",
    status: "unmatched_deposit",
    confidenceScore: 0,
    amountDifferenceMinor: 21245,
    currency: "USD",
    reasons: ["unmatched_deposit"],
    explanation: "This bank deposit is not tied to a known processor payout."
  }
];

const demoStripeStatus: StripeIntegrationStatus = {
  connected: true,
  provider: "stripe",
  accountId: "acct_demo_123",
  displayName: "Stripe Test Account",
  connectedAt: "2026-07-06T18:00:00Z",
  lastSyncAt: "2026-07-06T20:20:00Z"
};

const demoPlaidConnections = [
  { id: "plaid-demo-1", institution_name: "Plaid Sandbox Bank", products: ["transactions"], last_synced_at: "2026-07-06T20:58:26Z" }
];

const demoPortfolioHoldings = [
  { id: "holding-1", symbol: "VFV.TO", security_name: "Vanguard S&P 500 Index ETF", security_type: "etf", quantity: 45, average_cost: 112, market_value: 6675.75, currency: "CAD" },
  { id: "holding-2", symbol: "XEQT.TO", security_name: "iShares Core Equity ETF Portfolio", security_type: "etf", quantity: 82, average_cost: 29, market_value: 2986.44, currency: "CAD" },
  { id: "holding-3", symbol: "AAPL", security_name: "Apple Inc.", security_type: "stock", quantity: 26, average_cost: 152, market_value: 5571.54, currency: "USD" },
  { id: "holding-4", symbol: "MSFT", security_name: "Microsoft Corp.", security_type: "stock", quantity: 8, average_cost: 320, market_value: 3990.72, currency: "USD" },
  { id: "holding-5", symbol: "CASH", security_name: "Cash", security_type: "cash", quantity: 1800, average_cost: 1, market_value: 1800, currency: "CAD" }
];

const demoPortfolioSummary = {
  total_market_value: 21024.45,
  total_cost_basis: 15858,
  unrealized_gain_loss: 5166.45,
  unrealized_gain_loss_pct: 32.6,
  cash_value: 1800,
  invested_value: 19224.45,
  top_holdings: [
    { name: "VFV.TO", value: 6675.75, percent: 31.8 },
    { name: "AAPL", value: 5571.54, percent: 26.5 },
    { name: "MSFT", value: 3990.72, percent: 19.0 }
  ]
};

const demoPortfolioAllocation = {
  by_security_type: [
    { name: "etf", value: 9662.19, percent: 46 },
    { name: "stock", value: 9562.26, percent: 45.5 },
    { name: "cash", value: 1800, percent: 8.6 }
  ],
  by_symbol: demoPortfolioSummary.top_holdings
};

const demoPortfolioTransactions = [
  { id: "ptx-1", symbol: "CASH", transaction_type: "deposit", quantity: 0, price: 0, amount: 2500, fees: 0, currency: "CAD", occurred_at: "2026-04-10T00:00:00Z", description: "Monthly contribution" },
  { id: "ptx-2", symbol: "VFV.TO", transaction_type: "buy", quantity: 15, price: 142, amount: 2130, fees: 0, currency: "CAD", occurred_at: "2026-04-11T00:00:00Z", description: "Core ETF purchase" },
  { id: "ptx-3", symbol: "AAPL", transaction_type: "buy", quantity: 6, price: 186, amount: 1116, fees: 1.5, currency: "USD", occurred_at: "2026-05-11T00:00:00Z", description: "Individual stock purchase" },
  { id: "ptx-4", symbol: "VFV.TO", transaction_type: "dividend", quantity: 0, price: 0, amount: 32.4, fees: 0, currency: "CAD", occurred_at: "2026-05-20T00:00:00Z", description: "ETF distribution" }
];

const demoPortfolioImports = [
  { id: "pimp-1", import_type: "holdings", original_filename: "sample_holdings.csv", row_count: 5, imported_count: 5, failed_count: 0, created_at: "2026-07-06T20:00:00Z" },
  { id: "pimp-2", import_type: "portfolio_transactions", original_filename: "sample_portfolio_transactions.csv", row_count: 8, imported_count: 8, failed_count: 0, created_at: "2026-07-06T20:01:00Z" }
];

const demoPortfolioRisk = [
  { severity: "medium", title: "Single-holding concentration", explanation: "VFV.TO is above 25% of tracked holdings. Review diversification against your goals." },
  { severity: "medium", title: "Top-five concentration", explanation: "Your top five holdings exceed 60% of tracked value. This can increase portfolio-specific risk." }
];

const demoPagination = { limit: 25, offset: 0, count: 3 };

const demoJobs = [
  { id: "job-recon-001", organization_id: "demo-org", type: "reconciliation.run", status: "completed", attempts: 1, max_attempts: 3, created_at: "2026-07-06T20:22:00Z", updated_at: "2026-07-06T20:22:04Z" },
  { id: "job-stripe-001", organization_id: "demo-org", type: "stripe.sync", status: "completed", attempts: 1, max_attempts: 3, created_at: "2026-07-06T20:20:00Z", updated_at: "2026-07-06T20:20:02Z" },
  { id: "job-bank-001", organization_id: "demo-org", type: "bank.sync", status: "completed", attempts: 1, max_attempts: 3, created_at: "2026-07-06T20:21:00Z", updated_at: "2026-07-06T20:21:02Z" }
];

const demoAuditLogs = [
  { id: "audit-1", action: "reconciliation.run_completed", target_type: "reconciliation_run", target_id: "run-001", created_at: "2026-07-06T20:22:04Z" },
  { id: "audit-2", action: "stripe.mock_synced", target_type: "payout", target_id: "po-1", created_at: "2026-07-06T20:20:02Z" },
  { id: "audit-3", action: "bank.mock_synced", target_type: "bank_transaction", target_id: "bank-1", created_at: "2026-07-06T20:21:02Z" }
];

const demoMetrics = {
  http_requests_total: 184,
  http_errors_total: 2,
  jobs_queued_total: 3,
  jobs_completed_total: 3,
  jobs_failed_total: 0,
  job_queue_depth: 0,
  plaid_webhooks_received_total: 1,
  stripe_webhook_events_total: 1,
  idempotency_replays_total: 2,
  redis_rate_limit_hits_total: 0,
  redis_idempotency_locks_total: 2,
  otel_traces_started_total: 18
};
