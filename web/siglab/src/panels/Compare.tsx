import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Line } from "react-chartjs-2";
import { useStore } from "../store/shared";
import { ensureChart } from "../viz/chartSetup";
import { api } from "../api/client";
import type { Analysis } from "../store/shared";

ensureChart();

const COLORS = ["#38bdf8", "#22c55e", "#eab308", "#f472b6", "#a78bfa", "#fb7185"];

// Compare overlays multiple analyzed captures: their power spectra on one axis
// and a side-by-side metrics table. Pick a synthesized capture as the
// idealized reference (Synthesize tab) and compare an uploaded capture against
// it. Captures must be run first (Captures tab).
export function Compare() {
  const compareIds = useStore((s) => s.compareIds);
  const captures = useStore((s) => s.captures);
  const analyses = useStore((s) => s.analyses);

  const items = compareIds
    .map((id) => ({
      capture: captures.find((c) => c.id === id),
      analysis: analyses[id],
    }))
    .filter((x) => x.capture);

  if (items.length === 0) {
    return (
      <div className="card text-sm text-muted">
        Tick the <span className="text-fg">cmp</span> box on two or more captures in the{" "}
        <Link to="/captures" className="text-accent">
          Captures
        </Link>{" "}
        tab, run them, then return here to overlay and diff them.
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <SpectraOverlay items={items} />
      <MetricsTable items={items} />
    </div>
  );
}

interface Item {
  capture?: { id: string; name: string };
  analysis?: Analysis;
}

function SpectraOverlay({ items }: { items: Item[] }) {
  const config = useStore((s) => s.config);
  const [series, setSeries] = useState<
    { name: string; freqHz: number[]; psdDb: number[]; color: string }[]
  >([]);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setBusy(true);
    (async () => {
      const out: { name: string; freqHz: number[]; psdDb: number[]; color: string }[] = [];
      let i = 0;
      for (const it of items) {
        // The server computes the PSD from the job's captured IQ (no in-browser FFT).
        if (it.analysis?.jobId && it.analysis?.iqTaps) {
          const psd = await api.jobPSD(config, it.analysis.jobId, 1024);
          const base = -psd.sample_rate_hz / 2;
          const stepHz = psd.bins.length > 0 ? psd.sample_rate_hz / psd.bins.length : 0;
          out.push({
            name: it.capture?.name ?? "?",
            freqHz: psd.bins.map((_, b) => base + b * stepHz),
            psdDb: psd.bins,
            color: COLORS[i % COLORS.length],
          });
        }
        i++;
      }
      if (!cancelled) setSeries(out);
    })().finally(() => {
      if (!cancelled) setBusy(false);
    });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [items.map((i) => i.capture?.id + (i.analysis?.jobId ?? "")).join(",")]);

  const labels = series[0]?.freqHz.map((f) => (f / 1000).toFixed(1)) ?? [];

  return (
    <div className="card">
      <h3 className="mb-2 text-sm font-semibold">Power-spectrum overlay</h3>
      {busy && <p className="text-xs text-muted">computing spectra…</p>}
      {series.length === 0 ? (
        <p className="text-xs text-muted">
          No captured IQ to overlay — run the selected captures with “capture IQ” enabled.
        </p>
      ) : (
        <Line
          data={{
            labels,
            datasets: series.map((s) => ({
              label: s.name,
              data: s.psdDb,
              borderColor: s.color,
              pointRadius: 0,
              borderWidth: 1,
            })),
          }}
          options={{
            responsive: true,
            interaction: { mode: "index", intersect: false },
            scales: {
              x: { title: { display: true, text: "kHz" }, ticks: { maxTicksLimit: 12 } },
              y: { title: { display: true, text: "dB" } },
            },
          }}
        />
      )}
    </div>
  );
}

function MetricsTable({ items }: { items: Item[] }) {
  const rows: { label: string; get: (a?: Analysis) => string }[] = [
    { label: "Protocol", get: (a) => a?.result?.protocol ?? "—" },
    { label: "Locked", get: (a) => (a?.result?.locked ? "yes" : "no") },
    {
      label: "Lock latency (s)",
      get: (a) => (a?.result ? a.result.lock_latency_sec.toFixed(3) : "—"),
    },
    {
      label: "Effective baud",
      get: (a) => (a?.result ? a.result.effective_baud.toFixed(0) : "—"),
    },
    {
      label: "Baud dev (%)",
      get: (a) => (a?.result ? a.result.baud_deviation_pct.toFixed(2) : "—"),
    },
    {
      label: "Decode err /ksym",
      get: (a) =>
        a?.result?.signal ? a.result.signal.decode_error_rate_per_ksym.toFixed(2) : "—",
    },
    {
      label: "Demod EVM (%)",
      get: (a) =>
        a?.result?.signal?.demod ? a.result.signal.demod.evm_pct.toFixed(1) : "—",
    },
    {
      label: "Demod SNR (dB)",
      get: (a) =>
        a?.result?.signal?.demod ? a.result.signal.demod.snr_estimate_db.toFixed(1) : "—",
    },
    {
      label: "IQ gain imb (dB)",
      get: (a) =>
        a?.result?.signal?.iq_observed
          ? a.result.signal.iq_gain_imbalance_db.toFixed(2)
          : "—",
    },
    { label: "Grants", get: (a) => String(a?.result?.grants.length ?? "—") },
  ];

  return (
    <div className="card overflow-x-auto">
      <h3 className="mb-2 text-sm font-semibold">Metrics comparison</h3>
      <table className="w-full text-left text-xs">
        <thead className="text-muted">
          <tr>
            <th className="py-1 pr-4">Metric</th>
            {items.map((it) => (
              <th key={it.capture?.id} className="px-3">
                {it.capture?.name}
                {it.analysis?.state !== "done" && (
                  <span className="ml-1 text-warn">({it.analysis?.state ?? "not run"})</span>
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.label} className="border-t border-line">
              <td className="py-1 pr-4 text-muted">{row.label}</td>
              {items.map((it) => (
                <td key={it.capture?.id} className="px-3">
                  {row.get(it.analysis)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
