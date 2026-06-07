import { Section } from "../components/Section";
import { NumberField, SelectField, TextField } from "../components/fields";
import { ListEditor } from "../components/ListEditor";
import { useStore } from "../store/shared";
import { useSection } from "./useSection";
import type { ToneOutConfig, ToneOutToneConfig, ToneProfileConfig } from "../api/types";

export function ToneOutSection() {
  const [cfg, set] = useSection("ToneOut");
  const c = (cfg as ToneOutConfig) ?? { Profiles: null };
  const systems = useStore((s) => s.config?.Trunking?.Systems) ?? [];
  const systemNames = systems.map((s) => s.Name).filter(Boolean);

  return (
    <Section
      sectionKey="tone_out"
      title="Tone-out"
      instructions="Two-tone / single-tone paging detection. Supply two tones (A then B) for sequential paging, or one for single-tone supervision. Durations are Go duration strings (e.g. 250ms, 1.5s)."
    >
      <ListEditor<ToneProfileConfig>
        label="Profiles"
        items={c.Profiles}
        onChange={(x) => set({ ...c, Profiles: x })}
        makeNew={() => ({
          Name: "",
          AlphaTag: "",
          Tones: [],
          ToleranceHz: 0,
          MagnitudeThreshold: 0,
          MaxGap: "",
          Cooldown: "",
          System: systemNames[0] ?? "",
          GroupID: 0,
        })}
        itemTitle={(p) => p.Name || "profile"}
        emptyHint="No tone-out profiles (detector disabled)."
        renderItem={(p, setP) => (
          <div className="space-y-3">
            <div className="grid gap-3 sm:grid-cols-2">
              <TextField label="Name" value={p.Name} onChange={(v) => setP({ ...p, Name: v })} />
              <TextField label="Alpha tag" value={p.AlphaTag} onChange={(v) => setP({ ...p, AlphaTag: v })} />
              <NumberField label="Tolerance (Hz)" value={p.ToleranceHz} onChange={(v) => setP({ ...p, ToleranceHz: v })} />
              <NumberField label="Magnitude threshold" value={p.MagnitudeThreshold} onChange={(v) => setP({ ...p, MagnitudeThreshold: v })} />
              <TextField label="Max gap" value={p.MaxGap} onChange={(v) => setP({ ...p, MaxGap: v })} placeholder="250ms" />
              <TextField label="Cooldown" value={p.Cooldown} onChange={(v) => setP({ ...p, Cooldown: v })} placeholder="5s" />
              {systemNames.length > 0 ? (
                <SelectField
                  label="System"
                  value={p.System}
                  onChange={(v) => setP({ ...p, System: v })}
                  options={systemNames.map((n) => ({ value: n, label: n }))}
                />
              ) : (
                <TextField label="System" value={p.System} onChange={(v) => setP({ ...p, System: v })} />
              )}
              <NumberField label="Group ID" value={p.GroupID} onChange={(v) => setP({ ...p, GroupID: v })} />
            </div>
            <ListEditor<ToneOutToneConfig>
              label="Tones (A then B)"
              items={p.Tones}
              onChange={(t) => setP({ ...p, Tones: t })}
              makeNew={() => ({ FrequencyHz: 0, MinDuration: "", MaxDuration: "" })}
              itemTitle={(_, i) => `Tone ${i + 1}`}
              emptyHint="Add one tone (single-tone) or two (A then B)."
              renderItem={(t, setT) => (
                <div className="grid gap-3 sm:grid-cols-3">
                  <NumberField label="Frequency (Hz, audio)" value={t.FrequencyHz} onChange={(v) => setT({ ...t, FrequencyHz: v })} />
                  <TextField label="Min duration" value={t.MinDuration} onChange={(v) => setT({ ...t, MinDuration: v })} placeholder="250ms" />
                  <TextField label="Max duration" value={t.MaxDuration} onChange={(v) => setT({ ...t, MaxDuration: v })} placeholder="(0 = no max)" />
                </div>
              )}
            />
          </div>
        )}
      />
    </Section>
  );
}
