---
title: "RF Scope, Part 6: Topology — Emitters, Frequency Hoppers & Conversations"
description: rfscope's topology analyzer is the RF analog of Wireshark's Conversations and Endpoints — it clusters bursts into emitters by RF fingerprint, collapses a frequency hopper's scattered bursts into one logical source, links emitters into conversations, and defers to hunt's authoritative map when a trunking control channel decodes.
category: tutorials
keywords: rfscope, topology, emitters, frequency hopper, conversations, wireshark conversations, endpoints, rf fingerprint, hop sequence, co-active, request-response, gophertrunk
tags: [rfscope, rf-analysis, advanced, gophertrunk, dsp, sdr]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Scope"
series_part: 6
---

*Part 6 of **RF Scope**, GopherTrunk's protocol-agnostic RF network analyzer. Every
view so far looked down one channel; this one looks across the whole band and asks who
is talking to whom.*

> **TL;DR:** The `topology` analyzer is Wireshark's Conversations/Endpoints for RF. It
> clusters bursts into **emitters** by RF fingerprint — Stage A groups by
> frequency + class family, Stage B conservatively merges several single-channel
> emitters into one **frequency hopper** when they never transmit at the same time and
> share a power band — then links emitters into **conversations**: hop sequences,
> co-active pairs (lag-0 correlation ≥ 0.5), and request-response pairs (off-lag peak
> ≥ 0.4). When a trunking control channel decodes, it defers to `hunt`'s authoritative
> map. **Mercury** surfaces here as a frequency hopper.

**Key takeaways**

- **Emitters are bursts clustered by RF fingerprint** — the protocol-agnostic
  generalization of a radio's identity.
- **A frequency hopper collapses into one emitter** via a deliberately conservative
  two-stage merge.
- **Conversations link correlated emitters:** hop sequence, co-active, or
  request-response.
- **When trunking is present, `hunt` is the authority** — topology folds its map in
  rather than reinventing it.

## Cheat sheet

| Concept / constant | Meaning |
|---|---|
| Stage A cluster | Group by (frequency, class family) |
| Stage B merge | Collapse time-disjoint same-power emitters into a hopper |
| `HopSet` | The >1 frequencies a hopper visits |
| `coactiveThresh` = 0.5 | Lag-0 correlation ⇒ co-active conversation |
| `rrThresh` = 0.4 | Off-lag peak ⇒ request-response conversation |
| `hopperPowerBucketDb` = 6 | Power band width for hopper candidates |

## In this post

- **What RF topology is** and how it maps to Wireshark's Conversations.
- **Stage A** — clustering bursts into per-channel emitters.
- **Stage B** — merging a frequency hopper into one emitter.
- **Conversations** — hop sequences, co-active, and request-response.
- **Deferring to `hunt`** when real trunking decodes.
- **Mercury the hopper.**

## RF topology

Wireshark's Conversations tab turns a flat packet list into a graph of endpoints and
the flows between them. RF Scope's `topology` analyzer does the equivalent for a band:
it turns a flat list of bursts into a graph of **emitters** (the sources) and
**conversations** (the relationships between them). This is the closest RF Scope comes
to `hunt`'s trunking map — but generalized, so it works for an unknown IoT mesh or a
hopping data link that `hunt` would never recognize.

The analyzer has **no dependencies** (it clusters raw bursts), and it does two things
in order: cluster emitters, then detect conversations.

## Stage A: per-channel emitters

The first pass groups bursts by exact frequency and coarse modulation family:

```go
// internal/rfscope/topology.go
type key struct {
    freq   uint32
    family string
}
groups := map[key][]int{}
for i, b := range sc.Bursts {
    k := key{b.FreqHz, classFamily(b.Class)}
    groups[k] = append(groups[k], i)
}
```

`classFamily` folds the fine-grained classes into four coarse families — `am`,
`analog-fm`, `digital`, `unknown` — deliberately, so that a hopper whose hops get
classified slightly inconsistently (C4FM on one visit, FSK on the next) still lands in
one family and can be merged later. Each group becomes an emitter with a **fingerprint**:
center frequency, occupied bandwidth, class family, median power, median burst length,
and median spectral flatness. This is the common case: most emitters live on one
channel, and Stage A alone handles them.

## Stage B: collapsing a frequency hopper

A frequency hopper defeats per-channel clustering — its bursts scatter across many
channels, so Stage A sees them as many separate one-burst emitters. Stage B stitches
them back together, but *carefully*, because the failure mode (merging two distinct
co-channel radios into a phantom hopper) is worse than missing a hopper.

The merge fires only when all of these hold:

1. **Same class family and same power band.** Candidates are bucketed by family and by
   median power in `hopperPowerBucketDb` = 6 dB bins — a single radio's power does not
   swing wildly between hops.
