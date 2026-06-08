import type { ReactNode } from "react";
import { useStore } from "../store/shared";

// Section wraps a config section editor with its title, instructions, a
// "Docs ↗" link (opens in a new tab), and a live validation badge driven
// by the section's entry in the whole-config validation result.
//
// Instructions come from the shared docs registry (Go configbuilder.Sections,
// served over /config/docs) so the web and terminal builders show identical
// section help from one source. The optional `instructions` prop is only a
// fallback for before the registry has loaded.
export function Section(props: {
  sectionKey: string;
  title: string;
  instructions?: ReactNode;
  children: ReactNode;
}) {
  const docs = useStore((s) => s.docs[props.sectionKey]);
  const validation = useStore((s) => s.validation);
  const errs = (validation?.errors ?? []).filter((e) => e.section === props.sectionKey);
  const instructions = docs?.instructions ?? props.instructions;

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold">{props.title}</h2>
          {instructions ? <p className="help mt-1 max-w-2xl">{instructions}</p> : null}
        </div>
        {docs ? (
          <a
            className="btn-ghost whitespace-nowrap"
            href={docs.url}
            target="_blank"
            rel="noreferrer"
            title={docs.description}
          >
            Docs ↗
          </a>
        ) : null}
      </div>

      {errs.length > 0 ? (
        <div className="space-y-1 rounded-md border border-err/40 bg-err/10 px-3 py-2 text-sm text-err">
          {errs.map((e, i) => (
            <div key={i}>✗ {e.message}</div>
          ))}
        </div>
      ) : validation ? (
        <div className="rounded-md border border-ok/30 bg-ok/10 px-3 py-2 text-sm text-ok">
          ✓ This section validates.
        </div>
      ) : null}

      <div className="card space-y-4">{props.children}</div>
    </div>
  );
}
