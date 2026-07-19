---
title: "Protocol Decoders, Part 8: The Legacy Family — EDACS, LTR & MPT-1327"
description: How GopherTrunk follows the pre-digital trunking systems — EDACS 9600-baud CCWs, LTR's central-control-free per-repeater status words, and MPT-1327 FFSK codewords — and why they decode nothing like a modern digital control channel.
category: deep-dives
keywords: edacs decoder, ltr trunking, mpt-1327 ffsk, legacy trunking scanner, edacs ccw, ltr status word, go to channel grant, gophertrunk legacy protocols
tags: [edacs, ltr, mpt1327, trunking, decoding, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Protocol Decoders"
series_part: 8
---

*Part 8 of **Protocol Decoders**. Parts 2–7 built up the modern digital control
channel — typed PDUs, strong FEC, an explicit grant message with a talkgroup and
a frequency. This episode goes the other direction, to the three legacy families
still on the air: EDACS, LTR and MPT-1327. They predate the "one clean digital CC"
idea, and each solves the trunking problem with what the 1970s–80s had on hand.
Understanding them is also a reminder for the **Mercury** hunt ahead — an unknown
emitter might not be modern at all.*

> **TL;DR:** EDACS runs a continuous **9600-baud** GFSK control channel of 40-bit
> Control Channel Words (CCWs), each protected by a shortened BCH(40,28,2). LTR
> has **no central control channel at all** — every repeater transmits a 41-bit
> status word at **300 bps** under the voice, and the scanner follows calls by
> watching all of them. MPT-1327 uses **1200-baud FFSK** carrying 64-bit
> codewords (38 info + 26 BCH check). All three land in the same engine through
> `events.KindGrant`, but the decode front-ends could hardly be more different.

**Key takeaways**

- **EDACS** is the closest to modern: a dedicated fast digital CC with a per-word
  BCH and a clean `Command` opcode enum.
- **LTR** is *distributed* trunking — there is no control channel; call-follow
  means reading every repeater's sub-audible status word.
- **MPT-1327** is FFSK tone signalling: 64-bit codewords, a `GoToChannel` (GTC)
  grant, and BCH left to the caller.
- All three converge on the **same `trunking.Grant`** the engine consumes — the
  legacy weirdness is quarantined in the decoders, not the engine.

## Cheat sheet

| System | Control scheme | Unit | Grant message | Where |
|---|---|---|---|---|
| EDACS | continuous 9600-baud GFSK CC | 40-bit CCW | `GroupVoiceGrant` | `internal/radio/edacs/` |
| LTR | *no* central CC — per-repeater | 41-bit status word @ 300 bps | active `Status` (F-bit) | `internal/radio/ltr/` |
| MPT-1327 | continuous 1200-baud FFSK CC | 64-bit codeword | `GoToChannel` (GTC) | `internal/radio/mpt1327/` |

## In this post

- **What "legacy trunking" means** — three answers to one problem.
- **EDACS** — the fast digital CC and the CCW.
- **LTR** — distributed trunking with no control channel.
- **MPT-1327** — FFSK codewords and the GTC grant.
- **The common denominator** — how all three feed one engine.

## Three answers to one problem

Every trunked system answers the same question: *when a user keys up, which
frequency does everyone's radio jump to?* Modern digital systems (P25, DMR, NXDN,
TETRA) answer it with a dedicated digital control channel streaming typed,
FEC-protected PDUs — the world Parts 2–7 lived in. The legacy family answers it
three different, older ways, and GopherTrunk keeps each one in its own package
with the same overall shape: a wire parser, an opcode/field layer, a band-plan
resolver, and a `control.go` state machine that emits `cc.locked` and
`events.KindGrant`.

