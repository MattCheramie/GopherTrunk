import { useMemo, useRef, useState } from "react";
import {
  bookmarks as bookmarksAPI,
  type Bookmark,
  type BookmarkInput,
} from "../api/bookmarks";
import { Column, DataTable } from "../components/DataTable";
import { PageHeader } from "../components/ui/PageHeader";
import { Card } from "../components/ui/Card";
import { Field } from "../components/ui/Field";
import { Input } from "../components/ui/Input";
import { Select } from "../components/ui/Select";
import { Button } from "../components/ui/Button";
import { StaleIndicator } from "../components/ui/StaleIndicator";
import { useDataPoll } from "../hooks/useDataPoll";
import { selectClientConfig, useShared } from "../store/shared";

// Bookmarks panel — operator-managed conventional channel list backed
// by the SQLite bookmarks table on the daemon. An inline create/edit
// form with per-field validation, plus a searchable/sortable table with
// per-row edit/delete actions.

const EMPTY_DRAFT: BookmarkInput = {
  name: "",
  freq_hz: 0,
  mode: "FM",
  ctcss_hz: 0,
  dcs_code: 0,
  notes: "",
  group: "",
};

type Draft = BookmarkInput & { id?: number };

const MODES = ["FM", "NFM", "AM", "USB", "LSB", "CW", "DMR", "P25"];

