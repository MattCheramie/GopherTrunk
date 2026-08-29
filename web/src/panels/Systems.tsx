import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import { Column, DataTable } from "../components/DataTable";
import { DetailField, DetailModal } from "../components/DetailModal";
import { PageHeader } from "../components/ui/PageHeader";
import { Section } from "../components/ui/Section";
import { StaleIndicator } from "../components/ui/StaleIndicator";
import { PositionMap, type MapPoint } from "../components/PositionMap";
import { useDataPoll } from "../hooks/useDataPoll";
import { formatIdNumber } from "../lib/idFormat";
import type {
  BandPlanSlotDTO,
  DMRBandPlanDTO,
  DMRBandPlanLearnedDTO,
  EventDTO,
  NeighborDTO,
  SiteChannelDTO,
  SiteDTO,
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
  if (n.channel_id || n.channel_number) {
    s += ` · CH ${n.channel_id ?? 0}-${n.channel_number ?? 0}`;
  }
  if (n.downlink_hz) {
    s += ` → ${(n.downlink_hz / 1e6).toFixed(6)} MHz`;
    if (n.uplink_hz) {
      s += ` / ↑ ${(n.uplink_hz / 1e6).toFixed(6)} MHz`;
    }
  }
  // CFVA flags from the adjacent-status broadcast — SDRtrunk shows these as
  // STATUS:[VALID INFORMATION] etc.; "none" means observed all-clear.
  if (n.status) {
    s += ` [${n.status}]`;
  }
  return s;
}

// formatSiteChannel renders one advertised control channel of the camped site
// ("CH 2-1620 → 450.125000 MHz / ↑ 460.687500 MHz").
function formatSiteChannel(c: SiteChannelDTO): string {
  let s = `CH ${c.channel_id ?? 0}-${c.channel_number ?? 0}`;
  if (c.downlink_hz) {
    s += ` → ${(c.downlink_hz / 1e6).toFixed(6)} MHz`;
    if (c.uplink_hz) {
      s += ` / ↑ ${(c.uplink_hz / 1e6).toFixed(6)} MHz`;
    }
  }
  return s;
}

