import { Section } from "../components/Section";
import { Fieldset, SelectField, TextField } from "../components/fields";
import { ListEditor } from "../components/ListEditor";
import { useSection } from "./useSection";
import type { BasebandConfig, BasebandRecordConfig, BasebandReplayConfig } from "../api/types";

const ROLES = [
  { value: "", label: "(auto)" },
  { value: "control", label: "control" },
  { value: "voice", label: "voice" },
  { value: "auto", label: "auto" },
];

// Loop is *bool (nil defaults to true). Map to a tri-state select.
function loopValue(loop: boolean | null): string {
  if (loop === null || loop === undefined) return "default";
  return loop ? "true" : "false";
}
function loopFromValue(v: string): boolean | null {
  if (v === "default") return null;
  return v === "true";
}

export function BasebandSection() {
  const [cfg, set] = useSection("Baseband");
  const c = (cfg as BasebandConfig) ?? { Record: null, Replay: null };
  return (
    <Section
      sectionKey="baseband"
      title="Baseband"
      instructions="Wideband IQ recording and offline replay. 'Record' taps live tuners to WAV; 'Replay' mounts recorded WAVs as virtual tuners (record at sdr.sample_rate for real-time-correct playback)."
    >
      <Fieldset legend="Record (tap live tuners to WAV)" defaultOpen>
        <ListEditor<BasebandRecordConfig>
          label="Recorders"
          items={c.Record}
          onChange={(x) => set({ ...c, Record: x })}
          makeNew={() => ({ Serial: "", Dir: "" })}
          itemTitle={(r) => r.Serial || "recorder"}
          emptyHint="No recorders."
          renderItem={(r, setR) => (
            <div className="grid gap-3 sm:grid-cols-2">
              <TextField label="Serial" value={r.Serial} onChange={(v) => setR({ ...r, Serial: v })} />
              <TextField label="Directory" value={r.Dir} onChange={(v) => setR({ ...r, Dir: v })} />
            </div>
          )}
        />
      </Fieldset>
      <Fieldset legend="Replay (mount WAVs as virtual tuners)">
        <ListEditor<BasebandReplayConfig>
          label="Replays"
          items={c.Replay}
          onChange={(x) => set({ ...c, Replay: x })}
          makeNew={() => ({ File: "", Serial: "", Role: "", Loop: null })}
          itemTitle={(r) => r.File || "replay"}
          emptyHint="No replay sources."
          renderItem={(r, setR) => (
            <div className="grid gap-3 sm:grid-cols-2">
              <TextField label="File" value={r.File} onChange={(v) => setR({ ...r, File: v })} placeholder="capture.wav" />
              <TextField label="Serial (virtual; blank = auto)" value={r.Serial} onChange={(v) => setR({ ...r, Serial: v })} />
              <SelectField label="Role" value={r.Role} onChange={(v) => setR({ ...r, Role: v })} options={ROLES} />
              <SelectField
                label="Loop on EOF"
                value={loopValue(r.Loop)}
                onChange={(v) => setR({ ...r, Loop: loopFromValue(v) })}
                options={[
                  { value: "default", label: "(default: loop)" },
                  { value: "true", label: "loop" },
                  { value: "false", label: "play once" },
                ]}
              />
            </div>
          )}
        />
      </Fieldset>
    </Section>
  );
}
