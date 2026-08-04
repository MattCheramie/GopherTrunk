---
slug: when-decoding-fails
title: When decoding fails
description: The field troubleshooting guide for trunked scanning — no control-channel lock, garbled or silent voice, missed calls — with a symptom-first method for telling a signal problem from a setup problem so you fix the right thing.
keywords: trunked decode fails, no control channel lock, garbled voice, missed calls, silent calls encryption, signal vs setup problem, scanner troubleshooting, wrong system type, gain overload, field checklist
level: advanced
status: full
prereq:
  - multisite-and-roaming
faq:
  - q: My trunked system won't decode — where do I start?
    a: Start by separating a setup problem from a signal problem, because the fixes are completely different. Confirm the system type and control-channel frequency are right (setup), then check whether the signal is clean enough to decode — strong but not overloading, on frequency, not distorted by simulcast. Work it as a checklist rather than changing things at random.
  - q: Calls appear but play back silent — is that broken?
    a: Usually not. If the control channel decodes and grants show up but one talkgroup's audio is silent, that talkgroup is almost certainly encrypted, which no receiver can decode. Check whether other talkgroups on the system produce audio; if they do, your setup is fine and the silent one is simply encrypted.
  - q: How do I tell a signal problem from a setup problem?
    a: A setup problem is consistent and total — nothing ever locks, or the wrong decoder is selected — and it doesn't change with the weather or where you point the antenna. A signal problem varies — better on strong signals, worse in overload or simulcast zones, sensitive to gain and antenna. If moving the antenna or adjusting gain changes things, it's signal.
---

# When decoding fails

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
When a trunked system won't decode, **work a checklist and change one thing at a time**. The single
most useful question is whether you have a **setup problem** (wrong system type or control channel —
consistent and total) or a **signal problem** (weak, overloading, off-frequency, or simulcast-
distorted — varies with gain and antenna). The usual culprits: **no control-channel lock**,
**garbled voice**, **silent calls** (encryption — *not fixable*), and **missed calls**. Each has a
distinct symptom, so match what you see to its cause. The digital-radio version of this checklist is
[troubleshooting a decode](/learn/digital-trunking/troubleshooting-a-decode/).
</div>

Everything before this lesson assumed things work. This one is for when they don't — the control
channel won't lock, the voice is garbled, or calls slip past you. The temptation is to change
settings at random until something helps, which usually just moves the problem around. Instead, this
lesson gives you a **symptom-first method**: read what you're seeing, decide whether it's setup or
signal, and apply the one fix that matches.

## First, split setup from signal

Before touching anything, ask the diagnostic question that saves the most time: **is this a setup
problem or a signal problem?** They look different once you know to look:

- A **setup problem** is *consistent and total*. Nothing ever locks, or the wrong decoder is
  selected. It doesn't change with the weather, the time of day, or where you aim the antenna,
  because it's a configuration mismatch — usually the **system type** or the **control-channel
  frequency**.
- A **signal problem** *varies*. It's better on strong signals and worse when the signal is weak,
  overloading, off-frequency, or distorted by [simulcast](/learn/scanning/multisite-and-roaming/).
  If moving the antenna or adjusting gain changes what you get, it's signal.

Make this call first and you've halved the search: fix the config, or fix the reception.

## No control-channel lock

The most common failure is the control channel never locking — no chatter, no grants, nothing. Walk
it in order. First, **setup**: is the **system type** right (it selects the decoder), and is the
**control-channel frequency** correct and *currently active*? Many systems list alternates; the one
you entered may be idle — try the others. A voice frequency mistaken for the control channel gives a
carrier but no data.

If setup checks out, it's **signal**: too little **gain** buries the control channel in noise, too
much **overloads** the front end and sprays distortion, and a **frequency/PPM error** on a cheap
dongle lands the signal off-channel so it never quite locks. Because everything downstream depends on
the control channel, this is always the first thing to get right — the same lesson the
[following a call](/learn/scanning/following-a-call/) sequence drove home.

## Garbled or distorted voice

When the control channel locks and calls come through but the **voice is garbled**, the control
channel is healthy and the problem is on the **voice channel or the signal**. A strong meter with a
fuzzy, un-lockable signal in an area of overlapping towers is the classic **simulcast** signature
from the [multisite lesson](/learn/scanning/multisite-and-roaming/) — favour one transmitter with a
directional antenna and lower the gain, don't raise it. Otherwise, garbled voice usually tracks a
marginal signal: weak, fading, or overloaded. Improve the reception and the audio usually clears.

## Silent calls — the one you can't fix

A special case worth naming clearly: calls that **appear in the log with a talkgroup and radio ID but
play back silent**. This is almost always **encryption**, not a fault. Encrypted voice is scrambled
with a key only authorised radios hold, and **no receiver can recover it**. Confirm by checking
whether *other* talkgroups on the same system produce audio — if they do, your setup and signal are
both fine, and the silent one is simply encrypted. Chasing it is wasted effort; it's a fundamental
limit, not a bug.

## Missed calls

Sometimes everything decodes but you keep **missing calls** you expected to hear. This is usually not
a decode failure at all — it's your [scan-list](/learn/scanning/talkgroups-and-scan-lists/) setup. A
talkgroup you forgot to add, one you locked out, or the absence of a **priority** flag on the channel
that matters can all make calls slip past while you're on something else. On a
[multisite](/learn/scanning/multisite-and-roaming/) system, a "missed" call may simply be traffic on
a site you can't hear. Check the list and the site before you suspect the decode.

<div class="knowledge-check" data-quiz data-correct-msg="Right — if it varies with gain and antenna position, it's a signal problem; a setup problem is consistent and doesn't change when you move the antenna." markdown="0">
  <p class="knowledge-check__q">Quick check: moving your antenna and adjusting gain changes whether the system decodes. Setup problem or signal problem?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Setup — the system type or control channel is wrong</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Signal — something that responds to gain and antenna is a reception issue</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Neither — the system is encrypted</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Split setup from signal first**: setup problems are consistent and total; signal problems vary
  with gain and antenna.
- **No lock** is usually the **system type** or **control-channel frequency** (setup), or gain/PPM
  (signal) — try alternate control channels.
- **Garbled voice** with a locked control channel points at the signal — often **simulcast**
  distortion, which a directional antenna and *less* gain fix.
- **Silent calls** with visible grants are **encryption** — expected and **not fixable**; confirm by
  checking other talkgroups.
- **Missed calls** are usually a **scan-list** or **site** issue, not a decode failure.
- Change **one thing at a time** so you know which fix actually worked.

Next up: [Logging &amp; recording calls](/learn/scanning/logging-and-recording/).
