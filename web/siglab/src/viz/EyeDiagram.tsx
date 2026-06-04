import { useEffect, useRef } from "react";
import * as d3 from "d3";
import type { SoftEye } from "../api/types";

// EyeDiagram folds the pre-slicer soft-sample stream into an eye-like overlay
// (consecutive 2-symbol segments superimposed) and summarizes the engine's
// SoftEye rail statistics (p25eye.go). The four C4FM rails (-3,-1,+1,+3)
// should separate cleanly; a closed eye means symbol-timing / SNR trouble.
export function EyeDiagram({
  soft,
  eye,
}: {
  soft: number[];
  eye?: SoftEye;
}) {
  const ref = useRef<SVGSVGElement | null>(null);

  useEffect(() => {
    const svg = d3.select(ref.current);
    svg.selectAll("*").remove();
    if (soft.length < 2) return;

    const w = 360;
    const h = 240;
    const m = 20;
    const ext = d3.max(soft, (s) => Math.abs(s)) ?? 1;
    const x = d3.scaleLinear().domain([0, 1]).range([m, w - m]);
    const y = d3.scaleLinear().domain([-ext, ext]).range([h - m, m]);

    svg.attr("width", w).attr("height", h).attr("viewBox", `0 0 ${w} ${h}`);

    // Decision rails for C4FM at ±1, ±3 (normalized to the soft scale).
    [-3, -1, 1, 3].forEach((lvl) => {
      const yy = y((lvl / 3) * ext);
      svg
        .append("line")
        .attr("x1", m).attr("x2", w - m).attr("y1", yy).attr("y2", yy)
        .attr("stroke", "rgba(255,255,255,0.08)");
    });

    const line = d3
      .line<[number, number]>()
      .x((d) => x(d[0]))
      .y((d) => y(d[1]));

    const max = 2500;
    const step = Math.max(1, Math.floor(soft.length / max));
    const g = svg.append("g");
    for (let i = 0; i + 1 < soft.length; i += step) {
      g.append("path")
        .attr("d", line([
          [0, soft[i]],
          [1, soft[i + 1]],
        ]))
        .attr("fill", "none")
        .attr("stroke", "rgba(56,189,248,0.18)")
        .attr("stroke-width", 1);
    }
  }, [soft]);

  return (
    <div className="card">
      <h3 className="mb-2 text-sm font-semibold">Eye diagram (soft samples)</h3>
      {soft.length < 2 ? (
        <p className="text-xs text-muted">
          No soft samples (P25 C4FM deep path only). Enable “collect IQ diag”.
        </p>
      ) : (
        <svg ref={ref} className="mx-auto block" />
      )}
      {eye && (
        <div className="mt-2 text-xs text-muted">
          <div>
            DC {eye.signed_mean_dc.toFixed(3)} · σ {eye.std_dev.toFixed(3)} · max|x|{" "}
            {eye.max_abs.toFixed(3)}
          </div>
          {eye.verdict && <div className="mt-1 text-warn">{eye.verdict}</div>}
        </div>
      )}
    </div>
  );
}
