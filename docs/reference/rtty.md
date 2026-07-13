---
slug: rtty
title: RTTY
entry_type: protocol
category: amateur-digital
description: "RTTY (Radioteletype) is a classic digital mode that sends text as 5-bit Baudot characters over frequency-shift keying, historically at 45.45 baud with a 170 Hz shift."
keywords: RTTY, radioteletype, Baudot, ITA2, FSK, frequency shift keying, mark, space, 45.45 baud, 170 Hz shift, amateur radio, teletype
aka: [RTTY, Radioteletype]
autolink: true
infobox:
  - { label: Type, value: Text-over-radio digital mode }
  - { label: Origin, value: Landline teletype, on-air from the 1940s }
  - { label: Alphabet, value: 5-bit Baudot / ITA2 }
  - { label: Modulation, value: FSK (mark/space), 170 Hz shift }
  - { label: Rate, value: 45.45 baud (amateur), 50/75 baud (commercial) }
  - { label: GopherTrunk support, value: Not decoded (use fldigi) }
see_also: [frequency-shift-keying, hellschreiber, afsk, morse-code]
cite_urls:
  - https://en.wikipedia.org/wiki/Radioteletype
  - https://en.wikipedia.org/wiki/Baudot_code
---

**RTTY** (**Radioteletype**) is the classic way of sending typed text over radio: each
character is a 5-bit **Baudot** code shifted onto the
carrier by [frequency-shift keying](/reference/frequency-shift-keying/), toggling between
two tones called **mark** and **space**. In its standard amateur form it runs at 45.45
baud with a 170 Hz shift, a specification inherited almost unchanged from the mechanical
teleprinters of the mid-20th century.[^wiki] Simple, robust, and instantly recognizable
by its two-tone warble, RTTY remains in daily use on the HF bands nearly a century after
its introduction.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="RTTY keys the carrier between a mark tone and a space tone; each character is framed by a start bit, five Baudot data bits, and a stop bit." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="45" x2="440" y2="45" stroke="currentColor" stroke-width="0.6" stroke-dasharray="2 3"/>
  <line x1="30" y1="85" x2="440" y2="85" stroke="currentColor" stroke-width="0.6" stroke-dasharray="2 3"/>
  <text x="24" y="48" font-size="8" fill="currentColor" text-anchor="end">mark</text>
  <text x="24" y="88" font-size="8" fill="currentColor" text-anchor="end">space</text>
  <path d="M40 85 V85 M40 85 H80 V45 H120 V85 H160 V45 H200 V85 H240 V45 H280 V85 H320 V45 H360 V45 H400 V85" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <g font-size="7" fill="currentColor" text-anchor="middle"><text x="60" y="103">start</text><text x="100" y="103">b1</text><text x="140" y="103">b2</text><text x="180" y="103">b3</text><text x="220" y="103">b4</text><text x="260" y="103">b5</text><text x="360" y="103">stop</text></g>
</svg>
<figcaption>RTTY shifts between mark and space tones; each character is a start bit, five Baudot data bits, and a stop bit.</figcaption>
</figure>

## Overview

RTTY has no error correction and no handshaking — it is a raw, one-way stream of framed
characters. A start bit, five data bits, and a 1.5-bit stop period make up each character,
and the receiver simply tracks whether the incoming tone is mark or space. Because Baudot's
5-bit alphabet only has 32 codes, two "shift" characters (LTRS and FIGS) switch the decoder
between letters and figures/punctuation, doubling the usable symbol set.

## Technical characteristics

| Property | Value |
|----------|-------|
| Alphabet | 5-bit Baudot / ITA2, with LTRS/FIGS shift |
| Modulation | FSK — mark and space tones |
| Shift | 170 Hz (amateur standard; 425/850 Hz also used) |
| Rate | 45.45 baud (amateur), 50 / 75 baud (commercial) |
| Framing | 1 start + 5 data + 1.5 stop bits |
| Error control | None |

## History

Radioteletype grew out of landline teleprinter networks; on-air use expanded during and
after World War II for military, press, and weather traffic, and amateurs adopted surplus
mechanical machines in the 1950s and 60s. The 45.45-baud/170 Hz amateur convention dates to
that era of Teletype hardware and has persisted into the software-decoding age.[^baudot]
Commercial and diplomatic services used related speeds and shifts, and RTTY-style FSK long
carried weather and news via services such as NAVTEX.

## Deployment

On the amateur HF bands RTTY is still common for ragchews, DX, and especially contests,
where its speed and simplicity remain competitive. Utility and maritime FSK services
historically used the same signaling, and much surviving off-air RTTY on shortwave is
weather, news-agency, or navigational traffic.

## Decoding it with GopherTrunk

GopherTrunk does not decode RTTY — it is a trunked land-mobile scanner, not an HF text-mode
decoder. RTTY is received with an SSB receiver or SDR feeding audio into a multimode program
such as **fldigi**, MMTTY, or MultiPSK, which tracks the mark/space tones and reassembles
Baudot characters. Any SDR that provides a clean audio slice on the right frequency and
sideband will serve as the front end.

## Sources

[^wiki]: [Radioteletype](https://en.wikipedia.org/wiki/Radioteletype) — Wikipedia, for RTTY's mark/space FSK signaling, standard amateur speed and shift, framing, and history.
[^baudot]: [Baudot code](https://en.wikipedia.org/wiki/Baudot_code) — Wikipedia, for the 5-bit ITA2/Baudot alphabet and the LTRS/FIGS shift mechanism RTTY uses.
