"use client";

import { useEffect, useState } from "react";
import { Card, Shell } from "@/components/Shell";
import { Header } from "@/components/Common";
import { useToast } from "@/components/ToastProvider";
import { api, logout } from "@/lib/api";

type Session = {
  id: string;
  created_at: string;
  last_used_at?: string;
  expires_at: string;
  revoked_at?: string;
  user_agent?: string;
};

export default function Settings() {
  const { pushToast } = useToast();
  const [resetting, setResetting] = useState(false);
  const [sessions, setSessions] = useState<Session[]>([]);

  useEffect(() => {
    loadSessions();
  }, []);

  async function loadSessions() {
    try {
      setSessions(await api<Session[]>("/api/v1/auth/sessions"));
    } catch {
      setSessions([]);
    }
  }

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

  async function revokeSession(sessionId: string) {
    try {
      await api(`/api/v1/auth/sessions/${sessionId}`, { method: "DELETE" });
      pushToast({ tone: "success", title: "Session revoked", detail: "That refresh session can no longer be used." });
      await loadSessions();
    } catch (err) {
      pushToast({ tone: "error", title: "Could not revoke session", detail: (err as Error).message });
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
      <div className="mt-5">
        <Card title="Sessions">
          {sessions.length ? (
            <div className="grid gap-2">
              {sessions.map((session) => (
                <div key={session.id} className="grid gap-2 rounded-md border border-ink/10 p-3 text-sm md:grid-cols-[1fr_auto]">
                  <div>
                    <p className="font-semibold">{session.revoked_at ? "Revoked session" : "Active session"}</p>
                    <p className="text-ink/55">Created {new Date(session.created_at).toLocaleString()} · Expires {new Date(session.expires_at).toLocaleString()}</p>
                    {session.user_agent ? <p className="mt-1 truncate text-xs text-ink/45">{session.user_agent}</p> : null}
                  </div>
                  <button onClick={() => revokeSession(session.id)} disabled={Boolean(session.revoked_at)} className="rounded-md border border-ink/15 px-3 py-2 text-sm font-semibold text-ink disabled:opacity-40">
                    Revoke
                  </button>
                </div>
              ))}
            </div>
          ) : <p className="text-sm text-ink/55">No refresh sessions found for this user.</p>}
        </Card>
      </div>
    </Shell>
  );
}
