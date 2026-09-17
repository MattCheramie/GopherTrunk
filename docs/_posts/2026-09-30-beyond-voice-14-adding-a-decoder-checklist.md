---
title: "Beyond Voice, Part 14: Adding a Decoder in a Day — The Checklist"
description: "The finale — the eleven places a new GopherTrunk decoder touches, from front end to config.example.yaml, with the real FleetSync file for each; the three tests that police the pattern and the CI job vitest cannot replace; a decision table for message decoders versus trunked protocols; and the honest list of what this series leaves open."
category: deep-dives
keywords: adding a decoder checklist, sdr decoder wiring pattern, go event bus sqlite rest panel, config builder field help test, tsc noemit vitest typecheck, doctor preflight storage path, config example yaml rule, fleetsync decoder wiring, decoder verification gates, gophertrunk adding a decoder
tags: [beyond-voice, checklist, architecture, testing, methodology, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Beyond Voice"
series_part: 14
---

*Part 14 — the last — of **Beyond Voice**, a 14-part deep dive into
everything GopherTrunk decodes that is not a trunked voice call — the AFSK
signalling formats, paging, APRS, ADS-B, AIS, DSC, LoRa, M17 and the amateur
digital-voice family — and the one eleven-place wiring pattern that carries
each of them from a burst on the air to a row in the web console.
[Part 13]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }})
finished the tour of physical layers; this closing part turns the pattern
into a checklist, names the tests that fail when a place is missed, sorts
message decoders from trunked protocols, and lists what remains open.*

> **TL;DR:** A GopherTrunk decoder is done when eleven places agree;
> FleetSync is the worked example: protocol core + `afsk` front end
> (`internal/radio/fleetsync`), `events.KindFleetSyncMessage`,
> `storage.FleetSyncLog` + the `fleetsync_log` table + `decoderLogTables`,
> `GET /api/v1/fleetsync/messages`, the `/fleetsync` panel + nav entry,
> `FleetSyncConfig` + the `fleetsync` tab key, FieldMeta help, the Config
> Builder `types.ts` + section, the daemon wiring, the `doctor` "needs
> storage.path" list, and the `config.example.yaml` block + `docs/fleetsync.md`.
> Three tests police it — `TestFieldHelpCoverage`,
> `TestConfigSchemaCoveredByWebBuilder`, and the web `registry.test.ts` /
> `App.panels.test.tsx` route lists — plus the `web-typecheck` CI job, because
> `vitest` never typechecks. Trunked protocols take a different route
> (`Protocol` enum, pipeline factory, composer voice kind, recorder vocoder
> map). M17 shows a decoder that stopped short: REST but no panel.

**Key takeaways**

- **The pattern is the product.** Twelve physical layers reduce to one
  shape because everything above the framer — bus, storage, REST, panel,
  config, help, doctor, example, docs — is identical machinery.
- **Tests police places, not behaviour.** `TestFieldHelpCoverage` and
  `TestConfigSchemaCoveredByWebBuilder` walk `config.Config` by reflection;
  the web route tests walk `App.tsx`.
- **Green `npm test` is not a built console.** `vitest` never typechecks;
  only `tsc --noEmit` catches an unused parameter, so `web-typecheck` runs
  it over all five SPAs.
- **Wired is not verified.** All eleven places can be present and the
  decoder still synthetic-green; the open list below is the ledger.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Field help guard | every exported `config` field needs `FieldMeta.Help` | `internal/configbuilder/fieldmeta_test.go` |
| Web schema guard | every config field appears in the builder's `types.ts` | `internal/configbuilder/webschema_test.go` |
| TUI twin | every leaf gets a handled widget kind | `internal/configtui/meta_drift_test.go` (`TestEveryLeafEditable`) |
| Route guards | every `<Route>` has a nav entry and mounts clean | `web/src/nav/registry.test.ts`, `web/src/App.panels.test.tsx` |
| Typecheck job | `tsc --noEmit` over all five SPAs | `.github/workflows/ci.yml` (`web-typecheck`) |
| Doctor preflight | names decoders configured without `storage.path` | `cmd/gophertrunk/preflight.go` |
| Retention membership | decoder log tables swept by `retention.log_days` | `internal/storage/retention.go` (`decoderLogTables`) |

## In this post

