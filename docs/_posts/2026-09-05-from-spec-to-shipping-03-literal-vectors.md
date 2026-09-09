---
title: "From Spec to Shipping, Part 3: Literal Vectors, Not Round-Trips"
description: "Why round-trip tests let parser bugs live — the P25 SCCB opcode that read channel B one byte early while its test stayed green — and the fix as method: pin every parser with literal byte vectors cross-checked against an independent decoder, kept as bytes in the test, never generated."
category: deep-dives
keywords: round-trip test trap, literal test vectors, parser regression testing, p25 sccb tsbk 0x39, self-consistent test bug, byte layout off by one, cross-check independent decoder, pinning constants in tests, gophertrunk from spec to shipping
tags: [from-spec-to-shipping, testing, p25, tsbk, go, methodology]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From Spec to Shipping"
series_part: 3
---

*Part 3 of **From Spec to Shipping**, a 14-part series on how a protocol
decoder actually gets written — from standards documents and independent
references to code you can trust on air.
[Part 2]({{ '/blog/deep-dives/from-spec-to-shipping-02-choosing-references/' | relative_url }})
assembled the reference stable and ended on a warning: a reference quoted in
a comment decorates your code, while a reference wired into your tests
disagrees with it. This part is the wiring. It opens with the cleanest
specimen of the series villain this project owns — a P25 parser that read a
field one byte early for as long as it existed, under a green test — and
turns the fix into a rule you can apply to any wire format: literal
vectors, not round-trips.*

> **TL;DR:** GopherTrunk's P25 SCCB parser (TSBK opcode 0x39,
> `internal/radio/p25/phase1/opcodes.go`) read secondary channel B from
> payload bytes 4–5 instead of 5–6 — splicing the service-class byte into
> the channel field and inventing a phantom control channel — and its
> round-trip test **passed the whole time**, because the test's assembler
> encoded the same wrong layout. Encode∘decode = identity proves
> *consistency*, never *correctness*. The fix as method: pin parsers with
> **literal byte vectors** cross-checked against an independent decoder or
> a real capture — like the four on-air LOC_REG_RSP payloads in
> `tsbk_test.go`, or the SmartNet tests that pin sync bits, the stride-19
> interleave permutation and the `0xCC38`/`0x0D5` XOR masks against OP25's
> literals — and keep the vectors as bytes in the test, never generated.

**Key takeaways**

- **A round-trip test has zero external information.** Parse(Assemble(x)) ==
  x holds for *every* self-consistent layout, including wrong ones — the
  assembler and parser are the same hypothesis written twice.
- **Literal vectors import a fact from outside.** A byte string from a
  capture, a spec example, or another decoder's output constrains your
  parser against the world instead of against itself.
- **Reference-literal tests are the only tests that catch constant drift.**
  A sync word, an interleave stride, an XOR mask — if the test computes the
  expected value from the same constant, it verifies nothing.
- **Vectors live in the test as bytes.** The moment a "vector" is generated
  by code under test — or by a sibling that shares its tables — it stops
  being evidence.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The specimen bug | SCCB channel B read at bytes 4–5, not 5–6 | `internal/radio/p25/phase1/opcodes.go` (`ParseSecondaryControlChannelBroadcast`) |
| On-air vectors | four real LOC_REG_RSP payloads pin the layout | `phase1/tsbk_test.go` (`TestParseLocationRegistrationResponse`) |
| Constant pins | sync bits, interleave permutation, XOR masks vs OP25 literals | `internal/radio/motorola/frame_test.go` |
| Opcode pins | RPC ids vs upstream `SoapyRemoteDefs.hpp` literals | `internal/sdr/soapyremote/driver_test.go` (`TestOpenSetAntennaUsesUpstreamOpcode`) |
| Independent re-derivation | scrambler rebuilt from the recurrence, not the LFSR | `internal/radio/framing/scramble_tetra_test.go` (`refScrambleTetraETSI`) |
| The postmortem twin | the trap's greatest hits, told as failures | [From the Issue Tracker Part 20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}) |

## In this post

- **The SCCB story in full** — one byte early, green the whole time.
- **Why round-trips can't catch it** — the geometry of self-consistency.
- **Literal vectors, and where to harvest them** — captures, references,
  spec examples, independent re-derivations.
- **Pinning constants, not just layouts** — sync words, permutations,
  masks, opcodes.
- **The rules** — five habits that transfer to any wire format.

## The SCCB story in full

