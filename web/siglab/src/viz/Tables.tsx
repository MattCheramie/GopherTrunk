import type { EventRecord, GrantRecord } from "../api/types";

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
