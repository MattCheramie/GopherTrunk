import { Section } from "../components/Section";
import {
  BoolField,
  Fieldset,
  HzField,
  NumberField,
  SelectField,
  TextField,
} from "../components/fields";
import { ListEditor } from "../components/ListEditor";
import { RRCredentials } from "../components/RRCredentials";
import { formatCommaList, parseCommaList } from "../lib/csvList";
import { useStore } from "../store/shared";
import type {
  APIConfig,
  AudioConfig,
  ConvChannelConfig,
  DiagnosticsConfig,
  GTConfig,
  LogConfig,
  MetricsConfig,
  RecordingsConfig,
  RetentionConfig,
  ScannerConfig,
  StorageConfig,
  WebConfig,
} from "../api/types";

// useSection returns the current value of a typed config section plus a
// setter that patches just that section into the draft.
function useSection<K extends keyof GTConfig>(key: K): [GTConfig[K], (v: GTConfig[K]) => void] {
  const value = useStore((s) => (s.config ? s.config[key] : null)) as GTConfig[K];
  const patch = useStore((s) => s.patchSection);
  return [value, (v) => patch(key, v)];
}

export function LogSection() {
  const [v, set] = useSection("Log");
  const cfg = v as LogConfig;
  return (
    <Section
      sectionKey="log"
      title="Logging"
    >
      <SelectField
        label="Level"
        value={cfg.Level}
        onChange={(x) => set({ ...cfg, Level: x })}
        options={[
          { value: "", label: "(default: info)" },
          { value: "debug", label: "debug" },
          { value: "info", label: "info" },
          { value: "warn", label: "warn" },
          { value: "error", label: "error" },
        ]}
      />
      <SelectField
        label="Format"
        value={cfg.Format}
        onChange={(x) => set({ ...cfg, Format: x })}
        options={[
          { value: "", label: "(default: text)" },
          { value: "text", label: "text" },
          { value: "json", label: "json" },
        ]}
      />
      <Fieldset legend="Decoded-message log">
        <BoolField
          label="Enabled"
          value={!!cfg.MessageLog?.Enabled}
          onChange={(x) => set({ ...cfg, MessageLog: { ...cfg.MessageLog, Enabled: x } })}
          help="Human-readable per-event log of trunking activity (grants, lock/loss, affiliations)."
        />
        <TextField
          label="Path"
          value={cfg.MessageLog?.Path ?? ""}
          onChange={(x) => set({ ...cfg, MessageLog: { ...cfg.MessageLog, Path: x } })}
          placeholder="/var/log/gophertrunk/messages.log"
        />
        <NumberField
          label="Max size (MB)"
          value={cfg.MessageLog?.MaxSizeMB ?? 0}
          onChange={(x) => set({ ...cfg, MessageLog: { ...cfg.MessageLog, MaxSizeMB: x } })}
          placeholder="16"
        />
      </Fieldset>
      <Fieldset legend="Event log (JSONL)">
        <BoolField
          label="Enabled"
          value={!!cfg.EventLog?.Enabled}
          onChange={(x) => set({ ...cfg, EventLog: { ...cfg.EventLog, Enabled: x } })}
          help="Record every bus event as one JSON line (JSONL/NDJSON), in the same envelope the web UI consumes — a full session for offline inspection."
        />
        <TextField
          label="Path"
          value={cfg.EventLog?.Path ?? ""}
          onChange={(x) => set({ ...cfg, EventLog: { ...cfg.EventLog, Path: x } })}
          placeholder="/var/log/gophertrunk/events.jsonl"
        />
        <NumberField
          label="Max size (MB)"
          value={cfg.EventLog?.MaxSizeMB ?? 0}
          onChange={(x) => set({ ...cfg, EventLog: { ...cfg.EventLog, MaxSizeMB: x } })}
          placeholder="16"
        />
      </Fieldset>
    </Section>
  );
}

export function DiagnosticsSection() {
  const [v, set] = useSection("Diagnostics");
  const cfg = v as DiagnosticsConfig;
  return (
    <Section
      sectionKey="diagnostics"
      title="Diagnostics"
    >
      <BoolField
        label="Verbose errors"
        value={cfg.VerboseErrors}
        onChange={(x) => set({ ...cfg, VerboseErrors: x })}
      />
      <NumberField
        label="Memory limit (MB)"
        value={cfg.MemoryLimitMB ?? 0}
        onChange={(x) => set({ ...cfg, MemoryLimitMB: x })}
        placeholder="0"
      />
      <NumberField
        label="Heartbeat (seconds)"
        value={cfg.HeartbeatSeconds ?? 0}
        onChange={(x) => set({ ...cfg, HeartbeatSeconds: x })}
        placeholder="0"
      />
    </Section>
  );
}

