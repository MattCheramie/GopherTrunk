---
title: "Protocol Decoders, Part 9: Conventional, Wideband & the Symbol Scope"
description: How GopherTrunk decodes what isn't a trunked control channel — fixed-frequency conventional scanning with tone squelch, wideband multi-carrier Phase 2, DMR LCN band-plan learning, and the symbol-scope diagnostic.
category: deep-dives
keywords: conventional scanner, ctcss dcs squelch, wideband p25 phase 2, dmr lcn band plan, symbol scope op25, multi carrier sdr decode, gophertrunk conventional, iq power squelch
tags: [conventional, wideband, dmr, symbol-scope, dsp, decoding]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Protocol Decoders"
series_part: 9
---

*Part 9 of **Protocol Decoders**. Eight episodes have followed control channels
into grants. This one covers the four decode paths that *aren't* a single trunked
CC: fixed-frequency conventional scanning, one dongle watching many carriers at
once, learning a DMR band plan from thin air, and the symbol scope that lets you
*see* a modulation before you trust it. That last tool is exactly what you reach
for when a capture like **Mercury** refuses to name itself — which is where Parts
10–11 pick up.*

> **TL;DR:** Not every signal is a trunked control channel. GopherTrunk decodes
> **conventional** channels (fixed frequency, IQ-power squelch, optional CTCSS/DCS
> tone gate) via a scan-list state machine; monitors **many carriers on one
> wideband dongle** with a tuner `Bank` (including P25 Phase 2 and per-grant voice
> taps); **learns a DMR Tier III LCN→frequency band plan** by correlating grants
> with the carriers that key up; and exposes a **symbol scope** that reuses the
> production DSP to stream pre-slicer soft symbols and dibits to the web console.

**Key takeaways**

- Conventional decode has no grants — it's a **squelch + tone** state machine
  that manufactures synthetic calls for the engine.
- The wideband engine channelizes one SDR into many narrowband streams and can
  decode its **own voice grants** with per-grant taps, no second radio.
- The DMR LCN **Learner** discovers a band plan at runtime and hot-swaps it into
  the live control channel.
- The **symbol scope** is a read-only diagnostic that reuses the production
  down-converter and receiver taps — it never touches live decode.

## Cheat sheet

| Path | Package | What it does |
|---|---|---|
| Conventional | `internal/scanner/conventional/` | scan a fixed-frequency list, squelch + CTCSS/DCS, synth calls |
| Wideband multi-carrier | `internal/scanner/widebandt2/` | one dongle → many channels via `tuner.Bank`; P25 P1/P2, DMR |
| DMR LCN learning | `internal/scanner/dmrlcn/` | learn Tier III LCN→Hz from grants + onsets, hot-swap resolver |
| Symbol scope | `internal/scanner/symbolscope/` | stream pre-slicer soft symbols + dibits to the web console |

## In this post

- **Conventional** — squelch, tone gating, and the synthetic call.
- **Wideband Phase 2** — one dongle, many carriers, and voice taps.
- **DMR LCN learning** — reconstructing a band plan from evidence.
- **The symbol scope** — a microscope that reuses production DSP.

## Conventional: no grant, just a carrier

A conventional channel is the simplest radio there is: a fixed frequency with
analog FM voice and no trunking at all. There's nothing to decode in the
control-channel sense — the decode question is just *is someone talking right
now?* The scanner answers it with an **IQ-power squelch** and an optional
**sub-audible tone gate**:

```go
// internal/scanner/conventional/scanner.go (shape)
type Channel struct {
    Label       string
    FrequencyHz uint32
    Mode        string        // "fm" or "nfm" (narrows the audio LPF)
    SquelchDbFS float64       // carrier-present threshold, e.g. -50 dBFS
    Hangtime    time.Duration // below-threshold time before the call ends
    Priority    int
    Tone        ToneConfig    // optional CTCSS / DCS gate
}
```

The scanner is a three-state machine — `SCANNING`, `DWELL`, `HELD` — that hops the
list, measures IQ power in a short window per channel, and opens on a channel whose
power crosses `SquelchDbFS`. When a `ToneConfig` is set, the channel only counts as
"carrier present" while **both** the power squelch is open *and* the configured
CTCSS frequency (a Goertzel bin) or DCS code is detected — the classic way to share
a frequency between systems. CTCSS is fully wired; DCS parses and validates but its
detector is a tracked follow-up, and the code says so rather than pretending
otherwise.

