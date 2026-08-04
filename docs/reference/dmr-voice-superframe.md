---
slug: dmr-voice-superframe
title: DMR voice superframe
entry_type: term
category: trunked-radio
description: A DMR voice superframe is six bursts A–F spanning 360 ms, each carrying three 72-bit AMBE+2 voice frames (18 total), with embedded link-control signalling spread across bursts B–E and burst A framed by a voice sync word that anchors the TDMA cadence.
keywords: DMR voice superframe, bursts A to F, 18 AMBE frames, 360 ms, embedded signalling B-E, voice sync, TDMA cadence, ETSI TS 102 361-1 6.2
aka: ["voice superframe", "DMR voice superframe"]
autolink: true
infobox:
  - { label: Span, value: "6 bursts A–F, 360 ms" }
  - { label: Voice, value: 18 × 72-bit AMBE+2 frames }
  - { label: Signalling, value: embedded LC in bursts B–E }
  - { label: Spec, value: ETSI TS 102 361-1 §6.2 }
see_also: [ambe-plus-2, ambe-plus-2-fec, dmr-emb, dmr-embedded-lc, dmr-reverse-channel, dmr-sync-patterns, dmr-cach, tdma, vocoder, dmr]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://en.wikipedia.org/wiki/Time-division_multiple_access
---

A **DMR voice superframe** is the repeating unit of a DMR voice call: six
[bursts](/reference/dmr-burst/) labelled A–F, spanning 360 ms, carrying **18** on-air
[AMBE+2](/reference/ambe-plus-2/) voice frames of 72 bits each.[^wiki] Burst A is framed by a voice
[sync word](/reference/dmr-sync-patterns/); bursts B–F carry no sync and instead put
[embedded signalling](/reference/dmr-emb/) in their central field, so they are located not by a
sync search but by [TDMA](/reference/tdma/) cadence relative to burst A.[^tdma]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 140" role="img" aria-label="Six DMR bursts labelled A through F in a row; burst A carries a voice sync word, bursts B through E carry embedded link-control fragments one through four, and every burst carries three AMBE-plus-2 voice frames for eighteen frames total across the superframe." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor">
    <rect x="12" y="30" width="70" height="30" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/><text x="47" y="49" text-anchor="middle">A · sync</text>
    <rect x="86" y="30" width="70" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="121" y="49" text-anchor="middle">B · LC1</text>
    <rect x="160" y="30" width="70" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="195" y="49" text-anchor="middle">C · LC2</text>
    <rect x="234" y="30" width="70" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="269" y="49" text-anchor="middle">D · LC3</text>
    <rect x="308" y="30" width="70" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="343" y="49" text-anchor="middle">E · LC4</text>
    <rect x="382" y="30" width="70" height="30" fill="none" stroke="currentColor" stroke-width="1"/><text x="417" y="49" text-anchor="middle">F</text>
  </g>
  <text x="232" y="86" text-anchor="middle" font-size="8" fill="currentColor">each burst = 3 × 72-bit AMBE+2 frame → 18 frames / superframe</text>
  <text x="232" y="104" text-anchor="middle" font-size="8" fill="currentColor">bursts B–E fragments 1–4 → one 72-bit embedded Link Control</text>
  <text x="232" y="122" text-anchor="middle" font-size="8" fill="currentColor">360 ms · TDMA cadence locates B–F from burst A</text>
</svg>
<figcaption>Six bursts carry 18 AMBE+2 voice frames; burst A is sync-framed while bursts B–E each contribute one of the four embedded link-control fragments, and burst F carries voice only.</figcaption>
</figure>

## Structure

GopherTrunk decodes the superframe in `internal/radio/dmr/voice/superframe.go` into a
`VoiceSuperframe` holding 18 AMBE frames in transmission order (bursts A–F, three per burst). The
central-field roles across the six bursts are:

| Burst | Central field | Voice frames |
|---|---|---|
| A | voice sync word (anchors the frame) | 3 |
| B | embedded LC fragment 1 (LCSS First) | 3 |
| C | embedded LC fragment 2 (Continuation) | 3 |
| D | embedded LC fragment 3 (Continuation) | 3 |
| E | embedded LC fragment 4 (LCSS Last) | 3 |
| F | embedded signalling (no LC fragment) | 3 |

The four fragments from bursts B–E reassemble into a 72-bit
[embedded Link Control](/reference/dmr-embedded-lc/); when it passes its BPTC + checksum, its
destination and source addresses label the call. A single-fragment (`LCSS == Single`) field in any of
those bursts instead carries a [Reverse Channel](/reference/dmr-reverse-channel/) word or the null idle.

## Cadence and phase lock

Only burst A produces a sync match, so the decoder locks onto it and slices B–F at a fixed same-slot
stride. On a contiguous single-slot stream that stride is 132 dibits; on a real 2-slot TDMA carrier
the other timeslot's burst sits between each of a call's bursts, making same-slot bursts 264 dibits
apart (no inter-burst [CACH](/reference/dmr-cach/)) or 288 when a 12-dibit CACH precedes each burst on
live outbound air. `NewInterleavedDecoder` auto-detects which stride is in use per call — a
CRC-valid embedded LC is authoritative and locks immediately, and absent one it scores each candidate
by [AMBE FEC](/reference/ambe-plus-2-fec/) quality (the Golay-corrected-bit count across all 18 frames)
and locks the clearly-best cadence. That AMBE-quality fallback is what let a 288-cadence CACH carrier
decode even when its embedded LC never did (issue #644). Each superframe also carries a `Phase` — the
burst-A anchor's parity over the physical-burst period — which is a stable relative discriminator
between the two interleaved calls, not an absolute TS1/TS2 label.

## Relevance to SDR

The superframe is the frame GopherTrunk's DMR voice pipeline actually operates on: locking burst A,
slicing the six bursts at the right cadence, pulling 18 AMBE+2 frames for the [vocoder](/reference/vocoder/),
and reassembling the embedded LC to name the call. Handling the 264-vs-288 cadence and the phase lock
correctly is what keeps a decoder from pulling dibits out of the wrong timeslot on a busy carrier — the
difference between clean audio for one talkgroup and a garble of two interleaved calls. Because burst A's
BS-sourced sync is identical on both slots, the embedded LC (or `Phase` bound to a talkgroup) is the only
thing that tells the two apart.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on the DMR standard and its voice superframe.
[^tdma]: [Time-division multiple access](https://en.wikipedia.org/wiki/Time-division_multiple_access) — Wikipedia, on the two-slot framing the superframe's cadence rides.
