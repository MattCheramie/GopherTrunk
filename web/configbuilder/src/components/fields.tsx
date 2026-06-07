import type { ReactNode } from "react";

export function TextField(props: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  help?: ReactNode;
  type?: string;
}) {
  return (
    <label className="block">
      <span className="label">{props.label}</span>
      <input
        className="input"
        type={props.type ?? "text"}
        value={props.value ?? ""}
        placeholder={props.placeholder}
        onChange={(e) => props.onChange(e.target.value)}
      />
      {props.help ? <p className="help mt-1">{props.help}</p> : null}
    </label>
  );
}

export function NumberField(props: {
  label: string;
  value: number;
  onChange: (v: number) => void;
  placeholder?: string;
  help?: ReactNode;
  step?: number;
}) {
  return (
    <label className="block">
      <span className="label">{props.label}</span>
      <input
        className="input"
        type="number"
        step={props.step}
        value={Number.isFinite(props.value) ? props.value : 0}
        placeholder={props.placeholder}
        onChange={(e) => props.onChange(e.target.value === "" ? 0 : Number(e.target.value))}
      />
      {props.help ? <p className="help mt-1">{props.help}</p> : null}
    </label>
  );
}

export function BoolField(props: {
  label: string;
  value: boolean;
  onChange: (v: boolean) => void;
  help?: ReactNode;
}) {
  return (
    <label className="flex items-start gap-2 py-1">
      <input
        type="checkbox"
        className="mt-1"
        checked={!!props.value}
        onChange={(e) => props.onChange(e.target.checked)}
      />
      <span>
        <span className="text-sm">{props.label}</span>
        {props.help ? <p className="help">{props.help}</p> : null}
      </span>
    </label>
  );
}

export function SelectField(props: {
  label: string;
  value: string;
  options: { value: string; label: string }[];
  onChange: (v: string) => void;
  help?: ReactNode;
}) {
  return (
    <label className="block">
      <span className="label">{props.label}</span>
      <select
        className="input"
        value={props.value ?? ""}
        onChange={(e) => props.onChange(e.target.value)}
      >
        {props.options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
      {props.help ? <p className="help mt-1">{props.help}</p> : null}
    </label>
  );
}

// FreqListField edits a list of integer Hz values as a comma/space/newline
// separated textarea, accepting MHz or Hz per line (values < 10000 are
// treated as MHz for convenience).
export function FreqListField(props: {
  label: string;
  value: number[] | null;
  onChange: (v: number[]) => void;
  help?: ReactNode;
}) {
  const text = (props.value ?? []).map(hzToDisplay).join("\n");
  return (
    <label className="block">
      <span className="label">{props.label}</span>
      <textarea
        className="input font-mono"
        rows={Math.max(2, (props.value ?? []).length + 1)}
        value={text}
        onChange={(e) => props.onChange(parseFreqList(e.target.value))}
      />
      <p className="help mt-1">
        One frequency per line. Accepts MHz (851.0375) or Hz (851037500).
        {props.help ? <> {props.help}</> : null}
      </p>
    </label>
  );
}

export function hzToDisplay(hz: number): string {
  if (!hz) return "0";
  // Show MHz with up to 6 decimals, trimming trailing zeros.
  return (hz / 1e6).toFixed(6).replace(/\.?0+$/, "") + " MHz";
}

export function parseFreqList(text: string): number[] {
  const out: number[] = [];
  for (const raw of text.split(/[\n,;]+/)) {
    const tok = raw.replace(/mhz|hz/gi, "").trim();
    if (!tok) continue;
    const n = Number(tok);
    if (!Number.isFinite(n) || n <= 0) continue;
    // Heuristic: a bare value below 10000 is MHz; otherwise Hz.
    out.push(n < 10000 ? Math.round(n * 1e6) : Math.round(n));
  }
  return out;
}
