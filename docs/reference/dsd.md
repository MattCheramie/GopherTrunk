---
slug: dsd
title: DSD (Digital Speech Decoder)
entry_type: technology
category: sdr-software
description: "DSD (Digital Speech Decoder) is an open-source tool that decodes P25, DMR, NXDN and other digital-voice modes from discriminator audio, turning AMBE/IMBE frames into speech."
keywords: DSD, Digital Speech Decoder, DSDPlus, digital voice decoder, AMBE decoder, IMBE decoder, P25 decoder, DMR decoder, NXDN decoder, discriminator audio
aka: [DSD, Digital Speech Decoder, DSDPlus, dsd-fme]
autolink: true
infobox:
  - { label: Type, value: Digital-voice decoder }
  - { label: Decodes, value: "P25, DMR, NXDN, D-STAR, YSF, X2-TDMA" }
  - { label: Vocoder, value: "AMBE / IMBE frames to audio" }
see_also: [ambe, imbe, p25-phase-1, dmr, nxdn, vocoder]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_Speech_Decoder
  - https://github.com/szechyjs/dsd
---

**DSD (Digital Speech Decoder)** is an open-source program that decodes several digital-voice
radio modes — including [P25 Phase 1](/reference/p25-phase-1/), [DMR](/reference/dmr/),
[NXDN](/reference/nxdn/), D-STAR, and [System Fusion/YSF](/reference/system-fusion-ysf/) — by
taking demodulated FM *discriminator* audio and recovering the underlying
[vocoder](/reference/vocoder/) frames as intelligible speech.[^proj] It is the tool that made
listening to unencrypted digital-voice traffic practical for hobbyists, and its variants
(notably DSD+ and the open dsd-fme fork) are widely used.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="DSD pipeline: discriminator audio is sliced into symbols, the mode is identified, frames are FEC-corrected, and AMBE or IMBE vocoder frames are synthesized into speech." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="dsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="6" y="46" width="72" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="42" y="58">discrim.</text><text x="42" y="68">audio</text>
    <rect x="104" y="46" width="72" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="140" y="58">symbol +</text><text x="140" y="68">sync</text>
    <rect x="202" y="46" width="72" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="238" y="58">FEC +</text><text x="238" y="68">deframe</text>
    <rect x="300" y="46" width="72" height="28" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="336" y="58">AMBE/IMBE</text><text x="336" y="68">vocoder</text>
    <rect x="398" y="46" width="56" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="426" y="63">speech</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="78" y1="60" x2="102" y2="60" marker-end="url(#dsar)"/>
    <line x1="176" y1="60" x2="200" y2="60" marker-end="url(#dsar)"/>
    <line x1="274" y1="60" x2="298" y2="60" marker-end="url(#dsar)"/>
    <line x1="372" y1="60" x2="396" y2="60" marker-end="url(#dsar)"/>
  </g>
</svg>
<figcaption>DSD recovers symbols and sync from discriminator audio, corrects and deframes the bitstream, then synthesizes AMBE/IMBE vocoder frames into speech.</figcaption>
</figure>

## How it works

Classic DSD is fed the raw **discriminator output** of an FM receiver — the instantaneous
frequency, before any de-emphasis or filtering — which for the 4-level modulations used by
these systems ([C4FM](/reference/c4fm/) and related) looks like a noisy four-level baseband
signal. DSD performs **symbol timing recovery** to sample once per symbol, then a slicer maps
each sample to one of the levels (dibits). It searches for each mode's **frame-sync** pattern
to both identify the protocol (P25, DMR, NXDN, and so on) and align to frame boundaries.

Once aligned, DSD applies the mode's **forward error correction** and deinterleaving to
recover the payload, separating signalling from the voice frames. The voice frames are
[vocoder](/reference/vocoder/) parameters — [IMBE](/reference/imbe/) for P25 Phase 1,
[AMBE](/reference/ambe/)/AMBE+2 for DMR, NXDN, D-STAR, and YSF — which DSD passes to a vocoder
**synthesizer** that reconstructs the audio waveform played out to the listener. Encrypted
traffic still decodes as frames but yields only noise, since DSD has no keys. Modern usage
often skips a physical radio's discriminator and instead feeds DSD from an SDR demodulator's
audio, while forks like dsd-fme add direct digital output, trunk-following helpers, and more
modes.

## Relevance to SDR

DSD (and its DSD+ and dsd-fme relatives) is the classic answer to "how do I hear a digital
scanner signal on my SDR?" It handles the hard part — recovering speech from a vocoder-encoded,
FEC-protected bitstream — for the major North American and international digital-voice
standards. In a typical SDR setup, a receiver demodulates the FM signal and routes audio into
DSD, which prints call metadata and outputs speech. It is a voice/per-channel decoder, though;
following a trunked system's control channel and hopping to voice grants is handled by
companion tooling (or by the DSD+ suite's own trunking components).

**GopherTrunk** overlaps DSD in purpose but is an independent, from-scratch implementation.
GopherTrunk is a pure-Go **trunking scanner** that does its own IQ channelization, symbol
recovery, framing, and control-channel following, and it decodes the clear/scrambled
signalling and audio of P25, DMR, NXDN, TETRA, and more without relying on DSD. Like DSD, it
cannot decode keyed **encryption** — both recover only unencrypted traffic. The key
distinction is scope and integration: DSD is a focused per-channel voice decoder often paired
with separate control software, while GopherTrunk is an integrated scanner that manages the
whole trunked system from control channel to voice. They are best thought of as parallel tools
that solve the same digital-voice problem in different ecosystems.

## Sources

[^proj]: [DSD (Digital Speech Decoder)](https://github.com/szechyjs/dsd) — the source repository, and the [Wikipedia article](https://en.wikipedia.org/wiki/Digital_Speech_Decoder), documenting the supported modes, discriminator-audio input, symbol/FEC processing, and AMBE/IMBE vocoder synthesis.
