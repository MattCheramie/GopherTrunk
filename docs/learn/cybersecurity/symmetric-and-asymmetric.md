---
slug: symmetric-and-asymmetric
title: Symmetric & public-key encryption
description: The two families of encryption in plain language — symmetric encryption with one shared key (like AES) for fast bulk data, asymmetric public-key encryption with a public/private key pair (like RSA) that solves key exchange, and how real systems combine them in a hybrid scheme.
keywords: symmetric encryption, asymmetric encryption, public key, private key, AES, RSA, key exchange, hybrid encryption, elliptic curve, key distribution, encryption keys
level: intermediate
status: full
prereq:
  - what-is-cryptography
faq:
  - q: What is the difference between symmetric and asymmetric encryption?
    a: "Symmetric encryption uses one shared secret key to both encrypt and decrypt, so it's fast and ideal for bulk data — its catch is safely getting that key to the other side. Asymmetric (public-key) encryption uses a linked pair of keys: a public key anyone can use to encrypt, and a private key only you hold to decrypt. It solves the key-sharing problem but is much slower, so real systems use it to exchange a symmetric key and then switch to symmetric encryption."
  - q: Which type of encryption uses a public and private key pair?
    a: "Asymmetric encryption, also called public-key encryption. The two keys are mathematically linked: what the public key locks, only the matching private key can unlock. You can hand out the public key freely while keeping the private key secret. RSA and elliptic-curve cryptography are the common examples."
  - q: What is hybrid encryption and why is it used?
    a: "Hybrid encryption combines both families: it uses slow public-key crypto once, just to agree on a fresh random symmetric key, then encrypts the actual data with fast symmetric encryption. This gives you the easy key exchange of public-key crypto and the speed of symmetric crypto at the same time. HTTPS/TLS works exactly this way on every secure web connection."
  - q: Is AES symmetric or asymmetric?
    a: "AES is a symmetric algorithm — the same key encrypts and decrypts, and it's built for encrypting large amounts of data quickly. RSA and elliptic-curve cryptography, by contrast, are asymmetric: they use a public/private key pair. In practice AES and RSA often work together, with the asymmetric algorithm delivering the AES key."
---

# Symmetric & public-key encryption

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
There are two families of encryption. **Symmetric (one shared key)** — the same
secret encrypts and decrypts (like AES); it's fast, but you have to get the key
to the other side safely. **Asymmetric (key pair)** — a public key anyone can
encrypt with and a private key only you can decrypt with (like RSA); it solves
key sharing but is slow. Real systems use **hybrid** encryption: public-key crypto
to exchange a symmetric key, then symmetric crypto for the data. They solve
different problems, which is why, as
[what is cryptography?](/learn/cybersecurity/what-is-cryptography/) sets up, secure
systems lean on both.
</div>

The last lesson introduced the idea of a key: a secret that turns readable
plaintext into scrambled ciphertext and back. But that raised a question it
didn't answer — how do the two sides come to share a key in the first place,
especially if they've never met? The answer is that there isn't one kind of
encryption, there are two, and they exist precisely because they solve different
halves of that problem.

## Symmetric encryption

**Symmetric encryption** uses a **single shared secret key**. The same key that
scrambles the message also unscrambles it, so both the sender and the receiver
must hold an identical copy. The modern workhorse here is **AES** (the Advanced
Encryption Standard), and it's everywhere: your disk encryption, the contents of
a secure web page, the files in an encrypted archive.

Its great strength is **speed**. Symmetric algorithms are fast and efficient
enough to encrypt huge amounts of data — video streams, whole drives — without
slowing things down, which is why they carry the actual payload in almost every
secure system.

The catch is right there in the word *shared*. Both ends need the same secret
key, and handing it over safely is genuinely hard: you can't just email it,
because anyone who intercepts that message now has the key and can read
everything. This is the **key-distribution problem**, and for a long time it was
the fundamental obstacle in cryptography. If the only safe way to share a key is
to meet in person, secure communication between strangers is impossible.

## Public-key (asymmetric) encryption

