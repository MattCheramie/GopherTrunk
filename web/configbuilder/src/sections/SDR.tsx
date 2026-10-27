import { Section } from "../components/Section";
import {
  BoolField,
  Fieldset,
  HzField,
  NumberField,
  SelectField,
  TextField,
} from "../components/fields";
import { ListEditor } from "../components/ListEditor";
import { AdvancedJSON } from "../components/AdvancedJSON";
import { useStore } from "../store/shared";
import { formatCommaList, parseCommaList } from "../lib/csvList";
import type {
  ConvChannelConfig,
  DeviceChannelConfig,
  DeviceConfig,
  EncryptionKey,
  RTLTCPConfig,
  SDRConfig,
  SidecarConfig,
  SoapyRemoteConfig,
} from "../api/types";

const ROLES = [
  { value: "", label: "(auto)" },
  { value: "control", label: "control" },
  { value: "voice", label: "voice" },
  { value: "auto", label: "auto" },
  { value: "wideband", label: "wideband" },
];

// Remote sources can't be wideband.
const REMOTE_ROLES = [
  { value: "", label: "(auto)" },
  { value: "control", label: "control" },
  { value: "voice", label: "voice" },
  { value: "auto", label: "auto" },
];

const DEVICE_ADVANCED: (keyof DeviceConfig)[] = [
  "BlogV4", "BlogV4Lite", "IQCorrect", "IQInvert", "DCAvoid", "DCAvoidOffsetHz",
];

