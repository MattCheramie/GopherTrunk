---
slug: obfuscation
title: Obfuscation
entry_type: term
category: cryptography
description: Obfuscation hides the meaning of data without a secret key, so anyone who learns the method can reverse it — security by obscurity, distinct from true encryption.
keywords: obfuscation, security by obscurity, no key, reversible, scrambling, talker alias, encryption, cipher, Kerckhoffs
aka: [security by obscurity]
autolink: true
infobox:
  - { label: Goal, value: "Hide meaning, not keep a secret" }
  - { label: Key, value: "None" }
  - { label: Reversible by, value: "Anyone who knows the method" }
see_also: [encryption, cryptography, cipher, cryptanalysis]
cite_urls:
  - https://en.wikipedia.org/wiki/Obfuscation
---

**Obfuscation** hides the meaning of data without a secret key, so anyone who learns the
method can reverse it.[^wiki] It is "security by obscurity," and it is fundamentally
different from [encryption](/reference/encryption/): there is no key whose absence keeps the
data safe.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="Data transformed by a known method with no key, reversible by anyone who knows the method." xmlns="http://www.w3.org/2000/svg">
  <rect x="18" y="40" width="60" height="24" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="48" y="55" text-anchor="middle" font-size="8" fill="currentColor">data</text>
  <line x1="78" y1="52" x2="128" y2="52" stroke="currentColor" marker-end="url(#obar)"/>
  <rect x="130" y="38" width="96" height="28" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="178" y="50" text-anchor="middle" font-size="8" fill="currentColor">obfuscate</text><text x="178" y="61" text-anchor="middle" font-size="7" fill="currentColor">(no key)</text>
  <line x1="226" y1="52" x2="276" y2="52" stroke="currentColor" marker-end="url(#obar)"/>
  <text x="284" y="50" font-size="8" fill="currentColor">scrambled —</text>
  <text x="284" y="63" font-size="8" fill="currentColor">reversible by method</text>
  <defs><marker id="obar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Obfuscation has no key: knowing the method is enough to undo it.</figcaption>
</figure>

## How it works

Obfuscation transforms data by a fixed procedure — a permutation, a substitution table, a
shift-register update — with no secret parameter. Because the only thing protecting it is
that the method is undocumented, it fails the moment the method is recovered. This is the
exact situation Kerckhoffs's principle warns against: a system whose security collapses once
the algorithm is known offers no real confidentiality.

Working out an obfuscation scheme is a [cryptanalysis](/reference/cryptanalysis/) exercise,
but a tractable one: with enough observed input/output, the transformation can be
reconstructed because there is no key to search for. Once reconstructed, it is reversible
forever by anyone.

## Relevance to SDR

Obfuscation appears regularly on trunked systems and is easy to mistake for
[encryption](/reference/encryption/). The Motorola [P25](/reference/project-25/)
talker-alias scheme is the leading example: the operator-entered alias text is transformed
by a keyless, reversible method, which makes it *obfuscation*, not encryption. GopherTrunk's
handling of it was developed clean-room in issue #773 — the transformation was reconstructed
solely from publicly observable over-the-air data, never from any third-party source, by
hypothesizing candidate update rules (shift-register and round-function shaped models) and
testing them until a 256-entry substitution table was recovered. Because there is no key,
the alias can be displayed; truly encrypted voice ([RC4](/reference/rc4-cipher/) or AES) is
a different matter entirely and stays opaque. Related keyless transforms include
[scrambling](/reference/scrambling/) whitening, which balances a signal rather than hiding
its meaning.

## Sources

[^wiki]: [Obfuscation](https://en.wikipedia.org/wiki/Obfuscation) — Wikipedia, for obfuscation as hiding meaning without a key and its relation to security by obscurity.
