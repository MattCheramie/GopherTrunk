---
title: "From Spec to Shipping, Part 2: Choosing Reference Implementations You Can Trust"
description: The reference stable a protocol decoder is built against — OP25, trunk-recorder, SDRTrunk, osmo-tetra, the ETSI reference codec, mbelib — what each is authoritative for, why proven-on-air beats popular, and how to know what a reference does not decide.
category: deep-dives
keywords: op25 reference implementation, trunk-recorder smartnet parser, sdrtrunk field layouts, osmo-tetra scrambler, etsi reference codec, mbelib vocoder ground truth, validating a protocol decoder, independent decoder cross-check, gophertrunk from spec to shipping
tags: [from-spec-to-shipping, op25, sdrtrunk, osmo-tetra, references, methodology]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From Spec to Shipping"
series_part: 2
---

*Part 2 of **From Spec to Shipping**, a 14-part series on how a protocol
decoder actually gets written — from standards documents and independent
references to code you can trust on air.
[Part 1]({{ '/blog/deep-dives/from-spec-to-shipping-01-reading-a-radio-standard/' | relative_url }})
read the spec and ended on its gaps: no test vectors, unnamed enum values,
ambiguous bit order. Every gap gets filled the same way — with an
independent fact from outside your own head. This part is about where those
facts come from: the open implementations, reference codecs and dissectors
GopherTrunk validates against, what each one is authoritative for, and the
rules that separate a reference you can trust from one that quietly hands
you its own bugs.*

> **TL;DR:** GopherTrunk's reference stable, each trusted for a specific
> layer: **OP25** for P25 PDU/CRC spans (`process_PDU`) and the real
> SmartNet framing (`rx_smartnet.cc`), **trunk-recorder** for SmartNet band
> plans (`SmartnetParser`), **SDRTrunk** for per-field bit offsets (the
> AMBTC classes behind `internal/radio/p25/phase1/mbt.go`), **osmo-tetra /
> osmo-tetra-dmo** for scrambler seeds (`tetra_scramb_get_init`) and SYNC
> PDU offsets, the **ETSI EN 300 395-2 reference C codec** for bit-exact
> ACELP, **mbelib/DSD-FME** for vocoder ground truth, and
> **Wireshark/tetra-kit/sq5bpf** for CMCE PDU layouts. The rules: prefer
> implementations *proven on air*; two independent references that agree
> byte-for-byte beat one popular one; and know what a reference does NOT
> decide — three decoders confirmed TETRA's comms-type bits and none names
> the enum values.

**Key takeaways**

- **A reference earns trust by decoding real air, not by stars on GitHub.**
  OP25 and trunk-recorder run daily against live networks; that operational
  history is the evidence a spec reading can never provide.
- **References are scoped, not global.** Each one is authoritative for the
  layer it demonstrably gets right on air — SDRTrunk for field offsets,
  osmo-tetra for scrambler seeds — and merely suggestive everywhere else.
- **Two independent agreements beat one authority.** GopherTrunk's MBT
  decoder pins layouts against OP25 *and* SDRTrunk; where implementations
  share ancestry, their agreement counts once.
- **Know what the reference does not decide.** A field three decoders parse
  but none interprets is still an open question — writing down the boundary
  of a reference's authority is as valuable as using it.

## Cheat sheet

| Reference | Authoritative for | Where GopherTrunk cites it |
|---|---|---|
| OP25 | P25 MBT/PDU CRC spans; SmartNet framing | `phase1/mbt.go` (`process_PDU`), `internal/radio/motorola/frame.go` (`rx_smartnet.cc`) |
| trunk-recorder | SmartNet band plans, OSW semantics | `motorola/bandplan.go` (`SmartnetParser::get_freq`) |
| SDRTrunk | P25 per-field bit offsets, Phase 2 dibit map | `phase1/mbt.go` (AMBTC classes), `demod/piover4_dqpsk_modulator.go` (`Dibit.java`) |
| osmo-tetra / osmo-tetra-dmo | scrambler seeds, SYNC PDU offsets, MAC PDUs | `tetra/dmo_decode.go` (`tetra_scramb_get_init`), `tetra/mac.go` |
| ETSI EN 300 395-2 ref codec | bit-exact ACELP decode | `internal/voice/acelp/etsi_reference_test.go` |
| mbelib / DSD-FME | IMBE/AMBE synthesis ground truth | `internal/voice/imbe/params.go`; [vocoders guide]({{ '/vocoders.html' | relative_url }}) |
| Wireshark / tetra-kit / sq5bpf | TETRA CMCE (D-SETUP) layouts | `tetra/cmce_parse.go`, `tetra/llc.go`, `tetra/freq.go` |

## In this post

- **What makes a reference trustworthy** — proven on air, independent
  lineage, inspectable output.