The clever part is what happens on squelch-open: the scanner doesn't have its own
recording path. It **manufactures a synthetic grant** and hands it to the trunking
engine, which records it exactly like a trunked call:

```go
// internal/scanner/conventional/scanner.go (shape) — the Engine seam
type Engine interface {
    HandleSyntheticCall(g trunking.Grant, deviceSerial string)
    EndSyntheticCall(deviceSerial string, reason trunking.EndReason) bool
    Touch(deviceSerial string)
}
```

So a conventional channel reuses the entire recorder, call-log and metrics stack by
pretending to be a one-call trunked system. This is the same decoupling lesson from
[Part 1]({{ '/blog/deep-dives/protocol-decoders-01-anatomy-of-a-cc-decoder/' | relative_url }}),
applied from the *other* side: instead of a new consumer subscribing to the engine,
a new *producer* speaks the engine's grant vocabulary.

## Wideband: one dongle, many carriers

A wideband dongle sees several MHz at once, which is wasteful to spend on a single
control channel. The `widebandt2` engine pins one SDR to a centre frequency and
uses a `dsp/tuner.Bank` to extract **one narrowband IQ stream per configured
channel** inside the dongle's band — each stream decimated to a 48 kHz per-tap rate
and fed to its own protocol receiver and state machine:

<figure class="lab-figure">
<svg viewBox="0 0 680 176" width="680" height="176" role="img" aria-label="One wideband SDR pinned to a centre frequency feeds a tuner bank that extracts several narrowband channel streams, each decoded by its own protocol receiver, and per-grant voice taps decode voice from the same wideband IQ without a second radio">
  <rect x="14" y="66" width="120" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="74" y="86" text-anchor="middle" fill="var(--accent)" font-size="12">wideband SDR</text>
  <text x="74" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="10">pinned centre</text>
  <line x1="134" y1="89" x2="170" y2="89" stroke="currentColor"/>
  <polygon points="170,85 180,89 170,93" fill="currentColor"/>
  <rect x="180" y="60" width="96" height="58" rx="6" fill="none" stroke="currentColor"/>
  <text x="228" y="82" text-anchor="middle" fill="currentColor" font-size="12">tuner.Bank</text>
  <text x="228" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="10">→ 48 kHz taps</text>
  <g stroke="currentColor">
    <line x1="276" y1="72" x2="320" y2="44"/><polygon points="318,40 328,42 320,50" fill="currentColor"/>
    <line x1="276" y1="89" x2="320" y2="89"/><polygon points="320,85 330,89 320,93" fill="currentColor"/>
    <line x1="276" y1="106" x2="320" y2="134"/><polygon points="320,128 330,134 318,138" fill="currentColor"/>
  </g>
  <rect x="330" y="28" width="150" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="405" y="47" text-anchor="middle" fill="currentColor" font-size="10">P25 Phase 1 CC</text>
  <rect x="330" y="74" width="150" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="405" y="93" text-anchor="middle" fill="currentColor" font-size="10">P25 Phase 2 CC</text>
  <rect x="330" y="120" width="150" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="405" y="139" text-anchor="middle" fill="currentColor" font-size="10">DMR Tier II / III</text>
  <line x1="480" y1="89" x2="516" y2="89" stroke="var(--accent)"/>
  <polygon points="516,85 526,89 516,93" fill="var(--accent)"/>
  <rect x="526" y="64" width="140" height="50" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="596" y="84" text-anchor="middle" fill="var(--accent)" font-size="11">voice taps</text>
  <text x="596" y="100" text-anchor="middle" fill="var(--fg-muted)" font-size="10">per-grant DDC, same IQ</text>
</svg>
<figcaption>One wideband SDR channelized by a tuner bank into several decode streams; per-grant voice taps decode the voice from the same IQ window, so no second radio is needed unless a grant lands outside the band.</figcaption>
</figure>

Its supported channels are the modern digital control channels: DMR Tier II
conventional, DMR Tier III trunked, P25 Phase 1, and **P25 Phase 2** (the
`radio/p25/phase2` state machine consuming superframes and emitting grants from the
MAC PDU chain covered in
[Part 4]({{ '/blog/deep-dives/protocol-decoders-04-p25-phase-2-tdma-mac/' | relative_url }})).
The payoff is **voice taps**: because the voice channel is often already inside the
dongle's IQ window, a grant can be followed by spinning up a per-grant DDC on the
*same* wideband stream (`internal/sdr/wbvoice`) instead of allocating a physical
voice SDR — falling back to a real radio only when the grant lands outside the band
or every tap is busy.

