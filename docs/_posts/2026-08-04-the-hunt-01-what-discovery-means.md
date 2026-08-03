---
title: "The Hunt, Part 1: What Discovery Means — From a Blank Band to a Known System"
description: How GopherTrunk turns an empty stretch of spectrum into a named, mapped trunked system — the sweep, identify, and map pipeline behind the hunt engine, and the one discovery contract that the CLI, the daemon, and the web cockpit all drive.
category: deep-dives
keywords: signal discovery, control channel hunting, wideband sweep, trunked system discovery, sdr survey, sweep identify map, discovered system, gophertrunk the hunt, sdr in go
tags: [the-hunt, discovery, trunking, sdr, architecture, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Hunt"
series_part: 1
---

*Part 1 of **The Hunt**, a 14-part deep dive into how GopherTrunk finds trunked
systems you didn't know were there. Every earlier series started from something
known — a control channel to decode, a grant to record. This one starts from a
**blank band** and works forward to a named, mapped, exportable system. This
opener is the map of that whole journey, and it plants a thread we follow the
rest of the way: a stray carrier that shows up in a routine survey.*

> **TL;DR:** Discovery is a **pipeline**, not a magic button: **sweep** a band
> for carriers, **identify** what each one is, and **map** the trunked ones into
> a single `DiscoveredSystem`. GopherTrunk factors that into one package
> (`internal/hunt`) with one contract — feed it captures or an `IQSource`, get
> back a system map plus a per-carrier `CaptureReport` explaining every outcome.
> The offline CLI, the live daemon `Manager`, and the web cockpit all drive the
> *same* pipeline, so what you find on a recording is what you'd find on the air.

**Key takeaways**

- **Discovery has three stages** — sweep → identify → map — and, like the decode
  pipeline, the details change per band but the order never does.
- **The unit of a find is a `DiscoveredSystem`**, accumulated from many captures;
  every capture also produces a `CaptureReport` so nothing is a silent failure.
- **Not-trunked is skipped, not errored.** A wideband sweep surfaces analog,
  paging, and noise; the pipeline degrades gracefully instead of aborting.
- **One engine, three drivers.** The `hunt.Manager` wraps the same functions the
  offline CLI calls, behind an `Acquirer` seam so the package never imports the
  SDR pool.

## Cheat sheet

| Stage | What it does | Where it lives |
|---|---|---|
| Sweep | scan a band, list candidate carriers by power | `internal/hunt/wideband_sweep.go`, `internal/carriers` |
| Classify | analog / digital / encrypted / trunked? | `internal/survey`, `internal/hunt/enctype.go` |
| Identify | which protocol, at what confidence | `internal/hunt/discover.go` (`Discover`) |
| Decode | lock the control channel, read grants | `internal/hunt/decode.go` |
| Accumulate | fold captures into one system map | `internal/hunt/accumulate.go` |
| Name / export | alias it, write RR / TrunkRecorder / SigMF | `internal/hunt/naming.go`, `export_*.go` |
| Orchestrate | own the live run, publish `hunt.*` events | `internal/hunt/manager.go` (`Manager`) |

## In this post

- **What "discovery" actually is** — and why it's a search, not a decode.
- **The three stages** — sweep, identify, map — and where each one lives.
- **The `DiscoveredSystem`** — the thing a hunt produces, and the report beside it.
- **One engine, three drivers** — how the CLI, daemon, and cockpit share it.
- **The carrier we're chasing** — the thread that runs the series.

## What "discovery" actually is

Every other GopherTrunk series has had the luxury of a **known** starting point.
[Protocol Decoders]({{ '/blog/series/protocol-decoders/' | relative_url }}) began
with a control channel to decode. The
[Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }}) began with
a grant already on the bus. Even
[Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }}) began with a
capture someone had already centered on the interesting thing.

