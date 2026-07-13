import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import IntegrationsPage from "@/app/integrations/page";

vi.mock("next/navigation", () => ({ usePathname: () => "/integrations" }));
vi.mock("@/hooks/useApi", () => ({
  useApi: () => ({ data: [{ id: "plaid_1", institution_name: "Plaid Test Bank", products: ["transactions"], last_synced_at: "2026-07-06T12:00:00Z" }], loading: false, error: "" })
}));
vi.mock("@/lib/api", () => ({
  activeDemoScenario: () => ({ id: "student_org", name: "Demo Organization", type: "student_organization", currency: "USD" }),
  setDemoScenario: () => {},
  api: () => Promise.resolve({}),
  getStripeStatus: () => Promise.resolve({ connected: true, provider: "stripe", accountId: "acct_123", displayName: "Stripe Test Account", connectedAt: "2026-07-06T12:00:00Z" }),
  getStripeConnectUrl: () => Promise.resolve({ url: "https://connect.stripe.com/oauth/authorize", state: "state" }),
  disconnectStripe: () => Promise.resolve({ connected: false, provider: "stripe" }),
  isDemoFallbackMode: () => false,
  logout: () => {}
}));

describe("integrations page", () => {
  it("shows Stripe and Plaid connection states", async () => {
    render(<IntegrationsPage />);
    expect(screen.getByRole("heading", { name: "Integrations", level: 1 })).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText("Stripe Test Account")).toBeInTheDocument());
    expect(screen.getByText("Plaid Test Bank")).toBeInTheDocument();
    expect(screen.getAllByText("connected").length).toBeGreaterThan(0);
  });
});
