---
title: "Protocol Decoders, Part 1: Anatomy of a Control-Channel Decoder"
description: How every GopherTrunk control-channel decoder shares one shape — symbols to framing to FEC to PDU to opcode to bus event — behind a single pipeline interface and a protocol-keyed factory registry.
category: deep-dives
keywords: control channel decoder, p25 dmr nxdn decoder, sdr protocol pipeline, symbol to pdu, fec framing, decoder registry, dibit stream, gophertrunk protocol decoders, trunking decode pipeline
tags: [protocol-decoders, p25, dmr, dsp, architecture, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Protocol Decoders"
series_part: 1
---

*Part 1 of **Protocol Decoders**, a 12-part deep dive into how GopherTrunk turns
raw symbols into structured trunking events. Every protocol — P25, DMR, NXDN,
TETRA, EDACS, and the rest — runs through the same eight-stage shape, and this
opener is a map of that shape. It also plants a mystery that runs the length of
the series: one Motorola field that decodes to garbage, and stubbornly stays that
way until Parts 10 and 11.*

> **TL;DR:** A control-channel decoder is a **pipeline** with a fixed skeleton:
> IQ → symbols → sync/framing → FEC → PDU → opcode dispatch → a typed event on
> the bus. GopherTrunk factors that skeleton into one interface
> (`ProtocolPipeline`) and one registry (a `map[trunking.Protocol]PipelineFactory`),
> so a dozen protocols share the connector, the down-converter, and the event
> plumbing — and only differ in the middle. The daemon and the offline Signal Lab
> drive the *same* pipelines through the *same* factory map, which is why lab
> results transfer to the air.

**Key takeaways**

- Every decoder is the same eight stages; protocols differ only in *how* each
  stage is done, not in the order or the contract.
- The pipeline contract is three methods — `Process`, `Reset`, `Close` — and
  nothing else. The connector doesn't know which protocol it's driving.
- A **factory registry** keyed on `trunking.Protocol` is the single source of
  truth for "which protocols can decode," used by the daemon, the picker, and the
  test harness alike.
- The decoder **publishes** `KindCCLocked` and `KindGrant`; it never calls the
  trunking engine. That grant is the hand-off to the
  [Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }}) series.

## Cheat sheet

| Stage | What it does | Where it lives |
|---|---|---|
| Down-convert | raw SDR IQ → narrowband channel (~48 kHz) | `ccdecoder/decoder.go` (`Downconverter`) |
| Demod + timing | channel IQ → symbols (dibits or bits) | `internal/radio/<proto>/receiver` |
| Sync / framing | find the frame sync word, slice a burst | e.g. `p25/phase1/sync.go` |
| FEC | correct channel errors (trellis, BCH, Golay…) | e.g. `p25/phase1/trellis.go` |
| PDU parse | bytes → a structured block | e.g. `p25/phase1/tsbk.go` |
| Opcode dispatch | block → a specific message type | e.g. `p25/phase1/opcodes.go` |
| Event | publish `KindGrant` / `KindCCLocked` | `internal/events` |
| Registry | pick the pipeline for a protocol | `ccdecoder/pipelines.go` |

## In this post

- **The eight stages** every decoder walks, and why they're always in that order.
- **The pipeline contract** — the three-method interface that hides the protocol.
- **The registry** — one map that answers "what can we decode?"
- **The connector** — how one IQ stream feeds whichever pipeline is active.
- **The mystery** — a Motorola alias field that decodes to noise, on purpose.

## The shape every decoder shares

Point GopherTrunk at a P25 control channel and a DMR Tier III control channel and
the code paths look nothing alike in the middle — different sync words, different
FEC, different PDUs. But zoom out and they are the *same program*. Both take a
stream of recovered symbols, hunt for a frame sync pattern, run forward error
correction, parse the corrected bits into a protocol data unit, dispatch on an
opcode, and emit a structured event. The protocol picks the details; the pipeline
picks the order, and the order never changes.

