---
slug: vita-49
title: VITA 49 (VRT)
entry_type: technology
category: sdr-data-streaming
description: "VITA 49 (VITA Radio Transport) is a standard for packetizing digitized IQ with synchronized metadata — timestamps, centre frequency, and reference levels — for networked SDR."
keywords: VITA 49, VITA Radio Transport, VRT, digital IF interoperability, packetized IQ, signal data packet, context packet, ANSI VITA 49.2, networked SDR standard
aka: [VITA 49, VRT, VITA Radio Transport, VITA-49]
autolink: true
infobox:
  - { label: Type, value: Packetized IQ + metadata transport standard }
  - { label: Packets, value: Signal data + context (metadata) }
  - { label: Body, value: "VITA / ANSI (VITA 49.2)" }
see_also: [network-iq-streaming, sample-format, iq-data, usrp-ettus, digital-down-converter]
cite_urls:
  - https://en.wikipedia.org/wiki/VITA_Radio_Transport
  - https://www.vita.com/Standards
---

**VITA 49**, the VITA Radio Transport (VRT) standard, defines a packet format for carrying
digitized [IQ data](/reference/iq-data/) together with synchronized metadata — precise timestamps,
centre frequency, sample rate, bandwidth, and reference levels — so a digitizer and its downstream
processing can be separate boxes on a network and still agree on exactly what each sample means.[^vita]
It is the interoperability layer of professional and defence SDR, standardising the "digital IF"
that vendors previously each defined privately.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A VITA 49 stream interleaves signal data packets carrying timestamped IQ payloads with context packets carrying centre frequency, sample rate and reference-level metadata, linked by a stream identifier." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="24" y="30" width="130" height="40" rx="4" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="89" y="46">Signal Data Packet</text><text x="89" y="60" font-size="7.5">header · timestamp · IQ</text>
    <rect x="176" y="30" width="130" height="40" rx="4" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="241" y="46">Signal Data Packet</text><text x="241" y="60" font-size="7.5">header · timestamp · IQ</text>
    <rect x="328" y="30" width="108" height="40" rx="4" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="382" y="52">· · ·</text>
    <rect x="100" y="96" width="200" height="40" rx="4" fill="none" stroke="currentColor" stroke-width="1.3" stroke-dasharray="4 3"/><text x="200" y="112">Context Packet</text><text x="200" y="126" font-size="7.5">freq · rate · bandwidth · ref level</text>
    <line x1="200" y1="96" x2="130" y2="71" stroke="currentColor" stroke-width="1" stroke-opacity="0.7"/>
    <line x1="200" y1="96" x2="241" y2="71" stroke="currentColor" stroke-width="1" stroke-opacity="0.7"/>
    <text x="330" y="112" font-size="7.5">linked by Stream ID</text>
  </g>
</svg>
<figcaption>A VITA 49 stream interleaves timestamped signal-data packets with context packets describing the RF parameters, tied together by a stream identifier.</figcaption>
</figure>

## How it works

A VRT stream is a sequence of two kinds of packet, sharing a **Stream Identifier** so a receiver
knows which data belongs with which description:

- **Signal Data packets** carry the payload — the IQ samples themselves, in a declared
  [sample format](/reference/sample-format/) (integer or float, a chosen bit width) — prefixed by a
  header and an optional high-precision **timestamp**. The timestamp pairs an integer-seconds field
  with a fractional field (often picosecond-resolution sample counts), so every packet is pinned to
  an absolute moment. This is what lets multiple digitizers be sample-aligned for phased arrays,
  direction finding, or [beamforming](/reference/beamforming/).
- **Context packets** carry the metadata that describes those samples: RF centre frequency, IF,
  sample rate, bandwidth, gain, reference level, and GPS/geolocation fields. They are sent when
  parameters change (and periodically as a keep-alive), so a late-joining receiver can recover the
  full state without a side channel.

VITA 49.2 extended the standard with a richer set of context fields and command packets for control,
turning VRT from a one-way transport into a fuller interface. Because the format is self-describing
in-band — every parameter needed to interpret the IQ rides in the same stream — VRT scales to
many-channel, multi-receiver systems where a filename convention or a static config would not.

## Relevance to SDR

VITA 49 dominates the professional, test-and-measurement, and defence SDR world: Ettus/NI USRP
radios can emit VRT, signal-analysis platforms and the GNU Radio `gr-vrt` work speak it, and large
sensor networks use it precisely because the timestamped context model keeps geographically separated
digitizers coherent. It is the heavyweight counterpart to the hobbyist
[network IQ streaming](/reference/network-iq-streaming/) protocols like rtl_tcp and spyserver: where
those ship a bare byte stream and rely on out-of-band tuning commands, VRT bakes the metadata into
every packet.

GopherTrunk does not implement VITA 49 — it is a consumer-SDR trunking decoder whose network inputs
are the lightweight raw-IQ servers, and whose offline path reads plain
[IQ files](/reference/iq-file-format/) with the parameters given on the command line. VRT is worth
knowing here as the reference point for how a *rigorous* streaming format solves the same problem
GopherTrunk solves informally: the timestamp and context fields VITA 49 standardises are exactly the
sample rate, centre frequency, and timing that a bare capture forces you to track by hand. If a
future GopherTrunk needed to ingest from professional receiver infrastructure, VRT is the format it
would have to learn.

## Sources

[^vita]: [VITA Radio Transport](https://en.wikipedia.org/wiki/VITA_Radio_Transport) — Wikipedia, on the VRT/VITA 49 packet model, signal-data vs context packets, timestamping, and its role as the digital-IF interoperability standard.
