import { useEffect, useState } from "react";
import { Section } from "../components/Section";
import {
  Fieldset,
  FreqListField,
  HzField,
  NumberField,
  SelectField,
  TextField,
} from "../components/fields";
import { ListEditor } from "../components/ListEditor";
import { AdvancedJSON } from "../components/AdvancedJSON";
import { RRCredentials } from "../components/RRCredentials";
import { dmrBandPlanForMode, dmrBandPlanMode } from "../lib/dmrBandPlan";
import { useStore } from "../store/shared";
import { api, HTTPError } from "../api/client";
import type {
  DMRBandPlanTableEntry,
  EncryptedCallsConfig,
  EncryptionKey,
  P25BandPlanEntry,
  ParsedSystemDTO,
  RRGeoRef,
  RRSearchHit,
  SystemConfig,
  TalkgroupCSVRow,
  TrunkingConfig,
} from "../api/types";

const PROTOCOLS = [
  "p25", "p25-phase2", "dmr", "dmr-tier2", "dmr-tier1", "nxdn", "dpmr",
  "edacs", "motorola", "ltr", "mpt1327", "tetra", "ysf", "dstar",
];

// Long-tail protocol decoder knobs surfaced via AdvancedJSON. None are
// enforced by config.Validate, so a free JSON editor is lossless.
const PROTOCOL_KNOBS: (keyof SystemConfig)[] = [
  "TETRAColourCode", "TETRAChannel", "TETRAChannelCoding", "TETRAClockMode",
  "LTRFCSMode", "LTRManchesterMode", "P25Phase1DemodMode", "DMRInterleavedVoice",
  "P25Phase2TrellisMode", "P25Phase2RSMode", "P25Phase2InterleaveMode",
  "P25Phase2ScramblerMode", "P25Phase2SoftDecision", "P25Phase2ClockMode", "NXDNViterbiMode",
  "NXDNDeviationHz", "EDACSBCHMode", "MPT1327BCHMode", "MPT1327CWSCTolerance",
  "MotorolaBCHMode", "DStarFECMode",
];

function slug(name: string): string {
  return (name || "system").toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "");
}

export function TrunkingSection() {
  const cfg = useStore((s) => s.config?.Trunking) as TrunkingConfig;
  const patch = useStore((s) => s.patchSection);
  const stageTalkgroups = useStore((s) => s.stageTalkgroups);
  const set = (v: TrunkingConfig) => patch("Trunking", v);

  const [rrOpen, setRROpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);

  const systems = cfg.Systems ?? [];
  const setSystem = (i: number, sys: SystemConfig) => {
    const next = systems.slice();
    next[i] = sys;
    set({ ...cfg, Systems: next });
  };
  const removeSystem = (i: number) => set({ ...cfg, Systems: systems.filter((_, k) => k !== i) });
  const addSystem = (sys: SystemConfig, tgs?: TalkgroupCSVRow[]) => {
    if (tgs && tgs.length) {
      const rel = `${slug(sys.Name)}-talkgroups.csv`;
      sys = { ...sys, TalkgroupFile: rel };
      stageTalkgroups(rel, tgs);
    }
    set({ ...cfg, Systems: [...systems, sys] });
  };
  const addBlank = () =>
    addSystem({ Name: "", Protocol: "p25", ControlChannels: [], TalkgroupFile: "" });

  return (
    <Section
      sectionKey="trunking"
      title="Trunking Systems"
    >
      <div className="grid gap-3 sm:grid-cols-2">
        <NumberField
          label="Call timeout (ms)"
          value={cfg.CallTimeoutMs}
          onChange={(x) => set({ ...cfg, CallTimeoutMs: x })}
          placeholder="30000"
          help="Inactivity window before the engine ends a call."
        />
        <NumberField
          label="Voice hangtime (ms)"
          value={cfg.VoiceHangtimeMs}
          onChange={(x) => set({ ...cfg, VoiceHangtimeMs: x })}
          placeholder="3500"
          help="End-of-transmission window applied to every voice protocol: the composer ends a call this long after the last decoded voice frame."
        />
      </div>

      <SelectField
        label="Voice call grouping"
        value={cfg.VoiceCallGrouping || "transmission"}
        onChange={(x) => set({ ...cfg, VoiceCallGrouping: x })}
        options={[
          { value: "transmission", label: "transmission — one file per over/PTT" },
          { value: "conversation", label: "conversation — group consecutive overs of a talkgroup" },
        ]}
        help="How voice recordings are split, for every voice protocol."
      />

      <div className="flex flex-wrap gap-2">
        <button className="btn-ghost" onClick={addBlank}>+ Add system</button>
        <button className="btn" onClick={() => setRROpen(true)}>Add from RadioReference</button>
        <button className="btn-ghost" onClick={() => setImportOpen(true)}>Import PDF / CSV</button>
      </div>

      {systems.length === 0 ? (
        <p className="help">No systems yet.</p>
      ) : null}

      {systems.map((sys, i) => (
        <div key={i} className="rounded-md border border-line p-3 space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">{sys.Name || `System ${i + 1}`}</span>
            <button className="btn-danger" onClick={() => removeSystem(i)}>Remove</button>
          </div>
          <SystemEditor sys={sys} onChange={(next) => setSystem(i, next)} />
        </div>
      ))}

      {rrOpen ? <RRBrowseModal onClose={() => setRROpen(false)} onAdd={addSystem} /> : null}
      {importOpen ? <ImportModal onClose={() => setImportOpen(false)} onAdd={addSystem} /> : null}
    </Section>
  );
}

