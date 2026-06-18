import { useEffect, useMemo, useState } from "react";
import { api } from "../api/client";
import { Column, DataTable } from "../components/DataTable";
import { DetailField, DetailModal } from "../components/DetailModal";
import type {
  DMRBandPlanDTO,
  DMRBandPlanLearnedDTO,
  EventDTO,
  SystemDTO,
  SystemHuntStatusDTO,
} from "../api/types";
import { selectClientConfig, useShared } from "../store/shared";

const POLL_INTERVAL_MS = 10_000;

// formatBandPlan renders a compact one-line summary of a DMR Tier III band
// plan (the LCN→frequency map voice grants resolve through). (#638)
function formatBandPlan(bp: DMRBandPlanDTO | undefined): string | null {
  if (!bp) return null;
  if (bp.linear && bp.linear.base_hz) {
    const base = (bp.linear.base_hz / 1e6).toFixed(4);
    const spacing = (bp.linear.spacing_hz / 1e3).toFixed(2);
    return `${base} MHz · ${spacing} kHz`;
  }
  if (bp.table && bp.table.length > 0) {
    return `${bp.table.length} LCNs (table)`;
  }
  return null;
}

// latestLearned returns the newest dmr.bandplan.learned event payload for a
// system, so the detail modal can show that the plan was learned live by the
// autoconfig learner (vs. operator-configured). (#638)
function latestLearned(
  events: EventDTO[],
  systemName: string,
): DMRBandPlanLearnedDTO | null {
  for (let i = events.length - 1; i >= 0; i--) {
    const ev = events[i];
    if (ev.kind !== "dmr.bandplan.learned") continue;
    const p = ev.payload as DMRBandPlanLearnedDTO | undefined;
    if (p && p.system === systemName) return p;
  }
  return null;
}

// Network identity (WACN/System ID/RFSS/Site) is populated from
// decoded P25 TSBKs 0x3A/0x3B, not config. Translate the live hunt
// state into operator-facing copy so an empty cell explains itself.
function identityEmptyHint(hunt: SystemHuntStatusDTO | undefined): string {
  if (!hunt) return "Scanner offline";
  switch (hunt.state) {
    case "locked":
      return "Awaiting status broadcasts";
    case "hunting":
      return "Hunting control channel";
    default:
      return hunt.state ? `System not locked (${hunt.state})` : "System not locked";
  }
}

