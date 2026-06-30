import { create } from "zustand";
import { api } from "../api/client";
import type { OpSpec, RecipeResult, UploadDTO } from "../api/types";

// BuilderStep is one step in the editable pipeline. Params are held as strings
// (the input controls' values) and coerced to the right JSON type at run time.
export interface BuilderStep {
  id: number;
  op: string;
  params: Record<string, string>;
}

let nextStepID = 1;

interface RecipeStore {
  ops: OpSpec[];
  loadingOps: boolean;
  opsError?: string;
  loadOps: () => Promise<void>;
  opSpec: (op: string) => OpSpec | undefined;

  steps: BuilderStep[];
  addStep: (op: string) => void;
  removeStep: (id: number) => void;
  moveStep: (id: number, dir: -1 | 1) => void;
  duplicateStep: (id: number) => void;
  setStepParam: (id: number, name: string, value: string) => void;
  clearSteps: () => void;

  input?: UploadDTO;
  uploading: boolean;
  uploadInput: (file: File) => Promise<void>;

  running: boolean;
  result?: RecipeResult;
  error?: string;
  run: () => Promise<void>;
}

export const useRecipe = create<RecipeStore>((set, get) => ({
  ops: [],
  loadingOps: false,

  async loadOps() {
    if (get().ops.length > 0 || get().loadingOps) return;
    set({ loadingOps: true, opsError: undefined });
    try {
      const ops = await api.recipeOps();
      set({ ops, loadingOps: false });
    } catch (e) {
      set({ loadingOps: false, opsError: String(e) });
    }
  },

  opSpec(op) {
    return get().ops.find((o) => o.name === op);
  },

  steps: [],

  addStep(op) {
    set((s) => ({ steps: [...s.steps, { id: nextStepID++, op, params: {} }] }));
  },

  removeStep(id) {
    set((s) => ({ steps: s.steps.filter((st) => st.id !== id) }));
  },

  moveStep(id, dir) {
    set((s) => {
      const i = s.steps.findIndex((st) => st.id === id);
      const j = i + dir;
      if (i < 0 || j < 0 || j >= s.steps.length) return s;
      const steps = s.steps.slice();
      [steps[i], steps[j]] = [steps[j], steps[i]];
      return { steps };
    });
  },

  duplicateStep(id) {
    set((s) => {
      const i = s.steps.findIndex((st) => st.id === id);
      if (i < 0) return s;
      const copy: BuilderStep = { id: nextStepID++, op: s.steps[i].op, params: { ...s.steps[i].params } };
      const steps = s.steps.slice();
      steps.splice(i + 1, 0, copy);
      return { steps };
    });
  },

  setStepParam(id, name, value) {
    set((s) => ({
      steps: s.steps.map((st) => (st.id === id ? { ...st, params: { ...st.params, [name]: value } } : st)),
    }));
  },

  clearSteps() {
    set({ steps: [], result: undefined, error: undefined });
  },

  uploading: false,

  async uploadInput(file) {
    set({ uploading: true });
    try {
      const up = await api.upload(file);
      set({ input: up, result: undefined, error: undefined });
    } catch (e) {
      set({ error: String(e) });
    } finally {
      set({ uploading: false });
    }
  },

  running: false,

  async run() {
    const { input, steps, opSpec } = get();
    if (!input) {
      set({ error: "Upload an input file first." });
      return;
    }
    if (steps.length === 0) {
      set({ error: "Add at least one operation." });
      return;
    }
    set({ running: true, error: undefined, result: undefined });
    const payload = steps.map((st) => {
      const spec = opSpec(st.op);
      const params: Record<string, unknown> = {};
      for (const p of spec?.params ?? []) {
        const v = st.params[p.name];
        if (v == null || v === "") continue;
        params[p.name] = p.kind === "int" ? Number(v) : v;
      }
      return { op: st.op, params };
    });
    try {
      const result = await api.recipe({ input_id: input.id, steps: payload });
      set({ running: false, result });
    } catch (e) {
      set({ running: false, error: String(e) });
    }
  },
}));
