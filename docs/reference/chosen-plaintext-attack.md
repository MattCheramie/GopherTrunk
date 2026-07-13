---
slug: chosen-plaintext-attack
title: Chosen-plaintext attack
entry_type: term
category: cryptography
description: A chosen-plaintext attack lets the attacker pick the plaintexts and observe the resulting ciphertexts, using controlled inputs to expose a cipher's internal structure one variable at a time.
keywords: chosen-plaintext attack, CPA, cryptanalysis, controlled inputs, adaptive, IND-CPA, differential, structure probing
aka: [chosen-plaintext attack, CPA]
autolink: true
infobox:
  - { label: Type, value: Cryptanalytic attack model }
  - { label: Attacker has, value: Plaintexts of their choosing + resulting ciphertexts }
  - { label: Power, value: Controlled / adaptive probing }
  - { label: Strength, value: Stronger than known-plaintext }
see_also: [known-plaintext-attack, ciphertext-only-attack, differential-cryptanalysis, brute-force-attack, tetra-tea]
cite_urls:
  - https://en.wikipedia.org/wiki/Chosen-plaintext_attack
  - https://en.wikipedia.org/wiki/Ciphertext_indistinguishability
---

**A chosen-plaintext attack (CPA)** gives the attacker the power to *choose* the plaintexts
and observe the matching ciphertexts.[^wiki] Controlled inputs are far more informative than
passively observed ones: by varying a single character, length, or bit at a time, the
attacker drives the cipher's internal state deliberately and reads its response — making it
strictly stronger than a [known-plaintext attack](/reference/known-plaintext-attack/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="The attacker chooses plaintexts that differ in one position and compares the resulting ciphertexts." xmlns="http://www.w3.org/2000/svg">
  <g font-family="monospace" font-size="9" fill="currentColor"><text x="24" y="34">AAAAA</text><text x="24" y="58">AAAAB</text></g>
  <line x1="78" y1="30" x2="150" y2="40" stroke="currentColor" marker-end="url(#cpar)"/><line x1="78" y1="54" x2="150" y2="56" stroke="currentColor" marker-end="url(#cpar)"/>
  <rect x="152" y="28" width="92" height="40" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="198" y="52" text-anchor="middle" font-size="8" fill="currentColor">cipher</text>
  <line x1="244" y1="40" x2="300" y2="34" stroke="currentColor" marker-end="url(#cpar)"/><line x1="244" y1="56" x2="300" y2="62" stroke="currentColor" marker-end="url(#cpar)"/>
  <g font-family="monospace" font-size="9" fill="currentColor"><text x="304" y="38">…7A</text><text x="304" y="66">…D2</text></g>
  <text x="360" y="52" font-size="8" fill="currentColor">compare</text>
  <defs><marker id="cpar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Choosing inputs that differ in one place isolates how that change propagates — the densest possible view of the cipher's state machine.</figcaption>
</figure>

## How it works

The attacker designs input sets that hold most of the state fixed and vary one factor:
a *length sweep* exposes how the cipher is seeded; a *single-character sweep* exposes the
per-position transform; *single-bit differences* feed directly into
[differential cryptanalysis](/reference/differential-cryptanalysis/). Because the inputs are
chosen, the attacker can systematically cover the state×input combinations a passive corpus
never reaches, often pinning an internal table outright.

## Variants

Two sub-flavours differ in when the choices are made. In a **batch** (non-adaptive) attack the
attacker fixes all plaintexts up front. In an **adaptive** chosen-plaintext attack (CPA2) each
new query depends on the answers so far, so the attacker can zoom in — probe, look at the
ciphertext, then design the next probe to resolve the remaining ambiguity. Adaptivity is
what makes CPA the natural setting for interactive reverse engineering. The model also sets
the modern security bar: a cipher is called **IND-CPA secure** when an adversary who may
encrypt any plaintexts of their choosing still cannot distinguish which of two messages was
encrypted — a property that requires randomised or nonce-based encryption, since a
deterministic cipher always leaks when two identical plaintexts recur.[^indcpa]

## In practice

Mounting a CPA requires the ability to *inject* plaintext into the target, which is often the
practical obstacle. Against a stored-data cipher this may be trivial; against a live radio
system it means transmitting chosen content — feasible only where you are licensed and
authorized to key up. When available it is decisive, because it converts a statistics problem
(hope the right pair appears) into an experiment (make the right pair appear).

## Relevance to SDR

For an over-the-air obfuscation, a chosen-plaintext attack means programming a transmitter
with selected text and capturing the result on an SDR — only on systems and spectrum you are
licensed or authorized to key up. It is the decisive lever where passive
[known-plaintext](/reference/known-plaintext-attack/) data stalls: GopherTrunk's analysis of
the Motorola P25 talker-alias obfuscation (issue #773) found the per-character update is
sparsely covered by real callsigns, so a short controlled sweep would supply exactly the
dense coverage needed to finish it.

## Sources

[^wiki]: [Chosen-plaintext attack](https://en.wikipedia.org/wiki/Chosen-plaintext_attack) — Wikipedia, for the attack model and its advantage over passively observed plaintext.
[^indcpa]: [Ciphertext indistinguishability](https://en.wikipedia.org/wiki/Ciphertext_indistinguishability) — Wikipedia, for IND-CPA as the security notion defined against this attack model.
