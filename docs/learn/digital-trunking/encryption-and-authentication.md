---
slug: encryption-and-authentication
title: "Encryption & authentication"
description: How trunked systems encrypt voice — DES-OFB, AES-256, ADP/RC4 and proprietary schemes — why encryption is signaled in the headers so a decoder can see it, and authentication, stun and kill commands.
keywords: encryption, DES-OFB, AES-256, ADP, RC4, P25 encryption, ALGID, key ID, encryption flag, authentication, stun, kill, clear vs secure talkgroup, DMR encryption
level: intermediate
status: full
prereq:
  - control-channel-signaling
  - encryption
faq:
  - q: What encryption do trunked systems use?
    a: Common schemes include DES-OFB, AES-256, and ADP (a lighter RC4-based option), plus vendor-proprietary modes like Motorola DVP. AES-256 is the strong, standard choice for P25 public safety. Each is identified in the signaling by an algorithm ID, so a decoder can tell which one is in use even without the key.
  - q: Can GopherTrunk decode encrypted calls?
    a: No. Without the encryption key, the voice payload is unintelligible, and GopherTrunk does not attempt to break encryption. What it can do is read the headers and show that a call is encrypted, including which algorithm and key ID the call advertises, so you know why there's no audio.
  - q: How does a decoder know a call is encrypted?
    a: Encryption is signaled in the call's headers. A flag marks the call as encrypted, and fields like the algorithm ID (ALGID) and key ID name the scheme and key. Because these travel in the clear alongside the voice, a decoder can see and display the encryption status even though it cannot read the protected audio.
  - q: What are stun and kill commands?
    a: Stun and kill are system commands sent to a specific radio to disable it — stun temporarily, kill more permanently — typically when a radio is lost or stolen. They ride the control channel like other unit-addressed messages. They concern the radios on the system, not a monitor's ability to receive.
gophertrunk_links:
  - title: DMR encryption
    url: /dmr-encryption.html
    note: see how GopherTrunk surfaces DMR encryption status on a call.
  - title: CC Activity
    url: /cc-activity.html
    note: watch for the encryption flags and key IDs in the signaling.
---

# Encryption & authentication

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Trunked systems protect voice with encryption — **AES-256** (the strong standard),
**DES-OFB**, **ADP** (an RC4-based lighter option), and vendor-proprietary modes. Crucially,
encryption is **signaled in the headers**: an **encryption flag** plus an **algorithm ID
(ALGID)** and **key ID** ride in the clear, so a decoder can **see that a call is
encrypted** — and which scheme — even though it **cannot decode the audio** without the
key. Talkgroups can be **clear** or **secure**. Systems also support **authentication** and
**stun/kill** commands. GopherTrunk shows an **encrypted indicator**; it does **not** break
encryption.

</div>

Encryption is the wall a monitor inevitably meets. You did the [conceptual
groundwork](/learn/rf-sdr/encryption/) in the RF & SDR path; here we make it concrete for
trunking — *which* schemes appear, *how* the system tells you a call is protected, and what
a decoder can and can't do about it.

## What encrypts the voice

Encryption sits on the **voice payload** — the [vocoder](voice-to-bits-vocoders/) bits —
scrambling them so only a radio holding the matching key can recover audio. The schemes you
meet on trunked systems include:

| Scheme | Notes |
|--------|-------|
| AES-256 | The strong, standard choice for P25 public safety |
| DES-OFB | Older block cipher in output-feedback mode; still in use |
| ADP (RC4-based) | A lighter, lower-cost option; weaker than AES |
| Proprietary (e.g. Motorola DVP family) | Vendor-specific modes, including legacy designs |

The takeaway isn't the cryptographic detail — it's that **strong, modern encryption (AES)
is effectively unbreakable to a monitor**, and even the lighter schemes require the key. A
receiver without the key gets noise where the audio should be.

## Encryption is signaled in the clear

Here's the part that matters for decoding. The **encryption status travels in the call's
headers**, *not* hidden inside the encrypted payload. A call carries:

