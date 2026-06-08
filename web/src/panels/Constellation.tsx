import { useEffect, useMemo, useRef, useState } from "react";
import {
  fetchSpectrumDevices,
  type SpectrumDevice,
} from "../api/spectrum";
import {
  openIQStream,
  type IQFrame,
  type IQPoint,
} from "../api/diag";
import { selectClientConfig, useShared } from "../store/shared";
import { prefs } from "../store/prefs";
import { TuningControls } from "../components/TuningControls";

// Constellation panel — 2D scatter of decimated IQ samples. Useful
// for spotting the symbol-domain shape of whatever the SDR is tuned
// to without launching a separate SDR receiver.
//
//   - DC bias / unmodulated carrier   →  one bright dot off-centre
//   - PSK / QPSK                       →  two or four clusters at
//                                         ±0.5+0i etc.
//   - C4FM / FSK                       →  two or four arcs
//   - AM voice                          →  rotating cluster, modulated
//                                         in amplitude
//   - Wideband noise                    →  diffuse circle around the
//                                         origin
//
// The catch with a centre-tuned view is the DC spike: an SDR's DDC
// leaks a residual carrier at 0 Hz that lands on top of anything in
// the middle of the band, so the constellation reads as one fat blob.
// The offset control mixes an off-centre locked channel down to
// baseband (server-side, before decimation) so its symbols can be
// seen clear of the spike — the same approach OP25's plot takes. With
// Hold off the offset follows the newest active call on the selected
// SDR; Hold pins it. DC-block (subtract the residual mean) and
// auto-scale (fill the unit circle) clean up the render further.
//
// 2 ksps decimated stream → ~50 ms / frame at 4 chunks per frame → a
// canvas redraw every 50 ms is responsive without burning CPU.

const TARGET_RATE_SPS = 2000;
const POINT_BUFFER = 2000;       // how many recent points to render
// The plot is a responsive square: it fills the panel column (so it's as
// large as OP25's, not a fixed postage stamp) but stays bounded so it
// can't run away on ultrawide layouts. Measured at runtime via
// ResizeObserver; rendered at devicePixelRatio for crispness.
const MIN_CANVAS_PX = 320;
const MAX_CANVAS_PX = 880;
const LEGACY_RADIUS_PX = 187;    // old 420px plot radius — dot-scale baseline
const MARGIN = { top: 12, right: 12, bottom: 24, left: 34 };
// Base dot edge in CSS px at zoom 1 and the legacy plot size; scaled by
// the Zoom control and the live plot size so the scatter reads clearly
// instead of as pin-pricks (issue #557 follow-up).
const BASE_DOT_PX = 2;
// Auto-scale fills the ~95th-percentile radius to this fraction of the
// unit circle at zoom 1 (a touch under OP25's ~0.75 so Zoom has headroom
// to punch in without immediately clipping the cloud).
const FILL_TARGET = 0.6;

// GopherTrunk's sky-400 accent (matches --gt-accent and the Spectrum
// waterfall's blue→cyan ramp), deliberately not OP25's phosphor green.
// Drawn additively so dense clusters bloom toward cyan-white while the
// diffuse noise floor stays dim.
const POINT_RGB = "56, 189, 248";
// Neutral slate-400 for the grid / axis labels, on-brand with the
// muted slate tokens used elsewhere.
const GRID_RGB = "148, 163, 184";

type ConnState = "connecting" | "open" | "closed";

interface RenderOpts {
  dcBlock: boolean;
  autoScale: boolean;
  zoom: number;
  scaleRef: { current: number };
}

