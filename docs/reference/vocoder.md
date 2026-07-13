---
slug: vocoder
title: Vocoder
entry_type: technology
category: voice-coding
description: A vocoder is a speech codec that compresses voice to a few kbps by modelling how speech is produced rather than recording it — the basis of all digital voice radio.
keywords: vocoder, voice codec, speech coding, IMBE, AMBE, source filter model, digital voice, linear predictive coding, CELP
aka: [vocoder]
autolink: true
infobox:
  - { label: Type, value: Speech codec }
  - { label: Approach, value: Model speech (pitch + spectrum) }
  - { label: Bit rate, value: A few kbps }
  - { label: Examples, value: IMBE, AMBE+2, Codec 2 }
see_also: [imbe, ambe, ambe-plus-2, codec2, multi-band-excitation, linear-predictive-coding, code-excited-linear-prediction, twelp]
related_lessons:
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/rf-sdr/vocoders/ }
  - { title: "Analog vs. digital voice", url: /learn/rf-sdr/digital-voice/ }
external:
  - { title: "GopherTrunk vocoders", url: /vocoders.html }
related_reading:
  - { title: "SDR Internals, Part 12: Voice coding & vocoders", url: /blog/deep-dives/sdr-internals-12-voice-coding-vocoders/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Vocoder
  - https://en.wikipedia.org/wiki/Speech_coding
---

A **vocoder** (voice coder) is a speech codec that compresses voice into a few kilobits
per second by **modelling how speech is produced** — pitch, voicing, and spectral shape
— rather than recording the waveform.[^wiki] It is what makes
[digital voice](/learn/rf-sdr/digital-voice/) radio possible: a raw
[pulse-code-modulation](/reference/pulse-code-modulation/) audio stream of 64–128 kbps is
far too wide for a narrowband land-mobile channel, so the vocoder throws away the waveform
and keeps only the parameters needed to rebuild an intelligible voice.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 120" role="img" aria-label="Speech analysed into a source-and-filter model, sent as a tiny parameter frame, then re-synthesised back into speech." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="40" y="63">speech</text>
    <rect x="78" y="44" width="86" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="121" y="58">analyse</text><text x="121" y="70" font-size="8">pitch · spectrum</text>
    <rect x="200" y="48" width="90" height="26" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="245" y="65">~2–4 kbps frame</text>
    <rect x="326" y="44" width="90" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="371" y="58">synthesise</text><text x="371" y="70" font-size="8">re-create voice</text>
    <text x="470" y="63">audio</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="64" y1="61" x2="77" y2="61"/><line x1="164" y1="61" x2="199" y2="61"/><line x1="290" y1="61" x2="325" y2="61"/><line x1="416" y1="61" x2="448" y2="61"/></g>
  </g>
</svg>
<figcaption>A vocoder models how speech is produced and sends only parameters, so a voice fits in a few kbps.</figcaption>
</figure>

## How it works

Almost every modern vocoder rests on the **source–filter model** of human speech: the
lungs and vocal folds are a *source* (a periodic buzz at the pitch frequency for voiced
sounds like vowels, or turbulent noise for unvoiced sounds like "s" and "f"), and the
throat, mouth, and nasal cavity form a slowly varying *filter* that shapes that source
into recognizable phonemes.[^sc] If you can measure the source (pitch, voiced/unvoiced
decision, energy) and the filter (the spectral envelope) a few dozen times a second, you
can transmit just those numbers and let the receiver drive a matching synthesizer.

Concretely, the encoder slices audio into short frames (typically 20 ms) and, for each
frame, estimates:

- **Pitch / fundamental frequency** — how fast the vocal folds vibrate.
- **Voicing** — whether the frame (or each sub-band of it) is periodic or noise-like.
- **Spectral envelope** — the resonances (formants) of the vocal tract, coded either as
  [linear-predictive-coding](/reference/linear-predictive-coding/) coefficients or, in the
  MBE family, as a set of harmonic magnitudes.
- **Gain / energy** — the loudness of the frame.

The decoder feeds a pitched pulse train or noise through the reconstructed spectral filter
to synthesize output. Because only parameters cross the channel, a corrupted parameter is
much more damaging than a corrupted audio sample would be — which is why vocoded speech on
a weak signal sounds warbly or "R2-D2" robotic rather than merely noisy, and why the bit
stream is wrapped in heavy [forward error correction](/reference/forward-error-correction/).

## Variants

Vocoders split into two broad lineages. **Parametric / sinusoidal** coders (the
Multi-Band Excitation family — [IMBE](/reference/imbe/), [AMBE](/reference/ambe/),
[AMBE+2](/reference/ambe-plus-2/), and [TWELP](/reference/twelp/)) model speech as a sum of
harmonics with per-band voicing decisions; they excel at very low rates and are dominant in
land-mobile radio. **[CELP](/reference/code-excited-linear-prediction/)** coders
([ACELP](/reference/acelp/) and its descendants like [AMR](/reference/amr/)) instead search
a codebook of excitation vectors that, run through an LPC filter, best match the original —
this "analysis by synthesis" approach powers cellular telephony (GSM, UMTS, LTE). The
open-source [Codec 2](/reference/codec2/) is parametric like MBE but royalty-free.

## In practice

Vocoder choice is a trade between bit rate, audio quality, robustness, and licensing.
Land-mobile systems run 2.4–4.4 kbps of net voice so two conversations fit a 12.5 kHz
channel; cellular systems can afford 5–13 kbps for near-toll quality. The MBE/AMBE codecs
are proprietary (licensed by [DVSI](/reference/dvsi/)), which is precisely why the amateur
[M17](/reference/m17/) project chose the open Codec 2 instead.

## Relevance to SDR

Decoding digital voice requires running the **matching** vocoder — [IMBE](/reference/imbe/)
for [P25 Phase 1](/reference/p25-phase-1/), [AMBE+2](/reference/ambe-plus-2/) for
[DMR](/reference/dmr/) and P25 Phase 2, AMBE for [D-STAR](/reference/d-star/), or
[Codec 2](/reference/codec2/) for [M17](/reference/m17/). GopherTrunk implements the
land-mobile vocoders in pure Go so it can render clear (unencrypted) voice without the DVSI
hardware dongle that older scanners required.

## Sources

[^wiki]: [Vocoder](https://en.wikipedia.org/wiki/Vocoder) — Wikipedia, on speech coding that models how voice is produced.
[^sc]: [Speech coding](https://en.wikipedia.org/wiki/Speech_coding) — Wikipedia, on the source–filter model and the parametric vs. CELP families of low-rate codecs.
