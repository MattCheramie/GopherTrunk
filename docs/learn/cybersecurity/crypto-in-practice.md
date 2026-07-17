---
slug: crypto-in-practice
title: Cryptography in practice
description: Where the cryptography you learned actually shows up — TLS for the web, SSH for remote access, VPNs, full-disk encryption, end-to-end messaging, and encrypted radio — nearly all of it a hybrid of public-key exchange plus symmetric encryption, with key management as the part that decides whether any of it holds.
keywords: TLS, SSH, disk encryption, end-to-end encryption, VPN, password manager, radio encryption, key management, encryption in transit, encryption at rest, hybrid encryption, OTAR
level: intermediate
status: full
prereq:
  - symmetric-and-asymmetric
faq:
  - q: Where is cryptography actually used day to day?
    a: "Almost everywhere, usually invisibly. TLS protects websites and apps (the padlock in your browser), SSH protects remote logins, VPNs wrap your whole connection, full-disk encryption guards a lost laptop, and end-to-end encryption keeps messaging apps readable only to the people in the chat. The same math even encrypts two-way radio voice. In nearly every case it's a hybrid: public-key methods agree on a key, then fast symmetric encryption protects the actual data."
  - q: Why do most systems combine public-key and symmetric encryption?
    a: "Because each is good at a different job. Public-key (asymmetric) cryptography solves the hard problem of two strangers agreeing on a secret over an open network without meeting first, but it's slow. Symmetric encryption is fast and handles bulk data well, but both sides need the same key. So real systems use public-key methods to exchange or agree on a symmetric key, then switch to symmetric encryption for everything after. That hybrid is exactly what TLS, SSH, and messaging apps do."
  - q: If everything is encrypted, why do breaches still happen?
    a: "Because the algorithms are rarely the weak point — key management is. Keys get stored insecurely, shared over the wrong channel, never rotated, or left on a device that's stolen. Old, broken protocols get left enabled. Strong encryption around a mishandled key protects nothing, which is why so much of security is really about protecting, rotating, and retiring keys rather than choosing a cipher."
  - q: Can a scanner or SDR decrypt encrypted radio traffic?
    a: "No. A scanner can detect that a transmission is encrypted and identify the system, but it cannot recover the spoken content without the key — that's the whole point of the encryption. Encrypted voice on trunked and digital radio uses symmetric keys distributed by a key-management process (often over the air, OTAR), and those keys never travel in the clear. GopherTrunk reports that a call is encrypted; it does not and cannot decode it."
gophertrunk_links:
  - title: DMR encryption
    url: /dmr-encryption.html
    note: how GopherTrunk reports that radio traffic is encrypted.
---

# Cryptography in practice

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The cryptography you learned isn't abstract — it's running right now behind
**TLS / SSH / disk / E2EE**. These all lean on the **same building blocks**
from
[symmetric & asymmetric encryption](/learn/cybersecurity/symmetric-and-asymmetric/):
almost every real system is a **hybrid** of a public-key exchange to agree on a
key, then fast symmetric encryption for the data. And in every one of them,
**key management is the hard part** — the algorithms are strong; keeping the
keys safe is where security is won or lost.
</div>

You've seen how symmetric and asymmetric encryption work. Now the payoff: those
same ideas are protecting nearly everything you touch online and off, usually
without announcing themselves. Learn to spot the pattern and the whole
landscape gets simpler — it's the same two tools, combined the same way, over
and over.

## Encryption in transit

**In transit** means protecting data while it moves across a network, where
anyone in the path could otherwise read or alter it.

- **TLS/HTTPS** is what puts the padlock in your browser. Every login, message,
  and payment on the modern web rides inside it — see
  [TLS & HTTPS](/learn/networking/tls-and-https/). It's a textbook hybrid: a
  public-key handshake agrees on a session key, then symmetric encryption
  carries the page.
- **SSH** protects remote logins and file transfers to servers, so your
  commands and credentials don't cross the network in the clear — see
  [SSH & remote access](/learn/linux-cli/ssh-and-remote/).
- **VPNs** wrap your *entire* connection in an encrypted tunnel, so a hostile
  network sees only ciphertext instead of which sites you reach — see
  [VPNs](/learn/networking/vpns/).

Different tools, same job: make the data unreadable to anyone who isn't the
intended endpoint while it's on the wire.

## Encryption at rest

