---
slug: hashing-and-integrity
title: Hashing & integrity
description: What a cryptographic hash is and why it matters — a one-way fingerprint of any data, how checksums verify a file wasn't changed, how to store passwords safely with salted slow hashes like bcrypt and argon2, why old hashes are broken, and how hashing differs from encryption.
keywords: hash, hashing, SHA-256, checksum, integrity, password hashing, salt, bcrypt, argon2, collision, digest, one-way function
level: intermediate
status: full
prereq:
  - what-is-cryptography
faq:
  - q: What is a cryptographic hash?
    a: "A cryptographic hash is a one-way function that turns any input — a file, a password, a message — into a fixed-size string of characters called a digest. The same input always produces the same hash, even a tiny change produces a completely different one, and you can't work backward from the hash to the original input. It acts like a fingerprint for data."
  - q: How does hashing verify file integrity?
    a: "You hash the file when you know it's good and publish that digest. Anyone who downloads the file hashes their copy and compares. If the two hashes match, the file is bit-for-bit identical to the original; if they differ by even one character, something changed in transit or storage. This is how downloads, backups, and tamper-detection systems confirm data wasn't altered."
  - q: Why hash passwords instead of storing them?
    a: "If you store passwords directly and the database leaks, every account is instantly exposed. Storing only a hash means the site can still check a login (hash the entered password, compare) without ever keeping the real one. To resist cracking you add a unique salt per password and use a deliberately slow algorithm like bcrypt, scrypt, or argon2, so guessing at scale is expensive."
  - q: Is hashing the same as encryption?
    a: "No. Encryption is reversible — with the right key you can turn the ciphertext back into the original. Hashing is one-way and has no key: there's no operation that recovers the input from a digest. Encryption keeps data secret so the right person can read it later; hashing proves data is unchanged or checks a secret without storing it. They solve different problems."
---

# Hashing & integrity

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **hash** is a **one-way** function: it turns any input into a fixed-size
fingerprint that's easy to compute but practically impossible to reverse.
Comparing hashes gives you **integrity** — a **checksum** that proves a file or
message wasn't changed. For **password hashing** you add a **salt** and a
deliberately **slow** algorithm so a leaked database resists cracking. And
remember: **hashing is not encryption** — there's no key and no way back. See
[what is cryptography?](/learn/cybersecurity/what-is-cryptography/) for the bigger
picture.
</div>

Hashing is one of the most useful ideas in security, and one of the most misused.
Get it right and you can verify a download, detect tampering, and store passwords
that survive a breach. Get it wrong — a broken algorithm, no salt, or confusing
it with encryption — and you build something that only *looks* secure. This
lesson keeps it plain and defensive.

## What a hash is

A **hash function** takes an input of any size and produces a fixed-size output
called a **digest** — often written as a string of hexadecimal characters. A
good cryptographic hash has three properties that make it useful:

- **Deterministic** — the same input always produces the same digest. Hash the
  word `hello` today and next year and you get the identical result.
- **Avalanche effect** — change the input by a single bit and the output changes
  completely. There's no partial resemblance to leak how close you were.
- **One-way** — given a digest, there's no practical way to work backward to the
  input. You can go forward cheaply, but not back.

That combination is why a hash works like a **fingerprint** for data: small,
fixed-size, and uniquely tied to whatever produced it. SHA-256 — part of the
SHA-2 family — is a common modern example, producing a 256-bit digest for any
input you feed it.

## Integrity & checksums

The most direct use of a hash is proving that data **hasn't changed**. Hash the
data when you know it's trustworthy, keep that digest somewhere safe, and hash it
again later. If the two match, the data is bit-for-bit identical; if they differ
at all, something altered it. A published hash used this way is called a
**checksum**.

You'll see this everywhere:

- **Downloads** — a project publishes the SHA-256 of a release so you can confirm
  your copy arrived intact and wasn't swapped for a malicious one.
- **Backups** — comparing stored hashes flags files that got corrupted on disk.
- **Tamper detection** — a monitoring tool hashes important system files and
  alerts you when a digest changes unexpectedly.

