import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";

vi.mock("../api/spectrum", () => ({
  fetchSpectrumDevices: vi.fn(),
}));

vi.mock("../api/symbols", () => ({
  openSymbolStream: vi.fn(),
}));

import { fetchSpectrumDevices } from "../api/spectrum";
import { openSymbolStream } from "../api/symbols";
import { useShared } from "../store/shared";
import { EyeDiagram } from "./EyeDiagram";

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

const ONE_DEVICE = [
  {
    serial: "rtl-1",
    driver: "rtlsdr",
    role: "control",
    center_hz: 851_012_500,
    sample_rate_hz: 2_048_000,
  },
];

describe("EyeDiagram panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
    window.localStorage.clear();
    vi.mocked(openSymbolStream).mockReturnValue({ close: vi.fn() });
  });

  it("renders an empty-state when no SDRs are available", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([]);
    render(<EyeDiagram />);
    await waitFor(() => {
      expect(screen.getByText("No SDRs available")).toBeInTheDocument();
    });
    expect(openSymbolStream).not.toHaveBeenCalled();
  });

  it("opens a C4FM symbol stream against the first device", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);

    render(<EyeDiagram />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalledTimes(1);
    });
    const callArgs = vi.mocked(openSymbolStream).mock.calls[0]?.[1];
    expect(callArgs?.serial).toBe("rtl-1");
    // The eye is C4FM-only — the panel must request the C4FM receiver.
    expect(callArgs?.proto).toBe("p25-c4fm");
  });

  it("shows the live status pill when the WS reports open", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);
    vi.mocked(openSymbolStream).mockImplementation((_cfg, opts) => {
      opts.onStatus?.("open");
      return { close: vi.fn() };
    });

    render(<EyeDiagram />);
    await waitFor(() => {
      expect(screen.getByText("live")).toBeInTheDocument();
    });
  });

  it("reports the samples-per-symbol once eye frames arrive", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);
    vi.mocked(openSymbolStream).mockImplementation((_cfg, opts) => {
      opts.onStatus?.("open");
      opts.onFrame({
        ts_ns: 1,
        symbol_rate_hz: 4800,
        center_hz: 851_012_500,
        offset_hz: 0,
        soft: [],
        sym_i: [],
        sym_q: [],
        eye_soft: [0, 0.1, 0.2, 0.3, 0.2, 0.1, 0, -0.1, -0.2, -0.1],
        eye_sps: 5,
        dibits: [1],
        is_bits: false,
        base_idx: 0,
      });
      return { close: vi.fn() };
    });

    render(<EyeDiagram />);
    await waitFor(() => {
      expect(screen.getByText(/samples\/symbol/)).toBeInTheDocument();
    });
  });
});
