---
title: "The Hunt, Part 14: The Hunt Cockpit & Testing Discovery Without Radios"
description: The finale — how the interactive hunt cockpit drives the whole sweep-identify-map-export pipeline live over REST from a browser or terminal, and how the entire discovery path is tested end to end with synthesized IQ and canned sources so a find is proven reproducible with no radio attached.
category: deep-dives
keywords: hunt cockpit, live discovery rest api, synthesized iq testing, fileiqsource, acquirer seam, canned sdr source, pipeline test without radio, hunt manager, gophertrunk the hunt
tags: [the-hunt, cockpit, testing, api, architecture, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Hunt"
series_part: 14
---

*Part 14 — the finale — of **The Hunt**. Thirteen posts ago, a stray carrier
showed up in a routine 851–869 MHz survey: strong, digital, unnamed. We swept it
up, settled the gain until it locked, hunted its control channel, watched its
sites in parallel, replayed it from a recording, named as much as the RF allowed,
harvested the aliases riding its traffic, and exported the whole thing four ways.
This post closes the loop from two directions: the interactive **cockpit** that
drives all of that live, and the **tests** that prove the pipeline works with no
radio in the room — which is the only reason you should believe any of it.*

> **TL;DR:** The daemon exposes the hunt as a REST cockpit — `GET /api/v1/hunt`
> plus start / stop / export / capture / commit / cross-reference — by adapting
> the `hunt.Manager` behind one `huntCockpit` type that mirrors the scanner
> cockpit. The browser and TUI drive the same `Manager` the CLI does. And the
> entire discovery pipeline is testable **without hardware**: the `Acquirer` seam
> from Part 1 lets a test inject a `FileIQSource` of synthesized P25 control-channel
> IQ, so `TestRunLiveHunt_CandidateList` proves candidate → identify → decode →
> accumulate produces the same mapped system offline `Discover` does, and
> `TestManagerCaptureSignal` proves the capture path tunes and releases the SDR —
> all under `go test`, no dongle attached.

**Key takeaways**

- **One manager, three cockpits.** `huntCockpit` adapts `hunt.Manager` to the REST
  surface; the web console and TUI poll it, and both drive the exact `Manager` the
  offline CLI wraps — the "one engine, many drivers" pattern all the way to the UI.
- **A trunked hunt opts into the long-dwell monitor.** The cockpit defaults
  `MonitorSeconds` on so a site's slow-cycling status broadcasts arrive; converge-and-stop
  ends it early once identity and topology settle.
- **The `Acquirer` seam makes hardware optional.** Because the manager gets its
  SDR through an injected function, a test hands it a `FileIQSource` and the whole
  live path runs in memory.
- **Synthesized IQ closes the proof.** `siglab.Synthesize` builds a real P25
  control-channel buffer, so a test asserts an actual lock, protocol, and
  control-channel-on-a-site — the discovery pipeline is verified end to end with
  no radio.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| `huntCockpit` | adapts `hunt.Manager` to the REST surface | `cmd/gophertrunk/hunt_cockpit.go` |
| `Start` / `Stop` / `Status` | launch, cancel, snapshot a live run | `hunt_cockpit.go` |
| `CaptureSignal` / `Export` / `Commit` | record a row, serialize, merge to config | `hunt_cockpit.go` |
| `TestRunLiveHunt_CandidateList` | synthesized P25 → full live-hunt assertions | `internal/hunt/livehunt_test.go` |
| `TestManagerCaptureSignal` | canned source → capture tunes + releases | `internal/hunt/capture_signal_test.go` |
| `TestCchuntSystems` | wideband-hosted CCs leave the sequential hunt set | `cmd/gophertrunk/cchunt_systems_test.go` |

## In this post

- **The cockpit** — how the daemon turns the manager into a REST surface.
- **Why a hunt runs the long-dwell monitor** — slow status broadcasts.
- **The Acquirer seam** — the one design choice that makes testing possible.
- **Testing a live hunt with synthesized IQ** — asserting a real lock, no radio.
- **The carrier, found** — the thread, resolved.

## The cockpit: a manager behind a REST surface

The daemon's job in the hunt is small and specific: take the `hunt.Manager` — the
stateful shell around `Discover` from [Part 1]({{ '/blog/deep-dives/the-hunt-01-what-discovery-means/' | relative_url }})
— and expose it to the outside world without leaking its internals. `huntCockpit`
is that adapter, and it mirrors the scanner cockpit so the two feel identical to a
client:

```go
// cmd/gophertrunk/hunt_cockpit.go (shape)
// huntCockpit adapts the daemon's hunt.Manager to the api.HuntCockpit surface
// (GET /api/v1/hunt + start/stop/export/commit). It mirrors scannerCockpit.
type huntCockpit struct {
    mgr     *hunt.Manager
    cfgPath string
    rrAuth  radioreference.Auth
}

func (c huntCockpit) Status() api.HuntStatus {
    st := c.mgr.Status()
    out := api.HuntStatus{
        State: string(st.State), Running: st.Running, Mode: st.Mode,
        Phase: string(st.Progress.Phase), Detail: st.Progress.Detail,
        Sites: st.Sites, Talkgroups: st.Talkgroups, SystemName: st.SystemName,
        // …Signals, Error, GainRecommendations
    }
    if sys, reports, ok := c.mgr.Current(); ok {
        out.System, out.Reports = sys, reports
    }
    return out
}
```

`Start` translates a REST request into `hunt.LiveHuntOptions` and launches the
run; `Stop` cancels it; `Status` is the snapshot the browser and TUI poll to draw
progress. The rest of the surface is the whole workflow the series described,
now as endpoints: `ExportSurvey` and `Export` serialize the run's inventory and
system, `CaptureSignal` records one row and routes it to SigLab/CryptoLab (the
daemon side of Part 10's `-survey-capture`), `RadioReference` runs the Part 13
cross-reference against RR, and `Commit` merges the discovered system into
`config.yaml` through the shared importer. Every one of them is a thin method over
the same `Manager` — no decode logic lives here, only translation.

## Why a hunt runs the long-dwell monitor

One line in `Start` is worth pausing on, because it encodes something the whole
series has been circling. A trunked system reveals its full identity slowly: the
NAC and talkgroups land fast, but the Network Status, RFSS Status, and adjacent-site
broadcasts cycle on a slow timer and rarely fall inside a short buffered dwell. So
the cockpit opts a trunked hunt into the streaming long-dwell monitor by default:

```go
// cmd/gophertrunk/hunt_cockpit.go (shape) — Start
// A trunked-system hunt needs the streaming long-dwell monitor to see the site's
// slow-cycling status broadcasts … a default hunt would surface NAC + talkgroups
// but leave WACN/SystemID/RFSS/Site and neighbors "awaiting status broadcasts".
// Opt into the monitor by default; converge-and-stop ends it early once identity
// + topology settle. Survey runs and explicit caller overrides are left untouched.
if !req.Survey && opts.MonitorSeconds <= 0 {
    opts.MonitorSeconds = defaultHuntMonitorSeconds // 120s cap
}
```

The 120-second figure is only a ceiling. Converge-and-stop normally ends the dwell
well before it, the moment the site's identity, neighbours, and band plan stop
changing — the cap only bounds a site whose status broadcasts never fully arrive.
This is why our carrier came back as a *complete* multi-site map and not just
"P25, some talkgroups": the cockpit waited exactly long enough, and no longer.

## The Acquirer seam: the choice that makes testing possible

Now the other half of the finale, and the more important one. Everything above is
only trustworthy if it's *tested*, and a discovery pipeline is exactly the kind of
thing that's murder to test — it wants an SDR, an antenna, and a live trunked
system on the air. GopherTrunk sidesteps all of that with one design decision made
back in Part 1: the manager never acquires hardware directly. It gets its SDR
through an injected `Acquirer` function. In production the daemon supplies one
backed by the SDR pool; in a test, you supply one backed by a buffer:

```go
// internal/hunt/capture_signal_test.go (shape)
const rate = 2_048_000
src := NewFileIQSource(make([]complex64, rate), rate) // 1 s of IQ, cycles
released := false
mgr, _ := NewManager(ManagerOptions{
    Acquire: func(context.Context, LiveHuntOptions) (IQSource, func(), error) {
        return src, func() { released = true }, nil
    },
    Bus: events.NewBus(256),
})

iq, gotRate, err := mgr.CaptureSignal(context.Background(), 460_000_000, 0.1)
// …asserts: gotRate == rate, len(iq) == 0.1*rate, released == true,
//           and src.TuneCalls() last == 460_000_000
```

That test proves the capture path end to end — it tunes the source to the
requested frequency, grabs exactly the requested duration, and *releases* the SDR
afterward — with no radio anywhere. A companion,
`TestManagerCaptureSignalRefusesDuringRun`, forces the manager into the running
state and asserts `CaptureSignal` returns a busy error, pinning the rule that you
can't grab a capture while a sweep owns the dongle. The `Acquirer` seam is doing
all the work: it's the single door to hardware, and a test simply hands it a
different door.

<figure class="lab-figure">
<svg viewBox="0 0 660 200" width="660" height="200" role="img" aria-label="The Acquirer seam under test. A test builds a FileIQSource from synthesized P25 control-channel IQ and injects it through the Acquirer function into the hunt Manager, which runs the same sweep, identify, decode, and accumulate pipeline as production. The test then asserts on the resulting DiscoveredSystem: the control channel locked, the protocol is p25, and the candidate frequency is recorded as a control channel on a site.">
  <rect x="10" y="80" width="140" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="80" y="99" text-anchor="middle" fill="var(--accent)" font-size="10">siglab.Synthesize</text>
  <text x="80" y="113" text-anchor="middle" fill="var(--fg-muted)" font-size="9">real P25 CC IQ</text>
  <line x1="150" y1="103" x2="180" y2="103" stroke="currentColor"/><polygon points="180,99 190,103 180,107" fill="currentColor"/>
  <rect x="190" y="80" width="130" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="255" y="99" text-anchor="middle" fill="currentColor" font-size="10">FileIQSource</text>
  <text x="255" y="113" text-anchor="middle" fill="var(--fg-muted)" font-size="9">via Acquirer seam</text>
  <line x1="320" y1="103" x2="350" y2="103" stroke="currentColor"/><polygon points="350,99 360,103 350,107" fill="currentColor"/>
  <rect x="360" y="72" width="140" height="62" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="430" y="90" text-anchor="middle" fill="var(--accent)" font-size="10">hunt.Manager</text>
  <text x="430" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">sweep → identify</text>
  <text x="430" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="9">decode → accumulate</text>
  <line x1="500" y1="103" x2="530" y2="103" stroke="currentColor"/><polygon points="530,99 540,103 530,107" fill="currentColor"/>
  <rect x="540" y="72" width="110" height="62" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="595" y="90" text-anchor="middle" fill="var(--accent)" font-size="10">assert</text>
  <text x="595" y="103" text-anchor="middle" fill="var(--fg-muted)" font-size="9">locked · p25</text>
  <text x="595" y="115" text-anchor="middle" fill="var(--fg-muted)" font-size="9">CC on a site</text>
  <text x="330" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="10">no SDR, no antenna, no live system — the whole live path runs under go test</text>
</svg>
<figcaption>Synthesized P25 IQ, injected through the Acquirer seam, runs the real live-hunt pipeline in memory — the test asserts an actual lock and a mapped system.</figcaption>
</figure>

## Testing a live hunt with synthesized IQ

`CaptureSignal` is the plumbing; the real proof is that a *hunt* — the full
sweep-identify-map — produces a correct system from IQ alone.
`TestRunLiveHunt_CandidateList` does exactly that, and it's the cleanest summary of
the entire series in one test:

```go
// internal/hunt/livehunt_test.go (shape)
iq, meta, _ := siglab.Synthesize(siglab.SynthOptions{Protocol: trunking.ProtocolP25, Format: siglab.FormatF32})
src := NewFileIQSource(iq, uint32(meta.SampleRateHz))
const candHz = 851_000_000
sys, reports, _ := RunLiveHunt(context.Background(), LiveHuntOptions{
    Source: src, Candidates: []uint32{candHz},
    DwellSeconds: float64(len(iq)) / float64(meta.SampleRateHz),
    MinConfidence: 0.3, Name: "Live Test",
})
r := reports[0]
// asserts: r.Protocol == "p25", r.Locked, not skipped/errored;
//          candHz recorded as a control channel on sys.Sites[0];
//          sys.DisplayName() == "Live Test"
```

`siglab.Synthesize` builds a genuine P25 control-channel buffer — not a mock, a
real modulated signal — and the test drives it through `RunLiveHunt` on a candidate
list, skipping the sweep. Then it asserts everything the series has been about: the
protocol identifies as P25, the control channel **locks**, the report is neither
skipped nor errored, and the candidate frequency ends up recorded as a control
channel on a site in the resulting `DiscoveredSystem`. The comment on the test says
it plainly — it "proves the candidate → identify → decode → accumulate path
produces the same kind of mapped system the offline `Discover` does." That
sentence is the whole [offline/live symmetry]({{ '/blog/deep-dives/the-hunt-10-offline-vs-live-surveys/' | relative_url }})
turned into an assertion.

The tests reach past the pipeline into the wiring, too.
`TestCchuntSystems` pins a config-level rule from the
[wideband engine]({{ '/blog/deep-dives/the-hunt-09-wideband-multisite-p25/' | relative_url }}):
control channels already decoded in parallel on a `role: wideband` device must be
removed from the *sequential* hunt supervisor's set, and a system whose control
channels are all wideband-hosted is dropped entirely — with a careful aliasing
guard that the shared input slice is never mutated. Between the synthesized-IQ
pipeline tests and these wiring tests, the discovery path is verified from the
symbols up to the daemon's scheduling, all under `go test`, all without a dongle.

### How that principle shaped the Go code

- **The seam is the same one everywhere.** `huntCockpit` drives the `Manager`, and
  the `Manager` gets hardware through `Acquire` — production injects the pool, a
  test injects a `FileIQSource`. There is no separate "test mode."
- **Synthesized signals, not mocks.** `siglab.Synthesize` produces real modulated
  IQ, so the tests exercise the actual demod, framing, and FEC — a mock would prove
  only that the plumbing is connected, not that the decode works.
- **The cockpit adds no logic to test.** Because it's a thin adapter, its
  correctness is the manager's correctness plus a translation, and the manager is
  what the pipeline tests already cover.

## The carrier, found

Fourteen posts ago there was a smudge of power at the edge of a routine survey —
strong, digital, and belonging to nothing we'd named. We asked the open question:
*is there a trunked system here, and what and where is it?* Now there's an answer,
and it's a file. The sweep pulled the carrier out of the noise floor; the
classifier called it digital and not encrypted; the auto-gain settled the front
end until it locked; the CC hunter confirmed a P25 control channel; the wideband
engine watched its neighbour sites decode in parallel; the accumulator folded
them into one `DiscoveredSystem` with a band plan and a talkgroup list; naming
gave the system a stable handle and every carrier a Service and Purpose; the
signalling follower harvested the aliases — and the ciphertext of the one we still
can't read — off its traffic; and the exporters wrote it out as a RadioReference
submission, a TrunkRecorder stanza, a round-tripping bundle, and a SigMF-tagged
capture. The stray carrier is a **named, mapped, exported system**, and every step
of the path is pinned by a test that runs without a radio. That is the whole hunt,
end to end — a blank band turned into a known system, and the proof it can be done
again.

## FAQ

**Do the browser, TUI, and CLI run different hunt code?**
No. The CLI wraps `hunt.Discover`/`RunLiveHunt` directly; the daemon wraps the
same functions in a `hunt.Manager`; and `huntCockpit` is a thin REST adapter over
that manager. The web console and TUI poll the cockpit. One engine, three
drivers — the pattern the whole series (and the whole codebase) uses.

**Why does a hunt keep listening after it's already identified the protocol?**
Because a trunked system reveals its full identity slowly. NAC and talkgroups
arrive fast, but WACN/SystemID/RFSS/Site and neighbours ride slow-cycling status
broadcasts. The cockpit opts a trunked hunt into a long-dwell monitor (capped at
120 s) that converge-and-stop ends early once identity and topology settle.

**How can the discovery pipeline be tested without an SDR?**
Through the `Acquirer` seam. The manager never opens hardware itself — it calls an
injected function that returns an `IQSource`. Production injects the SDR pool;
tests inject a `FileIQSource` built from synthesized IQ, so the entire live path
runs in memory under `go test`.

**Are the tests using real signals or mocks?**
Real signals. `siglab.Synthesize` generates a genuine modulated P25 control-channel
buffer, so `TestRunLiveHunt_CandidateList` exercises the actual demod, framing,
FEC, and accumulation — asserting a true lock and a mapped system, not just that
the plumbing is wired.

**What happens to my find after the hunt ends?**
Whatever you ask: `Export` serializes it to any format, `CaptureSignal` grabs raw
IQ of a row for deeper analysis, `RadioReference` cross-references it against RR,
and `Commit` merges it into `config.yaml` so the daemon can scan it — turning a
discovery straight into a configured, running system.

## Series navigation

**Part 14 of 14** · ←
[Part 13: Exporting Your Finds — RadioReference, TrunkRecorder, SigMF]({{ '/blog/deep-dives/the-hunt-13-exporting-your-finds/' | relative_url }})
· This is the finale — back to the [series index]({{ '/blog/series/the-hunt/' | relative_url }}).

*Where to next? You've found, mapped, named, and exported a system — now go [operate it live]({{ '/blog/series/operator-cockpit/' | relative_url }}) from a browser or terminal in **The Operator's Cockpit**.*
