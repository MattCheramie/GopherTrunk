---
title: "From Spec to Shipping, Part 1: How to Read a Radio Standard"
description: How ETSI EN and TIA-102 standards families are organized, the order a decoder author should read them — physical layer up to PDU layouts — and the constants to extract on day one: sync words, CRC definitions, and the exact bit offsets everything else leans on.
category: deep-dives
keywords: how to read a radio standard, etsi en 300 392-2, tia-102 document family, p25 specification structure, tetra air interface specification, crc polynomial from spec, bit order transmission order, writing a protocol decoder from a spec, gophertrunk from spec to shipping
tags: [from-spec-to-shipping, standards, etsi, tia-102, protocols, methodology]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From Spec to Shipping"
series_part: 1
---

*Part 1 of **From Spec to Shipping**, a 14-part series on how a protocol
decoder actually gets written — from standards documents and independent
references to code you can trust on air. The
[From the Issue Tracker]({{ '/blog/solution-postmortem/from-the-issue-tracker-01-first-p25-lock/' | relative_url }})
series told these lessons as postmortems: fabricated framing, placeholder
constants, green tests over a decoder no real system could feed. This series
teaches the same discipline forward, as method — with one recurring villain,
**the test that passes because both sides share the same assumption**, and
one recurring hero, the independent reference. This opener starts where
every decoder starts: the PDF itself — how to read it, and what to pull out
of it first.*

