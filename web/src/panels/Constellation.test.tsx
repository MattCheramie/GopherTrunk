import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";

vi.mock("../api/spectrum", () => ({
  fetchSpectrumDevices: vi.fn(),
}));

vi.mock("../api/diag", () => ({
  openIQStream: vi.fn(),
}));

import { fetchSpectrumDevices } from "../api/spectrum";
import { openIQStream } from "../api/diag";
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

describe("Constellation panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
  });

  it("renders an empty-state when no SDRs are available", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([]);
    render(<Constellation />);
    await waitFor(() => {
      expect(screen.getByText("No SDRs available")).toBeInTheDocument();
    });
    expect(openIQStream).not.toHaveBeenCalled();
  });

  it("opens a diag IQ stream against the first device", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      {
        serial: "rtl-1",
        driver: "rtlsdr",
        role: "control",
        center_hz: 851_012_500,
        sample_rate_hz: 2_048_000,
      },
    ]);
    vi.mocked(openIQStream).mockReturnValue({ close: vi.fn() });

    render(<Constellation />);
    await waitFor(() => {
      expect(openIQStream).toHaveBeenCalledTimes(1);
    });
    const callArgs = vi.mocked(openIQStream).mock.calls[0]?.[1];
    expect(callArgs?.serial).toBe("rtl-1");
    expect(callArgs?.rate).toBeGreaterThan(0);
  });

  it("shows the live status pill when the WS reports open", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      {
        serial: "rtl-1",
        driver: "rtlsdr",
        role: "control",
        center_hz: 0,
        sample_rate_hz: 2_048_000,
      },
    ]);
    vi.mocked(openIQStream).mockImplementation((_cfg, opts) => {
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
    ]);
    vi.mocked(openIQStream).mockReturnValue({ close: vi.fn() });

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
      const calls = vi.mocked(openIQStream).mock.calls;
      const last = calls[calls.length - 1]?.[1];
      expect(last?.offset).toBe(25_000);
    });
  });

  it("exposes DC-block, auto-scale, and hold view controls", async () => {
    vi.mocked(fetchSpectrumDevices).mockResolvedValue([
      {
        serial: "rtl-1",
        driver: "rtlsdr",
        role: "control",
        center_hz: 851_000_000,
        sample_rate_hz: 2_048_000,
      },
    ]);
    vi.mocked(openIQStream).mockReturnValue({ close: vi.fn() });

    render(<Constellation />);
    await waitFor(() => {
      expect(openIQStream).toHaveBeenCalled();
    });
    // Hold + DC-block + auto-scale checkboxes, plus a Centre button.
    expect(screen.getAllByRole("checkbox").length).toBeGreaterThanOrEqual(3);
    expect(screen.getByText("Hold")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Centre" }),
    ).toBeInTheDocument();
  });
});
