import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { GuideMarker } from "@/components/GuideMarker";

describe("GuideMarker", () => {
  it("shows guidance on hover and hides on hover off", () => {
    render(<GuideMarker guide={{ number: 1, title: "Cash-flow forecast", body: "Use this chart to project cash." }} />);
    const marker = screen.getByRole("button", { name: "1. Cash-flow forecast" });

    fireEvent.mouseEnter(marker);
    expect(screen.getByRole("tooltip")).toHaveTextContent("Use this chart to project cash.");

    fireEvent.mouseLeave(marker);
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });

  it("pins guidance on click and closes on blur", () => {
    render(<GuideMarker guide={{ number: 2, title: "Exception queue", body: "Review open breaks here." }} />);
    const marker = screen.getByRole("button", { name: "2. Exception queue" });

    fireEvent.click(marker);
    expect(screen.getByRole("tooltip")).toHaveTextContent("Review open breaks here.");

    fireEvent.blur(marker);
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });
});
