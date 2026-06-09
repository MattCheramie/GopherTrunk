import type {
  ConfigListResponse,
  ConfigLoadResponse,
  ConfigSaveResponse,
  DocLink,
  FieldMeta,
  GTConfig,
  ParsedSystemDTO,
  RRGeoRef,
  RRSearchHit,
  RRSystemResponse,
  RRVerifyResponse,
  TalkgroupCSVRow,
  ValidationResult,
} from "./types";

// The SPA is served same-origin (standalone at /, or /config/ on the
// daemon), so relative /api/v1 URLs resolve to the right server. An
// optional token is sent as a Bearer header when the daemon requires auth.
let authToken = "";
export function setToken(t: string) {
  authToken = t.trim();
}
export function getToken() {
  return authToken;
}

export class HTTPError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function req<T>(
  method: string,
  path: string,
  body?: unknown,
  extraHeaders?: Record<string, string>,
): Promise<T> {
  const headers: Record<string, string> = { ...(extraHeaders ?? {}) };
  if (authToken) headers["Authorization"] = `Bearer ${authToken}`;
  let payload: BodyInit | undefined;
  if (body instanceof FormData) {
    payload = body;
  } else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const resp = await fetch(`/api/v1${path}`, { method, headers, body: payload });
  const text = await resp.text();
  if (!resp.ok) {
    let msg = text;
    try {
      const j = JSON.parse(text);
      msg = j.error ?? text;
    } catch {
      /* keep raw text */
    }
    throw new HTTPError(resp.status, msg || `HTTP ${resp.status}`);
  }
  return (text ? JSON.parse(text) : {}) as T;
}

export const api = {
  listFiles: () => req<ConfigListResponse>("GET", "/config/files"),
  loadFile: (path: string) =>
    req<ConfigLoadResponse>("GET", `/config/file?path=${encodeURIComponent(path)}`),
  talkgroups: (path: string) =>
    req<{ talkgroups: Record<string, TalkgroupCSVRow[]> }>(
      "GET",
      `/config/file/talkgroups?path=${encodeURIComponent(path)}`,
    ),
  deleteFile: (path: string) =>
    req<{ deleted: string }>("DELETE", `/config/file?path=${encodeURIComponent(path)}`),
  renameFile: (from: string, to: string) =>
    req<{ from: string; to: string }>("POST", "/config/file/rename", { from, to }),
  mkdir: (path: string) => req<{ path: string }>("POST", "/config/dir", { path }),
  defaults: () => req<GTConfig>("GET", "/config/defaults"),
  docs: () => req<Record<string, DocLink>>("GET", "/config/docs"),
  fieldmeta: () => req<Record<string, FieldMeta>>("GET", "/config/fieldmeta"),
  validate: (config: GTConfig, section?: string) =>
    req<ValidationResult>("POST", "/config/validate", { config, section: section ?? "" }),
  marshal: (config: GTConfig) =>
    req<{ yaml: string }>("POST", "/config/marshal", config),
  save: (body: {
    path: string;
    config: GTConfig;
    mtime?: number;
    overwrite: boolean;
    talkgroups?: Record<string, TalkgroupCSVRow[]>;
  }) => req<ConfigSaveResponse>("POST", "/config/file", body),
  parse: (files: File[]) => {
    const fd = new FormData();
    for (const f of files) fd.append("files", f);
    return req<{ systems: ParsedSystemDTO[] }>("POST", "/config/parse", fd);
  },
  rrSearch: (kind: "zip" | "county" | "state", value: string, creds?: RRCreds) =>
    req<{ results: RRSearchHit[] | null }>(
      "GET",
      `/config/rr/search?${kind}=${encodeURIComponent(value)}`,
      undefined,
      rrHeaders(creds),
    ),
  rrStates: (creds?: RRCreds) =>
    req<{ results: RRGeoRef[] | null }>("GET", "/config/rr/states", undefined, rrHeaders(creds)),
  rrCounties: (stid: number, creds?: RRCreds) =>
    req<{ results: RRGeoRef[] | null }>(
      "GET",
      `/config/rr/counties?state=${stid}`,
      undefined,
      rrHeaders(creds),
    ),
  rrSystem: (sid: number, creds?: RRCreds) =>
    req<RRSystemResponse>("GET", `/config/rr/system/${sid}`, undefined, rrHeaders(creds)),
  rrVerify: (creds?: RRCreds) =>
    req<RRVerifyResponse>("POST", "/config/rr/verify", undefined, rrHeaders(creds)),
};

// RRCreds are the RadioReference credentials the user is editing, sent on the
// browse/verify calls as X-RR-* headers so what they type is what reaches RR
// (and verifies their premium subscription) — kept out of the URL/query.
export type RRCreds = { key?: string; user?: string; pass?: string };

function rrHeaders(c?: RRCreds): Record<string, string> {
  const h: Record<string, string> = {};
  if (c?.key?.trim()) h["X-RR-Key"] = c.key.trim();
  if (c?.user?.trim()) h["X-RR-User"] = c.user.trim();
  if (c?.pass?.trim()) h["X-RR-Pass"] = c.pass.trim();
  return h;
}
