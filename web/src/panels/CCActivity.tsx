import { useMemo, useRef, useState } from "react";
import { useShared } from "../store/shared";
import { groupEvents } from "../lib/groupEvents";
import type { EventDTO, TalkgroupDTO } from "../api/types";
import { PageHeader } from "../components/ui/PageHeader";
import { RIDLink } from "../components/RIDLink";
import { formatClock } from "../lib/formatTime";

// CC Activity panel — a focused view of the trunked control-channel
// "chatter" already flowing on the events bus. Filters the rolling
// event log down to the kinds an operator wants to watch live while a
// system is being decoded: voice grants, affiliations, registrations,
// patch / regroup announcements, talker-alias completions, control-
// channel lock / loss, and call start/end markers. Useful for spotting
// what a system is doing in real time without scrolling through the
// raw Events log.
//
// Pure filter-and-render — no extra backend wire. The bus events are
// already in the shared store thanks to the existing SSE consumer.

const CC_KINDS: Record<string, string> = {
  "grant": "Grant",
  "call.start": "Call start",
  "call.end": "Call end",
  "affiliation": "Affiliation",
  "registration": "Registration",
  "unit.request": "Unit→Unit",
  "patch": "Patch",
  "talker.alias": "Talker alias",
  "cc.locked": "CC locked",
  "cc.lost": "CC lost",
  "cchunt.progress": "CC hunt",
  "cchunt.failed": "CC hunt failed",
  "dmr.grant.observed": "DMR grant",
  "dmr.bandplan.learned": "Band plan learned",
};

interface Row {
  ts: string;
  kind: string;
  label: string;
  system: string;
  // details can be a plain string or JSX so RID chips (etc.) can be
  // rendered as react-router Links into /rids/{id} without escaping
  // the surrounding text.
  details: React.ReactNode;
  raw: unknown;
  // count > 1 when "group duplicates" collapsed N identical events into this row.
  count?: number;
}

// ridLink renders an inline link to the per-RID detail page, styled
// like the surrounding mono text but underlined on hover so it reads
// as actionable in the CC Activity feed. Thin wrapper over the shared
// RIDLink so the call sites below read unchanged.
function ridLink(rid: number) {
  return <RIDLink rid={rid} />;
}

