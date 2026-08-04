---
slug: ambe-plus-2-fec
title: AMBE+2 on-air FEC
entry_type: algorithm
category: voice-coding
description: The AMBE+2 on-air FEC is the channel coding wrapped around each 49-bit DMR voice payload — a bit deinterleave, a C0..C3 sub-vector split, Golay(23,12) on C0 and C1, and a C0-seeded PRNG that de-whitens C1.
keywords: AMBE+2 FEC, DMR voice FEC, ambe3600x2450, C0 C1 C2 C3, Golay 23 12 voice, AMBE deinterleave, mbelib, voice channel coding
aka: ["AMBE+2 channel coding", "3600x2450 FEC", "AMBE voice FEC"]
autolink: true
infobox:
  - { label: On-air frame, value: 72 bits → 49 info bits }
  - { label: Sub-vectors, value: "C0:24 C1:23 C2:11 C3:14" }
  - { label: ECC, value: Golay(23,12) on C0 & C1 }
  - { label: De-whitening, value: C0-seeded PRNG on C1 }
see_also: [ambe-plus-2, golay-code, dmr-voice-superframe, vocoder, forward-error-correction, interleaving]
cite_urls:
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
  - https://en.wikipedia.org/wiki/Binary_Golay_code
---

The **AMBE+2 on-air FEC** is the forward-error-correction envelope DMR wraps around each
[AMBE+2](/reference/ambe-plus-2/) voice frame before it goes on the air.[^mbe] It is
*channel coding*, not the vocoder itself: the [vocoder](/reference/vocoder/) turns speech
into a 49-bit parameter payload, and this layer protects those bits over a noisy RF path by
deinterleaving, error-correcting, and de-whitening them. Each 72-bit on-air frame in DMR's
"3600×2450" variant carries 49 information bits plus the redundancy that lets a receiver
recover them a few dB into the noise.[^golay]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="A 72-bit on-air AMBE frame is deinterleaved into four sub-vectors C0, C1, C2 and C3; C0 and C1 pass through Golay(23,12) decoders, the recovered C0 word seeds a pseudo-random keystream that de-whitens C1, and the corrected data bits plus the raw C2 and C3 bits assemble into the 49-bit payload." xmlns="http://www.w3.org/2000/svg">
  <rect x="18" y="16" width="150" height="22" rx="3" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="93" y="31" text-anchor="middle" font-size="8.5" fill="currentColor">72-bit on-air frame</text>
  <path d="M93 38 L93 50" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <text x="150" y="47" font-size="7.5" fill="currentColor">deinterleave rW/rX/rY/rZ</text>
  <g stroke="currentColor" stroke-width="1.1" font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="18" y="54" width="70" height="24" fill="currentColor" fill-opacity="0.22"/><text x="53" y="70">C0 · 24</text>
    <rect x="94" y="54" width="70" height="24" fill="currentColor" fill-opacity="0.22"/><text x="129" y="70">C1 · 23</text>
    <rect x="170" y="54" width="60" height="24" fill="none"/><text x="200" y="70">C2 · 11</text>
    <rect x="236" y="54" width="60" height="24" fill="none"/><text x="266" y="70">C3 · 14</text>
  </g>
  <text x="53" y="94" text-anchor="middle" font-size="7.5" fill="currentColor">Golay(23,12)</text>
  <text x="129" y="94" text-anchor="middle" font-size="7.5" fill="currentColor">Golay(23,12)</text>
  <path d="M53 100 L129 108" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <text x="60" y="120" font-size="7.5" fill="currentColor">C0 seeds PRNG → de-whitens C1</text>
  <rect x="18" y="126" width="278" height="18" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="157" y="139" text-anchor="middle" font-size="8" fill="currentColor">49-bit vocoder payload (C0:12 + C1:12 + C2:11 + C3:14)</text>
