import { useEffect, useState } from "react";
import { api } from "../api/client";
import { writes } from "../api/write";
import type { HuntStatus } from "../api/types";
import { selectCanMutate, selectClientConfig, useShared } from "../store/shared";

const POLL_INTERVAL_MS = 2_000;

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
        name: name || undefined,
        state: stateCode || undefined,
        county: county || undefined,
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
        {status?.system_name ? (
          <div>
            Discovered: <strong>{status.system_name}</strong> — {status.sites} site(s),{" "}
            {status.talkgroups} talkgroup(s)
          </div>
        ) : (
          <div>No system discovered yet.</div>
        )}
      </section>

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

      {!canMutate ? (
        <p className="hint">Mutations are read-only on this connection (no auth token).</p>
      ) : null}
    </div>
  );
}