// EncryptedCallsField edits a system's encrypted_calls policy (issue
// #711): how scarce voice SDRs are allocated to encrypted calls on this
// system. The metadata-follow window is only relevant in "metadata" mode.
function EncryptedCallsField(props: {
  value: EncryptedCallsConfig | undefined;
  onChange: (next: EncryptedCallsConfig) => void;
}) {
  const enc: EncryptedCallsConfig = props.value ?? { Mode: "", MetadataFollowMs: 0 };
  const set = (patch: Partial<EncryptedCallsConfig>) => props.onChange({ ...enc, ...patch });
  const mode = enc.Mode || "follow";
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      <SelectField
        label="Encrypted call handling"
        value={mode}
        onChange={(x) => set({ Mode: x })}
        options={[
          { value: "follow", label: "follow — hold a voice SDR for the whole call" },
          { value: "metadata", label: "metadata — follow briefly, then release" },
          { value: "ignore", label: "ignore — never tie up a voice SDR" },
        ]}
        help="How encrypted calls use scarce voice SDRs. Calls you have a matching encryption key configured for are always followed."
      />
      {mode === "metadata" ? (
        <NumberField
          label="Metadata follow (ms)"
          value={enc.MetadataFollowMs}
          onChange={(x) => set({ MetadataFollowMs: x })}
          placeholder="1500"
          help="How long to follow an encrypted call (capturing talker alias / RID) before releasing the voice SDR — or until the talker alias completes, whichever comes first. 0 = default (1500 ms)."
        />
      ) : null}
    </div>
  );
}

