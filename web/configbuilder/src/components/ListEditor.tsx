import type { ReactNode } from "react";

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

  const setItem = (i: number, next: T) => {
    const copy = items.slice();
    copy[i] = next;
    props.onChange(copy);
  };
  const remove = (i: number) => props.onChange(items.filter((_, k) => k !== i));
  const add = () => props.onChange([...items, props.makeNew()]);

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
        <div key={i} className="rounded-md border border-white/10 p-3 space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">
              {props.itemTitle ? props.itemTitle(item, i) : `Item ${i + 1}`}
            </span>
            <button className="btn-danger" onClick={() => remove(i)}>
              Remove
            </button>
          </div>
          {props.renderItem(item, (next) => setItem(i, next), i)}
        </div>
      ))}
    </div>
  );
}
