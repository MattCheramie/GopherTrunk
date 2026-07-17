---
slug: rf-and-radio-security
title: Wireless & RF security
description: Why radio is a broadcast medium anyone can receive, how Wi-Fi and radio encryption protect content but not metadata, the threats that exist on the air — and above all the legal and ethical boundaries that must come first.
keywords: wireless security, RF security, Wi-Fi WPA3, radio encryption, jamming, replay attack, GPS spoofing, monitoring legality, metadata, OTAR
level: intermediate
status: full
prereq:
  - crypto-in-practice
faq:
  - q: Can someone hear my wireless or radio traffic just by having a receiver?
    a: "Yes — that's the nature of radio. RF is a broadcast medium, so anyone within range with a suitable receiver can pick up the signal. That's why confidentiality on the air depends on encryption of the content, not on any assumption that the transmission is private. Whether you may lawfully receive, record, or act on a given transmission is a separate question governed by your local laws."
  - q: If radio voice is encrypted, can a scanner still see anything?
    a: "It can see that traffic exists and often quite a lot about it — that a transmission happened, when, how often, and sometimes system, unit, or talkgroup identifiers — but not the spoken content. That leftover pattern is metadata, and it leaks even when the payload is protected. A receiver can detect that a call is encrypted; recovering the content without the key is not possible by design, and attempting to defeat encryption is generally illegal."
  - q: Is it legal to monitor wireless and radio signals?
    a: "It depends entirely on where you are. Laws on what you may receive, record, decrypt, or transmit vary by country and region and are strict about private communications. Passively receiving is not the same as unlawfully intercepting, and defeating encryption is commonly prohibited outright. Always check the rules for your own jurisdiction first. This lesson is educational information, not legal advice."
  - q: What's the single most important habit for RF security?
    a: "Operate within the law and with respect for privacy. The technology makes it easy to receive far more than you should touch, so the discipline is knowing where the legal and ethical lines are and staying well inside them. On the defensive side, use strong modern encryption (WPA3, up-to-date radio key management), treat open networks as untrusted, and remember that encryption hides content, not the fact that you are transmitting."
gophertrunk_links:
  - title: DMR encryption
    url: /dmr-encryption.html
    note: how GopherTrunk reports that radio traffic is encrypted.
---

# Wireless & RF security

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The airwaves are a shared, open medium** — anyone with a receiver can hear
radio, so you can't rely on the medium for privacy. **Encryption protects
content, not metadata**: even a protected call still reveals that it happened,
when, and often who was involved. And **the law and ethics come first** — much
of what's technically receivable is legally or ethically off-limits. Security on
the air rests on strong encryption plus responsible, lawful use, built on the
same tools from
[cryptography in practice](/learn/cybersecurity/crypto-in-practice/).
</div>

Wireless is convenient precisely because it isn't a wire — but that same
freedom is its central security problem. This lesson looks at what that means
for defenders and, most of all, at the boundaries that govern how any of it may
be used.

## Radio is a broadcast medium

A wire carries a signal to a specific place; you can, at least in principle,
control who is plugged into it. Radio does the opposite. RF spreads outward in
every direction, and **you cannot control who receives it**. Anyone in range
with the right receiver picks up the same energy you do.

That single fact reshapes security. Confidentiality on the air cannot come from
the medium — it can only come from **encryption of the content**. This is
exactly the
[confidentiality](/learn/cybersecurity/cia-triad/) goal of the CIA triad, and
on radio it is the encryption doing the work, never the assumption that "nobody
is listening." This is the world a scanner like GopherTrunk observes: a sea of
open transmissions plus some encrypted ones it can see but not read.

## Wi-Fi and wireless security

Wi-Fi is the wireless most people touch daily, and it shows the pattern
clearly. Modern Wi-Fi encrypts the traffic between your device and the access
point: **WPA2** and the newer **WPA3** scramble the payload so nearby receivers
can't read it. An **open** network — one with no passphrase — does none of
this; its traffic travels in the clear for anyone in range.

The defender's takeaways are simple:

- **Use WPA3** where your devices support it, WPA2 otherwise; never run an open
  network for anything you care about.
- **Use a strong, unique passphrase** — the encryption is only as good as the
  secret behind it.
