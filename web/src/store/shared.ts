// SharedState for the SPA. Mirrors internal/tui/state/state.go in
// shape and intent: a single live snapshot of everything the panels
// render, plus connection metadata. Built on Zustand to keep
// boilerplate low.

import { create } from "zustand";
import type {
  ActiveCallDTO,
  AudioStatusDTO,
  CallEncryptionEvent,
  DeviceDTO,
  EventDTO,
  Health,
  Mutations,
  RIDDTO,
  ScannerStatusDTO,
  SystemDTO,
  TalkgroupDTO,
} from "../api/types";
import type { ClientConfig } from "../api/client";
import { prefs } from "./prefs";

export type ConnectionStatus = "idle" | "connecting" | "open" | "closed";

export type ToastKind = "success" | "error" | "info" | "warning";

export interface Toast {
  id: string;
  kind: ToastKind;
  message: string;
}

// Default time-to-live per kind. Errors and warnings linger longer so
// the operator has time to read them; success/info are transient.
const TOAST_TTL_MS: Record<ToastKind, number> = {
  success: 4_000,
  info: 4_000,
  warning: 7_000,
  error: 8_000,
};

let toastSeq = 0;

interface SharedState {
  serverURL: string | null;
  token: string | null;
  rememberToken: boolean;
  connected: boolean;
  wsStatus: ConnectionStatus;
  writeMode: boolean;
  mutations: Mutations | null;

  health: Health | null;
  audio: AudioStatusDTO | null;
  systems: SystemDTO[];
  talkgroups: TalkgroupDTO[];
  rids: RIDDTO[];
  activeCalls: ActiveCallDTO[];
  devices: DeviceDTO[];
  scanner: ScannerStatusDTO | null;
  events: EventDTO[];
  /** Cap the event ring at this many entries to mirror the TUI. */
  eventCap: number;

  /** Tab keys the operator switched off via web.tabs in config. The
   *  nav bar filters these out. Sourced from /api/v1/runtime. */
  hiddenTabs: string[];

  /** True when the daemon serves the Crypto Lab SPA at /cryptolab/
   *  (a -tags cryptolab build). Gates the Crypto Lab nav link. */
  cryptolabConsole: boolean;

  /** Base for rendering identity numbers (WACN, System ID, NAC, RFSS,
   *  Site): "hex" (default, P25 field convention) or "dec". Sourced from
   *  web.id_base config via /api/v1/runtime. */
  idBase: "hex" | "dec";

  /** Absolute path of the config.yaml backing the daemon, or "" when it
   *  runs on built-in defaults (no -config). null until the runtime
   *  snapshot has been fetched. Sourced from /api/v1/runtime. */
  configPath: string | null;

  /** Last error surfaced from any request. Retained for compatibility;
   *  `setError` now also enqueues an error toast. */
  lastError: string | null;

  /** Transient notifications shown by the ToastViewport. */
  toasts: Toast[];

  setCredentials(url: string | null, token: string | null, persist: boolean): void;
  setConnected(open: boolean): void;
  setWSStatus(status: ConnectionStatus): void;
  setWriteMode(enabled: boolean): void;
  setMutations(m: Mutations | null): void;
  setHealth(h: Health | null): void;
  setAudio(a: AudioStatusDTO | null): void;
  setSystems(s: SystemDTO[]): void;
  setTalkgroups(t: TalkgroupDTO[]): void;
  setRIDs(r: RIDDTO[]): void;
  setActiveCalls(a: ActiveCallDTO[]): void;
  setDevices(d: DeviceDTO[]): void;
  setScanner(s: ScannerStatusDTO | null): void;
  setHiddenTabs(tabs: string[]): void;
  setCryptolabConsole(v: boolean): void;
  setIDBase(base: "hex" | "dec"): void;
  setConfigPath(path: string | null): void;
  appendEvents(evs: EventDTO[]): void;
  setError(msg: string | null): void;
  /** Enqueue a toast. Returns the toast id (so callers can dismiss it
   *  early if needed). Auto-dismisses after a kind-dependent TTL. */
  notify(kind: ToastKind, message: string, ttlMs?: number): string;
  dismissToast(id: string): void;
  reset(): void;
}

