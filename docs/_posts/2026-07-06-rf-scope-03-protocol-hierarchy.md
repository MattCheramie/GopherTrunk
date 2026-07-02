---
title: "RF Scope, Part 3: Protocol Hierarchy — What's on the Air"
description: rfscope's hierarchy analyzer is the RF analog of Wireshark's Protocol Hierarchy — it groups bursts by modulation class, then occupied-bandwidth bucket, then names the protocol of the longest digital burst with siglab, attaching burst counts, airtime, spectrum share, and symbol volume to every node.
category: tutorials
keywords: rfscope, protocol hierarchy, wireshark protocol hierarchy, modulation class, occupied bandwidth, siglab identify, rf analysis, unknown protocol, airtime, spectrum share, gophertrunk
tags: [rfscope, rf-analysis, siglab, getting-started, gophertrunk, dsp]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Scope"
series_part: 3
---

*Part 3 of **RF Scope**, GopherTrunk's protocol-agnostic RF network analyzer. With
bursts in hand from segmentation, the first structured view is the one that answers
"what kinds of signal are here?"*

> **TL;DR:** The `hierarchy` analyzer is Wireshark's Protocol Hierarchy, one layer
> down. It groups bursts into a three-level tree — **modulation class → occupied-bandwidth
> bucket → identified protocol** — and attaches burst count, airtime, spectrum share,
> time share, and estimated symbol volume to each node. It names a protocol by running
> [`siglab.IdentifyIQ`]({{ '/blog/tutorials/signal-lab-08-naming-the-unknown/' | relative_url }})
> on the longest digital burst in a bucket. Signals it cannot name — like **Mercury** —
> appear as an unknown-protocol node.

**Key takeaways**

- **The hierarchy is a three-level tree:** modulation class, then bandwidth bucket,
  then named protocol.
- **Protocol naming is delegated to Signal Lab.** RF Scope runs `siglab.IdentifyIQ`
  on the longest digital burst per bucket — it does not re-implement identification.
- **Every node carries stats:** bursts, airtime, spectrum %, time %, and an estimated
  symbol volume (Σ duration × baud).
- **Unknowns are first-class.** A digital carrier that identifies as nothing simply
  has no protocol child — which is exactly how Mercury shows up.

## Cheat sheet

| Command / concept | What it does |
|---|---|
| `rfscope analyze -in cap -analyzers hierarchy` | Run just the hierarchy view |
| Depth 0 node | A modulation class (C4FM, NBFM, FSK…) |
| Depth 1 node | An occupied-bandwidth bucket (e.g. 12.5 kHz) |
| Depth 2 node | A named protocol from `siglab.IdentifyIQ` |
| `SymbolVolume` | Estimated symbols on air: Σ duration × baud |

## In this post

- **What the RF protocol hierarchy is** and how it maps to Wireshark's.
- **The three levels** — class, bandwidth bucket, protocol.
- **How protocols get named** — the Signal Lab delegation.
- **The stats on each node** — and how to read spectrum vs time share.
- **Mercury** — reading an unknown-protocol node.

## The RF Protocol Hierarchy

Open any pcap in Wireshark and the Protocol Hierarchy Statistics window tells you, at
a glance, the *make-up* of the capture: what fraction is TCP, how much of that is
TLS, how much HTTP underneath. It is the first thing many analysts look at because it
turns thousands of packets into a handful of rows you can reason about.

RF Scope's `hierarchy` analyzer does the same for a band. Instead of protocol layers
it uses the only structure available without decoding — **modulation class**, then
**how wide the signal is**, then, where it can, **the actual protocol**. The result
is a compact tree that says "this band is mostly narrowband C4FM at 12.5 kHz, which
identifies as P25; plus some FSK paging; plus one wide unknown thing." That is the
reconnaissance summary Ada wants before she decides what to chase.

The analyzer has **no dependencies** — it reads bursts directly — so it runs whether
or not you asked for timeline, topology, or the rest.

## Three levels

The tree is built in `internal/rfscope/hierarchy.go` by bucketing bursts twice:

```go
// internal/rfscope/hierarchy.go
for i, b := range sc.Bursts {
    bucket := survey.SnapChannelBandwidth(b.OccupiedBwHz)
    byClass[b.Class][bucket] = append(byClass[b.Class][bucket], i)
}
```

- **Depth 0 — modulation class.** Every burst already carries a blind class from
  `survey.Classify` in segmentation: NBFM, wide FM, AM, C4FM, FSK, PSK, continuous
  data, paging, trunk control, trunk voice. Bursts are grouped by class first.
