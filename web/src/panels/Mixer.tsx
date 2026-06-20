import { useEffect, useMemo, useRef, useState } from "react";
import { PageHeader } from "../components/ui/PageHeader";
import {
  fetchSpectrumDevices,
  defaultSymbolDevice,
  type SpectrumDevice,
} from "../api/spectrum";
import { openMixerStream, type MixerFrame } from "../api/mixer";
import { selectClientConfig, useShared } from "../store/shared";
import { prefs } from "../store/prefs";
import { TuningControls } from "../components/TuningControls";

// Mixer plot — GopherTrunk's take on OP25's Raw Mixer / Tuned Mixer tabs.
// The daemon runs a parallel P25 receiver on the selected channel and
// FFTs the channelized baseband two ways: "raw" (the carrier wherever its
// residual offset leaves it) and "tuned" (the same window re-mixed by the
// receiver's carrier-offset estimate, so a locked loop pulls the carrier
// to centre). Comparing the two shows the carrier-recovery correction at a
// glance: a raw peak that sits off-centre and a tuned peak that snaps back
// to 0 Hz is a healthy lock.

const DB_FLOOR = -100;
const DB_CEIL = 0;
const PLOT_W = 1024; // internal canvas width (bins resample to this)
const PLOT_H = 200;

const PROTOS: { value: string; label: string }[] = [
  { value: "p25-c4fm", label: "P25 C4FM" },
  { value: "p25-cqpsk", label: "P25 CQPSK" },
];

type ConnState = "connecting" | "open" | "closed";

