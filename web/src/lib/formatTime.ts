// formatClock renders an RFC3339 / ISO event timestamp as a compact
// HH:MM:SS wall-clock string in the BROWSER'S LOCAL timezone.
//
// The live-event panels previously did `new Date(ts).toISOString().slice(11,19)`,
// which always emits UTC — so an operator in any non-UTC zone saw GMT times in
// CC Activity and the other event logs (the reported bug). toLocaleTimeString
// with hour12:false keeps the same 24-hour HH:MM:SS shape but in local time.
//
// On an unparseable timestamp it falls back to the raw time-of-day slice so a
// malformed daemon value is still shown rather than throwing.
export function formatClock(ts: string): string {
  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) {
    return ts.slice(11, 19);
  }
  return d.toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
}

// formatLocalDateTime renders an RFC3339 / ISO timestamp as a compact
// "YYYY-MM-DD HH:MM:SS" string in the BROWSER'S LOCAL timezone.
//
// Several panels (History, Events, Dashboard digest, Active, Tones, the
// scanner log) used to do `ts.replace("T", " ").replace(/\..*$/, "")` on the
// daemon's UTC string — a pure string slice that always shows GMT. An
// operator in UTC+3 therefore saw the History panel three hours behind the
// live activity feed (which formats locally) and read it as "history stops
// updating": the 00:15 row WAS the 03:15 call. Keep the same fixed-width
// numeric shape (sortable, monospace-friendly) but in local time, so every
// panel agrees with the activity feed and the daemon log.
//
// On an unparseable timestamp it falls back to the old raw slice so a
// malformed daemon value is still visible rather than blank.
export function formatLocalDateTime(ts: string): string {
  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) {
    return ts.replace("T", " ").replace(/\..*$/, "");
  }
  const pad = (n: number) => n.toString().padStart(2, "0");
  return (
    `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ` +
    `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  );
}
