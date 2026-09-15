import { Section } from "../components/Section";
import { BoolField, HzField, NumberField, TextField } from "../components/fields";
import { ListEditor } from "../components/ListEditor";
import { useSection } from "./useSection";
import type { FleetSyncChannelConfig, FleetSyncConfig } from "../api/types";

export function FleetSyncSection() {
  const [cfg, set] = useSection("FleetSync");
  const c = (cfg as FleetSyncConfig) ?? { Channels: null };
  return (
    <Section
      sectionKey="fleetsync"
      title="FleetSync"
    >
      <ListEditor<FleetSyncChannelConfig>
        label="Channels"
        items={c.Channels}
        onChange={(x) => set({ ...c, Channels: x })}
        makeNew={() => ({ Serial: "", FrequencyHz: 0, BaudHz: 0, DropBadCRC: false })}
        itemTitle={(ch) => ch.Serial || "channel"}
        emptyHint="No FleetSync channels."
        renderItem={(ch, setCh) => (
          <div className="space-y-3">
            <div className="grid gap-3 sm:grid-cols-2">
              <TextField label="Serial" value={ch.Serial} onChange={(v) => setCh({ ...ch, Serial: v })} />
              <HzField label="Frequency" value={ch.FrequencyHz} onChange={(v) => setCh({ ...ch, FrequencyHz: v })} />
              <NumberField
                label="Baud (0 = 1200)"
                value={ch.BaudHz}
                onChange={(v) => setCh({ ...ch, BaudHz: v })}
                placeholder="1200"
              />
            </div>
            <BoolField label="Drop block-check-failed bursts" value={ch.DropBadCRC} onChange={(v) => setCh({ ...ch, DropBadCRC: v })} />
          </div>
        )}
      />
    </Section>
  );
}
