---
slug: single-sideband
title: Single sideband (SSB)
entry_type: technology
category: modulation
description: Single sideband (SSB) is an efficient AM variant that suppresses the carrier and one sideband, halving bandwidth and concentrating power for long-distance HF voice.
keywords: single sideband, SSB, USB, LSB, suppressed carrier, HF voice, amateur, Hilbert transform, BFO, weaver method
aka: [single sideband, SSB]
autolink: true
infobox:
  - { label: Type, value: Analog modulation (AM variant) }
  - { label: Sends, value: One sideband, suppressed carrier }
  - { label: Used for, value: Long-distance HF voice, amateur }
see_also: [amplitude-modulation, modulation, frequency-bands, ionospheric-propagation, double-sideband, vestigial-sideband, carrier-wave, hilbert-transform]
related_lessons:
  - { title: "Analog modulation — AM, FM, SSB", url: /learn/rf-sdr/analog-modulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Single-sideband_modulation
  - https://en.wikipedia.org/wiki/Hilbert_transform
---

**Single sideband** (**SSB**) is a refined form of
[amplitude modulation](/reference/amplitude-modulation/) that removes the
[carrier](/reference/carrier-wave/) and one of the two redundant sidebands,
transmitting only **one sideband** — upper (USB) or lower (LSB).[^wiki] Because ordinary
AM spends half its bandwidth and most of its power on redundant information, SSB is the
efficiency-maximising member of the AM family and the backbone of long-distance HF voice.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A double-sideband AM spectrum with carrier and two sidebands on the left, an arrow, and an SSB spectrum with only one sideband on the right." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.4">
    <line x1="40" y1="100" x2="170" y2="100" stroke-opacity="0.4"/>
    <line x1="105" y1="100" x2="105" y2="40"/>
    <rect x="70" y="70" width="20" height="30" fill="currentColor" fill-opacity="0.2"/>
    <rect x="120" y="70" width="20" height="30" fill="currentColor" fill-opacity="0.2"/>
  </g>
  <text x="105" y="118" text-anchor="middle" font-size="9" fill="currentColor">AM: carrier + 2 sidebands</text>
  <line x1="195" y1="70" x2="235" y2="70" stroke="currentColor" marker-end="url(#ssbar)"/>
  <g stroke="currentColor" stroke-width="1.4">
    <line x1="280" y1="100" x2="430" y2="100" stroke-opacity="0.4"/>
    <rect x="360" y="70" width="20" height="30" fill="currentColor" fill-opacity="0.3"/>
  </g>
  <text x="355" y="118" text-anchor="middle" font-size="9" fill="currentColor">SSB: one sideband only</text>
  <defs><marker id="ssbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>SSB removes the carrier and one redundant sideband — half the bandwidth, all the power on the information.</figcaption>
</figure>

## How it works

Standard AM produces two sidebands that are mirror images of each other and a carrier
that conveys nothing but a frequency reference. All the message information is present in
*either* sideband alone, so SSB throws away the carrier and one sideband entirely. The
result uses about **half the bandwidth** of AM — roughly 2.7 kHz for a voice channel
instead of 6 kHz — and, with no power wasted on a carrier or a redundant sideband, puts
essentially all transmitter power into the information. That efficiency lets modest
transmitters reach across continents on [HF](/reference/frequency-bands/) via
[ionospheric propagation](/reference/ionospheric-propagation/).

Two classic methods generate SSB. The *filter method* creates a double-sideband
suppressed-carrier signal in a balanced mixer, then uses a very sharp crystal or
mechanical filter to keep just one sideband. The *phasing method* uses a
[Hilbert transform](/reference/hilbert-transform/) — a 90° broadband phase shift — so
that the unwanted sideband cancels while the wanted one adds; this is the form an SDR
implements naturally, since generating a complex (analytic) signal and shifting it is
just arithmetic on [IQ](/reference/iq-data/) samples.

The cost of suppressing the carrier is that the receiver must supply its own. A beat
frequency oscillator (BFO) reinserts a local carrier at exactly the right frequency to
demodulate the sideband. If that reinjected carrier is off by even 50–100 Hz the voice
sounds unnaturally high- or low-pitched ("Donald Duck"), which is why SSB tuning must be
precise and why USB versus LSB matters — pick the wrong sideband and the audio is
gibberish. By convention, amateur HF below 10 MHz uses LSB and above uses USB.

## Variants

SSB sits on a spectrum of carrier- and sideband-trimming schemes.
[Double-sideband suppressed carrier](/reference/double-sideband/) (DSB-SC) removes the
carrier but keeps both sidebands. [Vestigial sideband](/reference/vestigial-sideband/)
(VSB) keeps one full sideband plus a filtered remnant of the other, used where the
message extends down to DC (analog TV video). Variants of SSB itself include AME
(amplitude-modulation equivalent, with a reduced pilot carrier for easier tuning) and
independent-sideband (ISB), which sends different information on each sideband to carry
two channels in one AM-width slot.

## Relevance to SDR

SSB is where SDRs shine: a software receiver can generate the analytic signal, shift it,
and low-pass filter to isolate a single sideband with no analog filter at all, and can
switch USB/LSB or fine-tune the reinserted carrier with a slider. Receiving SSB needs an
HF-capable SDR — an [upconverter](/reference/upconverter/) ahead of an RTL dongle, a
[direct-sampling](/reference/direct-sampling/) receiver, or a dedicated HF SDR like the
[Airspy HF+](/reference/airspy-hf-plus/) — plus accurate tuning to reinsert the missing
carrier. GopherTrunk targets digital trunking rather than HF voice, so SSB is out of its
decode scope, but the same phasing/analytic-signal math underpins how it forms complex
baseband.

## Sources

[^wiki]: [Single-sideband modulation](https://en.wikipedia.org/wiki/Single-sideband_modulation) — Wikipedia, for the suppressed-carrier definition, USB/LSB, and bandwidth/power efficiency.
[^hilbert]: [Hilbert transform](https://en.wikipedia.org/wiki/Hilbert_transform) — Wikipedia, for the 90° phase-shift used by the phasing method of SSB generation.
