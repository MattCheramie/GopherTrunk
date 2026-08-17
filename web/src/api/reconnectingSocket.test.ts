import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  GONE_AFTER_FAILURES,
  INITIAL_BACKOFF,
  OPEN_GRACE_MS,
  openReconnectingSocket,
} from "./reconnectingSocket";

// A minimal WebSocket stand-in whose lifecycle the test drives by hand.
class FakeSocket {
  static instances: FakeSocket[] = [];
  onopen: ((ev?: unknown) => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onerror: ((ev?: unknown) => void) | null = null;
  onclose: ((ev?: unknown) => void) | null = null;
  closed = false;

  constructor(public url: string) {
    FakeSocket.instances.push(this);
  }
  close() {
    this.closed = true;
  }

  /** The old server shape: handshake succeeds, then closes immediately. */
  openThenClose() {
    this.onopen?.();
    this.onerror?.();
    this.onclose?.();
  }
}

describe("openReconnectingSocket", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    FakeSocket.instances = [];
    vi.stubGlobal("WebSocket", FakeSocket);
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  // The storm: the daemon used to upgrade the WebSocket before resolving the
  // requested SDR, so an unknown serial produced a successful handshake and an
  // immediate 1011 close. The old client reset its backoff in `onopen`, so it
  // reconnected every 250-500 ms indefinitely. Backoff must grow across
  // attempts that open but never deliver a frame.
  it("grows backoff when the socket opens but never delivers a frame", () => {
    const waits: number[] = [];
    const spy = vi
      .spyOn(window, "setTimeout")
      .mockImplementation(((fn: () => void, ms?: number) => {
        waits.push(ms ?? 0);
        return 0 as unknown as number;
      }) as typeof window.setTimeout);

    openReconnectingSocket({ url: () => "ws://x", onMessage: () => {} });
    for (let i = 0; i < 3; i++) {
      FakeSocket.instances[i].openThenClose();
      // Run the scheduled reconnect by hand so a fresh socket is created.
      (spy.mock.calls[i][0] as () => void)();
    }

    expect(waits.length).toBeGreaterThanOrEqual(3);
    // Jitter is base/2..base, so successive ceilings double.
    expect(waits[0]).toBeLessThanOrEqual(INITIAL_BACKOFF);
    expect(waits[1]).toBeGreaterThan(waits[0]);
    expect(waits[2]).toBeGreaterThan(waits[1]);
    spy.mockRestore();
  });

  // onDown was bound to both onerror and onclose with no guard, so a single
  // failure scheduled two reconnect timers while only the last handle was
  // remembered — timers leaked and the reconnect rate compounded.
  it("schedules exactly one reconnect when a failure fires both onerror and onclose", () => {
    const spy = vi
      .spyOn(window, "setTimeout")
      .mockImplementation((() => 0 as unknown as number) as typeof window.setTimeout);

    openReconnectingSocket({ url: () => "ws://x", onMessage: () => {} });
    const s = FakeSocket.instances[0];
    s.onerror?.();
    s.onclose?.();

    expect(spy).toHaveBeenCalledTimes(1);
    spy.mockRestore();
  });

  it("resets backoff once a frame actually arrives", () => {
    const waits: number[] = [];
    const spy = vi
      .spyOn(window, "setTimeout")
      .mockImplementation(((fn: () => void, ms?: number) => {
        waits.push(ms ?? 0);
        return 0 as unknown as number;
      }) as typeof window.setTimeout);

    const seen: string[] = [];
    openReconnectingSocket({
      url: () => "ws://x",
      onMessage: (d) => seen.push(d),
    });

    // Two failed attempts push the backoff up.
    FakeSocket.instances[0].openThenClose();
    (spy.mock.calls[0][0] as () => void)();
    FakeSocket.instances[1].openThenClose();
    (spy.mock.calls[1][0] as () => void)();
    const grown = waits[1];

    // Third attempt delivers data, then drops.
    const s = FakeSocket.instances[2];
    s.onopen?.();
    s.onmessage?.({ data: '{"ok":true}' });
    s.onclose?.();

    expect(seen).toEqual(['{"ok":true}']);
    expect(waits[2]).toBeLessThanOrEqual(INITIAL_BACKOFF);
    expect(waits[2]).toBeLessThan(grown);
    spy.mockRestore();
  });

  it("treats a long-lived silent stream as a success", () => {
    const waits: number[] = [];
    const spy = vi
      .spyOn(window, "setTimeout")
      .mockImplementation(((fn: () => void, ms?: number) => {
        waits.push(ms ?? 0);
        return 0 as unknown as number;
      }) as typeof window.setTimeout);
    const now = vi.spyOn(Date, "now");

    now.mockReturnValue(0);
    openReconnectingSocket({ url: () => "ws://x", onMessage: () => {} });
    const s = FakeSocket.instances[0];
    s.onopen?.();
    // A channel that is simply quiet must not be punished.
    now.mockReturnValue(OPEN_GRACE_MS + 1);
    s.onclose?.();

    expect(waits[0]).toBeLessThanOrEqual(INITIAL_BACKOFF);
    now.mockRestore();
    spy.mockRestore();
  });

  // The give-up path: after enough failures the caller is told to stop relying
  // on this device so it can re-enumerate instead of retrying a serial that is
  // never coming back.
  it("declares the stream gone and stops retrying", () => {
    const spy = vi
      .spyOn(window, "setTimeout")
      .mockImplementation((() => 0 as unknown as number) as typeof window.setTimeout);
    const statuses: string[] = [];
    const onGone = vi.fn();

    openReconnectingSocket({
      url: () => "ws://x",
      onMessage: () => {},
      onStatus: (s) => statuses.push(s),
      onGone,
    });

    for (let i = 0; i < GONE_AFTER_FAILURES; i++) {
      FakeSocket.instances[i].openThenClose();
      const call = spy.mock.calls[i];
      if (call) (call[0] as () => void)();
    }

    expect(onGone).toHaveBeenCalledTimes(1);
    expect(statuses).toContain("gone");
    // No socket is created after giving up.
    const created = FakeSocket.instances.length;
    vi.advanceTimersByTime(60_000);
    expect(FakeSocket.instances.length).toBe(created);
    spy.mockRestore();
  });
});
