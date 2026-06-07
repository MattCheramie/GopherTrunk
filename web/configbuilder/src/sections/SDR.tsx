import { Section } from "../components/Section";
import { NumberField, SelectField, TextField, BoolField } from "../components/fields";
import { useStore } from "../store/shared";
import type { DeviceConfig, SDRConfig } from "../api/types";

const ROLES = [
  { value: "", label: "(auto)" },
  { value: "control", label: "control" },
  { value: "voice", label: "voice" },
  { value: "auto", label: "auto" },
  { value: "wideband", label: "wideband" },
];

export function SDRSection() {
  const cfg = useStore((s) => s.config?.SDR) as SDRConfig;
  const patch = useStore((s) => s.patchSection);
  const set = (v: SDRConfig) => patch("SDR", v);

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
  const removeDevice = (i: number) =>
    set({ ...cfg, Devices: devices.filter((_, k) => k !== i) });

  return (
    <Section
      sectionKey="sdr"
      title="SDR Hardware"
      instructions="Sample rate (225 kHz–20 MHz) every tuner is programmed to, and the list of SDR devices. P25 trunking needs separate control and voice dongles (distinct serials)."
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
          <span className="label mb-0">Devices ({devices.length})</span>
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
              <span className="text-sm font-medium">Device {i + 1}</span>
              <button className="btn-danger" onClick={() => removeDevice(i)}>
                Remove
              </button>
            </div>
            <div className="grid gap-3 sm:grid-cols-2">
              <TextField
                label="Serial"
                value={d.Serial}
                onChange={(x) => setDevice(i, { ...d, Serial: x })}
                placeholder="00000001"
              />
              <SelectField
                label="Role"
                value={d.Role}
                onChange={(x) => setDevice(i, { ...d, Role: x })}
                options={ROLES}
              />
              <TextField
                label="Gain"
                value={d.Gain}
                onChange={(x) => setDevice(i, { ...d, Gain: x })}
                placeholder="auto or tenths-dB e.g. 496"
              />
              <NumberField
                label="PPM correction"
                value={d.PPM}
                onChange={(x) => setDevice(i, { ...d, PPM: x })}
              />
              {d.Role === "wideband" ? (
                <NumberField
                  label="Center freq (Hz)"
                  value={d.CenterFreqHz}
                  onChange={(x) => setDevice(i, { ...d, CenterFreqHz: x })}
                />
              ) : null}
            </div>
            <BoolField
              label="Bias-tee (5V antenna power)"
              value={d.BiasTee}
              onChange={(x) => setDevice(i, { ...d, BiasTee: x })}
            />
          </div>
        ))}
      </div>
    </Section>
  );
}
