import { useEffect, useState } from "react";
import { api, HTTPError } from "../api/client";
import { writes } from "../api/write";
import type { ImportPreview, ImportResult, RuntimeDTO } from "../api/types";
import {
  selectCanMutate,
  selectClientConfig,
  useShared,
} from "../store/shared";

// Import mirrors the TUI's live-import panel. Three views:
//
//  - Stage     drop / pick files; preview after upload
//  - Preview   parsed systems table; commit or discard
//  - Result    systems_added / systems_replaced / csv_paths
//
// The endpoint returns 503 when the daemon was started without
// -config — in that case we render a read-only banner pointing the
// operator at the docs.
export function Import() {
  const cfg = useShared(selectClientConfig);
  const canMutate = useShared(selectCanMutate);
  const setError = useShared((s) => s.setError);

  const [files, setFiles] = useState<File[]>([]);
  const [preview, setPreview] = useState<ImportPreview | null>(null);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [runtime, setRuntime] = useState<RuntimeDTO | null>(null);

  useEffect(() => {
    let cancel = false;
    api.runtime(cfg)
      .then((r) => { if (!cancel) setRuntime(r); })
      .catch(() => {});
    return () => { cancel = true; };
  }, [cfg]);

  const hasConfig = !!runtime?.config_path;

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const list = e.target.files;
    if (!list) return;
    setFiles(Array.from(list));
    setPreview(null);
    setResult(null);
  }

  async function upload() {
    if (files.length === 0) return;
    setBusy(true);
    setError(null);
    try {
      const p = await writes.importUpload(cfg, files);
      setPreview(p);
    } catch (e) {
      if (e instanceof HTTPError) {
        setError(`Import upload failed: ${e.message}`);
      } else {
        setError(String(e));
      }
    } finally {
      setBusy(false);
    }
  }

  async function commit() {
    if (!preview) return;
    setBusy(true);
    setError(null);
    try {
      const r = await writes.importCommit(cfg, preview.id, false);
      setResult(r);
      setPreview(null);
    } catch (e) {
      if (e instanceof HTTPError) {
        setError(`Import commit failed: ${e.message}`);
      } else {
        setError(String(e));
      }
    } finally {
      setBusy(false);
    }
  }

  async function discard() {
    if (!preview) return;
    setBusy(true);
    try {
      await writes.importDiscard(cfg, preview.id);
    } catch {
      // Discard is a courtesy — failure is fine; the TTL sweeper
      // will drop the entry eventually.
    } finally {
      setPreview(null);
      setBusy(false);
    }
  }

  function reset() {
    setFiles([]);
    setPreview(null);
    setResult(null);
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-semibold">Import systems / talkgroups</h1>

      {!hasConfig && (
        <div className="panel bg-warn/15 border-warn/40 text-warn p-3 text-sm">
          The daemon is running without a <code>-config</code> file, so the
          import endpoint returns 503. Restart the daemon with
          <code> -config /path/to/config.yaml</code> to enable in-process
          imports.
        </div>
      )}

      {!canMutate && (
        <div className="panel bg-warn/15 border-warn/40 text-warn p-3 text-sm">
          Mutations are disabled on this daemon. Pass the token / configure
          auth so this client can write.
        </div>
      )}

      {result ? (
        <ResultView result={result} onReset={reset} />
      ) : preview ? (
        <PreviewView
          preview={preview}
          busy={busy}
          onCommit={commit}
          onDiscard={discard}
        />
      ) : (
        <StageView
          files={files}
          busy={busy}
          disabled={!hasConfig || !canMutate}
          onFileChange={handleFileChange}
          onUpload={upload}
          onClear={() => setFiles([])}
        />
      )}
    </div>
  );
}

