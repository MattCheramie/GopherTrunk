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

  it("orders Recent activity by event timestamp, not store arrival order", () => {
    // Regression: the feed used to .slice(-40).reverse() the arrival-ordered
    // ring, so an event that ARRIVES late but carries an older wall-clock
    // timestamp (e.g. an audio.state re-pushed on a state change) floated to
    // the top — the "23:24 shown above 02:17" bug. It must sort below the
    // newer-timestamped events instead. Same fix as the Events tab.
    useShared.setState({
      events: [
        // Arrival order: the stale-timestamp event arrives LAST.
        { kind: "call.release", timestamp: "2026-08-22T02:17:42Z", payload: {} },
        { kind: "audio.state", timestamp: "2026-08-21T23:24:13Z", payload: {} },
      ],
    });
    renderDash();
    const kinds = screen
      .getAllByText(/call\.release|audio\.state/)
      .map((el) => el.textContent);
    expect(kinds[0]).toBe("call.release"); // newest timestamp on top
    expect(kinds[1]).toBe("audio.state"); // stale timestamp sinks below
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
