---
slug: nxdn-cac
title: NXDN CAC
entry_type: algorithm
category: trunked-radio
description: "The NXDN Common Access Channel (CAC) is the RCCH control-channel signalling block — an 8-bit opcode, 64-bit payload, and CRC-16 — carried through a full outbound coding chain of K=5 half-rate convolution, 50/350 puncturing, and a 25×12 interleave, recovered by Viterbi decoding."
keywords: NXDN CAC, common access channel, RCCH, NXDN control channel, 88-bit message, K=5 convolutional, 50 350 puncture, 25x12 interleave, Viterbi, CRC-16
aka: [CAC, "common access channel", RCCH]
autolink: true
infobox:
  - { label: Message, value: "8-bit opcode + 64-bit payload + CRC-16 (88 bits)" }
  - { label: Coding chain, value: "155 info → conv → puncture → interleave → 300 bits" }
  - { label: FEC, value: "K=5 R=1/2 convolution + Viterbi" }
  - { label: Spec, value: "NXDN TS 1-A §4.5.1.1 / §6.4" }
see_also: [nxdn, nxdn-frame-structure, nxdn-sacch, viterbi-algorithm, puncturing, interleaving, convolutional-code, cyclic-redundancy-check, control-channel, channel-grant]
cite_urls:
  - https://en.wikipedia.org/wiki/NXDN
  - https://en.wikipedia.org/wiki/Convolutional_code
---

The **NXDN Common Access Channel** (**CAC**) is the signalling block an NXDN
[control channel](/reference/control-channel/) transmits to coordinate a trunked system — the
RCCH messages that announce the site, register radios, and grant voice and data calls onto
traffic channels.[^wiki] GopherTrunk models it at two layers: a **message layer**, an 8-bit
RCCH opcode plus a 64-bit payload protected by a 16-bit CRC (88 bits total), and a
**channel-coding layer**, the full outbound chain that wraps a 155-bit information block in
[convolutional](/reference/convolutional-code/) FEC, [puncturing](/reference/puncturing/), and
[interleaving](/reference/interleaving/) to survive the air.[^conv]

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 150" role="img" aria-label="The NXDN CAC coding chain shown as a left-to-right pipeline: a 155-bit information block gains a 16-bit CRC and 4 tail bits to make 175 bits, is convolutionally encoded at half rate to 350 bits, punctured down to 300 bits by dropping 50, and block-interleaved 25 by 12 to 300 on-air channel bits equal to 150 dibits." xmlns="http://www.w3.org/2000/svg">
  <g font-size="7.5" fill="currentColor">
    <rect x="12" y="46" width="62" height="30" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1"/>
    <text x="43" y="60" text-anchor="middle">155 info</text>
    <text x="43" y="70" text-anchor="middle">+CRC+tail</text>
    <path d="M74 61 L94 61" stroke="currentColor" stroke-width="1" marker-end="url(#a)"/>
    <rect x="94" y="46" width="66" height="30" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
    <text x="127" y="60" text-anchor="middle">K=5 R=1/2</text>
    <text x="127" y="70" text-anchor="middle">→ 350</text>
    <path d="M160 61 L180 61" stroke="currentColor" stroke-width="1" marker-end="url(#a)"/>
    <rect x="180" y="46" width="66" height="30" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
    <text x="213" y="60" text-anchor="middle">puncture</text>
    <text x="213" y="70" text-anchor="middle">drop 50 → 300</text>
    <path d="M246 61 L266 61" stroke="currentColor" stroke-width="1" marker-end="url(#a)"/>
    <rect x="266" y="46" width="66" height="30" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
    <text x="299" y="60" text-anchor="middle">25×12</text>
    <text x="299" y="70" text-anchor="middle">interleave</text>
    <path d="M332 61 L352 61" stroke="currentColor" stroke-width="1" marker-end="url(#a)"/>
    <rect x="352" y="46" width="72" height="30" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
    <text x="388" y="60" text-anchor="middle">300 bits</text>
    <text x="388" y="70" text-anchor="middle">= 150 dibits</text>
  </g>
  <text x="12" y="100" font-size="8" fill="currentColor">decode reverses it: deinterleave → depuncture → Viterbi (175 stages) → strip tail → CRC verify</text>
  <defs><marker id="a" markerWidth="6" markerHeight="6" refX="5" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The CAC outbound coding chain turns a 155-bit information block into 300 on-air bits (150 dibits): CRC and tail bits are appended, the result is convolutionally encoded at half rate, punctured from 350 down to 300 bits, and block-interleaved; the receiver runs the exact inverse and Viterbi-decodes.</figcaption>
