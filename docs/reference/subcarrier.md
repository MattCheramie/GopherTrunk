---
slug: subcarrier
title: Subcarrier
entry_type: term
category: modulation
description: "A subcarrier is a secondary carrier modulated onto a main signal's baseband, carrying an extra channel such as FM stereo difference audio or the RDS data stream."
keywords: subcarrier, SCA, RDS subcarrier, stereo pilot, 19 kHz pilot, 38 kHz, 57 kHz, multiplex baseband, frequency-division multiplex
aka: [subcarrier, SCA, subsidiary carrier]
autolink: true
infobox:
  - { label: Symbol, value: "fsc" }
  - { label: Unit, value: "Hertz (e.g. 38 kHz, 57 kHz)" }
  - { label: Type, value: "Secondary carrier within a baseband" }
see_also: [rds, broadcast-fm, carrier-wave, frequency-modulation, modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Subcarrier
  - https://en.wikipedia.org/wiki/Radio_Data_System
---

**A subcarrier** is a secondary [carrier](/reference/carrier-wave/) that is modulated with its
own information and then placed within the baseband of a larger signal, which is in turn
modulated onto the main RF carrier.[^wiki] It is a way to stack several independent channels into
one transmission by frequency-division multiplexing at baseband — one primary channel near DC and
one or more subcarriers at higher baseband frequencies, each demodulated separately at the
receiver.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An FM broadcast baseband spectrum showing mono audio near DC, a 19 kHz pilot tone, a 38 kHz stereo difference subcarrier, and a 57 kHz RDS subcarrier." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="115" x2="440" y2="115" stroke="currentColor" stroke-opacity="0.4"/>
  <rect x="40" y="60" width="60" height="55" fill="currentColor" fill-opacity="0.2"/>
  <text x="70" y="132" text-anchor="middle" font-size="7" fill="currentColor">mono L+R</text>
  <line x1="170" y1="115" x2="170" y2="55" stroke="currentColor" stroke-width="1.6"/>
  <text x="170" y="132" text-anchor="middle" font-size="7" fill="currentColor">19k pilot</text>
  <rect x="210" y="72" width="70" height="43" fill="currentColor" fill-opacity="0.2"/>
  <text x="245" y="132" text-anchor="middle" font-size="7" fill="currentColor">38k L−R</text>
  <rect x="330" y="86" width="34" height="29" fill="currentColor" fill-opacity="0.2"/>
  <text x="347" y="132" text-anchor="middle" font-size="7" fill="currentColor">57k RDS</text>
  <text x="440" y="112" text-anchor="end" font-size="8" fill="currentColor">baseband freq →</text>
</svg>
<figcaption>Subcarriers stack extra channels above the main audio in the baseband; the FM multiplex carries stereo and RDS as subcarriers.</figcaption>
</figure>

## How it works

The transmitter builds a **composite baseband**: the primary channel occupies the lowest
frequencies, and each subcarrier is a tone placed higher up the baseband and modulated (in AM,
FM, or a suppressed-carrier scheme) with additional data. That whole composite is then applied to
the main RF carrier as if it were ordinary audio. At the receiver, demodulating the main carrier
regenerates the composite baseband; a bank of filters then splits out each subcarrier, and each is
demodulated in turn. Because a suppressed-carrier subcarrier saves power and spectrum, systems
often send only a low-level **pilot tone** so the receiver can regenerate the missing subcarrier
frequency and phase coherently.

The canonical example is the [FM broadcast](/reference/broadcast-fm/) multiplex. Mono audio (L+R)
sits from 30 Hz to 15 kHz; a 19 kHz **pilot** marks a stereo station; the stereo difference signal
(L−R) is sent as a double-sideband suppressed-carrier subcarrier centered at 38 kHz (twice the
pilot), which the receiver reconstructs by doubling the pilot; and [RDS](/reference/rds/) data
rides a further subcarrier at 57 kHz (three times the pilot). Additional SCA subcarriers around
67 kHz and 92 kHz have historically carried background music and reading services.

Notice the arithmetic: the 38 kHz stereo subcarrier is exactly twice the 19 kHz pilot and the
57 kHz RDS subcarrier exactly three times it. This is deliberate — deriving every subcarrier
from harmonics of one pilot lets the receiver regenerate all of them from a single locked
reference, and it keeps the subcarriers from drifting relative to one another. It is the analog
ancestor of the coherent-reference thinking that pervades digital systems.

## Variants

Subcarriers appear far beyond FM stereo. Analog color television carried the chrominance
(color) information on a suppressed subcarrier (about 3.58 MHz in NTSC) sitting inside the
luminance band, itself a small-scale frequency-division multiplex. Telemetry and instrumentation
systems have long used FM/FM schemes in which many sensor channels each frequency-modulate their
own subcarrier and the whole stack then modulates the RF carrier. And in the digital world the
idea generalizes completely: [OFDM](/reference/ofdm/) splits a channel into hundreds or thousands
of closely spaced, individually modulated subcarriers, the technique behind Wi-Fi, LTE, 5G NR,
DAB, and DVB-T. The humble FM pilot-and-subcarrier scheme and a 5G resource grid are points on
the same continuum.

## Relevance to SDR

Subcarriers are where a lot of "hidden" data lives in an otherwise analog broadcast, and they are
a favorite SDR target. A software receiver that FM-demodulates a broadcast station gets the whole
composite baseband for free and can then dig out the 57 kHz [RDS](/reference/rds/) subcarrier to
display station name, program type, and RadioText — a common demonstration project. Decoding a
suppressed-carrier subcarrier like the 38 kHz stereo channel requires regenerating it from the
pilot, exactly the coherent-detection problem that
[double-sideband](/reference/double-sideband/) suppressed-carrier signals pose. GopherTrunk focuses
on digital land-mobile trunking rather than broadcast, so it does not decode FM subcarriers, but
the frequency-division-multiplex idea recurs throughout RF, and the pilot-regeneration technique is
the same math its digital carriers use for synchronization.

Reading a subcarrier well depends on cleanly recovering the whole composite baseband first, which
in turn needs enough receiver bandwidth and a low-distortion FM demodulator — a subcarrier riding
at 57 kHz is useless if the receiver's channel filter rolls off at 20 kHz. This is why
subcarrier decoding rewards a wideband front end: the SDR must pass the full multiplex, not just
the audio a listener hears. The same requirement applies in reverse as a warning about aliasing —
a subcarrier sits at a real baseband frequency and must be sampled fast enough to represent it, or
it folds back on top of the wanted audio.

## Sources

[^wiki]: [Subcarrier](https://en.wikipedia.org/wiki/Subcarrier) — Wikipedia, for the definition and the FM stereo/RDS multiplex example.
