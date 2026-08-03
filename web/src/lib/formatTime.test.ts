import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { formatClock } from "./formatTime";

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
