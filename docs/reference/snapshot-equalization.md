---
slug: snapshot-equalization
title: Snapshot equalization
entry_type: algorithm
category: equalization
description: "Snapshot equalization adapts an equalizer's taps continuously but applies only frozen copies of them, so each burst sees a constant filter — the design that makes adaptive equalization safe ahead of a differential decoder."
keywords: snapshot equalization, frozen taps, SnapshotCMA, SnapshotLMS, differential decoder, phase wander, CMA rotation invariance, burst equalization, TETRA equalizer, adapt and freeze
aka: [frozen-tap equalization, snapshot CMA, snapshot LMS]
autolink: true
infobox:
  - { label: Type, value: Equalizer application strategy }
  - { label: Rule, value: "Adapt continuously, apply a frozen snapshot" }
  - { label: Solves, value: Phase wander ahead of differential decoders }
  - { label: In GopherTrunk, value: SnapshotCMA (blind) and SnapshotLMS (trained) }
see_also: [cma-equalizer, lms-algorithm, differential-decoding, adaptive-filter, pi-4-dqpsk, log-likelihood-ratio, equalizer-gotchas]
related_reading:
  - { title: "Weak-Signal Engineering, Part 5: The Snapshot Trick — Frozen Taps & Differential Decoders", url: /blog/deep-dives/weak-signal-engineering-05-snapshot-trick/ }
  - { title: "TETRA End to End, Part 9: The Equalizer and the Voice Path", url: /blog/deep-dives/tetra-end-to-end-09-equalizer-voice-path/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Blind_equalization
  - https://en.wikipedia.org/wiki/Differential_coding
---

**Snapshot equalization** is a way of *applying* an adaptive equalizer, not a new way of
adapting one: a tracking filter adapts continuously in the background, but the signal is
filtered through a **frozen copy** of the taps, refreshed only between bursts, so every
burst sees one constant filter.[^wiki] The design exists to resolve a genuine dilemma that
appears whenever an adaptive equalizer feeds a
[differential decoder](/reference/differential-decoding/) — and both horns of the dilemma
score exactly zero.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A tracking equalizer adapts continuously along the top path while frozen snapshots of its taps are copied down at intervals to the apply filter that actually processes each burst." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="26" width="150" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3" stroke-dasharray="4 3"/>
  <text x="105" y="45" font-size="9" fill="currentColor" text-anchor="middle">tracking taps (adapt always)</text>
  <rect x="30" y="94" width="150" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="105" y="113" font-size="9" fill="currentColor" text-anchor="middle">applied taps (frozen)</text>
  <line x1="105" y1="58" x2="105" y2="92" stroke="currentColor" stroke-width="1.2" marker-end="url(#snar)"/>
  <text x="112" y="78" font-size="8" fill="currentColor">copy between bursts</text>
  <line x1="182" y1="109" x2="242" y2="109" stroke="currentColor" stroke-width="1.2" marker-end="url(#snar)"/>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <rect x="246" y="99" width="52" height="20" rx="3"/><rect x="304" y="99" width="52" height="20" rx="3"/><rect x="362" y="99" width="52" height="20" rx="3"/>
  </g>
  <text x="330" y="93" font-size="8" fill="currentColor" text-anchor="middle">each burst sees one constant filter</text>
  <defs><marker id="snar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The tracking filter follows the channel; the applied filter never moves mid-burst, so the equalizer's phase is constant across every differential pair.</figcaption>
</figure>

## The dilemma it resolves

A blind [CMA equalizer](/reference/cma-equalizer/)'s cost depends only on output
*magnitude*, so nothing constrains its output phase: as the taps adapt, the phase wanders.
A differential decoder recovers each symbol as `s·conj(previous)`, which cancels any
*constant* rotation perfectly — that is the point of
[π/4-DQPSK](/reference/pi-4-dqpsk/) — but a phase that moves *between* the two symbols of a
pair does not cancel; it corrupts every dibit. Measured on real TETRA captures, a
streaming-adaptive CMA ahead of the differential decoder scored **zero** CRC-valid frames.
The obvious retreat — freeze the taps permanently — only ever reproduces the unequalized
baseline, because a frozen filter cannot follow the channel. Adapt-continuously,
apply-frozen takes the third path: within a burst the applied filter is constant (its
rotation cancels differentially, the same property that makes any constant offset
harmless), while between bursts the snapshot refreshes from the tracker. Refresh every
several bursts' worth of symbols; the single symbol pair straddling a refresh is absorbed by
the FEC.

## What else it needs to work

Field-tested supporting rules, each learned from a variant that failed:

- **Normalise by a constant, not a local average.** The CMA update scales with |x|³; an EMA
  that tracks a TDMA downlink's slot-to-slot power swings gives a moving modulus target and
  the equalizer converges to garbage. Use a cumulative-mean (effectively global) normaliser.
- **Guard against divergence.** A normalisation transient or deep fade can blow the
  tracking taps up; re-seed the tracker to pass-through when tap magnitude explodes, so one
  bad patch cannot poison later snapshots.
- **Judge by decoded output, never by [EVM](/reference/error-vector-magnitude/).** Blind
  CMA minimises modulus, not correctness — one unstable variant showed differential EVM
  collapsing 34% → 8% while CRC-valid frames stayed at zero. CRC yield is the verdict.

These and their siblings are collected in
[equalizer gotchas](/reference/equalizer-gotchas/).

## Relevance to SDR

GopherTrunk ships two snapshot equalizers (`internal/dsp/equalizer/`).
**`SnapshotCMA`** — blind, in the TETRA receiver's stream path — roughly *doubled*
CRC-valid voice frames across a set of marginal captures (410 → 778) and lifted a
control-channel fixture from ~12% to ~100% clean BSCH, with no loss on clean signal.
**`SnapshotLMS`** — trained, in the traffic extractor — goes further where framing is
known: it trains on each burst's
[midamble](/reference/tetra-training-sequences/) from the raw (pre-differential) symbols,
freezes, equalizes that one burst, and re-derives the
[soft-decision](/reference/soft-decision/) LLRs from the equalized symbols. The raw-symbol
detail matters: the channel is a clean convolution only in the raw symbol domain —
`s·conj(prev)` is a nonlinear product — so the trained equalizer must run before the
differential decode. The same discipline transfers to any burst waveform with a
differential slicer downstream.

## Sources

[^wiki]: [Blind equalization](https://en.wikipedia.org/wiki/Blind_equalization) — Wikipedia, on blind adaptive equalizers and the phase indeterminacy of constant-modulus cost functions.