- **The stable, layer by layer** — who decides what, from framing to
  vocoder.
- **Two agreeing references beat one popular one** — the MBT and SmartNet
  cross-checks.
- **Knowing what a reference does not decide** — the comms-type lesson.
- **When your only reference is wrong** — inherited bugs, and the capture
  as the court of appeal.

## What makes a reference trustworthy

The question to ask of any candidate reference is not "is it maintained?"
or "is it popular?" but **"has this exact code decoded real transmissions,
recently, for people who would notice if it stopped?"** That is the property
a spec reading cannot have and a synthetic test cannot manufacture — the
running villain of this series is the test that passes because both sides
share an assumption, and an on-air implementation is the one artifact that
provably doesn't share *your* assumptions.

GopherTrunk learned this by counterexample. The original SmartNet decoder
was written from prose descriptions and shipped with green tests; it matched
no real reference and never locked on air
([#1143](https://github.com/MattCheramie/GopherTrunk/issues/1143)). The
rebuild inverted the method: every constant and transform in
`internal/radio/motorola/frame.go` is ported from OP25's `rx_smartnet.cc`
and cross-checked against trunk-recorder's `SmartnetParser` — two codebases
that decode live SmartNet systems daily. The package doc comment says so
explicitly, and says *why*: these are "implementations proven against live
SmartNet/SmartZone systems — NOT prose specs."

Three secondary properties matter when candidates tie:

| Property | Why it matters | What it looks like |
|---|---|---|
| Independent lineage | shared ancestry means agreement counts once | check headers/credits before counting votes |
| Inspectable intermediates | you can pin *layers*, not just outcomes | prints deinterleaved bits, CRC inputs, parameters |
| A community that files bugs | years of operator scrutiny = thousands of antennas | active trackers on OP25, SDRTrunk |

The second row is the quiet one:
[Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})'s
whole method — harvesting literal vectors from a reference's output —
depends on references that show their work, not just their verdicts.

## The stable, layer by layer

Here is the actual matrix — which reference decides which layer, and the
GopherTrunk code that leans on it:

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="A matrix of reference implementations against protocol layers. Rows are framing and sync, channel coding and CRC, PDU field layouts, and vocoder. Columns are OP25, trunk-recorder, SDRTrunk, osmo-tetra, the ETSI reference codec, and mbelib. Filled accent dots mark where each reference is authoritative and proven on air; muted rings mark useful but secondary coverage.">
  <text x="14" y="42" fill="currentColor" font-size="10">framing / sync</text>
  <text x="14" y="82" fill="currentColor" font-size="10">coding / CRC</text>
  <text x="14" y="122" fill="currentColor" font-size="10">PDU layouts</text>
  <text x="14" y="162" fill="currentColor" font-size="10">vocoder</text>
  <text x="160" y="16" text-anchor="middle" fill="currentColor" font-size="10">OP25</text>
  <text x="250" y="16" text-anchor="middle" fill="currentColor" font-size="10">trunk-rec.</text>
  <text x="340" y="16" text-anchor="middle" fill="currentColor" font-size="10">SDRTrunk</text>
  <text x="430" y="16" text-anchor="middle" fill="currentColor" font-size="10">osmo-tetra</text>
  <text x="520" y="16" text-anchor="middle" fill="currentColor" font-size="10">ETSI codec</text>
  <text x="610" y="16" text-anchor="middle" fill="currentColor" font-size="10">mbelib</text>
  <circle cx="160" cy="38" r="7" fill="var(--accent)"/>
  <circle cx="250" cy="38" r="7" fill="none" stroke="var(--fg-muted)"/>
  <circle cx="430" cy="38" r="7" fill="var(--accent)"/>
  <circle cx="160" cy="78" r="7" fill="var(--accent)"/>
  <circle cx="430" cy="78" r="7" fill="var(--accent)"/>
  <circle cx="520" cy="78" r="7" fill="none" stroke="var(--fg-muted)"/>
  <circle cx="250" cy="118" r="7" fill="var(--accent)"/>
  <circle cx="340" cy="118" r="7" fill="var(--accent)"/>
  <circle cx="430" cy="118" r="7" fill="none" stroke="var(--fg-muted)"/>
  <circle cx="160" cy="118" r="7" fill="none" stroke="var(--fg-muted)"/>
  <circle cx="520" cy="158" r="7" fill="var(--accent)"/>
  <circle cx="610" cy="158" r="7" fill="var(--accent)"/>
  <circle cx="60" cy="200" r="7" fill="var(--accent)"/>
  <text x="76" y="204" fill="currentColor" font-size="10">authoritative, proven on air</text>
  <circle cx="290" cy="200" r="7" fill="none" stroke="var(--fg-muted)"/>
  <text x="306" y="204" fill="var(--fg-muted)" font-size="10">useful cross-check, secondary</text>
  <text x="340" y="228" text-anchor="middle" fill="var(--fg-muted)" font-size="10">no single reference covers every layer — the stable is chosen per layer, per protocol</text>
</svg>
<figcaption>The reference matrix: each layer of each protocol gets its own authoritative reference, and no single project covers the whole decode chain.</figcaption>
</figure>

**Framing and channel coding: OP25.** For P25's Multi-Block Trunking,
OP25's `p25p1_fdma::process_PDU` is GopherTrunk's validation reference for
both CRC algorithms and the header extraction — when an operator's log
showed "MBT data CRC failed" lines whose identity fields matched decoded
broadcasts, the CRC span was re-verified byte-for-byte against `process_PDU`
before anyone was allowed to chase a parser bug. For SmartNet, `rx_smartnet`
supplied the whole framing: the 8-bit `0xAC` sync, the stride-19
deinterleave, the convolutional parity rule, the XOR masks.

**Field layouts: SDRTrunk.** OP25 proves spans; SDRTrunk's message classes
document *per-field bit offsets* with unusual clarity. GopherTrunk's MBT
header comment names both:

```go
// internal/radio/p25/phase1/mbt.go (shape) — package comment
// Layouts are cross-checked against two independent decoders — OP25's
// p25p1_fdma::process_PDU (header/format/SAP/opcode extraction, both
// CRC algorithms) and SDRTrunk's PDUHeader / AMBTCHeader / AMBTC*
// message classes (per-field bit offsets) — so this is not a working
// model.
```

"Not a working model" is the point: a layout confirmed by two independent
implementations stops being a hypothesis.

**Scramblers and sync PDUs: osmo-tetra.** LFSR seed packing is exactly the
kind of fact a spec underdetermines
([Part 1]({{ '/blog/deep-dives/from-spec-to-shipping-01-reading-a-radio-standard/' | relative_url }})'s
endianness trap), and osmo-tetra's `tetra_scramb_get_init` is the
executable answer. When the DMO colour-code saga reached the MNI question,
GopherTrunk's `ParseSyncPDU` offsets and extended-colour seed were verified
byte-for-byte against osmo-tetra-dmo — and cross-checked against a *second*
independent DMO receiver — before the fix landed
([TETRA End to End Part 12]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }})).

