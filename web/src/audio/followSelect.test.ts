import { describe, it, expect } from "vitest";
import { pickFollowSerial } from "./followSelect";

const call = (device_serial: string, started_at: string) => ({
  device_serial,
  started_at,
});

describe("pickFollowSerial", () => {
  it("returns null when nothing is active", () => {
    expect(pickFollowSerial([], null)).toBeNull();
    expect(pickFollowSerial([], "dev-a")).toBeNull();
  });

  it("picks the newest call when nothing is held yet", () => {
    const calls = [call("dev-a", "2026-01-01T00:00:01Z"), call("dev-b", "2026-01-01T00:00:03Z")];
    expect(pickFollowSerial(calls, null)).toBe("dev-b");
  });

  it("HOLDS the current call while its device is still active", () => {
    // dev-a is held; dev-b is newer but must NOT steal the stream mid-call.
    const calls = [
      call("dev-a", "2026-01-01T00:00:01Z"),
      call("dev-b", "2026-01-01T00:00:05Z"),
    ];
    expect(pickFollowSerial(calls, "dev-a")).toBe("dev-a");
  });

  it("advances to the newest once the held call ends", () => {
    // dev-a dropped out of the active set → follow whatever is live now.
    const calls = [
      call("dev-b", "2026-01-01T00:00:05Z"),
      call("dev-c", "2026-01-01T00:00:04Z"),
    ];
    expect(pickFollowSerial(calls, "dev-a")).toBe("dev-b");
  });

  it("ignores calls without a device serial (announced, no tuner)", () => {
    const calls = [call("", "2026-01-01T00:00:09Z"), call("dev-a", "2026-01-01T00:00:01Z")];
    expect(pickFollowSerial(calls, null)).toBe("dev-a");
  });
});