2. **At least two distinct frequencies.** A "hopper" on one frequency is just an
   emitter.
3. **Time-disjoint bursts.** The candidates' bursts must be pairwise non-overlapping in
   time — *a single radio cannot key two channels at once.* This is the decisive test:

```go
// internal/rfscope/topology.go
for i := 1; i < len(ivs); i++ {
    if ivs[i].start < ivs[i-1].end {
        return false // two bursts overlap ⇒ not one radio
    }
}
```

Only when a group of single-channel emitters shares a family and power band, spans
multiple frequencies, and never overlaps in time does RF Scope collapse them into one
emitter with a `HopSet` of the visited frequencies. Everything that fails the test
stays as it was. Reese calls this *"the physics veto"* — the time-disjoint rule is not
a heuristic, it is a hard constraint from how radios work.

## Conversations

With emitters clustered, the analyzer links temporally correlated ones. It builds a
binary activity signal per emitter (which timeline bins it was active in) and looks for
three relationship kinds:

- **Hop sequence** (intra-emitter). Any emitter with a `HopSet` of more than one
  frequency is its own hop-sequence conversation — the ordered visits *are* the
  relationship.
- **Co-active** (inter-emitter). Two emitters whose activity signals correlate at
  **lag 0** with correlation ≥ `coactiveThresh` (0.5) are transmitting *together* — a
  paired uplink/downlink, or two slots of the same call. They start and stop in step.
- **Request-response** (inter-emitter). Two emitters whose correlation peaks at a
  **non-zero lag** ≥ `rrThresh` (0.4), while their lag-0 correlation stays low, are
  *alternating* — one transmits, then the other, like a query and its reply.

```go
// internal/rfscope/topology.go
switch {
case lag0 >= coactiveThresh:
    kind, score = "co-active", lag0
case bestVal >= rrThresh && bestLag != 0 && lag0 < 0.3:
    kind, score = "request-response", bestVal
}
```

Pairwise comparison is capped at the top `maxConvEmitters` (24) talkers so a busy band
does not become an O(n²) blowup. Each conversation records the emitter IDs, the kind,
the correlation score, and an exchange count.

<figure class="lab-figure">
<svg viewBox="0 0 600 230" width="600" height="230" role="img" aria-label="Emitter and conversation graph">
  <circle cx="90" cy="70" r="30" fill="none" stroke="currentColor"/>
  <text x="90" y="66" text-anchor="middle" fill="currentColor" font-size="11">#1</text>
  <text x="90" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="9">453.325</text>
  <circle cx="90" cy="180" r="30" fill="none" stroke="currentColor"/>
  <text x="90" y="176" text-anchor="middle" fill="currentColor" font-size="11">#2</text>
  <text x="90" y="190" text-anchor="middle" fill="var(--fg-muted)" font-size="9">453.350</text>
  <line x1="90" y1="100" x2="90" y2="150" stroke="var(--accent)"/>
  <text x="150" y="128" fill="var(--accent)" font-size="10">co-active (0.71)</text>
  <circle cx="330" cy="70" r="30" fill="none" stroke="currentColor"/>
  <text x="330" y="66" text-anchor="middle" fill="currentColor" font-size="11">#3</text>
  <text x="330" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="9">ctl</text>
  <circle cx="330" cy="180" r="30" fill="none" stroke="currentColor"/>
  <text x="330" y="176" text-anchor="middle" fill="currentColor" font-size="11">#4</text>
  <text x="330" y="190" text-anchor="middle" fill="var(--fg-muted)" font-size="9">voice</text>
  <line x1="330" y1="100" x2="330" y2="150" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="360" y="128" fill="var(--fg-muted)" font-size="10">req-response (0.48)</text>
  <circle cx="520" cy="125" r="34" fill="none" stroke="var(--accent)"/>
  <text x="520" y="120" text-anchor="middle" fill="var(--accent)" font-size="11">#5</text>
  <text x="520" y="135" text-anchor="middle" fill="var(--fg-muted)" font-size="9">hopper ×4</text>
  <path d="M 520 91 A 30 30 0 1 1 514 91" fill="none" stroke="var(--accent)"/>
  <text x="520" y="185" text-anchor="middle" fill="var(--accent)" font-size="10">hop-sequence</text>
</svg>
<figcaption>An emitter/conversation graph: a co-active pair, an alternating request-response pair, and a self-looping frequency hopper — Mercury.</figcaption>
</figure>

## The fingerprint, and why clustering is conservative

