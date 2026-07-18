import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import Home from "@/app/page";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() })
}));

vi.mock("@/lib/api", () => ({
  api: () => Promise.resolve({ token: "demo-token" }),
  clearAuth: vi.fn(),
  setToken: vi.fn()
}));

describe("home positioning", () => {
  it("leads with the restaurant revenue close use case", () => {
    render(<Home />);

    expect(screen.getByText("Daily revenue close for restaurants and small teams")).toBeInTheDocument();
    expect(screen.getByText(/POS sales, delivery payouts, refunds, fees, and bank deposits/)).toBeInTheDocument();
    expect(screen.getByText("Delivery payout short by $12.00")).toBeInTheDocument();
  });
});
