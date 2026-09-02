---
title: "From Spec to Shipping, Part 5: Clean-Room Rules — Reading Without Copying"
description: "How a pure-Go, Apache-2.0 decoder uses GPL decoders as validation oracles without copying them: where the line sits between wire-format facts and copyrightable expression, the vocoder patent aisle, and the licensing hygiene that CI enforces on every build."
category: deep-dives
keywords: clean room implementation sdr decoder, gpl code as validation oracle, wire format facts copyright, mbelib isc license attribution, ambe patent status, etsi reference codec licensing, third party licenses go binary, apache 2.0 pure go decoder, gophertrunk from spec to shipping
tags: [from-spec-to-shipping, licensing, clean-room, vocoders, methodology, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From Spec to Shipping"
series_part: 5
---

*Part 5 of **From Spec to Shipping**, a 14-part series on how a protocol
decoder actually gets written — from standards documents and independent
references to code you can trust on air.
[Part 4]({{ '/blog/deep-dives/from-spec-to-shipping-04-conformance-harness/' | relative_url }})
built the conformance harness: the ETSI reference codec and GopherTrunk's
decoder fed the same 137-bit bitstream, bit-identical PCM or bust. That
harness only works because the reference is allowed to be *run* — and this
part is about the rule that makes everything else in the series legal to
do: how to lean on other people's decoders as hard as we do, every day,
without ever copying them into an Apache-2.0 binary.*

> **TL;DR:** GopherTrunk is **pure Go under Apache-2.0** — while its most
> valuable references (OP25, trunk-recorder, osmo-tetra-dmo, DSD-FME) are
> GPL. The rule that reconciles them: **implement from the spec; use other
> decoders as validation oracles** — run them, compare outputs, cite their
> constants — never as source to translate. Wire-format *facts* (a sync
> word, an interleave stride, a CRC register, a bit offset) cross that
> wall freely with a citation; expression-for-expression translation does
> not. The paperwork lives in `LICENSE`, `THIRD_PARTY_LICENSES.md` and a
> `make licenses` CI job that fails the build on any non-permissive
> dependency; the vocoders each carry an explicit provenance statement
> (mbelib's ISC codebook tables attributed, the ACELP port's ETSI lineage
> declared in `internal/voice/acelp/ops.go`) because patents and licenses
> are separate questions with separate answers.

**Key takeaways**

- **An oracle is run, not read into your editor.** The highest-value use
  of a GPL decoder is feeding it the same input as your code and
  comparing outputs — DSD-FME decoding the same `.amb` frames, the ETSI
  reference decoder producing the PCM your decoder must match
  sample-for-sample. Nothing crosses the license wall but bytes.
- **Facts are free; expression is not.** The SmartNet sync word `0xAC`,
  the stride-19 interleave, the XOR masks — those are properties of the
  radio waves, learned from OP25 and cited to it. The Go code around them
  is written fresh, in this project's idioms, against this project's
  interfaces.
- **Patents are a separate ledger from licenses.** mbelib is permissively
  licensed (ISC) and AMBE+2 is still patent-encumbered anyway; IMBE and
  core ACELP patents have expired. A pure-Go rewrite changes the license
  answer and does not move the patent answer one inch.
- **Hygiene is a build gate, not a document.** `make licenses` inventories
  every module the binary links; CI diffs the result and fails on a
  non-permissive newcomer. A licensing policy that is not enforced by the
  build is a policy that drifts.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Project license | Apache-2.0 for everything original | `LICENSE` |
| Third-party inventory | direct deps table + in-tree attributions | `THIRD_PARTY_LICENSES.md` |
| License CI gate | inventories transitive deps, fails on non-permissive | `make licenses`, the `licenses` job in `.github/workflows/ci.yml` |
| mbelib table data | ISC-licensed quantiser codebooks, attributed | `internal/voice/ambe2/`, `internal/voice/imbe/` |
| ACELP provenance | ETSI reference lineage declared at the package door | `internal/voice/acelp/ops.go` (package doc) |
| Validation oracles | skip-guarded conformance vs reference binaries | `internal/voice/acelp/etsi_reference_test.go` |
| Patent posture | IMBE/ACELP expired, AMBE+2 encumbered — stated plainly | `docs/vocoders.md` |

## In this post

- **Two rooms, one wall** — the working model: spec into the
  implementation room, binaries into the validation room.
- **Facts cross the wall; expression does not** — where the line actually
  sits, with the SmartNet constants as the worked example.
- **The vocoder aisle** — three codecs, three different provenance
  stories, one honesty rule.
- **Hygiene as a build gate** — `THIRD_PARTY_LICENSES.md`, `make
  licenses`, and what deliberately never links into the binary.
- **How that principle shaped the Go code** — the pattern in four bullets.

## Two rooms, one wall

Picture the project as two rooms. The **implementation room** holds the
specs — ETSI EN 300 392-2, EN 300 395-2, the TIA-102 family from
[Part 1]({{ '/blog/deep-dives/from-spec-to-shipping-01-reading-a-radio-standard/' | relative_url }})
— plus this project's own Go code. The **validation room** holds
everything else: OP25, trunk-recorder, osmo-tetra-dmo, DSD-FME, the
compiled ETSI reference tools. The wall between them passes exactly two
kinds of traffic: **outputs** (bytes to compare against) and **facts**
(constants and layouts, with a citation).

The validation room earns its keep constantly.
[Part 4]({{ '/blog/deep-dives/from-spec-to-shipping-04-conformance-harness/' | relative_url }})'s
conformance harness runs the ETSI `scoder`/`sdecoder` binaries and
demands GopherTrunk's ACELP output match the reference decoder
**sample-for-sample** — 0 mismatches over 96,000 samples. The AMBE+2
band-energy investigation used DSD-FME's decode of the *same* `.amb`
frames as ground truth to localize a high-band deficit to one parameter
unpacker. The DMO scrambler seed was cross-checked byte-for-byte against
osmo-tetra-dmo's `tetra_scramb_get_init`. In every case the reference ran
as a black box and only its *output* crossed the wall — which is what
makes a GPL reference safe to use this hard: running a GPL program
imposes nothing on your code; translating its source into your tree
makes your tree a derivative work.

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="Two rooms separated by a wall. The implementation room on the left receives the specs and contains the pure-Go Apache-2.0 code. The validation room on the right contains GPL and reference binaries: OP25, trunk-recorder, osmo-tetra-dmo, DSD-FME, and the ETSI reference codec. Two arrows cross the wall: outputs flowing left into a comparator, and cited facts such as sync words and CRC registers flowing left into constants. A crossed-out arrow labelled source translation shows what never crosses.">
  <rect x="20" y="30" width="270" height="180" rx="8" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="155" y="52" text-anchor="middle" fill="var(--accent)" font-size="11" font-weight="bold">implementation room</text>
  <text x="155" y="72" text-anchor="middle" fill="currentColor" font-size="10">specs: ETSI EN / TIA-102</text>
  <text x="155" y="90" text-anchor="middle" fill="currentColor" font-size="10">pure-Go code, Apache-2.0</text>
  <rect x="45" y="110" width="220" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="155" y="129" text-anchor="middle" fill="currentColor" font-size="10">cited constants (0xAC, stride 19, …)</text>
  <rect x="45" y="150" width="220" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="155" y="169" text-anchor="middle" fill="currentColor" font-size="10">comparator: bit-exact / yield A/B</text>
  <rect x="390" y="30" width="270" height="180" rx="8" fill="none" stroke="var(--fg-muted)" stroke-width="2"/>
  <text x="525" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="11" font-weight="bold">validation room (run, not read)</text>
  <text x="525" y="74" text-anchor="middle" fill="currentColor" font-size="10">OP25 · trunk-recorder (GPL)</text>
  <text x="525" y="92" text-anchor="middle" fill="currentColor" font-size="10">osmo-tetra-dmo · DSD-FME</text>
  <text x="525" y="110" text-anchor="middle" fill="currentColor" font-size="10">ETSI reference codec (ETSI terms)</text>
  <line x1="330" y1="10" x2="330" y2="230" stroke="currentColor" stroke-width="2"/>
  <line x1="390" y1="125" x2="290" y2="125" stroke="var(--accent)" stroke-width="2"/>
  <polygon points="292,121 284,125 292,129" fill="var(--accent)"/>
  <text x="340" y="118" text-anchor="middle" fill="var(--accent)" font-size="9">facts +</text>
  <text x="340" y="140" text-anchor="middle" fill="var(--accent)" font-size="9">outputs</text>
  <line x1="390" y1="185" x2="290" y2="185" stroke="var(--fg-muted)" stroke-dasharray="5 4"/>
  <line x1="322" y1="173" x2="342" y2="197" stroke="var(--fg-muted)" stroke-width="2"/>
  <line x1="342" y1="173" x2="322" y2="197" stroke="var(--fg-muted)" stroke-width="2"/>
  <text x="340" y="212" text-anchor="middle" fill="var(--fg-muted)" font-size="9">source translation: never</text>
</svg>
<figcaption>The wall passes outputs and cited facts in one direction only — source code never crosses it, which is what lets an Apache-2.0 binary lean on GPL oracles daily.</figcaption>
</figure>

## Facts cross the wall; expression does not

The line the wall enforces is the one copyright itself draws: facts and
methods of operation are not copyrightable; a particular *expression* of
them is. A wire format is a fact about the world — the radio waves carry
the sync word `0xAC` whether or not anyone wrote a decoder. So when the
SmartNet framing was rebuilt from proven decoders
([Part 8]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})'s
case study, [#1143](https://github.com/MattCheramie/GopherTrunk/issues/1143)),
what crossed the wall from OP25's `rx_smartnet.cc` was a set of facts,
landed as cited constants:

```go
// internal/radio/motorola/frame.go (shape)
const (
    // OutboundSyncHex is the 8-bit outbound sync word 10101100.
    OutboundSyncHex uint32 = 0xAC

    // idXORMask / cmdXORMask un-invert the address and command
    // fields (the wire carries the data bits inverted).
    // From rx_smartnet.h: ID_XOR 0x33C7, CMD_XOR 0x32A.
    idXORMask  uint16 = ^uint16(0x33C7)        // 0xCC38
    cmdXORMask uint16 = ^uint16(0x32A) & 0x3FF // 0x0D5

    // CRC-10 registers (rx_smartnet.cc crc_check).
    crcInit uint16 = 0x0393
    crcOp   uint16 = 0x036E
    crcPoly uint16 = 0x0225
)
```

Every number names its source, and the tests that pin them
(`TestOutboundSyncBitsMatchReference`, `TestXORMasksMatchReference`)
cite the same OP25 literals —
[Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})'s
discipline applied at the license boundary. What did *not* cross the
wall is everything around those numbers: the buffering, the framer state
machine's shape, the error handling, the interfaces — all written fresh
in Go against GopherTrunk's own `BitSink`/`events.Bus` contracts, in a
codebase that shares no line with the C++ it validated against.

