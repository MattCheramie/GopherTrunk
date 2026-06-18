import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";

vi.mock("../api/client", () => ({
  api: { systems: vi.fn(), scanner: vi.fn() },
}));

import { api } from "../api/client";
import { useShared } from "../store/shared";
import { Systems } from "./Systems";

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

describe("Systems panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
    vi.mocked(api.scanner).mockResolvedValue({
      scan_mode: "all",
      systems: [],
      conventional: { enabled: false, channels: [] },
      tg_scan_count: 0,
      tg_total: 0,
    });
  });

  it("shows the DMR band plan summary in the table (#638)", async () => {
    vi.mocked(api.systems).mockResolvedValue([
      {
        name: "County T3",
        protocol: "dmr",
        control_channels: [851_000_000],
        dmr_band_plan: {
          linear: { base_hz: 851_012_500, spacing_hz: 12_500, offset: 1 },
        },
      },
    ]);

    render(<Systems />);
    await waitFor(() => expect(api.systems).toHaveBeenCalled());

    await waitFor(() => {
      const row = screen.getByText("County T3").closest("tr");
      expect(row).not.toBeNull();
      expect(row!.textContent).toMatch(/851\.0125 MHz/);
      expect(row!.textContent).toMatch(/12\.50 kHz/);
    });
  });

  it("renders an em dash for systems without a band plan", async () => {
    vi.mocked(api.systems).mockResolvedValue([
      { name: "Metro P25", protocol: "p25", control_channels: [851_000_000] },
    ]);

    render(<Systems />);
    await waitFor(() => {
      const row = screen.getByText("Metro P25").closest("tr");
      expect(row).not.toBeNull();
    });
    const row = screen.getByText("Metro P25").closest("tr")!;
    // Band plan cell shows the em dash placeholder.
    expect(row.textContent).toContain("—");
  });

  it("opens the detail modal when the scanner has a null systems list", async () => {
    // The daemon marshals an empty hunt list as JSON `null` (Go nil
    // slice), so `scanner.systems` can be null at runtime even though
    // the type says it's an array. The detail modal must not crash
    // calling `.find` on it.
    vi.mocked(api.scanner).mockResolvedValue({
      scan_mode: "all",
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      systems: null as any,
      conventional: { enabled: false, channels: [] },
      tg_scan_count: 0,
      tg_total: 0,
    });
    vi.mocked(api.systems).mockResolvedValue([
      { name: "Main_Site_1", protocol: "p25", control_channels: [765_000_000] },
    ]);

    render(<Systems />);
    await waitFor(() => expect(api.scanner).toHaveBeenCalled());

    fireEvent.click(screen.getByText("Main_Site_1"));

    // Modal renders the live-identity section instead of throwing.
    expect(
      await screen.findByText(/Network identity \(decoded live\)/i),
    ).toBeInTheDocument();
  });
});
