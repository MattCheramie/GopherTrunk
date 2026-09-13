import { useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { api } from "../api/client";
import { writes } from "../api/write";
import { Column, DataTable } from "../components/DataTable";
import { ConfirmModal } from "../components/ConfirmModal";
import { DetailField, DetailModal } from "../components/DetailModal";
import { RIDLink } from "../components/RIDLink";
import { RecordingPlayer } from "../components/RecordingPlayer";
import { CallHealth } from "../components/SignalHealth";
import { PageHeader } from "../components/ui/PageHeader";
import type { CallRow } from "../api/types";
import { formatP25Algorithm, formatP25KeyID } from "../api/p25Algorithm";
import { formatLocalDateTime } from "../lib/formatTime";
import {
  selectCanMutate,
  selectClientConfig,
  useShared,
} from "../store/shared";

// HISTORY_POLL_MS is the fallback refresh cadence while the tab is visible.
// The primary trigger is the live event feed (a call.end below); the poll
// covers a dropped SSE/WS connection so the table can never quietly freeze.
const HISTORY_POLL_MS = 30_000;
// HISTORY_REFRESH_DELAY_MS is how long after a call.end the refetch waits, so
// the recorder's call.complete (has_recording) has landed on the row first.
const HISTORY_REFRESH_DELAY_MS = 1_500;

type HistoryFilter = {
  limit?: number;
  system?: string;
  group_id?: number;
  source_id?: number;
};

// selectLastCallEnd picks the timestamp of the newest call.end in the live
// feed — a primitive, so the zustand subscription only re-renders on a new
// call ending, not on every event.
function selectLastCallEnd(s: { events: { kind: string; timestamp: string }[] }) {
  for (let i = s.events.length - 1; i >= 0; i--) {
    if (s.events[i].kind === "call.end") return s.events[i].timestamp;
  }
  return null;
}

// History reads /api/v1/calls/history with the same filter shape the
// daemon accepts: limit, system, group_id, source_id. The response envelope is
// {"calls":[...]} — reading any other key yields a silently empty table.
//
// The table is LIVE: it refetches shortly after every call.end in the event
// feed and on a slow visible-tab poll. It used to fetch exactly once per
// filter change, so the operator had to reload the page to see new calls
// (the "history updates too slow / stops reflecting latest calls" report —
// though most of that report was the UTC rendering, see formatLocalDateTime).
export function History() {
  const cfg = useShared(selectClientConfig);
  const canMutate = useShared(selectCanMutate);
  const setGlobalError = useShared((s) => s.setError);
  const notify = useShared((s) => s.notify);
  const lastCallEnd = useShared(selectLastCallEnd);
  // Bumped to refetch with the same filter (live refresh / poll).
  const [refreshTick, setRefreshTick] = useState(0);

  const [rows, setRows] = useState<CallRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirmSweep, setConfirmSweep] = useState(false);

  // Cross-links from Talkgroups / Systems / Radio IDs seed the filter via
  // query params (?system=, ?group_id=, ?source_id=) so "view this TG's /
  // system's / radio's calls" lands here pre-filtered — the cross-selection
  // trunking operators expect.
  const [searchParams] = useSearchParams();
  const initSystem = searchParams.get("system") ?? "";
  const initGroup = searchParams.get("group_id") ?? "";
  const initSource = searchParams.get("source_id") ?? "";

  // Form fields kept separate from the "submitted" filter object so
  // typing into the inputs doesn't trigger a fetch on every keystroke.
  const [limitInput, setLimitInput] = useState("200");
  const [systemInput, setSystemInput] = useState(initSystem);
  const [groupInput, setGroupInput] = useState(initGroup);
  const [sourceInput, setSourceInput] = useState(initSource);
  const [filter, setFilter] = useState<HistoryFilter>(() => {
    const f: HistoryFilter = { limit: 200 };
    if (initSystem) f.system = initSystem;
    const g = parseInt(initGroup, 10);
    if (Number.isFinite(g)) f.group_id = g;
    const src = parseInt(initSource, 10);
    if (Number.isFinite(src)) f.source_id = src;
    return f;
  });

  const [selected, setSelected] = useState<CallRow | null>(null);

  // A refetch with the SAME filter (live refresh / poll) is silent: the
  // rows stay on screen and the header's "loading…" doesn't flash every
  // 30 s. Only a filter change shows the loading state.
  const fetchedFilter = useRef<HistoryFilter | null>(null);
  useEffect(() => {
    let cancel = false;
    const silent = fetchedFilter.current === filter;
    fetchedFilter.current = filter;
    if (!silent) setLoading(true);
    setError(null);
    api
      .history(cfg, filter)
      .then((data) => {
        if (cancel) return;
        setRows(data);
      })
      .catch((e: unknown) => {
        if (cancel) return;
        setError(e instanceof Error ? e.message : "history fetch failed");
      })
      .finally(() => {
        if (!cancel) setLoading(false);
      });
    return () => {
      cancel = true;
    };
  }, [cfg, filter, refreshTick]);

  // Live refresh: a call.end in the event feed means a new row (or a row that
  // just gained its duration / recording). Debounced past the recorder's
  // call.complete so has_recording is right on the first paint. The feed
  // usually already holds a call.end when the panel mounts — that one is not
  // news, skip it.
  const seenCallEnd = useRef(lastCallEnd);
  useEffect(() => {
    if (lastCallEnd === null || lastCallEnd === seenCallEnd.current) return;
    seenCallEnd.current = lastCallEnd;
    const t = window.setTimeout(
      () => setRefreshTick((n) => n + 1),
      HISTORY_REFRESH_DELAY_MS,
    );
    return () => window.clearTimeout(t);
  }, [lastCallEnd]);

  // Fallback poll while the tab is visible (covers a dropped event stream).
  useEffect(() => {
    const t = window.setInterval(() => {
      if (document.visibilityState === "visible") {
        setRefreshTick((n) => n + 1);
      }
    }, HISTORY_POLL_MS);
    return () => window.clearInterval(t);
  }, []);

  function applyFilter(e: React.FormEvent) {
    e.preventDefault();
    const next: HistoryFilter = {};
    const lim = parseInt(limitInput, 10);
    if (Number.isFinite(lim) && lim > 0) next.limit = lim;
    if (systemInput.trim()) next.system = systemInput.trim();
    const gid = parseInt(groupInput, 10);
    if (Number.isFinite(gid)) next.group_id = gid;
    const src = parseInt(sourceInput, 10);
    if (Number.isFinite(src)) next.source_id = src;
    setFilter(next);
  }

  function clearFilter() {
    setLimitInput("200");
    setSystemInput("");
    setGroupInput("");
    setSourceInput("");
    setFilter({ limit: 200 });
  }

  const columns: Column<CallRow>[] = useMemo(
    () => [
      {
        key: "started",
        header: "Started",
        render: (r) => (
          <span className="font-mono text-xs text-muted whitespace-nowrap">
            {formatLocalDateTime(r.started_at)}
          </span>
        ),
        sort: (a, b) => a.started_at.localeCompare(b.started_at),
      },
      {
        key: "tg",
        header: "TG",
        render: (r) =>
          r.individual ? (
            // Individual (unit-to-unit) call: group_id is a target radio's SSI,
            // not a talkgroup — label it as a radio ID so it is never read as a TG.
            <span className="font-mono text-muted" title="Individual (unit-to-unit) call — this is a radio ID, not a talkgroup">
              →{r.group_id}
            </span>
          ) : (
            <span className="font-mono text-accent">{r.group_id}</span>
          ),
        sort: (a, b) => a.group_id - b.group_id,
      },
      {
        key: "alpha",
        header: "Alpha tag",
        render: (r) => (
          <span className="font-medium">
            {r.talkgroup_alpha ?? <em className="text-muted">—</em>}
          </span>
        ),
        sort: (a, b) =>
          (a.talkgroup_alpha ?? "").localeCompare(b.talkgroup_alpha ?? ""),
      },
      {
        // The transmitting radio, alias-resolved when known — "who was
        // talking", scannable in the row rather than hidden in the modal.
        key: "source",
        header: "Source",
        render: (r) =>
          r.source_id ? (
            <RIDLink
              rid={r.source_id}
              alias={r.source_alpha}
              className="text-xs text-accent hover:underline"
            />
          ) : (
            <span className="text-muted text-xs">—</span>
          ),
        sort: (a, b) =>
          (a.source_alpha ?? String(a.source_id ?? "")).localeCompare(
            b.source_alpha ?? String(b.source_id ?? ""),
          ),
        className: "hidden md:table-cell",
        headerClassName: "hidden md:table-cell",
      },
      {
        key: "system",
        header: "System",
        render: (r) => <span className="text-xs">{r.system}</span>,
        sort: (a, b) => a.system.localeCompare(b.system),
        className: "hidden lg:table-cell",
        headerClassName: "hidden lg:table-cell",
      },
      {
        // Per-call decode health from the figures the daemon already stamps on
        // the call (SNR / EVM / level). Lets an operator judge a recording's
        // quality without opening a scope.
        key: "signal",
        header: "Signal",
        render: (r) => (
          <CallHealth
            evmPct={r.evm_pct}
            snrDb={r.snr_db}
            signalDbfs={r.signal_dbfs}
          />
        ),
        className: "hidden lg:table-cell",
        headerClassName: "hidden lg:table-cell",
      },
      {
        key: "duration",
        header: "Duration",
        render: (r) => (
          <span className="font-mono text-xs tabular-nums">
            {formatDuration(r.duration_ms)}
          </span>
        ),
        sort: (a, b) => (a.duration_ms ?? 0) - (b.duration_ms ?? 0),
      },
      {
        key: "flags",
        header: "",
        render: (r) => (
          <div className="flex flex-wrap gap-1">
            {r.encrypted && (
              <span
                className="pill-warn"
                title={
                  r.algorithm_id != null && r.algorithm_id !== 0
                    ? `${formatP25Algorithm(r.algorithm_id)} key ${formatP25KeyID(r.key_id ?? 0)}`
                    : "encryption metadata not captured"
                }
              >
                {r.algorithm_id != null && r.algorithm_id !== 0
                  ? `enc ${formatP25Algorithm(r.algorithm_id)}`
                  : "enc"}
              </span>
            )}
            {r.emergency && <span className="pill-err">emerg</span>}
            {r.data_call && <span className="pill">data</span>}
            {r.individual && (
              <span className="pill" title="Unit-to-unit / private call (not a talkgroup)">
                individual
              </span>
            )}
            {r.has_recording && (
              <span className="pill-ok" title="A recording is available — open the call to play it">
                ▶ rec
              </span>
            )}
          </div>
        ),
      },
    ],
    [],
  );

  return (
    <div className="space-y-3">
      <PageHeader
        title="Call history"
        actions={
          <>
            <span className="text-xs text-muted">
              {loading
                ? "loading…"
                : `${rows.length} row${rows.length === 1 ? "" : "s"}`}
            </span>
            {canMutate && (
              <button
                className="btn-ghost text-xs"
                onClick={() => setConfirmSweep(true)}
              >
                Sweep retention
              </button>
            )}
          </>
        }
      />

      <form
        onSubmit={applyFilter}
        className="panel p-3 grid grid-cols-2 sm:grid-cols-5 gap-2 items-end"
      >
        <label className="text-xs space-y-1">
          <span className="text-muted uppercase tracking-wider">Limit</span>
          <input
            type="number"
            min={1}
            max={5000}
            className="input w-full"
            value={limitInput}
            onChange={(e) => setLimitInput(e.target.value)}
          />
        </label>
        <label className="text-xs space-y-1">
          <span className="text-muted uppercase tracking-wider">System</span>
          <input
            type="text"
            className="input w-full"
            placeholder="name"
            value={systemInput}
            onChange={(e) => setSystemInput(e.target.value)}
          />
        </label>
        <label className="text-xs space-y-1">
          <span className="text-muted uppercase tracking-wider">Group ID</span>
          <input
            type="number"
            min={0}
            className="input w-full"
            placeholder="e.g. 1001"
            value={groupInput}
            onChange={(e) => setGroupInput(e.target.value)}
          />
        </label>
        <label className="text-xs space-y-1">
          <span className="text-muted uppercase tracking-wider">Source RID</span>
          <input
            type="number"
            min={0}
            className="input w-full"
            placeholder="radio id"
            value={sourceInput}
            onChange={(e) => setSourceInput(e.target.value)}
          />
        </label>
        <div className="flex gap-2 col-span-2 sm:col-span-1">
          <button type="submit" className="btn-primary flex-1">
            Apply
          </button>
          <button
            type="button"
            className="btn-ghost"
            onClick={clearFilter}
          >
            Clear
          </button>
        </div>
      </form>

      {error && (
        <p className="text-sm text-err" role="alert">
          {error}
        </p>
      )}

      <DataTable
        rows={rows}
        columns={columns}
        rowKey={(r) => String(r.id)}
        defaultSortKey="started"
        defaultSortDirection="desc"
        loading={loading}
        pageSize={50}
        searchable
        searchAccessor={(r) =>
          [r.group_id, r.talkgroup_alpha, r.system, r.source_id]
            .filter(Boolean)
            .join(" ")
        }
        searchPlaceholder="Search loaded calls…"
        onRowClick={(r) => setSelected(r)}
        emptyMessage="No calls in the daemon's call log for this filter."
      />

      {selected && (
        <DetailModal
          title={selected.talkgroup_alpha || `TG ${selected.group_id}`}
          subtitle={`${selected.system} · ${selected.protocol}`}
          onClose={() => setSelected(null)}
        >
          <div className="grid grid-cols-2 gap-3">
            <DetailField label="Row ID" mono value={selected.id} />
            <DetailField label="TGID" mono value={selected.group_id} />
            <DetailField
              label="Source"
              mono
              value={
                selected.source_id ? (
                  <RIDLink rid={selected.source_id} alias={selected.source_alpha} />
                ) : null
              }
            />
            <DetailField
              label="Frequency"
              mono
              value={formatHz(selected.frequency_hz)}
            />
          </div>
          {(selected.snr_db != null ||
            selected.evm_pct != null ||
            selected.signal_dbfs != null) && (
            <div>
              <p className="text-xs uppercase tracking-wider text-muted mb-1">
                Signal
              </p>
              <CallHealth
                evmPct={selected.evm_pct}
                snrDb={selected.snr_db}
                signalDbfs={selected.signal_dbfs}
              />
            </div>
          )}
          <div className="grid grid-cols-2 gap-3">
            <DetailField
              label="Started"
              mono
              value={formatLocalDateTime(selected.started_at)}
            />
            <DetailField
              label="Ended"
              mono
              value={
                selected.ended_at
                  ? formatLocalDateTime(selected.ended_at)
                  : null
              }
            />
            <DetailField
              label="Duration"
              mono
              value={formatDuration(selected.duration_ms)}
            />
            <DetailField label="End reason" value={selected.end_reason} />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <DetailField
              label="Encrypted"
              value={selected.encrypted ? "yes" : "no"}
            />
            <DetailField
              label="Emergency"
              value={selected.emergency ? "yes" : "no"}
            />
            <DetailField
              label="Data"
              value={selected.data_call ? "yes" : "no"}
            />
          </div>
          {selected.encrypted && (
            <div className="grid grid-cols-2 gap-3">
              <DetailField
                label="Algorithm"
                mono
                value={
                  selected.algorithm_id != null && selected.algorithm_id !== 0
                    ? formatP25Algorithm(selected.algorithm_id)
                    : "not captured"
                }
              />
              <DetailField
                label="Key ID"
                mono
                value={
                  selected.key_id != null && selected.key_id !== 0
                    ? formatP25KeyID(selected.key_id)
                    : "not captured"
                }
              />
            </div>
          )}
          <DetailField
            label="Device"
            mono
            value={selected.device_serial ?? null}
          />
          {selected.has_recording && (
            <div>
              <p className="text-xs uppercase tracking-wider text-muted mb-1">
                Recording
              </p>
              <RecordingPlayer cfg={cfg} callId={selected.id} />
            </div>
          )}
        </DetailModal>
      )}

      {confirmSweep && (
        <ConfirmModal
          title="Run retention sweep?"
          message="Apply the daemon's storage.retention_days policy now — older call rows and WAV files are deleted."
          confirmLabel="Sweep"
          destructive
          onConfirm={async () => {
            try {
              await writes.sweepRetention(cfg);
              // Refresh the current view so the user sees the result.
              const fresh = await api.history(cfg, filter);
              setRows(fresh);
              notify("success", "Retention sweep complete");
            } catch (e: unknown) {
              setGlobalError(
                e instanceof Error ? e.message : "retention sweep failed",
              );
              throw e;
            }
            setConfirmSweep(false);
          }}
          onCancel={() => setConfirmSweep(false)}
        />
      )}
    </div>
  );
}

function formatDuration(ms?: number): string {
  if (ms == null || !Number.isFinite(ms)) return "—";
  const seconds = Math.floor(ms / 1000);
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
}

function formatHz(hz: number): string {
  if (!Number.isFinite(hz)) return "—";
  if (hz >= 1_000_000) return `${(hz / 1_000_000).toFixed(4)} MHz`;
  if (hz >= 1_000) return `${(hz / 1_000).toFixed(3)} kHz`;
  return `${hz} Hz`;
}
