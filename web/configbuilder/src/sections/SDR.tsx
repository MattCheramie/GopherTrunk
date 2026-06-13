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
import type {
  DeviceChannelConfig,
  DeviceConfig,
  RTLTCPConfig,
  SDRConfig,
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
          <div key={i} className="rounded-md border border-white/10 p-3 space-y-3">
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
          makeNew={() => ({ Addr: "", Driver: "", Args: "", MasterClockHz: 0, Serial: "", Role: "", Format: "", StreamProtocol: "", PPM: 0, Gain: "auto", BiasTee: false, ConnectTimeoutMs: 0 })}
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
              <TextField label="Gain" value={s.Gain} onChange={(v) => set({ ...s, Gain: v })} placeholder="auto or tenths-dB" />
              <NumberField label="PPM" value={s.PPM} onChange={(v) => set({ ...s, PPM: v })} />
              <NumberField label="Connect timeout (ms)" value={s.ConnectTimeoutMs} onChange={(v) => set({ ...s, ConnectTimeoutMs: v })} />
              <BoolField label="Bias-tee" value={s.BiasTee} onChange={(v) => set({ ...s, BiasTee: v })} />
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
            <NumberField label="Voice taps (0–8)" value={d.VoiceTaps ?? 0} onChange={(x) => onChange({ ...d, VoiceTaps: x })} />
          </>
        ) : null}
      </div>
      <BoolField
        label="Bias-tee (5V antenna power)"
        value={d.BiasTee}
        onChange={(x) => onChange({ ...d, BiasTee: x })}
      />

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
