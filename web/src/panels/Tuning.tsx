import {
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Title,
  Tooltip,
} from "chart.js";
import { useEffect, useMemo, useRef, useState } from "react";
import { Line } from "react-chartjs-2";
import { PageHeader } from "../components/ui/PageHeader";
import { useSearchParams } from "react-router-dom";
import {
  fetchSpectrumDevices,
  defaultSymbolDevice,
  initialDeviceSerial,
  parentSerial,
  type SpectrumDevice,
} from "../api/spectrum";
import { openSymbolStream, type SymbolFrame } from "../api/symbols";
import { selectClientConfig, useShared } from "../store/shared";
import { prefs } from "../store/prefs";
import { useActiveCallsPoll } from "../hooks/useActiveCallsPoll";
import { TuningControls } from "../components/TuningControls";

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
);

// Tuning panel — receiver-state meters off the live demod: GopherTrunk's
// take on OP25's Mixer / Tuner (FLL) tabs. The daemon runs a parallel P25
// receiver on the selected channel and stamps its loop state onto every
// symbol frame; here we trend the residual carrier-frequency error and
// surface the AGC, symbol-clock and (CQPSK) equalizer state. A locked,
// healthy receiver settles the carrier error toward 0 Hz and holds the
// AGC / clock steady.

const HISTORY_POINTS = 240; // ~12 s at the ~50 ms frame cadence

const PROTOS: { value: string; label: string }[] = [
  { value: "p25-c4fm", label: "P25 C4FM" },
  { value: "p25-cqpsk", label: "P25 CQPSK" },
  { value: "tetra", label: "TETRA" },
  { value: "dmr", label: "DMR" },
  { value: "nxdn", label: "NXDN" },
  { value: "p25-phase2", label: "P25 Phase 2" },
];

// Mirrors SocketStatus from api/reconnectingSocket. "gone" is terminal: the
// stream stopped retrying because the device is not coming back.
type ConnState = "connecting" | "open" | "closed" | "gone";

