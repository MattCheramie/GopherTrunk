---
slug: brute-force-attack
title: Brute-force attack (exhaustive key search)
entry_type: term
category: cryptography
description: A brute-force attack tries every possible key or parameter until one reproduces the observed data — guaranteed to succeed eventually, but only feasible when the search space is small enough.
keywords: brute-force attack, exhaustive key search, parameter search, key space, parallel search, dictionary attack, rainbow table, cryptanalysis
aka: [brute-force attack, exhaustive key search]
autolink: true
infobox:
  - { label: Type, value: Cryptanalytic technique }
  - { label: Method, value: Enumerate every key / parameter }
  - { label: Guaranteed, value: "Yes — if the space is searchable" }
  - { label: Limited by, value: Size of the key / parameter space }
see_also: [known-plaintext-attack, algebraic-attack, sat-smt-solving, differential-cryptanalysis, tetra-tea]
cite_urls:
  - https://en.wikipedia.org/wiki/Brute-force_attack
  - https://en.wikipedia.org/wiki/EFF_DES_cracker
---

**A brute-force attack** (exhaustive key search) simply tries every candidate key or
parameter set until one reproduces the observed ciphertext.[^wiki] It always works *in
principle*; in practice it is bounded by the size of the search space and the cost of testing
each candidate, so it is decisive only when that space is small — or when a cheap early-exit
test rejects most candidates quickly.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 96" role="img" aria-label="Candidate keys are enumerated and each is tested against the data; most are rejected, one matches." xmlns="http://www.w3.org/2000/svg">
  <g font-family="monospace" font-size="9" fill="currentColor"><text x="20" y="34">k=0</text><text x="20" y="50">k=1</text><text x="20" y="66">k=2 …</text></g>
  <line x1="70" y1="48" x2="150" y2="48" stroke="currentColor" marker-end="url(#bfar)"/>
  <rect x="152" y="32" width="96" height="32" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="200" y="51" text-anchor="middle" font-size="8" fill="currentColor">test vs data</text>
  <line x1="248" y1="48" x2="320" y2="34" stroke="currentColor" marker-end="url(#bfar)"/><text x="326" y="38" font-size="8" fill="currentColor">reject (most)</text>
  <line x1="248" y1="48" x2="320" y2="64" stroke="currentColor" marker-end="url(#bfar)"/><text x="326" y="68" font-size="8" fill="currentColor">match (one)</text>
  <defs><marker id="bfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Enumerate, test, reject. A fast rejection test on a small sample lets a huge space be swept before the rare survivor is fully verified.</figcaption>
</figure>

## How it works

The attacker fixes a candidate structure with a few unknown constants and loops over all
their values, simulating the cipher and comparing against the data. Two practices make large
sweeps tractable: a **cheap early-exit** — reject a candidate after a few mismatched bytes on
a small sample, before running the full corpus — and **parallelism** across cores. When the
unknowns number in the millions it is feasible; when they number a full 256-entry table it is
not, and an [algebraic](/reference/algebraic-attack/) or
[SAT/SMT](/reference/sat-smt-solving/) approach is needed instead.

## Variants

Pure enumeration is the baseline; several refinements cut the cost when the space is too large
for a naive sweep. A **dictionary attack** tries only likely values (common passwords, default
keys) instead of the whole space. A **rainbow table** trades memory for time by precomputing
chains of hash outputs so a later lookup is cheap — devastating against unsalted password
hashes. A **meet-in-the-middle** attack halves the exponent for constructions that apply a
cipher twice (the reason 2-key Triple-DES gives far less than 112 bits of real strength). Each
exploits some structure to shrink the effective 2ⁿ into something searchable.

## In practice

Feasibility is a moving target set by hardware and key length. The 56-bit key of DES was
searchable in days by the EFF's purpose-built "Deep Crack" machine in 1998, demonstrating that
a once-standard cipher had fallen to brute force outright.[^eff] A 32-bit effective key — the
size the [TETRA TEA1](/reference/tetra-tea/) algorithm was reduced to — is trivially
searchable on a laptop. This is exactly why key length matters: every added bit doubles the
work, so 128-bit and larger keys keep the exhaustive search permanently out of reach while a
short or deliberately weakened key collapses.

## Relevance to SDR

Reverse-engineering an undocumented encoder often reduces to a few unknown constants — a
multiplier, an additive step, a seed. GopherTrunk's clean-room analysis of the Motorola P25
talker-alias obfuscation (issue #773) used parallel brute-force sweeps over such constants to
rule out whole families of update rules (linear congruential, multiplicative-mod-prime,
multiply-with-carry) against a [known-plaintext](/reference/known-plaintext-attack/) corpus,
each with a fast early-exit on the longest messages.

## Sources

[^wiki]: [Brute-force attack](https://en.wikipedia.org/wiki/Brute-force_attack) — Wikipedia, for exhaustive key search and its dependence on key-space size.
[^eff]: [EFF DES cracker](https://en.wikipedia.org/wiki/EFF_DES_cracker) — Wikipedia, for a concrete demonstration that a 56-bit key is brute-forceable in practice.
