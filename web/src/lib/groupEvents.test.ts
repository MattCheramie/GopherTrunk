import { describe, it, expect } from "vitest";
import type { EventDTO } from "../api/types";
import { groupEvents, contentKey } from "./groupEvents";

const grant = (tg: number, ts: string): EventDTO => ({
  kind: "grant",
  timestamp: ts,
  payload: { system: "S", group_id: tg, frequency_hz: 451_000_000 },
});

describe("groupEvents", () => {
  it("collapses identical events into one entry with a count and latest timestamp", () => {
    const events = [
      grant(100, "2026-06-19T05:00:00Z"),
      grant(100, "2026-06-19T05:00:02Z"),
      grant(100, "2026-06-19T05:00:05Z"),
    ];
    const groups = groupEvents(events);
    expect(groups).toHaveLength(1);
    expect(groups[0].count).toBe(3);
    expect(groups[0].firstTimestamp).toBe("2026-06-19T05:00:00Z");
    expect(groups[0].lastTimestamp).toBe("2026-06-19T05:00:05Z");
    // representative is the most recent occurrence
    expect(groups[0].event.timestamp).toBe("2026-06-19T05:00:05Z");
  });

  it("keeps events with different payloads separate", () => {
    const groups = groupEvents([
      grant(100, "2026-06-19T05:00:00Z"),
      grant(200, "2026-06-19T05:00:01Z"),
      grant(100, "2026-06-19T05:00:02Z"),
    ]);
    expect(groups).toHaveLength(2);
    const tg100 = groups.find((g) => (g.event.payload as { group_id: number }).group_id === 100)!;
    expect(tg100.count).toBe(2);
  });

  it("orders groups by last occurrence (most-recently-active last)", () => {
    // tg200 fires last, so it should sort after tg100 even though tg100 appeared first.
    const groups = groupEvents([
      grant(100, "2026-06-19T05:00:00Z"),
      grant(200, "2026-06-19T05:00:01Z"),
      grant(100, "2026-06-19T05:00:02Z"),
      grant(200, "2026-06-19T05:00:09Z"),
    ]);
    expect(groups.map((g) => (g.event.payload as { group_id: number }).group_id)).toEqual([100, 200]);
  });

  it("groups regardless of payload key order (stable stringify)", () => {
    const a: EventDTO = { kind: "registration", timestamp: "t1", payload: { src: 1, system: "S" } };
    const b: EventDTO = { kind: "registration", timestamp: "t2", payload: { system: "S", src: 1 } };
    expect(contentKey(a)).toBe(contentKey(b));
    expect(groupEvents([a, b])).toHaveLength(1);
  });
});
