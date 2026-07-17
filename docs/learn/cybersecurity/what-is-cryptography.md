---
slug: what-is-cryptography
title: What is cryptography?
description: A plain-language introduction to cryptography — how encryption turns readable plaintext into scrambled ciphertext that only a key holder can undo, why the key is the secret and the algorithm is public, and why encoding and hashing are not the same thing.
keywords: cryptography, encryption, plaintext, ciphertext, key, cipher, decryption, Kerckhoffs principle, encoding vs encryption, hashing, don't roll your own crypto, confidentiality
level: beginner
status: full
prereq:
  - what-is-cybersecurity
faq:
  - q: What is cryptography in simple terms?
    a: Cryptography is the practice of scrambling information so that only someone holding the right key can read it. You start with readable plaintext, apply an algorithm and a key to produce scrambled ciphertext, and reverse the process with the key to get the plaintext back. Without the key, the ciphertext should be useless to anyone who intercepts it.
  - q: What's the difference between a cipher and a key?
    a: "The cipher is the algorithm — the fixed, public recipe for scrambling and unscrambling. The key is the secret input that makes your particular result unique. Anyone can know the cipher; safety comes from keeping the key secret. This is Kerckhoffs's principle: assume the attacker knows the algorithm, and rely only on the key."
  - q: Is Base64 encoding a form of encryption?
    a: "No. Encoding like Base64 just reformats data into another representation, and anyone can reverse it — there is no key and no secret. Encryption is reversible only with the correct key. Treating encoding as if it protects data is a common and dangerous mistake."
  - q: Why shouldn't I write my own encryption?
    a: Because homemade schemes fail in subtle ways that only show up under expert attack, long after they look fine to their author. Real cryptography is public, peer-reviewed, and battle-tested for years. The safe move is to use vetted standard libraries and protocols rather than inventing your own — the danger lives in the details.
---

# What is cryptography?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Cryptography scrambles information so only the key holder can read it. You turn
readable **plaintext** into scrambled **ciphertext** and back again using a
**key**. The crucial idea: **security lives in the key**, not in keeping the
algorithm secret. And the golden rule for everyone who isn't a cryptographer —
**don't roll your own**: use vetted, standard tools. This module builds on
[what cybersecurity is](/learn/cybersecurity/what-is-cybersecurity/).
</div>

Almost every protection you'll meet later in this path — private messages, secure
logins, tamper-proof updates, the padlock in your browser — rests on
cryptography. It's the machinery that lets two parties trust data even when it
travels across networks full of strangers. This lesson gives you the mental model;
the rest of the module fills in how each piece works.

## Encryption in one picture

At its core, encryption is a two-way trip:

- **Encrypt:** take **plaintext** (the readable message) plus a **key**, run them
  through a **cipher** (the algorithm), and out comes **ciphertext** — scrambled,
  meaningless-looking data.
- **Decrypt:** take that ciphertext plus the key, run it back through the cipher,
  and recover the original plaintext.

The whole point is what happens **without** the key. Someone who intercepts the
ciphertext — on a network, on a stolen disk, over the air — should be left with
noise. The key is the difference between a readable message and useless garbage.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A flow diagram. Plaintext plus a key feed into an encrypt box, producing ciphertext. The ciphertext plus the key feed into a decrypt box, producing plaintext again." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="14" y="44" width="70" height="30" rx="4"/>
    <rect x="150" y="30" width="60" height="30" rx="4"/>
    <rect x="270" y="44" width="70" height="30" rx="4"/>
    <rect x="380" y="30" width="66" height="30" rx="4"/>
    <path d="M84 59 h60" marker-end="url(#a)"/>
    <path d="M210 45 h60" marker-end="url(#a)"/>
    <path d="M340 59 h34" marker-end="url(#a)"/>
  </g>
  <defs>
    <marker id="a" markerWidth="7" markerHeight="7" refX="5" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker>
  </defs>
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <text x="49" y="62">plaintext</text>
    <text x="180" y="49">encrypt</text>
    <text x="305" y="62">ciphertext</text>
    <text x="413" y="49">decrypt</text>
    <text x="180" y="22">🔑 key</text>
    <text x="413" y="22">🔑 key</text>
    <text x="305" y="92">unreadable without the key</text>
  </g>
</svg>
<figcaption>Plaintext plus a key becomes ciphertext; the same key turns it back. Intercept the ciphertext without the key and you have nothing.</figcaption>
</figure>