</figure>

## RCCH message types

The 8-bit opcode that opens the message names the RCCH function. GopherTrunk enumerates the
subset the trunking state machine acts on:

| Opcode | Name | Purpose |
| --- | --- | --- |
| `0x01` | VCALL | voice call setup |
| `0x02` | VCALL_ACK | voice call acknowledgement |
| `0x04` | VCALL_ASSGN | voice channel assignment (grant) |
| `0x09` | DCALL | data call setup |
| `0x0A` | DCALL_ACK | data call acknowledgement |
| `0x0D` | DCALL_ASSGN | data channel assignment |
| `0x38` | SDCALL | short-data call |
| `0x3C` | SITE_INFO | site identification broadcast |
| `0x3D` | SRV_INFO | service information |
| `0x3F` | CCH | control-channel announcement |

A VCALL_ASSGN or DCALL_ASSGN is the [channel grant](/reference/channel-grant/) the engine
follows to a traffic channel; SITE_INFO tells it which system and site it is watching. The
message payload interpretation depends on the opcode — GopherTrunk parses the VCALL and
SITE_INFO variants into typed group/source/system fields.

## The coding chain

The channel-coding layer takes a 155-bit information block (8 SR bits plus 144 layer-3 data
bits, padded with 3 null zeros), appends a 16-bit CRC-CCITT (polynomial `0x1021`, init
`0xFFFF`, computed bit-level because 155 is not byte-aligned) and 4 zero tail bits to flush the
encoder, giving 175 input bits. Those are encoded by a constraint-length-5, rate-½
convolutional code — generators g1 = `1+D³+D⁴` (octal 31) and g2 = `1+D+D²+D⁴` (octal 27),
the same primitive the [SACCH](/reference/nxdn-sacch/) uses — producing 350 bits. A fixed
puncture matrix of period 7 (keep every G1, drop G2 at two of seven positions) removes 50 bits
to land on 300, and a 25-row by 12-column block interleaver — written row-by-row, read
column-by-column — spreads adjacent bits apart so a channel burst becomes scattered single
errors. The result is 300 channel bits, 150 dibits, carried in the CAC slot of the RCCH
outbound frame (FSW 20 + LICH 16 + CAC 300 + E 24 + Post 24).

The receiver runs the exact inverse: deinterleave, depuncture (inserting a zero-cost sentinel
at the 50 dropped positions so the metric ignores them), then a
[Viterbi](/reference/viterbi-algorithm/) decode over 175 stages constrained to end in state 0,
strip the tail, and verify the CRC. A CRC match is the gate — only a clean CAC is ingested.

## Relevance to SDR

`internal/radio/nxdn/cac.go` holds the message layer (`ParseCAC`, the `RCCHType` opcode enum,
and typed payload parsers) and `internal/radio/nxdn/cac_channel.go` holds the coding chain
(`EncodeCACChannel` / `DecodeCACChannel`) with the puncture positions and interleave
permutation computed and self-checked at package load. Getting the CAC right is what lets
GopherTrunk read an NXDN control channel at all: every grant it follows and every site it
identifies arrives as a CAC message, so the FEC chain — convolution, puncture, interleave,
Viterbi — sits directly on the critical path between a noisy control channel and a decoded
call.

## Sources

[^wiki]: [NXDN](https://en.wikipedia.org/wiki/NXDN) — Wikipedia, on the NXDN standard and its control-channel signalling.
[^conv]: [Convolutional code](https://en.wikipedia.org/wiki/Convolutional_code) — Wikipedia, on the encoder family the CAC uses and its Viterbi decoding.
