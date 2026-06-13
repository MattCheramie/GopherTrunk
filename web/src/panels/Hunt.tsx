import { useEffect, useState } from "react";
import { api } from "../api/client";
import { writes } from "../api/write";
import type {
  CaptureReport,
  DetectedSignal,
  HuntRRReport,
  HuntStatus,
} from "../api/types";
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

// captureResult summarises one hunt capture's decode outcome: locked /
// skipped (with reason) / errored / decoded-but-unlocked.
function captureResult(rep: CaptureReport): string {
  if (rep.skipped) return `skipped${rep.skip_reason ? ` — ${rep.skip_reason}` : ""}`;
  if (rep.error) return `error — ${rep.error}`;
  if (rep.locked) {
    return `locked${rep.confidence ? ` · conf ${(rep.confidence * 100).toFixed(0)}%` : ""}`;
  }
  return "no lock";
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
  const [persistSurvey, setPersistSurvey] = useState(false);
  const [resume, setResume] = useState(false);
  const [autoGain, setAutoGain] = useState(false);
  const [sortBy, setSortBy] = useState<"freq" | "class" | "snr">("freq");
  const [rrCounty, setRRCounty] = useState("");
  const [rrSID, setRRSID] = useState("");
  const [rrReport, setRRReport] = useState<HuntRRReport | null>(null);
  const [rrBusy, setRRBusy] = useState(false);

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
        persist_survey: (survey && persistSurvey) || undefined,
        resume: (survey && persistSurvey && resume) || undefined,
        auto_gain: autoGain || undefined,
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

  async function checkRR() {
    setRRBusy(true);
    try {
      const countyID = parseInt(rrCounty.trim(), 10);
      const checkSIDs = rrSID
        .split(",")
        .map((s) => parseInt(s.trim(), 10))
        .filter((n) => Number.isFinite(n) && n > 0);
      const rep = await api.huntRadioReference(cfg, {
        countyID: Number.isFinite(countyID) && countyID > 0 ? countyID : undefined,
        checkSIDs,
      });
      setRRReport(rep);
    } catch (e: unknown) {
      setError(
        e instanceof Error ? `RadioReference check failed: ${e.message}` : "RadioReference check failed",
      );
    } finally {
      setRRBusy(false);
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

      {status?.system?.sites && status.system.sites.length > 0 ? (
        <section className="hunt-sites">
          <h3>Sites ({status.system.sites.length})</h3>
          <table>
            <thead>
              <tr>
                <th>Site</th>
                <th>RFSS</th>
                <th>Control channels</th>
                <th>Neighbors</th>
              </tr>
            </thead>
            <tbody>
              {status.system.sites.map((site, i) => (
                <tr key={`${site.rfss}-${site.site_id}-${i}`}>
                  <td>{site.site_name || site.site_id || "—"}</td>
                  <td>{site.rfss ?? "—"}</td>
                  <td>
                    {site.control_channels && site.control_channels.length > 0
                      ? site.control_channels
                          .map((c) => `${(c.frequency_hz / 1e6).toFixed(4)}${c.is_control ? "*" : ""}`)
                          .join(", ")
                      : "—"}
                  </td>
                  <td>
                    {site.neighbors && site.neighbors.length > 0
                      ? site.neighbors
                          .map((n) => {
                            const id = `${n.rfss ?? 0}/${n.site ?? 0}`;
                            return n.frequency_hz
                              ? `${id} @ ${(n.frequency_hz / 1e6).toFixed(4)}`
                              : id;
                          })
                          .join(", ")
                      : "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      ) : null}

      {status?.reports && status.reports.length > 0 ? (
        <section className="hunt-reports">
          <h3>Captures ({status.reports.length})</h3>
          <table>
            <thead>
              <tr>
                <th>Control freq</th>
                <th>Protocol</th>
                <th>Result</th>
                <th>TGs</th>
              </tr>
            </thead>
            <tbody>
              {status.reports.map((rep, i) => (
                <tr key={`${rep.control_hz}-${i}`}>
                  <td>{rep.control_hz ? `${(rep.control_hz / 1e6).toFixed(4)} MHz` : "—"}</td>
                  <td>{rep.protocol || "—"}</td>
                  <td>{captureResult(rep)}</td>
                  <td>{rep.talkgroups ?? "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      ) : null}

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
        {survey ? (
          <label className="hunt-survey-toggle">
            <input
              type="checkbox"
              checked={persistSurvey}
              onChange={(e) => setPersistSurvey(e.target.checked)}
            />
            Persist survey — stream carriers to a crash-safe NDJSON file
          </label>
        ) : null}
        {survey && persistSurvey ? (
          <label className="hunt-survey-toggle">
            <input type="checkbox" checked={resume} onChange={(e) => setResume(e.target.checked)} />
            Resume — skip frequencies already surveyed in that file
          </label>
        ) : null}
        <label className="hunt-survey-toggle">
          <input type="checkbox" checked={autoGain} onChange={(e) => setAutoGain(e.target.checked)} />
          Auto-gain — after the run, recommend the best front-end gain (needs a dedicated SDR)
        </label>
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

      {status?.system_name ? (
        <section className="hunt-rr">
          <h3>RadioReference</h3>
          <div className="hunt-rr-controls">
            <label>
              County id (ctid)
              <input value={rrCounty} onChange={(e) => setRRCounty(e.target.value)} placeholder="e.g. 1234" />
            </label>
            <label>
              or System id(s)
              <input value={rrSID} onChange={(e) => setRRSID(e.target.value)} placeholder="comma-separated SIDs" />
            </label>
            <button onClick={checkRR} disabled={rrBusy || (!rrCounty.trim() && !rrSID.trim())}>
              {rrBusy ? "Checking…" : "Cross-reference"}
            </button>
          </div>
          {rrReport ? (
            <div className="hunt-rr-result">
              {rrReport.hints && rrReport.hints.length > 0 ? (
                <>
                  <h4>Possible existing systems</h4>
                  <ul>
                    {rrReport.hints.map((h, i) => (
                      <li key={i}>
                        SID {h.sid} — {h.name} ({(h.confidence * 100).toFixed(0)}%): {h.reason}
                      </li>
                    ))}
                  </ul>
                </>
              ) : (
                <p>No existing match among {rrReport.compared} system(s).</p>
              )}
              {rrReport.diff ? (
                <>
                  <h4>
                    Differences vs RadioReference (SID {rrReport.diff.sid}, {rrReport.diff.name})
                  </h4>
                  {rrReport.diff.freq_offsets && rrReport.diff.freq_offsets.length > 0 ? (
                    <div>
                      <strong>Frequency offsets:</strong>
                      <ul>
                        {rrReport.diff.freq_offsets.map((o, i) => (
                          <li key={i}>
                            {(o.discovered_hz / 1e6).toFixed(4)} vs {(o.rr_hz / 1e6).toFixed(4)} MHz (
                            {o.delta_hz > 0 ? "+" : ""}
                            {(o.delta_hz / 1e3).toFixed(2)} kHz)
                          </li>
                        ))}
                      </ul>
                    </div>
                  ) : null}
                  {rrReport.diff.freqs_not_in_rr && rrReport.diff.freqs_not_in_rr.length > 0 ? (
                    <div>
                      <strong>Frequencies not in RadioReference:</strong>{" "}
                      {rrReport.diff.freqs_not_in_rr.map((f) => (f / 1e6).toFixed(4)).join(", ")} MHz
                    </div>
                  ) : null}
                  {rrReport.diff.talkgroups_not_in_rr && rrReport.diff.talkgroups_not_in_rr.length > 0 ? (
                    <div>
                      <strong>Talkgroups not in RadioReference:</strong>{" "}
                      {rrReport.diff.talkgroups_not_in_rr.join(", ")}
                    </div>
                  ) : null}
                </>
              ) : null}
            </div>
          ) : null}
        </section>
      ) : null}

      {status?.gain_recommendations && status.gain_recommendations.length > 0 ? (
        <section className="hunt-gain">
          <h3>Auto-gain recommendations</h3>
          <table>
            <thead>
              <tr>
                <th>Control channel</th>
                <th>Best gain (dB)</th>
                <th>Error rate</th>
                <th>Locked</th>
              </tr>
            </thead>
            <tbody>
              {status.gain_recommendations.map((g, i) => (
                <tr key={i}>
                  <td>{(g.freq_hz / 1e6).toFixed(4)} MHz</td>
                  <td>{(g.best_gain_tenth_db / 10).toFixed(1)}</td>
                  <td>{g.best_error_rate.toFixed(3)}</td>
                  <td>{g.locked ? "yes" : "no"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      ) : null}
      {status?.gain_note ? <p className="hint">Auto-gain: {status.gain_note}</p> : null}

      {!canMutate ? (
        <p className="hint">Mutations are read-only on this connection (no auth token).</p>
      ) : null}
    </div>
  );
}
