"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

export function useApi<T>(path: string, fallback: T) {
  const [data, setData] = useState<T>(fallback);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    api<T>(path)
      .then(setData)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [path]);

  return { data, loading, error };
}
