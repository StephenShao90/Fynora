export type CashFlow = {
  average_monthly_income: number;
  average_fixed_expenses: number;
  average_variable_expenses: number;
  average_net_cash_flow: number;
  savings_capacity: number;
  safe_savings_recommendation: number;
  buffer_recommendation: number;
};

export type NamedAmount = { name: string; amount?: number; value?: number; percent?: number };
export type Transaction = { id: string; occurred_at: string; merchant: string; normalized_merchant: string; description: string; category: string; amount: number; direction: string; currency: string };
export type Holding = { id: string; symbol: string; security_name: string; security_type: string; quantity: number; average_cost: number; market_value: number; currency: string };
export type Summary = { total_market_value: number; total_cost_basis: number; unrealized_gain_loss: number; unrealized_gain_loss_pct: number; cash_value: number; invested_value: number; top_holdings: NamedAmount[] };
export type RiskFinding = { severity: string; title: string; explanation: string };
