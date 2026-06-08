import { useEffect, useState } from "react";
import { api } from "../api/client";
import { writes } from "../api/write";
import type { DetectedSignal, HuntStatus } from "../api/types";
import { selectCanMutate, selectClientConfig, useShared } from "../store/shared";

const POLL_INTERVAL_MS = 2_000;

// sortSignals returns a copy of the inventory sorted by the chosen column
// (frequency ascending, class alphabetical, or SNR descending).
function sortSignals(signals: DetectedSignal[], by: "freq" | "class" | "snr"): DetectedSignal[] {
  const out = [...signals];
  out.sort((a, b) => {
    switch (by) {
      case "class":
        return a.class.localeCompare(b.class) || a.freq_hz - b.freq_hz;
      case "snr":
        return b.snr_db - a.snr_db;
      default:
        return a.freq_hz - b.freq_hz;
    }
  });
  return out;
}

// signalDetail renders the per-class decode summary for one surveyed carrier.
function signalDetail(sig: DetectedSignal): string {
  if (sig.trunking) {
    return `${sig.trunking.protocol}${sig.trunking.locked ? " (locked)" : ""}`;
  }
  if (sig.pages && sig.pages.length > 0) {
    return `${sig.pages.length} page(s)`;
  }
  if (sig.analog?.active) {
    const tone = sig.analog.ctcss_hz
      ? ` CTCSS ${sig.analog.ctcss_hz.toFixed(1)}`
      : sig.analog.dcs_code
        ? ` DCS ${sig.analog.dcs_code}`
        : "";
    return `active${tone}`;
  }
  return "—";
}