- **Treat public Wi-Fi as untrusted.** You don't control the access point or
  who else is on it, so rely on end-to-end protection (HTTPS, and often a
  [VPN](/learn/networking/vpns/)) rather than the network itself. See also
  [staying safe online](/learn/cybersecurity/staying-safe-online/).

## Radio & trunked-system encryption

Public-safety and commercial two-way radio can encrypt voice as well. As you
saw in cryptography in practice, this uses **symmetric keys** — the same key to
encrypt and decrypt — because it's fast and well suited to a live audio stream.
The hard part is getting those keys to every radio securely, which is handled by
**key management**, often **over the air** (OTAR) so keys can be loaded and
rotated without touching each device.

- [Encryption & authentication](/learn/digital-trunking/encryption-and-authentication/)
  — how trunked systems protect voice and verify radios.
- [OTAR & key management](/learn/digital-trunking/otar-key-management/)
  — how keys are distributed and rotated.
- [Encryption & what you can decode](/learn/rf-sdr/encryption/)
  — what a receiver can and cannot recover.

The crucial point for security: a receiver can **detect that traffic is
encrypted** and identify the system, but it **cannot recover the content** —
that is the entire purpose of the encryption, and it holds by design.

## Metadata still leaks

Encryption protects the payload, not the existence of the conversation. Even
when every word is scrambled, patterns remain in the open:

- **That a transmission happened at all**, and **when**.
- **How often** activity occurs, and in what bursts.
- Sometimes **system, unit, or talkgroup identifiers** carried in the clear so
  the network can route the call.

This leftover is **metadata**, and analysing it is a real discipline. Defenders
and privacy-minded users should internalise the lesson: **encryption is not
invisibility**. Hiding the content of your traffic does not hide the fact that
you are transmitting, or the shape of when and how you do it.

## Threats on the air (conceptually)

Understanding RF security means knowing the categories of risk exist — so you
can reason about them defensively. Described only conceptually, they include:

- **Eavesdropping** — passively receiving traffic; the reason content
  encryption matters.
- **Jamming** — flooding a frequency to deny service, an attack on
  [availability](/learn/cybersecurity/cia-triad/) rather than secrecy.
- **Replay** — recording a legitimate transmission and sending it again later.
- **Spoofing** — impersonating a legitimate source, as with **GPS spoofing**
  that feeds a receiver false position or time.

These are named here so you understand the threat model, **not** how to carry
any of them out. Deliberately interfering with, forging, or transmitting on
systems you have no rights to is both harmful and, in nearly every jurisdiction,
illegal.

## The law and ethics come first

Above every technical detail sits the boundary that actually governs this
field. Laws on what you may **receive, record, decrypt, or transmit vary by
country and region**, and they are strict about **private communications**.

- **Receiving is not the same as intercepting unlawfully** — but which is which
  depends entirely on your local rules.
- **Defeating encryption is generally illegal**, and it's infeasible without the
  key anyway. Treat encrypted traffic as simply off-limits.
- **Operate only within the law and with respect for privacy.** Just because a
  signal reaches your antenna does not mean you may listen to, keep, or share
  it.

The responsible position is the standard one across this whole site: enjoy the
open systems, leave the closed ones closed, and check the rules first. See
[legal & ethical monitoring](/learn/rf-sdr/legal-ethical/) for how those rules
break down in practice. **This lesson is educational information, not legal
advice** — laws differ everywhere and change over time.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the content stays protected, but metadata about the call still leaks." markdown="0">
  <p class="knowledge-check__q">Quick check: even when a radio system's voice is encrypted, a monitor can usually still observe...</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The decrypted spoken content, just more slowly</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Metadata — that a transmission occurred, its timing, and often IDs — but not the content</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Nothing at all — encryption hides the entire transmission</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Radio is broadcast** — anyone in range can receive it, so confidentiality
  comes from encryption, never from the medium.
- **Wi-Fi**: use WPA3 and strong passphrases; treat open and public networks as
  untrusted.
- **Radio encryption** uses symmetric keys managed by key distribution and OTAR;
  a receiver can tell traffic is encrypted but cannot read it.
- **Metadata still leaks** — timing, frequency of use, and often IDs remain even
  when content is protected.
- **Threats** like eavesdropping, jamming, replay, and spoofing exist; know them
  to defend, not to perform them.
- **The law and ethics come first** — receive only what you may lawfully
  receive, never try to defeat encryption, and respect privacy.

Next up: [where to go next](/learn/cybersecurity/keep-learning-security/)
