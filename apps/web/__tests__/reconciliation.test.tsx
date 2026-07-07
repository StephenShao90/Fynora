import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ReconciliationMatches } from "@/components/reconciliation/ReconciliationMatches";

describe("reconciliation match intelligence", () => {
  it("renders status score and reasons", () => {
    render(<ReconciliationMatches loading={false} error="" matches={[{ id: "m1", processorPayoutId: "po_1", bankDepositId: "bank_1", status: "likely_match", confidenceScore: 0.87, amountDifferenceMinor: 212, currency: "USD", reasons: ["same_currency", "date_within_window"], explanation: "Likely match." }]} />);
    expect(screen.getByText("likely match")).toBeInTheDocument();
    expect(screen.getByText("87% confidence")).toBeInTheDocument();
    expect(screen.getByText("same_currency")).toBeInTheDocument();
  });
});
