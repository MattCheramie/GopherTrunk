---
slug: dstar-header-fec
title: D-STAR header FEC
entry_type: algorithm
category: amateur-digital
description: The D-STAR header FEC chain protects the 41-byte DV-mode header — the callsign-routing block that opens every D-STAR transmission — with a K=5 half-rate convolutional code, a PN15 scrambler, and a block interleaver, expanding 328 bits into 660 on-wire channel bits.
keywords: D-STAR header FEC, DV mode, PCH header, convolutional code, PN15 scrambler, block interleaver, CRC-16 CCITT, callsign routing, frame sync EAA060, K=5 half rate
aka: ["D-STAR PCH", "D-STAR header", "DV header FEC"]
autolink: true
infobox:
  - { label: Header, value: 41 bytes (328 bits) }
  - { label: On-wire, value: 660 channel bits }
  - { label: Chain, value: "conv → puncture → scramble → interleave" }
  - { label: Frame sync, value: "0xEAA060 (24 bits)" }
see_also: [d-star, convolutional-code, scrambling, interleaving, viterbi-algorithm, cyclic-redundancy-check, forward-error-correction, frame-synchronization]
cite_urls:
  - https://en.wikipedia.org/wiki/D-STAR
  - https://en.wikipedia.org/wiki/Convolutional_code
---

The **D-STAR header FEC** is the coding chain that protects the DV-mode Preamble + Header
(PCH) — the 41-byte block that opens every [D-STAR](/reference/d-star/) transmission and names
the source, destination, and repeater-routing callsigns.[^dstar] The header is transmitted
just once at the head of a transmission, so it cannot rely on repetition the way the voice
frames do; instead a heavy FEC chain — a [convolutional](/reference/convolutional-code/) code,
a PN15 [scrambler](/reference/scrambling/), and a block [interleaver](/reference/interleaving/)
— expands its 328 information bits into 660 on-wire channel bits so a receiver can recover the
routing even under a fading GMSK channel.[^conv]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 170" role="img" aria-label="The D-STAR header FEC chain: a 41-byte information field of 328 bits gains a 4-bit flush tail, a K=5 half-rate convolutional code produces 664 channel bits, four trailing bits are punctured to leave 660, a PN15 scrambler XORs a keystream over them, and a 22 by 30 block interleaver permutes them into the 660 on-wire bits the modulator emits." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
  <rect x="120" y="14" width="230" height="22" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/><text x="235" y="29">41-byte header (328 bits) + 4-bit tail</text>
  <path d="M235 36 L235 50" stroke="currentColor" stroke-width="1.1" marker-end="url(#b)"/>
  <rect x="120" y="50" width="230" height="22" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="235" y="65">K=5 ½-rate conv (G1=0x19, G2=0x17) → 664</text>
  <path d="M235 72 L235 86" stroke="currentColor" stroke-width="1.1" marker-end="url(#b)"/>
  <rect x="120" y="86" width="230" height="22" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/><text x="235" y="101">puncture 4 trailing bits → 660</text>
  <path d="M235 108 L235 122" stroke="currentColor" stroke-width="1.1" marker-end="url(#b)"/>
  <rect x="120" y="122" width="230" height="22" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="235" y="137">PN15 scramble (x¹⁵+x+1)</text>
  <path d="M235 144 L235 156" stroke="currentColor" stroke-width="1.1" marker-end="url(#b)"/>
  <rect x="120" y="156" width="230" height="14" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="235" y="166" font-size="7">22×30 interleave → 660 on-wire bits</text>
  </g>
  <defs><marker id="b" markerWidth="6" markerHeight="6" refX="5" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The header's 328 bits pass through convolutional coding, puncture, scrambling, and interleaving to become 660 on-wire channel bits; the decoder runs the chain in reverse and checks the Viterbi tail bits terminate in the zero state.</figcaption>
</figure>

## The FEC chain

