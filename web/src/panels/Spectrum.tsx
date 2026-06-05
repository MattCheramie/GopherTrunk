import { useEffect, useMemo, useRef, useState } from "react";
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

type ConnState = "connecting" | "open" | "closed";

export function Spectrum() {
  const cfg = useShared(selectClientConfig);
  const [devices, setDevices] = useState<SpectrumDevice[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [latest, setLatest] = useState<SpectrumFrame | null>(null);
  const [conn, setConn] = useState<ConnState>("closed");
  const [error, setError] = useState<string | null>(null);
  const [bookmarkList, setBookmarkList] = useState<Bookmark[]>([]);

  const [hover, setHover] = useState<{ hz: number; db: number } | null>(null);

  const canvasRef = useRef<HTMLCanvasElement | null>(null);
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
        if (list.length > 0 && selected == null) setSelected(list[0].serial);
      } catch (e) {
        if (cancel) return;
        setError(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancel = true;
    };
    // Re-fetch whenever the connection identity changes.
  }, [cfg, selected]);

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
    setLatest(null);

    const stream = openSpectrumStream(cfg, {
      serial: selected,
      bins: FFT_BINS,
      fps: FPS,
      onFrame: (f) => {
        setLatest(f);
        latestRef.current = f;
        const row = new Float32Array(f.bins);
        rowsRef.current = [row, ...rowsRef.current.slice(0, HISTORY_ROWS - 1)];
        renderWaterfall(
          canvasRef.current,
          rowsRef.current,
          f,
          bookmarksRef.current,
        );
      },
      onStatus: setConn,
    });
    return () => stream.close();
  }, [cfg, selected]);

  // Convert a click on the canvas into a centre frequency and post
  // it to the tune endpoint. Maps the click X position back through
  // the FFT-shifted bin layout: leftmost bin = (centerHz -
  // sampleRate/2), rightmost = (centerHz + sampleRate/2 -
  // sampleRate/N).
  const handleCanvasClick = async (e: React.MouseEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current;
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
    const canvas = canvasRef.current;
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
      <header className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-xl font-semibold">Spectrum</h2>
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
        {DB_FLOOR} dBFS (cold) → {DB_CEIL} dBFS (hot). New frames render at
        the top; the canvas scrolls down as history accumulates. Hover the
        waterfall to read the frequency and signal level under the cursor;
        click anywhere to retune the SDR to that frequency. Bookmark markers
        ({bookmarkList.length} visible) appear as cyan ticks along the top.
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