## DMR LCN learning: a band plan from evidence

DMR Tier III grants reference a voice channel by **LCN** (Logical Channel Number),
not a frequency, so without a band plan those grants are undecodable
(`decode.error stage=no-bandplan`). Usually you configure the plan by hand. The
`dmrlcn.Learner` discovers it automatically, and it's a small gem of correlation
engineering: it watches the wideband IQ for **carrier onsets** (a channel keying
up) and the event bus for **`KindDMRGrantObserved`** events, and pairs them in
time. An onset that lines up with a grant for LCN *n* is a *candidate* LCN→frequency
mapping.

Because a temporal coincidence can be a fluke, each candidate is **decode-confirmed**
off the main loop — a bounded set of DDC tap probes actually decode the suspected
frequency and vote — before it's fed to a band-plan fitter:

```go
// internal/scanner/dmrlcn/dmrlcn.go (shape)
func (l *Learner) handlePair(p Pair) {
    l.pairs = append(l.pairs, p)
    res, ok := FitBandPlan(l.pairs, l.fitOpts) // linear or table fit
    if !ok { return }
    l.locked = true
    l.apply(res) // hot-swap tier3.ResolverFromPlan into the live CC
}
```

`FitBandPlan` tries a **linear** model first (base frequency + spacing + offset —
most real DMR systems are linear) and falls back to an explicit **table**. Once it
locks, `apply` hot-swaps the resolver into the *running* Tier III control channel
via `SetResolver`, publishes `KindDMRBandPlanLearned`, and optionally persists the
plan to config. The learner then goes quiet. Bounded probe workers keep a ~700 ms
DDC confirmation tap from ever stalling the IQ pump into the onset detector — the
same "slow work off the hot path" discipline the engine uses.

## The symbol scope: a microscope, not a decoder

The last tool in this episode isn't a decoder at all — it's the diagnostic you use
when a *decode fails* and you need to know why. The `symbolscope` package recovers a
live stream of the demodulated symbols — the pre-slicer soft waveform *plus* the
sliced dibit decisions — from a wideband feed, for the web console's "Symbol" scope
(GopherTrunk's answer to OP25's Symbol plot).

The design rule is the one that makes it trustworthy: **it reuses the production
DSP rather than re-implementing demod.** The same `ccdecoder.Downconverter` the live
decoder channelizes with feeds the same protocol receiver, and the receiver's
existing `SoftSink` / `DibitSink` taps surface the symbols. It runs as a *separate*
`Engine` on an `iqtap` broker subscription, so production control-channel decode is
never touched:

```go
// internal/scanner/symbolscope/scope.go (shape)
type Frame struct {
    SymbolRateHz float64
    Soft         []float32 // pre-slicer soft waveform (empty on CQPSK)
    SymI, SymQ   []float32 // complex symbol-decision points (CQPSK constellation)
    Dibits       []uint8   // sliced decisions, aligned index-for-index with Soft
    BaseIdx      int       // absolute symbol index, so a client can detect gaps
}
```

For P25 Phase 1, the **C4FM** path emits the FM-discriminator soft waveform aligned
index-for-index with the sliced dibits — a four-level eye — while the **CQPSK** path
emits the complex symbol-decision points (`SymI`/`SymQ`), the true constellation
clustering at the four ±45°/±135° positions. The two views answer different
questions: the eye tells you *timing and level*, the constellation tells you *phase*.
When a signal produces valid dibits but no lock, the scope usually shows you which
one is wrong at a glance — a far faster diagnosis than staring at a decode-error
counter.

