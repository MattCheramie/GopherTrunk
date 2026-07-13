---
slug: color-code
title: Color code (DMR)
entry_type: term
category: trunked-radio
description: "A DMR color code is a 4-bit system identifier (0–15) carried in every burst that lets radios and repeaters accept only their own system's traffic on a shared frequency."
keywords: color code, DMR color code, CC, 4-bit code, DMR access, digital squelch, CACH, NAC equivalent, RAN equivalent, repeater slot
aka: [color code, DMR color code, CC]
autolink: true
infobox:
  - { label: Type, value: DMR system access code }
  - { label: Size, value: 4 bits (0–15) }
  - { label: Role, value: Repeater/system access filter }
see_also: [dmr, network-access-code, ran-nxdn, control-channel, tdma, dmr-tier-3]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
---

A **color code (CC)** is a 4-bit number, 0 through 15, carried in every
[DMR](/reference/dmr/) burst that identifies which system the transmission belongs to,
so a repeater or radio accepts only its own traffic on a shared frequency.[^wiki] It is
DMR's direct counterpart to the P25 [Network Access Code](/reference/network-access-code/)
and the NXDN [RAN](/reference/ran-nxdn/): a small per-burst "colour" that keeps two
co-channel systems from keying each other's repeaters or unmuting each other's radios.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A DMR two-slot TDMA frame whose signalling carries a 4-bit color code that a repeater checks before repeating a burst." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="30" y="34" width="180" height="28" rx="4" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1.1"/><text x="120" y="52">slot 1  (CC = 1)</text>
    <rect x="215" y="34" width="180" height="28" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="305" y="52">slot 2  (CC = 1)</text>
  </g>
  <text x="212" y="80" text-anchor="middle" font-size="8.5" fill="currentColor">30 ms TDMA frame — two timeslots share one 12.5 kHz channel</text>
  <line x1="120" y1="62" x2="120" y2="98" stroke="currentColor" stroke-width="1" marker-end="url(#ccar)"/>
  <rect x="70" y="100" width="284" height="26" rx="5" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/>
  <text x="212" y="117" text-anchor="middle" font-size="8.5" fill="currentColor">repeater repeats the burst only if CC matches its configured value</text>
  <defs><marker id="ccar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>DMR carries a 4-bit color code in each burst; a repeater keys up only for bursts whose color code matches its own.</figcaption>
</figure>

## How it works

DMR splits one 12.5 kHz channel into two [TDMA](/reference/tdma/) timeslots. The color
code appears in the burst signalling (the slot-type and embedded-signalling fields, and
in the CACH), so it is present on both timeslots and on the
[control channel](/reference/control-channel/) of trunked systems. A repeater is
configured with one color code and will only repeat bursts that carry it; a subscriber
radio likewise mutes anything with the wrong code. Because it is only 4 bits, planners
assign color codes so that neighbouring repeaters on the same frequency use different
values — the same reuse logic as CTCSS tones on analog channels.

The color code is an access filter, not an address. Talkgroup and radio identity are
separate fields, and full network identity on a trunked
[DMR Tier III](/reference/dmr-tier-3/) system comes from the system-identity signalling
on the control channel. The color code is simply the fast gate that runs first.

## In practice

- On a **conventional** DMR repeater the color code is fixed and set once; on a
  **trunked** [DMR Tier III](/reference/dmr-tier-3/) system every site's control
  channel and voice channels carry it, and it stays constant across the system.
- Two repeaters on the same output frequency must use different color codes to
  coexist, so hobbyists sometimes deduce coverage overlap by watching which codes
  appear on a channel.
- Because it is only 4 bits it is trivial to enumerate — a monitor can simply report
  whatever code arrives rather than needing a lookup table.

## Relevance to SDR

For a monitor the color code labels which system a decoded burst belongs to and helps
separate two DMR signals sharing a frequency. **GopherTrunk** reads the color code from
each burst it decodes and reports it; it does not need to be pre-set to a system's color
code the way a subscriber radio does, because a receiver simply observes whatever value
is present. Like the P25 NAC, it is metadata, not encryption — a matching color code
gives access to signalling and clear voice but does nothing against keyed traffic.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on the DMR air interface and its color code.
