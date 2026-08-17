import { describe, it, expect, afterEach, vi } from "vitest";
import { api, eventsWebSocketURL, type ClientConfig } from "./client";

// Regression coverage for issue #290: the WebSocket URL must always be
// a well-formed ws(s):// URL with a host — never "ws:/api/v1/events/ws".
describe("eventsWebSocketURL", () => {
  it("derives ws:// from an http base URL", () => {
    expect(
      eventsWebSocketURL({ baseURL: "http://host:8080", token: null }),
    ).toBe("ws://host:8080/api/v1/events/ws");
  });

  it("derives wss:// from an https base URL", () => {
    expect(eventsWebSocketURL({ baseURL: "https://host", token: null })).toBe(
      "wss://host/api/v1/events/ws",
    );
  });

  it("normalizes an uppercase scheme", () => {
    expect(eventsWebSocketURL({ baseURL: "HTTPS://host", token: null })).toBe(
      "wss://host/api/v1/events/ws",
    );
  });

  it("tolerates a trailing slash on the base URL", () => {
    expect(
      eventsWebSocketURL({ baseURL: "http://host:8080/", token: null }),
    ).toBe("ws://host:8080/api/v1/events/ws");
  });

  it("falls back to the document origin when baseURL is empty", () => {
    // Same-origin deployment: resolve against the current document.
    const expected = `${window.location.origin.replace(/^http/, "ws")}/api/v1/events/ws`;
    expect(eventsWebSocketURL({ baseURL: "", token: null })).toBe(expected);
  });
});

// The daemon answers GET /api/v1/calls/history with {"calls":[…]} — see
// internal/api/handlers.go. Reading any other key silently yields an empty
// table via the `?? []` fallback, which is what made the History page look
// like "search by talkgroup is broken" when the filter was fine all along.
describe("api.history", () => {
  const cfg: ClientConfig = { baseURL: "http://host:8080", token: null };

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  function stubFetch(body: unknown): ReturnType<typeof vi.fn> {
    const f = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(body), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", f);
    return f;
  }

  it("unwraps the daemon's `calls` envelope", async () => {
    stubFetch({ calls: [{ id: 7, system: "250_013", group_id: 1020545 }] });
    const rows = await api.history(cfg, {});
    expect(rows).toHaveLength(1);
    expect(rows[0].group_id).toBe(1020545);
  });

  it("passes limit / system / group_id through as query params", async () => {
    const f = stubFetch({ calls: [] });
    await api.history(cfg, { limit: 200, system: "250_013", group_id: 1020545 });
    const url = new URL(f.mock.calls[0][0] as string);
    expect(url.pathname).toBe("/api/v1/calls/history");
    expect(url.searchParams.get("limit")).toBe("200");
    expect(url.searchParams.get("system")).toBe("250_013");
    expect(url.searchParams.get("group_id")).toBe("1020545");
  });

  it("returns an empty list when the daemon sends no calls key", async () => {
    stubFetch({});
    await expect(api.history(cfg, {})).resolves.toEqual([]);
  });
});
