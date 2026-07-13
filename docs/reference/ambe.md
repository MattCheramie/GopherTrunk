---
slug: ambe
title: AMBE
entry_type: technology
category: voice-coding
description: AMBE (Advanced Multi-Band Excitation) is a family of low-bitrate speech vocoders from DVSI used across digital voice radio, including D-STAR and as the basis for AMBE+2.
keywords: AMBE, Advanced Multi-Band Excitation, DVSI, D-STAR, vocoder, low bitrate speech, ProVoice, multi-band excitation
aka: [AMBE]
autolink: true
infobox:
  - { label: Type, value: Speech vocoder family (MBE) }
  - { label: Developer, value: DVSI }
  - { label: Used by, value: D-STAR, ProVoice; basis of AMBE+2 }
see_also: [vocoder, imbe, ambe-plus-2, multi-band-excitation, d-star, dvsi, code-excited-linear-prediction, twelp]
related_lessons:
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/rf-sdr/vocoders/ }
related_reading:
  - { title: "SDR Internals, Part 12: Voice coding & vocoders", url: /blog/deep-dives/sdr-internals-12-voice-coding-vocoders/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
  - https://en.wikipedia.org/wiki/D-STAR
---

**AMBE** (**Advanced Multi-Band Excitation**) is a family of low-bitrate speech
[vocoders](/reference/vocoder/) from [DVSI](/reference/dvsi/), building on the
[MBE](/reference/multi-band-excitation/) model.[^wiki] It is used by
[D-STAR](/reference/d-star/) and EDACS ProVoice, and is the direct ancestor of the more
efficient [AMBE+2](/reference/ambe-plus-2/) used by DMR and P25 Phase 2. AMBE improved on
the earlier [IMBE](/reference/imbe/) by delivering comparable quality at lower net bit
rates, which mattered as digital systems pushed toward narrower channels and multi-slot
[TDMA](/reference/tdma/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A stream of small AMBE voice frames each carrying voice bits plus error-correction bits." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2" fill="none"><rect x="40" y="40" width="90" height="32"/><rect x="140" y="40" width="90" height="32"/><rect x="240" y="40" width="90" height="32"/><rect x="340" y="40" width="90" height="32"/></g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="85" y="60">voice+FEC</text><text x="185" y="60">voice+FEC</text><text x="285" y="60">voice+FEC</text><text x="385" y="60">voice+FEC</text></g>
  <text x="230" y="24" text-anchor="middle" font-size="9" fill="currentColor">compact frames, ~20 ms each</text>
</svg>
<figcaption>AMBE is the multi-band excitation vocoder family behind many digital-voice systems.</figcaption>
</figure>

## How it works

Like every [MBE](/reference/multi-band-excitation/) coder, AMBE decomposes each 20 ms
frame into a **pitch estimate**, a set of **harmonic spectral magnitudes**, and a
**voiced/unvoiced decision per band**. The decoder rebuilds voiced bands as a sum of pitch
harmonics and unvoiced bands as shaped noise. What distinguishes AMBE from the older IMBE
is a more efficient parameter representation — vector-quantized spectral magnitudes and
smarter bit allocation — that reaches similar intelligibility with fewer bits, plus
integrated [forward error correction](/reference/forward-error-correction/) that DVSI tuned
to the frame's perceptual sensitivity.

D-STAR's DV mode, for example, carries a **3600 bps** AMBE stream (2400 bps voice + 1200
bps FEC) alongside a low-rate data channel inside its 6.25 kHz-equivalent bandwidth. The
exact bit split and FEC scheme vary by system, but the analysis/synthesis core is the same
MBE machinery.

## Variants

AMBE spans several generations. The original **AMBE** chips (AMBE-1000/2000) served
D-STAR, EDACS/ProVoice, Iridium satphones, and Inmarsat. The refined
**[AMBE+2](/reference/ambe-plus-2/)** (AMBE-3000 series) added better quality at low rate
and explicit half-rate modes for [DMR](/reference/dmr/), [NXDN](/reference/nxdn/), and P25
Phase 2. All are proprietary to DVSI and share the MBE analysis model but are not
bit-compatible with one another or with IMBE.

## In practice

Because AMBE is patent-encumbered, commercial radios embedded DVSI's DSP chips and early
computer decoders needed a hardware "dongle." Independent software reimplementations exist
and are what let a general-purpose SDR render AMBE voice today. The trade AMBE makes —
proprietary, licensed, but very efficient — is exactly what motivated the open
[Codec 2](/reference/codec2/) alternative adopted by [M17](/reference/m17/).

## Relevance to SDR

GopherTrunk implements AMBE-family decoding in pure Go to render clear digital voice
without proprietary hardware. It applies where D-STAR and ProVoice appear, and the same
codebase covers the closely related [AMBE+2](/reference/ambe-plus-2/) used by the mainstream
DMR/NXDN/P25 Phase 2 systems. As always, encryption sits above the vocoder: GopherTrunk can
follow an encrypted AMBE call but cannot resynthesize its scrambled frames without the key.

## Sources

[^wiki]: [Multi-Band Excitation](https://en.wikipedia.org/wiki/Multi-Band_Excitation) — Wikipedia, on the MBE vocoder family that includes AMBE.
[^dstar]: [D-STAR](https://en.wikipedia.org/wiki/D-STAR) — Wikipedia, for AMBE as the D-STAR DV-mode voice codec at 3600 bps.
