import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";

vi.mock("../api/spectrum", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/spectrum")>()),
  fetchSpectrumDevices: vi.fn(),
}));

vi.mock("../api/symbols", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/symbols")>();
  return { ...actual, openSymbolStream: vi.fn() };
});

import { fetchSpectrumDevices } from "../api/spectrum";
import { openSymbolStream } from "../api/symbols";
import { useShared } from "../store/shared";
import { SymbolScope } from "./SymbolScope";

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

const oneDevice = [
  {
    serial: "rtl-1",
    driver: "rtlsdr",
    role: "control",
    center_hz: 851_000_000,
    sample_rate_hz: 2_048_000,
  },
];

describe("Symbol scope panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
    window.localStorage.clear();
  });

  it("renders an empty-state when no SDRs are available", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([]);
    render(<SymbolScope />);
    await waitFor(() => {
      expect(screen.getByText("No SDRs available")).toBeInTheDocument();
    });
    expect(openSymbolStream).not.toHaveBeenCalled();
  });

  it("opens a symbol stream against the first device with a proto", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(oneDevice);
    vi.mocked(openSymbolStream).mockReturnValue({ close: vi.fn() });

    render(<SymbolScope />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalled();
    });
    const callArgs = vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1];
    expect(callArgs?.serial).toBe("rtl-1");
    // Default Mode is Auto; with no reported modulation it resolves to C4FM.
    expect(callArgs?.proto).toBe("p25-c4fm");
  });

  it("auto-selects CQPSK when the device reports an LSM system", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      { ...oneDevice[0], p25_modulation: "cqpsk" },
    ]);
    vi.mocked(openSymbolStream).mockReturnValue({ close: vi.fn() });

    render(<SymbolScope />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalled();
    });
    // Auto (default) follows the device's reported modulation.
    expect(vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1]?.proto).toBe(
      "p25-cqpsk",
    );
  });

  it("shows the live status pill when the WS reports open", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(oneDevice);
    vi.mocked(openSymbolStream).mockImplementation((_cfg, opts) => {
      opts.onStatus?.("open");
      return { close: vi.fn() };
    });

    render(<SymbolScope />);
    await waitFor(() => {
      expect(screen.getByText("live")).toBeInTheDocument();
    });
  });

  it("surfaces device-discovery errors", async () => {
    vi.mocked(fetchSpectrumDevices).mockRejectedValue(new Error("boom"));
    render(<SymbolScope />);
    await waitFor(() => {
      expect(screen.getByText(/boom/)).toBeInTheDocument();
    });
  });

  it("follows the active call on the selected SDR with an offset", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(oneDevice);
    vi.mocked(openSymbolStream).mockReturnValue({ close: vi.fn() });

    useShared.setState({
      activeCalls: [
        {
          grant: {
            system: "sys",
            protocol: "p25",
            group_id: 1,
            frequency_hz: 851_025_000,
          },
          device_serial: "rtl-1",
          started_at: "2026-06-07T00:00:00Z",
        },
      ] as never,
    });

    render(<SymbolScope />);
    await waitFor(() => {
      const last = vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1];
      expect(last?.offset).toBe(25_000);
    });
  });

  it("rests on the control channel by default when no call is active (#557)", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      { ...oneDevice[0], control_channel_hz: 851_025_000 },
    ]);
    vi.mocked(openSymbolStream).mockReturnValue({ close: vi.fn() });

    render(<SymbolScope />);
    await waitFor(() => {
      const last = vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1];
      // CC is 25 kHz above the 851.000 MHz centre.
      expect(last?.offset).toBe(25_000);
    });
  });

  it("prefers an active call over the control channel (#557)", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      { ...oneDevice[0], control_channel_hz: 851_025_000 },
    ]);
    vi.mocked(openSymbolStream).mockReturnValue({ close: vi.fn() });

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

    render(<SymbolScope />);
    await waitFor(() => {
      const last = vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1];
      // Follows the call (50 kHz), not the CC (25 kHz).
      expect(last?.offset).toBe(50_000);
    });
  });

  it("stays at centre when the device reports no control channel (#557)", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(oneDevice);
    vi.mocked(openSymbolStream).mockReturnValue({ close: vi.fn() });

    render(<SymbolScope />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalled();
    });
    expect(vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1]?.offset).toBe(0);
  });

  it("shows the tuned frequency before any symbol frame arrives", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(oneDevice);
    // Open the stream but never deliver a frame — a quiet channel.
    vi.mocked(openSymbolStream).mockReturnValue({ close: vi.fn() });

    render(<SymbolScope />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalled();
    });
    // The frequency view is derived from the device centre, not from a
    // decoded frame, so it renders immediately. (851.000 MHz centre.)
    expect(screen.getByText(/851\.0000 MHz/)).toBeInTheDocument();
    const freq = screen.getByLabelText("Tuned frequency, MHz") as HTMLInputElement;
    expect(Number(freq.value)).toBeCloseTo(851, 6);
  });

  it("exposes the mode select and Hold / Centre controls", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(oneDevice);
    vi.mocked(openSymbolStream).mockReturnValue({ close: vi.fn() });

    render(<SymbolScope />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalled();
    });
    expect(screen.getByText("Hold")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Centre" })).toBeInTheDocument();
    // Mode select offers both P25 demod paths.
    expect(screen.getByRole("option", { name: "P25 C4FM" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "P25 CQPSK" })).toBeInTheDocument();
  });
});
