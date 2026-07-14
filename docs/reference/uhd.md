---
slug: uhd
title: UHD (USRP Hardware Driver)
entry_type: technology
category: sdr-frameworks
description: "UHD is Ettus Research's open-source driver and C++/Python API for the USRP family of software-defined radios, giving every host application a uniform way to tune, stream, and time-align samples."
keywords: UHD, USRP Hardware Driver, Ettus Research, USRP, uhd_usrp_source, multi_usrp, RFNoC, National Instruments, SDR driver, host API
aka: [UHD, USRP Hardware Driver]
autolink: true
infobox:
  - { label: Type, value: SDR device driver + host API }
  - { label: Vendor, value: Ettus Research (NI) }
  - { label: Drives, value: USRP family (B/N/X/E series) }
see_also: [usrp-ettus, gnuradio, soapysdr, gr-osmosdr, iq-data, sample-rate]
cite_urls:
  - https://en.wikipedia.org/wiki/Universal_Software_Radio_Peripheral
  - https://files.ettus.com/manual/
---

**UHD** (the **USRP Hardware Driver**) is the free, open-source driver and host-side API
that connects a computer to Ettus Research's [USRP](/reference/usrp-ettus/) family of
[software-defined radios](/reference/software-defined-radio/).[^uhd] It is the single
supported way to move [IQ](/reference/iq-data/) samples between a host program and a USRP,
whether that USRP is a $300 bus-powered B200 or a rack-mounted X310 on 10-gigabit Ethernet.
Applications link against UHD's library, and UHD hides the differences between USRP models
behind one consistent set of calls.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A host application calling the UHD library, which speaks over USB or Ethernet to a USRP whose FPGA and RF daughterboard handle sampling and tuning." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="uhdar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="8" y="52" width="92" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="54" y="66">host app</text><text x="54" y="77">(GNU Radio…)</text>
    <rect x="132" y="52" width="80" height="34" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="172" y="72">UHD lib</text>
    <rect x="244" y="52" width="86" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="287" y="66">USRP FPGA</text><text x="287" y="77">(DDC/DUC)</text>
    <rect x="362" y="52" width="86" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="405" y="66">RF front end</text><text x="405" y="77">(ADC/DAC)</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="100" y1="69" x2="130" y2="69" marker-end="url(#uhdar)"/>
    <line x1="212" y1="69" x2="242" y2="69" marker-end="url(#uhdar)"/>
    <line x1="330" y1="69" x2="360" y2="69" marker-end="url(#uhdar)"/>
  </g>
  <text x="172" y="42" text-anchor="middle" font-size="8" fill="currentColor">USB / Ethernet transport</text>
  <text x="230" y="20" text-anchor="middle" font-size="9" fill="currentColor">UHD: one API from host code down to the radio</text>
</svg>
<figcaption>UHD sits between a host application and the USRP hardware, carrying sample streams and control over USB or Ethernet while presenting one uniform API across all USRP models.</figcaption>
</figure>

## How it works

At the center of UHD is the `multi_usrp` object, a handle that represents one or more
USRP devices as a single logical radio. A program constructs it from a device-address
string (`type=b200`, `addr=192.168.10.2`, and so on), then issues uniform calls: set the
center frequency, set the [sample rate](/reference/sample-rate/), choose a gain, pick an
antenna port, and open a streamer to pull IQ buffers. The same code drives a USB-3 B-series
board and an Ethernet-attached X-series chassis; UHD selects the right transport and
negotiates the sample format underneath.

A USRP splits work between host and hardware. On the device, an FPGA runs a
[digital down-converter](/reference/digital-down-converter/) and interpolator so it can
resample and frequency-shift on-board, delivering exactly the rate the host asked for; the
host then does the protocol-specific DSP. UHD manages that division:

- **Streaming and flow control** — it moves fixed-size sample packets across USB or Ethernet,
  handling back-pressure, sequence numbers, and overflow ("O") / underflow ("U") reporting so
  the host learns when it fell behind.
- **Timed commands and timestamps** — every sample burst carries a timestamp from the device
  clock, and commands can be scheduled for a future time, which is what makes coherent
  multi-channel and [MIMO](/reference/mimo/) capture possible.
- **Clock and time synchronization** — UHD disciplines the USRP to an internal, GPSDO, or
  external 10 MHz / PPS reference so several radios share one time base.
- **Calibration and daughterboards** — it loads per-board gain, DC-offset, and IQ-imbalance
  corrections and abstracts the interchangeable RF daughterboards that set a USRP's frequency
  coverage.

Higher-end USRPs also expose **RFNoC** (RF Network-on-Chip), a framework for loading custom
DSP blocks into the FPGA; UHD carries the control and data streams to and from those blocks,
so heavy processing can run on the device rather than the host.

## Relevance to SDR

UHD is the foundation of the USRP ecosystem and, by extension, a large amount of research
and production SDR. [GNU Radio](/reference/gnuradio/) talks to USRPs through its
`gr-uhd` blocks; [gr-osmosdr](/reference/gr-osmosdr/) and
[SoapySDR](/reference/soapysdr/) both offer UHD back ends so a USRP looks like any other
source; and countless standalone tools — spectrum monitors, cellular test beds, radio-astronomy
receivers, and passive-radar rigs — link UHD directly. Because UHD exposes device timestamps
and a shared clock, it is the usual choice whenever an experiment needs phase-coherent
multichannel capture, which cheaper single-chip dongles cannot provide. It runs on Linux,
Windows, and macOS and ships both C++ and Python bindings.

**GopherTrunk** does not link UHD. GopherTrunk is a pure-Go decoder whose front-end support
targets inexpensive receive-only radios (RTL-SDR, Airspy, and network sources), so it has no
dependency on the USRP toolchain and ships as a single static binary. A USRP is overkill for
receiving one trunking control channel, but UHD matters to the broader context GopherTrunk
lives in: it is the reference example of a well-designed host driver — uniform API, explicit
overflow signaling, device timestamps, and a clean split of DSP between hardware and host —
the same concerns GopherTrunk handles in Go for its own supported radios.

## Sources

[^uhd]: [UHD manual](https://files.ettus.com/manual/) — Ettus Research, documenting the `multi_usrp` API, streaming and flow control, timed commands and timestamps, clock synchronization, and RFNoC.
