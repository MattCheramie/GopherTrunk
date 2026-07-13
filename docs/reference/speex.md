---
slug: speex
title: Speex
entry_type: technology
category: voice-coding
description: "Speex is an open, patent-free CELP speech codec for VoIP and low-bitrate voice, now largely superseded by Opus."
keywords: Speex, CELP codec, open source voice codec, VoIP, Xiph, patent-free, narrowband wideband, Opus predecessor
aka: [Speex]
autolink: true
infobox:
  - { label: Type, value: Open CELP speech codec }
  - { label: Idea, value: Royalty-free VoIP voice coding }
  - { label: Status, value: Deprecated, superseded by Opus }
see_also: [opus-codec, code-excited-linear-prediction, acelp, vocoder, linear-predictive-coding, g729]
cite_urls:
  - https://en.wikipedia.org/wiki/Speex
  - https://www.speex.org/
---

**Speex** is an open, patent-free speech codec built on
[code-excited linear prediction](/reference/code-excited-linear-prediction/)
(CELP) and designed for voice over IP and other low-bitrate speech
applications.[^wiki] Developed under the Xiph.Org Foundation and released
royalty-free, it aimed to give VoIP software a high-quality voice coder without
the licensing burden of contemporaries like [G.729](/reference/g729/). Speex is
now formally deprecated: Xiph declared it obsolete and steers new projects to
[Opus](/reference/opus-codec/), which subsumes and outperforms it across the
whole bitrate range.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="Speex encodes narrowband, wideband and ultra-wideband speech into a single embedded CELP bitstream whose lower layers can be stripped to reduce rate." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="spxar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="24" y="30" width="86" height="24" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="67" y="45">8 kHz narrowband</text>
    <rect x="24" y="58" width="86" height="24" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="67" y="73">16 kHz wideband</text>
    <rect x="24" y="86" width="86" height="24" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="67" y="101">32 kHz UWB</text>
  </g>
  <line x1="112" y1="70" x2="168" y2="70" stroke="currentColor" stroke-width="1.2" marker-end="url(#spxar)"/>
  <rect x="172" y="52" width="96" height="36" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
  <text x="220" y="74" text-anchor="middle" font-size="9" fill="currentColor">CELP encoder</text>
  <line x1="270" y1="70" x2="322" y2="70" stroke="currentColor" stroke-width="1.2" marker-end="url(#spxar)"/>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="326" y="46" width="110" height="20" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="381" y="60">core layer (2 kbps)</text>
    <rect x="326" y="70" width="110" height="20" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="381" y="84">enhancement layers</text>
  </g>
  <text x="230" y="122" text-anchor="middle" font-size="8" fill="currentColor">embedded bitstream — strip upper layers to lower the rate</text>
</svg>
<figcaption>Speex: a scalable, royalty-free CELP coder spanning narrowband to ultra-wideband speech.</figcaption>
</figure>

## How it works

Speex is a classic linear-prediction speech coder. Each frame it fits a
short-term [linear-prediction](/reference/linear-predictive-coding/) filter that
models the vocal-tract spectrum, then searches an adaptive codebook (the pitch
predictor) and a fixed codebook for an excitation signal that, run through that
filter, best reproduces the original waveform — the analysis-by-synthesis loop
at the heart of every [CELP](/reference/code-excited-linear-prediction/) design.
Only the filter coefficients, pitch lag and codebook indices are transmitted, so
a few kilobits per second reconstruct intelligible speech.

Several features distinguished Speex from single-rate telephony coders:

- **Multiple sample rates.** The same design handles narrowband (8 kHz),
  wideband (16 kHz) and ultra-wideband (32 kHz) audio, so it covers ordinary
  telephone voice up through fuller-sounding conferencing.
- **Variable and embedded bitrate.** Rates run from roughly 2 to 44 kbps, with
  an embedded-coding mode whose stream can be truncated to a lower rate without
  re-encoding, and a voice-activity/discontinuous mode that drops the rate during
  silence.
- **VoIP-oriented robustness.** It includes perceptual noise shaping, echo and
  noise suppression helpers, and packet-loss concealment aimed at lossy IP
  networks rather than clean circuit-switched links.

Because it was engineered from the outset with only public-domain techniques and
released under a BSD-style licence, Speex could be embedded freely — the same
motivation that later produced [Opus](/reference/opus-codec/).

## Relevance to SDR

Speex belongs to the internet-telephony lineage rather than the land-mobile
radio world. The vocoders inside trunked and digital voice systems that
GopherTrunk targets are the multi-band-excitation family — IMBE and
[AMBE+2](/reference/ambe-plus-2/) in P25, DMR and NXDN — not CELP coders like
Speex. Where Speex does surface in radio work is on the transport side: SDR and
digital-voice projects (EchoLink-style linking, ham VoIP gateways, some
allstar/reflector networks) have used Speex to carry audio between nodes over IP,
and it appears in older WebRTC and softphone stacks. For new work its role is
essentially historical, since [Opus](/reference/opus-codec/) — which merged the
CELP ideas behind Speex with Xiph's CELT transform coder — now covers the same
ground with better quality and lower delay. GopherTrunk does not use Speex in its
decode chain.

## Sources

[^wiki]: [Speex](https://en.wikipedia.org/wiki/Speex) — Wikipedia, on the open patent-free CELP speech codec and its deprecation in favour of Opus.
