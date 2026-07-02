---
title: "RF Scope, Part 1: Wireshark for the RF Physical Layer"
description: rfscope is GopherTrunk's protocol-agnostic RF network analyzer — point it at any band, recorded IQ or a live SDR, with no prior knowledge of the technology, modulation, framing, or encryption, and get a structured Scene of what is on the air and how it behaves.
category: tutorials
keywords: rfscope, rf network analysis, protocol-agnostic, wireshark for rf, sdr, iq capture, signal survey, rf scene, gophertrunk, spectrum analysis, unknown signal, rf physical layer
tags: [rfscope, sdr, rf-analysis, getting-started, gophertrunk, dsp]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Scope"
series_part: 1
---

*Part 1 of **RF Scope**, a 10-part series on GopherTrunk's protocol-agnostic RF
network analyzer. RF Scope is the middle series of the **Lab Bench trilogy** —
[Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }}) names and measures a
single signal, RF Scope maps a whole band, and
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}) breaks what comes out
the other end. A mystery signal named **Mercury** runs through all three.*

> **TL;DR:** `rfscope` is Wireshark for the RF physical layer. Point it at a band —
> a recorded IQ file or a live SDR — with **no idea** what technology is there, and
> it segments the span into bursts and builds a structured **Scene**: a protocol
> hierarchy, per-channel activity, burst timing, an emitter/conversation graph, an
> entropy/encryption triage, and an expert-info anomaly list. It adds no DSP of its
> own — it orchestrates primitives GopherTrunk already ships — and it is not a
> waterfall UI; the outputs are trees, tables, graphs, and JSON.

**Key takeaways**

- **RF Scope answers "what is on this band and how does it behave?"** without you
  naming a protocol, modulation, or framing first.
- **It is not a spectrum scope.** For live waterfalls and constellations you want
  [Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }}); RF Scope emits
  structured data, not a scrolling display.
- **It orchestrates, it does not invent DSP.** The survey classifier, carrier peak
  detection, spectrum occupancy, siglab identify, and the cryptolab randomness
  battery all already existed; RF Scope wires them into one **Scene**.
- **It sits between `hunt` and `siglab`.** `hunt` maps *known* trunked systems,
  `siglab` decodes *named* protocols, and RF Scope says something structural about a
  signal even when it is *neither*.

## Cheat sheet

| Command / flag | What it does |
|---|---|
| `rfscope analyze -in <cap>` | Segment + analyze a recorded IQ capture |
| `rfscope live -serial <sdr> -freq Hz` | Analyze a live SDR span |
| `rfscope cockpit` | Full-screen live Scene TUI |
| `rfscope serve` | Browser console at `127.0.0.1:8098` |
| `rfscope list` | Print the registered analyzers |
| `-analyzers hierarchy,timing` | Run a subset (dependencies auto-pulled) |
| `-out-format summary\|json\|jsonl\|yaml\|csv` | Choose the report shape |
| `-frames-out frames.jsonl` | Emit a Crypto Lab `ks` frames file |

## In this post

- **What RF Scope actually is** — and the expectation it is *not* a UI.
- **Where it sits** versus `hunt` and `siglab`.
- **The Scene data model** — Burst, Channel, Emitter, Conversation, Hierarchy,
  Anomaly.
- **The five commands** you will use across the series.
- **The cast, the trilogy, and Mercury** — the thread that ties all thirty posts.

<aside class="lab-cast">
  <span class="lab-cast__badge">A</span>
  <div class="lab-cast__body">
    <p class="lab-cast__who">Meet the cast.</p>
    <p><strong>Ada</strong> just unboxed her first SDR and is learning to trust her
    tools before she trusts her ears. <strong>Reese</strong> has been chasing
    signals for twenty years and supplies the "why it works that way." They turn up
    a paragraph at a time — narrative framing for the technical work, never a
    substitute for it.</p>
  </div>
</aside>

## What RF Scope is

