import { useMemo, useState } from "react";
import { fetchDSCMessages, type DSCMessage } from "../api/dsc";
import { PositionMap, type MapPoint } from "../components/PositionMap";
import { Column, DataTable } from "../components/DataTable";
import { PageHeader } from "../components/ui/PageHeader";
import { Badge } from "../components/ui/Badge";
import { StaleIndicator } from "../components/ui/StaleIndicator";
import { useDataPoll } from "../hooks/useDataPoll";
import { selectClientConfig, useShared } from "../store/shared";
import { formatClock } from "../lib/formatTime";

// DSC panel — list of recent decoded marine DSC sequences. The
// category badge reflects priority: distress = red, urgency = amber,
// safety/routine = neutral. Distress alerts surface nature and (when
// present) a position.
//
// Polls /api/v1/dsc/messages every 5 s.

const POLL_INTERVAL_MS = 5_000;

const dash = <span className="text-muted">—</span>;

function categoryTone(category: string): "ok" | "warn" | "err" | "neutral" {
  switch (category) {
    case "distress":
      return "err";
    case "urgency":
      return "warn";
    default:
      return "neutral";
  }
}

export function DSC() {
  const cfg = useShared(selectClientConfig);
  const [messages, setMessages] = useState<DSCMessage[]>([]);

  const { loading, error, stale, lastUpdated } = useDataPoll({
    fetcher: () => fetchDSCMessages(cfg, 200),
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
          id: `dsc-${m.id}`,
          latitude: m.latitude as number,
          longitude: m.longitude as number,
          kind: "dsc-distress" as const,
          label: `MMSI ${String(m.self_mmsi).padStart(9, "0")}`,
          detail: m.nature || m.format,
        })),
    [messages],
  );

  const columns: Column<DSCMessage>[] = useMemo(
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
        key: "format",
        header: "Format",
        render: (m) => <span className="uppercase">{m.format}</span>,
        sort: (a, b) => a.format.localeCompare(b.format),
      },
      {
        key: "category",
        header: "Category",
        render: (m) => (
          <Badge tone={categoryTone(m.category)}>
            <span className="uppercase">{m.category}</span>
          </Badge>
        ),
        sort: (a, b) => a.category.localeCompare(b.category),
      },
      {
        key: "self",
        header: "Self MMSI",
        render: (m) => (
          <span className="font-mono text-accent">
            {String(m.self_mmsi).padStart(9, "0")}
          </span>
        ),
      },
      {
        key: "target",
        header: "Target / Nature",
        render: (m) =>
          m.target_mmsi ? (
            <span className="font-mono">
              {String(m.target_mmsi).padStart(9, "0")}
            </span>
          ) : (
            m.nature || dash
          ),
      },
      {
        key: "body",
        header: "Body",
        render: (m) => (
          <span>
            {m.body || <span className="text-muted">(no payload)</span>}
            {m.time_utc && (
              <span className="block text-[10px] text-muted">UTC {m.time_utc}</span>
            )}
          </span>
        ),
      },
      {
        key: "pos",
        header: "Lat / Lon",
        className: "text-right font-mono",
        headerClassName: "text-right",
        render: (m) =>
          m.has_position && m.latitude !== undefined && m.longitude !== undefined ? (
            <span>
              {m.latitude.toFixed(4)}, {m.longitude.toFixed(4)}
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
        title="DSC"
        actions={
          <>
            <StaleIndicator stale={stale} lastUpdated={lastUpdated} />
            <span className="text-xs text-muted">
              {messages.length} sequence{messages.length === 1 ? "" : "s"}
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
        tableId="dsc"
        pageSize={50}
        loading={loading}
        searchable
        searchAccessor={(m) =>
          [m.format, m.category, m.self_mmsi, m.target_mmsi, m.nature, m.body]
            .filter(Boolean)
            .join(" ")
        }
        searchPlaceholder="Search by MMSI, category, nature…"
        emptyMessage={
          <>
            No DSC sequences yet. Add a{" "}
            <code className="text-accent">dsc.channels</code> entry to your config
            (marine VHF channel 70 = 156.525 MHz · HF 2.187.5 / 8.414.5 / 12.577
            / 16.804.5 kHz) and decoded distress / safety / routine calls will
            land here as they arrive.
          </>
        }
      />
    </div>
  );
}

const formatTime = formatClock;