- **Depth 1 — occupied-bandwidth bucket.** Within a class, bursts are snapped to a
  standard channel bandwidth (`survey.SnapChannelBandwidth`) and grouped — so all the
  12.5 kHz C4FM bursts land in one node, the 25 kHz ones in another. Bandwidth is the
  cheapest discriminator between two protocols that share a modulation family.
- **Depth 2 — named protocol.** For **digital** classes only, RF Scope tries to name
  the protocol of the bucket (below). Analog classes stop at depth 1 — there is no
  "protocol" to name for an FM voice channel.

The tree is flattened depth-first with a `Depth` field on each node, so rendering it
is a matter of indenting by depth.

<figure class="lab-figure">
<svg viewBox="0 0 560 210" width="560" height="210" role="img" aria-label="Indented RF protocol hierarchy tree">
  <text x="12" y="24" fill="var(--fg-muted)" font-size="11" font-family="monospace">Protocol hierarchy (bursts / time% / spectrum%)</text>
  <text x="12" y="52" fill="currentColor" font-size="13" font-family="monospace">c4fm</text>
  <text x="360" y="52" fill="var(--fg-muted)" font-size="12" font-family="monospace">128   41.0%   0.5%</text>
  <text x="40" y="76" fill="currentColor" font-size="13" font-family="monospace">12.5 kHz</text>
  <text x="360" y="76" fill="var(--fg-muted)" font-size="12" font-family="monospace">128   41.0%   0.5%</text>
  <text x="70" y="100" fill="var(--accent)" font-size="13" font-family="monospace">p25p1</text>
  <text x="360" y="100" fill="var(--fg-muted)" font-size="12" font-family="monospace">128   41.0%   0.5%</text>
  <text x="12" y="128" fill="currentColor" font-size="13" font-family="monospace">fsk</text>
  <text x="360" y="128" fill="var(--fg-muted)" font-size="12" font-family="monospace"> 34   6.2%    0.3%</text>
  <text x="40" y="152" fill="currentColor" font-size="13" font-family="monospace">12.5 kHz</text>
  <text x="360" y="152" fill="var(--fg-muted)" font-size="12" font-family="monospace"> 22   4.1%    0.2%</text>
  <text x="70" y="176" fill="var(--fg-muted)" font-size="13" font-family="monospace">(unnamed — Mercury)</text>
  <text x="360" y="176" fill="var(--fg-muted)" font-size="12" font-family="monospace"> 22   4.1%    0.2%</text>
</svg>
<figcaption>A flattened hierarchy: class → bandwidth bucket → protocol. The FSK 12.5 kHz bucket names no protocol — that is Mercury.</figcaption>
</figure>

## How protocols get named

RF Scope does **not** re-implement protocol identification. It calls Signal Lab:

```go
// internal/rfscope/hierarchy.go
res, err := siglab.IdentifyIQ(iq, "rfscope-burst", siglab.IdentifyConfig{
    SampleRateHz: rate,
    Format:       siglab.FormatF32,
    Log:          in.Log,
})
if err != nil || res == nil || res.Inconclusive || res.Winner == "" {
    return ""
}
return res.Winner
```

Two design choices are worth calling out. First, it identifies the **longest digital
burst** in the bucket, not a random one. A longer burst gives the identifier more
symbols to work with, so identification is more reliable — and one representative is
enough, because a bucket is already a group of same-class, same-bandwidth bursts.

Second, it uses the *decimated baseband IQ* that segmentation stored for that burst,
served lazily through `in.BurstIQ`, so RF Scope never holds the whole capture in
memory just to identify one signal. If identification is inconclusive or errors, the
function returns an empty string and the bucket simply has **no protocol child**.

This is the same `siglab.IdentifyIQ` entry point that
[Signal Lab Part 8]({{ '/blog/tutorials/signal-lab-08-naming-the-unknown/' | relative_url }})
covers in full — the "name the unknown" workflow. RF Scope is one of its callers.
Reese's framing: *"identification is a hard, specialized job; RF Scope's job is to
know when to ask for it and what to do with a 'no.'"*

## The stats on each node

Every node — total, class, bucket, protocol — carries a `HierStats`:

| Stat | Meaning |
|---|---|
| `BurstCount` | How many bursts fall under this node |
| `AirtimeSec` | Total on-air seconds of those bursts |
| `SpectrumPct` | Share of the span's bandwidth this node's channels occupy |
| `TimePct` | Airtime as a fraction of the observation window |
| `SymbolVolume` | Estimated symbols on air: Σ (duration × baud) |
| `EmitterCount` | Distinct emitters, once topology has clustered them |

The two percentages answer different questions and often disagree in a revealing way.
**Spectrum %** is "how much of the band does this occupy?" — a wide carrier scores
high even if it is rarely on. **Time %** is "how much of the window is this on air?" —
a busy narrow channel scores high even though it barely dents the spectrum. A control
channel is the classic case: tiny spectrum share, large time share, because it is
almost always transmitting on one narrow channel.

