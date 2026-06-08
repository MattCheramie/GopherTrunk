import { Section } from "../components/Section";
import { BoolField, HzField, TextField } from "../components/fields";
import { ListEditor } from "../components/ListEditor";
import { useSection } from "./useSection";
import type { AISChannelConfig, AISConfig } from "../api/types";

export function AISSection() {
  const [cfg, set] = useSection("AIS");
  const c = (cfg as AISConfig) ?? { Channels: null };
  return (
    <Section
      sectionKey="ais"
      title="AIS"
    >
      <ListEditor<AISChannelConfig>
        label="Channels"
        items={c.Channels}
        onChange={(x) => set({ ...c, Channels: x })}
        makeNew={() => ({ Serial: "", FrequencyHz: 161_975_000, DropBadFCS: false, DropNonPosition: false })}
        itemTitle={(ch) => ch.Serial || "channel"}
        emptyHint="No AIS channels."
        renderItem={(ch, setCh) => (
          <div className="space-y-3">
            <div className="grid gap-3 sm:grid-cols-2">
              <TextField label="Serial" value={ch.Serial} onChange={(v) => setCh({ ...ch, Serial: v })} />
              <HzField label="Frequency" value={ch.FrequencyHz} onChange={(v) => setCh({ ...ch, FrequencyHz: v })} />
            </div>
            <BoolField label="Drop bad-FCS frames" value={ch.DropBadFCS} onChange={(v) => setCh({ ...ch, DropBadFCS: v })} />
            <BoolField label="Drop non-position messages" value={ch.DropNonPosition} onChange={(v) => setCh({ ...ch, DropNonPosition: v })} />
          </div>
        )}
      />
    </Section>
  );
}
