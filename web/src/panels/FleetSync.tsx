import { useMemo, useState } from "react";
import { fetchFleetSyncMessages, type FleetSyncMessage } from "../api/fleetsync";
import { Column, DataTable } from "../components/DataTable";
import { PageHeader } from "../components/ui/PageHeader";
import { Badge } from "../components/ui/Badge";
import { StaleIndicator } from "../components/ui/StaleIndicator";
import { useDataPoll } from "../hooks/useDataPoll";
import { selectClientConfig, useShared } from "../store/shared";
import { formatClock } from "../lib/formatTime";

// FleetSync panel — list of recent decoded Kenwood FFSK ANI bursts off
// conventional analog voice channels. Each row shows the transmitting
// radio's fleet and unit ID, the channel (frequency + SDR serial) that
// produced it, which protocol variant (FleetSync I or the ECC-protected
// FleetSync II) decoded it, and whether the block check validated.
//
// Polls /api/v1/fleetsync/messages every 5 s.

const POLL_INTERVAL_MS = 5_000;

export function FleetSync() {
  const cfg = useShared(selectClientConfig);
  const [messages, setMessages] = useState<FleetSyncMessage[]>([]);

  const { loading, error, stale, lastUpdated } = useDataPoll({
    fetcher: () => fetchFleetSyncMessages(cfg, 200),
    onData: setMessages,
    intervalMs: POLL_INTERVAL_MS,
    resetKey: cfg.baseURL,
  });

  const columns: Column<FleetSyncMessage>[] = useMemo(
    () => [
      {
        key: "received",
        header: "Received",
        render: (m) => (
          <span className="font-mono text-muted">{formatClock(m.received_at)}</span>
        ),
        sort: (a, b) => a.received_at.localeCompare(b.received_at),
      },
      {
        key: "fleet",
        header: "Fleet",
        render: (m) => <span className="font-mono">{m.fleet}</span>,
        sort: (a, b) => a.fleet - b.fleet,
      },
      {
        key: "unit",
        header: "Unit ID",
        render: (m) => <span className="font-mono text-accent">{m.unit}</span>,
        sort: (a, b) => a.unit - b.unit,
      },
      {
        key: "channel",
        header: "Channel",
        render: (m) =>
          m.frequency_hz ? (
            <span className="font-mono" title={m.serial ? `SDR ${m.serial}` : undefined}>
              {(m.frequency_hz / 1e6).toFixed(4)} MHz
              {m.serial ? <span className="text-muted"> · {m.serial}</span> : null}
            </span>
          ) : (
            <span className="text-muted">—</span>
          ),
        sort: (a, b) => (a.frequency_hz ?? 0) - (b.frequency_hz ?? 0),
      },
      {
        key: "variant",
        header: "Variant",
        render: (m) => (
          <Badge tone="neutral" title={m.fs2 ? "FleetSync II (ECC)" : "FleetSync I"}>
            {m.fs2 ? "FS-II" : "FS-I"}
          </Badge>
        ),
        sort: (a, b) => Number(a.fs2) - Number(b.fs2),
      },
      {
        key: "raw",
        header: "Raw words",
        render: (m) =>
          m.raw_hex ? (
            <span className="font-mono text-muted">{m.raw_hex}</span>
          ) : (
            <span className="text-muted">—</span>
          ),
      },
      {
        key: "crc",
        header: "Check",
        className: "text-right",
        headerClassName: "text-right",
        render: (m) =>
          m.crc_ok ? <Badge tone="ok">ok</Badge> : <Badge tone="err">fail</Badge>,
      },
    ],
    [],
  );

  return (
    <div className="space-y-3">
      <PageHeader
        title="FleetSync"
        actions={
          <>
            <StaleIndicator stale={stale} lastUpdated={lastUpdated} />
            <span className="text-xs text-muted">
              {messages.length} burst{messages.length === 1 ? "" : "s"}
            </span>
          </>
        }
      />

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
        tableId="fleetsync"
        pageSize={50}
        loading={loading}
        searchable
        searchAccessor={(m) =>
          [String(m.fleet), String(m.unit), m.body, m.raw_hex].filter(Boolean).join(" ")
        }
        searchPlaceholder="Search by fleet, unit ID…"
        emptyMessage={
          <>
            No FleetSync bursts yet. Add a{" "}
            <code className="text-accent">fleetsync.channels</code> entry to your
            config (a conventional analog VHF / UHF voice channel your Kenwood
            fleet talks on) and each radio's fleet / unit ANI will land here as
            it keys up.
          </>
        }
      />
    </div>
  );
}
