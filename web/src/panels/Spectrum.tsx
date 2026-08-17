import { useEffect, useMemo, useRef, useState } from "react";
import { PageHeader } from "../components/ui/PageHeader";
import {
  fetchSpectrumDevices,
  openSpectrumStream,
  tuneSpectrumDevice,
  type SpectrumDevice,
  type SpectrumFrame,
} from "../api/spectrum";
import {
  bookmarks as bookmarksAPI,
  type Bookmark,
} from "../api/bookmarks";
import { selectClientConfig, useShared } from "../store/shared";

// Spectrum waterfall panel. Operator picks an SDR from the daemon's
// broker pool; we open a WebSocket to /api/v1/spectrum/stream and
// render a scrolling waterfall on a canvas. Frames arrive at the
// negotiated FPS (10 by default); each frame becomes one row of the
// waterfall.
//
// dBFS values are colour-mapped on a fixed [-100, 0] dB range and a
// blue→cyan→yellow→red palette. Range and palette are deliberately
// hard-coded for v1 — operator preference toggles can come later.
const FFT_BINS = 2048;
const FPS = 15;
const HISTORY_ROWS = 256;
const DB_FLOOR = -100;
const DB_CEIL = 0;
// Internal pixel height of the spectrum-analyzer line plot drawn above
// the waterfall. Shares the waterfall's full-width layout and FFT-shifted
// bin→x mapping so a peak in the analyzer lines up vertically with its
// streak in the waterfall below.
const ANALYZER_H = 160;

// Mirrors SocketStatus from api/reconnectingSocket. "gone" is terminal: the
// stream stopped retrying because the device is not coming back.
type ConnState = "connecting" | "open" | "closed" | "gone";

