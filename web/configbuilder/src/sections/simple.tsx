import { Section } from "../components/Section";
import {
  BoolField,
  NumberField,
  SelectField,
  TextField,
} from "../components/fields";
import { useStore } from "../store/shared";
import type {
  APIConfig,
  AudioConfig,
  DiagnosticsConfig,
  GTConfig,
  LogConfig,
  MetricsConfig,
  RadioReferenceConfig,
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
      instructions="Controls daemon log verbosity and output format."
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
      instructions="Verbose errors print full error chains + stack traces and expand API error envelopes (which expose host/dongle info). Enable only on trusted networks."
    >
      <BoolField
        label="Verbose errors"
        value={cfg.VerboseErrors}
        onChange={(x) => set({ ...cfg, VerboseErrors: x })}
      />
    </Section>
  );
}

export function RadioReferenceSection() {
  const [v, set] = useSection("RadioReference");
  const cfg = v as RadioReferenceConfig;
  return (
    <Section
      sectionKey="radioreference"
      title="RadioReference"
      instructions="Read-only RadioReference.com API credentials. Used by the hunt duplicate check and by this builder's RadioReference browse/import. Can also be supplied via GOPHERTRUNK_RR_KEY / _USER / _PASS env vars instead of storing the secret here."
    >
      <TextField label="API key" value={cfg.APIKey} onChange={(x) => set({ ...cfg, APIKey: x })} />
      <TextField label="Username" value={cfg.Username} onChange={(x) => set({ ...cfg, Username: x })} />
      <TextField
        label="Password"
        type="password"
        value={cfg.Password}
        onChange={(x) => set({ ...cfg, Password: x })}
      />
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
      instructions="HTTP/gRPC listen addresses and access control. allow_mutations lets the web UI write changes (talkgroup edits, settings, this builder's saves)."
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
      instructions="Where the call-log database and the control-channel cache live."
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
      instructions="Per-call WAV recorder output directory and sample rate (4000–48000 Hz)."
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
      instructions="Mount a Prometheus /metrics endpoint on the API HTTP server."
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
      instructions="Background sweeper that ages out call-log rows and recorded files. Zero days disables the corresponding sweep."
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
      instructions="Scan-list mode for the trunking engine. 'all' follows every non-locked-out grant; 'list' follows only talkgroups marked Scan=true."
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
      instructions="Live playback of decoded voice. Sample rate (4000–48000 Hz) should match recordings.sample_rate. Volume is 0–1."
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
      instructions="Show or hide navigation tabs in the operator console. Unchecked tabs are hidden from the nav (the route stays reachable by URL)."
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
