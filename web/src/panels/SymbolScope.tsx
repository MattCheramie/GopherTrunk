import { useEffect, useMemo, useRef, useState } from "react";
import { PageHeader } from "../components/ui/PageHeader";
import { useSearchParams } from "react-router-dom";
import {
  fetchSpectrumDevices,
  defaultSymbolDevice,
  initialDeviceSerial,
  parentSerial,
  type SpectrumDevice,
} from "../api/spectrum";
import {
  demodModeToProto,
  openSymbolStream,
  type SymbolFrame,
} from "../api/symbols";
import { selectClientConfig, useShared } from "../store/shared";
import { prefs } from "../store/prefs";
import { SymbolScopeChart } from "../components/SymbolScopeChart";
import { useActiveCallsPoll } from "../hooks/useActiveCallsPoll";
import { TuningControls } from "../components/TuningControls";

// Symbol scope panel — a live oscilloscope of the demodulated symbol
// stream, GopherTrunk's take on OP25's "Symbol" plot. The daemon runs a
// parallel decode (down-convert → protocol receiver) on the selected
// SDR and streams the pre-slicer soft waveform + sliced dibit decisions;
// we render a rolling window. For P25 C4FM a healthy channel reads as
// ~4 noisy bands; CQPSK streams the dibit decisions only.
//
// The offset / Hold / follow-active-call controls mirror the
// Constellation panel: dial the offset onto a locked control/voice
// channel (or let Hold-off follow the newest active call) to lift its
// symbols clear of the SDR centre DC spike.

const WINDOW_SYMBOLS = 2400; // rolling scope width

const PROTOS: { value: string; label: string }[] = [
  { value: "auto", label: "Auto" },
  { value: "p25-c4fm", label: "P25 C4FM" },
  { value: "p25-cqpsk", label: "P25 CQPSK" },
  { value: "tetra", label: "TETRA" },
  { value: "dmr", label: "DMR" },
];

type ConnState = "connecting" | "open" | "closed";

