import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { formatClock, formatLocalDateTime } from "./formatTime";

// Pin a non-UTC timezone so the "local, not GMT" behaviour is observable and
// deterministic regardless of where the suite runs.
beforeAll(() => {
  vi.stubEnv("TZ", "America/New_York"); // UTC-5 (EST) in January
});
afterAll(() => {
  vi.unstubAllEnvs();
});

describe("formatClock", () => {
  it("renders the timestamp in local time, not UTC", () => {
    // 2026-01-15T12:00:00Z is 07:00:00 in America/New_York (UTC-5).
    // The old toISOString().slice(11,19) would have returned "12:00:00".
    expect(formatClock("2026-01-15T12:00:00Z")).toBe("07:00:00");
  });

  it("keeps a 24-hour HH:MM:SS shape", () => {
    expect(formatClock("2026-01-15T23:30:45Z")).toBe("18:30:45");
  });

  it("falls back to the raw time slice on an unparseable value", () => {
    // Non-date input: Date() is NaN, so it returns raw.slice(11,19) — here the
    // 8 chars starting at index 11 ("XXXXXXXXXXX" is 11 chars) = "12:34:56".
    expect(formatClock("XXXXXXXXXXX12:34:56")).toBe("12:34:56");
  });
});

describe("formatLocalDateTime", () => {
  it("renders date + time in local time, not UTC", () => {
    // 2026-09-10T00:15:18Z is 20:15:18 the previous evening in New York.
    // The old `.replace("T", " ")` slice would have shown "2026-09-10 00:15:18"
    // — the operator's "history is 3 hours behind the activity feed" report.
    expect(formatLocalDateTime("2026-09-10T00:15:18Z")).toBe(
      "2026-09-09 20:15:18",
    );
  });

  it("drops fractional seconds and honours an explicit offset", () => {
    expect(formatLocalDateTime("2026-09-10T03:15:18.654+03:00")).toBe(
      "2026-09-09 20:15:18",
    );
  });

  it("falls back to the raw value on an unparseable timestamp", () => {
    // (V8's Date parser is permissive — even "xTy.1" yields a year — so the
    // fixture is plain words, which every engine rejects.)
    expect(formatLocalDateTime("not a date")).toBe("not a date");
  });
});
