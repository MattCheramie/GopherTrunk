import { useMemo, useState } from "react";
import { Column, DataTable } from "../components/DataTable";
import { PageHeader } from "../components/ui/PageHeader";
import { Badge } from "../components/ui/Badge";
import type { EventDTO } from "../api/types";
import { useShared } from "../store/shared";
import { contentKey, groupEvents, type GroupedEvent } from "../lib/groupEvents";

// Events renders the live ring buffer the WebSocket stream populates
// into the shared store. No polling — the stream pushes everything;
// see App.tsx's openEventStream wire-up.
export function Events() {
  const events = useShared((s) => s.events);
  const wsStatus = useShared((s) => s.wsStatus);
  const eventCap = useShared((s) => s.eventCap);
  const [paused, setPaused] = useState(false);
  const [snapshot, setSnapshot] = useState<EventDTO[] | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);
  // autoFrozen records that the freeze was triggered by expanding a row (so we
  // resume on collapse), as opposed to a manual Pause the operator wants kept.
  const [autoFrozen, setAutoFrozen] = useState(false);
  // Collapse repeated identical events (same kind + payload) into one row with
  // an ×N count and the latest timestamp, to cut the firehose. On by default.
  const [grouped, setGrouped] = useState(true);

  // When paused, freeze the table at the snapshot the user paused on;
  // when running, mirror the live store ring.
  const visible = paused ? (snapshot ?? events) : events;

  // Group identical events (when enabled), then newest (most-recently-active)
  // first — reverse so the latest is at the top, the mobile-friendly read order.
  // Search is applied by the DataTable on top of the grouped rows.
  const ordered = useMemo<GroupedEvent[]>(() => {
    const groups = grouped
      ? groupEvents(visible)
      : visible.map((e) => ({
          event: e,
          count: 1,
          firstTimestamp: e.timestamp,
          lastTimestamp: e.timestamp,
        }));
    return groups.slice().reverse();
  }, [visible, grouped]);

  const columns: Column<GroupedEvent>[] = useMemo(
    () => [
      {
        key: "ts",
        header: "Time",
        render: (g) => (
          <span className="font-mono text-xs text-muted whitespace-nowrap">
            {g.event.timestamp.replace("T", " ").replace(/\..*$/, "")}
            {g.count > 1 ? (
              <span
                className="ml-1 px-1 rounded bg-surface text-fg/70 text-[10px]"
                title={`${g.count} identical events (since ${g.firstTimestamp})`}
              >
                ×{g.count}
              </span>
            ) : null}
          </span>
        ),
        sort: (a, b) => a.event.timestamp.localeCompare(b.event.timestamp),
      },
      {
        key: "kind",
        header: "Kind",
        render: (g) => (
          <span className="font-mono text-accent">{g.event.kind}</span>
        ),
        sort: (a, b) => a.event.kind.localeCompare(b.event.kind),
      },
      {
        key: "preview",
        header: "Payload",
        render: (g) => (
          <span className="text-xs font-mono text-muted truncate inline-block max-w-[28ch] sm:max-w-[60ch]">
            {previewPayload(g.event.payload)}
          </span>
        ),
      },
    ],
    [],
  );

  function freeze() {
    setSnapshot(events.slice());
    setPaused(true);
  }
  function resume() {
    setPaused(false);
    setSnapshot(null);
  }
  function togglePause() {
    setAutoFrozen(false); // a manual toggle owns the pause state from here on
    if (paused) resume();
    else freeze();
  }

  // Expanding a row to inspect it auto-freezes the live stream so the row
  // doesn't scroll away as new events prepend (the whole point of "looking at
  // an event"); collapsing it resumes — unless the operator had paused manually.
  function handleRowClick(_g: GroupedEvent, key: string) {
    const collapsing = expanded === key;
    setExpanded(collapsing ? null : key);
    if (collapsing) {
      if (autoFrozen) {
        resume();
        setAutoFrozen(false);
      }
    } else if (!paused) {
      freeze();
      setAutoFrozen(true);
    }
  }

  return (
    <div className="space-y-3">
      <PageHeader
        title="Events"
        actions={
          <>
            <Badge
              tone={
                wsStatus === "open"
                  ? "ok"
                  : wsStatus === "connecting"
                    ? "warn"
                    : "err"
              }
            >
              {wsStatus}
            </Badge>
            <span className="text-xs text-muted">
              {events.length}/{eventCap}
            </span>
          </>
        }
      />

      <p className="text-xs text-muted">
        {paused
          ? autoFrozen
            ? "Frozen while you inspect — collapse the event (or Resume) to go live."
            : "Paused — display frozen; the stream keeps running underneath."
          : "Click an event to inspect its payload — the stream freezes so it stays put."}
      </p>

      <DataTable
        rows={ordered}
        columns={columns}
        rowKey={(g, i) => `${contentKey(g.event)}-${i}`}
        searchable
        searchAccessor={(g) =>
          `${g.event.kind} ${g.event.timestamp} ${JSON.stringify(g.event.payload ?? "")}`
        }
        searchPlaceholder="Search by kind, timestamp, or payload…"
        toolbar={
          <>
            <button
              className={grouped ? "btn-primary" : "btn-ghost"}
              onClick={() => setGrouped(!grouped)}
              aria-pressed={grouped}
              title="Collapse repeated identical events into one row with a count"
            >
              {grouped ? "Grouped" : "All"}
            </button>
            <button
              className={paused ? "btn-primary" : "btn-ghost"}
              onClick={togglePause}
              aria-pressed={paused}
            >
              {paused ? "Resume" : "Pause"}
            </button>
          </>
        }
        onRowClick={handleRowClick}
        renderExpansion={(g) => (
          <pre className="whitespace-pre-wrap break-words font-mono text-xs text-fg/80">
            {g.count > 1
              ? `// seen ${g.count}× — first ${g.firstTimestamp}, last ${g.lastTimestamp}\n`
              : ""}
            {JSON.stringify(g.event, null, 2)}
          </pre>
        )}
        expandedKey={expanded}
        emptyMessage={
          events.length === 0
            ? "No events yet — the WebSocket stream pushes new ones live."
            : "No events match the filter."
        }
      />
    </div>
  );
}

function previewPayload(p: unknown): string {
  if (p == null) return "";
  try {
    const s = JSON.stringify(p);
    return s.length > 120 ? `${s.slice(0, 117)}…` : s;
  } catch {
    return String(p);
  }
}
