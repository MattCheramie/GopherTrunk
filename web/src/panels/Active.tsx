import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import { parentSerial } from "../api/spectrum";
import { writes } from "../api/write";
import { Column, DataTable } from "../components/DataTable";
import { ConfirmModal } from "../components/ConfirmModal";
import { DetailField, DetailModal } from "../components/DetailModal";
import { RIDLink } from "../components/RIDLink";
import { StaleIndicator } from "../components/ui/StaleIndicator";
import { PageHeader } from "../components/ui/PageHeader";
import type { ActiveCallDTO } from "../api/types";
import { formatP25Algorithm, formatP25KeyID } from "../api/p25Algorithm";
import { useDataPoll } from "../hooks/useDataPoll";
import {
  useLingeringActiveCalls,
  callDuration,
  type LingeringCall,
} from "../hooks/useLingeringActiveCalls";
import {
  selectCanMutate,
  selectClientConfig,
  useShared,
} from "../store/shared";

const POLL_INTERVAL_MS = 2_000;

// Active mirrors the TUI's Active Calls panel. The dashboard already
// surfaces a thumbnail; this panel gives the full call list with
// per-call detail, grant breakdown, and a duration ticker.
export function Active() {
  const cfg = useShared(selectClientConfig);
  const canMutate = useShared(selectCanMutate);
  const setError = useShared((s) => s.setError);
  const notify = useShared((s) => s.notify);
  const activeCalls = useShared((s) => s.activeCalls);
  const setActiveCalls = useShared((s) => s.setActiveCalls);
  const [selected, setSelected] = useState<LingeringCall | null>(null);
  const [confirmEnd, setConfirmEnd] = useState<ActiveCallDTO | null>(null);
  const [now, setNow] = useState(() => Date.now());

  // Keep just-ended calls on screen (frozen) for a few seconds so the log shows
  // a final "call duration" and an "ended" marker instead of the row
  // vanishing the instant the call drops from the poll.
  const rows = useLingeringActiveCalls(activeCalls);

  const { stale, lastUpdated, refresh } = useDataPoll({
    fetcher: () => api.activeCalls(cfg),
    onData: setActiveCalls,
    intervalMs: POLL_INTERVAL_MS,
    resetKey: cfg.baseURL,
  });

  async function endCall(call: ActiveCallDTO) {
    try {
      await writes.endCall(cfg, call.device_serial);
      // Optimistically refresh the active list so the UI doesn't show
      // the ended call for the next poll cycle.
      refresh();
      setSelected(null);
      notify("success", `Ended call on ${call.device_serial}`);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "end-call request failed");
      throw e;
    }
    setConfirmEnd(null);
  }

  // Tick once a second so the elapsed-time column updates even when
  // no API response has come back yet.
  useEffect(() => {
    const t = window.setInterval(() => setNow(Date.now()), 1_000);
    return () => window.clearInterval(t);
  }, []);

  const columns: Column<LingeringCall>[] = useMemo(
    () => [
      {
        key: "tg",
        header: "TG",
        render: (r) => (
          <span className="font-mono text-accent">{r.grant.group_id}</span>
        ),
        sort: (a, b) => a.grant.group_id - b.grant.group_id,
      },
      {
        key: "alpha",
        header: "Alpha tag",
        render: (r) => (
          <span className="font-medium">
            {r.talkgroup?.alpha_tag || <em className="text-muted">—</em>}
          </span>
        ),
        sort: (a, b) =>
          (a.talkgroup?.alpha_tag ?? "").localeCompare(
            b.talkgroup?.alpha_tag ?? "",
          ),
      },
      {
        key: "system",
        header: "System",
        render: (r) => <span className="text-xs">{r.grant.system}</span>,
        sort: (a, b) => a.grant.system.localeCompare(b.grant.system),
        className: "hidden md:table-cell",
        headerClassName: "hidden md:table-cell",
      },
      {
        // Who is keyed up — the transmitting radio. A trunking operator scans
        // this at a glance, so it belongs in the row, not just the modal.
        key: "source",
        header: "Source",
        render: (r) =>
          r.grant.source_id ? (
            <RIDLink rid={r.grant.source_id} className="font-mono text-xs text-accent hover:underline" />
          ) : (
            <span className="text-muted text-xs">—</span>
          ),
        sort: (a, b) => (a.grant.source_id ?? 0) - (b.grant.source_id ?? 0),
        className: "hidden md:table-cell",
        headerClassName: "hidden md:table-cell",
      },
      {
        key: "flags",
        header: "",
        render: (r) => (
          <div className="flex flex-wrap gap-1">
            {r.grant.encrypted && (
              <span
                className="pill-warn"
                title={
                  r.grant.algorithm_id != null
                    ? `${formatP25Algorithm(r.grant.algorithm_id)} key ${formatP25KeyID(r.grant.key_id ?? 0)}`
                    : "encryption metadata pending"
                }
              >
                {r.grant.algorithm_id != null
                  ? `enc ${formatP25Algorithm(r.grant.algorithm_id)}`
                  : "enc"}
              </span>
            )}
            {r.grant.emergency && <span className="pill-err">emerg</span>}
            {r.grant.data_call && <span className="pill">data</span>}
            {r._ended ? (
              <span
                className="pill"
                title="This call has ended; still shown briefly with its final duration before it clears from the log"
              >
                ended
              </span>
            ) : (
              r.following === false && (
                <span
                  className="pill"
                  title="Announced on the control channel; no voice tuner is following it (add voice_taps or a voice SDR to decode more calls at once)"
                >
                  untuned
                </span>
              )
            )}
          </div>
        ),
      },
      {
        key: "duration",
        header: "Duration",
        render: (r) => (
          <span className="font-mono text-xs tabular-nums">
            {callDuration(r.started_at, r._endMs, now)}
          </span>
        ),
        sort: (a, b) => a.started_at.localeCompare(b.started_at),
      },
      {
        key: "device",
        header: "Device",
        render: (r) => (
          <span className="font-mono text-xs text-muted">
            {r.following === false ? "—" : r.device_serial}
          </span>
        ),
        sort: (a, b) => a.device_serial.localeCompare(b.device_serial),
        className: "hidden lg:table-cell",
        headerClassName: "hidden lg:table-cell",
      },
    ],
    [now],
  );

  return (
    <div className="space-y-3">
      <PageHeader
        title="Active calls"
        actions={
          <>
            <StaleIndicator stale={stale} lastUpdated={lastUpdated} />
            <span className="text-xs text-muted">
              {activeCalls.length} in flight
            </span>
          </>
        }
      />

      <DataTable
        rows={rows}
        columns={columns}
        rowKey={(r) =>
          `${r.device_serial}-${r.grant.system}-${r.grant.group_id}-${r.grant.timeslot ?? 0}-${r.started_at}`
        }
        defaultSortKey="duration"
        defaultSortDirection="desc"
        searchable
        searchAccessor={(r) =>
          [r.grant.group_id, r.talkgroup?.alpha_tag, r.grant.system, r.device_serial]
            .filter(Boolean)
            .join(" ")
        }
        searchPlaceholder="Search active calls…"
        onRowClick={(r) => setSelected(r)}
        emptyMessage="No calls right now. Active grants show up here as soon as the daemon allocates a voice device."
      />

      {selected && (
        <DetailModal
          title={selected.talkgroup?.alpha_tag || `TG ${selected.grant.group_id}`}
          subtitle={`${selected.grant.system} · ${selected.grant.protocol}`}
          onClose={() => setSelected(null)}
        >
          <div className="grid grid-cols-2 gap-3">
            <DetailField label="TGID" mono value={selected.grant.group_id} />
            <DetailField
              label="Source"
              mono
              value={
                selected.grant.source_id ? (
                  <RIDLink rid={selected.grant.source_id} />
                ) : null
              }
            />
            <DetailField
              label="Frequency"
              mono
              value={formatHz(selected.grant.frequency_hz)}
            />
            {selected.grant.timeslot ? (
              <DetailField
                label="Timeslot"
                mono
                value={`TS${selected.grant.timeslot}`}
              />
            ) : null}
            <DetailField
              label="Channel"
              mono
              value={
                selected.grant.channel_number ?? selected.grant.channel_id ?? null
              }
            />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <DetailField
              label="Encrypted"
              value={selected.grant.encrypted ? "yes" : "no"}
            />
            <DetailField
              label="Emergency"
              value={selected.grant.emergency ? "yes" : "no"}
            />
            <DetailField
              label="Data"
              value={selected.grant.data_call ? "yes" : "no"}
            />
          </div>
          {selected.grant.encrypted && (
            <div className="grid grid-cols-2 gap-3">
              <DetailField
                label="Algorithm"
                mono
                value={
                  selected.grant.algorithm_id != null
                    ? formatP25Algorithm(selected.grant.algorithm_id)
                    : "pending"
                }
              />
              <DetailField
                label="Key ID"
                mono
                value={
                  selected.grant.key_id != null
                    ? formatP25KeyID(selected.grant.key_id)
                    : "pending"
                }
              />
            </div>
          )}
          <DetailField
            label="Device"
            mono
            value={selected.device_serial}
          />
          <div className="grid grid-cols-2 gap-3">
            <DetailField
              label="Started"
              mono
              value={selected.started_at.replace("T", " ").replace(/\..*$/, "")}
            />
            <DetailField
              label="Duration"
              mono
              value={callDuration(selected.started_at, selected._endMs, now)}
            />
          </div>
          {selected.talkgroup && (
            <div className="grid grid-cols-2 gap-3 pt-2 border-t border-panel">
              <DetailField label="Tag" value={selected.talkgroup.tag} />
              <DetailField label="Group" value={selected.talkgroup.group} />
              <DetailField
                label="Priority"
                mono
                value={selected.talkgroup.priority ?? null}
              />
              <DetailField label="Mode" value={selected.talkgroup.mode} />
            </div>
          )}
          {selected.following !== false && selected.device_serial && (
            <Link
              to={`/plots/constellation?device=${encodeURIComponent(parentSerial(selected.device_serial))}`}
              className="inline-block text-sm text-accent hover:underline pt-1"
            >
              Signal detail — open the scopes on this SDR →
            </Link>
          )}
          {selected.following === false ? (
            <p className="text-xs text-muted pt-2">
              This call is only known from the control channel — no voice tuner
              is following it. Add <code>voice_taps</code> to a wideband device,
              or another <code>role: voice</code> SDR, to decode more calls at
              once.
            </p>
          ) : canMutate ? (
            <div className="pt-2">
              <button
                className="btn-danger w-full"
                onClick={() => setConfirmEnd(selected)}
              >
                End this call
              </button>
            </div>
          ) : (
            <p className="text-xs text-muted pt-2">
              Enable write mode in Settings to allow ending calls from
              this browser.
            </p>
          )}
        </DetailModal>
      )}

      {confirmEnd && (
        <ConfirmModal
          title={`End call on ${confirmEnd.device_serial}?`}
          message={`This releases the voice device. The recorder closes the WAV cleanly.`}
          confirmLabel="End call"
          destructive
          onConfirm={() => endCall(confirmEnd)}
          onCancel={() => setConfirmEnd(null)}
        />
      )}
    </div>
  );
}

function formatHz(hz: number): string {
  if (!Number.isFinite(hz)) return "—";
  if (hz >= 1_000_000) return `${(hz / 1_000_000).toFixed(4)} MHz`;
  if (hz >= 1_000) return `${(hz / 1_000).toFixed(3)} kHz`;
  return `${hz} Hz`;
}