<figure class="lab-figure">
<svg viewBox="0 0 680 168" width="680" height="168" role="img" aria-label="The symbol scope taps a separate engine off the IQ broker, reusing the production down-converter and receiver soft and dibit taps, rendering a four-level eye for C4FM and a QPSK constellation for CQPSK without touching live decode">
  <rect x="14" y="62" width="110" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="69" y="80" text-anchor="middle" fill="currentColor" font-size="11">iqtap broker</text>
  <text x="69" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="10">shared IQ</text>
  <line x1="124" y1="84" x2="158" y2="84" stroke="currentColor"/>
  <polygon points="158,80 168,84 158,88" fill="currentColor"/>
  <rect x="168" y="54" width="150" height="60" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="243" y="76" text-anchor="middle" fill="var(--accent)" font-size="11">separate Engine</text>
  <text x="243" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="10">prod Downconverter</text>
  <text x="243" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="10">+ receiver taps</text>
  <line x1="318" y1="72" x2="360" y2="52" stroke="currentColor"/>
  <polygon points="358,48 368,50 360,58" fill="currentColor"/>
  <line x1="318" y1="98" x2="360" y2="120" stroke="currentColor"/>
  <polygon points="358,114 368,120 356,124" fill="currentColor"/>
  <rect x="368" y="26" width="150" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="443" y="45" text-anchor="middle" fill="currentColor" font-size="11">4-level eye (C4FM)</text>
  <text x="443" y="61" text-anchor="middle" fill="var(--fg-muted)" font-size="10">Soft + Dibits</text>
  <rect x="368" y="100" width="150" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="443" y="119" text-anchor="middle" fill="currentColor" font-size="11">constellation (CQPSK)</text>
  <text x="443" y="135" text-anchor="middle" fill="var(--fg-muted)" font-size="10">SymI / SymQ</text>
  <rect x="548" y="62" width="118" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="607" y="80" text-anchor="middle" fill="var(--accent)" font-size="11">web console</text>
  <text x="607" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="10">Symbol scope</text>
  <line x1="518" y1="49" x2="548" y2="72" stroke="var(--fg-muted)"/>
  <line x1="518" y1="123" x2="548" y2="96" stroke="var(--fg-muted)"/>
</svg>
<figcaption>The symbol scope runs a second engine off the shared IQ broker, reusing the production down-converter and receiver taps — so what it renders is exactly what the live decoder sees, without perturbing it.</figcaption>
</figure>

## Where this goes next

[Part 10]({{ '/blog/deep-dives/protocol-decoders-10-alias-hunt-framing/' | relative_url }})
turns from decoding *messages* to decoding a *cipher*. The P25 talker-alias field —
one of the most obscure corners of the standard — carries an encrypted display name,
and framing it correctly is the whole first half of the puzzle the series has been
hinting at since Part 1. For the standards here, the
[conventional radio]({{ '/reference/conventional-radio/' | relative_url }}),
[DMR Tier III]({{ '/reference/dmr-tier-3/' | relative_url }}) and
[P25 Phase 2]({{ '/reference/p25-phase-2/' | relative_url }}) reference pages go
deeper.

## FAQ

**How does GopherTrunk record a conventional (non-trunked) channel?**
The conventional scanner detects a carrier with an IQ-power squelch (and optional
CTCSS/DCS tone gate), then manufactures a *synthetic* `trunking.Grant` and hands it
to the engine via `HandleSyntheticCall`. From the engine's side it's an ordinary
one-call system, so the whole recorder and call-log stack is reused unchanged.

**Can one SDR dongle decode multiple systems at once?**
Yes. The `widebandt2` engine pins one dongle to a centre frequency and uses a tuner
bank to extract a narrowband stream per channel, each feeding its own receiver.
With voice taps enabled it can even decode its own voice grants from the same IQ,
so a single dongle covers both control and voice within its band.

**What is DMR LCN learning?**
A runtime discovery of the Tier III LCN→frequency band plan. The learner correlates
carrier onsets on the wideband IQ with observed grants, decode-confirms the
candidates with bounded DDC probes, fits a linear-or-table band plan, and hot-swaps
it into the live control channel — so a system you didn't pre-configure starts
following voice.

**Is the symbol scope the same decoder as the live path?**
It reuses the same production down-converter and the receiver's soft/dibit taps, but
runs as a *separate* engine on an IQ broker subscription so it can't perturb live
decode. What it renders is faithful to what the live decoder sees; it just doesn't
share the same instance.

## Series navigation

**Part 9 of 12** · ←
[Part 8: EDACS, LTR & MPT-1327]({{ '/blog/deep-dives/protocol-decoders-08-edacs-ltr-mpt1327/' | relative_url }})
· Next →
[Part 10: The Alias Hunt I]({{ '/blog/deep-dives/protocol-decoders-10-alias-hunt-framing/' | relative_url }})
