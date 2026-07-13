---
slug: ralcwi
title: RALCWI
entry_type: technology
category: voice-coding
description: "RALCWI is CML Microcircuits' low-bitrate speech vocoder, used in some DMR and dPMR digital radios as an AMBE alternative."
keywords: RALCWI, CML Microcircuits, low bitrate vocoder, DMR, dPMR, digital voice, waveform interpolation, AMBE alternative
aka: [RALCWI]
autolink: true
infobox:
  - { label: Type, value: Low-rate speech vocoder }
  - { label: Vendor, value: CML Microcircuits }
  - { label: Seen in, value: Some DMR / dPMR radios }
see_also: [multi-band-excitation, ambe, dpmr, dmr, vocoder, ambe-plus-2, codec2]
cite_urls:
  - https://en.wikipedia.org/wiki/Vocoder
  - https://en.wikipedia.org/wiki/DPMR
---

**RALCWI** (Robust Advanced Low Complexity Waveform Interpolation) is a
low-bitrate speech [vocoder](/reference/vocoder/) developed by CML
Microcircuits and sold as a chip-based alternative to
[DVSI](/reference/dvsi/)'s AMBE for digital radio.[^wiki] It encodes voice at
roughly 2000–3600 bps and has been used in some [DMR](/reference/dmr/) and
[dPMR](/reference/dpmr/) handsets and modules, giving radio makers a vocoder they
could license and integrate as a dedicated IC without depending on the AMBE
supply chain.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A RALCWI vocoder IC sits between a microphone speech input and the radio modulator, turning speech into a low-rate bitstream and back." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rlcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M22 55 q8 -22 16 0 t16 0 t16 0" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="46" y="82" text-anchor="middle" font-size="8" fill="currentColor">speech</text>
  <line x1="88" y1="55" x2="124" y2="55" stroke="currentColor" stroke-width="1.2" marker-end="url(#rlcar)"/>
  <rect x="128" y="34" width="110" height="42" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
  <text x="183" y="52" text-anchor="middle" font-size="9" fill="currentColor">RALCWI</text>
  <text x="183" y="65" text-anchor="middle" font-size="7" fill="currentColor">vocoder IC (WI)</text>
  <line x1="238" y1="55" x2="274" y2="55" stroke="currentColor" stroke-width="1.2" marker-end="url(#rlcar)"/>
  <rect x="278" y="40" width="76" height="30" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="316" y="59" text-anchor="middle" font-size="8" fill="currentColor">~2.4 kbps</text>
  <line x1="354" y1="55" x2="390" y2="55" stroke="currentColor" stroke-width="1.2" marker-end="url(#rlcar)"/>
  <rect x="392" y="40" width="52" height="30" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="418" y="59" text-anchor="middle" font-size="8" fill="currentColor">FEC + RF</text>
</svg>
<figcaption>RALCWI is a chip-based waveform-interpolation vocoder feeding low-rate voice bits into a radio's FEC and modulator.</figcaption>
</figure>

## How it works

RALCWI, as its expansion states, is a waveform-interpolation coder. It analyses
each speech frame into a linear-prediction spectral envelope and a pitch estimate,
then represents the excitation as a slowly evolving prototype waveform sampled once
per pitch period. Splitting that prototype into slowly and rapidly changing parts
lets the model capture both the periodic (voiced) and noise-like (unvoiced)
character of the sound. Only the envelope, pitch, gains and the compact excitation
description are transmitted; the decoder interpolates between successive prototype
waveforms to rebuild a continuous excitation and drives the synthesis filter to
recover speech.

The design emphasises low computational complexity and robustness — the "RLC" of
the name — so it can run in a small fixed-function IC and tolerate the bit errors
of a mobile radio channel. CML packaged it as vocoder chips (and later soft/DSP
implementations) presenting a simple digital interface: audio in, a few kbps of
voice bits out, ready to hand to the radio's
[forward-error-correction](/reference/forward-error-correction/) and modulator.

## In practice

RALCWI's role, like [TWELP](/reference/twelp/)'s, was to offer a second source
for low-rate digital voice. Because AMBE and
[AMBE+2](/reference/ambe-plus-2/) are widely licensed but come from a single
vendor, an independent vocoder let some DMR and dPMR product lines ship digital
voice without that dependency. The trade-off is interoperability: radios using
RALCWI encode a different bitstream than AMBE radios, so at the vocoder level they
are not directly interchangeable even when they share the same
[multi-band-excitation](/reference/multi-band-excitation/)-era rate budget and the
same air interface framing.

## Relevance to SDR

For a decoder like GopherTrunk, the vocoder matters as much as the modulation. The
DMR and dPMR standards themselves define the radio layer, but the voice payload can
be AMBE+2 or, in some equipment, RALCWI — and a receiver can only turn voice bits
back into audio if it implements the matching vocoder. The bulk of trunked and
conventional digital traffic on the air uses the AMBE family, which is what
GopherTrunk focuses on; RALCWI-encoded voice from those specific radios would need
a separate RALCWI synthesiser to render. GopherTrunk does not implement RALCWI
decoding. Knowing it exists explains why occasional DMR/dPMR voice may not decode
even when the signalling does.

## Sources

[^wiki]: [Vocoder](https://en.wikipedia.org/wiki/Vocoder) — Wikipedia, background on the low-rate waveform-interpolation vocoder class that RALCWI belongs to; see also [dPMR](https://en.wikipedia.org/wiki/DPMR).
