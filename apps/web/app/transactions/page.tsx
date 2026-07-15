"use client";

import { useMemo, useState } from "react";
import { Card, Shell, money } from "@/components/layout";
import { Empty, Header } from "@/components/layout";
import { GuideMarker } from "@/components/help";
import { useToast } from "@/components/layout";
import { api } from "@/lib/api";
import { useApi } from "@/hooks/useApi";
import type { Transaction } from "@/types";

const categories = ["income", "software", "facilities", "food", "travel", "payroll", "marketing", "uncategorized"];

export default function Transactions() {
  const { pushToast } = useToast();
  const [reload, setReload] = useState(0);
  const [query, setQuery] = useState("");
  const [direction, setDirection] = useState("");
  const [selectedId, setSelectedId] = useState("");
  const rows = useApi<Transaction[]>(`/transactions?reload=${reload}`, []);
  const filtered = useMemo(() => rows.data.filter((row) => {
    const haystack = `${row.normalized_merchant} ${row.merchant} ${row.description} ${row.category}`.toLowerCase();
    return (!query || haystack.includes(query.toLowerCase())) && (!direction || row.direction === direction);
  }), [direction, query, rows.data]);
  const selected = filtered.find((row) => row.id === selectedId) || filtered[0];

  async function updateCategory(row: Transaction, category: string) {
    try {
      await api(`/transactions/${row.id}/category`, { method: "PATCH", body: JSON.stringify({ category }) });
      pushToast({ tone: "success", title: "Category updated", detail: `${row.normalized_merchant || row.merchant} -> ${category}` });
      setReload((value) => value + 1);
    } catch (err) {
      pushToast({ tone: "error", title: "Could not update category", detail: (err as Error).message });
    }
  }

  return (
    <Shell>
      <Header title="Transactions" subtitle="Search, inspect, and categorize normalized bank records." />

      <div className="mb-2 flex items-center justify-between"><p className="text-xs font-semibold uppercase tracking-wide text-ink/45">Search and filters</p><GuideMarker guide={{ number: 1, title: "Search and filters", body: "Use search and direction filters to narrow the ledger by merchant, description, category, credit, or debit." }} /></div>
      <div className="mb-4 grid gap-3 md:grid-cols-[1fr_180px]">
        <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search merchant, description, or category" className="rounded-md border border-ink/15 bg-white px-3 py-2 text-sm" />
        <select value={direction} onChange={(event) => setDirection(event.target.value)} className="rounded-md border border-ink/15 bg-white px-3 py-2 text-sm">
          <option value="">All directions</option>
          <option value="credit">Credits</option>
          <option value="debit">Debits</option>
        </select>
      </div>

      <div className="grid gap-4 xl:grid-cols-[1.15fr_.85fr]">
        <Card title="Ledger" guide={{ number: 2, title: "Transaction ledger", body: "Click any row to inspect it. This table is the normalized transaction record used by cash-flow insights." }}>
          {filtered.length ? (
            <div className="overflow-x-auto">
              <table className="min-w-full text-left text-sm">
                <thead className="border-b border-ink/10 text-xs uppercase tracking-wide text-ink/45">
                  <tr><th className="py-2 pr-4">Date</th><th className="pr-4">Merchant</th><th className="pr-4">Category</th><th className="pr-4">Direction</th><th className="text-right">Amount</th></tr>
                </thead>
                <tbody>
                  {filtered.map((t) => (
                    <tr key={t.id} onClick={() => setSelectedId(t.id)} className={`cursor-pointer border-b border-ink/10 last:border-0 ${selected?.id === t.id ? "bg-mint/50" : "hover:bg-ink/[0.03]"}`}>
                      <td className="py-3 pr-4">{t.occurred_at?.slice(0, 10)}</td>
                      <td className="pr-4"><p className="font-medium">{t.normalized_merchant || t.merchant}</p><p className="text-xs text-ink/45">{t.description}</p></td>
                      <td className="pr-4">{t.category}</td>
                      <td className="pr-4">{t.direction}</td>
                      <td className={`text-right font-medium ${t.direction === "debit" ? "text-coral" : "text-moss"}`}>{money(t.amount)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : <Empty text="No transactions match the current filters. Try the demo flow or upload the sample CSV." />}
        </Card>

        <Card title="Transaction detail" guide={{ number: 3, title: "Transaction detail", body: "Validate merchant normalization and change category here. Category edits feed future spending insights." }}>
          {selected ? (
            <div className="grid gap-4">
              <div>
                <p className="text-xl font-semibold">{selected.normalized_merchant || selected.merchant}</p>
                <p className={`mt-1 text-2xl font-semibold ${selected.direction === "debit" ? "text-coral" : "text-moss"}`}>{money(selected.amount)}</p>
              </div>
              <dl className="grid gap-2 text-sm">
                <Detail label="Description" value={selected.description} />
                <Detail label="Original merchant" value={selected.merchant} />
                <Detail label="Date" value={selected.occurred_at} />
                <Detail label="Direction" value={selected.direction} />
                <Detail label="Currency" value={selected.currency} />
              </dl>
              <label className="grid gap-1 text-sm font-medium">
                Category
                <select value={selected.category || "uncategorized"} onChange={(event) => updateCategory(selected, event.target.value)} className="rounded-md border border-ink/15 px-3 py-2 font-normal">
                  {categories.map((category) => <option key={category} value={category}>{category}</option>)}
                </select>
              </label>
              <div className="rounded-md bg-ink/[0.03] p-3 text-sm leading-6 text-ink/60">
                Use this detail panel to validate normalization. Category edits update the backend and feed future spending insights.
              </div>
            </div>
          ) : <Empty text="Select a transaction to inspect details." />}
        </Card>
      </div>
    </Shell>
  );
}

function Detail({ label, value }: { label: string; value?: string }) {
  return <div className="flex justify-between gap-3 border-b border-ink/10 py-2 last:border-0"><dt className="text-ink/45">{label}</dt><dd className="text-right font-medium text-ink/75">{value || "Not available"}</dd></div>;
}
