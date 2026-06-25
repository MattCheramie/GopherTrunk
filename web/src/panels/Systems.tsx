import { useMemo, useState } from "react";
import { api } from "../api/client";
import { Column, DataTable } from "../components/DataTable";
import { DetailField, DetailModal } from "../components/DetailModal";
import { PageHeader } from "../components/ui/PageHeader";
import { StaleIndicator } from "../components/ui/StaleIndicator";
import { useDataPoll } from "../hooks/useDataPoll";
import { formatIdNumber } from "../lib/idFormat";
import type {
  DMRBandPlanDTO,
  DMRBandPlanLearnedDTO,
  EventDTO,
  NeighborDTO,
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

// formatNeighbor renders one adjacent site as "RFSS x / Site y → downlink MHz",
// the way SDRtrunk's "Neighbor Sites" view does. Identity numbers follow the
// operator's id_base; the downlink is omitted when the band plan hasn't resolved
// it yet.
function formatNeighbor(n: NeighborDTO, idBase: "hex" | "dec"): string {
  const rfss = formatIdNumber(n.rfss, idBase) ?? String(n.rfss);
  const site = formatIdNumber(n.site, idBase) ?? String(n.site);
  let s = `RFSS ${rfss} / Site ${site}`;
  if (n.downlink_hz) s += ` → ${(n.downlink_hz / 1e6).toFixed(6)} MHz`;
  return s;
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

// Network identity (WACN/System ID/RFSS/Site) is decoded live from P25 status
// broadcasts (RFSS_STS_BCST 0x3A / NET_STS_BCST 0x3B), not config — the daemon
// overlays it onto the system as those TSBKs arrive. An empty cell therefore
// means "not decoded yet", not "scanner offline": the engine decodes identity
// whenever the control channel is being received, independent of the CC-hunt
// supervisor state (which is what `hunt` reflects, and is absent for a system
// the always-on trunking engine is decoding). Only call out an active hunt;
// otherwise say we're waiting on the (infrequent) status broadcasts.
function identityEmptyHint(hunt: SystemHuntStatusDTO | undefined): string {
  if (hunt?.state === "hunting") return "Hunting control channel";
  return "Awaiting status broadcasts";
}

// wacnEmptyHint explains a blank WACN specifically. The WACN is carried only
// by the P25 Network Status Broadcast (NSB) — System ID / RFSS / Site have
// other sources (RFSS Status 0x3A, Adjacent Site 0x3C, Location Registration
// 0x2B). So once the site is otherwise identified, a blank WACN is not "still
// acquiring": it means the NSB specifically hasn't decoded yet, and some
// systems emit it rarely or not at all on a given control channel. Saying so
// is more honest than the generic "Awaiting status broadcasts", which implies
// it's imminent. Falls back to the generic hint when nothing has decoded yet.
function wacnEmptyHint(
  sys: SystemDTO,
  hunt: SystemHuntStatusDTO | undefined,
): string {
  if (hunt?.state === "hunting") return "Hunting control channel";
  const identified =
    (sys.system_id ?? 0) !== 0 ||
    (sys.rfss ?? 0) !== 0 ||
    (sys.site ?? 0) !== 0;
  if (identified) return "No Network Status Broadcast yet";
  return "Awaiting status broadcasts";
}

export function Systems() {
  const cfg = useShared(selectClientConfig);
  const systems = useShared((s) => s.systems);
  const setSystems = useShared((s) => s.setSystems);
  const scanner = useShared((s) => s.scanner);
  const setScanner = useShared((s) => s.setScanner);
  const events = useShared((s) => s.events);
  const idBase = useShared((s) => s.idBase);
  const [selected, setSelected] = useState<SystemDTO | null>(null);

  // Poll the scanner snapshot alongside systems so the detail modal can
  // translate empty WACN/SystemID/RFSS/Site into a hunt-state hint even
  // when the Scanner panel isn't mounted.
  const { stale, lastUpdated } = useDataPoll({
    fetcher: async () => {
      const [sysRes, scanRes] = await Promise.allSettled([
        api.systems(cfg),
        api.scanner(cfg),
      ]);
      if (sysRes.status === "rejected" && scanRes.status === "rejected") {
        throw sysRes.reason;
      }
      return {
        systems: sysRes.status === "fulfilled" ? sysRes.value : null,
        scanner: scanRes.status === "fulfilled" ? scanRes.value : null,
      };
    },
    onData: (d) => {
      if (d.systems) setSystems(d.systems);
      if (d.scanner) setScanner(d.scanner);
    },
    intervalMs: POLL_INTERVAL_MS,
    resetKey: cfg.baseURL,
  });

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
      {
        key: "neighbors",
        header: "Neighbors",
        render: (r) =>
          r.neighbors?.length ? (
            <span className="font-mono text-xs">{r.neighbors.length}</span>
          ) : (
            <span className="text-muted">—</span>
          ),
        sort: (a, b) => (a.neighbors?.length ?? 0) - (b.neighbors?.length ?? 0),
      },
    ],
    [],
  );

  return (
    <div className="space-y-3">
      <PageHeader
        title="Systems"
        actions={
          <>
            <StaleIndicator stale={stale} lastUpdated={lastUpdated} />
            <span className="text-xs text-muted">{systems.length} total</span>
          </>
        }
      />

      <DataTable
        rows={systems}
        columns={columns}
        rowKey={(r) => r.name}
        defaultSortKey="name"
        searchable
        searchAccessor={(s) => `${s.name} ${s.protocol}`}
        searchPlaceholder="Search by name or protocol…"
        onRowClick={(r) => setSelected(r)}
        emptyMessage={
          systems.length === 0
            ? "No trunked systems configured."
            : "No systems match the search."
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
                    value={formatIdNumber(selected.wacn ?? null, idBase)}
                    emptyHint={wacnEmptyHint(selected, hunt)}
                  />
                  <DetailField
                    label="System ID"
                    mono
                    value={formatIdNumber(selected.system_id ?? null, idBase)}
                    emptyHint={hint}
                  />
                  <DetailField
                    label="RFSS"
                    mono
                    value={formatIdNumber(selected.rfss ?? null, idBase)}
                    emptyHint={hint}
                  />
                  <DetailField
                    label="Site"
                    mono
                    value={formatIdNumber(selected.site ?? null, idBase)}
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
          {selected.neighbors && selected.neighbors.length > 0 ? (
            <div>
              <p className="text-xs uppercase tracking-wider text-muted mb-2">
                Neighbor sites ({selected.neighbors.length})
              </p>
              <DetailField
                label="RFSS / Site → downlink"
                mono
                value={selected.neighbors
                  .map((n) => formatNeighbor(n, idBase))
                  .join("\n")}
              />
            </div>
          ) : null}
        </DetailModal>
      )}
    </div>
  );
}