- **The eleven places** — one table, FleetSync's file at each.
- **The three tests that police it** — reflection walks and route lists.
- **The CI job vitest cannot replace** — `web-typecheck` and TS6133.
- **Message decoder or trunked protocol?** — the decision table, and M17.
- **What's still open** — the honest ledger.
- **Where to go from here** — the series index and its siblings.

## The eleven places

[Part 1]({{ '/blog/deep-dives/beyond-voice-01-eleven-place-pattern/' | relative_url }})
introduced the pattern through FleetSync's landing
([Part 4]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }}));
here it is as the checklist, with the real file at every step and the test
that fails when the row is skipped.

| # | Place | FleetSync's file | Policed by |
|---|---|---|---|
| 1 | Protocol core + front end | `internal/radio/fleetsync/framer.go`, `fleetsync/afsk/receiver.go` | `TestReceiverPublishesRealAirBurstsOnBus` |
| 2 | Bus kind | `internal/events/bus.go` (`KindFleetSyncMessage = "fleetsync.message"`) | `docs/api-events.md` passthrough list |
| 3 | Storage log + table + retention | `internal/storage/fleetsynclog.go`, `sqlite.go` (`fleetsync_log`), `retention.go` | `TestFleetSyncLogSweptByRetention` |
| 4 | REST route + provider | `internal/api/handlers_fleetsync.go`, `server.go` (`Options.FleetSync`) | `TestFleetSyncMessagesReturns503WhenNotWired` |
| 5 | Web client, panel, route, nav | `web/src/api/fleetsync.ts`, `panels/FleetSync.tsx`, `App.tsx`, `nav/registry.ts` | `registry.test.ts`, `App.panels.test.tsx`, `FleetSync.test.tsx` |
| 6 | Config struct + tab key | `internal/config/config_peripherals.go` (`FleetSyncConfig`), `config.go` (`"fleetsync": true`) | `TestConfigSchemaCoveredByWebBuilder` |
| 7 | Field help | `internal/configbuilder/fieldmeta.go`, `sections.go` | `TestFieldHelpCoverage`, `TestSectionMetaComplete` |
| 8 | Config Builder | `web/configbuilder/src/api/types.ts`, `sections/FleetSync.tsx`, `sections/index.tsx` | `TestConfigSchemaCoveredByWebBuilder`, `web-typecheck` |
| 9 | Daemon wiring | `cmd/gophertrunk/daemon.go` (receivers, `NewFleetSyncLog`, provider) | `TestDaemonWiresFleetSyncChannels`, `TestDaemonOpensFleetSyncLogWithStorage` |
| 10 | Doctor preflight | `cmd/gophertrunk/preflight.go` (`needs = append(needs, "fleetsync")`) | the WARN text below |
| 11 | Example config + docs | `config.example.yaml` (`fleetsync:` block), `docs/fleetsync.md`, `CHANGELOG.md` | the 4 Sep rule; `guide-advanced.md` index |

