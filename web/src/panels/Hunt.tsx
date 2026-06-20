import { useEffect, useState, type ReactNode } from "react";
import { api } from "../api/client";
import { writes } from "../api/write";
import type {
  CaptureReport,
  DetectedSignal,
  HuntRRReport,
  HuntStatus,
} from "../api/types";
import { selectCanMutate, selectClientConfig, useShared } from "../store/shared";
import { PageHeader } from "../components/ui/PageHeader";
import { Card } from "../components/ui/Card";
import { Section } from "../components/ui/Section";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { Field } from "../components/ui/Field";
import { Input } from "../components/ui/Input";
import { Select } from "../components/ui/Select";
import { Checkbox } from "../components/ui/Checkbox";

const POLL_INTERVAL_MS = 2_000;

// CC_PROTOCOLS are the trunked protocols with a dedicated standalone control
// channel worth pointing the parser at. Values are the CLI identifiers the
// hunt API forwards to siglab.ParseProtocolCLI (internal/siglab/config.go).
// P25 Phase 2, DMR Tier II, LTR and D-STAR are excluded — no standalone CC.
const CC_PROTOCOLS: { value: string; label: string }[] = [
  { value: "p25", label: "P25 (Phase 1)" },
  { value: "dmr", label: "DMR (Tier III)" },
  { value: "nxdn", label: "NXDN" },
  { value: "dpmr", label: "dPMR" },
  { value: "edacs", label: "EDACS" },
  { value: "motorola", label: "Motorola Type II / SmartZone" },
  { value: "mpt1327", label: "MPT1327" },
  { value: "tetra", label: "TETRA" },
  { value: "ysf", label: "System Fusion (YSF)" },
];

// Shared table cell classes for the read-only result tables below.
const TH = "px-3 py-2 text-left font-medium";
const TD = "px-3 py-2";
const THEAD = "bg-panel/80 text-xs uppercase tracking-wider text-muted";

function HuntTable({ children }: { children: ReactNode }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">{children}</table>
    </div>
  );
}

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

