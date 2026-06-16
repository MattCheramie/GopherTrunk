---
slug: rc4-cipher
title: RC4 (ARC4) cipher
entry_type: algorithm
category: algorithms
description: RC4 (ARC4) is a stream cipher that XORs data with a key-derived keystream; it provides the "Enhanced Privacy" encryption option in some DMR systems.
keywords: RC4, ARC4, stream cipher, keystream, DMR Enhanced Privacy, encryption
aka: [RC4, ARC4]
autolink: true
infobox:
  - { label: Type, value: Stream cipher }
  - { label: Method, value: XOR with key-derived keystream }
  - { label: Seen in, value: DMR Enhanced Privacy }
see_also: [scrambling, dmr, forward-error-correction]
related_lessons:
  - { title: "Encryption & what you can decode", url: /learn/encryption/ }
external:
  - { title: "RC4 (Wikipedia)", url: https://en.wikipedia.org/wiki/RC4 }
---

**RC4** (also **ARC4**) is a stream cipher that generates a pseudo-random keystream from a
secret key and XORs it with the data. It provides the "Enhanced Privacy" encryption option
on some [DMR](/reference/dmr/) systems.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A key generating a keystream that is XORed with plaintext to produce ciphertext." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="44" width="50" height="26" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="55" y="61" text-anchor="middle" font-size="9" fill="currentColor">key</text>
  <line x1="80" y1="57" x2="120" y2="57" stroke="currentColor" marker-end="url(#rcar)"/>
  <rect x="122" y="44" width="80" height="26" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="162" y="61" text-anchor="middle" font-size="8" fill="currentColor">keystream</text>
  <circle cx="250" cy="57" r="12" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="250" y="61" text-anchor="middle" font-size="11" fill="currentColor">⊕</text>
  <text x="250" y="92" text-anchor="middle" font-size="8" fill="currentColor">plaintext</text><line x1="250" y1="88" x2="250" y2="70" stroke="currentColor"/>
  <line x1="202" y1="57" x2="238" y2="57" stroke="currentColor"/>
  <line x1="262" y1="57" x2="320" y2="57" stroke="currentColor" marker-end="url(#rcar)"/><text x="370" y="61" font-size="9" fill="currentColor">ciphertext</text>
  <defs><marker id="rcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>RC4 is a stream cipher: a key seeds a keystream that is XORed with the data — used by DMR's basic privacy.</figcaption>
</figure>

## How it works

Only holders of the correct key can reproduce the keystream and recover the plaintext.
Unlike [scrambling](/reference/scrambling/) (a public sequence), RC4's keystream is
secret — so without the key the voice cannot be recovered.

## Relevance to SDR

RC4 illustrates the line between reversible whitening and true encryption: GopherTrunk can
descramble whitening but cannot decode encrypted voice without the key. See
[DMR encryption](/dmr-encryption.html).
