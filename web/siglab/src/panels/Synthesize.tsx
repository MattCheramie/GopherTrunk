import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useStore } from "../store/shared";
import { api } from "../api/client";
import type { SynthRequest } from "../api/types";

// Synthesize mirrors the `gen` CLI: build an idealized or impaired capture for
// a protocol. The result is staged as a capture (so it can be run and used as
// a comparison reference) or downloaded.
export function Synthesize() {
  const fixtures = useStore((s) => s.fixtures);
  const config = useStore((s) => s.config);
  const synthesize = useStore((s) => s.synthesize);
  const navigate = useNavigate();
  const [req, setReq] = useState<SynthRequest>({
    protocol: fixtures[0] ?? "p25",
    format: "f32",
    seed: 1,
  });
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  function set<K extends keyof SynthRequest>(k: K, v: SynthRequest[K]) {
    setReq((r) => ({ ...r, [k]: v }));
  }

  async function onStage() {
    setBusy(true);
    setErr(null);
    try {
      const cap = await synthesize(req);
      navigate(`/captures`);
      void cap;
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function onDownload() {
    setBusy(true);
    setErr(null);
    try {
      const res = await fetch(api.synthDownloadURL(config), {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(config.token ? { Authorization: `Bearer ${config.token}` } : {}),
        },
        body: JSON.stringify(req),
        credentials: "include",
      });
      if (!res.ok) throw new Error(await res.text());
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${req.protocol}.${req.format}`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  const num = (k: keyof SynthRequest) => (e: React.ChangeEvent<HTMLInputElement>) =>
    set(k, (Number(e.target.value) || 0) as never);

  return (
    <div className="mx-auto max-w-2xl">
      <div className="card space-y-3">
        <h2 className="text-lg font-semibold">Synthesize idealized / impaired signal</h2>
        <div className="grid grid-cols-2 gap-3 md:grid-cols-3">
          <div>
            <label className="label">Protocol</label>
            <select
              className="input"
              value={req.protocol}
              onChange={(e) => set("protocol", e.target.value)}
            >
              {fixtures.map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="label">Format</label>
            <select className="input" value={req.format} onChange={(e) => set("format", e.target.value)}>
              <option value="f32">f32</option>
              <option value="u8">u8</option>
            </select>
          </div>
          <div>
            <label className="label">Modulation (P25)</label>
            <select
              className="input"
              value={req.modulation ?? ""}
              onChange={(e) => set("modulation", e.target.value || undefined)}
            >
              <option value="">default</option>
              <option value="c4fm">c4fm</option>
              <option value="cqpsk">cqpsk / lsm</option>
            </select>
          </div>
        </div>

        <h3 className="pt-2 text-sm font-semibold text-muted">Impairments (0 = clean)</h3>
        <div className="grid grid-cols-2 gap-3 md:grid-cols-3">
          <Field label="SNR (dB)" v={req.snr_db} on={num("snr_db")} />
          <Field label="Freq offset (Hz)" v={req.freq_offset_hz} on={num("freq_offset_hz")} />
          <Field label="Drift (Hz/s)" v={req.freq_drift_hz_per_sec} on={num("freq_drift_hz_per_sec")} />
          <Field label="DC offset" v={req.dc_offset} on={num("dc_offset")} />
          <Field label="IQ gain" v={req.iq_gain_imbalance} on={num("iq_gain_imbalance")} />
          <Field label="IQ phase (rad)" v={req.iq_phase_skew_rad} on={num("iq_phase_skew_rad")} />
          <Field label="Seed" v={req.seed} on={num("seed")} />
          <div className="col-span-2">
            <label className="label">Multipath</label>
            <input
              className="input"
              placeholder='"simulcast" or "delay:mag:deg,…"'
              value={req.multipath ?? ""}
              onChange={(e) => set("multipath", e.target.value || undefined)}
            />
          </div>
        </div>

        <div className="flex gap-2">
          <button className="btn" disabled={busy} onClick={onStage}>
            {busy ? "Working…" : "Stage as capture"}
          </button>
          <button className="btn-ghost" disabled={busy} onClick={onDownload}>
            Download
          </button>
        </div>
        {err && <p className="text-xs text-err">{err}</p>}
      </div>
    </div>
  );
}

function Field({
  label,
  v,
  on,
}: {
  label: string;
  v: number | undefined;
  on: (e: React.ChangeEvent<HTMLInputElement>) => void;
}) {
  return (
    <div>
      <label className="label">{label}</label>
      <input className="input" type="number" step="any" value={v ?? 0} onChange={on} />
    </div>
  );
}