</svg>
<figcaption>The on-air frame is deinterleaved into four sub-vectors; C0 and C1 are Golay-corrected, the recovered C0 word de-whitens C1, and the result assembles into the 49-bit payload the vocoder decodes.</figcaption>
</figure>

## Deinterleave and split

The 72 on-air bits arrive as 36 dibits that DMR has interleaved so a burst of channel errors
spreads across sub-vectors rather than wrecking one. GopherTrunk undoes that with the fixed
`rW`/`rX`/`rY`/`rZ` schedule ported verbatim from the szechyjs/dsd reference: for dibit *i*,
the high bit is written to `ambe_fr[rW[i]][rX[i]]` and the low bit to `ambe_fr[rY[i]][rZ[i]]`.
The result is four sub-vectors of unequal importance — **C0** (24 bits), **C1** (23), **C2**
(11), and **C3** (14) — ordered by how much each contributes to the decoded speech, so the
strongest protection lands on the most perceptually critical bits.

## Golay correction and de-whitening

C0 and C1 each carry a [Golay(23,12)](/reference/golay-code/) codeword: 12 data bits plus 11
parity bits, correcting up to three bit errors. C0 is decoded first because it does double
duty. Its recovered 12-bit data word seeds a small pseudo-random generator whose output is a
**whitening keystream** XORed onto C1 by the transmitter; the receiver must regenerate the
same sequence to recover C1, and it can only do so once C0 has been cleaned. The generator is
a linear-congruential recurrence seeded from C0:

```go
pr[0] = 16 * uint32(c0data&0x0FFF)
for i := 1; i < 24; i++ {
    pr[i] = (173*pr[i-1] + 13849) & 0xFFFF
}
// keystream bit i = pr[i] >> 15  (the top bit of each 16-bit state)
```

Only entries 1..23 are used — one keystream bit per C1 codeword bit — and the top bit of each
16-bit state supplies the bit. GopherTrunk XORs `ks[23-j]` onto C1 bit *j*, then runs the
second Golay decode. The remaining sub-vectors, C2 and C3, are the least significant bits and
carry no ECC of their own; they are copied straight through. The 49-bit payload is finally
assembled as C0(12) + C1(12) + C2(11) + C3(14) and handed to the vocoder that reconstructs a
[voice superframe](/reference/dmr-voice-superframe/).

## Why this layer is separate

Keeping the channel coding distinct from the vocoder algorithm matters when a call decodes to
garble. The vocoder can be bit-exact against its reference and still produce noise if this FEC
layer is wrong — a swapped deinterleave index, an off-by-one in the keystream tap, or decoding
C1 before C0 all corrupt the payload silently, and a synthetic round-trip test that encodes
and decodes with the same mistake will still pass. The de-whitening step is the subtlest: it
couples C1 to C0, so a single uncorrected C0 error re-randomises the entire C1 sub-vector
rather than flipping one bit. When "voice doesn't decode" but the vocoder unit tests are green,
this envelope — deinterleave, Golay, keystream — is where to look first.

## Relevance to SDR

`internal/radio/dmr/voice/ambefec.go` implements the whole path: the `rW`/`rX`/`rY`/`rZ`
tables, a Golay(23,12) syndrome decoder built at init from the mbelib generator, the
`c1Keystream` recurrence above, and `DecodeAMBEFrame`, which returns the 49-bit payload plus
the count of Golay errors it corrected across C0 and C1 — a per-frame quality signal the
receiver can trend to tell a clean call from one barely holding. `EncodeAMBEFrame` is the
inverse, present so the FEC chain can be exercised by round-trip and deliberate bit-error
tests independent of the vocoder.

## Sources

[^mbe]: [Multi-Band Excitation](https://en.wikipedia.org/wiki/Multi-Band_Excitation) — Wikipedia, on the AMBE family of voice codecs used by DMR.
[^golay]: [Binary Golay code](https://en.wikipedia.org/wiki/Binary_Golay_code) — Wikipedia, on the (23,12) perfect code that corrects the C0 and C1 sub-vectors.