export function RadioReferenceSection() {
  return (
    <Section
      sectionKey="radioreference"
      title="RadioReference Login"
    >
      <RRCredentials />
    </Section>
  );
}

export function APISection() {
  const [v, set] = useSection("API");
  const cfg = v as APIConfig;
  return (
    <Section
      sectionKey="api"
      title="API & Web"
    >
      <TextField
        label="HTTP address"
        value={cfg.HTTPAddr}
        onChange={(x) => set({ ...cfg, HTTPAddr: x })}
        placeholder="127.0.0.1:8080"
        help="Host:port the web UI + REST API listen on. Bind to loopback unless the network is trusted."
      />
      <TextField
        label="gRPC address"
        value={cfg.GRPCAddr}
        onChange={(x) => set({ ...cfg, GRPCAddr: x })}
        placeholder="127.0.0.1:9090 (optional)"
      />
      <BoolField
        label="Allow mutations"
        value={cfg.AllowMutations}
        onChange={(x) => set({ ...cfg, AllowMutations: x })}
        help="Permit write operations from the API. Required for live edits."
      />
      <TextField
        label="rigctld address"
        value={cfg.Rigctld}
        onChange={(x) => set({ ...cfg, Rigctld: x })}
        placeholder="127.0.0.1:4532 (optional)"
        help="Expose the control SDR over the Hamlib rigctld protocol. Leave empty to disable."
      />

      <Fieldset legend="Authentication">
        <SelectField
          label="Mode"
          value={cfg.Auth?.Mode ?? ""}
          onChange={(x) => set({ ...cfg, Auth: { ...cfg.Auth, Mode: x } })}
          options={[
            { value: "", label: "(auto)" },
            { value: "auto", label: "auto (loopback trusted, token on public binds)" },
            { value: "required", label: "required (always need a token)" },
            { value: "disabled", label: "disabled (wide open)" },
          ]}
        />
        <TextField
          label="Token"
          type="password"
          value={cfg.Auth?.Token ?? ""}
          onChange={(x) => set({ ...cfg, Auth: { ...cfg.Auth, Token: x } })}
          help="Inline bearer token. Prefer Token file so the secret stays out of config.yaml."
        />
        <TextField
          label="Token file"
          value={cfg.Auth?.TokenFile ?? ""}
          onChange={(x) => set({ ...cfg, Auth: { ...cfg.Auth, TokenFile: x } })}
          placeholder="/etc/gophertrunk/token"
        />
        <TextField
          label="Trusted networks"
          value={formatCommaList(cfg.Auth?.TrustedNetworks)}
          onChange={(x) => set({ ...cfg, Auth: { ...cfg.Auth, TrustedNetworks: parseCommaList(x) } })}
          placeholder="10.0.0.0/8, 192.168.1.0/24"
          help="Comma-separated CIDRs that bypass the token in auto mode."
        />
      </Fieldset>

      <Fieldset legend="CORS">
        <TextField
          label="Allowed origins"
          value={formatCommaList(cfg.CORS?.AllowedOrigins)}
          onChange={(x) => set({ ...cfg, CORS: { ...cfg.CORS, AllowedOrigins: parseCommaList(x) } })}
          placeholder='null, http://laptop.local:8000, *'
          help="Comma-separated origins echoed in Access-Control-Allow-Origin. Empty disables CORS."
        />
      </Fieldset>

      <Fieldset legend="TLS">
        <TextField
          label="TLS cert (PEM path)"
          value={cfg.TLSCert}
          onChange={(x) => set({ ...cfg, TLSCert: x })}
          placeholder="(optional) /etc/gophertrunk/cert.pem"
        />
        <TextField
          label="TLS key (PEM path)"
          value={cfg.TLSKey}
          onChange={(x) => set({ ...cfg, TLSKey: x })}
          placeholder="(optional) /etc/gophertrunk/key.pem"
          help="Both cert and key must be set to enable TLS."
        />
      </Fieldset>
    </Section>
  );
}

export function StorageSection() {
  const [v, set] = useSection("Storage");
  const cfg = v as StorageConfig;
  return (
    <Section
      sectionKey="storage"
      title="Storage"
    >
      <TextField
        label="Database path"
        value={cfg.Path}
        onChange={(x) => set({ ...cfg, Path: x })}
        placeholder="/var/lib/gophertrunk/calls.db"
      />
      <TextField
        label="CC cache file"
        value={cfg.CCCacheFile}
        onChange={(x) => set({ ...cfg, CCCacheFile: x })}
        placeholder="(optional) cc-cache.json"
        help="JSON cache used by the CC hunter. Empty disables it."
      />
    </Section>
  );
}