Three rows are where earlier landings went wrong. **Row 3** is one
generic: `storage.FleetSyncLog` embeds `*eventLog[FleetSyncMessage]`, the
shared subscribe → drain → close lifecycle, so a new decoder supplies only
its `insert` and `Recent` SQL — and must add its table to
`decoderLogTables`, or `retention.log_days` never sweeps it
([Recording Part 10]({{ '/blog/deep-dives/recording-streaming-10-call-log-sqlite/' | relative_url }})).
**Row 4** returns a bare array; the History panel once read `r.rows` from
a `{"calls": […]}` envelope and rendered "No calls" for every query, so the
row 5 panel test renders a real burst. **Row 10** exists because message decoders surface *only*
through SQLite: with `storage.path` empty the receiver runs, the route
returns 503 and the panel stays empty — a silent failure (#565) that
`doctor` now names:

```
WRN storage.path is empty but these decoders need it to surface decoded
    messages: paging, fleetsync — they will run, but their REST endpoints
    (e.g. /api/v1/pager/messages) return 503 and the web panels stay empty.
    Set storage.path to enable persistence.
```

Row 11's rule dates from 4 September, when `diversity_capture_format`
shipped in code but not in `config.example.yaml`.

## The three tests that police it

The guards walk the *structure*, so they fail the day a place is skipped.
The first walks `config.Config` by reflection and demands help text for
every exported field of every struct in `internal/config`:

```go
// internal/configbuilder/fieldmeta_test.go (shape)
func TestFieldHelpCoverage(t *testing.T) {
    walk = func(rt reflect.Type) {
        /* only structs whose PkgPath ends in internal/config */
        for i := 0; i < rt.NumField(); i++ {
            f := rt.Field(i)
            if !isRoot && FieldMetaFor(rt.Name(), f.Name).Help == "" {
                missing = append(missing, rt.Name()+"."+f.Name)
            }
            walk(f.Type)
        }
    }
    walk(reflect.TypeOf(config.Config{}))
    /* t.Errorf listing every missing Struct.Field */
}
```

It is load-bearing for both editors: the terminal and web builders read
help from the same registry (the web store fetches `/config/fieldmeta`), so
a `FleetSyncChannelConfig.DropBadCRC` without a `Help` ships in neither. The
second test reads the web builder's TypeScript schema off disk —
`readWebTypes` resolves `web/configbuilder/src/api/types.ts` from
`runtime.Caller` — and fails when a config field has no counterpart there
and is not on the round-trip allow-list, "a config change that the web
Config Builder would silently drop on save/load". The TUI has its twin,
`TestEveryLeafEditable`. The third is on the web side: `registry.test.ts`
holds a `ROUTED_PATHS` list — "every top-level route mounted by App.tsx
must have a registry entry, otherwise it becomes unreachable by navigation
(the pre-overhaul bug, where six DSP panels were orphaned)" — and
`App.panels.test.tsx` mounts each listed route connected and asserts the
error boundary never trips, with `vi.mock("./api/fleetsync")` supplying the
client
([Operator Cockpit Part 14]({{ '/blog/deep-dives/operator-cockpit-14-testing-uis/' | relative_url }})).
Adding `/fleetsync` to `App.tsx` without both lists fails one of them.

## The CI job vitest cannot replace

This guard came from the console, not a decoder. `npm test` runs `vitest`,
which transpiles with esbuild and never typechecks; `npm run build` is
`tsc --noEmit && vite build`. Three `TS6133` unused-parameter errors in
`web/src/api/reconnectingSocket.test.ts` landed on main with green SPA
jobs and broke `make web-build` for an operator. The only job that caught
them was "Windows installer (PR)" — not a required check — so the PR merged
red and read as passed.

`ci.yml` now carries `web-typecheck`, matrixed over `web`,
`web/configbuilder`, `web/siglab`, `web/rfscope` and `web/cryptolab` — the
last two built by no other workflow — running `npm run typecheck` in each. For a decoder landing: run `cd web && npm run typecheck` before
pushing a panel, remember `tsconfig.json` sets `noUnusedLocals` and
`noUnusedParameters` with test files inside `include: ["src"]` (an unused
mock parameter needs a leading underscore), and check the non-required
jobs on the PR too.

## Message decoder or trunked protocol?

Not every decoder in this series took the eleven-place route. The AFSK,
paging, APRS, ADS-B, AIS, DSC, LoRa and M17 decoders are **message
decoders** with their own config section, bus kind, table and panel. The
Part 13 trio are **trunked protocols**: they join the engine, whose Active,
History and recorder are their UI. Decide before touching a file:

| Question | Message decoder (FleetSync shape) | Trunked protocol (dPMR shape) |
|---|---|---|
| Identity | own config section (`fleetsync.channels`) | `Protocol` enum in `internal/trunking/site.go` |
| Channel rate | receiver picks (9600 Hz for 1200-baud FFSK) | `ddcTargetForProtocol` |
| Decode entry | daemon builds receivers from config | factory in the `ccdecoder` pipelines map |
| Events | a new `events.Kind` + payload struct | `KindCCLocked` / `KindGrant` with `trunking.Grant` |
| Storage | own `*_log` table in `decoderLogTables` | `call_log` via the engine |
| UI | own panel, route, nav entry, tab key | Active / History / Scanner, unchanged |
| Voice | none | composer `classifyVoiceKind` + recorder vocoder map |
| Doctor | in the "needs storage.path" list | not needed |

The routes are exclusive by design: the D-STAR landing touched `site.go`,
`pipelines.go`, `composer.go` and `recorder.go` but no `App.tsx`.
M17 is the reminder that the message route must be walked to the end: it
has rows 1–4 and 6–11 — `KindM17LinkSetup`, `m17_log`,
`GET /api/v1/m17/linksetups`, `M17Config`, FieldMeta, `M17.tsx`, the example
block, the doctor entry, `docs/m17.md` — and no row 5: no `/m17` in
`registry.ts`, no panel in `App.tsx`, no `m17` tab key. Nothing fails,
because the route tests police routes that exist. That is the one gap the
checklist cannot automate, and why the list is written down.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="The eleven places as two rows of boxes: data flow on top from front end to web panel, configuration below from config struct to example and docs. Braces mark the tests that police them, and a dashed panel box marks the place M17 never reached.">
  <rect x="8" y="30" width="112" height="30" fill="none" stroke="currentColor"/>
  <text x="64" y="49" text-anchor="middle" fill="currentColor" font-size="8">1 front end + framer</text>
  <rect x="130" y="30" width="96" height="30" fill="none" stroke="currentColor"/>
  <text x="178" y="49" text-anchor="middle" fill="currentColor" font-size="8">2 events.Kind</text>
  <rect x="236" y="30" width="120" height="30" fill="none" stroke="currentColor"/>
  <text x="296" y="49" text-anchor="middle" fill="currentColor" font-size="8">3 storage log + table</text>
  <rect x="366" y="30" width="110" height="30" fill="none" stroke="currentColor"/>
  <text x="421" y="49" text-anchor="middle" fill="currentColor" font-size="8">4 REST /api/v1/…</text>
  <rect x="486" y="30" width="186" height="30" fill="none" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <text x="579" y="49" text-anchor="middle" fill="var(--accent)" font-size="8">5 panel + route + nav (M17: missing)</text>
  <rect x="8" y="110" width="104" height="30" fill="none" stroke="currentColor"/>
  <text x="60" y="129" text-anchor="middle" fill="currentColor" font-size="8">6 config struct</text>
  <rect x="122" y="110" width="96" height="30" fill="none" stroke="currentColor"/>
  <text x="170" y="129" text-anchor="middle" fill="currentColor" font-size="8">7 FieldMeta help</text>
  <rect x="228" y="110" width="120" height="30" fill="none" stroke="currentColor"/>
  <text x="288" y="129" text-anchor="middle" fill="currentColor" font-size="8">8 builder types.ts</text>
  <rect x="358" y="110" width="96" height="30" fill="none" stroke="currentColor"/>
  <text x="406" y="129" text-anchor="middle" fill="currentColor" font-size="8">9 daemon wiring</text>
  <rect x="464" y="110" width="90" height="30" fill="none" stroke="currentColor"/>
  <text x="509" y="129" text-anchor="middle" fill="currentColor" font-size="8">10 doctor</text>
  <rect x="564" y="110" width="108" height="30" fill="none" stroke="currentColor"/>
  <text x="618" y="129" text-anchor="middle" fill="currentColor" font-size="8">11 example + docs</text>
  <path d="M 8 156 L 8 168 L 348 168 L 348 156" fill="none" stroke="var(--accent)"/>
  <text x="178" y="184" text-anchor="middle" fill="var(--accent)" font-size="8">TestFieldHelpCoverage · TestConfigSchemaCoveredByWebBuilder · TestEveryLeafEditable</text>
  <path d="M 486 66 L 486 74 L 672 74 L 672 66" fill="none" stroke="var(--accent)"/>
  <text x="579" y="92" text-anchor="middle" fill="var(--accent)" font-size="8">registry.test.ts · App.panels.test.tsx</text>
  <path d="M 228 200 L 228 212 L 672 212 L 672 200" fill="none" stroke="var(--fg-muted)"/>
  <text x="450" y="228" text-anchor="middle" fill="var(--fg-muted)" font-size="8">web-typecheck: tsc --noEmit — what vitest never runs</text>
</svg>
<figcaption>Eleven places in two rows, and the guards that span them. A test can only police a place that exists — how M17 shipped with row 5 empty.</figcaption>
</figure>

### How the checklist shaped the Go code

- **One generic per layer.** `eventLog[T]` for storage, `FieldMetaFor` for
  help, `useDataPoll` for panels: a new decoder adds data, not machinery.
- **Registries are walked, not listed.** The guards reflect over
  `config.Config` and scan `App.tsx`'s routes — the structure is the
  checklist.
- **Optional providers, nil-safe.** A nil `api.Options.FleetSync` disables
  the route with a 503 that names `storage.path`.
- **The doctor speaks the operator's language.** The preflight WARN names
  the decoder, the endpoint and the fix, because the failure it prevents
  was silent.

## What's still open

Each open item sits at a named gate:

- **FleetSync on a live Kenwood fleet** — offline-verified on the
  [#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184) captures
  (5/6 and 8/8 CRC-valid, Fleet 107 / Unit 1772); the reporter's on-air lab
  test is still open, and #437 with it.
- **M17's row 5 and its voice** — no `/m17` panel or tab key; no Codec 2
  path; Golay matrix and slicer constants synthetic-verified only
  ([Part 12]({{ '/blog/deep-dives/beyond-voice-12-m17-open-digital-voice/' | relative_url }})).
- **YSF's FICH caller** — `DecodeFICHOnAir` and `ProcessFICH` are tested but
  unjoined on the hot path, so a Fusion repeater locks and never grants
  ([Part 13]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }})).
