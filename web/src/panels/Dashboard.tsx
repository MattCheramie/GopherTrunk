import { api } from "../api/client";
import { PageHeader } from "../components/ui/PageHeader";
import { Badge } from "../components/ui/Badge";
import { useDataPoll } from "../hooks/useDataPoll";
import { selectClientConfig, useShared } from "../store/shared";

// Dashboard: top-line counts + a peek at the last few events. The
// fuller dashboard (with charts) is a follow-up PR; this one
// proves the live-update wiring works end-to-end.
export function Dashboard() {
  const cfg = useShared(selectClientConfig);
  const setHealth = useShared((s) => s.setHealth);
  const setActiveCalls = useShared((s) => s.setActiveCalls);
  const setDevices = useShared((s) => s.setDevices);
  const setAudio = useShared((s) => s.setAudio);

  const health = useShared((s) => s.health);
  const activeCalls = useShared((s) => s.activeCalls);
  const devices = useShared((s) => s.devices);
  const audio = useShared((s) => s.audio);
  const events = useShared((s) => s.events);
  const wsStatus = useShared((s) => s.wsStatus);

  useDataPoll({
    fetcher: async () => {
      const [h, calls, devs, aud] = await Promise.allSettled([
        api.health(cfg),
        api.activeCalls(cfg),
        api.devices(cfg),
        api.audio(cfg),
      ]);
      return {
        health: h.status === "fulfilled" ? h.value : null,
        calls: calls.status === "fulfilled" ? calls.value : null,
        devices: devs.status === "fulfilled" ? devs.value : null,
        audio: aud.status === "fulfilled" ? aud.value : null,
      };
    },
    onData: (d) => {
      if (d.health) setHealth(d.health);
      if (d.calls) setActiveCalls(d.calls);
      if (d.devices) setDevices(d.devices);
      if (d.audio) setAudio(d.audio);
    },
    intervalMs: 3_000,
    resetKey: cfg.baseURL,
  });

  return (
    <div className="space-y-4">
      <PageHeader
        title="Dashboard"
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

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        <StatCard label="Active calls" value={activeCalls.length} />
        <StatCard label="Devices attached" value={devices.length} />
        <StatCard
          label="Audio backend"
          value={
            audio
              ? audio.backend_enabled
                ? `${(audio.sample_rate / 1000).toFixed(1)} kHz`
                : "off"
              : "—"
          }
        />
        <StatCard
          label="Daemon"
          value={health?.status ?? "unknown"}
          ok={health?.status === "ok"}
        />
      </div>

      <section className="panel p-4">
        <h3 className="panel-title mb-2">Recent events</h3>
        {events.length === 0 ? (
          <p className="text-muted text-sm">
            No events yet. The stream connects automatically.
          </p>
        ) : (
          <ul className="space-y-1 text-xs font-mono max-h-72 overflow-auto">
            {events
              .slice(-50)
              .reverse()
              .map((ev, i) => (
                <li key={`${ev.timestamp}-${i}`} className="flex gap-3">
                  <span className="text-muted shrink-0">
                    {ev.timestamp.replace("T", " ").replace(/\..*$/, "")}
                  </span>
                  <span className="text-accent shrink-0">{ev.kind}</span>
                </li>
              ))}
          </ul>
        )}
      </section>

      <section className="panel p-4">
        <h3 className="panel-title mb-2">Active calls</h3>
        {activeCalls.length === 0 ? (
          <p className="text-muted text-sm">No calls right now.</p>
        ) : (
          <ul className="divide-y divide-panel">
            {activeCalls.map((c) => (
              <li
                key={`${c.device_serial}-${c.started_at}`}
                className="py-2 flex items-baseline gap-3"
              >
                <span className="font-mono text-accent">
                  TG {c.grant.group_id}
                </span>
                <span className="text-sm">
                  {c.talkgroup?.alpha_tag ?? c.grant.system}
                </span>
                <span className="ml-auto text-xs text-muted">
                  {c.device_serial}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
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
      <p
        className={
          ok === false ? "stat-value text-err" : "stat-value"
        }
      >
        {value}
      </p>
    </div>
  );
}
