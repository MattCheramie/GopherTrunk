import {
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  Legend,
  LinearScale,
  Title,
  Tooltip,
} from "chart.js";
import { useEffect, useMemo, useRef, useState } from "react";
import { Bar } from "react-chartjs-2";
import { PageHeader } from "../components/ui/PageHeader";
import {
  fetchSpectrumDevices,
  defaultSymbolDevice,
  parentSerial,
  type SpectrumDevice,
} from "../api/spectrum";
import { openSymbolStream, type SymbolFrame } from "../api/symbols";
import { selectClientConfig, useShared } from "../store/shared";
import { prefs } from "../store/prefs";
import { useActiveCallsPoll } from "../hooks/useActiveCallsPoll";
import { TuningControls } from "../components/TuningControls";

ChartJS.register(
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend,
);

// Histogram panel — the recovered-symbol distribution plus a derived
// signal-quality readout, off the live demod. A clean P25 control channel
// is scrambled, so its symbols are evenly spread: each of the 4 (C4FM /
// CQPSK) levels should sit near 25%. A collapsed slicer shows an empty or
// single-dominant bin. For C4FM the soft samples also give a per-level
// mean/spread and an MER-like SNR estimate (a "level meter").

const WINDOW_SYMBOLS = 4000;
const CARDINALITY = 4;

const PROTOS: { value: string; label: string }[] = [
  { value: "p25-c4fm", label: "P25 C4FM" },
  { value: "p25-cqpsk", label: "P25 CQPSK" },
];

type ConnState = "connecting" | "open" | "closed";

interface Quality {
  pct: number[]; // % per dibit bin
  total: number;
  balanceDev: number; // max |bin% − ideal%|
  snrDb: number | null; // MER-like estimate (C4FM soft only)
  levels: { mean: number; std: number }[] | null;
}

function computeQuality(dibits: number[], soft: number[]): Quality {
  const cnt = new Array(CARDINALITY).fill(0);
  for (const d of dibits) if (d >= 0 && d < CARDINALITY) cnt[d]++;
  const total = dibits.length;
  const pct = cnt.map((c) => (total > 0 ? (100 * c) / total : 0));
  const ideal = 100 / CARDINALITY;
  const balanceDev = pct.reduce((m, p) => Math.max(m, Math.abs(p - ideal)), 0);

  // Per-level soft stats + MER-like SNR, when the soft track is aligned.
  let snrDb: number | null = null;
  let levels: { mean: number; std: number }[] | null = null;
  if (soft.length > 0 && soft.length === dibits.length) {
    const sum = new Array(CARDINALITY).fill(0);
    const sumsq = new Array(CARDINALITY).fill(0);
    for (let i = 0; i < dibits.length; i++) {
      const d = dibits[i];
      if (d < 0 || d >= CARDINALITY) continue;
      sum[d] += soft[i];
      sumsq[d] += soft[i] * soft[i];
    }
    levels = [];
    let grandSum = 0;
    let noise = 0;
    let signal = 0;
    for (let d = 0; d < CARDINALITY; d++) {
      const c = cnt[d];
      const mean = c > 0 ? sum[d] / c : 0;
      const varr = c > 0 ? Math.max(0, sumsq[d] / c - mean * mean) : 0;
      levels.push({ mean, std: Math.sqrt(varr) });
      grandSum += sum[d];
      noise += c * varr;
    }
    const mu = total > 0 ? grandSum / total : 0;
    for (let d = 0; d < CARDINALITY; d++) {
      signal += cnt[d] * (levels[d].mean - mu) * (levels[d].mean - mu);
    }
    if (total > 0 && noise > 0) {
      snrDb = 10 * Math.log10(signal / noise);
    }
  }
  return { pct, total, balanceDev, snrDb, levels };
}

