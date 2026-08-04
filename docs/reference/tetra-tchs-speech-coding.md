---
slug: tetra-tchs-speech-coding
title: TETRA TCH/S speech coding
entry_type: algorithm
category: voice-coding
description: "TETRA TCH/S full-rate speech coding splits each 137-bit ACELP frame into class-0/1/2 bits, protects the class-2 bits with an 8-bit CRC and unequal-rate RCPC convolutional coding, and spreads a slot's two frames across a 24×18 block interleaver before scrambling."
keywords: TETRA TCH/S, speech channel coding, ACELP 137 bits, class-0 class-1 class-2, unequal error protection, class-2 CRC, RCPC, 24x18 interleave, soft-decision, EN 300 395-2 5.5
aka: [TCH/S, "TETRA speech channel coding", "full-rate traffic channel"]
autolink: true
infobox:
  - { label: Input, value: "2 × 137-bit ACELP frames per slot" }
  - { label: Protection, value: "Class-based UEP: CRC-8 + RCPC" }
  - { label: Output, value: "432 type-5 bits (BKN1 + BKN2)" }
  - { label: Spec, value: "ETSI EN 300 395-2 §5.5" }
see_also: [tetra, acelp, code-excited-linear-prediction, vocoder, tetra-rcpc-code, tetra-block-interleaver, soft-decision, tetra-traffic-slot-mapping, cyclic-redundancy-check, forward-error-correction]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://en.wikipedia.org/wiki/Algebraic_code-excited_linear_prediction
---

**TETRA TCH/S** (Traffic Channel / Speech, full rate) is the channel coding that carries a
[TETRA](/reference/tetra/) voice call.[^tetra] The [ACELP](/reference/acelp/) vocoder produces
a **137-bit** speech frame every 30 ms; one transmission slot carries *two* such frames coded
together into 432 on-air bits.[^acelp] The coding is *unequal error protection*: the 137 bits
are ranked by how much a bit error hurts perceived speech, sorted into three sensitivity
classes, and each class is protected proportionally to its importance.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 160" role="img" aria-label="Two 137-bit ACELP frames are reordered into class-0, class-1 and class-2 bit groups; class 2 gains an 8-bit CRC and tail bits and is convolutionally coded most heavily, class 1 is coded lightly, class 0 is sent uncoded, and the resulting 432 type-3 bits pass through a 24 by 18 block interleaver to type-4." xmlns="http://www.w3.org/2000/svg">
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="12" y="20" width="120" height="20" fill="currentColor" fill-opacity="0.20" stroke="currentColor" stroke-width="1"/>
    <text x="72" y="34">2 × 137 ACELP bits</text>
    <rect x="12" y="58" width="70" height="22" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1"/>
    <text x="47" y="72">class0 102</text>
    <rect x="90" y="58" width="70" height="22" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1"/>
    <text x="125" y="72">class1 112</text>
    <rect x="168" y="58" width="120" height="22" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1"/>
    <text x="228" y="69">class2 60 +CRC8 +tail4</text>
    <rect x="12" y="98" width="70" height="22" stroke="currentColor" stroke-width="1" fill="none"/>
    <text x="47" y="112">uncoded 102</text>
    <rect x="90" y="98" width="70" height="22" stroke="currentColor" stroke-width="1" fill="none"/>
    <text x="125" y="109">RCPC 8/12</text><text x="125" y="118" font-size="6.5">→ 168</text>
    <rect x="168" y="98" width="120" height="22" stroke="currentColor" stroke-width="1" fill="none"/>
    <text x="228" y="109">RCPC 8/18</text><text x="228" y="118" font-size="6.5">→ 162</text>
    <rect x="306" y="86" width="150" height="34" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
    <text x="381" y="100">24 × 18 interleave</text><text x="381" y="112">432 type-3 → type-4</text>
  </g>
  <g stroke="currentColor" stroke-width="1" fill="none"><line x1="72" y1="40" x2="72" y2="58"/><line x1="47" y1="80" x2="47" y2="98"/><line x1="125" y1="80" x2="125" y2="98"/><line x1="228" y1="80" x2="228" y2="98"/><line x1="288" y1="103" x2="306" y2="103"/></g>
  <text x="235" y="146" text-anchor="middle" font-size="8" fill="currentColor">102 + 168 + 162 = 432 type-3 bits per slot</text>
