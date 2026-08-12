import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

vi.mock("../api/client", () => ({
  api: {
    health: vi.fn().mockResolvedValue({ status: "ok" }),
    activeCalls: vi.fn().mockResolvedValue([]),
    devices: vi.fn().mockResolvedValue([]),
    systems: vi.fn().mockResolvedValue([]),
  },
}));

import { useShared } from "../store/shared";
import { Dashboard } from "./Dashboard";

function resetStore() {
  useShared.setState({
    serverURL: "http://localhost:8080",
    token: null,
    connected: true,
    wsStatus: "open",
    events: [],
    activeCalls: [],
    devices: [],
    systems: [],
    health: null,
  });
}

function renderDash() {
  return render(
    <MemoryRouter>
      <Dashboard />
    </MemoryRouter>,
  );
}

describe("Dashboard landing", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetStore();
  });

  it("leads with Scan and Hunt quick actions linking to the workflows", () => {
    renderDash();
    const scan = screen.getByRole("link", { name: /Scan known systems/i });
    const hunt = screen.getByRole("link", { name: /Hunt for new systems/i });
    expect(scan).toHaveAttribute("href", "/scanner");
    expect(hunt).toHaveAttribute("href", "/hunt");
  });

  it("summarizes a recent event with its talkgroup and system, not just the kind", async () => {
    useShared.setState({
      events: [
        {
          kind: "call.grant",
          timestamp: "2026-08-12T10:00:00Z",
          payload: { group_id: 1001, source_id: 4242, system: "CountyP25" },
        },
      ],
    });
    renderDash();
    // The compact feed summarizes the payload (TG · source · system), so it
    // says what happened rather than only naming the event kind.
    await waitFor(() => {
      expect(screen.getByText(/TG 1001.*4242.*CountyP25/)).toBeInTheDocument();
    });
  });

  it("shows the active-call roster with the transmitting radio", async () => {
    useShared.setState({
      activeCalls: [
        {
          grant: {
            system: "CountyP25",
            protocol: "p25",
            group_id: 1001,
            source_id: 4242,
            frequency_hz: 851_000_000,
          },
          talkgroup: { id: 1001, alpha_tag: "Dispatch" },
          device_serial: "SDR1",
          started_at: "2026-08-12T10:00:00Z",
          following: true,
        },
      ] as never,
    });
    renderDash();
    expect(screen.getByText("TG 1001")).toBeInTheDocument();
    expect(screen.getByText("Dispatch")).toBeInTheDocument();
  });
});
