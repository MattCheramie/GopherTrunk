import { describe, it, expect } from "vitest";
import { mixerWebSocketURL } from "./mixer";
import { type ClientConfig } from "./client";

const cfg: ClientConfig = { baseURL: "https://radio.example:8443", token: null };

describe("mixerWebSocketURL", () => {
  it("targets the diag/mixer endpoint and upgrades https → wss", () => {
    const url = mixerWebSocketURL(cfg, { serial: "abc", onFrame: () => {} });
    const u = new URL(url);
    expect(u.protocol).toBe("wss:");
    expect(u.pathname).toBe("/api/v1/diag/mixer");
    expect(u.searchParams.get("device")).toBe("abc");
  });

  it("includes proto and offset when set", () => {
    const url = mixerWebSocketURL(cfg, {
      serial: "abc",
      proto: "p25-cqpsk",
      offset: 12500.4,
      onFrame: () => {},
    });
    const u = new URL(url);
    expect(u.searchParams.get("proto")).toBe("p25-cqpsk");
    // Offsets are rounded to an integer Hz on the wire.
    expect(u.searchParams.get("offset")).toBe("12500");
  });

  it("omits a zero offset", () => {
    const url = mixerWebSocketURL(cfg, {
      serial: "abc",
      offset: 0,
      onFrame: () => {},
    });
    expect(new URL(url).searchParams.has("offset")).toBe(false);
  });
});
