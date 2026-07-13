---
slug: software-defined-radio
title: Software-defined radio (SDR)
entry_type: technology
category: sdr-dsp
description: Software-defined radio (SDR) implements traditionally hardware radio functions — tuning, filtering, demodulation — in software running on digitised IQ samples.
keywords: software defined radio, SDR, IQ, digital radio, RTL-SDR, direct sampling, GNU Radio, flexibility
aka: [software-defined radio, SDR]
autolink: true
infobox:
  - { label: Type, value: Radio architecture }
  - { label: Idea, value: Move tuning/demod into software }
  - { label: Hardware emits, value: IQ samples }
  - { label: Examples, value: RTL-SDR, HackRF, Airspy }
see_also: [iq-data, analog-to-digital-converter, superheterodyne-receiver, direct-sampling, direct-conversion-receiver, gnuradio, rtl-sdr, demodulation]
related_lessons:
  - { title: "What is software-defined radio?", url: /learn/rf-sdr/what-is-sdr/ }
related_reading:
  - { title: "SDR Internals, Part 1: What is software-defined radio?", url: /blog/deep-dives/sdr-internals-01-what-is-software-defined-radio/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://www.gnuradio.org/about/
---

**Software-defined radio** (**SDR**) moves the functions that were once fixed hardware —
tuning, filtering, [demodulation](/reference/demodulation/) — into **software** operating
on digitised [IQ samples](/reference/iq-data/).[^wiki] The hardware does only enough to
convert a slice of spectrum into numbers; every choice that used to be soldered in place —
which modulation, which channel spacing, which error-correcting code — becomes a line of
code that can be changed without touching the radio.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 110" role="img" aria-label="Antenna into SDR hardware which outputs IQ samples into software which outputs audio and data." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <path d="M40 78 v-28 m-9 0 l9 -12 l9 12" fill="none" stroke="currentColor" stroke-width="2"/><text x="40" y="96">antenna</text>
    <rect x="86" y="44" width="104" height="34" rx="6" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="138" y="58">SDR hardware</text><text x="138" y="71" font-size="8.5">tune · digitise</text>
    <rect x="262" y="44" width="116" height="34" rx="6" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="320" y="58">software</text><text x="320" y="71" font-size="8.5">filter · demod · decode</text>
    <rect x="450" y="44" width="78" height="34" rx="6" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.4"/><text x="489" y="64">audio &amp; data</text>
    <text x="226" y="38" font-size="8.5">IQ samples</text>
    <g stroke="currentColor" stroke-width="1.2"><line x1="52" y1="61" x2="85" y2="61"/><line x1="190" y1="61" x2="261" y2="61"/><line x1="378" y1="61" x2="449" y2="61"/></g>
  </g>
</svg>
<figcaption>An SDR moves tuning, filtering, and demodulation into software; the hardware just delivers IQ samples.</figcaption>
</figure>

## How it works

An SDR front-end does the analog work that cannot be done in software: an antenna and a
low-noise amplifier raise the signal above the noise, a mixer driven by a
[local oscillator](/reference/local-oscillator/) shifts the wanted band down toward
[baseband](/reference/baseband/), and an
[analog-to-digital converter](/reference/analog-to-digital-converter/) samples it into a
stream of complex [IQ](/reference/iq-data/) numbers. From that point on everything is
arithmetic. Software re-tunes by mixing the samples with a
[numerically controlled oscillator](/reference/numerically-controlled-oscillator/) inside a
[digital down-converter](/reference/digital-down-converter/), isolates a channel with a
[digital filter](/reference/digital-filter/), reduces the rate by
[decimation](/reference/decimation/), recovers symbols, and applies forward error
correction — all on the same samples, all changeable at run time.

Because the differences between radio systems now live in code rather than in metal, one
device can decode many incompatible protocols. The same RTL-SDR that follows a P25 control
channel can, with different software, demodulate broadcast FM, decode ADS-B aircraft
position reports, or track weather-satellite imagery. The physical hardware is close to
interchangeable; the intelligence is the program.

## Variants

SDR front-ends differ mainly in *how* they get a band down to something the ADC can
sample:

- **[Superheterodyne](/reference/superheterodyne-receiver/)** — one or more analog mixing
  stages down to a fixed intermediate frequency before digitising. Common in higher-end
  receivers for its selectivity.
- **[Direct-conversion](/reference/direct-conversion-receiver/) / [zero-IF](/reference/zero-if/)** —
  mix straight to baseband as quadrature IQ. Cheap and wideband, but prone to a DC spike
  and [IQ imbalance](/reference/iq-imbalance/). Many mass-market SDRs use a
  [low-IF](/reference/low-if/) offset to sidestep the DC problem.
- **[Direct sampling](/reference/direct-sampling/)** — no mixer at all; the ADC digitises
  the RF directly. Used by high-dynamic-range HF receivers and by RTL-SDR's HF "direct
  sampling" mode.

## In practice

The practical envelope of an SDR is set by three numbers: the
[sample rate](/reference/sample-rate/) (which fixes the instantaneous
[bandwidth](/reference/bandwidth/) you can watch at once, per the
[Nyquist theorem](/reference/nyquist-theorem/)), the ADC's
[dynamic range](/reference/dynamic-range/) (how far a weak signal can sit below a strong
neighbour before it is lost), and the CPU budget (wide captures produce a firehose of
samples). Free frameworks such as [GNU Radio](/reference/gnuradio/) make the software half
approachable by wiring DSP blocks into a flow graph; purpose-built decoders like
GopherTrunk trade that generality for a tuned, protocol-specific pipeline.

## Relevance to SDR

GopherTrunk is the software half of an SDR, specialised for digital trunked radio: it
consumes the IQ stream, channelises the control and voice channels, demodulates the C4FM /
CQPSK symbols, and decodes the trunking control messages. The hardware
(e.g. [RTL-SDR](/reference/rtl-sdr/), [Airspy](/reference/airspy/)) is almost
interchangeable. It decodes clear and scrambled traffic but not keyed encryption, and being
a receiver it does no transmitting or beamforming.

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, on the architecture that moves radio functions into software.
[^gnuradio]: [About GNU Radio](https://www.gnuradio.org/about/) — GNU Radio project, on the open-source toolkit for building SDR signal-processing flow graphs.
