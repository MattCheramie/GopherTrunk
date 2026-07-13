---
slug: cubicsdr
title: CubicSDR
entry_type: technology
category: sdr-software
description: "CubicSDR is a cross-platform, open-source SDR receiver with a distinctive 3D-style spectrum-and-waterfall view, built on SoapySDR for broad hardware support."
keywords: CubicSDR, cross-platform SDR, SoapySDR receiver, waterfall, wxWidgets, liquid-dsp, demodulator, open source SDR
aka: [CubicSDR]
autolink: true
infobox:
  - { label: Type, value: Desktop SDR receiver app }
  - { label: Platform, value: "Windows, Linux, macOS" }
  - { label: Idea, value: "SoapySDR-driven receiver with waterfall UI" }
see_also: [software-defined-radio, soapysdr, gqrx, sdr-sharp, waterfall-display, iq-data]
cite_urls:
  - https://en.wikipedia.org/wiki/CubicSDR
  - https://cubicsdr.com/
---

**CubicSDR** is a free, open-source, cross-platform [software-defined radio](/reference/software-defined-radio/)
receiver known for its combined spectrum-and-[waterfall](/reference/waterfall-display/) view
and for using [SoapySDR](/reference/soapysdr/) as its hardware layer, which gives it wide
device support out of the box.[^proj] It runs on Windows, Linux, and macOS with the same
interface, and lets a listener drag across the spectrum to tune and demodulate analog signals
in real time.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="CubicSDR software stack: a SoapySDR device layer feeds a liquid-dsp processing core, which drives a wxWidgets spectrum and waterfall interface." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="csar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="14" y="50" width="96" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="62" y="62">SoapySDR</text><text x="62" y="72">device layer</text>
    <rect x="150" y="50" width="96" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="198" y="62">liquid-dsp</text><text x="198" y="72">core</text>
    <rect x="286" y="50" width="160" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="366" y="62">spectrum + waterfall</text><text x="366" y="72">(wxWidgets UI)</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="110" y1="65" x2="148" y2="65" marker-end="url(#csar)"/>
    <line x1="246" y1="65" x2="284" y2="65" marker-end="url(#csar)"/>
  </g>
</svg>
<figcaption>CubicSDR layers a liquid-dsp processing core on the SoapySDR device abstraction, presenting a cross-platform spectrum-and-waterfall interface.</figcaption>
</figure>

## How it works

CubicSDR opens its front end through [SoapySDR](/reference/soapysdr/), so any radio with a
Soapy module — RTL-SDR, [Airspy](/reference/airspy/), [HackRF](/reference/hackrf/),
[LimeSDR](/reference/limesdr/), [SDRplay](/reference/sdrplay-rsp1a/), and networked sources —
works without app-specific drivers. The incoming [IQ](/reference/iq-data/) stream is processed
with the **liquid-dsp** library: an FFT produces the spectrum and waterfall, and a
[digital down-converter](/reference/digital-down-converter/) plus demodulator extract the
signal under the tuning cursor. Supported modes include WFM (with stereo),
NFM, AM, [SSB](/reference/single-sideband/), and CW.

The interface, built with **wxWidgets** for portability, emphasizes the visual: a large,
smoothly scrolling waterfall that the user drags across to retune, with the passband of the
receive filter shown directly on the spectrum. Multiple demodulators can be placed at once so
several signals stay visible, and audio can be routed per demodulator. CubicSDR can also act
as a SoapySDR **network client**, receiving IQ from a remote server so the radio can sit at
the antenna while the application runs elsewhere.

## Relevance to SDR

CubicSDR occupies the same niche as [Gqrx](/reference/gqrx/) and [SDR#](/reference/sdr-sharp/):
a friendly general-coverage receiver for browsing and listening to analog signals across the
spectrum. Its selling points are true cross-platform parity and the SoapySDR foundation, which
means the same build works with almost any front end and can talk to remote radios. Like other
general receivers, its demodulators target analog modes; decoding digital-voice or trunked
systems requires separate tools fed from its audio or IQ output.

**GopherTrunk** is an independent project unrelated to CubicSDR in code. Functionally they
overlap only at the front end and the "see the signal" stage: GopherTrunk is a headless,
pure-Go trunking scanner that consumes IQ and decodes digital protocols itself, whereas
CubicSDR is a GUI receiver for manual tuning of analog modes. Because both can draw on the
same SoapySDR-supported hardware (and CubicSDR can record IQ), CubicSDR is a handy companion
for locating a control channel visually or producing capture files to replay through
GopherTrunk offline.

## Sources

[^proj]: [CubicSDR](https://cubicsdr.com/) — the official project site and its [source repository](https://github.com/cjcliffe/CubicSDR), documenting the SoapySDR device layer, the liquid-dsp core, the wxWidgets interface, and supported demodulation modes.