export function Systems() {
  const cfg = useShared(selectClientConfig);
  const systems = useShared((s) => s.systems);
  const setSystems = useShared((s) => s.setSystems);
  const scanner = useShared((s) => s.scanner);
  const setScanner = useShared((s) => s.setScanner);
  const events = useShared((s) => s.events);
  const [selected, setSelected] = useState<SystemDTO | null>(null);
  const [filter, setFilter] = useState("");

  useEffect(() => {
    let cancel = false;
    const refresh = async () => {
      // Poll the scanner snapshot alongside systems so the detail
      // modal can translate empty WACN/SystemID/RFSS/Site into a
      // hunt-state hint even when the Scanner panel isn't mounted.
      const [sysRes, scanRes] = await Promise.allSettled([
        api.systems(cfg),
        api.scanner(cfg),
      ]);
      if (cancel) return;
      if (sysRes.status === "fulfilled") setSystems(sysRes.value);
      if (scanRes.status === "fulfilled") setScanner(scanRes.value);
    };
    refresh();
    const t = window.setInterval(refresh, POLL_INTERVAL_MS);
    return () => {
      cancel = true;
      window.clearInterval(t);
    };
  }, [cfg, setSystems, setScanner]);

  const filtered = useMemo(() => {
    if (!filter.trim()) return systems;
    const needle = filter.toLowerCase();
    return systems.filter(
      (s) =>
        s.name.toLowerCase().includes(needle) ||
        s.protocol.toLowerCase().includes(needle),
    );
  }, [systems, filter]);

  const columns: Column<SystemDTO>[] = useMemo(
    () => [
      {
        key: "name",
        header: "System",
        render: (r) => <span className="font-medium">{r.name}</span>,
        sort: (a, b) => a.name.localeCompare(b.name),
      },
      {
        key: "protocol",
        header: "Protocol",
        render: (r) => <span className="font-mono text-accent">{r.protocol}</span>,
        sort: (a, b) => a.protocol.localeCompare(b.protocol),
      },
      {
        key: "ccs",
        header: "Control channels",
        render: (r) => (
          <span className="font-mono text-xs text-muted">
            {r.control_channels?.length
              ? `${r.control_channels.length} freq${r.control_channels.length === 1 ? "" : "s"}`
              : "—"}
          </span>
        ),
        sort: (a, b) =>
          (a.control_channels?.length ?? 0) -
          (b.control_channels?.length ?? 0),
      },
      {
        key: "bandplan",
        header: "Band plan",
        render: (r) => {
          const bp = formatBandPlan(r.dmr_band_plan);
          return bp ? (
            <span className="font-mono text-xs text-muted">{bp}</span>
          ) : (
            <span className="text-muted">—</span>
          );
        },
      },
      {
        key: "site",
        header: "Site",
        render: (r) =>
          r.site != null ? (
            <span className="font-mono text-xs">{r.site}</span>
          ) : (
            <span className="text-muted">—</span>
          ),
        sort: (a, b) => (a.site ?? -1) - (b.site ?? -1),
      },
    ],
    [],
  );

  return (
    <div className="space-y-3">
      <header className="flex items-center justify-between gap-3">
        <h2 className="text-xl font-semibold">Systems</h2>
        <span className="text-xs text-muted">
          {filtered.length} of {systems.length}
        </span>
      </header>

      <input
        type="search"
        className="input w-full sm:max-w-xs"
        placeholder="Filter by name or protocol…"
        value={filter}
        onChange={(e) => setFilter(e.target.value)}
        aria-label="Filter systems"
      />

      <DataTable
        rows={filtered}
        columns={columns}
        rowKey={(r) => r.name}
        defaultSortKey="name"
        onRowClick={(r) => setSelected(r)}
        emptyMessage={
          systems.length === 0
            ? "No trunked systems configured."
            : "No systems match the filter."
        }
      />

      {selected && (
        <DetailModal
          title={selected.name}
          subtitle={selected.protocol}
          onClose={() => setSelected(null)}
        >
          <DetailField
            label="Control channels (Hz)"
            mono
            value={
              selected.control_channels?.length
                ? selected.control_channels.join("\n")
                : null
            }
          />
          {(() => {
            const hunt = scanner?.systems?.find((h) => h.name === selected.name);
            const hint = identityEmptyHint(hunt);
            return (
              <div>
                <p className="text-xs uppercase tracking-wider text-muted mb-2">
                  Network identity (decoded live)
                </p>
                <div className="grid grid-cols-2 gap-3">
                  <DetailField
                    label="WACN"
                    mono
                    value={selected.wacn ?? null}
                    emptyHint={hint}
                  />
                  <DetailField
                    label="System ID"
                    mono
                    value={selected.system_id ?? null}
                    emptyHint={hint}
                  />
                  <DetailField
                    label="RFSS"
                    mono
                    value={selected.rfss ?? null}
                    emptyHint={hint}
                  />
                  <DetailField
                    label="Site"
                    mono
                    value={selected.site ?? null}
                    emptyHint={hint}
                  />
                </div>
              </div>
            );
          })()}
          {(() => {
            const bp = selected.dmr_band_plan;
            if (!bp) return null;
            const learned = latestLearned(events, selected.name);
            const lines: string[] = [];
            if (bp.linear && bp.linear.base_hz) {
              lines.push(`base ${(bp.linear.base_hz / 1e6).toFixed(4)} MHz`);
              lines.push(`spacing ${(bp.linear.spacing_hz / 1e3).toFixed(3)} kHz`);
              if (bp.linear.offset) lines.push(`offset ${bp.linear.offset}`);
            } else if (bp.table) {
              for (const e of bp.table) {
                lines.push(`LCN ${e.lcn} → ${(e.freq_hz / 1e6).toFixed(4)} MHz`);
              }
            }
            return (
              <div>
                <p className="text-xs uppercase tracking-wider text-muted mb-2">
                  DMR band plan
                  {learned ? (
                    <span className="ml-2 text-accent normal-case tracking-normal">
                      · learned live (conf {(learned.confidence * 100).toFixed(0)}%,{" "}
                      {learned.num_pairs} pairs)
                    </span>
                  ) : null}
                </p>
                <DetailField
                  label={bp.linear ? "Linear plan" : "LCN → frequency"}
                  mono
                  value={lines.length ? lines.join("\n") : null}
                />
              </div>
            );
          })()}
        </DetailModal>
      )}
    </div>
  );
}
