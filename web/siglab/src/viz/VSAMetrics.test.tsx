import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { VSAMetrics } from "./VSAMetrics";
import type { DemodMetrics } from "../api/types";

const base: DemodMetrics = {
  modulation: "cqpsk",
  evm_pct: 4.2,
  snr_estimate_db: 27.5,
  symbols_analyzed: 9000,
  peak_evm_pct: 11.3,
  mag_err_pct: 2.1,
  phase_err_deg: 3.4,
  carrier_freq_error_hz: 512,
  iq_gain_imbalance_db: 0.3,
  quadrature_error_deg: 1.1,
  origin_offset_pct: 0.8,
  evm_trace: [1, 2, 3, 4, 5],
  error_vector_spectrum: [-40, -38, -50, -42],
};

describe("VSAMetrics", () => {
  it("renders the VSA decomposition for the CQPSK path", () => {
    render(<VSAMetrics demod={base} />);
    expect(screen.getByText("Modulation quality (VSA)")).toBeInTheDocument();
    expect(screen.getByText("4.20 %")).toBeInTheDocument(); // RMS EVM
    expect(screen.getByText("11.30 %")).toBeInTheDocument(); // peak EVM
    expect(screen.getByText("512 Hz")).toBeInTheDocument(); // carrier error
    expect(screen.getByText("Phase err")).toBeInTheDocument();
  });

  it("hides phase/IQ fields on the C4FM (1-D) path", () => {
    render(<VSAMetrics demod={{ ...base, modulation: "c4fm" }} />);
    expect(screen.queryByText("Phase err")).not.toBeInTheDocument();
    expect(screen.queryByText("Quadrature err")).not.toBeInTheDocument();
    expect(screen.getByText("Magnitude err")).toBeInTheDocument();
  });
});