Discovery has none of that. You point a radio at 851–869 MHz in a county you've
never scanned and ask a genuinely open question: *is there a trunked system here,
and if so, what and where is it?* That is a **search problem** wrapped around the
decode problem. The decode is the easy part once you know where to point — the
hard part is the pointing, and the pointing is what `internal/hunt` automates.

The instinct to name the shape is the same one the decoder series leaned on: it
tells you where to look when a hunt comes back empty. No carriers in the sweep is
a *front-end* problem (gain, antenna, band). Carriers but nothing classifies as
digital is a *classification* problem. A digital carrier that never locks is a
*decode* problem. Naming the stages turns "it didn't find anything" into a
question with an address.

## The three stages

A hunt is three stages in a fixed order. The middle one is where a band's
identity lives; the outer two are shared machinery — exactly the pattern the
[decode pipeline]({{ '/blog/deep-dives/protocol-decoders-01-anatomy-of-a-cc-decoder/' | relative_url }})
uses one layer down.

<figure class="lab-figure">
<svg viewBox="0 0 680 156" width="680" height="156" role="img" aria-label="The three-stage discovery pipeline: sweep a band into candidate carriers, identify and classify each carrier, then accumulate the trunked ones into a single discovered system map">
  <rect x="6" y="56" width="120" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="66" y="76" text-anchor="middle" fill="currentColor" font-size="12">sweep</text>
  <text x="66" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="9">band → carriers</text>
  <line x1="126" y1="79" x2="150" y2="79" stroke="currentColor"/><polygon points="150,75 160,79 150,83" fill="currentColor"/>
  <rect x="160" y="56" width="130" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="225" y="76" text-anchor="middle" fill="var(--accent)" font-size="12">identify</text>
  <text x="225" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="9">classify + protocol</text>
  <line x1="290" y1="79" x2="314" y2="79" stroke="currentColor"/><polygon points="314,75 324,79 314,83" fill="currentColor"/>
  <rect x="324" y="56" width="130" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="389" y="76" text-anchor="middle" fill="currentColor" font-size="12">map</text>
  <text x="389" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="9">accumulate system</text>
  <line x1="454" y1="79" x2="478" y2="79" stroke="currentColor"/><polygon points="478,75 488,79 478,83" fill="currentColor"/>
  <rect x="488" y="56" width="186" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="581" y="76" text-anchor="middle" fill="var(--accent)" font-size="12">DiscoveredSystem</text>
  <text x="581" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="9">+ per-carrier reports</text>
  <text x="340" y="134" text-anchor="middle" fill="var(--fg-muted)" font-size="10">not-trunked carriers are classified and set aside, never fatal — the sweep degrades gracefully</text>
</svg>
<figcaption>Discovery is sweep → identify → map. The accent stages carry the answer; the outer stages are the same for every band.</figcaption>
</figure>

**Sweep** turns a band into a ranked list of carriers. It steps a wide receiver
across the range, estimates a power spectrum, and pulls peaks out of the noise
floor — the occupancy grid and peak detector in `internal/carriers`, driven by
`internal/hunt/wideband_sweep.go`. Part 2 and Part 3 are entirely about this
stage.

**Identify** asks, of each candidate, *what is this?* First a coarse
classification — analog vs digital vs encrypted vs empty (`internal/survey`) —
then, for the digital ones, an actual protocol identification that tries to decode
a prefix and reports a confidence. That's the job of `Discover`:

```go
// internal/hunt/discover.go (shape)
// Discover folds every capture into a single DiscoveredSystem. Captures whose
// protocol can't be identified with sufficient confidence are skipped, not
// errored, so a wideband sweep that surfaced non-trunked carriers degrades
// gracefully. Per-capture reports are always returned, even on a nil error.
func Discover(inputs []CaptureInput, cfg DiscoverConfig) (*DiscoveredSystem, []CaptureReport, error)
```

Note the contract in that doc comment: a capture that *can't* be identified is a
**skip**, not a failure. A wideband sweep hands `Discover` a pile of carriers, and
most of them will be a paging transmitter or an analog repeater or plain noise.
The pipeline classifies each, sets the non-trunked ones aside with a reason, and
keeps going. That single decision — degrade, don't abort — is what makes an
unattended survey usable.

