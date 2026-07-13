---
slug: cellular-handover
title: Handover (cellular)
entry_type: term
category: cellular
description: Handover is the process of moving an active call or data session from one cell to another as a device moves, spanning hard handover, soft handover, and idle-mode cell reselection.
keywords: handover, handoff, cellular handover, hard handover, soft handover, softer handover, cell reselection, make-before-break, break-before-make, X2 handover, mobility, CDMA soft handoff
aka: [handover, handoff, cellular handover]
autolink: true
infobox:
  - { label: Purpose, value: Keep a session alive across cell boundaries }
  - { label: Hard vs soft, value: "Break-before-make vs make-before-break" }
  - { label: Idle mode, value: Cell reselection (no active session) }
see_also: [cdma, lte, roaming, neighbor-site, multisite-trunking, registration]
cite_urls:
  - https://en.wikipedia.org/wiki/Handover
  - https://en.wikipedia.org/wiki/Mobile_phone_signal
---

**Handover** (also **handoff**) is the process by which a cellular network transfers an
active call or data session from one cell to another as the user moves, so that the
connection continues without being dropped.[^wiki] It is the mechanism that makes
mobility work: a moving handset is continually handed between base stations, ideally
without the user noticing.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A vehicle moves from the coverage area of cell A into the overlapping coverage of cell B; a hard handover switches abruptly at the boundary, while a soft handover briefly connects to both cells before releasing the first." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="char" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <circle cx="140" cy="70" r="60" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-dasharray="3 3"/>
  <circle cx="300" cy="70" r="60" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-dasharray="3 3"/>
  <line x1="140" y1="70" x2="140" y2="45" stroke="currentColor" stroke-width="1.4"/><circle cx="140" cy="70" r="3" fill="currentColor"/><text x="140" y="98" text-anchor="middle" font-size="9" fill="currentColor">cell A</text>
  <line x1="300" y1="70" x2="300" y2="45" stroke="currentColor" stroke-width="1.4"/><circle cx="300" cy="70" r="3" fill="currentColor"/><text x="300" y="98" text-anchor="middle" font-size="9" fill="currentColor">cell B</text>
  <line x1="60" y1="130" x2="400" y2="130" stroke="currentColor" stroke-opacity="0.5" marker-end="url(#char)"/>
  <text x="220" y="145" text-anchor="middle" font-size="8" fill="currentColor">device moves A → B; overlap zone allows make-before-break</text>
  <rect x="205" y="55" width="30" height="30" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/>
  <text x="220" y="30" text-anchor="middle" font-size="8" fill="currentColor">overlap</text>
</svg>
<figcaption>In the overlap between two cells the network moves the session from A to B; soft handover holds both links briefly, hard handover switches at a single instant.</figcaption>
</figure>

## How it works

The handset continuously measures the signal quality of its serving cell and of
neighbouring cells it can hear. When a neighbour becomes sufficiently stronger — subject
to a hysteresis margin and timer that prevent rapid ping-ponging — the network commands a
handover. Two broad styles exist:

- **Hard handover (break-before-make):** the old link is released *before* the new one
  is established, so the device is momentarily connected to exactly one cell. This is the
  norm for [GSM](/reference/gsm/) and [LTE](/reference/lte/), where cells use different
  frequencies or the transition is fast enough to be seamless. LTE handovers are often
  coordinated directly between base stations over the X2 interface.
- **Soft handover (make-before-break):** the device connects to *both* cells at once and
  releases the old one only after the new one is solid. This is characteristic of
  [CDMA](/reference/cdma/) systems, where all cells share one frequency so a phone can
  combine signals from several simultaneously; a special case within one base station's
  own sectors is called **softer handover**.

When there is no active session, the equivalent idle-mode process is **cell
reselection**: the camped-on handset simply re-picks the best cell using slower, more
power-efficient measurements, without any signalling handover.

## Relevance to SDR

Handover is a cellular-network procedure, and the signalling that carries it is part of
the licensed, and on modern networks encrypted, control plane — not something a scanner
tunes directly. It is relevant to the SDR world because the same core problem appears in
the trunked land-mobile systems **GopherTrunk does** decode: as a user drives across a
wide-area network, the radio must move between [neighbor sites](/reference/neighbor-site/)
in a [multisite trunking](/reference/multisite-trunking/) system, and the control channel
announces those site transitions. Following that mobility — tracking a talkgroup as it
roams across sites — is squarely in GopherTrunk's wheelhouse, even though cellular
handover itself is out of scope for its air interfaces. The idea is also distinct from
[roaming](/reference/roaming/), which is switching between *operators* rather than cells.

## Sources

[^wiki]: [Handover](https://en.wikipedia.org/wiki/Handover) — Wikipedia, for the definition of handover and the hard/soft distinction.
[^signal]: [Mobile phone signal](https://en.wikipedia.org/wiki/Mobile_phone_signal) — Wikipedia, for cell coverage, measurement-driven mobility, and reselection.
