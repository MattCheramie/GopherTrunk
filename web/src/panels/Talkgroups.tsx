import { useEffect, useMemo, useState } from "react";
import { api } from "../api/client";
import { writes } from "../api/write";
import { Column, DataTable } from "../components/DataTable";
import { DetailField, DetailModal } from "../components/DetailModal";
import type { TalkgroupDTO } from "../api/types";
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
  const talkgroups = useShared((s) => s.talkgroups);
  const setTalkgroups = useShared((s) => s.setTalkgroups);
  const [selected, setSelected] = useState<TalkgroupDTO | null>(null);
  const [filter, setFilter] = useState("");
  const [busy, setBusy] = useState(false);

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
    } catch (e: unknown) {
      setError(
        e instanceof Error ? e.message : "talkgroup update failed",
      );
    } finally {
      setBusy(false);
    }
  }

  useEffect(() => {
    let cancel = false;
    const refresh = async () => {
      try {
        const data = await api.talkgroups(cfg);
        if (!cancel) setTalkgroups(data);
      } catch {
        // Keep the previous snapshot.
      }
    };
    refresh();
    const t = window.setInterval(refresh, POLL_INTERVAL_MS);
    return () => {
      cancel = true;
      window.clearInterval(t);
    };
  }, [cfg, setTalkgroups]);

  const filtered = useMemo(() => {
    if (!filter.trim()) return talkgroups;
    const needle = filter.toLowerCase();
    return talkgroups.filter((t) => {
      const idStr = String(t.id);
      return (
        idStr.includes(needle) ||
        (t.alpha_tag ?? "").toLowerCase().includes(needle) ||
        (t.description ?? "").toLowerCase().includes(needle) ||
        (t.tag ?? "").toLowerCase().includes(needle) ||
        (t.group ?? "").toLowerCase().includes(needle)
      );
    });
  }, [talkgroups, filter]);

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
            {r.scan && <span className="pill-ok">scan</span>}
            {r.lockout && <span className="pill-err">lock</span>}
            {r.priority != null && r.priority > 0 && (
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
      <header className="flex items-center justify-between gap-3">
        <h2 className="text-xl font-semibold">Talkgroups</h2>
        <span className="text-xs text-muted">
          {filtered.length} of {talkgroups.length}
        </span>
      </header>

      <input
        type="search"
        className="input w-full sm:max-w-xs"
        placeholder="Filter by id, alpha tag, group, tag…"
        value={filter}
        onChange={(e) => setFilter(e.target.value)}
        aria-label="Filter talkgroups"
      />

      <DataTable
        rows={filtered}
        columns={columns}
        rowKey={(r) => String(r.id)}
        defaultSortKey="id"
        onRowClick={(r) => setSelected(r)}
        emptyMessage={
          talkgroups.length === 0
            ? "No talkgroups configured."
            : "No talkgroups match the filter."
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
              <p className="text-xs uppercase tracking-wider text-muted">
                Mutations
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