export function Mixer() {
  const cfg = useShared(selectClientConfig);
  const activeCalls = useShared((s) => s.activeCalls);
  const [devices, setDevices] = useState<SpectrumDevice[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [conn, setConn] = useState<ConnState>("closed");
  const [latest, setLatest] = useState<MixerFrame | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [proto, setProto] = useState<string>(() => prefs.mixerProto());
  const [offsetKHz, setOffsetKHz] = useState<number>(() =>
    prefs.mixerOffsetKHz(),
  );
  const [hold, setHold] = useState<boolean>(() => prefs.mixerHold());

  const rawRef = useRef<HTMLCanvasElement | null>(null);
  const tunedRef = useRef<HTMLCanvasElement | null>(null);

  useEffect(() => {
    let cancel = false;
    (async () => {
      try {
        const list = await fetchSpectrumDevices(cfg);
        if (cancel) return;
        setDevices(list);
        setError(null);
        if (selected == null) {
          const def = defaultSymbolDevice(list);
          if (def) setSelected(def.serial);
        }
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

  // Follow the newest active call on this SDR (unless Hold is on), exactly
  // like the Tuning panel — so the plot lands on a live channel.
  const followOffsetKHz = useMemo(() => {
    if (!device) return null;
    const mine = activeCalls.filter(
      (c) => c.device_serial === selected && !c.ended_at,
    );
    if (mine.length === 0) return null;
    const newest = mine.reduce((a, b) => (b.started_at > a.started_at ? b : a));
    return (newest.grant.frequency_hz - device.center_hz) / 1000;
  }, [activeCalls, device, selected]);

  // Offset (kHz) that lands the view on this SDR's control channel — the
  // rest position when Hold is off and no call is active (#557). Absent
  // (null) for devices with no in-band CC, leaving the old centre default.
  const controlOffsetKHz = useMemo(() => {
    if (!device?.control_channel_hz) return null;
    return (device.control_channel_hz - device.center_hz) / 1000;
  }, [device]);

  // Active call wins; otherwise rest on the control channel (#557).
  const restOffsetKHz = followOffsetKHz ?? controlOffsetKHz;
  useEffect(() => {
    if (hold || restOffsetKHz == null) return;
    setOffsetKHz((cur) =>
      Math.abs(cur - restOffsetKHz) < 0.05 ? cur : restOffsetKHz,
    );
  }, [hold, restOffsetKHz]);

  const maxOffsetKHz = device ? device.sample_rate_hz / 2000 : 1500;
  const clampedOffsetKHz = Math.max(
    -maxOffsetKHz,
    Math.min(maxOffsetKHz, offsetKHz),
  );

  useEffect(() => {
    prefs.setMixerOffsetKHz(offsetKHz);
  }, [offsetKHz]);
  useEffect(() => {
    prefs.setMixerHold(hold);
  }, [hold]);
  useEffect(() => {
    prefs.setMixerProto(proto);
  }, [proto]);

  useEffect(() => {
    if (!selected) return;
    setLatest(null);
    renderFftLine(rawRef.current, null, 0, null);
    renderFftLine(tunedRef.current, null, 0, 0);

    const stream = openMixerStream(cfg, {
      serial: selected,
      proto,
      offset: Math.round(clampedOffsetKHz * 1000),
      onFrame: (f) => {
        setLatest(f);
        renderFftLine(
          rawRef.current,
          f.raw_bins,
          f.sample_rate_hz,
          f.carrier_offset_hz,
        );
        renderFftLine(tunedRef.current, f.tuned_bins, f.sample_rate_hz, 0);
      },
      onStatus: setConn,
    });
    return () => stream.close();
  }, [cfg, selected, proto, clampedOffsetKHz]);

  const centerHz = device?.center_hz ?? latest?.center_hz ?? null;
  const viewHz = centerHz != null ? centerHz + clampedOffsetKHz * 1000 : null;

  const tuningLabel = useMemo(() => {
    if (viewHz == null) return "";
    const off =
      clampedOffsetKHz === 0
        ? "centre"
        : `${clampedOffsetKHz >= 0 ? "+" : ""}${clampedOffsetKHz.toFixed(3).replace(/\.?0+$/, "")} kHz`;
    const head = `${(viewHz / 1e6).toFixed(4)} MHz (${off})`;
    if (!latest) return `${head} · waiting for receiver…`;
    const span = latest.sample_rate_hz / 1000;
    return `${head} · ±${(span / 2).toFixed(1)} kHz span · carrier ${latest.carrier_offset_hz.toFixed(0)} Hz`;
  }, [latest, viewHz, clampedOffsetKHz]);

  return (
    <div className="space-y-3">
      <PageHeader
        title="Mixer"
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

      <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-xs">
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
        </label>

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
      </div>

      <div className="font-mono text-xs text-muted">{tuningLabel || "—"}</div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
        <figure className="rounded border border-border bg-black overflow-hidden">
          <figcaption className="px-2 py-1 text-xs text-muted border-b border-border">
            Raw mixer — carrier at residual offset (amber marker)
          </figcaption>
          <canvas
            ref={rawRef}
            width={PLOT_W}
            height={PLOT_H}
            className="block w-full"
            style={{ height: PLOT_H }}
            aria-label="Raw mixer spectrum — channel baseband before carrier recovery"
          />
        </figure>
        <figure className="rounded border border-border bg-black overflow-hidden">
          <figcaption className="px-2 py-1 text-xs text-muted border-b border-border">
            Tuned mixer — carrier pulled to centre
          </figcaption>
          <canvas
            ref={tunedRef}
            width={PLOT_W}
            height={PLOT_H}
            className="block w-full"
            style={{ height: PLOT_H }}
            aria-label="Tuned mixer spectrum — channel baseband after carrier recovery"
          />
        </figure>
      </div>

      <div className="text-[11px] text-muted">
        Channel-baseband power spectrum off the live receiver (OP25's Raw /
        Tuned Mixer). <strong>Raw</strong> shows the carrier where its
        residual frequency offset leaves it — the amber line marks the
        loop's carrier-offset estimate. <strong>Tuned</strong> re-mixes that
        same window by the estimate, so a locked receiver snaps the carrier
        onto the centre (0&nbsp;Hz) line. A raw peak that sits off-centre and
        a tuned peak at centre is a healthy lock; a tuned peak still off
        centre means the loop hasn't acquired. Dial the <em>Offset</em> onto
        an active channel and pick the <em>Mode</em> that matches the site.
      </div>
    </div>
  );
}

// dbToY maps a dBFS magnitude to a canvas Y coordinate: DB_CEIL at the top
// (y=0), DB_FLOOR at the bottom. Out-of-range clamps to the edges.
function dbToY(db: number, height: number): number {
  let t = (db - DB_FLOOR) / (DB_CEIL - DB_FLOOR);
  if (t < 0) t = 0;
  else if (t > 1) t = 1;
  return (1 - t) * height;
}

// renderFftLine draws one FFT power-vs-frequency trace. bins are dBFS,
// FFT-shifted (index 0 = -sampleRate/2, centre = DC). A dim vertical line
// marks DC; markerHz (when non-null) draws an amber marker at that
// frequency — used to flag the carrier offset on the raw plot. A null
// bins blanks the plot (device change / no data yet).
function renderFftLine(
  canvas: HTMLCanvasElement | null,
  bins: number[] | null,
  sampleRateHz: number,
  markerHz: number | null,
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

  // Centre (0 Hz) reference line.
  const cx = Math.round(width / 2) + 0.5;
  ctx.strokeStyle = "rgba(148, 163, 184, 0.35)";
  ctx.beginPath();
  ctx.moveTo(cx, 0);
  ctx.lineTo(cx, height);
  ctx.stroke();

  // Carrier-offset marker (amber) on the raw plot.
  if (markerHz != null && markerHz !== 0 && sampleRateHz > 0) {
    const ratio = markerHz / sampleRateHz + 0.5;
    if (ratio >= 0 && ratio <= 1) {
      const mx = Math.round(ratio * width) + 0.5;
      ctx.strokeStyle = "rgba(251, 191, 36, 0.8)";
      ctx.beginPath();
      ctx.moveTo(mx, 0);
      ctx.lineTo(mx, height);
      ctx.stroke();
    }
  }

  if (!bins || bins.length === 0) return;

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

function ConnPill({ state }: { state: ConnState }) {
  if (state === "open") return <span className="pill-ok">live</span>;
  if (state === "connecting")
    return <span className="pill-warn">connecting</span>;
  return <span className="pill-err">offline</span>;
}
