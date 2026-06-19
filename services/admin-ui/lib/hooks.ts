"use client";
import { useCallback, useEffect, useRef, useState } from "react";

type FetchState<T> = {
  data: T | null;
  loading: boolean;
  error: string;
  refresh: () => void;
};

type Options = {
  /** If set, the fetcher is re-invoked automatically every `pollMs` milliseconds. */
  pollMs?: number;
};

/**
 * Fires `fetcher` on mount and whenever `refresh()` is called.
 * Handles cancellation — no stale state updates after unmount.
 */
export function useDataFetch<T>(fetcher: () => Promise<T>, options: Options = {}): FetchState<T> {
  const { pollMs } = options;
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [refreshKey, setRefreshKey] = useState(0);
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    fetcherRef
      .current()
      .then((d) => {
        if (!cancelled) { setData(d); setError(""); setLoading(false); }
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : String(e));
          setLoading(false);
        }
      });
    return () => { cancelled = true; };
  }, [refreshKey]);

  useEffect(() => {
    if (!pollMs) return;
    const id = setInterval(() => setRefreshKey((k) => k + 1), pollMs);
    return () => clearInterval(id);
  }, [pollMs]);

  const refresh = useCallback(() => setRefreshKey((k) => k + 1), []);

  return { data, loading, error, refresh };
}
