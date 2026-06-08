import { Section } from "../components/Section";
import { Fieldset, HzField, NumberField, SelectField, TextField } from "../components/fields";
import { ListEditor } from "../components/ListEditor";
import { useSection } from "./useSection";
import type {
  PagingConfig,
  PagingFLEXConfig,
  PagingPOCSAGConfig,
  PagingWidebandChannel,
  PagingWidebandConfig,
} from "../api/types";

export function PagingSection() {
  const [cfg, set] = useSection("Paging");
  const c = (cfg as PagingConfig) ?? { POCSAG: null, FLEX: null, Wideband: null };
  return (
    <Section
      sectionKey="paging"
      title="Paging"
    >
      <Fieldset legend="POCSAG channels" defaultOpen>
        <ListEditor<PagingPOCSAGConfig>
          label="POCSAG"
          items={c.POCSAG}
          onChange={(x) => set({ ...c, POCSAG: x })}
          makeNew={() => ({ Serial: "", FrequencyHz: 0, BaudHz: 1200 })}
          itemTitle={(ch) => ch.Serial || "channel"}
          emptyHint="No POCSAG channels."
          renderItem={(ch, setCh) => (
            <div className="grid gap-3 sm:grid-cols-3">
              <TextField label="Serial" value={ch.Serial} onChange={(v) => setCh({ ...ch, Serial: v })} />
              <HzField label="Frequency" value={ch.FrequencyHz} onChange={(v) => setCh({ ...ch, FrequencyHz: v })} />
              <SelectField
                label="Baud"
                value={String(ch.BaudHz || 1200)}
                onChange={(v) => setCh({ ...ch, BaudHz: Number(v) })}
                options={[
                  { value: "512", label: "512" },
                  { value: "1200", label: "1200" },
                  { value: "2400", label: "2400" },
                ]}
              />
            </div>
          )}
        />
      </Fieldset>
      <Fieldset legend="FLEX channels">
        <ListEditor<PagingFLEXConfig>
          label="FLEX"
          items={c.FLEX}
          onChange={(x) => set({ ...c, FLEX: x })}
          makeNew={() => ({ Serial: "", FrequencyHz: 0 })}
          itemTitle={(ch) => ch.Serial || "channel"}
          emptyHint="No FLEX channels."
          renderItem={(ch, setCh) => (
            <div className="grid gap-3 sm:grid-cols-2">
              <TextField label="Serial" value={ch.Serial} onChange={(v) => setCh({ ...ch, Serial: v })} />
              <HzField label="Frequency" value={ch.FrequencyHz} onChange={(v) => setCh({ ...ch, FrequencyHz: v })} />
            </div>
          )}
        />
      </Fieldset>
      <Fieldset legend="Wideband groups">
        <ListEditor<PagingWidebandConfig>
          label="Wideband"
          items={c.Wideband}
          onChange={(x) => set({ ...c, Wideband: x })}
          makeNew={() => ({ Serial: "", CenterFreqHz: 0, Channels: null })}
          itemTitle={(g) => g.Serial || "group"}
          emptyHint="No wideband groups. A group puts several POCSAG / FLEX channels on one SDR via a DDC bank."
          renderItem={(g, setG) => (
            <div className="space-y-3">
              <div className="grid gap-3 sm:grid-cols-2">
                <TextField label="Serial" value={g.Serial} onChange={(v) => setG({ ...g, Serial: v })} help="Single SDR shared by the whole group." />
                <HzField
                  label="Center frequency"
                  value={g.CenterFreqHz}
                  onChange={(v) => setG({ ...g, CenterFreqHz: v })}
                  help="Optional; auto = midpoint of the channel frequencies when 0."
                />
              </div>
              <Fieldset legend="Channels" defaultOpen>
                <ListEditor<PagingWidebandChannel>
                  label="Channels"
                  items={g.Channels}
                  onChange={(x) => setG({ ...g, Channels: x })}
                  makeNew={() => ({ Protocol: "pocsag", FrequencyHz: 0, BaudHz: 1200 })}
                  itemTitle={(ch) => (ch.Protocol || "channel").toUpperCase()}
                  emptyHint="Each frequency must sit within the dongle's IQ window (center ± sample_rate/2)."
                  renderItem={(ch, setCh) => {
                    const proto = (ch.Protocol || "pocsag").toLowerCase();
                    return (
                      <div className="grid gap-3 sm:grid-cols-3">
                        <SelectField
                          label="Protocol"
                          value={proto}
                          onChange={(v) => setCh({ ...ch, Protocol: v })}
                          options={[
                            { value: "pocsag", label: "POCSAG" },
                            { value: "flex", label: "FLEX" },
                          ]}
                        />
                        <HzField label="Frequency" value={ch.FrequencyHz} onChange={(v) => setCh({ ...ch, FrequencyHz: v })} />
                        {proto === "flex" ? null : (
                          <NumberField label="Baud" value={ch.BaudHz || 1200} onChange={(v) => setCh({ ...ch, BaudHz: v })} help="512 / 1200 / 2400; default 1200." />
                        )}
                      </div>
                    );
                  }}
                />
              </Fieldset>
            </div>
          )}
        />
      </Fieldset>
    </Section>
  );
}
