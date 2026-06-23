// formatIdNumber renders a P25 identity number — WACN, System ID, NAC, RFSS,
// or Site — in the operator-selected base (web.id_base config, surfaced via
// /api/v1/runtime). "hex" is the convention these are quoted in the field, and
// keeps the decimal in parentheses so no information is lost; "dec" shows plain
// decimal. Returns null for null/undefined so callers can fall back to an
// "awaiting" hint.
export function formatIdNumber(
  value: number | null | undefined,
  base: "hex" | "dec",
): string | null {
  if (value == null) return null;
  if (base === "dec") return String(value);
  return `0x${value.toString(16).toUpperCase()} (${value})`;
}
