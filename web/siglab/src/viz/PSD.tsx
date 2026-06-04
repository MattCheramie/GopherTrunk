import { useEffect, useState } from "react";
import { Line } from "react-chartjs-2";
import type { IQPoint } from "../api/types";
import { ensureChart } from "./chartSetup";
import { welchPSD, type PSDResult } from "../dsp/transforms";

ensureChart();

// PSD renders a Welch power-spectral-density estimate of the decimated IQ.
// The FFT is computed with TensorFlow.js (lazy-loaded by ./dsp/transforms).
export function PSD({ points, rateHz }: { points: IQPoint[]; rateHz: number }) {
  const [psd, setPsd] = useState<PSDResult | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    if (points.length < 16) return;
    setBusy(true);
    welchPSD(points, rateHz, 1024, 0.5)
      .then((r) => {
        if (!cancelled) setPsd(r);
      })
      .finally(() => {
        if (!cancelled) setBusy(false);
      });
    return () => {
      cancelled = true;
    };
  }, [points, rateHz]);

  return (
    <div className="card">
      <h3 className="mb-2 text-sm font-semibold">Power spectral density</h3>
      {busy && <p className="text-xs text-muted">computing FFT…</p>}
      {psd && psd.freqHz.length > 0 ? (
        <Line
          data={{
            labels: psd.freqHz.map((f) => (f / 1000).toFixed(1)),
            datasets: [
              {
                label: "PSD (dB)",
                data: psd.psdDb,
                borderColor: "#38bdf8",
                pointRadius: 0,
                borderWidth: 1,
              },
            ],
          }}
          options={{
            responsive: true,
            plugins: { legend: { display: false } },
            scales: {
              x: { title: { display: true, text: "kHz" }, ticks: { maxTicksLimit: 12 } },
              y: { title: { display: true, text: "dB" } },
            },
          }}
        />
      ) : (
        !busy && <p className="text-xs text-muted">No IQ captured.</p>
      )}
    </div>
  );
}
