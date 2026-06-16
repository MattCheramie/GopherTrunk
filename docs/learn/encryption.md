---
slug: encryption
title: Encryption & what you can decode
description: Radio encryption explained for scanner users — why some talkgroups are silent even with a perfect signal, the difference between encryption and an unsupported protocol, how to recognise encrypted traffic in GopherTrunk, and what metadata you can still see.
keywords: radio encryption, encrypted talkgroup, P25 encryption, DMR encryption, AES DES radio, why is my scanner silent, decode encrypted radio, monitoring encryption
level: intermediate
status: full
prereq:
  - what-is-trunking
faq:
  - q: Why is a talkgroup silent even though my signal is perfect?
    a: The most common reason is encryption. If a talkgroup is encrypted, the voice data is scrambled with a key you don't have, so even a flawless decode produces no intelligible audio. A strong signal and a locked control channel can't help — encryption is a deliberate barrier, not a reception problem.
  - q: Can GopherTrunk decode encrypted voice?
    a: No. Encryption is designed to be unbreakable without the key, and GopherTrunk does not attempt to defeat it — nor should any tool you'd use. You can still see the call happening and its metadata, but the audio itself stays scrambled. Encrypted talkgroups are simply not listenable.
  - q: How do I know if traffic is encrypted versus just unsupported?
    a: An unsupported protocol means the software can't decode the signal at all. Encryption means the protocol decodes fine — you see the call, talkgroup, and radio ID on the control channel — but the voice payload is scrambled and plays as silence, noise, or a brief digital chirp. Seeing healthy call metadata with no usable audio points to encryption.
  - q: Is it legal to listen to encrypted radio?
    a: You generally cannot listen to encrypted radio at all without the key, so the question is usually moot. Separately, what you may legally receive varies by country and region, and attempting to defeat encryption is often specifically prohibited. Always follow the laws where you live — see the legal and ethical monitoring lesson.
gophertrunk_links:
  - title: DMR encryption
    url: /dmr-encryption.html
    note: how encryption appears in DMR specifically.
  - title: CC Activity
    url: /cc-activity.html
    note: encrypted calls still show as control-channel activity.
---

# Encryption & what you can decode

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
If a talkgroup is **silent despite a perfect signal**, the usual cause is
**encryption** — the voice payload is scrambled with a key you don't have, so even a
flawless [decode](/learn/antenna-to-audio/) yields no audio. This is different from an
**unsupported protocol** (where nothing decodes at all): with encryption the
[control channel](/learn/what-is-trunking/) decodes fine, so you still see the **call,
talkgroup, and radio ID** — just no listenable voice. GopherTrunk **does not** break
encryption, and neither should any tool. Knowing the difference saves hours of
fruitless troubleshooting.
</div>

This lesson answers the question every new digital-monitoring user eventually asks: *my
signal is great, the system is locked, so why is this talkgroup silent?* Usually the
answer isn't a reception problem at all.

## Encryption vs. an unsupported protocol

These two produce different symptoms, and telling them apart is the key skill:

| | Unsupported protocol | Encryption |
|--|----------------------|------------|
| Control channel | May not decode | **Decodes fine** |
| Call metadata (TG, ID) | Often missing | **Visible** |
| Voice audio | Nothing / garbage | **Silent / noise** |
| Cause | Software can't read this signal | Payload deliberately scrambled |
| Fixable? | Maybe (add support, settings) | **No — by design** |

So if GopherTrunk is happily showing you the [control-channel
activity](/cc-activity.html), the talkgroup, and the transmitting [radio
ID](/radio-ids.html) — but the call plays as silence — you're almost certainly looking
at **encryption**, not a decode failure.

## Why encrypted voice can't be decoded

Once the system has the voice digitised as [vocoder](/learn/vocoders/) frames, an
encrypted system **scrambles those bits with a cipher** (such as AES or, on older
systems, DES) using a secret **key**. Only radios holding the matching key can
unscramble them. Modern ciphers are designed so that *without* the key, recovering the
audio is computationally infeasible — that's the entire point.

GopherTrunk demodulates and decodes the signal perfectly; it simply hands you scrambled
payload because that's all that was transmitted. **It does not attempt to defeat
encryption**, which is both impractical and, in many places, specifically prohibited.

## How to recognise encryption in GopherTrunk

Signs you're dealing with encryption rather than a weak or unsupported signal:

- The **system locks** and other (unencrypted) talkgroups decode normally.
- The encrypted talkgroup's calls **appear** in the activity and history — with
  talkgroup and radio ID — but produce **no intelligible audio** (silence, noise, or a
  short digital chirp).
- Many protocols flag encryption in the signalling, so the call may be **marked
  encrypted**.
- It's **consistent** for that talkgroup regardless of signal strength — unlike the
  [cliff effect](/learn/digital-voice/), which varies with SNR.

The [DMR encryption](/dmr-encryption.html) page covers how this looks for DMR
specifically.

## What you can still see

Encryption hides the *voice*, not the *traffic*. The control channel is typically **not**
encrypted, so you can still observe a lot:

- **Which talkgroups are active** and how busy the system is.
- **Radio IDs** affiliating and transmitting.
- **Call patterns** and timing.

For many monitoring purposes — understanding a system's structure, activity levels, and
units — this metadata is valuable even when the audio is locked away.

## A note on legality and expectations

Set expectations realistically: **encrypted talkgroups are not listenable**, full stop.
Trying to circumvent encryption is often explicitly illegal and outside the spirit of
the hobby. What you may legally *receive* in the first place also varies by jurisdiction
— the [legal & ethical monitoring](/learn/legal-ethical/) lesson covers that. The healthy
mindset: enjoy the open systems and the metadata, and treat encrypted talkgroups as
simply off-limits.

<div class="knowledge-check" data-quiz data-correct-msg="Right — visible metadata but no audio is the signature of encryption." markdown="0">
  <p class="knowledge-check__q">Quick check: the system is locked, you see the talkgroup and radio ID, but the call is silent. Most likely cause?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Weak signal / cliff effect</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The talkgroup is encrypted</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The antenna is unplugged</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **silent talkgroup with a perfect signal** is usually **encryption**.
- Encryption ≠ unsupported protocol: with encryption the **control channel and metadata
  decode**, only the **voice** is scrambled.
- GopherTrunk **cannot and does not** break encryption — it's infeasible by design.
- You can still see **talkgroups, radio IDs, and activity**.
- Respect the **law and the hobby** — encrypted talkgroups are off-limits.

Next: the many non-trunked signals your SDR can also decode.
