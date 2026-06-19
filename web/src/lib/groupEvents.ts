import type { EventDTO } from "../api/types";

// GroupedEvent collapses repeated events with identical content into one entry
// that carries an occurrence count and the first/last timestamps it was seen.
export interface GroupedEvent {
  // Representative occurrence — the most recent one; its timestamp equals
  // lastTimestamp, so existing newest-first views keep working unchanged.
  event: EventDTO;
  count: number;
  firstTimestamp: string;
  lastTimestamp: string;
}

// stableStringify produces a deterministic JSON string (object keys sorted
// recursively) so two payloads with identical content but different key order
// still collapse into the same group.
function stableStringify(value: unknown): string {
  if (value === null || typeof value !== "object") return JSON.stringify(value) ?? "null";
  if (Array.isArray(value)) return `[${value.map(stableStringify).join(",")}]`;
  const obj = value as Record<string, unknown>;
  const keys = Object.keys(obj).sort();
  return `{${keys.map((k) => `${JSON.stringify(k)}:${stableStringify(obj[k])}`).join(",")}}`;
}

// contentKey identifies an event by kind + payload content (ignoring its
// timestamp), so the spammy repeats — grants, registrations, affiliations re-
// emitted with identical fields — share one key. Event kinds contain no spaces,
// so a space cleanly separates the kind from the serialized payload.
export function contentKey(ev: EventDTO): string {
  return `${ev.kind} ${stableStringify(ev.payload ?? null)}`;
}

// groupEvents collapses events with identical kind + payload into a single
// entry carrying an occurrence count and the first/last timestamps. Input is
// assumed chronological (oldest→newest, as the store ring holds it); the output
// preserves that order keyed on each group's LAST occurrence, so a caller that
// reverse()s for a newest-first view shows the most-recently-active group on
// top (a still-firing grant stays live rather than sinking).
export function groupEvents(events: EventDTO[]): GroupedEvent[] {
  const byKey = new Map<string, GroupedEvent>();
  for (const ev of events) {
    const key = contentKey(ev);
    const existing = byKey.get(key);
    if (existing) {
      existing.count += 1;
      existing.event = ev; // most-recent representative
      existing.lastTimestamp = ev.timestamp;
      byKey.delete(key); // re-insert to move to the most-recent position
      byKey.set(key, existing);
    } else {
      byKey.set(key, {
        event: ev,
        count: 1,
        firstTimestamp: ev.timestamp,
        lastTimestamp: ev.timestamp,
      });
    }
  }
  return Array.from(byKey.values());
}
