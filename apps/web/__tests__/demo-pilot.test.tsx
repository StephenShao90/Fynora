import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DemoPilot } from "@/components/demo";

const apiMock = vi.fn();

vi.mock("@/lib/api", () => ({
  api: (...args: unknown[]) => apiMock(...args)
}));

describe("DemoPilot", () => {
  beforeEach(() => {
    apiMock.mockReset();
    apiMock.mockResolvedValue({});
  });

  it("runs the guided demo workflow in order", async () => {
    render(<DemoPilot />);

    fireEvent.click(screen.getByRole("button", { name: "Run full demo setup" }));

    await waitFor(() => expect(screen.getByText(/Ready at/)).toBeInTheDocument());
    expect(apiMock).toHaveBeenCalledTimes(4);
    expect(apiMock.mock.calls.map((call) => call[0])).toEqual([
      "/api/v1/onboarding/status",
      "/sync/stripe",
      "/sync/bank",
      "/reconciliation/runs"
    ]);
    expect(screen.getAllByText("passed")).toHaveLength(4);
  });

  it("marks the active step failed when the workflow stops", async () => {
    apiMock.mockResolvedValueOnce({}).mockRejectedValueOnce(new Error("stripe unavailable"));
    render(<DemoPilot />);

    fireEvent.click(screen.getByRole("button", { name: "Run full demo setup" }));

    await waitFor(() => expect(screen.getByText("stripe unavailable")).toBeInTheDocument());
    expect(screen.getByText("failed")).toBeInTheDocument();
    expect(apiMock).toHaveBeenCalledTimes(2);
  });
});
