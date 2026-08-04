---
slug: p25-phase-2
title: "P25 Phase 2: TDMA"
description: P25 Phase 2 explained — two-slot TDMA traffic channels for double voice capacity, H-DQPSK/H-CPM modulation, the AMBE+2 half-rate vocoder, ISCH sync, and what changes for a decoder.
keywords: P25 Phase 2, TDMA, H-DQPSK, H-CPM, AMBE+2, ISCH, two-slot, 6.25 kHz equivalent, P25 traffic channel, half-rate vocoder, time slot
level: advanced
status: full
prereq:
  - p25-phase-1
  - tdma-vs-fdma
faq:
  - q: What does P25 Phase 2 add over Phase 1?
    a: Phase 2 puts two-slot TDMA on the traffic channels, so a single 12.5 kHz voice channel carries two simultaneous calls — about 6.25 kHz of spectrum per call. It doubles voice capacity without needing new frequencies. The control channel usually stays a Phase 1 C4FM channel, so a system is effectively Phase 1 signaling with Phase 2 traffic.
  - q: What modulation does P25 Phase 2 use?
    a: Phase 2 traffic channels use a phase-modulated scheme — H-DQPSK on the inbound (subscriber-to-system) link and H-CPM on the outbound link — rather than the C4FM of Phase 1. These are engineered for the tight timing a two-slot TDMA channel needs. The control channel typically remains C4FM.
  - q: What vocoder does P25 Phase 2 use?
    a: Phase 2 uses the AMBE+2 half-rate vocoder instead of Phase 1's IMBE. Half-rate voice fits two conversations into the bits of one Phase 1 channel, which is what makes the two-slot TDMA capacity gain possible. AMBE+2 is patent-encumbered, unlike the now-expired IMBE patents.
  - q: Why must a Phase 2 decoder track two time slots?
    a: Because a single Phase 2 traffic frequency interleaves two independent calls in alternating time slots. A decoder has to recover slot timing from the ISCH and demultiplex the bursts, treating each slot as its own voice stream. Locking the frequency is not enough — you must lock the slot.
gophertrunk_links:
  - title: CC Activity
    url: /cc-activity.html
    note: the Phase 1 control channel that still grants Phase 2 calls.
  - title: Symbol scope
    url: /symbol-scope.html
    note: inspect the recovered symbols on a TDMA traffic channel.
  - title: Vocoders
    url: /vocoders.html
    note: see how GopherTrunk maps Phase 2 grants to the AMBE+2 decoder.
---

# P25 Phase 2: TDMA

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**P25 Phase 2** doubles voice capacity by putting **two-slot TDMA** on the traffic
channels: one **12.5 kHz** frequency now carries **two simultaneous calls** — roughly
**6.25 kHz equivalent** each. Traffic uses a **phase-modulated** scheme (**H-DQPSK**
inbound, **H-CPM** outbound), not C4FM, and the **AMBE+2 half-rate** vocoder instead of
IMBE. Crucially, the **control channel usually stays a Phase 1 C4FM channel** — so the
signaling is Phase 1 and only the *traffic* is TDMA. For a decoder, the new job is
**tracking two time slots per frequency**, recovering slot timing from the **ISCH**.
</div>

[P25 Phase 1](/learn/digital-trunking/p25-phase-1/) gave each call its own 12.5 kHz frequency. That is clean but
spectrum-hungry, and busy metropolitan systems ran out of channels. **Phase 2** is the
answer: keep the same frequencies, but fit two calls on each traffic channel using
[TDMA](/learn/digital-trunking/tdma-vs-fdma/).

## The capacity idea: two calls, one frequency

In Phase 2 a traffic channel is divided in **time** into **two slots** that alternate in
rapid turns. Two independent calls share the frequency — one in each slot — so a 12.5 kHz
channel does the work of two. The standard often describes this as **6.25 kHz
equivalent** per call: not that the channel is physically narrower, but that each
conversation consumes half the channel's time, achieving the same spectral efficiency as
a 6.25 kHz channel would.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 180" role="img" aria-label="A two-slot TDMA frame for P25 Phase 2. A single frequency is divided into alternating slot 1 and slot 2 bursts in time, with an ISCH sync marker between bursts. Slot 1 carries call A, slot 2 carries call B." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor">
    <text x="260" y="20" text-anchor="middle" font-weight="600">One Phase 2 traffic frequency — two time slots</text>
    <line x1="30" y1="150" x2="490" y2="150" stroke="currentColor" stroke-opacity="0.4"/>
    <text x="260" y="170" text-anchor="middle" font-size="9">time →</text>
    <rect x="40" y="60" width="90" height="70" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="0.8"/><text x="85" y="92" text-anchor="middle" font-size="9">Slot 1</text><text x="85" y="106" text-anchor="middle" font-size="9">call A</text>
    <rect x="130" y="60" width="14" height="70" fill="currentColor" fill-opacity="0.55"/><text x="137" y="48" text-anchor="middle" font-size="8">ISCH</text>
    <rect x="144" y="60" width="90" height="70" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="0.8"/><text x="189" y="92" text-anchor="middle" font-size="9">Slot 2</text><text x="189" y="106" text-anchor="middle" font-size="9">call B</text>
    <rect x="234" y="60" width="14" height="70" fill="currentColor" fill-opacity="0.55"/>
    <rect x="248" y="60" width="90" height="70" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="0.8"/><text x="293" y="92" text-anchor="middle" font-size="9">Slot 1</text><text x="293" y="106" text-anchor="middle" font-size="9">call A</text>
    <rect x="338" y="60" width="14" height="70" fill="currentColor" fill-opacity="0.55"/>
    <rect x="352" y="60" width="90" height="70" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="0.8"/><text x="397" y="92" text-anchor="middle" font-size="9">Slot 2</text><text x="397" y="106" text-anchor="middle" font-size="9">call B</text>
  </g>
