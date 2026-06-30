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

// --- Recipe builder ---

export interface OpParam {
  name: string;
  label: string;
  kind: "hex" | "int";
  help?: string;
}

export interface OpSpec {
  name: string;
  synopsis: string;
  transform: boolean;
  external?: boolean;
  params?: OpParam[];
}

export interface RecipeStepResult {
  op: string;
  transform: boolean;
  bytes_in: number;
  bytes_out: number;
  info?: Record<string, unknown>;
  note?: string;
}

export interface RecipeResult {
  input_bytes: number;
  steps: RecipeStepResult[];
  final_hex: string;
  final_ascii: string;
  final_bytes: number;
  output_b64?: string;
  error?: string;
}
