import { useEffect, useMemo, useState } from "react";
import { api } from "../api/client";
import { useStore } from "../store/shared";
import type { ConfigFileInfo } from "../api/types";

// BrowseConfigs is a prominent, full file browser for the config files
// discovered in the server's allowed directories — available and
// previously-created configs alike. It groups files by directory and shows
// name, last-modified time, size, and a valid/⚠ badge so an operator can find
// and open the right one at a glance. Opening a file delegates to the store's
// load(); the lightweight "Open ▾" dropdown in the FileBar stays for quick
// switches.
export function BrowseConfigs(props: { onClose: () => void }) {
  const load = useStore((s) => s.load);
  const setError = useStore((s) => s.setError);
  const current = useStore((s) => s.path);
  const busy = useStore((s) => s.busy);

  const [files, setFiles] = useState<ConfigFileInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState("");

  useEffect(() => {
    let cancel = false;
    (async () => {
      try {
        const r = await api.listFiles();
        if (!cancel) setFiles(r.files ?? []);
      } catch (e) {
        if (!cancel) setError(`List failed: ${(e as Error).message}`);
      } finally {
        if (!cancel) setLoading(false);
      }
    })();
    return () => {
      cancel = true;
    };
  }, [setError]);

  const f = filter.trim().toLowerCase();
  // Group the (filtered) files by their containing directory, preserving the
  // discovery order the server returned.
  const groups = useMemo(() => {
    const out: { dir: string; files: ConfigFileInfo[] }[] = [];
    for (const file of files) {
      if (f && !file.name.toLowerCase().includes(f) && !file.dir.toLowerCase().includes(f)) {
        continue;
      }
      let g = out.find((x) => x.dir === file.dir);
      if (!g) {
        g = { dir: file.dir, files: [] };
        out.push(g);
      }
      g.files.push(file);
    }
    return out;
  }, [files, f]);

  const openFile = async (path: string) => {
    props.onClose();
    await load(path);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center bg-black/40 p-4 overflow-y-auto">
      <div className="card w-full max-w-2xl space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-base font-semibold">Browse config files</h3>
          <button className="btn-ghost" onClick={props.onClose}>
            Close
          </button>
        </div>

        <input
          className="input"
          placeholder="filter by file name or folder"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        />

        {loading ? (
          <p className="help">Loading…</p>
        ) : groups.length === 0 ? (
          <p className="help">
            {files.length === 0
              ? "No config files found in the discovery directories. Use “New” to start one."
              : "No files match the filter."}
          </p>
        ) : (
          <div className="max-h-[60vh] space-y-4 overflow-y-auto">
            {groups.map((g) => (
              <div key={g.dir} className="space-y-1">
                <div className="help truncate font-mono text-xs">{g.dir}</div>
                {g.files.map((file) => (
                  <button
                    key={file.path}
                    disabled={busy}
                    onClick={() => openFile(file.path)}
                    title={file.path}
                    className={`flex w-full items-center justify-between gap-3 rounded border px-3 py-2 text-left text-sm hover:bg-panel ${
                      file.path === current ? "border-accent/50 bg-accent/10" : "border-line"
                    }`}
                  >
                    <span className="min-w-0">
                      <span className={file.valid ? "" : "text-warn"}>
                        {file.valid ? "" : "⚠ "}
                        {file.name}
                      </span>
                      {file.path === current ? (
                        <span className="ml-2 text-xs text-accent">current</span>
                      ) : null}
                      <span className="help block truncate">
                        {fmtModified(file.modified)} · {fmtSize(file.size)}
                        {file.valid ? "" : " · invalid"}
                      </span>
                    </span>
                    <span className="shrink-0 text-xs text-muted">Open</span>
                  </button>
                ))}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// fmtModified renders the RFC3339 timestamp as a compact local date-time,
// falling back to the raw string if it can't be parsed.
function fmtModified(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso || "—";
  return d.toLocaleString();
}

// fmtSize renders a byte count in human units.
function fmtSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return "—";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
