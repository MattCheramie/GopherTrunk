import { describe, it, expect } from "vitest";
import { eventsWebSocketURL } from "./client";

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
