import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../api/client";
import { PageHeader } from "../components/ui/PageHeader";
import { Badge } from "../components/ui/Badge";
import { RIDLink } from "../components/RIDLink";
import { SignalSummary } from "../components/SignalHealth";
import { useDataPoll } from "../hooks/useDataPoll";
import { useSignalQuality } from "../hooks/useSignalQuality";
import { useLockedSystemSignal } from "../hooks/useLockedSystemSignal";
import {
  useLingeringActiveCalls,
  callDuration,
  liveCallCount,
} from "../hooks/useLingeringActiveCalls";
import { selectClientConfig, useShared } from "../store/shared";
import { groupEvents } from "../lib/groupEvents";
import type { EventDTO } from "../api/types";

// DIGEST_EXCLUDE names the high-frequency status/diagnostic event kinds that
// don't belong in the human "what happened" digest: a P25 control channel
// rebroadcasts site.update many times a second (and channel.power ticks once a
// second per active channel), which buried the actual grants/affiliations under
// a wall of repeats. They're still available in full on the CC Activity, Sites
// and Signal panels — this only trims the dashboard summary. (TETRA never
// spammed the feed because its site publish is edge-triggered; the P25 backend
// now edge-triggers too, but the feed is a summary either way.)
const DIGEST_EXCLUDE = new Set(["site.update", "channel.power"]);