- **dPMR's CSBK channel coding** — the cyclic + rate-¾ convolutional +
  interleave stage is absent; CSBKs are parsed as already-clean bits.
- **D-STAR and dPMR voice tables** — placeholder deinterleaves, synthetic
  round-trip only; the D-STAR header scrambler and interleaver await
  MMDVMHost table confirmation. Each needs a voice capture.
- **Rows 10 and 11 have no test.** A reflection walk diffing
  `config.Config` against `config.example.yaml` would close that.

None is blocked on effort; each is blocked on a capture, a reporter, or a
wiring step whose verification needs one — the closing line of
[From Spec to Shipping Part 14]({{ '/blog/deep-dives/from-spec-to-shipping-14-definition-of-verified/' | relative_url }}),
because it is the project's operating principle: green synthetic is not
on-air correct.

## Where to go from here

The [series index]({{ '/blog/series/beyond-voice/' | relative_url }})
lists all fourteen parts. For the machinery every row rests on,
[SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }}) is the
foundation, starting with
[Part 11]({{ '/blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/' | relative_url }})
where `events.Kind` begins. For the method behind every "unverified on air"
above — [From Spec to Shipping]({{ '/blog/series/from-spec-to-shipping/' | relative_url }}).
Two series launch alongside this one:
[DMR End to End]({{ '/blog/series/dmr-end-to-end/' | relative_url }})
follows the trunked-protocol route to decrypted voice, and
[The Field Notebook]({{ '/blog/series/field-notebook/' | relative_url }})
reads the daemon's log one line family at a time.

