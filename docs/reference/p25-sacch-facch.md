---
slug: p25-sacch-facch
title: P25 SACCH & FACCH
entry_type: term
category: trunked-radio
description: SACCH and FACCH are the P25 Phase 2 associated control channels — the slow and fast paths that carry signalling alongside voice on a traffic channel — delivering MAC PDUs (grants, PTT, encryption sync, talker alias) protected by the P25 Reed-Solomon and trellis codes.
keywords: P25 SACCH, P25 FACCH, associated control channel, Phase 2 signalling, FACCH-S talker alias, MAC_PTT encryption sync, Reed-Solomon trellis, TIA-102.BBAB
aka: [SACCH, FACCH, "FACCH-S", "associated control channel"]
autolink: true
infobox:
  - { label: SACCH, value: Slow Associated Control Channel }
  - { label: FACCH, value: Fast Associated Control Channel }
  - { label: Carries, value: MAC PDUs (grants, PTT, alias) }
  - { label: Spec, value: TIA-102.BBAB }
see_also: [p25-reed-solomon, p25-trellis-code, p25-mac-pdu, p25-phase-2, p25-isch, p25-mac-vendor, p25-phase-2-superframe, ambe-plus-2]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Associated_control_channel
---

**SACCH** (**Slow Associated Control Channel**) and **FACCH** (**Fast Associated Control Channel**) are
the two associated control channels that carry signalling *alongside* voice on a
[P25 Phase 2](/reference/p25-phase-2/) traffic channel.[^wiki] Phase 2 has no separate control channel
during a call: once a system grants a traffic channel, the signalling that keeps the call coordinated —
grant updates, PTT and end-of-transmission, encryption sync, talker alias — rides in the same two-slot
stream as the [AMBE+2](/reference/ambe-plus-2/) voice, folded in as
[MAC PDUs](/reference/p25-mac-pdu/) on sub-frames the [ISCH](/reference/p25-isch/) types as MAC.[^acch]
The difference between the two channels is *urgency*: FACCH preempts capacity for time-critical
messages, while SACCH trickles slower signalling across the superframe.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="Across a superframe, most sub-frames carry voice while occasional sub-frames carry signalling: FACCH steals a voice frame's capacity for urgent messages like PTT and grant updates, and SACCH occupies dedicated signalling sub-frames for slower messages such as the talker alias during hangtime." xmlns="http://www.w3.org/2000/svg">
  <g font-size="7" fill="currentColor" text-anchor="middle">
    <rect x="20" y="34" width="40" height="24" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/><text x="40" y="49">voice</text>
    <rect x="60" y="34" width="40" height="24" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/><text x="80" y="49">voice</text>
    <rect x="100" y="34" width="40" height="24" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1"/><text x="120" y="49">FACCH</text>
    <rect x="140" y="34" width="40" height="24" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/><text x="160" y="49">voice</text>
    <rect x="180" y="34" width="40" height="24" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/><text x="200" y="49">voice</text>
    <rect x="220" y="34" width="40" height="24" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1"/><text x="240" y="49">FACCH</text>
    <rect x="260" y="34" width="40" height="24" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/><text x="280" y="49">voice</text>
    <rect x="300" y="34" width="40" height="24" fill="currentColor" fill-opacity="0.20" stroke="currentColor" stroke-width="1" stroke-dasharray="3 2"/><text x="320" y="49">SACCH</text>
  </g>
  <text x="20" y="82" font-size="8" fill="currentColor">FACCH: urgent — steals voice capacity (PTT, grant update, encryption sync)</text>
  <text x="20" y="100" font-size="8" fill="currentColor">SACCH: slow — dedicated signalling (talker alias on FACCH-S during hangtime)</text>
  <text x="20" y="128" font-size="8" fill="currentColor">both are MAC PDUs · outer Reed-Solomon RS(24,16,9) + trellis FEC</text>
</svg>
<figcaption>During a call, signalling rides in-band with voice: FACCH steals a voice frame for urgent MAC PDUs, SACCH uses dedicated signalling sub-frames for slower ones, and both are protected by the same RS + trellis FEC as the control channel.</figcaption>
</figure>

## Fast versus slow

**FACCH** is the fast path: it *borrows* a voice frame's slot to inject an urgent MAC PDU, trading a
moment of speech for immediate signalling. It is where the messages that must not wait travel — a
call's PTT open (which in Phase 2 also carries the ALGID/KID/MI encryption sync), grant updates that
retune following radios mid-call, and end-of-transmission. **SACCH** is the slow path: it occupies
sub-frames set aside for signalling and carries messages that can be spread out over the superframe,
where stealing voice capacity is not warranted. The voice frames a FACCH steals are simply not decoded
for that slot, so a decoder that ignores the associated control channels would still hear the call —
just without following its state.

A practical consequence in GopherTrunk: because the Phase 2 encryption sync is delivered in the
`MAC_PTT` message rather than as a distinct opcode, a caller that needs to tell PTT signalling from
ordinary FACCH/SACCH signalling must inspect the sub-frame's `SlotType`, not the MAC PDU alone. The
[superframe](/reference/p25-phase-2-superframe/) decoder therefore pairs each decoded PDU with the slot
type it rode in. The Motorola talker alias is a FACCH-S message: it rides the fast associated control
channel during call *hangtime*, as the header/data
[vendor MAC opcodes](/reference/p25-mac-vendor/) 0x91 and 0x95.

## Outer FEC

Whatever the associated channel, the signalling payload is protected by the P25 Phase 2 FEC stack: the
inner ½-rate [trellis code](/reference/p25-trellis-code/) that a Viterbi decoder inverts, and the outer
[Reed-Solomon RS(24,16,9)](/reference/p25-reed-solomon/) code that detects — or, in the corrector mode,
repairs up to four symbol errors in — the recovered MAC PDU. The RS check is also what makes the
descrambler's blind slot-offset probe safe, since only the correct phase yields zero syndromes. This is
the same coding pairing the Phase 2 control channel uses, so signalling that rides alongside voice is
protected exactly as strongly as signalling on a dedicated control channel.

## Relevance to SDR

GopherTrunk does not carve out separate SACCH and FACCH code paths; both surface as MAC PDUs on
MAC-typed sub-frames, distinguished by `SlotType`, and are decoded by the shared Phase 2 MAC chain in
`internal/radio/p25/phase2`. The talker-alias FACCH-S handling lives in `talker_alias.go` and
`mac_vendor.go`. Following these associated channels is what turns a decoded Phase 2 call from bare audio
into a fully attributed event — who is talking, on what talkgroup, encrypted or not. The spec is
TIA-102.BBAB.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on P25 Phase 2 traffic-channel signalling.
[^acch]: [Associated control channel](https://en.wikipedia.org/wiki/Associated_control_channel) — Wikipedia, on slow and fast associated control channels that carry signalling in-band with voice.
