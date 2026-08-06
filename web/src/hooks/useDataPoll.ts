import { useCallback, useEffect, useRef, useState } from "react";

export interface PollState {
  /** True until the first response (success or failure) arrives. */
  loading: boolean;
  /** Message from the most recent failed fetch, else null. */
  error: string | null;
  /** True when we hold a previous good snapshot but the latest fetch
   *  failed — i.e. the data on screen may be out of date. */
  stale: boolean;
  /** Epoch ms of the last successful fetch, or null. */
  lastUpdated: number | null;
  /** Force an immediate refresh outside the interval. */
  refresh: () => void;
}

interface Options<T> {
  /** Fetches a fresh snapshot. Should reject on error. */
  fetcher: () => Promise<T>;
  /** Receives a snapshot only when it differs from the last one — this
   *  is what kills the poll-induced re-render flicker. */
  onData: (data: T) => void;
  intervalMs: number;
  /** Pause polling when false (e.g. panel not visible). Default true. */
  enabled?: boolean;
  /** When this value changes the loop restarts and refetches at once
   *  (and the change-detection cache is cleared). Pass the connection
   *  identity — e.g. the daemon base URL — so switching servers swaps
   *  the data immediately instead of after the next interval. */
  resetKey?: string | number;
}

// useDataPoll centralizes the setInterval + cancel-flag pattern that
// every panel hand-rolled, and adds the loading/error/stale bookkeeping
// the old code lacked. Critically, it compares each snapshot against the
// last and only invokes `onData` when the payload actually changed, so a
// steady stream of identical polls no longer thrashes the store and
// re-renders the table.
export function useDataPoll<T>({
  fetcher,
  onData,
  intervalMs,
  enabled = true,
  resetKey,
}: Options<T>): PollState {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [stale, setStale] = useState(false);
  const [lastUpdated, setLastUpdated] = useState<number | null>(null);

  // Keep the latest callbacks in refs so changing their identity (a new
  // inline fetcher each render) doesn't tear down and restart the poll
  // loop — only `intervalMs`/`enabled` should do that.
  const fetcherRef = useRef(fetcher);
  const onDataRef = useRef(onData);
  fetcherRef.current = fetcher;
  onDataRef.current = onData;

  // Serialized form of the last delivered snapshot, for change detection.
  const lastSerialized = useRef<string | null>(null);
  const tick = useRef(0);

  // Guards state updates that a still-in-flight fetch would otherwise apply
  // after the component unmounted — the setState-after-unmount that made a
  // late poll throw "window is not defined" during test teardown (an
  // unhandled rejection that failed the web CI job even though every test
  // passed). Set false on unmount; every setter below is gated on it.
  const mountedRef = useRef(true);
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const run = useCallback(async () => {
    try {
      const data = await fetcherRef.current();
      if (!mountedRef.current) return;
      const serialized = JSON.stringify(data);
      if (serialized !== lastSerialized.current) {
        lastSerialized.current = serialized;
        onDataRef.current(data);
      }
      setError(null);
      setStale(false);
      setLastUpdated(Date.now());
    } catch (e: unknown) {
      if (!mountedRef.current) return;
      const msg = e instanceof Error ? e.message : "request failed";
      setError(msg);
      // Only "stale" if we already showed good data; otherwise it's a
      // plain initial-load error.
      setStale(lastSerialized.current !== null);
    } finally {
      if (mountedRef.current) setLoading(false);
    }
  }, []);

  const refresh = useCallback(() => {
    void run();
  }, [run]);

  useEffect(() => {
    if (!enabled) return;
    // A new connection target invalidates the change-detection cache so
    // the next server's data is always delivered, even if it happens to
    // serialize identically to the previous server's.
    lastSerialized.current = null;
    setLoading(true);
    let cancelled = false;
    const myTick = ++tick.current;
    const guarded = async () => {
      if (cancelled || myTick !== tick.current) return;
      await run();
    };
    void guarded();
    const t = window.setInterval(guarded, intervalMs);
    return () => {
      cancelled = true;
      window.clearInterval(t);
    };
  }, [enabled, intervalMs, run, resetKey]);

  return { loading, error, stale, lastUpdated, refresh };
}