export function RecordingsSection() {
  const [v, set] = useSection("Recordings");
  const cfg = v as RecordingsConfig;
  return (
    <Section
      sectionKey="recordings"
      title="Recordings"
    >
      <TextField
        label="Directory"
        value={cfg.Dir}
        onChange={(x) => set({ ...cfg, Dir: x })}
        placeholder="/var/lib/gophertrunk/recordings"
      />
      <NumberField
        label="Sample rate (Hz)"
        value={cfg.SampleRate}
        onChange={(x) => set({ ...cfg, SampleRate: x })}
        placeholder="8000"
      />
      <BoolField
        label="Write raw"
        value={cfg.WriteRaw}
        onChange={(x) => set({ ...cfg, WriteRaw: x })}
      />
      <BoolField
        label="dsd-fme MBE files"
        value={!!cfg.MBEFiles}
        onChange={(x) => set({ ...cfg, MBEFiles: x })}
        help="Also write a dsd-fme-playable sidecar per call (.imb for P25 Phase 1, .amb for DMR/NXDN/P25p2) so recordings decode offline with dsd-fme -r. TETRA/ProVoice/analog produce no file."
      />
      <BoolField
        label="Skip encrypted calls"
        value={cfg.SkipEncrypted}
        onChange={(x) => set({ ...cfg, SkipEncrypted: x })}
        help="Don't record calls flagged encrypted. Aborts and deletes the file if encryption is only detected mid-call."
      />
      <Fieldset legend="Equalizer (CMA blind equalizer)">
        <BoolField
          label="Enabled"
          value={!!cfg.Equalizer?.Enabled}
          onChange={(x) => set({ ...cfg, Equalizer: { ...cfg.Equalizer, Enabled: x } })}
          help="Per-call CMA equalizer for simulcast systems (multiple TX at slightly different delays)."
        />
        <NumberField
          label="Taps"
          value={cfg.Equalizer?.Taps ?? 0}
          onChange={(x) => set({ ...cfg, Equalizer: { ...cfg.Equalizer, Taps: x } })}
          placeholder="8"
        />
        <NumberField
          label="Step size"
          step={0.0001}
          value={cfg.Equalizer?.StepSize ?? 0}
          onChange={(x) => set({ ...cfg, Equalizer: { ...cfg.Equalizer, StepSize: x } })}
          placeholder="0.0001"
        />
      </Fieldset>
      <Fieldset legend="Loudness normalization (EBU R128)">
        <BoolField
          label="Enabled"
          value={!!cfg.Normalize?.Enabled}
          onChange={(x) => set({ ...cfg, Normalize: { ...cfg.Normalize, Enabled: x } })}
          help="Normalize each finished call to a consistent perceptual loudness (pure Go, no ffmpeg)."
        />
        <SelectField
          label="Apply to"
          value={cfg.Normalize?.ApplyTo ?? ""}
          onChange={(x) => set({ ...cfg, Normalize: { ...cfg.Normalize, ApplyTo: x } })}
          options={[
            { value: "", label: "(default: recording)" },
            { value: "recording", label: "recording (rewrite WAV)" },
            { value: "distributed", label: "distributed (outbound MP3 only)" },
            { value: "both", label: "both" },
          ]}
        />
        <NumberField
          label="Target LUFS"
          value={cfg.Normalize?.TargetLUFS ?? 0}
          onChange={(x) => set({ ...cfg, Normalize: { ...cfg.Normalize, TargetLUFS: x } })}
          placeholder="-16"
        />
        <NumberField
          label="True peak (dBTP)"
          step={0.1}
          value={cfg.Normalize?.TruePeakDBTP ?? 0}
          onChange={(x) => set({ ...cfg, Normalize: { ...cfg.Normalize, TruePeakDBTP: x } })}
          placeholder="-1.5"
        />
        <NumberField
          label="Max gain (dB)"
          value={cfg.Normalize?.MaxBoostDB ?? 0}
          onChange={(x) => set({ ...cfg, Normalize: { ...cfg.Normalize, MaxBoostDB: x } })}
          placeholder="12"
        />
      </Fieldset>
    </Section>
  );
}

export function MetricsSection() {
  const [v, set] = useSection("Metrics");
  const cfg = v as MetricsConfig;
  return (
    <Section
      sectionKey="metrics"
      title="Metrics"
    >
      <BoolField label="Enabled" value={cfg.Enabled} onChange={(x) => set({ ...cfg, Enabled: x })} />
    </Section>
  );
}

