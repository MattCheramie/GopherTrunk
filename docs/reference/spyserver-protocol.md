---
slug: spyserver-protocol
title: SpyServer protocol
entry_type: technology
category: sdr-data-streaming
description: "SpyServer is Airspy's network SDR protocol that streams a decimated, retunable slice of the spectrum over TCP, saving bandwidth versus shipping the full sample rate."
keywords: SpyServer, spyserver protocol, Airspy network, SDR#, network SDR, decimated IQ streaming, remote Airspy, IQ over TCP, spectrum sharing, airspyhf
aka: [SpyServer, spyserver]
autolink: true
infobox:
  - { label: Type, value: Network SDR streaming protocol }
  - { label: Vendor, value: Airspy (SDR# ecosystem) }
  - { label: Key idea, value: Server-side decimation of a tunable slice }
see_also: [web-sdr, network-iq-streaming, airspy, rtl-tcp, sdr-sharp]
cite_urls:
  - https://airspy.com/quickstart/
  - https://en.wikipedia.org/wiki/Airspy
---

The **SpyServer protocol** is [Airspy](/reference/airspy/)'s network SDR streaming protocol: a
server attached to the radio ships a **decimated, retunable slice** of the captured spectrum to
remote SDR# and compatible clients over TCP, rather than the whole wideband stream.[^airspy] That
server-side channelization is its defining trick — it lets many users share one radio, or one user
work over a modest internet link, without moving the radio's full sample rate across the wire.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A SpyServer captures a wide band, decimates a client-selected narrow slice around a chosen frequency on the server, and streams only that slice to each remote client at a low data rate." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <path d="M34 74 v-20 m-6 0 l6 -9 l6 9" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <rect x="60" y="56" width="120" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="120" y="72">Airspy + SpyServer</text><text x="120" y="84" font-size="7.5">wide capture → decimate slice</text>
    <rect x="300" y="30" width="130" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="365" y="49">Client A · slice @ f₁</text>
    <rect x="300" y="92" width="130" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="365" y="111">Client B · slice @ f₂</text>
    <line x1="180" y1="72" x2="299" y2="46" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3" marker-end="url(#ssar)"/>
    <line x1="180" y1="80" x2="299" y2="106" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3" marker-end="url(#ssar)"/>
    <text x="250" y="132" font-size="7.5">only the selected slice crosses the network</text>
  </g>
  <defs><marker id="ssar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>SpyServer decimates a client-chosen narrow slice on the server, so each remote client receives only its slice — not the full wideband stream.</figcaption>
</figure>

## How it works

The client opens a TCP connection and negotiates a session: it queries the device (tuner range,
maximum sample rate, gain stages) and then subscribes to an **IQ stream** at a chosen centre
frequency and a chosen decimation stage. The server, which is continuously digitizing a wide band,
runs a [digital down-converter](/reference/digital-down-converter/) and decimator to extract just the
requested channel and sends that reduced-rate IQ down the socket. Because the heavy lifting happens
on the server, the client only ever receives the bandwidth it asked for — a few hundred kHz of a
several-MHz capture — which is what makes it usable over the public internet where
[rtl_tcp](/reference/rtl-tcp/)'s full-rate raw stream would not fit.

The protocol also offers a separate low-rate **FFT/spectrum stream** so a client can render a
waterfall of the whole band while pulling audio-bandwidth IQ from one spot within it, and it can hand
different clients different slices of the same radio simultaneously. Streams can be sent as reduced
bit-depth IQ to shave bandwidth further. The wire format is Airspy's own binary framing — richer than
rtl_tcp's one-byte commands, because it has to convey the decimation state and dual streams — and it
is the transport behind many public Airspy-based receivers.

## Relevance to SDR

SpyServer is the standard way to put an Airspy (or Airspy HF+) on the network and the backbone of a
large public network of shared receivers browsable from SDR#. Conceptually it sits between the
dumb-pipe [network IQ streaming](/reference/network-iq-streaming/) of rtl_tcp and a full
browser-based [WebSDR](/reference/web-sdr/): like WebSDR it serves many users a narrow slice each, but
it streams IQ to a native client for local demodulation rather than delivering finished audio in a
web page.

GopherTrunk does not speak the SpyServer protocol. Its network sources are the raw-IQ servers, and an
Airspy is used locally through GopherTrunk's own [Airspy](/reference/airspy/) support, feeding the
decode chain from the device directly rather than over SpyServer. The protocol is relevant here as the
model of *server-side channelization* — pushing the down-conversion to the radio end — which is exactly
the operation GopherTrunk performs internally on a local wideband capture to isolate each control
channel. Where SpyServer decimates for the network, GopherTrunk decimates for the decoder.

## In practice

A SpyServer instance is configured with a device, a maximum bandwidth to expose, and per-client limits,
then advertised so clients connect by `spyserver://host:port`. Because the server can cap the maximum
decimation a client may request, an operator running a public receiver can guarantee that no single user
consumes the whole radio or saturates the uplink. For a trunking listener the trade-off matters: a P25
or DMR control channel is only a handful of kHz wide, so the decimated slice SpyServer sends is tiny and
comfortably fits a home internet connection — but following voice grants that hop across a wide band can
demand retuning or a second stream, which is where a locally attached radio feeding a decoder directly,
as GopherTrunk uses it, avoids the round-trip latency of asking a remote server to move.

## Sources

[^airspy]: [Airspy Quickstart](https://airspy.com/quickstart/) — Airspy's documentation, covering the SpyServer network protocol, server-side decimation of a tunable slice, and the SDR# client ecosystem.
