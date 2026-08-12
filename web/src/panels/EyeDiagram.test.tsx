import { describe, it, expect, vi, beforeEach } from "vitest";
import { render as rtlRender, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

// Panels render react-router links / read search params, so every
// render needs a Router context.
function render(ui: Parameters<typeof rtlRender>[0]) {
  return rtlRender(<MemoryRouter>{ui}</MemoryRouter>);
}

vi.mock("../api/spectrum", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/spectrum")>()),
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

  it("rests on the control channel by default when no call is active (#557)", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      {
        serial: "rtl-1",
        driver: "rtlsdr",
        role: "control",
        center_hz: 851_000_000,
        sample_rate_hz: 2_048_000,
        control_channel_hz: 851_025_000,
      },
    ] as never);

    render(<EyeDiagram />);
    await waitFor(() => {
      const last = vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1];
      // CC is 25 kHz above the 851.000 MHz centre.
      expect(last?.offset).toBe(25_000);
    });
  });

  it("prefers an active call over the control channel (#557)", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      {
        serial: "rtl-1",
        driver: "rtlsdr",
        role: "control",
        center_hz: 851_000_000,
        sample_rate_hz: 2_048_000,
        control_channel_hz: 851_025_000,
      },
    ] as never);

    useShared.setState({
      activeCalls: [
        {
          grant: {
            system: "sys",
            protocol: "p25",
            group_id: 1,
            frequency_hz: 851_050_000,
          },
          device_serial: "rtl-1",
          started_at: "2026-06-07T00:00:00Z",
        },
      ] as never,
    });

    render(<EyeDiagram />);
    await waitFor(() => {
      const last = vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1];
      // Follows the call (50 kHz), not the CC (25 kHz).
      expect(last?.offset).toBe(50_000);
    });
  });

  it("stays at centre when the device reports no control channel (#557)", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);

    render(<EyeDiagram />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalled();
    });
    expect(vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1]?.offset).toBe(0);
  });

  it("parks the view on the control channel when Hold is re-checked (#557)", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      {
        serial: "rtl-1",
        driver: "rtlsdr",
        role: "control",
        center_hz: 851_000_000,
        sample_rate_hz: 2_048_000,
        control_channel_hz: 851_025_000,
      },
    ] as never);
    // Hold off + an active call → the view follows the call (50 kHz).
    useShared.setState({
      activeCalls: [
        {
          grant: {
            system: "sys",
            protocol: "p25",
            group_id: 1,
            frequency_hz: 851_050_000,
          },
          device_serial: "rtl-1",
          started_at: "2026-06-07T00:00:00Z",
        },
      ] as never,
    });

    render(<EyeDiagram />);
    await waitFor(() => {
      expect(vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1]?.offset).toBe(
        50_000,
      );
    });

    // Re-checking Hold must jump the view to the control channel (25 kHz),
    // not freeze on the (possibly stale) call frequency.
    fireEvent.click(screen.getByRole("checkbox"));
    await waitFor(() => {
      expect(vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1]?.offset).toBe(
        25_000,
      );
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
        carrier_offset_hz: 0,
        agc_level: 0,
        agc_target: 0,
        clock_mu: 0,
        clock_sps: 10,
        cma_error: 0,
      });
      return { close: vi.fn() };
    });

    render(<EyeDiagram />);
    await waitFor(() => {
      expect(screen.getByText(/samples\/symbol/)).toBeInTheDocument();
    });
  });
});