- an **encryption flag** marking it as protected,
- an **algorithm ID** (P25 calls this **ALGID**) naming the scheme, and
- a **key ID** indicating which key the call uses.

These fields are in the clear so that *receiving radios* know how to handle the call — and
that same visibility lets a **decoder see and display** the encryption status. So even
though the audio is locked, a monitor can tell you *this call is encrypted, with this
algorithm, under this key*. That's a genuinely useful thing to know: it distinguishes "I
can't hear it because it's encrypted" from "I can't hear it because my lock is bad."

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 170" role="img" aria-label="A diagram of a call frame showing a clear header carrying an encryption flag, algorithm ID, and key ID, followed by an encrypted voice payload that the decoder cannot read." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="50" width="200" height="50" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.5"/>
  <text x="140" y="42" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">Header (in the clear)</text>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="140" y="70">ENC flag · ALGID · key ID</text>
    <text x="140" y="88" font-size="8">decoder reads these</text>
  </g>
  <rect x="260" y="50" width="240" height="50" rx="5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-dasharray="5 3"/>
  <text x="380" y="42" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">Voice payload (encrypted)</text>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="380" y="78">unreadable without the key</text>
  </g>
  <text x="270" y="135" text-anchor="middle" font-size="11" fill="currentColor">The decoder can SEE the call is encrypted; it cannot HEAR the audio.</text>
</svg>
<figcaption>The header travels in the clear with an encryption flag, algorithm ID, and key ID, so a decoder can report exactly that a call is encrypted — while the voice payload stays locked without the key.</figcaption>
</figure>

## Clear vs. secure talkgroups

Encryption is usually applied per **talkgroup** (and can vary per call). A **clear**
talkgroup carries unencrypted audio you can decode normally; a **secure** talkgroup is
encrypted and yields no audio without the key. Many systems mix the two — some channels
clear, some secure — and you'll often find that monitoring a system means hearing the clear
talkgroups while the secure ones show only as *encrypted* in the log.

## Authentication, stun, and kill

Beyond scrambling voice, trunked systems manage the *radios themselves*. **Authentication**
lets the system verify a radio's identity before granting it service, so a cloned or rogue
radio can be rejected. And **stun**/**kill** commands — unit-addressed messages on the
control channel — can disable a radio remotely, temporarily (stun) or more permanently
(kill), typically when one is lost or stolen. These are operational features of the system;
they don't affect whether *you* can receive, but they're part of the signaling vocabulary
you may see.

## What GopherTrunk can and can't do

To be plain about it: GopherTrunk **shows an encrypted indicator** for protected calls,
reading the flag and identifiers from the headers — see how DMR encryption status surfaces
in **[DMR encryption](/dmr-encryption.html)**. It does **not** attempt to break encryption,
because modern schemes like AES-256 are designed to resist exactly that. Encrypted audio
stays encrypted; the honest, useful thing a decoder offers is *telling you so clearly*,
rather than leaving you to wonder why a strong, well-locked signal produces no voice.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the encryption flag and IDs are in the clear, so a decoder can see (not hear) an encrypted call." markdown="0">
  <p class="knowledge-check__q">Quick check: how can a decoder show that a call is encrypted without the key?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">By partially decrypting the audio</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The encryption flag and algorithm/key IDs travel in the clear in the header</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It can't — encryption is completely invisible</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Trunked voice is encrypted with **AES-256**, **DES-OFB**, **ADP (RC4)**, or proprietary
  schemes.
- Encryption is **signaled in the clear** — an **encryption flag**, **ALGID**, and **key
  ID** in the header.
- A decoder can therefore **see that a call is encrypted** without being able to **decode**
  it.
- Talkgroups can be **clear** or **secure**; systems also use **authentication** and
  **stun/kill**.
- GopherTrunk shows an **encrypted indicator** but does **not** break encryption.

Encryption raises an obvious question: how do all those radios get — and change — their
keys without a technician touching each one? Next up: [OTAR & key
management](/learn/digital-trunking/otar-key-management/).
