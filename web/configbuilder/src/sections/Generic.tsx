import { useEffect, useState } from "react";
import { Section } from "../components/Section";
import { useStore } from "../store/shared";
import type { GTConfig } from "../api/types";

// GenericSection edits an advanced config section that has no bespoke form
// yet, as JSON. It still carries the section's instructions, docs link, and
// the live server-side validator badge. Invalid JSON is held locally and
// not pushed into the draft until it parses.
export function GenericSection(props: {
  sectionKey: keyof GTConfig;
  title: string;
  instructions?: string;
}) {
  const value = useStore((s) => (s.config ? s.config[props.sectionKey] : null));
  const patch = useStore((s) => s.patchSection);

  const [text, setText] = useState("");
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    setText(JSON.stringify(value ?? null, null, 2));
    setErr(null);
  }, [props.sectionKey]); // reset when switching sections

  const apply = (next: string) => {
    setText(next);
    try {
      const parsed = JSON.parse(next);
      setErr(null);
      patch(props.sectionKey, parsed);
    } catch (e) {
      setErr((e as Error).message);
    }
  };

  return (
    <Section sectionKey={String(props.sectionKey)} title={props.title} instructions={props.instructions}>
      <p className="help">
        Advanced section — edit as JSON. Changes apply to the draft as soon as
        the JSON is valid, and are validated by the server like every other
        section.
      </p>
      <textarea
        className="input font-mono"
        rows={16}
        value={text}
        onChange={(e) => apply(e.target.value)}
        spellCheck={false}
      />
      {err ? <p className="text-xs text-err">Invalid JSON: {err}</p> : null}
    </Section>
  );
}
