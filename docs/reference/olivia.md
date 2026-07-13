---
slug: olivia
title: Olivia MFSK
entry_type: protocol
category: amateur-digital
description: "Olivia is a robust amateur HF keyboard mode that combines multi-tone MFSK with a Walsh/Hadamard forward-error-correction block, decoding text well below the noise floor."
keywords: Olivia, Olivia MFSK, MFSK, Hadamard FEC, Walsh code, HF digital mode, weak-signal text, 8/250, 32/1000, Pawel Jalocha
aka: [Olivia, Olivia MFSK]
autolink: true
infobox:
  - { label: Type, value: HF keyboard text mode }
  - { label: Standards body, value: Amateur convention }
  - { label: Introduced, value: 2003 (Pawel Jalocha) }
  - { label: Access, value: Simplex, one QSO per channel }
  - { label: Channel spacing, value: 125 Hz–2 kHz (configurable) }
  - { label: Modulation, value: MFSK (2–256 tones) + Hadamard FEC }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [m-ary-fsk, rtty, contestia, frequency-shift-keying, forward-error-correction]
cite_urls:
  - https://en.wikipedia.org/wiki/Olivia_MFSK
---

**Olivia** (often written **Olivia MFSK**) is an amateur HF text mode built for the worst
conditions — deep fading, static, and interference — where it can copy full sentences of
error-free text several decibels below the audible noise floor. It sends characters as
one of many tones ([M-ary FSK](/reference/m-ary-fsk/)) wrapped in a strong
[forward-error-correction](/reference/forward-error-correction/) block, trading raw speed
for extraordinary robustness.[^wiki] Where plain [RTTY](/reference/rtty/) collapses in a
flutter, an Olivia link keeps delivering clean text.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Olivia encodes a character into a Hadamard codeword whose bits select one of several MFSK tones spread across the channel, so a lost tone still leaves the codeword decodable." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="olar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="20" y="55" width="70" height="28" fill="currentColor" fill-opacity="0.14" stroke="currentColor"/>
  <text x="55" y="72" font-size="8.5" fill="currentColor" text-anchor="middle">character</text>
  <line x1="90" y1="69" x2="130" y2="69" stroke="currentColor" marker-end="url(#olar)"/>
  <rect x="130" y="55" width="90" height="28" fill="currentColor" fill-opacity="0.22" stroke="currentColor"/>
  <text x="175" y="68" font-size="8" fill="currentColor" text-anchor="middle">Hadamard</text>
  <text x="175" y="78" font-size="8" fill="currentColor" text-anchor="middle">codeword</text>
  <line x1="220" y1="69" x2="260" y2="69" stroke="currentColor" marker-end="url(#olar)"/>
  <g stroke="currentColor" stroke-width="1"><line x1="270" y1="30" x2="270" y2="115"/><line x1="270" y1="115" x2="440" y2="115" marker-end="url(#olar)"/></g>
  <g fill="currentColor"><rect x="285" y="45" width="10" height="6"/><rect x="315" y="60" width="10" height="6"/><rect x="345" y="38" width="10" height="6"/><rect x="375" y="72" width="10" height="6"/><rect x="405" y="52" width="10" height="6"/></g>
  <text x="355" y="130" font-size="8" fill="currentColor" text-anchor="middle">tones across the channel (freq up)</text>
</svg>
<figcaption>Each character becomes a Hadamard codeword; its bits are spread over several MFSK tones, so noise that kills one tone rarely defeats the codeword.</figcaption>
</figure>

## Overview

Olivia is described by two numbers, tones/bandwidth — e.g. **8/250**, **16/500**, or
**32/1000**. The first is the number of MFSK tones, the second the channel width in hertz.
More tones and narrower spacing make the link slower but more sensitive. Each 7-bit ASCII
character is turned into a 64-bit **Walsh/Hadamard** codeword; those bits are interleaved
and scrambled across a block of MFSK symbols so that a burst of noise damages many
codewords slightly rather than any one fatally. The receiver correlates the received
tones against all possible codewords and picks the best match, which is what lets it work
under the noise.

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | MFSK, 2–256 tones |
| FEC | 64-bit Walsh/Hadamard block code |
| Common variants | 8/250, 16/500, 32/1000, 8/500 |
| Throughput | ~1–150 characters/min (mode-dependent) |
| Sensitivity | Copy at roughly −14 dB SNR (2.5 kHz) |
| Carrier | SSB audio, HF |

## Variants

The tones/bandwidth pairs form a family: fast, wider modes for good conditions and slow,
narrow modes for marginal paths. Because so many combinations exist, operators usually
agree on a standard set (32/1000 and 16/500 are common calling modes). A related mode,
[Contestia](/reference/contestia/), was derived directly from Olivia with a smaller
character set and lighter FEC for a bit more speed.

## History

Olivia was designed in 2003 by Pawel Jalocha (SP9VRC), who set out to beat RTTY on poor
HF paths using ideas from spread-spectrum and coding theory. It quickly became a staple of
HF digital chat and weak-signal experimentation, supported by Fldigi and similar
sound-card software.[^wiki]

## Deployment

Olivia is an amateur-only mode used for keyboard-to-keyboard QSOs on HF, with informal
calling frequencies on 20, 30, and 40 metres. It is popular for reliable text exchange
across long, noisy paths where speed matters less than getting the message through.

## Decoding it with GopherTrunk

**GopherTrunk does not decode Olivia.** It is an HF amateur text mode outside GopherTrunk's
land-mobile trunking scope; Fldigi and MultiPSK are the usual decoders. GopherTrunk does
implement the underlying primitives Olivia leans on — [MFSK](/reference/m-ary-fsk/)
tone detection and block [FEC](/reference/forward-error-correction/) — but not this
protocol's framing.

## Sources

[^wiki]: [Olivia MFSK](https://en.wikipedia.org/wiki/Olivia_MFSK) — Wikipedia, for Olivia's MFSK-plus-Hadamard structure, tones/bandwidth variants, sensitivity, and origin.