`SymbolVolume` is a rough "how much did this thing actually *say*?" — duration times
the classifier's estimated baud, summed. It is an estimate, not a decode, but it lets
you rank nodes by information carried rather than just airtime.

## Why bandwidth is the middle level

It is worth asking why the tree groups by *occupied bandwidth* between class and
protocol, rather than by frequency or power. The answer is that bandwidth is the
cheapest, most protocol-revealing discriminator available before you decode anything. Two
signals in the same modulation family are often distinguished purely by how wide they
are: a 12.5 kHz C4FM channel and a 25 kHz C4FM channel are almost certainly different
systems, and the split falls out of the occupied-bandwidth measurement segmentation
already computed. Frequency would fragment the tree into one node per channel — too fine
to summarize a band — and power says more about your antenna and the emitter's distance
than about the protocol.

The bandwidths are also **snapped** to standard channel widths via
`survey.SnapChannelBandwidth` before bucketing, so a burst measured at 12.3 kHz and one
at 12.7 kHz land in the same 12.5 kHz node instead of splitting into two near-duplicate
buckets. That snapping is what keeps the tree readable: it collapses measurement jitter
into the small set of channel rasters real systems actually use, and it is the same
raster logic the `-min-spacing` flag exposes in segmentation. The result is a tree whose
middle level reads like a channel plan — "12.5 kHz," "25 kHz," "6.25 kHz" — rather than a
scatter of noisy Hz values.

## Reading an unknown node: Mercury

This is the part that matters for the trilogy. Ada's 453 MHz capture segments into a
handful of short FSK-family bursts around a 12.5 kHz channel. The hierarchy analyzer
buckets them: class `fsk`, bandwidth `12.5 kHz`. Because FSK is a digital class, it
tries to name the protocol — it runs `siglab.IdentifyIQ` on the longest of those
bursts.

Signal Lab comes back **inconclusive**. Its best blind guess had been around 4800
sym/s and 4-level-FSK-like, but nothing locked as a named protocol. So
`identifyRepresentative` returns `""`, and the bucket gets **no depth-2 child**. In
the tree, Mercury is a bandwidth bucket with real bursts, real airtime, and *no
protocol name under it*:

```text
fsk                     22   4.1%   0.2%
  12.5 kHz              22   4.1%   0.2%
    (no protocol identified)
```

An unnamed digital node is not a failure — it is a **finding**. It says "there is real
digital traffic here that is none of the 13 protocols GopherTrunk decodes." That is
precisely the class of signal RF Scope exists to surface, and it is the reason Ada
keeps digging. In Part 6 the topology analyzer will notice Mercury hops; in Part 7 the
entropy analyzer will triage its payload and hand it to Crypto Lab.

## Where this goes next

The hierarchy tells you *what kinds* of signal share the band. It says nothing about
*when* each channel is active. [Part
4]({{ '/blog/tutorials/rf-scope-04-io-graph-channel-activity/' | relative_url }})
builds the **I/O graph** — the `timeline` analyzer — which gives every channel a
time-bucketed occupancy series, a duty cycle, and a burst rate, so you can see the
band breathe over the capture. If you want the full identification story behind the
protocol names, read
[Signal Lab Part 8]({{ '/blog/tutorials/signal-lab-08-naming-the-unknown/' | relative_url }})
and the [Signal Lab hub]({{ '/blog/series/signal-lab/' | relative_url }}).

## FAQ

**Why does RF Scope only name protocols for digital classes?**
Analog FM/AM voice channels have no protocol to identify — the "protocol" *is* the
modulation. Naming only applies to digital classes, where framing and symbol rate can
match a known protocol via `siglab.IdentifyIQ`.

**Why the longest burst per bucket?**
More symbols means more reliable identification. Since a bucket already groups
same-class, same-bandwidth bursts, one strong representative is enough to name the
whole bucket.

**What does an unnamed digital node mean?**
Real digital traffic that matches none of the named protocols GopherTrunk decodes —
an unknown. That is a first-class result, not an error, and it is where the topology
and entropy analyzers take over.

**Does the hierarchy analyzer need any other analyzer to run first?**
No. It depends on nothing and reads bursts directly, so you can run it alone with
`-analyzers hierarchy`. `EmitterCount` on its nodes only fills in once topology has
also run.

## Series navigation

**Part 3 of 10** · ←
[Part 2: From IQ to Bursts]({{ '/blog/tutorials/rf-scope-02-segmentation-iq-to-bursts/' | relative_url }})
· Next →
[Part 4: The I/O Graph]({{ '/blog/tutorials/rf-scope-04-io-graph-channel-activity/' | relative_url }})
