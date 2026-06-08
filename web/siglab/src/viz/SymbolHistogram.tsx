import { Bar } from "react-chartjs-2";
import type { SignalQuality } from "../api/types";
import { ensureChart } from "./chartSetup";

ensureChart();

// SymbolHistogram renders the recovered-symbol distribution (signal.go's
// SymbolHistogramPct). A clean 4-level C4FM control channel sits near 25% per
// bin; a collapsed slicer shows an empty or single-dominant bin.
export function SymbolHistogram({ signal }: { signal: SignalQuality }) {
  const pct = signal.symbol_histogram_pct ?? [];
  const labels = pct.map((_, i) => `sym ${i}`);
  return (
    <div className="card">
      <h3 className="mb-2 text-sm font-semibold">Symbol histogram</h3>
      <Bar
        data={{
          labels,
          datasets: [
            {
              label: "% of symbols",
              data: pct,
              backgroundColor: "rgba(56,189,248,0.6)",
            },
          ],
        }}
        options={{
          responsive: true,
          plugins: { legend: { display: false } },
          scales: { y: { beginAtZero: true, title: { display: true, text: "%" } } },
        }}
      />
      <p className="mt-2 text-xs text-muted">
        cardinality {signal.symbol_cardinality} · decode-error rate{" "}
        {signal.decode_error_rate_per_ksym.toFixed(2)} / 1000 sym
        {signal.iq_observed && (
          <>
            {" "}· IQ gain {signal.iq_gain_imbalance_db.toFixed(2)} dB · phase{" "}
            {signal.iq_phase_imbalance_deg.toFixed(2)}° · image rej{" "}
            {signal.iq_image_rejection_db.toFixed(1)} dB
          </>
        )}
        {signal.demod && (
          <>
            {" "}· demod ({signal.demod.modulation}) EVM{" "}
            {signal.demod.evm_pct.toFixed(1)}% · SNR≈{" "}
            {signal.demod.snr_estimate_db.toFixed(1)} dB
          </>
        )}
      </p>
    </div>
  );
}