export function SymbolScope() {
  const cfg = useShared(selectClientConfig);
  const activeCalls = useShared((s) => s.activeCalls);
  // Keep the follow logic live: this panel reads activeCalls but, unlike
  // Active/Dashboard, didn't refresh it — so the view froze on a stale
  // call ("following call" forever, single freq) when opened directly.
  useActiveCallsPoll();
  const [devices, setDevices] = useState<SpectrumDevice[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  // A "signal detail" link from a call / scanner hit can name the SDR to
  // scope via ?device=; honoured on first load, else the panel default.
  const targetDevice = useSearchParams()[0].get("device");
  const [conn, setConn] = useState<ConnState>("closed");
  const [latest, setLatest] = useState<SymbolFrame | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [proto, setProto] = useState<string>(() => prefs.symbolScopeProto());
  const [offsetKHz, setOffsetKHz] = useState<number>(() =>
    prefs.symbolScopeOffsetKHz(),
  );
  const [hold, setHold] = useState<boolean>(() => prefs.symbolScopeHold());

  // Rolling window of recovered symbols, recreated per frame so the
  // chart sees fresh array references.
  const [window, setWindow] = useState<{ soft: number[]; dibits: number[] }>({
    soft: [],
    dibits: [],
  });
  const softRef = useRef<number[]>([]);
  const dibitRef = useRef<number[]>([]);

  // Discover SDRs (reuse the spectrum devices endpoint — same pool).
  useEffect(() => {
    let cancel = false;
    (async () => {
      try {
        const list = await fetchSpectrumDevices(cfg);
        if (cancel) return;
        setDevices(list);
        setError(null);
        if (selected == null) {
          const s = initialDeviceSerial(list, targetDevice, defaultSymbolDevice);
          if (s) setSelected(s);
        }
      } catch (e) {
        if (cancel) return;
        setError(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancel = true;
    };
  }, [cfg, selected, targetDevice]);

  const device = useMemo(
    () => devices.find((d) => d.serial === selected) ?? null,
    [devices, selected],
  );

  // Resolve the Mode selection: "auto" follows the modulation the
  // selected SDR's system is decoding (device.p25_modulation, C4FM when
  // unknown); an explicit choice is used verbatim.
  const effectiveProto = useMemo(
    () => (proto === "auto" ? demodModeToProto(device?.p25_modulation) : proto),
    [proto, device],
  );

  // Newest active call on the selected SDR → the offset (kHz) that
  // centres it. Followed live when Hold is off.
  const followOffsetKHz = useMemo(() => {
    if (!device) return null;
    const mine = activeCalls.filter(
      (c) => parentSerial(c.device_serial) === selected && !c.ended_at,
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

  // Persist view options.
  useEffect(() => {
    prefs.setSymbolScopeOffsetKHz(offsetKHz);
  }, [offsetKHz]);
  useEffect(() => {
    prefs.setSymbolScopeHold(hold);
  }, [hold]);
  useEffect(() => {
    prefs.setSymbolScopeProto(proto);
  }, [proto]);

  // Open the symbol stream. Re-subscribe on device / proto / offset
  // change so the server re-tunes and re-channelizes.
  useEffect(() => {
    if (!selected) return;
    softRef.current = [];
    dibitRef.current = [];
    setWindow({ soft: [], dibits: [] });
    setLatest(null);

    const stream = openSymbolStream(cfg, {
      serial: selected,
      proto: effectiveProto,
      offset: Math.round(clampedOffsetKHz * 1000),
      onFrame: (f) => {
        setLatest(f);
        const sb = softRef.current.concat(f.soft ?? []);
        const db = dibitRef.current.concat(f.dibits ?? []);
        // Keep soft aligned with dibits only when this frame carried a
        // full soft track; otherwise the scope shows the dibit rows.
        const aligned = sb.length === db.length;
        softRef.current = aligned
          ? sb.slice(Math.max(0, sb.length - WINDOW_SYMBOLS))
          : [];
        dibitRef.current = db.slice(Math.max(0, db.length - WINDOW_SYMBOLS));
        setWindow({ soft: softRef.current, dibits: dibitRef.current });
      },
      onStatus: setConn,
    });
    return () => stream.close();
  }, [cfg, selected, effectiveProto, clampedOffsetKHz]);

  // Centre comes from the selected device so the frequency view renders
  // immediately — symbol frames only arrive once the receiver decodes, so
  // gating the label on `latest` would leave it blank on a quiet channel.
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
    const soft = latest.soft && latest.soft.length > 0 ? "soft+dibits" : "dibits";
    return `${head} · ${latest.symbol_rate_hz.toFixed(0)} sym/s · ${soft}`;
  }, [latest, viewHz, clampedOffsetKHz]);

  return (
    <div className="space-y-3">
      <PageHeader
        title="Symbol scope"
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
          {proto === "auto" && (
            <span className="text-muted">
              ·&nbsp;{effectiveProto === "p25-c4fm" ? "C4FM" : "CQPSK"}
            </span>
          )}
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
          onHoldChange={(next) => {
            // Re-checking Hold parks the view on the control channel (a
            // stable, known reference) rather than freezing on a call
            // frequency that may already be stale.
            if (next && controlOffsetKHz != null) setOffsetKHz(controlOffsetKHz);
            setHold(next);
          }}
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
        <SymbolScopeChart soft={window.soft} dibits={window.dibits} />
      </div>

      <div className="text-[11px] text-muted">
        Live demodulated symbols off the selected SDR. P25 C4FM shows the
        pre-slicer soft waveform (~4 bands for a healthy channel) with
        rails at each decided level; CQPSK shows the sliced dibit rows.
        Dial the <em>Offset</em> onto a locked channel to lift it off the
        centre DC spike.
      </div>
    </div>
  );
}

function ConnPill({ state }: { state: ConnState }) {
  if (state === "open") return <span className="pill-ok">live</span>;
  if (state === "connecting") return <span className="pill-warn">connecting</span>;
  return <span className="pill-err">offline</span>;
}
