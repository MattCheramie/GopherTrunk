---
slug: tdma-vs-fdma
title: "TDMA vs. FDMA: fitting more calls on a channel"
description: FDMA gives one call per frequency; TDMA splits a frequency into time slots so several calls share it. How DMR, P25 Phase 2, and TETRA multiply capacity, and what it means for a decoder.
keywords: TDMA, FDMA, time slot, channel access, DMR two slot, P25 Phase 2, TETRA four slot, 6.25 kHz equivalent, capacity, time division, slot tracking, digital trunking
level: intermediate
status: full
prereq:
  - framing-fec-interleaving
faq:
  - q: What is the difference between TDMA and FDMA?
    a: FDMA, frequency-division multiple access, gives each call its own frequency, so one channel carries one conversation. TDMA, time-division multiple access, splits a single frequency into repeating time slots, so several calls take turns on the same frequency. TDMA packs more calls into the same spectrum at the cost of more complex timing.
  - q: Which trunking systems use TDMA and which use FDMA?
    a: P25 Phase 1 and NXDN are FDMA, one call per frequency. DMR and P25 Phase 2 use two-slot TDMA, so two calls share a channel. TETRA uses four-slot TDMA. Knowing which a system uses tells you how many simultaneous calls a single voice frequency can carry.
  - q: How does two-slot TDMA give 6.25 kHz-equivalent capacity?
    a: A DMR or P25 Phase 2 channel is 12.5 kHz wide but carries two independent voice paths in alternating time slots. Two calls in 12.5 kHz works out to the same spectrum per call as a 6.25 kHz channel would provide, which is why these systems are described as 6.25 kHz-equivalent.
  - q: What does TDMA mean for a decoder?
    a: With TDMA the decoder must track which time slot it is looking at, because slot 1 and slot 2 can carry entirely different calls and talkgroups. It demodulates the whole channel continuously but separates the bursts by slot, following one call in slot 1 while another runs in slot 2.
---

# TDMA vs. FDMA: fitting more calls on a channel

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**FDMA** (frequency-division multiple access) puts **one call per frequency** — the way
**P25 Phase 1** and **NXDN** work. **TDMA** (time-division multiple access) splits one
frequency into **repeating time slots** so several calls share it: **DMR** and **P25
Phase 2** use **2 slots**, **TETRA** uses **4**. Two voice paths in a 12.5 kHz channel
works out to roughly **6.25 kHz-equivalent** capacity per call. For a decoder, TDMA
adds a job: you must **track which slot** a call is in, because each slot can carry a
different conversation and talkgroup.
</div>

The last lesson packed bits into frames and bursts. This lesson asks the next
question: how do many users share a limited set of frequencies *at the same instant*?
There are two answers, and a system's choice between them shapes its capacity, its
channel plan, and how you decode it.

## FDMA: one call per frequency

**FDMA** is the straightforward approach: every call gets its **own frequency** for its
duration. A trunked FDMA system still shares frequencies *over time* — the
[control channel](/learn/digital-trunking/the-control-channel/) hands a free one to each call and reclaims it
afterward — but at any single moment, one frequency carries exactly one conversation.
**P25 Phase 1** and **NXDN** are FDMA.

FDMA is simple to reason about and simple to decode: tune the voice frequency and the
call is *there*, the only one on it. The downside is capacity. To carry more
simultaneous calls you need more frequencies, and spectrum is the scarce, expensive
resource trunking exists to conserve.

## TDMA: many calls per frequency

