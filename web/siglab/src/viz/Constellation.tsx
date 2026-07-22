import { useEffect, useRef, useState } from "react";
import * as d3 from "d3";
import type { IQPoint } from "../api/types";

// Constellation renders a 2D I/Q scatter using D3. Two modes:
//   - "raw": the decimated channelized IQ (decimated_iq). PSK/QPSK shows
//     clusters, C4FM shows four arcs, noise shows a diffuse disc — for
//     π/4-DQPSK the raw stream is a filled disc because it is sampled off the
//     symbol instants.
//   - "symbols": the recovered per-symbol decision points (symbol_iq), which
//     render as discrete decision clusters (8 for a healthy π/4-DQPSK control
//     channel). Only available on the CQPSK / π4-DQPSK path.
// The mode toggle appears only when symbol points are present; it defaults to
// "symbols" so the cleaner view shows by default when we have it.
export function Constellation({ points, symbols }: { points: IQPoint[]; symbols?: IQPoint[] }) {
  const ref = useRef<SVGSVGElement | null>(null);
  const hasSymbols = !!symbols && symbols.length > 0;
  const [mode, setMode] = useState<"raw" | "symbols">(hasSymbols ? "symbols" : "raw");

  const showSymbols = mode === "symbols" && hasSymbols;
  const active: IQPoint[] = showSymbols ? symbols! : points;

  useEffect(() => {
    const svg = d3.select(ref.current);
    svg.selectAll("*").remove();
    if (active.length === 0) return;

    const size = 320;
    const m = 24;
    // Subsample for render performance; the density still reads clearly.
    const max = 6000;
    const step = Math.max(1, Math.floor(active.length / max));
    const pts: IQPoint[] = [];
    for (let i = 0; i < active.length; i += step) pts.push(active[i]);

    const extent = d3.max(pts, (p) => Math.max(Math.abs(p.i), Math.abs(p.q))) ?? 1;
    const scale = d3.scaleLinear().domain([-extent, extent]).range([m, size - m]);

    svg.attr("width", size).attr("height", size).attr("viewBox", `0 0 ${size} ${size}`);

    // Axes through the origin.
    svg
      .append("line")
      .attr("x1", m).attr("x2", size - m).attr("y1", scale(0)).attr("y2", scale(0))
      .attr("stroke", "rgba(255,255,255,0.12)");
    svg
      .append("line")
      .attr("y1", m).attr("y2", size - m).attr("x1", scale(0)).attr("x2", scale(0))
      .attr("stroke", "rgba(255,255,255,0.12)");

    // Decision points are sparser and want to read as clusters, so draw them a
    // touch larger and more opaque than the dense raw cloud.
    const r = showSymbols ? 1.6 : 1.1;
    const fill = showSymbols ? "rgba(56,189,248,0.7)" : "rgba(56,189,248,0.45)";

    svg
      .append("g")
      .selectAll("circle")
      .data(pts)
      .join("circle")
      .attr("cx", (p) => scale(p.i))
      .attr("cy", (p) => scale(-p.q))
      .attr("r", r)
      .attr("fill", fill);
  }, [active, showSymbols]);

  return (
    <div className="card">
      <div className="mb-2 flex items-center justify-between gap-2">
        <h3 className="text-sm font-semibold">Constellation (I/Q)</h3>
        {hasSymbols && (
          <label className="flex items-center gap-1 text-xs text-muted">
            <input
              type="checkbox"
              checked={mode === "symbols"}
              onChange={(e) => setMode(e.target.checked ? "symbols" : "raw")}
            />
            Symbol decisions
          </label>
        )}
      </div>
      <svg ref={ref} className="mx-auto block" />
      <p className="mt-1 text-center text-xs text-muted">
        {active.length.toLocaleString()} {showSymbols ? "symbol decisions" : "decimated samples"}
      </p>
    </div>
  );
}
