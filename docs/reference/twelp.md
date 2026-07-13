---
slug: twelp
title: TWELP
entry_type: technology
category: voice-coding
description: "TWELP is DSP Group's low-rate speech vocoder, a Waveform-Interpolation competitor to DVSI's AMBE for digital radio."
keywords: TWELP, DSP Group vocoder, waveform interpolation, low rate speech, AMBE competitor, digital voice radio, 2400 bps
aka: [TWELP]
autolink: true
infobox:
  - { label: Type, value: Low-rate speech vocoder }
  - { label: Idea, value: Waveform-interpolation voice model }
  - { label: Vendor, value: DSP Group }
see_also: [multi-band-excitation, ambe, codec2, vocoder, ambe-plus-2, melp, dvsi]
cite_urls:
  - https://en.wikipedia.org/wiki/Vocoder
  - https://en.wikipedia.org/wiki/Speech_coding
---

**TWELP** (Time-Warped-Excitation Linear Prediction / Waveform Interpolation) is
a family of very-low-bitrate speech [vocoders](/reference/vocoder/) marketed by
DSP Group as a competitor to [DVSI](/reference/dvsi/)'s AMBE codecs for digital
land-mobile radio.[^wiki] It targets the same 2000–4000 bps envelope that IMBE and
[AMBE+2](/reference/ambe-plus-2/) occupy, promising comparable intelligibility and
noise robustness at those rates while offering an alternative licensing route for
radio manufacturers who did not want to depend on a single vocoder supplier.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="TWELP analyses a speech frame into a linear-prediction spectral model plus a warped-excitation characteristic waveform, transmits a few kilobits, and interpolates the waveform back at the decoder." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="twlpar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M20 60 q10 -26 20 0 t20 0 t20 0 t20 0" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="60" y="90" text-anchor="middle" font-size="8" fill="currentColor">speech</text>
  <line x1="104" y1="60" x2="140" y2="60" stroke="currentColor" stroke-width="1.2" marker-end="url(#twlpar)"/>
  <rect x="144" y="40" width="96" height="40" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
  <text x="192" y="57" text-anchor="middle" font-size="8" fill="currentColor">LP model +</text>
  <text x="192" y="69" text-anchor="middle" font-size="8" fill="currentColor">warped excitation</text>
  <line x1="240" y1="60" x2="276" y2="60" stroke="currentColor" stroke-width="1.2" marker-end="url(#twlpar)"/>
  <rect x="280" y="46" width="70" height="28" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="315" y="63" text-anchor="middle" font-size="8" fill="currentColor">~2.4 kbps</text>
  <line x1="350" y1="60" x2="386" y2="60" stroke="currentColor" stroke-width="1.2" marker-end="url(#twlpar)"/>
  <path d="M392 60 q6 -22 12 0 t12 0 t12 0" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="410" y="90" text-anchor="middle" font-size="8" fill="currentColor">rebuilt</text>
</svg>
<figcaption>TWELP models each frame as a spectral envelope plus a warped characteristic waveform, coded at a few kbps.</figcaption>
</figure>

## How it works

TWELP is rooted in waveform-interpolation (also called prototype-waveform
interpolation) speech modelling combined with linear prediction. Rather than
splitting the spectrum into voiced and unvoiced bands the way the
[multi-band-excitation](/reference/multi-band-excitation/) family does, a
waveform-interpolation coder extracts a *characteristic* excitation waveform once
per pitch period, decomposes it into slowly and rapidly evolving components
(capturing the voiced and noise-like parts of the sound), and transmits those
along with a linear-prediction spectral envelope and pitch track. At the decoder
the successive prototype waveforms are interpolated over time to regenerate a
continuous excitation, which drives the synthesis filter to reconstruct speech.

Because only a compact set of parameters — envelope, pitch, gain and the evolving
excitation description — is sent per frame, TWELP reaches rates around 2400 bps
(and lower variants) while remaining reasonably robust in the noisy, mobile
environments that radio must survive. Vendors published TWELP profiles aimed
specifically at professional and public-safety radio, with error-tolerance and
tandem-friendly behaviour comparable to AMBE-class coders.

## In practice

TWELP's practical appeal was as a second source. DVSI's AMBE and
[AMBE+2](/reference/ambe-plus-2/) are the mandated vocoders in several standards,
and their licensing and hardware-DSP ecosystem locked manufacturers into one
supplier. A codec covering the same rate and quality band gave designers of
non-standardised or proprietary radios a way to build low-rate digital voice
without that dependency, in the same spirit that the open
[Codec 2](/reference/codec2/) does for the amateur community — though TWELP itself
is a commercial, licensed product rather than an open one.

## Relevance to SDR

TWELP sits in the same problem space GopherTrunk cares about — squeezing
intelligible voice into a few kbps of a narrowband radio channel — but it is not
the vocoder used by the mainstream trunked standards GopherTrunk decodes. P25,
DMR and NXDN standardise on the MBE family (IMBE and AMBE+2), so a GopherTrunk
receiver following those air interfaces meets AMBE-class bitstreams, not TWELP.
TWELP mainly appears in specific vendor products and some non-standard digital
radios. GopherTrunk does not implement TWELP decoding; understanding it is useful
mostly as context for why the AMBE-versus-alternatives licensing question shaped
the digital-voice radio market.

## Sources

[^wiki]: [Vocoder](https://en.wikipedia.org/wiki/Vocoder) — Wikipedia, background on low-bitrate speech vocoders of the class TWELP belongs to; see also [Speech coding](https://en.wikipedia.org/wiki/Speech_coding).
