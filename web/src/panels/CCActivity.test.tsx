import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import { useShared } from "../store/shared";
import type { EventDTO } from "../api/types";
import { CCActivity } from "./CCActivity";

// CCActivity embeds react-router <Link> for clickable RIDs, so every
// render() needs a Router context.
function renderPanel() {
  return render(
    <MemoryRouter>
      <CCActivity />
    </MemoryRouter>,
  );
}

function setEvents(events: EventDTO[]) {
  useShared.setState({
    serverURL: "http://localhost:8080",
    token: null,
    connected: true,
    wsStatus: "idle",
    mutations: null,
    lastError: null,
    events,
    activeCalls: [],
    devices: [],
    systems: [],
    talkgroups: [],
    health: null,
    audio: null,
    scanner: null,
  });
}

describe("CCActivity panel", () => {
  beforeEach(() => {
    setEvents([]);
  });

  it("renders the empty state when no CC events are in the store", () => {
    renderPanel();
    expect(screen.getByText(/Nothing here yet/)).toBeInTheDocument();
  });

  it("ignores non-CC events (silence on unrelated kinds)", () => {
    setEvents([
      {
        kind: "tone.alert",
        timestamp: "2026-05-26T12:00:00Z",
        payload: { profile: "knox" },
      },
      {
        kind: "sdr.attached",
        timestamp: "2026-05-26T12:00:00Z",
        payload: { serial: "rtl-1" },
      },
    ]);
    renderPanel();
    expect(screen.getByText(/Nothing here yet/)).toBeInTheDocument();
  });

  it("renders grant events with talkgroup and frequency", () => {
    setEvents([
      {
        kind: "grant",
        timestamp: "2026-05-26T12:34:56Z",
        payload: {
          system: "Metro P25",
          protocol: "p25",
          group_id: 1234,
          source_id: 5678,
          frequency_hz: 851_012_500,
          emergency: true,
        },
      },
    ]);
    renderPanel();
    // "Grant" appears in both the kind-filter <option> and the row's
    // Kind cell; assert via the row containing the system label.
    const row = screen.getByText("Metro P25").closest("tr");
    expect(row).not.toBeNull();
    expect(row!.textContent).toMatch(/Grant/);
    expect(row!.textContent).toMatch(/TG 1234/);
    expect(row!.textContent).toMatch(/851\.0125 MHz/);
    expect(row!.textContent).toMatch(/EMERG/);
  });

  it("collapses repeated identical grants into one row with a count (grouped by default)", () => {
    const grant = (ts: string) => ({
      kind: "grant",
      timestamp: ts,
      payload: { system: "Metro P25", protocol: "p25", group_id: 1234, frequency_hz: 851_012_500 },
    });
    setEvents([grant("2026-05-26T12:00:00Z"), grant("2026-05-26T12:00:03Z"), grant("2026-05-26T12:00:07Z")]);
    renderPanel();
    const rows = screen.getAllByText("Metro P25").map((el) => el.closest("tr"));
    // Three identical grants collapse to a single row carrying ×3.
    expect(rows).toHaveLength(1);
    expect(rows[0]!.textContent).toMatch(/×3/);
  });

  it("renders affiliation events with response code", () => {
    setEvents([
      {
        kind: "affiliation",
        timestamp: "2026-05-26T12:00:00Z",
        payload: {
          system: "Metro P25",
          radio_id: 999,
          group_id: 100,
          response_code: 2,
        },
      },
    ]);
    renderPanel();
    const row = screen.getByText("Metro P25").closest("tr");
    expect(row).not.toBeNull();
    expect(row!.textContent).toMatch(/Affiliation/);
    expect(row!.textContent).toMatch(/radio 999/);
    expect(row!.textContent).toMatch(/TG 100/);
    expect(row!.textContent).toMatch(/resp 2/);
  });

  it("supports filtering by kind", async () => {
    setEvents([
      {
        kind: "grant",
        timestamp: "2026-05-26T12:00:00Z",
        payload: { system: "Sys-A", group_id: 1 },
      },
      {
        kind: "affiliation",
        timestamp: "2026-05-26T12:01:00Z",
        payload: { system: "Sys-A", radio_id: 100, group_id: 1 },
      },
    ]);
    renderPanel();
    // Two rows initially.
    expect(screen.getByText("2 matching events")).toBeInTheDocument();

    await userEvent.selectOptions(screen.getByRole("combobox"), "grant");
    expect(screen.getByText("1 matching event")).toBeInTheDocument();
  });

  it("supports filtering by system substring", async () => {
    setEvents([
      {
        kind: "grant",
        timestamp: "2026-05-26T12:00:00Z",
        payload: { system: "Metro P25", group_id: 1 },
      },
      {
        kind: "grant",
        timestamp: "2026-05-26T12:01:00Z",
        payload: { system: "County DMR", group_id: 2 },
      },
    ]);
    renderPanel();
    expect(screen.getByText("Metro P25")).toBeInTheDocument();
    expect(screen.getByText("County DMR")).toBeInTheDocument();

    await userEvent.type(screen.getByPlaceholderText(/filter system/), "metro");
    expect(screen.getByText("Metro P25")).toBeInTheDocument();
    expect(screen.queryByText("County DMR")).not.toBeInTheDocument();
  });

  it("renders patch add events with member TGs promoted over the SG", () => {
    setEvents([
      {
        kind: "patch",
        timestamp: "2026-05-26T12:00:00Z",
        payload: {
          system: "Metro P25",
          super_group: 999,
          members: [101, 102, 103],
          add: true,
        },
      },
    ]);
    renderPanel();
    const row = screen.getByText("Metro P25").closest("tr");
    expect(row).not.toBeNull();
    expect(row!.textContent).toMatch(/Patch/);
    expect(row!.textContent).toMatch(/Patch Active:/);
    expect(row!.textContent).toMatch(/TG 101/);
    expect(row!.textContent).toMatch(/TG 102/);
    expect(row!.textContent).toMatch(/TG 103/);
    expect(row!.textContent).toMatch(/\[SG 999\]/);
  });

  it("resolves member talkgroup aliases from the shared TGDB", () => {
    setEvents([
      {
        kind: "patch",
        timestamp: "2026-05-26T12:00:00Z",
        payload: {
          system: "Metro P25",
          super_group: 30201,
          members: [101, 102],
          add: true,
        },
      },
    ]);
    useShared.setState({
      talkgroups: [
        { id: 101, alpha_tag: "NORTH DISP 1" },
        { id: 102, alpha_tag: "NORTH TAC 2" },
      ],
    });
    renderPanel();
    const row = screen.getByText("Metro P25").closest("tr");
    expect(row).not.toBeNull();
    expect(row!.textContent).toMatch(/Patch Active:/);
    expect(row!.textContent).toMatch(/NORTH DISP 1 \(101\)/);
    expect(row!.textContent).toMatch(/NORTH TAC 2 \(102\)/);
    expect(row!.textContent).toMatch(/\[SG 30201\]/);
  });

  it("renders patch cancel events when add is false", () => {
    setEvents([
      {
        kind: "patch",
        timestamp: "2026-05-26T12:00:00Z",
        payload: {
          system: "Metro P25",
          super_group: 999,
          members: [101],
          add: false,
        },
      },
    ]);
    renderPanel();
    const row = screen.getByText("Metro P25").closest("tr");
    expect(row).not.toBeNull();
    expect(row!.textContent).toMatch(/Patch Cancelled:/);
    expect(row!.textContent).toMatch(/TG 101/);
    expect(row!.textContent).toMatch(/\[SG 999\]/);
  });

  it("renders observed DMR grants with LCN, timeslot and source (#638)", () => {
    setEvents([
      {
        kind: "dmr.grant.observed",
        timestamp: "2026-05-26T12:00:00Z",
        payload: {
          system: "County T3",
          color_code: 1,
          lcn: 12,
          timeslot: 1, // raw 1 = TS2
          group_id: 1001,
          source_id: 2002,
          cc_freq_hz: 851_000_000,
        },
      },
    ]);
    renderPanel();
    const row = screen.getByText("County T3").closest("tr");
    expect(row).not.toBeNull();
    expect(row!.textContent).toMatch(/DMR grant/);
    expect(row!.textContent).toMatch(/LCN 12/);
    expect(row!.textContent).toMatch(/TS2/);
    expect(row!.textContent).toMatch(/TG 1001/);
    expect(row!.textContent).toMatch(/2002/);
    expect(row!.textContent).toMatch(/CC 851\.0000 MHz/);
  });

  it("renders a learned DMR band plan with spacing and confidence (#638)", () => {
    setEvents([
      {
        kind: "dmr.bandplan.learned",
        timestamp: "2026-05-26T12:00:00Z",
        payload: {
          system: "County T3",
          base_hz: 851_012_500,
          spacing_hz: 12_500,
          offset: 1,
          num_pairs: 6,
          confidence: 0.97,
        },
      },
    ]);
    renderPanel();
    const row = screen.getByText("County T3").closest("tr");
    expect(row).not.toBeNull();
    expect(row!.textContent).toMatch(/Band plan learned/);
    expect(row!.textContent).toMatch(/851\.0125 MHz/);
    expect(row!.textContent).toMatch(/12\.50 kHz spacing/);
    expect(row!.textContent).toMatch(/6 pairs/);
    expect(row!.textContent).toMatch(/conf 97%/);
  });

  it("renders talker alias events", () => {
    setEvents([
      {
        kind: "talker.alias",
        timestamp: "2026-05-26T12:00:00Z",
        payload: {
          system: "Metro P25",
          source: 1234,
          alias: "ENG-5",
        },
      },
    ]);
    renderPanel();
    const row = screen.getByText("Metro P25").closest("tr");
    expect(row).not.toBeNull();
    expect(row!.textContent).toMatch(/Talker alias/);
    expect(row!.textContent).toMatch(/radio 1234: "ENG-5"/);
  });
});