**Map** folds the trunked captures into one system. A trunked system has a
control channel, a set of talkgroups, a band plan, maybe multiple sites; a single
capture only ever sees a slice of that. `internal/hunt/accumulate.go` merges the
slices — dedups the control frequencies, unions the talkgroups, reconciles the
identity broadcasts — into the thing a hunt actually produces.

## The `DiscoveredSystem` and its report

Every capture that goes through the pipeline comes back with a `CaptureReport`,
and this struct is worth reading in full because it *is* the honesty contract of
the whole engine — every field is a way of not lying about what happened:

```go
// internal/hunt/discover.go (shape)
type CaptureReport struct {
    Path       string  // which capture this is
    Protocol   string  // identified protocol, or "" if unidentified
    Confidence float64 // identifier confidence [0,1]
    Locked     bool    // did the control channel actually lock?
    ControlHz  uint32  // the decoded control frequency (not the tuned one)
    Talkgroups int     // distinct talkgroups seen on the CC
    ErrorRate  float64 // decode errors / 1000 symbols — a demod-quality proxy
    Encrypted  bool    // any encrypted grant seen
    EncType    string  // algorithm name when known (AES-256, ADP/RC4, …)
    Skipped    bool    // set aside (e.g. not trunked), not an error
    SkipReason string  // why it was skipped, in words
    Error      string  // hard failure, in words
    // …Verdict, IdentityNote for wideband + partial-identity cases
}
```

Three outcomes, and the struct makes all three legible. **Decoded**: `Locked` is
true, `ControlHz` and `Talkgroups` are populated. **Skipped**: `Skipped` is true
with a human `SkipReason` ("not trunked", "below min confidence"). **Errored**:
`Error` carries the message. There is no fourth, silent outcome — the CLI and the
cockpit render these reports verbatim, so a hunt that finds nothing still *tells
you why* on every carrier it looked at.

`ErrorRate` earns its place too: it's the protocol-neutral demod-quality number
(decode-error events per 1000 symbols, straight from Signal Lab's
`Signal.DecodeErrorRate`), and Part 5's auto-gain sweep uses exactly this field
to decide whether nudging the front-end gain improved the lock.

## One engine, three drivers

`Discover` is a pure function — captures in, system out. That's perfect for the
offline CLI, but a *live* hunt needs to acquire an SDR, run for minutes in the
background, stream progress to a cockpit, and be cancellable. That's the
`Manager`, and its design is the same decoupling trick the decoder used with the
event bus: the package must not import the SDR pool, so the daemon injects the
acquisition behind a function seam.

```go
// internal/hunt/manager.go (shape)
// Acquirer obtains an IQSource for one run plus a release callback. The daemon
// supplies this so the hunt package stays free of SDR/pool dependencies;
// release is invoked exactly once when the run ends (success, error, or stop).
type Acquirer func(ctx context.Context, opts LiveHuntOptions) (IQSource, func(), error)

// Manager owns the daemon's single live-hunt run: acquire an SDR, run the
// sweep→identify→map pipeline in the background, publish hunt.* bus events,
// and hold the latest DiscoveredSystem for export/commit.
type Manager struct {
    acquire Acquirer
    bus     *events.Bus
    // …mu, state, progress, sys, reports, cancel
}
```

The `Manager` carries a `RunState` — `idle → running → done | stopped | failed` —
that the REST and TUI cockpits poll, and it publishes `hunt.*` events on the same
[event bus]({{ '/blog/deep-dives/trunking-engine-02-event-bus/' | relative_url }})
every other subsystem uses. So there are three drivers over one engine:

- **The offline CLI** (`gophertrunk hunt …`) calls `Discover`/`DiscoverWideband`
  directly on captures or a live tune, prints reports, and exits.
