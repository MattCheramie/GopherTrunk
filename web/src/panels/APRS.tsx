import { useMemo, useState } from "react";
import { fetchAPRSPackets, type APRSPacket } from "../api/aprs";
import { PositionMap, type MapPoint } from "../components/PositionMap";
import { Column, DataTable } from "../components/DataTable";
import { PageHeader } from "../components/ui/PageHeader";
import { Badge } from "../components/ui/Badge";
import { StaleIndicator } from "../components/ui/StaleIndicator";
import { useDataPoll } from "../hooks/useDataPoll";
import { selectClientConfig, useShared } from "../store/shared";
import { formatClock } from "../lib/formatTime";

// APRS panel — list of recent decoded APRS / AX.25 packets. Each row
// shows the AX.25 envelope (src → dst + path), the decoded APRS
// sub-type, and a one-line summary body. Positions get a coordinate
// column.
//
// Polls /api/v1/aprs/packets every 5 s.

const POLL_INTERVAL_MS = 5_000;

const dash = <span className="text-muted">—</span>;

export function APRS() {
  const cfg = useShared(selectClientConfig);
  const [packets, setPackets] = useState<APRSPacket[]>([]);

  const { loading, error, stale, lastUpdated } = useDataPoll({
    fetcher: () => fetchAPRSPackets(cfg, 200),
    onData: setPackets,
    intervalMs: POLL_INTERVAL_MS,
    resetKey: cfg.baseURL,
  });

  const mapPoints = useMemo<MapPoint[]>(
    () =>
      packets
        .filter(
          (p) =>
            typeof p.latitude === "number" &&
            typeof p.longitude === "number" &&
            !(p.latitude === 0 && p.longitude === 0),
        )
        .map((p) => ({
          id: `aprs-${p.id}`,
          latitude: p.latitude as number,
          longitude: p.longitude as number,
          kind: "aprs" as const,
          label: p.src,
          detail: `${p.type} · ${p.body ?? ""}`.trim(),
        })),
    [packets],
  );

  const columns: Column<APRSPacket>[] = useMemo(
    () => [
      {
        key: "received",
        header: "Received",
        render: (p) => (
          <span className="font-mono text-muted">{formatTime(p.received_at)}</span>
        ),
        sort: (a, b) => a.received_at.localeCompare(b.received_at),
      },
      {
        key: "env",
        header: "Src → Dst",
        render: (p) => (
          <span className="font-mono">
            <span className="inline-flex items-center gap-1">
              <span className="text-accent">{p.src}</span>
              <span className="text-muted"> → {p.dst}</span>
              {!p.fcs_ok && <Badge tone="warn">fcs</Badge>}
            </span>
            {p.path && (
              <span className="block text-[10px] text-muted">via {p.path}</span>
            )}
          </span>
        ),
        sort: (a, b) => a.src.localeCompare(b.src),
      },
      {
        key: "type",
        header: "Type",
        render: (p) => <span className="uppercase">{p.type}</span>,
        sort: (a, b) => a.type.localeCompare(b.type),
      },
      {
        key: "body",
        header: "Body",
        render: (p) => p.body || <span className="text-muted">(no payload)</span>,
      },
      {
        key: "pos",
        header: "Lat / Lon",
        className: "text-right font-mono",
        headerClassName: "text-right",
        render: (p) =>
          p.latitude || p.longitude ? (
            <span>
              {p.latitude?.toFixed(4)}, {p.longitude?.toFixed(4)}
            </span>
          ) : (
            dash
          ),
      },
    ],
    [],
  );

  return (
    <div className="space-y-3">
      <PageHeader
        title="APRS"
        actions={
          <>
            <StaleIndicator stale={stale} lastUpdated={lastUpdated} />
            <span className="text-xs text-muted">
              {packets.length} packet{packets.length === 1 ? "" : "s"}
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
        rows={packets}
        columns={columns}
        rowKey={(p) => String(p.id)}
        defaultSortKey="received"
        defaultSortDirection="desc"
        tableId="aprs"
        pageSize={50}
        loading={loading}
        searchable
        searchAccessor={(p) =>
          [p.src, p.dst, p.path, p.type, p.body].filter(Boolean).join(" ")
        }
        searchPlaceholder="Search by callsign, type, body…"
        emptyMessage={
          <>
            No APRS packets yet. Add an{" "}
            <code className="text-accent">aprs.channels</code> entry to your
            config (NA primary: 144.39 MHz · EU R1: 144.800 MHz · JP: 144.64 MHz
            · ISS: 145.825 MHz) and decoded packets — position beacons, messages,
            bulletins, status, Mic-E — will land here as they arrive.
          </>
        }
      />
    </div>
  );
}

const formatTime = formatClock;
