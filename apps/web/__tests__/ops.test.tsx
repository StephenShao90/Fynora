import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import OpsPage from "@/app/ops/page";

vi.mock("next/navigation", () => ({ usePathname: () => "/ops" }));
vi.mock("@/lib/api", () => ({
  activeDemoScenario: () => ({ id: "student_org", name: "Demo Organization", type: "student_organization", currency: "USD" }),
  setDemoScenario: () => {},
  isDemoFallbackMode: () => false,
  logout: () => {},
  api: (path: string) => {
    if (path === "/organizations") return Promise.resolve([{ id: "org_1", name: "Demo Organization" }]);
    if (path.startsWith("/api/v1/jobs")) return Promise.resolve({ data: [{ id: "job_123456789", type: "reconciliation.run", status: "completed", attempts: 1, max_attempts: 3, created_at: "2026-07-14T12:00:00Z", updated_at: "2026-07-14T12:01:00Z" }] });
    if (path.startsWith("/api/v1/audit-logs")) return Promise.resolve({ data: [{ id: "audit_1", action: "job.completed", target_type: "job", target_id: "job_123456789", created_at: "2026-07-14T12:01:00Z" }] });
    if (path === "/api/v1/ops/metrics") return Promise.resolve({ http_requests_total: 42, jobs_queued_total: 1, jobs_completed_total: 5, job_queue_depth: 0, stripe_webhook_events_total: 1, plaid_webhooks_received_total: 1, idempotency_replays_total: 2 });
    return Promise.resolve({});
  }
}));

describe("ops page", () => {
  it("renders bank-grade control evidence from operational data", async () => {
    render(<OpsPage />);

    await waitFor(() => expect(screen.getByText("Bank-grade control evidence")).toBeInTheDocument());
    await waitFor(() => expect(screen.getByText("100%")).toBeInTheDocument());
    expect(screen.getByText("Worker health")).toBeInTheDocument();
    expect(screen.getByText("Financial write safety")).toBeInTheDocument();
    expect(screen.getByText("Auditability")).toBeInTheDocument();
    expect(screen.getByText("Provider event handling")).toBeInTheDocument();
    expect(screen.getByText("Job durability")).toBeInTheDocument();
  });
});
