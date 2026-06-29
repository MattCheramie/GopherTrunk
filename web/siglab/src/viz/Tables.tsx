import { useMemo, useState } from "react";
import type { EventRecord, GrantRecord, PDURecord } from "../api/types";

export function GrantsTable({ grants }: { grants: GrantRecord[] }) {
  return (
    <div className="card overflow-x-auto">
      <h3 className="mb-2 text-sm font-semibold">Grants ({grants.length})</h3>
      {grants.length === 0 ? (
        <p className="text-xs text-muted">No traffic grants observed.</p>
      ) : (
        <table className="w-full text-left text-xs">
          <thead className="text-muted">
            <tr>
              <th className="py-1 pr-3">t (s)</th>
              <th className="pr-3">TG</th>
              <th className="pr-3">Src</th>
              <th className="pr-3">Freq (Hz)</th>
              <th className="pr-3">TS</th>
              <th className="pr-3">Enc</th>
              <th>Emerg</th>
            </tr>
          </thead>
          <tbody>
            {grants.map((g, i) => (
              <tr key={i} className="border-t border-line">
                <td className="py-1 pr-3">{g.offset_sec.toFixed(2)}</td>
                <td className="pr-3">{g.group_id}</td>
                <td className="pr-3">{g.source_id}</td>
                <td className="pr-3">{g.frequency_hz || "—"}</td>
                <td className="pr-3">{g.timeslot || "—"}</td>
                <td className="pr-3">{g.encrypted ? "🔒" : ""}</td>
                <td>{g.emergency ? "⚠" : ""}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

// PDUTable renders the data-over-RF signaling dissection with client-side
// filters: by opcode name, CRC pass/fail, and a source/talkgroup id substring.
export function PDUTable({ pdus, truncated }: { pdus: PDURecord[]; truncated?: boolean }) {
  const [opcode, setOpcode] = useState("");
  const [crcOnly, setCrcOnly] = useState(false);
  const [idFilter, setIdFilter] = useState("");

  const opcodes = useMemo(
    () => [...new Set(pdus.map((p) => p.opcode_name))].sort(),
    [pdus],
  );
  const rows = useMemo(
    () =>
      pdus.filter((p) => {
        if (opcode && p.opcode_name !== opcode) return false;
        if (crcOnly && !p.crc_ok) return false;
        if (idFilter) {
          const hay = `${p.source_id ?? ""} ${p.dest_id ?? ""} ${p.talkgroup ?? ""}`;
          if (!hay.includes(idFilter)) return false;
        }
        return true;
      }),
    [pdus, opcode, crcOnly, idFilter],
  );

  return (
    <div className="card overflow-x-auto">
      <h3 className="mb-2 text-sm font-semibold">
        Signaling / data PDUs ({rows.length}/{pdus.length})
        {truncated && <span className="ml-1 text-warn">· truncated</span>}
      </h3>
      <div className="mb-2 flex flex-wrap items-center gap-2 text-xs">
        <select
          className="rounded bg-panel px-1 py-0.5"
          value={opcode}
          onChange={(e) => setOpcode(e.target.value)}
        >
          <option value="">all opcodes</option>
          {opcodes.map((o) => (
            <option key={o} value={o}>
              {o}
            </option>
          ))}
        </select>
        <label className="flex items-center gap-1">
          <input type="checkbox" checked={crcOnly} onChange={(e) => setCrcOnly(e.target.checked)} />
          CRC-OK only
        </label>
        <input
          className="rounded bg-panel px-1 py-0.5"
          placeholder="src / TG id"
          value={idFilter}
          onChange={(e) => setIdFilter(e.target.value)}
        />
      </div>
      <table className="w-full text-left text-xs">
        <thead className="text-muted">
          <tr>
            <th className="py-1 pr-3">t (s)</th>
            <th className="pr-3">Opcode</th>
            <th className="pr-3">NAC</th>
            <th className="pr-3">Src</th>
            <th className="pr-3">Dest/TG</th>
            <th className="pr-3">CRC</th>
            <th className="pr-3">FEC</th>
            <th>Raw</th>
          </tr>
        </thead>
        <tbody>
          {rows.slice(0, 500).map((p) => (
            <tr key={p.seq} className="border-t border-line">
              <td className="py-1 pr-3">{p.offset_sec.toFixed(2)}</td>
              <td className="pr-3">
                {p.opcode_name}
                <span className="ml-1 text-muted">0x{p.opcode.toString(16)}</span>
              </td>
              <td className="pr-3">{p.nac ? `0x${p.nac.toString(16)}` : "—"}</td>
              <td className="pr-3">{p.source_id || "—"}</td>
              <td className="pr-3">{p.talkgroup || p.dest_id || "—"}</td>
              <td className="pr-3">{p.crc_ok ? "✓" : <span className="text-err">✗</span>}</td>
              <td className="pr-3">{p.fec_metric}</td>
              <td className="font-mono text-muted">{p.raw_hex}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function EventTimeline({ events }: { events: EventRecord[] }) {
  const counts = new Map<string, number>();
  for (const e of events) counts.set(e.kind, (counts.get(e.kind) ?? 0) + 1);
  return (
    <div className="card">
      <h3 className="mb-2 text-sm font-semibold">Events ({events.length})</h3>
      <div className="mb-2 flex flex-wrap gap-1">
        {[...counts.entries()].map(([k, n]) => (
          <span key={k} className="rounded bg-panel px-2 py-0.5 text-xs">
            {k} <span className="text-muted">×{n}</span>
          </span>
        ))}
      </div>
      <div className="max-h-64 overflow-y-auto font-mono text-xs">
        {events.slice(-300).map((e) => (
          <div key={e.seq} className="border-t border-line py-0.5">
            <span className="text-muted">{e.offset_sec.toFixed(2)}s</span>{" "}
            <span className="text-accent">{e.kind}</span>{" "}
            <span className="text-muted">{briefFields(e.fields)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function briefFields(fields: Record<string, unknown>): string {
  const keys = ["NAC", "ColorCode", "SystemID", "FrequencyHz", "GroupID", "SourceID", "Stage"];
  return keys
    .filter((k) => k in fields)
    .map((k) => `${k}=${String(fields[k])}`)
    .join(" ");
}