**RF Scope is protocol-agnostic RF network analysis.** You point it at a slice of
spectrum — a wideband `.cfile` on disk, or a live receiver — and it tells you what
is transmitting, on which channels, how often, in what shape, and whether any of it
looks encrypted. Crucially, you supply **no prior knowledge**. You do not tell it
"this is P25" or "expect 4-level FSK at 4800 baud." It discovers the carriers,
slices them into bursts, classifies each burst's modulation blindly, and assembles
the results into a single structured **Scene**.

That is the same move Wireshark makes one layer up. Wireshark takes a pcap it knows
nothing about and produces a Protocol Hierarchy, I/O Graphs, a Conversations table,
and Expert Information. RF Scope produces the RF-physical-layer analogs of exactly
those views. If you have ever opened a strange capture in Wireshark just to see
*what is in here*, you already understand the workflow.

**It is decidedly not a spectrum scope.** There is no scrolling waterfall, no live
constellation, no eye diagram. Those belong to
[Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }}), the sibling series
built for staring at one signal in detail. RF Scope's outputs are *structured*:
indented hierarchy trees, channel tables, occupancy timelines, an emitter graph,
and a JSON Scene you can pipe into other tools. When you want to *see* a signal, use
Signal Lab. When you want to *inventory a band*, use RF Scope.

**And it adds no DSP of its own.** This is a design principle, not an accident. The
package docstring is blunt about it: RF Scope "imports only downward" and
"orchestrates the existing primitives and accumulates their output into one shared,
JSON-exportable Scene." The pieces it wires together already ship in GopherTrunk:

- the `survey` blind modulation classifier,
- `carriers` wideband peak detection,
- `spectrum` occupancy metrics (occupied bandwidth, channel power, ACPR, spectral
  flatness),
- `siglab` protocol identification, and
- the `cryptolab` randomness / keystream engines.

Because it reuses proven primitives, a bug in RF Scope is almost always a wiring
bug, not a new DSP bug — and every measurement it reports is one you could compute
by hand with the underlying tool.

## Where it sits: hunt vs siglab vs rfscope

Three GopherTrunk tools look at RF, and it helps to keep their jobs distinct:

| Tool | Question it answers | Needs to know the protocol? |
|---|---|---|
| `hunt` | "Which *known trunked systems* are here, and what are their WACN / site / neighbors?" | Yes — it maps named trunking systems |
| `siglab` | "What *named protocol* is this one signal, and can I decode its PDUs?" | It identifies among 13 named protocols |
| `rfscope` | "What is *structurally* on this band, even if it is nothing I recognize?" | **No** |

RF Scope is the layer below the other two. An unknown IoT mesh, a telemetry link, a
paging variant, an encrypted data burst, a frequency hopper — none of those decode
as a named protocol, and `hunt` will never map them, but RF Scope will still tell
you they exist, how wide they are, how bursty they are, and whether they look
random. And when a real trunking control channel *does* turn up in the band, RF
Scope does not reinvent the map: its topology analyzer **defers to `hunt`'s
authoritative map** for that system (Part 6).

Reese's rule of thumb: *"`siglab` is a microscope, `hunt` is an atlas, and RF Scope
is the reconnaissance photo you take before you know which one you need."*

## The Scene data model

Everything RF Scope produces lives on one struct, `Scene`. Six views matter:

- **Burst** — the atom. One onset→offset activity segment on one frequency, with
  its spectral metrics (occupied bandwidth, channel power in dBFS, ACPR, spectral
  flatness) and a blindly-classified modulation class. Every other statistic is
  built from bursts.
- **Channel** — the per-frequency, time-aggregated row: duty cycle, occupancy
  percentage, burst rate, dominant class, and a 100-bin activity timeline.
- **Emitter** — a cluster of bursts sharing an RF fingerprint. A frequency hopper's
  scattered bursts collapse into *one* emitter.
- **Conversation** — temporally correlated emitters linked together: a hop sequence,
  a co-active pair, or a request/response exchange.