// SystemEditor renders all fields of one trunking system: the common
// fields inline, plus advanced fields (band plans, encryption keys,
// protocol knobs) grouped in collapsible Fieldsets.
function SystemEditor(props: { sys: SystemConfig; onChange: (next: SystemConfig) => void }) {
  const { sys, onChange } = props;
  const dmrMode = dmrBandPlanMode(sys.DMRBandPlan);

  return (
    <div className="space-y-3">
      <div className="grid gap-3 sm:grid-cols-2">
        <TextField label="Name" value={sys.Name} onChange={(x) => onChange({ ...sys, Name: x })} />
        <SelectField
          label="Protocol"
          value={sys.Protocol}
          onChange={(x) => onChange({ ...sys, Protocol: x })}
          options={PROTOCOLS.map((p) => ({ value: p, label: p }))}
        />
      </div>
      <FreqListField
        label="Control channels"
        value={sys.ControlChannels}
        onChange={(x) => onChange({ ...sys, ControlChannels: x })}
      />
      <div className="grid gap-3 sm:grid-cols-2">
        <TextField
          label="Talkgroup file"
          value={sys.TalkgroupFile}
          onChange={(x) => onChange({ ...sys, TalkgroupFile: x })}
          placeholder="(optional) talkgroups.csv"
          help="CSV/JSON of talkgroup aliases, relative to the config file."
        />
        <TextField
          label="RID alias file"
          value={sys.RIDAliasFile ?? ""}
          onChange={(x) => onChange({ ...sys, RIDAliasFile: x })}
          placeholder="(optional) rids.csv"
          help="CSV/JSON of radio-ID aliases, relative to the config file."
        />
      </div>

      <Fieldset legend="P25 band plan (manual IDEN_UP override)">
        <ListEditor<P25BandPlanEntry>
          label="Channels"
          items={sys.P25BandPlan}
          onChange={(x) => onChange({ ...sys, P25BandPlan: x })}
          makeNew={() => ({ ChannelID: 0, BaseHz: 0, SpacingHz: 0, TxOffsetHz: 0, BandwidthHz: 0 })}
          itemTitle={(e) => `Channel ID ${e.ChannelID}`}
          emptyHint="Only needed when a site never broadcasts IDEN_UP for a channel id."
          renderItem={(e, set) => (
            <div className="grid gap-3 sm:grid-cols-2">
              <NumberField label="Channel ID (0–15)" value={e.ChannelID} onChange={(v) => set({ ...e, ChannelID: v })} />
              <HzField label="Base freq" value={e.BaseHz} onChange={(v) => set({ ...e, BaseHz: v })} />
              <HzField label="Spacing" value={e.SpacingHz} onChange={(v) => set({ ...e, SpacingHz: v })} />
              <NumberField label="Tx offset (Hz, signed)" value={e.TxOffsetHz} onChange={(v) => set({ ...e, TxOffsetHz: v })} />
              <HzField label="Bandwidth" value={e.BandwidthHz} onChange={(v) => set({ ...e, BandwidthHz: v })} />
            </div>
          )}
        />
      </Fieldset>

      <Fieldset legend="DMR band plan (required for DMR Tier III voice)">
        <SelectField
          label="Mode"
          value={dmrMode}
          onChange={(m) =>
            onChange({ ...sys, DMRBandPlan: dmrBandPlanForMode(m as "none" | "linear" | "table", sys.DMRBandPlan) })
          }
          options={[
            { value: "none", label: "none" },
            { value: "linear", label: "linear (regular grid)" },
            { value: "table", label: "table (explicit LCN→freq)" },
          ]}
        />
        {dmrMode === "linear" && sys.DMRBandPlan?.Linear ? (
          <div className="grid gap-3 sm:grid-cols-3">
            <HzField
              label="Base freq"
              value={sys.DMRBandPlan.Linear.BaseHz}
              onChange={(v) =>
                onChange({ ...sys, DMRBandPlan: { Linear: { ...sys.DMRBandPlan!.Linear!, BaseHz: v }, Table: null } })
              }
            />
            <HzField
              label="Spacing"
              value={sys.DMRBandPlan.Linear.SpacingHz}
              onChange={(v) =>
                onChange({ ...sys, DMRBandPlan: { Linear: { ...sys.DMRBandPlan!.Linear!, SpacingHz: v }, Table: null } })
              }
            />
            <NumberField
              label="Offset"
              value={sys.DMRBandPlan.Linear.Offset}
              onChange={(v) =>
                onChange({ ...sys, DMRBandPlan: { Linear: { ...sys.DMRBandPlan!.Linear!, Offset: v }, Table: null } })
              }
            />
          </div>
        ) : null}
        {dmrMode === "table" ? (
          <ListEditor<DMRBandPlanTableEntry>
            label="LCN → frequency"
            items={sys.DMRBandPlan?.Table}
            onChange={(x) => onChange({ ...sys, DMRBandPlan: { Linear: null, Table: x } })}
            makeNew={() => ({ LCN: 0, FreqHz: 0 })}
            itemTitle={(e) => `LCN ${e.LCN}`}
            renderItem={(e, set) => (
              <div className="grid gap-3 sm:grid-cols-2">
                <NumberField label="LCN" value={e.LCN} onChange={(v) => set({ ...e, LCN: v })} />
                <HzField label="Frequency" value={e.FreqHz} onChange={(v) => set({ ...e, FreqHz: v })} />
              </div>
            )}
          />
        ) : null}
      </Fieldset>

      <Fieldset legend="Encryption keys">
        <ListEditor<EncryptionKey>
          label="Keys"
          items={sys.EncryptionKeys}
          onChange={(x) => onChange({ ...sys, EncryptionKeys: x })}
          makeNew={() => ({ KeyID: 0, Algorithm: "rc4", Key: "" })}
          itemTitle={(e) => `Key ID ${e.KeyID}`}
          emptyHint="Operator-supplied decryption keys (DMR RC4 / Enhanced Privacy)."
          renderItem={(e, set) => (
            <div className="grid gap-3 sm:grid-cols-3">
              <NumberField label="Key ID" value={e.KeyID} onChange={(v) => set({ ...e, KeyID: v })} />
              <SelectField
                label="Algorithm"
                value={e.Algorithm}
                onChange={(v) => set({ ...e, Algorithm: v })}
                options={[{ value: "rc4", label: "rc4 (DMR Enhanced Privacy)" }]}
              />
              <TextField label="Key (hex)" value={e.Key} onChange={(v) => set({ ...e, Key: v })} />
            </div>
          )}
        />
      </Fieldset>

      <Fieldset legend="Encrypted call handling">
        <EncryptedCallsField
          value={sys.EncryptedCalls}
          onChange={(x) => onChange({ ...sys, EncryptedCalls: x })}
        />
      </Fieldset>

      <TalkgroupsField sys={sys} onChange={onChange} />

      <AdvancedJSON<SystemConfig>
        label="Advanced protocol knobs (JSON)"
        value={sys}
        onChange={onChange}
        pick={PROTOCOL_KNOBS}
        help="Protocol-specific decoder settings (TETRA/LTR/P25 Phase 1+2/NXDN/EDACS/MPT1327/Motorola/D-STAR). See the Trunking docs for accepted values; only set the keys you need."
      />
    </div>
  );
}

