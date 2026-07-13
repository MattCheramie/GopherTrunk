---
slug: ambe-plus-2
title: AMBE+2
entry_type: technology
category: voice-coding
description: AMBE+2 is an efficient successor to IMBE/AMBE from DVSI, used by P25 Phase 2, DMR, and NXDN, and supporting half-rate operation for higher capacity.
keywords: AMBE+2, AMBE plus 2, DVSI, DMR, P25 Phase 2, NXDN, half-rate vocoder, AMBE-3000, multi-band excitation
aka: [AMBE+2]
autolink: true
infobox:
  - { label: Type, value: Speech vocoder (MBE family) }
  - { label: Developer, value: DVSI }
  - { label: Used by, value: P25 Phase 2, DMR, NXDN }
  - { label: Feature, value: Half-rate operation }
see_also: [vocoder, imbe, ambe, dmr, p25-phase-2, nxdn, dvsi, multi-band-excitation, tdma]
related_lessons:
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/rf-sdr/vocoders/ }
related_reading:
  - { title: "SDR Internals, Part 12: Voice coding & vocoders", url: /blog/deep-dives/sdr-internals-12-voice-coding-vocoders/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
---

**AMBE+2** is the efficient successor to [IMBE](/reference/imbe/) and
[AMBE](/reference/ambe/) from [DVSI](/reference/dvsi/).[^wiki] It powers
[P25 Phase 2](/reference/p25-phase-2/), [DMR](/reference/dmr/), and
[NXDN](/reference/nxdn/), and supports **half-rate** operation — the feature that lets
two conversations share the airspace one analog channel used to occupy.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A 12.5 kHz channel split into two TDMA slots, each carrying a half-rate AMBE+2 voice stream." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="40" width="380" height="40" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <line x1="230" y1="40" x2="230" y2="80" stroke="currentColor" stroke-width="1.2"/>
  <text x="135" y="64" text-anchor="middle" font-size="9" fill="currentColor">slot 1 · AMBE+2</text>
  <text x="325" y="64" text-anchor="middle" font-size="9" fill="currentColor">slot 2 · AMBE+2</text>
  <text x="230" y="28" text-anchor="middle" font-size="9" fill="currentColor">one 12.5 kHz channel</text>
  <text x="230" y="100" text-anchor="middle" font-size="9" fill="currentColor">half-rate coding fits two voices where one used to go</text>
</svg>
<figcaption>AMBE+2 is a more efficient successor used by P25 Phase 2, DMR, and NXDN, supporting half-rate streams.</figcaption>
</figure>

## How it works

AMBE+2 keeps the [Multi-Band Excitation](/reference/multi-band-excitation/) analysis core —
pitch, per-band voicing, harmonic magnitudes on a 20 ms frame — but refines the parameter
quantization and [forward error correction](/reference/forward-error-correction/) so that
audio quality at a given bit rate is noticeably better than IMBE, or equivalently that the
same quality survives fewer bits. That efficiency is what enables **half-rate** modes:
DMR's per-slot vocoder runs about **2450 bps of voice + 1150 bps FEC (3600 bps total)**,
and P25 Phase 2 uses a **7200×4400 half-rate** configuration. Two such streams interleave
in the two [TDMA](/reference/tdma/) slots of one 12.5 kHz channel, doubling call capacity
without new spectrum.

## Variants

AMBE+2 is configurable: DVSI ships it as a family of "rate tables" so a system designer
picks a total bit rate and FEC split to fit the channel. DMR, NXDN, and P25 Phase 2 each
specify a different configuration, so their frames are not directly interchangeable even
though they share the same codec engine (the AMBE-3000 chip). It is the third generation of
the MBE line — [IMBE](/reference/imbe/) → [AMBE](/reference/ambe/) → AMBE+2 — and, like its
predecessors, is not bit-compatible with them.

## In practice

AMBE+2 is proprietary and license-gated; radios embed DVSI silicon and early PC decoders
required a USB dongle. The core patents underpinning independent software decoding have
matured enough for open reimplementations, which is what a modern SDR relies on. Where an
open, royalty-free path is required — as in [M17](/reference/m17/) — designers reach for
[Codec 2](/reference/codec2/) instead of AMBE+2.

## Relevance to SDR

GopherTrunk runs AMBE+2 decoding to produce audio from the most common modern digital
voice systems: [DMR](/reference/dmr/) (Tier II/III), [NXDN](/reference/nxdn/), and
[P25 Phase 2](/reference/p25-phase-2/). It parses the per-slot frames, unwinds the FEC, and
synthesizes voice in pure Go. Encrypted traffic (for example DMR's RC4 Enhanced Privacy or
P25 AES) still produces AMBE+2 frames, but their payload is scrambled and cannot be
resynthesized to intelligible audio without the key.

## Sources

[^wiki]: [Multi-Band Excitation](https://en.wikipedia.org/wiki/Multi-Band_Excitation) — Wikipedia, on the MBE vocoder family that includes AMBE+2.
[^dmr]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, for the AMBE+2 half-rate vocoder in DMR's two-slot TDMA channel.
