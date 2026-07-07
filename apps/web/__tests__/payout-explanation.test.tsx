import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PayoutExplanationPanel } from "@/components/payouts/PayoutExplanationPanel";

describe("payout explanation", () => {
  it("renders payout warnings", () => {
    render(<PayoutExplanationPanel loading={false} error="" explanation={{ payoutId: "po_1", processor: "stripe", grossAmountMinor: 209872, feesMinor: 25812, refundsMinor: 17500, netAmountMinor: 166572, currency: "USD", summary: "Payout summary", lineItems: [], warnings: ["processor fees are elevated"] }} />);
    expect(screen.getByText("Payout explanation")).toBeInTheDocument();
    expect(screen.getByText("Warnings")).toBeInTheDocument();
    expect(screen.getByText("processor fees are elevated")).toBeInTheDocument();
  });
});
