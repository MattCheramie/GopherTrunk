import { useMemo, useState } from "react";
import { api } from "../api/client";
import { writes } from "../api/write";
import { Column, DataTable } from "../components/DataTable";
import { DetailField, DetailModal } from "../components/DetailModal";
import { StaleIndicator } from "../components/ui/StaleIndicator";
import { PageHeader } from "../components/ui/PageHeader";
import { Badge } from "../components/ui/Badge";
import type { TalkgroupDTO } from "../api/types";
import { useDataPoll } from "../hooks/useDataPoll";
import {
  selectCanMutate,
  selectClientConfig,
  useShared,
} from "../store/shared";

const POLL_INTERVAL_MS = 15_000;

// Talkgroups is read-only in this PR. Priority / lockout / scan
// mutations (PATCH /api/v1/talkgroups/{id}) land in the mutation pass
// that introduces the daemon-write capability gate UI.
export function Talkgroups() {
  const cfg = useShared(selectClientConfig);
  const canMutate = useShared(selectCanMutate);
  const setError = useShared((s) => s.setError);
  const notify = useShared((s) => s.notify);
  const talkgroups = useShared((s) => s.talkgroups);
  const setTalkgroups = useShared((s) => s.setTalkgroups);
  const [selected, setSelected] = useState<TalkgroupDTO | null>(null);
  const [busy, setBusy] = useState(false);
  const [hideDiscovered, setHideDiscovered] = useState(false);

  const { stale, lastUpdated } = useDataPoll({
    fetcher: () => api.talkgroups(cfg),
    onData: setTalkgroups,
    intervalMs: POLL_INTERVAL_MS,
    resetKey: cfg.baseURL,
  });

  async function patch(id: number, body: { priority?: number; lockout?: boolean; scan?: boolean }) {
    setBusy(true);
    try {
      const updated = await writes.updateTalkgroup(cfg, id, body);
      // Optimistically merge the response into the local list and
      // selected detail so the UI updates without waiting for the poll.
      setTalkgroups(
        talkgroups.map((t) => (t.id === id ? { ...t, ...updated } : t)),
      );
      setSelected((s) => (s && s.id === id ? { ...s, ...updated } : s));
      notify("success", `Updated talkgroup ${id}`);
    } catch (e: unknown) {
      setError(
        e instanceof Error ? e.message : "talkgroup update failed",
      );
    } finally {
      setBusy(false);
    }
  }

  const columns: Column<TalkgroupDTO>[] = useMemo(
    () => [
      {
        key: "id",
        header: "ID",
        render: (r) => <span className="font-mono">{r.id}</span>,
        sort: (a, b) => a.id - b.id,
      },
      {
        key: "alpha",
        header: "Alpha tag",
        render: (r) => (
          <span className="font-medium">{r.alpha_tag ?? <em className="text-muted">—</em>}</span>
        ),
        sort: (a, b) =>
          (a.alpha_tag ?? "").localeCompare(b.alpha_tag ?? ""),
      },
      {
        key: "group",
        header: "Group",
        render: (r) => <span className="text-xs">{r.group ?? "—"}</span>,
        sort: (a, b) => (a.group ?? "").localeCompare(b.group ?? ""),
        className: "hidden md:table-cell",
        headerClassName: "hidden md:table-cell",
      },
      {
        key: "priority",
        header: "Pri",
        render: (r) => (
          <span className="font-mono text-xs">{r.priority ?? "—"}</span>
        ),
        sort: (a, b) => (a.priority ?? 99) - (b.priority ?? 99),
      },
      {
        key: "flags",
        header: "Flags",
        render: (r) => (
          <div className="flex flex-wrap gap-1">
            {r.scan && <Badge tone="ok">scan</Badge>}
            {r.lockout && <Badge tone="err">lock</Badge>}
            {r.priority != null && r.priority > 0 && (
              <Badge tone="warn">pri</Badge>
            )}
            {r.discovered && <Badge tone="neutral">discovered</Badge>}
          </div>
        ),
      },
    ],
    [],
  );

  // Auto-discovered entries self-populate from the air; the operator can hide
  // them to collapse phantom radio-ID leaks that slipped past the backend
  // known-radio filter. Curated entries are always shown.
  const discoveredCount = useMemo(
    () => talkgroups.filter((t) => t.discovered).length,
    [talkgroups],
  );
  const visibleTalkgroups = useMemo(
    () => (hideDiscovered ? talkgroups.filter((t) => !t.discovered) : talkgroups),
    [talkgroups, hideDiscovered],
  );

  return (
    <div className="space-y-3">
      <PageHeader
        title="Talkgroups"
        actions={
          <>
            <StaleIndicator stale={stale} lastUpdated={lastUpdated} />
            {discoveredCount > 0 && (
              <label className="flex items-center gap-1 text-xs text-muted">
                <input
                  type="checkbox"
                  checked={hideDiscovered}
                  onChange={(e) => setHideDiscovered(e.target.checked)}
                />
                Hide auto-discovered ({discoveredCount})
              </label>
            )}
            <span className="text-xs text-muted">
              {visibleTalkgroups.length} shown
            </span>
          </>
        }
      />

      <DataTable
        rows={visibleTalkgroups}
        columns={columns}
        rowKey={(r) => String(r.id)}
        defaultSortKey="id"
        tableId="talkgroups"
        searchable
        searchAccessor={(r) =>
          [r.id, r.alpha_tag, r.description, r.tag, r.group]
            .filter(Boolean)
            .join(" ")
        }
        searchPlaceholder="Search by id, alpha tag, group, tag…"
        rowActions={
          canMutate
            ? (r) => (
                <span className="inline-flex gap-1">
                  <button
                    className={r.scan ? "pill-ok" : "pill"}
                    disabled={busy}
                    title="Toggle scan"
                    onClick={() => patch(r.id, { scan: !r.scan })}
                  >
                    scan
                  </button>
                  <button
                    className={r.lockout ? "pill-err" : "pill"}
                    disabled={busy}
                    title="Toggle lockout"
                    onClick={() => patch(r.id, { lockout: !r.lockout })}
                  >
                    lock
                  </button>
                </span>
              )
            : undefined
        }
        onRowClick={(r) => setSelected(r)}
        emptyMessage={
          talkgroups.length === 0
            ? "No talkgroups configured."
            : "No talkgroups match the search."
        }
      />

      {selected && (
        <DetailModal
          title={selected.alpha_tag ?? `Talkgroup ${selected.id}`}
          subtitle={`TGID ${selected.id}`}
          onClose={() => setSelected(null)}
        >
          <DetailField label="Description" value={selected.description} />
          <div className="grid grid-cols-2 gap-3">
            <DetailField label="Tag" value={selected.tag} />
            <DetailField label="Group" value={selected.group} />
            <DetailField label="Mode" value={selected.mode} />
            <DetailField
              label="Priority"
              mono
              value={selected.priority ?? null}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <DetailField
              label="Scan"
              value={selected.scan ? "enabled" : "disabled"}
            />
            <DetailField
              label="Lockout"
              value={selected.lockout ? "locked out" : "active"}
            />
          </div>
          {canMutate ? (
            <div className="pt-3 border-t border-panel space-y-3">
              <p className="text-xs uppercase tracking-wider text-muted flex items-center gap-2">
                Mutations
                {busy && (
                  <span className="text-accent normal-case tracking-normal">
                    saving…
                  </span>
                )}
              </p>
              <label className="flex items-center gap-3 text-sm">
                <input
                  type="checkbox"
                  className="h-5 w-5"
                  checked={!!selected.scan}
                  disabled={busy}
                  onChange={(e) => patch(selected.id, { scan: e.target.checked })}
                />
                <span>Scan</span>
              </label>
              <label className="flex items-center gap-3 text-sm">
                <input
                  type="checkbox"
                  className="h-5 w-5"
                  checked={!!selected.lockout}
                  disabled={busy}
                  onChange={(e) =>
                    patch(selected.id, { lockout: e.target.checked })
                  }
                />
                <span>Lockout</span>
              </label>
              <label className="flex items-center gap-3 text-sm">
                <span className="w-20">Priority</span>
                <input
                  type="number"
                  min={0}
                  max={9}
                  className="input w-20"
                  value={selected.priority ?? 0}
                  disabled={busy}
                  onChange={(e) => {
                    const v = parseInt(e.target.value, 10);
                    if (Number.isFinite(v)) patch(selected.id, { priority: v });
                  }}
                />
              </label>
            </div>
          ) : (
            <p className="text-xs text-muted pt-2">
              Enable write mode in Settings to edit scan / lockout /
              priority from this browser.
            </p>
          )}
        </DetailModal>
      )}
    </div>
  );
}
