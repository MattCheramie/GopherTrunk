---
slug: pactor
title: Pactor
entry_type: protocol
category: amateur-digital
description: "Pactor is a family of HF ARQ data modes that combine PSK/QAM signalling with memory-ARQ retransmission and adaptive speed to move error-free data across fading shortwave paths."
keywords: Pactor, PACTOR, HF ARQ, memory ARQ, SCS modem, Pactor-I II III IV, DPSK, QAM, radio email, Winlink data mode
aka: [Pactor, PACTOR]
autolink: true
infobox:
  - { label: Type, value: HF ARQ data mode }
  - { label: Standards body, value: SCS (proprietary); Pactor-I open }
  - { label: Introduced, value: 1991 (Pactor-I) }
  - { label: Access, value: ARQ handshake, master/slave }
  - { label: Channel spacing, value: ~500 Hz (I/II), ~2.4 kHz (III/IV) }
  - { label: Modulation, value: DPSK / PSK / QAM, adaptive }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [phase-shift-keying, winlink, quadrature-amplitude-modulation, forward-error-correction, single-sideband]
cite_urls:
  - https://en.wikipedia.org/wiki/PACTOR
---

**Pactor** (styled **PACTOR**, from the Latin *pactor*, "one who makes a contract") is a
family of HF radio data modes designed to deliver **error-free data over fading, noisy
shortwave paths**. It pairs [phase-shift-keying](/reference/phase-shift-keying/) (and, in
later versions, [QAM](/reference/quadrature-amplitude-modulation/)) with an **ARQ**
(Automatic Repeat reQuest) handshake and adaptive speed, so the two stations continuously
renegotiate rate and re-send any block that fails its checksum.[^wiki] It is the classic
data mode behind [Winlink](/reference/winlink/) radio email and marine/HF messaging.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="Pactor uses an ARQ handshake in which the sending station transmits a data block, the receiver returns an acknowledgement, and any block failing its checksum is retransmitted." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="pcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="55" y="25" font-size="9" fill="currentColor" text-anchor="middle">master</text>
  <text x="405" y="25" font-size="9" fill="currentColor" text-anchor="middle">slave</text>
  <line x1="55" y1="30" x2="55" y2="110" stroke="currentColor"/>
  <line x1="405" y1="30" x2="405" y2="110" stroke="currentColor"/>
  <line x1="55" y1="45" x2="405" y2="55" stroke="currentColor" marker-end="url(#pcar)"/>
  <text x="230" y="42" font-size="8" fill="currentColor" text-anchor="middle">data block</text>
  <line x1="405" y1="68" x2="55" y2="78" stroke="currentColor" marker-end="url(#pcar)"/>
  <text x="230" y="66" font-size="8" fill="currentColor" text-anchor="middle">ACK / NAK (speed up or repeat)</text>
  <line x1="55" y1="90" x2="405" y2="100" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#pcar)"/>
  <text x="230" y="88" font-size="8" fill="currentColor" text-anchor="middle">next / retransmitted block</text>
</svg>
<figcaption>Pactor is an ARQ conversation: the master sends a block, the slave acknowledges, and the link speeds up on clean copy or repeats blocks that fail — with memory-ARQ combining across retries.</figcaption>
</figure>

## Overview

Pactor is a synchronous, half-duplex **master/slave** protocol. The calling station becomes
master and sends fixed-length data frames; the other end replies each cycle with a short
control burst that acknowledges good frames and reports link quality. A defining trick is
**memory ARQ**: rather than discarding a corrupt frame, the receiver stores the soft samples
and *combines* them with later retransmissions, so several marginal copies add up to a
correct decode. The modem also adapts its speed level in real time — dropping to slower,
more robust signalling as conditions worsen and climbing back as they improve.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | ARQ, half-duplex master/slave |
| Pactor-I | 100/200 baud FSK, ~200 bps, open spec |
| Pactor-II | DPSK in ~500 Hz, memory ARQ, up to ~800 bps |
| Pactor-III | Multi-carrier PSK in ~2.4 kHz, up to ~5.2 kbps |
| Pactor-IV | Adaptive PSK/QAM, up to ~10+ kbps |
| Error control | CRC per frame + memory ARQ + FEC |
| Carrier | SSB audio, HF |

## History

Pactor-I appeared in 1991 as an amateur mode blending the best of AMTOR and packet radio,
and its specification is open. From **Pactor-II** (1994) onward the modes were developed and
sold by the German firm **SCS** as proprietary hardware modems, adding DPSK, memory ARQ, and
strong FEC; **Pactor-III** (2002) widened the signal for higher throughput, and **Pactor-IV**
(2013) added adaptive QAM.[^wiki] The proprietary nature of II–IV means legal transmission
generally requires an SCS modem.

## Deployment

Pactor is used for amateur HF email via Winlink, by sailors and expeditions for shore
messaging, and historically by some commercial and government HF links needing reliable
low-rate data. Because the later versions are proprietary and (in the US) limited by
symbol-rate and bandwidth rules, VARA has taken over much amateur traffic, but Pactor remains
valued for its robustness.

## Decoding it with GopherTrunk

**GopherTrunk does not decode Pactor.** Pactor-II/III/IV are proprietary SCS modes with no
open decoder, and even Pactor-I is an HF amateur mode outside GopherTrunk's land-mobile
trunking scope. GopherTrunk implements the general building blocks Pactor draws on —
[PSK](/reference/phase-shift-keying/) demodulation and
[FEC](/reference/forward-error-correction/) — but not this protocol's framing or its ARQ
state machine.

## Sources

[^wiki]: [PACTOR](https://en.wikipedia.org/wiki/PACTOR) — Wikipedia, for the Pactor version history, ARQ and memory-ARQ operation, modulation per version, and proprietary status.
