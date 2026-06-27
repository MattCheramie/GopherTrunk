---
slug: frequency-analysis
title: Frequency analysis
entry_type: term
category: cryptography
description: Frequency analysis is a classic ciphertext-only cryptanalytic technique that breaks substitution ciphers by exploiting the uneven frequencies of letters and symbols in the underlying plaintext language.
keywords: frequency analysis, cryptanalysis, substitution cipher, ciphertext-only attack, letter frequency, statistical attack, monoalphabetic, classical cipher, symbol distribution
aka: [letter-frequency analysis]
autolink: true
infobox:
  - { label: Type, value: Cryptanalytic technique }
  - { label: Attack model, value: Ciphertext-only }
  - { label: Exploits, value: Non-uniform symbol frequencies }
see_also: [cryptanalysis, s-box, one-time-pad, advanced-encryption-standard, hash-function]
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency_analysis
---

**Frequency analysis** breaks a substitution cipher by exploiting the fact that letters and
symbols of a natural language occur with characteristic, uneven frequencies that survive a
simple letter-for-letter substitution.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="Bars of differing height showing that some ciphertext symbols occur far more often than others." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <line x1="40" y1="80" x2="430" y2="80" stroke="currentColor" stroke-width="1.2"/>
    <rect x="60" y="30" width="22" height="50" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/><text x="71" y="92">Q</text>
    <rect x="100" y="58" width="22" height="22" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/><text x="111" y="92">X</text>
    <rect x="140" y="44" width="22" height="36" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/><text x="151" y="92">M</text>
    <rect x="180" y="22" width="22" height="58" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/><text x="191" y="92">K</text>
    <rect x="220" y="64" width="22" height="16" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/><text x="231" y="92">B</text>
    <rect x="260" y="50" width="22" height="30" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/><text x="271" y="92">V</text>
    <text x="370" y="44" font-size="9" text-anchor="start">tallest symbol</text>
    <text x="370" y="56" font-size="9" text-anchor="start">≈ most common</text>
    <text x="370" y="68" font-size="9" text-anchor="start">plaintext letter</text>
  </g>
</svg>
<figcaption>Ciphertext symbol counts mirror the plaintext language's letter frequencies, leaking the substitution.</figcaption>
</figure>

## How it works

In a monoalphabetic substitution cipher each plaintext letter is consistently replaced by
one ciphertext symbol. The substitution hides *which* symbol stands for which letter, but it
does not change *how often* each appears: if the most common ciphertext symbol shows up about
as often as the most common letter does in the plaintext language, those two probably match.
The analyst tabulates symbol frequencies, aligns them with known language statistics, and
confirms guesses using common pairs, doublets, and short words. Because it needs only
intercepted ciphertext, it is the archetypal
[ciphertext-only](/reference/cryptanalysis/) attack.

Modern ciphers are built specifically to defeat this. Strong *diffusion* and *confusion*
spread each plaintext symbol's influence across the whole output, so a fixed
[S-box](/reference/s-box/) used inside many rounds — rather than as a single standalone
substitution — flattens the output statistics. A [one-time pad](/reference/one-time-pad/)
defeats frequency analysis completely: its output is uniform by construction.

## Relevance to SDR

Frequency analysis does not apply to the encrypted voice GopherTrunk encounters — keyed
[AES](/reference/advanced-encryption-standard/) and DES produce statistically flat output by
design. Its relevance is to *obfuscation*, not encryption. When reverse-engineering an
unknown, keyless transform clean-room (as in the talker-alias work, issue #773), counting how
often each byte value appears is one of the first diagnostics: a non-flat distribution betrays
a simple substitution and helps recover a fixed lookup table, whereas a flat distribution
suggests a stronger construction. The technique is a tool for understanding *unkeyed*
encodings, not for breaking real encryption.

## Sources

[^wiki]: [Frequency analysis](https://en.wikipedia.org/wiki/Frequency_analysis) — Wikipedia, for the ciphertext-only attack on substitution ciphers and the use of language letter statistics.
