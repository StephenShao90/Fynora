"use client";

import { Card, Shell, money } from "@/components/Shell";
import { Empty, Header } from "@/components/Common";
import { useApi } from "@/hooks/useApi";
import type { Transaction } from "@/types";

export default function Transactions() {
  const rows = useApi<Transaction[]>("/transactions", []);
  return (
    <Shell>
      <Header title="Transactions" subtitle="Searchable normalized spending records with categories and merchants." />
      <Card>
        {rows.data.length ? <table className="w-full text-left text-sm"><thead className="text-ink/50"><tr><th className="py-2">Date</th><th>Merchant</th><th>Category</th><th>Direction</th><th className="text-right">Amount</th></tr></thead><tbody>{rows.data.map((t) => <tr key={t.id} className="border-t border-ink/10"><td className="py-3">{t.occurred_at?.slice(0,10)}</td><td>{t.normalized_merchant}</td><td>{t.category}</td><td>{t.direction}</td><td className="text-right">{money(t.amount)}</td></tr>)}</tbody></table> : <Empty text="No transactions yet. Try the demo flow or upload the sample CSV." />}
      </Card>
    </Shell>
  );
}
