---
slug: phil-karn
title: Phil Karn
entry_type: person
category: people
description: Phil Karn (KA9Q) is an American engineer and radio amateur known for Karn's algorithm, the KA9Q packet-radio software, and widely used forward-error-correction libraries.
keywords: Phil Karn, KA9Q, Karn's algorithm, packet radio, AX.25, NOS, forward error correction, Viterbi decoder, Reed-Solomon, TCP retransmission
aka: [Phil Karn, KA9Q]
autolink: true
infobox:
  - { label: Lived, value: "1956–present" }
  - { label: Field, value: "Communications / software engineering" }
  - { label: Known for, value: "KA9Q packet radio, Karn's algorithm, FEC code" }
see_also: [packet-radio, ax25, forward-error-correction, viterbi-algorithm]
cite_urls:
  - https://en.wikipedia.org/wiki/Phil_Karn
---

**Phil Karn** (amateur radio callsign KA9Q) is an American engineer and radio amateur
whose work bridges the internet and radio: he devised **Karn's algorithm** for TCP
round-trip estimation, wrote the influential KA9Q **[packet-radio](/reference/packet-radio/)**
networking software, and authored open-source
[forward-error-correction](/reference/forward-error-correction/) libraries used across the
SDR and amateur communities.[^wiki] His career spans professional communications
engineering at companies such as Bell Labs, Bellcore, and Qualcomm, and decades of
volunteer contribution to amateur digital radio.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="An IP packet encapsulated inside an AX.25 frame, error-protected with forward error correction, then transmitted over a radio link." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="25" y="50" width="55" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="52" y="68">IP packet</text>
    <rect x="95" y="50" width="60" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="125" y="65">AX.25</text><text x="125" y="75">frame</text>
    <rect x="170" y="50" width="55" height="30" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="197" y="68">FEC</text>
    <rect x="240" y="50" width="60" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="270" y="68">modem</text>
    <g stroke="currentColor" stroke-width="1"><line x1="80" y1="65" x2="94" y2="65" marker-end="url(#pkar)"/><line x1="155" y1="65" x2="169" y2="65" marker-end="url(#pkar)"/><line x1="225" y1="65" x2="239" y2="65" marker-end="url(#pkar)"/></g>
    <line x1="300" y1="65" x2="335" y2="65" stroke="currentColor" stroke-width="1" marker-end="url(#pkar)"/>
    <g stroke="currentColor" stroke-width="1.2" fill="none"><line x1="345" y1="65" x2="345" y2="35"/><path d="M335 40 A 14 14 0 0 1 355 40" stroke-dasharray="3 2"/><path d="M328 33 A 24 24 0 0 1 362 33" stroke-dasharray="3 2"/></g>
    <text x="345" y="95" font-size="8">RF link</text>
  </g>
  <defs><marker id="pkar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Karn's KA9Q software carried IP over AX.25 packet radio; his FEC libraries add error correction to noisy radio links.</figcaption>
</figure>

## Life and work

Karn was born in 1956 and earned degrees in electrical engineering from Cornell and
Carnegie Mellon. As a professional he worked on data communications at Bell Labs and
Bellcore and later on cellular systems at Qualcomm. In parallel, as radio amateur KA9Q,
he became one of the central figures in the amateur packet-radio movement of the 1980s and
1990s.[^wiki]

His **KA9Q Network Operating System** ("NOS") was a landmark: a self-contained TCP/IP
stack that ran on ordinary personal computers and carried internet protocols over amateur
radio links using the [AX.25](/reference/ax25/) data-link protocol. For many operators it
was their first hands-on TCP/IP implementation, and it demonstrated that packet-switched
internetworking could run over slow, noisy [radio](/reference/packet-radio/) channels.

## Contribution

In professional networking, Karn is known for **Karn's algorithm**, a simple but important
rule for measuring TCP's round-trip time: retransmitted segments are excluded from
round-trip samples, because you cannot tell whether an acknowledgement matches the original
transmission or the retransmission. The rule prevents corrupted timing estimates and is
standard in TCP implementations.[^wiki]

For radio, Karn's most enduring gift to the SDR world is a set of carefully optimised,
freely licensed **[forward-error-correction](/reference/forward-error-correction/)**
routines: fast [Viterbi](/reference/viterbi-algorithm/) decoders for convolutional codes,
Reed-Solomon encoders and decoders, and related building blocks. These libraries have been
reused across amateur and open-source projects — from deep-space telemetry experiments to
software radios — because they are correct, efficient, and unencumbered.

## Legacy

Karn's combination of professional rigor and open contribution made him one of the quiet
architects of amateur digital radio. His NOS software seeded a generation of packet
operators, his TCP algorithm is embedded in the internet's transport layer, and his FEC
code continues to be pulled into new SDR projects whenever a developer needs a proven
Viterbi or Reed-Solomon implementation. He also became a prominent advocate for strong
cryptography and open standards. His work is a reminder that the same coding theory
protects data whether it travels over fiber or over a fading radio path.

## Sources

[^wiki]: [Phil Karn](https://en.wikipedia.org/wiki/Phil_Karn) — Wikipedia, for biography, Karn's algorithm, the KA9Q NOS software, and his forward-error-correction libraries.
