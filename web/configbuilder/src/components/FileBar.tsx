import { useEffect, useState } from "react";
import { api } from "../api/client";
import { useStore } from "../store/shared";
import { downloadName, joinPath, pathCollides } from "../lib/fileName";
import type { ConfigFileInfo } from "../api/types";

export function FileBar() {
  const path = useStore((s) => s.path);
  const dirty = useStore((s) => s.dirty);
  const load = useStore((s) => s.load);
  const newConfig = useStore((s) => s.newConfig);
  const save = useStore((s) => s.save);
  const revert = useStore((s) => s.revert);
  const validateAll = useStore((s) => s.validateAll);
  const setError = useStore((s) => s.setError);
  const busy = useStore((s) => s.busy);

  const [files, setFiles] = useState<ConfigFileInfo[]>([]);
  const [dirs, setDirs] = useState<string[]>([]);
  const [openMenu, setOpenMenu] = useState(false);
  const [saveOpen, setSaveOpen] = useState(false);

  const refresh = async () => {
    try {
      const r = await api.listFiles();
      setFiles(r.files ?? []);
      setDirs(r.dirs ?? []);
    } catch (e) {
      setError(`List failed: ${(e as Error).message}`);
    }
  };
  useEffect(() => {
    refresh();
  }, []);

  // Ctrl+S on an untitled config asks us to open Save As (App dispatches it).
  useEffect(() => {
    const h = () => setSaveOpen(true);
    window.addEventListener("configbuilder:saveas", h);
    return () => window.removeEventListener("configbuilder:saveas", h);
  }, []);

  return (
    <div className="flex flex-wrap items-center gap-2">
      <button className="btn-ghost" disabled={busy} onClick={() => newConfig()}>
        New
      </button>

      <div className="relative">
        <button
          className="btn-ghost"
          disabled={busy}
          onClick={() => {
            setOpenMenu((o) => !o);
            if (!openMenu) refresh();
          }}
        >
          Open ▾
        </button>
        {openMenu ? (
          <div className="absolute z-30 mt-1 max-h-80 w-80 overflow-y-auto rounded-md border border-white/15 bg-panel p-1 shadow-lg">
            {files.length === 0 ? (
              <div className="help p-2">No config files found in the discovery directories.</div>
            ) : (
              files.map((f) => (
                <button
                  key={f.path}
                  className="block w-full rounded px-2 py-1.5 text-left text-sm hover:bg-white/5"
                  onClick={() => {
                    setOpenMenu(false);
                    load(f.path);
                  }}
                  title={f.path}
                >
                  <span className={f.valid ? "" : "text-warn"}>
                    {f.valid ? "" : "⚠ "}
                    {f.name}
                  </span>
                  <span className="help block truncate">{f.dir}</span>
                </button>
              ))
            )}
          </div>
        ) : null}
      </div>

      <button
        className="btn"
        disabled={busy}
        onClick={async () => {
          if (!path) {
            setSaveOpen(true);
            return;
          }
          if (await save(path, true)) refresh();
        }}
      >
        Save
      </button>
      <button className="btn-ghost" disabled={busy} onClick={() => setSaveOpen(true)}>
        Save As…
      </button>
      {dirty ? (
        <button
          className="btn-ghost"
          disabled={busy}
          title="Discard unsaved changes and reload from disk"
          onClick={() => {
            if (window.confirm("Discard unsaved changes?")) void revert();
          }}
        >
          Revert
        </button>
      ) : null}
      <button className="btn-ghost" onClick={() => validateAll()}>
        Validate all
      </button>
      <button
        className="btn-ghost"
        title="Download the current config as a YAML file (no server write)"
        onClick={async () => {
          const cfg = useStore.getState().config;
          if (!cfg) return;
          try {
            const { yaml } = await api.marshal(cfg);
            const url = URL.createObjectURL(new Blob([yaml], { type: "text/yaml" }));
            const a = document.createElement("a");
            a.href = url;
            a.download = downloadName(path);
            a.click();
            URL.revokeObjectURL(url);
          } catch (e) {
            setError(`Download failed: ${(e as Error).message}`);
          }
        }}
      >
        Download YAML
      </button>

      <span className="help ml-2 truncate">
        {path ? path : "untitled (new config)"}
        {dirty ? " — unsaved changes" : ""}
      </span>

      {saveOpen ? (
        <SaveDialog
          dirs={dirs}
          files={files}
          defaultPath={path}
          onClose={() => setSaveOpen(false)}
          onSave={async (target, overwrite) => {
            const ok = await save(target, overwrite);
            if (ok) {
              setSaveOpen(false);
              refresh();
            }
          }}
        />
      ) : null}
    </div>
  );
}

