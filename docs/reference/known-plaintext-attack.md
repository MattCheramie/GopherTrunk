---
slug: known-plaintext-attack
title: Known-plaintext attack
entry_type: term
category: algorithms
description: A known-plaintext attack recovers a cipher's key or algorithm from matched plaintext–ciphertext pairs the attacker happens to possess but did not choose.
keywords: known-plaintext attack, KPA, cryptanalysis, plaintext ciphertext pairs, crib, ground truth, reverse engineering
aka: [known-plaintext attack, KPA]
autolink: true
infobox:
  - { label: Type, value: Cryptanalytic attack model }
  - { label: Attacker has, value: Matched plaintext–ciphertext pairs }
  - { label: Pairs are, value: Observed, not chosen }
  - { label: Classic term, value: "Crib" }
see_also: [ciphertext-only-attack, chosen-plaintext-attack, differential-cryptanalysis, algebraic-attack, rc4-cipher]
cite_urls:
  - https://en.wikipedia.org/wiki/Known-plaintext_attack
---

**A known-plaintext attack (KPA)** gives the attacker a set of matched plaintext–ciphertext
pairs that they *observe but do not choose*, and uses them to recover the key or the
algorithm itself.[^wiki] Such a matched pair is historically called a **crib**. It is
stronger than a [ciphertext-only attack](/reference/ciphertext-only-attack/) and weaker than
a [chosen-plaintext attack](/reference/chosen-plaintext-attack/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="Matched plaintext and ciphertext pairs are fed to a solver that recovers the key or algorithm." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="20" width="120" height="22" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="80" y="35" text-anchor="middle" font-size="8" fill="currentColor">plaintext "TMA 5"</text>
  <rect x="20" y="54" width="120" height="22" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="80" y="69" text-anchor="middle" font-size="8" fill="currentColor">ciphertext 1B18…</text>
  <line x1="140" y1="31" x2="206" y2="44" stroke="currentColor"/><line x1="140" y1="65" x2="206" y2="52" stroke="currentColor"/>
  <rect x="208" y="32" width="96" height="32" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="256" y="51" text-anchor="middle" font-size="8" fill="currentColor">fit / solve</text>
  <line x1="304" y1="48" x2="378" y2="48" stroke="currentColor" marker-end="url(#kpar)"/><text x="384" y="52" font-size="8" fill="currentColor">key / algorithm</text>
  <defs><marker id="kpar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Matched pairs over-determine the cipher: enough cribs pin the key or the per-step transform uniquely.</figcaption>
</figure>

## How it works

Each matched pair is an equation linking the unknown key or transform to observed bytes.
A handful of pairs may under-determine the cipher, but many pairs at varied lengths
over-determine it — fitting any candidate model and rejecting those that fail to round-trip
all pairs. The method underlies most practical reverse engineering of an undocumented
encoder: collect pairs, hypothesize a structure, and keep only parameters consistent with
every pair.

## Relevance to SDR

When a system's text (unit aliases, callsigns) is sometimes broadcast in clear and sometimes
obfuscated, a listener can assemble a known-plaintext corpus. GopherTrunk's clean-room work
on the Motorola P25 talker-alias obfuscation (issue #773) used a 3,607-pair corpus to recover
the cipher's 256-entry substitution table and characterize it as a length-seeded 16-bit state
machine — without reading any third-party source. It pairs naturally with
[differential](/reference/differential-cryptanalysis/) and
[algebraic](/reference/algebraic-attack/) techniques.

## Sources

[^wiki]: [Known-plaintext attack](https://en.wikipedia.org/wiki/Known-plaintext_attack) — Wikipedia, for the attack model and the term "crib."
