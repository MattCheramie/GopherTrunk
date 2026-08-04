---
slug: ambe-3003-packet
title: AMBE-3003 packet
entry_type: term
category: voice-coding
description: The AMBE-3003 packet is the serial wire format that carries frames to and from a DVSI AMBE-3000/3003 hardware vocoder chip — a 0x61 sync byte, a big-endian length, a type byte, and a payload of channel data or PCM.
keywords: AMBE-3003 packet, AMBE-3000, DVSI vocoder chip, 0x61 sync byte, packet type, channel data, speech data, FTDI, dvsi build tag, hardware vocoder
aka: [AMBE-3000 packet, "DVSI vocoder packet", "AMBE-3003 wire format"]
autolink: true
infobox:
  - { label: Sync byte, value: "0x61" }
  - { label: Header, value: "sync + length(2) + type" }
  - { label: Length, value: "big-endian, covers type + payload" }
  - { label: Build tag, value: dvsi }
see_also: [dvsi, ambe-plus-2, ambe-plus-2-codebooks, pulse-code-modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_Voice_Systems
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
---

The **AMBE-3003 packet** is the serial wire format that carries data to and from a
[DVSI](/reference/dvsi/) AMBE-3000/3003 hardware vocoder chip — the physical DSP device that
encodes and decodes [AMBE+2](/reference/ambe-plus-2/) speech. When GopherTrunk is built to offload
vocoding to such a dongle (behind the `dvsi` build tag), it speaks this framing over the chip's
serial link: a fixed sync byte, a length, a type, and a payload.[^dvsi] The packet framing is the
only part of the hardware path that is pure protocol — no MBE math — so it compiles
unconditionally, while the code that actually drives the chip is gated behind the build tag.

<figure class="figure" markdown="0">
<svg viewBox="0 0 462 118" role="img" aria-label="An AMBE-3003 packet laid out left to right: a one-byte sync field of hexadecimal 61, a two-byte big-endian length field, a one-byte type field, and a variable-length payload; the length field covers the type byte plus the payload." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="16" y="34" width="56" height="30" fill="currentColor" fill-opacity="0.28"/><text x="44" y="49">sync</text><text x="44" y="59" font-size="7">0x61</text>
    <rect x="72" y="34" width="90" height="30" fill="currentColor" fill-opacity="0.16"/><text x="117" y="49">length</text><text x="117" y="59" font-size="7">2 bytes, BE</text>
    <rect x="162" y="34" width="56" height="30" fill="currentColor" fill-opacity="0.22"/><text x="190" y="49">type</text><text x="190" y="59" font-size="7">1 byte</text>
    <rect x="218" y="34" width="228" height="30" fill="none"/><text x="332" y="52">payload</text>
  </g>
  <path d="M162 74 L162 82 L446 82 L446 74" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="304" y="96" text-anchor="middle" font-size="7.5" fill="currentColor">length covers type + payload</text>
</svg>
<figcaption>Every AMBE-3003 packet opens with the 0x61 sync byte, a big-endian length covering the type byte plus payload, a type byte, then the payload.</figcaption>
</figure>

## Wire format

The header is four bytes: the sync byte `0x61`, a two-byte big-endian length, and a one-byte type.
The length field covers the type byte *plus* the payload, so a packet with no payload has
length 1, and the full packet on the wire is three header bytes plus the declared length. A decoder
validates the sync byte first, then checks that the buffer length matches the declared length
exactly — trailing bytes are rejected rather than silently accepted, which keeps a byte-stream
resync honest. A `SplitPackets` helper walks a stream of back-to-back packets and returns the
leftover trailing bytes of an incomplete packet for buffering across the next read, stopping on the
first bad sync so the caller can resync upstream.

## Packet types

The one-byte type field names the packet's purpose:

| Type | Value | Payload |
|------|-------|---------|
| `PktControl` | 0x00 | configuration / status exchange (sample rate, channel format) |
| `PktChannelData` | 0x01 | one 49-bit AMBE+2 frame, packed in 7 bytes |
| `PktSpeechData` | 0x02 | 160 samples of 16-bit signed PCM (320 bytes, little-endian, 8 kHz) |
| `PktAck` | 0x06 | the chip's response to a control request |

The two data types are the working pair. A `PktChannelData` packet carries the compressed
49-bit [AMBE+2](/reference/ambe-plus-2-codebooks/) frame — from host to chip for synthesis, or from
chip to host for encoding. A `PktSpeechData` packet carries the decompressed side: one 20 ms voice
frame as 160 samples of little-endian 16-bit [PCM](/reference/pulse-code-modulation/) at 8 kHz
mono, 320 bytes. Control packets configure the chip and draw an ack in reply.

## Why the framing is unconditional

The framing primitives — encode, decode, split — carry no patent surface, because describing a
chip's serial wire protocol is not the same as implementing the proprietary vocoder algorithm. So
GopherTrunk compiles them into every build, unconditionally, and keeps them unit-testable without
any hardware present. Only the `Vocoder` that opens the FTDI serial link and streams packets to the
physical chip sits behind `//go:build dvsi`. This split lets the packet format be reasoned about,
fuzzed, and round-tripped in ordinary CI while the hardware-dependent transport stays optional.

## Relevance to SDR

Most GopherTrunk deployments decode AMBE+2 in pure Go and never touch a chip. The AMBE-3003 packet
path exists for the cases where an operator has a DVSI dongle and prefers to offload vocoding to it
— for throughput, for bit-exact conformance against DVSI's own silicon, or because a build cannot
use the software decoder. In that mode GopherTrunk becomes a packet pump: it pulls 49-bit
[AMBE+2](/reference/ambe-plus-2/) frames out of the P25 Phase 2, DMR, or NXDN bitstream, wraps each
in a `PktChannelData` packet, and reads back `PktSpeechData` PCM to feed the audio pipeline. The
framing is small, but it is the contract between GopherTrunk and the hardware, so its length and
sync handling have to be exact.

## Sources

[^dvsi]: [Digital Voice Systems](https://en.wikipedia.org/wiki/Digital_Voice_Systems) — Wikipedia, on DVSI, the AMBE vocoder family, and its hardware vocoder products.
