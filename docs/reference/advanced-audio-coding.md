---
slug: advanced-audio-coding
title: Advanced Audio Coding (AAC)
entry_type: technology
category: voice-coding
description: Advanced Audio Coding is the MPEG perceptual audio codec that succeeded MP3, using the MDCT and a psychoacoustic model to code full-band audio at low bitrates; DAB+ and DRM carry HE-AAC.
keywords: AAC, Advanced Audio Coding, HE-AAC, AAC-LC, MDCT, psychoacoustic model, SBR, parametric stereo, DAB+, perceptual codec, MPEG-4
aka: [AAC, Advanced Audio Coding, HE-AAC, AAC-LC]
autolink: true
infobox:
  - { label: Type, value: Perceptual audio codec }
  - { label: Family, value: MPEG-2 / MPEG-4 }
  - { label: Core tools, value: "MDCT + psychoacoustic model" }
  - { label: Used by, value: "DAB+, DRM, ATSC/DVB" }
see_also: [vocoder, opus-codec, pulse-code-modulation, dab, drm-broadcast, hd-radio, spectral-efficiency]
cite_urls:
  - https://en.wikipedia.org/wiki/Advanced_Audio_Coding
  - https://www.iso.org/standard/76383.html
---

**Advanced Audio Coding** (**AAC**) is the MPEG-2/MPEG-4 perceptual audio codec that
succeeded MP3, delivering better quality at the same bitrate.[^wiki] Unlike a speech
[vocoder](/reference/vocoder/), which models the human voice, AAC codes full-band music
and general audio; it works by transforming blocks of
[PCM](/reference/pulse-code-modulation/) samples and discarding detail a listener cannot
hear, in the same perceptual-coding family as the [Opus codec](/reference/opus-codec/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 130" role="img" aria-label="Signal chain: PCM audio enters a modified discrete cosine transform, then a psychoacoustic model drives quantization, then entropy coding produces a compressed bitstream." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="12" y="50" width="86" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="55" y="64">PCM audio</text><text x="55" y="76" font-size="7.5">samples</text>
    <rect x="128" y="50" width="86" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="171" y="64">MDCT</text><text x="171" y="76" font-size="7.5">time → freq</text>
    <rect x="244" y="50" width="96" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="292" y="61">quantize</text><text x="292" y="73" font-size="7.5">psychoacoustic</text>
    <rect x="370" y="50" width="88" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="414" y="64">entropy</text><text x="414" y="76" font-size="7.5">bitstream</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="98" y1="67" x2="127" y2="67" marker-end="url(#rel_aac)"/><line x1="214" y1="67" x2="243" y2="67" marker-end="url(#rel_aac)"/><line x1="340" y1="67" x2="369" y2="67" marker-end="url(#rel_aac)"/></g>
    <text x="292" y="34" font-size="7.5" fill="currentColor" fill-opacity="0.85">masking curve hides inaudible detail</text>
    <line x1="292" y1="38" x2="292" y2="49" stroke="currentColor" stroke-width="1" stroke-opacity="0.6"/>
  </g>
  <defs><marker id="rel_aac" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>AAC transforms PCM audio with the MDCT, quantizes each frequency band according to a psychoacoustic masking model, then entropy-codes the result into a compact bitstream.</figcaption>
</figure>

## How it works

AAC splits the audio into overlapping blocks and applies a **modified discrete cosine
transform (MDCT)** to move each block from the time domain into a set of frequency
coefficients. A **psychoacoustic model** then estimates, band by band, how much noise the
ear will mask behind louder nearby tones, and the encoder quantizes each band only as
finely as audibility requires — spending bits where the ear is sensitive and coarsening
or dropping detail where it is not. The quantized coefficients are finally packed with
entropy (Huffman) coding, so the output bitstream is both perceptually transparent and
small. Because it is a *lossy transform* codec rather than a source model of speech, AAC
handles music, applause, and mixed program material that a low-rate voice coder cannot.

Several **profiles** trade complexity for efficiency. **AAC-LC** (Low Complexity) is the
baseline. **HE-AAC** — where "HE" is High-Efficiency — adds **Spectral Band Replication
(SBR)**, which reconstructs high frequencies from a compact parametric description instead
of coding them outright. **HE-AAC v2** adds **Parametric Stereo (PS)**, describing the
stereo image with a few side parameters, and reaches broadcast-usable quality near 24–48
kbps — a fraction of what MP3 needs for the same result.

## In practice

AAC's low-bitrate strength made it the audio codec of choice for constrained broadcast
links. **[DAB+](/reference/dab/)** carries HE-AAC v2, which let it pack many more stations
into a multiplex than the original DAB's MP2. **[DRM](/reference/drm-broadcast/)** (Digital
Radio Mondiale) also uses AAC-family coding for shortwave and mediumwave digital audio, and
digital television (ATSC and DVB) uses AAC for audio. One notable exception: US
**[HD Radio](/reference/hd-radio/)** does *not* use AAC — it carries its own proprietary
**HDC** codec, a perceptually similar but separate design. Choosing AAC over an outright PCM
stream is a large gain in
[spectral efficiency](/reference/spectral-efficiency/): the same channel carries far more
program content.

## Relevance to SDR

An SDR listener tuning a digital broadcast multiplex is, after demodulation and
error-correction, left with an AAC (or HDC) elementary stream that a software decoder turns
back into audio. Knowing which codec a mode uses tells you which decoder library to reach
for — welle.io and similar DAB+ receivers embed an HE-AAC decoder, while HD Radio tools need
the HDC path. GopherTrunk's own decode targets are land-mobile trunking voice (P25, DMR,
NXDN, TETRA) that use low-rate speech vocoders, not AAC; AAC is documented here as the
broadcast-audio counterpart an SDR user meets outside the trunking world.

## Sources

[^wiki]: [Advanced Audio Coding](https://en.wikipedia.org/wiki/Advanced_Audio_Coding) — Wikipedia, for AAC's history as the MP3 successor, the MDCT and psychoacoustic model, and the AAC-LC / HE-AAC / HE-AAC v2 profiles.
[^iso]: [ISO/IEC 14496-3 (MPEG-4 Audio)](https://www.iso.org/standard/76383.html) — the standard that defines the AAC profiles, SBR, and Parametric Stereo tools.
