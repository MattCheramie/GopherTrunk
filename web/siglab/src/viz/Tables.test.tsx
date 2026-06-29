import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { EventTimeline, GrantsTable, PDUTable } from "./Tables";
import type { EventRecord, GrantRecord, PDURecord } from "../api/types";

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

describe("PDUTable", () => {
  const pdus: PDURecord[] = [
    {
      seq: 0,
      offset_sec: 0.12,
      protocol: "p25p1",
      opcode: 0,
      opcode_name: "GRP_V_CH_GRANT",
      nac: 0x293,
      source_id: 5005,
      talkgroup: 101,
      raw_hex: "0001100000cc",
      crc_ok: true,
      fec_metric: 0,
      dibit_start: 100,
      dibit_len: 98,
    },
    {
      seq: 1,
      offset_sec: 0.2,
      protocol: "p25p1",
      opcode: 58,
      opcode_name: "RFSS_STS_BCST",
      nac: 0x293,
      raw_hex: "deadbeef",
      crc_ok: false,
      fec_metric: 3,
      dibit_start: 198,
      dibit_len: 98,
    },
  ];

  it("renders PDU rows and filters by opcode", () => {
    render(<PDUTable pdus={pdus} />);
    expect(screen.getByText(/Signaling \/ data PDUs \(2\/2\)/)).toBeInTheDocument();
    expect(screen.getByText("5005")).toBeInTheDocument(); // grant row source id

    fireEvent.change(screen.getByRole("combobox"), { target: { value: "RFSS_STS_BCST" } });
    expect(screen.getByText(/\(1\/2\)/)).toBeInTheDocument();
    expect(screen.queryByText("5005")).not.toBeInTheDocument(); // grant row filtered out
  });

  it("filters to CRC-OK blocks", () => {
    render(<PDUTable pdus={pdus} />);
    fireEvent.click(screen.getByLabelText(/CRC-OK only/));
    expect(screen.getByText(/\(1\/2\)/)).toBeInTheDocument();
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
