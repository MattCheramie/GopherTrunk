---
slug: richard-hamming
title: Richard Hamming
entry_type: person
category: people
description: Richard Hamming (1915–1998) was an American mathematician who created the first practical error-correcting codes, the Hamming codes, founding the field of coding theory.
keywords: Richard Hamming, Hamming code, error correction, coding theory, Bell Labs, Hamming distance, parity
aka: [Richard Hamming]
autolink: true
infobox:
  - { label: Lived, value: "1915–1998" }
  - { label: Field, value: Mathematics }
  - { label: Known for, value: Hamming codes, Hamming distance }
see_also: [hamming-code, forward-error-correction, golay-code, claude-shannon, robert-gallager, cyclic-redundancy-check]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Richard_Hamming
  - https://amturing.acm.org/award_winners/hamming_1000652.cfm
---

**Richard Hamming** (1915–1998) was an American mathematician who created the first
practical **error-correcting codes** — the [Hamming codes](/reference/hamming-code/) —
launching the field of coding theory on which all reliable digital radio, storage, and
networking now depends.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="Data bits with interspersed parity bits that detect and correct a single error." xmlns="http://www.w3.org/2000/svg">
  <g font-family="monospace" font-size="11" fill="currentColor" text-anchor="middle"><text x="60" y="50">P</text><text x="100" y="50">P</text><text x="140" y="50">D</text><text x="180" y="50">P</text><text x="220" y="50">D</text><text x="260" y="50">D</text><text x="300" y="50">D</text></g>
  <path d="M60 60 q40 16 80 0" fill="none" stroke="currentColor" stroke-opacity="0.6"/><path d="M100 64 q60 20 120 0" fill="none" stroke="currentColor" stroke-opacity="0.6"/>
  <text x="230" y="96" text-anchor="middle" font-size="9" fill="currentColor">parity bits locate and fix a single-bit error</text>
</svg>
<figcaption>Hamming created the first practical error-correcting codes, the ancestors of the FEC used in digital radio.</figcaption>
</figure>

## Life and work

Richard Wesley Hamming was born in Chicago in 1915, took a bachelor's degree from the
University of Chicago, a master's from Nebraska, and a PhD in mathematics from the University
of Illinois in 1942. During the Second World War he was recruited to the Manhattan Project at
Los Alamos, where he ran the mechanical calculating machines that carried out the physicists'
computations — an early, formative encounter with computing as an industrial process. In 1946
he joined Bell Telephone Laboratories, then at the height of its influence, sharing an
environment with [Claude Shannon](/reference/claude-shannon/), John Tukey, and others, and he
stayed for thirty years. After retiring from Bell Labs in 1976 he taught at the Naval
Postgraduate School in Monterey, where his lectures on "The Art of Doing Science and
Engineering" and his talk "You and Your Research" became famous meditations on how to do work
that matters.

The story behind his most famous invention is well known. Bell Labs' relay computers ran
batch jobs over the weekend, and when the machine detected a parity error it simply halted
and moved on, wasting the whole run. Hamming, frustrated at arriving on Monday to find his
job aborted, reasoned that if the machine could *detect* an error it ought to be able to
*locate and correct* it, and over 1947–1950 he worked out how.

## Contribution

Hamming's construction arranges several parity checks so that each data bit is covered by a
distinct combination of them. When an error occurs, the pattern of failed checks — read as a
binary number, the "syndrome" — points directly to the position of the flipped bit, which the
decoder simply inverts. The classic Hamming(7,4) code protects four data bits with three
parity bits and corrects any single-bit error in the seven; adding one more overall parity
bit lets it also *detect* (without correcting) any double error. Underlying all of this is
the concept he formalised, the **Hamming distance** — the number of bit positions in which
two codewords differ. A code that keeps all its valid codewords at least distance three apart
can always correct one error, because any single flip still leaves the received word closest
to the original.[^acm] That geometric way of thinking about codes as points spread out in a
space became the organising idea of the whole discipline, generalised soon after in the
richer [Golay code](/reference/golay-code/) and later in the algebraic codes that dominate
modern systems.

Hamming's 1950 paper appeared just two years after Shannon's, and the two are complementary:
Shannon proved good codes must exist, while Hamming actually built one and gave engineers the
tools to reason about it.

## Legacy

Hamming received the ACM Turing Award in 1968 and the IEEE Hamming Medal, established in his
honour, is awarded annually for contributions to information sciences and systems. His single
error-correcting codes are still used directly in computer memory (SECDED ECC) and in many
communication headers, and the distance concept underpins every block code that followed,
from BCH and Reed–Solomon to the modern LDPC codes associated with
[Robert Gallager](/reference/robert-gallager/). In land-mobile radio the lineage is very
concrete: P25 uses Hamming and Golay codes to protect its control and signalling words, so a
decoder such as GopherTrunk is, at the bit level, running exactly the kind of syndrome-based
[forward error correction](/reference/forward-error-correction/) Hamming invented to keep his
weekend computer runs from dying on a single flipped bit.

## Sources

[^wiki]: [Richard Hamming](https://en.wikipedia.org/wiki/Richard_Hamming) — Wikipedia, for biography and his creation of the first practical error-correcting codes.
[^acm]: [Richard W. Hamming — A.M. Turing Award](https://amturing.acm.org/award_winners/hamming_1000652.cfm) — ACM, for the Hamming codes, Hamming distance, and his 1968 Turing Award.
