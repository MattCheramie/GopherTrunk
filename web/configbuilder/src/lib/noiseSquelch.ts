// The scan-list channel's noise_squelch setting (#1329): unset and true both
// mean on, false means the squelch level alone. The select keeps "unset"
// apart from "on" so a config that never mentioned the key stays that way.

export const NOISE_SQUELCH_OPTIONS = [
  { value: "", label: "(on)" },
  { value: "on", label: "on" },
  { value: "off", label: "off (squelch level only)" },
];

// isAnalogFM reports whether a channel mode has the noise squelch: fm, nfm,
// or unset (which the daemon reads as fm).
export function isAnalogFM(mode: string | undefined): boolean {
  return mode === undefined || mode === "" || mode === "fm" || mode === "nfm";
}

export function noiseSquelchValue(v: boolean | null | undefined): string {
  if (v === true) return "on";
  if (v === false) return "off";
  return "";
}

export function parseNoiseSquelch(s: string): boolean | undefined {
  if (s === "on") return true;
  if (s === "off") return false;
  return undefined;
}