// formatBand renders one decoded IDEN_UP slot the way SDRtrunk's "Details" tab
// shows the band plan: base downlink frequency, channel spacing, and the signed
// transmit offset, tagging TDMA bands. Offset is omitted when zero/unknown.
function formatBand(b: BandPlanSlotDTO): string {
  let s = `BAND ${b.channel_id}`;
  if (b.access_tdma) s += " TDMA";
  s += ` · BASE ${(b.base_hz / 1e6).toFixed(6)} MHz`;
  s += ` · SPACING ${b.spacing_hz} Hz`;
  if (b.tx_offset_hz)
    s += ` · OFFSET ${(b.tx_offset_hz / 1e6).toFixed(6)} MHz`;
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
  const [sites, setSites] = useState<SiteDTO[]>([]);

  // Poll the scanner snapshot alongside systems so the detail modal can
  // translate empty WACN/SystemID/RFSS/Site into a hunt-state hint even
  // when the Scanner panel isn't mounted. Sites ride along for the map.
  const { stale, lastUpdated } = useDataPoll({
    fetcher: async () => {
      const [sysRes, scanRes, siteRes] = await Promise.allSettled([
        api.systems(cfg),
        api.scanner(cfg),
        api.sites(cfg),
      ]);
      if (sysRes.status === "rejected" && scanRes.status === "rejected") {
        throw sysRes.reason;
      }
      return {
        systems: sysRes.status === "fulfilled" ? sysRes.value : null,
        scanner: scanRes.status === "fulfilled" ? scanRes.value : null,
        sites: siteRes.status === "fulfilled" ? siteRes.value : null,
      };
    },
    onData: (d) => {
      if (d.systems) setSystems(d.systems);
      if (d.scanner) setScanner(d.scanner);
      if (d.sites) setSites(d.sites);
    },
    intervalMs: POLL_INTERVAL_MS,
    resetKey: cfg.baseURL,
  });

  // Sites with a known position become map markers, one per (system, rfss,
  // site). Sites without coordinates (most, until configured or RR-imported)
  // are simply omitted, so the map hides entirely when none are located.
  const sitePoints = useMemo<MapPoint[]>(() => {
    return sites
      .filter(
        (s) =>
          typeof s.latitude === "number" &&
          typeof s.longitude === "number" &&
          (s.latitude !== 0 || s.longitude !== 0),
      )
      .map((s) => ({
        id: `${s.system}-${s.rfss_id}-${s.site_id}`,
        latitude: s.latitude as number,
        longitude: s.longitude as number,
        kind: "site" as const,
        label: s.name || `RFSS ${s.rfss_id} / Site ${s.site_id}`,
        detail: `${s.system}${s.control_channel_hz ? ` · ${(s.control_channel_hz / 1e6).toFixed(4)} MHz` : ""}`,
      }));
  }, [sites]);

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

      {sitePoints.length > 0 && (
        <Section
          id="site-map"
          title={`Site map (${sitePoints.length})`}
          description="Located trunked sites (from config or RadioReference import). Sites without coordinates are listed below but not mapped."
        >
          <PositionMap points={sitePoints} heightPx={320} />
        </Section>
      )}

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
          <Link
            to={`/history?system=${encodeURIComponent(selected.name)}`}
            className="inline-block text-sm text-accent hover:underline"
          >
            View calls on this system →
          </Link>
          {(() => {
            const hunt = scanner?.systems?.find((h) => h.name === selected.name);
            // TETRA has no WACN/RFSS/Site; it identifies a cell by MNI (MCC/MNC),
            // Location Area, and colour code, all decoded from MLE-SYSINFO. Render
            // those instead of the P25 fields, which would sit empty forever.
            if (selected.protocol === "tetra") {
              const tHint =
                hunt?.state === "hunting"
                  ? "Hunting control channel"
                  : "Awaiting SYSINFO broadcast";
              const has = selected.has_tetra_identity ?? false;
              return (
                <div>
                  <p className="text-xs uppercase tracking-wider text-muted mb-2">
                    Network identity (decoded live)
                  </p>
                  <div className="grid grid-cols-2 gap-3">
                    <DetailField
                      label="MNI (MCC / MNC)"
                      mono
                      value={
                        has
                          ? `${selected.tetra_mcc ?? 0} / ${selected.tetra_mnc ?? 0}`
                          : null
                      }
                      emptyHint={tHint}
                    />
                    <DetailField
                      label="Location area"
                      mono
                      value={has ? String(selected.tetra_location_area ?? 0) : null}
                      emptyHint={tHint}
                    />
                    <DetailField
                      label="Colour code"
                      mono
                      value={
                        has ? String(selected.tetra_decoded_colour_code ?? 0) : null
                      }
                      emptyHint={tHint}
                    />
                  </div>
                </div>
              );
            }
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
                  <DetailField
                    label="NAC"
                    mono
                    value={formatIdNumber(selected.nac ?? null, idBase)}
                    emptyHint={hint}
                  />
                  <DetailField
                    label="LRA"
                    mono
                    value={formatIdNumber(selected.lra ?? null, idBase)}
                    emptyHint={hint}
                  />
                </div>
              </div>
            );
          })()}
          {selected.primary_control_channel ||
          (selected.secondary_control_channels &&
            selected.secondary_control_channels.length > 0) ? (
            <div>
              <p className="text-xs uppercase tracking-wider text-muted mb-2">
                Site control channels (decoded live)
              </p>
              <DetailField
                label="Primary / secondaries → downlink / uplink"
                mono
                value={[
                  ...(selected.primary_control_channel
                    ? [`PRI ${formatSiteChannel(selected.primary_control_channel)}`]
                    : []),
                  ...(selected.secondary_control_channels ?? []).map(
                    (c) => `SEC ${formatSiteChannel(c)}`,
                  ),
                ].join("\n")}
              />
            </div>
          ) : null}
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
                label="RFSS / Site · channel → downlink / uplink"
                mono
                value={selected.neighbors
                  .map((n) => formatNeighbor(n, idBase))
                  .join("\n")}
              />
            </div>
          ) : null}
          {selected.frequency_bands && selected.frequency_bands.length > 0 ? (
            <div>
              <p className="text-xs uppercase tracking-wider text-muted mb-2">
                Frequency bands ({selected.frequency_bands.length})
              </p>
              <DetailField
                label="Band → base / spacing / offset"
                mono
                value={selected.frequency_bands
                  .map((b) => formatBand(b))
                  .join("\n")}
              />
            </div>
          ) : null}
        </DetailModal>
      )}
    </div>
  );
}