**Vocoders: the reference codec and mbelib.** EN 300 395-2 ships reference
C code, which upgrades validation from "plausible audio" to **bit-identical
PCM** — [Part 4]({{ '/blog/deep-dives/from-spec-to-shipping-04-conformance-harness/' | relative_url }})
is entirely about that harness. On the MBE side there is no reference
source, so mbelib (via DSD-FME) serves as behavioral ground truth: when an
operator reported DMR audio "sounds awful," the diagnosis came from decoding
the *same* `.amb` frames through both stacks and comparing octave-band
energies — mbelib's output localized the deficit to one quantization table
family in GopherTrunk's 3600×2450 path, turning an opinion about audio into
a measured, file-pinned defect.

**Dissectors: Wireshark and friends.** For TETRA's CMCE PDUs, GopherTrunk
pinned the D-SETUP basic-service decomposition against Wireshark's generated
dissector and confirmed the LLC handling against sq5bpf's telive lineage and
tetra-kit (`internal/radio/tetra/llc.go` cites both). A dissector is a weak
reference for *behavior* — it never has to act on what it parses — but a
strong one for *layout*, because it is generated from the ASN.1.

## Two agreeing references beat one popular one

The strongest form of evidence this method produces is **independent
byte-for-byte agreement**. One reference can be wrong; two references with
separate lineages that agree on every field of a layout are wrong together
only if they share an ancestor — which is why lineage is worth a check
before counting the votes. OP25's `rx_smartnet` itself descends from
gr-smartnet (its header credits the line), so agreement inside that family
counts as one vote; GopherTrunk leans on trunk-recorder where it
contributes independently — the band-plan arithmetic
(`SmartnetParser::get_freq`), which its users exercise against live
800 MHz systems every day.

The TETRA DMO seed work shows the full pattern: the extended-colour formula
was confirmed against osmo-tetra-dmo *and* an independent DMO receiver
project, and GopherTrunk's regression for the MNI fix scrambles its test
vector with the osmo formula — the *independent* derivation — so the test
cannot inherit GopherTrunk's own packing assumptions. A reference is most
valuable when it is wired into your tests as the adversary, not quoted in a
comment.

## Knowing what a reference does not decide

The subtlest skill is reading a reference's *silence*. TETRA's D-SETUP
carries a 2-bit communication-type field. Three independent decoders —
sq5bpf's, tetra-kit, Wireshark — all parse it; GopherTrunk used them to
confirm the bit positions. But **none of the three names the enum values**,
and none classifies calls from it — they defer to downstream tooling keyed
on other evidence. So the layout is reference-confirmed while the meaning
is only spec-derived, and `cmce_parse.go` marks the boundary in code: the
`CommsPointToPoint` constants exist, the raw value is logged on every
D-SETUP for empirical confirmation, and classification refuses to use it
until an operator's known individual-vs-group capture pins the mapping.