// Dashboard — the operations landing. Trunking scanners lead with the two
// core tasks (scan / hunt) and a live "who's talking now" roster, not a
// spectrum or a wall of stats. This mirrors that: prominent Scan/Hunt entry
// points, the active-call roster as the hero, top-line health, and a
// recent-activity feed that actually says what happened.
export function Dashboard() {
  const cfg = useShared(selectClientConfig);
  const navigate = useNavigate();
  const setHealth = useShared((s) => s.setHealth);
  const setActiveCalls = useShared((s) => s.setActiveCalls);
  const setDevices = useShared((s) => s.setDevices);
  const setSystems = useShared((s) => s.setSystems);

  const health = useShared((s) => s.health);
  const activeCalls = useShared((s) => s.activeCalls);
  const devices = useShared((s) => s.devices);
  const systems = useShared((s) => s.systems);
  const events = useShared((s) => s.events);
  const wsStatus = useShared((s) => s.wsStatus);

  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const t = window.setInterval(() => setNow(Date.now()), 1_000);
    return () => window.clearInterval(t);
  }, []);

  // Signal health belongs on the operations landing (we're monitoring a locked
  // system), not buried in the Scanner/hunter tab. Show both axes: the locked
  // system's decode health (GET /api/v1/scanner) and the live symbol SNR/eye.
  const { quality: symbolQuality, status: symbolStatus } = useSignalQuality();
  const lockedSignal = useLockedSystemSignal();

  // Freeze just-ended calls briefly so the roster shows a final duration +
  // "ended" marker rather than the row blinking out.
  const callRows = useLingeringActiveCalls(activeCalls);

  useDataPoll({
    fetcher: async () => {
      const [h, calls, devs, sys] = await Promise.allSettled([
        api.health(cfg),
        api.activeCalls(cfg),
        api.devices(cfg),
        api.systems(cfg),
      ]);
      return {
        health: h.status === "fulfilled" ? h.value : null,
        calls: calls.status === "fulfilled" ? calls.value : null,
        devices: devs.status === "fulfilled" ? devs.value : null,
        systems: sys.status === "fulfilled" ? sys.value : null,
      };
    },
    onData: (d) => {
      if (d.health) setHealth(d.health);
      if (d.calls) setActiveCalls(d.calls);
      if (d.devices) setDevices(d.devices);
      if (d.systems) setSystems(d.systems);
    },
    intervalMs: 3_000,
    resetKey: cfg.baseURL,
  });

  return (
    <div className="space-y-4">
      <PageHeader
        title="Operations"
        actions={
          <Badge
            tone={
              wsStatus === "open"
                ? "ok"
                : wsStatus === "connecting"
                  ? "warn"
                  : "err"
            }
          >
            {wsStatus === "open"
              ? "Live"
              : wsStatus === "connecting"
                ? "Connecting"
                : "Offline"}
          </Badge>
        }
      />

      {/* The two core trunking workflows, front and center. */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <TaskCard
          to="/scanner"
          icon="≋"
          title="Scan known systems"
          desc="Follow talkgroups on your configured trunked and conventional systems."
        />
        <TaskCard
          to="/hunt"
          icon="🔍"
          title="Hunt for new systems"
          desc="Discover, identify and map unknown control channels and sites."
        />
      </div>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        <StatCard label="Active calls" value={liveCallCount(callRows)} />
        <StatCard label="Systems" value={systems.length} />
        <StatCard label="Devices attached" value={devices.length} />
        <StatCard
          label="Daemon"
          value={health?.status ?? "unknown"}
          ok={health?.status === "ok"}
        />
      </div>

      {/* Signal health for the locked system — both axes, labelled. */}
      <SignalSummary
        decodeQuality={
          lockedSignal?.has_decode_health ? lockedSignal.decode_quality : null
        }
        offsetHz={lockedSignal?.carrier_offset_hz}
        signalDbfs={lockedSignal?.has_signal ? lockedSignal.signal_dbfs : null}
        symbolQuality={symbolQuality}
        status={symbolStatus}
      />

      {/* Hero: who's talking right now. */}
      <section className="panel p-4">
        <div className="flex items-baseline justify-between mb-2">
          <h3 className="panel-title">Active calls</h3>
          <Link to="/active" className="text-xs text-accent hover:underline">
            Open Active →
          </Link>
        </div>
        {callRows.length === 0 ? (
          <p className="text-muted text-sm">
            No calls right now. Grants appear here the moment the daemon
            allocates a voice device.
          </p>
        ) : (
          <ul className="divide-y divide-panel">
            {callRows.slice(0, 8).map((c) => (
              <li
                key={`${c.device_serial}-${c.grant.system}-${c.grant.group_id}-${c.started_at}`}
                className="py-2 flex items-center gap-3 cursor-pointer hover:bg-panel/40 -mx-2 px-2 rounded"
                onClick={() => navigate("/active")}
              >
                <span className="font-mono text-accent shrink-0">
                  TG {c.grant.group_id}
                </span>
                <span className="text-sm truncate">
                  {/* || not ??: auto-discovered talkgroups carry alpha_tag "" (present,
                      not null), which must still fall back to the system name. */}
                  {c.talkgroup?.alpha_tag || c.grant.system}
                </span>
                {c.grant.source_id ? (
                  <span className="text-xs text-muted shrink-0">
                    from <RIDLink rid={c.grant.source_id} className="text-accent hover:underline" />
                  </span>
                ) : null}
                <span className="ml-auto flex items-center gap-1 shrink-0">
                  {c.grant.encrypted && <span className="pill-warn">enc</span>}
                  {c.grant.emergency && <span className="pill-err">emerg</span>}
                  {c._ended ? (
                    <span className="pill" title="This call has ended; still shown briefly with its final duration before it clears">
                      ended
                    </span>
                  ) : (
                    c.following === false && (
                      <span className="pill" title="Announced on the control channel, but no voice tuner is following it — no audio is being recorded. Add voice_taps or a voice SDR to decode more calls at once.">
                        untuned
                      </span>
                    )
                  )}
                  <span className="font-mono text-xs tabular-nums text-muted">
                    {callDuration(c.started_at, c._endMs, now)}
                  </span>
                </span>
              </li>
            ))}
          </ul>
        )}
        {callRows.length > 8 && (
          <p className="text-xs text-muted mt-2">
            +{callRows.length - 8} more — open Active for the full list.
          </p>
        )}
      </section>

      {/* Recent activity — actually says what happened, not just the kind. */}
      <section className="panel p-4">
        <div className="flex items-baseline justify-between mb-2">
          <h3 className="panel-title">Recent activity</h3>
          <Link to="/cc" className="text-xs text-accent hover:underline">
            Open CC Activity →
          </Link>
        </div>
        {events.length === 0 ? (
          <p className="text-muted text-sm">
            No events yet. The stream connects automatically.
          </p>
        ) : (
          <ul className="space-y-1 text-xs font-mono max-h-72 overflow-auto">
            {/*
              A digest, not the raw firehose: drop the high-frequency status
              kinds (DIGEST_EXCLUDE) and collapse identical repeats (e.g. a
              grant re-sent every few seconds) into one row with an ×N count via
              groupEvents — the same de-dup CC Activity uses. Then order by the
              event's own wall-clock timestamp, newest first (NOT store arrival
              order): the events ring is appended in arrival order, so a late-
              arriving event carrying an older timestamp (e.g. an audio.state
              re-push) would float to the top under a plain reverse().
            */}
            {groupEvents(events.filter((ev) => !DIGEST_EXCLUDE.has(ev.kind)))
              .sort((a, b) => b.lastTimestamp.localeCompare(a.lastTimestamp))
              .slice(0, 40)
              .map((g, i) => (
                <li key={`${g.lastTimestamp}-${i}`} className="flex gap-3">
                  <span className="text-muted shrink-0">
                    {g.lastTimestamp.replace("T", " ").replace(/\..*$/, "")}
                  </span>
                  <span className="text-accent shrink-0 w-28 truncate">
                    {g.event.kind}
                    {g.count > 1 ? (
                      <span className="ml-1 text-muted">×{g.count}</span>
                    ) : null}
                  </span>
                  <span className="truncate text-fg/80">
                    {summarizeEvent(g.event)}
                  </span>
                </li>
              ))}
          </ul>
        )}
      </section>
    </div>
  );
}

