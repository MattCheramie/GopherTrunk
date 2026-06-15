import { Line } from "react-chartjs-2";
import { ensureChart } from "./chartSetup";

ensureChart();

// The four ideal π/4-DQPSK symbol rotations. A clean CQPSK control channel's
// per-symbol differential phase clusters tightly on these levels; carrier or
// timing error smears it between them.
const IDEAL = [Math.PI / 4, (3 * Math.PI) / 4, -Math.PI / 4, (-3 * Math.PI) / 4];

// RotationTracker plots the per-symbol differential phase (the rotation between
// consecutive constellation points) over symbol index — the matplotlib-style
// view of π/4-DQPSK rotation that the BSCH colour-code recovery hinges on. The
// data is IQTaps.diff_phase, computed natively in the analyzer via
// dsp/phase.Diff; it is populated only on the CQPSK path.
export function RotationTracker({ phase }: { phase: number[] }) {
  if (phase.length === 0) return null;

  // Decimate to keep the chart responsive on long captures.
  const max = 4000;
  const step = Math.max(1, Math.floor(phase.length / max));
  const labels: number[] = [];
  const series: number[] = [];
  for (let i = 0; i < phase.length; i += step) {
    labels.push(i);
    series.push(phase[i]);
  }

  // Flat reference lines at each ideal rotation level.
  const railDatasets = IDEAL.map((lvl) => ({
    label: `${(lvl / Math.PI).toFixed(2)}π`,
    data: labels.map(() => lvl),
    borderColor: "rgba(148,163,184,0.35)",
    borderWidth: 1,
    borderDash: [4, 4],
    pointRadius: 0,
    fill: false,
  }));

  return (
    <div className="card">
      <h3 className="mb-2 text-sm font-semibold">
        Rotation tracker (π/4-DQPSK differential phase)
      </h3>
      <Line
        data={{
          labels,
          datasets: [
            {
              label: "Δφ (rad)",
              data: series,
              borderColor: "rgba(56,189,248,0.9)",
              backgroundColor: "rgba(56,189,248,0.9)",
              showLine: false,
              pointRadius: 1.2,
            },
            ...railDatasets,
          ],
        }}
        options={{
          responsive: true,
          interaction: { mode: "index", intersect: false },
          plugins: {
            legend: { display: false },
            title: { display: false },
          },
          scales: {
            x: { title: { display: true, text: "symbol index" } },
            y: {
              title: { display: true, text: "Δφ (rad)" },
              min: -Math.PI,
              max: Math.PI,
            },
          },
        }}
      />
      <p className="mt-2 text-xs text-muted">
        Per-symbol differential phase. Tight clustering on the dashed ±π/4 / ±3π/4
        rails means a clean rotation; smearing means carrier or timing error.
      </p>
    </div>
  );
}