P25's Secondary Control Channel Broadcast (TSBK opcode 0x39) announces
alternate control-channel frequencies for a site. Its 8-byte payload packs
two channels: RFSS and site IDs, channel A in bytes 2–3, channel A's
service class in byte 4, channel B in bytes 5–6, channel B's service class
in byte 7 — per TIA-102.AABC and SDRTrunk's
`SecondaryControlChannelBroadcast`.

GopherTrunk's parser read channel B from bytes **4–5**. One byte early:
the high byte of its "channel" was actually service class A, and the field
that should have been read as class B was half of a channel number. The
result was a **phantom secondary control channel** — a plausible-looking
but nonexistent frequency fed into the control-channel hunt machinery.
The comment now in the tree tells the rest:

```go
// internal/radio/p25/phase1/opcodes.go (shape)
// The previous working model read channel B from bytes 4-5 — one byte
// early, splicing service class A into the channel field and producing a
// phantom secondary CC; the round-trip test passed because the assembler
// encoded the same wrong layout (the self-consistent-synthetic trap).
func ParseSecondaryControlChannelBroadcast(p [8]byte) SecondaryControlChannelBroadcast {
    cA := binary.BigEndian.Uint16(p[2:4])
    cB := binary.BigEndian.Uint16(p[5:7]) // was p[4:6]
    /* … */
}
```

Note what the bug was *not*: not a logic error, not a misunderstanding of
the protocol, not a weak-signal artifact. A transcription slip — the kind
Part 1 warned lives in exact bit offsets — protected from discovery by the
exact test that appeared to cover it.

## Why round-trips can't catch it

The SCCB test called `Parse(Assemble(x))` and compared the result to `x`.
It passed before the fix and passes after, because assembler and parser
were written by the same person from the same (wrong) reading, and each is
the other's inverse *by construction*. A round-trip test verifies one
proposition: "these two functions agree." It says nothing about whether
either agrees with the air.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Two testing topologies compared. Left: a round-trip loop where an assembler with a wrong layout feeds a parser with the same wrong layout, the comparison passes, and a green check sits on a closed loop labelled zero external information. Right: a literal vector — bytes from a capture or an independent decoder — feeds only the parser, whose output is compared to independently known field values; the wrong parser now fails, marked with an accent cross labelled the outside fact wins.">
  <text x="160" y="20" text-anchor="middle" fill="currentColor" font-size="11" font-weight="bold">round-trip: a closed loop</text>
  <rect x="60" y="40" width="200" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="160" y="59" text-anchor="middle" fill="currentColor" font-size="10">Assemble — wrong layout (B at 4–5)</text>
  <rect x="60" y="110" width="200" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="160" y="129" text-anchor="middle" fill="currentColor" font-size="10">Parse — same wrong layout</text>
  <path d="M 250 70 C 290 80 290 100 250 110" fill="none" stroke="var(--fg-muted)"/>
  <polygon points="255,106 246,113 258,116" fill="var(--fg-muted)"/>
  <path d="M 70 110 C 30 100 30 80 70 70" fill="none" stroke="var(--fg-muted)"/>
  <polygon points="65,74 74,67 62,64" fill="var(--fg-muted)"/>
  <text x="160" y="170" text-anchor="middle" fill="currentColor" font-size="11">✓ passes — forever</text>
  <text x="160" y="190" text-anchor="middle" fill="var(--fg-muted)" font-size="10">zero external information:</text>
  <text x="160" y="204" text-anchor="middle" fill="var(--fg-muted)" font-size="10">both sides share one hypothesis</text>
  <line x1="340" y1="30" x2="340" y2="210" stroke="var(--fg-muted)" stroke-dasharray="3 4"/>
  <text x="520" y="20" text-anchor="middle" fill="currentColor" font-size="11" font-weight="bold">literal vector: one side pinned</text>
  <rect x="410" y="40" width="220" height="30" rx="5" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="520" y="59" text-anchor="middle" fill="var(--accent)" font-size="10">bytes from capture / independent decoder</text>
  <line x1="520" y1="70" x2="520" y2="94" stroke="currentColor"/>
  <polygon points="516,92 520,100 524,92" fill="currentColor"/>
  <rect x="410" y="100" width="220" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="520" y="119" text-anchor="middle" fill="currentColor" font-size="10">Parse — wrong layout</text>
  <line x1="520" y1="130" x2="520" y2="154" stroke="currentColor"/>
  <polygon points="516,152 520,160 524,152" fill="currentColor"/>
  <text x="520" y="176" text-anchor="middle" fill="var(--accent)" font-size="11" font-weight="bold">✗ fails: fields ≠ known values</text>
  <text x="520" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the outside fact wins — the bug</text>
  <text x="520" y="210" text-anchor="middle" fill="var(--fg-muted)" font-size="10">cannot hide on both sides at once</text>
