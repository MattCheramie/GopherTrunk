---
slug: system-fusion-ysf
title: System Fusion (YSF)
entry_type: protocol
category: amateur-digital
description: "System Fusion (Yaesu System Fusion, YSF) is Yaesu's amateur C4FM digital-voice system, supporting digital and analog modes with an AMBE-family vocoder and internet-linked rooms."
keywords: System Fusion, YSF, Yaesu, C4FM, amateur digital voice, Fusion, Wires-X, AMBE, AMS, DN VW mode, digital narrow, voice full rate
aka: [System Fusion, Yaesu System Fusion, YSF, Fusion, WIRES-X]
autolink: true
infobox:
  - { label: Type, value: Amateur digital voice }
  - { label: Developer, value: Yaesu }
  - { label: Access, value: FDMA }
  - { label: Modulation, value: C4FM (4-level FSK) }
  - { label: Vocoder, value: AMBE family }
  - { label: GopherTrunk support, value: See Status }
see_also: [d-star, m17, dmr, c4fm, four-fsk, ambe]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/System_Fusion
  - https://systemfusion.yaesu.com/
---

**System Fusion** (**Yaesu System Fusion**, **YSF**) is Yaesu's amateur digital-voice
system, using [C4FM](/reference/c4fm/) modulation — the same
[4-level FSK](/reference/four-fsk/) family as [P25 Phase 1](/reference/p25-phase-1/) —
and an [AMBE](/reference/ambe/)-family [vocoder](/reference/vocoder/). Its signature
feature is **Automatic Mode Select (AMS)**: a Fusion repeater or radio inspects each
incoming transmission and switches itself between digital and conventional analog FM
on a call-by-call basis, so digital and analog users share one channel without manual
mode juggling.[^wiki] It is the third of the big three amateur digital systems
alongside [D-STAR](/reference/d-star/) and amateur [DMR](/reference/dmr/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 110" role="img" aria-label="System Fusion digital voice linked over WIRES-X rooms, bridging analog and digital users." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="44" width="70" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="65" y="63">radio</text>
    <rect x="150" y="44" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="190" y="63">repeater</text>
    <rect x="290" y="44" width="120" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="350" y="58">internet</text><text x="350" y="69" font-size="8">WIRES-X rooms</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="100" y1="59" x2="149" y2="59" marker-end="url(#am_system-fusion-ysf)"/><line x1="230" y1="59" x2="289" y2="59" marker-end="url(#am_system-fusion-ysf)"/></g>
    <text x="65" y="30" font-size="8">C4FM</text>
  </g>
  <defs><marker id="am_system-fusion-ysf" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>System Fusion (YSF) uses C4FM and can automatically bridge analog and digital users, linked worldwide via WIRES-X rooms.</figcaption>
</figure>

## Overview

System Fusion transmits at 4800 symbols/s (9600 bps) using C4FM, four amplitude
levels of FSK that map two bits per symbol. What makes Fusion distinctive is that a
single C4FM waveform is used across several **communication modes** the operator can
choose per transmission, trading voice quality against data:

- **V/D (Voice + Data, "DN")** — the default digital mode: about half the payload is
  the AMBE voice frame and the other half is simultaneous data (callsign, GPS,
  short messages), so text and position ride alongside speech.
- **Voice FR (Full Rate, "VW", Voice Wide)** — the whole payload is voice for higher
  audio fidelity, with no simultaneous data.
- **Data FR** — the whole payload is data, used for image and file transfer.
- **Analog FM** — ordinary narrowband FM, selected automatically by AMS when the
  incoming signal is analog.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [FDMA](/reference/fdma/) |
| Modulation | C4FM (4-level FSK), 4800 sym/s = 9600 bps |
| Channel spacing | 12.5 kHz |
| Vocoder | AMBE+2 family (DVSI) |
| Modes | DN (voice+data), VW (full-rate voice), Data FR, analog FM |
| Mode select | Automatic Mode Select (AMS) per transmission |
| Networking | WIRES-X rooms + reflectors |

## How it works

Because Fusion and P25 Phase 1 share the C4FM physical layer, the demodulation front
end is essentially identical: a 4-level eye, root-raised-cosine-shaped, recovered with
a symbol-timing loop and sliced into [dibits](/reference/dibit/). What differs is the
frame structure above the symbols — Fusion's sync pattern, its interleaving and FEC,
and its AMBE+2 voice frames are Yaesu's own, distinct from P25's Reed-Solomon and
Golay-protected framing. The AMS decision is made by reading the incoming frame sync:
a Fusion sync means switch to digital, its absence means fall back to analog FM. Over
the network side, **WIRES-X** (Wide-coverage Internet Repeater Enhancement System)
links Fusion repeaters and personal nodes into "rooms," the Fusion equivalent of
D-STAR reflectors and DMR talkgroups.

## History

Yaesu introduced System Fusion in 2013 as its entry into amateur digital voice,
positioning AMS and analog/digital coexistence as the migration-friendly answer to
D-STAR and DMR.[^yaesu] To seed adoption it sold repeaters at low cost and shipped
C4FM in mainstream handhelds and mobiles. Like the other AMBE-based systems it
depends on a licensed DVSI vocoder, which — together with D-STAR and DMR — motivated
the fully open [M17](/reference/m17/) protocol and its [Codec 2](/reference/codec2/)
vocoder.

## Deployment

Fusion is used by amateur operators worldwide through Fusion repeaters, personal
hotspots, and WIRES-X rooms. Multiprotocol reflectors and hotspot firmware now bridge
Fusion with D-STAR and DMR, so a single operator's traffic can appear on any of the
three networks. It is popular with clubs precisely because AMS lets a repeater serve
analog members and digital members on one machine during a transition period.

## Decoding it with GopherTrunk

GopherTrunk already handles C4FM for [P25 Phase 1](/reference/p25-phase-1/), so the
same demodulator, timing recovery, and symbol slicer apply to Fusion's physical
layer; recovering YSF's frame sync and link-layer metadata is a framing problem on top
of that shared front end. As with D-STAR and DMR, the **voice is AMBE+2** — a licensed
proprietary vocoder — so GopherTrunk's honest scope is the link layer and metadata,
not re-synthesised audio. See [Status](/status.html) for GopherTrunk's current YSF
coverage.

## Sources

[^wiki]: [System Fusion](https://en.wikipedia.org/wiki/System_Fusion) — Wikipedia, for Yaesu's amateur C4FM digital-voice system, its communication modes, Automatic Mode Select, and WIRES-X linking.
[^yaesu]: [Yaesu System Fusion](https://systemfusion.yaesu.com/) — Yaesu's official System Fusion site, for C4FM, AMS, the DN/VW modes, and the WIRES-X network.
