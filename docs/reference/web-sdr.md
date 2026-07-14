---
slug: web-sdr
title: Web SDR (OpenWebRX / KiwiSDR)
entry_type: technology
category: sdr-app-building
description: "A Web SDR is a browser-accessible remote SDR receiver that streams demodulated audio or raw IQ over the network, letting many users tune one shared antenna."
keywords: Web SDR, WebSDR, OpenWebRX, KiwiSDR, remote SDR receiver, browser radio, online receiver, network SDR, WebSocket audio, remote tuning, shared antenna
aka: [WebSDR, Web SDR, OpenWebRX, KiwiSDR, online SDR]
autolink: true
infobox:
  - { label: Type, value: Remote receiver web app }
  - { label: Idea, value: Share one antenna to many browsers }
  - { label: Examples, value: "OpenWebRX, KiwiSDR, WebSDR.org" }
see_also: [software-defined-radio, spyserver-protocol, network-iq-streaming, waterfall-rendering, csdr, rtl-sdr]
cite_urls:
  - https://en.wikipedia.org/wiki/WebSDR
  - https://www.openwebrx.de/
---

A **Web SDR** is a remote [software-defined radio](/reference/software-defined-radio/)
receiver whose tuning, waterfall, and audio are delivered to an ordinary web browser, so
anyone with a URL can listen without owning hardware.[^wiki] The server sits at an antenna,
digitizes a slice of spectrum, and streams it out; each connected browser opens its own
virtual receiver inside that slice, tuning and demodulating independently. Popular
implementations include **OpenWebRX**, the **KiwiSDR** appliance, and the long-running
university **WebSDR** network, which together put thousands of receivers online worldwide.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A Web SDR server at an antenna digitizes spectrum and channelizes it, then streams waterfall and audio over the network to several browsers, each tuned to a different frequency." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="wsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <path d="M30 30 L30 60 M18 30 L42 30 M22 24 L38 24" stroke="currentColor" stroke-width="1.2" fill="none"/>
    <rect x="60" y="46" width="86" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="103" y="60">SDR + server</text><text x="103" y="71">FFT / channelize</text>
    <rect x="196" y="48" width="70" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="231" y="60">network</text><text x="231" y="71">WebSocket</text>
    <rect x="320" y="16" width="120" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="380" y="32">browser — 7.1 MHz</text>
    <rect x="320" y="58" width="120" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="380" y="74">browser — 14.2 MHz</text>
    <rect x="320" y="100" width="120" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="380" y="116">browser — 145 MHz</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="30" y1="60" x2="58" y2="63" marker-end="url(#wsar)"/>
    <line x1="146" y1="63" x2="194" y2="63" marker-end="url(#wsar)"/>
    <line x1="266" y1="60" x2="318" y2="29" marker-end="url(#wsar)"/>
    <line x1="266" y1="63" x2="318" y2="71" marker-end="url(#wsar)"/>
    <line x1="266" y1="66" x2="318" y2="113" marker-end="url(#wsar)"/>
  </g>
</svg>
<figcaption>One Web SDR server digitizes a band and streams it to many browsers, each demodulating its own frequency within the shared spectrum.</figcaption>
</figure>

## How it works

A Web SDR is three cooperating pieces: a **front end**, a **DSP/streaming server**, and a
**browser client**. The front end is a physical receiver — a [KiwiSDR](/reference/rtl-sdr/)-class
direct-sampling board, an [RTL-SDR](/reference/rtl-sdr/), or a wideband SDR — that hands the
server a block of spectrum as [IQ samples](/reference/iq-data/). The server runs the heavy
signal processing: it computes an FFT for the [waterfall](/reference/waterfall-rendering/),
and for each connected user it spins up an independent virtual receiver that
[digitally down-converts](/reference/network-iq-streaming/), filters, and demodulates the
user's chosen channel. OpenWebRX, for instance, leans on the
[csdr](/reference/csdr/) DSP library for these per-client chains.

The browser side is deliberately thin. The client renders the scrolling waterfall on a
canvas, captures tuning gestures, and plays back audio, but it does the demodulation on the
**server** and receives only the finished product. Two streams flow over
**WebSockets**: a compressed waterfall/spectrum feed and a low-bitrate audio feed (often
Opus- or ADPCM-coded). When you drag the tuning cursor, the browser sends the new center
frequency and mode back up the socket, and the server re-points that user's virtual
receiver. Because every listener has a private DSP chain but shares one FFT and one antenna,
a single modest server can serve dozens of simultaneous, independently tuned users across
the whole captured band.

Some deployments instead stream **raw IQ** to the client and demodulate in JavaScript or
WebAssembly, trading bandwidth for a richer client and offloading the server. The
[SpyServer protocol](/reference/spyserver-protocol/) takes a related but non-browser
approach: it streams IQ over TCP to a native client such as SDR#. Web SDRs are the pure-web
sibling of that idea, requiring nothing but a browser.

## Relevance to SDR

Web SDRs are how most people first touch SDR without buying hardware, and they are a
genuinely useful engineering tool. Shortwave listeners use them to check propagation from
another continent; amateurs monitor their own transmissions from a distant receiver; and
signal hunters compare how a mode looks on antennas spread across the globe. For someone
*building* SDR software, a public Web SDR is a live, always-on source of real signals and a
reference implementation of the hard parts of remote radio: many-user channelization,
bandwidth-frugal waterfall coding, and latency-tolerant audio streaming over lossy links.

**GopherTrunk** is not a Web SDR — it is a headless trunking scanner/decoder that turns IQ
into decoded traffic, not a multi-user browser receiver, and it has no built-in web tuning
UI. The relationship is complementary and architectural. A Web SDR and GopherTrunk both
begin by channelizing wideband IQ into per-signal streams, and both must handle sample flow
without dropping data, so the streaming and buffering lessons carry across. In a larger
monitoring rig the two can even chain: a networked SDR front end (SpyServer- or
IQ-streaming-style) feeds GopherTrunk for decoding while a Web SDR on the same antenna gives
humans a browser view of the same spectrum. GopherTrunk stays focused on the decode side and
leaves the browser-facing receiver to purpose-built projects like OpenWebRX and KiwiSDR.

## Sources

[^wiki]: [WebSDR](https://en.wikipedia.org/wiki/WebSDR) — Wikipedia, on browser-accessible remote SDR receivers and the shared-antenna, per-user virtual-receiver model. See also the [OpenWebRX project site](https://www.openwebrx.de/) for a concrete multi-user, csdr-backed implementation.
