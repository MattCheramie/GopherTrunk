---
slug: network-access-code
title: Network Access Code (NAC)
entry_type: term
category: trunked-radio
description: "A Network Access Code (NAC) is a 12-bit P25 system identifier carried in every voice and data frame that lets radios accept only their own system's traffic."
keywords: network access code, NAC, P25 NAC, 12-bit code, system access, squelch, color code, CTCSS analog, P25 CAI, digital squelch
aka: [network access code, NAC, P25 NAC]
autolink: true
infobox:
  - { label: Type, value: P25 system access code }
  - { label: Size, value: 12 bits (0x000–0xFFF) }
  - { label: Role, value: Digital squelch / system filter }
see_also: [project-25, color-code, control-channel, ran-nxdn, p25-cai, ctcss]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
---

A **Network Access Code (NAC)** is a 12-bit code embedded in every
[P25](/reference/project-25/) voice and data frame that identifies which system the
transmission belongs to, so a radio can ignore traffic from neighbouring systems that
happen to share the frequency.[^wiki] It is the digital equivalent of an analog
[CTCSS](/reference/ctcss/) tone or DCS code: a low-level "colour" that gates the
receiver before it ever unmutes, with 4096 possible values (0x000–0xFFF).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A P25 frame carrying a NAC field feeding a comparator that opens the receiver only when the NAC matches the configured value." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="30" width="240" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <line x1="90" y1="30" x2="90" y2="60" stroke="currentColor" stroke-width="1"/>
  <rect x="30" y="30" width="60" height="30" fill="currentColor" fill-opacity="0.18"/>
  <text x="60" y="49" text-anchor="middle" font-size="8.5" fill="currentColor">NAC</text>
  <text x="180" y="49" text-anchor="middle" font-size="8.5" fill="currentColor">payload (voice / data)</text>
  <line x1="270" y1="45" x2="330" y2="45" stroke="currentColor" stroke-width="1.1" marker-end="url(#nacar)"/>
  <rect x="332" y="28" width="98" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/>
  <text x="381" y="42" text-anchor="middle" font-size="8.5" fill="currentColor">NAC == 0x293 ?</text>
  <text x="381" y="55" text-anchor="middle" font-size="8" fill="currentColor">yes → unmute</text>
  <text x="230" y="95" text-anchor="middle" font-size="9" fill="currentColor">the NAC gates the receiver before any audio is heard</text>
  <text x="230" y="120" text-anchor="middle" font-size="8.5" fill="currentColor">12 bits → 4096 codes; a few (0xF7E, 0xF7F) have special meaning</text>
  <defs><marker id="nacar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Every P25 frame carries a NAC; a receiver opens only for frames whose NAC matches the system it is programmed for.</figcaption>
</figure>

## How it works

The NAC sits in the frame header alongside the network ID and data-unit type, so it is
present on both the [control channel](/reference/control-channel/) and voice channels.
A P25 radio is programmed to accept one NAC and rejects everything else — this is how
two systems can share a repeater output frequency without their users hearing each
other. A handful of values are reserved: **0xF7E** means "accept any NAC" (used for
monitoring), and **0xF7F** tells a repeater to retransmit using the received NAC.

The NAC alone does not identify a system uniquely across a wide area — many systems
reuse the same value. Full P25 identity comes from the [WACN](/reference/wacn/),
[System ID](/reference/system-id/), and [RFSS](/reference/rfss/) fields carried on the
control channel; the NAC is the fast, frame-by-frame filter, while those longer codes
are the registered network identity.

## Variants

The idea appears under different names across the digital land-mobile protocols. DMR
calls its equivalent the [color code](/reference/color-code/); NXDN calls it the
[Radio Access Number (RAN)](/reference/ran-nxdn/). All three serve the same purpose —
a small per-frame code that stops co-channel systems from interfering at the receiver —
and all three descend conceptually from analog CTCSS/DCS sub-audible signalling.

## Relevance to SDR

For a monitor the NAC is a useful discriminator: it labels which system a decoded
frame came from and lets a scanner separate overlapping [P25](/reference/project-25/)
signals. **GopherTrunk** reads the NAC from each frame it decodes and reports it, and
can decode regardless of the value (it does not need to be pre-programmed with the
system's NAC the way a subscriber radio does) — the code is data to be observed, not a
key. It is not encryption: a matching NAC grants access to signalling and clear voice,
but does nothing to protect keyed traffic.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 Common Air Interface and the Network Access Code.
