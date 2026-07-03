---
title: "RF Scope, Part 8: Expert Info — Anomalies & What They Mean"
description: rfscope's expert analyzer is the RF analog of Wireshark's Expert Information — rule-based anomaly flags at note, warn, and alert severity for frequency hoppers, intermittent emitters, abnormally wide or narrow carriers, noise-like spectra, and the encrypted or obfuscated findings the entropy triage produced.
category: tutorials
keywords: rfscope, expert info, wireshark expert information, anomalies, frequency hopper, intermittent emitter, wide carrier, narrow carrier, noise-like, spectral flatness, encrypted, obfuscated, gophertrunk
tags: [rfscope, rf-analysis, advanced, gophertrunk, dsp, sdr]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Scope"
series_part: 8
---

*Part 8 of **RF Scope**, GopherTrunk's protocol-agnostic RF network analyzer. Every
analyzer so far produced data; this one produces judgments — the "look here" list.*

> **TL;DR:** The `expert` analyzer is Wireshark's Expert Information for RF. It runs
> cheap rules over the finished Scene and emits **anomalies** at three severities —
> `note`, `warn`, `alert`. It flags `hopper` (from topology), `intermittent` (duty
> < 5% with ≥ 3 bursts), `wide-carrier` (> 30 kHz), `narrow-carrier` (< 5 kHz), and
> `noise-like` (spectral flatness ≥ 0.80), plus `encrypted`, `obfuscated`, and
> `unknown-protocol` findings promoted from the entropy triage. It depends on
> `topology`, `timeline`, and `entropy`, and it is the last analyzer to run.

**Key takeaways**

- **Anomalies are rule-based, not learned** — every flag traces to one readable
  threshold you can predict and tune.
- **Three severities** sort the list: `alert` (encrypted) above `warn` (obfuscated)
  above `note` (everything else).
- **It reuses upstream results** — hoppers from topology, duty from timeline,
  verdicts from entropy — rather than recomputing anything.
- **Mercury collects a stack of flags**, which is what an interesting signal looks
  like: several notes plus a warn.

## Cheat sheet

| Kind | Severity | Rule |
|---|---|---|
| `hopper` | note | Emitter `HopSet` > 1 (from topology) |
| `intermittent` | note | Duty < 0.05 **and** ≥ 3 bursts |
| `wide-carrier` | note | Occupied bandwidth > 30 kHz |
| `narrow-carrier` | note | Occupied bandwidth in (0, 5 kHz) |
| `noise-like` | note | Median spectral flatness ≥ 0.80 |
| `obfuscated` | warn | Entropy verdict: xor / scrambler / substitution |
| `encrypted` | alert | Entropy verdict: strong-encrypted |
| `unknown-protocol` | note | Any other entropy finding |

## In this post

- **What RF Expert Info is** and why rules beat magic here.
- **The topology rule** — frequency hoppers.
- **The timeline + segmentation rules** — intermittent, wide, narrow, noise-like.
- **The entropy rules** — encrypted, obfuscated, unknown-protocol.
- **Severity ordering** — how the list is sorted.
- **Mercury's anomaly stack.**

## RF Expert Information

Wireshark's Expert Information collects the notable things it noticed while dissecting
a capture — retransmissions, malformed packets, resets — and grades them by severity so
you triage the important ones first. RF Scope's `expert` analyzer is the same idea for a
band: it distills the whole Scene into a short, severity-sorted list of *"here is what
is unusual, and how worried to be."*

Every rule is deliberately **cheap and legible**. There is no model, no learned
threshold, no black box — each anomaly is one comparison against one named constant,
listed at the top of `expert.go`. That means you can always answer "why did it flag
this?" and you can tune any rule by changing one number. As Reese puts it, *"an alert
you can't explain is an alert you can't trust."*

Because the rules read topology's emitters, timeline's channels, and entropy's
verdicts, the analyzer declares all three as dependencies:

```go
// internal/rfscope/expert.go
func (expertAnalyzer) DependsOn() []string { return []string{"topology", "timeline", "entropy"} }
```

So it always runs last, over a fully-populated Scene.

## The topology rule: hoppers

The first rule is the simplest: any emitter that topology collapsed into a frequency
hopper — a `HopSet` with more than one frequency — gets a `hopper` note:

```go
// internal/rfscope/expert.go
if len(e.HopSet) > 1 {
    out = append(out, Anomaly{
        Severity: "note", Kind: "hopper", EmitterID: e.ID, FreqHz: e.Fingerprint.CenterHz,
        Detail: fmt.Sprintf("emitter hops %d channels", len(e.HopSet)),
    })
}
```

It is a `note` rather than a `warn` because hopping is *interesting*, not inherently
suspicious — plenty of legitimate systems hop. The severity says "worth your
attention," and the detail tells you how many channels.