</svg>
<figcaption>Two independent calls (A and B) interleave in alternating time slots on a single Phase 2 frequency. The ISCH sync between bursts is what lets a decoder recover slot timing and demultiplex them.</figcaption>
</figure>

## The modulation changes — but not on the control channel

To fit a clean two-slot burst structure, Phase 2 traffic channels drop C4FM in favour of
a **phase-modulated** scheme: **H-DQPSK** (Harmonized Differential QPSK) on the inbound
link from subscriber to system, and **H-CPM** (Harmonized Continuous Phase Modulation) on
the outbound link from system to subscriber. These are tuned for the precise timing a
shared TDMA channel demands.

The important subtlety: in nearly all deployments the **control channel stays a Phase 1
C4FM channel**. A Phase 2 system is therefore *Phase 1 signaling with Phase 2 traffic* —
the same [TSBK](/learn/digital-trunking/control-channel-signaling/) grants you already know, but the channels they
point to are TDMA. This is also why Phase 2 systems remain compatible with the wider P25
world and why a follower's control-channel logic is essentially unchanged.

## Half-rate voice: AMBE+2

Two calls in the bits of one means each call gets **half the data rate**, so Phase 2
swaps Phase 1's IMBE for the **AMBE+2 half-rate** [vocoder](/learn/rf-sdr/vocoders/). The
same family of math, but compressing voice into roughly half the bits — exactly what a
single TDMA slot can carry. Unlike IMBE, whose core patents have expired, **AMBE+2 is
patent-encumbered**; GopherTrunk ships a pure-Go AMBE+2 decoder by default but the patent
risk falls on the deployer (see [Vocoders](/vocoders.html)). Phase 2 grants map to the
`ambe2` vocoder in GopherTrunk's protocol table.

| Aspect | Phase 1 | Phase 2 |
|--------|---------|---------|
| Traffic access | FDMA | 2-slot TDMA |
| Calls per 12.5 kHz | 1 | 2 |
| Spectral efficiency | 12.5 kHz/call | ≈6.25 kHz/call equiv. |
| Traffic modulation | C4FM | H-DQPSK / H-CPM |
| Control channel | C4FM | usually C4FM (Phase 1) |
| Vocoder | IMBE (full-rate) | AMBE+2 (half-rate) |

## Slot structure and ISCH

A Phase 2 frequency carries a repeating sequence of **bursts**, alternating between the
two slots. To make sense of the stream a receiver must know *which slot it is looking at*
and *where each burst begins*. That synchronization comes from the **ISCH**
(Inter-Slot Signaling CHannel) embedded in the burst structure: it provides the sync and
slot-identification markers a decoder uses to align to the frame, distinguish slot 1 from
slot 2, and carry per-slot signaling. Lose ISCH sync and the two calls smear into noise;
hold it and each slot becomes a clean, independent voice stream.

## What changes for a decoder

For GopherTrunk the control-channel side barely changes — it is still reading Phase 1
TSBK grants. The traffic side is where Phase 2 demands more:

- **Demodulate a different scheme.** The traffic channel is H-DQPSK/H-CPM, not C4FM, so
  the symbol recovery differs from Phase 1. You can still inspect recovered symbols on the
  **[Symbol scope](/symbol-scope.html)**.
- **Recover slot timing.** Lock the **ISCH**, find the burst boundaries, and identify
  which slot is which.
- **Demultiplex two calls.** Treat **each slot as its own voice stream**, decoding
  AMBE+2 separately per slot, because a single grant's frequency may host a second,
  unrelated call in the other slot.
- **Map grants to slots.** A Phase 2 grant references not just a frequency but a *slot*,
  so the follower tunes the frequency and then selects the correct time slot.

<div class="knowledge-check" data-quiz data-correct-msg="Right — Phase 2 traffic is two-slot TDMA, so one 12.5 kHz channel carries two calls." markdown="0">
  <p class="knowledge-check__q">Quick check: how does P25 Phase 2 double voice capacity?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">By adding more frequencies to the pool</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">By putting two-slot TDMA on each traffic channel</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">By switching the control channel to AMBE+2</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **P25 Phase 2** adds **two-slot TDMA** on the traffic channels, doubling capacity to
  **two calls per 12.5 kHz** (≈6.25 kHz equivalent each).
- Traffic uses **H-DQPSK/H-CPM** phase modulation; the **control channel usually stays
  Phase 1 C4FM**, so signaling is unchanged.
- Voice uses the **AMBE+2 half-rate** vocoder instead of IMBE.
- The **ISCH** provides the sync and slot identification a decoder needs to demultiplex
  the two slots.
- For a decoder the new work is **tracking two time slots per frequency** and decoding
  each as its own stream.

Next we cross to the other dominant digital standard, with its own three-tier story:
[DMR Tier II & Tier III](/learn/digital-trunking/dmr-tier-2-3/).
