---
slug: ctcss
title: CTCSS
entry_type: technology
category: modulation
description: "CTCSS is a sub-audible tone (67-254 Hz) transmitted under FM voice so a receiver's squelch opens only for calls carrying the matching tone."
keywords: CTCSS, PL, Private Line, sub-audible tone, tone squelch, continuous tone-coded squelch, 100 Hz, Channel Guard, shared repeater
aka: [CTCSS, PL, tone squelch, Continuous Tone-Coded Squelch System]
autolink: true
infobox:
  - { label: Type, value: "Analog sub-audible signaling" }
  - { label: Idea, value: "Low-frequency tone gates the squelch" }
  - { label: Examples, value: "67-254 Hz PL tones on FM repeaters" }
see_also: [dcs, squelch, frequency-modulation, subcarrier]
cite_urls:
  - https://en.wikipedia.org/wiki/Continuous_Tone-Coded_Squelch_System
---

**CTCSS** (Continuous Tone-Coded Squelch System, and known by trade names such as Motorola's
**PL** / Private Line) is a scheme that transmits a continuous **sub-audible tone** — a single
pure tone between about 67 and 254 Hz — underneath [FM](/reference/frequency-modulation/) voice so
that a receiver's [squelch](/reference/squelch/) opens only when it detects the matching tone.[^wiki]
It lets several user groups share one radio channel without hearing each other's traffic.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A baseband spectrum with a narrow low-frequency CTCSS tone below 300 Hz and the voice band from 300 Hz to 3 kHz above it, with the tone region shaded separately." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="115" x2="440" y2="115" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="55" y1="115" x2="55" y2="55" stroke="currentColor" stroke-width="1.8"/>
  <text x="55" y="132" text-anchor="middle" font-size="8" fill="currentColor">CTCSS tone</text>
  <text x="55" y="48" text-anchor="middle" font-size="7" fill="currentColor">67-254 Hz</text>
  <path d="M120 115 Q 170 70 230 78 Q 300 90 360 82 Q 400 76 420 115" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/>
  <text x="270" y="132" text-anchor="middle" font-size="8" fill="currentColor">voice 300 Hz - 3 kHz</text>
  <line x1="90" y1="115" x2="90" y2="40" stroke="currentColor" stroke-opacity="0.3" stroke-dasharray="3 2"/>
  <text x="90" y="35" text-anchor="middle" font-size="7" fill="currentColor">high-pass split</text>
</svg>
<figcaption>The CTCSS tone sits below the 300 Hz voice band, so a simple filter separates the tone from the audio a listener hears.</figcaption>
</figure>

## How it works

The transmitting radio adds a low-level continuous tone to the audio that frequency-modulates the
carrier, at a modest deviation (typically ~500-800 Hz, well below the voice peaks). Because the tone
sits below the 300 Hz low end of communications voice, the receiver can split it off with a simple
high-pass/low-pass pair: voice above 300 Hz goes to the speaker, and the sub-audible region below it
goes to a tone detector. If the detected tone matches the programmed frequency from the standardized
set (there are around 50 defined tones, e.g. 100.0, 141.3, 203.5 Hz), the receiver's squelch opens;
otherwise the audio stays muted even though a signal is fully present and quieting the noise squelch.
Being continuous, the tone is present the whole time the carrier is keyed, so the squelch can close
promptly when it disappears.

CTCSS does not provide privacy — anyone with carrier squelch or a scanner hears everything on the
channel — it only manages who *your* radio bothers you with. It is also widely used as a **repeater
access tone**: a repeater keys up only for users transmitting the correct CTCSS, rejecting interference
and distant co-channel signals.

## Relevance to SDR

For a scanner, CTCSS is a useful sorting and identification tool. A software receiver that FM-demodulates
a channel can run a narrow tone detector (a Goertzel or a short bank of bandpass filters) over the
sub-audible region to read out which tone is present, letting the user log or filter transmissions by
group even on a shared channel. This is exactly analogous to the digital
[DCS](/reference/dcs/) code that serves the same purpose with a low-speed data burst rather than a tone.
CTCSS belongs to the analog FM world; GopherTrunk targets digital trunking, where group separation is
handled at the protocol layer by talkgroup IDs rather than sub-audible tones, so it does not decode CTCSS
in its trunking path — but the same tone-detection technique applies when working with conventional FM
audio.

## Sources

[^wiki]: [Continuous Tone-Coded Squelch System](https://en.wikipedia.org/wiki/Continuous_Tone-Coded_Squelch_System) — Wikipedia, for the sub-audible tone set, squelch gating, and repeater-access use.
