import { useMemo, useState } from "react";
import { fetchMDC1200Messages, type MDC1200Message } from "../api/mdc1200";
import { Column, DataTable } from "../components/DataTable";
import { PageHeader } from "../components/ui/PageHeader";
import { Badge } from "../components/ui/Badge";
import { StaleIndicator } from "../components/ui/StaleIndicator";
import { useDataPoll } from "../hooks/useDataPoll";
import { selectClientConfig, useShared } from "../store/shared";
import { formatClock } from "../lib/formatTime";

// MDC1200 panel — list of recent decoded Motorola FFSK signaling
// bursts off conventional analog voice channels. Each row shows the
// transmitting radio's unit ID, the decoded operation (PTT ID,
// emergency, status, radio check, ...) and whether the CRC validated.
//
// Polls /api/v1/mdc1200/messages every 5 s.

const POLL_INTERVAL_MS = 5_000;

export function MDC1200() {
  const cfg = useShared(selectClientConfig);
  const [messages, setMessages] = useState<MDC1200Message[]>([]);

  const { loading, error, stale, lastUpdated } = useDataPoll({
    fetcher: () => fetchMDC1200Messages(cfg, 200),
    onData: setMessages,
    intervalMs: POLL_INTERVAL_MS,
    resetKey: cfg.baseURL,
  });

  const columns: Column<MDC1200Message>[] = useMemo(
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
        key: "unit",
        header: "Unit ID",
        render: (m) => (
          <span className="font-mono text-accent">{unitHex(m.unit_id)}</span>
        ),
        sort: (a, b) => a.unit_id - b.unit_id,
      },
      {
        key: "operation",
        header: "Operation",
        render: (m) => (
          <span className="inline-flex items-center gap-1">
            {m.operation || <span className="text-muted">unknown</span>}
            {m.operation === "Emergency" && <Badge tone="err">!</Badge>}
          </span>
        ),
        sort: (a, b) => (a.operation ?? "").localeCompare(b.operation ?? ""),
      },
      {
        key: "oparg",
        header: "Op / Arg",
        render: (m) => (
          <span className="font-mono text-muted">
            {hex2(m.op)} / {hex2(m.arg)}
          </span>
        ),
      },
      {
        key: "body",
        header: "Body",
        render: (m) => m.body || <span className="text-muted">—</span>,
      },
      {
        key: "crc",
        header: "CRC",
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
        title="MDC1200"
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
        tableId="mdc1200"
        pageSize={50}
        loading={loading}
        searchable
        searchAccessor={(m) =>
          [unitHex(m.unit_id), m.operation, m.body].filter(Boolean).join(" ")
        }
        searchPlaceholder="Search by unit ID, operation…"
        emptyMessage={
          <>
            No MDC1200 bursts yet. Add an{" "}
            <code className="text-accent">mdc1200.channels</code> entry to your
            config (a conventional analog VHF / UHF voice channel carrying
            Motorola signaling) and decoded unit IDs, PTT ANI, emergency / status
            / radio-check bursts will land here as they arrive.
          </>
        }
      />
    </div>
  );
}

function unitHex(id: number): string {
  return id.toString(16).toUpperCase().padStart(4, "0");
}

function hex2(v: number): string {
  return "0x" + v.toString(16).toUpperCase().padStart(2, "0");
}

const formatTime = formatClock;
