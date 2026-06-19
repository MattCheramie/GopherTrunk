---
slug: vocoder
title: Vocoder
entry_type: technology
category: voice-coding
description: A vocoder is a speech codec that compresses voice to a few kbps by modelling how speech is produced rather than recording it — the basis of all digital voice radio.
keywords: vocoder, voice codec, speech coding, IMBE, AMBE, source filter model, digital voice
aka: [vocoder]
autolink: true
infobox:
  - { label: Type, value: Speech codec }
  - { label: Approach, value: Model speech (pitch + spectrum) }
  - { label: Bit rate, value: A few kbps }
  - { label: Examples, value: IMBE, AMBE+2, Codec 2 }
see_also: [imbe, ambe, ambe-plus-2, codec2, multi-band-excitation]
related_lessons:
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/rf-sdr/vocoders/ }
  - { title: "Analog vs. digital voice", url: /learn/rf-sdr/digital-voice/ }
external:
  - { title: "GopherTrunk vocoders", url: /vocoders.html }
related_reading:
  - { title: "SDR Internals, Part 12: Voice coding & vocoders", url: /blog/deep-dives/sdr-internals-12-voice-coding-vocoders/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Vocoder
---

A **vocoder** (voice coder) is a speech codec that compresses voice into a few kilobits
per second by **modelling how speech is produced** — pitch, voicing, and spectral shape
— rather than recording the waveform.[^wiki] It is what makes
[digital voice](/learn/rf-sdr/digital-voice/) radio possible.

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

Many times a second the vocoder extracts compact parameters of a short speech segment
and transmits only those; the receiver re-synthesises an audible voice from them. This
is why digital voice can sound slightly robotic, especially on a weak signal.

## Relevance to SDR

Decoding digital voice requires running the **matching** vocoder — [IMBE](/reference/imbe/)
for [P25 Phase 1](/reference/p25-phase-1/), [AMBE+2](/reference/ambe-plus-2/) for DMR and
P25 Phase 2, or [Codec 2](/reference/codec2/) for [M17](/reference/m17/).

## Sources

[^wiki]: [Vocoder](https://en.wikipedia.org/wiki/Vocoder) — Wikipedia, on speech coding that models how voice is produced.
