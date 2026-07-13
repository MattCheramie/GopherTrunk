---
slug: vara
title: VARA
entry_type: protocol
category: amateur-digital
description: "VARA is a software OFDM modem for amateur radio that carries error-free ARQ data over HF and VHF/FM channels, widely used as a Pactor alternative for Winlink email."
keywords: VARA, VARA HF, VARA FM, OFDM modem, sound-card modem, ARQ, Winlink data mode, EA5HVK, radio email, HF data
aka: [VARA, VARA HF, VARA FM]
autolink: true
infobox:
  - { label: Type, value: Software OFDM data modem }
  - { label: Standards body, value: Proprietary (EA5HVK) }
  - { label: Introduced, value: 2017 }
  - { label: Access, value: ARQ, half-duplex }
  - { label: Channel spacing, value: ~2.3 kHz (HF), voice channel (FM) }
  - { label: Modulation, value: OFDM with adaptive PSK/QAM subcarriers }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [ofdm, winlink, phase-shift-keying, quadrature-amplitude-modulation, subcarrier]
cite_urls:
  - https://en.wikipedia.org/wiki/VARA_(software)
---

**VARA** is a **software modem** for amateur radio that moves error-free data over ordinary
HF and VHF/FM voice channels using [OFDM](/reference/ofdm/) — dividing the audio passband
into many closely spaced [subcarriers](/reference/subcarrier/), each carrying a slice of the
data.[^wiki] Running entirely as a PC sound-card application (no special hardware), it pairs
that OFDM waveform with an adaptive **ARQ** protocol, and has become the dominant data mode
for [Winlink](/reference/winlink/) radio email, largely displacing the hardware-modem
[Pactor](/reference/pactor/) it was designed to rival.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="VARA fills the audio channel with many narrow OFDM subcarriers, each modulated with PSK or QAM at a level chosen to match the current signal-to-noise ratio." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="95" x2="440" y2="95" stroke="currentColor"/>
  <g stroke="currentColor" stroke-width="1.2">
    <line x1="55" y1="95" x2="55" y2="55"/><line x1="80" y1="95" x2="80" y2="45"/><line x1="105" y1="95" x2="105" y2="40"/><line x1="130" y1="95" x2="130" y2="42"/><line x1="155" y1="95" x2="155" y2="38"/><line x1="180" y1="95" x2="180" y2="44"/><line x1="205" y1="95" x2="205" y2="41"/><line x1="230" y1="95" x2="230" y2="46"/><line x1="255" y1="95" x2="255" y2="40"/><line x1="280" y1="95" x2="280" y2="43"/><line x1="305" y1="95" x2="305" y2="39"/><line x1="330" y1="95" x2="330" y2="47"/><line x1="355" y1="95" x2="355" y2="42"/><line x1="380" y1="95" x2="380" y2="50"/><line x1="405" y1="95" x2="405" y2="58"/>
  </g>
  <text x="230" y="115" font-size="8.5" fill="currentColor" text-anchor="middle">many OFDM subcarriers across ~2.3 kHz →</text>
  <text x="230" y="28" font-size="8.5" fill="currentColor" text-anchor="middle">each subcarrier: PSK/QAM, level set by SNR</text>
</svg>
<figcaption>VARA packs dozens of narrow OFDM subcarriers into the voice channel; each is modulated with PSK or QAM at a density the modem raises or lowers to track conditions.</figcaption>
</figure>

## Overview

VARA comes in two forms. **VARA HF** occupies about a 2.3 kHz SSB channel and is tuned for
the fading, multipath conditions of shortwave; **VARA FM** runs through a standard VHF/UHF FM
voice radio for high-speed local links. Both use OFDM: spreading the data across many
subcarriers makes each symbol long in time, so a multipath echo or a narrow interferer
damages only part of the signal rather than the whole. On top of that, VARA runs an
**adaptive ARQ** — it measures the channel, chooses a modulation/coding level for the
subcarriers, and retransmits any block that fails, so throughput scales smoothly from a few
hundred bits per second on a bad HF path up to tens of kilobits on a clean FM link.

## Technical characteristics

| Property | Value |
|----------|-------|
| Waveform | OFDM, adaptive PSK/QAM per subcarrier |
| Access | ARQ, half-duplex, connected sessions |
| VARA HF | ~2.3 kHz SSB; ~few hundred bps to ~7 kbps |
| VARA FM | Through an FM radio; up to ~25 kbps |
| Error control | FEC + CRC + ARQ retransmission |
| Implementation | Windows sound-card software (proprietary) |

## History

VARA was written by José Alberto Nieto Ros (EA5HVK) and released around 2017, with VARA FM
following soon after. By offering Pactor-like (and often better) throughput using only a
sound card and a licence fee to unlock full speed, it rapidly became the default HF and VHF
data mode for Winlink, especially in emergency-communications circles.[^wiki]

## Deployment

VARA is amateur-focused, used above all for Winlink email over HF and for high-speed local
VARA FM gateways. Its low cost of entry — a computer, a sound-card interface, and any SSB or
FM radio — is the main reason it has spread so widely for served-agency and disaster
communications.

## Decoding it with GopherTrunk

**GopherTrunk does not decode VARA.** It is a closed, proprietary OFDM modem for amateur HF/FM
data, entirely outside GopherTrunk's digital land-mobile trunking scope, and only the author's
software can transmit or receive it. GopherTrunk does handle the general primitives VARA is
built from — [OFDM](/reference/ofdm/) concepts,
[PSK](/reference/phase-shift-keying/)/[QAM](/reference/quadrature-amplitude-modulation/)
subcarriers — but not this specific waveform or its ARQ protocol.

## Sources

[^wiki]: [VARA (software)](https://en.wikipedia.org/wiki/VARA_(software)) — Wikipedia, for VARA's OFDM waveform, HF and FM variants, adaptive ARQ, throughput, and origin.
