---
slug: cryptanalysis
title: Cryptanalysis
entry_type: term
category: cryptography
description: Cryptanalysis is the study of analyzing and breaking cryptosystems — recovering plaintext, keys, or algorithm structure without being given the key — and is the analytic counterpart to cryptography.
keywords: cryptanalysis, codebreaking, attack, ciphertext-only, known-plaintext, chosen-plaintext, brute force, Kerckhoffs, key recovery, differential, linear, algebraic
aka: [codebreaking]
autolink: true
infobox:
  - { label: Field, value: "Information security" }
  - { label: Goal, value: "Recover plaintext, key, or structure" }
  - { label: Assumption, value: "Algorithm public (Kerckhoffs)" }
see_also: [cryptography, cipher, encryption, obfuscation, ciphertext-only-attack, known-plaintext-attack, chosen-plaintext-attack, brute-force-attack, differential-cryptanalysis, frequency-analysis, algebraic-attack]
cite_urls:
  - https://en.wikipedia.org/wiki/Cryptanalysis
  - https://en.wikipedia.org/wiki/Kerckhoffs%27s_principle
---

**Cryptanalysis** is the study of analyzing and breaking cryptosystems — recovering the
plaintext, the key, or the internal structure of an algorithm without being handed the
key.[^wiki] It is the analytic counterpart to [cryptography](/reference/cryptography/): one
side builds, the other probes, and progress on each drives the other.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="Ciphertext analyzed to recover plaintext or key." xmlns="http://www.w3.org/2000/svg">
  <rect x="24" y="40" width="74" height="26" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="61" y="57" text-anchor="middle" font-size="8" fill="currentColor">ciphertext</text>
  <line x1="98" y1="53" x2="150" y2="53" stroke="currentColor" marker-end="url(#caar)"/>
  <rect x="152" y="40" width="92" height="26" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="198" y="57" text-anchor="middle" font-size="8" fill="currentColor">cryptanalysis</text>
  <line x1="244" y1="53" x2="296" y2="53" stroke="currentColor" marker-end="url(#caar)"/>
  <text x="302" y="50" font-size="8" fill="currentColor">recovered key</text>
  <text x="302" y="64" font-size="8" fill="currentColor">or plaintext</text>
  <defs><marker id="caar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Cryptanalysis works from intercepted data toward the secret the cipher was meant to hide.</figcaption>
</figure>

## How it works

Cryptanalysts almost always assume [Kerckhoffs's premise](/reference/kerckhoffs-principle/)
— that the algorithm is known and only the key is secret — so the work is to exploit any
structure the [cipher](/reference/cipher/) leaks. Classic attacks are named by what the
analyst can obtain:

- **[Ciphertext-only](/reference/ciphertext-only-attack/)** — only intercepted ciphertext
  is available, as in classic [frequency analysis](/reference/frequency-analysis/) of a
  substitution cipher.
- **[Known-plaintext](/reference/known-plaintext-attack/)** — some plaintext/ciphertext
  pairs are known.
- **[Chosen-plaintext](/reference/chosen-plaintext-attack/)** — the analyst can encrypt
  inputs of their choosing.
- **[Brute force](/reference/brute-force-attack/)** — exhaustively trying keys, bounded by
  the key space.

Deeper techniques attack a cipher's mathematics directly —
[differential](/reference/differential-cryptanalysis/) and linear analysis,
[algebraic modeling](/reference/algebraic-attack/), and solver-driven approaches that treat
recovery as a constraint problem. A scheme that relies on the method staying secret is not
really analyzed but merely reverse-engineered; that is
[obfuscation](/reference/obfuscation/), not [encryption](/reference/encryption/).

## Variants

The strength of an attack is judged by two things: the **resources** it needs (data,
chosen queries, memory) and the **work factor** — how much faster than brute force it runs.
A "break" in the academic sense may still be wholly impractical (e.g. requiring 2¹²⁶
operations against a 128-bit key), yet it signals structural weakness. Attacks also target
the *implementation* rather than the algorithm: side-channel analysis reads timing, power,
or electromagnetic emissions, and protocol-level flaws (key reuse, predictable IVs, weak
random number generators) are where real systems most often fall.

## In practice

Historically, cryptanalysis decided wars — the breaking of the Enigma and Lorenz ciphers is
the canonical example, and it is where systematic codebreaking, statistical attacks, and
early computing converged. Today the same discipline vets civilian standards: a cipher like
[AES](/reference/advanced-encryption-standard/) earns trust precisely because decades of
public cryptanalysis have failed to dent it, while weaker or aging designs
([DES](/reference/data-encryption-standard/), [RC4](/reference/rc4-cipher/)) are retired as
attacks accumulate.

## Relevance to SDR

Cryptanalysis in the radio world is mostly about *understanding formats*, not breaking
strong ciphers. Recovering an undocumented on-air transformation — framing, scrambling, or
an [obfuscation](/reference/obfuscation/) layer — is a cryptanalytic exercise carried out
entirely from public observation. GopherTrunk's clean-room work on the Motorola P25
talker-alias scheme (issue #773) is a worked example: candidate models such as
shift-register and round-function update rules were hypothesized and tested against
captured data until the actual substitution table emerged, all without reference to any
third-party source code. Properly encrypted voice — [P25](/reference/project-25/)
[AES](/reference/advanced-encryption-standard/)-256, DMR
[RC4](/reference/rc4-cipher/), or [TETRA TEA](/reference/tetra-tea/) — is not in scope: the
math is sound and the key is absent.

## Sources

[^wiki]: [Cryptanalysis](https://en.wikipedia.org/wiki/Cryptanalysis) — Wikipedia, for attack models and the assumption that the algorithm is public.
[^kerck]: [Kerckhoffs's principle](https://en.wikipedia.org/wiki/Kerckhoffs%27s_principle) — Wikipedia, for the assumption underlying cryptanalysis that the algorithm is known and only the key is secret.
