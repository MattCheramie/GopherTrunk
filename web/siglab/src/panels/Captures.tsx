import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useStore } from "../store/shared";
import { api } from "../api/client";
import type { CaptureDTO, CaptureDevice, RunConfig } from "../api/types";

export function Captures() {
  const captures = useStore((s) => s.captures);
  const selectedId = useStore((s) => s.selectedCaptureId);
  const select = useStore((s) => s.select);
  const toggleCompare = useStore((s) => s.toggleCompare);
  const compareIds = useStore((s) => s.compareIds);
  const removeCapture = useStore((s) => s.removeCapture);
  const selected = captures.find((c) => c.id === selectedId);

  return (
    <div className="grid gap-4 lg:grid-cols-[360px_1fr]">
      <div className="space-y-4">
        <UploadForm />
        <CaptureForm />
        <div className="card">
          <h3 className="mb-2 text-sm font-semibold">Captures ({captures.length})</h3>
          {captures.length === 0 ? (
            <p className="text-xs text-muted">
              Upload a raw IQ capture, or synthesize one on the Synthesize tab.
            </p>
          ) : (
            <ul className="space-y-1">
              {captures.map((c) => (
                <li
                  key={c.id}
                  className={`flex items-center gap-2 rounded-md px-2 py-1.5 text-sm ${
                    c.id === selectedId ? "bg-accent/15" : "hover:bg-panel"
                  }`}
                >
                  <button className="flex-1 text-left" onClick={() => select(c.id)}>
                    <div className="truncate">{c.name}</div>
                    <div className="text-xs text-muted">
                      {c.format} · {c.sample_rate_hz ? `${c.sample_rate_hz} Hz` : "rate?"} ·{" "}
                      {(c.size / 1024).toFixed(0)} KiB
                    </div>
                  </button>
                  <label className="flex items-center gap-1 text-xs text-muted">
                    <input
                      type="checkbox"
                      checked={compareIds.includes(c.id)}
                      onChange={() => toggleCompare(c.id)}
                    />
                    cmp
                  </label>
                  <button
                    className="text-xs text-err/80 hover:text-err"
                    onClick={() => removeCapture(c.id)}
                  >
                    ✕
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      <div>
        {selected ? (
          <RunForm capture={selected} />
        ) : (
          <div className="card text-sm text-muted">
            Select a capture to configure and run the engine.
          </div>
        )}
      </div>
    </div>
  );
}

function UploadForm() {
  const upload = useStore((s) => s.uploadCapture);
  const formats = useStore((s) => s.formats);
  const fileRef = useRef<HTMLInputElement | null>(null);
  const [format, setFormat] = useState("f32");
  const [rate, setRate] = useState("2400000");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    const file = fileRef.current?.files?.[0];
    if (!file) return;
    setBusy(true);
    setErr(null);
    try {
      await upload(file, format, Number(rate) || undefined);
      if (fileRef.current) fileRef.current.value = "";
    } catch (e2) {
      setErr(e2 instanceof Error ? e2.message : String(e2));
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="card space-y-2" onSubmit={onSubmit}>
      <h3 className="text-sm font-semibold">Upload capture</h3>
      <input ref={fileRef} type="file" className="input" />
      <div className="grid grid-cols-2 gap-2">
        <div>
          <label className="label">Format</label>
          <select className="input" value={format} onChange={(e) => setFormat(e.target.value)}>
            {formats.map((f) => (
              <option key={f} value={f}>
                {f}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="label">Sample rate (Hz)</label>
          <input className="input" value={rate} onChange={(e) => setRate(e.target.value)} />
        </div>
      </div>
      <button className="btn" disabled={busy}>
        {busy ? "Uploading…" : "Upload"}
      </button>
      {err && <p className="text-xs text-err">{err}</p>}
    </form>
  );
}

// CaptureForm records a fixed-length raw-IQ capture off a live tuner, stages
// it (so it appears in the capture list and is immediately runnable), and
// surfaces a .cfile download link. Only functional when the SigLab console is
// served by a running daemon with an SDR; the offline `siglab serve` host (and
// a daemon with no SDR) returns 503, which we render as a disabled hint.
function CaptureForm() {
  const config = useStore((s) => s.config);
  const captureFromTuner = useStore((s) => s.captureFromTuner);
  const formats = useStore((s) => s.formats);
  const protocols = useStore((s) => s.protocols);

  const [devices, setDevices] = useState<CaptureDevice[] | null>(null);
  const [unavailable, setUnavailable] = useState<string | null>(null);
  const [serial, setSerial] = useState("");
  const [seconds, setSeconds] = useState("10");
  const [format, setFormat] = useState("f32");
  const [protocol, setProtocol] = useState("");
  const [centerMHz, setCenterMHz] = useState("");
  const [bandwidthKHz, setBandwidthKHz] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [downloadID, setDownloadID] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .captureDevices(config)
      .then((d) => {
        if (cancelled) return;
        setDevices(d);
        if (d.length > 0) setSerial((s) => s || d[0].serial);
      })
      .catch((e) => {
        if (cancelled) return;
        setUnavailable(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancelled = true;
    };
  }, [config]);

  async function onCapture() {
    setBusy(true);
    setErr(null);
    setDownloadID(null);
    try {
      const bandwidthHz = bandwidthKHz.trim() ? Math.round(Number(bandwidthKHz) * 1e3) : 0;
      const centerHz = centerMHz.trim() ? Math.round(Number(centerMHz) * 1e6) : 0;
      const res = await captureFromTuner({
        serial,
        seconds: Number(seconds) || 1,
        format,
        protocol: protocol || undefined,
        // A narrowband slice is requested only when a bandwidth is given; the
        // centre defaults to the tuner centre server-side when omitted.
        bandwidth_hz: bandwidthHz > 0 ? bandwidthHz : undefined,
        center_hz: bandwidthHz > 0 && centerHz > 0 ? centerHz : undefined,
      });
      setDownloadID(res.capture.id);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  if (unavailable) {
    return (
      <div className="card space-y-1">
        <h3 className="text-sm font-semibold">Capture from tuner</h3>
        <p className="text-xs text-muted">
          Live capture is unavailable — connect the console to a running daemon with an SDR.
        </p>
      </div>
    );
  }

  return (
    <div className="card space-y-2">
      <h3 className="text-sm font-semibold">Capture from tuner</h3>
      <div>
        <label className="label">SDR</label>
        <select className="input" value={serial} onChange={(e) => setSerial(e.target.value)}>
          {(devices ?? []).map((d) => (
            <option key={d.serial} value={d.serial}>
              {d.serial} · {d.driver}
              {d.center_hz ? ` · ${(d.center_hz / 1e6).toFixed(3)} MHz` : ""}
            </option>
          ))}
        </select>
      </div>
      <div className="grid grid-cols-3 gap-2">
        <div>
          <label className="label">Seconds</label>
          <input className="input" value={seconds} onChange={(e) => setSeconds(e.target.value)} />
        </div>
        <div>
          <label className="label">Format</label>
          <select className="input" value={format} onChange={(e) => setFormat(e.target.value)}>
            {formats.map((f) => (
              <option key={f} value={f}>
                {f}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="label">Protocol</label>
          <select
            className="input"
            value={protocol}
            onChange={(e) => setProtocol(e.target.value)}
          >
            <option value="">(none)</option>
            {protocols.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
        </div>
      </div>
      <div className="grid grid-cols-2 gap-2">
        <div>
          <label className="label">Center MHz (optional)</label>
          <input
            className="input"
            placeholder="tuner centre"
            value={centerMHz}
            onChange={(e) => setCenterMHz(e.target.value)}
          />
        </div>
        <div>
          <label className="label">Bandwidth kHz (optional)</label>
          <input
            className="input"
            placeholder="full band"
            value={bandwidthKHz}
            onChange={(e) => setBandwidthKHz(e.target.value)}
          />
        </div>
      </div>
      <p className="text-xs text-muted">
        Set a bandwidth to save a small narrowband slice around the centre (carved from the
        tuner&rsquo;s current span, no retune). Leave blank for a full-band grab. cs16 = 16-bit raw
        (half the size of f32).
      </p>
      <button className="btn" disabled={busy || !serial} onClick={onCapture}>
        {busy ? "Capturing…" : "Capture"}
      </button>
      {downloadID && (
        <a
          className="btn-ghost block text-center"
          href={api.captureDownloadURL(config, downloadID)}
        >
          Download capture
        </a>
      )}
      {err && <p className="text-xs text-err">{err}</p>}
    </div>
  );
}

function RunForm({ capture }: { capture: CaptureDTO }) {
  const protocols = useStore((s) => s.protocols);
  const run = useStore((s) => s.run);
  const navigate = useNavigate();
  const [cfg, setCfg] = useState<RunConfig>({
    protocol: protocols[0] ?? "p25",
    sample_rate_hz: capture.sample_rate_hz || undefined,
    format: capture.format,
    collect_iq_diag: true,
    capture_iq: true,
  });
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  function set<K extends keyof RunConfig>(k: K, v: RunConfig[K]) {
    setCfg((c) => ({ ...c, [k]: v }));
  }

  async function onRun() {
    setBusy(true);
    setErr(null);
    try {
      navigate(`/results/${capture.id}`);
      await run(capture.id, cfg);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="card space-y-3">
      <h3 className="text-sm font-semibold">Run engine — {capture.name}</h3>
      <div className="grid grid-cols-2 gap-3 md:grid-cols-3">
        <div>
          <label className="label">Protocol</label>
          <select
            className="input"
            value={cfg.protocol}
            onChange={(e) => set("protocol", e.target.value)}
          >
            {protocols.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="label">Sample rate (Hz)</label>
          <input
            className="input"
            value={cfg.sample_rate_hz ?? ""}
            onChange={(e) => set("sample_rate_hz", Number(e.target.value) || undefined)}
          />
        </div>
        <div>
          <label className="label">Tune (Hz)</label>
          <input
            className="input"
            value={cfg.tune_hz ?? ""}
            onChange={(e) => set("tune_hz", Number(e.target.value) || undefined)}
          />
        </div>
      </div>
      <div className="flex flex-wrap gap-4 text-sm">
        <Toggle label="auto-tune" v={!!cfg.auto_tune} on={(v) => set("auto_tune", v)} />
        <Toggle label="conjugate" v={!!cfg.conjugate} on={(v) => set("conjugate", v)} />
        <Toggle label="iq-correct" v={!!cfg.iq_correct} on={(v) => set("iq_correct", v)} />
        <Toggle
          label="collect IQ diag"
          v={!!cfg.collect_iq_diag}
          on={(v) => set("collect_iq_diag", v)}
        />
        <Toggle label="capture IQ" v={!!cfg.capture_iq} on={(v) => set("capture_iq", v)} />
        <Toggle
          label="receiver state"
          v={!!cfg.collect_receiver_state}
          on={(v) => set("collect_receiver_state", v)}
        />
        <Toggle label="collect PDUs" v={!!cfg.collect_pdus} on={(v) => set("collect_pdus", v)} />
      </div>
      <details className="text-sm">
        <summary className="cursor-pointer text-muted">P25 deep knobs</summary>
        <div className="mt-2 grid grid-cols-2 gap-3 md:grid-cols-3">
          <div>
            <label className="label">Demod mode</label>
            <select
              className="input"
              value={cfg.demod_mode ?? ""}
              onChange={(e) => set("demod_mode", e.target.value || undefined)}
            >
              <option value="">default (c4fm)</option>
              <option value="c4fm">c4fm</option>
              <option value="cqpsk">cqpsk</option>
            </select>
          </div>
          <Toggle label="DDA" v={!!cfg.enable_dda} on={(v) => set("enable_dda", v)} />
          <Toggle
            label="adaptive slicer"
            v={!!cfg.enable_adaptive_slicer}
            on={(v) => set("enable_adaptive_slicer", v)}
          />
        </div>
      </details>
      <button className="btn" disabled={busy} onClick={onRun}>
        {busy ? "Running…" : "Run"}
      </button>
      {err && <p className="text-xs text-err">{err}</p>}
    </div>
  );
}

function Toggle({ label, v, on }: { label: string; v: boolean; on: (v: boolean) => void }) {
  return (
    <label className="flex items-center gap-1.5">
      <input type="checkbox" checked={v} onChange={(e) => on(e.target.checked)} />
      {label}
    </label>
  );
}