export function RetentionSection() {
  const [v, set] = useSection("Retention");
  const cfg = v as RetentionConfig;
  return (
    <Section
      sectionKey="retention"
      title="Retention"
    >
      <NumberField
        label="Call-log days"
        value={cfg.CallLogDays}
        onChange={(x) => set({ ...cfg, CallLogDays: x })}
      />
      <NumberField
        label="Decoder-log days"
        value={cfg.LogDays}
        onChange={(x) => set({ ...cfg, LogDays: x })}
      />
      <NumberField
        label="Files days"
        value={cfg.FilesDays}
        onChange={(x) => set({ ...cfg, FilesDays: x })}
      />
      <TextField
        label="Sweep interval"
        value={cfg.Interval}
        onChange={(x) => set({ ...cfg, Interval: x })}
        placeholder="1h"
        help="Go duration string, e.g. 30m, 1h, 24h."
      />
    </Section>
  );
}

export function ScannerSection() {
  const [v, set] = useSection("Scanner");
  const cfg = v as ScannerConfig;
  return (
    <Section
      sectionKey="scanner"
      title="Scanner"
    >
      <SelectField
        label="Scan mode"
        value={cfg.ScanMode}
        onChange={(x) => set({ ...cfg, ScanMode: x })}
        options={[
          { value: "", label: "(default: all)" },
          { value: "all", label: "all" },
          { value: "list", label: "list" },
        ]}
      />

      <Fieldset legend="Control-channel hunter">
        <BoolField
          label="Enabled"
          value={!!cfg.CCHunt?.Enabled}
          onChange={(x) => set({ ...cfg, CCHunt: { ...cfg.CCHunt, Enabled: x } })}
          help="Multi-system control-channel hunter (cycles candidate CCs until one locks)."
        />
        <div className="grid gap-3 sm:grid-cols-3">
          <NumberField label="Dwell (ms)" value={cfg.CCHunt?.DwellMs ?? 0} onChange={(x) => set({ ...cfg, CCHunt: { ...cfg.CCHunt, DwellMs: x } })} />
          <NumberField label="Backoff (ms)" value={cfg.CCHunt?.BackoffMs ?? 0} onChange={(x) => set({ ...cfg, CCHunt: { ...cfg.CCHunt, BackoffMs: x } })} />
          <NumberField label="Max backoff (ms)" value={cfg.CCHunt?.MaxBackoffMs ?? 0} onChange={(x) => set({ ...cfg, CCHunt: { ...cfg.CCHunt, MaxBackoffMs: x } })} />
        </div>
      </Fieldset>

      <Fieldset legend="Conventional channels (analog FM scan list)">
        <ListEditor<ConvChannelConfig>
          label="Channels"
          items={cfg.Conventional}
          onChange={(x) => set({ ...cfg, Conventional: x })}
          makeNew={() => ({
            Label: "",
            FrequencyHz: 0,
            Mode: "nfm",
            SquelchDbFS: 0,
            HangtimeMs: 0,
            ActivityDebounceMs: 0,
            SquelchHysteresisDb: 0,
            Priority: 0,
            TalkgroupID: 0,
            Tone: { Mode: "", CTCSSHz: 0, DCSCode: "" },
          })}
          itemTitle={(ch) => ch.Label || "channel"}
          emptyHint="Fixed-frequency analog channels the scanner sweeps alongside trunking."
          renderItem={(ch, setCh) => (
            <div className="space-y-3">
              <div className="grid gap-3 sm:grid-cols-2">
                <TextField label="Label" value={ch.Label} onChange={(x) => setCh({ ...ch, Label: x })} />
                <HzField label="Frequency" value={ch.FrequencyHz} onChange={(x) => setCh({ ...ch, FrequencyHz: x })} />
                <SelectField
                  label="Mode"
                  value={ch.Mode}
                  onChange={(x) => setCh({ ...ch, Mode: x })}
                  options={[
                    { value: "", label: "(fm)" },
                    { value: "fm", label: "fm" },
                    { value: "nfm", label: "nfm" },
                  ]}
                />
                <NumberField label="Squelch (dBFS)" step={0.1} value={ch.SquelchDbFS} onChange={(x) => setCh({ ...ch, SquelchDbFS: x })} />
                <NumberField label="Hangtime (ms)" value={ch.HangtimeMs} onChange={(x) => setCh({ ...ch, HangtimeMs: x })} />
                <NumberField label="Activity debounce (ms)" value={ch.ActivityDebounceMs} onChange={(x) => setCh({ ...ch, ActivityDebounceMs: x })} />
                <NumberField label="Squelch hysteresis (dB)" step={0.1} value={ch.SquelchHysteresisDb} onChange={(x) => setCh({ ...ch, SquelchHysteresisDb: x })} />
                <NumberField label="Priority" value={ch.Priority} onChange={(x) => setCh({ ...ch, Priority: x })} />
                <NumberField label="Talkgroup ID (0 = positional)" value={ch.TalkgroupID} onChange={(x) => setCh({ ...ch, TalkgroupID: x })} />
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <SelectField
                  label="Tone mode"
                  value={ch.Tone?.Mode ?? ""}
                  onChange={(x) => setCh({ ...ch, Tone: { ...ch.Tone, Mode: x } })}
                  options={[
                    { value: "", label: "none" },
                    { value: "none", label: "none" },
                    { value: "ctcss", label: "CTCSS" },
                    { value: "dcs", label: "DCS" },
                  ]}
                />
                {ch.Tone?.Mode === "ctcss" ? (
                  <NumberField
                    label="CTCSS (Hz, 50–300)"
                    step={0.1}
                    value={ch.Tone?.CTCSSHz ?? 0}
                    onChange={(x) => setCh({ ...ch, Tone: { ...ch.Tone, CTCSSHz: x } })}
                  />
                ) : null}
                {ch.Tone?.Mode === "dcs" ? (
                  <TextField
                    label="DCS code (3 octal digits)"
                    value={ch.Tone?.DCSCode ?? ""}
                    onChange={(x) => setCh({ ...ch, Tone: { ...ch.Tone, DCSCode: x } })}
                    placeholder="023"
                  />
                ) : null}
              </div>
            </div>
          )}
        />
      </Fieldset>

      <BoolField
        label="Manual tune enabled"
        value={!!cfg.ManualTuneEnabled}
        onChange={(x) => set({ ...cfg, ManualTuneEnabled: x })}
        help="Force a VFO tuner so the TUI 'f' key / manual-tune API works even with no static channels (steals one Voice SDR)."
      />
      <BoolField
        label="Manual tune disabled"
        value={!!cfg.ManualTuneDisabled}
        onChange={(x) => set({ ...cfg, ManualTuneDisabled: x })}
        help="Opt out of the auto-detect rule that builds the scanner when ≥2 Voice SDRs are present."
      />
    </Section>
  );
}

