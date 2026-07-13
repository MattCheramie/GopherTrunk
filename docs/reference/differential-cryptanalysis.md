---
slug: differential-cryptanalysis
title: Differential cryptanalysis
entry_type: term
category: cryptography
description: Differential cryptanalysis studies how a controlled difference between two inputs propagates to a difference between their outputs, using the bias in that propagation to recover a cipher's structure or key.
keywords: differential cryptanalysis, input difference, output difference, differential characteristic, Biham Shamir, S-box, linear cryptanalysis, minimal pairs, propagation
aka: [differential cryptanalysis]
autolink: true
infobox:
  - { label: Type, value: Cryptanalytic technique }
  - { label: Studies, value: Input difference → output difference }
  - { label: Needs, value: Pairs differing in one place }
  - { label: Introduced by, value: "Biham & Shamir (publicly, 1990)" }
see_also: [chosen-plaintext-attack, known-plaintext-attack, algebraic-attack, brute-force-attack, tetra-tea]
cite_urls:
  - https://en.wikipedia.org/wiki/Differential_cryptanalysis
  - https://en.wikipedia.org/wiki/Linear_cryptanalysis
---

**Differential cryptanalysis** examines how a fixed *difference* between two plaintexts
propagates through a cipher to a difference between their ciphertexts.[^wiki] Because the
absolute values cancel, the propagation isolates the cipher's nonlinear behavior; non-uniform
("biased") propagation leaks structure and, ultimately, key bits. It was introduced publicly
by Eli Biham and Adi Shamir around 1990.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 104" role="img" aria-label="Two inputs differing by delta-in pass through the cipher and emerge differing by delta-out." xmlns="http://www.w3.org/2000/svg">
  <text x="24" y="36" font-size="9" fill="currentColor">P</text><text x="24" y="66" font-size="9" fill="currentColor">P'</text>
  <text x="40" y="51" font-size="8" fill="currentColor">Δin</text><line x1="30" y1="40" x2="30" y2="60" stroke="currentColor" stroke-dasharray="3 2"/>
  <line x1="50" y1="32" x2="150" y2="46" stroke="currentColor"/><line x1="50" y1="62" x2="150" y2="50" stroke="currentColor"/>
  <rect x="152" y="30" width="96" height="38" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="200" y="53" text-anchor="middle" font-size="8" fill="currentColor">cipher</text>
  <line x1="248" y1="44" x2="348" y2="34" stroke="currentColor"/><line x1="248" y1="52" x2="348" y2="64" stroke="currentColor"/>
  <text x="352" y="38" font-size="9" fill="currentColor">C</text><text x="352" y="68" font-size="9" fill="currentColor">C'</text><text x="368" y="53" font-size="8" fill="currentColor">Δout</text>
  <defs><marker id="dcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Holding everything fixed but one difference makes the output difference depend only on how the cipher treats that change.</figcaption>
</figure>

## How it works

The analyst collects pairs that differ in exactly one position — adjacent lengths, a single
changed character, or a one-bit flip — and tabulates the resulting output differences. If a
particular input difference maps to a particular output difference far more often than chance,
that "differential characteristic" constrains the internal transform and can be propagated to
recover round keys or table entries. The core object is the S-box's *difference distribution
table*: for each input difference it counts how the output differences spread, and any entry
much larger than the average is a foothold. It is most powerful in the
[chosen-plaintext](/reference/chosen-plaintext-attack/) setting, where the analyst can
manufacture the exact pairs needed, but useful minimal pairs sometimes occur in a passive
[known-plaintext](/reference/known-plaintext-attack/) corpus.

## Variants

The closely related **linear cryptanalysis** (Matsui, 1993) works with linear approximations —
probabilistic XOR relations among plaintext, ciphertext, and key bits — rather than
differences, and the two are the classic pair of statistical attacks on block ciphers.[^lin]
Later generalisations include *truncated differentials* (tracking only part of the difference),
*higher-order differentials* (differences of differences), and *impossible differentials*
(exploiting a difference that can *never* occur to rule out keys). All share the same premise:
a well-designed cipher should make every output difference equally likely, so any measurable
bias is a defect.

## In practice

Differential cryptanalysis reshaped cipher design. It later emerged that IBM and the NSA had
tuned the DES S-boxes in the 1970s specifically to resist it — years before Biham and Shamir
published the technique openly — which is why DES held up far better against it than a random
S-box choice would have. Modern designs now quote their maximum differential probability as an
explicit security margin, and choosing S-boxes with a flat difference distribution table is a
standard part of the design process. A cipher that skips this analysis, or hides its S-boxes
rather than publishing them, may carry an exploitable differential that review would have
caught.

## Relevance to SDR

Naturally occurring minimal pairs help reverse-engineer an obfuscation. In GopherTrunk's
clean-room analysis of the Motorola P25 talker-alias scheme (issue #773), same-length aliases
that shared a prefix but differed later showed that one character change perturbs all
downstream ciphertext — direct evidence the encoder carries feedback state — and the same
alias seen under two radio IDs, differing only in a trailing
[CRC](/reference/cyclic-redundancy-check/), exposed the framing.

## Sources

[^wiki]: [Differential cryptanalysis](https://en.wikipedia.org/wiki/Differential_cryptanalysis) — Wikipedia, for input/output differences and the Biham–Shamir origin.
[^lin]: [Linear cryptanalysis](https://en.wikipedia.org/wiki/Linear_cryptanalysis) — Wikipedia, for the complementary statistical attack using linear approximations.
