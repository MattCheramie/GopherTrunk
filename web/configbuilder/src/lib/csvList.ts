// Helpers for editing a Go []string (e.g. a broadcast feed's `Systems`
// allow-list) as a single comma-separated text input. An empty result is
// returned as null so the marshaled YAML omits the key (matching "empty =
// every system").

export function formatCommaList(arr: string[] | null | undefined): string {
  return (arr ?? []).join(", ");
}

export function parseCommaList(text: string): string[] | null {
  const out = text
    .split(",")
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
  return out.length ? out : null;
}