export function AudioSection() {
  const [v, set] = useSection("Audio");
  const cfg = v as AudioConfig;
  return (
    <Section
      sectionKey="audio"
      title="Audio"
    >
      <BoolField label="Enabled" value={cfg.Enabled} onChange={(x) => set({ ...cfg, Enabled: x })} />
      <TextField
        label="Output device"
        value={cfg.Device}
        onChange={(x) => set({ ...cfg, Device: x })}
        placeholder="(default sink)"
      />
      <NumberField
        label="Sample rate (Hz)"
        value={cfg.SampleRate}
        onChange={(x) => set({ ...cfg, SampleRate: x })}
        placeholder="8000"
      />
      <NumberField
        label="Buffer (ms)"
        value={cfg.BufferMs}
        onChange={(x) => set({ ...cfg, BufferMs: x })}
        placeholder="80"
      />
      <NumberField
        label="Volume (0–1)"
        step={0.05}
        value={cfg.Volume}
        onChange={(x) => set({ ...cfg, Volume: x })}
      />
      <BoolField label="Muted" value={cfg.Muted} onChange={(x) => set({ ...cfg, Muted: x })} />
    </Section>
  );
}

const KNOWN_TABS = [
  "dashboard", "active", "scanner", "settings", "hunt", "systems", "talkgroups",
  "rids", "history", "events", "cc", "tones", "pagers", "aprs", "ais", "dsc",
  "adsb", "mdc1200", "spectrum", "constellation", "symbols", "bookmarks",
  "metrics", "devices", "import",
];

export function WebSection() {
  const [v, set] = useSection("Web");
  const cfg = v as WebConfig;
  const tabs = cfg.Tabs ?? {};
  const toggle = (tab: string, shown: boolean) => {
    const next = { ...tabs };
    if (shown) delete next[tab];
    else next[tab] = false;
    set({ ...cfg, Tabs: Object.keys(next).length ? next : null });
  };
  return (
    <Section
      sectionKey="web"
      title="Web UI"
    >
      <div className="grid grid-cols-2 gap-x-4 sm:grid-cols-3">
        {KNOWN_TABS.map((tab) => {
          const shown = tabs[tab] !== false;
          return (
            <BoolField
              key={tab}
              label={tab}
              value={shown}
              onChange={(x) => toggle(tab, x)}
            />
          );
        })}
      </div>
    </Section>
  );
}
