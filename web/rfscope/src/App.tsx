import { useEffect, useState } from "react";
import {
  analyze,
  analyzers as fetchAnalyzers,
  type AnalyzeResponse,
  type AnalyzerInfo,
  type Channel,
  type Scene,
} from "./api";
import { ConsoleNav } from "./ConsoleNav";

export function App() {
  const [file, setFile] = useState<File | null>(null);
  const [sampleRate, setSampleRate] = useState("2400000");
  const [freq, setFreq] = useState("0");
  const [format, setFormat] = useState("f32");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [result, setResult] = useState<AnalyzeResponse | null>(null);
  const [analyzerList, setAnalyzerList] = useState<AnalyzerInfo[]>([]);

  useEffect(() => {
    fetchAnalyzers()
      .then(setAnalyzerList)
      .catch(() => setAnalyzerList([]));
  }, []);

  async function onAnalyze(e: React.FormEvent) {
    e.preventDefault();
    if (!file) {
      setErr("choose an IQ capture file first");
      return;
    }
    setBusy(true);
    setErr(null);
    try {
      const form = new FormData();
      form.set("file", file);
      form.set("format", format);
      form.set("sample_rate_hz", sampleRate);
      form.set("freq", freq);
      setResult(await analyze(form));
    } catch (e2) {
      setErr(String(e2));
      setResult(null);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-5xl p-4">
      <header className="mb-4 flex flex-wrap items-center justify-between gap-3 border-b border-line pb-3">
        <div className="flex items-center gap-2">
          <span className="text-accent">⧉</span>
          <h1 className="text-lg font-bold">GopherTrunk RF Scope</h1>
        </div>
        <div className="flex flex-wrap items-center gap-3 text-xs text-muted">
          <ConsoleNav self="rfscope" />
          <span>
            protocol-agnostic RF network analysis · {analyzerList.length} analyzers
          </span>
        </div>
      </header>

      <form className="card mb-4 grid grid-cols-1 gap-3 sm:grid-cols-5" onSubmit={onAnalyze}>
        <div className="sm:col-span-2">
          <label className="label">IQ capture</label>
          <input
            className="input"
            type="file"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          />
        </div>
        <div>
          <label className="label">Format</label>
          <select className="input" value={format} onChange={(e) => setFormat(e.target.value)}>
            <option value="f32">f32 (cfile)</option>
            <option value="u8">u8 (rtl_sdr)</option>
          </select>
        </div>
        <div>
          <label className="label">Sample rate (Hz)</label>
          <input className="input" value={sampleRate} onChange={(e) => setSampleRate(e.target.value)} />
        </div>
        <div>
          <label className="label">Center (Hz)</label>
          <input className="input" value={freq} onChange={(e) => setFreq(e.target.value)} />
        </div>
        <div className="sm:col-span-5">
          <button className="btn" type="submit" disabled={busy}>
            {busy ? "analyzing…" : "Analyze scene"}
          </button>
        </div>
      </form>

      {err && <div className="card mb-4 border-err text-err">{err}</div>}

      {result && <SceneView resp={result} />}
    </div>
  );
}

function SceneView({ resp }: { resp: AnalyzeResponse }) {
  const sc = resp.scene;
  return (
    <div className="grid grid-cols-1 gap-4">
      <SceneHeader sc={sc} />
      <Hierarchy sc={sc} />
      <Channels sc={sc} />
      <Talkers sc={sc} />
      <Conversations sc={sc} />
      <Expert sc={sc} />
      <CryptoBridge resp={resp} />
    </div>
  );
}

function SceneHeader({ sc }: { sc: Scene }) {
  return (
    <div className="card text-sm">
      <div className="font-semibold text-accent">{sc.source || "scene"}</div>
      <div className="mt-1 text-fg">
        center {(sc.center_hz / 1e6).toFixed(4)} MHz · span {(sc.span_hz / 1e6).toFixed(3)} MHz ·
        window {sc.window_sec.toFixed(1)}s · floor {sc.noise_floor_dbfs.toFixed(0)} dBFS
      </div>
      <div className="mt-1 text-muted">
        {(sc.bursts?.length ?? 0)} bursts · {(sc.channels?.length ?? 0)} channels ·{" "}
        {(sc.emitters?.length ?? 0)} emitters · {(sc.anomalies?.length ?? 0)} anomalies
      </div>
    </div>
  );
}

function Panel({ title, empty, children }: { title: string; empty: boolean; children: React.ReactNode }) {
  return (
    <div className="card">
      <h2 className="mb-2 text-sm font-semibold text-accent">{title}</h2>
      {empty ? <p className="text-xs text-muted">(none)</p> : children}
    </div>
  );
}

function Hierarchy({ sc }: { sc: Scene }) {
  const nodes = sc.hierarchy?.nodes ?? [];
  return (
    <Panel title="Protocol hierarchy" empty={nodes.length === 0}>
      <ul className="text-sm">
        {nodes.map((n, i) => (
          <li key={i} style={{ paddingLeft: `${n.depth * 16}px` }} className="text-fg">
            <span className={n.depth === 0 ? "font-semibold" : ""}>{n.label}</span>{" "}
            <span className="text-muted">
              {n.stats.burst_count} bursts · {n.stats.time_pct.toFixed(0)}% time ·{" "}
              {n.stats.spectrum_pct.toFixed(1)}% spectrum
            </span>
          </li>
        ))}
      </ul>
    </Panel>
  );
}

function sparkline(ch: Channel, width = 40): string {
  const tl = ch.timeline ?? [];
  if (tl.length === 0) return "·".repeat(width);
  let out = "";
  for (let i = 0; i < width; i++) {
    const lo = Math.floor((i * tl.length) / width);
    const hi = Math.max(lo + 1, Math.floor(((i + 1) * tl.length) / width));
    let occ = false;
    for (let j = lo; j < hi && j < tl.length; j++) if (tl[j].occupied) occ = true;
    out += occ ? "█" : "·";
  }
  return out;
}

function Channels({ sc }: { sc: Scene }) {
  const chans = sc.channels ?? [];
  return (
    <Panel title="Channels (I/O graph)" empty={chans.length === 0}>
      <table className="w-full text-sm">
        <thead className="text-left text-xs text-muted">
          <tr>
            <th className="pr-3">Freq (MHz)</th>
            <th className="pr-3">Class</th>
            <th className="pr-3">Activity</th>
            <th className="pr-3">Duty</th>
            <th className="pr-3">Period</th>
          </tr>
        </thead>
        <tbody>
          {chans.map((c, i) => (
            <tr key={i} className="text-fg">
              <td className="pr-3 font-mono">{(c.freq_hz / 1e6).toFixed(4)}</td>
              <td className="pr-3">{c.dominant_class}</td>
              <td className="pr-3 spark">{sparkline(c)}</td>
              <td className="pr-3">{(c.duty_cycle * 100).toFixed(0)}%</td>
              <td className="pr-3">{c.period_sec > 0 ? `${c.period_sec.toFixed(3)}s` : "—"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </Panel>
  );
}

function Talkers({ sc }: { sc: Scene }) {
  const ems = sc.emitters ?? [];
  return (
    <Panel title="Top talkers" empty={ems.length === 0}>
      <ul className="text-sm">
        {ems.slice(0, 10).map((e) => (
          <li key={e.id} className="text-fg">
            <span className="text-muted">#{e.id}</span> {e.label}{" "}
            <span className="text-muted">{e.total_airtime_sec.toFixed(2)}s</span>
            {e.hop_set && e.hop_set.length > 1 && (
              <span className="ml-1 text-warn">hop×{e.hop_set.length}</span>
            )}
          </li>
        ))}
      </ul>
    </Panel>
  );
}

function Conversations({ sc }: { sc: Scene }) {
  const convs = sc.conversations ?? [];
  return (
    <Panel title="Conversations" empty={convs.length === 0}>
      <ul className="text-sm">
        {convs.map((c) => (
          <li key={c.id} className="text-fg">
            <span className="font-mono">{c.kind}</span>{" "}
            <span className="text-muted">{c.emitter_ids.map((id) => `#${id}`).join(" ↔ ")}</span>{" "}
            <span className="text-muted">(corr {c.correlation.toFixed(2)})</span>
          </li>
        ))}
      </ul>
    </Panel>
  );
}

function severityColor(sev: string): string {
  if (sev === "alert") return "text-err";
  if (sev === "warn") return "text-warn";
  return "text-fg";
}

function Expert({ sc }: { sc: Scene }) {
  const an = sc.anomalies ?? [];
  return (
    <Panel title="Expert info" empty={an.length === 0}>
      <ul className="text-sm">
        {an.map((a, i) => (
          <li key={i} className={severityColor(a.severity)}>
            [{a.severity}] <span className="font-mono">{a.kind}</span> — {a.detail}
          </li>
        ))}
      </ul>
    </Panel>
  );
}

function CryptoBridge({ resp }: { resp: AnalyzeResponse }) {
  if (!resp.frame_count) return null;
  function download() {
    const blob = new Blob([resp.frames ?? ""], { type: "application/x-ndjson" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "rfscope-frames.jsonl";
    a.click();
    URL.revokeObjectURL(url);
  }
  return (
    <div className="card text-sm">
      <h2 className="mb-2 text-sm font-semibold text-accent">Crypto Lab bridge</h2>
      <p className="text-fg">
        {resp.frame_count} unknown payload{resp.frame_count === 1 ? "" : "s"} recovered as a cryptolab{" "}
        <code className="text-accent">ks</code> frames file.
      </p>
      <div className="mt-2 flex gap-2">
        <button className="btn" onClick={download}>
          Download frames
        </button>
        <span className="self-center text-xs text-muted">
          then: <code>cryptolab classify auto -in rfscope-frames.jsonl</code> or{" "}
          <code>cryptolab ks reuse -in rfscope-frames.jsonl</code>
        </span>
      </div>
    </div>
  );
}
