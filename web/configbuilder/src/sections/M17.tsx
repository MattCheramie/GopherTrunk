import { Section } from "../components/Section";
import { HzField, TextField } from "../components/fields";
import { ListEditor } from "../components/ListEditor";
import { useSection } from "./useSection";
import type { M17ChannelConfig, M17Config } from "../api/types";

export function M17Section() {
  const [cfg, set] = useSection("M17");
  const c = (cfg as M17Config) ?? { Channels: null };
  return (
    <Section
      sectionKey="m17"
      title="M17"
    >
      <ListEditor<M17ChannelConfig>
        label="Channels"
        items={c.Channels}
        onChange={(x) => set({ ...c, Channels: x })}
        makeNew={() => ({ Serial: "", FrequencyHz: 144_975_000 })}
        itemTitle={(ch) => ch.Serial || "channel"}
        emptyHint="No M17 channels."
        renderItem={(ch, setCh) => (
          <div className="grid gap-3 sm:grid-cols-2">
            <TextField label="Serial" value={ch.Serial} onChange={(v) => setCh({ ...ch, Serial: v })} />
            <HzField label="Frequency" value={ch.FrequencyHz} onChange={(v) => setCh({ ...ch, FrequencyHz: v })} />
          </div>
        )}
      />
    </Section>
  );
}
