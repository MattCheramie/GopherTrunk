import { useEffect, useRef, useState } from "react";
import type { ActiveCallDTO } from "../api/types";

// How long an ended call keeps showing in the list (frozen) after it drops out
// of GET /api/v1/calls/active, so an operator sees it come to rest with a final
// duration and an "ended" marker instead of it just vanishing.
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

// talkgroupKey identifies the CALL, as an operator thinks of it: one talkgroup
// up on one system. Deliberately coarser than callKey — it drops source,
// timeslot and frequency.
function talkgroupKey(c: ActiveCallDTO): string {
  return `${c.grant.system}|${c.grant.group_id}`;
}

// dedupeByTalkgroup collapses rows that describe the same talkgroup on the same
// system down to one, so a single conversation is a single line in the roster.
//
// One talkgroup could previously produce three rows at once. The daemon reports
// followed calls and control-channel-observed calls from two different maps
// (engine.ActiveCalls / engine.ObservedCalls), keyed by system|talkgroup|timeslot,
// while callKey above additionally splits on source_id and frequency_hz — so a
// re-grant on another frequency, a second talker, or a followed call whose
// observed-only sibling carries a different timeslot each opened a new row, and
// the linger window kept the old ones on screen alongside.
//
// Preference order within a group: a live row beats a lingering ended one (the
// call is still up), and among live rows a followed one beats an untuned one
// (audio is being recorded, which is the more informative state). Ties keep the
// earliest start so the duration reflects the whole conversation.
export function dedupeByTalkgroup(rows: LingeringCall[]): LingeringCall[] {
  const best = new Map<string, LingeringCall>();
  const rank = (c: LingeringCall) => (c._ended ? 0 : c.following === false ? 1 : 2);
  for (const c of rows) {
    const k = talkgroupKey(c);
    const cur = best.get(k);
    if (
      !cur ||
      rank(c) > rank(cur) ||
      (rank(c) === rank(cur) && c.started_at < cur.started_at)
    ) {
      best.set(k, c);
    }
  }
  return [...best.values()];
}

// useLingeringActiveCalls augments the live active-call list with calls that
// have just ended: when a call drops out of the poll it is kept for LINGER_MS,
// marked _ended with a frozen _endMs, then removed. This is what lets the call
// log show a call's final "call duration" (stopped, not still ticking) and flag
// it "ended" while it is still on screen. Purely client-side — no backend
// change; the daemon still just omits ended calls from the active endpoint.
//
// The result is de-duplicated by talkgroup (see dedupeByTalkgroup), so one
// conversation is one row rather than one row per (source, timeslot, frequency).
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
  return dedupeByTalkgroup(out);
}

// liveCallCount counts the rows that represent a call still up, for the
// "Active calls" stat tile. It counts the SAME rows the roster renders (minus
// the lingering ended ones), so the tile can no longer read 0 above a visible
// call — it used to count the raw poll response while the roster rendered the
// de-duplicated, linger-augmented list.
export function liveCallCount(rows: LingeringCall[]): number {
  return rows.reduce((n, c) => (c._ended ? n : n + 1), 0);
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
