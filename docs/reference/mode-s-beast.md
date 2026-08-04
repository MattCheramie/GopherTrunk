---
slug: mode-s-beast
title: Mode-S Beast protocol
entry_type: term
category: aviation-marine
description: The Mode-S Beast protocol is the de-facto binary wire format for shipping raw Mode-S/ADS-B frames — a 0x1A escape byte, a type code, a 6-byte MLAT timestamp and signal level, then the frame — spoken by dump1090, readsb and every ADS-B hub, and consumed by GopherTrunk over TCP.
keywords: Beast protocol, Mode-S Beast, 0x1A escape, dump1090, readsb, port 30005, MLAT timestamp, Mode-S short long, ADS-B binary format, BeastSplitter
aka: [Beast protocol, "Beast binary format", "Mode-S Beast"]
autolink: true
infobox:
  - { label: Type, value: Binary wire format }
  - { label: Escape byte, value: "0x1A (doubled to escape)" }
  - { label: Frame types, value: "Mode-AC / S short / S long" }
  - { label: Transport, value: "TCP, typically port 30005" }
see_also: [ads-b, mode-s, dump1090, compact-position-reporting, pulse-position-modulation, frame-synchronization]
cite_urls:
  - https://wiki.jetvision.de/wiki/Mode-S_Beast:Data_Output_Formats
  - https://en.wikipedia.org/wiki/Automatic_Dependent_Surveillance%E2%80%93Broadcast
---

The **Mode-S Beast protocol** is the de-facto binary wire format for shipping raw
[Mode-S](/reference/mode-s/) and [ADS-B](/reference/ads-b/) frames between programs.[^beast]
[dump1090](/reference/dump1090/), readsb, dump1090-fa, BeastSplitter and every commercial
ADS-B hub speak it — typically a listener on TCP port 30005 — and GopherTrunk consumes it as a
client so an operator can keep an existing 1090 MHz receiver and feed its decoded Mode-S frames
straight into the same aircraft-report bus the native pipeline uses.[^adsb]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 130" role="img" aria-label="A Beast frame shown as a 0x1A escape byte, a one-byte type code, a six-byte MLAT timestamp, a one-byte signal level and the raw Mode-S payload, with a note that any 0x1A inside the frame is doubled to escape it." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="30" width="40" height="28" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.1"/>
  <text x="40" y="48" text-anchor="middle" font-size="8" fill="currentColor">0x1A</text>
  <rect x="60" y="30" width="46" height="28" fill="currentColor" fill-opacity="0.20" stroke="currentColor" stroke-width="1.1"/>
  <text x="83" y="48" text-anchor="middle" font-size="8" fill="currentColor">type</text>
  <rect x="106" y="30" width="120" height="28" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/>
  <text x="166" y="48" text-anchor="middle" font-size="8" fill="currentColor">timestamp · 6 B</text>
  <rect x="226" y="30" width="56" height="28" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/>
  <text x="254" y="48" text-anchor="middle" font-size="8" fill="currentColor">signal</text>
  <rect x="282" y="30" width="158" height="28" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="361" y="48" text-anchor="middle" font-size="8" fill="currentColor">Mode-S payload (7 / 14 B)</text>
  <text x="20" y="82" font-size="7.5" fill="currentColor">0x32 = Mode-S short (56-bit) · 0x33 = Mode-S long (112-bit) · 0x31 = Mode-AC (skipped)</text>
  <text x="20" y="100" font-size="7.5" fill="currentColor">any 0x1A inside timestamp / signal / payload is sent as 0x1A 0x1A; the receiver un-stuffs it</text>
</svg>
<figcaption>Each Beast frame opens with a 0x1A escape byte and a type code, then a 6-byte MLAT timestamp, a signal-level byte and the raw Mode-S frame; embedded 0x1A bytes are doubled so the framer can resync on a lone 0x1A.</figcaption>
</figure>

## How it works

A Beast frame is `0x1A <type> <timestamp 6B> <signal 1B> <payload>`. The type code fixes the
payload length, and GopherTrunk's `payloadLen` maps the three defined types:

| Type | Meaning | Payload | Frame |
| --- | --- | --- | --- |
| `0x31` | Mode-AC | 2 bytes | (skipped) |
| `0x32` | Mode-S short | 7 bytes | 56 bits |
| `0x33` | Mode-S long | 14 bytes | 112 bits |

The 6-byte timestamp is the receiver's MLAT (multilateration) clock, useful for correlating the
same aircraft across multiple stations; the signal byte is a 0..255 level. GopherTrunk reads the
timestamp and signal but only forwards the two Mode-S types — Mode-AC frames carry no ADS-B
data and are dropped. Each surviving payload is handed to the shared ADS-B decoder, which pairs
[compact-position-reporting](/reference/compact-position-reporting/) halves through a tracker
and publishes one aircraft report.

## Byte-stuffing and resync

The single non-obvious rule is the escape. Because `0x1A` marks a frame boundary, any `0x1A`
that occurs *inside* the timestamp, signal or payload is transmitted doubled — `0x1A 0x1A` — and
the receiver un-stuffs it back to one byte. GopherTrunk's `ReadFrame` uses this to resynchronise
after a dropped byte: it hunts for a `0x1A` that is **not** followed by another `0x1A`. A
doubled pair means it has landed inside an escaped data byte, so it advances and keeps scanning;
a lone `0x1A` is a genuine frame start. `readUnstuffed` then reads exactly the expected number
of logical bytes, collapsing each `0x1A 0x1A` pair as it goes, and treats an un-paired mid-frame
`0x1A` as a protocol/​sync error.

## Relevance

Beast support means GopherTrunk does not need its own 1090 MHz pulse-position demodulator to
show aircraft: it dials a running dump1090 or readsb over TCP, ingests the Mode-S frames that
receiver already extracts, and reconnects with backoff if the upstream restarts. The framing
here is deliberately minimal — an escape byte, a length-by-type header, and a raw frame — which
is exactly why it became the interchange format the whole ADS-B ecosystem settled on.

## Sources

[^beast]: [Mode-S Beast data output formats](https://wiki.jetvision.de/wiki/Mode-S_Beast:Data_Output_Formats) — Jetvision wiki, the reference for the Beast binary framing, type codes and escape rule.
[^adsb]: [Automatic Dependent Surveillance–Broadcast](https://en.wikipedia.org/wiki/Automatic_Dependent_Surveillance%E2%80%93Broadcast) — Wikipedia, on the ADS-B / Mode-S messages the Beast format carries.
