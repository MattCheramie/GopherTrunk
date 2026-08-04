---
slug: nxdn-facch
title: NXDN FACCH
entry_type: term
category: trunked-radio
description: "The NXDN Fast Associated Control Channel (FACCH) is the in-band signalling variant that steals a traffic frame's 144-dibit information field to carry an urgent control message in place of voice, delivering it in a single frame rather than across a superframe like the SACCH."
keywords: NXDN FACCH, fast associated control channel, bit stealing, information field, in-band signalling, NXDN traffic channel, FACCH vs SACCH
aka: [FACCH, "fast associated control channel"]
autolink: true
infobox:
  - { label: Carried in, value: 144-dibit information field }
  - { label: Mode, value: steals a frame from voice/data }
  - { label: Versus SACCH, value: "one frame, not a superframe" }
  - { label: Spec, value: NXDN TS 1-A §6.5 }
see_also: [nxdn, nxdn-frame-structure, nxdn-sacch, nxdn-cac, nxdn-lich, control-channel, channel-grant]
cite_urls:
  - https://en.wikipedia.org/wiki/NXDN
  - https://en.wikipedia.org/wiki/Associated_Control_Channel
---

The **NXDN Fast Associated Control Channel** (**FACCH**) is the in-band signalling variant
that carries an urgent control message on a traffic channel by **stealing** a frame's 144-dibit
[information field](/reference/nxdn-frame-structure/) — the field that would otherwise hold
voice or data — and using it for signalling instead.[^wiki] Where the
[SACCH](/reference/nxdn-sacch/) trickles a message out fragment by fragment across a
superframe, the FACCH delivers a complete message in a single frame, at the cost of one frame's
worth of audio.[^acch] It is NXDN's answer to the times when signalling cannot wait for the
slow channel: call setup and teardown, emergency indication, and other events that must land
immediately.

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 150" role="img" aria-label="A row of NXDN traffic frames whose information fields normally carry voice; one frame in the row has its information field stolen and re-used to carry a FACCH control message, while the frame sync word, LICH, and SACCH fields remain unchanged in every frame." xmlns="http://www.w3.org/2000/svg">
  <g font-size="7" fill="currentColor">
    <!-- frame 1 -->
    <rect x="14" y="40" width="10" height="30" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="0.8"/>
    <rect x="24" y="40" width="10" height="30" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="0.8"/>
    <rect x="34" y="40" width="16" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="0.8"/>
    <rect x="50" y="40" width="90" height="30" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="0.8"/>
    <text x="95" y="59" text-anchor="middle">voice</text>
    <!-- frame 2 stolen -->
    <rect x="150" y="40" width="10" height="30" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="0.8"/>
    <rect x="160" y="40" width="10" height="30" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="0.8"/>
    <rect x="170" y="40" width="16" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="0.8"/>
    <rect x="186" y="40" width="90" height="30" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="1.3"/>
    <text x="231" y="59" text-anchor="middle">FACCH</text>
    <!-- frame 3 -->
    <rect x="286" y="40" width="10" height="30" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="0.8"/>
    <rect x="296" y="40" width="10" height="30" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="0.8"/>
    <rect x="306" y="40" width="16" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="0.8"/>
    <rect x="322" y="40" width="90" height="30" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="0.8"/>
    <text x="367" y="59" text-anchor="middle">voice</text>
  </g>
  <text x="14" y="92" font-size="7" fill="currentColor">FSW · LICH · SACCH stay put in every frame; only the info field is stolen</text>
  <path d="M231 74 L231 86" stroke="currentColor" stroke-width="1" marker-end="url(#f)"/>
  <text x="231" y="98" text-anchor="middle" font-size="7.5" fill="currentColor">one frame of signalling replaces one frame of audio</text>
  <defs><marker id="f" markerWidth="6" markerHeight="6" refX="5" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>On a traffic channel the FACCH borrows a single frame's information field for a control message; the frame sync word, LICH, and SACCH remain in place, so the receiver stays synchronised and simply routes that frame's payload to the signalling decoder instead of the vocoder.</figcaption>
</figure>

## Fast versus slow

NXDN carries two associated control channels on a traffic link, and the split is a classic
latency-versus-overhead trade. The SACCH is always present but tiny (26 payload bits per
frame) and slow: a full message takes a superframe to arrive, but it never interrupts the
audio because it lives in its own dedicated slot. The FACCH is the opposite: it appears only
when needed, and when it does it takes the entire 288-bit information field for one frame, so
it can carry a whole message at once — fast — but it costs that frame's voice. A system uses
the SACCH for continuous background state and reaches for the FACCH when an event (a grant, a
disconnect, an emergency) must be signalled without a superframe of delay.

Crucially, bit-stealing touches only the information field. The [FSW](/reference/nxdn-fsw/),
[LICH](/reference/nxdn-lich/), and SACCH are present and coded identically whether the frame
carries voice or a FACCH message, so the receiver never loses frame sync during a steal — it
recognises the frame as FACCH (from the LICH function bits and context) and hands the stolen
information field to the signalling decoder rather than the vocoder.

## Relevance to SDR

In GopherTrunk, `internal/radio/nxdn/frame.go` names FACCH as one of the payload types the
144-dibit information field can carry — alongside CAC on the control channel and VCH/UDCH voice
and data on traffic — and the `Frame.Info()` accessor returns exactly that field for whichever
decoder the context selects. The frame geometry, sync, LICH, and SACCH decode identically
regardless, so recognising a stolen frame is a routing decision on top of an already-locked
frame rather than a separate sync problem. The dedicated FACCH channel-coder (its own FEC and
CRC over the stolen field) follows the same convolution/puncture/interleave family as the
[CAC](/reference/nxdn-cac/); GopherTrunk's current NXDN surface focuses the structured decode
on the control-channel CAC path and the frame/LICH/SACCH machinery, with the traffic-side voice
and FACCH extraction called out as follow-on work. Getting the distinction right matters
because a FACCH frame mistaken for voice produces a burst of vocoder noise, while a voice frame
mistaken for FACCH drops audio — the LICH and frame context are what keep the two apart.

## Sources

[^wiki]: [NXDN](https://en.wikipedia.org/wiki/NXDN) — Wikipedia, on the NXDN standard and its logical channels.
[^acch]: [Associated Control Channel](https://en.wikipedia.org/wiki/Associated_Control_Channel) — Wikipedia, on the fast/slow associated-control-channel concept shared across digital land-mobile standards.
