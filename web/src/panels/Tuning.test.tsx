import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";

vi.mock("../api/spectrum", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/spectrum")>()),
  fetchSpectrumDevices: vi.fn(),
}));

vi.mock("../api/symbols", () => ({
  openSymbolStream: vi.fn(),
}));

// Stub the chart libs so the panel mounts under jsdom without a real
// <canvas> backend (mirrors App.panels.test.tsx).
vi.mock("react-chartjs-2", () => ({ Line: () => null }));
vi.mock("chart.js", () => {
  const noop = class {};
  return {
    Chart: { register: () => {} },
    CategoryScale: noop,
    Filler: noop,
    Legend: noop,
    LinearScale: noop,
    LineElement: noop,
    PointElement: noop,
    Title: noop,
    Tooltip: noop,
  };
});

import { fetchSpectrumDevices } from "../api/spectrum";
import { openSymbolStream } from "../api/symbols";
import { useShared } from "../store/shared";
import { Tuning } from "./Tuning";

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

describe("Tuning panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
    window.localStorage.clear();
    vi.mocked(openSymbolStream).mockReturnValue({ close: vi.fn() });
  });

  it("renders an empty-state when no SDRs are available", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([]);
    render(<Tuning />);
    await waitFor(() => {
      expect(screen.getByText("No SDRs available")).toBeInTheDocument();
    });
    expect(openSymbolStream).not.toHaveBeenCalled();
  });

  it("opens a symbol stream and re-subscribes on mode change", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);

    render(<Tuning />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalledTimes(1);
    });
    fireEvent.change(screen.getByLabelText("Demodulation mode"), {
      target: { value: "p25-cqpsk" },
    });
    await waitFor(() => {
      const last = vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1];
      expect(last?.proto).toBe("p25-cqpsk");
    });
  });

  it("snaps the view to the control channel when Hold is re-checked", async () => {
    // SDR centred at 851.0125 MHz with its control channel +50 kHz up.
    const CTRL_DEVICE = [
      {
        serial: "rtl-1",
        driver: "rtlsdr",
        role: "control",
        center_hz: 851_012_500,
        sample_rate_hz: 2_048_000,
        control_channel_hz: 851_062_500, // +50 kHz
      },
    ];
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(CTRL_DEVICE as never);
    // A live call on this SDR sits +100 kHz from centre. With Hold off the
    // view follows it; the poll's real fetch rejects under jsdom, so this
    // seeded snapshot survives.
    useShared.setState({
      activeCalls: [
        {
          grant: {
            group_id: 1,
            system: "Sys",
            protocol: "p25",
            frequency_hz: 851_112_500, // +100 kHz
          },
          device_serial: "rtl-1",
          started_at: "2026-01-01T00:00:00Z",
        },
      ] as never,
    });

    render(<Tuning />);

    const offset = (await screen.findByLabelText(
      "View offset from SDR centre in kHz (numeric)",
    )) as HTMLInputElement;
    // Follows the call: +100 kHz.
    await waitFor(() => expect(offset.value).toBe("100"));
    expect(screen.getByText(/following call/)).toBeInTheDocument();

    // Tick Hold → snap to the control channel (+50 kHz), not frozen on the call.
    fireEvent.click(screen.getByRole("checkbox"));
    await waitFor(() => expect(offset.value).toBe("50"));
    expect(screen.queryByText(/following call/)).not.toBeInTheDocument();
  });

  it("shows the carrier-error meter once a frame arrives", async () => {
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
        eye_soft: [],
        eye_sps: 0,
        dibits: [1],
        is_bits: false,
        base_idx: 0,
        carrier_offset_hz: 137,
        agc_level: 0.42,
        agc_target: 0.16,
        clock_mu: 0.5,
        clock_sps: 10,
        cma_error: 0,
      });
      return { close: vi.fn() };
    });

    render(<Tuning />);
    await waitFor(() => {
      expect(screen.getByText("137 Hz")).toBeInTheDocument();
    });
  });
});