<figure class="lab-figure">
<svg viewBox="0 0 680 150" width="680" height="150" role="img" aria-label="The eight-stage decoder pipeline: IQ down-conversion, demodulation to symbols, frame sync, forward error correction, PDU parse, opcode dispatch, and a published bus event">
  <rect x="4" y="54" width="96" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="52" y="72" text-anchor="middle" fill="currentColor" font-size="11">down-conv</text>
  <text x="52" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">IQ → channel</text>
  <line x1="100" y1="76" x2="118" y2="76" stroke="currentColor"/><polygon points="118,72 128,76 118,80" fill="currentColor"/>
  <rect x="128" y="54" width="90" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="173" y="72" text-anchor="middle" fill="currentColor" font-size="11">demod</text>
  <text x="173" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ symbols</text>
  <line x1="218" y1="76" x2="236" y2="76" stroke="currentColor"/><polygon points="236,72 246,76 236,80" fill="currentColor"/>
  <rect x="246" y="54" width="80" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="286" y="72" text-anchor="middle" fill="var(--accent)" font-size="11">sync</text>
  <text x="286" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">framing</text>
  <line x1="326" y1="76" x2="344" y2="76" stroke="currentColor"/><polygon points="344,72 354,76 344,80" fill="currentColor"/>
  <rect x="354" y="54" width="70" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="389" y="79" text-anchor="middle" fill="var(--accent)" font-size="11">FEC</text>
  <line x1="424" y1="76" x2="442" y2="76" stroke="currentColor"/><polygon points="442,72 452,76 442,80" fill="currentColor"/>
  <rect x="452" y="54" width="76" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="490" y="72" text-anchor="middle" fill="currentColor" font-size="11">PDU</text>
  <text x="490" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">+ opcode</text>
  <line x1="528" y1="76" x2="546" y2="76" stroke="currentColor"/><polygon points="546,72 556,76 546,80" fill="currentColor"/>
  <rect x="556" y="54" width="120" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="616" y="72" text-anchor="middle" fill="var(--accent)" font-size="11">bus event</text>
  <text x="616" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Grant / CCLocked</text>
  <text x="340" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the details change per protocol; the order never does</text>
</svg>
<figcaption>Every control-channel decoder in GopherTrunk is this pipeline. The accent stages are where the protocol identity lives; the outer stages are shared machinery.</figcaption>
</figure>

The reason to name the shape is that it tells you where to look. A capture that
demods clean but never locks is a *framing* problem. One that locks but throws CRC
errors is *FEC* or *PDU*. One that decodes blocks but produces no calls is
*opcode dispatch*. The [Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }})
dashboard reads exactly these stage boundaries, and the rest of this series walks
them protocol by protocol.

## The pipeline contract

The connector that owns the SDR's IQ stream does not import P25, DMR, or any
protocol package. It talks to one interface:

```go
// internal/scanner/ccdecoder/pipelines.go (shape)
type ProtocolPipeline interface {
    Process(iq []complex64) // consume one chunk of channel IQ
    Reset()                 // clear symbol-domain state on re-sync
    Close() error           // release resources (idempotent)
}
```

Three methods. `Process` is the hot path — a chunk of down-converted IQ goes in,
and somewhere at the bottom of the call stack a `KindGrant` may come out on the
bus. `Reset` flushes symbol-clock and framing state when the stream re-syncs.
`Close` tears the pipeline down on a retune. That's the whole surface. Whether the
protocol is 4-level C4FM at 4800 baud or π/4-DQPSK TETRA at 18000 baud is
invisible here — the pipeline swallows the difference.

Construction is where a protocol's identity enters, through a small options
struct the connector fills in the same way for everyone:

```go
// internal/scanner/ccdecoder/pipelines.go (shape)
type PipelineOptions struct {
    Bus          *events.Bus
    Log          *slog.Logger
    SystemName   string
    FrequencyHz  uint32
    SampleRateHz float64          // the *channelized* rate, not the SDR rate
    System       trunking.System  // per-protocol config rides here
    SymbolTap    func(symbols []uint8, isBits bool, baseIdx int)
    // …FECObserver, CarrierBiasHz
}
```

Note `SampleRateHz` is the **decimated** channel rate, not the raw SDR rate. The
connector down-converts every chunk to a narrowband stream (~48 kHz for the
4800-baud C4FM family, wider for TETRA) *before* the pipeline sees it, and each
per-protocol receiver sizes its matched filter to that rate. This is what makes
the decode path rate-invariant to the capture rate — a property the DSP notes in
this repo lean on hard. `SymbolTap` is a pure observation hook: production leaves
it nil, and the offline toolkit wires it to count symbols and grade signal quality
for *every* protocol without re-building the receiver.

## The registry: one map, one truth

"Which protocols can GopherTrunk actually decode?" has exactly one answer in the
codebase, and it's a map:

```go
// internal/scanner/ccdecoder/pipelines.go (shape)
var factories = map[trunking.Protocol]PipelineFactory{
    trunking.ProtocolP25:       newP25Phase1Pipeline,
    trunking.ProtocolP25Phase2: newP25Phase2Pipeline,
    trunking.ProtocolDMR:       newDMRTier3Pipeline,
    trunking.ProtocolNXDN:      newNXDNPipeline,
    trunking.ProtocolTETRA:     newTETRAPipeline,
    trunking.ProtocolEDACS:     newEDACSPipeline,
    trunking.ProtocolMotorola:  newMotorolaPipeline,
    // …dPMR, LTR, MPT1327, YSF, D-STAR, DMR Tier 1/2
}

type PipelineFactory func(PipelineOptions) (ProtocolPipeline, error)
```