## FAQ

**What files does a new GopherTrunk message decoder touch?**
Eleven places: the protocol package and front end, an `events.Kind`, a
storage log with its SQLite table and retention membership, a REST handler,
the web client, panel, route and nav entry, the config struct
and tab key, FieldMeta help, the Config Builder `types.ts` and section, the
daemon wiring, the doctor preflight list, and `config.example.yaml` plus docs.

**Which tests fail if I forget a config field's help text?**
`TestFieldHelpCoverage` in `internal/configbuilder` walks every struct
reachable from `config.Config` by reflection and lists each exported field
without a non-empty `Help` in the FieldMeta registry. Both the terminal and
web Config Builders read that registry, so the field ships in neither
editor until help is written.

**Why does `npm test` pass on code that fails `npm run build`?**
`vitest` transpiles with esbuild and never typechecks; `npm run build` runs
`tsc --noEmit` first. With `noUnusedLocals` and `noUnusedParameters` set,
an unused parameter is a `TS6133` error the tests never see. Run
`npm run typecheck` before pushing; the `web-typecheck` job also runs it.

**Why does my decoder's REST endpoint return 503?**
The daemon started without `storage.path`. Message decoders surface only
through SQLite, so with storage off the receiver runs but nothing persists,
and the handler answers 503 naming the fix. `gophertrunk doctor`
warns at preflight for every message decoder that needs storage.

**Should a new protocol be a message decoder or a trunked protocol?**
If it grants voice onto a channel the engine follows, it is a trunked
protocol: add a `Protocol` enum, a pipeline factory, a composer voice kind
and a vocoder mapping. If it produces messages to log and list —
ANI bursts, pages, positions — it takes the eleven-place route.

## Series navigation

**Part 14 of 14** · ←
[Part 13: D-STAR, System Fusion & dPMR]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }})
· [Back to the series index]({{ '/blog/series/beyond-voice/' | relative_url }})
