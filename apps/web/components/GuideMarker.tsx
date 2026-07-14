"use client";

import { useEffect, useState } from "react";
import { GUIDE_MODE_EVENT } from "./HelpFlow";

export type Guide = {
  number: number;
  title: string;
  body: string;
};

export function GuideMarker({ guide }: { guide: Guide }) {
  const [visible, setVisible] = useState(false);
  const [hovered, setHovered] = useState(false);
  const [pinned, setPinned] = useState(false);
  const open = hovered || pinned;

  useEffect(() => {
    setVisible(Boolean(window.__clearflowGuideMode));
    function onGuideMode(event: Event) {
      const active = Boolean((event as CustomEvent<{ active: boolean }>).detail?.active);
      setVisible(active);
      if (!active) {
        setHovered(false);
        setPinned(false);
      }
    }
    window.addEventListener(GUIDE_MODE_EVENT, onGuideMode);
    return () => window.removeEventListener(GUIDE_MODE_EVENT, onGuideMode);
  }, []);

  if (!visible) return null;

  return (
    <span className="relative inline-flex">
      <button
        type="button"
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
        onFocus={() => setHovered(true)}
        onBlur={() => {
          setHovered(false);
          setPinned(false);
        }}
        onClick={() => setPinned((value) => !value)}
        className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-ink/20 bg-[#5ea8ff] text-sm font-bold text-black shadow-sm"
        aria-label={`${guide.number}. ${guide.title}`}
      >
        {guide.number}
      </button>
      {open ? (
        <span
          role="tooltip"
          className="absolute right-0 top-9 z-40 w-72 rounded-md border border-ink/15 bg-white p-3 text-left shadow-panel"
        >
          <span className="block text-sm font-semibold text-ink">{guide.title}</span>
          <span className="mt-1 block text-sm font-normal leading-6 text-ink/65">{guide.body}</span>
        </span>
      ) : null}
    </span>
  );
}
