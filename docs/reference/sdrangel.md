---
slug: sdrangel
title: SDRangel
entry_type: technology
category: sdr-software
description: "SDRangel is an open-source, multi-mode SDR application supporting many devices, simultaneous receive and transmit channels, and digital-voice plugins."
keywords: SDRangel, multi-mode SDR, transmit receive SDR, digital voice plugins, DSD demod, channel plugins, cross-platform SDR
aka: [SDRangel]
autolink: true
infobox:
  - { label: Type, value: Multi-mode SDR application }
  - { label: Platform, value: "Windows, Linux, macOS (Qt)" }
  - { label: Idea, value: "Many devices, many RX/TX channels at once" }
see_also: [software-defined-radio, gqrx, sdr-sharp, hackrf, limesdr, iq-data]
cite_urls:
  - https://en.wikipedia.org/wiki/SDRangel
  - https://github.com/f4exb/sdrangel
---

**SDRangel** is a free, open-source, cross-platform [software-defined radio](/reference/software-defined-radio/)
application that supports a broad range of devices and can run **many receive and transmit
channels simultaneously**.[^proj] Where a typical hobby receiver tunes one signal at a time,
SDRangel treats a wideband capture as a workspace on which multiple **channel plugins** —
each a demodulator or modulator — operate in parallel, making it a Swiss-army toolkit rather
than a single-purpose receiver.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 132" role="img" aria-label="SDRangel processes one wideband device stream through several parallel channel plugins, each demodulating a different signal within the captured band." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="saar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="10" y="50" width="96" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
  <text x="58" y="62" text-anchor="middle" font-size="8" fill="currentColor">wideband</text><text x="58" y="72" text-anchor="middle" font-size="8" fill="currentColor">device stream</text>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="300" y="10" width="150" height="24" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="375" y="26">NFM channel</text>
    <rect x="300" y="52" width="150" height="24" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="375" y="68">digital-voice channel</text>
    <rect x="300" y="94" width="150" height="24" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="375" y="110">TX channel</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="106" y1="60" x2="298" y2="22" marker-end="url(#saar)"/>
    <line x1="106" y1="65" x2="298" y2="64" marker-end="url(#saar)"/>
    <line x1="106" y1="70" x2="298" y2="106" marker-end="url(#saar)"/>
  </g>
</svg>
<figcaption>SDRangel splits one wideband device stream into multiple parallel channel plugins — several demodulators, and even a transmit channel, at the same time.</figcaption>
</figure>

## How it works

SDRangel is organized around **device plugins** and **channel plugins**. A device plugin
opens a front end — RTL-SDR, [Airspy](/reference/airspy/), [HackRF](/reference/hackrf/),
[LimeSDR](/reference/limesdr/), [PlutoSDR](/reference/plutosdr/),
[USRP](/reference/usrp-ettus/), and more — and produces (or consumes, for transmit-capable
radios) a stream of [IQ](/reference/iq-data/) samples spanning the device bandwidth. That
wideband stream is fed to any number of channel plugins, each of which digitally tunes to an
offset within the band, filters and decimates its slice, and runs a demodulator or modulator.

Channel plugins cover a wide catalog: broadcast and narrowband FM, AM, [SSB](/reference/single-sideband/),
several digital-voice modes (D-STAR, DMR, [System Fusion/YSF](/reference/system-fusion-ysf/),
and P25 via an integrated DSD-style decoder), analog TV, packet, and utility modes, plus a
matching set of transmit modulators on TX-capable hardware. A server variant runs headless
and is controlled over a REST API, and the whole thing can be scripted for automated
monitoring. This parallel, plugin-per-channel architecture is what distinguishes SDRangel
from single-VFO receivers.

## Relevance to SDR

SDRangel appeals to users who want one program to do a lot: monitor several channels of a
band at once, experiment with transmit as well as receive, or decode digital-voice modes
without stitching together separate tools. Its built-in digital-voice channels overlap
functionally with standalone decoders — it can, for instance, follow some P25 and DMR traffic
directly — which makes it a capable all-in-one for casual monitoring across many protocols.

**GopherTrunk** is a separate project and shares no code with SDRangel; the two are
functionally adjacent in that both can turn digital-voice IQ into audio. The differences are
in scope and shape: GopherTrunk is a headless, pure-Go scanner focused specifically on
**trunked-radio** control-channel following and channel-grant tracking across systems (P25,
DMR, NXDN, TETRA, and others), shipping as a single static binary; SDRangel is a broad Qt
GUI toolkit whose digital modes are a subset of its features and whose trunk-following is not
its central purpose. They can share the same front-end hardware, and an operator might use
SDRangel's multi-channel view to survey a band before dedicating GopherTrunk to a specific
trunked system.

## Sources

[^proj]: [SDRangel](https://github.com/f4exb/sdrangel) — the project repository and wiki, documenting the device- and channel-plugin architecture, supported hardware, simultaneous RX/TX channels, and digital-voice modes.
