"use client";

import { useState } from "react";
import { Card, Shell } from "@/components/Shell";
import { Header } from "@/components/Common";
import { useToast } from "@/components/ToastProvider";
import { api, logout } from "@/lib/api";

export default function Settings() {
  const { pushToast } = useToast();
  const [resetting, setResetting] = useState(false);

  async function resetDemo() {
    setResetting(true);
    try {
      await api("/debug/clearflow/reset-demo", { method: "POST", body: "{}" });
      pushToast({ tone: "success", title: "Demo data reset", detail: "The local scenario is back to a known reconciliation state." });
      window.setTimeout(() => window.location.assign("/dashboard"), 800);
    } catch (err) {
      pushToast({ tone: "error", title: "Could not reset demo", detail: (err as Error).message });
    } finally {
      setResetting(false);
    }
  }

  return (
    <Shell>
      <Header title="Settings" subtitle="Profile and session controls." />
      <Card>
        <p className="text-sm text-ink/60">Clearflow stores only demo/local data in this MVP. Do not enter brokerage credentials.</p>
        <div className="mt-4 flex flex-wrap gap-3">
          <button onClick={resetDemo} disabled={resetting} className="rounded-md border border-moss/30 bg-mint px-4 py-2 text-sm font-semibold text-moss disabled:opacity-50">{resetting ? "Resetting..." : "Reset demo data"}</button>
          <button onClick={logout} className="rounded-md bg-ink px-4 py-2 text-sm font-semibold text-white">Log out</button>
        </div>
      </Card>
    </Shell>
  );
}