**Public-key encryption**, also called **asymmetric encryption**, breaks that
deadlock. Instead of one shared secret, each person has a **pair of
mathematically linked keys**: a **public key** and a **private key**. The two are
generated together and bound by hard math (as in **RSA** or **elliptic-curve
cryptography**), but knowing the public key doesn't let anyone work out the
private one.

The pair works in a beautifully lopsided way:

- Your **public key** is not a secret. You can publish it, print it, hand it to
  anyone. Anything encrypted *with* your public key can only be decrypted by the
  matching private key.
- Your **private key** never leaves your possession. It's the only thing that can
  unlock messages sealed with your public key.

So anyone in the world can encrypt a message that **only you** can read, without
ever having shared a secret with you beforehand. That directly solves the
key-distribution problem — no prior meeting required.

The trade-off is **performance**. Asymmetric math is far slower and more
computationally expensive than symmetric encryption, so it would be painful to
use for encrypting large amounts of data. It's brilliant for setting things up,
but a poor choice for doing the heavy lifting.

## Hybrid: the best of both

Notice that the two families have exactly opposite strengths and weaknesses.
Symmetric is **fast but hard to set up** (the key-sharing problem); asymmetric is
**easy to set up but slow**. The obvious move is to use each for what it's good
at — and that's precisely what real protocols do. It's called **hybrid
encryption**.

The pattern is:

1. Use slow **public-key** crypto once, at the start, just to agree on a fresh,
   random **symmetric key** — safely, over an open channel.
2. Then switch to fast **symmetric** encryption (AES) using that key to protect
   all the actual data for the rest of the session.

This is exactly how **[TLS](/learn/networking/tls-and-https/)** — the protocol
behind HTTPS — secures every web page you load. The little padlock in your
browser represents a hybrid handshake: public-key crypto to exchange a key, then
symmetric crypto for the traffic. You get the easy setup of one family and the
speed of the other, and neither is asked to do the job it's bad at.

## Signatures use the pair in reverse

There's one more trick hiding in the key pair, and it's worth previewing. So far
we've encrypted **with the public key** so that only the private key can decrypt.
But you can also run the pair the other way: something processed **with the
private key** can be checked by **anyone** using the public key.

Because only you hold your private key, that becomes a way to prove *you*
produced something — and that it wasn't altered afterwards. That's a **digital
signature**, and it's how your device knows a software update or a website is
genuine. We'll unpack it fully in
[signatures & certificates](/learn/cybersecurity/signatures-and-certificates/).

## Where you'll meet them

These two families aren't academic — they're underneath nearly everything you do
online. **HTTPS** secures web pages, **SSH** protects remote logins, encrypted
**messaging** apps guard your chats, and even **encrypted radio** systems on
trunked networks rely on the same ideas to keep transmissions private. In every
case the pattern is the same: asymmetric crypto to establish trust and exchange a
key, symmetric crypto to protect the bulk of the data. We'll see these real-world
uses side by side in
[cryptography in practice](/learn/cybersecurity/crypto-in-practice/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a public/private key pair is the defining feature of asymmetric (public-key) encryption." markdown="0">
  <p class="knowledge-check__q">Quick check: which type of encryption uses a public/private key pair?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Symmetric encryption (one shared key)</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Asymmetric (public-key) encryption</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Hashing</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- There are two families of encryption, and they solve **different problems**.
- **Symmetric** encryption uses **one shared secret key** (e.g. AES); it's fast
  and carries bulk data, but sharing the key safely is the hard part.
- **Asymmetric** (public-key) encryption uses a linked **public/private key pair**
  (e.g. RSA, elliptic curve); anyone can encrypt with the public key, only the
  private key decrypts — solving key exchange, but slowly.
- **Hybrid** encryption combines them: public-key crypto to exchange a fresh
  symmetric key, then symmetric crypto for the data — exactly how TLS/HTTPS works.
- Run the key pair **in reverse** and you get digital signatures, which prove who
  sent something and that it wasn't changed.
- HTTPS, SSH, messaging, and encrypted radio all rest on these two families
  working together.

Next up: [hashing & integrity](/learn/cybersecurity/hashing-and-integrity/)
