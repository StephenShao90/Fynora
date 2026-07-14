"use client";

import { useCallback, useEffect, useState } from "react";
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
type Organization = { id: string; name: string; type?: string; currency?: string; role?: string };
type Member = { id?: string; organization_id: string; user_id: string; user_email: string; user_name?: string; role: string; created_at: string };

export default function Settings() {
  const { pushToast } = useToast();
  const [resetting, setResetting] = useState(false);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [members, setMembers] = useState<Member[]>([]);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("viewer");
  const org = organizations[0];

  const loadSessions = useCallback(async function loadSessions() {
    try {
      setSessions(await api<Session[]>("/api/v1/auth/sessions"));
    } catch {
      setSessions([]);
    }
  }, []);

  const loadMembers = useCallback(async function loadMembers(orgId: string) {
    try {
      setMembers(await api<Member[]>(`/api/v1/organizations/${orgId}/members`));
    } catch {
      setMembers([]);
    }
  }, []);

  const loadOrganizations = useCallback(async function loadOrganizations() {
    try {
      const rows = await api<Organization[]>("/api/v1/organizations");
      setOrganizations(rows);
      if (rows[0]?.id) await loadMembers(rows[0].id);
    } catch {
      setOrganizations([]);
      setMembers([]);
    }
  }, [loadMembers]);

  useEffect(() => {
    loadSessions();
    loadOrganizations();
  }, [loadOrganizations, loadSessions]);

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

  async function inviteMember() {
    if (!org?.id) {
      pushToast({ tone: "error", title: "No workspace found", detail: "Create a workspace from Onboarding first." });
      return;
    }
    try {
      await api<Member>(`/api/v1/organizations/${org.id}/members`, {
        method: "POST",
        body: JSON.stringify({ email: inviteEmail, role: inviteRole })
      });
      pushToast({ tone: "success", title: "Team member added", detail: `${inviteEmail} was added as ${inviteRole}.` });
      setInviteEmail("");
      await loadMembers(org.id);
    } catch (err) {
      pushToast({ tone: "error", title: "Could not add member", detail: (err as Error).message });
    }
  }

  async function updateMember(member: Member, role: string) {
    if (!org?.id) return;
    try {
      await api<Member>(`/api/v1/organizations/${org.id}/members/${member.user_id}`, { method: "PATCH", body: JSON.stringify({ role }) });
      pushToast({ tone: "success", title: "Role updated", detail: `${member.user_email} is now ${role}.` });
      await loadMembers(org.id);
    } catch (err) {
      pushToast({ tone: "error", title: "Could not update role", detail: (err as Error).message });
    }
  }

  async function removeMember(member: Member) {
    if (!org?.id) return;
    try {
      await api(`/api/v1/organizations/${org.id}/members/${member.user_id}`, { method: "DELETE" });
      pushToast({ tone: "success", title: "Member removed", detail: member.user_email });
      await loadMembers(org.id);
    } catch (err) {
      pushToast({ tone: "error", title: "Could not remove member", detail: (err as Error).message });
    }
  }

  return (
    <Shell>
      <Header title="Settings" subtitle="Workspace, team, profile, deployment, and session controls." />

      <div className="grid gap-4 xl:grid-cols-[.8fr_1.2fr]">
        <Card title="Workspace" guide={{ number: 1, title: "Workspace", body: "Review the active organization context, reset demo data to a known state, or log out of the local session." }}>
          <p className="text-lg font-semibold text-ink">{org?.name || "No workspace"}</p>
          <p className="mt-1 text-sm text-ink/55">{org?.type || "unknown"} · {org?.currency || "USD"} · {org?.role || "member"}</p>
          <p className="mt-4 text-sm leading-6 text-ink/60">Clearflow stores only demo/local data in this MVP. Do not enter brokerage credentials.</p>
          <div className="mt-4 flex flex-wrap gap-3">
            <button onClick={resetDemo} disabled={resetting} className="rounded-md border border-moss/30 bg-mint px-4 py-2 text-sm font-semibold text-moss disabled:opacity-50">{resetting ? "Resetting..." : "Reset demo data"}</button>
            <button onClick={logout} className="rounded-md bg-ink px-4 py-2 text-sm font-semibold text-white">Log out</button>
          </div>
        </Card>

        <Card title="Team" guide={{ number: 2, title: "Team", body: "Invite teammates and change roles. Owners are protected so you cannot accidentally remove the last owner." }}>
          <div className="grid gap-2 md:grid-cols-[1fr_160px_auto]">
            <input value={inviteEmail} onChange={(event) => setInviteEmail(event.target.value)} placeholder="teammate@example.com" className="rounded-md border border-ink/15 px-3 py-2 text-sm" />
            <select value={inviteRole} onChange={(event) => setInviteRole(event.target.value)} className="rounded-md border border-ink/15 px-3 py-2 text-sm">
              <option value="viewer">Viewer</option>
              <option value="analyst">Analyst</option>
              <option value="admin">Admin</option>
            </select>
            <button onClick={inviteMember} className="rounded-md bg-ink px-4 py-2 text-sm font-semibold text-white">Add member</button>
          </div>
          <div className="mt-4 grid gap-2">
            {members.length ? members.map((member) => (
              <div key={member.user_id} className="grid gap-2 rounded-md border border-ink/10 p-3 text-sm md:grid-cols-[1fr_150px_auto]">
                <div>
                  <p className="font-semibold">{member.user_name || member.user_email}</p>
                  <p className="text-ink/50">{member.user_email}</p>
                </div>
                <select value={member.role} onChange={(event) => updateMember(member, event.target.value)} className="rounded-md border border-ink/15 px-3 py-2 text-sm" disabled={member.role === "owner"}>
                  <option value="owner">Owner</option>
                  <option value="admin">Admin</option>
                  <option value="analyst">Analyst</option>
                  <option value="viewer">Viewer</option>
                </select>
                <button onClick={() => removeMember(member)} disabled={member.role === "owner"} className="rounded-md border border-coral/25 px-3 py-2 text-sm font-semibold text-coral disabled:opacity-40">Remove</button>
              </div>
            )) : <p className="text-sm text-ink/55">No members loaded.</p>}
          </div>
        </Card>
      </div>

      <div className="mt-5 grid gap-4 xl:grid-cols-[1fr_.9fr]">
        <Card title="Role model" guide={{ number: 3, title: "Role model", body: "Explains the RBAC levels Clearflow uses to separate owner/admin operations from analyst and viewer access." }}>
          <div className="grid gap-3 text-sm leading-6 text-ink/60 md:grid-cols-4">
            <p><span className="font-semibold text-ink">Owner</span><br />Billing, team, data, and all operations.</p>
            <p><span className="font-semibold text-ink">Admin</span><br />Can manage team and financial operations.</p>
            <p><span className="font-semibold text-ink">Analyst</span><br />Can reconcile, review reports, and inspect logs.</p>
            <p><span className="font-semibold text-ink">Viewer</span><br />Read-only access for advisors or officers.</p>
          </div>
        </Card>
        <Card title="Production readiness" guide={{ number: 4, title: "Production readiness", body: "Use this as the deployment checklist: frontend on Vercel, API/worker on backend host, secrets only on backend." }}>
          <ul className="grid gap-2 text-sm leading-6 text-ink/60">
            <li>Backend should run as API + worker services with hosted Postgres and Redis.</li>
            <li>Vercel should point `NEXT_PUBLIC_API_BASE_URL` at the deployed API.</li>
            <li>Provider secrets and webhook secrets belong on the backend host only.</li>
          </ul>
        </Card>
      </div>

      <div className="mt-5">
        <Card title="Sessions" guide={{ number: 5, title: "Sessions", body: "Shows refresh sessions. Revoke a session to test auth/session management and demonstrate account security controls." }}>
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
