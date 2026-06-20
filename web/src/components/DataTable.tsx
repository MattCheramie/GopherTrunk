import { useEffect, useMemo, useState } from "react";
import { Input } from "./ui/Input";
import { EmptyState } from "./ui/EmptyState";
import { SkeletonRows } from "./ui/Skeleton";
import { prefs } from "../store/prefs";

export interface Column<T> {
  key: string;
  header: string;
  render: (row: T) => React.ReactNode;
  sort?: (a: T, b: T) => number;
  className?: string;
  headerClassName?: string;
  /** Set false to keep a column out of the column-visibility menu (it
   *  stays always-visible). Defaults to true. */
  hideable?: boolean;
}

interface Props<T> {
  rows: T[];
  columns: Column<T>[];
  rowKey: (row: T, index: number) => string;
  defaultSortKey?: string;
  defaultSortDirection?: "asc" | "desc";
  onRowClick?: (row: T, key: string) => void;
  emptyMessage?: React.ReactNode;
  // Optional second-row "expanded" content rendered beneath the row
  // (used by Events to show payload JSON inline).
  renderExpansion?: (row: T) => React.ReactNode;
  expandedKey?: string | null;

  // --- Phase 4 additions (all optional, backward compatible) ---

  /** Show a built-in search box that filters rows by `searchAccessor`. */
  searchable?: boolean;
  /** Returns the haystack string a row is matched against when searching. */
  searchAccessor?: (row: T) => string;
  searchPlaceholder?: string;
  /** Enable pagination at this page size. Omit to render every row. */
  pageSize?: number;
  /** Render skeleton rows instead of the empty message while true. */
  loading?: boolean;
  /** Trailing inline-action cell per row (clicks don't trigger onRowClick). */
  rowActions?: (row: T) => React.ReactNode;
  /** Extra controls rendered in the toolbar beside the search box. */
  toolbar?: React.ReactNode;
  /** Stable id enabling the column-visibility menu + its persistence. */
  tableId?: string;
}

