---
slug: tetra-training-sequences
title: TETRA training sequences
entry_type: term
category: synchronization
description: TETRA training sequences are the fixed bit patterns — Normal (22 bits), Extended (30 bits), and Synchronisation (38 bits) — embedded in every burst that a receiver correlates against to find the slot boundary and de-rotate the π/4-DQPSK constellation.
keywords: TETRA training sequence, normal training sequence, synchronisation training sequence, extended training sequence, TETRA sync word, bit to dibit, Gray mapping, EN 300 392-2 9.4.4.3
aka: [TETRA sync word, "normal training sequence", "synchronisation training sequence"]
autolink: true
infobox:
  - { label: Normal, value: "22 bits / 11 dibits" }
  - { label: Extended, value: "30 bits / 15 dibits" }
  - { label: Sync, value: "38 bits / 19 dibits" }
  - { label: Spec, value: EN 300 392-2 §9.4.4.3 }
see_also: [tetra-burst-formats, tetra-receiver-chain, correlate-access-code, frame-synchronization, differential-decoding, tetra-scrambler, pi-4-dqpsk, tetra, tetra-logical-channels, tetra-dmo]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://en.wikipedia.org/wiki/Gray_code
---

**TETRA training sequences** are the fixed bit patterns embedded at known positions in every
[TETRA](/reference/tetra/) burst that let a receiver find the slot boundary and resolve the constellation
rotation.[^tetra] They play the role a frame sync word plays elsewhere: a decoder slides the known pattern
over the demodulated symbol stream and declares a burst wherever the mismatch count falls under a
threshold — exactly [access-code correlation](/reference/correlate-access-code/). TETRA defines three, by
length: the **Normal** training sequence (22 bits / 11 [dibits](/reference/dibit/)), the **Extended**
training sequence (30 bits / 15 dibits), and the longer **Synchronisation** training sequence (38 bits /
19 dibits) carried by the synchronisation burst.[^gray]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 120" role="img" aria-label="Three horizontal bars of increasing length representing the 22-bit normal, 30-bit extended, and 38-bit synchronisation training sequences, each labelled with its bit and dibit count." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="26" width="150" height="20" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="180" y="40" font-size="8.5" fill="currentColor">Normal · 22 bits / 11 dibits</text>
  <rect x="20" y="56" width="205" height="20" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="235" y="70" font-size="8.5" fill="currentColor">Extended · 30 bits / 15 dibits</text>
  <rect x="20" y="86" width="260" height="20" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="290" y="100" font-size="8.5" fill="currentColor">Synchronisation · 38 bits / 19 dibits</text>
</svg>
<figcaption>The three fixed training sequences differ only in length; the longer synchronisation pattern gives the cold-acquisition synchronisation burst a sharper, more reliable correlation peak.</figcaption>
</figure>

## The fixed patterns

The training sequences are specified as literal on-air bit arrays, MSB-first. GopherTrunk stores the real
over-the-air values — validated against live captures — rather than any packed constant. The synchronisation
training sequence, the longest and the one a receiver hunts first, is:

```go
// SyncTrainingSeq — synchronisation training sequence (§9.4.4.3.4), carried by
// the synchronisation downlink burst (SB) at slot 1 of frame 18 per multiframe.
// internal/radio/tetra/sync.go, ETSI EN 300 392-2 §9.4.4.3.4.
var SyncTrainingSeq = []uint8{1, 1, 0, 0, 0, 0, 0, 1, 1, 0, 0, 1, 1, 1, 0, 0, 1, 1, 1,
    0, 1, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 1, 1, 0, 0, 1, 1, 1}
```

The two Normal training sequences (NTS1 and NTS2) and the Extended sequence are stored the same way. An
earlier GopherTrunk implementation used packed `uint64` "hex" constants that were both truncated and matched
no spec value, so the control channel could never lock on real air — a reminder that a training sequence must
be the *exact* on-air pattern, not an approximation.

## The bit-to-dibit convention

TETRA is π/4-DQPSK, so the demodulator does not emit bits — it emits a **dibit value 0..3** per symbol,
derived from the phase step between successive symbols. The mapping from a transmitted bit pair `(b1, b2)`
to a dibit value is TETRA's own [Gray mapping](/reference/gray-code/): `(b1<<1) | (b1^b2)`, giving
`00→0, 01→1, 11→2, 10→3`. This is **deliberately distinct** from the linear dibit convention the C4FM family
(P25, DMR, NXDN, dPMR) uses, and mixing them silently mis-decodes everything. GopherTrunk therefore keeps
`TetraBitsToDibits` / `TetraDibitsToBits` as the single source of truth: the correlator converts each
training sequence to its dibit form once, then slides that dibit pattern over the receiver's dibit output.
Because π/4-DQPSK leaves a constant residual rotation (from carrier offset), the detector correlates against
all four rotations of the pattern and records which one matched, so the rest of the burst can be de-rotated
before channel decoding.

## Relevance to SDR

`internal/radio/tetra/sync.go` holds the four sequences as MSB-first bit arrays, the bit↔dibit Gray helpers,
and a `SyncDetector` that reports each position where a pattern matches within tolerance. The
[traffic extractor](/reference/tetra-burst-formats/) and the control-channel state machine both build their
detectors from these — normal sequences at tolerance 2, the synchronisation sequence at tolerance 3 — under
all four rotations. Because the training sequence is the *only* part of a burst that is not scrambled, it
still correlates even when a downstream bug leaves every scrambled block undecodable, which is why lock
succeeding while nothing decodes is a classic TETRA failure signature.

## Sources

[^tetra]: [Terrestrial Trunked Radio](https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio) — Wikipedia, on the TETRA air interface and its burst structure.
[^gray]: [Gray code](https://en.wikipedia.org/wiki/Gray_code) — Wikipedia, on the reflected-binary mapping TETRA uses between symbol bit pairs and dibit values.
