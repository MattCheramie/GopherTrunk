---
slug: tetra-logical-channels
title: TETRA logical channels
entry_type: term
category: trunked-radio
description: "The TETRA control-plane logical channels — BSCH, SCH/HD, BNCH, STCH, SCH/HU and SCH/F — each map a fixed number of information bits onto a fixed number of coded bits through a shared CRC, RCPC, interleave and scramble chain."
keywords: TETRA logical channels, BSCH, SCH/F, SCH/HD, SCH/HU, BNCH, STCH, TETRA channel coding, type-1 type-5 bits, EN 300 392-2 8.3.1
aka: [BSCH, SCH/F, SCH/HD, SCH/HU, "TETRA signalling channels"]
autolink: true
infobox:
  - { label: Channels, value: "BSCH, SCH/HD, BNCH, STCH, SCH/HU, SCH/F" }
  - { label: Coding chain, value: "CRC-16 + tail → RCPC 2/3 → interleave → scramble" }
  - { label: Modulation, value: "π/4-DQPSK, 18 ksym/s" }
  - { label: Spec, value: "ETSI EN 300 392-2 §8.3.1" }
see_also: [tetra, tetra-rcpc-code, tetra-block-interleaver, tetra-scrambler, tetra-aach, tetra-mac-pdu, tetra-extended-colour-code, control-channel, cyclic-redundancy-check, convolutional-code, puncturing, interleaving]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://en.wikipedia.org/wiki/Convolutional_code
---

The **TETRA logical channels** are the named control-plane bearers a
[TETRA](/reference/tetra/) downlink carries — the broadcast, common-control and
dedicated-signalling channels that ride inside its slots.[^tetra] Each channel is defined by
a pair of sizes: how many *type-1* information bits it accepts and how many *type-5* on-air
bits they become after channel coding. A receiver that knows those sizes and the coding
chain can invert the chain and recover the bits, which are then a
[MAC PDU](/reference/tetra-mac-pdu/) for the upper layers to parse.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="The shared TETRA signalling coding chain: K1 type-1 information bits gain a 16-bit CRC and 4 tail bits to form type-2, a rate two-thirds RCPC convolutional code expands them to type-3, a block interleaver permutes them to type-4, and the scrambler produces the K3 type-5 on-air bits." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="lcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="10" y="40" width="70" height="30" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
    <text x="45" y="52">type-1</text><text x="45" y="63">K1 info</text>
    <rect x="100" y="40" width="76" height="30" fill="none" stroke="currentColor" stroke-width="1"/>
    <text x="138" y="52">+CRC16</text><text x="138" y="63">+tail → type-2</text>
    <rect x="196" y="40" width="76" height="30" fill="none" stroke="currentColor" stroke-width="1"/>
    <text x="234" y="52">RCPC 2/3</text><text x="234" y="63">→ type-3</text>
    <rect x="292" y="40" width="76" height="30" fill="none" stroke="currentColor" stroke-width="1"/>
    <text x="330" y="52">interleave</text><text x="330" y="63">→ type-4</text>
    <rect x="388" y="40" width="72" height="30" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
    <text x="424" y="52">scramble</text><text x="424" y="63">type-5 K3</text>
  </g>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <line x1="80" y1="55" x2="100" y2="55" marker-end="url(#lcar)"/>
    <line x1="176" y1="55" x2="196" y2="55" marker-end="url(#lcar)"/>
    <line x1="272" y1="55" x2="292" y2="55" marker-end="url(#lcar)"/>
    <line x1="368" y1="55" x2="388" y2="55" marker-end="url(#lcar)"/>
  </g>
  <text x="235" y="110" text-anchor="middle" font-size="8" fill="currentColor">BSCH 60→120 · SCH/HD 124→216 · SCH/HU 92→168 · SCH/F 268→432</text>
  <text x="235" y="128" text-anchor="middle" font-size="7.5" fill="currentColor">AACH 14→30 is the exception: RM(30,14) block code, no RCPC or interleave</text>
