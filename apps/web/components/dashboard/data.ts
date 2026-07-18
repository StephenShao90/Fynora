"use client";

import { activeDemoScenario } from "@/lib/api";

export type Payout = { id: string; processor_payout_id: string; amount: number; status: string; expected_arrival_at: string };
export type Payment = { id: string; amount: number; status: string; description: string; occurred_at: string };
export type BankTransaction = { id: string; amount: number; direction: string; description: string; posted_at: string };
export type Exception = { id: string; title: string; explanation: string; status: string; severity: string };
export type Connection = { id: string; institution_name: string };

export type DashboardSummary = {
  cash: Record<string, number>;
  forecast: Array<Record<string, number>>;
  exceptions: Exception[];
  payouts: Payout[];
  payments: Payment[];
  bank_transactions: BankTransaction[];
  connections: Connection[];
  metrics: Record<string, number>;
};

export function normalizeDashboardSummary(
  input: Partial<DashboardSummary> | null | undefined,
  fallback = dashboardFallback()
): DashboardSummary {
  const data = input || {};
  return {
    cash: data.cash && typeof data.cash === "object" ? data.cash : fallback.cash,
    forecast: Array.isArray(data.forecast) ? data.forecast : fallback.forecast,
    exceptions: Array.isArray(data.exceptions) ? data.exceptions : [],
    payouts: Array.isArray(data.payouts) ? data.payouts : [],
    payments: Array.isArray(data.payments) ? data.payments : [],
    bank_transactions: Array.isArray(data.bank_transactions) ? data.bank_transactions : [],
    connections: Array.isArray(data.connections) ? data.connections : [],
    metrics: data.metrics && typeof data.metrics === "object" ? data.metrics : {}
  };
}

export function dashboardFallback(): DashboardSummary {
  const scenario = activeDemoScenario();
  const now = new Date().toISOString();
  const cash = {
    cash_balance: scenario.cashBalance,
    income: scenario.income,
    expenses: scenario.expenses,
    pending_payouts: 0,
    fees: scenario.fees,
    refunds: scenario.refunds,
    net_cash_flow: scenario.income - scenario.expenses - scenario.fees - scenario.refunds
  };
  return {
    cash,
    forecast: [
      { days: 7, projected_cash: scenario.cashBalance, expected_payouts: 0, expected_expenses: 0 },
      { days: 30, projected_cash: scenario.cashBalance - scenario.expenses * 0.9, expected_payouts: 0, expected_expenses: scenario.expenses * 0.9 },
      { days: 60, projected_cash: scenario.cashBalance - scenario.expenses * 1.7, expected_payouts: 0, expected_expenses: scenario.expenses * 1.7 }
    ],
    exceptions: [
      { id: "fallback-ex-1", title: "Likely payout amount mismatch", explanation: "Processor payout is close to a bank deposit but differs by expected fees or reserves.", status: "open", severity: "medium" }
    ],
    payouts: [
      { id: "fallback-po-1", processor_payout_id: "po_demo_001", amount: Math.max(0, scenario.income - scenario.fees - scenario.refunds), status: "paid", expected_arrival_at: now }
    ],
    payments: [
      { id: "fallback-pay-1", amount: Math.round(scenario.income * 0.55), status: "succeeded", description: "Customer payments batch", occurred_at: now },
      { id: "fallback-pay-2", amount: Math.round(scenario.income * 0.45), status: "succeeded", description: "Sponsor or subscription payments", occurred_at: now }
    ],
    bank_transactions: [
      { id: "fallback-bank-1", amount: scenario.cashBalance, direction: "credit", description: "Stripe payout deposit", posted_at: now },
      { id: "fallback-bank-2", amount: scenario.expenses, direction: "debit", description: "Operating expenses", posted_at: now }
    ],
    connections: [{ id: "fallback-conn-1", institution_name: "Connected bank" }],
    metrics: { jobs_completed_total: 3, job_queue_depth: 0, http_requests_total: 0 }
  };
}
