import { useEffect, useMemo, useRef, useState } from "react";
import { PageHeader } from "../components/ui/PageHeader";
import {
  fetchSpectrumDevices,
  parentSerial,
  type SpectrumDevice,
} from "../api/spectrum";
import {
  openIQStream,
  type IQFrame,
  type IQPoint,
} from "../api/diag";
import {
  demodModeToProto,
  openSymbolStream,
  type SymbolFrame,
} from "../api/symbols";
import { selectClientConfig, useShared } from "../store/shared";
import { prefs } from "../store/prefs";
import { TuningControls } from "../components/TuningControls";

// Constellation panel — a 2D scatter of the signal in the complex
// plane. Two sources, switchable via the View control:
//
//   • Symbols (default) — the receiver's symbol-decision points, the
//     *true* constellation: post matched-filter, timing recovery and
//     carrier recovery, sampled once per symbol. P25 CQPSK reads as four
//     tight clusters at ±45°/±135° (degrading to a smeared X as the eye
//     closes). P25 C4FM has no complex symbol domain, so the Display
//     control picks how to show it: "IQ ring" (default) draws the raw
//     constant-envelope circle the operator expects, while "Soft levels"
//     plots the four soft decisions on the real axis (the open 4-level
//     eye/symbol scope is C4FM's natural quality view). This is the
//     modulation-quality picture OP25 shows on its Constellation tab.
//
//   • Vector scope (raw IQ) — the wideband decimated IQ trajectory,
//     every sample including the transitions between symbols (no matched
//     filter, no symbol clock). Useful as a general what-is-this-signal
//     view: a DC/unmodulated carrier is one off-centre dot, PSK draws
//     clusters joined by transition arcs, C4FM/FSK draws concentric
//     arcs, noise a diffuse circle.
//
// The catch with a centre-tuned view is the DC spike: an SDR's DDC
// leaks a residual carrier at 0 Hz that lands on top of anything in
// the middle of the band. The offset control mixes an off-centre locked
// channel down to baseband (server-side) so its symbols can be seen
// clear of the spike — the same approach OP25's plot takes. With Hold
// off the offset follows the newest active call on the selected SDR;
// Hold pins it. DC-block (subtract the residual mean) and auto-scale
// (fill the unit circle) clean up the render further.

const TARGET_RATE_SPS = 2000;

// View source for the scatter (persisted in prefs).
type View = "symbols" | "raw";
// How the C4FM constellation is drawn (persisted in prefs): the raw IQ
// "ring" (constant-envelope circle) or the four soft-decision "levels".
type C4fmDisplay = "ring" | "soft";

const PROTOS: { value: string; label: string }[] = [
  { value: "auto", label: "Auto" },
  { value: "p25-cqpsk", label: "P25 CQPSK" },
  { value: "p25-c4fm", label: "P25 C4FM" },
];

// Reference markers (ideal post-normalisation cluster centres) drawn as
// hollow rings to guide the eye. Coordinates are in unit-plot space
// (1.0 = plot radius), placed where auto-scale lands a clean signal: the
// CQPSK clusters sit at the auto-scale fill radius on the 45° diagonals;
// the C4FM soft levels sit on the real axis at the 1:3 inner/outer ratio.
const CQPSK_MARKERS: IQPoint[] = [
  { i: 0.6, q: 0.6 },
  { i: -0.6, q: 0.6 },
  { i: -0.6, q: -0.6 },
  { i: 0.6, q: -0.6 },
].map((p) => ({ i: p.i / Math.SQRT2, q: p.q / Math.SQRT2 }));
const C4FM_MARKERS: IQPoint[] = [
  { i: -0.6, q: 0 },
  { i: -0.2, q: 0 },
  { i: 0.2, q: 0 },
  { i: 0.6, q: 0 },
];
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
  // Ideal cluster-centre rings, in unit-plot space (not gain-scaled).
  markers: IQPoint[];
}