export function Bookmarks() {
  const cfg = useShared(selectClientConfig);
  const notify = useShared((s) => s.notify);
  const [rows, setRows] = useState<Bookmark[]>([]);
  const [draft, setDraft] = useState<Draft>(EMPTY_DRAFT);
  const [error, setError] = useState<string | null>(null);
  const [fieldErr, setFieldErr] = useState<{ name?: string; freq?: string }>({});
  const [busy, setBusy] = useState(false);

  const nameRef = useRef<HTMLInputElement>(null);
  const freqRef = useRef<HTMLInputElement>(null);

  const { loading, stale, lastUpdated, refresh } = useDataPoll({
    // Bookmarks rarely change — a long poll catches peer edits without
    // SSE wiring; mutations call refresh() for an immediate update.
    fetcher: () => bookmarksAPI.list(cfg),
    onData: setRows,
    intervalMs: 30_000,
    resetKey: cfg.baseURL,
  });

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    const errs: { name?: string; freq?: string } = {};
    if (!draft.name.trim()) errs.name = "Name is required.";
    if (!(draft.freq_hz > 0)) errs.freq = "Enter a frequency in Hz.";
    setFieldErr(errs);
    if (errs.name) {
      nameRef.current?.focus();
      return;
    }
    if (errs.freq) {
      freqRef.current?.focus();
      return;
    }

    setBusy(true);
    try {
      const payload: BookmarkInput = {
        name: draft.name.trim(),
        freq_hz: draft.freq_hz,
        mode: draft.mode || "FM",
        ctcss_hz: draft.ctcss_hz || 0,
        dcs_code: draft.dcs_code || 0,
        notes: draft.notes ?? "",
        group: draft.group ?? "",
      };
      if (draft.id) {
        await bookmarksAPI.update(cfg, draft.id, payload);
        notify("success", `Saved "${payload.name}"`);
      } else {
        await bookmarksAPI.create(cfg, payload);
        notify("success", `Added "${payload.name}"`);
      }
      setDraft(EMPTY_DRAFT);
      refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const remove = async (b: Bookmark) => {
    setBusy(true);
    setError(null);
    try {
      await bookmarksAPI.remove(cfg, b.id);
      if (draft.id === b.id) setDraft(EMPTY_DRAFT);
      notify("success", `Deleted "${b.name}"`);
      refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const startEdit = (b: Bookmark) => {
    setFieldErr({});
    setDraft({
      id: b.id,
      name: b.name,
      freq_hz: b.freq_hz,
      mode: b.mode,
      ctcss_hz: b.ctcss_hz ?? 0,
      dcs_code: b.dcs_code ?? 0,
      notes: b.notes ?? "",
      group: b.group ?? "",
    });
  };

  const columns: Column<Bookmark>[] = useMemo(
    () => [
      {
        key: "name",
        header: "Name",
        render: (b) => <span className="font-medium">{b.name}</span>,
        sort: (a, b) => a.name.localeCompare(b.name),
      },
      {
        key: "freq",
        header: "Freq (MHz)",
        className: "text-right font-mono text-accent",
        headerClassName: "text-right",
        render: (b) => (b.freq_hz / 1e6).toFixed(4),
        sort: (a, b) => a.freq_hz - b.freq_hz,
      },
      {
        key: "mode",
        header: "Mode",
        render: (b) => b.mode,
        sort: (a, b) => a.mode.localeCompare(b.mode),
      },
      {
        key: "group",
        header: "Group",
        render: (b) => b.group || <span className="text-muted">—</span>,
        sort: (a, b) => (a.group ?? "").localeCompare(b.group ?? ""),
      },
      {
        key: "notes",
        header: "Notes",
        className: "hidden md:table-cell text-muted",
        headerClassName: "hidden md:table-cell",
        render: (b) => b.notes || "",
      },
    ],
    [],
  );

  return (
    <div className="space-y-4">
      <PageHeader
        title="Bookmarks"
        actions={
          <>
            <StaleIndicator stale={stale} lastUpdated={lastUpdated} />
            <span className="text-xs text-muted">
              {rows.length} channel{rows.length === 1 ? "" : "s"}
            </span>
          </>
        }
      />

      {error && (
        <div
          role="alert"
          className="rounded-md border border-err/40 bg-err/15 px-3 py-2 text-sm text-err"
        >
          {error}
        </div>
      )}

      <Card title={draft.id ? "Edit bookmark" : "New bookmark"}>
        <form onSubmit={submit} className="space-y-3" noValidate>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            <Field label="Name" error={fieldErr.name}>
              {(p) => (
                <Input
                  {...p}
                  ref={nameRef}
                  className="w-full"
                  value={draft.name}
                  onChange={(e) => setDraft({ ...draft, name: e.target.value })}
                  placeholder="Marine Ch 16"
                />
              )}
            </Field>
            <Field label="Frequency (Hz)" error={fieldErr.freq}>
              {(p) => (
                <Input
                  {...p}
                  ref={freqRef}
                  type="number"
                  className="w-full font-mono"
                  value={draft.freq_hz || ""}
                  onChange={(e) =>
                    setDraft({ ...draft, freq_hz: Number(e.target.value) })
                  }
                  placeholder="156800000"
                />
              )}
            </Field>
            <Field label="Mode">
              {(p) => (
                <Select
                  {...p}
                  className="w-full"
                  value={draft.mode}
                  onChange={(e) => setDraft({ ...draft, mode: e.target.value })}
                >
                  {MODES.map((m) => (
                    <option key={m} value={m}>
                      {m}
                    </option>
                  ))}
                </Select>
              )}
            </Field>
            <Field label="Group">
              {(p) => (
                <Input
                  {...p}
                  className="w-full"
                  value={draft.group}
                  onChange={(e) => setDraft({ ...draft, group: e.target.value })}
                  placeholder="marine / ham-2m / utility"
                />
              )}
            </Field>
            <Field label="Notes" className="sm:col-span-2 lg:col-span-4">
              {(p) => (
                <Input
                  {...p}
                  className="w-full"
                  value={draft.notes ?? ""}
                  onChange={(e) => setDraft({ ...draft, notes: e.target.value })}
                  placeholder="International distress (NMEA)"
                />
              )}
            </Field>
          </div>
          <div className="flex gap-2">
            <Button type="submit" loading={busy}>
              {draft.id ? "Save edits" : "Add bookmark"}
            </Button>
            {draft.id && (
              <Button
                type="button"
                variant="ghost"
                onClick={() => {
                  setDraft(EMPTY_DRAFT);
                  setFieldErr({});
                }}
              >
                Cancel
              </Button>
            )}
          </div>
        </form>
      </Card>

      <DataTable
        rows={rows}
        columns={columns}
        rowKey={(b) => String(b.id)}
        defaultSortKey="group"
        tableId="bookmarks"
        loading={loading}
        searchable
        searchAccessor={(b) =>
          [b.name, b.mode, b.group, b.notes, (b.freq_hz / 1e6).toFixed(4)]
            .filter(Boolean)
            .join(" ")
        }
        searchPlaceholder="Search by name, group, mode…"
        rowActions={(b) => (
          <span className="inline-flex gap-2">
            <button className="text-xs underline" onClick={() => startEdit(b)}>
              edit
            </button>
            <button
              className="text-xs underline text-err"
              onClick={() => remove(b)}
              disabled={busy}
            >
              delete
            </button>
          </span>
        )}
        emptyMessage="No bookmarks yet. Add one above — typical use is to bookmark marine VHF Ch 16, NOAA weather channels, FRS/GMRS, repeater outputs, and the local public-safety conventional fall-backs."
      />
    </div>
  );
}
