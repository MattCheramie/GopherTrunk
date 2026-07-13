---
slug: opus-codec
title: Opus
entry_type: technology
category: voice-coding
description: "Opus is an open, royalty-free IETF audio codec combining the SILK speech coder and the CELT transform coder, dominant for VoIP, WebRTC, and streaming from 6 kbps up to 510 kbps."
keywords: Opus, Opus codec, IETF RFC 6716, SILK, CELT, royalty-free audio codec, VoIP, WebRTC, low latency, hybrid codec, variable bitrate, packet loss concealment, Xiph
aka: [Opus, Opus codec]
autolink: true
infobox:
  - { label: Type, value: Open royalty-free audio codec (IETF) }
  - { label: Structure, value: "SILK (speech) + CELT (transform)" }
  - { label: Rates, value: "6–510 kbps, 5–66.5 ms latency" }
see_also: [silk-codec, code-excited-linear-prediction, evs-codec, volte]
cite_urls:
  - https://en.wikipedia.org/wiki/Opus_(audio_format)
  - https://www.rfc-editor.org/rfc/rfc6716
---

**Opus** is an open, royalty-free audio codec standardised by the IETF as RFC 6716, designed to
handle everything from low-bitrate speech to high-fidelity music in a single format.[^wiki] It is
the default codec of WebRTC and is ubiquitous in VoIP, conferencing, and streaming. Opus is a
*hybrid*: it fuses [SILK](/reference/silk-codec/) — a linear-predictive speech coder in the
[CELP](/reference/code-excited-linear-prediction/) tradition — with CELT, a low-delay transform
coder, and switches between or blends them depending on the content and bitrate.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="Input audio can be coded by the SILK linear-prediction speech layer, the CELT transform layer, or a hybrid of both, with a mode decision selecting the path based on content and bitrate before packing into an Opus packet." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="opar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1.2" fill="none" font-size="8">
    <rect x="16" y="52" width="60" height="26"/>
    <rect x="150" y="24" width="90" height="26"/>
    <rect x="150" y="80" width="90" height="26"/>
    <rect x="320" y="52" width="80" height="26"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="46" y="68">audio in</text>
    <text x="195" y="41">SILK (LP speech)</text>
    <text x="195" y="97">CELT (transform)</text>
    <text x="360" y="68">Opus packet</text>
    <text x="112" y="46">mode</text><text x="112" y="56">decision</text>
  </g>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <line x1="76" y1="65" x2="150" y2="37" marker-end="url(#opar)"/>
    <line x1="76" y1="65" x2="150" y2="93" marker-end="url(#opar)"/>
    <line x1="240" y1="37" x2="320" y2="60" marker-end="url(#opar)"/>
    <line x1="240" y1="93" x2="320" y2="70" marker-end="url(#opar)"/>
  </g>
</svg>
<figcaption>Opus routes audio through a SILK speech layer, a CELT transform layer, or a hybrid of both, chosen per frame by a mode decision.</figcaption>
</figure>

## How it works

Opus reuses two prior codecs. **SILK**, contributed by Skype, is a linear-predictive coder good
at speech at low-to-moderate bitrates; **CELT**, from Xiph.Org, is an MDCT transform coder
engineered for very low algorithmic delay and good music quality. Opus runs SILK below roughly
8–12 kHz audio bandwidth, CELT for wideband/fullband and low-latency needs, and a *hybrid* mode
in between where SILK codes the low band and CELT the high band. A single stream can change mode,
bitrate, bandwidth, and channel count seamlessly at any 20 ms boundary.

That flexibility is the point. Opus offers bitrates from about 6 kbps to 510 kbps, sampling rates
up to 48 kHz, mono and stereo, and configurable frame sizes from 2.5 to 60 ms — so an application
dials in latency versus quality without changing codecs. Built-in variable bitrate, forward error
correction, and packet-loss concealment make it robust over lossy networks, which is why WebRTC
and most modern VoIP stacks adopted it. Being royalty-free removed the licensing barrier that
codecs like AMR and ACELP-based ones carry.

## Relevance to SDR

Opus is a network audio codec, not an over-the-air radio waveform, so it is not decoded from RF.
It is relevant to this guide as the leading *open, royalty-free* alternative in the same design
space as the licensed cellular codecs, and it frequently sits at the receiving end of the chain:
SDR decoders and trunking loggers often re-encode recovered voice to Opus for storage or streaming
because it is compact and unencumbered. GopherTrunk decodes the on-air vocoder to PCM; delivering
or archiving that audio as Opus is a downstream packaging choice rather than part of the RF decode
itself.

## Sources

[^wiki]: [Opus (audio format)](https://en.wikipedia.org/wiki/Opus_(audio_format)) — Wikipedia, on the SILK+CELT hybrid structure, bitrate and latency range, and royalty-free IETF status.
