import { useEffect, useRef, useState } from "react";
import { useStore } from "../store/shared";
import { api } from "../api/client";

// Spectrogram renders a short-time-FFT waterfall as a Plotly heatmap. The STFT
// is computed server-side (GET /jobs/{id}/spectrogram via internal/dsp/spectrum)
// so the browser needs no FFT library; only Plotly is lazy-loaded for rendering.
export function Spectrogram({ jobId }: { jobId: string }) {
  const config = useStore((s) => s.config);
  const ref = useRef<HTMLDivElement | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    if (!jobId || !ref.current) return;
    setBusy(true);
    (async () => {
      const sg = await api.jobSpectrogram(config, jobId, 256, 128);
      if (cancelled || !ref.current || sg.frames === 0) return;
      const Plotly = (await import("plotly.js-dist-min")).default;
      const n = sg.fft_size;
      const freqs = Array.from({ length: n }, (_, i) => ((i - n / 2) * sg.sample_rate_hz) / n / 1000);
      // Transpose so frequency is the y-axis and time the x-axis.
      const z: number[][] = [];
      for (let f = 0; f < n; f++) z.push(sg.z.map((row) => row[f]));
      await Plotly.react(
        ref.current,
        [{ z, y: freqs, type: "heatmap", colorscale: "Viridis", showscale: true }],
        {
          margin: { l: 50, r: 10, t: 10, b: 35 },
          paper_bgcolor: "rgba(0,0,0,0)",
          plot_bgcolor: "rgba(0,0,0,0)",
          font: { color: "#94a3b8", size: 10 },
          xaxis: { title: { text: "frame" } },
          yaxis: { title: { text: "kHz" } },
          height: 320,
        },
        { displayModeBar: false, responsive: true } as Record<string, unknown>,
      );
    })()
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setBusy(false);
      });
    return () => {
      cancelled = true;
    };
  }, [jobId, config]);

  return (
    <div className="card">
      <h3 className="mb-2 text-sm font-semibold">Spectrogram / waterfall</h3>
      {busy && <p className="text-xs text-muted">computing STFT…</p>}
      <div ref={ref} />
    </div>
  );
}
