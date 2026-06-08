import { useEffect, useRef } from "react";

// EyeDiagramChart folds the oversampled, AGC-scaled matched-filter output
// over the symbol period and overlays the resulting 2-symbol windows —
// GopherTrunk's take on OP25's "datascope". A healthy P25 C4FM channel
// shows four open horizontal bands (the ±1/±3 rails) with clear vertical
// gaps between them at the symbol-decision instant; a closed eye (bands
// merging, no gap) means symbol-timing or SNR trouble.
//
// Unlike the Symbol scope's 1-sample-per-symbol soft trace, this needs
// the oversampled stream (eye_sps samples/symbol) so the intra-symbol
// transitions — the eye's "lids" — are actually drawn.

export interface EyeDiagramChartProps {
  // Oversampled soft samples (sps per symbol), AGC-scaled to soft units.
  eye: number[];
  // Samples per symbol to fold on.
  sps: number;
  height?: number;
  className?: string;
}

const CANVAS_W = 720;
const DEFAULT_H = 260;
const PAD = { top: 14, right: 12, bottom: 18, left: 30 };

// Sky-400 accent / neutral slate grid, matching the other scopes.
const TRACE_RGB = "56, 189, 248";
const GRID_RGB = "148, 163, 184";

// Cap overlaid traces so a large buffer can't stall the paint.
const MAX_TRACES = 500;

export function EyeDiagramChart({
  eye,
  sps,
  height = DEFAULT_H,
  className,
}: EyeDiagramChartProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  useEffect(() => {
    paint(canvasRef.current, eye, sps);
  }, [eye, sps]);

  return (
    <canvas
      ref={canvasRef}
      width={CANVAS_W}
      height={height}
      className={className ?? "block w-full"}
      style={{ height }}
      aria-label="C4FM eye diagram"
    />
  );
}

function paint(canvas: HTMLCanvasElement | null, eye: number[], sps: number) {
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

  // Y extent from the data, with headroom; symmetric about zero so the
  // four rails sit at ±ext and ±ext/3.
  let ext = 0;
  for (const v of eye) {
    const a = Math.abs(v);
    if (a > ext) ext = a;
  }
  if (!(ext > 0)) ext = 1;
  const top = ext * 1.15;
  const toY = (v: number) => y0 + plotH * (1 - (v + top) / (2 * top));

  // Decision rails at the four C4FM levels (±1, ±3 of the eye scale).
  ctx.strokeStyle = `rgba(${GRID_RGB}, 0.18)`;
  ctx.lineWidth = 1;
  for (const lvl of [-ext, -ext / 3, ext / 3, ext]) {
    const ry = toY(lvl);
    ctx.beginPath();
    ctx.moveTo(x0, ry);
    ctx.lineTo(x0 + plotW, ry);
    ctx.stroke();
  }
  // Zero line + symbol-centre marker (vertical, mid-window).
  ctx.strokeStyle = `rgba(${GRID_RGB}, 0.30)`;
  ctx.beginPath();
  ctx.moveTo(x0, toY(0));
  ctx.lineTo(x0 + plotW, toY(0));
  ctx.moveTo(x0 + plotW / 2, y0);
  ctx.lineTo(x0 + plotW / 2, y0 + plotH);
  ctx.stroke();

  const period = Math.max(2, Math.floor(sps) || 2);
  const win = 2 * period; // 2-symbol eye window
  if (eye.length < win) return;

  // Step so we never draw more than MAX_TRACES overlaid windows.
  const nWindows = Math.floor((eye.length - win) / period) + 1;
  const stride = Math.max(period, Math.ceil(nWindows / MAX_TRACES) * period);

  ctx.globalCompositeOperation = "lighter";
  ctx.strokeStyle = `rgba(${TRACE_RGB}, 0.16)`;
  ctx.lineWidth = 1;
  for (let start = 0; start + win <= eye.length; start += stride) {
    ctx.beginPath();
    for (let k = 0; k < win; k++) {
      const x = x0 + (k / (win - 1)) * plotW;
      const y = toY(eye[start + k]);
      if (k === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    }
    ctx.stroke();
  }
  ctx.globalCompositeOperation = "source-over";
}