> **TL;DR:** A radio standard is a **family of documents**, not one PDF —
> TETRA's air interface is ETSI EN 300 392-2, its voice codec EN 300 395-2,
> its direct mode EN 300 396-2/-3; P25 spreads across the TIA-102 series
> (BAAA physical layer, AABC/AABF trunking messages, BABA vocoder, BBAB/BBAC
> Phase 2). Read in **decode order** — physical → burst geometry → channel
> coding → CRCs → PDU layouts — and extract the load-bearing constants
> first: sync patterns, CRC definitions, exact bit offsets. Every such
> constant in GopherTrunk cites its section (grep `§8.2.5` in
> `internal/radio/framing/scramble_tetra.go`) — because
> [placeholder constants]({{ '/blog/solution-postmortem/from-the-issue-tracker-17-placeholder-constants/' | relative_url }})
> are how GopherTrunk shipped a SmartNet decoder that could never lock
> ([#1143](https://github.com/MattCheramie/GopherTrunk/issues/1143)).

**Key takeaways**

- **The spec is a family with a division of labor.** One part owns the
  modulation, another the burst geometry, another the codec. Knowing which
  part owns your question is half of finding the answer.
- **Read in decode order.** A decoder consumes physical symbols, then
  bursts, then channel coding, then CRC-gated payloads, then PDUs. Reading
  in that order makes every finished chapter testable against the last.
- **Constants first, prose second.** Sync words, polynomial taps, interleave
  strides and bit offsets are the facts everything else leans on. Extract
  them into named, cited constants on day one; a wrong constant silently
  poisons everything downstream.
- **The spec alone is not enough.** Standards ship without test vectors,
  define fields without naming their values, and leave bit order ambiguous.
  Each gap is a place where your reading and your test can share the same
  mistake — which is why Part 2 is about independent references.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| TETRA air interface | bursts, channel coding, scrambling, MAC/CMCE PDUs | ETSI EN 300 392-2; cited across `internal/radio/tetra/` |
| TETRA voice codec | ACELP full-rate speech codec + reference C code | ETSI EN 300 395-2; `internal/voice/acelp/` |
| TETRA direct mode | radio layer (396-2), MS-MS protocol (396-3) | ETSI EN 300 396-2/-3; `internal/radio/tetra/dmo.go` |
| P25 physical + FDMA | C4FM, NID, frame structure, data units | TIA-102.BAAA; `internal/radio/p25/phase1/` |
| P25 trunking messages | TSBK/MBT opcodes and field layouts | TIA-102.AABC / TIA-102.AABF; `phase1/opcodes.go` |
| P25 vocoder | IMBE frame format, FEC, synthesis | TIA-102.BABA; `internal/voice/imbe/` |
| Scrambling seed example | LFSR polynomial + seed packing, section-cited | `internal/radio/framing/scramble_tetra.go` (§8.2.5) |

## In this post

- **A standard is a family, not a document** — how ETSI and TIA carve up a
  protocol, and which parts a decoder author needs.
- **Read in decode order, not page order** — physical → bursts → coding →
  CRCs → PDUs, so each chapter is testable.
- **Extract the constants first** — sync patterns, CRC definitions, exact
  bit offsets.
- **The traps that break parsers silently** — bit order, transmission
  order, fields defined elsewhere, "reserved" that isn't.
- **What the spec cannot tell you** — the gaps that make Part 2 necessary.

## A standard is a family, not a document

Nobody publishes "the TETRA spec." ETSI publishes a *series*: EN 300 392 is
"Voice plus Data," and its Part 2 — the air interface — is the ~1000-page
volume a decoder author lives in. The speech codec is a different series
(EN 300 395-2, which ships reference C code — the anchor for
[Part 4]({{ '/blog/deep-dives/from-spec-to-shipping-04-conformance-harness/' | relative_url }})).
Direct mode is yet another: EN 300 396-2 is the DMO radio layer, and the
MS-to-MS call-control protocol riding on it is EN 300 396-3 — a separation
GopherTrunk's `internal/radio/tetra/dmo.go` doc comment spells out, because
burst slicing and call semantics arrive from two different books.

TIA slices P25 just as finely. TIA-102.BAAA is the FDMA common air
interface — modulation, NID, frame structure. The trunking control-channel
*message layouts* GopherTrunk's `opcodes.go` decodes cite TIA-102.AABC and
TIA-102.AABF throughout; the IMBE vocoder is TIA-102.BABA; Phase 2's
two-slot TDMA splits across TIA-102.BBAB (physical) and TIA-102.BBAC (MAC);
encryption is TIA-102.AACE. Revisions ride as suffixes —
`internal/dsp/demod/c4fm_modulator.go` cites TIA-102.BAAA-**A** §6.1.1, and
a revision letter can silently change a constant you extracted from the
base document.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Two standards families side by side: the ETSI TETRA documents and the TIA-102 P25 documents, with the decoder-relevant parts highlighted and the conformance-testing, uplink-only and encryption parts muted; a note beneath says each highlighted document maps to one GopherTrunk package.">
  <text x="150" y="22" text-anchor="middle" fill="currentColor" font-size="11" font-weight="bold">ETSI TETRA family</text>
  <rect x="30" y="34" width="240" height="30" rx="5" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="150" y="53" text-anchor="middle" fill="var(--accent)" font-size="10">EN 300 392-2 — V+D air interface</text>
  <rect x="30" y="70" width="240" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="150" y="89" text-anchor="middle" fill="var(--accent)" font-size="10">EN 300 395-2 — ACELP codec + ref C</text>
  <rect x="30" y="106" width="240" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="150" y="125" text-anchor="middle" fill="var(--accent)" font-size="10">EN 300 396-2 / -3 — DMO radio / protocol</text>
  <rect x="30" y="142" width="240" height="26" rx="5" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="150" y="159" text-anchor="middle" fill="var(--fg-muted)" font-size="10">conformance testing, uplink-only parts</text>
  <text x="530" y="22" text-anchor="middle" fill="currentColor" font-size="11" font-weight="bold">TIA-102 P25 family</text>
  <rect x="410" y="34" width="240" height="30" rx="5" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="530" y="53" text-anchor="middle" fill="var(--accent)" font-size="10">BAAA — FDMA physical layer / CAI</text>
  <rect x="410" y="70" width="240" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="530" y="89" text-anchor="middle" fill="var(--accent)" font-size="10">AABC / AABF — trunking messages</text>
  <rect x="410" y="106" width="240" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="530" y="125" text-anchor="middle" fill="var(--accent)" font-size="10">BABA — IMBE · BBAB/BBAC — Phase 2</text>
  <rect x="410" y="142" width="240" height="26" rx="5" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="530" y="159" text-anchor="middle" fill="var(--fg-muted)" font-size="10">AACE — encryption (metadata only)</text>
  <line x1="150" y1="176" x2="150" y2="200" stroke="var(--fg-muted)"/>
  <line x1="530" y1="176" x2="530" y2="200" stroke="var(--fg-muted)"/>
  <line x1="150" y1="200" x2="530" y2="200" stroke="var(--fg-muted)"/>
  <line x1="340" y1="200" x2="340" y2="216" stroke="var(--fg-muted)"/>
  <polygon points="336,214 340,222 344,214" fill="var(--fg-muted)"/>
  <text x="340" y="238" text-anchor="middle" fill="currentColor" font-size="10">each highlighted document maps to a GopherTrunk package citing it by section</text>
</svg>
<figcaption>The decoder-relevant subset of two standards families: a handful of highlighted documents carry every constant a receive-only decoder needs.</figcaption>
</figure>

Two consequences. First, **get the right parts**: a receive-only scanner
needs the air interface, the codec, and the trunking messages — not the
conformance-testing or uplink-heavy parts. Second, **normative versus
informative matters**: worked-example annexes are informative and can lag
the normative clauses; when they disagree, the normative text wins — and
the disagreement is itself a flag to find an independent reference.

## Read in decode order, not page order

Standards are organized for transmitter implementers as much as receivers,
and page order rarely matches the order a decoder consumes the signal. The
reading order that works is the decode chain itself:

| Stage | What to read for | TETRA example (EN 300 392-2) |
|---|---|---|
| 1. Physical | modulation, symbol rate, pulse shaping | π/4-DQPSK, 18000 sym/s ([TETRA End to End Part 1]({{ '/blog/deep-dives/tetra-end-to-end-01-pi4-dqpsk-carrier/' | relative_url }})) |
| 2. Burst geometry | burst types, training sequences, slot grid | §9.4.4 burst layouts |
| 3. Channel coding | codes, interleaving, scrambling | RCPC mother code, §8.2.5 scrambling |
| 4. CRCs | polynomial, init, span, complement | the CRC each logical channel gates on |
| 5. PDU layouts | field-by-field bit offsets | MAC (§21) and CMCE (§14) PDU tables |

The reason to hold this order is testability: each layer's output is the
next layer's input, so a finished layer can be pinned before the next one
exists. Finish the burst slicer and you can count training-sequence hits on
a capture before you've written a single CRC; finish the channel coding and
CRC-valid counts become your universal yield metric — the number the
[Weak-Signal Engineering series]({{ '/blog/series/weak-signal-engineering/' | relative_url }})
treats as the only verdict that cannot flatter you. Skip ahead to PDU
parsing — the tempting part, where talkgroups live — and nothing you write
is checkable until every layer below it also works. That is exactly how a
stack of untested assumptions forms.

## Extract the constants first

Within each layer, the first pass is not to understand everything — it's a
harvest. Three kinds of facts get extracted into named, cited constants
before any logic is written:

**Sync and training patterns.** The only thing a decoder can find in an
unsynchronized dibit stream — and one wrong bit means zero decodes forever,
with no error message.

**CRC and scrambling definitions.** Polynomial, initialization, bit span,
and any final complement — all four, because a CRC that's "almost right"
fails exactly like a weak signal does. GopherTrunk's TETRA scrambler is the
template for how these land in code:

```go
// internal/radio/framing/scramble_tetra.go (shape)
// Connection polynomial (§8.2.5.2 eq. 8.40):
//
//	c(x) = 1 + X + X^2 + X^4 + X^5 + X^7 + X^8 + X^10 + X^11
//	     + X^12 + X^16 + X^22 + X^23 + X^26 + X^32
//
// The recurrence is p(k) = sum_{i=1..32} c_i * p(k-i) (§8.2.5.2
// eq. 8.41) with initialisation p(-31) = p(-30) = 1, p(-29) =
// e(30), ..., p(0) = e(1) per §8.2.5.2 eq. 8.42.
const scrambleTetraTapMask uint32 = 0x82608EDB
```

Equation numbers, section numbers, the exact seed packing — all in the
comment, so the constant is *auditable* years later. (That seed rule,
eq. 8.42, is also why "colour code 0" is not a no-op — the LFSR seeds
non-zero even with all colour bits zero, a fact that cost the
[DMO investigation]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }})
weeks when one decode path skipped descrambling at colour 0.)

