import { useState } from "react";

// Bundle inspects a GopherTrunk Bundle (.gtb.tar.gz) reachable on the daemon
// host via the /api/v1/bundle/* endpoints (internal/api/handlers_bundle.go):
// show its manifest + file inventory, verify every file's sha256, and offer a
// download. Packing runs through the CLI; this is the read/inspect touch point.

interface BundleFile {
  path: string;
  role: string;
  media_type: string;
  size_bytes: number;
  sha256: string;
  produced_by?: string;
}

interface BundleInfo {
  name: string;
  title?: string;
  format: string;
  schema_version: number;
  capture_intent: string;
  created_at: string;
  generator: string;
  protocol?: string;
  source?: string;
  future_schema?: boolean;
  files: BundleFile[];
  role_counts: Record<string, number>;
}

interface VerifyFile {
  path: string;
  role?: string;
  ok: boolean;
  untracked?: boolean;
  error?: string;
}

interface VerifyResult {
  ok: boolean;
  files: VerifyFile[];
  failures: number;
  untracked: number;
}

function humanSize(n: number): string {
  if (n < 1024) return `${n} B`;
  const units = ["KiB", "MiB", "GiB", "TiB"];
  let v = n / 1024;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v.toFixed(1)} ${units[i]}`;
}

export function Bundles() {
  const [path, setPath] = useState("");
  const [info, setInfo] = useState<BundleInfo | null>(null);
  const [verify, setVerify] = useState<VerifyResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function onInspect() {
    if (!path) return;
    setBusy(true);
    setErr(null);
    setVerify(null);
    try {
      const r = await fetch(`/api/v1/bundle/info?path=${encodeURIComponent(path)}`);
      if (!r.ok) throw new Error(await r.text());
      setInfo((await r.json()) as BundleInfo);
    } catch (e) {
      setInfo(null);
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function onVerify() {
    if (!path) return;
    setBusy(true);
    setErr(null);
    try {
      const r = await fetch("/api/v1/bundle/verify", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path }),
      });
      if (!r.ok) throw new Error(await r.text());
      setVerify((await r.json()) as VerifyResult);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-4xl space-y-4">
      <div className="card space-y-2">
        <div className="flex flex-wrap items-end gap-3">
          <div className="flex-1">
            <label className="label">Bundle path (on the daemon host)</label>
            <input
              className="input"
              placeholder="/var/lib/gophertrunk/recordings/mesa-p25.gtb.tar.gz"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && onInspect()}
            />
          </div>
          <button className="btn" disabled={busy || !path} onClick={onInspect}>
            {busy ? "Loading…" : "Inspect"}
          </button>
          <button className="btn" disabled={busy || !path} onClick={onVerify}>
            Verify
          </button>
          {info && (
            <a className="btn" href={`/api/v1/bundle/download?path=${encodeURIComponent(path)}`}>
              Download
            </a>
          )}
        </div>
        <p className="text-xs text-muted">
          Reads a GopherTrunk Bundle already on the daemon host. Build one with{" "}
          <code>gophertrunk capture -bundle …</code> or <code>gophertrunk bundle pack</code>.
        </p>
      </div>

      {err && <div className="card text-xs text-err">{err}</div>}

      {info && (
        <div className="card space-y-3">
          <div className="flex flex-wrap items-baseline gap-x-6 gap-y-1">
            <span className="font-semibold">{info.title || info.name}</span>
            <span className="text-xs text-muted">intent: {info.capture_intent}</span>
            {info.protocol && <span className="text-xs text-muted">protocol: {info.protocol}</span>}
            <span className="text-xs text-muted">{info.generator}</span>
            <span className="text-xs text-muted">{info.created_at}</span>
          </div>
          {info.future_schema && (
            <div className="text-xs text-err">
              schema v{info.schema_version} is newer than this build supports — reading best-effort
            </div>
          )}
          {info.source && <div className="text-xs text-muted">{info.source}</div>}
          <table className="w-full text-xs">
            <thead>
              <tr className="text-left text-muted">
                <th className="py-1">Role</th>
                <th>Path</th>
                <th className="text-right">Size</th>
                <th>By</th>
                {verify && <th className="text-right">Verify</th>}
              </tr>
            </thead>
            <tbody>
              {info.files.map((f) => {
                const v = verify?.files.find((x) => x.path === f.path);
                return (
                  <tr key={f.path} className="border-t border-line/50">
                    <td className="py-1 font-mono">{f.role}</td>
                    <td className="font-mono">{f.path}</td>
                    <td className="text-right">{humanSize(f.size_bytes)}</td>
                    <td>{f.produced_by || "—"}</td>
                    {verify && (
                      <td className="text-right">
                        {v ? (v.ok ? <span className="text-ok">✓</span> : <span className="text-err">✗</span>) : "—"}
                      </td>
                    )}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {verify && (
        <div className={`card text-xs ${verify.ok ? "text-ok" : "text-err"}`}>
          {verify.ok
            ? `Integrity OK — ${verify.files.length} file(s) checked, ${verify.untracked} untracked`
            : `Integrity FAILED — ${verify.failures} file(s) do not match their sha256`}
        </div>
      )}
    </div>
  );
}
