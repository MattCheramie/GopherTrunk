import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

// jsdom has no canvas; stub the chart so the component renders its tables.
vi.mock("react-chartjs-2", () => ({
  Line: () => <div data-testid="chart" />,
}));

import { WidebandSurvey } from "./WidebandSurvey";
import type { WidebandResult } from "../api/types";

function sampleResult(): WidebandResult {
  return {
    source: "wide.cf32",
    sample_rate_hz: 2_000_000,
    center_hz: 441_000_000,
    channel_rate_hz: 48_000,
    strategy: "ddc",
    spectrum: { center_hz: 441_000_000, sample_rate_hz: 2_000_000, bins: [-80, -40, -10, -50] },
    detected_count: 2,
    carriers: [
      {
        offset_hz: 48_000,
        freq_hz: 441_048_000,
        snr_db: 22.5,
        protocol: "dmr",
        confidence: 0.9,
        inconclusive: false,
        locked: true,
        role: "control",
        color_code: 1,
        system_id: 7,
        grants: [],
        score: 1.2,
      },
      {
        offset_hz: -48_000,
        freq_hz: 440_952_000,
        snr_db: 18.0,
        protocol: "dmr",
        confidence: 0.8,
        inconclusive: false,
        locked: false,
        role: "voice",
        color_code: 1,
        grants: [
          {
            offset_sec: 0,
            group_id: 1,
            source_id: 2,
            channel_id: 0,
            channel_num: 0,
            timeslot: 1,
            frequency_hz: 440_952_000,
            encrypted: false,
            emergency: false,
          },
        ],
        score: 0.9,
      },
    ],
    systems: [
      {
        protocol: "dmr",
        tier3: true,
        system_id: 7,
        color_code: 1,
        control_count: 1,
        voice_count: 1,
        lo_hz: 440_952_000,
        hi_hz: 441_048_000,
        carrier_idx: [0, 1],
        verdict: "DMR Tier III system",
      },
    ],
  };
}

describe("WidebandSurvey", () => {
  it("renders the survey summary, system rollup, and carrier rows", () => {
    render(<WidebandSurvey result={sampleResult()} />);

    expect(screen.getByTestId("chart")).toBeInTheDocument();
    expect(screen.getByText(/2 carrier peaks/)).toBeInTheDocument();

    // System rollup.
    expect(screen.getByText(/DMR Tier III system/)).toBeInTheDocument();
    expect(screen.getByText("440.9520–441.0480")).toBeInTheDocument();

    // Carrier rows with roles and absolute frequencies.
    expect(screen.getByText("441.0480")).toBeInTheDocument();
    expect(screen.getByText("440.9520")).toBeInTheDocument();
    expect(screen.getByText("control")).toBeInTheDocument();
    expect(screen.getByText("voice")).toBeInTheDocument();
  });

  it("names a non-decoding carrier from the reference DB", () => {
    const r = sampleResult();
    r.carriers.push({
      offset_hz: 96_000,
      freq_hz: 441_096_000,
      snr_db: 14,
      protocol: "", // did not decode
      confidence: 0.2,
      inconclusive: true,
      locked: false,
      role: "",
      grants: [],
      score: 0,
      blind: {
        symbol_rate_hz: 18000,
        symbol_rate_conf: 0.9,
        mod_class: "psk",
        levels: 4,
        occupied_bw_hz: 24000,
        method: "autocorr+spectral",
      },
      reference_matches: [
        {
          entry: {
            protocol: "tetra",
            display_name: "TETRA",
            mod_class: "psk",
            mod_labels: ["π/4-DQPSK"],
            symbol_rate_hz: 18000,
            channel_bw_hz: 25000,
            levels: 4,
            decodable: true,
          },
          score: 0.71,
          why: "25 kHz, π/4-DQPSK, 18000 sym/s",
        },
      ],
    });
    render(<WidebandSurvey result={r} />);
    expect(screen.getByText("TETRA (71%)")).toBeInTheDocument();
  });

  it("handles a survey with no carriers", () => {
    const empty: WidebandResult = {
      ...sampleResult(),
      detected_count: 0,
      carriers: [],
      systems: [],
    };
    render(<WidebandSurvey result={empty} />);
    expect(screen.getByText(/0 identified/)).toBeInTheDocument();
  });
});