// TalkgroupsField shows the per-system talkgroup count and opens an editor
// modal. Rows are staged in the store keyed by the system's TalkgroupFile
// (defaulted from the name) and written as a CSV sidecar on save.
function TalkgroupsField(props: { sys: SystemConfig; onChange: (next: SystemConfig) => void }) {
  const { sys, onChange } = props;
  const talkgroups = useStore((s) => s.talkgroups);
  const stage = useStore((s) => s.stageTalkgroups);
  const [open, setOpen] = useState(false);
  const rel = sys.TalkgroupFile || `${slug(sys.Name)}-talkgroups.csv`;
  const rows = talkgroups[rel] ?? [];
  return (
    <Fieldset legend={`Talkgroups (${rows.length})`}>
      <p className="help">Alias list written to the sidecar CSV <code>{rel}</code>.</p>
      <button className="btn-ghost" onClick={() => setOpen(true)}>
        Edit talkgroups
      </button>
      {open ? (
        <TalkgroupModal
          rel={rel}
          rows={rows}
          onClose={() => setOpen(false)}
          onChange={(next) => {
            stage(rel, next);
            if (!sys.TalkgroupFile) onChange({ ...sys, TalkgroupFile: rel });
          }}
        />
      ) : null}
    </Fieldset>
  );
}

function tgMatches(r: TalkgroupCSVRow, f: string): boolean {
  if (!f) return true;
  return (
    String(r.decimal).includes(f) ||
    (r.alpha_tag ?? "").toLowerCase().includes(f) ||
    (r.tag ?? "").toLowerCase().includes(f) ||
    (r.description ?? "").toLowerCase().includes(f)
  );
}

