"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

type CacheEntry<T> = {
  data?: T;
  error?: string;
  promise?: Promise<T>;
  updatedAt: number;
};

const CACHE_TTL_MS = 30000;
const apiCache = new Map<string, CacheEntry<unknown>>();

export function useApi<T>(path: string, fallback: T) {
  const cached = apiCache.get(path) as CacheEntry<T> | undefined;
  const [data, setData] = useState<T>(cached?.data !== undefined ? cached.data as T : fallback);
  const [loading, setLoading] = useState(cached?.data === undefined);
  const [error, setError] = useState(cached?.data !== undefined ? "" : cached?.error || "");

  useEffect(() => {
    let cancelled = false;
    const cached = apiCache.get(path) as CacheEntry<T> | undefined;
    const fresh = cached?.data !== undefined && Date.now() - cached.updatedAt < CACHE_TTL_MS;

    if (fresh) {
      setData(cached.data as T);
      setError("");
      setLoading(false);
      return () => {
        cancelled = true;
      };
    }

    setLoading(true);
    setError("");
    const request = cached?.promise as Promise<T> | undefined || api<T>(path);
    apiCache.set(path, { ...cached, promise: request, updatedAt: cached?.updatedAt || 0 });

    request
      .then((result) => {
        apiCache.set(path, { data: result, updatedAt: Date.now() });
        if (!cancelled) setData(result);
      })
      .catch((err) => {
        const message = err instanceof Error ? err.message : "Request failed";
        apiCache.set(path, { data: cached?.data, error: message, updatedAt: cached?.updatedAt || 0 });
        if (!cancelled) setError(message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [path]);

  return { data, loading, error };
}

export function invalidateApiCache(match?: string | RegExp) {
  if (!match) {
    apiCache.clear();
    return;
  }
  for (const key of apiCache.keys()) {
    if (typeof match === "string" ? key.includes(match) : match.test(key)) apiCache.delete(key);
  }
}
