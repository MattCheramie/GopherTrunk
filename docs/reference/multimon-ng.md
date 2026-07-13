---
slug: multimon-ng
title: multimon-ng
entry_type: technology
category: sdr-software
description: "multimon-ng is an open-source digital-mode decoder that turns demodulated audio into text for POCSAG, FLEX, AFSK/APRS, DTMF and other legacy signalling modes."
keywords: multimon-ng, POCSAG decoder, FLEX decoder, AFSK decoder, APRS decoder, DTMF, pager decoder, audio decoder, multimon
aka: [multimon-ng, multimon]
autolink: true
infobox:
  - { label: Type, value: Multi-mode audio decoder }
  - { label: Decodes, value: "POCSAG, FLEX, AFSK, DTMF, and more" }
  - { label: Input, value: "Demodulated audio (pipe or sound card)" }
see_also: [pocsag, flex, afsk, aprs, ax25, frequency-shift-keying]
cite_urls:
  - https://en.wikipedia.org/wiki/POCSAG
  - https://github.com/EliasOenal/multimon-ng
---

**multimon-ng** is an open-source, command-line decoder for a grab-bag of legacy digital
signalling modes carried in audio, most famously the [POCSAG](/reference/pocsag/) and
[FLEX](/reference/flex/) pager protocols, along with [AFSK](/reference/afsk/) packet,
[DTMF](/reference/dtmf/), and several others.[^proj] It takes demodulated audio — typically FM
discriminator or receiver audio — and prints the decoded messages as text, making it the usual
back end for pager and low-speed data monitoring on an SDR.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 116" role="img" aria-label="multimon-ng takes one demodulated audio stream and runs several parallel decoders (POCSAG, FLEX, AFSK, DTMF), emitting decoded text messages." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="mmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="8" y="44" width="96" height="28" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
  <text x="56" y="56" text-anchor="middle" font-size="8" fill="currentColor">demod audio</text><text x="56" y="66" text-anchor="middle" font-size="8" fill="currentColor">(pipe / .wav)</text>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="200" y="8" width="88" height="22" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="244" y="23">POCSAG</text>
    <rect x="200" y="36" width="88" height="22" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="244" y="51">FLEX</text>
    <rect x="200" y="64" width="88" height="22" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="244" y="79">AFSK / DTMF</text>
    <rect x="340" y="36" width="110" height="22" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="395" y="51">decoded text</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="104" y1="52" x2="198" y2="20" marker-end="url(#mmar)"/>
    <line x1="104" y1="58" x2="198" y2="47" marker-end="url(#mmar)"/>
    <line x1="104" y1="64" x2="198" y2="74" marker-end="url(#mmar)"/>
    <line x1="288" y1="47" x2="338" y2="47" marker-end="url(#mmar)"/>
  </g>
</svg>
<figcaption>multimon-ng runs several selectable decoders on one demodulated audio stream and prints the recovered messages as text.</figcaption>
</figure>

## How it works

multimon-ng expects **demodulated audio**, not raw IQ: a stream of 22050 Hz samples fed from a
pipe (commonly from an FM demodulator such as `rtl_fm` or an SDR application's audio output),
a WAV file, or the sound card. The user selects one or more decoders on the command line, and
each runs on the same audio in parallel.

Each decoder implements its mode's low-level recovery. For [POCSAG](/reference/pocsag/) it
tracks the two-level [FSK](/reference/frequency-shift-keying/) at 512, 1200, or 2400 bit/s,
finds the preamble and frame-sync word, groups the interleaved code words, applies the
BCH error-correcting code, and reassembles numeric or alphanumeric pages. [FLEX](/reference/flex/)
decoding handles its 1600/3200/6400 bit/s multi-level FSK framing similarly. The
[AFSK](/reference/afsk/) decoders recover 1200-baud (Bell 202) and related tones and hand off
frames that can be [AX.25](/reference/ax25/)/[APRS](/reference/aprs/) packets, and there are
decoders for [DTMF](/reference/dtmf/), ZVEI and other selective-calling tones, and more. Output
is plain text lines, one per decoded message, easily logged or piped into other programs.

Because it works from audio, multimon-ng is front-end agnostic — anything that can produce the
right demodulated audio can feed it, which is why it pairs naturally with cheap SDR dongles.

## Relevance to SDR

multimon-ng is a mainstay for monitoring pagers and simple data bursts on an SDR: a common
recipe pipes `rtl_fm` audio straight into multimon-ng to read POCSAG/FLEX traffic in real
time. Its breadth of legacy modes makes it a handy utility for utility-signal hunting and for
teaching how these older FSK/AFSK protocols are framed and error-corrected. It is strictly a
decoder of the payload once a signal is demodulated; tuning, filtering, and demodulation are
the SDR front end's job.

**GopherTrunk** is a separate project with different scope. GopherTrunk decodes IQ directly and
concentrates on **trunked-radio** and digital-voice systems (P25, DMR, NXDN, TETRA, and more),
handling its own channelization and demodulation rather than consuming pre-demodulated audio.
Their coverage overlaps only at the edges — GopherTrunk's protocol list does include some of the
data/paging modes multimon-ng handles (such as POCSAG) — but the tools take different inputs and
serve different workflows: multimon-ng is a lightweight audio-in, text-out decoder for legacy
signalling, while GopherTrunk is an integrated IQ scanner. They can share hardware, and
multimon-ng remains the simplest route to reading classic pager traffic.

## Sources

[^proj]: [multimon-ng](https://github.com/EliasOenal/multimon-ng) — the source repository listing the supported decoders and audio input methods; background on the flagship mode is in the [POCSAG Wikipedia article](https://en.wikipedia.org/wiki/POCSAG).
