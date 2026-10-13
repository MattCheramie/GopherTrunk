import { describe, it, expect, vi, beforeEach } from "vitest";
import { act, render, screen, waitFor, fireEvent } from "@testing-library/react";

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

import {
  fetchSpectrumDevices,
  openSpectrumStream,
  tuneSpectrumDevice,
} from "../api/spectrum";
import { useShared } from "../store/shared";
import { Spectrum, parseTuneMHz } from "./Spectrum";

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

    const canvas = await screen.findByLabelText(/spectrum waterfall — hover/i);
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

  it("renders the spectrum analyzer line plot and reads out on hover", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      {
        serial: "rtl-1",
        driver: "rtlsdr",
        role: "control",
        center_hz: 153_000_000,
        sample_rate_hz: 2_048_000,
      },
    ]);
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

    // The analyzer canvas sits above the waterfall and shares the same
    // X→frequency mapping, so hovering it produces the same readout.
    const analyzer = await screen.findByLabelText(/spectrum analyzer/i);
    analyzer.getBoundingClientRect = () =>
      ({ left: 0, top: 0, width: 100, height: 160 }) as DOMRect;
    fireEvent.mouseMove(analyzer, { clientX: 50, clientY: 10 });

    await waitFor(() => {
      expect(screen.getByText(/153\.0000 MHz · -40\.0 dBFS/)).toBeInTheDocument();
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

  it("tunes the selected SDR to a typed frequency", async () => {
    // An SDR no decoder has tuned sits at 0 Hz, where click-to-tune cannot
    // reach a real band — the typed entry is the way out.
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      { serial: "00000001", driver: "rtlsdr", role: "control", center_hz: 0, sample_rate_hz: 2_400_000 },
    ]);
    vi.mocked(openSpectrumStream).mockReturnValue({ close: vi.fn() });
    render(<Spectrum />);
    await waitFor(() => expect(openSpectrumStream).toHaveBeenCalled());

    const input = screen.getByLabelText(/tune the sdr to a centre frequency/i);
    fireEvent.change(input, { target: { value: "162,550" } });
    fireEvent.click(screen.getByRole("button", { name: "Tune" }));
    await waitFor(() => {
      expect(tuneSpectrumDevice).toHaveBeenCalledWith(expect.anything(), "00000001", 162_550_000);
    });

    fireEvent.change(input, { target: { value: "16x2" } });
    fireEvent.click(screen.getByRole("button", { name: "Tune" }));
    await waitFor(() => {
      expect(screen.getByText(/is not a frequency/)).toBeInTheDocument();
    });
    expect(tuneSpectrumDevice).toHaveBeenCalledTimes(1);
  });

  it("explains an open stream that delivers no frames on a voice SDR", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      vi.mocked(fetchSpectrumDevices).mockResolvedValue([
        { serial: "00000002", driver: "rtlsdr", role: "voice", center_hz: 0, sample_rate_hz: 2_400_000 },
      ]);
      vi.mocked(openSpectrumStream).mockImplementation((_cfg, opts) => {
        opts.onStatus?.("open");
        return { close: vi.fn() };
      });
      render(<Spectrum />);
      await waitFor(() => expect(screen.getByText("live")).toBeInTheDocument());
      expect(screen.queryByRole("status")).toBeNull();
      // "live" renders as soon as the stream reports open, but the effect
      // that arms the stall timer for that render can still be pending — on
      // a loaded runner the advance below used to run first, and the timer,
      // armed after it, never fired within waitFor's budget. Flush pending
      // effects, then advance inside act.
      await act(async () => {});
      act(() => {
        vi.advanceTimersByTime(5_000);
      });
      await waitFor(() => {
        expect(screen.getByRole("status")).toHaveTextContent(/only streams while it is following a call/);
      });
    } finally {
      vi.useRealTimers();
    }
  });
});

describe("parseTuneMHz", () => {
  it("parses MHz with a decimal point or comma", () => {
    expect(parseTuneMHz("162.55")).toBe(162_550_000);
    expect(parseTuneMHz(" 442,3875 ")).toBe(442_387_500);
    expect(parseTuneMHz("851")).toBe(851_000_000);
  });
  it("rejects anything that is not a positive frequency", () => {
    for (const bad of ["", "abc", "-5", "0", "1e3", "16x2", "5000"]) {
      expect(parseTuneMHz(bad)).toBeNull();
    }
  });
});