## The key is the secret, not the algorithm

Beginners assume good cryptography works by keeping the method secret. It's the
opposite. The guiding rule is **Kerckhoffs's principle**: design your system so it
stays secure **even if the attacker knows everything about the algorithm** — the
only thing they must not have is the key.

That sounds backwards until you see why it's strength, not weakness. A secret
algorithm has been examined by exactly the people who wrote it. A **public**
algorithm is attacked for years by cryptographers worldwide trying to break it,
and the ones still standing are the ones we trust. Secrecy of the method is a
crutch that snaps the moment someone reverse-engineers your software or leaks a
document. Secrecy of the **key** is a promise you can actually keep — and can
change if it's ever exposed.

So the modern standards — AES, RSA, the algorithms behind HTTPS — are all
completely public. Their safety rests entirely on the key.

## Don't roll your own crypto

Here's the single most repeated piece of advice in the field: **don't roll your
own crypto.** Writing an encryption scheme that looks fine is easy. Writing one
that survives expert attack is extraordinarily hard, and the failures are
**subtle** — a predictable random number, a reused value, a timing quirk that
leaks the key one bit at a time. None of these show up in normal testing. The code
runs, the output looks scrambled, and it's quietly broken.

The professional move is to **use vetted, standard libraries and protocols** built
and reviewed by specialists, rather than inventing your own cipher or stitching
one together from primitives you don't fully understand. The danger is always in
the details, and the details are exactly what a homegrown scheme gets wrong.

## Encoding vs. encryption vs. hashing

Three ideas get confused constantly. They are not interchangeable:

- **Encoding** (for example **Base64**) reformats data into another
  representation — say, to survive being sent through a text-only channel. It's
  **reversible by anyone**, uses **no key**, and provides **no secrecy at all**.
  Base64 is not protection; treating it as such is a classic mistake.
- **Encryption** is reversible **only with the correct key**. That's the whole
  point — it hides the content from anyone without the key.
- **Hashing** is a **one-way** transformation: it turns data into a fixed
  fingerprint that can't be reversed back into the original. It's used to detect
  tampering and to store passwords safely, and it's covered in
  [hashing & integrity](/learn/cybersecurity/hashing-and-integrity/).

If you remember one line: encoding is not secret, encryption is secret with a key,
and hashing doesn't come back at all.

## What crypto does and doesn't do

Used correctly, cryptography protects **confidentiality** (only the key holder can
read the data) and helps guarantee **integrity** (you can tell if data was
changed) — both for data **in transit** across a network and data **at rest** on a
disk. That's a lot of the security you rely on every day.

But cryptography is not magic, and it protects a narrow thing. It **cannot** fix:

- **Bad key handling** — a key emailed in plaintext or hard-coded in your app is
  no longer secret, and the encryption around it is worthless. Handling keys well
  is its own discipline; see [managing secrets & keys](/learn/cybersecurity/secrets-management/).
- **Weak passwords** — if the key or password is guessable, the strongest cipher
  in the world folds instantly.
- **An already-compromised machine** — if an attacker is already on the device
  reading your plaintext before it's encrypted (or after it's decrypted),
  encryption on the wire never sees the data they're stealing.

Cryptography secures data as it moves and rests. It does nothing for the endpoints
and the humans handling the keys — which is exactly why the rest of this path
exists.

<div class="knowledge-check" data-quiz data-correct-msg="Right — assume the attacker knows the algorithm; only the key stays secret." markdown="0">
  <p class="knowledge-check__q">Quick check: in a well-designed cipher, what must stay secret for it to be safe?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The algorithm — nobody should know how it works</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The key — not the algorithm</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Both the key and the algorithm, equally</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Cryptography **scrambles information** so only the key holder can read it:
  **plaintext** + key → **ciphertext**, and back again.
- Without the key, ciphertext should be **useless** to anyone who intercepts it.
- **Security lives in the key**, not the algorithm — Kerckhoffs's principle means
  good crypto is public and peer-reviewed.
- **Don't roll your own** — use vetted, standard libraries and protocols, because
  the failures are subtle.
- **Encoding** (like Base64) is reversible and not secret; **encryption** needs a
  key; **hashing** is one-way — don't confuse them.
- Crypto protects **confidentiality and integrity** in transit and at rest, but
  can't fix bad key handling, weak passwords, or an already-compromised machine.

Next up: [symmetric & public-key encryption](/learn/cybersecurity/symmetric-and-asymmetric/)
