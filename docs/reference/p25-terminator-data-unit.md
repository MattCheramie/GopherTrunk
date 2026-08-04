---
slug: p25-terminator-data-unit
title: P25 Terminator Data Unit
entry_type: term
category: trunked-radio
description: The P25 Terminator Data Unit ends a voice transmission — TDU (DUID 0x3) is a bare terminator, while TDULC (DUID 0xF) also carries a 72-bit Link Control word behind inner Golay and outer RS(24,12,13) FEC.
keywords: P25 TDU, TDULC, terminator data unit, terminator with link control, end of transmission, Golay RS terminator, talker alias, DUID 0x3 0xF, TIA-102 BAAA
aka: [TDU, TDULC, "terminator data unit", "terminator with link control"]
autolink: true
infobox:
  - { label: TDU DUID, value: 0x3 (no Link Control) }
  - { label: TDULC DUID, value: 0xF (with Link Control) }
  - { label: FEC (TDULC), value: "Golay(24,12,8) inner + RS(24,12,13)" }
  - { label: Spec, value: TIA-102.BAAA §8 }
see_also: [p25-link-control-word, p25-logical-data-unit, golay-code, p25-reed-solomon, p25-nid-duid, p25-status-symbols]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Binary_Golay_code
---

The **P25 Terminator Data Unit** is the frame that closes a P25 Phase 1 voice transmission.[^wiki]
It comes in two forms distinguished by [DUID](/reference/p25-nid-duid/): the **TDU** (`0x3`) is a
bare terminator — [frame sync](/reference/p25-frame-sync-word/) and NID marking end-of-transmission
with no further payload — while the **TDULC** (`0xF`) also carries a 72-bit
[Link Control word](/reference/p25-link-control-word/), the same LCW an
[LDU1](/reference/p25-logical-data-unit/) carries but behind a different FEC: inner
[Golay(24,12,8)](/reference/golay-code/) codewords under an outer
[RS(24,12,13)](/reference/p25-reed-solomon/) code.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A bare TDU of frame sync plus NID contrasted with a TDULC that adds a link-control region of 24 RS symbols carried by 12 Golay codewords totalling 288 bits; the outer RS(24,12,13) recovers the 72-bit link-control word." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="24" font-size="8.5" fill="currentColor">TDU (0x3)</text>
  <rect x="20" y="30" width="40" height="24" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.1"/><text x="40" y="46" text-anchor="middle" font-size="8" fill="currentColor">FS</text>
  <rect x="60" y="30" width="46" height="24" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.1"/><text x="83" y="46" text-anchor="middle" font-size="8" fill="currentColor">NID</text>
  <text x="240" y="24" font-size="8.5" fill="currentColor">TDULC (0xF)</text>
  <rect x="240" y="30" width="34" height="24" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.1"/><text x="257" y="46" text-anchor="middle" font-size="8" fill="currentColor">FS</text>
  <rect x="274" y="30" width="40" height="24" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.1"/><text x="294" y="46" text-anchor="middle" font-size="8" fill="currentColor">NID</text>
  <rect x="314" y="30" width="126" height="24" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="377" y="46" text-anchor="middle" font-size="8" fill="currentColor">12 × Golay = 288b LC</text>
  <text x="240" y="80" font-size="8" fill="currentColor">12 Golay(24,12,8) codewords → 24 RS symbols → RS(24,12,13) → 72-bit Link Control word</text>
</svg>
<figcaption>A TDU is just frame sync and NID; a TDULC appends a 288-bit link-control region — twelve Golay codewords carrying the 24 symbols of an RS(24,12,13) codeword that recovers the 72-bit Link Control word.</figcaption>
</figure>

## How it works

Both terminators start like any P25 frame, with the frame sync and the BCH-protected
[NID](/reference/p25-nid-duid/) whose DUID identifies the type. The TDU stops there — its purpose
is purely to signal that voice has ended, so a receiver can close the call and release the
channel. The TDULC adds a link-control region immediately after the NID in the
status-stripped payload: 288 bits of twelve [Golay(24,12,8)](/reference/golay-code/) codewords.
Each Golay codeword's 12 data bits carry two 6-bit RS symbols, so the twelve codewords yield the
24 symbols of an outer [RS(24,12,13)](/reference/p25-reed-solomon/) codeword; RS-decoding that
recovers the 12 information symbols — the 72-bit (9-octet) Link Control word. The decode fails
safe: if the layout or FEC does not resolve, the outer RS flags the word uncorrectable rather
than surfacing a garbled Link Control.

## In practice

The TDULC matters because some Link Control content rides *there* rather than in the voice
frames. On Motorola Phase 1 systems, talker-alias Link Control (LCO `0x15` header, `0x17` data)
appears primarily in the terminator, not LDU1 — so a decoder that returns at the terminator
without parsing its Link Control misses the alias entirely. GopherTrunk's TDULC extractor closes
that gap, feeding the recovered nine LC octets into the same carriage-independent talker-alias
buffer the LDU1 path uses.

One caveat is documented honestly in the source: the exact on-air bit **interleaving** of the
TDULC LC region is not reachable from the post-FEC ground-truth dumps the project has, and no IQ
fixture is committed, so GopherTrunk currently reads the twelve Golay codewords sequentially
without a separate deinterleave pass. The RS/Golay math is exact and round-trip tested; if a
capture later reveals an interleave, only that one seam changes, and a wrong layout fails
safe — the outer RS rejects the word rather than emitting junk.

## Relevance to SDR

`internal/radio/p25/phase1/tdulc.go` implements `ExtractTDULCLinkControl`, chaining
`framing.GolayDecode24_12` over the twelve inner codewords into `framing.DecodeRS24_12` for the
outer layer, and returns the LC opcode, the nine content octets, and a corrected-error count.
The bare TDU needs no such parsing — its value is the DUID alone. Together the two terminators
tell a scanner when a call ends and, in the TDULC case, deliver a last Link Control payload
(often the talker's alias) that would otherwise be lost at the tail of the transmission.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard and its Phase 1 data units.
[^golay]: [Binary Golay code](https://en.wikipedia.org/wiki/Binary_Golay_code) — Wikipedia, on the Golay(24,12,8) inner code protecting the TDULC link control.
