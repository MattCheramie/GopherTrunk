import { useEffect, useMemo, useRef, useState } from "react";
import { PageHeader } from "../components/ui/PageHeader";
import {
  fetchSpectrumDevices,
  defaultSymbolDevice,
  type SpectrumDevice,
} from "../api/spectrum";
import { openSymbolStream, type SymbolFrame } from "../api/symbols";
import { selectClientConfig, useShared } from "../store/shared";
import { prefs } from "../store/prefs";
import { EyeDiagramChart } from "../components/EyeDiagramChart";
import { TuningControls } from "../components/TuningControls";

// Eye diagram panel — GopherTrunk's take on OP25's "datascope". The
// daemon runs a parallel P25 C4FM receiver on the selected channel and
// streams the oversampled, AGC-scaled matched-filter output; we fold it
// over the symbol period and overlay the windows so the four-level eye
// is visible. A healthy channel shows four open bands with clear gaps at
// the symbol-decision instant; a closed eye means timing / SNR trouble.
//
// C4FM only: the eye is a property of the FM-discriminated baseband.
// CQPSK/LSM has no 4-level eye — its quality view is the complex
// constellation on the Constellation panel.

const WINDOW_SAMPLES = 4000; // rolling oversampled-sample buffer

type ConnState = "connecting" | "open" | "closed";

export function EyeDiagram() {
  const cfg = useShared(selectClientConfig);
  const activeCalls = useShared((s) => s.activeCalls);
  const [devices, setDevices] = useState<SpectrumDevice[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [conn, setConn] = useState<ConnState>("closed");
  const [latest, setLatest] = useState<SymbolFrame | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [offsetKHz, setOffsetKHz] = useState<number>(() =>
    prefs.eyeOffsetKHz(),
  );
  const [hold, setHold] = useState<boolean>(() => prefs.eyeHold());

  // Rolling oversampled-sample window + its fold period, recreated per
  // frame so the chart sees fresh references.
  const [eye, setEye] = useState<{ samples: number[]; sps: number }>({
    samples: [],
    sps: 0,
  });
  const eyeRef = useRef<number[]>([]);

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
    prefs.setEyeOffsetKHz(offsetKHz);
  }, [offsetKHz]);
  useEffect(() => {
    prefs.setEyeHold(hold);
  }, [hold]);

  // The eye is C4FM-only; the stream proto is fixed accordingly.
  useEffect(() => {
    if (!selected) return;
    eyeRef.current = [];
    setEye({ samples: [], sps: 0 });
    setLatest(null);

    const stream = openSymbolStream(cfg, {
      serial: selected,
      proto: "p25-c4fm",
      offset: Math.round(clampedOffsetKHz * 1000),
      onFrame: (f) => {
        setLatest(f);
        if (!f.eye_soft || f.eye_soft.length === 0) return;
        const buf = eyeRef.current.concat(f.eye_soft);
        eyeRef.current = buf.slice(Math.max(0, buf.length - WINDOW_SAMPLES));
        setEye({ samples: eyeRef.current, sps: f.eye_sps || 0 });
      },
      onStatus: setConn,
    });
    return () => stream.close();
  }, [cfg, selected, clampedOffsetKHz]);

  const centerHz = device?.center_hz ?? latest?.center_hz ?? null;
  const viewHz = centerHz != null ? centerHz + clampedOffsetKHz * 1000 : null;

  const tuningLabel = useMemo(() => {
    if (viewHz == null) return "";
    const off =
      clampedOffsetKHz === 0
        ? "centre"
        : `${clampedOffsetKHz >= 0 ? "+" : ""}${clampedOffsetKHz.toFixed(3).replace(/\.?0+$/, "")} kHz`;
    const head = `${(viewHz / 1e6).toFixed(4)} MHz (${off})`;
    if (!latest) return `${head} · waiting for symbols…`;
    if (eye.sps > 0) return `${head} · ${eye.sps} samples/symbol`;
    return `${head} · waiting for eye samples…`;
  }, [latest, eye.sps, viewHz, clampedOffsetKHz]);

  return (
    <div className="space-y-3">
      <PageHeader
        title="Eye diagram"
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
        <span className="text-muted">Mode: P25 C4FM</span>
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

      <div className="rounded border border-border bg-black p-2">
        <EyeDiagramChart eye={eye.samples} sps={eye.sps} />
      </div>

      <div className="text-[11px] text-muted">
        Live C4FM eye, folded over the symbol period (OP25's datascope). A
        healthy channel shows four open horizontal bands with clear gaps at
        the centre line (the decision instant); merged bands or a closed
        centre mean symbol-timing or SNR trouble. Dial the <em>Offset</em>{" "}
        onto a locked control/voice channel. CQPSK/LSM has no 4-level eye —
        use the Constellation panel for its complex constellation.
      </div>
    </div>
  );
}

function ConnPill({ state }: { state: ConnState }) {
  if (state === "open") return <span className="pill-ok">live</span>;
  if (state === "connecting") return <span className="pill-warn">connecting</span>;
  return <span className="pill-err">offline</span>;
}
