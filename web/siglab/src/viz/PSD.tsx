import { useEffect, useState } from "react";
import { Line } from "react-chartjs-2";
import { useStore } from "../store/shared";
import { api } from "../api/client";
import type { PSDResult } from "../api/types";
import { ensureChart } from "./chartSetup";

ensureChart();

// PSD renders the Welch power-spectral-density of a job's captured IQ. The FFT
// is computed server-side (GET /jobs/{id}/psd via internal/dsp/spectrum), so the
// browser needs no FFT library.
export function PSD({ jobId }: { jobId: string }) {
  const config = useStore((s) => s.config);
  const [psd, setPsd] = useState<PSDResult | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    if (!jobId) return;
    setBusy(true);
    api
      .jobPSD(config, jobId, 1024)
      .then((r) => {
        if (!cancelled) setPsd(r);
      })
      .catch(() => {
        if (!cancelled) setPsd(null);
      })
      .finally(() => {
        if (!cancelled) setBusy(false);
      });
    return () => {
      cancelled = true;
    };
  }, [jobId, config]);

  // FFT-shifted bins: bin 0 is -rate/2, stepping by rate/len.
  const n = psd?.bins.length ?? 0;
  const base = psd ? -psd.sample_rate_hz / 2 : 0;
  const step = psd && n > 0 ? psd.sample_rate_hz / n : 0;

  return (
    <div className="card">
      <h3 className="mb-2 text-sm font-semibold">Power spectral density</h3>
      {busy && <p className="text-xs text-muted">computing FFT…</p>}
      {psd && n > 0 ? (
        <Line
          data={{
            labels: psd.bins.map((_, i) => ((base + i * step) / 1000).toFixed(1)),
            datasets: [
              {
                label: "PSD (dBFS)",
                data: psd.bins,
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
              y: { title: { display: true, text: "dBFS" } },
            },
          }}
        />
      ) : (
        !busy && <p className="text-xs text-muted">No IQ captured.</p>
      )}
    </div>
  );
}
