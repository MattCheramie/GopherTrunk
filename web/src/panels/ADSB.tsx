import { useMemo, useState } from "react";
import { fetchAircraftReports, type AircraftReport } from "../api/adsb";
import { PositionMap, type MapPoint } from "../components/PositionMap";
import { Column, DataTable } from "../components/DataTable";
import { PageHeader } from "../components/ui/PageHeader";
import { Badge } from "../components/ui/Badge";
import { StaleIndicator } from "../components/ui/StaleIndicator";
import { useDataPoll } from "../hooks/useDataPoll";
import { selectClientConfig, useShared } from "../store/shared";
import { formatClock } from "../lib/formatTime";

// ADS-B panel — list of recent decoded Mode-S frames. Each row
// shows the ICAO 24-bit address (hex form, the standard "tail
// identifier" aviation people use), the kind (ident / airborne-
// pos / velocity / ...), and kind-specific data: callsign for
// identification rows, lat/lon + altitude for position rows,
// ground speed + track + vertical rate for velocity rows.
//
// Polls /api/v1/adsb/aircraft every 5 s. ADS-B is the highest-
// rate decoder (each aircraft sends 2-3 msg/s); the default
// limit shows the most recent 200, which is roughly 1 min of
// traffic on a busy channel.

const POLL_INTERVAL_MS = 5_000;

const dash = <span className="text-muted">—</span>;

export function ADSB() {
  const cfg = useShared(selectClientConfig);
  const [reports, setReports] = useState<AircraftReport[]>([]);

  const { loading, error, stale, lastUpdated } = useDataPoll({
    fetcher: () => fetchAircraftReports(cfg, 200),
    onData: setReports,
    intervalMs: POLL_INTERVAL_MS,
    resetKey: cfg.baseURL,
  });

  const mapPoints = useMemo<MapPoint[]>(
    () =>
      reports
        .filter(
          (r) =>
            r.has_position &&
            typeof r.latitude === "number" &&
            typeof r.longitude === "number",
        )
        .map((r) => ({
          id: `adsb-${r.id}`,
          latitude: r.latitude as number,
          longitude: r.longitude as number,
          kind: "adsb" as const,
          label: r.callsign?.trim() || r.icao_hex,
          detail: r.has_altitude
            ? `${r.altitude_ft?.toLocaleString()} ft`
            : undefined,
        })),
    [reports],
  );

  const columns: Column<AircraftReport>[] = useMemo(
    () => [
      {
        key: "received",
        header: "Received",
        render: (r) => (
          <span className="font-mono text-muted">{formatTime(r.received_at)}</span>
        ),
        sort: (a, b) => a.received_at.localeCompare(b.received_at),
      },
      {
        key: "icao",
        header: "ICAO",
        render: (r) => (
          <span className="inline-flex items-center gap-1">
            <span className="font-mono text-accent">{r.icao_hex}</span>
            {!r.crc_valid && <Badge tone="warn">crc</Badge>}
          </span>
        ),
        sort: (a, b) => a.icao_hex.localeCompare(b.icao_hex),
      },
      {
        key: "kind",
        header: "Kind",
        render: (r) => <span className="uppercase">{r.kind}</span>,
        sort: (a, b) => a.kind.localeCompare(b.kind),
      },
      {
        key: "callsign",
        header: "Callsign",
        render: (r) =>
          r.callsign ? <span className="font-mono">{r.callsign}</span> : dash,
      },
      {
        key: "pos",
        header: "Lat / Lon",
        className: "text-right",
        headerClassName: "text-right",
        render: (r) =>
          r.has_position && r.latitude !== undefined && r.longitude !== undefined ? (
            <span className="font-mono">
              {r.latitude.toFixed(4)}, {r.longitude.toFixed(4)}
            </span>
          ) : (
            dash
          ),
      },
      {
        key: "alt",
        header: "Alt (ft)",
        className: "text-right font-mono",
        headerClassName: "text-right",
        render: (r) =>
          r.has_altitude && r.altitude_ft !== undefined
            ? r.altitude_ft.toLocaleString()
            : dash,
        sort: (a, b) => (a.altitude_ft ?? 0) - (b.altitude_ft ?? 0),
      },
      {
        key: "vel",
        header: "GS / Track",
        className: "text-right font-mono",
        headerClassName: "text-right",
        render: (r) =>
          r.ground_speed_kn ? (
            <span>
              {r.ground_speed_kn}kn / {(r.track_deg ?? 0).toFixed(0)}°
            </span>
          ) : (
            dash
          ),
      },
      {
        key: "vr",
        header: "VR (fpm)",
        className: "text-right font-mono",
        headerClassName: "text-right",
        render: (r) =>
          r.vertical_rate_fpm
            ? r.vertical_rate_fpm > 0
              ? `+${r.vertical_rate_fpm}`
              : String(r.vertical_rate_fpm)
            : dash,
      },
    ],
    [],
  );

  return (
    <div className="space-y-3">
      <PageHeader
        title="ADS-B"
        actions={
          <>
            <StaleIndicator stale={stale} lastUpdated={lastUpdated} />
            <span className="text-xs text-muted">
              {reports.length} message{reports.length === 1 ? "" : "s"}
            </span>
          </>
        }
      />

      {mapPoints.length > 0 && <PositionMap points={mapPoints} />}

      {error && !stale && (
        <div
          role="alert"
          className="rounded-md border border-err/40 bg-err/15 px-3 py-2 text-sm text-err"
        >
          {error}
        </div>
      )}

      <DataTable
        rows={reports}
        columns={columns}
        rowKey={(r) => String(r.id)}
        defaultSortKey="received"
        defaultSortDirection="desc"
        tableId="adsb"
        pageSize={50}
        loading={loading}
        searchable
        searchAccessor={(r) =>
          [r.icao_hex, r.kind, r.callsign].filter(Boolean).join(" ")
        }
        searchPlaceholder="Search by ICAO, kind, callsign…"
        emptyMessage={
          <>
            No ADS-B messages yet. Add an{" "}
            <code className="text-accent">adsb.channels</code> entry to your
            config (1090.000 MHz centre, 1 Msps required — dedicate one RTL-SDR
            with a 1090 MHz antenna to it) and decoded aircraft will land here as
            they arrive.
          </>
        }
      />
    </div>
  );
}

const formatTime = formatClock;
