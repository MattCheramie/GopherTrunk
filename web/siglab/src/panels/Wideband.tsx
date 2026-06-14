import { useState } from "react";
import { useStore } from "../store/shared";
import { WidebandSurvey } from "../viz/WidebandSurvey";
import type { WidebandResult } from "../api/types";

// Wideband surveys a multi-carrier capture: it splits the band into carriers,
// identifies each, and rolls them into trunked-system verdicts (siglab
// SurveyWideband). center_hz is the absolute tuner center of the grab.
export function Wideband() {
  const captures = useStore((s) => s.captures);
  const wideband = useStore((s) => s.wideband);
  const [captureId, setCaptureId] = useState("");
  const [centerMHz, setCenterMHz] = useState("");
  const [result, setResult] = useState<WidebandResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function onSurvey() {
    if (!captureId) return;
    setBusy(true);
    setErr(null);
    try {
      const centerHz = centerMHz ? Math.round(parseFloat(centerMHz) * 1e6) : undefined;
      setResult(await wideband(captureId, { center_hz: centerHz }));
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-4xl space-y-4">
      <div className="card flex flex-wrap items-end gap-3">
        <div className="flex-1">
          <label className="label">Capture</label>
          <select
            className="input"
            value={captureId}
            onChange={(e) => setCaptureId(e.target.value)}
          >
            <option value="">— select —</option>
            {captures.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="label">Center (MHz)</label>
          <input
            className="input w-32"
            value={centerMHz}
            placeholder="441.0"
            onChange={(e) => setCenterMHz(e.target.value)}
          />
        </div>
        <button className="btn" disabled={busy || !captureId} onClick={onSurvey}>
          {busy ? "Surveying…" : "Survey"}
        </button>
      </div>
      {err && <div className="card text-xs text-err">{err}</div>}

      {result && <WidebandSurvey result={result} />}
    </div>
  );
}
