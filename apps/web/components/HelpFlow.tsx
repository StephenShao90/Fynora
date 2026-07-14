"use client";

import { useEffect, useState } from "react";

declare global {
  interface Window {
    __clearflowGuideMode?: boolean;
  }
}

export const GUIDE_MODE_EVENT = "clearflow-guide-mode";

export function HelpFlow({ page }: { page: string }) {
  const [active, setActive] = useState(false);

  useEffect(() => {
    window.__clearflowGuideMode = false;
    window.dispatchEvent(new CustomEvent(GUIDE_MODE_EVENT, { detail: { active: false } }));
  }, [page]);

  function toggleGuideMode() {
    const next = !active;
    setActive(next);
    window.__clearflowGuideMode = next;
    window.dispatchEvent(new CustomEvent(GUIDE_MODE_EVENT, { detail: { active: next } }));
  }

  return (
    <button
      type="button"
      onClick={toggleGuideMode}
      className={`inline-flex h-9 w-9 items-center justify-center rounded-full border text-sm font-semibold shadow-sm ${active ? "border-[#5ea8ff] bg-[#5ea8ff] text-black" : "border-ink/15 bg-white text-ink/70 hover:bg-ink/[0.03]"}`}
      aria-label={`${active ? "Hide" : "Show"} help markers for ${page}`}
      aria-pressed={active}
      title={`${active ? "Hide" : "Show"} help markers for ${page}`}
    >
      ?
    </button>
  );
}
