import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CashflowForecastCard } from "@/components/insights/CashflowForecastCard";
import { AnomalyList } from "@/components/insights/AnomalyList";
import { CashRecommendations } from "@/components/insights/CashRecommendations";

describe("intelligence components", () => {
  it("renders a cash-flow forecast", () => {
    render(
      <CashflowForecastCard
        horizon={30}
        onHorizonChange={() => {}}
        loading={false}
        error=""
        forecast={{
          organizationId: "org",
          horizonDays: 30,
          startingBalanceMinor: 296727,
          projectedEndingBalanceMinor: 241727,
          currency: "USD",
          confidence: "medium",
          assumptions: ["Uses recent bank activity."],
          series: [{ date: "2026-07-08", projectedBalanceMinor: 290000, expectedInflowsMinor: 0, expectedOutflowsMinor: 5000 }]
        }}
      />
    );
    expect(screen.getByText("Cash-flow forecast")).toBeInTheDocument();
    expect(screen.getByText("Projected ending")).toBeInTheDocument();
    expect(screen.getByText("medium")).toBeInTheDocument();
  });

  it("renders anomaly severity and recommended action", () => {
    render(<AnomalyList loading={false} error="" anomalies={[{ id: "a1", type: "missing_payout", severity: "high", title: "Expected payout has not reached the bank", description: "No matching bank deposit.", resourceType: "processor_payout", resourceId: "po_1", detectedAt: "2026-07-06T12:00:00Z", recommendedAction: "Check processor payout status." }]} />);
    expect(screen.getByText("high")).toBeInTheDocument();
    expect(screen.getByText("Check processor payout status.")).toBeInTheDocument();
  });

  it("renders cash recommendation priority and title", () => {
    render(<CashRecommendations loading={false} error="" recommendations={[{ type: "reserve", priority: "high", title: "Keep at least 30 days of operating cash", description: "Maintain a reserve.", amountMinor: 51245, currency: "USD" }]} />);
    expect(screen.getByText("high")).toBeInTheDocument();
    expect(screen.getByText("Keep at least 30 days of operating cash")).toBeInTheDocument();
  });
});
