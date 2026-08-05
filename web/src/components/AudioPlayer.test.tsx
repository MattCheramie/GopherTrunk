import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";

// Capture how the player is pointed at streams. The component must build the
// player (and its single AudioContext) ONCE and then re-point it with
// reconnect() as it follows calls — building a fresh player per switch is the
// leak that silenced Chrome. `built` counts createStreamPlayer calls; `opened`
// records every URL the stream is pointed at (initial + each reconnect).
const built: number[] = [];
const opened: string[] = [];
vi.mock("../audio/streamPlayer", () => ({
  createStreamPlayer: (opts: { url: string }) => {
    built.push(1);
    opened.push(opts.url);
    return {
      start: vi.fn(),
      stop: vi.fn(),
      reconnect: (url: string) => opened.push(url),
      setVolume: vi.fn(),
      setMuted: vi.fn(),
    };
  },
}));

import { AudioPlayer } from "./AudioPlayer";
import { useShared } from "../store/shared";
import type { ActiveCallDTO } from "../api/types";

function call(serial: string, startedAt: string): ActiveCallDTO {
  return {
    grant: { group_id: 1, system: "Sys", protocol: "p25", frequency_hz: 1 },
    device_serial: serial,
    started_at: startedAt,
  } as ActiveCallDTO;
}

describe("AudioPlayer live-audio follow", () => {
  beforeEach(() => {
    built.length = 0;
    opened.length = 0;
    useShared.setState({
      serverURL: "http://localhost:8080",
      token: null,
      activeCalls: [],
    });
  });

  it("holds the current call to completion, then follows the next", async () => {
    useShared.setState({ activeCalls: [call("VOICE-1", "2026-01-01T00:00:00Z")] });

    render(<AudioPlayer />);

    // Tap to enable → stream opens filtered to the active call's device.
    fireEvent.click(screen.getByLabelText("Enable audio"));
    await waitFor(() => expect(opened).toHaveLength(1));
    expect(opened[0]).toContain("device=VOICE-1");

    // A newer call appears while VOICE-1 is STILL active → we must NOT cut
    // VOICE-1 off. This is the "eating conversations" regression: the stream
    // stays on VOICE-1.
    act(() => {
      useShared.setState({
        activeCalls: [
          call("VOICE-1", "2026-01-01T00:00:00Z"),
          call("VOICE-2", "2026-01-01T00:00:05Z"),
        ],
      });
    });
    // Give the follow effect a chance to (wrongly) fire.
    await new Promise((r) => setTimeout(r, 20));
    expect(opened).toHaveLength(1); // still only VOICE-1

    // VOICE-1 ends (drops out of the active set) → advance to VOICE-2.
    act(() => {
      useShared.setState({
        activeCalls: [call("VOICE-2", "2026-01-01T00:00:05Z")],
      });
    });
    await waitFor(() => expect(opened).toHaveLength(2));
    expect(opened[1]).toContain("device=VOICE-2");

    // The whole point of the leak fix: one player/context, re-pointed — not a
    // fresh one per switch.
    expect(built).toHaveLength(1);
  });

  it("opens unfiltered when no call is active", async () => {
    render(<AudioPlayer />);
    fireEvent.click(screen.getByLabelText("Enable audio"));
    await waitFor(() => expect(opened).toHaveLength(1));
    expect(opened[0]).not.toContain("device=");
  });
});
