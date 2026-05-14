import { useEffect, useMemo, useState } from "react";
import { api } from "../api/client";
import { Column, DataTable } from "../components/DataTable";
import { DetailField, DetailModal } from "../components/DetailModal";
import type { CallRow } from "../api/types";
import { selectClientConfig, useShared } from "../store/shared";

// History reads /api/v1/calls/history with the same filter shape the
// daemon accepts: limit, system, group_id. Read-only in this pass;
// retention-sweep button lands in the mutation pass.
export function History() {
  const cfg = useShared(selectClientConfig);

  const [rows, setRows] = useState<CallRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Form fields kept separate from the "submitted" filter object so
  // typing into the inputs doesn't trigger a fetch on every keystroke.
  const [limitInput, setLimitInput] = useState("200");
  const [systemInput, setSystemInput] = useState("");
  const [groupInput, setGroupInput] = useState("");
  const [filter, setFilter] = useState<{
    limit?: number;
    system?: string;
    group_id?: number;
  }>({ limit: 200 });

  const [selected, setSelected] = useState<CallRow | null>(null);

  useEffect(() => {
    let cancel = false;
    setLoading(true);
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
  }, [cfg, filter]);

  function applyFilter(e: React.FormEvent) {
    e.preventDefault();
    const next: typeof filter = {};
    const lim = parseInt(limitInput, 10);
    if (Number.isFinite(lim) && lim > 0) next.limit = lim;
    if (systemInput.trim()) next.system = systemInput.trim();
    const gid = parseInt(groupInput, 10);
    if (Number.isFinite(gid)) next.group_id = gid;
    setFilter(next);
  }

  function clearFilter() {
    setLimitInput("200");
    setSystemInput("");
    setGroupInput("");
    setFilter({ limit: 200 });
  }

  const columns: Column<CallRow>[] = useMemo(
    () => [
      {
        key: "started",
        header: "Started",
        render: (r) => (
          <span className="font-mono text-xs text-muted whitespace-nowrap">
            {r.started_at.replace("T", " ").replace(/\..*$/, "")}
          </span>
        ),
        sort: (a, b) => a.started_at.localeCompare(b.started_at),
      },
      {
        key: "tg",
        header: "TG",
        render: (r) => (
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
        key: "system",
        header: "System",
        render: (r) => <span className="text-xs">{r.system}</span>,
        sort: (a, b) => a.system.localeCompare(b.system),
        className: "hidden md:table-cell",
        headerClassName: "hidden md:table-cell",
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
            {r.encrypted && <span className="pill-warn">enc</span>}
            {r.emergency && <span className="pill-err">emerg</span>}
            {r.data_call && <span className="pill">data</span>}
          </div>
        ),
      },
    ],
    [],
  );

  return (
    <div className="space-y-3">
      <header className="flex items-center justify-between gap-3">
        <h2 className="text-xl font-semibold">Call history</h2>
        <span className="text-xs text-muted">
          {loading
            ? "loading…"
            : `${rows.length} row${rows.length === 1 ? "" : "s"}`}
        </span>
      </header>

      <form
        onSubmit={applyFilter}
        className="panel p-3 grid grid-cols-2 sm:grid-cols-4 gap-2 items-end"
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
        onRowClick={(r) => setSelected(r)}
        emptyMessage={
          loading
            ? "loading…"
            : "No calls in the daemon's call log for this filter."
        }
      />

      {selected && (
        <DetailModal
          title={selected.talkgroup_alpha ?? `TG ${selected.group_id}`}
          subtitle={`${selected.system} · ${selected.protocol}`}
          onClose={() => setSelected(null)}
        >
          <div className="grid grid-cols-2 gap-3">
            <DetailField label="Row ID" mono value={selected.id} />
            <DetailField label="TGID" mono value={selected.group_id} />
            <DetailField
              label="Source"
              mono
              value={selected.source_id ?? null}
            />
            <DetailField
              label="Frequency"
              mono
              value={formatHz(selected.frequency_hz)}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <DetailField
              label="Started"
              mono
              value={selected.started_at.replace("T", " ").replace(/\..*$/, "")}
            />
            <DetailField
              label="Ended"
              mono
              value={
                selected.ended_at
                  ? selected.ended_at.replace("T", " ").replace(/\..*$/, "")
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
          <DetailField
            label="Device"
            mono
            value={selected.device_serial ?? null}
          />
        </DetailModal>
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
