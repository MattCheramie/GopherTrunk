import { create } from "zustand";
import { api, setToken } from "../api/client";
import type {
  DocLink,
  FieldMeta,
  GTConfig,
  TalkgroupCSVRow,
  ValidationResult,
} from "../api/types";

// pushErr appends a message to the error toast stack.
function pushErr(set: (fn: (s: State) => Partial<State>) => void, msg: string) {
  set((s) => ({ errors: [...s.errors, msg] }));
}

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
  // fieldmeta is the shared per-field registry (keyed "StructName.FieldName")
  // fetched from /config/fieldmeta; fieldHelp() reads it so web field help is
  // sourced from the same Go registry the terminal builder uses.
  fieldmeta: Record<string, FieldMeta>;
  defaults: GTConfig | null; // config.Default() seed, for "configured" nav cues
  token: string;
  errors: string[]; // toast stack
  busy: boolean; // an async op (load/save/validate) is in flight
  lastSaved: string | null;
  validation: ValidationResult | null;

  // Actions.
  init: () => Promise<void>;
  newConfig: () => Promise<void>;
  load: (path: string) => Promise<void>;
  revert: () => Promise<void>;
  setConfig: (c: GTConfig) => void;
  patchSection: <K extends keyof GTConfig>(key: K, value: GTConfig[K]) => void;
  stageTalkgroups: (rel: string, rows: TalkgroupCSVRow[]) => void;
  setToken: (t: string) => void;
  setError: (e: string | null) => void; // null clears all; string pushes
  dismissError: (i: number) => void;
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
  fieldmeta: {},
  defaults: null,
  token: "",
  errors: [],
  busy: false,
  lastSaved: null,
  validation: null,

  init: async () => {
    try {
      const docs = await api.docs();
      set({ docs });
    } catch {
      /* docs are best-effort */
    }
    try {
      const fieldmeta = await api.fieldmeta();
      set({ fieldmeta });
    } catch {
      /* field help is best-effort */
    }
    // Cache the defaults once so the nav can flag which sections differ
    // from a blank config ("configured" cues).
    try {
      set({ defaults: await api.defaults() });
    } catch {
      /* best-effort */
    }
    if (!get().config) {
      await get().newConfig();
    }
  },

  revert: async () => {
    const p = get().path;
    if (p) {
      await get().load(p);
    } else {
      await get().newConfig();
    }
  },

  newConfig: async () => {
    set({ busy: true });
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
      pushErr(set, `Could not load defaults: ${(e as Error).message}`);
    } finally {
      set({ busy: false });
    }
  },

  load: async (path) => {
    set({ busy: true });
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
      pushErr(set, `Load failed: ${(e as Error).message}`);
    } finally {
      set({ busy: false });
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

  setError: (e) =>
    set((s) => (e === null ? { errors: [] } : { errors: [...s.errors, e] })),
  dismissError: (i) => set((s) => ({ errors: s.errors.filter((_, k) => k !== i) })),

  validateAll: async () => {
    const cfg = get().config;
    if (!cfg) return;
    try {
      const v = await api.validate(cfg);
      set({ validation: v });
    } catch (e) {
      pushErr(set, `Validate failed: ${(e as Error).message}`);
    }
  },

  save: async (path, overwrite) => {
    const cfg = get().config;
    if (!cfg) return false;
    set({ busy: true });
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
        errors: [],
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
      pushErr(set, `Save failed: ${(e as Error).message}`);
      return false;
    } finally {
      set({ busy: false });
    }
  },
}));

// fieldHelp returns the shared per-field help for a "StructName"+"FieldName"
// pair from the fieldmeta registry (fetched once at init), or undefined when
// the registry hasn't loaded / has no entry. Non-reactive on purpose — the
// registry is static for the session — so it can feed widget `help` props
// inline. The Go side guarantees every config leaf has help (TestFieldHelpCoverage).
export function fieldHelp(struct: string, field: string): string | undefined {
  return useStore.getState().fieldmeta[`${struct}.${field}`]?.help || undefined;
}
