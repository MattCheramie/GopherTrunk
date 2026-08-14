import { describe, expect, it } from "vitest";
import {
  callDuration,
  dedupeByTalkgroup,
  liveCallCount,
} from "./useLingeringActiveCalls";
import type { LingeringCall } from "./useLingeringActiveCalls";

// The hook's exported helpers are pure, so they are tested directly rather than
// through a renderer. dedupeByTalkgroup is exercised via liveCallCount and the
// roster shape the panels consume.

function row(
  over: Partial<LingeringCall> & { group_id: number; started_at: string },
): LingeringCall {
  const { group_id, started_at, ...rest } = over;
  return {
    grant: {
      system: "DMO",
      group_id,
      source_id: 0,
      frequency_hz: 438_900_000,
      encrypted: false,
      emergency: false,
    },
    started_at,
    device_serial: "d",
    following: true,
    _ended: false,
    _endMs: null,
    ...rest,
  } as unknown as LingeringCall;
}

describe("liveCallCount", () => {
  // The reported symptom: the "Active calls" tile read 0 while a call row was
  // visible right below it. The tile counted the raw poll response; the roster
  // rendered the linger-augmented list, which still held the just-ended call.
  it("counts the live rows the roster renders, not the ended ones", () => {
    const rows = [
      row({ group_id: 1, started_at: "2026-08-13T23:57:26Z" }),
      row({ group_id: 2, started_at: "2026-08-13T23:57:20Z", _ended: true, _endMs: 1 }),
    ];
    expect(liveCallCount(rows)).toBe(1);
  });

  it("is zero when every row is a lingering ended call", () => {
    const rows = [
      row({ group_id: 1, started_at: "2026-08-13T23:57:26Z", _ended: true, _endMs: 1 }),
    ];
    expect(liveCallCount(rows)).toBe(0);
  });
});

describe("callDuration", () => {
  it("ticks while live and freezes once ended", () => {
    const start = "2026-08-13T23:57:00Z";
    const now = Date.parse("2026-08-13T23:57:34Z");
    expect(callDuration(start, null, now)).toBe("00:34");
    expect(callDuration(start, Date.parse("2026-08-13T23:57:09Z"), now)).toBe("00:09");
  });

  it("renders an em dash for an unparseable start", () => {
    expect(callDuration("not-a-date", null, Date.now())).toBe("—");
  });
});

describe("dedupeByTalkgroup", () => {
  // The reported symptom: "funny spam of same TG in Active calls" — one
  // talkgroup occupying three rows at once (an ended one, an untuned one, and a
  // second talker), because the row key splits on source_id, timeslot and
  // frequency while the daemon reports followed and observed-only calls from two
  // separate maps.
  it("collapses one talkgroup on one system to a single row", () => {
    const rows = [
      row({ group_id: 1020539, started_at: "2026-08-13T23:58:20Z", _ended: true, _endMs: 1 }),
      row({ group_id: 1020539, started_at: "2026-08-13T23:58:10Z", following: false }),
      row({ group_id: 1020539, started_at: "2026-08-13T23:58:25Z" }),
    ];
    const out = dedupeByTalkgroup(rows);
    expect(out).toHaveLength(1);
    // A followed live call is the most informative state — audio is recording.
    expect(out[0].following).toBe(true);
    expect(out[0]._ended).toBe(false);
  });

  it("prefers a live row over a lingering ended one", () => {
    const out = dedupeByTalkgroup([
      row({ group_id: 7, started_at: "2026-08-13T23:58:20Z", _ended: true, _endMs: 1 }),
      row({ group_id: 7, started_at: "2026-08-13T23:58:25Z", following: false }),
    ]);
    expect(out).toHaveLength(1);
    expect(out[0]._ended).toBe(false);
  });

  it("keeps the earliest start so the duration spans the whole conversation", () => {
    const out = dedupeByTalkgroup([
      row({ group_id: 7, started_at: "2026-08-13T23:58:25Z" }),
      row({ group_id: 7, started_at: "2026-08-13T23:58:05Z" }),
    ]);
    expect(out[0].started_at).toBe("2026-08-13T23:58:05Z");
  });

  it("keeps genuinely different talkgroups and systems apart", () => {
    const a = row({ group_id: 1, started_at: "2026-08-13T23:58:00Z" });
    const b = row({ group_id: 2, started_at: "2026-08-13T23:58:00Z" });
    const c = row({ group_id: 1, started_at: "2026-08-13T23:58:00Z" });
    (c.grant as { system: string }).system = "OTHER";
    expect(dedupeByTalkgroup([a, b, c])).toHaveLength(3);
  });
});
