---
slug: signatures-and-certificates
title: Digital signatures & certificates
description: How digital signatures prove authenticity and integrity — signing with a private key and verifying with the public key — and how certificates and PKI bind a public key to a real identity through a chain of trust rooted in certificate authorities.
keywords: digital signature, non-repudiation, PKI, certificate, certificate authority, chain of trust, code signing, public key, authenticity, integrity
level: intermediate
status: full
prereq:
  - symmetric-and-asymmetric
faq:
  - q: What does a digital signature prove?
    a: "A valid digital signature proves two things at once — authenticity (the message really came from the holder of a particular private key) and integrity (the message wasn't altered after it was signed). Because only the signer holds that private key, it also gives non-repudiation: they can't credibly deny signing it. What a signature does NOT prove on its own is whose key it is — that's the job of a certificate."
  - q: How is a digital signature different from encryption?
    a: They use the same key pair but for opposite goals. Encryption hides content — you encrypt with the recipient's public key so only their private key can read it. A signature proves origin — you sign with your own private key so anyone with your public key can verify it. Signing doesn't hide anything; the message stays readable and the signature just travels alongside it.
  - q: What is a certificate authority?
    a: A certificate authority (CA) is an organization that vouches for identities by signing certificates. A certificate binds a name — a domain, person, or organization — to a public key, and the CA's signature says "we checked, and this key really belongs to that name." Your device ships with a list of trusted root CAs, and everything they sign is trusted by extension, forming a chain of trust.
  - q: Why do expired or untrusted certificates trigger warnings?
    a: A certificate has a validity window and a signing chain that must lead back to a root your device trusts. If it's expired, self-signed, or signed by an unknown authority, the binding between the identity and the key can't be verified — so your device can't be sure you're talking to who you think. That's a genuine red flag worth heeding, not a nuisance to click through.
---

# Digital signatures & certificates

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **signature** is made by **signing with a private key** and checked by
**verifying with the matching public key**. A valid one proves **authenticity +
integrity** — who sent something, and that it wasn't changed. But a key alone
doesn't say *whose* it is: **certificates** and **PKI** fix that by binding an
identity to a public key, vouched for through a **chain of trust**. Together they
answer "did this really come from them, untampered?" — building on
[symmetric & asymmetric crypto](/learn/cybersecurity/symmetric-and-asymmetric/).
</div>

Encryption keeps a message secret. Signatures answer a different, equally
important question: **can I trust where this came from?** Almost every secure
system you touch — websites, app updates, package managers — leans on signatures
and certificates to know it's talking to the real thing.

## How a signature works

A digital signature reuses the public/private key pair from
[asymmetric crypto](/learn/cybersecurity/symmetric-and-asymmetric/), but runs it
in the opposite direction from encryption.

1. The signer takes the message and computes a
   [hash](/learn/cybersecurity/hashing-and-integrity/) of it — a short, fixed-size
   fingerprint.
2. They **encrypt that hash with their private key**. The result is the signature,
   attached alongside the message.
3. Anyone can **verify** it: hash the received message themselves, then use the
   signer's **public key** to check the signature against that hash.

If the two match, three things follow at once:

- **Authenticity** — only the private-key holder could have produced a signature
  that verifies with their public key.
- **Integrity** — change one bit of the message and the recomputed hash won't
  match, so verification fails.
- **Non-repudiation** — the signer can't credibly deny it later, because nobody
  else has their private key.

Note the signature **does not hide** the message. It travels next to readable
content and simply proves the content's origin and that it's intact.

## The identity problem

Here's the gap. A signature that verifies with "this public key" only tells you
the message came from **whoever holds the matching private key**. It says nothing
about *who that is*.

An attacker can generate their own key pair, sign a fake message, and hand you
the matching public key. The math checks out perfectly — it just proves the
forgery was signed by the forger. A public key on its own is an anonymous string
of bytes. Something has to connect a key to a real-world identity you can trust.

That something is a **certificate**.

## Certificates & PKI

A **certificate** binds an **identity** — a domain name, a person, or an
organization — to a **public key**. Crucially, the certificate is itself
**signed**, by a trusted third party called a **Certificate Authority (CA)**.
The CA's signature is a statement: "we verified that this public key really
belongs to this identity."