export function Constellation() {
  const cfg = useShared(selectClientConfig);
  const activeCalls = useShared((s) => s.activeCalls);
  const [devices, setDevices] = useState<SpectrumDevice[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [conn, setConn] = useState<ConnState>("closed");
  const [latest, setLatest] = useState<IQFrame | null>(null);
  const [symLatest, setSymLatest] = useState<SymbolFrame | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [view, setView] = useState<View>(() => prefs.constellationView());
  const [proto, setProto] = useState<string>(() => prefs.constellationProto());
  const [c4fmDisplay, setC4fmDisplay] = useState<C4fmDisplay>(() =>
    prefs.constellationC4fmDisplay(),
  );
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

  const device = useMemo(
    () => devices.find((d) => d.serial === selected) ?? null,
    [devices, selected],
  );

  // Resolve the Mode selection to a concrete receiver. "auto" follows the
  // modulation the selected SDR's system is decoding (device.p25_modulation),
  // falling back to C4FM when unknown; an explicit choice is used as-is.
  // The symbol stream and ideal-cluster markers both key off this.
  const effectiveProto = useMemo(
    () => (proto === "auto" ? demodModeToProto(device?.p25_modulation) : proto),
    [proto, device],
  );

  // Which stream feeds the scatter. The raw View always uses the wideband
  // IQ trajectory. In the symbols View, CQPSK uses the symbol-decision
  // stream; C4FM has no symbol constellation, so it shows the raw IQ ring
  // by default (the constant-envelope circle the operator expects) unless
  // they pick "Soft levels" for the legacy real-axis decision points.
  const source: "iq" | "symbols" =
    view === "raw"
      ? "iq"
      : effectiveProto === "p25-c4fm" && c4fmDisplay === "ring"
        ? "iq"
        : "symbols";

  // Ideal cluster markers depend on the active source: clusters on the
  // diagonals for CQPSK symbols, levels on the real axis for the C4FM
  // soft-level view, none for the raw vector scope / IQ ring.
  const markers = useMemo<IQPoint[]>(() => {
    if (source !== "symbols") return [];
    return effectiveProto === "p25-c4fm" ? C4FM_MARKERS : CQPSK_MARKERS;
  }, [source, effectiveProto]);
  const optsRef = useRef<RenderOpts>({
    dcBlock,
    autoScale,
    zoom,
    scaleRef,
    markers,
  });
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
    optsRef.current = { dcBlock, autoScale, zoom, scaleRef, markers };
    renderConstellation(canvasRef.current, bufferRef.current, optsRef.current);
  }, [dcBlock, autoScale, zoom, markers]);

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

  // Newest active call on the selected SDR, and the view offset (kHz)
  // that would centre it. This is the "last locked channel" the issue
  // asks for: when Hold is off we track it live.
  const followOffsetKHz = useMemo(() => {
    if (!device) return null;
    const mine = activeCalls.filter(
      (c) => parentSerial(c.device_serial) === selected && !c.ended_at,
    );
    if (mine.length === 0) return null;
    const newest = mine.reduce((a, b) =>
      b.started_at > a.started_at ? b : a,
    );
    return (newest.grant.frequency_hz - device.center_hz) / 1000;
  }, [activeCalls, device, selected]);

  // Offset (kHz) that lands the view on this SDR's control channel — the
  // rest position when Hold is off and no call is active (#557). Absent
  // (null) for devices with no in-band CC, leaving the old centre default.
  const controlOffsetKHz = useMemo(() => {
    if (!device?.control_channel_hz) return null;
    return (device.control_channel_hz - device.center_hz) / 1000;
  }, [device]);

  // Follow the locked channel unless the operator pinned the view: an
  // active call wins, otherwise rest on the control channel (#557).
  const restOffsetKHz = followOffsetKHz ?? controlOffsetKHz;
  useEffect(() => {
    if (hold || restOffsetKHz == null) return;
    setOffsetKHz((cur) =>
      Math.abs(cur - restOffsetKHz) < 0.05 ? cur : restOffsetKHz,
    );
  }, [hold, restOffsetKHz]);

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
    prefs.setConstellationView(view);
  }, [view]);
  useEffect(() => {
    prefs.setConstellationProto(proto);
  }, [proto]);
  useEffect(() => {
    prefs.setConstellationC4fmDisplay(c4fmDisplay);
  }, [c4fmDisplay]);

  // Push a fresh batch of points into the rolling buffer and repaint.
  const pushPoints = (pts: IQPoint[]) => {
    const buf = bufferRef.current;
    for (const p of pts) buf.push(p);
    if (buf.length > POINT_BUFFER) {
      bufferRef.current = buf.slice(buf.length - POINT_BUFFER);
    }
    renderConstellation(canvasRef.current, bufferRef.current, optsRef.current);
  };

  // Raw IQ stream: the wideband decimated IQ trajectory (vector scope, and
  // the C4FM constant-envelope ring).
  useEffect(() => {
    if (!selected || source !== "iq") return;
    bufferRef.current = [];
    scaleRef.current = 1;
    setLatest(null);

    const stream = openIQStream(cfg, {
      serial: selected,
      rate: TARGET_RATE_SPS,
      offset: Math.round(clampedOffsetKHz * 1000),
      onFrame: (f) => {
        setLatest(f);
        pushPoints(f.points);
      },
      onStatus: setConn,
    });
    return () => stream.close();
    // Re-subscribe when the offset changes so the server re-mixes.
  }, [cfg, selected, source, clampedOffsetKHz]);

  // Symbols stream: the receiver's symbol-decision points (true
  // constellation). CQPSK carries complex points (sym_i/sym_q); C4FM has
  // none, so its 4 soft levels are plotted on the real axis.
  useEffect(() => {
    if (!selected || source !== "symbols") return;
    bufferRef.current = [];
    scaleRef.current = 1;
    setSymLatest(null);

    const stream = openSymbolStream(cfg, {
      serial: selected,
      proto: effectiveProto,
      offset: Math.round(clampedOffsetKHz * 1000),
      onFrame: (f) => {
        setSymLatest(f);
        const pts: IQPoint[] = [];
        if (f.sym_i && f.sym_i.length > 0) {
          const n = Math.min(f.sym_i.length, f.sym_q?.length ?? 0);
          for (let k = 0; k < n; k++) pts.push({ i: f.sym_i[k], q: f.sym_q[k] });
        } else if (f.soft && f.soft.length > 0) {
          for (const s of f.soft) pts.push({ i: s, q: 0 });
        }
        pushPoints(pts);
      },
      onStatus: setConn,
    });
    return () => stream.close();
  }, [cfg, selected, source, effectiveProto, clampedOffsetKHz]);

  // Prefer the device centre so the frequency view shows before the first
  // frame; fall back to the frame's stamped centre.
  const centerHz =
    device?.center_hz ?? latest?.center_hz ?? symLatest?.center_hz ?? null;
  const viewHz = centerHz != null ? centerHz + clampedOffsetKHz * 1000 : null;

  const tuningLabel = useMemo(() => {
    if (viewHz == null) return "";
    const off =
      clampedOffsetKHz === 0
        ? "centre"
        : `${clampedOffsetKHz >= 0 ? "+" : ""}${clampedOffsetKHz.toFixed(3).replace(/\.?0+$/, "")} kHz`;
    const head = `${(viewHz / 1e6).toFixed(4)} MHz (${off})`;
    if (source === "symbols") {
      if (!symLatest) return `${head} · waiting for symbols…`;
      const kind =
        symLatest.sym_i && symLatest.sym_i.length > 0
          ? "symbols"
          : "soft levels";
      return `${head} · ${symLatest.symbol_rate_hz.toFixed(0)} sym/s · ${kind}`;
    }
    if (!latest) return `${head} · waiting for samples…`;
    return `${head} · ${latest.sample_rate} sps · ${latest.energy_dbfs.toFixed(1)} dBFS`;
  }, [source, latest, symLatest, viewHz, clampedOffsetKHz]);

  return (
    <div className="space-y-3">
      <PageHeader
        title="Constellation"
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

      {/* View controls — source / offset / follow-locked-channel / cleanup. */}
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-xs">
        <label className="flex items-center gap-2">
          <span className="text-muted">View</span>
          <select
            className="bg-surface border border-border rounded px-2 py-1"
            value={view}
            onChange={(e) => setView(e.target.value as View)}
            aria-label="Constellation source"
          >
            <option value="symbols">Symbols</option>
            <option value="raw">Vector scope (raw IQ)</option>
          </select>
        </label>

        {view === "symbols" && (
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
            {proto === "auto" && (
              <span className="text-muted">
                ·&nbsp;{effectiveProto === "p25-c4fm" ? "C4FM" : "CQPSK"}
              </span>
            )}
          </label>
        )}

        {view === "symbols" && effectiveProto === "p25-c4fm" && (
          <label className="flex items-center gap-2">
            <span className="text-muted">Display</span>
            <select
              className="bg-surface border border-border rounded px-2 py-1"
              value={c4fmDisplay}
              onChange={(e) => setC4fmDisplay(e.target.value as C4fmDisplay)}
              aria-label="C4FM display"
            >
              <option value="ring">IQ ring (circle)</option>
              <option value="soft">Soft levels (line)</option>
            </select>
          </label>
        )}

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
        {view === "symbols" && effectiveProto === "p25-c4fm" ? (
          c4fmDisplay === "ring" ? (
            <>
              C4FM is constant-envelope FM with no symbol constellation, so
              this shows the raw IQ vector scope — the constant-envelope
              ring / concentric arcs the signal traces. Tune the{" "}
              <em>Offset</em> onto a locked channel to lift it off the centre
              DC spike. Switch <em>Display</em> to <em>Soft levels</em> for
              the 4-level decision points, or use the <em>Eye</em> /{" "}
              <em>Symbol scope</em> panel for C4FM modulation quality.
            </>
          ) : (
            <>
              Plots the receiver's recovered C4FM symbols. C4FM has no
              complex constellation, so its four soft levels appear on the
              real axis (rings mark the ideal ±1/±3 positions) — the open
              4-level eye on the <em>Symbol scope</em> is C4FM's natural
              quality view. Switch <em>Display</em> to <em>IQ ring</em> for
              the constant-envelope circle, or <em>Mode</em> to CQPSK for a
              2D constellation.
            </>
          )
        ) : view === "symbols" ? (
          <>
            Plots the receiver's symbol-decision points — the true
            constellation. A clean CQPSK/LSM signal forms four tight
            clusters on the ±45° diagonals (rings); a closing eye smears
            them into an X. Tune the <em>Offset</em> onto a locked
            channel; brightest = most recent.
          </>
        ) : (
          <>
            Vector scope: plots decimated IQ samples ({TARGET_RATE_SPS} sps,
            transitions included), brightest = most recent. Tune the{" "}
            <em>Offset</em> onto a locked channel to lift it off the centre
            DC spike; clusters suggest PSK, concentric arcs suggest C4FM/FSK,
            a diffuse circle is noise. Switch <em>View</em> to Symbols for
            the symbol-timed constellation.
          </>
        )}
      </div>
    </div>
  );
}

