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

## How it works

Only holders of the correct key can reproduce the keystream and recover the plaintext.
Unlike [scrambling](/reference/scrambling/) (a public sequence), RC4's keystream is
secret — so without the key the voice cannot be recovered.

## Relevance to SDR

RC4 illustrates the line between reversible whitening and true encryption: GopherTrunk can
descramble whitening but cannot decode encrypted voice without the key. See
[DMR encryption](/dmr-encryption.html).
