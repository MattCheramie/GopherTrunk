import { useMemo, useState } from "react";
import { crLabel, fetchLoRaFrames, type LoRaFrame } from "../api/lora";
import { Column, DataTable } from "../components/DataTable";
import { PageHeader } from "../components/ui/PageHeader";
import { Badge } from "../components/ui/Badge";
import { StaleIndicator } from "../components/ui/StaleIndicator";
import { useDataPoll } from "../hooks/useDataPoll";
import { selectClientConfig, useShared } from "../store/shared";
import { formatClock } from "../lib/formatTime";

// LoRa panel — list of recent decoded LoRa frames. Each row shows the
// sub-channel frequency, the PHY parameters (SF / CR / BW), link
// metrics (SNR / CFO), the CRC flag, the LoRaWAN envelope, and the
// de-whitened payload.
//
// Polls /api/v1/lora/frames every 5 s.

const POLL_INTERVAL_MS = 5_000;

export function LoRa() {
  const cfg = useShared(selectClientConfig);
  const [frames, setFrames] = useState<LoRaFrame[]>([]);

  const { loading, error, stale, lastUpdated } = useDataPoll({
    fetcher: () => fetchLoRaFrames(cfg, 200),
    onData: setFrames,
    intervalMs: POLL_INTERVAL_MS,
    resetKey: cfg.baseURL,
  });

  const columns: Column<LoRaFrame>[] = useMemo(
    () => [
      {
        key: "received",
        header: "Received",
        render: (f) => (
          <span className="font-mono text-muted">{formatTime(f.received_at)}</span>
        ),
        sort: (a, b) => a.received_at.localeCompare(b.received_at),
      },
      {
        key: "freq",
        header: "Freq",
        render: (f) => (
          <span className="inline-flex items-center gap-1">
            <span className="font-mono text-accent">
              {(f.frequency_hz / 1e6).toFixed(3)}
            </span>
            {!f.crc_ok && <Badge tone="warn">crc</Badge>}
          </span>
        ),
        sort: (a, b) => a.frequency_hz - b.frequency_hz,
      },
      {
        key: "phy",
        header: "SF / CR / BW",
        render: (f) => (
          <span className="font-mono">
            SF{f.sf} · {crLabel(f.cr)} · {(f.bandwidth_hz / 1e3).toFixed(0)}k
          </span>
        ),
      },
      {
        key: "link",
        header: "SNR / CFO",
        className: "text-right font-mono",
        headerClassName: "text-right",
        render: (f) => (
          <span>
            {f.snr.toFixed(1)}dB / {f.cfo_hz.toFixed(0)}Hz
          </span>
        ),
        sort: (a, b) => a.snr - b.snr,
      },
      {
        key: "lorawan",
        header: "LoRaWAN",
        render: (f) =>
          f.is_lorawan ? (
            <div>
              <span className="uppercase text-accent">{f.mtype}</span>
              {f.dev_addr && (
                <span className="font-mono text-[10px] text-muted">
                  {" "}
                  {f.dev_addr} #{f.fcnt}
                </span>
              )}
              <div className="text-[10px]">
                <span className={f.mic_ok ? "text-ok" : "text-muted"}>
                  MIC {f.mic_ok ? "ok" : "—"}
                </span>
                {f.decrypted && <span className="text-ok"> · decrypted</span>}
              </div>
            </div>
          ) : (
            <span className="text-muted">—</span>
          ),
      },
      {
        key: "payload",
        header: "Payload",
        className: "font-mono break-all",
        render: (f) =>
          f.is_lorawan && f.decrypted && f.decoded
            ? f.decoded
            : f.payload_hex || <span className="text-muted">(empty)</span>,
      },
    ],
    [],
  );

  return (
    <div className="space-y-3">
      <PageHeader
        title="LoRa"
        actions={
          <>
            <StaleIndicator stale={stale} lastUpdated={lastUpdated} />
            <span className="text-xs text-muted">
              {frames.length} frame{frames.length === 1 ? "" : "s"}
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
        rows={frames}
        columns={columns}
        rowKey={(f) => String(f.id)}
        defaultSortKey="received"
        defaultSortDirection="desc"
        tableId="lora"
        pageSize={50}
        loading={loading}
        searchable
        searchAccessor={(f) =>
          [
            (f.frequency_hz / 1e6).toFixed(3),
            f.mtype,
            f.dev_addr,
            f.decoded,
            f.payload_hex,
          ]
            .filter(Boolean)
            .join(" ")
        }
        searchPlaceholder="Search by freq, DevAddr, payload…"
        emptyMessage={
          <>
            No LoRa frames yet. Add a{" "}
            <code className="text-accent">lora.channels</code> entry to your
            config (an ISM centre frequency — 915 MHz US / 868 MHz EU — plus one
            or more sub-channels) and decoded frames will land here as they
            arrive.
          </>
        }
      />
    </div>
  );
}

const formatTime = formatClock;
