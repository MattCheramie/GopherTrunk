import { useEffect, useRef } from "react";
import * as d3 from "d3";

// SymbolScope renders the recovered-symbol stream as an oscilloscope-style
// time series — the offline twin of the live web console's Symbol scope,
// and GopherTrunk's take on OP25's "Symbol" plot. Props contract is shared
// verbatim with the live component (web/src/components/SymbolScopeChart):
//
//   - soft present (C4FM): scatter the pre-slicer soft samples; a 4-level
//     channel reads as ~4 bands, with rails at each decided level's mean.
//   - soft absent (dibit-only): plot the decisions as `cardinality` rows.
export function SymbolScope({
  soft,
  dibits,
  cardinality = 4,
}: {
  soft: number[];
  dibits: number[];
  cardinality?: number;
}) {
  const ref = useRef<SVGSVGElement | null>(null);

  useEffect(() => {
    const svg = d3.select(ref.current);
    svg.selectAll("*").remove();
    const n = dibits.length;
    if (n === 0) return;

    const w = 720;
    const h = 240;
    const m = 24;
    const hasSoft = soft.length === n && n > 0;

    svg.attr("width", w).attr("height", h).attr("viewBox", `0 0 ${w} ${h}`);

    const x = d3.scaleLinear().domain([0, Math.max(1, n - 1)]).range([m, w - m]);

    let y: d3.ScaleLinear<number, number>;
    let rails: number[];
    if (hasSoft) {
      const ext = d3.max(soft, (s) => Math.abs(s)) ?? 1;
      y = d3.scaleLinear().domain([-ext * 1.1, ext * 1.1]).range([h - m, m]);
      // Rails at each decided level's mean soft value.
      const sum = new Array(cardinality).fill(0);
      const cnt = new Array(cardinality).fill(0);
      for (let i = 0; i < n; i++) {
        const d = dibits[i];
        if (d >= 0 && d < cardinality) {
          sum[d] += soft[i];
          cnt[d]++;
        }
      }
      rails = [];
      for (let d = 0; d < cardinality; d++) {
        if (cnt[d] > 0) rails.push(sum[d] / cnt[d]);
      }
    } else {
      y = d3.scaleLinear().domain([-0.5, cardinality - 0.5]).range([h - m, m]);
      rails = d3.range(cardinality);
    }

    rails.forEach((lvl) => {
      svg
        .append("line")
        .attr("x1", m)
        .attr("x2", w - m)
        .attr("y1", y(lvl))
        .attr("y2", y(lvl))
        .attr("stroke", "rgba(148,163,184,0.18)");
    });

    // Scatter, decimated to a sane point count.
    const max = 2500;
    const step = Math.max(1, Math.floor(n / max));
    const g = svg.append("g");
    for (let i = 0; i < n; i += step) {
      const yv = hasSoft ? soft[i] : dibits[i];
      g.append("rect")
        .attr("x", x(i) - 1)
        .attr("y", y(yv) - 1)
        .attr("width", 2)
        .attr("height", 2)
        .attr("fill", "rgba(56,189,248,0.55)");
    }
  }, [soft, dibits, cardinality]);

  return (
    <div className="card">
      <h3 className="mb-2 text-sm font-semibold">Symbol scope</h3>
      {dibits.length === 0 ? (
        <p className="text-xs text-muted">
          No symbols captured. Run with “collect IQ diag” + “capture IQ”.
        </p>
      ) : (
        <svg ref={ref} className="mx-auto block w-full" />
      )}
      <p className="mt-2 text-xs text-muted">
        {soft.length === dibits.length && dibits.length > 0
          ? "Pre-slicer soft waveform with rails at each decided level."
          : "Sliced dibit decisions (no soft track on this path)."}
      </p>
    </div>
  );
}
