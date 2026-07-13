---
slug: direwolf
title: Dire Wolf
entry_type: technology
category: sdr-software
description: "Dire Wolf is an open-source software TNC (terminal node controller) that modems AX.25/APRS packet radio in software, replacing a hardware modem with a sound card or SDR."
keywords: Dire Wolf, software TNC, APRS, AX.25, packet radio, KISS TNC, AGWPE, sound card modem, AFSK 1200, digipeater, iGate
aka: [Dire Wolf, direwolf]
autolink: true
infobox:
  - { label: Type, value: Software TNC / packet modem }
  - { label: Modems, value: "AX.25 / APRS (AFSK, others)" }
  - { label: Interface, value: "KISS and AGWPE to client apps" }
see_also: [aprs, ax25, kiss-tnc, packet-radio, afsk, frequency-shift-keying]
cite_urls:
  - https://en.wikipedia.org/wiki/Terminal_node_controller
  - https://github.com/wb2osz/direwolf
---

**Dire Wolf** is an open-source **software TNC** — a terminal node controller implemented
entirely in software — that modulates and demodulates [AX.25](/reference/ax25/) and
[APRS](/reference/aprs/) [packet radio](/reference/packet-radio/) using a computer's sound
card or an SDR, replacing what used to require a dedicated hardware modem.[^proj] It handles
the modem and link layer and presents standard [KISS](/reference/kiss-tnc/) and AGWPE
interfaces so applications like APRS clients, mail programs, and digipeaters can send and
receive frames over the radio.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 122" role="img" aria-label="Dire Wolf sits between the radio audio and client applications: it demodulates AFSK to AX.25 frames on receive and modulates on transmit, exposing a KISS interface." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="dwar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="8" y="46" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="48" y="58">radio audio</text><text x="48" y="68">(soundcard/SDR)</text>
    <rect x="150" y="46" width="90" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="195" y="58">Dire Wolf</text><text x="195" y="68">AFSK modem + AX.25</text>
    <rect x="302" y="46" width="98" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="351" y="58">client app</text><text x="351" y="68">(APRS / KISS)</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="88" y1="55" x2="148" y2="55" marker-end="url(#dwar)"/>
    <line x1="148" y1="67" x2="90" y2="67" marker-end="url(#dwar)"/>
    <line x1="240" y1="55" x2="300" y2="55" marker-end="url(#dwar)"/>
    <line x1="300" y1="67" x2="242" y2="67" marker-end="url(#dwar)"/>
  </g>
  <text x="118" y="92" text-anchor="middle" font-size="7" fill="currentColor">RX up / TX down</text>
  <text x="351" y="92" text-anchor="middle" font-size="7" fill="currentColor">KISS / AGWPE</text>
</svg>
<figcaption>Dire Wolf is the modem and link layer between radio audio and packet applications: it recovers AX.25 frames on receive and builds them on transmit, over a KISS interface.</figcaption>
</figure>

## How it works

On receive, Dire Wolf takes **audio** from the radio — most commonly the classic
[AFSK](/reference/afsk/) 1200-baud (Bell 202) tones used by VHF [APRS](/reference/aprs/), but
also 300-baud HF, 9600-baud [GFSK](/reference/gfsk/), and other schemes — and runs a software
demodulator to recover the bitstream. It performs **NRZI** decoding and HDLC framing, finds the
flag bytes that bound each [AX.25](/reference/ax25/) frame, checks the frame CRC, and extracts
the source/destination callsigns, digipeater path, and payload. A notable strength is that
Dire Wolf runs **multiple decoders per channel** at slightly different settings and picks the
best result, which markedly improves copy of weak or distorted packets versus a single hardware
modem.

On transmit it does the reverse: it builds the AX.25 frame, applies HDLC framing and bit
stuffing, and generates the modulated audio, keying the transmitter via PTT (serial, GPIO, or
VOX). To applications it looks like a TNC through the [KISS](/reference/kiss-tnc/) protocol
(over serial or TCP) and the AGWPE network interface, so existing packet software needs no
special support. Beyond a plain modem, Dire Wolf can act as an APRS **digipeater** (relaying
packets) and an **iGate** (bridging RF packets to the APRS Internet System), and it supports
sound cards or IQ from SDRs as its audio source.

## Relevance to SDR

Dire Wolf is the de-facto software modem for amateur packet and APRS, valued for its strong
weak-signal decoding and for eliminating dedicated TNC hardware. With an SDR or even a simple
receiver feeding audio, it turns a computer into a full APRS station, digipeater, or iGate, and
it serves as the packet engine behind many Winlink and AX.25 networking setups. It is a link-
and modem-layer tool: the SDR handles RF tuning and demodulation to audio, and Dire Wolf takes
it from there to frames.

**GopherTrunk** is unrelated in code and aims at a different part of the radio world.
GopherTrunk is a pure-Go **trunked-radio** and digital-voice scanner (P25, DMR, NXDN, TETRA,
and more) that ingests IQ and follows control channels; it is not a packet TNC and does not
implement AX.25/APRS. The connection is thematic — both are software replacements for what used
to need dedicated hardware, and both can run on the same class of SDR front end — but they solve
different problems. For AX.25/APRS packet work, Dire Wolf is the tool; GopherTrunk covers the
trunked-voice side instead.

## Sources

[^proj]: [Dire Wolf](https://github.com/wb2osz/direwolf) — the source repository and user guide, documenting the AFSK/other modems, AX.25/HDLC framing, KISS/AGWPE interfaces, and digipeater/iGate functions; background on TNCs is in the [terminal node controller Wikipedia article](https://en.wikipedia.org/wiki/Terminal_node_controller).