// Hunt drives the live system-discovery (blind spectrum-sweep) cockpit: start a
// run over operator-given bands (or a candidate list), watch its progress, and
// download / commit the discovered system. Mutations are gated behind
// selectCanMutate.
export function Hunt() {
  const cfg = useShared(selectClientConfig);
  const canMutate = useShared(selectCanMutate);
  const setError = useShared((s) => s.setError);

  const [status, setStatus] = useState<HuntStatus | null>(null);
  const [bands, setBands] = useState("851:869");
  const [candidates, setCandidates] = useState("");
  const [name, setName] = useState("");
  const [stateCode, setStateCode] = useState("");
  const [county, setCounty] = useState("");
  const [serial, setSerial] = useState("");
  const [protocol, setProtocol] = useState("");
  const [survey, setSurvey] = useState(false);
  const [classifyOnly, setClassifyOnly] = useState(false);
  const [sortBy, setSortBy] = useState<"freq" | "class" | "snr">("freq");

  useEffect(() => {
    let cancel = false;
    const refresh = async () => {
      try {
        const data = await api.hunt(cfg);
        if (!cancel) setStatus(data);
      } catch {
        // keep the previous snapshot
      }
    };
    refresh();
    const t = window.setInterval(refresh, POLL_INTERVAL_MS);
    return () => {
      cancel = true;
      window.clearInterval(t);
    };
  }, [cfg]);

  async function start() {
    const bandList = bands
      .split(",")
      .map((b) => b.trim())
      .filter(Boolean);
    const candList = candidates
      .split(",")
      .map((c) => parseFloat(c.trim()))
      .filter((n) => !Number.isNaN(n));
    try {
      await writes.huntStart(cfg, {
        bands: bandList.length ? bandList : undefined,
        candidates: candList.length ? candList : undefined,
        no_sweep: candList.length > 0 && bandList.length === 0,
        survey: survey || undefined,
        classify_only: (survey && classifyOnly) || undefined,
        name: name || undefined,
        state: stateCode || undefined,
        county: county || undefined,
        serial: serial || undefined,
        protocol: protocol || undefined,
      });
    } catch (e: unknown) {
      setError(e instanceof Error ? `start hunt failed: ${e.message}` : "start hunt failed");
    }
  }

  async function stop() {
    try {
      await writes.huntStop(cfg);
    } catch (e: unknown) {
      setError(e instanceof Error ? `stop hunt failed: ${e.message}` : "stop hunt failed");
    }
  }

  const running = status?.running ?? false;
  const exportBase = `${cfg.baseURL}/api/v1/hunt/export`;

  return (
    <div className="panel hunt-panel">
      <h2>Hunt — discover an unknown system</h2>

      <section className="hunt-status">
        <div>
          State: <strong>{status?.state ?? "idle"}</strong>
          {running ? " ●" : ""}
        </div>
        {status?.phase ? (
          <div>
            Phase: {status.phase}
            {status.detail ? ` — ${status.detail}` : ""}
          </div>
        ) : null}
        {status?.error ? <div className="error">Error: {status.error}</div> : null}
        {status?.mode ? (
          <div>
            Mode: <strong>{status.mode}</strong>
          </div>
        ) : null}
        {status?.system_name ? (
          <div>
            Discovered: <strong>{status.system_name}</strong> — {status.sites} site(s),{" "}
            {status.talkgroups} talkgroup(s)
          </div>
        ) : status?.mode === "survey" ? null : (
          <div>No system discovered yet.</div>
        )}
      </section>

      {status?.signals && status.signals.length > 0 ? (
        <section className="hunt-signals">
          <h3>Signals ({status.signals.length})</h3>
          <table>
            <thead>
              <tr>
                <th onClick={() => setSortBy("freq")} style={{ cursor: "pointer" }}>
                  Frequency{sortBy === "freq" ? " ▾" : ""}
                </th>
                <th onClick={() => setSortBy("class")} style={{ cursor: "pointer" }}>
                  Class{sortBy === "class" ? " ▾" : ""}
                </th>
                <th>BW (kHz)</th>
                <th onClick={() => setSortBy("snr")} style={{ cursor: "pointer" }}>
                  SNR (dB){sortBy === "snr" ? " ▾" : ""}
                </th>
                <th>Decode</th>
              </tr>
            </thead>
            <tbody>
              {sortSignals(status.signals, sortBy).map((sig) => (
                <tr key={sig.freq_hz}>
                  <td>{(sig.freq_hz / 1e6).toFixed(4)} MHz</td>
                  <td>{sig.class}</td>
                  <td>{(sig.occupied_bw_hz / 1e3).toFixed(1)}</td>
                  <td>{sig.snr_db.toFixed(1)}</td>
                  <td>{signalDetail(sig)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      ) : null}

      <section className="hunt-controls">
        <label>
          Bands (MHz, low:high, comma-separated)
          <input value={bands} onChange={(e) => setBands(e.target.value)} placeholder="851:869" />
        </label>
        <label>
          Candidates (MHz, comma-separated — skips the sweep)
          <input
            value={candidates}
            onChange={(e) => setCandidates(e.target.value)}
            placeholder="851.0125, 853.5125"
          />
        </label>
        <label>
          Name
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder="New County P25" />
        </label>
        <label>
          State
          <input value={stateCode} onChange={(e) => setStateCode(e.target.value)} placeholder="AZ" />
        </label>
        <label>
          County
          <input value={county} onChange={(e) => setCounty(e.target.value)} placeholder="Maricopa" />
        </label>
        <label>
          SDR serial (optional — auto-selects a spare, else borrows control)
          <input value={serial} onChange={(e) => setSerial(e.target.value)} placeholder="00000001" />
        </label>
        <label>
          Protocol (optional — default auto-identifies)
          <input value={protocol} onChange={(e) => setProtocol(e.target.value)} placeholder="p25" />
        </label>
        <label className="hunt-survey-toggle">
          <input type="checkbox" checked={survey} onChange={(e) => setSurvey(e.target.checked)} />
          Survey mode — classify &amp; decode every signal (analog, paging, trunking)
        </label>
        {survey ? (
          <label className="hunt-survey-toggle">
            <input
              type="checkbox"
              checked={classifyOnly}
              onChange={(e) => setClassifyOnly(e.target.checked)}
            />
            Classify only — skip decoding (fast inventory)
          </label>
        ) : null}
        <div className="hunt-buttons">
          <button onClick={start} disabled={!canMutate || running}>
            Start hunt
          </button>
          <button onClick={stop} disabled={!canMutate || !running}>
            Stop
          </button>
        </div>
      </section>

      {status?.system_name ? (
        <section className="hunt-export">
          <span>Export:</span>
          <a href={`${exportBase}?format=bundle`}>GopherTrunk bundle</a>
          <a href={`${exportBase}?format=trunk-recorder`}>trunk-recorder</a>
          <a href={`${exportBase}?format=rr`}>RadioReference package</a>
        </section>
      ) : null}

      {status?.signals && status.signals.length > 0 ? (
        <section className="hunt-export">
          <span>Survey:</span>
          <a href={`${cfg.baseURL}/api/v1/hunt/survey?format=json`}>signals JSON</a>
          <a href={`${cfg.baseURL}/api/v1/hunt/survey?format=csv`}>signals CSV</a>
        </section>
      ) : null}

      {!canMutate ? (
        <p className="hint">Mutations are read-only on this connection (no auth token).</p>
      ) : null}
    </div>
  );
}