- **The live daemon** builds a `Manager`, hands it an `Acquirer` backed by the
  SDR pool, and lets the sweep run while the rest of the daemon keeps scanning.
- **The web/TUI cockpit** drives that `Manager` over REST + SSE — start, watch
  `hunt.*` progress, stop, then commit or export the result.

### How that principle shaped the Go code

- **The package never imports the pool.** `hunt` depends on `events`, `storage`,
  `survey`, and `siglab` — not on any SDR driver. The `Acquirer` seam is the only
  door to hardware, and the daemon owns it, so the whole engine is testable with
  a canned `IQSource`.
- **Pure core, stateful shell.** `Discover` is a function; `Manager` is the
  goroutine-owning, mutex-guarded shell around it. The core has no lifecycle to
  get wrong, and the shell has no decoding logic to get wrong.
- **Reports are returned even on success.** `Discover` returns
  `(*DiscoveredSystem, []CaptureReport, error)` — the reports come back beside a
  nil error, so progress and outcomes are never trapped inside an error path.
- **Skips keep the run alive.** Encoding "not trunked" as a `Skipped` report
  rather than an `error` is what lets one bad carrier out of forty not sink the
  survey.

## The carrier we're chasing

> **📡 One carrier we keep coming back to.** Somewhere in a routine 851–869 MHz
> survey there's a strong digital carrier that classifies as *digital, not
> analog, probably not encrypted* — and doesn't obviously belong to any system
> we've named. Over the next thirteen posts we sweep it up (Part 2–3), classify
> it (Part 4), settle the gain until it locks (Part 5), hunt and confirm its
> control channel (Part 6–7), reconstruct its neighbours and sites (Part 9),
> name it (Part 11), harvest the aliases riding its traffic (Part 12), and export
> the finished find (Part 13). Keep it in mind; it's the thread.

Discovery is where GopherTrunk stops being a scanner you *configure* and starts
being one that *tells you what's out there*. The rest of the series is the
machinery that makes that claim true, one stage at a time.

## Where this goes next

[Part 2]({{ '/blog/deep-dives/the-hunt-02-wideband-sweep-engine/' | relative_url }})
drops into the first stage: the wideband sweep engine — how GopherTrunk steps a
receiver across megahertz of spectrum, estimates a power spectrum from short IQ
grabs, and stitches the pieces into one occupancy picture without missing a
carrier at a tile boundary. From there we pull peaks out of the noise (Part 3) and
start classifying what we found (Part 4).

## FAQ

**What's the difference between hunting and scanning?**
Scanning follows a system you've already configured — its control channel, its
talkgroups. Hunting starts without that: it searches a band for carriers,
identifies which are trunked systems, and *produces* the configuration a scanner
would then use. The Hunt series is about that upstream search.

**Does discovery need a radio?**
No. `Discover` takes IQ captures just as happily as a live tune — offline surveys
replay recordings through the exact same sweep→identify→map pipeline. Part 10 is
entirely about the offline-vs-live symmetry, and it's why a find reproduces.

**What is a `DiscoveredSystem`?**
The accumulated result of a hunt: one trunked system's control frequencies, band
plan, talkgroups, identity (WACN/System ID for P25, and so on), and — after Part
11 — a name. It's built by folding many partial captures together, because no
single capture sees the whole system.

**Why does a hunt "skip" carriers instead of failing?**
A wideband sweep surfaces everything with power — analog repeaters, pagers, noise.
Only some are trunked. Encoding a non-trunked carrier as a `Skipped` report with a
reason (rather than an error) lets an unattended survey look at forty carriers and
still finish, explaining each one.

**Is the same engine used live and offline?**
Yes. `Discover` is the pure core; the daemon's `Manager` wraps it for live runs
behind an `Acquirer` seam so the package never imports the SDR pool. CLI, daemon,
and cockpit are three drivers over one engine — which is why lab results transfer
to the air.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: The Wideband Sweep Engine]({{ '/blog/deep-dives/the-hunt-02-wideband-sweep-engine/' | relative_url }})
