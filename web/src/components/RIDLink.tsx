import { Link } from "react-router-dom";

// RIDLink renders an inline link to the per-radio detail page
// (/rids/{id}). Shared by the CC Activity feed, the Active-calls list, and
// Call history so a source radio ID is always a click into that radio's
// history — the cross-selection every trunking UI is expected to support.
//
// Styled like the surrounding mono text, underlined on hover so it reads as
// actionable. Stops click propagation so it works inside a row that is
// itself clickable (opens a detail modal). Renders the resolved alias when
// one is supplied, otherwise the bare numeric id.
export function RIDLink({
  rid,
  alias,
  className = "font-mono text-accent hover:underline",
}: {
  rid: number;
  alias?: string | null;
  className?: string;
}) {
  return (
    <Link to={`/rids/${rid}`} className={className} onClick={(e) => e.stopPropagation()}>
      {alias || rid}
    </Link>
  );
}
