import { useEffect, useState } from "react";
import {
  fetchPagerMessages,
  functionLabel,
  type PagerMessage,
} from "../api/pagers";
import { selectClientConfig, useShared } from "../store/shared";

// Pagers panel — list of recent POCSAG and FLEX pages decoded by the
// daemon. Each row carries its protocol (POCSAG / FLEX), RIC, function
// code (A/B/C/D), numeric / alphanumeric encoding, decoded body, and the
// BCH bit-error count (a non-zero value indicates the page was marginal).
//
// Live updates piggyback on the events bus once SSE delivery is wired;
// for v1 the panel polls /api/v1/pager/messages every 5 s.

const POLL_INTERVAL_MS = 5_000;

export function Pagers() {
  const cfg = useShared(selectClientConfig);
  const [messages, setMessages] = useState<PagerMessage[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancel = false;
    const refresh = async () => {
      try {
        const list = await fetchPagerMessages(cfg, 200);
        if (cancel) return;
        setMessages(list);
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
        <h2 className="text-xl font-semibold">Pagers</h2>
        <span className="text-xs text-muted">
          {messages.length} message{messages.length === 1 ? "" : "s"}
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
              <th className="text-left px-3 py-1 font-normal w-32">Received</th>
              <th className="text-left px-3 py-1 font-normal w-20">Type</th>
              <th className="text-right px-3 py-1 font-normal w-24">RIC</th>
              <th className="text-left px-3 py-1 font-normal w-12">Fn</th>
              <th className="text-left px-3 py-1 font-normal w-16">Enc</th>
              <th className="text-left px-3 py-1 font-normal">Body</th>
              <th className="text-right px-3 py-1 font-normal w-16">BER</th>
            </tr>
          </thead>
          <tbody>
            {messages.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-3 py-4 text-center text-muted">
                  No pager messages yet. POCSAG pages decoded by the
                  daemon will appear here as they arrive — typical
                  workflow is to point one SDR at the local paging
                  frequency (e.g. 152.0075 MHz commercial paging,
                  the local fire department dispatch tone-out
                  freq, or 439.9875 MHz DAPNET).
                </td>
              </tr>
            ) : (
              messages.map((m) => (
                <tr key={m.id} className="border-t border-border/60">
                  <td className="px-3 py-1 font-mono text-muted">
                    {formatTime(m.received_at)}
                  </td>
                  <td className="px-3 py-1">
                    <ProtocolBadge protocol={m.protocol} />
                  </td>
                  <td className="px-3 py-1 text-right font-mono text-accent">
                    {m.ric}
                  </td>
                  <td className="px-3 py-1">{functionLabel(m.func)}</td>
                  <td className="px-3 py-1 uppercase text-muted">
                    {m.encoding || "?"}
                  </td>
                  <td className="px-3 py-1">{m.body || <span className="text-muted">(empty)</span>}</td>
                  <td className="px-3 py-1 text-right font-mono">
                    {m.corrected > 0 ? (
                      <span className="text-yellow-300">{m.corrected}</span>
                    ) : (
                      <span className="text-muted">0</span>
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

// ProtocolBadge renders the page's signalling protocol as a small colored
// pill so POCSAG and FLEX rows are distinguishable at a glance. Falls back
// to "POCSAG" when the backend left the field empty (older rows persisted
// before the protocol column was populated default to POCSAG server-side).
function ProtocolBadge({ protocol }: { protocol: string }) {
  const proto = (protocol || "pocsag").toLowerCase();
  const label = proto.toUpperCase();
  const cls =
    proto === "flex"
      ? "bg-purple-900/40 text-purple-200 border-purple-700/40"
      : "bg-sky-900/40 text-sky-200 border-sky-700/40";
  return (
    <span className={`inline-block rounded border px-1.5 py-0.5 text-[10px] font-medium ${cls}`}>
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
