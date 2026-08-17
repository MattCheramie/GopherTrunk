import { useEffect, useState } from "react";

interface Props {
  label: string;
  value: string;
  onCommit: (value: string) => void;
  disabled?: boolean;
  placeholder?: string;
}

// NameField is a text input that commits on blur or Enter rather than on
// every keystroke — a PATCH per character would hammer the daemon and race
// itself, and the mutation endpoints are not debounced server-side.
//
// Escape reverts to the last committed value, so a mistyped name can be
// abandoned without saving it first.
export function NameField({
  label,
  value,
  onCommit,
  disabled,
  placeholder,
}: Props) {
  const [draft, setDraft] = useState(value);

  // Re-sync when the record changes underneath (another console renamed it,
  // or the operator selected a different row).
  useEffect(() => {
    setDraft(value);
  }, [value]);

  function commit() {
    const next = draft.trim();
    if (next === value) return;
    onCommit(next);
  }

  return (
    <label className="flex items-center gap-3 text-sm">
      <span className="w-24 shrink-0">{label}</span>
      <input
        type="text"
        className="input flex-1"
        value={draft}
        disabled={disabled}
        placeholder={placeholder}
        aria-label={label}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            commit();
          } else if (e.key === "Escape") {
            setDraft(value);
            (e.target as HTMLInputElement).blur();
          }
        }}
      />
    </label>
  );
}