export function CCActivity() {
  const events = useShared((s) => s.events);
  const talkgroups = useShared((s) => s.talkgroups);
  const [paused, setPaused] = useState(false);
  const [systemFilter, setSystemFilter] = useState("");
  const [kindFilter, setKindFilter] = useState<string>("");
  // Collapse repeated identical events (e.g. the same grant re-sent every few
  // seconds) into one row with an ×N count and the latest timestamp, to cut the
  // firehose. On by default; toggle off for a raw per-event view.
  const [grouped, setGrouped] = useState(true);

  const rows = useMemo<Row[]>(() => {
    const tgByID = new Map<number, TalkgroupDTO>();
    for (const tg of talkgroups) tgByID.set(tg.id, tg);
    const list: Row[] = [];
    const source = grouped
      ? groupEvents(events)
      : events.map((e) => ({ event: e, count: 1 }));
    for (const g of source) {
      const ev = g.event;
      const label = CC_KINDS[ev.kind];
      if (!label) continue;
      const row = renderRow(ev, label, tgByID);
      if (!row) continue;
      if (systemFilter && !row.system.toLowerCase().includes(systemFilter.toLowerCase())) {
        continue;
      }
      if (kindFilter && ev.kind !== kindFilter) continue;
      list.push(g.count > 1 ? { ...row, count: g.count } : row);
    }
    // Newest (most-recently-active) first.
    return list.reverse();
  }, [events, talkgroups, systemFilter, kindFilter, grouped]);

  // Pause freezes the table so an operator can read/click a row without it
  // churning. Snapshot synchronously during render the first time we see
  // `paused`, so there is no effect-timing window where a live SSE batch can
  // leak through after the click (#772). An earlier fix snapshotted in a
  // post-paint useEffect, which left a render cycle where the live rows were
  // still committed — and because the store is a useSyncExternalStore, an event
  // batch arriving in that window re-rendered before the freeze engaged.
  // Cleared on resume.
  const frozenRef = useRef<Row[] | null>(null);
  if (paused) {
    if (frozenRef.current === null) frozenRef.current = rows;
  } else {
    frozenRef.current = null;
  }
  const displayRows = paused ? (frozenRef.current ?? rows) : rows;

  return (
    <div className="space-y-3">
      <PageHeader
        title="CC Activity"
        actions={
        <div className="flex items-center gap-2 text-xs">
          <select
            className="bg-surface border border-border rounded px-2 py-1"
            value={kindFilter}
            onChange={(e) => setKindFilter(e.target.value)}
          >
            <option value="">All kinds</option>
            {Object.entries(CC_KINDS).map(([k, label]) => (
              <option key={k} value={k}>{label}</option>
            ))}
          </select>
          <input
            type="text"
            placeholder="filter system…"
            className="bg-surface border border-border rounded px-2 py-1"
            value={systemFilter}
            onChange={(e) => setSystemFilter(e.target.value)}
          />
          <button
            type="button"
            className="px-2 py-1 rounded border border-border text-xs"
            onClick={() => setGrouped(!grouped)}
            aria-pressed={grouped}
            title="Collapse repeated identical events into one row with a count"
          >
            {grouped ? "▦ grouped" : "≡ all"}
          </button>
          <button
            type="button"
            className="px-2 py-1 rounded border border-border text-xs"
            onClick={() => setPaused(!paused)}
            aria-pressed={paused}
          >
            {paused ? "▶ resume" : "❚❚ pause"}
          </button>
        </div>
        }
      />

      <div className="text-xs text-muted">
        {displayRows.length} matching event{displayRows.length === 1 ? "" : "s"}
        {paused && " (paused — display frozen, the daemon is still receiving)"}
      </div>

      <div className="rounded border border-border overflow-hidden">
        <table className="w-full text-xs">
          <thead className="bg-surface text-muted">
            <tr>
              <th className="text-left px-3 py-1 font-normal w-24">Time</th>
              <th className="text-left px-3 py-1 font-normal w-28">Kind</th>
              <th className="text-left px-3 py-1 font-normal">System</th>
              <th className="text-left px-3 py-1 font-normal">Details</th>
            </tr>
          </thead>
          <tbody>
            {displayRows.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-3 py-4 text-center text-muted">
                  Nothing here yet — control-channel activity will appear
                  as the daemon decodes it. Try removing filters or
                  pointing at a busy system.
                </td>
              </tr>
            ) : (
              displayRows.slice(0, 500).map((r, i) => (
                <tr key={i} className="border-t border-border/60">
                  <td className="px-3 py-1 font-mono text-muted">
                    {formatTime(r.ts)}
                  </td>
                  <td className="px-3 py-1">
                    {r.label}
                    {r.count && r.count > 1 ? (
                      <span
                        className="ml-1 px-1 rounded bg-surface text-muted text-[10px] font-mono"
                        title={`${r.count} identical events collapsed`}
                      >
                        ×{r.count}
                      </span>
                    ) : null}
                  </td>
                  <td className="px-3 py-1 font-mono text-accent">{r.system || "—"}</td>
                  <td className="px-3 py-1">{r.details}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function renderRow(
  ev: EventDTO,
  label: string,
  tgByID: Map<number, TalkgroupDTO>,
): Row | null {
  const payload = (ev.payload ?? {}) as Record<string, unknown>;
  switch (ev.kind) {
    case "grant":
    case "call.start": {
      const g = (payload.grant ?? payload) as Record<string, unknown>;
      const system = str(g.system);
      const protocol = str(g.protocol);
      const groupID = num(g.group_id);
      const sourceID = num(g.source_id);
      const freq = num(g.frequency_hz);
      const tag: string[] = [];
      if (g.encrypted) tag.push("ENC");
      if (g.emergency) tag.push("EMERG");
      if (g.data_call) tag.push("DATA");
      const suffix =
        (freq ? ` @ ${(freq / 1e6).toFixed(4)} MHz` : "") +
        (protocol ? ` · ${protocol}` : "") +
        (tag.length ? ` · ${tag.join(" ")}` : "");
      const details = (
        <>
          {`TG ${groupID}`}
          {sourceID ? (
            <>
              {" ← "}
              {ridLink(sourceID)}
            </>
          ) : null}
          {suffix}
        </>
      );
      return { ts: ev.timestamp, kind: ev.kind, label, system, details, raw: ev.payload };
    }
    case "call.end": {
      const g = (payload.grant ?? payload) as Record<string, unknown>;
      const system = str(g.system);
      const groupID = num(g.group_id);
      const reason = str(payload.reason);
      const details = `TG ${groupID}` + (reason ? ` · ${reason}` : "");
      return { ts: ev.timestamp, kind: ev.kind, label, system, details, raw: ev.payload };
    }
    case "affiliation":
    case "registration": {
      const system = str(payload.system);
      const radio = num(payload.radio_id ?? payload.source_id ?? payload.source);
      const group = num(payload.group_id ?? payload.talkgroup_id);
      const code = num(payload.response_code ?? payload.response);
      const suffix =
        (group ? ` → TG ${group}` : "") +
        (code !== 0 ? ` · resp ${code}` : "");
      const details = (
        <>
          {"radio "}
          {radio ? ridLink(radio) : "?"}
          {suffix}
        </>
      );
      return { ts: ev.timestamp, kind: ev.kind, label, system, details, raw: ev.payload };
    }
    case "unit.request": {
      const system = str(payload.system);
      const src = num(payload.source_id ?? payload.source);
      const target = num(payload.target_id ?? payload.target);
      const details = (
        <>
          {src ? ridLink(src) : "?"}
          {" → "}
          {target ? ridLink(target) : "?"}
        </>
      );
      return { ts: ev.timestamp, kind: ev.kind, label, system, details, raw: ev.payload };
    }
    case "patch": {
      const system = str(payload.system);
      const superGroup = num(payload.super_group ?? payload.regroup_id);
      const memberIDs = ((payload.members ?? []) as unknown[])
        .map((m) => num(m))
        .filter((n) => n > 0);
      const op = payload.add === false || payload.cancelled || payload.removed ? "cancel" : "add";
      const verb = op === "add" ? "Patch Active" : "Patch Cancelled";
      const memberNodes = memberIDs.map((id, i) => {
        const tg = tgByID.get(id);
        const text = tg?.alpha_tag ? `${tg.alpha_tag} (${id})` : `TG ${id}`;
        return (
          <span key={id}>
            {i > 0 ? " & " : ""}
            {text}
          </span>
        );
      });
      const details = (
        <>
          {`${verb}: `}
          {memberNodes.length > 0 ? memberNodes : <span className="text-muted">no members</span>}
          {superGroup ? <span className="text-muted">{` [SG ${superGroup}]`}</span> : null}
        </>
      );
      return { ts: ev.timestamp, kind: ev.kind, label, system, details, raw: ev.payload };
    }
    case "talker.alias": {
      const system = str(payload.system);
      const source = num(payload.source ?? payload.radio_id);
      const alias = str(payload.alias);
      const details = (
        <>
          {"radio "}
          {source ? ridLink(source) : "?"}
          {`: "${alias}"`}
        </>
      );
      return { ts: ev.timestamp, kind: ev.kind, label, system, details, raw: ev.payload };
    }
    case "cc.locked":
    case "cc.lost": {
      const system = str(payload.system);
      const freq = num(payload.frequency_hz);
      const details =
        freq ? `@ ${(freq / 1e6).toFixed(4)} MHz` : "";
      return { ts: ev.timestamp, kind: ev.kind, label, system, details, raw: ev.payload };
    }
    case "cchunt.progress": {
      const system = str(payload.system);
      const freq = num(payload.attempted_freq_hz);
      const idx = num(payload.attempt_index);
      const total = num(payload.total_candidates);
      const pos = total ? ` (${idx + 1}/${total})` : "";
      const details = freq ? `trying ${(freq / 1e6).toFixed(4)} MHz${pos}` : `trying…${pos}`;
      return { ts: ev.timestamp, kind: ev.kind, label, system, details, raw: ev.payload };
    }
    case "cchunt.failed": {
      const system = str(payload.system);
      const backoff = num(payload.backoff_ms);
      const details =
        "candidates exhausted" + (backoff ? ` · retry in ${(backoff / 1000).toFixed(0)}s` : "");
      return { ts: ev.timestamp, kind: ev.kind, label, system, details, raw: ev.payload };
    }
    case "dmr.grant.observed": {
      const system = str(payload.system);
      const lcn = num(payload.lcn);
      const ts = num(payload.timeslot); // raw 0 = TS1, 1 = TS2
      const groupID = num(payload.group_id);
      const sourceID = num(payload.source_id);
      const cc = num(payload.cc_freq_hz);
      const details = (
        <>
          {`LCN ${lcn} · TS${ts + 1} · TG ${groupID}`}
          {sourceID ? (
            <>
              {" ← "}
              {ridLink(sourceID)}
            </>
          ) : null}
          {cc ? <span className="text-muted">{` · CC ${(cc / 1e6).toFixed(4)} MHz`}</span> : null}
        </>
      );
      return { ts: ev.timestamp, kind: ev.kind, label, system, details, raw: ev.payload };
    }
    case "dmr.bandplan.learned": {
      const system = str(payload.system);
      const baseHz = num(payload.base_hz);
      const spacingHz = num(payload.spacing_hz);
      const table = (payload.table ?? []) as unknown[];
      const numPairs = num(payload.num_pairs);
      const conf = num(payload.confidence);
      const shape =
        table.length > 0
          ? `${table.length} LCNs`
          : baseHz
            ? `${(baseHz / 1e6).toFixed(4)} MHz · ${(spacingHz / 1e3).toFixed(2)} kHz spacing`
            : "learned";
      const details =
        shape +
        (numPairs ? ` · ${numPairs} pairs` : "") +
        (conf ? ` · conf ${(conf * 100).toFixed(0)}%` : "");
      return { ts: ev.timestamp, kind: ev.kind, label, system, details, raw: ev.payload };
    }
    default:
      return null;
  }
}

function str(v: unknown): string {
  return typeof v === "string" ? v : "";
}

function num(v: unknown): number {
  if (typeof v === "number") return v;
  if (typeof v === "string") {
    const n = parseInt(v, 10);
    return Number.isFinite(n) ? n : 0;
  }
  return 0;
}

const formatTime = formatClock;
