import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";

vi.mock("../api/spectrum", () => ({
  fetchSpectrumDevices: vi.fn(),
  openSpectrumStream: vi.fn(),
  tuneSpectrumDevice: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("../api/bookmarks", () => ({
  bookmarks: {
    list: vi.fn().mockResolvedValue([]),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
  },
}));

import { fetchSpectrumDevices, openSpectrumStream } from "../api/spectrum";
import { useShared } from "../store/shared";
import { Spectrum } from "./Spectrum";

function resetStore() {
  useShared.setState({
    serverURL: "http://localhost:8080",
    token: null,
    connected: true,
    wsStatus: "idle",
    mutations: null,
    lastError: null,
    events: [],
    activeCalls: [],
    devices: [],
    systems: [],
    talkgroups: [],
    health: null,
    audio: null,
    scanner: null,
  });
}

describe("Spectrum panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
  });

  it("renders an empty-state when no SDRs are available", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([]);
    render(<Spectrum />);
    await waitFor(() => {
      expect(screen.getByText("No SDRs available")).toBeInTheDocument();
    });
    // No stream should be opened when there's no device.
    expect(openSpectrumStream).not.toHaveBeenCalled();
  });

  it("opens a stream against the first device returned by the daemon", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      {
        serial: "rtl-1",
        driver: "rtlsdr",
        product: "NESDR",
        role: "control",
        center_hz: 851_012_500,
        sample_rate_hz: 2_048_000,
      },
    ]);
    vi.mocked(openSpectrumStream).mockReturnValue({ close: vi.fn() });

    render(<Spectrum />);

    await waitFor(() => {
      expect(openSpectrumStream).toHaveBeenCalledTimes(1);
    });
    const callArgs = vi.mocked(openSpectrumStream).mock.calls[0]?.[1];
    expect(callArgs?.serial).toBe("rtl-1");
  });

  it("surfaces an error message when device discovery fails", async () => {
    vi.mocked(fetchSpectrumDevices).mockRejectedValue(new Error("kaboom"));
    render(<Spectrum />);
    await waitFor(() => {
      expect(screen.getByText(/kaboom/)).toBeInTheDocument();
    });
  });

  it("shows live status pill when the WebSocket reports open", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      {
        serial: "rtl-1",
        driver: "rtlsdr",
        role: "control",
        center_hz: 100,
        sample_rate_hz: 2_048_000,
      },
    ]);
    vi.mocked(openSpectrumStream).mockImplementation((_cfg, opts) => {
      opts.onStatus?.("open");
      return { close: vi.fn() };
    });

    render(<Spectrum />);
    await waitFor(() => {
      expect(screen.getByText("live")).toBeInTheDocument();
    });
  });

  it("shows the frequency and signal level under the cursor on hover", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      {
        serial: "rtl-1",
        driver: "rtlsdr",
        role: "control",
        center_hz: 153_000_000,
        sample_rate_hz: 2_048_000,
      },
    ]);
    // Push one frame so the panel has a center/sample-rate to map against
    // and a waterfall row to read the bin power from.
    vi.mocked(openSpectrumStream).mockImplementation((_cfg, opts) => {
      opts.onFrame?.({
        ts_ns: 0,
        center_hz: 153_000_000,
        sample_rate_hz: 2_048_000,
        bins: [-80, -70, -60, -50, -40, -30, -20, -10],
      });
      return { close: vi.fn() };
    });

    render(<Spectrum />);

    const canvas = await screen.findByLabelText(/hover to read the frequency/i);
    // Map cursor → frequency: at the canvas midpoint (xRatio 0.5) the
    // frequency equals the center, and the matching bin (index 4) is -40.
    canvas.getBoundingClientRect = () =>
      ({ left: 0, top: 0, width: 100, height: 320 }) as DOMRect;
    fireEvent.mouseMove(canvas, { clientX: 50, clientY: 10 });

    await waitFor(() => {
      expect(screen.getByText(/153\.0000 MHz · -40\.0 dBFS/)).toBeInTheDocument();
    });

    // Leaving the canvas clears the readout.
    fireEvent.mouseLeave(canvas);
    await waitFor(() => {
      expect(screen.queryByText(/153\.0000 MHz · -40\.0 dBFS/)).toBeNull();
    });
  });

  it("fetches and shows bookmark marker count", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      {
        serial: "rtl-1",
        driver: "rtlsdr",
        role: "control",
        center_hz: 851_012_500,
        sample_rate_hz: 2_048_000,
      },
    ]);
    vi.mocked(openSpectrumStream).mockReturnValue({ close: vi.fn() });
    const { bookmarks } = await import("../api/bookmarks");
    vi.mocked(bookmarks.list).mockResolvedValue([
      {
        id: 1,
        name: "marker-1",
        freq_hz: 851_500_000,
        mode: "FM",
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
      {
        id: 2,
        name: "marker-2",
        freq_hz: 851_800_000,
        mode: "FM",
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
    ]);

    render(<Spectrum />);
    await waitFor(() => {
      expect(bookmarks.list).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(screen.getByText(/2 visible/)).toBeInTheDocument();
    });
  });
});
