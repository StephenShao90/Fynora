import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { GuideMarker, HelpFlow } from "@/components/help";

describe("HelpFlow", () => {
  it("opens, advances, returns, and closes the page slideshow", () => {
    render(<HelpFlow page="Payout reconciliation" />);

    fireEvent.click(screen.getByRole("button", { name: "Open Payout reconciliation help slideshow" }));
    expect(screen.getByRole("dialog", { name: "Payout reconciliation help slideshow" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Match expected cash to posted cash" })).toBeInTheDocument();
    expect(screen.getAllByText("Stripe payout").length).toBeGreaterThan(1);

    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.getByRole("heading", { name: "Resolve breaks with evidence" })).toBeInTheDocument();
    expect(screen.getAllByText("Open break").length).toBeGreaterThan(1);

    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(screen.getByRole("heading", { name: "Match expected cash to posted cash" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Close help slideshow" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("uses a generic slideshow for pages without custom slides", () => {
    render(<HelpFlow page="Unknown page" />);

    fireEvent.click(screen.getByRole("button", { name: "Open Unknown page help slideshow" }));
    expect(screen.getByRole("heading", { name: "Start with the close question" })).toBeInTheDocument();
    expect(screen.getAllByText("Processor payouts").length).toBeGreaterThan(1);
  });
});

describe("GuideMarker", () => {
  it("keeps legacy guide props inert now that help is centralized", () => {
    const { container } = render(<GuideMarker guide={{ number: 1, title: "Old marker", body: "Hidden by design" }} />);
    expect(container).toBeEmptyDOMElement();
  });
});