**TDMA** squeezes more out of each frequency by dividing **time**. The channel is cut
into short, repeating **time slots**, and each slot is an independent path. Slot 1 and
slot 2 alternate continuously, so two calls run on the *same frequency* without ever
colliding — each simply uses its own slots. **DMR** and **P25 Phase 2** use **two
slots**; **TETRA** uses **four**.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 150" role="img" aria-label="A timeline of one TDMA frequency divided into alternating time slots. The top row shows slot 1 bursts carrying call A; the bottom row shows slot 2 bursts carrying call B; together they fill one frequency." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="28" font-size="11" fill="currentColor" font-weight="600">One 12.5 kHz frequency, two time slots:</text>
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <rect x="20" y="45" width="60" height="30" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.3"/><text x="50" y="64">A</text>
    <rect x="80" y="45" width="60" height="30" rx="3" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.3"/><text x="110" y="64">B</text>
    <rect x="140" y="45" width="60" height="30" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.3"/><text x="170" y="64">A</text>
    <rect x="200" y="45" width="60" height="30" rx="3" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.3"/><text x="230" y="64">B</text>
    <rect x="260" y="45" width="60" height="30" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.3"/><text x="290" y="64">A</text>
    <rect x="320" y="45" width="60" height="30" rx="3" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.3"/><text x="350" y="64">B</text>
    <rect x="380" y="45" width="60" height="30" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.3"/><text x="410" y="64">A</text>
    <rect x="440" y="45" width="60" height="30" rx="3" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.3"/><text x="470" y="64">B</text>
  </g>
  <text x="20" y="100" font-size="10" fill="currentColor">slot 1 (call A) ▦</text>
  <text x="200" y="100" font-size="10" fill="currentColor">slot 2 (call B) ▦</text>
  <line x1="20" y1="115" x2="500" y2="115" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#t1)"/>
  <text x="260" y="135" text-anchor="middle" font-size="10" fill="currentColor">time →</text>
  <defs><marker id="t1" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A single TDMA frequency alternates between slot 1 (call A) and slot 2 (call B). Two independent conversations share one frequency by taking turns faster than the ear can notice.</figcaption>
</figure>

The slots cycle far faster than a conversation, and the vocoder frames are buffered so
neither talker hears a gap — each call sounds continuous even though it's only
on the air half (or a quarter) of the time.

## The capacity win

The payoff is spectrum efficiency. A DMR or P25 Phase 2 channel is **12.5 kHz** wide
but carries **two** voice paths. Two calls in 12.5 kHz is the same amount of spectrum
per call as a **6.25 kHz** channel would give — which is why these systems are marketed
as **"6.25 kHz-equivalent."** TETRA's four slots in its channel push the sharing
further still.

| System | Access | Slots | Calls per voice frequency |
|--------|--------|-------|---------------------------|
| P25 Phase 1 | FDMA | 1 | 1 |
| NXDN | FDMA | 1 | 1 |
| DMR | TDMA | 2 | 2 |
| P25 Phase 2 | TDMA | 2 | 2 |
| TETRA | TDMA | 4 | 4 |

This is one of the headline reasons systems migrate from P25 Phase 1 to Phase 2:
**double the voice capacity** on the same frequencies, without buying more spectrum.

## What TDMA means for a decoder

FDMA is easy on a decoder — tune the frequency, decode the one call. TDMA adds a
crucial extra job: **slot tracking**. The decoder demodulates the whole channel
continuously, but it must sort the incoming bursts by slot, because **slot 1 and slot 2
can be entirely different calls**, with different talkgroups and radio IDs. Follow the
wrong slot and you'd splice two conversations together.

Practically, this means a TDMA channel grant from the
[control channel](/learn/digital-trunking/the-control-channel/) specifies not just a frequency but a **slot**,
and the decoder honours both. GopherTrunk handles this internally when it follows a
DMR or P25 Phase 2 call — it locks the right frequency *and* the right slot — but it's
worth knowing it's happening, because a system that decodes on one slot and not the
other is a slot-tracking symptom, not a tuning one.

<div class="knowledge-check" data-quiz data-correct-msg="Right — TDMA splits a frequency into repeating time slots, so several calls share it." markdown="0">
  <p class="knowledge-check__q">Quick check: how does TDMA fit more than one call on a single frequency?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It gives each call a slightly different frequency</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It splits the frequency into repeating time slots the calls take turns using</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It compresses both calls into one vocoder stream</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **FDMA** gives **one call per frequency** — used by P25 Phase 1 and NXDN.
- **TDMA** splits a frequency into **repeating time slots** — DMR and P25 Phase 2 use
  **2**, TETRA uses **4**.
- Two voice paths in a 12.5 kHz channel ≈ **6.25 kHz-equivalent**, the capacity win
  behind Phase 1 → Phase 2 migration.
- A TDMA decoder must **track which slot** a call is in, because each slot is a separate
  conversation.
- A channel grant on TDMA names both a **frequency and a slot**.

Next, we meet the channel that coordinates all of this:
[the control channel](/learn/digital-trunking/the-control-channel/), the system's heartbeat.
