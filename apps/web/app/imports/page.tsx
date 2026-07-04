"use client";

import { useState } from "react";
import { Card, Shell } from "@/components/Shell";
import { Header } from "@/components/Common";
import { upload } from "@/lib/api";

const uploads = [
  ["Transactions CSV", "/imports/transactions-csv", "sample_transactions.csv"],
  ["Holdings CSV", "/portfolio/import/holdings-csv", "sample_holdings.csv"],
  ["Portfolio transactions CSV", "/portfolio/import/transactions-csv", "sample_portfolio_transactions.csv"]
];

export default function Imports() {
  const [result, setResult] = useState("");
  async function send(path: string, file?: File) {
    if (!file) return;
    const res = await upload(path, file);
    setResult(JSON.stringify(res, null, 2));
  }
  return (
    <Shell>
      <Header title="Imports" subtitle="Upload bank transactions and brokerage CSV exports without sharing credentials." />
      <div className="grid gap-4 lg:grid-cols-3">
        {uploads.map(([label, path, sample]) => (
          <Card key={path} title={label}>
            <input type="file" accept=".csv" onChange={(e) => send(path, e.target.files?.[0])} className="w-full text-sm" />
            <a className="mt-4 block text-sm text-moss" href={`/sample-data/${sample}`}>Download {sample}</a>
          </Card>
        ))}
      </div>
      {result ? <pre className="mt-5 overflow-auto rounded-lg bg-ink p-4 text-xs text-white">{result}</pre> : null}
    </Shell>
  );
}
