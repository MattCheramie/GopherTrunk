import {
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Title,
  Tooltip,
} from "chart.js";
import { useEffect, useMemo, useRef, useState } from "react";
import { Line } from "react-chartjs-2";
import { api } from "../api/client";
import { selectClientConfig, useShared } from "../store/shared";

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
);

const CURATED = [
  "gophertrunk_calls_active",
  "gophertrunk_calls_total",
  "gophertrunk_grants_total",
  "gophertrunk_cc_locked",
  "gophertrunk_sse_clients",
  "gophertrunk_devices_attached",
  "gophertrunk_tone_alerts_total",
] as const;

const POLL_INTERVAL_MS = 5_000;
const HISTORY_POINTS = 60; // 60 samples × 5 s ≈ 5 minutes

interface Sample {
  t: number; // wall-clock ms
  values: Map<string, number>;
}

export function Metrics() {
  const cfg = useShared(selectClientConfig);
  const [latest, setLatest] = useState<Map<string, number>>(new Map());
  const [error, setError] = useState<string | null>(null);
  const historyRef = useRef<Sample[]>([]);
  const [historyTick, setHistoryTick] = useState(0);

  useEffect(() => {
    let cancel = false;
    const refresh = async () => {
      try {
        const text = await api.metricsText(cfg);
        if (cancel) return;
        const parsed = parsePrometheus(text);
        setLatest(parsed);
        setError(null);
        const hist = historyRef.current.slice();
        hist.push({ t: Date.now(), values: parsed });
        if (hist.length > HISTORY_POINTS) {
          hist.splice(0, hist.length - HISTORY_POINTS);
        }
        historyRef.current = hist;
        setHistoryTick((t) => t + 1);
      } catch (e: unknown) {
        if (cancel) return;
        setError(
          e instanceof Error ? e.message : "metrics fetch failed",
        );
      }
    };
    refresh();
    const timer = window.setInterval(refresh, POLL_INTERVAL_MS);
    return () => {
      cancel = true;
      window.clearInterval(timer);
    };
  }, [cfg]);

  const curatedRows = useMemo(
    () =>
      CURATED.map((name) => ({
        name,
        value: latest.get(name),
      })),
    [latest],
  );

  const extras = useMemo(() => {
    const seen = new Set<string>(CURATED);
    const out: { name: string; value: number }[] = [];
    for (const [k, v] of latest) {
      if (seen.has(k)) continue;
      if (k.startsWith("gophertrunk_") && !k.startsWith("gophertrunk_build_")) {
        out.push({ name: k, value: v });
      }
    }
    out.sort((a, b) => a.name.localeCompare(b.name));
    return out;
  }, [latest]);

  const chartData = useMemo(() => {
    const hist = historyRef.current;
    const labels = hist.map((s) => formatTime(s.t));
    return {
      labels,
      datasets: [
        {
          label: "calls_active",
          data: hist.map((s) => s.values.get("gophertrunk_calls_active") ?? 0),
          borderColor: "rgb(56, 189, 248)",
          backgroundColor: "rgba(56, 189, 248, 0.15)",
          fill: true,
          tension: 0.2,
          pointRadius: 0,
        },
        {
          label: "devices_attached",
          data: hist.map(
            (s) => s.values.get("gophertrunk_devices_attached") ?? 0,
          ),
          borderColor: "rgb(34, 197, 94)",
          backgroundColor: "rgba(34, 197, 94, 0.15)",
          fill: true,
          tension: 0.2,
          pointRadius: 0,
        },
        {
          label: "cc_locked",
          data: hist.map((s) => s.values.get("gophertrunk_cc_locked") ?? 0),
          borderColor: "rgb(234, 179, 8)",
          backgroundColor: "rgba(234, 179, 8, 0.15)",
          fill: true,
          tension: 0.2,
          pointRadius: 0,
        },
      ],
    };
    // historyTick forces this useMemo to recompute on each poll.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [historyTick]);

  const chartOptions = useMemo(
    () => ({
      responsive: true,
      maintainAspectRatio: false,
      animation: false as const,
      scales: {
        y: {
          beginAtZero: true,
          ticks: { color: "rgba(226, 232, 240, 0.7)" },
          grid: { color: "rgba(148, 163, 184, 0.1)" },
        },
        x: {
          ticks: { color: "rgba(226, 232, 240, 0.7)", maxTicksLimit: 6 },
          grid: { display: false },
        },
      },
      plugins: {
        legend: {
          position: "top" as const,
          labels: { color: "rgba(226, 232, 240, 0.85)" },
        },
        tooltip: { intersect: false, mode: "index" as const },
      },
    }),
    [],
  );

  return (
    <div className="space-y-4">
      <header className="flex items-center justify-between gap-3">
        <h2 className="text-xl font-semibold">Metrics</h2>
        <span className="text-xs text-muted">
          polled every {POLL_INTERVAL_MS / 1000}s · {historyRef.current.length}/
          {HISTORY_POINTS} samples
        </span>
      </header>

      {error && (
        <p className="text-sm text-err" role="alert">
          {error}
        </p>
      )}

      <section className="panel p-4">
        <h3 className="panel-title mb-3">Trend (last ~5 min)</h3>
        <div className="h-64">
          <Line data={chartData} options={chartOptions} />
        </div>
      </section>

      <section className="panel p-4">
        <h3 className="panel-title mb-3">Curated counters</h3>
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-2">
          {curatedRows.map((m) => (
            <MetricTile key={m.name} name={m.name} value={m.value} />
          ))}
        </div>
      </section>

      {extras.length > 0 && (
        <section className="panel p-4">
          <h3 className="panel-title mb-3">
            Other gophertrunk_* metrics ({extras.length})
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-4 gap-y-1 text-xs font-mono">
            {extras.map((m) => (
              <div key={m.name} className="flex justify-between gap-2">
                <span className="text-muted truncate">{m.name}</span>
                <span className="text-fg tabular-nums">{formatValue(m.value)}</span>
              </div>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}

function MetricTile({ name, value }: { name: string; value?: number }) {
  return (
    <div className="panel p-3">
      <p className="text-xs text-muted font-mono break-all">{name}</p>
      <p className="stat-value mt-1 text-2xl">
        {value === undefined ? (
          <span className="text-muted">—</span>
        ) : (
          formatValue(value)
        )}
      </p>
    </div>
  );
}

function formatValue(v: number): string {
  if (!Number.isFinite(v)) return "—";
  if (Number.isInteger(v)) return v.toLocaleString();
  return v.toFixed(2);
}

function formatTime(ms: number): string {
  const d = new Date(ms);
  const hh = d.getHours().toString().padStart(2, "0");
  const mm = d.getMinutes().toString().padStart(2, "0");
  const ss = d.getSeconds().toString().padStart(2, "0");
  return `${hh}:${mm}:${ss}`;
}

// parsePrometheus parses Prometheus exposition format and returns the
// **sum** of all label-disambiguated time-series for each metric name.
// For a `gophertrunk_calls_total{system="x"} 12` plus `…{system="y"} 7`
// it returns 19 against `gophertrunk_calls_total`. This matches the
// TUI's curated-metrics view; per-label breakdowns are out of scope
// for the operator dashboard.
export function parsePrometheus(text: string): Map<string, number> {
  const out = new Map<string, number>();
  for (const line of text.split("\n")) {
    if (!line || line.startsWith("#")) continue;
    // metric{labels} value [timestamp]
    const m = line.match(/^([a-zA-Z_:][a-zA-Z0-9_:]*)(\{[^}]*\})?\s+(\S+)/);
    if (!m) continue;
    const name = m[1];
    const valStr = m[3];
    const val = parseFloat(valStr);
    if (Number.isNaN(val)) continue;
    out.set(name, (out.get(name) ?? 0) + val);
  }
  return out;
}
