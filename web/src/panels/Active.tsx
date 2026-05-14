import { useEffect, useMemo, useState } from "react";
import { api } from "../api/client";
import { writes } from "../api/write";
import { Column, DataTable } from "../components/DataTable";
import { ConfirmModal } from "../components/ConfirmModal";
import { DetailField, DetailModal } from "../components/DetailModal";
import type { ActiveCallDTO } from "../api/types";
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
  const activeCalls = useShared((s) => s.activeCalls);
  const setActiveCalls = useShared((s) => s.setActiveCalls);
  const [selected, setSelected] = useState<ActiveCallDTO | null>(null);
  const [confirmEnd, setConfirmEnd] = useState<ActiveCallDTO | null>(null);
  const [now, setNow] = useState(() => Date.now());

  async function endCall(call: ActiveCallDTO) {
    try {
      await writes.endCall(cfg, call.device_serial);
      // Optimistically refresh the active list so the UI doesn't show
      // the ended call for the next poll cycle.
      const fresh = await api.activeCalls(cfg);
      setActiveCalls(fresh);
      setSelected(null);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "end-call request failed");
      throw e;
    }
    setConfirmEnd(null);
  }

  useEffect(() => {
    let cancel = false;
    const refresh = async () => {
      try {
        const data = await api.activeCalls(cfg);
        if (!cancel) setActiveCalls(data);
      } catch {
        // Toast strip surfaces request errors elsewhere; keep the
        // previous snapshot rather than blanking the table.
      }
    };
    refresh();
    const t = window.setInterval(refresh, POLL_INTERVAL_MS);
    return () => {
      cancel = true;
      window.clearInterval(t);
    };
  }, [cfg, setActiveCalls]);

  // Tick once a second so the elapsed-time column updates even when
  // no API response has come back yet.
  useEffect(() => {
    const t = window.setInterval(() => setNow(Date.now()), 1_000);
    return () => window.clearInterval(t);
  }, []);

  const columns: Column<ActiveCallDTO>[] = useMemo(
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
            {r.talkgroup?.alpha_tag ?? <em className="text-muted">—</em>}
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
        key: "flags",
        header: "",
        render: (r) => (
          <div className="flex flex-wrap gap-1">
            {r.grant.encrypted && <span className="pill-warn">enc</span>}
            {r.grant.emergency && <span className="pill-err">emerg</span>}
            {r.grant.data_call && <span className="pill">data</span>}
          </div>
        ),
      },
      {
        key: "elapsed",
        header: "Elapsed",
        render: (r) => (
          <span className="font-mono text-xs tabular-nums">
            {elapsed(r.started_at, now)}
          </span>
        ),
        sort: (a, b) => a.started_at.localeCompare(b.started_at),
      },
      {
        key: "device",
        header: "Device",
        render: (r) => (
          <span className="font-mono text-xs text-muted">{r.device_serial}</span>
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
      <header className="flex items-center justify-between gap-3">
        <h2 className="text-xl font-semibold">Active calls</h2>
        <span className="text-xs text-muted">{activeCalls.length} in flight</span>
      </header>

      <DataTable
        rows={activeCalls}
        columns={columns}
        rowKey={(r) => `${r.device_serial}-${r.started_at}`}
        defaultSortKey="elapsed"
        defaultSortDirection="desc"
        onRowClick={(r) => setSelected(r)}
        emptyMessage="No calls right now. Active grants show up here as soon as the daemon allocates a voice device."
      />

      {selected && (
        <DetailModal
          title={selected.talkgroup?.alpha_tag ?? `TG ${selected.grant.group_id}`}
          subtitle={`${selected.grant.system} · ${selected.grant.protocol}`}
          onClose={() => setSelected(null)}
        >
          <div className="grid grid-cols-2 gap-3">
            <DetailField label="TGID" mono value={selected.grant.group_id} />
            <DetailField
              label="Source"
              mono
              value={selected.grant.source_id ?? null}
            />
            <DetailField
              label="Frequency"
              mono
              value={formatHz(selected.grant.frequency_hz)}
            />
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
              label="Elapsed"
              mono
              value={elapsed(selected.started_at, now)}
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
          {canMutate ? (
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

function elapsed(startedAt: string, now: number): string {
  const startMs = Date.parse(startedAt);
  if (Number.isNaN(startMs)) return "—";
  const ms = Math.max(0, now - startMs);
  const totalSeconds = Math.floor(ms / 1000);
  const m = Math.floor(totalSeconds / 60);
  const s = totalSeconds % 60;
  return `${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
}

function formatHz(hz: number): string {
  if (!Number.isFinite(hz)) return "—";
  if (hz >= 1_000_000) return `${(hz / 1_000_000).toFixed(4)} MHz`;
  if (hz >= 1_000) return `${(hz / 1_000).toFixed(3)} kHz`;
  return `${hz} Hz`;
}
