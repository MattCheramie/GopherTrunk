---
slug: stream-cipher
title: Stream cipher
entry_type: algorithm
category: cryptography
description: A stream cipher is a symmetric cipher that encrypts data one bit or byte at a time by XORing it with a pseudo-random keystream derived from a secret key and nonce.
keywords: stream cipher, keystream, XOR, RC4, ChaCha20, synchronous, self-synchronizing, OFB, CTR, symmetric encryption, key reuse, nonce, pseudo-random sequence
aka: [stream cipher]
autolink: true
infobox:
  - { label: Type, value: "Symmetric cipher" }
  - { label: Unit, value: "Bit or byte at a time" }
  - { label: Operation, value: "Plaintext XOR keystream" }
see_also: [keystream, block-cipher, rc4-cipher, one-time-pad, advanced-encryption-standard, data-encryption-standard, linear-feedback-shift-register]
cite_urls:
  - https://en.wikipedia.org/wiki/Stream_cipher
  - https://csrc.nist.gov/pubs/sp/800/38/a/final
---

**A stream cipher** encrypts data one bit or byte at a time by combining each unit of
plaintext — almost always with XOR — against a pseudo-random [keystream](/reference/keystream/)
generated from a secret key.[^wiki] It contrasts with a
[block cipher](/reference/block-cipher/), which transforms fixed-size blocks at once. Because
the same keystream both masks and unmasks the data, a stream cipher is inherently
[symmetric](/reference/symmetric-key-cryptography/): sender and receiver must share the key
and stay bit-aligned on the stream.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A keystream generator feeds a keystream into an XOR with the plaintext stream to produce ciphertext." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="40" width="86" height="30" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="63" y="59" text-anchor="middle" font-size="8" fill="currentColor">keystream gen</text>
  <line x1="106" y1="55" x2="170" y2="55" stroke="currentColor" marker-end="url(#scar)"/>
  <circle cx="195" cy="55" r="13" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="195" y="59" text-anchor="middle" font-size="11" fill="currentColor">⊕</text>
  <text x="195" y="90" text-anchor="middle" font-size="8" fill="currentColor">plaintext</text><line x1="195" y1="86" x2="195" y2="68" stroke="currentColor"/>
  <line x1="208" y1="55" x2="320" y2="55" stroke="currentColor" marker-end="url(#scar)"/><text x="375" y="59" font-size="9" fill="currentColor">ciphertext</text>
  <defs><marker id="scar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A stream cipher XORs each plaintext bit with a keystream bit; the same operation decrypts.</figcaption>
</figure>

## How it works

A stream cipher's strength lives entirely in its keystream generator. The key (usually with
a nonce or initialization vector) seeds a state machine that emits a long pseudo-random
sequence; XORing that sequence with the plaintext gives the ciphertext, and XORing the *same*
sequence with the ciphertext recovers the plaintext, because XOR is its own inverse. The
ciphertext is exactly the length of the plaintext — there is no block padding and no
expansion, which is one reason stream ciphers suit continuous, byte-by-byte data like
digitized voice.

Two rules follow directly from the XOR structure:

- **Never reuse a keystream.** If two messages are encrypted under the same keystream,
  XORing the two ciphertexts cancels the keystream and leaks the XOR of the two plaintexts —
  the "two-time pad" break. This is why a stream cipher pairs its key with a per-message
  nonce or IV so that every message runs off a fresh keystream.
- **The keystream must look random.** A predictable generator lets an attacker reconstruct
  the sequence without the key. A bare [linear-feedback shift register](/reference/linear-feedback-shift-register/)
  fails here — its output is linear and can be solved from a short run of known bits — so real
  designs add nonlinearity. The [one-time pad](/reference/one-time-pad/) is the idealized
  limit: a truly random keystream as long as the message, provably unbreakable but impractical
  because the key is as large as the data.

## Variants

Stream ciphers split into two families by how the keystream depends on the message:

- **Synchronous** — the keystream is generated purely from the key and nonce, independent of
  the plaintext or ciphertext. Sender and receiver must stay perfectly synchronized; a lost
  or inserted bit desynchronizes the stream and garbles everything after it, so these ciphers
  need framing or resynchronization. [RC4](/reference/rc4-cipher/) and ChaCha20 are
  synchronous.
- **Self-synchronizing** (asynchronous) — each keystream unit is computed from the last few
  *ciphertext* bits, so after a bit slip the cipher automatically re-locks within a fixed
  window. Cipher-feedback (CFB) mode of a [block cipher](/reference/block-cipher/) is the
  classic example.

A distinct and very common variant is a **block cipher run as a stream cipher**. In
Output-Feedback (OFB) and Counter (CTR) modes, the block cipher is never applied to the
plaintext at all — it is repeatedly encrypted over a feedback register (OFB) or an
incrementing counter (CTR) to *manufacture* a keystream, which is then XORed with the data.
This lets a strong, well-studied block cipher such as
[AES](/reference/advanced-encryption-standard/) or [DES](/reference/data-encryption-standard/)
protect an arbitrary-length stream while behaving, on the wire, exactly like a stream
cipher.[^nist]

## In practice

CTR mode is now the workhorse: it is parallelizable, allows random access into the stream,
and only needs the block cipher's forward (encrypt) direction. Its one hard requirement is a
never-repeating counter/nonce per key — repeat it and you are back to the two-time-pad break
above. Modern authenticated modes (GCM) wrap CTR with a message-authentication tag so that
tampering is detected, closing the malleability weakness where flipping a ciphertext bit
predictably flips the same plaintext bit.

## Relevance to SDR

Stream ciphers are the encryption scheme most often seen on the trunked systems GopherTrunk
monitors. DMR "Enhanced Privacy" uses [RC4](/reference/rc4-cipher/), and P25 voice can be
protected with DES-OFB or AES — both run their underlying
[block cipher](/reference/block-cipher/) in a feedback mode that turns it into a keystream
generator, so the on-air result is a synchronous stream cipher applied to the vocoder frames.
Without the key the keystream cannot be reproduced, so GopherTrunk can detect and follow an
encrypted call but cannot recover the audio — distinct from reversible
[scrambling](/reference/scrambling/), where the whitening sequence is public and keyless.

## Sources

[^wiki]: [Stream cipher](https://en.wikipedia.org/wiki/Stream_cipher) — Wikipedia, for the per-symbol XOR-with-keystream model, synchronous vs self-synchronizing families, and the keystream-reuse pitfall.
[^nist]: [SP 800-38A, Recommendation for Block Cipher Modes of Operation](https://csrc.nist.gov/pubs/sp/800/38/a/final) — NIST, for OFB and CTR modes that turn a block cipher into a keystream generator.
