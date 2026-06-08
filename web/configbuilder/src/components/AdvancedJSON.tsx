import { useEffect, useRef, useState, type ReactNode } from "react";

// AdvancedJSON is a collapsible per-object JSON editor for the long tail of
// fields that don't warrant a bespoke widget (protocol decoder knobs,
// device tuning flags). It generalizes the section-level Generic editor to
// a single object value: invalid JSON is held locally (shown in red) and
// only merged back into `value` when it parses, so a typo never corrupts
// the draft.
//
// When `pick` is given, only those keys are surfaced and merged — the
// bespoke fields above own the rest, and unmodeled keys on `value` survive
// untouched.
export function AdvancedJSON<T extends object>(props: {
  value: T;
  onChange: (next: T) => void;
  label?: string;
  pick?: (keyof T)[];
  help?: ReactNode;
}) {
  const projected = (): Partial<T> => {
    if (!props.pick) return props.value;
    const out: Partial<T> = {};
    for (const k of props.pick) {
      if (props.value[k] !== undefined) out[k] = props.value[k];
    }
    return out;
  };

  const [text, setText] = useState(() => JSON.stringify(projected(), null, 2));
  const [err, setErr] = useState<string | null>(null);
  // Track the value we last emitted so an external change (switching items)
  // re-seeds the textarea, but our own edits don't clobber the buffer.
  const emitted = useRef(props.value);

  useEffect(() => {
    if (props.value !== emitted.current) {
      emitted.current = props.value;
      setText(JSON.stringify(projected(), null, 2));
      setErr(null);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.value]);

  const apply = (next: string) => {
    setText(next);
    let parsed: unknown;
    try {
      parsed = JSON.parse(next);
    } catch (e) {
      setErr((e as Error).message);
      return;
    }
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
      setErr("expected a JSON object");
      return;
    }
    setErr(null);
    const merged = { ...props.value, ...(parsed as Partial<T>) } as T;
    emitted.current = merged;
    props.onChange(merged);
  };

  return (
    <details className="rounded-md border border-white/10">
      <summary className="cursor-pointer select-none px-3 py-2 text-sm font-medium">
        {props.label ?? "Advanced (JSON)"}
      </summary>
      <div className="space-y-2 p-3 pt-0">
        {props.help ? <p className="help">{props.help}</p> : null}
        <textarea
          className="input font-mono"
          rows={Math.max(4, text.split("\n").length)}
          value={text}
          onChange={(e) => apply(e.target.value)}
          spellCheck={false}
        />
        {err ? <p className="text-xs text-err">Invalid JSON: {err}</p> : null}
      </div>
    </details>
  );
}
