---
slug: vestigial-sideband
title: Vestigial sideband (VSB)
entry_type: technology
category: modulation
description: "Vestigial sideband (VSB) transmits one full sideband plus a shaped remnant of the other, saving bandwidth while preserving low-frequency content — the basis of analog TV video and ATSC 8VSB."
keywords: vestigial sideband, VSB, 8VSB, analog television video, ATSC, NTSC, Nyquist slope, partial sideband
aka: [vestigial sideband, VSB, 8VSB]
autolink: true
infobox:
  - { label: Type, value: Analog/digital modulation (AM family) }
  - { label: Sends, value: One sideband + a vestige of the other }
  - { label: Examples, value: Analog TV video, ATSC 8VSB }
see_also: [single-sideband, double-sideband, amplitude-modulation, atsc-1, modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Vestigial_sideband
  - https://en.wikipedia.org/wiki/8VSB
---

**Vestigial sideband** (**VSB**) is an [amplitude-modulation](/reference/amplitude-modulation/)
variant that transmits **one complete sideband plus a shaped remnant — a "vestige" — of the
other**, together with a reduced carrier.[^wiki] It is a compromise between wasteful
[double sideband](/reference/double-sideband/) and hard-to-filter
[single sideband](/reference/single-sideband/): VSB gets most of SSB's bandwidth saving while
keeping the near-DC content that SSB filtering would destroy.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A VSB spectrum with a small carrier, a full upper sideband, and a sloped vestige of the lower sideband tapering off near the carrier." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="105" x2="440" y2="105" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="140" y1="105" x2="140" y2="55" stroke="currentColor" stroke-width="1.6"/>
  <text x="140" y="48" text-anchor="middle" font-size="9" fill="currentColor">carrier</text>
  <path d="M110 105 L140 60 L140 105 Z" fill="currentColor" fill-opacity="0.18"/>
  <text x="98" y="122" text-anchor="middle" font-size="8" fill="currentColor">vestige</text>
  <rect x="140" y="60" width="230" height="45" fill="currentColor" fill-opacity="0.22"/>
  <text x="255" y="122" text-anchor="middle" font-size="9" fill="currentColor">full upper sideband</text>
  <path d="M110 105 Q 140 80 140 60" fill="none" stroke="currentColor" stroke-width="1.3" stroke-dasharray="3 2"/>
  <text x="118" y="70" text-anchor="start" font-size="7" fill="currentColor">Nyquist slope</text>
</svg>
<figcaption>VSB keeps one sideband in full and lets the other roll off along a symmetric slope through the carrier.</figcaption>
</figure>

## How it works

Pure SSB removes an entire sideband with a sharp filter, but a real filter cannot have an
infinitely steep edge, and any attempt to cut right at the carrier also strips out the
signal's lowest frequencies — fatal for a video signal that carries DC brightness and
large flat areas. VSB solves this by shaping the transmit filter so its response falls off
gradually through the carrier along a **symmetric Nyquist slope**: the small amount of
lower-sideband energy that survives near the carrier exactly complements the upper-sideband
energy that is attenuated there. When the receiver's own filter has the matching
odd-symmetric roll-off, the two partial contributions add back to a flat response, so
low-frequency content is recovered without distortion. The vestige only needs to span the
first megahertz or so, so the total occupied bandwidth is far closer to SSB than to DSB.

The odd symmetry is the crucial trick. Around the carrier the transmit filter's response
is not a cliff but a straight-ish ramp from full transmission on the wanted-sideband side
down to zero on the vestige side, and it is antisymmetric about the carrier point: whatever
fraction of a low-frequency component is attenuated in the surviving sideband, the same
fraction survives in the vestige, so the two add back to unity. This is the same Nyquist
half-amplitude-at-the-edge condition that governs [pulse shaping](/reference/pulse-shaping/)
in digital systems, applied here to an analog spectrum. Get the receiver's complementary
shaping wrong and the near-DC content — flat, bright picture areas in video — comes back
distorted while the fine detail is fine.

## Variants

Analog television video used AM-VSB with the lower sideband trimmed and the upper sideband
full, fitting a ~6 MHz channel that would otherwise need far more for double-sideband video.
The digital successor, **8VSB**, keeps the same single-sideband-with-vestige spectral shape
but replaces the analog video with an eight-level amplitude-modulated digital symbol stream
plus a small pilot tone at the vestigial carrier frequency. 8VSB carries roughly 19.4 Mbit/s
in a 6 MHz channel and is the physical layer of North American
[ATSC 1.0](/reference/atsc-1/) digital television. (The later ATSC 3.0 standard abandons VSB
for OFDM.)

## Relevance to SDR

VSB is the signal you see when a wideband SDR sweeps the old TV bands: a strong pilot at the
lower channel edge with energy trailing off across the channel is the 8VSB signature. Demod
requires a pilot-locked carrier recovery, an adaptive equalizer to fight the long multipath
echoes that plague terrestrial TV, and a trellis decoder. GopherTrunk is a land-mobile
trunking decoder and does not process television signals, so VSB is out of its scope; its
interest here is conceptual, as the middle point on the DSB → VSB → SSB bandwidth-versus-
filtering spectrum, and as a reminder that practical modulation choices are driven as much by
realizable filters as by theory.

## In practice

Analog NTSC video fit a full-detail luminance signal, a color subcarrier, and an
FM-modulated audio carrier into a single 6 MHz channel precisely because the video used
VSB rather than double-sideband — DSB video would have needed roughly 8 MHz for the picture
alone. The digital 8VSB successor keeps the vestigial shape but is notoriously demanding on
the receiver: terrestrial multipath produces long delayed echoes that smear the eight-level
symbols, so an 8VSB demodulator leans on a long adaptive
[equalizer](/reference/decision-feedback-equalizer/) and a trellis decoder to recover the
data, and the small pilot at the vestigial carrier frequency exists specifically to give the
receiver a robust phase reference to lock onto before the equalizer converges. That
sensitivity to multipath was a recurring complaint against 8VSB, and it is one reason
[ATSC 3.0](/reference/atsc-3/) abandoned VSB entirely in favor of OFDM.

## Sources

[^wiki]: [Vestigial sideband](https://en.wikipedia.org/wiki/Vestigial_sideband) — Wikipedia, for the partial-sideband definition, Nyquist slope, and TV/8VSB applications.