**At rest** means protecting data while it's sitting on a disk, phone, or backup
drive — the risk being a device that's lost or stolen rather than intercepted.

**Full-disk encryption** scrambles everything on a laptop or phone so that,
without your passphrase, a thief who pockets the hardware gets a brick full of
noise instead of your files. **Encrypted files and backups** do the same for
individual documents or archive copies, so a leaked backup is useless to whoever
finds it.

Here the workhorse is usually **symmetric** encryption — fast, and there's only
one party involved, so there's no stranger to exchange keys with. The catch
moves entirely to the key: the encryption is only as good as the passphrase or
key protecting it, and a device with the key left unlocked isn't protected at
all.

## End-to-end encryption

**End-to-end encryption (E2EE)** means only the two endpoints — the people in
the conversation — hold the keys. The service carrying the messages can move the
ciphertext around but can't read it, because it never has the key. This is how
private messaging apps promise that not even the company can see your chats.

The building blocks are familiar: public-key methods let two devices agree on
keys without a shared secret up front, and symmetric encryption protects each
message. The important honesty here is the **tradeoff**: E2EE hides *content*,
but **metadata still leaks** — who you talked to, when, how often, and how much.
Encryption of the message body doesn't hide the envelope, and that pattern of
contacts and timing can reveal a great deal on its own.

## Encrypted radio

The same ideas leave the internet and go onto the airwaves. Trunked and digital
radio systems can **encrypt voice** so that only radios holding the right key
hear the conversation — everyone else hears noise. The key is **symmetric**
(all the radios in a talkgroup share it), and the hard problem is getting that
key to every radio and changing it safely, which is handled by a
**key-management** process — often **OTAR**, over-the-air rekeying, so keys can
be updated without collecting every radio. See
[encryption & authentication](/learn/digital-trunking/encryption-and-authentication/),
[OTAR & key management](/learn/digital-trunking/otar-key-management/), and the
radio-side view in [encryption](/learn/rf-sdr/encryption/).

For a monitoring tool the boundary is sharp: a scanner can **detect that a
transmission is encrypted** and tell you which system it belongs to, but it
**cannot recover the content** without the key. That's not a limitation to work
around — it's the encryption doing exactly its job, by design.

## Key management is where crypto lives or dies

Notice what every section above had in common: the algorithm was never the weak
point. **Strong algorithms don't help if the keys are mishandled.** A perfect
cipher around a key that's emailed in plaintext, reused for years, or left on a
stolen laptop protects nothing.

So the real work of applied cryptography is **key management**:

- **Protect keys** — store them where attackers can't reach them, and never send
  them over the channel they're meant to secure.
- **Rotate keys** — change them periodically and after any suspected exposure, so
  one leaked key doesn't unlock everything forever. (You'll go deeper in
  secrets management later in the path.)
- **Retire weak protocols** — keep software up to date so old, broken versions of
  TLS and other schemes are turned off before someone exploits them.

Get the keys right and ordinary, well-tested algorithms are more than enough.
Get the keys wrong and the strongest cipher in the world won't save you.

<div class="knowledge-check" data-quiz data-correct-msg="Right — they all use cryptography the same way: symmetric encryption for the data, with keys exchanged or managed via public-key methods." markdown="0">
  <p class="knowledge-check__q">Quick check: what do HTTPS, SSH, and encrypted radio fundamentally share?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">They all run over the public internet</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">They use cryptography — symmetric encryption for the data, usually with keys exchanged or managed via public-key methods</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">They all use the same fixed password</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The cryptography you learned is everywhere: **TLS** for the web, **SSH** for
  remote access, **VPNs**, **disk encryption**, **E2EE** messaging, and
  encrypted radio.
- Nearly all of it is a **hybrid** — a public-key exchange to agree on a key,
  then fast **symmetric** encryption for the data.
- **In transit** protects data crossing a network; **at rest** protects data on
  a device that could be lost or stolen.
- **End-to-end** encryption keeps content readable only to the endpoints, but
  **metadata still leaks**.
- **Encrypted radio** uses shared symmetric keys distributed by key management or
  OTAR; a scanner can see that traffic is encrypted but can't read it.
- **Key management** — protecting, rotating, and retiring keys — is the part that
  decides whether any of it actually holds.

Next up: Module 3 turns to identity and access — [authentication basics](/learn/cybersecurity/authentication-basics/)