Everything topology does hinges on the **fingerprint** — the feature vector each emitter
is summarized by: center frequency, occupied bandwidth, class family, median power,
median burst length, and median spectral flatness. Notice what is *not* in it. There is
no attempt to fingerprint the transmitter's hardware — no LO-drift signature, no
transient-onset shape, none of the RF-fingerprinting that a lab with a controlled
reference might attempt. Those techniques are fragile: they depend on capture SNR,
receiver calibration, and a training set, and they fail quietly. RF Scope's fingerprint
is built from features it can measure robustly on a single blind capture, which is why it
uses medians throughout — one bad burst cannot move the emitter's identity.

That conservatism is a theme. Stage A clusters on *exact* frequency and coarse family,
which never merges two things that should be separate; its only failure mode is
*over*-splitting a hopper, which Stage B then repairs under strict conditions. The whole
design leans toward "leave it separate unless the physics forces a merge," because in
reconnaissance a false merge is a lie — it invents a device that does not exist — while a
false split is merely a missed simplification you can spot by eye. When the analyzer does
collapse four channels into one hopper, you can trust that the time-disjoint, same-power,
same-family evidence was strong enough to rule out coincidence.

## Deferring to hunt

There is one case where RF Scope should *not* trust its own generic clustering: a real
trunking system, where GopherTrunk already has a purpose-built, authoritative map. When
an emitter's representative burst is long enough to decode and identifies as a trunking
protocol, topology decodes it with `siglab`, folds the result into a
`hunt.DiscoveredSystem`, and attaches that under `AnalyzerOutputs["topology"]`:

```go
// internal/rfscope/topology.go
hunt.Accumulate(sys, hunt.Observation{
    Protocol:       ident.Winner,
    Confidence:     ident.Confidence,
    Result:         res,
    FallbackFreqHz: e.Fingerprint.CenterHz,
})
```

So the WACN, SysID, RFSS, site, neighbors, and band plan of a trunked system come from
`hunt` — the code that specializes in exactly that — and hunt's map becomes one
*specialization* of the generic emitter graph. RF Scope caps this at
`maxTrunkDecode` = 8 systems and only attempts it for digital emitters whose median
burst is at least `minTrunkDecodeSec` = 0.5 s, so the expensive full decode runs only
where it will pay off.

## Mercury the hopper

This is Mercury's defining moment in RF Scope. In the timing view (Part 5) its bursts
looked aperiodic on every single channel — no clean frame anywhere. Topology explains
why: those bursts, scattered across four nearby 12.5 kHz channels around 453 MHz, all
share the same digital class family, sit in the same 6 dB power band, and — critically
— **never overlap in time**. Stage B collapses them into one emitter with a four-entry
`HopSet`, labeled something like `digital hopper (4 ch)`, and detectConversations emits
a **hop-sequence** conversation for it.

That reframes everything. Mercury is not four weak mystery signals; it is *one*
intermittent, frequency-hopping emitter that has been deliberately spreading itself
across the band. That is the kind of behavior that motivates a closer look at its
payload — which is exactly where Part 7's entropy analyzer comes in.

## Where this goes next

Topology tells you Mercury is one hopping emitter that no protocol names. The obvious
next question is: *what is in its bursts, and is it encrypted?* [Part
7]({{ '/blog/tutorials/rf-scope-07-entropy-encryption-triage/' | relative_url }})
introduces the `entropy` analyzer — which depends on topology — and the **Crypto Lab
bridge**: it blind-demodulates an unknown emitter's payload, runs the cryptolab
randomness battery over it, and can emit the bytes as a frames file for the
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}) toolkit.

## FAQ

**How can RF Scope tell a hopper from two separate radios?**
The time-disjoint test: a single radio physically cannot transmit on two channels at
once, so if any two candidate bursts overlap in time, the merge is rejected. Combined
with matching class family and power band, that keeps distinct co-channel radios from
being fused into a phantom hopper.

**What's the difference between co-active and request-response?**
Co-active emitters start and stop *together* — their occupancy correlates at lag 0
(≥ 0.5). Request-response emitters *alternate* — the correlation peaks at a non-zero
lag (≥ 0.4) while lag 0 stays low. One is simultaneous, the other is turn-taking.

**Why defer to `hunt` for trunking?**
`hunt` is GopherTrunk's specialized trunking-map accumulator; it knows how to assemble
WACN/site/neighbor data authoritatively. Topology folds hunt's map in as a
specialization rather than approximating it with generic clustering.

**Does topology need timeline or timing first?**
No — topology declares no dependencies and clusters raw bursts. It does use the same
100-bin activity model for conversation correlation, but it builds that itself from the
bursts.

## Series navigation

**Part 6 of 10** · ←
[Part 5: Timing & Periodicity]({{ '/blog/tutorials/rf-scope-05-timing-and-tdma-period/' | relative_url }})
· Next →
[Part 7: Entropy & Encryption Triage]({{ '/blog/tutorials/rf-scope-07-entropy-encryption-triage/' | relative_url }})