Each value is a `PipelineFactory` — a function that wires a protocol's receiver to
its control-channel state machine and returns something satisfying
`ProtocolPipeline`. The map is populated once at package load and treated as
immutable for the process lifetime. Three small accessors read it:
`NewPipeline` (build one), `HasFactory` (can we decode this?), and
`RegisteredProtocols` (list them, sorted). The Signal Lab protocol picker, a
gen→decode test sweep, and the live daemon all go through those same three doors.

### How that principle shaped the Go code

- **The connector is protocol-agnostic.** `ccdecoder`'s `Decoder` imports
  `internal/events` and `internal/trunking` — not P25 or DMR. It looks up a factory
  by `trunking.Protocol` and drives whatever comes back. Adding a protocol is
  adding a map entry plus a factory, never editing the pump loop.
- **One rate, sized once.** Because factories receive the channelized
  `SampleRateHz`, a receiver's matched filter is correct whether the SDR ran at
  2.048 MS/s or 10 MS/s. The DDC absorbs the capture rate.
- **A factory may decline.** Returning an error means "this protocol isn't wired
  end-to-end yet," and the connector leaves the system in `hunting` rather than
  decoding noise into spurious events. The registry encodes capability honestly.
- **The same map powers offline tools.** `NewPipeline` is the out-of-package entry
  point the offline toolkit uses, so what the lab decodes is byte-for-byte what the
  daemon decodes.

<figure class="lab-figure">
<svg viewBox="0 0 640 180" width="640" height="180" role="img" aria-label="One IQ stream feeds a factory registry that constructs the active per-protocol pipeline, whose control-channel state machine publishes grant and lock events on the shared bus">
  <rect x="8" y="72" width="110" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="63" y="90" text-anchor="middle" fill="currentColor" font-size="11">control SDR</text>
  <text x="63" y="105" text-anchor="middle" fill="var(--fg-muted)" font-size="9">one IQ stream</text>
  <line x1="118" y1="92" x2="150" y2="92" stroke="currentColor"/><polygon points="150,88 160,92 150,96" fill="currentColor"/>
  <rect x="160" y="60" width="120" height="64" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="220" y="84" text-anchor="middle" fill="var(--accent)" font-size="11">ccdecoder</text>
  <text x="220" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">DDC + pump</text>
  <text x="220" y="113" text-anchor="middle" fill="var(--fg-muted)" font-size="9">swaps pipeline</text>
  <line x1="280" y1="92" x2="312" y2="92" stroke="currentColor"/><polygon points="312,88 322,92 312,96" fill="currentColor"/>
  <rect x="322" y="30" width="150" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="397" y="49" text-anchor="middle" fill="var(--fg-muted)" font-size="10">factory registry</text>
  <line x1="397" y1="60" x2="397" y2="74" stroke="var(--fg-muted)"/><polygon points="393,74 397,84 401,74" fill="var(--fg-muted)"/>
  <rect x="322" y="72" width="150" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="397" y="90" text-anchor="middle" fill="var(--accent)" font-size="11">active pipeline</text>
  <text x="397" y="105" text-anchor="middle" fill="var(--fg-muted)" font-size="9">rx → CC state machine</text>
  <line x1="472" y1="92" x2="504" y2="92" stroke="currentColor"/><polygon points="504,88 514,92 504,96" fill="currentColor"/>
  <rect x="514" y="72" width="118" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="573" y="90" text-anchor="middle" fill="var(--accent)" font-size="11">event bus</text>
  <text x="573" y="105" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Grant · CCLocked</text>
  <text x="320" y="152" text-anchor="middle" fill="var(--fg-muted)" font-size="10">HuntProgress tells the connector which system/frequency to attempt; it swaps the pipeline to match</text>
</svg>
<figcaption>The connector owns one IQ stream and swaps the active pipeline as the hunter retunes. The registry is the lookup; the bus is the exit.</figcaption>
</figure>

## The connector, in one loop

