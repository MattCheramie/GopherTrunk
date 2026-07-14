---
slug: software-defined-receiver
title: Software-defined receiver (SDR receiver)
entry_type: concept
category: sdr-app-building
description: A software-defined receiver is the end-to-end software chain that turns raw IQ samples from an SDR front end into decoded data, doing tune, filter, demod, and decode in code.
keywords: software-defined receiver, SDR receiver, receive chain, IQ processing, demodulator software, digital receiver, DSP pipeline, P25 DMR decoder, pure-Go receiver
aka: [SDR receiver, software radio receiver, digital receiver]
autolink: true
infobox:
  - { label: Type, value: RX software architecture }
  - { label: Input, value: Raw IQ samples }
  - { label: Output, value: Decoded bits / messages }
see_also: [receiver-chain, software-defined-radio, demodulation, digital-down-converter, iq-data, real-time-dsp]
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://en.wikipedia.org/wiki/Digital_down_converter
---

A **software-defined receiver** is the end-to-end *software* chain that turns the raw
[IQ](/reference/iq-data/) sample stream from a radio front end into decoded data —
performing tuning, filtering, [demodulation](/reference/demodulation/), and decoding in code
rather than in fixed analog or digital hardware.[^wiki] Where a classic superheterodyne set
buries those steps in tuned circuits and discriminators, the software-defined receiver reads
that a general-purpose CPU is fast enough to do the same arithmetic on sampled numbers, so
the "radio" becomes a program you can change without touching a soldering iron.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="An antenna and SDR front end feed IQ samples into a software block that filters, demodulates, and decodes to produce output messages." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="sdrxar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M28 60 L28 30 M20 30 L36 30" stroke="currentColor" stroke-width="1.4" fill="none"/>
  <rect x="52" y="45" width="60" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="82" y="63" font-size="8" fill="currentColor" text-anchor="middle">front end</text>
  <rect x="150" y="30" width="230" height="60" rx="6" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <text x="265" y="26" font-size="8" fill="currentColor" text-anchor="middle">software</text>
  <rect x="162" y="48" width="46" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="185" y="63" font-size="7" fill="currentColor" text-anchor="middle">filter</text>
  <rect x="214" y="48" width="46" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="237" y="63" font-size="7" fill="currentColor" text-anchor="middle">demod</text>
  <rect x="266" y="48" width="46" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="289" y="63" font-size="7" fill="currentColor" text-anchor="middle">decode</text>
  <line x1="112" y1="60" x2="149" y2="60" stroke="currentColor" stroke-width="1.1" marker-end="url(#sdrxar)"/>
  <text x="130" y="52" font-size="7" fill="currentColor" text-anchor="middle">IQ</text>
  <line x1="380" y1="60" x2="440" y2="60" stroke="currentColor" stroke-width="1.1" marker-end="url(#sdrxar)"/>
  <text x="410" y="52" font-size="7" fill="currentColor" text-anchor="middle">msgs</text>
</svg>
<figcaption>A software-defined receiver: the SDR front end digitises; all radio work happens in software.</figcaption>
</figure>

## How it works

The boundary between hardware and software is drawn as early as the ADC allows. A tuner
mixes the signal of interest near [baseband](/reference/baseband/), an
[analog-to-digital converter](/reference/analog-to-digital-converter/) samples it, and from
that point on the receiver is code operating on a stream of complex numbers:

- **Tune and channel-select.** A [digital down converter](/reference/digital-down-converter/)
  shifts the wanted channel to 0 Hz and a low-pass filter removes everything else.
- **Resample.** The stream is decimated to a rate matched to the signal's symbol rate — a
  handful of samples per symbol — which both shrinks the work and sizes every later stage.
- **Demodulate.** The [demodulator](/reference/demodulation/) recovers the modulating
  quantity (frequency for FSK/C4FM, phase for PSK), producing a symbol-bearing waveform.
- **Synchronise and decode.** [Clock recovery](/reference/clock-recovery/) slices the
  waveform into symbols, [frame synchronization](/reference/frame-synchronization/) finds
  message boundaries, and error-correction turns symbols into bytes.

Because these stages are software, one program can host many of them at once, retune
instantly, and be tested against recorded files instead of live RF.

## In practice

The decisive property of a good software-defined receiver is that its decode path is
**rate-invariant**: the DDC normalises every capture to a fixed per-protocol channel rate, so
the demodulator and its loops behave identically whether the front end sampled at 2 MS/s or
10 MS/s. That lets the whole receiver be validated deterministically from
[IQ recordings](/reference/iq-recording-playback/), with no radio attached — the same bytes
in always produce the same bytes out.

## Relevance to SDR

The software-defined receiver *is* the application when you build SDR software: cheap dongles
like the [RTL-SDR](/reference/rtl-sdr/) and wideband front ends like the
[Airspy](/reference/airspy/) provide only samples, and everything that makes them a scanner,
a decoder, or an ADS-B tracker lives in the receiver code. **GopherTrunk** is exactly such a
program — a pure-Go software-defined receiver whose front end (RTL-SDR, Airspy, others) hands
it IQ, and whose `internal/scanner/ccdecoder` down-converter, per-protocol demodulators, and
framing decode P25, DMR, NXDN, TETRA and more entirely in software. Its
[receiver chain](/reference/receiver-chain/) is the concrete instance of the abstract stages
above; it demodulates and decodes clear and scrambled traffic but does not break keyed
encryption, and it is a receiver only — it has no transmit path.

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, on moving receiver functions from hardware into software operating on sampled signals.
[^ddc]: [Digital down converter](https://en.wikipedia.org/wiki/Digital_down_converter) — Wikipedia, on the tune-and-decimate front stage of a software receiver.