// hexUpper renders a decoded identity field (WACN / System ID / NAC) as the
// uppercase hex operators copy when programming a radio, or "—" when absent.
function hexUpper(n: number | undefined): string {
  return n ? n.toString(16).toUpperCase() : "—";
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
  const [ccFreq, setCCFreq] = useState(""); // single control-channel freq, MHz
  const [ccProto, setCCProto] = useState("p25"); // trunked protocol to decode
  const [ccDwell, setCCDwell] = useState(15); // listen seconds (see parseCC)
  const [ccMonitorMin, setCCMonitorMin] = useState(0); // streaming long-dwell minutes; 0 = off

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

  // parseCC tunes the hunt engine straight at one trunked control-channel
  // frequency (no sweep) and decodes it for ccDwell seconds, accumulating the
  // band plan / talkgroups / identity into status.system. Keep dwell short:
  // the live IQ source runs at the full SDR rate (~2.4 MHz), so each second of
  // capture is ~19 MB — re-run to discover more rather than dwelling for
  // minutes.
  async function parseCC() {
    const mhz = parseFloat(ccFreq.trim());
    if (Number.isNaN(mhz)) {
      setError("enter a control-channel frequency in MHz (e.g. 851.0125)");
      return;
    }
    try {
      await writes.huntStart(cfg, {
        candidates: [mhz],
        no_sweep: true,
        protocol: ccProto,
        dwell_seconds: ccDwell,
        monitor_seconds: ccMonitorMin > 0 ? ccMonitorMin * 60 : undefined,
      });
    } catch (e: unknown) {
      setError(e instanceof Error ? `parse CC failed: ${e.message}` : "parse CC failed");
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
    <div className="space-y-4 max-w-5xl">
      <PageHeader
        title="Hunt"
        subtitle="Discover an unknown system"
        actions={
          running ? (
            <Badge tone="ok">running ●</Badge>
          ) : (
            <Badge>{status?.state ?? "idle"}</Badge>
          )
        }
      />

      {!canMutate && (
        <div className="rounded-md border border-warn/40 bg-warn/15 px-4 py-2 text-sm text-warn">
          Mutations are read-only on this connection (no auth token) — start a
          daemon with write access to run a hunt.
        </div>
      )}

      {/* Quick start: the most common task, expanded by default. */}
      <Section
        id="hunt-ccparse"
        title="Parse a control channel"
        description="Already know a control-channel frequency? Decode it directly — no sweep."
      >
        <p className="text-xs text-muted">
          The band plan, talkgroups and identity below fill in as the CC is read.
          Each second buffers ~19 MB of IQ, so prefer re-running over long dwells —
          or set <em>Monitor</em> to stream the CC in real time (bounded memory).
        </p>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <Field label="Protocol">
            {(p) => (
              <Select
                {...p}
                className="w-full"
                value={ccProto}
                onChange={(e) => setCCProto(e.target.value)}
              >
                {CC_PROTOCOLS.map((proto) => (
                  <option key={proto.value} value={proto.value}>
                    {proto.label}
                  </option>
                ))}
              </Select>
            )}
          </Field>
          <Field label="CC frequency (MHz)">
            {(p) => (
              <Input
                {...p}
                className="w-full"
                value={ccFreq}
                onChange={(e) => setCCFreq(e.target.value)}
                placeholder="851.0125"
                inputMode="decimal"
              />
            )}
          </Field>
          <Field label="Listen for (seconds)">
            {(p) => (
              <Input
                {...p}
                className="w-full"
                type="number"
                min={1}
                step={1}
                value={ccDwell}
                onChange={(e) => setCCDwell(Number(e.target.value))}
              />
            )}
          </Field>
          <Field
            label="Monitor (minutes, 0 = off)"
            hint="Streams the CC and stops once identity, neighbors and band plan settle."
          >
            {(p) => (
              <Input
                {...p}
                className="w-full"
                type="number"
                min={0}
                step={1}
                value={ccMonitorMin}
                onChange={(e) => setCCMonitorMin(Number(e.target.value))}
              />
            )}
          </Field>
        </div>
        <Button onClick={parseCC} disabled={!canMutate || running}>
          Parse control channel
        </Button>
      </Section>

      {/* Blind sweep: advanced; collapsed by default to keep the panel calm. */}
      <Section
        id="hunt-sweep"
        title="Blind sweep"
        description="Don't know the frequencies? Sweep a band or a candidate list."
        defaultCollapsed
      >
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <Field label="Bands (MHz, low:high, comma-separated)">
            {(p) => (
              <Input
                {...p}
                className="w-full"
                value={bands}
                onChange={(e) => setBands(e.target.value)}
                placeholder="851:869"
              />
            )}
          </Field>
          <Field label="Candidates (MHz, comma-separated — skips the sweep)">
            {(p) => (
              <Input
                {...p}
                className="w-full"
                value={candidates}
                onChange={(e) => setCandidates(e.target.value)}
                placeholder="851.0125, 853.5125"
              />
            )}
          </Field>
          <Field label="Name">
            {(p) => (
              <Input
                {...p}
                className="w-full"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="New County P25"
              />
            )}
          </Field>
          <Field label="State">
            {(p) => (
              <Input
                {...p}
                className="w-full"
                value={stateCode}
                onChange={(e) => setStateCode(e.target.value)}
                placeholder="AZ"
              />
            )}
          </Field>
          <Field label="County">
            {(p) => (
              <Input
                {...p}
                className="w-full"
                value={county}
                onChange={(e) => setCounty(e.target.value)}
                placeholder="Maricopa"
              />
            )}
          </Field>
          <Field label="SDR serial (optional — auto-selects a spare)">
            {(p) => (
              <Input
                {...p}
                className="w-full"
                value={serial}
                onChange={(e) => setSerial(e.target.value)}
                placeholder="00000001"
              />
            )}
          </Field>
          <Field label="Protocol (optional — default auto-identifies)">
            {(p) => (
              <Input
                {...p}
                className="w-full"
                value={protocol}
                onChange={(e) => setProtocol(e.target.value)}
                placeholder="p25"
              />
            )}
          </Field>
        </div>

        <div className="pt-2 space-y-2 border-t border-line">
          <p className="text-xs uppercase tracking-wider text-muted">
            Survey &amp; advanced options
          </p>
          <Checkbox
            label="Survey mode — classify & decode every signal (analog, paging, trunking)"
            checked={survey}
            onChange={(e) => setSurvey(e.target.checked)}
          />
          {survey && (
            <Checkbox
              label="Classify only — skip decoding (fast inventory)"
              checked={classifyOnly}
              onChange={(e) => setClassifyOnly(e.target.checked)}
            />
          )}
          {survey && (
            <Checkbox
              label="Persist survey — stream carriers to a crash-safe NDJSON file"
              checked={persistSurvey}
              onChange={(e) => setPersistSurvey(e.target.checked)}
            />
          )}
          {survey && persistSurvey && (
            <Checkbox
              label="Resume — skip frequencies already surveyed in that file"
              checked={resume}
              onChange={(e) => setResume(e.target.checked)}
            />
          )}
          <Checkbox
            label="Auto-gain — recommend the best front-end gain after the run (needs a dedicated SDR)"
            checked={autoGain}
            onChange={(e) => setAutoGain(e.target.checked)}
          />
        </div>

        <div className="flex gap-2 pt-1">
          <Button onClick={start} disabled={!canMutate || running}>
            Start hunt
          </Button>
          <Button variant="ghost" onClick={stop} disabled={!canMutate || !running}>
            Stop
          </Button>
        </div>
      </Section>

      {/* Live status. */}
      <Card title="Status">
        <div className="space-y-1 text-sm">
          <div>
            State: <strong>{status?.state ?? "idle"}</strong>
            {running ? " ●" : ""}
          </div>
          {status?.phase && (
            <div className="text-muted">
              Phase: {status.phase}
              {status.detail ? ` — ${status.detail}` : ""}
            </div>
          )}
          {status?.error && <div className="text-err">Error: {status.error}</div>}
          {status?.mode && (
            <div>
              Mode: <strong>{status.mode}</strong>
            </div>
          )}
          {status?.system_name ? (
            <div>
              Discovered: <strong>{status.system_name}</strong> — {status.sites}{" "}
              site(s), {status.talkgroups} talkgroup(s)
            </div>
          ) : status?.mode === "survey" ? null : (
            <div className="text-muted">No system discovered yet.</div>
          )}
        </div>
      </Card>

      {status?.system &&
      (status.system.wacn || status.system.system_id || status.system.nac) ? (
        <Card title="System identity">
          <HuntTable>
            <tbody>
              <tr>
                <th className={TH}>Protocol</th>
                <td className={TD}>{status.system.protocol || "—"}</td>
                <th className={TH}>WACN</th>
                <td className={TD}>{hexUpper(status.system.wacn)}</td>
              </tr>
              <tr>
                <th className={TH}>System ID</th>
                <td className={TD}>{hexUpper(status.system.system_id)}</td>
                <th className={TH}>NAC</th>
                <td className={TD}>{hexUpper(status.system.nac)}</td>
              </tr>
              {status.system.confidence ? (
                <tr>
                  <th className={TH}>Confidence</th>
                  <td className={TD} colSpan={3}>
                    {(status.system.confidence * 100).toFixed(0)}%
                  </td>
                </tr>
              ) : null}
            </tbody>
          </HuntTable>
        </Card>
      ) : null}

      {status?.system?.band_plan && status.system.band_plan.length > 0 ? (
        <Card title={`Band plan (${status.system.band_plan.length})`}>
          <HuntTable>
            <thead className={THEAD}>
              <tr>
                <th className={TH}>Channel ID</th>
                <th className={TH}>Base (MHz)</th>
                <th className={TH}>Spacing (kHz)</th>
                <th className={TH}>BW (kHz)</th>
                <th className={TH}>TX offset (MHz)</th>
              </tr>
            </thead>
            <tbody>
              {status.system.band_plan.map((b) => (
                <tr key={b.channel_id} className="border-t border-panel">
                  <td className={TD}>{b.channel_id}</td>
                  <td className={TD}>{(b.base_hz / 1e6).toFixed(5)}</td>
                  <td className={TD}>{(b.spacing_hz / 1e3).toFixed(3)}</td>
                  <td className={TD}>{b.bandwidth_hz ? (b.bandwidth_hz / 1e3).toFixed(1) : "—"}</td>
                  <td className={TD}>{b.tx_offset_hz ? (b.tx_offset_hz / 1e6).toFixed(4) : "—"}</td>
                </tr>
              ))}
            </tbody>
          </HuntTable>
        </Card>
      ) : null}

      {status?.system?.talkgroups && status.system.talkgroups.length > 0 ? (
        <Card title={`Talkgroups (${status.system.talkgroups.length})`}>
          <HuntTable>
            <thead className={THEAD}>
              <tr>
                <th className={TH}>Dec</th>
                <th className={TH}>Hex</th>
                <th className={TH}>Encrypted</th>
                <th className={TH}>Activity</th>
              </tr>
            </thead>
            <tbody>
              {status.system.talkgroups.map((tg) => (
                <tr key={tg.dec} className="border-t border-panel">
                  <td className={TD}>{tg.dec}</td>
                  <td className={TD}>{tg.hex}</td>
                  <td className={TD}>{tg.encrypted ? "🔒" : "—"}</td>
                  <td className={TD}>{tg.count}</td>
                </tr>
              ))}
            </tbody>
          </HuntTable>
        </Card>
      ) : null}

      {status?.system?.sites && status.system.sites.length > 0 ? (
        <Card title={`Sites (${status.system.sites.length})`}>
          <HuntTable>
            <thead className={THEAD}>
              <tr>
                <th className={TH}>Site</th>
                <th className={TH}>RFSS</th>
                <th className={TH}>Control channels</th>
                <th className={TH}>Neighbors</th>
                <th className={TH}>Voice channels</th>
              </tr>
            </thead>
            <tbody>
              {status.system.sites.map((site, i) => (
                <tr key={`${site.rfss}-${site.site_id}-${i}`} className="border-t border-panel">
                  <td className={TD}>{site.site_name || site.site_id || "—"}</td>
                  <td className={TD}>{site.rfss ?? "—"}</td>
                  <td className={TD}>
                    {site.control_channels && site.control_channels.length > 0
                      ? site.control_channels
                          .map((c) => `${(c.frequency_hz / 1e6).toFixed(4)}${c.is_control ? "*" : ""}`)
                          .join(", ")
                      : "—"}
                  </td>
                  <td className={TD}>
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
                  <td className={TD}>
                    {site.voice_channels && site.voice_channels.length > 0
                      ? site.voice_channels.map((h) => (h / 1e6).toFixed(4)).join(", ")
                      : "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </HuntTable>
        </Card>
      ) : null}

      {status?.reports && status.reports.length > 0 ? (
        <Card title={`Captures (${status.reports.length})`}>
          <HuntTable>
            <thead className={THEAD}>
              <tr>
                <th className={TH}>Control freq</th>
                <th className={TH}>Protocol</th>
                <th className={TH}>Result</th>
                <th className={TH}>TGs</th>
              </tr>
            </thead>
            <tbody>
              {status.reports.map((rep, i) => (
                <tr key={`${rep.control_hz}-${i}`} className="border-t border-panel">
                  <td className={TD}>{rep.control_hz ? `${(rep.control_hz / 1e6).toFixed(4)} MHz` : "—"}</td>
                  <td className={TD}>{rep.protocol || "—"}</td>
                  <td className={TD}>{captureResult(rep)}</td>
                  <td className={TD}>{rep.talkgroups ?? "—"}</td>
                </tr>
              ))}
            </tbody>
          </HuntTable>
        </Card>
      ) : null}

      {status?.signals && status.signals.length > 0 ? (
        <Card title={`Signals (${status.signals.length})`}>
          <HuntTable>
            <thead className={THEAD}>
              <tr>
                <th className={`${TH} cursor-pointer`} onClick={() => setSortBy("freq")}>
                  Frequency{sortBy === "freq" ? " ▾" : ""}
                </th>
                <th className={`${TH} cursor-pointer`} onClick={() => setSortBy("class")}>
                  Class{sortBy === "class" ? " ▾" : ""}
                </th>
                <th className={TH}>BW (kHz)</th>
                <th className={`${TH} cursor-pointer`} onClick={() => setSortBy("snr")}>
                  SNR (dB){sortBy === "snr" ? " ▾" : ""}
                </th>
                <th className={TH}>Decode</th>
              </tr>
            </thead>
            <tbody>
              {sortSignals(status.signals, sortBy).map((sig) => (
                <tr key={sig.freq_hz} className="border-t border-panel">
                  <td className={TD}>{(sig.freq_hz / 1e6).toFixed(4)} MHz</td>
                  <td className={TD}>{sig.class}</td>
                  <td className={TD}>{(sig.occupied_bw_hz / 1e3).toFixed(1)}</td>
                  <td className={TD}>{sig.snr_db.toFixed(1)}</td>
                  <td className={TD}>{signalDetail(sig)}</td>
                </tr>
              ))}
            </tbody>
          </HuntTable>
        </Card>
      ) : null}

      {(status?.system_name || (status?.signals && status.signals.length > 0)) && (
        <Card title="Export">
          <div className="flex flex-wrap gap-2">
            {status?.system_name && (
              <>
                <a className="btn-ghost text-xs" href={`${exportBase}?format=bundle`}>
                  GopherTrunk bundle
                </a>
                <a className="btn-ghost text-xs" href={`${exportBase}?format=trunk-recorder`}>
                  trunk-recorder
                </a>
                <a className="btn-ghost text-xs" href={`${exportBase}?format=rr`}>
                  RadioReference package
                </a>
              </>
            )}
            {status?.signals && status.signals.length > 0 && (
              <>
                <a className="btn-ghost text-xs" href={`${cfg.baseURL}/api/v1/hunt/survey?format=json`}>
                  survey JSON
                </a>
                <a className="btn-ghost text-xs" href={`${cfg.baseURL}/api/v1/hunt/survey?format=csv`}>
                  survey CSV
                </a>
              </>
            )}
          </div>
        </Card>
      )}

      {status?.system_name ? (
        <Section
          id="hunt-rr"
          title="Cross-reference RadioReference"
          description="Compare the discovered system against RadioReference by county or system id."
          defaultCollapsed
        >
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Field label="County id (ctid)">
              {(p) => (
                <Input
                  {...p}
                  className="w-full"
                  value={rrCounty}
                  onChange={(e) => setRRCounty(e.target.value)}
                  placeholder="e.g. 1234"
                />
              )}
            </Field>
            <Field label="or System id(s)">
              {(p) => (
                <Input
                  {...p}
                  className="w-full"
                  value={rrSID}
                  onChange={(e) => setRRSID(e.target.value)}
                  placeholder="comma-separated SIDs"
                />
              )}
            </Field>
          </div>
          <Button
            loading={rrBusy}
            onClick={checkRR}
            disabled={rrBusy || (!rrCounty.trim() && !rrSID.trim())}
          >
            Cross-reference
          </Button>
          {rrReport && (
            <div className="text-sm space-y-2">
              {rrReport.hints && rrReport.hints.length > 0 ? (
                <>
                  <h4 className="font-medium">Possible existing systems</h4>
                  <ul className="list-disc pl-5 space-y-1">
                    {rrReport.hints.map((h, i) => (
                      <li key={i}>
                        SID {h.sid} — {h.name} ({(h.confidence * 100).toFixed(0)}%): {h.reason}
                      </li>
                    ))}
                  </ul>
                </>
              ) : (
                <p className="text-muted">No existing match among {rrReport.compared} system(s).</p>
              )}
              {rrReport.diff && (
                <>
                  <h4 className="font-medium">
                    Differences vs RadioReference (SID {rrReport.diff.sid}, {rrReport.diff.name})
                  </h4>
                  {rrReport.diff.freq_offsets && rrReport.diff.freq_offsets.length > 0 && (
                    <div>
                      <strong>Frequency offsets:</strong>
                      <ul className="list-disc pl-5">
                        {rrReport.diff.freq_offsets.map((o, i) => (
                          <li key={i}>
                            {(o.discovered_hz / 1e6).toFixed(4)} vs {(o.rr_hz / 1e6).toFixed(4)} MHz (
                            {o.delta_hz > 0 ? "+" : ""}
                            {(o.delta_hz / 1e3).toFixed(2)} kHz)
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}
                  {rrReport.diff.freqs_not_in_rr && rrReport.diff.freqs_not_in_rr.length > 0 && (
                    <div>
                      <strong>Frequencies not in RadioReference:</strong>{" "}
                      {rrReport.diff.freqs_not_in_rr.map((f) => (f / 1e6).toFixed(4)).join(", ")} MHz
                    </div>
                  )}
                  {rrReport.diff.talkgroups_not_in_rr && rrReport.diff.talkgroups_not_in_rr.length > 0 && (
                    <div>
                      <strong>Talkgroups not in RadioReference:</strong>{" "}
                      {rrReport.diff.talkgroups_not_in_rr.join(", ")}
                    </div>
                  )}
                </>
              )}
            </div>
          )}
        </Section>
      ) : null}

      {status?.gain_recommendations && status.gain_recommendations.length > 0 ? (
        <Card title="Auto-gain recommendations">
          <HuntTable>
            <thead className={THEAD}>
              <tr>
                <th className={TH}>Control channel</th>
                <th className={TH}>Best gain (dB)</th>
                <th className={TH}>Error rate</th>
                <th className={TH}>Locked</th>
              </tr>
            </thead>
            <tbody>
              {status.gain_recommendations.map((g, i) => (
                <tr key={i} className="border-t border-panel">
                  <td className={TD}>{(g.freq_hz / 1e6).toFixed(4)} MHz</td>
                  <td className={TD}>{(g.best_gain_tenth_db / 10).toFixed(1)}</td>
                  <td className={TD}>{g.best_error_rate.toFixed(3)}</td>
                  <td className={TD}>{g.locked ? "yes" : "no"}</td>
                </tr>
              ))}
            </tbody>
          </HuntTable>
          {status.gain_note && (
            <p className="text-xs text-muted mt-2">Auto-gain: {status.gain_note}</p>
          )}
        </Card>
      ) : null}
    </div>
  );
}
