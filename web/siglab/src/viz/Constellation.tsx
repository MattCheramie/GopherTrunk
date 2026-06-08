import { useEffect, useRef } from "react";
import * as d3 from "d3";
import type { IQPoint } from "../api/types";

// Constellation renders the decimated channelized IQ as a 2D scatter using
// D3. PSK/QPSK shows clusters; C4FM shows four arcs; noise shows a diffuse
// disc; DC bias shows an offset blob.
export function Constellation({ points }: { points: IQPoint[] }) {
  const ref = useRef<SVGSVGElement | null>(null);

  useEffect(() => {
    const svg = d3.select(ref.current);
    svg.selectAll("*").remove();
    if (points.length === 0) return;

    const size = 320;
    const m = 24;
    // Subsample for render performance; the density still reads clearly.
    const max = 6000;
    const step = Math.max(1, Math.floor(points.length / max));
    const pts: IQPoint[] = [];
    for (let i = 0; i < points.length; i += step) pts.push(points[i]);

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

    svg
      .append("g")
      .selectAll("circle")
      .data(pts)
      .join("circle")
      .attr("cx", (p) => scale(p.i))
      .attr("cy", (p) => scale(-p.q))
      .attr("r", 1.1)
      .attr("fill", "rgba(56,189,248,0.45)");
  }, [points]);

  return (
    <div className="card">
      <h3 className="mb-2 text-sm font-semibold">Constellation (I/Q)</h3>
      <svg ref={ref} className="mx-auto block" />
      <p className="mt-1 text-center text-xs text-muted">
        {points.length.toLocaleString()} decimated samples
      </p>
    </div>
  );
}