**Exact bit offsets.** PDU field tables get transcribed as offsets with the
spec's own field names. The corollary is the hard lesson of
[From the Issue Tracker Part 17]({{ '/blog/solution-postmortem/from-the-issue-tracker-17-placeholder-constants/' | relative_url }}):
a constant you *couldn't* verify and wrote down anyway is worse than a
`panic("unimplemented")`, because it fails silently. GopherTrunk's original
SmartNet decoder was built on a 24-bit sync word and a BCH code matching
**no real reference** — every synthetic test passed, no real system ever
locked, and the framing had to be rebuilt from proven decoders
([#1143](https://github.com/MattCheramie/GopherTrunk/issues/1143), Part 8's
case study).

## The traps that break parsers silently

Four recurring traps, each drawn from a bug this project shipped or
narrowly avoided:

**Bit order and endianness.** A spec says "e(1) through e(30)" and you must
determine whether e(1) is the MSB or LSB of the packed value. The vicious
part: get it wrong and your encoder and decoder still agree. GopherTrunk's
independent scrambler cross-check (`refScrambleTetraETSI` in
`scramble_tetra_test.go`) documents it exactly: the wrong endianness "still
round-trips and still passes BSCH," but matches no real base station, so no
BNCH/SCH burst ever decodes. The series villain in its purest form — a
self-consistent mistake, invisible to any test that doesn't bring in an
outside fact.

**Transmission order versus logical order.** Interleavers, bit-reversal
conventions, and MSB-first-on-the-wire rules mean the bytes in the PDU table
are not the bits in the air; the layers must be undone in reverse order, and
an off-by-one in an interleave stride is wrong *everywhere* at once — see
[From the Issue Tracker Part 8]({{ '/blog/solution-postmortem/from-the-issue-tracker-08-nineteen-dibits/' | relative_url }})
for what nineteen dibits of slip looks like from the outside.

**Fields defined by reference.** TETRA's D-SETUP carries a "basic service
information" element whose sub-field layout is effectively defined through
the ASN.1 description — GopherTrunk pinned the decomposition against
Wireshark's generated dissector because the prose alone underdetermines it
(`internal/radio/tetra/cmce_parse.go`). When a field's definition lives in
another document — or another *format* — the spec you're holding tells you
it exists, not what it means.

**"Reserved" and vendor space that isn't.** P25's opcode space has
manufacturer-specific regions, and rendering a vendor TSBK through the
standard opcode name map actively *mislabels* it — MFID 0x90 opcode 0x00
reads back as `GRP_V_CH_GRANT` if you let it, so GopherTrunk refuses to
name vendor or undecoded opcodes through the standard map. Treat every
"reserved" region as defined somewhere you haven't looked yet.

## What the spec cannot tell you

Read perfectly, extract diligently, and you still hold a document with
structural gaps. Most standards ship **no test vectors** for the receive
chain — EN 300 395-2's reference codec is a happy exception, and Part 4
builds a whole conformance harness on it. Fields get defined but their
values left unnamed: the D-SETUP communication-type bits are decoded by
three independent open decoders, yet *none names the enum values* — so
GopherTrunk carries the spec-derived mapping as constants deliberately
**not** wired into call classification until a capture confirms them. And
which of several permitted broadcast forms a real network uses is an
empirical fact the spec never settles — some P25 systems announce their
neighbours *only* in AMBT form, which GopherTrunk once dropped as
"non-control" spam.

Every gap has the same resolution: an independent fact from outside your
own head — another implementation, a reference codec, a capture.

### How that principle shaped the Go code

- **Every constant carries its citation.** Grepping `§8.2.5` or
  `TIA-102.AABC` finds the constant *and* its provenance in one step; an
  uncited constant is a review flag.
- **Spec-derived but unconfirmed facts are quarantined.** The comms-type
  constants exist (`CommsPointToPoint` and friends in `cmce_parse.go`) but
  are logged for empirical confirmation, not wired into behaviour.
- **Layer boundaries in the spec are package boundaries in the code.**
  Channel coding in `internal/radio/framing`, protocol PDUs in
  `internal/radio/tetra` and `internal/radio/p25`, vocoders in
  `internal/voice/` — each layer pinnable at exactly the seams the reading
  order established.
- **Ambiguity gets a second implementation, not a decision.** Where the text
  underdetermines bit order, the tree carries an independent re-derivation
  built to disagree with the production code — the pattern
  [Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
  develops in full.

## Where this goes next

A spec plus your own reading of it is one source — and one source can be
self-consistently wrong.
[Part 2]({{ '/blog/deep-dives/from-spec-to-shipping-02-choosing-references/' | relative_url }})
assembles the reference stable — OP25, trunk-recorder, SDRTrunk, osmo-tetra,
the ETSI reference codec, mbelib — what each is authoritative for, and the
rules for trusting them, starting with the one that matters most: *proven on
air beats popular*.

## FAQ

**Do I need to buy the standards to write a decoder?**
ETSI standards are free downloads from etsi.org, including the EN 300 392
and EN 300 395 families. The TIA-102 series is sold commercially; much
layout knowledge is also recoverable from open implementations — Part 2's
subject, with clean-room rules in Part 5.

**What should I read first in a spec I've never opened?**
The physical layer chapter and the burst/frame structure tables — then stop
and find the sync patterns and CRC definitions wherever they live. Those let
you build the first testable thing (a sync correlator counting hits on a
capture) before you understand the other 900 pages.

**Why extract constants before writing any parsing logic?**
Constants fail silently and logic fails loudly. A wrong branch produces
garbage you notice; a wrong sync word or CRC polynomial produces *zero
decodes*, indistinguishable from a weak signal. Named, cited constants can
be pinned by literal-vector tests (Part 3) independently of the logic
around them.

**How do I know which document in a family defines a given field?**
Follow the layer: physical facts in the air-interface part, message layouts
in the signalling part, codec bits in the codec series. GopherTrunk cites at
every use, so the codebase doubles as an index — the fastest answer to
"where is SCCB defined" is the comment on
`ParseSecondaryControlChannelBroadcast`.

**Is the newest revision of a standard always the right one to implement?**
Implement what deployed systems transmit, which often trails the paper.
When a revision and a fielded system disagree, the capture referees — a
theme this series returns to in Parts 6 and 10.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: Choosing Reference Implementations You Can Trust]({{ '/blog/deep-dives/from-spec-to-shipping-02-choosing-references/' | relative_url }})
