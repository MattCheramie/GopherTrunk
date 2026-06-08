// Browser-storage-backed preferences. All settings live here so panels
// read/write via a typed API rather than scattering `localStorage`
// calls. Durable preferences land in `localStorage`; per-tab transient
// values (the bearer token, last-viewed tab) live in `sessionStorage`
// by default. The Settings panel offers a "remember on this device"
// toggle that promotes the token to `localStorage` for convenience on
// trusted personal devices.

const LS_KEYS = {
  serverURL: "gt.server.url",
  rememberToken: "gt.token.persist",
  tokenPersistent: "gt.token.persistent",
  theme: "gt.ui.theme",
  density: "gt.ui.density",
  writeMode: "gt.ui.writeMode",
  audioVolume: "gt.audio.volume",
  installPromptDismissed: "gt.pwa.installDismissed",
  constellationOffsetKHz: "gt.constellation.offsetKHz",
  constellationHold: "gt.constellation.hold",
  constellationDcBlock: "gt.constellation.dcBlock",
  constellationAutoScale: "gt.constellation.autoScale",
  constellationZoom: "gt.constellation.zoom",
  constellationView: "gt.constellation.view",
  constellationProto: "gt.constellation.proto",
  symbolScopeOffsetKHz: "gt.symbolScope.offsetKHz",
  symbolScopeHold: "gt.symbolScope.hold",
  symbolScopeProto: "gt.symbolScope.proto",
  eyeOffsetKHz: "gt.eye.offsetKHz",
  eyeHold: "gt.eye.hold",
  tuningOffsetKHz: "gt.tuning.offsetKHz",
  tuningHold: "gt.tuning.hold",
  tuningProto: "gt.tuning.proto",
} as const;

const SS_KEYS = {
  tokenSession: "gt.token.session",
  lastTab: "gt.ui.lastTab",
} as const;

export type Theme = "dark" | "monochrome";
export type Density = "comfortable" | "compact";

function readLS(key: string): string | null {
  try {
    return window.localStorage.getItem(key);
  } catch {
    return null;
  }
}
function writeLS(key: string, value: string | null) {
  try {
    if (value === null) window.localStorage.removeItem(key);
    else window.localStorage.setItem(key, value);
  } catch {
    /* private mode or quota: silently drop. */
  }
}
function readSS(key: string): string | null {
  try {
    return window.sessionStorage.getItem(key);
  } catch {
    return null;
  }
}
function writeSS(key: string, value: string | null) {
  try {
    if (value === null) window.sessionStorage.removeItem(key);
    else window.sessionStorage.setItem(key, value);
  } catch {
    /* see readLS. */
  }
}