The distinction changes how you work. A translator rewrites someone
else's function statement by statement — inheriting its structure, its
bugs, and its license. A clean-room implementer extracts the *claims*
the code makes about the wire ("payload position `k + l*19`
deinterleaves to `k*4 + l`"), writes them down as testable facts, and
implements them however the target codebase wants. More work up front,
repaid immediately: you cannot extract a claim you don't understand, so
every fact that crosses the wall arrives *understood* — exactly the
property the
[self-consistent-trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
series villain preys on the absence of.

## The vocoder aisle: licenses, patents, and provenance

Nowhere do these questions get sharper than vocoders. GopherTrunk's
three tell three different stories, each documented in
`docs/vocoders.md` and the packages themselves.

| Codec | License story | Patent story |
|---|---|---|
| IMBE (P25 Phase 1) | pure Go, mbelib's ISC table data attributed | core patents (early-90s filings) **expired** |
| AMBE+2 (DMR, P25 P2, NXDN) | pure Go, same ISC attribution | **still encumbered** — DVSI's, deployer's risk |
| TETRA ACELP | port of the spec's own reference, ETSI terms declared | core algebraic-CELP patents **expired** |

**The mbelib case** is the narrowest possible borrowing. The AMBE+2 and
IMBE decoders (`internal/voice/ambe2/`, `internal/voice/imbe/`) use
quantisation codebook tables originally extracted from mbelib — pure
spec-fixed data, released under ISC, which requires attribution and gets
it in `THIRD_PARTY_LICENSES.md`. The synthesis pipelines around those
tables were re-implemented from scratch, and the attribution note says
exactly that — an attribution that overclaims is as misleading as one
that's missing. The patent line is stated with equal bluntness:
**re-implementing AMBE+2 in pure Go does not change the patent posture** —
the patents cover the algorithm, not the language. The code ships; the
legal evaluation belongs to the deployer.

**The ACELP case** is the opposite shape: here the *spec itself ships
the code*. EN 300 395-2's normative artifact is fixed-point ANSI C, and
implementing the standard means reproducing that arithmetic bit-exactly —
so the package doc opens with a provenance block instead of burying it:

```go
// internal/voice/acelp/ops.go (shape) — package doc
// PROVENANCE / LICENSING: this port derives from the ETSI reference
// codec (fixed-point ANSI C), obtained via
// github.com/curlyboi/libtetradec. The ETSI reference implementation
// is copyrighted by ETSI and distributed under ETSI's terms; the
// codec is also historically patent-encumbered (algebraic CELP,
// though the core patents are largely expired). […] This is why the
// decoder lives in its own package behind an explicit vocoder
// registration.
package acelp
```

The decoder was ported from the algorithmic description and the ITU-T
fixed-point operator definitions, with the fixed quantiser tables —
numeric constants, again — sourced from the reference. The gate that
makes the port trustworthy is Part 4's bit-exact conformance run, and
the gate that keeps it honest is structural: the decoder registers
behind an explicit `tetra-acelp` vocoder name so a build can omit it,
and the ETSI sources and vectors are **not committed** — the
skip-guarded harness points at files the operator builds themselves.
Different provenance, same principle: say exactly where every byte came
from, in the file where the bytes live.

## Hygiene as a build gate

A licensing posture that lives only in a Markdown file decays. So the
inventory is mechanical: `THIRD_PARTY_LICENSES.md` hand-curates the
direct dependency table (thirteen modules, all MIT / BSD / Apache-2.0
families), and `make licenses` generates the full transitive inventory
into a committed CSV. The CI `licenses` job re-runs the target, diffs
against the committed copy, and **fails the build** on any
non-permissive newcomer — a PR that adds a dependency must regenerate
both files, in the open, in the diff.

Three boundaries are worth naming because each is easy to get wrong:

**Test-only references never link.** The ETSI tools, DSD-FME, the
operator-capture harnesses — all reachable only from skip-guarded tests
behind environment variables (`GT_ETSI_SERIAL`, `GT_TETRA_DMO_IQ`, …).
Nothing in `go.mod` pulls a reference implementation.

**Bundled is not linked.** The Windows installer ships Zadig — GPL-3.0 —
as an unmodified upstream executable launched from a Start Menu shortcut,
never linked into the daemon. `THIRD_PARTY_LICENSES.md` spells out the
distinction: "we distribute a GPL binary" and "our binary is GPL-derived"
are different sentences with different consequences.

**Specs get archived with their terms.** The reference PDFs under
`docs/specs/` ride under the standards bodies' own distribution terms,
noted per-document — the input side of the implementation room gets the
same paper trail as the output side. (For the general landscape behind
these choices — copyleft versus permissive, derivative works, attribution
clauses — see the
[software licensing learn module]({{ '/learn/software-licensing/' | relative_url }}).)

### How that principle shaped the Go code

- **Provenance is a package-doc concern.** `acelp`'s license block sits
  at the top of `ops.go`; `motorola/frame.go`'s header names OP25 and
  says outright that the constants are ported facts. The place you read
  the code is the place you learn where it came from.
- **The oracle interface is bytes.** Conformance harnesses consume files
  the reference tools *produced* (`serial.bin`, `ref_out.pcm`) — never
  the reference's internals — so the comparison stays legal and stays
  honest at once.
- **Registration boundaries mirror legal boundaries.** Each vocoder is a
  separately-registered package, so posture decisions ("omit this codec
  in that jurisdiction") map to a build decision, not a refactor.
- **The inventory is diffable.** Hand-curated table for humans, generated
  CSV for machines, CI to keep them synchronized — the same
  literal-vector instinct as [Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }}),
  pointed at the dependency graph.

## Where this goes next

Clean-room discipline assumes your references agree. They don't, always —
and when two implementations you trust give different answers for the
same burst geometry, no amount of reading settles it.
[Part 6]({{ '/blog/deep-dives/from-spec-to-shipping-06-when-references-disagree/' | relative_url }})
is about the referee that outranks every implementation: a measurement on
a real capture, designed so the right answer wins by a wide margin — and
so the machinery *refuses to answer* when it doesn't.

## FAQ

**Can I legally use a GPL decoder to test my permissively-licensed one?**
Yes — running a GPL program and comparing its output against yours
imposes no license obligation on your code; the GPL governs copying and
derivation of the *source*, not use of the program. The discipline is
keeping the boundary clean and auditable: outputs and cited facts cross,
source structure does not.

**Are protocol constants like sync words and CRC polynomials
copyrightable?**
Constants that describe a wire format are facts about an external system,
and facts are not copyrightable — a sync word learned from OP25 is the
same sync word the radio transmits. Copyright protects creative
expression: the surrounding code's structure and phrasing. GopherTrunk
cites the source of every borrowed constant anyway, as courtesy and as
engineering provenance.

**Why is the ACELP decoder a port of the reference when everything else
is implemented from prose?**
Because for ACELP the reference *is* the spec: EN 300 395-2's normative
definition of the codec is fixed-point C whose exact saturation behaviour
the standard requires, and ETSI distributes it to be implemented from.
The obligations that path carries — ETSI's terms, the provenance
statement, the not-committed vectors — are documented at the package door.

**Does rewriting a patented codec in another language avoid the patent?**
No. Patents cover the algorithm, not its expression, so a pure-Go AMBE+2
decoder sits in exactly the same patent posture as mbelib's C —
`docs/vocoders.md` says so in as many words. The rewrite changes the
*copyright* story (no borrowed expression) and the engineering story
(testable, idiomatic, memory-safe), not the patent one.

**What stops a contributor from accidentally adding a GPL dependency?**
The build. `make licenses` inventories every module the binary links, CI
diffs the result against the committed copy, and a non-permissive
newcomer fails the job — the question surfaces in the introducing PR,
when it costs a conversation instead of a rewrite.

## Series navigation

**Part 5 of 14** · ←
[Part 4: The Conformance Harness — Bit-Identical or Bust]({{ '/blog/deep-dives/from-spec-to-shipping-04-conformance-harness/' | relative_url }})
· Next →
[Part 6: When References Disagree, the Capture Referees]({{ '/blog/deep-dives/from-spec-to-shipping-06-when-references-disagree/' | relative_url }})
