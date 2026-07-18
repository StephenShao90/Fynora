import { describe, expect, it } from "vitest";
import { normalizeDashboardSummary, type DashboardSummary } from "@/components/dashboard/data";

const fallback: DashboardSummary = {
  cash: { cash_balance: 0, net_cash_flow: 0, fees: 0, refunds: 0 },
  forecast: [{ days: 7, projected_cash: 0 }],
  exceptions: [],
  payouts: [],
  payments: [],
  bank_transactions: [],
  connections: [],
  metrics: {}
};

describe("dashboard data normalization", () => {
  it("renders API null arrays as empty dashboard collections", () => {
    const summary = normalizeDashboardSummary(
      {
        cash: { cash_balance: 1250, net_cash_flow: 200, fees: 25, refunds: 10 },
        forecast: null,
        exceptions: null,
        payouts: null,
        payments: null,
        bank_transactions: null,
        connections: null,
        metrics: null
      } as never,
      fallback
    );

    expect(summary.cash.cash_balance).toBe(1250);
    expect(summary.forecast).toEqual(expect.any(Array));
    expect(summary.exceptions).toEqual([]);
    expect(summary.payouts).toEqual([]);
    expect(summary.payments).toEqual([]);
    expect(summary.bank_transactions).toEqual([]);
    expect(summary.connections).toEqual([]);
    expect(summary.metrics).toEqual({});
  });
});
