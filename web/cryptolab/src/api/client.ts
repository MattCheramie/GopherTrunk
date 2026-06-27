import type { JobDTO, ModeSchema, UploadDTO } from "./types";

// Same-origin API. The standalone `cryptolab serve` binds loopback with no
// token; a remote bind can pass one via #token=… (see main.tsx).
let token: string | null = null;
export function setToken(t: string): void {
  token = t;
}

function authHeaders(): Record<string, string> {
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function reqJSON<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { Accept: "application/json", ...authHeaders() };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const res = await fetch(path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await res.text();
  if (!res.ok) {
    let msg = text;
    try {
      const j = JSON.parse(text) as { error?: string };
      if (j.error) msg = j.error;
    } catch {
      /* keep raw text */
    }
    throw new Error(msg || `${res.status} ${res.statusText}`);
  }
  return (text ? JSON.parse(text) : {}) as T;
}

export interface RunBody {
  tool: string;
  mode: string;
  params: Record<string, string>;
  files: Record<string, string>;
}

export const api = {
  tools: () => reqJSON<ModeSchema[]>("GET", "/api/v1/cryptolab/tools"),

  upload: async (file: File): Promise<UploadDTO> => {
    const fd = new FormData();
    fd.append("file", file);
    const res = await fetch("/api/v1/cryptolab/files", {
      method: "POST",
      headers: authHeaders(),
      body: fd,
    });
    if (!res.ok) throw new Error(await res.text());
    return (await res.json()) as UploadDTO;
  },

  run: (body: RunBody) => reqJSON<{ job_id: string }>("POST", "/api/v1/cryptolab/run", body),

  job: (id: string) => reqJSON<JobDTO>("GET", `/api/v1/cryptolab/jobs/${id}`),

  eventsURL: (id: string) => `/api/v1/cryptolab/jobs/${id}/events`,

  artifactURL: (id: string, name: string) =>
    `/api/v1/cryptolab/jobs/${id}/artifacts/${encodeURIComponent(name)}`,
};