export function DataTable<T>({
  rows,
  columns,
  rowKey,
  defaultSortKey,
  defaultSortDirection = "asc",
  onRowClick,
  emptyMessage = "No data.",
  renderExpansion,
  expandedKey,
  searchable,
  searchAccessor,
  searchPlaceholder = "Search…",
  pageSize,
  loading,
  rowActions,
  toolbar,
  tableId,
}: Props<T>) {
  const [sortKey, setSortKey] = useState<string | null>(
    defaultSortKey ?? null,
  );
  const [sortDir, setSortDir] = useState<"asc" | "desc">(
    defaultSortDirection,
  );
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(0);

  // Column visibility (only when a tableId opts in). Persisted per table.
  const showColumnMenu = !!tableId;
  const [hiddenCols, setHiddenCols] = useState<Set<string>>(() =>
    tableId ? new Set(prefs.tableHiddenColumns(tableId)) : new Set(),
  );

  function toggleColumn(key: string) {
    if (!tableId) return;
    setHiddenCols((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      prefs.setTableHiddenColumns(tableId, [...next]);
      return next;
    });
  }

  const visibleColumns = useMemo(
    () => columns.filter((c) => !hiddenCols.has(c.key)),
    [columns, hiddenCols],
  );

  const searched = useMemo(() => {
    if (!searchable || !searchAccessor || !query.trim()) return rows;
    const needle = query.toLowerCase();
    return rows.filter((r) => searchAccessor(r).toLowerCase().includes(needle));
  }, [rows, searchable, searchAccessor, query]);

  const sorted = useMemo(() => {
    if (!sortKey) return searched;
    const col = columns.find((c) => c.key === sortKey);
    if (!col?.sort) return searched;
    const copy = searched.slice();
    copy.sort(col.sort);
    if (sortDir === "desc") copy.reverse();
    return copy;
  }, [searched, columns, sortKey, sortDir]);

  const pageCount = pageSize ? Math.max(1, Math.ceil(sorted.length / pageSize)) : 1;
  // Keep the page index valid as the row set shrinks/grows.
  useEffect(() => {
    if (page > pageCount - 1) setPage(pageCount - 1);
  }, [page, pageCount]);
  // A new search resets to the first page.
  useEffect(() => {
    setPage(0);
  }, [query]);

  const pageRows = useMemo(() => {
    if (!pageSize) return sorted;
    const start = page * pageSize;
    return sorted.slice(start, start + pageSize);
  }, [sorted, pageSize, page]);

  function toggleSort(key: string) {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  }

  const hasToolbar = searchable || !!toolbar || showColumnMenu;
  const totalCols = visibleColumns.length + (rowActions ? 1 : 0);

  return (
    <div className="space-y-2">
      {hasToolbar && (
        <div className="flex items-center gap-2 flex-wrap">
          {searchable && (
            <Input
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={searchPlaceholder}
              aria-label="Search table"
              className="w-full sm:max-w-xs"
            />
          )}
          {toolbar}
          <div className="ml-auto flex items-center gap-2">
            {searchable && query && (
              <span className="text-xs text-muted">
                {sorted.length} of {rows.length}
              </span>
            )}
            {showColumnMenu && (
              <ColumnMenu
                columns={columns}
                hidden={hiddenCols}
                onToggle={toggleColumn}
              />
            )}
          </div>
        </div>
      )}

      <div className="panel overflow-hidden">
        {loading && rows.length === 0 ? (
          <div className="p-3">
            <SkeletonRows rows={pageSize ? Math.min(pageSize, 8) : 6} />
          </div>
        ) : sorted.length === 0 ? (
          <EmptyState message={emptyMessage} />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-panel/80 text-xs uppercase tracking-wider text-muted">
                <tr>
                  {visibleColumns.map((c) => {
                    const sortable = !!c.sort;
                    const active = sortKey === c.key;
                    return (
                      <th
                        key={c.key}
                        scope="col"
                        className={`px-3 py-2 text-left font-medium ${
                          c.headerClassName ?? ""
                        } ${sortable ? "cursor-pointer select-none" : ""}`}
                        onClick={sortable ? () => toggleSort(c.key) : undefined}
                        aria-sort={
                          !sortable
                            ? undefined
                            : active
                              ? sortDir === "asc"
                                ? "ascending"
                                : "descending"
                              : "none"
                        }
                      >
                        <span className="inline-flex items-center gap-1">
                          {c.header}
                          {sortable && (
                            <span
                              aria-hidden
                              className={active ? "text-accent" : "opacity-40"}
                            >
                              {active ? (sortDir === "asc" ? "▲" : "▼") : "⇅"}
                            </span>
                          )}
                        </span>
                      </th>
                    );
                  })}
                  {rowActions && (
                    <th scope="col" className="px-3 py-2 text-right font-medium">
                      <span className="sr-only">Actions</span>
                    </th>
                  )}
                </tr>
              </thead>
              <tbody className="divide-y divide-panel">
                {pageRows.map((row, i) => {
                  const key = rowKey(row, i);
                  const expanded = expandedKey === key;
                  return (
                    <Row
                      key={key}
                      row={row}
                      columns={visibleColumns}
                      colSpan={totalCols}
                      rowActions={rowActions}
                      onClick={
                        onRowClick ? () => onRowClick(row, key) : undefined
                      }
                      expansion={
                        renderExpansion && expanded ? renderExpansion(row) : null
                      }
                    />
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {pageSize && sorted.length > pageSize && (
        <div className="flex items-center justify-between gap-2 text-xs text-muted">
          <span>
            {page * pageSize + 1}–{Math.min((page + 1) * pageSize, sorted.length)}{" "}
            of {sorted.length}
          </span>
          <div className="flex items-center gap-2">
            <button
              className="btn-ghost !min-h-0 !py-1 text-xs"
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              disabled={page === 0}
            >
              ‹ Prev
            </button>
            <span>
              {page + 1} / {pageCount}
            </span>
            <button
              className="btn-ghost !min-h-0 !py-1 text-xs"
              onClick={() => setPage((p) => Math.min(pageCount - 1, p + 1))}
              disabled={page >= pageCount - 1}
            >
              Next ›
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function Row<T>({
  row,
  columns,
  colSpan,
  rowActions,
  onClick,
  expansion,
}: {
  row: T;
  columns: Column<T>[];
  colSpan: number;
  rowActions?: (row: T) => React.ReactNode;
  onClick?: () => void;
  expansion: React.ReactNode | null;
}) {
  return (
    <>
      <tr
        className={
          onClick
            ? "hover:bg-panel/40 cursor-pointer focus-within:bg-panel/40"
            : ""
        }
        onClick={onClick}
        tabIndex={onClick ? 0 : -1}
        onKeyDown={
          onClick
            ? (e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  onClick();
                }
              }
            : undefined
        }
      >
        {columns.map((c) => (
          <td key={c.key} className={`px-3 py-2 align-top ${c.className ?? ""}`}>
            {c.render(row)}
          </td>
        ))}
        {rowActions && (
          // Stop propagation so action clicks don't also fire the row's
          // onClick (e.g. opening the detail modal).
          <td
            className="px-3 py-2 align-top text-right whitespace-nowrap"
            onClick={(e) => e.stopPropagation()}
          >
            {rowActions(row)}
          </td>
        )}
      </tr>
      {expansion && (
        <tr>
          <td colSpan={colSpan} className="px-3 pb-3 pt-0 bg-panel/30 text-xs">
            {expansion}
          </td>
        </tr>
      )}
    </>
  );
}

// ColumnMenu is a native <details> popover listing the hideable columns
// as checkboxes. <details> gives us open/close + Escape + accessibility
// without click-away bookkeeping.
function ColumnMenu<T>({
  columns,
  hidden,
  onToggle,
}: {
  columns: Column<T>[];
  hidden: Set<string>;
  onToggle: (key: string) => void;
}) {
  const hideable = columns.filter((c) => c.hideable !== false);
  if (hideable.length === 0) return null;
  return (
    <details className="relative">
      <summary className="btn-ghost !min-h-0 !py-1 text-xs cursor-pointer list-none">
        Columns
      </summary>
      <div className="absolute right-0 z-30 mt-1 w-48 panel bg-bg p-2 space-y-1 shadow-lg">
        {hideable.map((c) => (
          <label
            key={c.key}
            className="flex items-center gap-2 text-sm px-1 py-0.5"
          >
            <input
              type="checkbox"
              className="h-4 w-4"
              checked={!hidden.has(c.key)}
              onChange={() => onToggle(c.key)}
            />
            <span>{c.header}</span>
          </label>
        ))}
      </div>
    </details>
  );
}
