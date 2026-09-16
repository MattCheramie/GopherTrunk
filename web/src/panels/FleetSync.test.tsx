import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";

vi.mock("../api/fleetsync", () => ({
  fetchFleetSyncMessages: vi.fn(),
}));

import { fetchFleetSyncMessages } from "../api/fleetsync";
import { useShared } from "../store/shared";
import { FleetSync } from "./FleetSync";

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

describe("FleetSync panel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
  });

  it("renders an empty-state when no bursts are present", async () => {
    vi.mocked(fetchFleetSyncMessages).mockResolvedValue([]);
    render(<FleetSync />);
    await waitFor(() => {
      expect(screen.getByText(/No FleetSync bursts yet/)).toBeInTheDocument();
    });
  });

  it("renders a FleetSync-I ANI burst with fleet and unit", async () => {
    vi.mocked(fetchFleetSyncMessages).mockResolvedValue([
      {
        id: 1,
        received_at: "2026-09-15T12:34:56Z",
        fleet: 107,
        unit: 1772,
        fs2: false,
        body: "FleetSync ANI: fleet=107 unit=1772",
        raw_hex: "FE80083053059B6C",
        crc_ok: true,
        serial: "R1",
        frequency_hz: 462_562_500,
      },
    ]);
    render(<FleetSync />);
    await waitFor(() => {
      expect(screen.getByText("107")).toBeInTheDocument();
      expect(screen.getByText("1772")).toBeInTheDocument();
      expect(screen.getByText("FS-I")).toBeInTheDocument();
      expect(screen.getByText("FE80083053059B6C")).toBeInTheDocument();
      expect(screen.getByText("ok")).toBeInTheDocument();
      // #1184: the channel that produced the ID — frequency and SDR serial.
      expect(screen.getByText(/462\.5625 MHz/)).toBeInTheDocument();
      expect(screen.getByText(/R1/)).toBeInTheDocument();
    });
  });

  it("flags a FleetSync-II burst whose block check failed", async () => {
    vi.mocked(fetchFleetSyncMessages).mockResolvedValue([
      {
        id: 2,
        received_at: "2026-09-15T12:34:56Z",
        fleet: 107,
        unit: 1772,
        fs2: true,
        body: "FleetSync II ANI: fleet=107 unit=1772 (CRC?)",
        crc_ok: false,
      },
    ]);
    render(<FleetSync />);
    await waitFor(() => {
      expect(screen.getByText("FS-II")).toBeInTheDocument();
      expect(screen.getByText("fail")).toBeInTheDocument();
    });
  });

  it("surfaces fetch errors", async () => {
    vi.mocked(fetchFleetSyncMessages).mockRejectedValue(new Error("daemon down"));
    render(<FleetSync />);
    await waitFor(() => {
      expect(screen.getByText(/daemon down/)).toBeInTheDocument();
    });
  });
});
