import { Section } from "../components/Section";
import { Fieldset, HzField, TextField } from "../components/fields";
import { ListEditor } from "../components/ListEditor";
import { useSection } from "./useSection";
import type { ADSBBeastConfig, ADSBChannelConfig, ADSBConfig } from "../api/types";

export function ADSBSection() {
  const [cfg, set] = useSection("ADSB");
  const c = (cfg as ADSBConfig) ?? { BeastUpstreams: null, Channels: null };
  return (
    <Section
      sectionKey="adsb"
      title="ADS-B"
      instructions="Aircraft tracking. Consume Mode-S frames from a BEAST upstream (dump1090 / readsb, typically host:30005), or pin an SDR to 1090 MHz for the native PPM receiver."
    >
      <Fieldset legend="BEAST upstreams" defaultOpen>
        <ListEditor<ADSBBeastConfig>
          label="Upstreams"
          items={c.BeastUpstreams}
          onChange={(x) => set({ ...c, BeastUpstreams: x })}
          makeNew={() => ({ Addr: "", Name: "" })}
          itemTitle={(u) => u.Name || u.Addr || "upstream"}
          emptyHint="No BEAST upstreams."
          renderItem={(u, setU) => (
            <div className="grid gap-3 sm:grid-cols-2">
              <TextField label="Address" value={u.Addr} onChange={(v) => setU({ ...u, Addr: v })} placeholder="host:30005" />
              <TextField label="Name (label)" value={u.Name} onChange={(v) => setU({ ...u, Name: v })} />
            </div>
          )}
        />
      </Fieldset>
      <Fieldset legend="Native 1090 MHz SDR channels">
        <ListEditor<ADSBChannelConfig>
          label="Channels"
          items={c.Channels}
          onChange={(x) => set({ ...c, Channels: x })}
          makeNew={() => ({ Serial: "", FrequencyHz: 1_090_000_000 })}
          itemTitle={(ch) => ch.Serial || "channel"}
          emptyHint="No native channels (a 1090 MHz SAW filter + LNA is strongly recommended)."
          renderItem={(ch, setCh) => (
            <div className="grid gap-3 sm:grid-cols-2">
              <TextField label="Serial" value={ch.Serial} onChange={(v) => setCh({ ...ch, Serial: v })} />
              <HzField label="Frequency" value={ch.FrequencyHz} onChange={(v) => setCh({ ...ch, FrequencyHz: v })} />
            </div>
          )}
        />
      </Fieldset>
    </Section>
  );
}
