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

  /** Clear all GopherTrunk-owned keys. Used by Settings → "forget this device". */
  clearAll() {
    for (const k of Object.values(LS_KEYS)) writeLS(k, null);
    for (const k of Object.values(SS_KEYS)) writeSS(k, null);
  },
};
