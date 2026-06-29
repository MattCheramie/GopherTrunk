import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";

// Capture every stream the player opens so we can assert the URL it
// filtered to. start/stop/setVolume/setMuted are inert.
const opened: string[] = [];
vi.mock("../audio/streamPlayer", () => ({
  createStreamPlayer: (opts: { url: string }) => {
    opened.push(opts.url);
    return {
      start: vi.fn(),
      stop: vi.fn(),
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
    opened.length = 0;
    useShared.setState({
      serverURL: "http://localhost:8080",
      token: null,
      activeCalls: [],
    });
  });

  it("opens the stream filtered to the primary call and follows the newest", async () => {
    useShared.setState({ activeCalls: [call("VOICE-1", "2026-01-01T00:00:00Z")] });

    render(<AudioPlayer />);

    // Tap to enable → stream opens filtered to the active call's device.
    fireEvent.click(screen.getByLabelText("Enable audio"));
    await waitFor(() => expect(opened).toHaveLength(1));
    expect(opened[0]).toContain("device=VOICE-1");

    // A newer call on a different device → follow it (one call at a time).
    act(() => {
      useShared.setState({
        activeCalls: [
          call("VOICE-1", "2026-01-01T00:00:00Z"),
          call("VOICE-2", "2026-01-01T00:00:05Z"),
        ],
      });
    });
    await waitFor(() => expect(opened).toHaveLength(2));
    expect(opened[1]).toContain("device=VOICE-2");
  });

  it("opens unfiltered when no call is active", async () => {
    render(<AudioPlayer />);
    fireEvent.click(screen.getByLabelText("Enable audio"));
    await waitFor(() => expect(opened).toHaveLength(1));
    expect(opened[0]).not.toContain("device=");
  });
});