This whole arrangement — the CAs, the certificates, and the rules for issuing and
checking them — is called **PKI (Public Key Infrastructure)**.

Trust doesn't hang on a single signer. Your device ships with a built-in list of
**root CAs** it trusts. A root CA can sign an intermediate CA's certificate,
which in turn signs an end-entity certificate — a **chain of trust**. To trust
the certificate in front of you, your device follows that chain link by link back
to a root it already trusts. Break any link and the trust collapses.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 120" role="img" aria-label="A chain of trust drawn as three linked boxes. Root CA on the left signs an Intermediate CA in the middle, which signs a site certificate on the right, each connected by an arrow labelled signs." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none">
    <rect x="14" y="42" width="110" height="36" rx="4"/>
    <rect x="165" y="42" width="110" height="36" rx="4"/>
    <rect x="316" y="42" width="110" height="36" rx="4"/>
    <line x1="124" y1="60" x2="163" y2="60"/>
    <line x1="275" y1="60" x2="314" y2="60"/>
    <path d="M158 56 l6 4 -6 4" stroke-width="1.2"/>
    <path d="M309 56 l6 4 -6 4" stroke-width="1.2"/>
  </g>
  <g font-size="11" fill="currentColor" text-anchor="middle">
    <text x="69" y="64">Root CA</text>
    <text x="220" y="64">Intermediate</text>
    <text x="371" y="64">Site cert</text>
    <text x="144" y="52">signs</text>
    <text x="295" y="52">signs</text>
    <text x="69" y="30">trusted by device</text>
  </g>
</svg>
<figcaption>A chain of trust: your device trusts the root, the root vouches for the intermediate, the intermediate vouches for the site.</figcaption>
</figure>

## Where you rely on it

You lean on signatures and certificates constantly, usually without noticing:

- **HTTPS certificates** — the padlock in your browser means a CA-signed
  certificate proved the site's identity before any traffic flowed. See
  [TLS & HTTPS](/learn/networking/tls-and-https/).
- **Signed software updates (code signing)** — your operating system and apps
  verify a vendor's signature before installing an update, so tampered or spoofed
  updates are rejected.
- **Signed packages** — package managers check signatures so you install what the
  maintainer actually published, not a swapped-out copy.
- **Signed emails** — a signature lets a recipient confirm a message really came
  from the sender and wasn't altered in transit.

In each case a **fake, mismatched, or expired certificate is a red flag**. When
your device refuses one, the binding between identity and key couldn't be
verified — that warning is doing its job.

## Trust, and its limits

Signatures and certificates are powerful, but they're not magic. The system is
only as trustworthy as **the CAs it rests on** and **how well private keys are
guarded**.

- If a CA is careless or compromised, it can issue certificates for names it
  never should have — and everything under it inherits that mistake.
- If a signer's **private key is stolen**, the thief can produce signatures that
  verify perfectly. The math still says "authentic" because, as far as the key is
  concerned, it *is*. This is why protecting private keys is its own discipline —
  see secrets management (coming up).

The takeaway: a valid signature proves control of a key, and a certificate proves
someone once vouched for that key's owner. Both are only as strong as the humans
and processes behind them.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a valid signature ties the message to the private-key holder and guarantees it wasn't changed." markdown="0">
  <p class="knowledge-check__q">Quick check: a valid digital signature on a message proves what?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The message was kept secret from everyone else</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Who signed it and that the message wasn't altered</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The signer is a trustworthy person</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A signature is made by **signing with a private key** and checked by
  **verifying with the public key**.
- A valid signature proves **authenticity, integrity, and non-repudiation** — but
  not *whose* key it is.
- A **certificate** binds an identity to a public key and is itself signed by a
  **Certificate Authority**.
- **PKI** and the **chain of trust** let your device follow signatures back to a
  root CA it already trusts.
- You rely on this for **HTTPS, code signing, signed packages, and signed email**;
  a fake or expired certificate is a red flag.
- The whole system is only as strong as the **CAs** and your **private-key
  handling** — a stolen key breaks it.

Next up: [cryptography in practice](/learn/cybersecurity/crypto-in-practice/)