That is the general rule: a reference's authority extends exactly as far as
behavior it demonstrably exercises on air. A field a reference parses but
never acts on is *layout-confirmed, semantics-open* — and conflating those
two levels is how spec-derived guesses sneak into shipping behavior wearing
a reference's credibility.

## When your only reference is wrong

Sometimes the stable fails you. osmo-tetra-dmo — the best available DMO
reference — carries TMO-copied burst-geometry offsets (−115/+19 relative to
the training sequence) that measurably underperform on real DMO air;
GopherTrunk's −108/+11 (`dmDNB*Start` in `internal/radio/tetra/dmo.go`) won
not by argument from authority but by a sharp CRC-yield optimum on an
operator's capture. A reference's bug becomes your bug the moment it is
your *only* input — which is why every reference-derived layout still faces
the capture eventually
([Part 6]({{ '/blog/deep-dives/from-spec-to-shipping-06-when-references-disagree/' | relative_url }})
tells that arbitration story in full, and
[Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})
makes the gate explicit).

The failure mode to fear most is inheriting a reference's *assumption*
rather than its code: port a decoder's parsing and its test fixtures
together and you have recreated the self-consistent trap at one remove.
The antidote is the next part's subject — literal vectors harvested from a
reference's *output*, pinned in your tests as bytes.

### How that principle shaped the Go code

- **The citation names the function, not just the project.**
  `motorola/bandplan.go` cites `SmartnetParser::get_freq`;
  `dmo_decode.go` cites `tetra_scramb_get_init` — auditable to the line.
- **Two-reference layouts are labelled as such.** `mbt.go`'s "cross-checked
  against two independent decoders … not a working model" tells the next
  reader how much to trust the file, and what evidence re-opening it
  requires.
- **Reference-derived vs spec-derived is an explicit split.** Compare the
  MBT comment with `cmce_parse.go`'s "SPEC-DERIVED, not capture-confirmed"
  on `CommsType` — the tree distinguishes its evidence classes.
- **References live in tests, not only comments.** The MNI regression
  scrambles with the osmo formula; the SmartNet tests pin OP25's literals —
  the reference actively disagrees with drift instead of decorating it.

## Where this goes next

A trusted reference gives you facts; the discipline is keeping them from
degrading into assumptions once they're inside your codebase.
[Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
is about the mechanics: pinning parsers with literal byte vectors
cross-checked against an independent decoder — and the SCCB bug that showed
exactly how a round-trip test lets an off-by-one live for months.

## FAQ

**How do I find a reference implementation for an obscure protocol?**
Start from the ecosystem that *uses* the protocol: scanner communities
(OP25, SDRTrunk, DSD-FME for North American trunking; osmo-tetra, tetra-kit,
telive for TETRA), Wireshark dissectors, and academic or SDR-framework
projects. Then verify the on-air property — look for user reports of live
decodes, not just a README claiming support.

**What if the only available reference is GPL and my project isn't?**
Use it as a *validation oracle* — run it, compare outputs, pin its literals
in tests — rather than as source to translate. That line, and the licensing
hygiene around it, is exactly
[Part 5]({{ '/blog/deep-dives/from-spec-to-shipping-05-clean-room-rules/' | relative_url }})'s
subject.

**Is a hardware radio a valid reference?**
As an end-to-end oracle, yes — an operator's Astro Spectra decoding a
marginal P25 call that GopherTrunk drops is hard evidence the samples
contain the call. But hardware exposes no intermediate layers, so it can
tell you *that* you're wrong, rarely *where*. Pair it with a software
reference that prints its work.

**Why not just trust the most popular implementation?**
Popularity measures usefulness, not per-layer correctness — and popular
projects share code more than their download counts suggest. The DNB
geometry case is the standing warning: the best DMO reference available
carried a measurably wrong offset pair. Scope trust per layer, demand
on-air evidence for that layer, and let captures referee.

**How many references are enough?**
One proven-on-air reference plus the spec is a workable floor; two with
independent lineage agreeing byte-for-byte is the standard GopherTrunk
aims for on anything load-bearing (MBT layouts, scrambler seeds, SmartNet
framing). Below that floor, the fact stays quarantined the way the
comms-type mapping is — decoded, logged, but not driving behavior.

## Series navigation

**Part 2 of 14** · ←
[Part 1: How to Read a Radio Standard]({{ '/blog/deep-dives/from-spec-to-shipping-01-reading-a-radio-standard/' | relative_url }})
· Next →
[Part 3: Literal Vectors, Not Round-Trips]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
