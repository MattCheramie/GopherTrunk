import { useMemo, useState } from "react";
import {
  fetchPagerMessages,
  functionLabel,
  type PagerMessage,
} from "../api/pagers";
import { Column, DataTable } from "../components/DataTable";
import { PageHeader } from "../components/ui/PageHeader";
import { StaleIndicator } from "../components/ui/StaleIndicator";
import { useDataPoll } from "../hooks/useDataPoll";
import { selectClientConfig, useShared } from "../store/shared";

// Pagers panel — list of recent POCSAG and FLEX pages decoded by the
// daemon. Each row carries its protocol, RIC, function code, encoding,
// decoded body, and the BCH bit-error count (non-zero = marginal).
//
// Polls /api/v1/pager/messages every 5 s.

const POLL_INTERVAL_MS = 5_000;

export function Pagers() {
  const cfg = useShared(selectClientConfig);
  const [messages, setMessages] = useState<PagerMessage[]>([]);

  const { loading, error, stale, lastUpdated } = useDataPoll({
    fetcher: () => fetchPagerMessages(cfg, 200),
    onData: setMessages,
    intervalMs: POLL_INTERVAL_MS,
    resetKey: cfg.baseURL,
  });

  const columns: Column<PagerMessage>[] = useMemo(
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
        key: "type",
        header: "Type",
        render: (m) => <ProtocolBadge protocol={m.protocol} />,
        sort: (a, b) => (a.protocol ?? "").localeCompare(b.protocol ?? ""),
      },
      {
        key: "ric",
        header: "RIC",
        className: "text-right font-mono text-accent",
        headerClassName: "text-right",
        render: (m) => m.ric,
        sort: (a, b) => a.ric - b.ric,
      },
      {
        key: "fn",
        header: "Fn",
        render: (m) => functionLabel(m.func),
      },
      {
        key: "enc",
        header: "Enc",
        render: (m) => (
          <span className="uppercase text-muted">{m.encoding || "?"}</span>
        ),
      },
      {
        key: "body",
        header: "Body",
        render: (m) => m.body || <span className="text-muted">(empty)</span>,
      },
      {
        key: "ber",
        header: "BER",
        className: "text-right font-mono",
        headerClassName: "text-right",
        render: (m) =>
          m.corrected > 0 ? (
            <span className="text-warn">{m.corrected}</span>
          ) : (
            <span className="text-muted">0</span>
          ),
        sort: (a, b) => a.corrected - b.corrected,
      },
    ],
    [],
  );

  return (
    <div className="space-y-3">
      <PageHeader
        title="Pagers"
        actions={
          <>
            <StaleIndicator stale={stale} lastUpdated={lastUpdated} />
            <span className="text-xs text-muted">
              {messages.length} message{messages.length === 1 ? "" : "s"}
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
        tableId="pagers"
        pageSize={50}
        loading={loading}
        searchable
        searchAccessor={(m) =>
          [m.protocol, m.ric, m.body, functionLabel(m.func)]
            .filter(Boolean)
            .join(" ")
        }
        searchPlaceholder="Search by RIC, protocol, body…"
        emptyMessage={
          <>
            No pager messages yet. POCSAG pages decoded by the daemon will appear
            here as they arrive — typical workflow is to point one SDR at the
            local paging frequency (e.g. 152.0075 MHz commercial paging, the
            local fire department dispatch tone-out freq, or 439.9875 MHz
            DAPNET).
          </>
        }
      />
    </div>
  );
}

// ProtocolBadge renders the page's signalling protocol as a small colored
// pill so POCSAG and FLEX rows are distinguishable at a glance.
function ProtocolBadge({ protocol }: { protocol: string }) {
  const proto = (protocol || "pocsag").toLowerCase();
  const label = proto.toUpperCase();
  const cls =
    proto === "flex"
      ? "bg-purple-900/40 text-purple-200 border-purple-700/40"
      : "bg-sky-900/40 text-sky-200 border-sky-700/40";
  return (
    <span
      className={`inline-block rounded border px-1.5 py-0.5 text-[10px] font-medium ${cls}`}
    >
      {label}
    </span>
  );
}

function formatTime(ts: string): string {
  try {
    const d = new Date(ts);
    return d.toISOString().slice(11, 19);
  } catch {
    return ts.slice(11, 19);
  }
}