<figure class="lab-figure">
<svg viewBox="0 0 680 176" width="680" height="176" role="img" aria-label="Three legacy trunking control schemes: EDACS 9600-baud CCW stream, LTR distributed per-repeater status words with no central control channel, and MPT-1327 1200-baud FFSK codewords, all converging on one trunking grant into the engine">
  <text x="112" y="20" text-anchor="middle" fill="var(--accent)" font-size="12" font-weight="bold">EDACS</text>
  <rect x="20" y="30" width="184" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="112" y="48" text-anchor="middle" fill="currentColor" font-size="11">9600-baud GFSK CC</text>
  <text x="112" y="63" text-anchor="middle" fill="var(--fg-muted)" font-size="10">40-bit CCW · BCH(40,28,2)</text>
  <text x="340" y="20" text-anchor="middle" fill="var(--accent)" font-size="12" font-weight="bold">LTR</text>
  <rect x="238" y="30" width="204" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="340" y="48" text-anchor="middle" fill="currentColor" font-size="11">no central CC</text>
  <text x="340" y="63" text-anchor="middle" fill="var(--fg-muted)" font-size="10">41-bit status @ 300 bps / repeater</text>
  <text x="576" y="20" text-anchor="middle" fill="var(--accent)" font-size="12" font-weight="bold">MPT-1327</text>
  <rect x="476" y="30" width="184" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="568" y="48" text-anchor="middle" fill="currentColor" font-size="11">1200-baud FFSK CC</text>
  <text x="568" y="63" text-anchor="middle" fill="var(--fg-muted)" font-size="10">64-bit codeword · GTC grant</text>
  <line x1="112" y1="72" x2="300" y2="120" stroke="var(--fg-muted)"/>
  <line x1="340" y1="72" x2="340" y2="120" stroke="var(--fg-muted)"/>
  <line x1="568" y1="72" x2="380" y2="120" stroke="var(--fg-muted)"/>
  <rect x="250" y="120" width="180" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="340" y="138" text-anchor="middle" fill="var(--accent)" font-size="12">trunking.Grant</text>
  <text x="340" y="153" text-anchor="middle" fill="var(--fg-muted)" font-size="10">events.KindGrant → engine</text>
</svg>
<figcaption>Three unrelated control schemes, one convergence point: each decoder emits the same protocol-neutral grant the trunking engine already knows how to follow.</figcaption>
</figure>

## EDACS: the fast digital CC

EDACS (Enhanced Digital Access Communications System, the GE-Marc / Ericsson
lineage) is the most "modern-feeling" of the three. It runs a **continuous
9600-baud GFSK control channel** over which 40-bit **Control Channel Words** flow,
each prefaced by a 24-bit sync pattern and protected by a shortened
**BCH(40,28,2)** code. A CCW packs four fields — `{Command, Status, Address, LCN,
Aux}` — and the `Command` nibble is a clean opcode enum:

```go
// internal/radio/edacs/opcodes.go (shape)
const (
    CmdIdle            Command = 0x0
    CmdGroupVoiceGrant Command = 0x1
    CmdProVoiceGrant   Command = 0x2 // EDACS ProVoice (digital voice)
    CmdDataGrant       Command = 0x4
    CmdSystemID        Command = 0x5
    CmdAdjacentSite    Command = 0x6
    // ...
)
```

A voice grant is a CCW whose command is `CmdGroupVoiceGrant` (or the digital
`CmdProVoiceGrant`); `AsGroupVoiceGrant` lifts the talkgroup out of `Address` and
resolves the **LCN** (Logical Channel Number) through the operator's band plan to a
frequency. The encrypted and emergency flags are the low two bits of the `Status`
nibble. The BCH decode is an opt-in (`SetBCHMode`, wired to an
`edacs_bch_mode` YAML key) because per the canonical open reference it's the
*only* on-wire FEC — an earlier package comment claiming an interleaved
Reed-Solomon layer above the BCH was simply wrong, and the code says so. Honesty
about what's real matters more than a tidy-sounding claim; the same instinct runs
through the whole codebase.

What's *not* wired is honest too: EDACS ProVoice / Aegis digital voice uses a
proprietary AMBE-derived vocoder GopherTrunk doesn't decode, so a `CmdProVoiceGrant`
gets you the grant metadata but not (yet) the audio.

## LTR: trunking with no control channel

LTR (Logic Trunked Radio, E.F. Johnson, 1970s) is the outlier that makes the whole
"control channel" abstraction wobble, because **it doesn't have one.** Instead,
every repeater in the system continuously transmits its own **41-bit status word
at 300 bps**, riding *under* the in-band voice. There is no central coordinator; a
scanner follows a call by watching every repeater's status word and tuning to
whichever one currently announces the talkgroup of interest.

