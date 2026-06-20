import { useMemo, useState } from "react";
import { fetchAISVessels, type AISMessage } from "../api/ais";
import { PositionMap, type MapPoint } from "../components/PositionMap";
import { Column, DataTable } from "../components/DataTable";
import { PageHeader } from "../components/ui/PageHeader";
import { Badge } from "../components/ui/Badge";
import { StaleIndicator } from "../components/ui/StaleIndicator";
import { useDataPoll } from "../hooks/useDataPoll";
import { selectClientConfig, useShared } from "../store/shared";

// AIS panel — list of recent decoded marine-AIS messages. Each row
// shows the MMSI, the message-type tag, a short body summary, and
// either lat/lon + SOG/COG (position-bearing types) or vessel-name /
// call-sign / destination (static types).
//
// Polls /api/v1/ais/vessels every 5 s.

const POLL_INTERVAL_MS = 5_000;

const dash = <span className="text-muted">—</span>;

export function AIS() {
  const cfg = useShared(selectClientConfig);
  const [messages, setMessages] = useState<AISMessage[]>([]);

  const { loading, error, stale, lastUpdated } = useDataPoll({
    fetcher: () => fetchAISVessels(cfg, 200),
    onData: setMessages,
    intervalMs: POLL_INTERVAL_MS,
    resetKey: cfg.baseURL,
  });

  const mapPoints = useMemo<MapPoint[]>(
    () =>
      messages
        .filter(
          (m) =>
            m.has_position &&
            typeof m.latitude === "number" &&
            typeof m.longitude === "number",
        )
        .map((m) => ({
          id: `ais-${m.id}`,
          latitude: m.latitude as number,
          longitude: m.longitude as number,
          kind: "ais" as const,
          label: m.vessel_name || String(m.mmsi).padStart(9, "0"),
          detail: m.body ?? "",
        })),
    [messages],
  );

  const columns: Column<AISMessage>[] = useMemo(
    () => [
      {
        key: "received",
        header: "Received",
        render: (m) => (
          <span className="font-mono text-muted">{formatTime(m.received_at)}</span>
        ),
        sort: (a, b) => a.received_at.localeCompare(b.received_at),
      },
      {
        key: "mmsi",
        header: "MMSI",
        render: (m) => (
          <span className="font-mono">
            <span className="inline-flex items-center gap-1">
              <span className="text-accent">{m.mmsi}</span>
              {!m.fcs_ok && <Badge tone="warn">fcs</Badge>}
            </span>
            {m.vessel_name && (
              <span className="block text-[10px] text-muted">{m.vessel_name}</span>
            )}
          </span>
        ),
        sort: (a, b) => a.mmsi - b.mmsi,
      },
      {
        key: "type",
        header: "Type",
        render: (m) => <span className="uppercase">{m.type}</span>,
        sort: (a, b) => a.type.localeCompare(b.type),
      },
      {
        key: "body",
        header: "Body",
        render: (m) =>
          m.body || <span className="text-muted">(no payload)</span>,
      },
      {
        key: "pos",
        header: "Lat / Lon",
        className: "text-right font-mono",
        headerClassName: "text-right",
        render: (m) =>
          m.has_position ? (
            <span>
              {m.latitude?.toFixed(4)}, {m.longitude?.toFixed(4)}
            </span>
          ) : (
            dash
          ),
      },
      {
        key: "nav",
        header: "SOG / COG",
        className: "text-right font-mono",
        headerClassName: "text-right",
        render: (m) =>
          m.has_position && (m.sog || m.cog) ? (
            <span>
              {(m.sog ?? 0).toFixed(1)}kn / {(m.cog ?? 0).toFixed(0)}°
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
        title="AIS"
        actions={
          <>
            <StaleIndicator stale={stale} lastUpdated={lastUpdated} />
            <span className="text-xs text-muted">
              {messages.length} message{messages.length === 1 ? "" : "s"}
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
        rows={messages}
        columns={columns}
        rowKey={(m) => String(m.id)}
        defaultSortKey="received"
        defaultSortDirection="desc"
        tableId="ais"
        pageSize={50}
        loading={loading}
        searchable
        searchAccessor={(m) =>
          [m.mmsi, m.vessel_name, m.type, m.body].filter(Boolean).join(" ")
        }
        searchPlaceholder="Search by MMSI, name, type…"
        emptyMessage={
          <>
            No AIS messages yet. Add an{" "}
            <code className="text-accent">ais.channels</code> entry to your
            config (marine VHF 87B = 161.975 MHz · 88B = 162.025 MHz) and decoded
            vessels — position reports, static + voyage data, base-station
            reports — will land here as they arrive.
          </>
        }
      />
    </div>
  );
}

function formatTime(ts: string): string {
  try {
    return new Date(ts).toISOString().slice(11, 19);
  } catch {
    return ts.slice(11, 19);
  }
}
