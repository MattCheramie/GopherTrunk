// DTOs mirroring the Go cryptolab schema, Result, and web-server job shapes.

export interface Param {
  name: string;
  label: string;
  kind: "file" | "outfile" | "string" | "int" | "bool" | "select";
  required: boolean;
  default?: string;
  help?: string;
  options?: string[];
}

export interface ModeSchema {
  tool: string;
  mode: string;
  synopsis: string;
  params: Param[] | null;
}

export interface ResultField {
  key: string;
  value: unknown;
}

export interface Finding {
  label: string;
  score: number;
  detail?: Record<string, unknown>;
}

export interface Result {
  tool: string;
  mode: string;
  summary: string;
  fields?: ResultField[];
  findings?: Finding[];
  notes?: string[];
}

export interface Artifact {
  name: string;
  size: number;
}

export interface JobDTO {
  id: string;
  tool: string;
  mode: string;
  state: "running" | "done" | "error";
  error?: string;
  events: number;
  result?: Result;
  artifacts?: Artifact[];
}

export interface UploadDTO {
  id: string;
  name: string;
  size: number;
}

export interface LogEvent {
  seq: number;
  time: string;
  level: string;
  msg: string;
  attrs?: string;
}
