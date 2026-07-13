---
slug: amr
title: AMR / AMR-WB
entry_type: technology
category: voice-coding
description: "AMR (Adaptive Multi-Rate) is the 3GPP cellular speech codec family, an ACELP coder that switches bitrate to trade voice quality against channel error protection in GSM, UMTS, and VoLTE."
keywords: AMR, Adaptive Multi-Rate, AMR-NB, AMR-WB, HD Voice, ACELP, 3GPP speech codec, GSM voice, UMTS voice, VoLTE, codec mode, source-controlled rate, discontinuous transmission
aka: [AMR, AMR-NB, AMR-WB]
autolink: true
infobox:
  - { label: Type, value: Cellular speech codec family (ACELP) }
  - { label: Rates, value: "NB 4.75–12.2 kbps; WB 6.6–23.85 kbps" }
  - { label: Used by, value: "GSM, UMTS, VoLTE" }
see_also: [acelp, code-excited-linear-prediction, volte, evs-codec, gsm]
cite_urls:
  - https://en.wikipedia.org/wiki/Adaptive_Multi-Rate_audio_codec
  - https://en.wikipedia.org/wiki/Adaptive_Multi-Rate_Wideband
---

**AMR** (**Adaptive Multi-Rate**) is the family of speech codecs standardised by 3GPP for
cellular telephony.[^wiki] At its core AMR is an [ACELP](/reference/acelp/) coder — an
[algebraic code-excited](/reference/code-excited-linear-prediction/) speech model — but its
defining feature is *adaptivity*: it can switch among eight bitrates frame by frame so that,
when radio conditions worsen, bits shift from the voice codec to stronger channel coding while
keeping the total rate constant. AMR became the mandatory codec for [GSM](/reference/gsm/) and
UMTS voice and, as AMR-WB, is the original "HD Voice" heard on [VoLTE](/reference/volte/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A fixed total bit budget is split between an AMR speech codec and channel coding; in good radio conditions most bits go to voice at a high codec mode, and in poor conditions the codec drops to a lower rate so more bits protect against errors." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="120" y="18">good channel</text><text x="340" y="18">poor channel</text>
    <rect x="30" y="30" width="120" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.2"/>
    <rect x="150" y="30" width="60" height="26" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <text x="90" y="47">voice 12.2k</text><text x="180" y="47">FEC</text>
    <rect x="250" y="30" width="60" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.2"/>
    <rect x="310" y="30" width="120" height="26" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <text x="280" y="47">voice 4.75k</text><text x="370" y="47">FEC (more)</text>
    <text x="230" y="82" font-size="8">same total gross bitrate — the split adapts</text>
    <line x1="30" y1="66" x2="430" y2="66" stroke="currentColor" stroke-width="1"/>
  </g>
</svg>
<figcaption>AMR keeps the gross bitrate fixed and reallocates bits between the ACELP speech coder and error protection as the radio channel changes.</figcaption>
</figure>

## How it works

AMR narrowband (AMR-NB) operates on 8 kHz speech in 20 ms frames and defines eight codec modes
from 4.75 to 12.2 kbps, all built on ACELP. Each mode uses the same short-term
[LPC](/reference/linear-predictive-coding/) analysis and adaptive-plus-algebraic excitation; the
modes differ mainly in how many algebraic pulses and how much gain/LSF precision they spend, so
higher modes sound better but leave fewer bits for forward error correction. A source-controlled
rate mechanism plus discontinuous transmission (DTX) with comfort-noise generation lets the
encoder go silent between talkspurts to save capacity.

**AMR-WB** extends the same idea to a 16 kHz sampling rate, coding audio out to ~7 kHz with nine
modes from 6.6 to 23.85 kbps. The wider band is what makes voices sound markedly clearer, and
AMR-WB was standardised in parallel as ITU-T G.722.2. Crucially, the *link* — not the codec —
decides the mode: the network signals a codec-mode request based on measured channel quality, so
the two endpoints track conditions together.

## Relevance to SDR

AMR is a cellular baseband codec, carried inside encrypted, tightly scheduled GSM/UMTS/LTE air
interfaces rather than in the clear on land-mobile channels. It matters here as the reference
point for how digital *cellular* voice is compressed, and as the direct ancestor of 3GPP
[EVS](/reference/evs-codec/), which keeps an AMR-WB-compatible mode for interworking. GopherTrunk
targets trunked land-mobile radio and does not decode cellular traffic, so AMR is context rather
than a codec in GopherTrunk's chain — the trunking systems it does follow use MBE-family or, for
TETRA, plain ACELP vocoders instead.

## Sources

[^wiki]: [Adaptive Multi-Rate audio codec](https://en.wikipedia.org/wiki/Adaptive_Multi-Rate_audio_codec) — Wikipedia, on AMR's ACELP core, the eight narrowband modes, DTX, and its role in GSM/UMTS.