</svg>
<figcaption>Each slot's two 137-bit frames are reordered into three sensitivity classes; class 2 gains an 8-bit CRC and is coded most heavily, class 1 lightly, class 0 not at all, and the 432 type-3 bits are spread across a 24×18 interleaver.</figcaption>
</figure>

## Class split and coding

Reordered into type-2 order (EN 300 395-2 Table 5), the two frames' 274 speech bits group as
**class 0** (102 bits — 2×51 of the least sensitive), **class 1** (112 bits — 2×56), and
**class 2** (60 bits — 2×30, the most sensitive). An 8-bit CRC and 4 tail bits follow class 2.
The three classes are coded differently:

- **Class 0** is sent *uncoded* — its bits map straight into the output.
- **Class 1** is [RCPC](/reference/tetra-rcpc-code/)-coded at rate 8/12, expanding 112 bits to
  168.
- **Class 2 + CRC + tail** are RCPC-coded at rate 8/18, expanding the 72 bits to 162.

Both coded regions come from one continuous K=5 rate-1/3 mother stream, punctured to the two
different rates. The result — 102 + 168 + 162 = 432 type-3 bits — is then permuted by a
**24×18 block interleaver** (`C4(i·24+j) = C3(j·18+i)`, reading the matrix column by column)
to type-4, and scrambled to the type-5 bits carried on air as BKN1 + BKN2 around the slot's
training sequence.

## The class-2 CRC trap

The 8-bit class-2 CRC (§5.5.1) is the gate the decoder uses to accept or reject a burst — and
its definition is a landmine. It is **not** a cyclic `G(X)` LFSR CRC. Each of the 8 CRC bits
is the even parity (XOR) of the class-2 bits at a fixed set of ranks — a fixed parity-check
matrix, verbatim the ETSI reference codec's `TAB_CRC1..8` tables. An earlier
`G(X) = 1 + X³ + X⁷` approximation looked plausible and passed synthetic encode-decode
round-trips (it set the bits the same wrong way on both ends), but it did not match on-air
frames, so *every real TCH/S burst failed the CRC and was silently dropped*. This is the
canonical "self-consistent bug": the vocoder unit tests passed while no live voice decoded,
because the fault was in the channel coding, not the codec.

## Soft-decision path

GopherTrunk decodes TCH/S [soft-decision](/reference/soft-decision/): the receiver's
per-symbol differentials become type-5 LLRs, descrambled and deinterleaved in the soft domain,
soft-depunctured into the mother stream, and run through a soft Viterbi decoder before the
hard class-2 CRC check. The class-0 bits, being uncoded, are hard-sliced straight from their
LLRs. The soft path recovers roughly 2 dB of coding gain the hard slicer discards, which on a
marginal same-carrier call is the difference between a clean recording and a short, garbled
one — it is what fixed the same-carrier voice-follow captures.

## Relevance to SDR

`internal/radio/tetra/tch.go` implements the chain both ways — `DecodeTCHS` /
`TCHSpeechFrames` (hard) and `DecodeTCHSSoft` / `TCHSpeechFramesSoft` (soft) — and
`tch_tables.go` holds the generated type-2 bit order and the `tchCRCTaps` parity matrix. The
CRC gate also does double duty as a slot demux on a single-carrier system: a burst that is not
TCH/S speech descrambles to essentially random class-2 bits and passes the 8-bit CRC only
about 1 in 256 times, so a passing CRC reliably isolates the granted call's speech from the
other TDMA timeslots' bursts. The 137-bit frames it recovers feed the clean-room ACELP
[vocoder](/reference/vocoder/) in `internal/voice/acelp`.

## Sources

[^tetra]: [Terrestrial Trunked Radio](https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio) — Wikipedia, on the TETRA voice traffic channel.
[^acelp]: [Algebraic code-excited linear prediction](https://en.wikipedia.org/wiki/Algebraic_code-excited_linear_prediction) — Wikipedia, on the ACELP speech coder whose frames TCH/S carries.
