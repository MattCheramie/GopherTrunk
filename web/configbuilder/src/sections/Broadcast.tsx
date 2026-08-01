import { Section } from "../components/Section";
import { BoolField, Fieldset, NumberField, TextField } from "../components/fields";
import { ListEditor } from "../components/ListEditor";
import { formatCommaList, parseCommaList } from "../lib/csvList";
import { useSection } from "./useSection";
import type {
  BroadcastConfig,
  BroadcastifyFeed,
  GrantWebhookFeed,
  IcecastFeed,
  OpenMHzFeed,
  RdioScannerFeed,
  WebhookFeed,
} from "../api/types";

const SYSTEMS_HELP = "Comma-separated system names to include; leave empty for every system.";

export function BroadcastSection() {
  const [cfg, set] = useSection("Broadcast");
  const c =
    (cfg as BroadcastConfig) ??
    ({ MinDurationMs: 0, Workers: 0, Broadcastify: null, RdioScanner: null, OpenMHz: null, Icecast: null, Webhook: null, GrantWebhook: null } as BroadcastConfig);
  return (
    <Section
      sectionKey="broadcast"
      title="Broadcast"
    >
      <div className="grid gap-3 sm:grid-cols-2">
        <NumberField
          label="Min duration (ms)"
          value={c.MinDurationMs}
          onChange={(v) => set({ ...c, MinDurationMs: v })}
          help="Drop calls shorter than this from every feed. 0 = stream any length."
        />
        <NumberField
          label="Workers"
          value={c.Workers}
          onChange={(v) => set({ ...c, Workers: v })}
          help="Concurrent upload goroutines. 0 = package default."
        />
      </div>

      <Fieldset legend="Broadcastify Calls">
        <ListEditor<BroadcastifyFeed>
          label="Feeds"
          items={c.Broadcastify}
          onChange={(x) => set({ ...c, Broadcastify: x })}
          makeNew={() => ({ Enabled: true, Name: "", APIKey: "", SystemID: 0, Systems: null })}
          itemTitle={(f) => f.Name || "feed"}
          emptyHint="No Broadcastify feeds."
          renderItem={(f, setF) => (
            <div className="space-y-3">
              <BoolField label="Enabled" value={f.Enabled} onChange={(v) => setF({ ...f, Enabled: v })} />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextField label="Name" value={f.Name} onChange={(v) => setF({ ...f, Name: v })} />
                <TextField label="API key" value={f.APIKey} onChange={(v) => setF({ ...f, APIKey: v })} />
                <NumberField label="System ID" value={f.SystemID} onChange={(v) => setF({ ...f, SystemID: v })} />
                <TextField label="Systems" value={formatCommaList(f.Systems)} onChange={(v) => setF({ ...f, Systems: parseCommaList(v) })} help={SYSTEMS_HELP} />
              </div>
            </div>
          )}
        />
      </Fieldset>

      <Fieldset legend="RdioScanner">
        <ListEditor<RdioScannerFeed>
          label="Feeds"
          items={c.RdioScanner}
          onChange={(x) => set({ ...c, RdioScanner: x })}
          makeNew={() => ({ Enabled: true, Name: "", URL: "", APIKey: "", SystemID: 0, Systems: null })}
          itemTitle={(f) => f.Name || "feed"}
          emptyHint="No RdioScanner feeds."
          renderItem={(f, setF) => (
            <div className="space-y-3">
              <BoolField label="Enabled" value={f.Enabled} onChange={(v) => setF({ ...f, Enabled: v })} />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextField label="Name" value={f.Name} onChange={(v) => setF({ ...f, Name: v })} />
                <TextField label="URL" value={f.URL} onChange={(v) => setF({ ...f, URL: v })} />
                <TextField label="API key" value={f.APIKey} onChange={(v) => setF({ ...f, APIKey: v })} />
                <NumberField label="System ID" value={f.SystemID} onChange={(v) => setF({ ...f, SystemID: v })} />
                <TextField label="Systems" value={formatCommaList(f.Systems)} onChange={(v) => setF({ ...f, Systems: parseCommaList(v) })} help={SYSTEMS_HELP} />
              </div>
            </div>
          )}
        />
      </Fieldset>

      <Fieldset legend="OpenMHz">
        <ListEditor<OpenMHzFeed>
          label="Feeds"
          items={c.OpenMHz}
          onChange={(x) => set({ ...c, OpenMHz: x })}
          makeNew={() => ({ Enabled: true, Name: "", APIKey: "", ShortName: "", Systems: null })}
          itemTitle={(f) => f.Name || "feed"}
          emptyHint="No OpenMHz feeds."
          renderItem={(f, setF) => (
            <div className="space-y-3">
              <BoolField label="Enabled" value={f.Enabled} onChange={(v) => setF({ ...f, Enabled: v })} />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextField label="Name" value={f.Name} onChange={(v) => setF({ ...f, Name: v })} />
                <TextField label="API key" value={f.APIKey} onChange={(v) => setF({ ...f, APIKey: v })} />
                <TextField label="Short name" value={f.ShortName} onChange={(v) => setF({ ...f, ShortName: v })} />
                <TextField label="Systems" value={formatCommaList(f.Systems)} onChange={(v) => setF({ ...f, Systems: parseCommaList(v) })} help={SYSTEMS_HELP} />
              </div>
            </div>
          )}
        />
      </Fieldset>

      <Fieldset legend="Icecast / ShoutCast (live)">
        <ListEditor<IcecastFeed>
          label="Feeds"
          items={c.Icecast}
          onChange={(x) => set({ ...c, Icecast: x })}
          makeNew={() => ({ Enabled: true, Name: "", Host: "", Port: 0, Mount: "", Username: "", Password: "", StreamName: "", Systems: null })}
          itemTitle={(f) => f.Name || "feed"}
          emptyHint="No Icecast feeds."
          renderItem={(f, setF) => (
            <div className="space-y-3">
              <BoolField label="Enabled" value={f.Enabled} onChange={(v) => setF({ ...f, Enabled: v })} />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextField label="Name" value={f.Name} onChange={(v) => setF({ ...f, Name: v })} />
                <TextField label="Host" value={f.Host} onChange={(v) => setF({ ...f, Host: v })} />
                <NumberField label="Port" value={f.Port} onChange={(v) => setF({ ...f, Port: v })} />
                <TextField label="Mount" value={f.Mount} onChange={(v) => setF({ ...f, Mount: v })} placeholder="/stream" />
                <TextField label="Username" value={f.Username} onChange={(v) => setF({ ...f, Username: v })} />
                <TextField label="Password" value={f.Password} onChange={(v) => setF({ ...f, Password: v })} />
                <TextField label="Stream name" value={f.StreamName} onChange={(v) => setF({ ...f, StreamName: v })} />
                <TextField label="Systems" value={formatCommaList(f.Systems)} onChange={(v) => setF({ ...f, Systems: parseCommaList(v) })} help={SYSTEMS_HELP} />
              </div>
            </div>
          )}
        />
      </Fieldset>

      <Fieldset legend="Webhook (JSON POST)">
        <ListEditor<WebhookFeed>
          label="Feeds"
          items={c.Webhook}
          onChange={(x) => set({ ...c, Webhook: x })}
          makeNew={() => ({ Enabled: true, Name: "", URL: "", AuthHeader: "", IncludeAudio: false, Systems: null })}
          itemTitle={(f) => f.Name || "feed"}
          emptyHint="No webhook feeds."
          renderItem={(f, setF) => (
            <div className="space-y-3">
              <BoolField label="Enabled" value={f.Enabled} onChange={(v) => setF({ ...f, Enabled: v })} />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextField label="Name" value={f.Name} onChange={(v) => setF({ ...f, Name: v })} />
                <TextField label="URL" value={f.URL} onChange={(v) => setF({ ...f, URL: v })} placeholder="https://example/hook" />
                <TextField label="Auth header" value={f.AuthHeader} onChange={(v) => setF({ ...f, AuthHeader: v })} placeholder="Bearer …" help="Sent verbatim as the Authorization header. Empty omits it." />
                <BoolField label="Include audio" value={f.IncludeAudio} onChange={(v) => setF({ ...f, IncludeAudio: v })} />
                <TextField label="Systems" value={formatCommaList(f.Systems)} onChange={(v) => setF({ ...f, Systems: parseCommaList(v) })} help={SYSTEMS_HELP} />
              </div>
            </div>
          )}
        />
      </Fieldset>

      <Fieldset legend="Grant webhook (JSON POST per grant)">
        <ListEditor<GrantWebhookFeed>
          label="Feeds"
          items={c.GrantWebhook}
          onChange={(x) => set({ ...c, GrantWebhook: x })}
          makeNew={() => ({ Enabled: true, Name: "", URL: "", AuthHeader: "", Systems: null })}
          itemTitle={(f) => f.Name || "feed"}
          emptyHint="No grant webhook feeds."
          renderItem={(f, setF) => (
            <div className="space-y-3">
              <BoolField label="Enabled" value={f.Enabled} onChange={(v) => setF({ ...f, Enabled: v })} />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextField label="Name" value={f.Name} onChange={(v) => setF({ ...f, Name: v })} />
                <TextField label="URL" value={f.URL} onChange={(v) => setF({ ...f, URL: v })} placeholder="https://example/grants" />
                <TextField label="Auth header" value={f.AuthHeader} onChange={(v) => setF({ ...f, AuthHeader: v })} placeholder="Bearer …" help="Sent verbatim as the Authorization header. Empty omits it." />
                <TextField label="Systems" value={formatCommaList(f.Systems)} onChange={(v) => setF({ ...f, Systems: parseCommaList(v) })} help={SYSTEMS_HELP} />
              </div>
            </div>
          )}
        />
      </Fieldset>
    </Section>
  );
}