function TalkgroupModal(props: {
  rel: string;
  rows: TalkgroupCSVRow[];
  onClose: () => void;
  onChange: (rows: TalkgroupCSVRow[]) => void;
}) {
  const { rows, onChange } = props;
  const [filter, setFilter] = useState("");
  const f = filter.trim().toLowerCase();
  const set = (i: number, row: TalkgroupCSVRow) => {
    const next = rows.slice();
    next[i] = row;
    onChange(next);
  };
  const remove = (i: number) => onChange(rows.filter((_, k) => k !== i));
  const add = () => onChange([{ decimal: 0 }, ...rows]);

  return (
    <Modal title={`Talkgroups — ${props.rel}`} onClose={props.onClose}>
      <div className="flex items-center gap-2">
        <input
          className="input"
          placeholder="filter by decimal / alpha tag / tag / description"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        />
        <button className="btn" onClick={add}>
          + Add
        </button>
      </div>
      <p className="help">
        {rows.length} talkgroup{rows.length === 1 ? "" : "s"}
        {f ? ` (filtered)` : ""}.
      </p>
      <div className="max-h-96 space-y-1 overflow-y-auto">
        {rows.map((r, i) =>
          tgMatches(r, f) ? (
            <div key={i} className="grid grid-cols-12 items-center gap-1 text-sm">
              <input
                className="input col-span-2"
                type="number"
                value={r.decimal}
                onChange={(e) => set(i, { ...r, decimal: Number(e.target.value) })}
                placeholder="dec"
              />
              <input
                className="input col-span-3"
                value={r.alpha_tag ?? ""}
                onChange={(e) => set(i, { ...r, alpha_tag: e.target.value })}
                placeholder="alpha tag"
              />
              <input
                className="input col-span-3"
                value={r.description ?? ""}
                onChange={(e) => set(i, { ...r, description: e.target.value })}
                placeholder="description"
              />
              <input
                className="input col-span-2"
                value={r.tag ?? ""}
                onChange={(e) => set(i, { ...r, tag: e.target.value })}
                placeholder="tag"
              />
              <input
                className="input col-span-1"
                value={r.mode ?? ""}
                onChange={(e) => set(i, { ...r, mode: e.target.value })}
                placeholder="mode"
              />
              <button className="btn-danger col-span-1" onClick={() => remove(i)}>
                ✕
              </button>
            </div>
          ) : null,
        )}
      </div>
    </Modal>
  );
}

function Modal(props: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center bg-black/40 p-4 overflow-y-auto">
      <div className="card w-full max-w-2xl space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-base font-semibold">{props.title}</h3>
          <button className="btn-ghost" onClick={props.onClose}>Close</button>
        </div>
        {props.children}
      </div>
    </div>
  );
}