export const useShared = create<SharedState>((set, get) => ({
  serverURL: prefs.serverURL(),
  token: prefs.token(),
  rememberToken: prefs.rememberToken(),
  connected: false,
  wsStatus: "idle",
  writeMode: prefs.writeMode(),
  mutations: null,

  health: null,
  audio: null,
  systems: [],
  talkgroups: [],
  rids: [],
  activeCalls: [],
  devices: [],
  scanner: null,
  events: [],
  eventCap: 500,
  hiddenTabs: [],
  cryptolabConsole: false,
  idBase: "hex",
  configPath: null,

  lastError: null,
  toasts: [],

  setCredentials(url, token, persist) {
    prefs.setServerURL(url);
    prefs.setToken(token, persist);
    set({ serverURL: url, token, rememberToken: persist });
  },
  setConnected(open) {
    set({ connected: open });
  },
  setWSStatus(status) {
    set({ wsStatus: status });
  },
  setWriteMode(enabled) {
    prefs.setWriteMode(enabled);
    set({ writeMode: enabled });
  },
  setMutations(m) {
    set({ mutations: m });
  },
  setHealth(h) {
    set({ health: h });
  },
  setAudio(a) {
    set({ audio: a });
  },
  setSystems(s) {
    set({ systems: s });
  },
  setTalkgroups(t) {
    set({ talkgroups: t });
  },
  setRIDs(r) {
    set({ rids: r });
  },
  setActiveCalls(a) {
    set({ activeCalls: a });
  },
  setDevices(d) {
    set({ devices: d });
  },
  setScanner(s) {
    set({ scanner: s });
  },
  setHiddenTabs(tabs) {
    set({ hiddenTabs: tabs });
  },
  setCryptolabConsole(v) {
    set({ cryptolabConsole: v });
  },
  setIDBase(base) {
    set({ idBase: base });
  },
  setConfigPath(path) {
    set({ configPath: path });
  },
  appendEvents(evs) {
    if (evs.length === 0) return;
    const cap = get().eventCap;
    const combined = get().events.concat(evs);
    const next =
      combined.length > cap
        ? combined.slice(combined.length - cap)
        : combined;
    // Patch active calls in flight when an encryption-metadata update
    // arrives. The poll loop in Active.tsx would pick this up within
    // ~2 s, but patching synchronously avoids the "enc" pill flashing
    // without its algorithm name for that window.
    let activeCalls = get().activeCalls;
    let patched = false;
    for (const ev of evs) {
      if (ev.kind !== "call.encryption" || ev.payload == null) continue;
      const p = ev.payload as CallEncryptionEvent;
      if (!p.device_serial) continue;
      const idx = activeCalls.findIndex(
        (ac) => ac.device_serial === p.device_serial,
      );
      if (idx < 0) continue;
      const ac = activeCalls[idx];
      activeCalls = activeCalls.slice();
      activeCalls[idx] = {
        ...ac,
        grant: {
          ...ac.grant,
          algorithm_id: p.algorithm_id,
          key_id: p.key_id,
        },
      };
      patched = true;
    }
    set(patched ? { events: next, activeCalls } : { events: next });
  },
  setError(msg) {
    // Retain lastError for any readers, and surface non-null errors as
    // a toast so every existing setError(...) call site now gives the
    // operator visible feedback.
    set({ lastError: msg });
    if (msg) get().notify("error", msg);
  },
  notify(kind, message, ttlMs) {
    const id = `t${++toastSeq}`;
    set({ toasts: [...get().toasts, { id, kind, message }] });
    const ttl = ttlMs ?? TOAST_TTL_MS[kind];
    if (ttl > 0 && typeof window !== "undefined") {
      window.setTimeout(() => get().dismissToast(id), ttl);
    }
    return id;
  },
  dismissToast(id) {
    set({ toasts: get().toasts.filter((t) => t.id !== id) });
  },
  reset() {
    set({
      connected: false,
      wsStatus: "idle",
      health: null,
      audio: null,
      systems: [],
      talkgroups: [],
      rids: [],
      activeCalls: [],
      devices: [],
      scanner: null,
      events: [],
      mutations: null,
      hiddenTabs: [],
      cryptolabConsole: false,
      idBase: "hex",
      configPath: null,
      lastError: null,
      toasts: [],
    });
  },
}));

/** Convenience selector: a referentially-stable ClientConfig snapshot.
 *  Returns the SAME object reference until serverURL or token actually
 *  change, so callers can place `cfg` in useEffect dependency arrays
 *  without triggering a render loop (see issue #290). */
let cachedCfg: ClientConfig = { baseURL: "", token: null };
export function selectClientConfig(s: SharedState): ClientConfig {
  const baseURL = s.serverURL ?? "";
  if (cachedCfg.baseURL !== baseURL || cachedCfg.token !== s.token) {
    cachedCfg = { baseURL, token: s.token };
  }
  return cachedCfg;
}

/** Can-mutate gate combining write-mode toggle with daemon capability. */
export function selectCanMutate(s: SharedState): boolean {
  return s.writeMode && (s.mutations?.allow_mutations ?? false);
}
