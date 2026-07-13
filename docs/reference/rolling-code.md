---
slug: rolling-code
title: Rolling code (hopping code)
entry_type: algorithm
category: cryptography
description: "A rolling code is an anti-replay scheme for remote keyless entry in which each button press sends a new, non-repeating encrypted counter value, so a captured transmission cannot be replayed."
keywords: rolling code, hopping code, code hopping, KeeLoq, remote keyless entry, RKE, garage door opener, replay attack, rolljam, synchronization counter, LFSR, block cipher, key fob, anti-replay
aka: [rolling code, hopping code, code-hopping]
autolink: true
infobox:
  - { label: Type, value: Anti-replay authentication }
  - { label: Used by, value: Car/garage remotes (RKE) }
  - { label: Core, value: Encrypted incrementing counter }
see_also: [remote-keyless-entry, linear-feedback-shift-register, block-cipher, keystream, cryptographic-key, encryption]
cite_urls:
  - https://en.wikipedia.org/wiki/Rolling_code
  - https://en.wikipedia.org/wiki/KeeLoq
---

**A rolling code** (or *hopping code*) is the scheme that stops a captured
[remote keyless entry](/reference/remote-keyless-entry/) transmission from being
replayed, by making every button press send a different, non-repeating value.[^wiki]
Instead of a fixed secret that anyone who records one press could re-transmit, the
fob encrypts an internal counter that increments on each use, so the code visibly
"rolls" forward and a receiver rejects any value it has already seen. It is the
security core of the vast majority of car and garage-door remotes.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 116" role="img" aria-label="A fob increments a counter, encrypts it under a shared key into a hopping code, and transmits it; the receiver decrypts and accepts the code only if the counter is ahead of the last one it saw within a forward window." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="60" y="16">fob</text>
    <rect x="20" y="24" width="80" height="24" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="60" y="39">counter n → n+1</text>
    <rect x="20" y="58" width="80" height="24" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="60" y="73">encrypt(key)</text>
    <line x1="60" y1="48" x2="60" y2="58" stroke="currentColor" marker-end="url(#rcar)"/>
    <line x1="100" y1="70" x2="180" y2="70" stroke="currentColor" stroke-dasharray="3 3" marker-end="url(#rcar)"/><text x="140" y="62">hopping code</text>
    <rect x="182" y="58" width="86" height="24" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="225" y="73">decrypt(key)</text>
    <line x1="268" y1="70" x2="300" y2="70" stroke="currentColor" marker-end="url(#rcar)"/>
    <rect x="302" y="46" width="130" height="48" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <text x="367" y="60">counter ahead of</text><text x="367" y="72">last, within window?</text><text x="367" y="86">accept &amp; resync</text>
    <text x="367" y="16">receiver</text>
  </g>
  <defs><marker id="rcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Each press encrypts an incremented counter; the receiver accepts only counters ahead of the last it saw, so a recorded code will not replay.</figcaption>
</figure>

## How it works

The fob and the receiver share a secret key and each keep a synchronization
**counter**. On every button press the fob increments its counter, encrypts that
counter (plus a fixed identifier and button bits) under the shared key, and
transmits the result — a value that looks random and differs from the last. The
receiver decrypts the incoming code, checks that the recovered counter is *greater*
than the last counter it accepted, and if so opens and advances its own stored
counter to match. A replayed old capture carries a stale counter the receiver has
already passed, so it is rejected.

Because a user may press the fob out of range of the receiver, the counter can drift
ahead of what the receiver has seen. Systems therefore accept any counter within a
**forward window** (typically a few hundred codes ahead) and resynchronize to it,
with a larger tolerance requiring two consecutive in-sequence presses. This window
is what makes the scheme usable in the real world — and, as attacks show, is also a
weak point.

## Variants

The dominant implementation is **KeeLoq**, a lightweight proprietary
[block cipher](/reference/block-cipher/) with a 32-bit block and 64-bit key, built
around a nonlinear-feedback shift register run for 528 rounds; it encrypts the
counter to produce the hopping value.[^keeloq] Simpler and older remotes used a
[linear-feedback shift register](/reference/linear-feedback-shift-register/) to
sequence codes, but a bare LFSR is linear and its sequence can be predicted from a
few observed codes, so it offers replay resistance without real cryptographic
strength. Higher-end automotive systems have since moved to AES-based rolling codes
and to two-way challenge–response, which resist the counter-window attacks below.

## In practice — known weaknesses

Rolling codes defeat naive replay but not more active attacks. The well-known
**RollJam** attack jams the receiver while recording a valid press, so the code is
never consumed by the car; the attacker replays the captured-but-unused code later.
Because KeeLoq's cipher was reverse-engineered, researchers also demonstrated
key-recovery and cloning attacks against fobs whose manufacturer key was weakly
derived. These are the reasons the field is migrating toward authenticated,
challenge–response, and AES-based designs. None of this is decryption of *content*
— an RKE frame carries no message, only an authenticator — so the attacks target
the protocol's freshness guarantee rather than confidentiality.

## Relevance to SDR

Rolling-code remotes are a classic short-range RF signal a software-defined radio
can observe: they transmit brief OOK/FSK bursts in the 315 MHz and 433 MHz ISM
bands, and their structure (preamble, fixed serial number, hopping code field) is
easy to demodulate and log. What an SDR *cannot* do without the fob's key is predict
or forge the next valid code, and replaying a recorded one fails by design against
the counter check. GopherTrunk is a trunked-radio scanner and does not target RKE
devices, so this entry is background on how the wider RF world uses lightweight
cryptography: it illustrates authentication-by-counter rather than the
[keystream](/reference/keystream/)-based voice encryption GopherTrunk actually
encounters on P25 and DMR.

## Sources

[^wiki]: [Rolling code](https://en.wikipedia.org/wiki/Rolling_code) — Wikipedia, for the incrementing-counter model, the resynchronization window, and the RollJam replay-with-jamming attack.
[^keeloq]: [KeeLoq](https://en.wikipedia.org/wiki/KeeLoq) — Wikipedia, for the KeeLoq block cipher's 32-bit block, 64-bit key, NLFSR round structure, and reverse-engineering/cloning attacks.
