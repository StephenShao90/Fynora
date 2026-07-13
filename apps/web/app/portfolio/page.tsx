"use client";

import { useMemo, useState } from "react";
import type { ReactNode } from "react";
import { Bar, BarChart, Pie, PieChart, ResponsiveContainer, Tooltip, XAxis, YAxis, Cell } from "recharts";
import { Card, Metric, Shell, money } from "@/components/Shell";
import { Empty, Header } from "@/components/Common";
import { useToast } from "@/components/ToastProvider";
import { useApi } from "@/hooks/useApi";
import { api, upload } from "@/lib/api";
import type { Holding, ImportError, NamedAmount, PortfolioImport, PortfolioTransaction, RiskFinding, Summary } from "@/types";

const palette = ["#315846", "#f07b63", "#d6a53a", "#5a8bb0", "#17231d", "#8aa398"];

export default function PortfolioPage() {
  const { pushToast } = useToast();
  const [refreshKey, setRefreshKey] = useState(0);
  const [result, setResult] = useState("");
  const refresh = `?refresh=${refreshKey}`;
  const summary = useApi<Summary>(`/portfolio/summary${refresh}`, {} as Summary);
  const allocation = useApi<{ by_security_type: NamedAmount[]; by_symbol: NamedAmount[] }>(`/portfolio/allocation${refresh}`, { by_security_type: [], by_symbol: [] });
  const holdings = useApi<Holding[]>(`/portfolio/holdings${refresh}`, []);
  const risk = useApi<RiskFinding[]>(`/portfolio/risk${refresh}`, []);
  const transactions = useApi<PortfolioTransaction[]>(`/portfolio/transactions${refresh}`, []);
  const imports = useApi<PortfolioImport[]>(`/portfolio/imports${refresh}`, []);
  const latestFailedImport = imports.data.find((item) => item.failed_count > 0);
  const importErrors = useApi<ImportError[]>(latestFailedImport ? `/portfolio/imports/${latestFailedImport.id}/errors${refresh}` : `/portfolio/imports/00000000-0000-0000-0000-000000000000/errors${refresh}`, []);

  const costBasis = summary.data.total_cost_basis || 0;
  const gainPct = costBasis ? `${summary.data.unrealized_gain_loss_pct.toFixed(1)}%` : "0.0%";
  const topSymbol = allocation.data.by_symbol?.[0]?.name || "None";
  const topSymbolPct = allocation.data.by_symbol?.[0]?.percent || 0;
  const sortedTransactions = useMemo(() => transactions.data.slice(0, 8), [transactions.data]);

  async function importCSV(path: string, file?: File) {
    if (!file) return;
    try {
      const payload = await upload<Record<string, unknown>>(path, file);
      setResult(JSON.stringify(payload, null, 2));
      setRefreshKey((key) => key + 1);
      const imported = (payload.import as { imported_count?: number } | undefined)?.imported_count;
      pushToast({ tone: "success", title: "Portfolio CSV imported", detail: `${file.name}${typeof imported === "number" ? ` · ${imported} rows` : ""}` });
    } catch (err) {
      pushToast({ tone: "error", title: "Portfolio import failed", detail: (err as Error).message });
    }
  }

  async function syncPlaidInvestments() {
    try {
      const payload = await api<Record<string, unknown>>("/connections/plaid/sync-investments", { method: "POST", body: "{}" });
      setResult(JSON.stringify(payload, null, 2));
      setRefreshKey((key) => key + 1);
      pushToast({ tone: "success", title: "Plaid Investments synced", detail: "Mock investment holdings and activity were imported into the portfolio ledger." });
    } catch (err) {
      pushToast({ tone: "error", title: "Plaid Investments sync failed", detail: (err as Error).message });
    }
  }

  return (
    <Shell>
      <Header title="Portfolio" subtitle="Import brokerage holdings and activity, normalize it into a clean ledger, then review allocation and concentration risk." />

      <div className="grid gap-4 md:grid-cols-4">
        <Metric label="Market value" value={money(summary.data.total_market_value)} />
        <Metric label="Unrealized gain/loss" value={`${money(summary.data.unrealized_gain_loss)} (${gainPct})`} tone={(summary.data.unrealized_gain_loss || 0) >= 0 ? "good" : "warn"} />
        <Metric label="Cash" value={money(summary.data.cash_value)} />
        <Metric label="Largest holding" value={`${topSymbol} ${topSymbolPct ? `${topSymbolPct}%` : ""}`} />
      </div>

      <div className="mt-5 grid gap-5 xl:grid-cols-[.95fr_1.05fr]">
        <Card title="Import brokerage data">
          <div className="grid gap-3 md:grid-cols-2">
            <ImportTile
              icon={<span aria-hidden="true">CSV</span>}
              title="Holdings snapshot"
              description="Positions, quantities, cost basis, prices, and market value from Fidelity, Schwab, Robinhood, Wealthsimple, or manual exports."
              sample="sample_holdings.csv"
              onFile={(file) => importCSV("/portfolio/import/holdings-csv", file)}
            />
            <ImportTile
              icon={<span aria-hidden="true">TX</span>}
              title="Activity ledger"
              description="Buys, sells, deposits, withdrawals, dividends, fees, transfers, and broker activity history."
              sample="sample_portfolio_transactions.csv"
              onFile={(file) => importCSV("/portfolio/import/transactions-csv", file)}
            />
          </div>
          <div className="mt-4 rounded-md border border-ink/10 bg-mint/50 p-3 text-sm leading-6 text-ink/65">
            CSV import is the MVP-safe path. Plaid Investments or another brokerage aggregator can be added behind the same holdings and transaction tables later, without changing the portfolio analytics UI.
          </div>
          <button onClick={syncPlaidInvestments} className="mt-4 rounded-md border border-moss/30 bg-mint/70 px-4 py-2 text-sm font-semibold text-moss">
            Sync Plaid Investments sample
          </button>
        </Card>

        <Card title="Recent portfolio imports">
          {imports.data.length ? (
            <div className="grid gap-2">
              {imports.data.slice(0, 5).map((item) => (
                <div key={item.id} className="grid gap-2 rounded-md border border-ink/10 p-3 text-sm md:grid-cols-[1fr_auto]">
                  <div>
                    <p className="font-semibold">{item.original_filename || item.import_type}</p>
                    <p className="text-ink/55">{item.import_type.replace("_", " ")} · {new Date(item.created_at).toLocaleString()}</p>
                  </div>
                  <div className="text-right text-ink/60">
                    <p>{item.imported_count}/{item.row_count} imported</p>
                    <p>{item.failed_count} failed</p>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <Empty text="No portfolio imports yet. Upload a holdings snapshot to populate this view." />
          )}
          {latestFailedImport && importErrors.data.length ? (
            <div className="mt-4 rounded-md border border-coral/25 bg-coral/10 p-3">
              <p className="text-sm font-semibold text-coral">Latest import errors</p>
              <div className="mt-2 grid gap-2">
                {importErrors.data.slice(0, 4).map((error) => (
                  <p key={error.id} className="text-xs text-ink/70">
                    Row {error.row_number} · {error.code}: {error.message}
                  </p>
                ))}
              </div>
            </div>
          ) : null}
        </Card>
      </div>

      <div className="mt-5 grid gap-5 xl:grid-cols-2">
        <Card title="Allocation by security type">
          <div className="h-72">
            <ResponsiveContainer>
              <PieChart>
                <Pie data={allocation.data.by_security_type} dataKey="value" nameKey="name">
                  {allocation.data.by_security_type.map((_, i) => <Cell key={i} fill={palette[i % palette.length]} />)}
                </Pie>
                <Tooltip formatter={(value) => money(Number(value))} />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </Card>
        <Card title="Top holdings">
          <div className="h-72">
            <ResponsiveContainer>
              <BarChart data={allocation.data.by_symbol?.slice(0, 8)}>
                <XAxis dataKey="name" />
                <YAxis tickFormatter={(value) => `$${Number(value) / 1000}k`} />
                <Tooltip formatter={(value) => money(Number(value))} />
                <Bar dataKey="value" fill="#315846" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </Card>
      </div>

      <div className="mt-5 grid gap-5 xl:grid-cols-[1.35fr_.65fr]">
        <Card title="Holdings">
          {holdings.data.length ? (
            <table className="w-full text-left text-sm">
              <thead className="text-xs uppercase tracking-wide text-ink/45">
                <tr><th className="py-2">Symbol</th><th>Type</th><th>Qty</th><th>Cost</th><th className="text-right">Value</th></tr>
              </thead>
              <tbody>
                {holdings.data.map((h) => (
                  <tr key={h.id} className="border-t border-ink/10">
                    <td className="py-3 font-medium">{h.symbol}<span className="block text-xs font-normal text-ink/45">{h.security_name}</span></td>
                    <td>{h.security_type}</td>
                    <td>{h.quantity}</td>
                    <td>{money(h.average_cost)}</td>
                    <td className="text-right">{money(h.market_value)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : <Empty text="No holdings yet. Import holdings CSV to populate this view." />}
        </Card>
        <Card title="Risk">
          {risk.data.length ? risk.data.map((r) => <div key={r.title} className="mb-3 rounded-md bg-gold/15 p-3 text-sm"><p className="font-medium">{r.title}</p><p className="text-ink/65">{r.explanation}</p></div>) : <Empty text="No concentration warnings." />}
        </Card>
      </div>

      <div className="mt-5 grid gap-5 xl:grid-cols-[1fr_.8fr]">
        <Card title="Recent portfolio activity">
          {sortedTransactions.length ? (
            <table className="w-full text-left text-sm">
              <thead className="text-xs uppercase tracking-wide text-ink/45">
                <tr><th className="py-2">Date</th><th>Type</th><th>Symbol</th><th className="text-right">Amount</th></tr>
              </thead>
              <tbody>
                {sortedTransactions.map((tx) => (
                  <tr key={tx.id} className="border-t border-ink/10">
                    <td className="py-3">{tx.occurred_at.slice(0, 10)}</td>
                    <td>{tx.transaction_type}</td>
                    <td>{tx.symbol || "Cash"}</td>
                    <td className="text-right">{money(tx.amount)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : <Empty text="No portfolio activity yet. Import an activity ledger CSV." />}
        </Card>
        <Card title="Import response">
          {result ? <pre className="max-h-80 overflow-auto rounded-lg bg-ink p-4 text-xs text-white">{result}</pre> : (
            <div className="rounded-md border border-dashed border-ink/15 p-5 text-sm leading-6 text-ink/55">
              <p className="mb-3 font-semibold text-moss">Awaiting import</p>
              Upload a CSV to see the raw import response, row counts, and any failed rows.
            </div>
          )}
        </Card>
      </div>
    </Shell>
  );
}

function ImportTile({ icon, title, description, sample, onFile }: { icon: ReactNode; title: string; description: string; sample: string; onFile: (file?: File) => void }) {
  return (
    <div className="rounded-md border border-ink/10 p-4">
      <div className="mb-3 flex items-center gap-2 font-semibold text-ink">{icon}{title}</div>
      <p className="min-h-20 text-sm leading-6 text-ink/60">{description}</p>
      <input type="file" accept=".csv" onChange={(e) => onFile(e.target.files?.[0])} className="mt-4 w-full text-sm" />
      <a className="mt-3 inline-flex items-center gap-2 text-sm font-medium text-moss" href={`/sample-data/${sample}`}>
        Download sample
      </a>
    </div>
  );
}