</svg>
<figcaption>Every SCH-family channel runs the same four-stage chain — CRC, RCPC, interleave, scramble — differing only in its block sizes; the AACH is the exception, using a Reed–Muller block code instead.</figcaption>
</figure>

## The channels

| Channel | Type-1 → type-5 | Role |
| --- | --- | --- |
| BSCH | 60 → 120 | Broadcast Synchronisation Channel — the MAC SYNC that carries colour code, slot/frame numbering and the network identity used to seed the scrambler |
| SCH/HD | 124 → 216 | Signalling Channel / Half slot Downlink — dedicated downlink signalling |
| BNCH | 124 → 216 | Broadcast Network Channel — SYSINFO / network-broadcast (same coding as SCH/HD) |
| STCH | 124 → 216 | Stealing Channel — signalling that steals a traffic half-slot (same coding as SCH/HD) |
| SCH/HU | 92 → 168 | Signalling Channel / Half slot Uplink — random-access uplink signalling |
| SCH/F | 268 → 432 | Signalling Channel / Full slot — the full-slot signalling bearer, one MAC PDU per slot |
| AACH | 14 → 30 | [Access Assignment Channel](/reference/tetra-aach/) — RM(30,14), not the SCH chain |

## The coding chain

Except for the AACH, every one of these channels runs the same four stages, §8.3.1:

1. **CRC-16.** A 16-bit [CRC](/reference/cyclic-redundancy-check/) is appended over the K1
   information bits (the (K1+16, K1) block code of §8.2.3.3). It is the standard CRC-CCITT
   — polynomial `0x1021`, initial fill `0xFFFF`, final XOR `0xFFFF` — and is the gate a
   receiver uses to accept or reject each recovered block.
2. **Tail + RCPC.** Four zero tail bits flush the encoder, and the
   [RCPC code](/reference/tetra-rcpc-code/) — a K=5 rate-1/4 mother [convolutional
   code](/reference/convolutional-code/) [punctured](/reference/puncturing/) to rate 2/3 —
   expands the type-2 bits to type-3 at a 3∶2 ratio.
3. **Interleaving.** A [block interleaver](/reference/tetra-block-interleaver/) permutes the
   type-3 bits to type-4, spreading a burst of channel errors across the codeword so the
   Viterbi decoder sees them as scattered.
4. **Scrambling.** The [scrambler](/reference/tetra-scrambler/) XORs a colour-code-seeded PN
   sequence over the bits to produce type-5. The BSCH is a special case: it is always
   scrambled with **colour code 0** (§8.2.5.2) so a cold receiver with no configuration can
   decode it, then read the network identity it needs to form the
   [extended colour code](/reference/tetra-extended-colour-code/) for every other channel.

GopherTrunk decodes each channel both hard-decision and [soft-decision](/reference/soft-decision/);
the soft-input Viterbi recovers roughly 1.5–2 dB the hard slicer discards, which matters
most on the longer SCH/F block where a marginal constellation accumulates more symbol errors
before the CRC gate.

## Relevance to SDR

`internal/radio/tetra/channel_coding.go` implements the whole set as `EncodeBSCH` /
`DecodeBSCH`, `EncodeSCHHD` / `DecodeSCHHD` (also used for BNCH and STCH), `EncodeSCHHU` /
`DecodeSCHHU`, and `EncodeSCHF` / `DecodeSCHF`, each with a `…Soft` twin, all composed from
the shared `signalingEncode` / `signalingDecode` helpers plus the framing primitives. The
BSCH decode is the bootstrap: `sync_pdu.go` parses its 60 bits into the `SyncPDU` whose MCC,
MNC and colour code form the extended colour code that unlocks BNCH, SCH/HD and SCH/F, so
getting the BSCH's colour-0 rule right is what lets everything else on the cell decode
without operator configuration.

## Sources

[^tetra]: [Terrestrial Trunked Radio](https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio) — Wikipedia, on the TETRA logical-channel structure.
[^conv]: [Convolutional code](https://en.wikipedia.org/wiki/Convolutional_code) — Wikipedia, on the code family the RCPC stage is built from.
