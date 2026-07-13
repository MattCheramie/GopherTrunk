---
slug: silk-codec
title: SILK
entry_type: technology
category: voice-coding
description: "SILK is Skype's linear-predictive speech codec, now the low-bitrate speech layer inside the Opus codec."
keywords: SILK, Skype codec, LPC speech codec, Opus, wideband voice, Xiph, IETF, variable bitrate
aka: [SILK]
autolink: true
infobox:
  - { label: Type, value: LPC speech codec }
  - { label: Idea, value: Adaptive-rate wideband voice }
  - { label: Now, value: Speech layer of Opus }
see_also: [opus-codec, linear-predictive-coding, code-excited-linear-prediction, speex, vocoder, g729]
cite_urls:
  - https://en.wikipedia.org/wiki/SILK
  - https://datatracker.ietf.org/doc/html/rfc6716
---

**SILK** is a speech codec developed by Skype and built around
[linear predictive coding](/reference/linear-predictive-coding/), designed to
carry conversational voice over the open internet at adaptive bitrates.[^wiki] It
was introduced to replace Skype's earlier SVOPC coder, then contributed to the
IETF and folded into [Opus](/reference/opus-codec/), where it forms the
low-bitrate speech layer. SILK targets a wide range of network conditions,
scaling sample rate, bitrate and complexity to whatever the link and endpoint can
sustain.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="SILK forms the speech layer of Opus, joined by a CELT transform layer and a hybrid mode that blends the two across the audio bandwidth." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="34" width="420" height="70" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="230" y="26" text-anchor="middle" font-size="9" fill="currentColor">Opus</text>
  <rect x="34" y="52" width="120" height="38" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
  <text x="94" y="68" text-anchor="middle" font-size="9" fill="currentColor">SILK</text>
  <text x="94" y="80" text-anchor="middle" font-size="7" fill="currentColor">LPC speech</text>
  <rect x="170" y="52" width="120" height="38" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="230" y="68" text-anchor="middle" font-size="9" fill="currentColor">Hybrid</text>
  <text x="230" y="80" text-anchor="middle" font-size="7" fill="currentColor">SILK + CELT</text>
  <rect x="306" y="52" width="120" height="38" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="366" y="68" text-anchor="middle" font-size="9" fill="currentColor">CELT</text>
  <text x="366" y="80" text-anchor="middle" font-size="7" fill="currentColor">transform / music</text>
  <text x="230" y="120" text-anchor="middle" font-size="8" fill="currentColor">low rate / speech &#8594; high rate / full-band audio</text>
</svg>
<figcaption>SILK is the linear-prediction speech engine that Opus uses at low rates and in its hybrid mode.</figcaption>
</figure>

## How it works

SILK is a linear-prediction coder, but it departs from the classic
[CELP](/reference/code-excited-linear-prediction/) analysis-by-synthesis loop.
It estimates a short-term
[LPC](/reference/linear-predictive-coding/) filter for the vocal-tract spectrum
and a long-term predictor for pitch, then quantises and transmits the *residual*
excitation with noise-shaping quantisation rather than searching a fixed codebook
in a closed loop. This open-loop style keeps complexity down while letting the
encoder shape quantisation noise under the perceptual masking curve.

Its defining trait is adaptivity:

- **Variable sample rate.** SILK works in narrowband, mediumband, wideband and
  super-wideband modes (8 to 24 kHz sampled audio), switching as the endpoint and
  network allow.
- **Variable bitrate.** Rates roughly span 6 to 40 kbps, adjusting continuously to
  packet loss, jitter and congestion signalled back from the far end.
- **Loss resilience.** In-band forward error correction and packet-loss
  concealment let a decoder mask dropped packets, important for the best-effort IP
  paths it was built for.

When Skype donated SILK to the IETF, it was combined with Xiph's CELT transform
coder to create Opus (RFC 6716). Inside Opus, SILK handles speech at low bitrates,
CELT handles music and high bitrates, and a hybrid mode runs SILK on the lower
band with CELT extending the top — so a single Opus stream can move smoothly from
a 6 kbps voice call to full-band stereo music.

## Relevance to SDR

SILK is an internet-voice codec, not a land-mobile radio vocoder, so it does not
appear inside P25, DMR, NXDN or TETRA — those use IMBE and
[AMBE+2](/reference/ambe-plus-2/) instead. Its relevance to SDR is indirect:
whenever a software-radio or digital-voice project streams received audio over IP
using Opus — reflectors, remote-receiver front ends, WebRTC-based SDR web
receivers — SILK is doing the speech coding underneath at conversational bitrates.
It sits alongside [Speex](/reference/speex/) as part of the open-codec lineage
that culminated in [Opus](/reference/opus-codec/). GopherTrunk does not use SILK
in its decode chain; it decodes the proprietary radio vocoders directly.

## Sources

[^wiki]: [SILK](https://en.wikipedia.org/wiki/SILK) — Wikipedia, on Skype's linear-predictive speech codec and its role inside Opus.
