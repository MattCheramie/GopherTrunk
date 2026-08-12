import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { api } from "../api/client";
import { writes } from "../api/write";
import { Column, DataTable } from "../components/DataTable";
import { DetailField, DetailModal } from "../components/DetailModal";
import { PageHeader } from "../components/ui/PageHeader";
import { Section } from "../components/ui/Section";
import { Select } from "../components/ui/Select";
import { PositionMap, type MapPoint } from "../components/PositionMap";
import type { CallRow, LocationFix, RIDDTO } from "../api/types";
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
  // Deep-link support: /rids/:id opens the detail modal for that radio,
  // so a source-radio click in CC Activity / call history lands on the
  // radio (issue: the link previously fell through to the dashboard).
  const { id: routeId } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const [selected, setSelected] = useState<RIDDTO | null>(null);
  const [history, setHistory] = useState<CallRow[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  // Selected system filter ("" = all systems). Populates the ?system= query so
  // a large multi-system RID roster stays legible.
  const [system, setSystem] = useState("");
  const [systemNames, setSystemNames] = useState<string[]>([]);
  // Recent subscriber-radio position fixes (DMR LRRP / unit GPS) for the map.
  const [locations, setLocations] = useState<LocationFix[]>([]);

  // Poll the merged list, scoped to the selected system.
  useEffect(() => {
    let cancel = false;
    const refresh = async () => {
      try {
        const data = await api.rids(cfg, system ? { system } : {});
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
  }, [cfg, setRIDs, system]);

  // Fetch the configured system list once for the filter dropdown (independent
  // of the current filter, so the options stay stable while scoped).
  useEffect(() => {
    let cancel = false;
    api
      .systems(cfg)
      .then((s) => {
        if (!cancel) setSystemNames(s.map((sys) => sys.name).filter(Boolean));
      })
      .catch(() => {
        // Non-fatal: the dropdown falls back to systems seen in the RID rows.
      });
    return () => {
      cancel = true;
    };
  }, [cfg]);

  // Poll recent position fixes for the unit-location map. Empty (no LRRP/GPS
  // decoded, or the subsystem is off) simply hides the map.
  useEffect(() => {
    let cancel = false;
    const refresh = async () => {
      try {
        const fixes = await api.locations(cfg, { limit: 500 });
        if (!cancel) setLocations(fixes);
      } catch {
        // Non-fatal — keep the previous fixes.
      }
    };
    refresh();
    const t = window.setInterval(refresh, POLL_INTERVAL_MS);
    return () => {
      cancel = true;
      window.clearInterval(t);
    };
  }, [cfg]);

  // Map the newest fix per radio to a marker, labelled by alias when the RID
  // roster resolves one. Keeping only the latest fix per radio avoids stacking
  // a trail of markers on one unit.
  const mapPoints = useMemo<MapPoint[]>(() => {
    const aliasByID = new Map<number, string>();
    for (const r of rids) if (r.alias) aliasByID.set(r.id, r.alias);
    const latest = new Map<number, LocationFix>();
    for (const f of locations) {
      if (!Number.isFinite(f.latitude) || !Number.isFinite(f.longitude)) continue;
      const prev = latest.get(f.radio_id);
      if (!prev || f.reported_at > prev.reported_at) latest.set(f.radio_id, f);
    }
    return [...latest.values()].map((f) => {
      const alias = aliasByID.get(f.radio_id);
      const speed = f.speed_knots > 0 ? ` · ${f.speed_knots.toFixed(0)} kn` : "";
      return {
        id: String(f.radio_id),
        latitude: f.latitude,
        longitude: f.longitude,
        kind: "unit" as const,
        label: alias ? `${alias} (${f.radio_id})` : `RID ${f.radio_id}`,
        detail: `${f.system}${f.talkgroup ? ` · TG ${f.talkgroup}` : ""}${speed}`,
      };
    });
  }, [locations, rids]);

  // Dropdown options: configured systems ∪ systems seen in the current rows ∪
  // the active selection (so a discovered-only or just-selected system never
  // vanishes from the list).
  const systemOptions = useMemo(() => {
    const set = new Set<string>();
    for (const n of systemNames) if (n) set.add(n);
    for (const r of rids) if (r.system) set.add(r.system);
    if (system) set.add(system);
    return [...set].sort((a, b) => a.localeCompare(b));
  }, [systemNames, rids, system]);

  // Open the detail modal for the /rids/:id route param. Prefer a row
  // already in the loaded list; otherwise fetch the single RID directly so a
  // deep link (or a radio filtered out by the current system scope) still
  // opens. An unknown id drops back to the list URL rather than stranding the
  // user on a dead link.
  useEffect(() => {
    if (!routeId) return;
    const idNum = Number(routeId);
    if (!Number.isFinite(idNum)) return;
    if (selected?.id === idNum) return;
    const inList = rids.find((r) => r.id === idNum);
    if (inList) {
      setSelected(inList);
      return;
    }
    let cancel = false;
    api
      .rid(cfg, idNum)
      .then((r) => {
        if (!cancel) setSelected(r);
      })
      .catch(() => {
        if (!cancel) navigate("/rids", { replace: true });
      });
    return () => {
      cancel = true;
    };
  }, [routeId, rids, cfg, selected, navigate]);

  // Close the detail modal, restoring the /rids list URL when the modal was
  // opened via a /rids/:id deep link.
  function closeDetail() {
    setSelected(null);
    if (routeId) navigate("/rids");
  }

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

      {mapPoints.length > 0 && (
        <Section
          id="rid-map"
          title={`Unit locations (${mapPoints.length})`}
          description="Latest GPS / LRRP position per radio. Secondary to the roster — collapse if not needed."
        >
          <PositionMap points={mapPoints} heightPx={320} />
        </Section>
      )}

      <DataTable
        rows={rids}
        columns={columns}
        rowKey={(r) => String(r.id)}
        defaultSortKey="id"
        tableId="rids"
        pageSize={50}
        searchable
        searchAccessor={(r) =>
          [
            r.id,
            r.alias,
            r.talker_alias,
            r.description,
            r.tag,
            r.group,
            r.owner,
            r.system,
          ]
            .filter(Boolean)
            .join(" ")
        }
        searchPlaceholder="Search by id, alias, talker alias, group, owner…"
        toolbar={
          <Select
            className="w-auto"
            aria-label="Filter by system"
            value={system}
            onChange={(e) => setSystem(e.target.value)}
          >
            <option value="">All systems</option>
            {systemOptions.map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </Select>
        }
        onRowClick={(r) => {
          // Open immediately (works regardless of the route) and reflect the
          // selection in the URL so it is shareable / back-button friendly.
          setSelected(r);
          navigate(`/rids/${r.id}`);
        }}
        emptyMessage={
          rids.length === 0
            ? system
              ? `No radio IDs on system “${system}” yet.`
              : "No radio IDs observed yet — load an rid_alias_file or wait for the affiliation tracker to surface live IDs."
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
          onClose={closeDetail}
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
