---
slug: gqrx
title: Gqrx
entry_type: technology
category: sdr-software
description: "Gqrx is an open-source, cross-platform SDR receiver built on GNU Radio and Qt, widely used on Linux and macOS for general listening."
keywords: Gqrx, Qt SDR receiver, GNU Radio receiver, Linux SDR, macOS SDR, waterfall, remote control, open source receiver
aka: [Gqrx]
autolink: true
infobox:
  - { label: Type, value: Desktop SDR receiver app }
  - { label: Platform, value: "Linux, macOS (Qt)" }
  - { label: Idea, value: "GNU Radio DSP with a Qt GUI" }
see_also: [software-defined-radio, soapysdr, gnuradio, sdr-sharp, waterfall-display, iq-data]
cite_urls:
  - https://en.wikipedia.org/wiki/Gqrx
  - https://gqrx.dk/
---

**Gqrx** is a free, open-source [software-defined radio](/reference/software-defined-radio/)
receiver application that pairs a [GNU Radio](/reference/gnuradio/) signal-processing back end
with a **Qt** graphical interface.[^proj] It is one of the most popular general-purpose SDR
receivers on Linux and macOS, offering a spectrum and
[waterfall display](/reference/waterfall-display/), a set of analog demodulators, and
recording and remote-control features. Gqrx fills much the same role on those platforms that
[SDR#](/reference/sdr-sharp/) does on Windows.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 132" role="img" aria-label="Gqrx architecture: an SDR device feeds GNU Radio DSP blocks, whose output is drawn by a Qt front end showing spectrum, waterfall, and audio controls." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="gqar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="10" y="52" width="80" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="50" y="63">SDR device</text><text x="50" y="73">(SoapySDR)</text>
    <rect x="140" y="52" width="90" height="28" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="185" y="63">GNU Radio</text><text x="185" y="73">DSP</text>
    <rect x="290" y="20" width="150" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="365" y="37">spectrum + waterfall</text>
    <rect x="290" y="86" width="150" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="365" y="103">demod + audio</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="90" y1="66" x2="138" y2="66" marker-end="url(#gqar)"/>
    <line x1="230" y1="60" x2="288" y2="34" marker-end="url(#gqar)"/>
    <line x1="230" y1="72" x2="288" y2="98" marker-end="url(#gqar)"/>
  </g>
  <text x="365" y="66" text-anchor="middle" font-size="8" fill="currentColor">Qt front end</text>
</svg>
<figcaption>Gqrx wraps a GNU Radio DSP chain in a Qt interface, driving many radios through SoapySDR and gr-osmosdr.</figcaption>
</figure>

## How it works

Under the hood, Gqrx constructs a [GNU Radio](/reference/gnuradio/) flowgraph: a hardware
source (via `gr-osmosdr` or [SoapySDR](/reference/soapysdr/)) feeds a
[digital down-converter](/reference/digital-down-converter/) that shifts and decimates the
tuned slice of spectrum, followed by mode-specific demodulators and an audio sink. The Qt
layer draws the FFT-based spectrum and [waterfall](/reference/waterfall-display/) and exposes
the controls — center frequency, gain, filter width, squelch, and the demodulator choice
(WFM stereo, NFM, AM, [SSB](/reference/single-sideband/), CW).

Because the DSP is standard GNU Radio, Gqrx inherits broad hardware support: RTL-SDR,
[Airspy](/reference/airspy/), [HackRF](/reference/hackrf/), [bladeRF](/reference/bladerf/),
[USRP](/reference/usrp-ettus/), and anything with a SoapySDR module. Useful conveniences
include baseband and audio [IQ](/reference/iq-data/) recording, an AGC, a noise blanker, and a
**remote-control** TCP interface (compatible with the `rigctld` protocol) so other software or
scripts can retune Gqrx and read its state. That remote interface makes Gqrx scriptable — for
example, an external program can command it to step across a band.

## Relevance to SDR

Gqrx is a staple general-coverage receiver for the Linux and macOS SDR community: it is the
tool many reach for to tune broadcast, air band, marine, ham, and utility signals, and to
confirm a new dongle is working. Being open source and GNU Radio-based, it is also a common
starting point for people who later dig into the underlying flowgraph. Like most such
receivers, its built-in demodulators cover analog modes; digital-voice and trunking work is
left to dedicated decoders, though Gqrx's audio or IQ output can be piped into one.

**GopherTrunk** is independent of Gqrx and shares no code with it. GopherTrunk is a headless,
pure-Go trunking decoder with its own DSP; Gqrx is a GUI receiver built on GNU Radio. The
overlap is at the hardware and the "look at the spectrum" stage: an operator can use Gqrx to
find and inspect a control-channel carrier on the waterfall, then hand that frequency to
GopherTrunk to follow the system. Gqrx's IQ recordings can also serve as capture files for
offline testing of a decoder.

## Sources

[^proj]: [Gqrx](https://gqrx.dk/) — the official project site and documentation, describing the GNU Radio + Qt architecture, supported hardware, demodulators, and the remote-control interface.