function RRBrowseModal(props: {
  onClose: () => void;
  onAdd: (sys: SystemConfig, tgs?: TalkgroupCSVRow[]) => void;
}) {
  const setError = useStore((s) => s.setError);
  // The RadioReference login the user is editing — sent with every RR call so
  // their premium login is what reaches the API (the SOAP app key is built into
  // the binary, so only username/password travel from here).
  const rr = useStore((s) => s.config?.RadioReference);
  const creds = { user: rr?.Username, pass: rr?.Password };
  const hasLogin = !!(rr?.Username?.trim() && rr?.Password?.trim());
  // The credentials block auto-expands when no login is set (or after a 503),
  // so "Add from RadioReference" asks for a login in place instead of throwing
  // the bare "credentials not configured" error.
  const [credsOpen, setCredsOpen] = useState(!hasLogin);
  const [credsHint, setCredsHint] = useState<string | null>(null);
  // handleRRError turns a "not configured" 503 into an inline prompt for the
  // login (expanding the block) instead of a red toast; everything else still
  // surfaces as a toast.
  const handleRRError = (e: unknown) => {
    if (e instanceof HTTPError && e.status === 503) {
      setCredsOpen(true);
      setCredsHint("Sign in with your RadioReference.com username and password to browse systems.");
      return;
    }
    setError(`RadioReference: ${(e as Error).message}`);
  };
  const [mode, setMode] = useState<"name" | "zip" | "advanced">("name");

  // name mode: state → county dropdowns.
  const [states, setStates] = useState<RRGeoRef[] | null>(null);
  const [counties, setCounties] = useState<RRGeoRef[] | null>(null);
  const [stid, setStid] = useState("");
  const [ctid, setCtid] = useState("");

  // zip + advanced modes.
  const [zip, setZip] = useState("");
  const [kind, setKind] = useState<"county" | "state">("county");
  const [value, setValue] = useState("");

  const [hits, setHits] = useState<RRSearchHit[] | null>(null);
  const [busy, setBusy] = useState(false);

  // Lazy-load the state list when name mode is first shown.
  useEffect(() => {
    if (mode !== "name" || states !== null) return;
    setBusy(true);
    api
      .rrStates(creds)
      .then((r) => setStates(r.results ?? []))
      .catch(handleRRError)
      .finally(() => setBusy(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode, states]);

  const onStateChange = async (v: string) => {
    setStid(v);
    setCtid("");
    setCounties(null);
    setHits(null);
    if (!v) return;
    setBusy(true);
    try {
      const r = await api.rrCounties(Number(v), creds);
      setCounties(r.results ?? []);
    } catch (e) {
      handleRRError(e);
    } finally {
      setBusy(false);
    }
  };

  const runSearch = async (fn: () => Promise<{ results: RRSearchHit[] | null }>) => {
    setBusy(true);
    setHits(null);
    try {
      const r = await fn();
      setHits(r.results ?? []);
    } catch (e) {
      handleRRError(e);
    } finally {
      setBusy(false);
    }
  };

  const importSystem = async (sid: number) => {
    setBusy(true);
    try {
      const r = await api.rrSystem(sid, creds);
      props.onAdd(r.config, r.talkgroups ?? []);
      props.onClose();
    } catch (e) {
      handleRRError(e);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal title="Browse RadioReference.com" onClose={props.onClose}>
      <p className="help">
        Find a trunked system and import it with its control channels and
        talkgroups. Sign in with your RadioReference.com account below — a
        premium subscription is required.
      </p>

      <div className="rounded-md border border-line p-3">
        <button
          className="flex w-full items-center justify-between text-left text-sm font-medium"
          onClick={() => setCredsOpen((o) => !o)}
        >
          <span>
            RadioReference login{hasLogin ? "" : " — required"}
            {hasLogin ? <span className="ml-2 text-xs text-ok">✓ set</span> : null}
          </span>
          <span className="text-muted">{credsOpen ? "▾" : "▸"}</span>
        </button>
        {credsHint ? <p className="mt-2 text-sm text-warn">{credsHint}</p> : null}
        {credsOpen ? (
          <div className="mt-3">
            <RRCredentials />
          </div>
        ) : null}
      </div>

      <div className="flex gap-2 text-sm">
        {(["name", "zip", "advanced"] as const).map((m) => (
          <button
            key={m}
            className={mode === m ? "btn" : "btn-ghost"}
            onClick={() => {
              setMode(m);
              setHits(null);
            }}
          >
            {m === "name" ? "By state/county" : m === "zip" ? "By ZIP" : "By ID"}
          </button>
        ))}
      </div>

      {mode === "name" ? (
        <div className="flex flex-wrap gap-2">
          <select className="input w-44" value={stid} onChange={(e) => onStateChange(e.target.value)}>
            <option value="">Select state…</option>
            {(states ?? []).map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </select>
          <select
            className="input w-48"
            value={ctid}
            disabled={!counties}
            onChange={(e) => {
              setCtid(e.target.value);
              if (e.target.value) runSearch(() => api.rrSearch("county", e.target.value, creds));
            }}
          >
            <option value="">Select county…</option>
            {(counties ?? []).map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
          {stid ? (
            <button className="btn-ghost" disabled={busy} onClick={() => runSearch(() => api.rrSearch("state", stid, creds))}>
              All systems in state
            </button>
          ) : null}
        </div>
      ) : null}

      {mode === "zip" ? (
        <div className="flex gap-2">
          <input
            className="input"
            value={zip}
            placeholder="78701"
            onChange={(e) => setZip(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && runSearch(() => api.rrSearch("zip", zip.trim(), creds))}
          />
          <button className="btn" disabled={busy || !zip.trim()} onClick={() => runSearch(() => api.rrSearch("zip", zip.trim(), creds))}>
            Search
          </button>
        </div>
      ) : null}

      {mode === "advanced" ? (
        <div className="flex gap-2">
          <select className="input w-32" value={kind} onChange={(e) => setKind(e.target.value as "county" | "state")}>
            <option value="county">County id</option>
            <option value="state">State id</option>
          </select>
          <input
            className="input"
            value={value}
            placeholder="numeric ctid / stid"
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && runSearch(() => api.rrSearch(kind, value.trim(), creds))}
          />
          <button className="btn" disabled={busy || !value.trim()} onClick={() => runSearch(() => api.rrSearch(kind, value.trim(), creds))}>
            Search
          </button>
        </div>
      ) : null}

      {busy ? <p className="help">Working…</p> : null}
      {hits ? (
        hits.length === 0 ? (
          <p className="help">No systems found.</p>
        ) : (
          <div className="max-h-80 space-y-1 overflow-y-auto">
            {hits.map((h) => (
              <div key={h.sid} className="flex items-center justify-between rounded border border-line px-3 py-2">
                <div className="text-sm">
                  <div className="font-medium">{h.name}</div>
                  <div className="help">{h.type} · sid {h.sid}</div>
                </div>
                <button className="btn-ghost" disabled={busy} onClick={() => importSystem(h.sid)}>
                  Import
                </button>
              </div>
            ))}
          </div>
        )
      ) : null}
    </Modal>
  );
}

function ImportModal(props: {
  onClose: () => void;
  onAdd: (sys: SystemConfig, tgs?: TalkgroupCSVRow[]) => void;
}) {
  const setError = useStore((s) => s.setError);
  const [parsed, setParsed] = useState<ParsedSystemDTO[] | null>(null);
  const [busy, setBusy] = useState(false);

  const onFiles = async (files: FileList | null) => {
    if (!files || files.length === 0) return;
    setBusy(true);
    try {
      const r = await api.parse(Array.from(files));
      setParsed(r.systems ?? []);
    } catch (e) {
      setError(`Import parse: ${(e as Error).message}`);
    } finally {
      setBusy(false);
    }
  };

  const addParsed = (p: ParsedSystemDTO) => {
    const tgs: TalkgroupCSVRow[] = (p.talkgroups ?? []).map((t) => ({
      decimal: t.dec,
      alpha_tag: t.alpha_tag,
      description: t.description,
      tag: t.tag,
      group: t.group,
      mode: t.mode,
    }));
    props.onAdd(
      {
        Name: p.name,
        Protocol: p.protocol || "p25",
        ControlChannels: p.control_channels ?? [],
        TalkgroupFile: "",
      },
      tgs,
    );
  };

  return (
    <Modal title="Import from PDF / CSV" onClose={props.onClose}>
      <p className="help">
        Upload one or more RadioReference PDF exports or CSV bundles. They are
        parsed on the server and added to your draft (nothing is written to disk
        until you save).
      </p>
      <input
        type="file"
        multiple
        accept=".pdf,.csv"
        className="input"
        onChange={(e) => onFiles(e.target.files)}
      />
      {busy ? <p className="help">Parsing…</p> : null}
      {parsed ? (
        parsed.length === 0 ? (
          <p className="help">Nothing parsed.</p>
        ) : (
          <div className="max-h-80 space-y-1 overflow-y-auto">
            {parsed.map((p, i) => (
              <div key={i} className="flex items-center justify-between rounded border border-line px-3 py-2">
                <div className="text-sm">
                  <div className="font-medium">{p.name || "(unnamed)"}</div>
                  <div className="help">
                    {p.protocol || "?"} · {(p.control_channels ?? []).length} CC ·{" "}
                    {p.talkgroup_count} TGs
                  </div>
                </div>
                <button className="btn-ghost" onClick={() => addParsed(p)}>Add</button>
              </div>
            ))}
          </div>
        )
      ) : null}
    </Modal>
  );
}
