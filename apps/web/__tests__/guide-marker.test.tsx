import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { GuideMarker } from "@/components/help";
import { GUIDE_MODE_EVENT } from "@/components/help";

describe("GuideMarker", () => {
  beforeEach(() => {
    window.__clearflowGuideMode = false;
  });

  function enableGuideMode() {
    act(() => {
      window.__clearflowGuideMode = true;
      window.dispatchEvent(new CustomEvent(GUIDE_MODE_EVENT, { detail: { active: true } }));
    });
  }

  it("stays hidden until guide mode is enabled", async () => {
    render(<GuideMarker guide={{ number: 1, title: "Cash-flow forecast", body: "Use this chart to project cash." }} />);
    expect(screen.queryByRole("button", { name: "1. Cash-flow forecast" })).not.toBeInTheDocument();

    enableGuideMode();
    await waitFor(() => expect(screen.getByRole("button", { name: "1. Cash-flow forecast" })).toBeInTheDocument());
  });

  it("shows guidance on hover and hides on hover off", () => {
    render(<GuideMarker guide={{ number: 1, title: "Cash-flow forecast", body: "Use this chart to project cash." }} />);
    enableGuideMode();
    const marker = screen.getByRole("button", { name: "1. Cash-flow forecast" });

    fireEvent.mouseEnter(marker);
    expect(screen.getByRole("tooltip")).toHaveTextContent("Use this chart to project cash.");

    fireEvent.mouseLeave(marker);
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });

  it("pins guidance on click and closes on blur", () => {
    render(<GuideMarker guide={{ number: 2, title: "Exception queue", body: "Review open breaks here." }} />);
    enableGuideMode();
    const marker = screen.getByRole("button", { name: "2. Exception queue" });

    fireEvent.click(marker);
    expect(screen.getByRole("tooltip")).toHaveTextContent("Review open breaks here.");

    fireEvent.blur(marker);
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });
});
