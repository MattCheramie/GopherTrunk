import { Section } from "../components/Section";
import { BoolField, HzField, TextField } from "../components/fields";
import { ListEditor } from "../components/ListEditor";
import { useSection } from "./useSection";
import type { DSCChannelConfig, DSCConfig } from "../api/types";

export function DSCSection() {
  const [cfg, set] = useSection("DSC");
  const c = (cfg as DSCConfig) ?? { Channels: null };
  return (
    <Section
      sectionKey="dsc"
      title="DSC"
    >
      <ListEditor<DSCChannelConfig>
        label="Channels"
        items={c.Channels}
        onChange={(x) => set({ ...c, Channels: x })}
        makeNew={() => ({ Serial: "", FrequencyHz: 156_525_000, DropBadFCS: false })}
        itemTitle={(ch) => ch.Serial || "channel"}
        emptyHint="No DSC channels."
        renderItem={(ch, setCh) => (
          <div className="space-y-3">
            <div className="grid gap-3 sm:grid-cols-2">
              <TextField label="Serial" value={ch.Serial} onChange={(v) => setCh({ ...ch, Serial: v })} />
              <HzField label="Frequency" value={ch.FrequencyHz} onChange={(v) => setCh({ ...ch, FrequencyHz: v })} />
            </div>
            <BoolField label="Drop BCH-marginal sequences" value={ch.DropBadFCS} onChange={(v) => setCh({ ...ch, DropBadFCS: v })} />
          </div>
        )}
      />
    </Section>
  );
}
