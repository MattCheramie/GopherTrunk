import { Section } from "../components/Section";
import { BoolField, HzField, TextField } from "../components/fields";
import { ListEditor } from "../components/ListEditor";
import { useSection } from "./useSection";
import type { APRSChannelConfig, APRSConfig } from "../api/types";

export function APRSSection() {
  const [cfg, set] = useSection("APRS");
  const c = (cfg as APRSConfig) ?? { Channels: null };
  return (
    <Section
      sectionKey="aprs"
      title="APRS"
      instructions="APRS / AX.25 Bell-202 AFSK receiver. Each channel pins an SDR to an APRS frequency (144.39 MHz is the North-America primary)."
    >
      <ListEditor<APRSChannelConfig>
        label="Channels"
        items={c.Channels}
        onChange={(x) => set({ ...c, Channels: x })}
        makeNew={() => ({ Serial: "", FrequencyHz: 144_390_000, DropBadFCS: false, DropNonUI: false })}
        itemTitle={(ch) => ch.Serial || "channel"}
        emptyHint="No APRS channels."
        renderItem={(ch, setCh) => (
          <div className="space-y-3">
            <div className="grid gap-3 sm:grid-cols-2">
              <TextField label="Serial" value={ch.Serial} onChange={(v) => setCh({ ...ch, Serial: v })} />
              <HzField label="Frequency" value={ch.FrequencyHz} onChange={(v) => setCh({ ...ch, FrequencyHz: v })} />
            </div>
            <BoolField label="Drop bad-FCS frames" value={ch.DropBadFCS} onChange={(v) => setCh({ ...ch, DropBadFCS: v })} />
            <BoolField label="Drop non-UI frames" value={ch.DropNonUI} onChange={(v) => setCh({ ...ch, DropNonUI: v })} />
          </div>
        )}
      />
    </Section>
  );
}
