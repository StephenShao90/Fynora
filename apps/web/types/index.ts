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
export type PortfolioTransaction = { id: string; symbol: string; transaction_type: string; quantity: number; price: number; amount: number; fees: number; currency: string; occurred_at: string; description: string };
export type PortfolioImport = { id: string; import_type: string; original_filename: string; row_count: number; imported_count: number; failed_count: number; created_at: string };
export type Summary = { total_market_value: number; total_cost_basis: number; unrealized_gain_loss: number; unrealized_gain_loss_pct: number; cash_value: number; invested_value: number; top_holdings: NamedAmount[] };
export type RiskFinding = { severity: string; title: string; explanation: string };
