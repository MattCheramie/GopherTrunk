import { useEffect, useState } from "react";
import { crLabel, fetchLoRaFrames, type LoRaFrame } from "../api/lora";
import { selectClientConfig, useShared } from "../store/shared";

// LoRa panel — list of recent decoded LoRa frames. Each row shows the
// sub-channel frequency, the PHY parameters (spreading factor, coding
// rate, bandwidth), link metrics (SNR / carrier offset), the CRC flag and
// the de-whitened payload. LoRaWAN frames additionally show the message
// type, DevAddr, frame counter, MIC-valid and decrypted payload.
//
// Polls /api/v1/lora/frames every 5 s. Live SSE delivery via the
// KindLoRaFrame bus event is a follow-up.

const POLL_INTERVAL_MS = 5_000;

export function LoRa() {
  const cfg = useShared(selectClientConfig);
  const [frames, setFrames] = useState<LoRaFrame[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancel = false;
    const refresh = async () => {
      try {
        const list = await fetchLoRaFrames(cfg, 200);
        if (cancel) return;
        setFrames(list);
        setError(null);
      } catch (e) {
        if (cancel) return;
        setError(e instanceof Error ? e.message : String(e));
      }
    };
    refresh();
    const t = window.setInterval(refresh, POLL_INTERVAL_MS);
    return () => {
      cancel = true;
      window.clearInterval(t);
    };
  }, [cfg]);

  return (
    <div className="space-y-3">
      <header className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">LoRa</h2>
        <span className="text-xs text-muted">
          {frames.length} frame{frames.length === 1 ? "" : "s"}
        </span>
      </header>

      {error && (
        <div className="rounded border border-red-700/40 bg-red-900/20 text-red-200 text-xs px-3 py-2">
          {error}
        </div>
      )}

      <div className="rounded border border-border overflow-hidden">
        <table className="w-full text-xs">
          <thead className="bg-surface text-muted">
            <tr>
              <th className="text-left px-3 py-1 font-normal w-24">Received</th>
              <th className="text-left px-3 py-1 font-normal w-28">Freq</th>
              <th className="text-left px-3 py-1 font-normal w-28">SF / CR / BW</th>
              <th className="text-right px-3 py-1 font-normal w-24">SNR / CFO</th>
              <th className="text-left px-3 py-1 font-normal w-36">LoRaWAN</th>
              <th className="text-left px-3 py-1 font-normal">Payload</th>
            </tr>
          </thead>
          <tbody>
            {frames.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-3 py-4 text-center text-muted">
                  No LoRa frames yet. Add a{" "}
                  <code className="text-accent">lora.channels</code> entry to
                  your config (an ISM centre frequency — 915 MHz US / 868 MHz
                  EU — plus one or more sub-channels) and decoded frames will
                  land here as they arrive.
                </td>
              </tr>
            ) : (
              frames.map((f) => (
                <tr
                  key={f.id}
                  className={
                    "border-t border-border/60 " +
                    (f.crc_ok ? "" : "bg-yellow-900/15")
                  }
                >
                  <td className="px-3 py-1 font-mono text-muted">
                    {formatTime(f.received_at)}
                  </td>
                  <td className="px-3 py-1 font-mono text-accent">
                    {(f.frequency_hz / 1e6).toFixed(3)}
                  </td>
                  <td className="px-3 py-1 font-mono">
                    SF{f.sf} · {crLabel(f.cr)} · {(f.bandwidth_hz / 1e3).toFixed(0)}k
                  </td>
                  <td className="px-3 py-1 text-right font-mono">
                    {f.snr.toFixed(1)}dB / {f.cfo_hz.toFixed(0)}Hz
                  </td>
                  <td className="px-3 py-1">
                    {f.is_lorawan ? (
                      <div>
                        <span className="uppercase text-accent">{f.mtype}</span>
                        {f.dev_addr && (
                          <span className="font-mono text-[10px] text-muted">
                            {" "}
                            {f.dev_addr} #{f.fcnt}
                          </span>
                        )}
                        <div className="text-[10px]">
                          <span className={f.mic_ok ? "text-green-400" : "text-muted"}>
                            MIC {f.mic_ok ? "ok" : "—"}
                          </span>
                          {f.decrypted && (
                            <span className="text-green-400"> · decrypted</span>
                          )}
                        </div>
                      </div>
                    ) : (
                      <span className="text-muted">—</span>
                    )}
                  </td>
                  <td className="px-3 py-1 font-mono break-all">
                    {f.is_lorawan && f.decrypted && f.decoded
                      ? f.decoded
                      : f.payload_hex || (
                          <span className="text-muted">(empty)</span>
                        )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
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
