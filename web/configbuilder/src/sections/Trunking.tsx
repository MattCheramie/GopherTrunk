import { useState } from "react";
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
import { dmrBandPlanForMode, dmrBandPlanMode } from "../lib/dmrBandPlan";
import { useStore } from "../store/shared";
import { api } from "../api/client";
import type {
  DMRBandPlanTableEntry,
  EncryptionKey,
  P25BandPlanEntry,
  ParsedSystemDTO,
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
  "P25Phase2ScramblerMode", "P25Phase2ClockMode", "NXDNViterbiMode",
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
      instructions="The trunked radio networks to follow. Each system needs a unique name, a protocol, and at least one control-channel frequency. Add systems by hand, by browsing RadioReference.com, or by importing a RadioReference PDF/CSV."
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
        <div key={i} className="rounded-md border border-white/10 p-3 space-y-3">
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

function Modal(props: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center bg-black/60 p-4 overflow-y-auto">
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
  const [kind, setKind] = useState<"zip" | "county" | "state">("zip");
  const [value, setValue] = useState("");
  const [hits, setHits] = useState<RRSearchHit[] | null>(null);
  const [busy, setBusy] = useState(false);

  const search = async () => {
    setBusy(true);
    setHits(null);
    try {
      const r = await api.rrSearch(kind, value.trim());
      setHits(r.results ?? []);
    } catch (e) {
      setError(`RadioReference: ${(e as Error).message}`);
    } finally {
      setBusy(false);
    }
  };

  const importSystem = async (sid: number) => {
    setBusy(true);
    try {
      const r = await api.rrSystem(sid);
      props.onAdd(r.config, r.talkgroups ?? []);
      props.onClose();
    } catch (e) {
      setError(`RadioReference: ${(e as Error).message}`);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal title="Browse RadioReference.com" onClose={props.onClose}>
      <p className="help">
        Search by ZIP code, county id (ctid), or state id (stid), then import a
        system with its control channels and talkgroups. Requires RadioReference
        credentials configured on the server.
      </p>
      <div className="flex gap-2">
        <select className="input w-32" value={kind} onChange={(e) => setKind(e.target.value as "zip" | "county" | "state")}>
          <option value="zip">ZIP</option>
          <option value="county">County id</option>
          <option value="state">State id</option>
        </select>
        <input
          className="input"
          value={value}
          placeholder={kind === "zip" ? "78701" : "numeric id"}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && search()}
        />
        <button className="btn" disabled={busy || !value.trim()} onClick={search}>
          Search
        </button>
      </div>
      {busy ? <p className="help">Working…</p> : null}
      {hits ? (
        hits.length === 0 ? (
          <p className="help">No systems found.</p>
        ) : (
          <div className="max-h-80 space-y-1 overflow-y-auto">
            {hits.map((h) => (
              <div key={h.sid} className="flex items-center justify-between rounded border border-white/10 px-3 py-2">
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
              <div key={i} className="flex items-center justify-between rounded border border-white/10 px-3 py-2">
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