- **Hierarchy** — the class → bandwidth → protocol tree, the RF Protocol Hierarchy.
- **Anomaly** — one expert-info finding, with a severity of `note`, `warn`, or
  `alert`.

<figure class="lab-figure">
<svg viewBox="0 0 660 150" width="660" height="150" role="img" aria-label="RF Scope pipeline from source to Scene to export">
  <rect x="6" y="54" width="96" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="54" y="72" text-anchor="middle" fill="currentColor" font-size="12">Source</text>
  <text x="54" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="10">IQ file / SDR</text>
  <line x1="102" y1="75" x2="132" y2="75" stroke="currentColor"/>
  <polygon points="132,71 140,75 132,79" fill="currentColor"/>
  <rect x="140" y="54" width="104" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="192" y="72" text-anchor="middle" fill="currentColor" font-size="12">Segment</text>
  <text x="192" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="10">carriers→bursts</text>
  <line x1="244" y1="75" x2="274" y2="75" stroke="currentColor"/>
  <polygon points="274,71 282,75 274,79" fill="currentColor"/>
  <rect x="282" y="42" width="130" height="66" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="347" y="66" text-anchor="middle" fill="var(--accent)" font-size="12">Analyzer</text>
  <text x="347" y="82" text-anchor="middle" fill="var(--accent)" font-size="12">registry</text>
  <text x="347" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="9">hierarchy · timeline …</text>
  <line x1="412" y1="75" x2="442" y2="75" stroke="currentColor"/>
  <polygon points="442,71 450,75 442,79" fill="currentColor"/>
  <rect x="450" y="54" width="96" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="498" y="78" text-anchor="middle" fill="var(--accent)" font-size="13">Scene</text>
  <line x1="546" y1="75" x2="576" y2="75" stroke="currentColor"/>
  <polygon points="576,71 584,75 576,79" fill="currentColor"/>
  <rect x="584" y="30" width="72" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="620" y="49" text-anchor="middle" fill="currentColor" font-size="11">export</text>
  <rect x="584" y="90" width="72" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="620" y="109" text-anchor="middle" fill="currentColor" font-size="11">bridge</text>
</svg>
<figcaption>A sample flows Source → Segment → the analyzer registry → the Scene, then out to an export format or the Crypto Lab bridge.</figcaption>
</figure>

That pipeline is the shape of the whole series. Part 2 is the *Segment* box; Parts
3–8 are the analyzers in the registry; Part 9 is how you watch the Scene live; Part
10 is tuning and the export/bridge outputs.

## The five commands

```text
gophertrunk rfscope analyze -in <capture> [flags]          analyze a recorded IQ capture
gophertrunk rfscope live -serial <sdr> -freq Hz [flags]    analyze a live SDR span
gophertrunk rfscope cockpit [-in <cap> | -serial <sdr> -freq Hz]   live scene TUI
gophertrunk rfscope serve [-addr host:port] [-open]        web console (browser)
gophertrunk rfscope list                                   list registered analyzers
```

The simplest useful invocation summarizes a wideband capture:

```bash
gophertrunk rfscope analyze -in wide.cfile -format f32 \
    -sample-rate 2400000 -freq 451000000
```

That prints a human-readable Scene: the hierarchy tree, a channel table, top
talkers, conversations, and expert info. Swap `-out-format json -out scene.json` and
you get the whole Scene as machine-readable data instead. Point `cockpit` at the
same file and you get a live, refreshing terminal dashboard of it. Everything in
this series is a variation on those.

## Mercury enters

While testing her new receiver, Ada parks it on **453 MHz** in the UHF business
band and catches something odd: a short, intermittent burst, roughly a 12.5 kHz
channel, that comes and goes with no obvious pattern. Signal Lab could not name it —
its best blind guess was around 4800 sym/s and 4-level-FSK-like, but no protocol
locked. So she hands the wideband capture to RF Scope. Over Parts 3, 6, and 7 we
will watch Mercury show up as an **unknown-protocol**, **frequency-hopper**-flagged
emitter, get triaged as *not-obviously-strong* encryption, and then get handed off
to Crypto Lab as a frames file. Keep an eye on it.

