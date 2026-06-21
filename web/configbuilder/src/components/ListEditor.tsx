import { useState, type ReactNode } from "react";

// ListEditor is a generic add/remove editor for arrays of sub-objects
// (trunking systems, SDR devices, channel/feed lists, band-plan entries,
// encryption keys, …). It is pure — no store coupling — so it composes
// anywhere and is trivially unit-testable. Each item renders in a bordered
// card with a header title and a Remove button; `set` replaces just that
// item immutably.
export function ListEditor<T>(props: {
  label: string;
  items: T[] | null | undefined;
  onChange: (next: T[]) => void;
  makeNew: () => T;
  renderItem: (item: T, set: (next: T) => void, index: number) => ReactNode;
  itemTitle?: (item: T, index: number) => string;
  addLabel?: string;
  emptyHint?: ReactNode;
}) {
  const items = props.items ?? [];
  // Index awaiting a remove confirmation (two-click guard).
  const [confirm, setConfirm] = useState<number | null>(null);

  const setItem = (i: number, next: T) => {
    const copy = items.slice();
    copy[i] = next;
    props.onChange(copy);
  };
  const remove = (i: number) => {
    setConfirm(null);
    props.onChange(items.filter((_, k) => k !== i));
  };
  const add = () => props.onChange([...items, props.makeNew()]);
  const duplicate = (i: number) => {
    const copy = items.slice();
    copy.splice(i + 1, 0, structuredClone(items[i]));
    props.onChange(copy);
  };
  const move = (i: number, delta: number) => {
    const j = i + delta;
    if (j < 0 || j >= items.length) return;
    const copy = items.slice();
    [copy[i], copy[j]] = [copy[j], copy[i]];
    props.onChange(copy);
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <span className="label mb-0">
          {props.label} ({items.length})
        </span>
        <button className="btn-ghost" onClick={add}>
          {props.addLabel ?? "+ Add"}
        </button>
      </div>
      {items.length === 0 && props.emptyHint ? (
        <p className="help">{props.emptyHint}</p>
      ) : null}
      {items.map((item, i) => (
        <div key={i} className="rounded-md border border-line p-3 space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">
              {props.itemTitle ? props.itemTitle(item, i) : `Item ${i + 1}`}
            </span>
            {confirm === i ? (
              <span className="flex items-center gap-1">
                <button className="btn-danger" onClick={() => remove(i)}>
                  Confirm remove
                </button>
                <button className="btn-ghost" onClick={() => setConfirm(null)}>
                  Cancel
                </button>
              </span>
            ) : (
              <span className="flex items-center gap-1">
                <button
                  className="btn-ghost"
                  title="Move up"
                  disabled={i === 0}
                  onClick={() => move(i, -1)}
                >
                  ↑
                </button>
                <button
                  className="btn-ghost"
                  title="Move down"
                  disabled={i === items.length - 1}
                  onClick={() => move(i, 1)}
                >
                  ↓
                </button>
                <button className="btn-ghost" title="Duplicate" onClick={() => duplicate(i)}>
                  Duplicate
                </button>
                <button className="btn-danger" onClick={() => setConfirm(i)}>
                  Remove
                </button>
              </span>
            )}
          </div>
          {props.renderItem(item, (next) => setItem(i, next), i)}
        </div>
      ))}
    </div>
  );
}