</svg>
<figcaption>A round-trip constrains two functions to each other; a literal vector constrains the parser to the world — only the second can fail on a shared mistake.</figcaption>
</figure>

This is not an isolated species. The same trap, at other layers of this
one codebase: the fabricated SmartNet framing decoded its own synthetic
fixtures perfectly while no real system could feed it
([#1143](https://github.com/MattCheramie/GopherTrunk/issues/1143)); the
SoapyRemote fake server switched on the same opcode constant the client
sent, so a wrong opcode moved both sides together
([From the Issue Tracker Part 13]({{ '/blog/solution-postmortem/from-the-issue-tracker-13-soapyremote-handshake/' | relative_url }}));
a TETRA DMO descramble skip survived because the test's encode side
skipped scrambling under the same condition
([TETRA End to End Part 12]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }})).
The [postmortem edition]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
collects the wreckage; this part is the constructive mirror.

Round-trips still earn their keep — they catch asymmetric typos, shift
errors that don't invert, mask mismatches between the two directions. Keep
them. Just never let one be the *only* test on a layout, because the class
of bug it cannot see is precisely the class that ships.

## Literal vectors, and where to harvest them

A literal vector is a byte string whose correct decode is known from
somewhere *other than the code under test*. Four harvesting grounds, in
rising order of authority:

**Spec worked examples.** Some standards include encoded examples in
annexes. Cheap when they exist; remember Part 1's caveat that informative
annexes can lag normative text.

**An independent decoder's output.** Run OP25, SDRTrunk, Wireshark or
osmo-tetra over a stream and transcribe what they print. This is how the
SCCB-class layouts get their second opinion — SDRTrunk's field offsets are
effectively vectors in prose form.

**An independent re-derivation.** When no external implementation is
available, write a *second* implementation from the spec by a different
route, structured so it cannot share the failure mode. GopherTrunk's
scrambler cross-check is the template: `refScrambleTetraETSI` builds the
sequence directly from the §8.2.5.2 recurrence "precisely so it can not
share a shift-direction bug with the production `ScramblerTetra`," which is
a register-shift LFSR. Same spec, different mechanics — a shared mistake
now requires two independent misreadings to align.

**A real capture's decoded bytes.** The gold standard. When GopherTrunk's
LOC_REG_RSP layout needed confirmation, the pin became four payloads lifted
from a live Motorola P25 site, asserted field by field:

```go
// internal/radio/p25/phase1/tsbk_test.go (shape) — TestParseLocationRegistrationResponse
// All four name the same camped site — RFSS 1, Site 1 — with plausible
// group addresses and 24-bit registering-unit IDs, which is how the
// layout (response, group, RFSS@p3, Site@p4, target) was confirmed.
vectors := []struct {
    payload [8]byte
    group   uint16
    target  uint32
}{
    {[8]byte{0x00, 0x05, 0x79, 0x01, 0x01, 0x02, 0x45, 0x01}, 0x0579, 0x024501},
    {[8]byte{0x00, 0x05, 0x15, 0x01, 0x01, 0x02, 0x13, 0x4A}, 0x0515, 0x02134A},
    /* … two more on-air payloads … */
}
```

The bytes are written out. Not built by an assembler, not loaded from a
fixture generator — *typed into the test*. The expected values carry their
own plausibility argument (same site in all four, sensible unit IDs), so
the vector documents its provenance alongside its content. The adjacent
`TestParseAdjacentSiteStatusBroadcast` does the same with an on-air
ADJ_STS_BCST payload from the Mt Anakie captures — the same site the
[#764](https://github.com/MattCheramie/GopherTrunk/issues/764)
investigation verified against.

## Pinning constants, not just layouts

Layouts are one drift surface; bare constants are the other, and they need
the same treatment. The rebuilt SmartNet decoder
([Part 8]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})'s
case study) ships three reference-literal tests in
`internal/radio/motorola/frame_test.go`, one per constant class:

| Pin | Against | Why a weaker test fails |
|---|---|---|
| sync bits: `0xAC` MSB-first | OP25 `frame_sync_magics.h` literal | any self-consistent sync "works" in a round-trip |
| deinterleave permutation | OP25's mapping, as the explicit sequence {1, 20, 39, 58, 2, 21, 40, 59, …} | encode/decode interleavers invert each other at *any* stride |
| XOR masks `0xCC38` / `0x0D5` | `rx_smartnet.h` `ID_XOR 0x33C7` / `CMD_XOR 0x32A`, complemented | masks applied symmetrically cancel in a loop |

