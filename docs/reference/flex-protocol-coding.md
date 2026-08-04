---
slug: flex-protocol-coding
title: FLEX protocol coding
entry_type: algorithm
category: paging-data
description: FLEX protocol coding is the framing and forward-error-correction of Motorola's FLEX paging — a 4-level FSK stream synced on 0xA6C6AAAA, organised into 88-word phases, de-interleaved and protected by a BCH(31,21) code laid out bit-reversed relative to POCSAG.
keywords: FLEX paging, FLEX coding, 0xA6C6AAAA, FLEX sync marker, FIW BIW VIW, FLEX de-interleave, BCH 31 21 bit-reversed, 4-level FSK paging, ARIB STD-T44
aka: [FLEX coding, "FLEX framing", "Motorola FLEX"]
autolink: true
infobox:
  - { label: Modulation, value: "2/4-level FSK, 1600–6400 bps" }
  - { label: Sync marker, value: "0xA6C6AAAA" }
  - { label: Phase, value: 88 words per phase }
  - { label: FEC, value: "BCH(31,21), bit-reversed" }
see_also: [flex, pocsag-codeword, bch-code, four-fsk, forward-error-correction, interleaving, frame-synchronization]
cite_urls:
  - https://en.wikipedia.org/wiki/FLEX_(protocol)
  - https://en.wikipedia.org/wiki/BCH_code
---

**FLEX protocol coding** is the framing and forward-error-correction of Motorola's
[FLEX](/reference/flex/) paging — the high-throughput successor to
[POCSAG](/reference/pocsag/) that multiplexes many pagers onto one channel with time-framed,
interleaved, [BCH](/reference/bch-code/)-protected data.[^flex] A receiver locks the 32-bit
sync marker `0xA6C6AAAA`, reads a mode code, reads a frame-info word, then de-interleaves an
88-codeword *phase* and BCH-decodes each word before any address or message can be read.[^bch]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="A FLEX frame shown as a sync marker followed by a 16-bit mode code, a 32-bit frame information word and an 88-codeword data phase, with a callout showing that the FLEX BCH codeword places its information bits in the low positions, the bit-reverse of the POCSAG layout." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="24" width="90" height="24" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.1"/>
  <text x="65" y="40" text-anchor="middle" font-size="8" fill="currentColor">sync A6C6AAAA</text>
  <rect x="110" y="24" width="60" height="24" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/>
  <text x="140" y="40" text-anchor="middle" font-size="8" fill="currentColor">mode 16b</text>
  <rect x="170" y="24" width="60" height="24" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/>
  <text x="200" y="40" text-anchor="middle" font-size="8" fill="currentColor">FIW 32b</text>
  <rect x="230" y="24" width="210" height="24" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="335" y="40" text-anchor="middle" font-size="8" fill="currentColor">phase · 88 words (2816 bits)</text>
  <text x="20" y="80" font-size="8" fill="currentColor">FLEX codeword:</text>
  <rect x="100" y="70" width="120" height="18" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1"/>
  <text x="160" y="83" text-anchor="middle" font-size="7.5" fill="currentColor">21 info (bits 0..20)</text>
  <rect x="220" y="70" width="90" height="18" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1"/>
  <text x="265" y="83" text-anchor="middle" font-size="7.5" fill="currentColor">10 parity</text>
  <rect x="310" y="70" width="30" height="18" fill="currentColor" fill-opacity="0.24" stroke="currentColor" stroke-width="1"/>
  <text x="325" y="83" text-anchor="middle" font-size="7" fill="currentColor">par</text>
  <text x="20" y="112" font-size="7.5" fill="currentColor">info in the LOW bits — the bit-reverse of POCSAG, which puts info in the high bits</text>
  <text x="20" y="128" font-size="7.5" fill="currentColor">bit-reversal preserves Hamming distance, so the same BCH(31,21) primitive decodes both</text>
</svg>
<figcaption>FLEX syncs on a fixed marker, reads a mode code and frame-info word, then de-interleaves an 88-word phase; each word is the POCSAG BCH(31,21) code with its bits reversed so the information lands in the low positions.</figcaption>
</figure>

## How it works

GopherTrunk's `Decoder` is a four-state machine driven one wire bit at a time. In
**hunt-marker** it slides a 32-bit window for `0xA6C6AAAA`; like the paging protocols
generally, it accepts the marker's bit-inverse too and remembers whether it locked on the
complemented sense. It then reads a 16-bit **mode code** (the speed/level word — `0x870C`
marks the 1600 bps two-level mode GopherTrunk decodes; other modes are recognised and cleanly
skipped), and a 32-bit **frame-info word** from which it extracts the cycle number
`(fiw>>4)&0x0F` and frame number `(fiw>>8)&0x7F`. Finally it collects the phase: 88 codewords
of 32 bits each, `PhaseBits = 2816` in total.

The phase is **block-interleaved**, so bits belonging to one codeword are scattered across the
2816-bit span to spread a fading burst into isolated single errors. The de-interleave maps the
running bit counter `c` to codeword index `((c>>5)&0xFFF8)|(c&7)` and bit position
`(c>>3)&31` within that word — reversing the standard's permutation to reassemble each 32-bit
codeword before it is decoded.

## Frame and phase structure

Once the phase is BCH-decoded, its 88 information words are walked as a small self-describing
structure:

| Word | Role | Fields GopherTrunk reads |
| --- | --- | --- |
| word 0 | Block Information Word (BIW) | `voffset = (biw>>10)&0x3F`, `aoffset = ((biw>>8)&0x03)+1` |
| words `[aoffset, voffset)` | address words | short capcode = word − 0x8000 |
| word `voffset + (i−aoffset)` | Vector Information Word (VIW) | type `(viw>>4)&0x07`, first message word `(viw>>7)&0x7F`, length `(viw>>14)&0x7F` |
| message-word span | message body | 7-bit alphanumeric or 4-bit numeric symbols |

Each address's vector word names where its message words live, so the decoder can jump
straight to a page's text. Idle address slots (all-zero or all-ones 21-bit words) are skipped.

## The bit-reversed BCH

FLEX uses the *same* BCH(31,21) code as POCSAG — generator `0x769`, correcting up to two bit
errors, plus an overall even-parity bit — but lays the codeword out the other way round. FLEX
puts the 21 information bits in the **low** positions (0..20) with the 10 parity bits above
them (21..30) and the even-parity bit at bit 31; POCSAG puts information in the high bits.
Because a bit-reversal of a codeword preserves Hamming distance, GopherTrunk reuses its tested
POCSAG primitive by reversing 31 bits into and out of it (`FLEXBCHDecode32`) rather than
maintaining a second BCH decoder. The package notes that the exact on-wire bit order is
reproduced from the de-facto reference decoders (multimon-ng, ARIB STD-T44) and should be
confirmed against a real capture — a best-effort calibration caveat the code carries
explicitly.

## Sources

[^flex]: [FLEX (protocol)](https://en.wikipedia.org/wiki/FLEX_(protocol)) — Wikipedia, on Motorola's FLEX paging protocol, its FSK modes and frame structure.
[^bch]: [BCH code](https://en.wikipedia.org/wiki/BCH_code) — Wikipedia, on the error-correcting code shared by FLEX and POCSAG.
