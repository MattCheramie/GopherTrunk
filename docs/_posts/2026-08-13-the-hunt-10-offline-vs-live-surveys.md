---
title: "The Hunt, Part 10: Offline vs Live Surveys — Hunting a Recording"
description: How GopherTrunk replays a recorded IQ capture through the exact same sweep, classify, identify, and accumulate pipeline it runs on the air — offline carrier detection with signed offsets, a shared per-candidate routing body, and why a find reproduces from a .cfile byte-for-byte.
category: deep-dives
keywords: offline sdr survey, replay iq capture, sweep identify map, signal classification, offline carrier detection, live survey symmetry, reproducible discovery, cfile replay, gophertrunk the hunt
tags: [the-hunt, survey, dsp, replay, testing, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Hunt"
series_part: 10
---

*Part 10 of **The Hunt**. Our 851–869 MHz carrier is now a mapped, multi-site
system — but every claim about it so far was made **live**, off a radio we can't
hand to anyone else. The one artifact we *can* share is the wideband `.cfile` we
first saw it in. This post is about replaying that recording: pushing it back
through the identical sweep → identify → map pipeline the daemon runs on the air,
and getting the same lock, the same protocol, the same map. That reproducibility
is what turns "I found a system" into a claim someone else can check.*

> **TL;DR:** GopherTrunk's live and offline surveys are the **same pipeline** with
> two front-ends. `RunLiveSurvey` sweeps an SDR; `RunOfflineSurvey` loads capture
> files — but both funnel every carrier through one shared body,
> `classifyAndRoute`, which measures occupied bandwidth, classifies the
> modulation, and routes trunking carriers into the *same* identify → decode →
> accumulate path a live hunt uses. A wideband capture is expanded into its
> constituent carriers by `detectOfflineCarriers` (the live peak detector run on
> an in-memory buffer, with **signed** offsets because a file has no SDR to
> retune), each shifted to baseband and routed identically. So a find reproduces
> from a `.cfile` because the offline path and the live path run byte-for-byte
> the same decode.

**Key takeaways**

- **Offline is the sibling of live, not a reimplementation.** `RunOfflineSurvey`
  mirrors `RunLiveSurvey` exactly the way `Discover` mirrors `RunLiveHunt` — one
  shared `classifyAndRoute` body does the real work for both.
- **A recording has no SDR to retune, so offsets are signed.** `detectOfflineCarriers`
  runs the live `DetectPeaks` on an averaged in-memory spectrum and returns each
  carrier as a signed offset from DC, then frequency-shifts it to baseband with
  an NCO.
- **A survey is a superset of a hunt.** `SignalSurvey` catalogues *every* carrier
  — trunking, analog, paging, wideband — and still folds any trunking control
  channel into the same `DiscoveredSystem` a hunt produces.
- **Reproducibility is the point.** Because the same body decodes a capture and a
  live tune, a `.cfile` you share replays to the same result — the basis for
  bug reports, regression tests, and trustworthy finds.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| `RunLiveSurvey` | sweep an SDR, classify + route every carrier | `internal/hunt/livesurvey.go` |
| `RunOfflineSurvey` | load capture files, route them the same way | `internal/hunt/livesurvey.go` |
| `classifyAndRoute` | the shared per-candidate body (both paths) | `internal/hunt/livesurvey.go` |
| `detectOfflineCarriers` | wideband capture → signed carrier offsets | `internal/hunt/offlinesweep.go` |
| `SignalSurvey` | the full classified-carrier inventory | `internal/hunt/survey.go` |
| `hunt -survey-capture` | record one survey row → SigLab / CryptoLab | `cmd/gophertrunk/survey_capture.go` |

## In this post

- **Two front-ends, one body** — where live and offline diverge, and where they don't.
- **Detecting carriers in a recording** — signed offsets and the NCO shift.
- **The shared route** — classify, then hand trunking carriers to the same identify.
- **The survey artifact** — why it is a strict superset of a hunt.
- **Capturing a row** — the bridge from a survey inventory to deeper analysis.

## Two front-ends, one body

The [opener]({{ '/blog/deep-dives/the-hunt-01-what-discovery-means/' | relative_url }})
made a promise: what you find on a recording is what you'd find on the air. A
survey keeps it structurally. `RunLiveSurvey` and `RunOfflineSurvey` are two
functions, and the difference between them is entirely in how they *get* carriers
— then both hand each carrier to the same `classifyAndRoute`:

```go
// internal/hunt/livesurvey.go (shape)
// RunOfflineSurvey classifies and routes a set of capture files without an SDR —
// the offline sibling of RunLiveSurvey, mirroring how Discover is the offline
// sibling of RunLiveHunt. Each capture is loaded, treated as one baseband
// candidate, classified, and routed through the same body the live survey uses.
func RunOfflineSurvey(captures []CaptureInput, opts LiveHuntOptions) (*SignalSurvey, []CaptureReport, error) {
    // …for each capture: load IQ, then
    ds, rep := classifyAndRoute(sv.System, iq, rate, cand, 0, opts, log)
    // …append ds to the survey, rep to the trunking reports
}
```

The live survey's front-end sweeps the SDR (or probes a candidate list), tunes to
each carrier, and captures a dwell; the offline one reads a file into memory.
After that, they are the same program. That "sibling, not reimplementation" shape
is the same one the hunt uses — `Discover` (offline) and `RunLiveHunt` (live) both
call `decodeAndAccumulate` — and it is the single reason a result transfers
between them.

## Detecting carriers in a recording

A live sweep has a luxury an offline one doesn't: it can retune the SDR, so it
only ever looks at one carrier at DC. A recording is fixed — the interesting
control channel might sit anywhere in the captured band, and it might not be the
loudest thing there. So an offline wideband capture has to be *searched* the same
way a live band is, but in memory. `detectOfflineCarriers` does exactly that,
reusing the live peak detector:

```go
// internal/hunt/offlinesweep.go (shape)
// detectOfflineCarriers finds every carrier in a recorded wideband IQ buffer by
// averaging a windowed power spectrum and running the same peak detector the
// live sweeper uses (DetectPeaks) — but on an in-memory buffer, with no SDR
// retune. It returns the carriers as SIGNED offsets from DC so the caller can
// frequency-shift each to baseband.
func detectOfflineCarriers(iq []complex64, rate uint32, fftSize int, peakOpts PeakOptions) []offlineCarrier

type offlineCarrier struct {
    OffsetHz int64   // signed: a CC can sit either side of the recorded centre
    SNRDb    float32
}
```

The signedness is the whole difference. On the air, a carrier's offset is always
"retune the radio by this much." In a file there is no radio to retune, so the
offset is a **signed** distance from the recorded centre, and the caller shifts
the carrier to baseband with a numerically-controlled oscillator instead:

```go
// internal/hunt/livesurvey.go (shape) — surveySweptCapture
carriers := detectOfflineCarriers(iq, rate, opts.FFTSize, opts.PeakOpts)
routeOpts := opts
routeOpts.AutoTune = false // each carrier is shifted to DC here, not by a tuner
for i, c := range carriers {
    shifted := iq
    if c.OffsetHz != 0 {
        // NCO.Mix shifts +offset → DC, bringing the carrier to baseband.
        shifted = dsp.NewNCO(float64(c.OffsetHz), float64(rate)).Mix(nil, iq)
    }
    ds, rep := classifyAndRoute(sv.System, shifted, rate, cand,
        offlineNeighborHz(carriers, i), routeOpts, log)
    // …append ds/rep
}
```

There is a subtlety hidden in `DetectPeaks`: it maps FFT bins to absolute Hz as
`centre + binOffset` and returns an unsigned frequency, which would underflow for
a carrier below a near-zero centre. `detectOfflineCarriers` sidesteps that with a
synthetic centre of one full sample rate, so every carrier in ±rate/2 maps to a
positive frequency, then recovers the signed offset. It's a small trick, but it
means a control channel *below* the recorded centre is found exactly like one
above it — the recording is treated as a band to sweep, not one channel to
demod.

<figure class="lab-figure">
<svg viewBox="0 0 660 210" width="660" height="210" role="img" aria-label="Two survey front-ends converging on one shared body. On the left, the live path sweeps an SDR and tunes to each carrier. On the right, the offline path loads a capture file and runs the offline carrier detector, shifting each carrier to baseband with an NCO. Both feed the same classifyAndRoute body, which measures bandwidth, classifies modulation, and routes trunking carriers into the shared identify, decode, and accumulate path.">
  <rect x="10" y="24" width="150" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="85" y="40" text-anchor="middle" fill="currentColor" font-size="10">live: sweep SDR</text>
  <text x="85" y="54" text-anchor="middle" fill="var(--fg-muted)" font-size="9">tune each carrier</text>
  <rect x="10" y="146" width="150" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="85" y="162" text-anchor="middle" fill="currentColor" font-size="10">offline: load .cfile</text>
  <text x="85" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">detect + NCO-shift</text>
  <line x1="160" y1="44" x2="252" y2="96" stroke="currentColor"/><polygon points="248,93 258,97 249,102" fill="currentColor"/>
  <line x1="160" y1="166" x2="252" y2="114" stroke="currentColor"/><polygon points="249,108 258,113 248,117" fill="currentColor"/>
  <rect x="258" y="86" width="150" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="333" y="102" text-anchor="middle" fill="var(--accent)" font-size="11">classifyAndRoute</text>
  <text x="333" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="9">bw · classify · route</text>
  <line x1="408" y1="106" x2="440" y2="106" stroke="currentColor"/><polygon points="440,102 450,106 440,110" fill="currentColor"/>
  <rect x="450" y="84" width="200" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="550" y="102" text-anchor="middle" fill="var(--accent)" font-size="10">identify → decode → accumulate</text>
  <text x="550" y="117" text-anchor="middle" fill="var(--fg-muted)" font-size="9">same body as a live hunt</text>
  <text x="330" y="200" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the front-ends differ; everything downstream of the carrier is identical</text>
</svg>
<figcaption>Live and offline surveys differ only in how they obtain carriers. Downstream of that, one shared body classifies and routes — which is why a find reproduces.</figcaption>
</figure>

## The shared route: classify, then identify

Inside `classifyAndRoute`, every carrier goes through the same three steps
regardless of where it came from. It measures **occupied bandwidth** on the
full-rate capture (bounded by the nearest-neighbour spacing so a dense DMR grid
doesn't bridge neighbours into one giant span), channelises to a narrow baseband
stream, and hands both to the blind classifier — analog / digital / paging /
wideband. Then `routeSignal` sends each class to its decoder. The important part
for our purposes is the trunking branch:

```go
// internal/hunt/livesurvey.go (shape) — routeSignal, trunking branch
if survey.IsDigital(ds.Class) {
    if in.opts.IdentifyMinConfidence > 0 && ds.Confidence < in.opts.IdentifyMinConfidence {
        return nil // operator opted into a gate; a real CC is never dropped at the default 0
    }
    return identifyTrunking(sys, ds, in, source, true)
}
```

`identifyTrunking` is where the survey rejoins the hunt: it hands the carrier to
the authoritative SigLab identify on the full-rate capture, and on a control-channel
lock it upgrades the row to `trunk-control` and folds the decode into `sys` — the
very same `DiscoveredSystem` accumulation Part 1 described. A blind-analog carrier
gets a lock-only reconsideration (deep survey) so a real DMR/MPT control channel
whose fragile baud line was lost isn't left mislabelled, but it is only promoted
on a genuine lock, never a weak runner-up guess. The routing has opinions; the
decode does not — it is the shared identify either way.

### How that principle shaped the Go code

- **One body, two entry points.** `classifyAndRoute` takes buffers and config, not
  a source. The live survey fills them from a dwell; the offline survey fills them
  from a file; neither knows which it is.
- **`AutoTune` is a front-end concern.** The offline swept path sets
  `AutoTune = false` because it already shifted each carrier to DC with the NCO —
  the decode is handed a baseband buffer exactly as the live tuner would deliver.
- **Non-finite floats can't escape.** Every row is `sanitize()`d before it is
  stored or streamed — a marginal carrier's `Inf`/`NaN` SNR would otherwise break
  `json.Marshal` and tear down the live survey's WebSocket (issue #648). Offline
  and live share that guard because they share the body.

## The survey artifact is a superset of a hunt

A hunt produces a `DiscoveredSystem`. A survey produces a `SignalSurvey` — the
full classified inventory of *everything* on the swept band — and, when it finds
a trunking control channel, still folds it into `System`:

```go
// internal/hunt/survey.go (shape)
// SignalSurvey … When the survey finds a trunking control channel it still folds
// it into System, so a survey is a strict superset of a hunt: it yields the same
// system map plus the surrounding signal landscape.
type SignalSurvey struct {
    Signals []DetectedSignal  `json:"signals"`
    System  *DiscoveredSystem `json:"system,omitempty"` // exported exactly like a hunt result
}
```

That relationship matters for the offline story. You can hunt a recording (just
the trunked map) or survey it (the map plus every analog repeater, pager, and
wideband allocation the capture contained), and the trunked half is identical
either way because it comes out of the same accumulation. A survey doesn't decode
*more* of the trunking; it decodes the *same* trunking and inventories the rest.

## Capturing a row for deeper analysis

The last piece closes the loop from an inventory back to raw IQ. `gophertrunk
hunt -survey-capture` reads a prior survey's JSON, selects one row — by frequency
in MHz or by `#index` — records raw IQ of just that signal, and routes the
capture to the offline workbench or the crypto toolkit:

```go
// cmd/gophertrunk/survey_capture.go (shape)
// runSurveyCapture records the selected signal and routes the capture. It is the
// list-driven bridge from the full-spectrum survey to deeper analysis: pick a
// row, get a `.cfile` + metadata sidecar, and a ready next step in SigLab or
// CryptoLab.
sig, _ := selectSurveySignal(sv, p.selector)     // "851.0125" or "#3"
written, _ := captureSignalToFile(dev, sig.FreqHz, uint32(rate), p.gain, p.ppm, out, p.seconds)
siglab.WriteMetadata(metaPath, &siglab.Metadata{ /* protocol, rate, centre, format */ })
switch to {
case "siglab":    routeToSigLab(out, rate)       // in-process identify + next command
case "cryptolab": routeToCryptoLab(rep, out, sig.FreqHz, rate)
}
```

This is the practical face of the offline/live symmetry. You survey the air once,
get an inventory, pick the row that intrigued you, capture *it* to a `.cfile`, and
then replay that file — through `RunOfflineSurvey`, through SigLab's identify,
through CryptoLab — as many times as you like, deterministically. The metadata
sidecar carries the protocol so `test`/`replay` can pick up exactly where the
survey left off. Our 851 MHz carrier stops being something you had to be there to
see, and becomes a file you can hand to a reviewer.

## Where this goes next

We can now reproduce our system from a recording. It has sites, control channels,
talkgroups, a band plan — everything except a **name**. Every talkgroup is still
a bare decimal; the system itself is `Unknown-p25-…`. [Part 11]({{ '/blog/deep-dives/the-hunt-11-naming-the-unknown/' | relative_url }})
is about turning those IDs into names: how GopherTrunk aliases a system and its
talkgroups, and how a blind discovery earns human labels from the frequency
allocation and the reference catalog.

## FAQ

**Is an offline survey less accurate than a live one?**
No — it runs the same classify and decode. It can even be *more* thorough, because
you can replay the same capture repeatedly with different settings without the
band changing under you. What it can't do is see traffic that wasn't in the
recording; a short capture may simply not contain a talkgroup that a longer live
dwell would.

**Why does the offline path use signed carrier offsets?**
Because a file has no SDR to retune. A live sweep expresses a carrier as "tune the
radio here"; a recording expresses it as a signed distance from the recorded
centre, and the code shifts the carrier to baseband with an NCO instead of a
tuner. A control channel below the recorded centre is found exactly like one
above it.

**What's the difference between a hunt and a survey?**
A hunt maps one trunked system. A survey catalogues *every* carrier on the band —
trunking, analog, paging, wideband — and still folds any trunking control channel
into the same system map. A survey is a strict superset: the same map plus the
surrounding signal landscape.

**Can I hand someone my capture and have them reproduce my find?**
Yes — that is the whole point of the symmetry. Share the `.cfile` and its metadata
sidecar; `RunOfflineSurvey` (or `siglab -in`) replays it through the identical
pipeline and produces the same lock, protocol, and map. It's the basis for the
repo's regression tests and for actionable bug reports.

**What does `-survey-capture` produce?**
A raw `.cfile` of the selected signal plus a `.metadata.json` sidecar (protocol,
sample rate, centre, format), and a printed next step — a SigLab identify command
or a CryptoLab frames file — depending on the `-to` target.

## Series navigation

**Part 10 of 14** · ←
[Part 9: Wideband Multi-Site P25 — Watching Many Channels at Once]({{ '/blog/deep-dives/the-hunt-09-wideband-multisite-p25/' | relative_url }})
· Next →
[Part 11: Naming the Unknown]({{ '/blog/deep-dives/the-hunt-11-naming-the-unknown/' | relative_url }})