## Series map

| Part | Topic | What you'll do |
|---|---|---|
| 1 | Wireshark for the RF physical layer (this post) | Understand the Scene model |
| [2]({{ '/blog/tutorials/rf-scope-02-segmentation-iq-to-bursts/' | relative_url }}) | Segmentation: IQ to bursts | Discover carriers, slice bursts |
| [3]({{ '/blog/tutorials/rf-scope-03-protocol-hierarchy/' | relative_url }}) | Protocol hierarchy | Read the class→bw→protocol tree |
| [4]({{ '/blog/tutorials/rf-scope-04-io-graph-channel-activity/' | relative_url }}) | The I/O graph | Per-channel activity over time |
| [5]({{ '/blog/tutorials/rf-scope-05-timing-and-tdma-period/' | relative_url }}) | Timing & periodicity | Recover a TDMA frame period |
| [6]({{ '/blog/tutorials/rf-scope-06-topology-emitters-conversations/' | relative_url }}) | Topology | Cluster emitters, find conversations |
| [7]({{ '/blog/tutorials/rf-scope-07-entropy-encryption-triage/' | relative_url }}) | Entropy triage | Bridge into Crypto Lab |
| [8]({{ '/blog/tutorials/rf-scope-08-expert-info-anomalies/' | relative_url }}) | Expert info | Read the anomaly flags |
| [9]({{ '/blog/tutorials/rf-scope-09-scene-cockpit-tui-web/' | relative_url }}) | The Scene cockpit | Live TUI and web console |
| [10]({{ '/blog/tutorials/rf-scope-10-advanced-tuning-pipelines/' | relative_url }}) | Advanced tuning | Selective analyzers & pipelines |

The full [rfscope docs]({{ '/rfscope.html' | relative_url }}) are the reference this
series expands on; the [Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }})
and [Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}) hubs are the other
two-thirds of the trilogy.

## Where this goes next

[Part 2]({{ '/blog/tutorials/rf-scope-02-segmentation-iq-to-bursts/' | relative_url }})
opens the *Segment* box: how a wideband FFT finds carriers, how a robust noise floor
is estimated, how each carrier is downconverted to a narrow baseband, and how a
hysteresis state machine slices that baseband into the onset→offset **bursts** every
analyzer downstream consumes. Nothing else in the Scene exists until segmentation
runs, so it is the right place to start.

## FAQ

**Is RF Scope a replacement for a spectrum analyzer or waterfall?**
No. It produces structured data — trees, tables, a JSON Scene — not a live visual
display. For waterfalls, constellations, and eye diagrams, use
[Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }}). The two are
complementary: reconnaissance versus close inspection.

**Do I need to know the protocol before I run it?**
No — that is the entire point. RF Scope discovers carriers, classifies modulation
blindly, and builds the Scene with zero prior knowledge of technology, framing, or
encryption. Naming a protocol is `siglab`'s job, and RF Scope will call `siglab` for
you where it helps.

**What hardware do I need?**
For `analyze` and offline `cockpit`, none — just a recorded IQ capture. For `live`
and live `cockpit` you need any SDR GopherTrunk supports. RF Scope is
rate-invariant: it normalizes each carrier to a narrow per-channel baseband, so the
capture rate is yours to choose.

**How does it relate to Crypto Lab?**
The entropy analyzer (Part 7) triages unknown digital payloads and can emit them as
a `ks` frames file with `-frames-out`, which feeds `cryptolab` directly. That is the
hand-off that eventually cracks Mercury.

## Series navigation

**Part 1 of 10** · Next →
[Part 2: From IQ to Bursts]({{ '/blog/tutorials/rf-scope-02-segmentation-iq-to-bursts/' | relative_url }})