This is the **integrity** leg of the
[CIA triad](/learn/cybersecurity/cia-triad/) made concrete: hashing doesn't keep
data secret, it lets you *detect* that data was modified. (On its own a checksum
only catches accidental change or an attacker who can't also replace the
published hash — pairing it with a signature closes that gap, which the next
lesson covers.)

## Storing passwords safely

The rule is simple: **never store passwords**. If your database holds the actual
passwords and it leaks, every account is compromised the instant the file walks
out the door. Instead, store a **hash** of each password. At login you hash what
the user typed and compare digests — the site verifies the password without ever
keeping it.

But a plain hash isn't enough for passwords. Two problems:

- People reuse passwords, so identical passwords produce identical hashes — and
  attackers precompute huge tables of common passwords and their hashes. The fix
  is a **salt**: a unique random value added to each password before hashing, so
  the same password yields a different digest for every user and precomputed
  tables become useless.
- Fast hashes like SHA-256 are built to be *quick*, which is exactly wrong here —
  it lets an attacker try billions of guesses per second against a leaked
  database. The fix is a **slow**, purpose-built password hash: **bcrypt**,
  **scrypt**, or **argon2**. These are deliberately expensive to compute (and
  tunable to stay expensive as hardware improves), so mass guessing is costly
  even after a breach.

So a password store should hold a **salted, slow** hash — never the password, and
never a bare fast hash like SHA-256 alone. There's more to protecting logins —
rate limiting, multi-factor — in
[passwords & MFA](/learn/cybersecurity/passwords-and-mfa/).

## Collisions & choosing a hash

A **collision** is when two different inputs produce the same digest. For a
strong hash, finding one should be so hard it never happens in practice. When
researchers *do* find a practical way to produce collisions, the hash is
considered **broken** for security use — an attacker could craft a malicious file
that matches the digest of a trusted one.

That's what happened to the older algorithms: **MD5** and **SHA-1** both have
known, demonstrated collision weaknesses and must not be used where security
depends on it. Prefer a modern hash — the **SHA-256** family (SHA-2) or newer —
for integrity and signatures. (You may still see MD5 or CRC used as a plain
error-check for accidental corruption, where no attacker is involved, but treat
that as a non-security checksum only.) You don't need to know *how* the attacks
work to follow the rule: pick a current, well-regarded hash and retire the old
ones.

## Hashing is not encryption

These get confused constantly, so keep them straight:

- **Encryption** is **reversible**. It scrambles data with a **key**, and someone
  with the right key can turn it back into the original. Its job is secrecy — the
  right person reads it later.
- **Hashing** is **one-way** and has **no key**. There's no operation that
  recovers the input from a digest. Its job is integrity and verification —
  proving data is unchanged, or checking a secret without storing it.

If someone talks about "reversing a hash" to get a password back, they're
describing *guessing* — hashing candidate inputs until one matches — not
decryption. There's no key to hashing and no way back by design. Encryption and
hashing solve **different problems**, and secure systems usually use both. See
[what is cryptography?](/learn/cybersecurity/what-is-cryptography/) for how the
pieces fit together.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a hash is one-way, so a leaked store can't be turned back into the original passwords." markdown="0">
  <p class="knowledge-check__q">Quick check: why store a hash of a password instead of encrypting it?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Hashes are smaller and save disk space</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Hashing is one-way — even if the store leaks, the original password can't be recovered from it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Hashing is faster than encryption at login time</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **hash** is a one-way function producing a fixed-size **digest** — a
  fingerprint that's cheap to compute and practically impossible to reverse.
- The same input always hashes the same way; a tiny change produces a completely
  different digest.
- Comparing hashes is a **checksum** for **integrity** — verifying downloads,
  backups, and untampered files.
- Store passwords as a **salted**, **slow** hash (bcrypt, scrypt, argon2), never
  as plaintext and never as a bare fast hash like SHA-256 alone.
- Old hashes (**MD5**, **SHA-1**) have known **collisions** and are broken for
  security use — prefer the **SHA-256** family.
- **Hashing is not encryption**: encryption is reversible with a key, hashing is
  one-way with none — they solve different problems.

Next up: [digital signatures & certificates](/learn/cybersecurity/signatures-and-certificates/)