## The per-channel rules

Three rules run over each channel, reading fields the timeline analyzer and
segmentation already computed:

- **`intermittent`** — `DutyCycle < 0.05` **and** `BurstCount ≥ 3`. This is the "low
  duty, high occupancy" bursty signature from Part 4, made into a flag. The
  burst-count guard matters: a channel with one stray burst is not intermittent, it is
  a fluke. Three or more short bursts on an otherwise-quiet channel is a real pattern —
  telemetry, paging, a beacon.
- **`wide-carrier`** — occupied bandwidth `> 30 kHz`. Wider than the widest common
  narrowband channel, so it stands out on an LMR band: a wideband data link, a spread
  signal, or two overlapping carriers the segmenter merged.
- **`narrow-carrier`** — occupied bandwidth in the open interval `(0, 5 kHz)`. Tighter
  than the tightest standard raster — a CW carrier, a very-low-rate telemetry tone, or a
  measurement artifact.
- **`noise-like`** — median spectral flatness `≥ 0.80`. Recall from Part 2 that flatness
  near 1 means a flat, noise-like spectrum. A carrier that is *on* but spectrally flat is
  either genuine noise mistaken for a carrier, or — more interestingly — a spread-spectrum
  or encrypted-PHY signal whose spectrum has been deliberately whitened. The detail says
  so: *"spread/encrypted-PHY or noise."*

```go
// internal/rfscope/expert.go
switch {
case c.OccupiedBwHz > wideCarrierHz:   // 30000
    // wide-carrier
case c.OccupiedBwHz > 0 && c.OccupiedBwHz < narrowCarrierHz:  // 5000
    // narrow-carrier
}
if c.MedianFlatness >= noiseLikeFlatness {  // 0.80
    // noise-like
}
```

## The entropy rules

The entropy triage from Part 7 gets promoted into anomalies, and this is where the
severities climb:

```go
// internal/rfscope/expert.go
sev, kind := "note", "unknown-protocol"
switch r.Class {
case "strong-encrypted":
    sev, kind = "alert", "encrypted"
case "repeating-xor", "lfsr-or-keyless-scrambler", "periodic-scrambler", "substitution-or-shift":
    sev, kind = "warn", "obfuscated"
}
```

- **`encrypted`** (`alert`) — the entropy verdict was `strong-encrypted`: no exploitable
  structure. This is the only `alert` the analyzer raises, because it is the one finding
  that says "you cannot get further without key material."
- **`obfuscated`** (`warn`) — an XOR, scrambler, substitution, or LFSR verdict: there
  *is* exploitable structure, and the detail carries the recommended cryptolab command.
- **`unknown-protocol`** (`note`) — any other entropy finding: a digital emitter that
  matched no protocol and no strong crypto signature.

Each anomaly's detail folds in the entropy value and the recommended next step, so the
expert list doubles as a to-do list: an `obfuscated` warn tells you both *what* it is
and *which* cryptolab command to run next.

## Severity ordering

Finally the list is sorted so the important things float to the top — severity first
(`alert` > `warn` > `note`), then frequency, then kind for a stable order:

```go
// internal/rfscope/expert.go
sort.SliceStable(out, func(i, j int) bool {
    if sevRank(out[i].Severity) != sevRank(out[j].Severity) {
        return sevRank(out[i].Severity) > sevRank(out[j].Severity)
    }
    ...
})
```

The cockpit (Part 9) renders this same ordering with color — red for `alert`, yellow
for `warn`, plain for `note` — so a glance at the expert panel tells you whether
anything needs urgent attention.

<figure class="lab-figure">
<svg viewBox="0 0 600 200" width="600" height="200" role="img" aria-label="Annotated severity-sorted anomaly list">
  <text x="12" y="22" fill="var(--fg-muted)" font-size="11" font-family="monospace">Expert info (severity-sorted)</text>
  <rect x="8" y="34" width="10" height="18" fill="#d64545"/>
  <text x="28" y="48" fill="currentColor" font-size="12" font-family="monospace">[alert] encrypted        453.550  strong-encrypted (entropy 7.94)</text>
  <rect x="8" y="60" width="10" height="18" fill="#d6a915"/>
  <text x="28" y="74" fill="currentColor" font-size="12" font-family="monospace">[warn]  obfuscated       453.550  lfsr-or-keyless-scrambler …</text>
  <rect x="8" y="86" width="10" height="18" fill="var(--fg-muted)"/>
  <text x="28" y="100" fill="currentColor" font-size="12" font-family="monospace">[note]  hopper           453.550  emitter hops 4 channels</text>
  <rect x="8" y="112" width="10" height="18" fill="var(--fg-muted)"/>
  <text x="28" y="126" fill="currentColor" font-size="12" font-family="monospace">[note]  intermittent     453.550  6 bursts at 2.1% duty</text>
  <rect x="8" y="138" width="10" height="18" fill="var(--fg-muted)"/>
  <text x="28" y="152" fill="currentColor" font-size="12" font-family="monospace">[note]  noise-like       453.550  spectral flatness 0.83 …</text>
