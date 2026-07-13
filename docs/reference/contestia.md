---
slug: contestia
title: Contestia
entry_type: protocol
category: amateur-digital
description: "Contestia is an amateur HF keyboard mode derived from Olivia, using multi-tone MFSK with a lighter Hadamard code and a reduced character set to run faster while staying robust."
keywords: Contestia, Olivia derivative, MFSK, Hadamard FEC, HF digital mode, 4/250, 8/500, weak-signal text, amateur radio
aka: [Contestia]
autolink: true
infobox:
  - { label: Type, value: HF keyboard text mode }
  - { label: Standards body, value: Amateur convention }
  - { label: Introduced, value: 2005 (Nick Fedoseev, UT2UZ) }
  - { label: Access, value: Simplex, one QSO per channel }
  - { label: Channel spacing, value: 125 Hz–2 kHz (configurable) }
  - { label: Modulation, value: MFSK (2–64 tones) + Hadamard FEC }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [olivia, m-ary-fsk, rtty, frequency-shift-keying, forward-error-correction]
cite_urls:
  - https://en.wikipedia.org/wiki/Contestia
---

**Contestia** is an amateur HF text mode created as a lighter, faster relative of
[Olivia](/reference/olivia/). It keeps Olivia's core idea — characters sent as one of
several tones ([M-ary FSK](/reference/m-ary-fsk/)) protected by a
[forward-error-correction](/reference/forward-error-correction/) block — but shrinks the
character set and thins the coding so it moves text about twice as fast while still copying
well under noise.[^wiki] The result sits between raw [RTTY](/reference/rtty/) speed and
Olivia-grade robustness.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="Contestia trades Olivia's heavier coding and full character set for a lighter code and reduced alphabet, moving the operating point toward higher speed at slightly lower sensitivity." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="ctar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="40" y1="90" x2="440" y2="90" stroke="currentColor" marker-end="url(#ctar)"/>
  <text x="240" y="108" font-size="8.5" fill="currentColor" text-anchor="middle">speed →</text>
  <line x1="60" y1="95" x2="60" y2="20" stroke="currentColor" marker-end="url(#ctar)"/>
  <text x="52" y="30" font-size="8" fill="currentColor" text-anchor="end">robustness</text>
  <circle cx="140" cy="45" r="6" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
  <text x="140" y="35" font-size="8.5" fill="currentColor" text-anchor="middle">Olivia</text>
  <circle cx="270" cy="62" r="6" fill="currentColor" stroke="currentColor"/>
  <text x="270" y="52" font-size="8.5" fill="currentColor" text-anchor="middle">Contestia</text>
  <circle cx="380" cy="78" r="6" fill="none" stroke="currentColor"/>
  <text x="380" y="68" font-size="8.5" fill="currentColor" text-anchor="middle">RTTY</text>
</svg>
<figcaption>Contestia sits between Olivia and RTTY: lighter coding and a smaller alphabet buy roughly double the speed at a modest cost in sensitivity.</figcaption>
</figure>

## Overview

Like Olivia, Contestia is named by a tones/bandwidth pair — for example **4/250**,
**8/250**, **8/500**, or **16/500**. It uses the same MFSK signalling and a
**Walsh/Hadamard** block code, but encodes only a 6-bit character set (upper-case letters,
digits, and common punctuation) instead of full 7-bit ASCII, and applies a shorter code.
Fewer bits per character and less coding overhead mean each character occupies fewer MFSK
symbols, so text scrolls roughly twice as fast as the equivalent Olivia mode. The receiver
still correlates incoming tones against the codeword set, preserving good weak-signal
behaviour.

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | MFSK, 2–64 tones |
| FEC | Walsh/Hadamard block code (lighter than Olivia) |
| Character set | Reduced 6-bit (no lower case) |
| Common variants | 4/250, 8/250, 8/500, 16/500 |
| Throughput | Roughly 2× the matching Olivia mode |
| Carrier | SSB audio, HF |

## History

Contestia was defined in 2005 by Nick Fedoseev (UT2UZ), together with Con Wassilieff
(ZL2AFP), specifically to keep much of Olivia's noise immunity while being brisk enough for
casual contacts and contest-style exchanges — hence the name.[^wiki] It is supported by the
same sound-card software that handles Olivia, most notably Fldigi.

## Deployment

Contestia is amateur-only, used for keyboard QSOs on HF where operators want more speed than
Olivia gives but more reliability than RTTY. It shares the general HF digital sub-bands and
has no formal calling frequency, so operators coordinate the exact tones/bandwidth by
agreement or by decoding the ongoing signal.

## Decoding it with GopherTrunk

**GopherTrunk does not decode Contestia.** As an HF amateur text mode it falls outside
GopherTrunk's trunking focus and is handled by Fldigi or MultiPSK. The primitives it relies
on — [MFSK](/reference/m-ary-fsk/) tone detection and block
[FEC](/reference/forward-error-correction/) — exist in GopherTrunk's DSP toolbox, but the
Contestia framing itself is not implemented.

## Sources

[^wiki]: [Contestia](https://en.wikipedia.org/wiki/Contestia) — Wikipedia, for Contestia's derivation from Olivia, its reduced character set and lighter FEC, variants, and authors.
