"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { getApiCache, invalidateApiCache, setApiCache } from "@/lib/api-cache";

const CACHE_TTL_MS = 30000;

type UseApiOptions = {
  instant?: boolean;
};

export function useApi<T>(path: string, fallback: T, options: UseApiOptions = {}) {
  const cached = getApiCache<T>(path);
  const [data, setData] = useState<T>(cached?.data !== undefined ? cached.data as T : fallback);
  const [loading, setLoading] = useState(options.instant ? false : cached?.data === undefined);
  const [error, setError] = useState(cached?.data !== undefined ? "" : cached?.error || "");

  useEffect(() => {
    let cancelled = false;
    const cached = getApiCache<T>(path);
    const fresh = cached?.data !== undefined && Date.now() - cached.updatedAt < CACHE_TTL_MS;

    if (fresh) {
      setData(cached.data as T);
      setError("");
      setLoading(false);
      return () => {
        cancelled = true;
      };
    }

    if (!options.instant) setLoading(true);
    setError("");
    const request = cached?.promise as Promise<T> | undefined || api<T>(path);
    setApiCache(path, { ...cached, promise: request, updatedAt: cached?.updatedAt || 0 });

    request
      .then((result) => {
        setApiCache(path, { data: result, updatedAt: Date.now() });
        if (!cancelled) setData(result);
      })
      .catch((err) => {
        const message = err instanceof Error ? err.message : "Request failed";
        setApiCache(path, { data: cached?.data, error: message, updatedAt: cached?.updatedAt || 0 });
        if (!cancelled) setError(message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [options.instant, path]);

  return { data, loading, error };
}

export { invalidateApiCache };
