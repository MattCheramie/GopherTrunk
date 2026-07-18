import type { CaptureDTO } from "../api/types";

// CaptureRow is one entry in the captures list. It is a pure presentational
// component (no store/router coupling) so it can be unit-tested directly, and
// so the layout fixes below are self-contained:
//
//   - The name button is `min-w-0` so its `truncate` child can actually shrink;
//     without it a long staged name (e.g. "capture-AIRSPY SN:…-467.913MHz")
//     forces the flex row wider than the card and pushes the cmp checkbox and
//     the delete button out of view (and out of reach).
//   - The cmp label, the download link, and the delete button are `shrink-0`
//     so they always stay inside the card and remain clickable.
//   - A persistent per-row Download link (⭳) exports the staged .cfile via the
//     backend download endpoint — the recorded capture is now always saveable
//     straight from the list, not only from a transient link on the last grab.
export function CaptureRow({
  capture,
  selected,
  inCompare,
  downloadURL,
  onSelect,
  onToggleCompare,
  onRemove,
}: {
  capture: CaptureDTO;
  selected: boolean;
  inCompare: boolean;
  downloadURL: string;
  onSelect: () => void;
  onToggleCompare: () => void;
  onRemove: () => void;
}) {
  return (
    <li
      className={`flex items-center gap-2 rounded-md px-2 py-1.5 text-sm ${
        selected ? "bg-accent/15" : "hover:bg-panel"
      }`}
    >
      <button className="min-w-0 flex-1 text-left" onClick={onSelect}>
        <div className="truncate" title={capture.name}>
          {capture.name}
        </div>
        <div className="text-xs text-muted">
          {capture.format} ·{" "}
          {capture.sample_rate_hz ? `${capture.sample_rate_hz} Hz` : "rate?"} ·{" "}
          {(capture.size / 1024).toFixed(0)} KiB
        </div>
      </button>
      <label className="flex shrink-0 items-center gap-1 text-xs text-muted">
        <input type="checkbox" checked={inCompare} onChange={onToggleCompare} />
        cmp
      </label>
      <a
        className="shrink-0 text-xs text-muted hover:text-fg"
        href={downloadURL}
        download
        title="Download capture"
        aria-label="Download capture"
        onClick={(e) => e.stopPropagation()}
      >
        ⭳
      </a>
      <button
        className="shrink-0 text-xs text-err/80 hover:text-err"
        onClick={onRemove}
        title="Remove capture"
        aria-label="Remove capture"
      >
        ✕
      </button>
    </li>
  );
}
