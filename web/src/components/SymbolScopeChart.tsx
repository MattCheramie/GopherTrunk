import { useEffect, useRef } from "react";

// SymbolScopeChart renders the demodulated symbol stream as an
// oscilloscope-style time series — GopherTrunk's take on OP25's "Symbol"
// plot. Two modes, picked automatically from the data:
//
//   - Soft waveform present (C4FM): scatter the pre-slicer soft samples
//     (x = symbol index, y = soft level). A 4-level signal reads as ~4
//     noisy horizontal bands; faint rails mark each decided level's mean.
//   - Soft absent (dibit-only, e.g. CQPSK): plot the sliced decisions as
//     `cardinality` discrete rows.
//
// The contract (props below) is shared verbatim with the offline SigLab
// viz so the two separate web builds render identically.

export interface SymbolScopeChartProps {
  soft: number[];
  dibits: number[];
  cardinality?: number;
  height?: number;
  className?: string;
}

// Drawing geometry — wide canvas; CSS scales to container width.
const CANVAS_W = 720;
const DEFAULT_H = 220;
const PAD = { top: 12, right: 12, bottom: 18, left: 28 };

// GopherTrunk sky-400 accent (matches the recolored Constellation and
// the Spectrum waterfall), with neutral slate grid/labels.
const TRACE_RGB = "56, 189, 248";
const GRID_RGB = "148, 163, 184";

export function SymbolScopeChart({
  soft,
  dibits,
  cardinality = 4,
  height = DEFAULT_H,
  className,
}: SymbolScopeChartProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  useEffect(() => {
    paint(canvasRef.current, soft, dibits, cardinality);
  }, [soft, dibits, cardinality]);

  return (
    <canvas
      ref={canvasRef}
      width={CANVAS_W}
      height={height}
      className={className ?? "block w-full"}
      style={{ height }}
      aria-label="Demodulated symbol scope"
    />
  );
}

// paint draws one frame of the scope. Exported-by-reference through the
// component; kept pure so the offline viz can reuse the identical body.
function paint(
  canvas: HTMLCanvasElement | null,
  soft: number[],
  dibits: number[],
  cardinality: number,
) {
  if (!canvas) return;
  const ctx = canvas.getContext("2d");
  if (!ctx) return;
  const w = canvas.width;
  const h = canvas.height;
  const plotW = w - PAD.left - PAD.right;
  const plotH = h - PAD.top - PAD.bottom;
  const x0 = PAD.left;
  const y0 = PAD.top;

  ctx.fillStyle = "rgb(0, 0, 0)";
  ctx.fillRect(0, 0, w, h);

  const n = dibits.length;
  const hasSoft = soft.length > 0 && soft.length === dibits.length;

  // Y mapper. Soft mode maps the soft range to the plot with padding;
  // dibit mode maps 0..cardinality-1 onto evenly-spaced rows.
  let toY: (i: number) => number;
  let rails: number[] = [];

  if (hasSoft) {
    let lo = Infinity;
    let hi = -Infinity;
    for (const v of soft) {
      if (v < lo) lo = v;
      if (v > hi) hi = v;
    }
    if (!(hi > lo)) {
      hi = lo + 1;
      lo -= 1;
    }
    const span = hi - lo;
    const pad = span * 0.12;
    const top = hi + pad;
    const bot = lo - pad;
    toY = (v) => y0 + plotH * (1 - (v - bot) / (top - bot));

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
    for (let d = 0; d < cardinality; d++) {
      if (cnt[d] > 0) rails.push(toY(sum[d] / cnt[d]));
    }
  } else {
    const rows = Math.max(1, cardinality);
    const rowY = (d: number) =>
      y0 + plotH * (1 - (d + 0.5) / rows);
    toY = () => 0; // unused in dibit mode
    for (let d = 0; d < rows; d++) rails.push(rowY(d));
  }

  // Grid: faint rails + a baseline frame.
  ctx.strokeStyle = `rgba(${GRID_RGB}, 0.18)`;
  ctx.lineWidth = 1;
  for (const ry of rails) {
    ctx.beginPath();
    ctx.moveTo(x0, ry);
    ctx.lineTo(x0 + plotW, ry);
    ctx.stroke();
  }
  ctx.strokeStyle = `rgba(${GRID_RGB}, 0.30)`;
  ctx.beginPath();
  ctx.moveTo(x0, y0 + plotH);
  ctx.lineTo(x0 + plotW, y0 + plotH);
  ctx.stroke();

  if (n === 0) return;

  // Plot points. Additive blend so dense bands bloom.
  ctx.globalCompositeOperation = "lighter";
  const stepX = n > 1 ? plotW / (n - 1) : 0;
  for (let i = 0; i < n; i++) {
    const x = x0 + i * stepX;
    let y: number;
    if (hasSoft) {
      y = toY(soft[i]);
    } else {
      const rows = Math.max(1, cardinality);
      y = y0 + plotH * (1 - (dibits[i] + 0.5) / rows);
    }
    ctx.fillStyle = `rgba(${TRACE_RGB}, 0.55)`;
    ctx.fillRect(x - 1, y - 1, 2, 2);
  }
  ctx.globalCompositeOperation = "source-over";
}
