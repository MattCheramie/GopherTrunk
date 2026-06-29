import { Line } from "react-chartjs-2";
import type { DemodMetrics } from "../api/types";
import { ensureChart } from "./chartSetup";

ensureChart();

// VSAMetrics renders the vector-signal-analyzer decomposition of the recovered
// constellation — the lab-VSA breakdown (carrier error, EVM split into
// magnitude/phase, IQ imbalance, origin offset) plus the per-symbol EVM trace
// and the error-vector spectrum. Mirrors siglab.DemodMetrics; phase/IQ/origin
// fields are meaningful only on the CQPSK path.
export function VSAMetrics({ demod }: { demod: DemodMetrics }) {
  const cqpsk = demod.modulation === "cqpsk";
  const evm = demod.evm_trace ?? [];
  const evs = demod.error_vector_spectrum ?? [];

  // Decimate the trace for a responsive chart.
  const max = 4000;
  const step = Math.max(1, Math.floor(evm.length / max));
  const evmLabels: number[] = [];
  const evmSeries: number[] = [];
  for (let i = 0; i < evm.length; i += step) {
    evmLabels.push(i);
    evmSeries.push(evm[i]);
  }

  return (
    <div className="card">
      <h3 className="mb-2 text-sm font-semibold">Modulation quality (VSA)</h3>
      <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs sm:grid-cols-3">
        <Metric label="RMS EVM" value={`${demod.evm_pct.toFixed(2)} %`} />
        <Metric label="Peak EVM" value={`${demod.peak_evm_pct.toFixed(2)} %`} />
        <Metric label="SNR est." value={`${demod.snr_estimate_db.toFixed(1)} dB`} />
        <Metric label="Magnitude err" value={`${demod.mag_err_pct.toFixed(2)} %`} />
        {cqpsk && <Metric label="Phase err" value={`${demod.phase_err_deg.toFixed(2)}°`} />}
        <Metric label="Carrier err" value={`${demod.carrier_freq_error_hz.toFixed(0)} Hz`} />
        {cqpsk && (
          <Metric label="IQ gain imb." value={`${demod.iq_gain_imbalance_db.toFixed(2)} dB`} />
        )}
        {cqpsk && (
          <Metric label="Quadrature err" value={`${demod.quadrature_error_deg.toFixed(2)}°`} />
        )}
        {cqpsk && <Metric label="Origin offset" value={`${demod.origin_offset_pct.toFixed(2)} %`} />}
      </dl>

      {evmSeries.length > 1 && (
        <div className="mt-3">
          <Line
            data={{
              labels: evmLabels,
              datasets: [
                {
                  label: "EVM (%)",
                  data: evmSeries,
                  borderColor: "rgba(56,189,248,0.9)",
                  pointRadius: 0,
                  borderWidth: 1,
                },
              ],
            }}
            options={{
              responsive: true,
              plugins: { legend: { display: false }, title: { display: true, text: "EVM vs symbol" } },
              scales: {
                x: { title: { display: true, text: "symbol index" }, ticks: { maxTicksLimit: 10 } },
                y: { title: { display: true, text: "EVM %" } },
              },
            }}
          />
        </div>
      )}

      {evs.length > 1 && (
        <div className="mt-3">
          <Line
            data={{
              labels: evs.map((_, i) => i - evs.length / 2),
              datasets: [
                {
                  label: "EV spectrum (dB)",
                  data: evs,
                  borderColor: "rgba(244,114,182,0.9)",
                  pointRadius: 0,
                  borderWidth: 1,
                },
              ],
            }}
            options={{
              responsive: true,
              plugins: {
                legend: { display: false },
                title: { display: true, text: "Error-vector spectrum" },
              },
              scales: {
                x: { title: { display: true, text: "bin (DC centred)" }, ticks: { maxTicksLimit: 9 } },
                y: { title: { display: true, text: "dB" } },
              },
            }}
          />
        </div>
      )}
      <p className="mt-2 text-xs text-muted">
        Error vectors split into magnitude vs phase. A peak away from centre in the
        error-vector spectrum points at a periodic impairment; a flat floor is white noise.
      </p>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-2">
      <dt className="text-muted">{label}</dt>
      <dd className="font-mono">{value}</dd>
    </div>
  );
}