And the pattern is not radio-specific. When the SoapyRemote antenna bug
was fixed, the regression pinned sixteen RPC opcodes against upstream
`SoapyRemoteDefs.hpp` values — with the comment doing the teaching:
"Written as literals on purpose: comparing the constant to itself proves
nothing" (`TestOpenSetAntennaUsesUpstreamOpcode`). An enum copied from a
`.hpp` is exactly as much a wire constant as a sync word;
[Part 9]({{ '/blog/deep-dives/from-spec-to-shipping-09-wire-protocols-without-schemas/' | relative_url }})
runs that story in full.

The test smell to hunt in review: an expected value that mentions the
production constant's *name*. `want := OutboundSyncHex` verifies the
compiler; `want := 0xAC // OP25 frame_sync_magics.h` verifies the world.

## The rules

Distilled, and none of them are about radio:

1. **Every parser gets at least one literal vector** whose expected decode
   comes from outside the codebase — a capture, an independent
   implementation, or a spec example. Round-trips are supplements.
2. **Vectors are bytes in the test file.** Generated input is code under
   test wearing a disguise. If a helper builds it, the helper must not
   share tables, masks or layout constants with the parser.
3. **Wire constants are pinned against upstream literals**, re-typed by
   hand at the pin site, with the source named in a comment.
4. **Record the vector's provenance in the test.** "Four payloads from a
   live Motorola site, all naming the camped site" is what lets a future
   maintainer trust — or correctly distrust — the pin when the layout
   question reopens.
5. **When you fix a layout bug, the failing vector goes in first.** The
   SCCB fix, the SmartNet rebuild and the DMO descramble fix each landed
   with a test that fails against the old code — the failing-first
   discipline [Part 12]({{ '/blog/deep-dives/from-spec-to-shipping-12-failing-first/' | relative_url }})
   makes general, and the habit
   [/learn/testing/]({{ '/learn/testing/' | relative_url }}) teaches from
   the ground up.

## Where this goes next

Literal vectors pin fields and constants — discrete facts. But a vocoder
or a channel-coding chain is thousands of arithmetic operations whose
correctness only shows in the aggregate, and for those the vector becomes
a *stream* and the assertion becomes **bit-identical output against a
reference implementation**.
[Part 4]({{ '/blog/deep-dives/from-spec-to-shipping-04-conformance-harness/' | relative_url }})
builds that harness around the ETSI ACELP codec — same bitstream into both
decoders, zero sample mismatches allowed — and meets the LP64 gotcha that
makes the reference itself lie on a modern machine.

## FAQ

**Are round-trip tests worthless, then?**
No — they cheaply catch asymmetries: a shift that doesn't invert, a mask
applied on one side only, a field dropped in reassembly. The rule is about
sufficiency: a round-trip must never be the only test on a wire layout,
because the shared-hypothesis bug class is invisible to it by construction.

**Where do I get literal vectors for an encrypted or rare protocol?**
Work down the authority ladder: an independent decoder's output over any
sample you can find, a second in-house implementation built by a different
route (the `refScrambleTetraETSI` pattern), or spec worked examples. Even
one confirmed vector beats a thousand round-trips — it is the only test in
the file carrying outside information.

**How many vectors does a parser need?**
Enough to make each field's offset load-bearing — usually two to four
well-chosen ones. GopherTrunk pinned LOC_REG_RSP with four on-air payloads
because they jointly confirm response, group, RFSS, site and target
occupy distinct, correct bytes; one all-zeros vector would pin almost
nothing.

**Should vectors live in testdata files instead of source?**
Short vectors belong inline — visible in review, diffed with the code,
impossible to regenerate accidentally. Bulk streams (a captured control
channel, an IQ fixture) belong in `testdata/` with a checksum or a
documented origin; the danger is not the file, it's any *pipeline* that
can silently rebuild it from the code under test.

**What's the fastest way to audit an existing codebase for this trap?**
Grep the tests for calls to your own assemblers and encoders. Every parser
whose only inputs are produced by its own inverse is unpinned — list them,
then harvest one literal vector each, worst-consequences first. The SCCB
bug lived precisely in that unpinned set, and only an independent
cross-check dislodged it.

## Series navigation

**Part 3 of 14** · ←
[Part 2: Choosing Reference Implementations You Can Trust]({{ '/blog/deep-dives/from-spec-to-shipping-02-choosing-references/' | relative_url }})
· Next →
[Part 4: The Conformance Harness — Bit-Identical or Bust]({{ '/blog/deep-dives/from-spec-to-shipping-04-conformance-harness/' | relative_url }})