function StageView({
  files,
  busy,
  disabled,
  onFileChange,
  onUpload,
  onClear,
}: {
  files: File[];
  busy: boolean;
  disabled: boolean;
  onFileChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onUpload: () => void;
  onClear: () => void;
}) {
  return (
    <div className="panel p-4 space-y-3">
      <p className="text-sm text-muted">
        Drop one or more <code>.pdf</code> (RadioReference) or{" "}
        <code>.csv</code> (multi-section bundle) files. The daemon parses them
        in memory and shows you a preview before merging into{" "}
        <code>config.yaml</code>.
      </p>
      <input
        type="file"
        multiple
        accept=".pdf,.csv"
        onChange={onFileChange}
        disabled={disabled || busy}
        className="text-sm"
      />
      {files.length > 0 && (
        <ul className="text-sm space-y-1">
          {files.map((f, i) => (
            <li key={i} className="text-muted">
              {f.name}{" "}
              <span className="text-xs opacity-70">
                ({Math.ceil(f.size / 1024)} KiB)
              </span>
            </li>
          ))}
        </ul>
      )}
      <div className="flex gap-2">
        <button
          disabled={disabled || busy || files.length === 0}
          onClick={onUpload}
          className="btn btn-primary disabled:opacity-50"
        >
          {busy ? "Uploading…" : `Upload ${files.length || ""} file(s)`}
        </button>
        <button
          disabled={busy || files.length === 0}
          onClick={onClear}
          className="btn disabled:opacity-50"
        >
          Clear
        </button>
      </div>
    </div>
  );
}

function PreviewView({
  preview,
  busy,
  onCommit,
  onDiscard,
}: {
  preview: ImportPreview;
  busy: boolean;
  onCommit: () => void;
  onDiscard: () => void;
}) {
  return (
    <div className="panel p-4 space-y-3">
      <h2 className="font-semibold">Parsed systems</h2>
      <table className="text-sm w-full">
        <thead>
          <tr className="text-muted text-left">
            <th className="py-1">Name</th>
            <th className="py-1">Protocol</th>
            <th className="py-1">Sites</th>
            <th className="py-1">Talkgroups</th>
            <th className="py-1">Location</th>
          </tr>
        </thead>
        <tbody>
          {preview.systems.map((s, i) => (
            <tr key={i} className="border-t border-panel">
              <td className="py-1 font-mono">{s.name}</td>
              <td className="py-1">{s.protocol}</td>
              <td className="py-1">{s.site_count}</td>
              <td className="py-1">{s.talkgroup_count}</td>
              <td className="py-1 text-muted">{s.location || "—"}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p className="text-xs text-muted">staging id: {preview.id}</p>
      <div className="flex gap-2">
        <button
          disabled={busy}
          onClick={onCommit}
          className="btn btn-primary disabled:opacity-50"
        >
          {busy ? "Committing…" : "Commit to config.yaml"}
        </button>
        <button
          disabled={busy}
          onClick={onDiscard}
          className="btn disabled:opacity-50"
        >
          Discard
        </button>
      </div>
    </div>
  );
}

function ResultView({
  result,
  onReset,
}: {
  result: ImportResult;
  onReset: () => void;
}) {
  return (
    <div className="panel p-4 space-y-3">
      <h2 className="font-semibold">Import committed</h2>
      {result.systems_added && result.systems_added.length > 0 && (
        <p className="text-sm">
          <span className="text-ok">Added:</span>{" "}
          {result.systems_added.join(", ")}
        </p>
      )}
      {result.systems_replaced && result.systems_replaced.length > 0 && (
        <p className="text-sm">
          <span className="text-warn">Replaced:</span>{" "}
          {result.systems_replaced.join(", ")}
        </p>
      )}
      {result.csv_paths && result.csv_paths.length > 0 && (
        <div className="text-xs text-muted">
          <p>Talkgroup CSVs:</p>
          <ul className="space-y-1 pl-4">
            {result.csv_paths.map((p, i) => (
              <li key={i}>{p}</li>
            ))}
          </ul>
        </div>
      )}
      {result.config_path && (
        <p className="text-xs text-muted">config.yaml @ {result.config_path}</p>
      )}
      <button onClick={onReset} className="btn">
        Import more
      </button>
    </div>
  );
}
