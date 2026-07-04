"use client";

import { Card, Shell } from "@/components/Shell";
import { Header } from "@/components/Common";
import { logout } from "@/lib/api";

export default function Settings() {
  return (
    <Shell>
      <Header title="Settings" subtitle="Profile and session controls." />
      <Card>
        <p className="text-sm text-ink/60">Fynora stores only demo/local data in this MVP. Do not enter brokerage credentials.</p>
        <button onClick={logout} className="mt-4 rounded-md bg-ink px-4 py-2 text-white">Log out</button>
      </Card>
    </Shell>
  );
}
