import { create } from "zustand";
import { api } from "../api/client";
import type { Artifact, LogEvent, ModeSchema, Result, UploadDTO } from "../api/types";

export type RunState = "idle" | "running" | "done" | "error";

interface Run {
  jobId?: string;
  state: RunState;
  error?: string;
  result?: Result;
  artifacts: Artifact[];
  events: LogEvent[];
}

interface Store {
  schema: ModeSchema[];
  loadingSchema: boolean;
  schemaError?: string;
  loadSchema: () => Promise<void>;

  tool?: string;
  mode?: string;
  select: (tool: string, mode: string) => void;
  selected: () => ModeSchema | undefined;

  params: Record<string, string>;
  files: Record<string, UploadDTO>;
  uploading: Record<string, boolean>;
  setParam: (name: string, value: string) => void;
  uploadFor: (name: string, file: File) => Promise<void>;

  run: Run;
  startRun: () => Promise<void>;
}

const idleRun: Run = { state: "idle", artifacts: [], events: [] };

export const useStore = create<Store>((set, get) => ({
  schema: [],
  loadingSchema: false,

  async loadSchema() {
    set({ loadingSchema: true, schemaError: undefined });
    try {
      const schema = await api.tools();
      set({ schema, loadingSchema: false });
      // Land on the triage front door (`classify auto` — "what am I looking
      // at?") when present, else the first mode, for a sensible initial view.
      if (!get().tool && schema.length > 0) {
        const front =
          schema.find((s) => s.tool === "classify" && s.mode === "auto") ?? schema[0];
        get().select(front.tool, front.mode);
      }
    } catch (e) {
      set({ loadingSchema: false, schemaError: String(e) });
    }
  },

  select(tool, mode) {
    const ms = get().schema.find((s) => s.tool === tool && s.mode === mode);
    const params: Record<string, string> = {};
    for (const p of ms?.params ?? []) {
      if (p.kind === "bool") params[p.name] = p.default === "true" ? "true" : "false";
      else if (p.default !== undefined) params[p.name] = p.default;
    }
    set({ tool, mode, params, files: {}, uploading: {}, run: idleRun });
  },

  selected() {
    const { schema, tool, mode } = get();
    return schema.find((s) => s.tool === tool && s.mode === mode);
  },

  params: {},
  files: {},
  uploading: {},

  setParam(name, value) {
    set((s) => ({ params: { ...s.params, [name]: value } }));
  },

  async uploadFor(name, file) {
    set((s) => ({ uploading: { ...s.uploading, [name]: true } }));
    try {
      const up = await api.upload(file);
      set((s) => ({ files: { ...s.files, [name]: up } }));
    } finally {
      set((s) => ({ uploading: { ...s.uploading, [name]: false } }));
    }
  },

  run: idleRun,

  async startRun() {
    const { tool, mode, params, files } = get();
    if (!tool || !mode) return;
    const fileIds: Record<string, string> = {};
    for (const [name, up] of Object.entries(files)) fileIds[name] = up.id;

    set({ run: { state: "running", artifacts: [], events: [] } });
    let jobId: string;
    try {
      const r = await api.run({ tool, mode, params, files: fileIds });
      jobId = r.job_id;
    } catch (e) {
      set({ run: { state: "error", error: String(e), artifacts: [], events: [] } });
      return;
    }
    set((s) => ({ run: { ...s.run, jobId } }));

    const es = new EventSource(api.eventsURL(jobId));
    es.addEventListener("log", (ev) => {
      try {
        const le = JSON.parse((ev as MessageEvent).data) as LogEvent;
        set((s) => ({ run: { ...s.run, events: [...s.run.events, le] } }));
      } catch {
        /* ignore malformed line */
      }
    });
    es.addEventListener("done", () => {
      es.close();
      void api
        .job(jobId)
        .then((job) =>
          set((s) => ({
            run: {
              ...s.run,
              state: job.state === "error" ? "error" : "done",
              error: job.error,
              result: job.result,
              artifacts: job.artifacts ?? [],
            },
          })),
        )
        .catch((e) =>
          set((s) => ({ run: { ...s.run, state: "error", error: String(e) } })),
        );
    });
    es.onerror = () => {
      // The stream closes itself on done; an error before that means the run
      // is unreachable — fall back to a one-shot poll.
      es.close();
      void api.job(jobId).then((job) =>
        set((s) => ({
          run: {
            ...s.run,
            state: job.state === "error" ? "error" : job.state === "done" ? "done" : s.run.state,
            error: job.error,
            result: job.result,
            artifacts: job.artifacts ?? [],
          },
        })),
      );
    };
  },
}));