`ccdecoder.Decoder` is the long-lived object gluing the SDR to the registry. It
subscribes to `KindHuntProgress` (the CC hunter saying "I'm trying system X on
frequency F now"), and on each transition it rebuilds the down-converter for that
protocol's channel rate, looks up the factory, and swaps the active pipeline:

```go
// internal/scanner/ccdecoder/decoder.go (shape) — handleProgress
factory, ok := factories[sys.Protocol]
if !ok {
    d.clearActive() // no decoder for this protocol; stay hunting
    return
}
d.ensureDownconverterLocked(ddcTargetForProtocol(sys.Protocol))
p2, err := factory(PipelineOptions{
    Bus: d.bus, SystemName: sys.Name,
    FrequencyHz: p.AttemptedFreqHz, SampleRateHz: d.pipelineRateHz,
    System: sys,
})
// …then, under lock: d.active = p2
```

From there, every IQ chunk goes `d.ddc.Process(...)` then `d.active.Process(...)`.
One subtlety is worth a callout: a `HuntProgress` that *repeats* the same
`(system, frequency)` must **not** tear down and rebuild the pipeline, or a
single-candidate system that re-hunts every dwell resets acquisition every few
seconds and never converges. So the swap is idempotent — same target, same
pipeline, keep decoding. Small rule, big consequence for lock stability.

The output is never a function call into the engine. The pipeline's state machine
`Publish`es `KindCCLocked` when it first locks and `KindGrant` when it decodes a
voice-channel grant — the seam between this series and the
[Trunking Engine]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }})
series: we produce the `Grant`; the engine turns it into a recorded call.

## The mystery we're planting

> **⚠️ One field that won't decode.** On a Motorola P25 Phase 1 system, radios can
> broadcast a **talker alias** — the display name of the transmitting unit —
> reassembled from Link Control words (`LCO 0x15` header, `0x17` data blocks) and
> passed through a proprietary per-byte substitution cipher. GopherTrunk
> reassembles the blocks perfectly and runs the decode, and it comes out as
> garbage. The code even knows it: `decodeAlias` will **never** report an alias as
> reliable, because `motorola.CipherVerified` is `false` — the substitution table
> is unconfirmed (issue #773). It is, quite literally, the cipher we haven't
> cracked. We'll pass this field lightly through the middle of the series, then
> hunt it down for real in
> [Part 10]({{ '/blog/deep-dives/protocol-decoders-10-alias-hunt-framing/' | relative_url }})
> and [Part 11]({{ '/blog/deep-dives/protocol-decoders-11-alias-hunt-cryptanalysis/' | relative_url }}).
> It is the same emitter the
> [Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}) series calls
> *Mercury*.

Keep it in the back of your mind. Everything up to Part 9 is the machinery you need
to even *see* that field arrive intact; Parts 10 and 11 are where we try to read it.

## Where this goes next

[Part 2]({{ '/blog/deep-dives/protocol-decoders-02-p25-phase-1-physical-layer/' | relative_url }})
drops into the first protocol in detail: P25 Phase 1's physical layer — the NAC/NID,
the 48-bit frame sync word, the C4FM 4-level symbols, and the trellis + Hamming FEC
that together act as the *lock gate*. From there we climb the stack: TSBKs and link
control (Part 3), Phase 2 TDMA (Part 4), then out through the DMR, NXDN/dPMR, TETRA,
and EDACS/LTR/MPT families before the two-part alias hunt.

## FAQ

**What is a control-channel decoder, exactly?**
It's the component that turns a trunked system's continuous signalling channel into
structured events — chiefly "talkgroup T is now on frequency F" (a grant). In
GopherTrunk it's a pipeline from IQ samples to a `KindGrant` event on the bus,
built per protocol but sharing one interface and one registry.

**Do all protocols really share the same pipeline?**
They share the same *contract* (`Process`/`Reset`/`Close`) and the same connector,
down-converter, and event plumbing. The middle stages — sync, FEC, PDU, opcode —
are protocol-specific implementations wired together by a per-protocol factory.

**How does GopherTrunk pick which decoder to run?**
The CC hunter publishes a `HuntProgress` event naming the system and frequency it's
attempting; the connector looks up that system's `trunking.Protocol` in the factory
map and builds the matching pipeline. If no factory is registered, it stays hunting.

**Where do grants go after decoding?**
Onto the event bus as `KindGrant`. The decoder never calls the trunking engine
directly — the engine is a subscriber. That decoupling is covered in the
[Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }}) series.

**What's the talker-alias mystery?**
A Motorola talker-alias field that reassembles correctly but decodes to noise
because its per-byte cipher is unverified. It threads through the series and is
paid off in Parts 10–11; it's the same emitter Crypto Lab calls *Mercury*.

## Series navigation

**Part 1 of 12** · Next →
[Part 2: P25 Phase 1 Physical Layer]({{ '/blog/deep-dives/protocol-decoders-02-p25-phase-1-physical-layer/' | relative_url }})
