---
slug: kiss-tnc
title: KISS TNC
entry_type: technology
category: paging-data
description: KISS is a minimal framing protocol between a computer and a terminal node controller, passing raw AX.25 packet-radio frames over a serial or TCP link with almost no overhead.
keywords: KISS, TNC, terminal node controller, AX.25, packet radio, SLIP framing, FEND, KISS mode, software TNC, Direwolf
aka: [KISS, "KISS TNC", "KISS protocol"]
autolink: true
infobox:
  - { label: Type, value: Host-to-TNC framing protocol }
  - { label: Idea, value: "Keep It Simple, Stupid — move AX.25 logic to the host" }
  - { label: Examples, value: "Direwolf, hardware TNCs, hotspots" }
see_also: [ax25, packet-radio, direwolf, afsk, multimon-ng, frame-synchronization]
cite_urls:
  - https://en.wikipedia.org/wiki/KISS_(TNC)
  - https://en.wikipedia.org/wiki/Terminal_node_controller
---

**KISS** ("Keep It Simple, Stupid") is a minimal framing protocol that connects a host
computer to a **terminal node controller** (TNC) — the modem that turns
[packet radio](/reference/packet-radio/) tones into bits. In KISS mode the TNC does
almost nothing but the physical modem work, passing raw **[AX.25](/reference/ax25/)**
frames to and from the host with only a few bytes of overhead, so all protocol logic
lives in software on the computer.[^wiki][^tnc]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A KISS frame wraps an AX.25 packet between FEND delimiter bytes with a leading command byte, carried over serial or TCP between host and TNC." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="ki_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1.1" font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="30" y="20" width="70" height="28" fill="currentColor" fill-opacity="0.18"/><text x="65" y="38">host / app</text>
    <rect x="360" y="20" width="70" height="28" fill="currentColor" fill-opacity="0.18"/><text x="395" y="38">TNC / modem</text>
  </g>
  <line x1="100" y1="34" x2="360" y2="34" stroke="currentColor" stroke-width="1" marker-end="url(#ki_ar)"/>
  <line x1="360" y1="40" x2="100" y2="40" stroke="currentColor" stroke-width="1" marker-end="url(#ki_ar)"/>
  <text x="230" y="30" text-anchor="middle" font-size="7.5" fill="currentColor">serial or TCP</text>
  <g stroke="currentColor" stroke-width="1.1" font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="55" y="70" width="34" height="26" fill="currentColor" fill-opacity="0.22"/><text x="72" y="87">FEND</text>
    <rect x="89" y="70" width="34" height="26" fill="currentColor" fill-opacity="0.12"/><text x="106" y="87">cmd</text>
    <rect x="123" y="70" width="220" height="26" fill="none"/><text x="233" y="87">AX.25 frame (data)</text>
    <rect x="343" y="70" width="34" height="26" fill="currentColor" fill-opacity="0.22"/><text x="360" y="87">FEND</text>
  </g>
  <text x="228" y="112" text-anchor="middle" font-size="8" fill="currentColor">FEND-delimited KISS frame carrying a raw AX.25 packet</text>
</svg>
<figcaption>KISS delimits each raw AX.25 frame with FEND bytes and a command byte, letting the host, not the TNC, run the protocol.</figcaption>
</figure>

## How it works

KISS uses SLIP-style framing: each frame is bracketed by a special **FEND** delimiter byte
(0xC0), with byte-stuffing (FESC escapes) so that any FEND or FESC appearing inside the
data cannot be mistaken for a boundary. The first byte after the opening FEND is a
command/port nibble — most often "data frame" — and everything up to the closing FEND is
the raw AX.25 packet. There is no error checking or acknowledgement in KISS itself; that
is deliberately left to the AX.25 layer and the application. The result is that a "dumb"
KISS TNC is just a bidirectional pipe between the radio's modem and host software.

The design philosophy is the point: earlier TNCs embedded the whole AX.25 state machine
in firmware, which was hard to update. KISS moves that intelligence into the host, where
it can evolve freely, and leaves the TNC responsible only for
[AFSK](/reference/afsk/)/GFSK modem duties and [frame
synchronization](/reference/frame-synchronization/).

## Relevance to SDR

KISS is the near-universal interface between packet-radio modems and applications. A
software-defined-radio setup commonly runs [Direwolf](/reference/direwolf/) as a software
TNC that demodulates the audio, then exposes a KISS TCP socket that APRS clients, Winlink,
or BBS software connect to — no hardware TNC required. Because the framing is trivial,
KISS is easy to implement and is supported by essentially every packet application, making
it the natural bridge from an SDR audio stream to higher-level data protocols.

GopherTrunk targets land-mobile trunking and paging, so it neither implements a KISS
interface nor decodes AX.25; KISS is documented here as the standard glue in the adjacent
amateur packet-radio ecosystem, alongside tools like Direwolf and
[multimon-ng](/reference/multimon-ng/).

## Sources

[^wiki]: [KISS (TNC)](https://en.wikipedia.org/wiki/KISS_(TNC)) — Wikipedia, for the KISS framing protocol, its FEND delimiters and command byte, and the move of AX.25 logic to the host.
[^tnc]: [Terminal node controller](https://en.wikipedia.org/wiki/Terminal_node_controller) — Wikipedia, for the TNC's role as the packet-radio modem between radio and computer.
