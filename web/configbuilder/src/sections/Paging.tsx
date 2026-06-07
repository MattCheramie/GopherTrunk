import { Section } from "../components/Section";
import { Fieldset, HzField, SelectField, TextField } from "../components/fields";
import { ListEditor } from "../components/ListEditor";
import { useSection } from "./useSection";
import type { PagingConfig, PagingFLEXConfig, PagingPOCSAGConfig } from "../api/types";

export function PagingSection() {
  const [cfg, set] = useSection("Paging");
  const c = (cfg as PagingConfig) ?? { POCSAG: null, FLEX: null };
  return (
    <Section
      sectionKey="paging"
      title="Paging"
      instructions="POCSAG and FLEX pager decoders. Each channel pins an SDR to a paging frequency."
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
    </Section>
  );
}
