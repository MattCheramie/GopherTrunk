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

// The daemon's audio cockpit state (GET /api/v1/audio) and the PATCH that
// changes it. `audioStatus` is what the next poll resolves with; `patched`
// records every PATCH body the player sent.
let audioStatus: Promise<AudioStatusDTO>;
const patched: AudioPatch[] = [];
vi.mock("../api/client", async () => {
  const actual = await vi.importActual<typeof import("../api/client")>("../api/client");
  return {
    ...actual,
    api: { activeCalls: vi.fn().mockResolvedValue([]), audio: vi.fn(() => audioStatus) },
  };
});
vi.mock("../api/write", () => ({
  writes: {
    setAudio: vi.fn(async (_c: unknown, patch: AudioPatch) => {
      patched.push(patch);
      return { ...(await audioStatus), ...patch };
    }),
  },
}));

import { AudioPlayer } from "./AudioPlayer";
import { useShared } from "../store/shared";
import type { ActiveCallDTO, AudioStatusDTO } from "../api/types";
import type { AudioPatch } from "../api/write";

function status(over: Partial<AudioStatusDTO> = {}): AudioStatusDTO {
  return {
    backend_enabled: false,
    sample_rate: 8000,
    volume: 0.8,
    muted: false,
    recording_enabled: true,
    drops_total: 0,
    stream_samples_total: 0,
    ...over,
  };
}

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
    patched.length = 0;
    audioStatus = Promise.resolve(status());
    useShared.setState({
      serverURL: "http://localhost:8080",
      token: null,
      activeCalls: [],
      audio: null,
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

// Issue #1351: nothing loaded the daemon's cockpit state into the store, so
// the REC button rendered "off" for ever and its first click sent
// recording_enabled: false — it switched recordings OFF while looking like
// it turned them on. Every test here fails against that code.
describe("AudioPlayer daemon state (#1351)", () => {
  beforeEach(() => {
    built.length = 0;
    opened.length = 0;
    patched.length = 0;
    audioStatus = Promise.resolve(status());
    useShared.setState({
      serverURL: "http://localhost:8080",
      token: null,
      activeCalls: [],
      audio: null,
    });
  });

  it("shows the daemon's recording and mute state before any click", async () => {
    audioStatus = Promise.resolve(status({ recording_enabled: true, muted: true }));
    render(<AudioPlayer />);
    fireEvent.click(screen.getByLabelText("Enable audio"));

    const rec = await screen.findByLabelText("Recording on (click to disable)");
    expect(rec).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByLabelText("Unmute")).toBeInTheDocument();
    expect(patched).toHaveLength(0);
  });

  it("never toggles recording while the daemon's state is unknown", async () => {
    // The GET never answers (daemon slow, or the cockpit not wired).
    audioStatus = new Promise(() => {});
    render(<AudioPlayer />);
    fireEvent.click(screen.getByLabelText("Enable audio"));

    const rec = screen.getByLabelText("Recording state unknown");
    expect(rec).toBeDisabled();
    fireEvent.click(rec);
    await new Promise((r) => setTimeout(r, 20));
    expect(patched).toHaveLength(0);
  });

  it("the first REC click sends the opposite of the daemon's state and keeps the reply", async () => {
    audioStatus = Promise.resolve(status({ recording_enabled: false }));
    render(<AudioPlayer />);
    fireEvent.click(screen.getByLabelText("Enable audio"));

    const rec = await screen.findByLabelText("Recording off (click to enable)");
    fireEvent.click(rec);
    await waitFor(() => expect(patched).toEqual([{ recording_enabled: true }]));
    await waitFor(() =>
      expect(screen.getByLabelText("Recording on (click to disable)")).toHaveAttribute(
        "aria-pressed",
        "true",
      ),
    );
    expect(useShared.getState().audio?.recording_enabled).toBe(true);
  });

  it("says no voice has been decoded when the daemon has received no audio", async () => {
    audioStatus = Promise.resolve(status({ stream_samples_total: 0 }));
    render(<AudioPlayer />);
    fireEvent.click(screen.getByLabelText("Enable audio"));

    const s = await screen.findByRole("status");
    await waitFor(() => expect(s).toHaveTextContent("no decoded audio yet"));
    expect(s.getAttribute("title")).toMatch(/control-channel lock/);
  });

  it("waits for a call once audio has flowed, and names the call it plays", async () => {
    audioStatus = Promise.resolve(status({ stream_samples_total: 16_000 }));
    render(<AudioPlayer />);
    fireEvent.click(screen.getByLabelText("Enable audio"));

    const s = await screen.findByRole("status");
    await waitFor(() => expect(s).toHaveTextContent("waiting for a call"));

    act(() => {
      useShared.setState({
        activeCalls: [
          {
            ...call("VOICE-1", "2026-01-01T00:00:00Z"),
            talkgroup: { alpha_tag: "FIRE DISP" },
          } as ActiveCallDTO,
        ],
      });
    });
    await waitFor(() => expect(s).toHaveTextContent("TG FIRE DISP"));
  });

  it("applies an audio.state event from another client", async () => {
    render(<AudioPlayer />);
    fireEvent.click(screen.getByLabelText("Enable audio"));
    await screen.findByLabelText("Recording on (click to disable)");

    act(() => {
      useShared.getState().appendEvents([
        {
          kind: "audio.state",
          timestamp: "2026-01-01T00:00:01Z",
          payload: status({ recording_enabled: false }),
        },
      ]);
    });
    await screen.findByLabelText("Recording off (click to enable)");
  });
});