</svg>
<figcaption>The expert list sorts alert above warn above note. A single interesting emitter can populate several rows — as Mercury does here.</figcaption>
</figure>

## Tuning the rules for your band

Because every rule is a comparison against one named constant, the anomaly list is
*tunable* by editing thresholds — and knowing the defaults tells you when a flag is
meaningful for your band. The four numeric constants:

| Constant | Default | Raise it to… | Lower it to… |
|---|---|---|---|
| `intermittentDuty` | 0.05 | flag more channels as intermittent | only the sparsest |
| `noiseLikeFlatness` | 0.80 | require a flatter spectrum before flagging | catch more spread signals |
| `wideCarrierHz` | 30000 | tolerate wider carriers on a wideband plan | flag anything above narrowband |
| `narrowCarrierHz` | 5000 | flag more marginal-width carriers | only the very narrowest |

The defaults encode assumptions about a *narrowband LMR band* — a 30 kHz "wide" bar and
a 5 kHz "narrow" bar make sense on a 12.5 kHz raster. On a wideband data band those bars
are wrong: everything would trip `wide-carrier` and the flag would carry no information.
The point of exposing the constants is that "unusual" is relative to the band you are
looking at, and the analyzer makes no attempt to guess your band for you — it applies a
fixed, legible rule and trusts you to interpret it. An anomaly is a *pointer*, not a
conviction; the operator supplies the context.

This is also why the analyzer emits so freely rather than trying to be clever about
suppression. It would be easy to write logic that hides a `wide-carrier` note when a
`hopper` note is already present on the same emitter, on the theory that you only need one
reason to look. RF Scope does the opposite: it surfaces *every* rule that fires, because
the **combination** is the signal. A wide carrier is mildly interesting; a wide carrier
that is also noise-like and obfuscated is a different animal entirely, and you only see
that if all three flags are present to be read together.

## Mercury's anomaly stack

By Part 8, Mercury has quietly accumulated flags from every upstream analyzer, and the
expert list is where they all surface together. From topology it earns a **`hopper`**
note (four channels). From timeline it earns an **`intermittent`** note — its duty cycle
on any channel is a couple of percent, well under 5%, with more than three bursts. Its
whitened spectrum may earn a **`noise-like`** note. And from entropy it earns an
**`obfuscated`** warn (the recommended cryptolab command right there in the detail).

That *stack* is the signature of a signal worth chasing. No single flag is alarming —
plenty of signals hop, plenty are intermittent — but a hopping, intermittent,
spectrally-whitened, structurally-obfuscated emitter that no protocol names is exactly
the profile that says *"this was built to be hard to notice."* The expert panel turns
Ada's vague unease about the 453 MHz burst into a concrete, prioritized list, with the
next command already written.

## Where this goes next

You have now seen every analyzer. [Part
9]({{ '/blog/tutorials/rf-scope-09-scene-cockpit-tui-web/' | relative_url }}) puts them
on screen: the `rfscope cockpit` full-screen TUI and the `rfscope serve` web console,
both of which render the hierarchy, channel sparklines, top talkers, conversations, and
this severity-colored expert panel live, refreshing as new IQ arrives.

## FAQ

**Why are the rules simple thresholds instead of something smarter?**
So every flag is explainable and tunable. You can always trace an anomaly to one named
constant and change it. A learned detector would be harder to trust and harder to
adjust for an unusual band.

**Why is `encrypted` the only `alert`?**
Because it is the only finding that says you cannot proceed without key material or IV
reuse. The `obfuscated` verdicts are `warn` because cryptolab can usually break them;
they are a lead, not a wall.

**Can one emitter produce multiple anomalies?**
Yes — that is normal and informative. A single hopping, intermittent, obfuscated
emitter legitimately produces a hopper note, an intermittent note, and an obfuscated
warn. The stack of flags is itself a signal.

**Does expert recompute duty cycle or entropy?**
No. It reads what timeline and entropy already computed. It only depends on those
analyzers so the fields it reads are populated before it runs.

## Series navigation

**Part 8 of 10** · ←
[Part 7: Entropy & Encryption Triage]({{ '/blog/tutorials/rf-scope-07-entropy-encryption-triage/' | relative_url }})
· Next →
[Part 9: The Scene Cockpit]({{ '/blog/tutorials/rf-scope-09-scene-cockpit-tui-web/' | relative_url }})
