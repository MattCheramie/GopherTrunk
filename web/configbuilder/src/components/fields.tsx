import { useEffect, useRef, useState, type ReactNode } from "react";

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
  // A fractional step (e.g. 0.1) opts a field into decimal entry. Controlled
  // type="number" inputs can't be typed with decimals — the browser reports an
  // intermediate value like "131." as "", which would reset the field to 0 and
  // wipe the keystroke — so decimal fields use a buffered text input instead.
  if (props.step != null && !Number.isInteger(props.step)) {
    return <DecimalField {...props} />;
  }
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

// DecimalField backs the decimal NumberField path. Like HzField it keeps a
// local text buffer so mid-edit values like "131." or "-" aren't reformatted or
// wiped on every keystroke, resyncing when the bound value changes out-of-band
// (e.g. switching items in a ListEditor). Emits a finite number, or 0 for
// empty/unparseable input, matching the integer path's contract.
function DecimalField(props: {
  label: string;
  value: number;
  onChange: (v: number) => void;
  placeholder?: string;
  help?: ReactNode;
}) {
  const [text, setText] = useState(Number.isFinite(props.value) ? String(props.value) : "");
  const emitted = useRef(props.value);
  useEffect(() => {
    if (props.value !== emitted.current) {
      emitted.current = props.value;
      setText(Number.isFinite(props.value) ? String(props.value) : "");
    }
  }, [props.value]);
  const onText = (raw: string) => {
    setText(raw);
    const n = raw.trim() === "" ? 0 : Number(raw);
    const v = Number.isFinite(n) ? n : 0;
    emitted.current = v;
    props.onChange(v);
  };
  return (
    <label className="block">
      <span className="label">{props.label}</span>
      <input
        className="input"
        type="text"
        inputMode="decimal"
        value={text}
        placeholder={props.placeholder}
        onChange={(e) => onText(e.target.value)}
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
//
// It keeps a local text buffer (like HzField) instead of deriving the textarea
// value from the parsed list every render. Deriving it swallowed newlines: a
// trailing "\n" (or a blank in-progress line) parses to no token, so the value
// round-tripped back to a newline-free string and the caret collapsed — the user
// could never open a second line to enter a second control channel. The buffer is
// the source of truth while editing and only resyncs from props.value on a genuine
// out-of-band change (e.g. loading a different config), so newlines survive.
export function FreqListField(props: {
  label: string;
  value: number[] | null;
  onChange: (v: number[]) => void;
  help?: ReactNode;
}) {
  const [text, setText] = useState(() => (props.value ?? []).map(hzToDisplay).join("\n"));
  const emitted = useRef<number[]>(props.value ?? []);
  useEffect(() => {
    const v = props.value ?? [];
    if (!sameFreqList(v, emitted.current)) {
      emitted.current = v;
      setText(v.map(hzToDisplay).join("\n"));
    }
  }, [props.value]);
  const onText = (raw: string) => {
    setText(raw);
    const parsed = parseFreqList(raw);
    emitted.current = parsed;
    props.onChange(parsed);
  };
  return (
    <label className="block">
      <span className="label">{props.label}</span>
      <textarea
        className="input font-mono"
        rows={Math.max(2, text.split("\n").length + 1)}
        value={text}
        onChange={(e) => onText(e.target.value)}
      />
      <p className="help mt-1">
        One frequency per line. Accepts MHz (851.0375) or Hz (851037500).
        {props.help ? <> {props.help}</> : null}
      </p>
    </label>
  );
}

// sameFreqList reports whether two Hz lists are element-for-element equal, so
// FreqListField only rebuilds its text buffer on a genuine out-of-band change and
// not on the value it just emitted itself (which would clobber the caret).
function sameFreqList(a: number[], b: number[]): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false;
  }
  return true;
}

// HzField edits a single integer-Hz value as text, accepting MHz or Hz
// (the scalar analogue of FreqListField, reusing parseFreqList/hzToDisplay).
// It keeps a local text buffer so mid-edit values like "851." aren't
// reformatted on every keystroke, and resyncs when the bound value changes
// out-of-band (e.g. switching between items in a ListEditor).
export function HzField(props: {
  label: string;
  value: number;
  onChange: (hz: number) => void;
  placeholder?: string;
  help?: ReactNode;
}) {
  const [text, setText] = useState(props.value ? hzToDisplay(props.value) : "");
  const emitted = useRef(props.value);
  useEffect(() => {
    if (props.value !== emitted.current) {
      emitted.current = props.value;
      setText(props.value ? hzToDisplay(props.value) : "");
    }
  }, [props.value]);
  const onText = (raw: string) => {
    setText(raw);
    const hz = parseFreqList(raw)[0] ?? 0;
    emitted.current = hz;
    props.onChange(hz);
  };
  return (
    <label className="block">
      <span className="label">{props.label}</span>
      <input
        className="input font-mono"
        value={text}
        placeholder={props.placeholder ?? "MHz or Hz"}
        onChange={(e) => onText(e.target.value)}
      />
      {props.help ? <p className="help mt-1">{props.help}</p> : null}
    </label>
  );
}

// Fieldset is a styled <details> accordion for grouping related controls
// (band plans, encryption keys, remote sources) so large cards stay
// scannable. Collapsed by default unless defaultOpen.
export function Fieldset(props: {
  legend: string;
  defaultOpen?: boolean;
  children: ReactNode;
}) {
  return (
    <details open={props.defaultOpen} className="rounded-md border border-line">
      <summary className="cursor-pointer select-none px-3 py-2 text-sm font-medium">
        {props.legend}
      </summary>
      <div className="space-y-3 p-3 pt-0">{props.children}</div>
    </details>
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
