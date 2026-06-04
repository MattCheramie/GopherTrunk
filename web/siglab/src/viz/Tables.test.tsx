import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { EventTimeline, GrantsTable } from "./Tables";
import type { EventRecord, GrantRecord } from "../api/types";

describe("GrantsTable", () => {
  it("renders grant rows", () => {
    const grants: GrantRecord[] = [
      {
        offset_sec: 1.25,
        group_id: 101,
        source_id: 5005,
        channel_id: 0,
        channel_num: 0,
        timeslot: 1,
        frequency_hz: 851_000_000,
        encrypted: true,
        emergency: false,
      },
    ];
    render(<GrantsTable grants={grants} />);
    expect(screen.getByText("Grants (1)")).toBeInTheDocument();
    expect(screen.getByText("101")).toBeInTheDocument();
    expect(screen.getByText("5005")).toBeInTheDocument();
  });

  it("shows an empty state", () => {
    render(<GrantsTable grants={[]} />);
    expect(screen.getByText(/No traffic grants/)).toBeInTheDocument();
  });
});

describe("EventTimeline", () => {
  it("summarizes event kinds", () => {
    const events: EventRecord[] = [
      { seq: 0, offset_sec: 0.1, kind: "cc_locked", fields: { NAC: 659 } },
      { seq: 1, offset_sec: 0.2, kind: "grant", fields: { GroupID: 1 } },
      { seq: 2, offset_sec: 0.3, kind: "grant", fields: { GroupID: 2 } },
    ];
    render(<EventTimeline events={events} />);
    expect(screen.getByText("Events (3)")).toBeInTheDocument();
    expect(screen.getByText("×2")).toBeInTheDocument();
  });
});
