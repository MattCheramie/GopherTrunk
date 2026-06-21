import { useEffect, useMemo, useState } from "react";
import { api } from "../api/client";
import { writes } from "../api/write";
import { Column, DataTable } from "../components/DataTable";
import { DetailField, DetailModal } from "../components/DetailModal";
import { PageHeader } from "../components/ui/PageHeader";
import type { CallRow, RIDDTO } from "../api/types";
import {
  selectCanMutate,
  selectClientConfig,
  useShared,
} from "../store/shared";

const POLL_INTERVAL_MS = 15_000;

// RadioIDs is the per-RID equivalent of Talkgroups. It merges the
// operator-configured static catalogue (rid_alias_file → RIDDB) with
// the live affiliation tracker (over-the-air observations), and lets
// operators page into per-RID call history.
export function RadioIDs() {
  const cfg = useShared(selectClientConfig);
  const canMutate = useShared(selectCanMutate);
  const setError = useShared((s) => s.setError);
  const notify = useShared((s) => s.notify);
  const rids = useShared((s) => s.rids);
  const setRIDs = useShared((s) => s.setRIDs);

  const [selected, setSelected] = useState<RIDDTO | null>(null);
  const [history, setHistory] = useState<CallRow[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [busy, setBusy] = useState(false);

  // Poll the merged list.
  useEffect(() => {
    let cancel = false;
    const refresh = async () => {
      try {
        const data = await api.rids(cfg);
        if (!cancel) setRIDs(data);
      } catch {
        // Keep the previous snapshot — a transient fetch failure
        // shouldn't clear the list out from under the operator.
      }
    };
    refresh();
    const t = window.setInterval(refresh, POLL_INTERVAL_MS);
    return () => {
      cancel = true;
      window.clearInterval(t);
    };
  }, [cfg, setRIDs]);

  // Fetch per-RID call history when the detail modal opens.
  useEffect(() => {
    if (!selected) {
      setHistory([]);
      return;
    }
    let cancel = false;
    setHistoryLoading(true);
    api
      .ridHistory(cfg, selected.id, { limit: 50 })
      .then((rows) => {
        if (!cancel) setHistory(rows);
      })
      .catch(() => {
        if (!cancel) setHistory([]);
      })
      .finally(() => {
        if (!cancel) setHistoryLoading(false);
      });
    return () => {
      cancel = true;
    };
  }, [cfg, selected]);

  async function patch(
    id: number,
    body: { alias?: string; watch?: boolean; lockout?: boolean; priority?: number },
  ) {
    setBusy(true);
    try {
      const updated = await writes.updateRID(cfg, id, body);
      setRIDs(rids.map((r) => (r.id === id ? { ...r, ...updated } : r)));
      setSelected((s) => (s && s.id === id ? { ...s, ...updated } : s));
      notify("success", `Updated radio ID ${id}`);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "rid update failed");
    } finally {
      setBusy(false);
    }
  }

  const columns: Column<RIDDTO>[] = useMemo(
    () => [
      {
        key: "id",
        header: "RID",
        render: (r) => <span className="font-mono">{r.id}</span>,
        sort: (a, b) => a.id - b.id,
      },
      {
        key: "alias",
        header: "Alias",
        render: (r) => (
          <span className="font-medium">
            {r.alias ?? <em className="text-muted">—</em>}
          </span>
        ),
        sort: (a, b) => (a.alias ?? "").localeCompare(b.alias ?? ""),
      },
      {
        key: "talker_alias",
        header: "Talker alias",
        render: (r) =>
          r.talker_alias == null ? (
            <span className="text-muted">—</span>
          ) : r.talker_alias_unreliable ? (
            <span
              className="text-xs italic text-warn"
              title="Decode unreliable — passed CRC but contains non-printable characters (possible bit errors)"
            >
              {r.talker_alias} ⚠
            </span>
          ) : (
            <span className="text-xs">{r.talker_alias}</span>
          ),
        sort: (a, b) =>
          (a.talker_alias ?? "").localeCompare(b.talker_alias ?? ""),
        className: "hidden md:table-cell",
        headerClassName: "hidden md:table-cell",
      },
      {
        key: "last_talkgroup",
        header: "Last TG",
        render: (r) =>
          r.last_talkgroup ? (
            <span className="font-mono text-xs">{r.last_talkgroup}</span>
          ) : (
            <span className="text-muted">—</span>
          ),
        sort: (a, b) => (a.last_talkgroup ?? 0) - (b.last_talkgroup ?? 0),
      },
      {
        key: "call_count",
        header: "Calls",
        render: (r) => (
          <span className="font-mono text-xs">{r.call_count ?? 0}</span>
        ),
        sort: (a, b) => (a.call_count ?? 0) - (b.call_count ?? 0),
      },
      {
        key: "flags",
        header: "Flags",
        render: (r) => (
          <div className="flex flex-wrap gap-1">
            {r.configured && <span className="pill-ok">cfg</span>}
            {r.watch && <span className="pill-ok">watch</span>}
            {r.lockout && <span className="pill-err">lock</span>}
            {(r.priority ?? 0) > 0 && (
              <span className="pill-warn">pri</span>
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
        title="Radio IDs"
        actions={
          <span className="text-xs text-muted">{rids.length} total</span>
        }
      />

      <DataTable
        rows={rids}
        columns={columns}
        rowKey={(r) => String(r.id)}
        defaultSortKey="id"
        tableId="rids"
        searchable
        searchAccessor={(r) =>
          [r.id, r.alias, r.talker_alias, r.description, r.tag, r.group, r.owner]
            .filter(Boolean)
            .join(" ")
        }
        searchPlaceholder="Search by id, alias, talker alias, group, owner…"
        onRowClick={(r) => setSelected(r)}
        emptyMessage={
          rids.length === 0
            ? "No radio IDs observed yet — load an rid_alias_file or wait for the affiliation tracker to surface live IDs."
            : "No radio IDs match the search."
        }
      />

      {selected && (
        <DetailModal
          title={
            selected.alias ??
            selected.talker_alias ??
            `RID ${selected.id}`
          }
          subtitle={`RID ${selected.id}${selected.configured ? "" : " · live only"}`}
          onClose={() => setSelected(null)}
        >
          <DetailField label="Description" value={selected.description} />
          <div className="grid grid-cols-2 gap-3">
            <DetailField label="Tag" value={selected.tag} />
            <DetailField label="Group" value={selected.group} />
            <DetailField label="Owner" value={selected.owner} />
            <DetailField
              label="Priority"
              mono
              value={selected.priority ?? null}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <DetailField
              label="Talker alias"
              value={
                selected.talker_alias == null
                  ? "—"
                  : selected.talker_alias_unreliable
                    ? `${selected.talker_alias} ⚠ (decode unreliable)`
                    : selected.talker_alias
              }
            />
            <DetailField
              label="Last system"
              value={selected.system ?? "—"}
            />
            <DetailField
              label="Last TG"
              mono
              value={selected.last_talkgroup ?? "—"}
            />
            <DetailField
              label="Calls observed"
              mono
              value={selected.call_count ?? 0}
            />
            <DetailField
              label="First seen"
              value={selected.first_seen ?? "—"}
            />
            <DetailField
              label="Last seen"
              value={selected.last_seen ?? "—"}
            />
          </div>

          <div className="pt-3 border-t border-panel">
            <p className="text-xs uppercase tracking-wider text-muted mb-2">
              Recent calls
            </p>
            {historyLoading ? (
              <p className="text-xs text-muted">Loading…</p>
            ) : history.length === 0 ? (
              <p className="text-xs text-muted">
                No calls in the persisted call log for this RID.
              </p>
            ) : (
              <ul className="space-y-1 max-h-56 overflow-y-auto text-xs">
                {history.map((c) => (
                  <li
                    key={c.id}
                    className="flex justify-between gap-3 font-mono"
                  >
                    <span>
                      {c.system} · TG {c.group_id}
                      {c.talkgroup_alpha ? ` · ${c.talkgroup_alpha}` : ""}
                    </span>
                    <span className="text-muted">{formatTime(c.started_at)}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>

          {canMutate && selected.configured ? (
            <div className="pt-3 border-t border-panel space-y-3">
              <p className="text-xs uppercase tracking-wider text-muted">
                Mutations
              </p>
              <label className="flex items-center gap-3 text-sm">
                <input
                  type="checkbox"
                  className="h-5 w-5"
                  checked={!!selected.watch}
                  disabled={busy}
                  onChange={(e) =>
                    patch(selected.id, { watch: e.target.checked })
                  }
                />
                <span>Watch list</span>
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
          ) : !selected.configured ? (
            <p className="text-xs text-muted pt-2">
              This RID was observed over the air but is not in any
              system's rid_alias_file. Add it to the file (and reload
              the daemon) to assign an alias, owner, or watch flag.
            </p>
          ) : (
            <p className="text-xs text-muted pt-2">
              Enable write mode in Settings to edit watch / lockout /
              priority from this browser.
            </p>
          )}
        </DetailModal>
      )}
    </div>
  );
}

function formatTime(rfc3339: string): string {
  // Render compact local time; fall back to the raw string if
  // parsing fails so a malformed daemon timestamp is still visible.
  const t = Date.parse(rfc3339);
  if (Number.isNaN(t)) return rfc3339;
  const d = new Date(t);
  return d.toLocaleString();
}
