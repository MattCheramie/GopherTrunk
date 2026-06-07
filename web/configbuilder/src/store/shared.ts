import { create } from "zustand";
import { api, setToken } from "../api/client";
import type {
  DocLink,
  GTConfig,
  TalkgroupCSVRow,
  ValidationResult,
} from "../api/types";

interface State {
  // The config being edited, plus where it came from.
  config: GTConfig | null;
  path: string; // current file path ("" for an unsaved new config)
  mtime: number; // guard for optimistic save (0 = new file)
  dirty: boolean;

  // Talkgroup CSV sidecars staged to write on save, keyed by the
  // relative TalkgroupFile path the config references.
  talkgroups: Record<string, TalkgroupCSVRow[]>;

  docs: Record<string, DocLink>;
  token: string;
  lastError: string | null;
  lastSaved: string | null;
  validation: ValidationResult | null;

  // Actions.
  init: () => Promise<void>;
  newConfig: () => Promise<void>;
  load: (path: string) => Promise<void>;
  setConfig: (c: GTConfig) => void;
  patchSection: <K extends keyof GTConfig>(key: K, value: GTConfig[K]) => void;
  stageTalkgroups: (rel: string, rows: TalkgroupCSVRow[]) => void;
  setToken: (t: string) => void;
  setError: (e: string | null) => void;
  validateAll: () => Promise<void>;
  save: (path: string, overwrite: boolean) => Promise<boolean>;
}

export const useStore = create<State>((set, get) => ({
  config: null,
  path: "",
  mtime: 0,
  dirty: false,
  talkgroups: {},
  docs: {},
  token: "",
  lastError: null,
  lastSaved: null,
  validation: null,

  init: async () => {
    try {
      const docs = await api.docs();
      set({ docs });
    } catch {
      /* docs are best-effort */
    }
    if (!get().config) {
      await get().newConfig();
    }
  },

  newConfig: async () => {
    try {
      const cfg = await api.defaults();
      set({
        config: cfg,
        path: "",
        mtime: 0,
        dirty: false,
        talkgroups: {},
        lastSaved: null,
        validation: null,
      });
      await get().validateAll();
    } catch (e) {
      set({ lastError: `Could not load defaults: ${(e as Error).message}` });
    }
  },

  load: async (path) => {
    try {
      const resp = await api.loadFile(path);
      set({
        config: resp.config,
        path: resp.path,
        mtime: resp.mtime,
        dirty: false,
        talkgroups: {},
        validation: resp.validation,
        lastSaved: null,
      });
      // Pull in the talkgroup sidecars so the per-system editor can show
      // and edit them (best-effort — a config with none just stays empty).
      try {
        const tg = await api.talkgroups(resp.path);
        if (tg.talkgroups) set({ talkgroups: tg.talkgroups });
      } catch {
        /* sidecars optional */
      }
    } catch (e) {
      set({ lastError: `Load failed: ${(e as Error).message}` });
    }
  },

  setConfig: (c) => set({ config: c, dirty: true }),

  patchSection: (key, value) => {
    const cur = get().config;
    if (!cur) return;
    set({ config: { ...cur, [key]: value }, dirty: true });
  },

  stageTalkgroups: (rel, rows) =>
    set((s) => ({ talkgroups: { ...s.talkgroups, [rel]: rows }, dirty: true })),

  setToken: (t) => {
    setToken(t);
    set({ token: t });
  },

  setError: (e) => set({ lastError: e }),

  validateAll: async () => {
    const cfg = get().config;
    if (!cfg) return;
    try {
      const v = await api.validate(cfg);
      set({ validation: v });
    } catch (e) {
      set({ lastError: `Validate failed: ${(e as Error).message}` });
    }
  },

  save: async (path, overwrite) => {
    const cfg = get().config;
    if (!cfg) return false;
    try {
      const resp = await api.save({
        path,
        config: cfg,
        mtime: path === get().path ? get().mtime : 0,
        overwrite,
        talkgroups: Object.keys(get().talkgroups).length ? get().talkgroups : undefined,
      });
      set({
        path: resp.path,
        mtime: resp.mtime,
        dirty: false,
        lastSaved: `Saved to ${resp.path}`,
        lastError: null,
      });
      // Resync staged talkgroups with what's now on disk so the editor
      // reflects the persisted state (and a later unrelated save doesn't
      // re-stage stale rows).
      try {
        const tg = await api.talkgroups(resp.path);
        set({ talkgroups: tg.talkgroups ?? {} });
      } catch {
        /* best-effort */
      }
      return true;
    } catch (e) {
      set({ lastError: `Save failed: ${(e as Error).message}` });
      return false;
    }
  },
}));
