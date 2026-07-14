---
slug: network-iq-streaming
title: Network IQ streaming
entry_type: concept
category: sdr-data-streaming
description: "Network IQ streaming ships an SDR's baseband IQ samples over TCP or UDP so the radio and the software decoding it can run on different machines."
keywords: network IQ streaming, remote SDR, IQ over TCP, IQ over UDP, SDR bandwidth, distributed SDR, headless SDR, streaming samples, rtl_tcp spyserver soapyremote
aka: [remote IQ streaming, networked SDR, IQ over network]
autolink: true
infobox:
  - { label: Type, value: Distributed SDR data transport }
  - { label: Idea, value: Radio at the antenna, decoder elsewhere }
  - { label: Cost, value: Bandwidth = rate × 2 × bytes-per-sample }
see_also: [rtl-tcp, spyserver-protocol, vita-49, zeromq-sdr, soapyremote, iq-data]
cite_urls:
  - https://osmocom.org/projects/rtl-sdr/wiki
  - https://en.wikipedia.org/wiki/Software-defined_radio
---

**Network IQ streaming** is the practice of shipping an SDR's baseband [IQ](/reference/iq-data/)
samples over a network — TCP or UDP — so the radio can sit in one place and the software decoding it
in another.[^osmo] It is what lets a dongle live at the antenna on a rooftop Raspberry Pi while the
CPU-heavy decoding runs on a server indoors, and it underlies remote receivers, distributed sensor
networks, and shared public radios.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A radio digitizes IQ at the antenna and streams it over a TCP or UDP link to a decoding host; the required wire bandwidth is the sample rate times two components times the bytes per component." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <path d="M40 64 v-20 m-6 0 l6 -9 l6 9" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <rect x="62" y="48" width="96" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="110" y="64">radio + ADC</text><text x="110" y="74" font-size="7">digitize IQ</text>
    <rect x="300" y="48" width="110" height="32" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="355" y="68">decoder host</text>
    <line x1="158" y1="64" x2="299" y2="64" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3" marker-end="url(#niar)"/>
    <text x="228" y="56" font-size="8">IQ over TCP/UDP</text>
    <text x="228" y="98" font-size="8">bits/s ≈ rate × 2 × bytes/sample × 8</text>
  </g>
  <defs><marker id="niar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The radio digitizes IQ at the antenna and streams it to the decoder; wire bandwidth grows directly with sample rate and bit depth.</figcaption>
</figure>

## How it works

At its simplest, network IQ streaming is a socket: the radio end reads samples from the ADC and
writes them to a TCP or UDP connection, and the far end reads them back and feeds its DSP. The
protocols differ mainly in how much intelligence sits at the radio end:

- **[rtl_tcp](/reference/rtl-tcp/)** — the minimal case: raw 8-bit IQ pushed down a TCP socket, with
  one-byte commands flowing back to retune and set gain. Dumb, universal, full-rate.
- **[SpyServer](/reference/spyserver-protocol/)** — decimates a client-chosen slice on the server, so
  only the needed bandwidth crosses the wire; internet-friendly.
- **[SoapyRemote](/reference/soapyremote/)** — a transparent bridge for any SoapySDR radio, marshalling
  the whole device API plus the stream.
- **[VITA 49](/reference/vita-49/)** — the professional standard: timestamped, self-describing packets
  for coherent multi-receiver systems.
- **[ZeroMQ](/reference/zeromq-sdr/)** — a message-queue transport GNU Radio uses to move IQ between
  processes or hosts.

### The bandwidth math

The cost is unforgiving and easy to compute: **bits per second ≈ sample rate × 2 × bytes-per-sample ×
8**. A 2.4 MS/s RTL stream at 8-bit is 2.4M × 2 × 1 × 8 ≈ 38 Mbit/s — fine on wired Ethernet, marginal
on Wi-Fi. Move to a 10 MS/s Airspy at 16-bit and it is 320 Mbit/s, which is why higher-rate radios
either decimate server-side or stay on a gigabit LAN. **TCP** guarantees delivery but a stall shows up
as a burst of lag; **UDP** drops a late packet, which a decoder sees as a momentary loss of lock but
never as accumulating delay — so real-time streaming often prefers UDP for the bulk samples and TCP for
control.

## Relevance to SDR

Network IQ streaming is what makes SDR a *distributed* technology rather than a desktop one. It
separates the two things that want to be in different places: the antenna, which wants a short low-loss
feedline and a clear sky, and the compute, which wants power, cooling, and a keyboard. It underpins
headless receivers, remote monitoring, KiwiSDR/WebSDR-style shared radios, and phased sensor arrays.

GopherTrunk fits this model as a decoder that can sit at the compute end of the link: it accepts a
remote raw-IQ source (an rtl_tcp-style server) exactly as it would a local dongle, so the radio can
live at the mast while GopherTrunk runs its trunking DSP on a capable host. Just as importantly, the
*offline* analogue of streaming — recording the IQ to a file and replaying it — is central to
GopherTrunk: a capture replayed through the same production pipeline reproduces an on-air decode
bit-for-bit, sidestepping the network entirely when the goal is a debuggable, repeatable test rather
than a live feed.

## Sources

[^osmo]: [rtl-sdr (osmocom)](https://osmocom.org/projects/rtl-sdr/wiki) — the Osmocom project wiki, home of rtl_tcp, the archetypal raw network-IQ server and its command protocol.
