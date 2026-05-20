// SharedState for the SPA. Mirrors internal/tui/state/state.go in
// shape and intent: a single live snapshot of everything the panels
// render, plus connection metadata. Built on Zustand to keep
// boilerplate low.

import { create } from "zustand";
import type {
  ActiveCallDTO,
  AudioStatusDTO,
  DeviceDTO,
  EventDTO,
  Health,
  Mutations,
  ScannerStatusDTO,
  SystemDTO,
  TalkgroupDTO,
} from "../api/types";
import type { ClientConfig } from "../api/client";
import { prefs } from "./prefs";

export type ConnectionStatus = "idle" | "connecting" | "open" | "closed";

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
  activeCalls: ActiveCallDTO[];
  devices: DeviceDTO[];
  scanner: ScannerStatusDTO | null;
  events: EventDTO[];
  /** Cap the event ring at this many entries to mirror the TUI. */
  eventCap: number;

  /** Last error surfaced from any request, for the toast strip. */
  lastError: string | null;

  setCredentials(url: string | null, token: string | null, persist: boolean): void;
  setConnected(open: boolean): void;
  setWSStatus(status: ConnectionStatus): void;
  setWriteMode(enabled: boolean): void;
  setMutations(m: Mutations | null): void;
  setHealth(h: Health | null): void;
  setAudio(a: AudioStatusDTO | null): void;
  setSystems(s: SystemDTO[]): void;
  setTalkgroups(t: TalkgroupDTO[]): void;
  setActiveCalls(a: ActiveCallDTO[]): void;
  setDevices(d: DeviceDTO[]): void;
  setScanner(s: ScannerStatusDTO | null): void;
  appendEvent(ev: EventDTO): void;
  setError(msg: string | null): void;
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
  activeCalls: [],
  devices: [],
  scanner: null,
  events: [],
  eventCap: 500,

  lastError: null,

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
  setActiveCalls(a) {
    set({ activeCalls: a });
  },
  setDevices(d) {
    set({ devices: d });
  },
  setScanner(s) {
    set({ scanner: s });
  },
  appendEvent(ev) {
    const cur = get().events;
    const cap = get().eventCap;
    const next = cur.length >= cap ? cur.slice(cur.length - cap + 1) : cur;
    set({ events: [...next, ev] });
  },
  setError(msg) {
    set({ lastError: msg });
  },
  reset() {
    set({
      connected: false,
      wsStatus: "idle",
      health: null,
      audio: null,
      systems: [],
      talkgroups: [],
      activeCalls: [],
      devices: [],
      scanner: null,
      events: [],
      mutations: null,
      lastError: null,
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