export function Tuning() {
  const cfg = useShared(selectClientConfig);
  const activeCalls = useShared((s) => s.activeCalls);
  // Keep the follow logic live: this panel reads activeCalls but, unlike
  // Active/Dashboard, didn't refresh it — so the view froze on a stale
  // call ("following call" forever, single freq) when opened directly.
  useActiveCallsPoll();
  const [devices, setDevices] = useState<SpectrumDevice[]>([]);
  // Bumped when an open stream gives up on the selected serial, forcing a
  // device re-enumeration. Without it the selection was set once and never
  // reconciled, so after a daemon restart with different hardware the panel
  // kept asking for a device that no longer existed.
  const [deviceEpoch, setDeviceEpoch] = useState(0);
  const [selected, setSelected] = useState<string | null>(null);
  // A "signal detail" link from a call / scanner hit can name the SDR to
  // scope via ?device=; honoured on first load, else the panel default.
  const targetDevice = useSearchParams()[0].get("device");
  const [conn, setConn] = useState<ConnState>("closed");
  const [latest, setLatest] = useState<SymbolFrame | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [proto, setProto] = useState<string>(() => prefs.tuningProto());
  const [offsetKHz, setOffsetKHz] = useState<number>(() =>
    prefs.tuningOffsetKHz(),
  );
  const [hold, setHold] = useState<boolean>(() => prefs.tuningHold());

  // Rolling carrier-error history.
  const histRef = useRef<number[]>([]);
  const [hist, setHist] = useState<number[]>([]);

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
        if (selected == null || !list.some((d) => d.serial === selected)) {
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
  }, [cfg, selected, targetDevice, deviceEpoch]);

  const device = useMemo(
    () => devices.find((d) => d.serial === selected) ?? null,
    [devices, selected],
  );

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

  useEffect(() => {
    prefs.setTuningOffsetKHz(offsetKHz);
  }, [offsetKHz]);
  useEffect(() => {
    prefs.setTuningHold(hold);
  }, [hold]);
  useEffect(() => {
    prefs.setTuningProto(proto);
  }, [proto]);

  useEffect(() => {
    if (!selected) return;
    histRef.current = [];
    setHist([]);
    setLatest(null);

    const stream = openSymbolStream(cfg, {
      serial: selected,
      proto,
      offset: Math.round(clampedOffsetKHz * 1000),
      onFrame: (f) => {
        setLatest(f);
        const h = histRef.current;
        h.push(f.carrier_offset_hz ?? 0);
        if (h.length > HISTORY_POINTS) h.splice(0, h.length - HISTORY_POINTS);
        setHist(h.slice());
      },
      onStatus: setConn,
      onGone: () => setDeviceEpoch((n) => n + 1),
    });
    return () => stream.close();
  }, [cfg, selected, proto, clampedOffsetKHz]);

  const isCQPSK = proto === "p25-cqpsk";

  const centerHz = device?.center_hz ?? latest?.center_hz ?? null;
  const viewHz = centerHz != null ? centerHz + clampedOffsetKHz * 1000 : null;

  const tuningLabel = useMemo(() => {
    if (viewHz == null) return "";
    const off =
      clampedOffsetKHz === 0
        ? "centre"
        : `${clampedOffsetKHz >= 0 ? "+" : ""}${clampedOffsetKHz.toFixed(3).replace(/\.?0+$/, "")} kHz`;
    const head = `${(viewHz / 1e6).toFixed(4)} MHz (${off})`;
    if (!latest) return `${head} · waiting for receiver state…`;
    return head;
  }, [latest, viewHz, clampedOffsetKHz]);

  const chartData = useMemo(
    () => ({
      labels: hist.map((_, i) => String(i)),
      datasets: [
        {
          label: "Carrier error (Hz)",
          data: hist,
          borderColor: "rgb(56, 189, 248)",
          backgroundColor: "rgba(56, 189, 248, 0.15)",
          borderWidth: 1.5,
          pointRadius: 0,
          fill: true,
          tension: 0.2,
        },
      ],
    }),
    [hist],
  );

  const chartOptions = useMemo(
    () => ({
      responsive: true,
      maintainAspectRatio: false,
      animation: false as const,
      plugins: { legend: { display: false } },
      scales: {
        x: { display: false },
        y: {
          title: { display: true, text: "Hz" },
          grid: { color: "rgba(148,163,184,0.15)" },
          ticks: { color: "rgba(148,163,184,0.8)" },
        },
      },
    }),
    [],
  );

  return (
    <div className="space-y-3">
      <PageHeader
        title="Tuning"
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

      {/* Carrier-error trend (the FLL view). */}
      <div
        className="rounded border border-border bg-black p-2"
        style={{ height: 220 }}
        role="img"
        aria-label="Carrier-error (frequency-lock-loop) trend over time; the loop-state meters below give the current values as text."
      >
        <Line data={chartData} options={chartOptions} />
      </div>

      {/* Current loop state. */}
      <div className="grid grid-cols-2 sm:grid-cols-3 gap-2 text-xs">
        <Meter
          label="Carrier error"
          value={latest ? `${latest.carrier_offset_hz.toFixed(0)} Hz` : "—"}
        />
        <Meter
          label={isCQPSK ? "AGC gain" : "AGC level"}
          value={latest ? latest.agc_level.toFixed(3) : "—"}
          sub={
            !isCQPSK && latest && latest.agc_target > 0
              ? `target ${latest.agc_target.toFixed(3)}`
              : undefined
          }
        />
        <Meter
          label="Clock μ"
          value={latest ? latest.clock_mu.toFixed(3) : "—"}
          sub={latest && latest.clock_sps > 0 ? `sps ${latest.clock_sps.toFixed(2)}` : undefined}
        />
        {isCQPSK && (
          <Meter
            label="CMA error"
            value={latest ? latest.cma_error.toFixed(4) : "—"}
          />
        )}
      </div>

      <div className="text-[11px] text-muted">
        Live receiver state off the selected channel (OP25's Mixer / Tuner
        FLL). <strong>Carrier error</strong> is the loop's residual
        frequency-offset estimate — it should trend toward 0 Hz and hold
        once locked; a persistent offset is tuner PPM, a wandering one is a
        loop that hasn't acquired. AGC, clock μ and (CQPSK) CMA error should
        settle and stay steady. Dial the <em>Offset</em> onto a locked
        channel; pick the <em>Mode</em> that matches the site.
      </div>
    </div>
  );
}

function Meter({
  label,
  value,
  sub,
}: {
  label: string;
  value: string;
  sub?: string;
}) {
  return (
    <div className="rounded border border-border bg-surface px-3 py-2">
      <div className="text-muted">{label}</div>
      <div className="font-mono text-sm">{value}</div>
      {sub && <div className="text-[10px] text-muted">{sub}</div>}
    </div>
  );
}

function ConnPill({ state }: { state: ConnState }) {
  if (state === "open") return <span className="pill-ok">live</span>;
  if (state === "connecting") return <span className="pill-warn">connecting</span>;
  return <span className="pill-err">offline</span>;
}
