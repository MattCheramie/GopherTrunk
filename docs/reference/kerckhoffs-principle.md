---
slug: kerckhoffs-principle
title: Kerckhoffs's principle
entry_type: term
category: cryptography
description: Kerckhoffs's principle holds that a cryptosystem should remain secure even if everything about it except the key is public knowledge; it is the standard argument against security through obscurity.
keywords: Kerckhoffs principle, security through obscurity, open design, secret key, public algorithm, cryptography, Shannon maxim, obfuscation, key secrecy
aka: [Kerckhoffs's law, Kerckhoffs's desideratum]
autolink: true
infobox:
  - { label: Claim, value: Only the key need be secret }
  - { label: Targets, value: Security through obscurity }
  - { label: Restated by, value: "Shannon's maxim: the enemy knows the system" }
see_also: [cryptographic-key, symmetric-key-cryptography, public-key-cryptography]
cite_urls:
  - https://en.wikipedia.org/wiki/Kerckhoffs%27s_principle
---

**Kerckhoffs's principle** states that a cryptosystem should stay secure even if everything
about it except the [key](/reference/cryptographic-key/) is publicly known.[^wiki] In other
words, the secrecy must live entirely in the key, never in the design of the algorithm.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A public algorithm box with a separate secret key; only the key is hidden." xmlns="http://www.w3.org/2000/svg">
  <rect x="60" y="34" width="160" height="40" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="140" y="50" text-anchor="middle" font-size="9" fill="currentColor">algorithm</text><text x="140" y="64" text-anchor="middle" font-size="8" fill="currentColor">(public)</text>
  <rect x="280" y="34" width="120" height="40" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="340" y="50" text-anchor="middle" font-size="9" fill="currentColor">key</text><text x="340" y="64" text-anchor="middle" font-size="8" fill="currentColor">(secret)</text>
  <line x1="220" y1="54" x2="280" y2="54" stroke="currentColor" marker-end="url(#kerkar)"/>
  <text x="250" y="46" text-anchor="middle" font-size="8" fill="currentColor">needs</text>
  <defs><marker id="kerkar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Publish the algorithm, keep only the key secret — security must not depend on hiding the design.</figcaption>
</figure>

## How it works

The principle was articulated by Auguste Kerckhoffs in the nineteenth century as one of
several design requirements for military ciphers, and later restated by Claude Shannon as the
maxim "the enemy knows the system." Its practical force is simple: assume your adversary has a
full description of the algorithm, and design so that this knowledge alone gives them no
advantage — the only thing they still lack is the [key](/reference/cryptographic-key/).

This is the direct rebuttal to **security through obscurity**, the idea that a secret
*method* can substitute for a secret key. Hidden methods tend to leak, get reverse-engineered,
or be independently rediscovered, after which a system whose safety depended on that secrecy
collapses. Open, publicly reviewed algorithms — such as those used in
[symmetric](/reference/symmetric-key-cryptography/) and
[public-key](/reference/public-key-cryptography/) cryptography — are scrutinised by many
analysts and gain confidence precisely because their internals are known yet they remain
unbroken.

## Relevance to SDR

The principle is the right lens for the difference between *encryption* and *obfuscation* in
radio systems. The open [P25](/reference/project-25/) and [DMR](/reference/dmr/) air interfaces
publish their framing and even their cipher choices, and their voice protection still holds
because the [key](/reference/cryptographic-key/) is secret — exactly as Kerckhoffs prescribes.
By contrast, a scheme that relies on keeping its *method* hidden is mere obfuscation: once the
method is recovered from public on-air data, anyone can reverse it. GopherTrunk's reference
work treats such schemes strictly clean-room — analysed only from observed signals and public
documentation, never from any third-party source — and the Motorola P25 talker-alias encoding
studied in issue #773 is an obfuscation of this kind rather than true encryption.

## Sources

[^wiki]: [Kerckhoffs's principle](https://en.wikipedia.org/wiki/Kerckhoffs%27s_principle) — Wikipedia, for the requirement that only the key be secret and Shannon's restatement.
