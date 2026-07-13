---
slug: maximal-length-sequence
title: Maximal-length sequence (m-sequence)
entry_type: algorithm
category: spread-spectrum
description: An m-sequence is the longest pseudo-random binary sequence an n-stage LFSR can produce, length 2^n-1, with near-ideal two-valued autocorrelation; the basis of Gold and Kasami codes, scramblers, and channel sounding.
keywords: maximal-length sequence, m-sequence, MLS, PN sequence, pseudo-noise, LFSR, primitive polynomial, autocorrelation, scrambler, channel sounding, spread spectrum
aka: [m-sequence, MLS, maximum-length sequence, PN sequence]
autolink: true
infobox:
  - { label: Type, value: PN sequence from an LFSR }
  - { label: Length, value: "2^n − 1 (max for n stages)" }
  - { label: Basis of, value: Gold/Kasami codes, scramblers }
see_also: [linear-feedback-shift-register, gold-code, barker-code, direct-sequence-spread-spectrum, scrambling, matched-filter]
cite_urls:
  - https://en.wikipedia.org/wiki/Maximum_length_sequence
  - https://en.wikipedia.org/wiki/Linear-feedback_shift_register
---

**A maximal-length sequence (m-sequence)** is the longest non-repeating binary sequence that
an n-stage [linear-feedback shift register](/reference/linear-feedback-shift-register/) can
generate — period **2ⁿ − 1** — and it has a near-ideal **two-valued autocorrelation** that
makes it behave like random noise while being fully deterministic.[^wiki] These
**pseudo-noise (PN)** properties make the m-sequence the foundational building block of
[spread-spectrum](/reference/direct-sequence-spread-spectrum/) codes,
[scramblers](/reference/scrambling/), and channel-measurement waveforms.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A four-stage shift register with an exclusive-OR feedback tap generating a repeating pseudo-random bit sequence of length fifteen." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="mseqar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" fill="none" stroke-width="1.1">
    <rect x="60" y="30" width="40" height="30"/><rect x="120" y="30" width="40" height="30"/><rect x="180" y="30" width="40" height="30"/><rect x="240" y="30" width="40" height="30"/>
    <circle cx="200" cy="105" r="14"/>
  </g>
  <g font-size="11" fill="currentColor" text-anchor="middle"><text x="80" y="50">D1</text><text x="140" y="50">D2</text><text x="200" y="50">D3</text><text x="260" y="50">D4</text><text x="200" y="109">⊕</text></g>
  <g stroke="currentColor" stroke-width="1.1" marker-end="url(#mseqar)">
    <path d="M100 45 H118"/><path d="M160 45 H178"/><path d="M220 45 H238"/>
    <path d="M280 45 H330"/>
    <path d="M280 55 C 300 90 260 105 216 105"/>
    <path d="M140 60 V105 H186"/>
    <path d="M200 91 V64"/>
    <path d="M186 105 C 120 105 40 90 40 45 H58"/>
  </g>
  <text x="345" y="49" text-anchor="middle" font-size="9" fill="currentColor">output</text>
  <text x="200" y="140" text-anchor="middle" font-size="9" fill="currentColor">n = 4 taps (primitive) → period 2⁴−1 = 15</text>
</svg>
<figcaption>An m-sequence comes from an LFSR whose feedback taps form a primitive polynomial; n stages give the maximal period 2ⁿ−1 before it repeats.</figcaption>
</figure>

## How it works

An [LFSR](/reference/linear-feedback-shift-register/) shifts its bits each clock and feeds
back the XOR of certain tap positions into the input. The register has 2ⁿ possible states, but
the all-zeros state is a dead end (it produces only zeros), so the best any n-stage LFSR can
do is cycle through all **2ⁿ − 1** non-zero states before repeating. It achieves that maximum
period precisely when the feedback taps correspond to a **primitive polynomial** over GF(2) —
that is the defining condition for a maximal-length sequence.

Over one full period an m-sequence has three "randomness" properties that make it *look* like
a coin-flip stream:

- **Balance** — the number of 1s exceeds the number of 0s by exactly one.
- **Run distribution** — runs of consecutive equal bits follow the geometric distribution
  expected of random data.
- **Two-valued autocorrelation** — correlated against a shifted copy of itself (bits mapped to
  ±1), it returns the full length N at zero shift and exactly **−1** at *every* other shift.
  That flat, single-valued sidelobe floor is the key property: it lets a
  [matched-filter](/reference/matched-filter/) correlator pinpoint code phase unambiguously,
  and it is what a spread-spectrum receiver exploits to despread its wanted signal.

The weakness is that a single m-sequence is *linear*: observing only 2n consecutive output
bits reveals the taps (via the Berlekamp–Massey algorithm), so on its own it is cryptographically
worthless and, for multi-user systems, different-length shifts of one m-sequence can
cross-correlate badly.

## Variants and derived codes

Because a lone m-sequence has poor *cross*-correlation, richer families are built from it:
XOR-ing two preferred m-sequences yields a [Gold code](/reference/gold-code/) family with
bounded mutual interference (the GPS/CDMA workhorse), and a related construction gives Kasami
codes. Where a strictly ±1 sidelobe floor and very short length are needed instead, the
distinct [Barker codes](/reference/barker-code/) are used.

## Relevance to SDR

M-sequences and their derivatives are everywhere in RF: they seed the
[Gold codes](/reference/gold-code/) of **GPS** and **CDMA**, act as **scrambling/whitening**
sequences in DVB, GSM, and digital voice framing, provide the excitation for **channel
sounding** (their flat autocorrelation makes correlating the received sequence a direct
impulse-response measurement), and serve as pilot/sync patterns.

Within GopherTrunk's land-mobile targets, LFSR-generated PN sequences appear as
[scramblers](/reference/scrambling/) that whiten payload data (for example the pseudo-random
scrambling in some digital voice frames) — the scanner reproduces the same PN generator to
de-scramble before decoding. GopherTrunk does not implement a spread-spectrum despreading
correlator, since its protocols are narrowband, but the m-sequence machinery documented here
is directly relevant to its de-scrambling and sync-detection code.

## Sources

[^wiki]: [Maximum length sequence](https://en.wikipedia.org/wiki/Maximum_length_sequence) — Wikipedia, for the 2ⁿ−1 period, primitive-polynomial condition, and two-valued autocorrelation.