export function Spectrum() {
  const cfg = useShared(selectClientConfig);
  const [devices, setDevices] = useState<SpectrumDevice[]>([]);
  // Bumped when an open stream gives up on the selected serial, forcing a
  // device re-enumeration. Without it the selection was set once and never
  // reconciled, so after a daemon restart with different hardware the panel
  // kept asking for a device that no longer existed.
  const [deviceEpoch, setDeviceEpoch] = useState(0);
  const [selected, setSelected] = useState<string | null>(null);
  const [latest, setLatest] = useState<SpectrumFrame | null>(null);
  const [conn, setConn] = useState<ConnState>("closed");
  const [error, setError] = useState<string | null>(null);
  const [bookmarkList, setBookmarkList] = useState<Bookmark[]>([]);

  const [hover, setHover] = useState<{ hz: number; db: number } | null>(null);

  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const analyzerRef = useRef<HTMLCanvasElement | null>(null);
  const rowsRef = useRef<Float32Array[]>([]);
  const latestRef = useRef<SpectrumFrame | null>(null);
  const bookmarksRef = useRef<Bookmark[]>([]);

  // Discover SDRs.
  useEffect(() => {
    let cancel = false;
    (async () => {
      try {
        const list = await fetchSpectrumDevices(cfg);
        if (cancel) return;
        setDevices(list);
        setError(null);
        // Re-pick when nothing is selected yet, or when the selection is no
        // longer one of the daemon's devices.
        const stale =
          selected != null && !list.some((d) => d.serial === selected);
        if (list.length > 0 && (selected == null || stale)) {
          setSelected(list[0].serial);
        }
      } catch (e) {
        if (cancel) return;
        setError(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancel = true;
    };
    // Re-fetch whenever the connection identity changes, and when an open
    // stream gives up on the selected serial.
  }, [cfg, selected, deviceEpoch]);

  // Fetch bookmarks for the click-to-tune + marker overlay. Refresh
  // on a long interval; SSE refresh is a follow-up.
  useEffect(() => {
    let cancel = false;
    const refresh = async () => {
      try {
        const list = await bookmarksAPI.list(cfg);
        if (cancel) return;
        bookmarksRef.current = list;
        setBookmarkList(list);
        // Re-render so the marker overlay updates against the latest
        // canvas state, even if no new frame has landed yet.
        if (latestRef.current) {
          renderWaterfall(
            canvasRef.current,
            rowsRef.current,
            latestRef.current,
            list,
          );
        }
      } catch {
        // bookmarks are best-effort here; silent failure keeps the
        // primary spectrum view alive when the daemon was started
        // without storage.
      }
    };
    refresh();
    const t = window.setInterval(refresh, 30_000);
    return () => {
      cancel = true;
      window.clearInterval(t);
    };
  }, [cfg]);

  // Open the WS stream for the selected SDR.
  useEffect(() => {
    if (!selected) return;
    // Clear history on device change so we don't render bins from a
    // different centre frequency on the same canvas row.
    rowsRef.current = [];
    latestRef.current = null;
    setLatest(null);
    // Blank the analyzer trace immediately; it otherwise keeps the last
    // device's curve painted until the first new frame lands.
    renderAnalyzer(analyzerRef.current, null);

    const stream = openSpectrumStream(cfg, {
      serial: selected,
      bins: FFT_BINS,
      fps: FPS,
      onFrame: (f) => {
        setLatest(f);
        latestRef.current = f;
        const row = new Float32Array(f.bins);
        rowsRef.current = [row, ...rowsRef.current.slice(0, HISTORY_ROWS - 1)];
        renderAnalyzer(analyzerRef.current, f);
        renderWaterfall(
          canvasRef.current,
          rowsRef.current,
          f,
          bookmarksRef.current,
        );
      },
      onStatus: setConn,
      onGone: () => setDeviceEpoch((n) => n + 1),
    });
    return () => stream.close();
  }, [cfg, selected]);

  // Convert a click on the canvas into a centre frequency and post
  // it to the tune endpoint. Maps the click X position back through
  // the FFT-shifted bin layout: leftmost bin = (centerHz -
  // sampleRate/2), rightmost = (centerHz + sampleRate/2 -
  // sampleRate/N).
  const handleCanvasClick = async (e: React.MouseEvent<HTMLCanvasElement>) => {
    // currentTarget is whichever canvas was clicked — the analyzer line
    // plot or the waterfall. Both share the same full-width X→frequency
    // mapping, so click-to-tune works identically on either.
    const canvas = e.currentTarget;
    const frame = latestRef.current;
    if (!canvas || !frame || !selected) return;
    const xRatio = cursorXRatio(canvas, e.clientX);
    if (xRatio < 0 || xRatio > 1) return;
    const targetHz = Math.round(xRatioToHz(frame, xRatio));
    if (targetHz <= 0) return;
    try {
      await tuneSpectrumDevice(cfg, selected, targetHz);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  // Track the cursor over the waterfall and surface the frequency it
  // sits over plus the signal level of the underlying bin. Reuses the
  // same FFT-shifted X→frequency mapping as click-to-tune; the dBFS
  // value comes from the newest waterfall row (rowsRef.current[0])
  // using the same nearest-neighbor bin index renderWaterfall draws.
  const handleCanvasMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const canvas = e.currentTarget;
    const frame = latestRef.current;
    if (!canvas || !frame) {
      setHover(null);
      return;
    }
    const xRatio = cursorXRatio(canvas, e.clientX);
    if (xRatio < 0 || xRatio > 1) {
      setHover(null);
      return;
    }
    const hz = xRatioToHz(frame, xRatio);
    const row = rowsRef.current[0];
    let db = NaN;
    if (row && row.length > 0) {
      const idx = Math.min(row.length - 1, Math.floor(xRatio * row.length));
      db = row[idx];
    }
    setHover({ hz, db });
  };

  const tuningLabel = useMemo(() => {
    if (!latest) return "";
    return `${(latest.center_hz / 1e6).toFixed(4)} MHz · ${(latest.sample_rate_hz / 1e6).toFixed(3)} MS/s · ${latest.bins.length} bins`;
  }, [latest]);

  return (
    <div className="space-y-3">
      <PageHeader
        title="Spectrum"
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

      <div className="flex items-center justify-between gap-3 font-mono text-xs">
        <span className="text-muted">{tuningLabel || "—"}</span>
        <span className="text-accent">
          {hover
            ? `${formatHz(hover.hz)}${
                Number.isFinite(hover.db) ? ` · ${hover.db.toFixed(1)} dBFS` : ""
              }`
            : ""}
        </span>
      </div>

      <div className="rounded border border-border bg-black overflow-hidden">
        <canvas
          ref={analyzerRef}
          width={FFT_BINS}
          height={ANALYZER_H}
          className="block w-full cursor-crosshair border-b border-border"
          style={{ height: 160 }}
          onClick={handleCanvasClick}
          onMouseMove={handleCanvasMove}
          onMouseLeave={() => setHover(null)}
          aria-label="Spectrum analyzer — live signal power across frequency. Hover to read the frequency and signal level, click to tune the SDR to that frequency"
        />
        <canvas
          ref={canvasRef}
          width={FFT_BINS}
          height={HISTORY_ROWS}
          className="block w-full cursor-crosshair"
          style={{ imageRendering: "pixelated", height: 320 }}
          onClick={handleCanvasClick}
          onMouseMove={handleCanvasMove}
          onMouseLeave={() => setHover(null)}
          aria-label="Spectrum waterfall — hover to read the frequency and signal level, click to tune the SDR to that frequency"
        />
      </div>

      <div className="text-[11px] text-muted">
        Top: live signal power vs frequency ({DB_FLOOR} to {DB_CEIL} dBFS,
        grid every 20 dB). Bottom: waterfall history — {DB_FLOOR} dBFS (cold)
        → {DB_CEIL} dBFS (hot); new frames render at the top and scroll down.
        Hover either view to read the frequency and signal level under the
        cursor; click anywhere to retune the SDR to that frequency. Bookmark
        markers ({bookmarkList.length} visible) appear as cyan ticks along the
        top of the waterfall.
      </div>
    </div>
  );
}

// cursorXRatio maps a pointer's clientX to a 0..1 position across the
// canvas. Values outside [0, 1] mean the pointer is off the canvas edge.
function cursorXRatio(canvas: HTMLCanvasElement, clientX: number): number {
  const rect = canvas.getBoundingClientRect();
  return (clientX - rect.left) / rect.width;
}

// xRatioToHz maps a 0..1 X position across the waterfall back through the
// FFT-shifted bin layout to a frequency in Hz: leftmost edge =
// (center - sampleRate/2), rightmost edge = (center + sampleRate/2).
// Shared by click-to-tune and the hover readout so both agree.
function xRatioToHz(frame: SpectrumFrame, xRatio: number): number {
  const sampleRate = frame.sample_rate_hz;
  return frame.center_hz - sampleRate / 2 + sampleRate * xRatio;
}

// formatHz renders a frequency for the hover readout. Mirrors the
// formatter used by the scanner panels (MHz with 4 decimals in the
// VHF/UHF range operators care about here).
function formatHz(hz: number): string {
  if (!Number.isFinite(hz)) return "—";
  if (hz >= 1_000_000) return `${(hz / 1_000_000).toFixed(4)} MHz`;
  if (hz >= 1_000) return `${(hz / 1_000).toFixed(3)} kHz`;
  return `${Math.round(hz)} Hz`;
}

function ConnPill({ state }: { state: ConnState }) {
  if (state === "open")
    return <span className="pill-ok">live</span>;
  if (state === "connecting")
    return <span className="pill-warn">connecting</span>;
  return <span className="pill-err">offline</span>;
}

// dbToY maps a dBFS magnitude to a canvas Y coordinate for the analyzer
// line plot: DB_CEIL (0 dBFS, strongest) sits at the top (y=0), DB_FLOOR
// (weakest) at the bottom (y=height). Out-of-range values clamp to the
// edges. Same [-100, 0] dB span as the waterfall colormap so the two
// views read consistently.
function dbToY(db: number, height: number): number {
  let t = (db - DB_FLOOR) / (DB_CEIL - DB_FLOOR);
  if (t < 0) t = 0;
  else if (t > 1) t = 1;
  return (1 - t) * height;
}

// renderAnalyzer draws the live power-vs-frequency trace (the "spectrum
// analyzer" curve, like SDR#'s top panel) for the most recent frame.
// Horizontal grid lines mark every 20 dB across the [-100, 0] dBFS span;
// the trace is the frame's dBFS bins resampled to the canvas width with
// the same nearest-neighbor mapping the waterfall uses, so a peak here
// lines up vertically with its streak in the waterfall below. A null
// frame blanks the plot (device change / no data yet).
function renderAnalyzer(
  canvas: HTMLCanvasElement | null,
  frame: SpectrumFrame | null,
) {
  if (!canvas) return;
  const ctx = canvas.getContext("2d");
  if (!ctx) return;
  const width = canvas.width;
  const height = canvas.height;

  ctx.fillStyle = "#000";
  ctx.fillRect(0, 0, width, height);

  // Horizontal dB grid every 20 dB.
  ctx.strokeStyle = "rgba(255, 255, 255, 0.08)";
  ctx.lineWidth = 1;
  for (let db = DB_CEIL; db >= DB_FLOOR; db -= 20) {
    const y = Math.round(dbToY(db, height)) + 0.5;
    ctx.beginPath();
    ctx.moveTo(0, y);
    ctx.lineTo(width, y);
    ctx.stroke();
  }

  const bins = frame?.bins;
  if (!bins || bins.length === 0) return;

  // Trace path (reused for the fill and the stroke).
  const trace = () => {
    ctx.beginPath();
    for (let x = 0; x < width; x++) {
      const srcIdx = Math.floor((x * bins.length) / width);
      const y = dbToY(bins[srcIdx], height);
      if (x === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    }
  };

  // Subtle fill under the curve.
  trace();
  ctx.lineTo(width, height);
  ctx.lineTo(0, height);
  ctx.closePath();
  ctx.fillStyle = "rgba(52, 211, 153, 0.12)";
  ctx.fill();

  // Trace stroke on top.
  trace();
  ctx.strokeStyle = "#34d399";
  ctx.lineWidth = 2;
  ctx.stroke();
}

// renderWaterfall draws the current history onto the canvas. Newest row
// at the top. dBFS → palette mapping is linear from DB_FLOOR (blue) to
// DB_CEIL (red). Off-canvas (canvas not yet mounted) is a no-op.
//
// frame is the most-recent SpectrumFrame (used for the bookmark-axis
// mapping); bookmarks is the operator's bookmark list — markers are
// drawn as 4-pixel cyan ticks across the top of the waterfall where
// any bookmark's freq_hz falls inside the visible band.
function renderWaterfall(
  canvas: HTMLCanvasElement | null,
  rows: Float32Array[],
  frame: SpectrumFrame | null,
  bookmarks: Bookmark[],
) {
  if (!canvas) return;
  const ctx = canvas.getContext("2d");
  if (!ctx) return;
  const width = canvas.width;
  const height = canvas.height;
  const img = ctx.createImageData(width, height);
  for (let y = 0; y < height; y++) {
    const row = rows[y];
    const base = y * width * 4;
    if (!row || row.length === 0) {
      // Fill with transparent-black.
      for (let x = 0; x < width; x++) {
        const i = base + x * 4;
        img.data[i] = 0;
        img.data[i + 1] = 0;
        img.data[i + 2] = 0;
        img.data[i + 3] = 255;
      }
      continue;
    }
    // Bin count may not equal canvas width; resample with nearest-neighbor.
    for (let x = 0; x < width; x++) {
      const srcIdx = Math.floor((x * row.length) / width);
      const db = row[srcIdx];
      const [r, g, b] = dbToColor(db);
      const i = base + x * 4;
      img.data[i] = r;
      img.data[i + 1] = g;
      img.data[i + 2] = b;
      img.data[i + 3] = 255;
    }
  }
  ctx.putImageData(img, 0, 0);

  // Bookmark markers along the top edge — drawn after putImageData so
  // they sit on top of the pixel data. Only render bookmarks whose
  // frequency lands inside the visible band; outside-band bookmarks
  // are simply omitted from this view (they're still listed on the
  // /bookmarks panel).
  if (frame && bookmarks.length > 0) {
    const sampleRate = frame.sample_rate_hz;
    if (sampleRate > 0) {
      const minHz = frame.center_hz - sampleRate / 2;
      const maxHz = frame.center_hz + sampleRate / 2;
      ctx.fillStyle = "rgba(120, 220, 255, 0.95)";
      for (const b of bookmarks) {
        if (b.freq_hz < minHz || b.freq_hz > maxHz) continue;
        const x = Math.round(((b.freq_hz - minHz) / sampleRate) * width);
        // 6 px tall, 2 px wide tick.
        ctx.fillRect(x - 1, 0, 2, 6);
      }
    }
  }
}

// dbToColor maps a dBFS magnitude to a 5-stop palette:
//   ≤-100 dBFS → black
//   -100..-70  → black → blue
//   -70..-50   → blue → cyan
//   -50..-30   → cyan → yellow
//   -30..0     → yellow → red
function dbToColor(db: number): [number, number, number] {
  if (db <= DB_FLOOR) return [0, 0, 0];
  if (db >= DB_CEIL) return [255, 0, 0];
  const t = (db - DB_FLOOR) / (DB_CEIL - DB_FLOOR); // 0..1
  if (t < 0.3) {
    // black → blue
    const k = t / 0.3;
    return [0, 0, Math.round(255 * k)];
  }
  if (t < 0.5) {
    // blue → cyan
    const k = (t - 0.3) / 0.2;
    return [0, Math.round(255 * k), 255];
  }
  if (t < 0.7) {
    // cyan → yellow
    const k = (t - 0.5) / 0.2;
    return [Math.round(255 * k), 255, Math.round(255 * (1 - k))];
  }
  // yellow → red
  const k = (t - 0.7) / 0.3;
  return [255, Math.round(255 * (1 - k)), 0];
}
