"use client";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";
const DEMO_TOKEN = "clearflow-demo-token";

export function token() {
  if (typeof window === "undefined") return "";
  return localStorage.getItem("fynora_token") || "";
}

export function setToken(value: string) {
  localStorage.setItem("fynora_token", value);
}

export function logout() {
  localStorage.removeItem("fynora_token");
  window.location.href = "/";
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const started = performance.now();
  const requestId = crypto.randomUUID();
  const headers = new Headers(init.headers);
  headers.set("Content-Type", headers.get("Content-Type") || "application/json");
  headers.set("X-Request-ID", requestId);
  const jwt = token();
  if (jwt) headers.set("Authorization", `Bearer ${jwt}`);
  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, { ...init, headers });
  } catch (err) {
    const fallback = demoResponse<T>(path, init);
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
    console.error("[clearflow-api:error]", { path, status: res.status, durationMs, requestId: responseRequestId, message: body?.error?.message });
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
    console.error("[clearflow-api:error]", { path, status: res.status, durationMs, requestId: responseRequestId });
    throw new Error("Upload failed");
  }
  console.info("[clearflow-api]", { path, method: "POST", status: res.status, durationMs, requestId: responseRequestId });
  return res.json();
}

function demoResponse<T>(path: string, init: RequestInit = {}): T | undefined {
  const method = init.method || "GET";
  if (path === "/auth/demo-token") return { token: DEMO_TOKEN, user: { id: "demo-user", email: "demo@clearflow.local" } } as T;
  if (path === "/me") return { id: "demo-user", email: "demo@clearflow.local" } as T;
  if (path === "/organizations") return [{ id: "demo-org", name: "Northside Student Association", type: "student_organization", currency: "USD" }] as T;
  if (path.startsWith("/cash-flow/summary")) return { cash_balance: 2967.27, income: 3179.72, expenses: 512.45, pending_payouts: 0, fees: 258.12, refunds: 175, net_cash_flow: 2667.27 } as T;
  if (path.startsWith("/cash-flow/forecast")) return [{ days: 7, projected_cash: 2967.27, expected_payouts: 0, expected_expenses: 0 }, { days: 30, projected_cash: 2517.27, expected_payouts: 0, expected_expenses: 450 }, { days: 60, projected_cash: 2517.27, expected_payouts: 0, expected_expenses: 450 }] as T;
  if (path.startsWith("/payments")) return demoPayments as T;
  if (path.startsWith("/payouts")) return demoPayouts as T;
  if (path.startsWith("/bank-transactions")) return demoBankTransactions as T;
  if (path.startsWith("/reconciliation/runs") && method === "GET") return demoRuns as T;
  if (path.startsWith("/reconciliation/runs") && method === "POST") return demoRuns[0] as T;
  if (path.startsWith("/reconciliation/exceptions") && method === "GET") return demoExceptions as T;
  if (path.startsWith("/reconciliation/exceptions") && method === "PATCH") return { ...demoExceptions[0], status: "resolved" } as T;
  if (path.startsWith("/sync/stripe")) return { payments: 5, refunds: 1, fees: 5, payout: demoPayouts[0] } as T;
  if (path.startsWith("/sync/bank")) return { bank_transactions: 3 } as T;
  if (path.startsWith("/connections")) return [] as T;
  if (path.startsWith("/transactions")) return [] as T;
  if (path.startsWith("/insights")) return [] as T;
  return undefined;
}

const demoPayments = [
  { id: "pay-1", processor_payment_id: "ch_hoodie_001", customer_email: "buyer1@example.com", amount: 48, status: "succeeded", description: "Hoodie order", occurred_at: "2026-07-01T15:30:00Z" },
  { id: "pay-2", processor_payment_id: "ch_dues_001", customer_email: "member@example.com", amount: 120, status: "succeeded", description: "Semester dues", occurred_at: "2026-07-02T16:10:00Z" },
  { id: "pay-3", processor_payment_id: "ch_sponsor_001", customer_email: "sponsor@example.com", amount: 1500, status: "succeeded", description: "Event sponsorship", occurred_at: "2026-07-03T13:20:00Z" },
  { id: "pay-4", processor_payment_id: "ch_ticket_001", customer_email: "guest@example.com", amount: 35, status: "refunded", description: "Event ticket", occurred_at: "2026-07-03T18:45:00Z" }
];

const demoPayouts = [
  { id: "po-1", processor_payout_id: "po_demo_001", amount: 1665.72, status: "paid", expected_arrival_at: "2026-07-04T12:00:00Z" },
  { id: "po-2", processor_payout_id: "po_demo_002", amount: 1301.55, status: "paid", expected_arrival_at: "2026-07-05T12:00:00Z" }
];

const demoBankTransactions = [
  { id: "bank-1", external_id: "bank_stripe_001", amount: 1665.72, direction: "credit", description: "STRIPE PAYOUT", posted_at: "2026-07-04T12:00:00Z" },
  { id: "bank-2", external_id: "bank_unknown_001", amount: 212.45, direction: "credit", description: "Unknown deposit", posted_at: "2026-07-05T12:00:00Z" },
  { id: "bank-3", external_id: "bank_venue_001", amount: 300, direction: "debit", description: "Venue deposit", posted_at: "2026-07-05T12:00:00Z" }
];

const demoRuns = [
  { id: "run-001", status: "completed", matched_count: 2, exception_count: 1, started_at: "2026-07-05T03:07:04Z" },
  { id: "run-002", status: "completed", matched_count: 2, exception_count: 1, started_at: "2026-07-04T21:20:00Z" }
];

const demoExceptions = [
  { id: "ex-1", severity: "medium", title: "Unmatched bank deposit", explanation: "Bank deposit Unknown deposit for $212.45 is not tied to a known payout.", status: "open", created_at: "2026-07-05T03:07:04Z" }
];
