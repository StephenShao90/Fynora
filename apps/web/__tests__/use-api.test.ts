import { describe, expect, it } from "vitest";
import { normalizeApiData } from "@/hooks/useApi";

describe("API data normalization", () => {
  it("keeps list consumers safe when an endpoint returns null", () => {
    expect(normalizeApiData(null, [])).toEqual([]);
    expect(normalizeApiData(undefined, [])).toEqual([]);
  });

  it("falls back for nullable object responses", () => {
    expect(normalizeApiData(null, { connected: false })).toEqual({ connected: false });
  });

  it("preserves valid API responses", () => {
    expect(normalizeApiData([{ id: "row-1" }], [])).toEqual([{ id: "row-1" }]);
    expect(normalizeApiData({ connected: true }, { connected: false })).toEqual({ connected: true });
  });
});