```go
// internal/radio/ltr/status.go (shape)
type Status struct {
    Sync    bool
    Area    uint8  // 5-bit — disambiguates co-channel LTR systems
    Group   bool   // the "F-bit": 1 = active call for GroupID on this repeater
    Channel uint8  // 4-bit physical channel (1..20)
    Home    uint8  // 5-bit home-repeater number for the active group
    GroupID uint16 // 8-bit talkgroup (1..250)
    Free    uint8  // 5-bit free-repeater hint (for handoff)
    FCS     uint16 // 12-bit frame check
}

func (s Status) IsActive() bool { return s.Group && s.GroupID != 0 }
```

The `IsActive` test — F-bit set and a non-zero group — is the entire "is this a
grant?" logic. When it's true, the per-repeater state machine republishes the
status as `events.KindGrant`; the first well-formed status from a repeater also
fires a one-shot `cc.locked` so the hunter can confirm it's tuned to a real LTR
site. Because noise can occasionally pass a bare sync test, `IsWellFormed` gates on
the fixed-range fields (channel 1..20, home 1..20) before the state machine trusts
a word — a small but important guard when your "control channel" is a 41-bit
sub-audible word with a 12-bit check.

### How that principle shaped the Go code

LTR forces the same abstraction every other protocol uses even though it violates
the abstraction's premise. The engine only understands "a grant arrived on some
frequency for some talkgroup." LTR has no grant *message* — it has a *state*
(this repeater is active for this group right now). The decoder's job is to
translate that state into the engine's event vocabulary, and it does so without
the engine ever learning that LTR is special. That's the payoff of the
publish-a-neutral-`Grant` contract from
[Part 1]({{ '/blog/deep-dives/protocol-decoders-01-anatomy-of-a-cc-decoder/' | relative_url }}):
a protocol with a completely alien topology bolts on as one more `control.go`, and
the [Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }}) is none
the wiser.

## MPT-1327: FFSK codewords

MPT-1327 (the 1988 UK Code of Practice, still running taxi, transport and utility
fleets across Europe, Australia and beyond) sits between the two. It has a proper
dedicated control channel like EDACS, but the physical layer is **1200-baud FFSK**
— audio-frequency-shift keying with 1200/1800 Hz mark and space, the same tone
family as classic POCSAG. It carries **64-bit codewords** back-to-back: 38
information bits plus a 26-bit BCH(63,38)-derived check folded into the unit.

The interesting decode detail is that the message type lives in the *upper 4 bits
of the 17-bit Function field*, the spec's "Address Categorisation" subfield:

```go
// internal/radio/mpt1327/opcodes.go (shape)
const (
    KindAloha    CodewordKind = 0x1 // ALH  — control-channel idle
    KindAhoy     CodewordKind = 0x2 // AHY  — paging / inquiry
    KindAhoyChan CodewordKind = 0x3 // AHYC — broadcast / system info
    KindGoToChan CodewordKind = 0x4 // GTC  — voice grant ("go to channel")
)

func (c Codeword) Kind() CodewordKind { return CodewordKind((c.Function >> 13) & 0xF) }
```

The grant is a **GTC** ("Go To Channel"): `AsGoToChannel` reads the assigned
channel number out of the lower 13 bits of Function and the called party from
`Prefix + Ident`. The state machine locks on the first valid **Aloha** (the idle
"I am a control channel" beacon) or **AHYC** broadcast, then republishes GTCs as
`events.KindGrant` with `Protocol="mpt1327"`. As with LTR, the demodulator (the
1200-baud FFSK front-end) and the BCH(63,38) correction are honest deferrals — the
codeword parser assumes clean, error-corrected bits arrive from upstream.

