import { describe, it, expect, vi, beforeEach } from "vitest";
import { render as rtlRender, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

// Panels render react-router links / read search params, so every
// render needs a Router context.
function render(ui: Parameters<typeof rtlRender>[0]) {
  return rtlRender(<MemoryRouter>{ui}</MemoryRouter>);
}

vi.mock("../api/spectrum", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/spectrum")>();
  // Keep the real pure helpers (parentSerial, defaultSymbolDevice); only the
  // network call is stubbed.
  return { ...actual, fetchSpectrumDevices: vi.fn() };
});

vi.mock("../api/diag", () => ({
  openIQStream: vi.fn(),
}));

vi.mock("../api/symbols", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/symbols")>();
  return { ...actual, openSymbolStream: vi.fn() };
});

import { fetchSpectrumDevices } from "../api/spectrum";
import { openIQStream } from "../api/diag";
import { openSymbolStream } from "../api/symbols";
import { useShared } from "../store/shared";
import { Constellation } from "./Constellation";

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
    // Report CQPSK so Auto resolves to the symbol-decision stream (C4FM has
    // no symbol constellation and now defaults to the raw IQ ring).
    p25_modulation: "cqpsk",
  },
];

describe("Constellation panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
    window.localStorage.clear();
    vi.mocked(openIQStream).mockReturnValue({ close: vi.fn() });
    vi.mocked(openSymbolStream).mockReturnValue({ close: vi.fn() });
  });

  it("renders an empty-state when no SDRs are available", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([]);
    render(<Constellation />);
    await waitFor(() => {
      expect(screen.getByText("No SDRs available")).toBeInTheDocument();
    });
    expect(openSymbolStream).not.toHaveBeenCalled();
    expect(openIQStream).not.toHaveBeenCalled();
  });

  it("defaults to the symbols view and opens a symbol stream", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);

    render(<Constellation />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalledTimes(1);
    });
    const callArgs = vi.mocked(openSymbolStream).mock.calls[0]?.[1];
    expect(callArgs?.serial).toBe("rtl-1");
    expect(callArgs?.proto).toBeTruthy();
    // The raw IQ stream stays closed until the operator switches View.
    expect(openIQStream).not.toHaveBeenCalled();
  });

  it("auto-selects CQPSK from the device's reported modulation", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      { ...ONE_DEVICE[0], p25_modulation: "cqpsk" },
    ] as never);

    render(<Constellation />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalled();
    });
    // Default Mode is Auto; it follows the device's reported modulation.
    expect(vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1]?.proto).toBe(
      "p25-cqpsk",
    );
  });

  it("switches to the vector scope and opens a raw IQ stream", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);

    render(<Constellation />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalled();
    });
    fireEvent.change(screen.getByLabelText("Constellation source"), {
      target: { value: "raw" },
    });
    await waitFor(() => {
      expect(openIQStream).toHaveBeenCalledTimes(1);
    });
    const callArgs = vi.mocked(openIQStream).mock.calls[0]?.[1];
    expect(callArgs?.serial).toBe("rtl-1");
    expect(callArgs?.rate).toBeGreaterThan(0);
  });

  it("re-subscribes the symbol stream when switching to CQPSK", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);

    render(<Constellation />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalledTimes(1);
    });
    // Default Mode is Auto, resolving to CQPSK here; flip to C4FM (raw IQ
    // ring) and back to confirm CQPSK re-subscribes the symbol stream.
    fireEvent.change(screen.getByLabelText("Demodulation mode"), {
      target: { value: "p25-c4fm" },
    });
    fireEvent.change(screen.getByLabelText("Demodulation mode"), {
      target: { value: "p25-cqpsk" },
    });
    await waitFor(() => {
      const last = vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1];
      expect(last?.proto).toBe("p25-cqpsk");
    });
  });

  it("shows the C4FM IQ ring by default and can switch to soft levels", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);

    render(<Constellation />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalledTimes(1);
    });

    // Switching Mode to C4FM (default Display = IQ ring) opens the raw IQ
    // stream, not the symbol stream — C4FM has no symbol constellation.
    fireEvent.change(screen.getByLabelText("Demodulation mode"), {
      target: { value: "p25-c4fm" },
    });
    await waitFor(() => {
      expect(openIQStream).toHaveBeenCalledTimes(1);
    });

    // Choosing "Soft levels" falls back to the symbol stream with C4FM.
    fireEvent.change(screen.getByLabelText("C4FM display"), {
      target: { value: "soft" },
    });
    await waitFor(() => {
      const last = vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1];
      expect(last?.proto).toBe("p25-c4fm");
    });
  });

  it("shows the live status pill when the WS reports open", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);
    vi.mocked(openSymbolStream).mockImplementation((_cfg, opts) => {
      opts.onStatus?.("open");
      return { close: vi.fn() };
    });

    render(<Constellation />);
    await waitFor(() => {
      expect(screen.getByText("live")).toBeInTheDocument();
    });
  });

  it("surfaces device-discovery errors", async () => {
    vi.mocked(fetchSpectrumDevices).mockRejectedValue(new Error("boom"));
    render(<Constellation />);
    await waitFor(() => {
      expect(screen.getByText(/boom/)).toBeInTheDocument();
    });
  });

  it("follows the active call on the selected SDR with a frequency offset", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      {
        serial: "rtl-1",
        driver: "rtlsdr",
        role: "control",
        center_hz: 851_000_000,
        sample_rate_hz: 2_048_000,
        p25_modulation: "cqpsk",
      },
    ] as never);

    // A locked voice call 25 kHz above centre → offset should be 25000 Hz.
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

    render(<Constellation />);
    await waitFor(() => {
      const calls = vi.mocked(openSymbolStream).mock.calls;
      const last = calls[calls.length - 1]?.[1];
      expect(last?.offset).toBe(25_000);
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
        p25_modulation: "cqpsk",
        control_channel_hz: 851_025_000,
      },
    ] as never);

    render(<Constellation />);
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
        p25_modulation: "cqpsk",
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

    render(<Constellation />);
    await waitFor(() => {
      const last = vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1];
      // Follows the call (50 kHz), not the CC (25 kHz).
      expect(last?.offset).toBe(50_000);
    });
  });

  it("stays at centre when the device reports no control channel (#557)", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);

    render(<Constellation />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalled();
    });
    expect(vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1]?.offset).toBe(0);
  });

  it("exposes a zoom control and an absolute-frequency (MHz) input", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);

    render(<Constellation />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalled();
    });
    expect(screen.getByLabelText("Constellation zoom")).toBeInTheDocument();
    expect(screen.getByLabelText("Tuned frequency, MHz")).toBeInTheDocument();
  });

  it("re-tunes when the operator enters a precise kHz offset", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);

    render(<Constellation />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalled();
    });
    const input = screen.getByLabelText(
      "View offset from SDR centre in kHz (numeric)",
    );
    fireEvent.change(input, { target: { value: "12.5" } });
    await waitFor(() => {
      const last = vi.mocked(openSymbolStream).mock.calls.at(-1)?.[1];
      expect(last?.offset).toBe(12_500);
    });
  });

  it("exposes DC-block, auto-scale, and hold view controls", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);

    render(<Constellation />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalled();
    });
    // Hold + DC-block + auto-scale checkboxes, plus a Centre button.
    expect(screen.getAllByRole("checkbox").length).toBeGreaterThanOrEqual(3);
    expect(screen.getByText("Hold")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Centre" }),
    ).toBeInTheDocument();
  });
});
