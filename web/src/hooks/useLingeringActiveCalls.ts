import { useEffect, useRef, useState } from "react";
import type { ActiveCallDTO } from "../api/types";

// How long an ended call keeps showing in the list (frozen) after it drops out
// of GET /api/v1/calls/active, so an operator sees it come to rest with a final
// duration and an "observed" (ended) marker instead of it just vanishing.
const LINGER_MS = 7_000;

export interface LingeringCall extends ActiveCallDTO {
  // _ended is true once the call has left the active poll set (it is now
  // lingering, frozen, on its way out). _endMs is the wall-clock the duration
  // freezes at (last_heard_at when known, else when it dropped); null while live
  // so the duration ticks.
  _ended: boolean;
  _endMs: number | null;
}

// callKey is a stable identity for an active call across polls. ActiveCallDTO
// carries no id, so derive one from the grant (system + talkgroup + source +
// timeslot + frequency uniquely identify a concurrent call, including two DMR
// timeslots on one carrier).
function callKey(c: ActiveCallDTO): string {
  const g = c.grant;
  return `${g.system}|${g.group_id}|${g.source_id ?? ""}|${g.timeslot ?? ""}|${g.frequency_hz}`;
}

// useLingeringActiveCalls augments the live active-call list with calls that
// have just ended: when a call drops out of the poll it is kept for LINGER_MS,
// marked _ended with a frozen _endMs, then removed. This is what lets the call
// log show a call's final "call duration" (stopped, not still ticking) and flag
// it "observed" while it is still on screen. Purely client-side — no backend
// change; the daemon still just omits ended calls from the active endpoint.
export function useLingeringActiveCalls(
  active: ActiveCallDTO[],
): LingeringCall[] {
  const cacheRef = useRef<
    Map<string, { call: ActiveCallDTO; endedAt: number | null }>
  >(new Map());
  const [, force] = useState(0);

  // Reconcile the cache whenever the poll set changes: refresh live calls,
  // stamp newly-absent ones as ended.
  useEffect(() => {
    const now = Date.now();
    const cache = cacheRef.current;
    const live = new Set<string>();
    for (const c of active) {
      const k = callKey(c);
      live.add(k);
      cache.set(k, { call: c, endedAt: null });
    }
    for (const [k, e] of cache) {
      if (!live.has(k) && e.endedAt == null) {
        cache.set(k, { call: e.call, endedAt: now });
      }
    }
    force((n) => n + 1);
  }, [active]);

  // Expire lingering ended calls on a 1 s tick.
  useEffect(() => {
    const t = window.setInterval(() => {
      const now = Date.now();
      const cache = cacheRef.current;
      let changed = false;
      for (const [k, e] of cache) {
        if (e.endedAt != null && now - e.endedAt > LINGER_MS) {
          cache.delete(k);
          changed = true;
        }
      }
      if (changed) force((n) => n + 1);
    }, 1_000);
    return () => window.clearInterval(t);
  }, []);

  const out: LingeringCall[] = [];
  for (const { call, endedAt } of cacheRef.current.values()) {
    const ended = endedAt != null;
    let endMs: number | null = null;
    if (ended) {
      const lh = call.last_heard_at ? Date.parse(call.last_heard_at) : NaN;
      endMs = Number.isNaN(lh) ? endedAt! : lh;
    }
    out.push({ ...call, _ended: ended, _endMs: endMs });
  }
  return out;
}

// callDuration formats a fixed mm:ss "call duration" that ticks while the call
// is live (endMs null → now) and freezes once it ends (endMs set).
export function callDuration(
  startedAt: string,
  endMs: number | null,
  now: number,
): string {
  const startMs = Date.parse(startedAt);
  if (Number.isNaN(startMs)) return "—";
  const ms = Math.max(0, (endMs ?? now) - startMs);
  const totalSeconds = Math.floor(ms / 1000);
  const m = Math.floor(totalSeconds / 60);
  const s = totalSeconds % 60;
  return `${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
}