export function SDRSection() {
  const cfg = useStore((s) => s.config?.SDR) as SDRConfig;
  const systems = useStore((s) => s.config?.Trunking?.Systems) ?? [];
  const patch = useStore((s) => s.patchSection);
  const set = (v: SDRConfig) => patch("SDR", v);
  const systemNames = systems.map((s) => s.Name).filter(Boolean);

  const devices = cfg.Devices ?? [];
  const setDevice = (i: number, d: DeviceConfig) => {
    const next = devices.slice();
    next[i] = d;
    set({ ...cfg, Devices: next });
  };
  const addDevice = () =>
    set({
      ...cfg,
      Devices: [...devices, { Serial: "", Role: "", PPM: 0, Gain: "auto", BiasTee: false, CenterFreqHz: 0 }],
    });
  const removeDevice = (i: number) => set({ ...cfg, Devices: devices.filter((_, k) => k !== i) });

  return (
    <Section
      sectionKey="sdr"
      title="SDR Hardware"
    >
      <NumberField
        label="Sample rate (Hz)"
        value={cfg.SampleRate}
        onChange={(x) => set({ ...cfg, SampleRate: x })}
        placeholder="2400000"
        help="RTL dongles cap at ~3.2 MHz; higher rates need wideband sources (USRP/Lime/etc)."
      />

      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <span className="label mb-0">Local devices ({devices.length})</span>
          <button className="btn-ghost" onClick={addDevice}>
            + Add device
          </button>
        </div>
        {devices.length === 0 ? (
          <p className="help">No devices yet. Add at least a control + voice dongle for trunking.</p>
        ) : null}
        {devices.map((d, i) => (
          <div key={i} className="rounded-md border border-line p-3 space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">{d.Serial || `Device ${i + 1}`}</span>
              <button className="btn-danger" onClick={() => removeDevice(i)}>
                Remove
              </button>
            </div>
            <DeviceEditor d={d} systemNames={systemNames} onChange={(next) => setDevice(i, next)} />
          </div>
        ))}
      </div>

      <Fieldset legend="Remote rtl_tcp sources">
        <ListEditor<RTLTCPConfig>
          label="rtl_tcp endpoints"
          items={cfg.RTLTCP}
          onChange={(x) => set({ ...cfg, RTLTCP: x })}
          makeNew={() => ({ Addr: "", Serial: "", Role: "", PPM: 0, Gain: "auto", BiasTee: false, ConnectTimeoutMs: 0 })}
          itemTitle={(r) => r.Addr || "rtl_tcp"}
          emptyHint="Network rtl_tcp dongles (host:port)."
          renderItem={(r, set) => (
            <div className="grid gap-3 sm:grid-cols-2">
              <TextField label="Address" value={r.Addr} onChange={(v) => set({ ...r, Addr: v })} placeholder="192.168.1.50:1234" />
              <SelectField label="Role" value={r.Role} onChange={(v) => set({ ...r, Role: v })} options={REMOTE_ROLES} />
              <TextField label="Serial" value={r.Serial} onChange={(v) => set({ ...r, Serial: v })} />
              <TextField label="Gain" value={r.Gain} onChange={(v) => set({ ...r, Gain: v })} placeholder="auto or tenths-dB" />
              <NumberField label="PPM" value={r.PPM} onChange={(v) => set({ ...r, PPM: v })} />
              <NumberField label="Connect timeout (ms)" value={r.ConnectTimeoutMs} onChange={(v) => set({ ...r, ConnectTimeoutMs: v })} />
              <BoolField label="Bias-tee" value={r.BiasTee} onChange={(v) => set({ ...r, BiasTee: v })} />
            </div>
          )}
        />
      </Fieldset>

      <Fieldset legend="Remote SoapySDR sources">
        <ListEditor<SoapyRemoteConfig>
          label="soapy_remote endpoints"
          items={cfg.SoapyRemote}
          onChange={(x) => set({ ...cfg, SoapyRemote: x })}
          makeNew={() => ({ Addr: "", Driver: "", Args: "", MasterClockHz: 0, Serial: "", Role: "", Format: "", StreamProtocol: "", StreamMTU: 0, StreamWindow: 0, PPM: 0, Gain: "auto", BiasTee: false, ConnectTimeoutMs: 0, Diversity: "", DiversityCapture: "", DiversityCaptureSeconds: 0, DiversityCaptureFormat: "", VerboseDebug: false })}
          itemTitle={(s) => s.Addr || "soapy_remote"}
          emptyHint="SoapySDRServer endpoints (USRP, Lime, bladeRF, HackRF, Airspy, …)."
          renderItem={(s, set) => (
            <div className="grid gap-3 sm:grid-cols-2">
              <TextField label="Address" value={s.Addr} onChange={(v) => set({ ...s, Addr: v })} placeholder="192.168.1.60:55132" />
              <TextField label="Driver" value={s.Driver} onChange={(v) => set({ ...s, Driver: v })} placeholder="uhd / lime / bladerf / hackrf / airspy / rtlsdr" />
              <TextField label="Args" value={s.Args} onChange={(v) => set({ ...s, Args: v })} placeholder="key=value,key2=value2" />
              <NumberField label="Master clock (Hz)" value={s.MasterClockHz} onChange={(v) => set({ ...s, MasterClockHz: v })} placeholder="0 = device default; B210: 61440000" />
              <TextField label="Serial" value={s.Serial} onChange={(v) => set({ ...s, Serial: v })} />
              <SelectField label="Role" value={s.Role} onChange={(v) => set({ ...s, Role: v })} options={REMOTE_ROLES} />
              <SelectField
                label="Sample format"
                value={s.Format}
                onChange={(v) => set({ ...s, Format: v })}
                options={[
                  { value: "", label: "(default)" },
                  { value: "CS16", label: "CS16" },
                  { value: "CF32", label: "CF32" },
                ]}
              />
              <SelectField
                label="Stream protocol"
                value={s.StreamProtocol}
                onChange={(v) => set({ ...s, StreamProtocol: v })}
                options={[
                  { value: "", label: "(default)" },
                  { value: "tcp", label: "tcp" },
                ]}
              />
              <NumberField label="Stream MTU (bytes)" value={s.StreamMTU} onChange={(v) => set({ ...s, StreamMTU: v })} placeholder="0 = default 1500" />
              <NumberField label="Stream window (bytes)" value={s.StreamWindow} onChange={(v) => set({ ...s, StreamWindow: v })} placeholder="0 = default 8 MiB" />
              <TextField label="Gain" value={s.Gain} onChange={(v) => set({ ...s, Gain: v })} placeholder="auto or tenths-dB" />
              <NumberField label="PPM" value={s.PPM} onChange={(v) => set({ ...s, PPM: v })} />
              <NumberField label="Connect timeout (ms)" value={s.ConnectTimeoutMs} onChange={(v) => set({ ...s, ConnectTimeoutMs: v })} />
              <BoolField label="Bias-tee" value={s.BiasTee} onChange={(v) => set({ ...s, BiasTee: v })} />
              <SelectField
                label="Diversity (experimental)"
                value={s.Diversity}
                onChange={(v) => set({ ...s, Diversity: v })}
                options={[
                  { value: "", label: "(none)" },
                  { value: "mrc", label: "mrc (RX0+RX1, tracking)" },
                  { value: "mrc-static", label: "mrc-static (one-shot gain)" },
                ]}
              />
              <TextField
                label="RX antenna (per channel)"
                // Read the primary antenna: field, falling back to the legacy
                // antennas: so an older config still shows its port. On change
                // we write antenna: and clear antennas: (setting both is an
                // error) — antenna=RX1 in args is silently ignored by make(),
                // so this list is the only path that selects the RX port.
                value={(s.Antenna ?? s.Antennas ?? []).join(", ")}
                onChange={(v) => {
                  const list = v
                    .split(",")
                    .map((x) => x.trim())
                    .filter((x) => x !== "");
                  set({
                    ...s,
                    Antenna: list.length ? list : undefined,
                    Antennas: undefined,
                  });
                }}
                placeholder="e.g. RX1 (single) or RX1, RX2 (X310 under mrc)"
              />
              <TextField
                label="Diversity capture (path prefix)"
                value={s.DiversityCapture}
                onChange={(v) => set({ ...s, DiversityCapture: v })}
                placeholder="pre-combine per-branch IQ, e.g. iq/mrc/x310"
              />
              <NumberField
                label="Diversity capture seconds"
                value={s.DiversityCaptureSeconds}
                onChange={(v) => set({ ...s, DiversityCaptureSeconds: v })}
                placeholder="0 = 5 s (1..1200)"
              />
              <SelectField
                label="Diversity capture format"
                value={s.DiversityCaptureFormat}
                onChange={(v) => set({ ...s, DiversityCaptureFormat: v })}
                options={[
                  { value: "", label: "cs16 (default)" },
                  { value: "flac", label: "flac (lossless, 30-50% smaller)" },
                ]}
              />
              <BoolField
                label="Verbose RPC debug"
                value={s.VerboseDebug}
                onChange={(v) => set({ ...s, VerboseDebug: v })}
              />
            </div>
          )}
        />
      </Fieldset>

      <Fieldset legend="Sidecar sources">
        <ListEditor<SidecarConfig>
          label="sidecar endpoints"
          items={cfg.Sidecar}
          onChange={(x) => set({ ...cfg, Sidecar: x })}
          makeNew={() => ({ Transport: "tcp", DataAddr: "", ControlAddr: "", Format: "cs16", SampleRateHz: 0, FreqMinHz: 0, FreqMaxHz: 0, Serial: "", Role: "", Gain: "auto", ConnectTimeoutMs: 0 })}
          itemTitle={(s) => s.DataAddr || "sidecar"}
          emptyHint="An external process that owns a radio and streams raw IQ to GopherTrunk (UHD/RFNoC, GNU Radio, a vendor tool with no SoapySDR support)."
          renderItem={(s, set) => (
            <div className="grid gap-3 sm:grid-cols-2">
              <SelectField
                label="Transport"
                value={s.Transport}
                onChange={(v) => set({ ...s, Transport: v })}
                options={[
                  { value: "", label: "(default: tcp)" },
                  { value: "unix_pipe", label: "unix_pipe (FIFO, POSIX only)" },
                  { value: "tcp", label: "tcp" },
                  { value: "udp", label: "udp" },
                ]}
              />
              <TextField label="Data address" value={s.DataAddr} onChange={(v) => set({ ...s, DataAddr: v })} placeholder="/tmp/iq.fifo or 127.0.0.1:5001" />
              <TextField label="Control address" value={s.ControlAddr} onChange={(v) => set({ ...s, ControlAddr: v })} placeholder="127.0.0.1:4001 (empty = sidecar owns tuning)" />
              <SelectField
                label="Sample format"
                value={s.Format}
                onChange={(v) => set({ ...s, Format: v })}
                options={[
                  { value: "", label: "(default: cs16)" },
                  { value: "cs16", label: "cs16" },
                  { value: "complex64", label: "complex64" },
                ]}
              />
              <NumberField label="Sample rate (Hz)" value={s.SampleRateHz} onChange={(v) => set({ ...s, SampleRateHz: v })} placeholder="required — the stream carries no metadata" />
              <TextField label="Serial" value={s.Serial} onChange={(v) => set({ ...s, Serial: v })} />
              <SelectField label="Role" value={s.Role} onChange={(v) => set({ ...s, Role: v })} options={REMOTE_ROLES} />
              <TextField label="Gain" value={s.Gain} onChange={(v) => set({ ...s, Gain: v })} placeholder="auto or tenths-dB" />
              <NumberField label="Min frequency (Hz)" value={s.FreqMinHz} onChange={(v) => set({ ...s, FreqMinHz: v })} placeholder="0 = unknown" />
              <NumberField label="Max frequency (Hz)" value={s.FreqMaxHz} onChange={(v) => set({ ...s, FreqMaxHz: v })} placeholder="0 = unknown" />
              <NumberField label="Connect timeout (ms)" value={s.ConnectTimeoutMs} onChange={(v) => set({ ...s, ConnectTimeoutMs: v })} placeholder="0 = default 3000" />
            </div>
          )}
        />
      </Fieldset>
    </Section>
  );
}

