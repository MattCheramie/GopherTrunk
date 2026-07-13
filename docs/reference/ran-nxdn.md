---
slug: ran-nxdn
title: Radio Access Number (RAN)
entry_type: term
category: trunked-radio
description: "A Radio Access Number (RAN) is a 6-bit NXDN system identifier carried in every frame that lets radios and repeaters accept only their own system's traffic on a shared channel."
keywords: radio access number, RAN, NXDN RAN, 6-bit code, NXDN access, digital squelch, color code, NAC equivalent, IDAS, NEXEDGE
aka: [radio access number, RAN, NXDN RAN]
autolink: true
infobox:
  - { label: Type, value: NXDN system access code }
  - { label: Size, value: 6 bits (0–63) }
  - { label: Role, value: Repeater/system access filter }
see_also: [nxdn, network-access-code, color-code, control-channel]
cite_urls:
  - https://en.wikipedia.org/wiki/NXDN
---

A **Radio Access Number (RAN)** is a 6-bit code, 0 through 63, carried in every
[NXDN](/reference/nxdn/) frame that identifies which system a transmission belongs to,
so a repeater or radio accepts only its own traffic on a shared channel.[^wiki] It is
NXDN's equivalent of the P25 [Network Access Code](/reference/network-access-code/) and
the DMR [color code](/reference/color-code/): a small per-frame code that keeps
co-channel systems from keying each other's repeaters or unmuting each other's radios.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="An NXDN frame carrying a 6-bit RAN field that a repeater compares against its configured value before repeating the frame." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="34" width="260" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <rect x="30" y="34" width="70" height="30" fill="currentColor" fill-opacity="0.18"/>
  <text x="65" y="53" text-anchor="middle" font-size="8.5" fill="currentColor">RAN</text>
  <text x="195" y="53" text-anchor="middle" font-size="8.5" fill="currentColor">frame payload (voice / data)</text>
  <line x1="290" y1="49" x2="345" y2="49" stroke="currentColor" stroke-width="1.1" marker-end="url(#ranar)"/>
  <rect x="347" y="32" width="83" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/>
  <text x="388" y="46" text-anchor="middle" font-size="8" fill="currentColor">RAN match?</text>
  <text x="388" y="59" text-anchor="middle" font-size="8" fill="currentColor">yes → repeat</text>
  <text x="230" y="100" text-anchor="middle" font-size="9" fill="currentColor">6 bits → 64 codes; the fast gate before addressing</text>
  <defs><marker id="ranar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Every NXDN frame carries a RAN; a repeater or radio acts only on frames whose RAN matches its configured value.</figcaption>
</figure>

## How it works

NXDN is a narrowband (6.25 kHz and 12.5 kHz) [FDMA](/reference/fdma/) digital protocol.
The RAN travels in the frame's Lich/signalling fields, so it is present on voice frames
and on the [control channel](/reference/control-channel/) of trunked NXDN systems. A
repeater is programmed with one RAN and will only repeat frames carrying it; radios
mute anything else. With 64 possible values, planners give overlapping repeaters on the
same frequency different RANs — the same interference-avoidance role that CTCSS tones
play on analog channels and that the color code plays in DMR.

The RAN is only an access filter. It does not by itself uniquely identify a system over
a wide area, and it says nothing about talkgroup or radio identity — those are separate
fields, with full network identity coming from the system signalling on the control
channel. The RAN is just the first, fastest check.

## In practice

- On a **conventional** NXDN channel the RAN is set once per repeater; on a
  **trunked** NXDN system it is carried on the control channel and every voice channel,
  constant across the system.
- Adjacent repeaters sharing an output frequency use different RANs to coexist, the same
  frequency-reuse discipline as CTCSS tones or DMR color codes.
- With 64 values the RAN is easy to observe directly; a monitor reports whatever value
  arrives rather than needing to be told it in advance.

## Relevance to SDR

For a monitor the RAN labels which system a decoded NXDN frame belongs to and helps
separate two systems sharing a frequency. **GopherTrunk** reads the RAN from each frame
it decodes and reports it; a receiver observes whatever value is present rather than
being pre-programmed with it. As with the P25 NAC and DMR color code, the RAN is
metadata, not encryption — matching it grants access to signalling and clear voice but
does nothing against keyed traffic. NXDN is marketed as IDAS (Icom) and NEXEDGE
(Kenwood), both of which use the RAN identically.

## Sources

[^wiki]: [NXDN](https://en.wikipedia.org/wiki/NXDN) — Wikipedia, on the NXDN common air interface and the Radio Access Number.
