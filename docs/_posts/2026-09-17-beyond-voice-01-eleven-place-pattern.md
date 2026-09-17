---
title: "Beyond Voice, Part 1: Why a Scanner Decodes Data — The Eleven-Place Pattern"
description: "Why a trunking scanner decodes in-band data at all, and the one wiring shape every non-voice decoder in GopherTrunk shares — front end, framer, bus event, SQLite log, REST route, web panel, config editors, doctor preflight and example config — eleven places and three policing tests, walked through FleetSync's landing."
category: deep-dives
keywords: sdr data decoder architecture, in-band signalling scanner, ani decoder sdr, events bus decoder pattern, sqlite decoder log, rest panel decoder, config builder field help, fleetsync mdc1200 aprs, adding a protocol decoder, gophertrunk beyond voice
tags: [beyond-voice, architecture, fleetsync, events, storage, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Beyond Voice"
series_part: 1
---

*Part 1 of **Beyond Voice**, a 14-part deep dive into everything GopherTrunk
decodes that is not a trunked voice call — the AFSK signalling formats, paging,
APRS, ADS-B, AIS, DSC, LoRa, M17 and the amateur digital-voice family — and the
one eleven-place wiring pattern that carries each of them from a burst on the
air to a row in the web console. It stands on three earlier series: the bus of
[SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }}), the
SQLite of
[Recording, Composition & Streaming]({{ '/blog/series/recording-streaming/' | relative_url }})
and the one-contract surface of
[The Operator's Cockpit]({{ '/blog/series/operator-cockpit/' | relative_url }}).
This opener asks why a scanner bothers with data at all, then teaches the
pattern once — with Kenwood FleetSync, the most recent landing, as the worked
example — so the next twelve parts can spend their words on physics.*

> **TL;DR:** A conventional analog channel tells you nothing about who is
> talking — until a 1200-baud FFSK burst at the head of the transmission does.
> GopherTrunk decodes those bursts and a dozen other non-voice signals with
> **one shape**: a DSP front end
> (`internal/radio/fleetsync/afsk`) → a callback-only framer
> (`fleetsync.Framer`) → a bus `Kind` (`events.KindFleetSyncMessage =
> "fleetsync.message"`) → a generic `eventLog[T]` drain into SQLite
> (`storage.FleetSyncLog`, `fleetsync_log`) → a `503`-when-unwired route
> (`GET /api/v1/fleetsync/messages`) → a polling panel (`/fleetsync`) — plus
> the config struct, both config editors, the `doctor` preflight and
> `config.example.yaml`. **Eleven places**, and three
> tests police them: `TestFieldHelpCoverage`,
> `TestConfigSchemaCoveredByWebBuilder` and the web `registry.test.ts` /
> `App.panels.test.tsx` route lists. FleetSync is capture-verified
> ([#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184)); its
> live Kenwood run is still open.

**Key takeaways**

- **Data decoders exist because analog voice is anonymous.** MDC1200 and
  FleetSync give an FM channel the radio-ID column digital systems carry
  natively; APRS, AIS and ADS-B give it a map. None is a call.
- **Every decoder is the same three packages.** A protocol core that frames
  bits through a callback and never imports the bus, a DSP front end that
  owns IQ-to-bits and publishes, and (in older decoders) a thin orchestrator
  between them.
- **The bus is the seam; everything downstream is a subscriber.** The front
  end publishes one `Kind`; storage, REST and the panel never learn the
  decoder exists.
- **The pattern is enforced, not documented.** Three tests fail the build
  on a config field with no help, a web schema missing a field, or a panel
  route with no nav entry.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Protocol core | sync hunt → capture → `DecodeFrame`, callback-only | `internal/radio/fleetsync/framer.go` (`Framer.Push`) |
| DSP front end | FM → resample → FFSK → MM timing → slicer, publishes | `internal/radio/fleetsync/afsk/receiver.go` (`Options.Bus`) |
| Bus kind | the one event every subscriber keys on | `internal/events/bus.go` (`KindFleetSyncMessage`) |
| Storage | generic drain → `INSERT INTO fleetsync_log` | `internal/storage/fleetsynclog.go`, `eventlog.go`, `sqlite.go` |
| REST | `GET /api/v1/fleetsync/messages?limit=N`, 503 unwired | `internal/api/handlers_fleetsync.go`, `server.go` |
| Web panel | 5 s poll: fleet, unit, FS-I/FS-II, block check | `web/src/panels/FleetSync.tsx`, `nav/registry.ts` |
| Config + editors | struct, Builder section, field help | `internal/config/config_peripherals.go`, `internal/configbuilder/fieldmeta.go` |

## In this post

- **Why a scanner decodes data** — the anonymous analog channel and what rides on it.
- **The three-package shape** — core, front end, orchestrator across three decoders.
- **The eleven places, verified in the tree** — FleetSync's landing as the example.
- **The three policing tests** — and the two places only policy guards.
- **What the pattern buys** — the principle, and how it shaped the Go code.

## Why a scanner decodes data

Point a receiver at a conventional analog VHF channel and you get voice and
nothing else: no talkgroup, no radio ID, none of what a control channel hands
the
[trunking engine]({{ '/blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/' | relative_url }}).
The fleets that run those channels solved this decades ago by putting a short
data burst *inside* the audio. Motorola's answer is MDC1200; Kenwood's is
FleetSync. Both are 1200-baud FFSK — audio tones at 1200 and 1800 Hz — keyed
for a fraction of a second at the head of each PTT and carrying the radio's
identity. The `mdc1200` package doc says what an operator wants from it: to
see *which* radio is transmitting, and to surface emergency and status events
"on systems that are otherwise just FM voice."

That is why a trunking scanner grows a data path, and the same machinery
then decodes everything else that is bits rather than a vocoder frame:
pagers (POCSAG, FLEX), APRS over AX.25, ship and aircraft transponders (AIS,
ADS-B), marine distress calling (DSC), LoRa chirps and the M17 link layer.
None is a *call* — no grant, no voice device, no recorder — so none touches
`internal/trunking`. Each needs only a way to get a typed message from a burst
on the air to a row an operator can read; GopherTrunk built that once, and the
shape is the running thread from here to Part 14.

## The three-package shape

Put the three AFSK decoders in the tree side by side and the shape jumps out:

| Layer | APRS (`internal/radio/aprs`) | MDC1200 (`internal/radio/mdc1200`) | FleetSync (`internal/radio/fleetsync`) |
|---|---|---|---|
| Protocol core | `ax25`, `hdlc`, `aprs.go` | `mdc1200.go` (`DecodeFrame`) | `fleetsync.go` + `framer.go` |
| Bit orchestrator | `aprs/receiver` (`Push(bit)`, publishes) | `mdc1200/receiver` (`Push(bit)`, publishes) | none — `Framer` is callback-only |
| DSP front end | `aprs/afsk` (1200/2200 Hz, NRZI) | `mdc1200/afsk` (1200/1800 Hz, NRZ) | `fleetsync/afsk` (1200/1800 Hz, publishes) |

The **protocol core** knows bits and nothing else. `fleetsync.Framer` is the
purest specimen: `Push(bit)` slides a 40-bit register hunting a 24-bit
alternating preamble plus the 16-bit sync word `0xA23E` (or its complement),
captures 260 payload bits, calls `DecodeFrame` and invokes the one callback
`NewFramer` was given. No bus import, no storage import, no clock — a
callback-based framer, the package doc says, "lets it be unit-tested in
isolation." The older decoders put a thin **orchestrator** between core and bus
(`mdc1200/receiver`, `aprs/receiver`); FleetSync moved the publish into the
front end instead.

The **DSP front end** owns IQ-to-bits. All three run the identical chain —
`demod.FM` → `dsp.RealResampler` to 9600 Hz → `demod.FFSK` at the protocol's
tone pair → `sync.MuellerMuller` at 8 samples per bit → a slicer — and differ
only in tone pair, line code and slicer threshold,
[Part 2]({{ '/blog/deep-dives/beyond-voice-02-afsk-ffsk-fundamentals/' | relative_url }})'s
subject. It is the "IQ → symbols → state machine" split
[Protocol Decoders Part 1]({{ '/blog/deep-dives/protocol-decoders-01-anatomy-of-a-cc-decoder/' | relative_url }})
described for control channels, applied to bursts that never become calls.

## The eleven places, verified in the tree

CLAUDE.md's note on the FleetSync landing says adding a decoder "touches
ELEVEN places and three tests police them." Here is that count, every place
named from the tree, following one burst from antenna to browser:

| # | Place | Where FleetSync landed |
|---|---|---|
| 1 | Protocol core | `internal/radio/fleetsync/{fleetsync,framer}.go` — `DecodeFrame`, `Framer` |
| 2 | DSP front end | `internal/radio/fleetsync/afsk/receiver.go` — `New(Options{Bus, BaudHz, DropBadCRC})` |
| 3 | Bus kind | `internal/events/bus.go` — `KindFleetSyncMessage = "fleetsync.message"` |
| 4 | Config schema | `internal/config/config_peripherals.go` — `FleetSyncConfig`; `config.go` — `KnownUITabs["fleetsync"]` |
| 5 | Daemon wiring | `cmd/gophertrunk/daemon.go` — construct, `spawn`, open the log, inject the provider |
| 6 | Storage | `internal/storage/fleetsynclog.go`, `sqlite.go` (`fleetsync_log`), `retention.go` (`decoderLogTables`) |
| 7 | REST | `internal/api/handlers_fleetsync.go` — `FleetSyncProvider`, `GET /api/v1/fleetsync/messages` |
| 8 | Web panel | `web/src/panels/FleetSync.tsx`, `App.tsx` route, `nav/registry.ts` entry |
| 9 | Config editors | `web/configbuilder/src/sections/FleetSync.tsx`, `api/types.ts`; `internal/configbuilder/fieldmeta.go` |
| 10 | Preflight | `cmd/gophertrunk/preflight.go` — `needs = append(needs, "fleetsync")` |
| 11 | Paper trail | `config.example.yaml`, `docs/fleetsync.md`, `docs/api-events.md` |

Three of these are where the decoupling lives.

**The publish.** The framer callback is the only place the decoder meets the
bus. It honours `DropBadCRC` for the bus alone — an `OnMessage` caller still
sees every burst — and converts the protocol `Message` into the payload the
rest of the chain shares:

```go
// internal/radio/fleetsync/afsk/receiver.go (shape)
func (r *Receiver) onFrame(m fleetsync.Message) {
    if r.onMessage != nil { r.onMessage(m) }
    if r.bus == nil { return }
    if !m.CRCOK && r.dropBadCRC { r.burstsDropped.Add(1); return }
    r.bus.Publish(events.Event{Kind: events.KindFleetSyncMessage,
        Timestamp: time.Now(), Payload: MessageToStorage(m, time.Now())})
}
```

**The drain.** Every per-domain log writer used to carry a byte-identical
copy of subscribe → filter one `Kind` → insert → close. They now embed one
generic, `eventLog[T]` (`internal/storage/eventlog.go`), so a decoder's
storage layer is an `insert` and a `Recent`; the subscription is taken at
construction so nothing published before `Run` is lost.

**The read surface.** `internal/api` never imports storage's concrete type.
The daemon adapts the log into a one-method `FleetSyncProvider`, and with none
injected the handler answers `503` naming the fix: "fleetsync subsystem not
enabled (set storage.path in config to persist and view decoded messages)" —
the optional-provider rule of
[Operator Cockpit Part 1]({{ '/blog/deep-dives/operator-cockpit-01-one-engine-many-frontends/' | relative_url }}).

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="The eleven-place decoder pattern as a chain: SDR IQ enters the DSP front end, which feeds bits to the callback-only framer and publishes the returned message as one bus Kind; the eventLog drain writes a SQLite row, the REST route reads it through a provider, and the web panel polls the route. A line beneath lists the config struct, Builder schema, field help, doctor preflight and config.example.yaml, with the three policing tests named under the places they guard.">
  <text x="50" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="10">SDR IQ →</text>
  <rect x="100" y="34" width="110" height="48" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="155" y="54" text-anchor="middle" fill="var(--accent)" font-size="10">front end (afsk)</text>
  <line x1="210" y1="50" x2="240" y2="50" stroke="currentColor"/>
  <rect x="240" y="34" width="96" height="48" rx="5" fill="none" stroke="currentColor"/>
  <text x="288" y="54" text-anchor="middle" fill="currentColor" font-size="10">Framer</text>
  <line x1="155" y1="82" x2="155" y2="118" stroke="currentColor"/>
  <rect x="100" y="118" width="110" height="34" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="155" y="134" text-anchor="middle" fill="var(--accent)" font-size="10">events.Bus</text>
  <text x="155" y="146" text-anchor="middle" fill="var(--fg-muted)" font-size="8">"fleetsync.message"</text>
  <line x1="210" y1="135" x2="250" y2="135" stroke="currentColor"/>
  <rect x="250" y="118" width="100" height="34" rx="5" fill="none" stroke="currentColor"/>
  <text x="300" y="134" text-anchor="middle" fill="currentColor" font-size="10">eventLog[T]</text>
  <line x1="350" y1="135" x2="390" y2="135" stroke="currentColor"/>
  <rect x="390" y="118" width="120" height="34" rx="5" fill="none" stroke="currentColor"/>
  <text x="450" y="134" text-anchor="middle" fill="currentColor" font-size="10">REST route</text>
  <line x1="510" y1="135" x2="550" y2="135" stroke="currentColor"/>
  <rect x="550" y="118" width="122" height="34" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="611" y="134" text-anchor="middle" fill="var(--accent)" font-size="10">/fleetsync panel</text>
  <text x="611" y="146" text-anchor="middle" fill="var(--fg-muted)" font-size="8">registry/App.panels tests</text>
  <line x1="6" y1="182" x2="672" y2="182" stroke="var(--fg-muted)" stroke-dasharray="3 4"/>
  <text x="340" y="206" text-anchor="middle" fill="currentColor" font-size="10">config struct · Builder types.ts · fieldmeta help · doctor preflight · config.example.yaml</text>
  <text x="340" y="226" text-anchor="middle" fill="var(--fg-muted)" font-size="8">TestConfigSchemaCoveredByWebBuilder · TestFieldHelpCoverage · (doctor list and example: policy, not tests)</text>
</svg>
<figcaption>The decode chain across the top is the same for every non-voice decoder; the row beneath makes it configurable and discoverable. Only the places with a test named under them are enforced.</figcaption>
</figure>

## The three policing tests

Eleven places would rot if they relied on memory. Three tests guard three
ways the chain silently breaks.

**`TestFieldHelpCoverage`** (`internal/configbuilder/fieldmeta_test.go`)
walks every struct reachable from `config.Config` by reflection and fails on
any exported field whose `FieldMeta` has an empty `Help`. Both editors — the
Bubbletea TUI form and the web Config Builder — source help from that one
registry
([Operator Cockpit Part 13]({{ '/blog/deep-dives/operator-cockpit-13-reflect-driven-config-form/' | relative_url }})).
FleetSync's five entries are what the test demanded, down to `BaudHz`'s
"0 = 1200 baud (FleetSync); 2400 is also accepted."

**`TestConfigSchemaCoveredByWebBuilder`** (`internal/configbuilder/webschema_test.go`)
compares the same walk against the web builder's hand-typed TypeScript schema,
`web/configbuilder/src/api/types.ts`, and fails when a Go field has no
counterpart — "a config change that the web Config Builder would silently
drop on save/load." The `FleetSyncChannelConfig` interface exists because it
refused to pass without one.

**`registry.test.ts` and `App.panels.test.tsx`** (`web/src`) guard a routed
panel with no nav entry, or one that mounts nothing: the first asserts "a nav
entry for every routed panel (no orphans)" against a `ROUTED_PATHS` list that
now carries `/fleetsync`; the second mounts every route with the API clients
mocked, the net
[Operator Cockpit Part 14]({{ '/blog/deep-dives/operator-cockpit-14-testing-uis/' | relative_url }})
built.

Two places are guarded by policy instead. The `doctor` preflight list —
`paging`, `aprs`, `ais`, `dsc`, `mdc1200`, `fleetsync`, `m17` — exists because
[issue #565](https://github.com/MattCheramie/GopherTrunk/issues/565) had a
POCSAG operator staring at a `503` until they learned about `storage.path`;
a new decoder joins it by hand. And `config.example.yaml` is CLAUDE.md's
4 Sep rule: a key that ships in code but not in the example is a key an
operator guesses at their rig.

## What the pattern buys

The principle underneath is the one
[SDR Internals Part 11]({{ '/blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/' | relative_url }})
stated for the engine — *publish, never call outward* — pushed one level
further. The engine at least knows what a grant is; a FleetSync burst is
invisible to it. The decoder publishes one `Kind` and stops; the daemon is the
only file that knows every subscriber exists, and its knowledge is a
construction loop and a `spawn` per receiver — named
`fleetsync-<serial>-<hz>`, marked non-essential, looking up
`d.iqBrokers[spec.serial]`, calling `SetCenterFreq` and handing a
single-channel IQ subscription to `rcv.Process`. Non-essential is
load-bearing: a missing SDR logs `WRN fleetsync: SDR not found, skipping
receiver` and the trunking pipeline keeps running; a malformed
`fleetsync.channels` entry becomes a startup warning and a `nil` slot
(`TestDaemonWiresFleetSyncChannels`). Retention rides along free:
`fleetsync_log` is one string in `decoderLogTables`, so
[the sweeper]({{ '/blog/deep-dives/recording-streaming-11-retention-housekeeping/' | relative_url }})
deletes its rows under `retention.log_days` (`TestFleetSyncLogSweptByRetention`).

What the pattern does *not* buy is on-air truth. Every layer is tested — the
bus path against a committed real-air slice
(`TestReceiverPublishesRealAirBurstsOnBus`: the FleetSync-II capture reaches
the bus as Fleet 107 / Unit 1772), storage, REST and panel with fakes. But the
daemon wiring has not seen a live Kenwood fleet; CLAUDE.md files that under
the #764/#771 discipline ("synthetic + offline ≠ on air"), and the reporter's
lab test is the open gate.
[Part 4]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }})
tells that story in full.

### How the pattern shaped the Go code

- **Framers take a callback, front ends take a bus.** `fleetsync.NewFramer`
  panics without `onMsg` and imports nothing above `math/bits`; `afsk.New`
  requires `OnMessage` *or* `Bus` (`TestNewRequiresASink`).
- **One generic drain, per-table SQL only.** `eventLog[T]` owns subscribe,
  filter, insert and close; `FleetSyncLog` is an `insert` and a `Recent`.
- **Providers are one-method interfaces the daemon fills.**
  `api.FleetSyncProvider` has `RecentFleetSyncMessages(limit)`; the adapter
  lives in `daemon.go`, so `internal/api` compiles without storage's types.
- **Failure is a warning and a nil slot.** Receivers are index-aligned with
  their specs so a skipped entry keeps its position; a data decoder can never
  cost an operator their calls.

## Where this goes next

The pattern is fixed; the physics varies. Every AFSK decoder above asks the
same question — which of two audio tones is present in this bit period — and
[Part 2]({{ '/blog/deep-dives/beyond-voice-02-afsk-ffsk-fundamentals/' | relative_url }})
answers it: how `demod.FFSK` mixes, filters and discriminates a tone pair, why
FFSK's integer-cycle tones differ from Bell 202's, and the slicer lesson that
cost a round.

## FAQ

**Why does a trunking scanner decode non-voice data like MDC1200 or APRS?**
Because analog conventional channels carry no identity of their own. In-band
bursts like MDC1200 and FleetSync add the transmitting radio's ID; APRS, AIS
and ADS-B add positions. GopherTrunk decodes them as messages on the events
bus, never as calls.

**What are the eleven places a new decoder touches in GopherTrunk?**
Protocol core, DSP front end, bus `Kind`, config struct, daemon wiring,
storage log and table and retention list, REST provider and route, web panel
with nav entry, Config Builder section with TypeScript schema and field help,
the `doctor` preflight list, and `config.example.yaml` plus the docs page.

**Which tests fail if a decoder is only half-wired?**
`TestFieldHelpCoverage` fails on a config field with no help text;
`TestConfigSchemaCoveredByWebBuilder` fails when the web builder's `types.ts`
lacks a field; `registry.test.ts` and `App.panels.test.tsx` fail on a routed
panel with no nav entry or one that does not mount.

**Why does the REST route return 503 instead of an empty list?**
Because the log writer only exists when `storage.path` is set. Without it the
receiver still decodes and publishes, but nothing persists, so the handler
answers `503` with the fix in the message rather than pretending the channel
is silent.

**Is the FleetSync decoder verified on air?**
Partly. It decodes the two SDR# captures from issue #1184 — FleetSync-I 5 of 6
bursts, FleetSync-II 8 of 8, Fleet 107 / Unit 1772 — and channelized slices
are committed fixtures. A live run on the reporter's Kenwood radios is still
the open gate.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: AFSK & FFSK — Two Tones, One Bit]({{ '/blog/deep-dives/beyond-voice-02-afsk-ffsk-fundamentals/' | relative_url }})
