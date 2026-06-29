import { Line } from "react-chartjs-2";
import type { ChartDataset } from "chart.js";
import type { WidebandResult, WidebandCarrier } from "../api/types";
import { ensureChart } from "./chartSetup";

ensureChart();

// roleColor maps a carrier's role to its marker color: control channels carry
// the system identity (amber), voice carriers carry grants (green), and
// unclassified carriers are muted (sky).
function roleColor(role: string): string {
  switch (role) {
    case "control":
      return "#f59e0b";
    case "voice":
      return "#22c55e";
    default:
      return "#38bdf8";
  }
}

// binDbAt returns the spectrum dB value nearest carrier frequency f, so a marker
// pins to the peak it represents. Bins are FFT-shifted: bins[0] is
// center - rate/2, stepping by rate/len.
function binDbAt(
  bins: number[],
  centerHz: number,
  rateHz: number,
  f: number,
): number {
  if (bins.length === 0) return 0;
  const base = centerHz - rateHz / 2;
  const step = rateHz / bins.length;
  let idx = Math.round((f - base) / step);
  if (idx < 0) idx = 0;
  if (idx >= bins.length) idx = bins.length - 1;
  return bins[idx];
}

// WidebandSurvey renders the matplotlib-style view of a wide-band survey: the
// Welch-averaged band spectrum with each detected carrier marked (colored by
// control/voice role), the trunked-system rollups, and the per-carrier table.
export function WidebandSurvey({ result }: { result: WidebandResult }) {
  const spec = result.spectrum;
  const datasets: ChartDataset<"line">[] = [];

  if (spec && spec.bins.length > 0) {
    const base = spec.center_hz - spec.sample_rate_hz / 2;
    const step = spec.sample_rate_hz / spec.bins.length;
    datasets.push({
      label: "PSD (dBFS)",
      data: spec.bins.map((db, i) => ({ x: (base + i * step) / 1e6, y: db })),
      borderColor: "#64748b",
      pointRadius: 0,
      borderWidth: 1,
    });
    if (result.carriers.length > 0) {
      datasets.push({
        type: "scatter",
        label: "carriers",
        data: result.carriers.map((c) => ({
          x: c.freq_hz / 1e6,
          y: binDbAt(spec.bins, spec.center_hz, spec.sample_rate_hz, c.freq_hz),
        })),
        pointBackgroundColor: result.carriers.map((c) => roleColor(c.role)),
        pointBorderColor: result.carriers.map((c) => roleColor(c.role)),
        pointRadius: 5,
        pointStyle: "triangle",
      } as unknown as ChartDataset<"line">);
    }
  }

  return (
    <div className="space-y-4">
      <div className="card">
        <h3 className="mb-2 text-sm font-semibold">
          Wide-band survey · {result.detected_count} carrier peaks ·{" "}
          {result.carriers.length} identified
        </h3>
        {spec && spec.bins.length > 0 ? (
          <Line
            data={{ datasets }}
            options={{
              responsive: true,
              plugins: {
                legend: { display: false },
                tooltip: {
                  callbacks: {
                    label: (ctx) => {
                      if (ctx.dataset.label !== "carriers") {
                        return `${(ctx.parsed.y ?? 0).toFixed(1)} dBFS`;
                      }
                      const c = result.carriers[ctx.dataIndex];
                      const id =
                        c.system_id != null && c.system_id !== 0
                          ? ` · sys ${c.system_id}`
                          : "";
                      return `${(c.freq_hz / 1e6).toFixed(4)} MHz · ${
                        c.protocol || "?"
                      } · ${c.role || "unknown"}${id}`;
                    },
                  },
                },
              },
              scales: {
                x: {
                  type: "linear",
                  title: { display: true, text: "MHz" },
                  ticks: { maxTicksLimit: 12 },
                },
                y: { title: { display: true, text: "dBFS" } },
              },
            }}
          />
        ) : (
          <p className="text-xs text-muted">No spectrum returned.</p>
        )}
      </div>

      {result.systems.length > 0 && (
        <div className="card">
          <h3 className="mb-2 text-sm font-semibold">Systems</h3>
          <table className="w-full text-left text-xs">
            <thead className="text-muted">
              <tr>
                <th className="pr-3">Verdict</th>
                <th className="pr-3">Protocol</th>
                <th className="pr-3">Control</th>
                <th className="pr-3">Voice</th>
                <th className="pr-3">System</th>
                <th className="pr-3">CC</th>
                <th>Span (MHz)</th>
              </tr>
            </thead>
            <tbody>
              {result.systems.map((sys, i) => (
                <tr key={i} className="border-t border-line">
                  <td className="py-1 pr-3">
                    {sys.tier3 && <span className="text-accent">★ </span>}
                    {sys.verdict}
                  </td>
                  <td className="pr-3 uppercase">{sys.protocol}</td>
                  <td className="pr-3">{sys.control_count}</td>
                  <td className="pr-3">{sys.voice_count}</td>
                  <td className="pr-3">{sys.system_id || "—"}</td>
                  <td className="pr-3">{sys.color_code || "—"}</td>
                  <td>
                    {(sys.lo_hz / 1e6).toFixed(4)}–{(sys.hi_hz / 1e6).toFixed(4)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {result.carriers.length > 0 && (
        <div className="card">
          <h3 className="mb-2 text-sm font-semibold">Carriers</h3>
          <table className="w-full text-left text-xs">
            <thead className="text-muted">
              <tr>
                <th className="pr-3">Freq (MHz)</th>
                <th className="pr-3">Role</th>
                <th className="pr-3">Protocol</th>
                <th className="pr-3">SNR (dB)</th>
                <th className="pr-3">Locked</th>
                <th className="pr-3">Grants</th>
                <th>Best guess</th>
              </tr>
            </thead>
            <tbody>
              {result.carriers.map((c: WidebandCarrier, i) => (
                <tr key={i} className="border-t border-line">
                  <td className="py-1 pr-3">{(c.freq_hz / 1e6).toFixed(4)}</td>
                  <td className="pr-3" style={{ color: roleColor(c.role) }}>
                    {c.role || "unknown"}
                  </td>
                  <td className="pr-3">{c.protocol || "—"}</td>
                  <td className="pr-3">{c.snr_db.toFixed(1)}</td>
                  <td className="pr-3">{c.locked ? "✓" : ""}</td>
                  <td className="pr-3">{c.grants?.length ?? 0}</td>
                  <td className="text-muted">
                    {!c.protocol && c.reference_matches && c.reference_matches.length > 0
                      ? `${c.reference_matches[0].entry.display_name} (${(c.reference_matches[0].score * 100).toFixed(0)}%)`
                      : "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