// summarizeEvent renders a compact one-line summary from an event payload —
// TG / source radio / system / frequency for the common trunking kinds — so
// the feed says what happened instead of only naming the event kind. Reads
// defensively (payload is typed unknown on the wire).
function summarizeEvent(ev: EventDTO): string {
  const p = ev.payload;
  if (!p || typeof p !== "object") return "";
  const o = p as Record<string, unknown>;
  const parts: string[] = [];
  const tg = o.group_id ?? o.talkgroup;
  if (typeof tg === "number") parts.push(`TG ${tg}`);
  const src = o.source_id ?? o.source ?? o.radio_id;
  if (typeof src === "number" && src !== 0) parts.push(`← ${src}`);
  if (typeof o.system === "string" && o.system) parts.push(o.system);
  const freq = o.frequency_hz ?? o.freq_hz ?? o.frequency;
  if (typeof freq === "number" && freq > 0)
    parts.push(`${(freq / 1e6).toFixed(4)} MHz`);
  return parts.join(" · ");
}

function TaskCard({
  to,
  icon,
  title,
  desc,
}: {
  to: string;
  icon: string;
  title: string;
  desc: string;
}) {
  return (
    <Link
      to={to}
      className="panel p-4 flex items-start gap-3 hover:bg-panel/50 transition-colors"
    >
      <span className="text-2xl leading-none shrink-0" aria-hidden>
        {icon}
      </span>
      <span>
        <span className="block font-semibold">{title}</span>
        <span className="block text-sm text-muted">{desc}</span>
      </span>
    </Link>
  );
}

function StatCard({
  label,
  value,
  ok,
}: {
  label: string;
  value: string | number;
  ok?: boolean;
}) {
  return (
    <div className="panel p-4">
      <p className="panel-title">{label}</p>
      <p className={ok === false ? "stat-value text-err" : "stat-value"}>
        {value}
      </p>
    </div>
  );
}