<figure class="lab-figure">
<svg viewBox="0 0 680 150" width="680" height="150" role="img" aria-label="Comparison of grant-message richness across four eras: LTR carries an implicit active-state F-bit, MPT-1327 a GTC channel number, EDACS a CCW command with talkgroup and LCN, and modern digital P25 a fully typed TSBK with explicit talkgroup, source and frequency">
  <line x1="30" y1="120" x2="650" y2="120" stroke="var(--fg-muted)"/>
  <polygon points="650,116 660,120 650,124" fill="var(--fg-muted)"/>
  <text x="640" y="140" text-anchor="end" fill="var(--fg-muted)" font-size="10">message richness →</text>
  <rect x="40" y="80" width="120" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="100" y="96" text-anchor="middle" fill="currentColor" font-size="11">LTR</text>
  <text x="100" y="109" text-anchor="middle" fill="var(--fg-muted)" font-size="9">implicit F-bit state</text>
  <rect x="185" y="70" width="120" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="245" y="86" text-anchor="middle" fill="currentColor" font-size="11">MPT-1327</text>
  <text x="245" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">GTC channel number</text>
  <rect x="330" y="58" width="120" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="390" y="74" text-anchor="middle" fill="currentColor" font-size="11">EDACS</text>
  <text x="390" y="87" text-anchor="middle" fill="var(--fg-muted)" font-size="9">CCW cmd + TG + LCN</text>
  <rect x="475" y="42" width="140" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="545" y="58" text-anchor="middle" fill="var(--accent)" font-size="11">P25 / digital</text>
  <text x="545" y="71" text-anchor="middle" fill="var(--fg-muted)" font-size="9">typed TSBK: TG+src+freq</text>
</svg>
<figcaption>Grant "richness" rises across eras: LTR conveys an implicit active state, MPT-1327 a bare channel number, EDACS a structured command, and modern digital a fully typed PDU with source and frequency.</figcaption>
</figure>

## The common denominator

For all their differences at the wire, the three legacy decoders converge exactly
where the modern ones do: each emits a `trunking.Grant` on `events.KindGrant`, and
each `control.go` follows the shared idiom — lock on the first credible frame, keep
a band-plan `Resolver` to turn a channel/LCN into Hz, and gate on a
strict-validation flag so a noisy word doesn't spawn a phantom call. The engine
that records the resulting call has no idea whether the grant came from a P25 TSBK,
a TETRA D-CONNECT, an EDACS CCW, an LTR F-bit, or an MPT-1327 GTC. That is the
entire point of decoupling the decoder from the engine, and it's why adding a
40-year-old protocol is a decoder-package problem, never an engine problem.

## Where this goes next

[Part 9]({{ '/blog/deep-dives/protocol-decoders-09-conventional-wideband/' | relative_url }})
leaves trunking behind entirely for the *non*-trunked world: conventional
(fixed-frequency) decode, wideband Phase 2, DMR LCN mapping, and the symbol-scope
diagnostic that lets you *see* a modulation before you trust a decode. For the
standards themselves, the [EDACS]({{ '/reference/edacs/' | relative_url }}),
[LTR]({{ '/reference/ltr/' | relative_url }}) and
[MPT-1327]({{ '/reference/mpt-1327/' | relative_url }}) reference pages go deeper on
each air interface.

## FAQ

**Why does LTR not have a control channel?**
Because it's *distributed* trunking — an intentionally decentralised 1970s design
where every repeater advertises its own state. Each transmits a 41-bit status word
under the voice at 300 bps, and any radio (or scanner) reconstructs the system's
state by listening to all of them. There is no central coordinator to point a
"control channel" at.

**What FEC does EDACS use on the control channel?**
A shortened BCH(40,28,2) per 40-bit Control Channel Word — and, per the canonical
open reference, that is the *only* on-wire FEC layer. GopherTrunk decodes it as an
opt-in (`edacs_bch_mode`); there is no additional Reed-Solomon stage above it,
despite older documentation to the contrary.

**What is an MPT-1327 GTC?**
"Go To Channel" — the voice grant. It's an address codeword whose Address
Categorisation subfield is `KindGoToChan`, carrying the assigned channel number in
the low 13 bits of the Function field. `AsGoToChannel` decodes it and the state
machine republishes it as a `trunking.Grant`.

**Do these legacy protocols decode voice in GopherTrunk?**
The trunking follow-along (control-channel decode, grants, retune) works; the voice
side varies. Analog FM voice records fine, but EDACS ProVoice / Aegis digital voice
uses a proprietary AMBE-derived vocoder that isn't decoded — an honest deferral
noted right in the package docs.

## Series navigation

**Part 8 of 12** · ←
[Part 7: TETRA]({{ '/blog/deep-dives/protocol-decoders-07-tetra/' | relative_url }})
· Next →
[Part 9: Conventional, Wideband & the Symbol Scope]({{ '/blog/deep-dives/protocol-decoders-09-conventional-wideband/' | relative_url }})