export const prefs = {
  serverURL(): string | null {
    return readLS(LS_KEYS.serverURL);
  },
  setServerURL(url: string | null) {
    writeLS(LS_KEYS.serverURL, url);
  },

  /** Return the active token regardless of which storage tier it lives in. */
  token(): string | null {
    return readLS(LS_KEYS.tokenPersistent) ?? readSS(SS_KEYS.tokenSession);
  },
  setToken(token: string | null, persist: boolean) {
    // Replace whichever tier currently holds the token. The two
    // tiers are mutually exclusive so the SPA never has a stale
    // value lurking in the other one.
    writeLS(LS_KEYS.tokenPersistent, persist ? token : null);
    writeSS(SS_KEYS.tokenSession, persist ? null : token);
    writeLS(LS_KEYS.rememberToken, persist ? "1" : "0");
  },
  rememberToken(): boolean {
    return readLS(LS_KEYS.rememberToken) === "1";
  },

  theme(): Theme {
    return (readLS(LS_KEYS.theme) as Theme | null) ?? "dark";
  },
  setTheme(theme: Theme) {
    writeLS(LS_KEYS.theme, theme);
  },

  density(): Density {
    return (readLS(LS_KEYS.density) as Density | null) ?? "comfortable";
  },
  setDensity(d: Density) {
    writeLS(LS_KEYS.density, d);
  },

  writeMode(): boolean {
    return readLS(LS_KEYS.writeMode) === "1";
  },
  setWriteMode(enabled: boolean) {
    writeLS(LS_KEYS.writeMode, enabled ? "1" : "0");
  },

  audioVolume(): number {
    const raw = readLS(LS_KEYS.audioVolume);
    if (raw === null) return 0.8;
    const n = Number(raw);
    return Number.isFinite(n) && n >= 0 && n <= 1 ? n : 0.8;
  },
  setAudioVolume(v: number) {
    writeLS(LS_KEYS.audioVolume, String(Math.max(0, Math.min(1, v))));
  },

  lastTab(): string | null {
    return readSS(SS_KEYS.lastTab);
  },
  setLastTab(name: string | null) {
    writeSS(SS_KEYS.lastTab, name);
  },

  installPromptDismissed(): boolean {
    return readLS(LS_KEYS.installPromptDismissed) === "1";
  },
  setInstallPromptDismissed(dismissed: boolean) {
    writeLS(LS_KEYS.installPromptDismissed, dismissed ? "1" : "0");
  },

  // Constellation panel view options. The offset (in kHz, relative to
  // the SDR centre) pulls an off-centre locked channel out from under
  // the centre DC spike; "hold" keeps that offset pinned across panel
  // visits. DC-block and auto-scale default on for a clean scatter.
  constellationOffsetKHz(): number {
    const raw = readLS(LS_KEYS.constellationOffsetKHz);
    if (raw === null) return 0;
    const n = Number(raw);
    return Number.isFinite(n) ? n : 0;
  },
  setConstellationOffsetKHz(khz: number) {
    writeLS(LS_KEYS.constellationOffsetKHz, String(khz));
  },
  constellationHold(): boolean {
    return readLS(LS_KEYS.constellationHold) === "1";
  },
  setConstellationHold(on: boolean) {
    writeLS(LS_KEYS.constellationHold, on ? "1" : "0");
  },
  constellationDcBlock(): boolean {
    // Default on — the whole point is to suppress the DC spike.
    return readLS(LS_KEYS.constellationDcBlock) !== "0";
  },
  setConstellationDcBlock(on: boolean) {
    writeLS(LS_KEYS.constellationDcBlock, on ? "1" : "0");
  },
  constellationAutoScale(): boolean {
    return readLS(LS_KEYS.constellationAutoScale) !== "0";
  },
  setConstellationAutoScale(on: boolean) {
    writeLS(LS_KEYS.constellationAutoScale, on ? "1" : "0");
  },
  // Zoom magnifies the plotted cloud and dot size so the scatter fills the
  // unit circle like OP25's plot. Clamped 0.5..8 (wide range so the view can
  // punch in hard); defaults a touch above 1 so the out-of-the-box view is
  // already larger than 1:1.
  constellationZoom(): number {
    const raw = readLS(LS_KEYS.constellationZoom);
    if (raw === null) return 1.5;
    const n = Number(raw);
    if (!Number.isFinite(n)) return 1.5;
    return Math.max(0.5, Math.min(8, n));
  },
  setConstellationZoom(z: number) {
    writeLS(LS_KEYS.constellationZoom, String(Math.max(0.5, Math.min(8, z))));
  },
  // Constellation render source: "symbols" plots the receiver's
  // symbol-decision points (the true constellation — tight clusters for
  // CQPSK, the 4 soft levels for C4FM); "raw" plots the wideband
  // decimated IQ trajectory (a vector scope). Default "symbols".
  constellationView(): "symbols" | "raw" {
    return readLS(LS_KEYS.constellationView) === "raw" ? "raw" : "symbols";
  },
  setConstellationView(v: "symbols" | "raw") {
    writeLS(LS_KEYS.constellationView, v);
  },
  // Receiver selector for the symbols view ("p25-c4fm" / "p25-cqpsk").
  constellationProto(): string {
    return readLS(LS_KEYS.constellationProto) ?? "p25-cqpsk";
  },
  setConstellationProto(p: string) {
    writeLS(LS_KEYS.constellationProto, p);
  },

  // Symbol-scope panel view options. Offset (kHz, relative to the SDR
  // centre) tunes a locked channel under the scope; Hold pins it (off =
  // follow the newest active call). Proto selects the receiver.
  symbolScopeOffsetKHz(): number {
    const raw = readLS(LS_KEYS.symbolScopeOffsetKHz);
    if (raw === null) return 0;
    const n = Number(raw);
    return Number.isFinite(n) ? n : 0;
  },
  setSymbolScopeOffsetKHz(khz: number) {
    writeLS(LS_KEYS.symbolScopeOffsetKHz, String(khz));
  },
  symbolScopeHold(): boolean {
    return readLS(LS_KEYS.symbolScopeHold) === "1";
  },
  setSymbolScopeHold(on: boolean) {
    writeLS(LS_KEYS.symbolScopeHold, on ? "1" : "0");
  },
  symbolScopeProto(): string {
    return readLS(LS_KEYS.symbolScopeProto) ?? "p25-c4fm";
  },
  setSymbolScopeProto(p: string) {
    writeLS(LS_KEYS.symbolScopeProto, p);
  },

  // Eye-diagram panel view options (C4FM datascope). Same offset/Hold
  // semantics as the Symbol scope.
  eyeOffsetKHz(): number {
    const raw = readLS(LS_KEYS.eyeOffsetKHz);
    if (raw === null) return 0;
    const n = Number(raw);
    return Number.isFinite(n) ? n : 0;
  },
  setEyeOffsetKHz(khz: number) {
    writeLS(LS_KEYS.eyeOffsetKHz, String(khz));
  },
  eyeHold(): boolean {
    return readLS(LS_KEYS.eyeHold) === "1";
  },
  setEyeHold(on: boolean) {
    writeLS(LS_KEYS.eyeHold, on ? "1" : "0");
  },

  // Tuning panel view options (receiver-state meters).
  tuningOffsetKHz(): number {
    const raw = readLS(LS_KEYS.tuningOffsetKHz);
    if (raw === null) return 0;
    const n = Number(raw);
    return Number.isFinite(n) ? n : 0;
  },
  setTuningOffsetKHz(khz: number) {
    writeLS(LS_KEYS.tuningOffsetKHz, String(khz));
  },
  tuningHold(): boolean {
    return readLS(LS_KEYS.tuningHold) === "1";
  },
  setTuningHold(on: boolean) {
    writeLS(LS_KEYS.tuningHold, on ? "1" : "0");
  },
  tuningProto(): string {
    return readLS(LS_KEYS.tuningProto) ?? "p25-c4fm";
  },
  setTuningProto(p: string) {
    writeLS(LS_KEYS.tuningProto, p);
  },

  /** Clear all GopherTrunk-owned keys. Used by Settings → "forget this device". */
  clearAll() {
    for (const k of Object.values(LS_KEYS)) writeLS(k, null);
    for (const k of Object.values(SS_KEYS)) writeSS(k, null);
  },
};