function DeviceEditor(props: {
  d: DeviceConfig;
  systemNames: string[];
  onChange: (next: DeviceConfig) => void;
}) {
  const { d, systemNames, onChange } = props;
  const isWideband = d.Role === "wideband";
  return (
    <div className="space-y-3">
      <div className="grid gap-3 sm:grid-cols-2">
        <TextField label="Serial" value={d.Serial} onChange={(x) => onChange({ ...d, Serial: x })} placeholder="00000001" />
        <SelectField label="Role" value={d.Role} onChange={(x) => onChange({ ...d, Role: x })} options={ROLES} />
        <TextField label="Gain" value={d.Gain} onChange={(x) => onChange({ ...d, Gain: x })} placeholder="auto or tenths-dB e.g. 496" />
        <NumberField label="PPM correction" value={d.PPM} onChange={(x) => onChange({ ...d, PPM: x })} />
        {isWideband ? (
          <>
            <HzField label="Center freq" value={d.CenterFreqHz} onChange={(x) => onChange({ ...d, CenterFreqHz: x })} />
            <SelectField
              label="Tuner strategy"
              value={d.TunerStrategy ?? ""}
              onChange={(x) => onChange({ ...d, TunerStrategy: x })}
              options={[
                { value: "", label: "(auto)" },
                { value: "ddc", label: "ddc" },
                { value: "polyphase", label: "polyphase" },
              ]}
            />
            <NumberField label="Voice taps" value={d.VoiceTaps ?? 0} onChange={(x) => onChange({ ...d, VoiceTaps: x })} />
            <NumberField label="Signalling taps (P25 P2 alias)" value={d.SignallingTaps ?? 0} onChange={(x) => onChange({ ...d, SignallingTaps: x })} />
          </>
        ) : null}
      </div>
      <BoolField
        label="Bias-tee (5V antenna power)"
        value={d.BiasTee}
        onChange={(x) => onChange({ ...d, BiasTee: x })}
      />
      {d.Role === "voice" ? (
        <NumberField
          label="Own sample rate (Hz, scanner-only SDR; 0 = shared)"
          value={d.SampleRate ?? 0}
          onChange={(x) => onChange({ ...d, SampleRate: x })}
        />
      ) : null}

      {isWideband ? (
        <Fieldset legend="Wideband channels" defaultOpen>
          <ListEditor<DeviceChannelConfig>
            label="Channels"
            items={d.Channels}
            onChange={(x) => onChange({ ...d, Channels: x })}
            makeNew={() => ({ FrequencyHz: 0, System: systemNames[0] ?? "" })}
            itemTitle={(c) => c.System || "channel"}
            emptyHint="Per-repeater carriers inside this dongle's IQ band; each binds to a trunking system."
            renderItem={(c, set) => (
              <div className="grid gap-3 sm:grid-cols-2">
                <HzField label="Frequency" value={c.FrequencyHz} onChange={(v) => set({ ...c, FrequencyHz: v })} />
                {systemNames.length > 0 ? (
                  <SelectField
                    label="System"
                    value={c.System}
                    onChange={(v) => set({ ...c, System: v })}
                    options={systemNames.map((n) => ({ value: n, label: n }))}
                  />
                ) : (
                  <TextField label="System" value={c.System} onChange={(v) => set({ ...c, System: v })} help="Must match a trunking system name." />
                )}
                <SelectField
                  label="P25 Ph1 demod override"
                  value={c.P25Phase1DemodMode ?? ""}
                  onChange={(v) => set({ ...c, P25Phase1DemodMode: v })}
                  options={[
                    { value: "", label: "Inherit system default" },
                    { value: "c4fm", label: "C4FM / FM" },
                    { value: "cqpsk", label: "CQPSK / LSM (linear)" },
                  ]}
                  help="Override the system's demod path for this site only. Use when one P25 system genuinely mixes CQPSK/linear (LSM) sites and C4FM sites — decided empirically (a strong signal won't lock in C4FM), NOT just because a site is simulcast (most simulcast is C4FM). Issue #935. P25 Phase 1 only."
                />
              </div>
            )}
          />
        </Fieldset>
      ) : null}

      {isWideband ? (
        <Fieldset legend="Conventional FM / AM channels (continuous)">
          <ListEditor<ConvChannelConfig>
            label="Conventional"
            items={d.Conventional}
            onChange={(x) => onChange({ ...d, Conventional: x })}
            makeNew={() => ({
              Label: "",
              FrequencyHz: 0,
              Mode: "nfm",
              SquelchDbFS: 0,
              SquelchCNDb: 0,
              HangtimeMs: 0,
              ActivityDebounceMs: 0,
              SquelchHysteresisDb: 0,
              Priority: 0,
              TalkgroupID: 0,
              Gain: "",
              AudioGainDB: 0,
              Device: "",
              Tone: { Mode: "", CTCSSHz: 0, DCSCode: "", DCSPolarity: "" },
              Decoders: null,
              NAC: "",
              P25Phase1DemodMode: "",
            })}
            itemTitle={(c) => c.Label || "channel"}
            emptyHint="Analog FM / AM channels inside this dongle's IQ band, each monitored all the time on its own tap: overlapping transmissions record as separate calls (no scanning)."
            renderItem={(c, set) => (
              <div className="grid gap-3 sm:grid-cols-2">
                <TextField label="Label" value={c.Label} onChange={(v) => set({ ...c, Label: v })} />
                <HzField label="Frequency" value={c.FrequencyHz} onChange={(v) => set({ ...c, FrequencyHz: v })} />
                <SelectField
                  label="Mode"
                  value={c.Mode}
                  onChange={(v) => set({ ...c, Mode: v })}
                  options={[
                    { value: "nfm", label: "nfm" },
                    { value: "fm", label: "fm" },
                    { value: "am", label: "am" },
                  ]}
                />
                {c.Mode === "am" ? (
                  <NumberField
                    label="Squelch C/N (dB, 0 = 12)"
                    step={0.5}
                    value={c.SquelchCNDb ?? 0}
                    onChange={(v) => set({ ...c, SquelchCNDb: v })}
                  />
                ) : (
                  <NumberField label="Squelch (dBFS)" step={0.1} value={c.SquelchDbFS} onChange={(v) => set({ ...c, SquelchDbFS: v })} />
                )}
                <NumberField label="Hangtime (ms)" value={c.HangtimeMs} onChange={(v) => set({ ...c, HangtimeMs: v })} />
                <NumberField label="Talkgroup ID (0 = positional)" value={c.TalkgroupID} onChange={(v) => set({ ...c, TalkgroupID: v })} />
                <NumberField label="Audio gain (dB)" step={0.5} value={c.AudioGainDB ?? 0} onChange={(v) => set({ ...c, AudioGainDB: v })} />
                <SelectField
                  label="Tone mode"
                  value={c.Tone?.Mode ?? ""}
                  onChange={(v) => set({ ...c, Tone: { ...c.Tone, Mode: v } })}
                  options={[
                    { value: "", label: "none" },
                    { value: "ctcss", label: "CTCSS" },
                    { value: "dcs", label: "DCS" },
                  ]}
                />
                {c.Tone?.Mode === "ctcss" ? (
                  <NumberField
                    label="CTCSS (Hz, 50–300)"
                    step={0.1}
                    value={c.Tone?.CTCSSHz ?? 0}
                    onChange={(v) => set({ ...c, Tone: { ...c.Tone, CTCSSHz: v } })}
                  />
                ) : null}
                {c.Tone?.Mode === "dcs" ? (
                  <TextField
                    label="DCS code (3 octal digits)"
                    value={c.Tone?.DCSCode ?? ""}
                    onChange={(v) => set({ ...c, Tone: { ...c.Tone, DCSCode: v } })}
                    placeholder="023"
                  />
                ) : null}
                <TextField
                  label="Data decoders"
                  value={formatCommaList(c.Decoders)}
                  onChange={(v) => set({ ...c, Decoders: parseCommaList(v) })}
                  placeholder="mdc1200, fleetsync"
                />
                <ListEditor<EncryptionKey>
                  label="Decryption keys"
                  items={c.EncryptionKeys ?? null}
                  onChange={(x) => set({ ...c, EncryptionKeys: x })}
                  makeNew={() => ({ KeyID: 0, Algorithm: "scrambler", Key: "" })}
                  itemTitle={(e) => `Key ID ${e.KeyID}`}
                  emptyHint="Digital channels only: an NXDN channel's scrambler keys (algorithm scrambler, the radio's decimal 15-bit key, key_id as the VCALL names it). The voice chain descrambles a call whose key ID matches."
                  renderItem={(e, setKey) => (
                    <div className="grid gap-3 sm:grid-cols-3">
                      <NumberField label="Key ID" value={e.KeyID} onChange={(v) => setKey({ ...e, KeyID: v })} />
                      <SelectField
                        label="Algorithm"
                        value={e.Algorithm}
                        onChange={(v) => setKey({ ...e, Algorithm: v })}
                        options={[
                          { value: "scrambler", label: "scrambler (NXDN 15-bit scrambler, decimal key)" },
                          { value: "rc4", label: "rc4 (DMR Enhanced Privacy / P25 ADP)" },
                          { value: "des", label: "des (P25 DES-OFB)" },
                          { value: "tdes", label: "tdes (P25 Triple-DES)" },
                          { value: "aes", label: "aes (P25 AES-128 / AES-256)" },
                        ]}
                      />
                      <TextField label="Key" type="password" value={e.Key} onChange={(v) => setKey({ ...e, Key: v })} />
                    </div>
                  )}
                />
              </div>
            )}
          />
        </Fieldset>
      ) : null}

      <AdvancedJSON<DeviceConfig>
        label="Advanced device flags (JSON)"
        value={d}
        onChange={onChange}
        pick={DEVICE_ADVANCED}
        help="RTL-SDR Blog V4 mode (blog_v4 / blog_v4_lite) and IQ correction/inversion flags."
      />
    </div>
  );
}
