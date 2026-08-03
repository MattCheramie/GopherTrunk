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