export function Histogram() {
  const cfg = useShared(selectClientConfig);
  const activeCalls = useShared((s) => s.activeCalls);
  // Keep the follow logic live: this panel reads activeCalls but, unlike
  // Active/Dashboard, didn't refresh it — so the view froze on a stale
  // call ("following call" forever, single freq) when opened directly.
  useActiveCallsPoll();
  const [devices, setDevices] = useState<SpectrumDevice[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [conn, setConn] = useState<ConnState>("closed");
  const [latest, setLatest] = useState<SymbolFrame | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [proto, setProto] = useState<string>(() => prefs.histogramProto());
  const [offsetKHz, setOffsetKHz] = useState<number>(() =>
    prefs.histogramOffsetKHz(),
  );
  const [hold, setHold] = useState<boolean>(() => prefs.histogramHold());

  const [win, setWin] = useState<{ soft: number[]; dibits: number[] }>({
    soft: [],
    dibits: [],
  });
  const softRef = useRef<number[]>([]);
  const dibitRef = useRef<number[]>([]);

  useEffect(() => {
    let cancel = false;
    (async () => {
      try {
        const list = await fetchSpectrumDevices(cfg);
        if (cancel) return;
        setDevices(list);
        setError(null);
        if (selected == null) {
          const def = defaultSymbolDevice(list);
          if (def) setSelected(def.serial);
        }
      } catch (e) {
        if (cancel) return;
        setError(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancel = true;
    };
  }, [cfg, selected]);

  const device = useMemo(
    () => devices.find((d) => d.serial === selected) ?? null,
    [devices, selected],
  );

  const followOffsetKHz = useMemo(() => {
    if (!device) return null;
    const mine = activeCalls.filter(
      (c) => parentSerial(c.device_serial) === selected && !c.ended_at,
    );
    if (mine.length === 0) return null;
    const newest = mine.reduce((a, b) => (b.started_at > a.started_at ? b : a));
    return (newest.grant.frequency_hz - device.center_hz) / 1000;
  }, [activeCalls, device, selected]);

  // Offset (kHz) that lands the view on this SDR's control channel — the
  // rest position when Hold is off and no call is active (#557). Absent
  // (null) for devices with no in-band CC, leaving the old centre default.
  const controlOffsetKHz = useMemo(() => {
    if (!device?.control_channel_hz) return null;
    return (device.control_channel_hz - device.center_hz) / 1000;
  }, [device]);

  // Active call wins; otherwise rest on the control channel (#557).
  const restOffsetKHz = followOffsetKHz ?? controlOffsetKHz;
  useEffect(() => {
    if (hold || restOffsetKHz == null) return;
    setOffsetKHz((cur) =>
      Math.abs(cur - restOffsetKHz) < 0.05 ? cur : restOffsetKHz,
    );
  }, [hold, restOffsetKHz]);

  const maxOffsetKHz = device ? device.sample_rate_hz / 2000 : 1500;
  const clampedOffsetKHz = Math.max(
    -maxOffsetKHz,
    Math.min(maxOffsetKHz, offsetKHz),
  );

  useEffect(() => {
    prefs.setHistogramOffsetKHz(offsetKHz);
  }, [offsetKHz]);
  useEffect(() => {
    prefs.setHistogramHold(hold);
  }, [hold]);
  useEffect(() => {
    prefs.setHistogramProto(proto);
  }, [proto]);

  useEffect(() => {
    if (!selected) return;
    softRef.current = [];
    dibitRef.current = [];
    setWin({ soft: [], dibits: [] });
    setLatest(null);

    const stream = openSymbolStream(cfg, {
      serial: selected,
      proto,
      offset: Math.round(clampedOffsetKHz * 1000),
      onFrame: (f) => {
        setLatest(f);
        const sb = softRef.current.concat(f.soft ?? []);
        const db = dibitRef.current.concat(f.dibits ?? []);
        const aligned = sb.length === db.length;
        softRef.current = aligned
          ? sb.slice(Math.max(0, sb.length - WINDOW_SYMBOLS))
          : [];
        dibitRef.current = db.slice(Math.max(0, db.length - WINDOW_SYMBOLS));
        setWin({ soft: softRef.current, dibits: dibitRef.current });
      },
      onStatus: setConn,
    });
    return () => stream.close();
  }, [cfg, selected, proto, clampedOffsetKHz]);

  const quality = useMemo(
    () => computeQuality(win.dibits, win.soft),
    [win],
  );

  const centerHz = device?.center_hz ?? latest?.center_hz ?? null;
  const viewHz = centerHz != null ? centerHz + clampedOffsetKHz * 1000 : null;
  const tuningLabel = useMemo(() => {
    if (viewHz == null) return "";
    const off =
      clampedOffsetKHz === 0
        ? "centre"
        : `${clampedOffsetKHz >= 0 ? "+" : ""}${clampedOffsetKHz.toFixed(3).replace(/\.?0+$/, "")} kHz`;
    const head = `${(viewHz / 1e6).toFixed(4)} MHz (${off})`;
    if (quality.total === 0) return `${head} · waiting for symbols…`;
    return `${head} · ${quality.total} symbols`;
  }, [quality.total, viewHz, clampedOffsetKHz]);

  const chartData = useMemo(
    () => ({
      labels: quality.pct.map((_, i) => `sym ${i}`),
      datasets: [
        {
          label: "% of symbols",
          data: quality.pct,
          backgroundColor: "rgba(56,189,248,0.6)",
        },
      ],
    }),
    [quality.pct],
  );

  const chartOptions = useMemo(
    () => ({
      responsive: true,
      maintainAspectRatio: false,
      animation: false as const,
      plugins: { legend: { display: false } },
      scales: {
        x: { grid: { display: false }, ticks: { color: "rgba(148,163,184,0.8)" } },
        y: {
          beginAtZero: true,
          title: { display: true, text: "%" },
          grid: { color: "rgba(148,163,184,0.15)" },
          ticks: { color: "rgba(148,163,184,0.8)" },
        },
      },
    }),
    [],
  );

  return (
    <div className="space-y-3">
      <PageHeader
        title="Symbol histogram"
        actions={
        <div className="flex items-center gap-2 text-xs">
          <span className="text-muted">SDR:</span>
          <select
            className="bg-surface border border-border rounded px-2 py-1"
            value={selected ?? ""}
            onChange={(e) => setSelected(e.target.value || null)}
            disabled={devices.length === 0}
          >
            {devices.length === 0 && <option value="">No SDRs available</option>}
            {devices.map((d) => (
              <option key={d.serial} value={d.serial}>
                {d.serial} · {d.role}
              </option>
            ))}
          </select>
          <ConnPill state={conn} />
        </div>
        }
      />

      {error && (
        <div className="rounded border border-red-700/40 bg-red-900/20 text-red-200 text-xs px-3 py-2">
          {error}
        </div>
      )}

      <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-xs">
        <label className="flex items-center gap-2">
          <span className="text-muted">Mode</span>
          <select
            className="bg-surface border border-border rounded px-2 py-1"
            value={proto}
            onChange={(e) => setProto(e.target.value)}
            aria-label="Demodulation mode"
          >
            {PROTOS.map((p) => (
              <option key={p.value} value={p.value}>
                {p.label}
              </option>
            ))}
          </select>
        </label>

        <TuningControls
          centerHz={centerHz}
          maxOffsetKHz={maxOffsetKHz}
          offsetKHz={clampedOffsetKHz}
          onOffsetChange={(khz) => {
            setHold(true);
            setOffsetKHz(khz);
          }}
          hold={hold}
          onHoldChange={(next) => {
            // Re-checking Hold parks the view on the control channel (a
            // stable, known reference) rather than freezing on a call
            // frequency that may already be stale.
            if (next && controlOffsetKHz != null) setOffsetKHz(controlOffsetKHz);
            setHold(next);
          }}
          following={
            followOffsetKHz != null
              ? "call"
              : controlOffsetKHz != null
                ? "control"
                : null
          }
          onCentre={() => {
            setHold(true);
            setOffsetKHz(0);
          }}
        />
      </div>

      <div className="font-mono text-xs text-muted">{tuningLabel || "—"}</div>

      <div
        className="rounded border border-border bg-black p-2"
        style={{ height: 240 }}
        role="img"
        aria-label="Symbol-level histogram; the balance and level meters below give the current values as text."
      >
        <Bar data={chartData} options={chartOptions} />
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-3 gap-2 text-xs">
        <Meter
          label="Balance"
          value={
            quality.total > 0 ? `±${quality.balanceDev.toFixed(1)}%` : "—"
          }
          sub="from 25% ideal"
        />
        <Meter
          label="SNR (MER)"
          value={quality.snrDb != null ? `${quality.snrDb.toFixed(1)} dB` : "—"}
          sub={quality.snrDb == null ? "C4FM only" : undefined}
        />
        <Meter label="Symbols" value={quality.total > 0 ? String(quality.total) : "—"} />
      </div>

      <div className="text-[11px] text-muted">
        Distribution of the recovered symbols. A scrambled P25 control
        channel spreads evenly, so each bin should sit near 25% — a low{" "}
        <strong>Balance</strong> deviation means a healthy slicer; a
        dominant or empty bin means a collapsed eye. <strong>SNR (MER)</strong>{" "}
        is an estimate from the C4FM soft levels (level separation vs
        within-level spread); higher is cleaner. Dial the <em>Offset</em>{" "}
        onto a locked channel.
      </div>
    </div>
  );
}

function Meter({
  label,
  value,
  sub,
}: {
  label: string;
  value: string;
  sub?: string;
}) {
  return (
    <div className="rounded border border-border bg-surface px-3 py-2">
      <div className="text-muted">{label}</div>
      <div className="font-mono text-sm">{value}</div>
      {sub && <div className="text-[10px] text-muted">{sub}</div>}
    </div>
  );
}

function ConnPill({ state }: { state: ConnState }) {
  if (state === "open") return <span className="pill-ok">live</span>;
  if (state === "connecting") return <span className="pill-warn">connecting</span>;
  return <span className="pill-err">offline</span>;
}
