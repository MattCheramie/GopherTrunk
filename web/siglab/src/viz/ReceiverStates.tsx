import { Line } from "react-chartjs-2";
import type { ReceiverState } from "../api/types";
import { ensureChart } from "./chartSetup";

ensureChart();

// ReceiverStates plots the P25 deep-path receiver loop telemetry over stream
// time (p25detail.go ReceiverState): AFC estimate, AGC level, and the M&M
// clock fractional interval.
export function ReceiverStates({ states }: { states: ReceiverState[] }) {
  if (states.length === 0) return null;
  const t = states.map((s) => s.time_sec);
  return (
    <div className="card">
      <h3 className="mb-2 text-sm font-semibold">Receiver state (per second)</h3>
      <Line
        data={{
          labels: t,
          datasets: [
            {
              label: "AFC (Hz)",
              data: states.map((s) => s.afc_hz_est),
              borderColor: "#38bdf8",
              yAxisID: "yHz",
              pointRadius: 0,
            },
            {
              label: "AGC level",
              data: states.map((s) => s.agc_level),
              borderColor: "#22c55e",
              yAxisID: "y",
              pointRadius: 0,
            },
            {
              label: "M&M μ",
              data: states.map((s) => s.mm_mu),
              borderColor: "#eab308",
              yAxisID: "y",
              pointRadius: 0,
            },
          ],
        }}
        options={{
          responsive: true,
          interaction: { mode: "index", intersect: false },
          scales: {
            x: { title: { display: true, text: "time (s)" } },
            y: { position: "left" },
            yHz: { position: "right", grid: { drawOnChartArea: false } },
          },
        }}
      />
    </div>
  );
}
