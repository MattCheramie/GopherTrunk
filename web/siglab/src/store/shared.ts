import { create } from "zustand";
import { api, type ClientConfig } from "../api/client";
import type {
  CaptureDTO,
  CaptureRequest,
  CaptureResponse,
  EventRecord,
  IQTaps,
  IdentifyResult,
  JobDTO,
  Result,
  RunConfig,
  SynthRequest,
  WidebandRequest,
  WidebandResult,
} from "../api/types";

// Analysis is the per-capture run state: the job id, live-streamed events,
// the final Result, and the (lazily fetched) decimated IQ taps.
export interface Analysis {
  captureId: string;
  jobId?: string;
  state: "idle" | "running" | "done" | "error";
  error?: string;
  events: EventRecord[];
  result?: Result;
  iqTaps?: IQTaps;
  config?: RunConfig;
}

interface Store {
  config: ClientConfig;
  setConfig: (c: Partial<ClientConfig>) => void;

  protocols: string[];
  fixtures: string[];
  formats: string[];
  loadProtocols: () => Promise<void>;

  captures: CaptureDTO[];
  analyses: Record<string, Analysis>;
  selectedCaptureId?: string;
  compareIds: string[];

  addCapture: (c: CaptureDTO) => void;
  uploadCapture: (file: File, format: string, sampleRateHz?: number) => Promise<CaptureDTO>;
  synthesize: (body: SynthRequest) => Promise<CaptureDTO>;
  captureFromTuner: (req: CaptureRequest) => Promise<CaptureResponse>;
  removeCapture: (id: string) => Promise<void>;
  select: (id: string) => void;
  toggleCompare: (id: string) => void;

  run: (captureId: string, config: RunConfig) => Promise<void>;
  fetchIQ: (captureId: string) => Promise<IQTaps | undefined>;
  identify: (captureId: string, sampleRateHz?: number) => Promise<IdentifyResult>;
  wideband: (captureId: string, opts: Partial<WidebandRequest>) => Promise<WidebandResult>;
}

