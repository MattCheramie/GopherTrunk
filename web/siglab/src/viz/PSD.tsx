import { useEffect, useState } from "react";
import { Line } from "react-chartjs-2";
import { useStore } from "../store/shared";
import { api } from "../api/client";
import type { OccupancyResult, PSDResult } from "../api/types";
import { ensureChart } from "./chartSetup";

ensureChart();

// PSD renders the Welch power-spectral-density of a job's captured IQ. The FFT
// is computed server-side (GET /jobs/{id}/psd via internal/dsp/spectrum), so the
// browser needs no FFT library. Beneath the chart it shows the lab-grade
// spectral-occupancy metrics computed off the same spectrum.
export function PSD({ jobId }: { jobId: string }) {
  const config = useStore((s) => s.config);
  const [psd, setPsd] = useState<PSDResult | null>(null);
  const [occ, setOcc] = useState<OccupancyResult | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    if (!jobId) return;
    setBusy(true);
    Promise.all([
      api.jobPSD(config, jobId, 1024).catch(() => null),
      api.jobOccupancy(config, jobId, 1024).catch(() => null),
    ])
      .then(([p, o]) => {
        if (cancelled) return;
        setPsd(p);
        setOcc(o);
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
      {occ && (
        <dl className="mt-2 grid grid-cols-2 gap-x-4 gap-y-1 text-xs sm:grid-cols-3">
          <Metric label="Occupied BW" value={`${(occ.occupancy.occupied_bandwidth_hz / 1000).toFixed(2)} kHz`} />
          <Metric label="Channel power" value={`${occ.occupancy.channel_power_dbfs.toFixed(1)} dBFS`} />
          <Metric label="Flatness" value={occ.occupancy.spectral_flatness.toFixed(3)} />
          <Metric label="ACPR upper" value={`${occ.occupancy.acpr_upper_db.toFixed(1)} dB`} />
          <Metric label="ACPR lower" value={`${occ.occupancy.acpr_lower_db.toFixed(1)} dB`} />
        </dl>
      )}
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