The encoder packs the 41-byte information field into 328 bits and appends a 4-bit flush tail
(K−1 for K=5), then runs the chain:

1. **Convolutional encode.** A K=5, rate-½ code with polynomials **G1 = 0x19** (1 + x³ + x⁴)
   and **G2 = 0x17** (1 + x + x² + x⁴) — the same pair MMDVMHost, DSDcc, and OpenDV use for the
   D-STAR header — turns 332 input bits into 664 channel bits.
2. **Puncture.** The four trailing channel bits are dropped to land on the JARL-spec 660-bit
   on-wire header window.
3. **Scramble.** A 15-bit PN15 LFSR (polynomial x¹⁵ + x + 1, initial register 0x0001) XORs a
   keystream over the 660 bits, breaking up long runs that would stress GMSK clock recovery.
4. **Interleave.** A 22 × 30 block interleaver (660 cells, an exact fit) writes column-major
   and reads row-major, spreading a channel burst across the codeword.

The decoder reverses the chain — deinterleave, descramble (the XOR is self-inverse),
depuncture with erasure marks, and a K=5 [Viterbi](/reference/viterbi-algorithm/) decode — and
then checks that the recovered flush tail bits are zero. A non-zero tail means the survivor
path did not terminate in the encoder's zero state, so the payload is rejected as an
unrecoverable error burst.

## Header fields

After the FEC is stripped, the 41-byte header carries:

| Bytes | Field | Meaning |
|---|---|---|
| 0 | FLAG1 | Data / Repeater / Interrupted / Control / Urgent / EMR / Break-In flags |
| 1–2 | FLAG2, FLAG3 | Supplementary flags |
| 3–10 | RPT2 | Destination repeater callsign (8 chars, space-padded) |
| 11–18 | RPT1 | Gateway / source repeater callsign |
| 19–26 | UR | Destination station ("CQCQCQ" = group call, "/…" = routing) |
| 27–34 | MY1 | Source / own-station callsign |
| 35–38 | MY2 | 4-character short suffix |
| 39–40 | CRC | CRC-16-CCITT (poly 0x1021, init 0xFFFF) over bytes 0–38 |

The UR field drives dispatch: a `CQCQCQ` tag or any `/`-prefixed repeater routing marks a
group transmission; a specific callsign marks a directed call. FLAG1 carries the emergency
(EMR) and break-in bits a trunking-style follower surfaces.

## In practice

The 24-bit Header Frame Sync is **0xEAA060**; a sliding detector locks onto it (tolerating a
couple of mismatched bits) before the adapter slices the 660-bit FEC-encoded payload. The
convolutional polynomials match the open-source decoders exactly, but GopherTrunk flags the
scrambler and interleaver as self-consistent encode/decode pairs whose exact permutation
tables still need calibrating against a captured live transmission — a best-effort caveat
worth respecting before trusting a marginal off-air header decode. The chain's structure is
the familiar burst-defence recipe: interleaving to disperse fades, a rate-½ code to correct
what disperses, scrambling to keep the clock, and a CRC as the final integrity gate.

## Relevance to SDR

`internal/radio/framing/dstar_header.go` implements the full chain
(`EncodeDStarHeaderFEC` / `DecodeDStarHeaderFEC`, the PN15 scrambler, the 22×30 interleaver),
and `internal/radio/dstar/header.go` holds the 41-byte field parse and the CRC-16-CCITT
(`ParseHeader`, `ComputeCRC`). Together they let GopherTrunk recover D-STAR routing metadata —
who is calling whom, through which repeaters — from the single header frame that opens each
transmission.

## Sources

[^dstar]: [D-STAR](https://en.wikipedia.org/wiki/D-STAR) — Wikipedia, on the JARL D-STAR digital voice/data protocol and its header.
[^conv]: [Convolutional code](https://en.wikipedia.org/wiki/Convolutional_code) — Wikipedia, on the rate-½ coding the header FEC uses.
