import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";

vi.mock("../api/spectrum", () => ({
  fetchSpectrumDevices: vi.fn(),
}));

vi.mock("../api/diag", () => ({
  openIQStream: vi.fn(),
}));

vi.mock("../api/symbols", () => ({
  openSymbolStream: vi.fn(),
}));

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

  it("re-subscribes the symbol stream when the demod mode changes", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue(ONE_DEVICE as never);

    render(<Constellation />);
    await waitFor(() => {
      expect(openSymbolStream).toHaveBeenCalledTimes(1);
    });
    fireEvent.change(screen.getByLabelText("Demodulation mode"), {
      target: { value: "p25-c4fm" },
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
