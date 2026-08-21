import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { useShared } from "../store/shared";
import type { EventDTO } from "../api/types";
import { Events } from "./Events";

function setStore(events: EventDTO[]) {
  useShared.setState({
    serverURL: "http://localhost:8080",
    token: null,
    connected: true,
    wsStatus: "open",
    mutations: null,
    lastError: null,
    events,
    eventCap: 1000,
    activeCalls: [],
    devices: [],
    systems: [],
    talkgroups: [],
    health: null,
    audio: null,
    scanner: null,
  });
}

const evA: EventDTO = { kind: "grant", timestamp: "2026-06-19T05:00:00Z", payload: { tg: 1340 } };
const evB: EventDTO = { kind: "registration", timestamp: "2026-06-19T05:00:01Z", payload: { src: 42 } };

describe("Events panel", () => {
  beforeEach(() => setStore([evA, evB]));

  it("freezes the live stream while an event is expanded for inspection", async () => {
    const user = userEvent.setup();
    render(<Events />);

    // Live by default — the hint invites inspection.
    expect(screen.getByText(/Click an event to inspect/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Pause" })).toBeInTheDocument();

    // Expand an event by clicking its row.
    await user.click(screen.getByText("registration"));

    // Now frozen for inspection: Resume shows and the freeze hint explains why.
    expect(screen.getByRole("button", { name: "Resume" })).toBeInTheDocument();
    expect(screen.getByText(/Frozen while you inspect/)).toBeInTheDocument();

    // New events arriving must NOT push into the frozen view.
    setStore([evA, evB, { kind: "patch", timestamp: "2026-06-19T05:00:02Z", payload: {} }]);
    expect(screen.queryByText("patch")).not.toBeInTheDocument();

    // Collapsing the event resumes the live stream.
    await user.click(screen.getByText("registration"));
    expect(screen.getByRole("button", { name: "Pause" })).toBeInTheDocument();
  });

  it("orders rows by event timestamp, not store arrival order", () => {
    // Regression: an event that ARRIVES late but carries an older wall-clock
    // timestamp (e.g. an audio.state re-pushed on a state change) used to float
    // to the top because the panel reverse()d the store's arrival order. It must
    // sort below the newer-timestamped events instead ("13:54 above 16:34" bug).
    const newer: EventDTO = { kind: "call.release", timestamp: "2026-08-20T16:34:15Z", payload: { g: 1 } };
    const staleLateArrival: EventDTO = { kind: "audio.state", timestamp: "2026-08-20T13:54:57Z", payload: { backend_enabled: false } };
    // Arrival order in the store: the stale-timestamp event arrives LAST.
    setStore([newer, staleLateArrival]);
    render(<Events />);

    const kinds = screen
      .getAllByText(/call\.release|audio\.state/)
      .map((el) => el.textContent);
    expect(kinds[0]).toBe("call.release"); // newest timestamp on top
    expect(kinds[1]).toBe("audio.state"); // stale timestamp sinks below
  });

  it("keeps a manual pause when an expanded event is collapsed", async () => {
    const user = userEvent.setup();
    render(<Events />);

    await user.click(screen.getByRole("button", { name: "Pause" }));
    await user.click(screen.getByText("grant")); // expand while already paused
    await user.click(screen.getByText("grant")); // collapse

    // Manual pause is preserved (not auto-resumed on collapse).
    expect(screen.getByRole("button", { name: "Resume" })).toBeInTheDocument();
  });
});
