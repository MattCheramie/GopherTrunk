---
slug: ciphertext-only-attack
title: Ciphertext-only attack
entry_type: term
category: algorithms
description: A ciphertext-only attack tries to recover a cipher's key, algorithm, or plaintext using only intercepted ciphertext — the weakest assumption available to an attacker, leaning on redundancy and known structure in the plaintext.
keywords: ciphertext-only attack, COA, cryptanalysis, attack model, known structure, plaintext redundancy, intercepted ciphertext
aka: [ciphertext-only attack, COA]
autolink: true
infobox:
  - { label: Type, value: Cryptanalytic attack model }
  - { label: Attacker has, value: Ciphertext only }
  - { label: Leans on, value: Known plaintext structure / redundancy }
  - { label: Strength, value: Weakest assumption (hardest for attacker) }
see_also: [known-plaintext-attack, chosen-plaintext-attack, brute-force-attack, scrambling, rc4-cipher]
cite_urls:
  - https://en.wikipedia.org/wiki/Ciphertext-only_attack
---

**A ciphertext-only attack (COA)** assumes the attacker has *only* intercepted ciphertext —
no matching plaintext and no ability to query the cipher.[^wiki] It is the weakest set of
assumptions, so a cipher that falls to it is badly broken; the attack works by exploiting
*structure the plaintext is known to have* even when its exact contents are unknown.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 96" role="img" aria-label="Only ciphertext is available; the attacker uses known plaintext structure to constrain the recovery." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="36" width="96" height="26" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="68" y="53" text-anchor="middle" font-size="9" fill="currentColor">ciphertext</text>
  <line x1="116" y1="49" x2="190" y2="49" stroke="currentColor" marker-end="url(#coar)"/>
  <rect x="192" y="30" width="120" height="38" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="252" y="46" text-anchor="middle" font-size="8" fill="currentColor">constrain with known</text><text x="252" y="58" text-anchor="middle" font-size="8" fill="currentColor">plaintext structure</text>
  <line x1="312" y1="49" x2="386" y2="49" stroke="currentColor" marker-end="url(#coar)"/><text x="392" y="53" font-size="9" fill="currentColor">recovery</text>
  <defs><marker id="coar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>With ciphertext alone, the attacker leans on what the plaintext <em>must</em> look like — character set, length rules, or fixed framing.</figcaption>
</figure>

## How it works

The attacker constrains the unknown plaintext using properties it is guaranteed to have:
a restricted character set (e.g. printable ASCII), known field lengths or framing, language
statistics, or a checksum that must validate. Each such constraint rules out candidate keys
or algorithm parameters; with enough intercepted messages, the surviving candidates collapse
to the true one. Classical frequency analysis of a substitution cipher is the textbook
example — only ciphertext is needed because letter frequencies are a known property of the
language.

## Relevance to SDR

Over-the-air interception naturally yields ciphertext-only conditions. In GopherTrunk's
clean-room effort to reverse-engineer the Motorola P25 talker-alias obfuscation
(issue #773), a ciphertext-only angle was available because every even-position byte must
decode to a `0x00` [UTF-16](/reference/bits-and-bytes/) pad and every alias byte must be
printable — constraints derived purely from the captured bytes. It complements the stronger
[known-plaintext](/reference/known-plaintext-attack/) and
[chosen-plaintext](/reference/chosen-plaintext-attack/) settings.

## Sources

[^wiki]: [Ciphertext-only attack](https://en.wikipedia.org/wiki/Ciphertext-only_attack) — Wikipedia, for the attack model and its reliance on known plaintext structure.