function SaveDialog(props: {
  dirs: string[];
  files: ConfigFileInfo[];
  defaultPath: string;
  onClose: () => void;
  onSave: (path: string, overwrite: boolean) => void;
}) {
  const setError = useStore((s) => s.setError);
  const lastError = useStore((s) => s.errors[s.errors.length - 1] ?? null);
  const [extraDirs, setExtraDirs] = useState<string[]>([]);
  const allDirs = [...props.dirs, ...extraDirs];
  const [dir, setDir] = useState(allDirs[0] ?? "");
  const [name, setName] = useState(
    props.defaultPath ? props.defaultPath.split(/[/\\]/).pop()! : "config.yaml",
  );
  const [newFolder, setNewFolder] = useState("");
  const [overwrite, setOverwrite] = useState(false);
  const target = joinPath(dir, name.trim());
  const collides = !!name.trim() && pathCollides(props.files, target);

  const createFolder = async () => {
    const folder = newFolder.trim();
    if (!folder || !dir) return;
    const full = joinPath(dir, folder);
    try {
      const r = await api.mkdir(full);
      setExtraDirs((d) => (d.includes(r.path) ? d : [...d, r.path]));
      setDir(r.path);
      setNewFolder("");
    } catch (e) {
      setError(`Create folder failed: ${(e as Error).message}`);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center bg-black/60 p-4">
      <div className="card w-full max-w-lg space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-base font-semibold">Save config</h3>
          <button className="btn-ghost" onClick={props.onClose}>
            Close
          </button>
        </div>
        <label className="block">
          <span className="label">Directory</span>
          <select className="input" value={dir} onChange={(e) => setDir(e.target.value)}>
            {allDirs.map((d) => (
              <option key={d} value={d}>
                {d}
              </option>
            ))}
          </select>
          <p className="help mt-1">
            Saves are restricted to the server's config directories (and
            subfolders you create below).
          </p>
        </label>
        <div className="flex items-end gap-2">
          <label className="block flex-1">
            <span className="label">New subfolder</span>
            <input
              className="input"
              value={newFolder}
              placeholder="e.g. clients/acme"
              onChange={(e) => setNewFolder(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && (e.preventDefault(), createFolder())}
            />
          </label>
          <button className="btn-ghost" disabled={!newFolder.trim()} onClick={createFolder}>
            Create
          </button>
        </div>
        <label className="block">
          <span className="label">Filename</span>
          <input
            className="input"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="config.yaml"
          />
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={overwrite} onChange={(e) => setOverwrite(e.target.checked)} />
          Overwrite if the file already exists
        </label>
        {collides ? (
          <p className="text-xs text-warn">
            ⚠ {target} already exists. Saving re-marshals the whole file and
            <strong> strips any YAML comments</strong> it had. Tick “Overwrite”
            to proceed.
          </p>
        ) : null}
        {lastError ? <p className="text-xs text-err">{lastError}</p> : null}
        <div className="flex justify-end gap-2">
          <button className="btn-ghost" onClick={props.onClose}>
            Cancel
          </button>
          <button
            className="btn"
            disabled={!name.trim() || !dir || (collides && !overwrite)}
            onClick={() => props.onSave(target, overwrite)}
          >
            Save
          </button>
        </div>
      </div>
    </div>
  );
}