export function Constellation() {
  const cfg = useShared(selectClientConfig);
  const activeCalls = useShared((s) => s.activeCalls);
  const [devices, setDevices] = useState<SpectrumDevice[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [conn, setConn] = useState<ConnState>("closed");
  const [latest, setLatest] = useState<IQFrame | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [offsetKHz, setOffsetKHz] = useState<number>(() =>
    prefs.constellationOffsetKHz(),
  );
  const [hold, setHold] = useState<boolean>(() => prefs.constellationHold());
  const [dcBlock, setDcBlock] = useState<boolean>(() =>
    prefs.constellationDcBlock(),
  );
  const [autoScale, setAutoScale] = useState<boolean>(() =>
    prefs.constellationAutoScale(),
  );
  const [zoom, setZoom] = useState<number>(() => prefs.constellationZoom());

  const wrapRef = useRef<HTMLDivElement | null>(null);
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const bufferRef = useRef<IQPoint[]>([]);
  const scaleRef = useRef<number>(1);
  const optsRef = useRef<RenderOpts>({ dcBlock, autoScale, zoom, scaleRef });
  // Live square edge of the plot in CSS px, tracked from the container.
  const [size, setSize] = useState<number>(MIN_CANVAS_PX);

  // Track the available width and keep the plot a square that fills it
  // (bounded). Falls back to a one-shot measure where ResizeObserver is
  // absent (e.g. jsdom under test).
  useEffect(() => {
    const el = wrapRef.current;
    if (!el) return;
    const measure = () => {
      const inner = el.clientWidth - 16; // minus the p-2 padding
      const next = Math.max(
        MIN_CANVAS_PX,
        Math.min(MAX_CANVAS_PX, Math.floor(inner)),
      );
      if (next > 0) setSize(next);
    };
    measure();
    if (typeof ResizeObserver === "undefined") return;
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  // Size the canvas buffer to the displayed square × devicePixelRatio so
  // the scatter stays crisp at any size, then repaint.
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const dpr = window.devicePixelRatio || 1;
    canvas.width = Math.round(size * dpr);
    canvas.height = Math.round(size * dpr);
    canvas.style.width = `${size}px`;
    canvas.style.height = `${size}px`;
    renderConstellation(canvas, bufferRef.current, optsRef.current);
  }, [size]);

  // Keep the render-time options the WS callback reads in sync without
  // re-subscribing the stream when a toggle flips.
  useEffect(() => {
    optsRef.current = { dcBlock, autoScale, zoom, scaleRef };
    renderConstellation(canvasRef.current, bufferRef.current, optsRef.current);
  }, [dcBlock, autoScale, zoom]);

  // Reuse the spectrum devices endpoint — same broker pool.
  useEffect(() => {
    let cancel = false;
    (async () => {
      try {
        const list = await fetchSpectrumDevices(cfg);
        if (cancel) return;
        setDevices(list);
        setError(null);
        if (list.length > 0 && selected == null) setSelected(list[0].serial);
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

  // Newest active call on the selected SDR, and the view offset (kHz)
  // that would centre it. This is the "last locked channel" the issue
  // asks for: when Hold is off we track it live.
  const followOffsetKHz = useMemo(() => {
    if (!device) return null;
    const mine = activeCalls.filter(
      (c) => c.device_serial === selected && !c.ended_at,
    );
    if (mine.length === 0) return null;
    const newest = mine.reduce((a, b) =>
      b.started_at > a.started_at ? b : a,
    );
    return (newest.grant.frequency_hz - device.center_hz) / 1000;
  }, [activeCalls, device, selected]);

  // Follow the locked channel unless the operator pinned the view.
  useEffect(() => {
    if (hold || followOffsetKHz == null) return;
    setOffsetKHz((cur) =>
      Math.abs(cur - followOffsetKHz) < 0.05 ? cur : followOffsetKHz,
    );
  }, [hold, followOffsetKHz]);

  // Clamp the offset to the device's Nyquist so the slider/input can't
  // request a mix that just aliases back into the band.
  const maxOffsetKHz = device ? device.sample_rate_hz / 2000 : 1500;
  const clampedOffsetKHz = Math.max(
    -maxOffsetKHz,
    Math.min(maxOffsetKHz, offsetKHz),
  );

  // Persist view options.
  useEffect(() => {
    prefs.setConstellationOffsetKHz(offsetKHz);
  }, [offsetKHz]);
  useEffect(() => {
    prefs.setConstellationHold(hold);
  }, [hold]);
  useEffect(() => {
    prefs.setConstellationDcBlock(dcBlock);
  }, [dcBlock]);
  useEffect(() => {
    prefs.setConstellationAutoScale(autoScale);
  }, [autoScale]);
  useEffect(() => {
    prefs.setConstellationZoom(zoom);
  }, [zoom]);

  useEffect(() => {
    if (!selected) return;
    bufferRef.current = [];
    scaleRef.current = 1;
    setLatest(null);

    const stream = openIQStream(cfg, {
      serial: selected,
      rate: TARGET_RATE_SPS,
      offset: Math.round(clampedOffsetKHz * 1000),
      onFrame: (f) => {
        setLatest(f);
        const buf = bufferRef.current;
        for (const p of f.points) buf.push(p);
        if (buf.length > POINT_BUFFER) {
          bufferRef.current = buf.slice(buf.length - POINT_BUFFER);
        }
        renderConstellation(
          canvasRef.current,
          bufferRef.current,
          optsRef.current,
        );
      },
      onStatus: setConn,
    });
    return () => stream.close();
    // Re-subscribe when the offset changes so the server re-mixes.
  }, [cfg, selected, clampedOffsetKHz]);

  // Prefer the device centre so the frequency view shows before the first
  // frame; fall back to the frame's stamped centre.
  const centerHz = device?.center_hz ?? latest?.center_hz ?? null;
  const viewHz = centerHz != null ? centerHz + clampedOffsetKHz * 1000 : null;

  const tuningLabel = useMemo(() => {
    if (viewHz == null) return "";
    const off = clampedOffsetKHz === 0 ? "centre" : `${clampedOffsetKHz >= 0 ? "+" : ""}${clampedOffsetKHz.toFixed(3).replace(/\.?0+$/, "")} kHz`;
    const head = `${(viewHz / 1e6).toFixed(4)} MHz (${off})`;
    if (!latest) return `${head} · waiting for samples…`;
    return `${head} · ${latest.sample_rate} sps · ${latest.energy_dbfs.toFixed(1)} dBFS`;
  }, [latest, viewHz, clampedOffsetKHz]);

  return (
    <div className="space-y-3">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-xl font-semibold">Constellation</h2>
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
      </header>

      {error && (
        <div className="rounded border border-red-700/40 bg-red-900/20 text-red-200 text-xs px-3 py-2">
          {error}
        </div>
      )}

      {/* View controls — offset / follow-locked-channel / cleanup. */}
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-xs">
        <TuningControls
          centerHz={centerHz}
          maxOffsetKHz={maxOffsetKHz}
          offsetKHz={clampedOffsetKHz}
          onOffsetChange={(khz) => {
            setHold(true);
            setOffsetKHz(khz);
          }}
          hold={hold}
          onHoldChange={setHold}
          followingActive={followOffsetKHz != null}
          onCentre={() => {
            setHold(true);
            setOffsetKHz(0);
          }}
        />

        <label className="flex items-center gap-1.5">
          <input
            type="checkbox"
            checked={dcBlock}
            onChange={(e) => setDcBlock(e.target.checked)}
            className="accent-sky-400"
          />
          <span>DC&nbsp;block</span>
        </label>

        <label className="flex items-center gap-1.5">
          <input
            type="checkbox"
            checked={autoScale}
            onChange={(e) => setAutoScale(e.target.checked)}
            className="accent-sky-400"
          />
          <span>Auto&nbsp;scale</span>
        </label>

        <label className="flex items-center gap-2">
          <span className="text-muted">Zoom</span>
          <input
            type="range"
            min={0.5}
            max={8}
            step={0.1}
            value={zoom}
            onChange={(e) => setZoom(Number(e.target.value))}
            className="w-28 accent-sky-400"
            aria-label="Constellation zoom"
          />
          <span className="w-10 text-right font-mono text-muted">
            {zoom.toFixed(1)}×
          </span>
        </label>
      </div>

      <div className="font-mono text-xs text-muted">{tuningLabel || "—"}</div>

      <div
        ref={wrapRef}
        className="rounded border border-border bg-black flex items-center justify-center p-2"
      >
        <canvas
          ref={canvasRef}
          className="block"
          aria-label="IQ constellation scatter"
        />
      </div>

      <div className="text-[11px] text-muted">
        Plots decimated IQ samples ({TARGET_RATE_SPS} sps), brightest =
        most recent. Tune the <em>Offset</em> onto a locked voice/control
        channel to lift its symbols off the centre DC spike; clusters at
        ±0.5 suggest PSK, concentric arcs suggest C4FM/FSK, a diffuse
        circle is noise.
      </div>
    </div>
  );
}

function ConnPill({ state }: { state: ConnState }) {
  if (state === "open") return <span className="pill-ok">live</span>;
  if (state === "connecting") return <span className="pill-warn">connecting</span>;
  return <span className="pill-err">offline</span>;
}

// renderConstellation paints the rolling IQ buffer as a phosphor-green
// vector scope: labelled ±1 axes, reference rings at 0.5 and 1.0, and
// additively-blended points so dense symbol clusters bloom while the
// noise floor stays dim. DC-block subtracts the buffer mean to kill
// any residual centre offset; auto-scale eases a gain so the cloud
// fills the unit circle regardless of input amplitude.
function renderConstellation(
  canvas: HTMLCanvasElement | null,
  points: IQPoint[],
  opts: RenderOpts,
) {
  if (!canvas) return;
  const ctx = canvas.getContext("2d");
  if (!ctx) return;
  // The buffer is sized at devicePixelRatio; draw in logical (CSS) px so
  // margins, fonts and dot sizes stay consistent at any zoom/DPI.
  const dpr = window.devicePixelRatio || 1;
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  const w = canvas.width / dpr;
  const h = canvas.height / dpr;

  // Square plot region inside the label margins.
  const plot = Math.min(
    w - MARGIN.left - MARGIN.right,
    h - MARGIN.top - MARGIN.bottom,
  );
  const cx = MARGIN.left + plot / 2;
  const cy = MARGIN.top + plot / 2;
  const radius = plot / 2;

  // Background.
  ctx.fillStyle = "rgb(0, 0, 0)";
  ctx.fillRect(0, 0, w, h);

  // Grid: crosshair axes + 0.5 / 1.0 reference rings.
  ctx.lineWidth = 1;
  ctx.strokeStyle = `rgba(${GRID_RGB}, 0.30)`;
  ctx.beginPath();
  ctx.moveTo(cx, cy - radius);
  ctx.lineTo(cx, cy + radius);
  ctx.moveTo(cx - radius, cy);
  ctx.lineTo(cx + radius, cy);
  ctx.stroke();
  ctx.strokeStyle = `rgba(${GRID_RGB}, 0.18)`;
  for (const r of [0.5, 1.0]) {
    ctx.beginPath();
    ctx.arc(cx, cy, radius * r, 0, Math.PI * 2);
    ctx.stroke();
  }

  // Axis tick labels (I on the bottom, Q on the left).
  ctx.fillStyle = `rgba(${GRID_RGB}, 0.80)`;
  ctx.font = "10px ui-monospace, monospace";
  ctx.textAlign = "center";
  ctx.textBaseline = "top";
  for (const t of [-1, -0.5, 0, 0.5, 1]) {
    if (t === 0) continue;
    ctx.fillText(String(t), cx + t * radius, cy + radius + 5);
  }
  ctx.textAlign = "right";
  ctx.textBaseline = "middle";
  for (const t of [-1, -0.5, 0.5, 1]) {
    ctx.fillText(String(t), MARGIN.left - 6, cy - t * radius);
  }

  const n = points.length;
  if (n === 0) return;

  // DC-block: subtract the buffer mean to remove residual carrier
  // leakage so the cloud centres on the origin.
  let mi = 0;
  let mq = 0;
  if (opts.dcBlock) {
    for (let k = 0; k < n; k++) {
      mi += points[k].i;
      mq += points[k].q;
    }
    mi /= n;
    mq /= n;
  }

  // Auto-scale: target gain so the ~95th-percentile radius reaches ~0.9
  // of the unit circle. Using a percentile (not the absolute max) keeps a
  // stray outlier from shrinking the whole cloud, so the scatter fills the
  // circle like OP25's plot. Eased toward the target to avoid frame jitter.
  let gain = opts.scaleRef.current || 1;
  if (opts.autoScale) {
    const radii = new Float64Array(n);
    for (let k = 0; k < n; k++) {
      const di = points[k].i - mi;
      const dq = points[k].q - mq;
      radii[k] = Math.hypot(di, dq);
    }
    radii.sort();
    const refR = Math.max(1e-6, radii[Math.min(n - 1, Math.floor(0.95 * n))]);
    const target = Math.max(0.2, Math.min(8, FILL_TARGET / refR));
    gain = gain + (target - gain) * 0.1;
    opts.scaleRef.current = gain;
  } else {
    gain = 1;
    opts.scaleRef.current = 1;
  }

  // Zoom magnifies both the cloud and the dot size so the scatter reads as
  // dots, not pin-pricks, and can be dialled to taste (issue #557 follow-up).
  // Dots also grow with the plot size so a large canvas keeps proportionate,
  // OP25-sized points rather than the same few pixels on a much bigger ring.
  const zoom = opts.zoom > 0 ? opts.zoom : 1;
  const finalGain = gain * zoom;
  const sizeScale = radius / LEGACY_RADIUS_PX;
  const dotPx = Math.max(
    2,
    Math.min(14, Math.round(BASE_DOT_PX * zoom * sizeScale)),
  );
  const dotHalf = dotPx / 2;

  // Additive blending so overlapping symbols accumulate brightness.
  ctx.globalCompositeOperation = "lighter";
  for (let idx = 0; idx < n; idx++) {
    const p = points[idx];
    const i = (p.i - mi) * finalGain;
    const q = (p.q - mq) * finalGain;
    const x = cx + i * radius;
    const y = cy - q * radius; // canvas Y grows downward
    // Fade: oldest samples are dim, newest are bright.
    const age = idx / n; // 0..1
    const alpha = 0.05 + 0.5 * age * age;
    ctx.fillStyle = `rgba(${POINT_RGB}, ${alpha})`;
    ctx.fillRect(x - dotHalf, y - dotHalf, dotPx, dotPx);
  }
  ctx.globalCompositeOperation = "source-over";
}