function ConnPill({ state }: { state: ConnState }) {
  if (state === "open") return <span className="pill-ok">live</span>;
  if (state === "connecting") return <span className="pill-warn">connecting</span>;
  return <span className="pill-err">offline</span>;
}

// renderConstellation paints the rolling IQ buffer: labelled ±1 axes,
// reference rings at 0.5 and 1.0, optional ideal-cluster markers (symbols
// view), and additively-blended points so dense symbol clusters bloom
// while the noise floor stays dim. DC-block subtracts the buffer mean to
// kill any residual centre offset; auto-scale eases a gain so the cloud
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

  // Ideal cluster-centre markers (symbols view): hollow amber rings at
  // the positions a clean signal settles on after auto-scale. Drawn in
  // unit-plot space so they don't move with the data gain, giving the
  // operator a fixed target to judge cluster tightness / rotation
  // against. Drawn before the points so the live scatter sits on top.
  if (opts.markers.length > 0) {
    ctx.strokeStyle = "rgba(251, 191, 36, 0.55)"; // amber-400
    ctx.lineWidth = 1.5;
    const mr = Math.max(4, radius * 0.05);
    for (const m of opts.markers) {
      const x = cx + m.i * radius;
      const y = cy - m.q * radius;
      ctx.beginPath();
      ctx.arc(x, y, mr, 0, Math.PI * 2);
      ctx.stroke();
    }
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
