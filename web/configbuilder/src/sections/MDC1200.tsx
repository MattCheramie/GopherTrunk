import { Section } from "../components/Section";
import { BoolField, HzField, TextField } from "../components/fields";
import { ListEditor } from "../components/ListEditor";
import { useSection } from "./useSection";
import type { MDC1200ChannelConfig, MDC1200Config } from "../api/types";

export function MDC1200Section() {
  const [cfg, set] = useSection("MDC1200");
  const c = (cfg as MDC1200Config) ?? { Channels: null };
  return (
    <Section
      sectionKey="mdc1200"
      title="MDC1200"
      instructions="Motorola MDC1200 FFSK signaling receiver. Target the conventional analog voice channels you monitor — bursts ride at the head/tail of each transmission."
    >
      <ListEditor<MDC1200ChannelConfig>
        label="Channels"
        items={c.Channels}
        onChange={(x) => set({ ...c, Channels: x })}
        makeNew={() => ({ Serial: "", FrequencyHz: 0, DropBadCRC: false })}
        itemTitle={(ch) => ch.Serial || "channel"}
        emptyHint="No MDC1200 channels."
        renderItem={(ch, setCh) => (
          <div className="space-y-3">
            <div className="grid gap-3 sm:grid-cols-2">
              <TextField label="Serial" value={ch.Serial} onChange={(v) => setCh({ ...ch, Serial: v })} />
              <HzField label="Frequency" value={ch.FrequencyHz} onChange={(v) => setCh({ ...ch, FrequencyHz: v })} />
            </div>
            <BoolField label="Drop CRC-failed bursts" value={ch.DropBadCRC} onChange={(v) => setCh({ ...ch, DropBadCRC: v })} />
          </div>
        )}
      />
    </Section>
  );
}
