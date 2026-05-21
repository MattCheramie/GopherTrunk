import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { Mock } from "vitest";
import { openEventStream, type EventsHandler, type StatusHandler } from "./events";
import type { ClientConfig } from "./client";

// jsdom ships no WebSocket, so install a controllable mock. It records
// every instance and lets a test drive the open/message/close events
// by hand, which is what makes the reconnect lifecycle deterministic.
class MockWebSocket {
  static instances: MockWebSocket[] = [];

  url: string;
  onopen: (() => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  closeCalls = 0;

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }
  close() {
    this.closeCalls += 1;
  }
  emitOpen() {
    this.onopen?.();
  }
  emitMessage(data: string) {
    this.onmessage?.({ data });
  }
  emitClose() {
    this.onclose?.();
  }
}

const cfg: ClientConfig = { baseURL: "http://host:8080", token: null };

describe("openEventStream", () => {
  let onEvents: Mock<EventsHandler>;
  let onStatus: Mock<StatusHandler>;

  beforeEach(() => {
    MockWebSocket.instances = [];
    globalThis.WebSocket = MockWebSocket as unknown as typeof WebSocket;
    onEvents = vi.fn<EventsHandler>();
    onStatus = vi.fn<StatusHandler>();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("grows the reconnect delay when the connection keeps flapping (issue #290)", () => {
    vi.useFakeTimers();
    // Pin jitter to its low end so each delay is exactly backoff / 2.
    vi.spyOn(Math, "random").mockReturnValue(0);

    openEventStream(cfg, { onEvents, onStatus });
    expect(MockWebSocket.instances).toHaveLength(1);

    // First flap: a socket that opens then immediately drops.
    MockWebSocket.instances[0].emitOpen();
    MockWebSocket.instances[0].emitClose();

    // Reconnect is scheduled at jittered(500) = 250 ms.
    vi.advanceTimersByTime(249);
    expect(MockWebSocket.instances).toHaveLength(1);
    vi.advanceTimersByTime(1);
    expect(MockWebSocket.instances).toHaveLength(2);

    // Second flap. The backoff has grown, so this reconnect must take
    // strictly longer than the first — the 500 ms storm is broken.
    MockWebSocket.instances[1].emitOpen();
    MockWebSocket.instances[1].emitClose();

    vi.advanceTimersByTime(250);
    expect(MockWebSocket.instances).toHaveLength(2); // 250 ms no longer enough
    vi.advanceTimersByTime(250);
    expect(MockWebSocket.instances).toHaveLength(3); // jittered(1000) = 500 ms
  });

  it("resets the backoff only after a connection stays stable", () => {
    vi.useFakeTimers();
    vi.spyOn(Math, "random").mockReturnValue(0);

    openEventStream(cfg, { onEvents, onStatus });

    // Flap once so the backoff grows past the floor.
    MockWebSocket.instances[0].emitOpen();
    MockWebSocket.instances[0].emitClose();
    vi.advanceTimersByTime(250);
    expect(MockWebSocket.instances).toHaveLength(2);

    // This connection holds open past the 5 s stability window.
    MockWebSocket.instances[1].emitOpen();
    vi.advanceTimersByTime(5_000);
    MockWebSocket.instances[1].emitClose();

    // Backoff is back at the floor: jittered(500) = 250 ms reconnect.
    vi.advanceTimersByTime(250);
    expect(MockWebSocket.instances).toHaveLength(3);
  });

  it("stops reconnecting once the stream is closed", () => {
    vi.useFakeTimers();
    const stream = openEventStream(cfg, { onEvents, onStatus });

    MockWebSocket.instances[0].emitClose(); // schedules a reconnect
    stream.close();

    vi.advanceTimersByTime(60_000);
    expect(MockWebSocket.instances).toHaveLength(1); // no reconnect fired
  });

  it("delivers no events or status after close()", () => {
    const stream = openEventStream(cfg, { onEvents, onStatus });
    const socket = MockWebSocket.instances[0];

    stream.close();
    onStatus.mockClear();
    onEvents.mockClear();

    // A late event from the in-flight socket must not reach the store.
    socket.emitOpen();
    socket.emitMessage(JSON.stringify({ kind: "call.start", timestamp: "t" }));
    socket.emitClose();

    expect(onStatus).not.toHaveBeenCalled();
    expect(onEvents).not.toHaveBeenCalled();
    expect(socket.closeCalls).toBe(1);
  });

  it("coalesces a burst of frames into a single batched delivery", () => {
    vi.useFakeTimers();
    openEventStream(cfg, { onEvents, onStatus });
    const socket = MockWebSocket.instances[0];
    socket.emitOpen();

    socket.emitMessage(JSON.stringify({ kind: "call.start", timestamp: "t1" }));
    socket.emitMessage(JSON.stringify({ kind: "cchunt.progress", timestamp: "t2" }));
    socket.emitMessage(JSON.stringify({ kind: "call.end", timestamp: "t3" }));

    // Nothing is delivered until the flush window elapses.
    expect(onEvents).not.toHaveBeenCalled();
    vi.advanceTimersByTime(100);

    // One store write for the whole burst, not three.
    expect(onEvents).toHaveBeenCalledTimes(1);
    expect(onEvents.mock.calls[0][0]).toHaveLength(3);
  });

  it("drops frames that aren't a well-formed event", () => {
    vi.useFakeTimers();
    openEventStream(cfg, { onEvents, onStatus });
    const socket = MockWebSocket.instances[0];
    socket.emitOpen();

    socket.emitMessage("not json at all");
    socket.emitMessage(JSON.stringify({ kind: 42, timestamp: "t" }));
    socket.emitMessage(JSON.stringify({ timestamp: "t" })); // no kind
    socket.emitMessage(JSON.stringify({ kind: "decode.error", timestamp: "t" }));

    vi.advanceTimersByTime(100);
    expect(onEvents).toHaveBeenCalledTimes(1);
    expect(onEvents.mock.calls[0][0]).toEqual([
      { kind: "decode.error", timestamp: "t" },
    ]);
  });

  it("reports the first connecting status asynchronously, not during setup", async () => {
    openEventStream(cfg, { onEvents, onStatus });
    // openEventStream must not write to the store synchronously inside
    // the React effect that calls it.
    expect(onStatus).not.toHaveBeenCalled();

    await Promise.resolve();
    expect(onStatus).toHaveBeenCalledWith("connecting");
  });
});