export const useStore = create<Store>((set, get) => ({
  config: { baseURL: "", token: null },
  setConfig: (c) => set((s) => ({ config: { ...s.config, ...c } })),

  protocols: [],
  fixtures: [],
  formats: ["u8", "f32"],
  loadProtocols: async () => {
    const p = await api.protocols(get().config);
    set({ protocols: p.protocols, fixtures: p.fixtures, formats: p.formats });
  },

  captures: [],
  analyses: {},
  selectedCaptureId: undefined,
  compareIds: [],

  addCapture: (c) =>
    set((s) => ({
      captures: [c, ...s.captures.filter((x) => x.id !== c.id)],
      selectedCaptureId: c.id,
      analyses: { ...s.analyses, [c.id]: emptyAnalysis(c.id) },
    })),

  uploadCapture: async (file, format, sampleRateHz) => {
    const dto = await api.uploadCapture(get().config, file, {
      format,
      sample_rate_hz: sampleRateHz,
    });
    get().addCapture(dto);
    return dto;
  },

  synthesize: async (body) => {
    const res = await api.synthesize(get().config, body);
    get().addCapture(res.capture);
    return res.capture;
  },

  captureFromTuner: async (req) => {
    const res = await api.capture(get().config, req);
    get().addCapture(res.capture);
    return res;
  },

  removeCapture: async (id) => {
    try {
      await api.deleteCapture(get().config, id);
    } catch {
      /* best-effort; drop locally regardless */
    }
    set((s) => {
      const analyses = { ...s.analyses };
      delete analyses[id];
      return {
        captures: s.captures.filter((c) => c.id !== id),
        analyses,
        compareIds: s.compareIds.filter((x) => x !== id),
        selectedCaptureId:
          s.selectedCaptureId === id ? undefined : s.selectedCaptureId,
      };
    });
  },

  select: (id) => set({ selectedCaptureId: id }),
  toggleCompare: (id) =>
    set((s) => ({
      compareIds: s.compareIds.includes(id)
        ? s.compareIds.filter((x) => x !== id)
        : [...s.compareIds, id],
    })),

  run: async (captureId, config) => {
    const cfg = get().config;
    set((s) => ({
      analyses: {
        ...s.analyses,
        [captureId]: { ...emptyAnalysis(captureId), state: "running", config },
      },
    }));
    const { job_id } = await api.run(cfg, captureId, config);
    patch(set, captureId, { jobId: job_id });

    // Live event stream via SSE. On the terminal "done" event we fetch the
    // final result (and IQ taps when captured).
    await new Promise<void>((resolve) => {
      const es = new EventSource(api.jobEventsURL(cfg, job_id), {
        withCredentials: true,
      });
      es.onmessage = (e) => appendEvent(set, get, captureId, e.data);
      // Named events: every EventRecord kind, plus the terminal "done".
      es.addEventListener("done", async () => {
        es.close();
        await finalize(set, get, captureId, job_id, config);
        resolve();
      });
      es.onerror = async () => {
        // SSE may error after the stream ends; finalize defensively.
        es.close();
        await finalize(set, get, captureId, job_id, config);
        resolve();
      };
    });
  },

  fetchIQ: async (captureId) => {
    const a = get().analyses[captureId];
    if (!a?.jobId) return undefined;
    if (a.iqTaps) return a.iqTaps;
    const taps = await api.jobIQ(get().config, a.jobId);
    patch(set, captureId, { iqTaps: taps });
    return taps;
  },

  identify: async (captureId, sampleRateHz) => {
    const cap = get().captures.find((c) => c.id === captureId);
    return api.identify(get().config, captureId, {
      sample_rate_hz: sampleRateHz ?? cap?.sample_rate_hz,
      format: cap?.format,
    });
  },

  wideband: async (captureId, opts) => {
    const cap = get().captures.find((c) => c.id === captureId);
    return api.wideband(get().config, {
      capture_id: captureId,
      sample_rate_hz: cap?.sample_rate_hz,
      format: cap?.format,
      ...opts,
    });
  },
}));

function emptyAnalysis(captureId: string): Analysis {
  return { captureId, state: "idle", events: [] };
}

function patch(
  set: (fn: (s: Store) => Partial<Store>) => void,
  captureId: string,
  p: Partial<Analysis>,
) {
  set((s) => {
    const prev = s.analyses[captureId] ?? emptyAnalysis(captureId);
    return { analyses: { ...s.analyses, [captureId]: { ...prev, ...p } } };
  });
}

function appendEvent(
  set: (fn: (s: Store) => Partial<Store>) => void,
  get: () => Store,
  captureId: string,
  data: string,
) {
  let ev: EventRecord;
  try {
    ev = JSON.parse(data) as EventRecord;
  } catch {
    return;
  }
  const prev = get().analyses[captureId];
  if (!prev) return;
  patch(set, captureId, { events: [...prev.events.slice(-2000), ev] });
}

async function finalize(
  set: (fn: (s: Store) => Partial<Store>) => void,
  get: () => Store,
  captureId: string,
  jobId: string,
  config: RunConfig,
) {
  const a = get().analyses[captureId];
  if (a && (a.state === "done" || a.state === "error")) return;
  try {
    const job: JobDTO = await api.job(get().config, jobId);
    if (job.state === "error") {
      patch(set, captureId, { state: "error", error: job.error });
      return;
    }
    patch(set, captureId, { state: "done", result: job.result });
    if (job.has_iq && config.capture_iq) {
      try {
        const taps = await api.jobIQ(get().config, jobId);
        patch(set, captureId, { iqTaps: taps });
      } catch {
        /* IQ optional */
      }
    }
  } catch (err) {
    patch(set, captureId, {
      state: "error",
      error: err instanceof Error ? err.message : String(err),
    });
  }
}
