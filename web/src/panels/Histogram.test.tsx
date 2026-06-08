import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";

vi.mock("../api/spectrum", () => ({
  fetchSpectrumDevices: vi.fn(),
}));

vi.mock("../api/symbols", () => ({
  openSymbolStream: vi.fn(),
}));

vi.mock("react-chartjs-2", () => ({ Bar: () => null }));
vi.mock("chart.js", () => {
  const noop = class {};
  return {
    Chart: { register: () => {} },
    BarElement: noop,
    CategoryScale: noop,
    Legend: noop,
    LinearScale: noop,
    Title: noop,
    Tooltip: noop,
  };
});

import { fetchSpectrumDevices } from "../api/spectrum";
import { openSymbolStream } from "../api/symbols";
import { useShared } from "../store/shared";
import { Histogram } from "./Histogram";

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

describe("Histogram panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
    window.localStorage.clear();
    vi.mocked(openSymbolStream).mockReturnValue({ close: vi.fn() });
  });

  it("renders an empty-state when no SDRs are available", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([]);
    render(<Histogram />);
    await waitFor(() => {
      expect(screen.getByText("No SDRs available")).toBeInTheDocument();
    });
    expect(openSymbolStream).not.toHaveBeenCalled();
  });

  it("derives a balance metric and MER from a C4FM frame", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);
    vi.mocked(openSymbolStream).mockImplementation((_cfg, opts) => {
      opts.onStatus?.("open");
      // Four levels, evenly balanced, with tight clusters → finite SNR.
      opts.onFrame({
        ts_ns: 1,
        symbol_rate_hz: 4800,
        center_hz: 851_012_500,
        offset_hz: 0,
        soft: [-3.1, -0.9, 1.1, 2.9, -2.9, -1.1, 0.9, 3.1],
        sym_i: [],
        sym_q: [],
        eye_soft: [],
        eye_sps: 0,
        dibits: [0, 1, 2, 3, 0, 1, 2, 3],
        is_bits: false,
        base_idx: 0,
        carrier_offset_hz: 0,
        agc_level: 0,
        agc_target: 0,
        clock_mu: 0,
        clock_sps: 10,
        cma_error: 0,
      });
      return { close: vi.fn() };
    });

    render(<Histogram />);
    // Perfectly balanced → balance deviation 0.0%.
    await waitFor(() => {
      expect(screen.getByText("±0.0%")).toBeInTheDocument();
    });
    // A finite dB SNR readout is shown (clusters separated, zero variance
    // → very high but finite once rendered).
    expect(screen.getByText(/dB/)).toBeInTheDocument();
  });
});
