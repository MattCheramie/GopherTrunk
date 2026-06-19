---
slug: birth-of-trunking
title: "The birth of trunking: sharing channels"
description: How trunked radio was born — a central controller assigns a free channel per call and reclaims it, packing many groups onto a few frequencies, and why this breaks old-style scanning.
keywords: birth of trunking, trunked radio history, control channel, channel sharing, conventional radio, Motorola trunking, GE trunking, channel assignment, trunking explained, scanner
level: beginner
status: full
prereq:
  - why-radio-went-digital
faq:
  - q: What is trunking in radio?
    a: Trunking is automatic channel sharing. Instead of each group owning a permanent frequency, a central controller keeps a small pool of channels and hands out a free one for the duration of each call, then reclaims it. Because real conversations are short and bursty, a few shared channels can serve far more groups than fixed assignments ever could.
  - q: How is trunking different from conventional radio?
    a: In conventional radio every group has a fixed frequency it always uses, so you scan by listening to that frequency. In trunked radio the frequency for a given call is chosen on the fly by a controller, so there is no permanent home channel. You have to read the control channel to know where each conversation went.
  - q: Why does trunking break old-style scanning?
    a: An old scanner assumes a group lives on one frequency, so it parks there and listens. On a trunked system the same group lands on a different voice channel almost every call, while other groups reuse the channel it just vacated. Parked on one frequency you hear a meaningless jumble. You have to follow the control channel instead.
  - q: When did trunking appear?
    a: Trunked land-mobile systems emerged in the 1970s and 1980s as spectrum filled up, with Motorola and GE among the pioneers. The earliest systems carried analog voice but used digital signalling to assign channels, the same core idea that every modern digital trunked system still uses.
gophertrunk_links:
  - title: CC Activity
    url: /cc-activity.html
    note: watch the controller hand out channels in real time.
  - title: Hunt (discover systems)
    url: /hunt.html
    note: find the control channel that runs a trunked system.
---

# The birth of trunking: sharing channels

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Conventional** radio gives each group its own permanent frequency — wasteful, because
most channels sit **idle** most of the time. **Trunking** fixes that with a **central
controller** that assigns a **free channel per call** and **reclaims** it when the call
ends. A dedicated **control channel** coordinates the dance, telling each radio where to
go. Trunking emerged in the **1970s–80s** with **Motorola** and **GE** among the pioneers.
Because the frequency for a call is chosen on the fly, trunking **breaks old-style
scanning** — you must follow the control channel, not a fixed frequency.
</div>

The last lesson listed *more users per slice of spectrum* as a digital promise. Trunking
is the single biggest idea that delivers it — and, crucially, it predates digital voice.
It's a way of *organising* channels that's so effective every modern digital standard
builds on it.

## The waste in conventional radio

Start with how radio worked before trunking: **conventional**. Each group — Dispatch,
Fire Ground, Public Works — owns a permanent frequency. It's simple and dependable. It's
also enormously wasteful, because radio traffic is **bursty**. A dispatcher might transmit
for ten seconds, then nothing for two minutes. All that time, the group's dedicated
frequency sits silent, *reserved* and unusable by anyone else, while a neighbouring busy
channel has users waiting in line. Multiply that across forty groups in a county and you
have most of your spectrum idle most of the time, yet still "full" because every channel
is assigned to someone.

## The trunking idea: a controller and a pool

Trunking pools the channels and shares them on demand. A **central controller** — a
computer at the system's heart — keeps a small set of frequencies and a rule: when a radio
wants to talk, find a **free channel**, assign it for *just that call*, and **reclaim** it
the instant the call ends, back into the pool for whoever needs it next.

The classic analogy is a **bank**. The old conventional way is one teller per customer,
each with their own roped-off line — most tellers idle while one line backs up. Trunking is
the modern way: one queue feeds *all* the tellers, and you're routed to whichever opens up.
Far fewer tellers serve far more customers with almost no waiting, because nobody is locked
to a specific window. Channels are the tellers; calls are the customers.

## The control channel makes it work

The coordination has to happen somewhere, and that somewhere is the **control channel** —
one frequency dedicated to carrying *data*, never voice. It runs constantly, and it's the
hinge the whole system turns on:

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 200" role="img" aria-label="A central controller connected to a control channel at top and a pool of voice channels below; the controller assigns a free voice channel to a call and reclaims it afterward." xmlns="http://www.w3.org/2000/svg">
  <rect x="200" y="14" width="140" height="34" rx="6" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.5"/>
  <text x="270" y="35" text-anchor="middle" font-size="12" fill="currentColor" font-weight="600">Central controller</text>
  <rect x="40" y="74" width="460" height="30" rx="5" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.3"/>
  <text x="270" y="93" text-anchor="middle" font-size="11" fill="currentColor">Control channel — data: “Group A, take channel 2”</text>
  <line x1="270" y1="48" x2="270" y2="74" stroke="currentColor" stroke-width="1.4"/>
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <rect x="40" y="138" width="100" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
    <text x="90" y="159">Voice 1 (free)</text>
    <rect x="155" y="138" width="100" height="34" rx="5" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/>
    <text x="205" y="155">Voice 2</text>
    <text x="205" y="167" font-size="8">Group A active</text>
    <rect x="270" y="138" width="100" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
    <text x="320" y="159">Voice 3 (free)</text>
    <rect x="385" y="138" width="100" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
    <text x="435" y="159">Voice 4 (free)</text>
  </g>
  <line x1="205" y1="104" x2="205" y2="136" stroke="currentColor" stroke-width="1.3" stroke-dasharray="4 3"/>
  <text x="270" y="192" text-anchor="middle" font-size="10" fill="currentColor">Assigned for one call, then released back to the pool.</text>
</svg>
<figcaption>A central controller listens for requests, assigns a free voice channel via the control channel, and reclaims it when the call ends.</figcaption>
</figure>

A radio keys up and sends a request on the control channel. The controller finds a free
voice channel and broadcasts the assignment — "Group A, take channel 2." Every radio in
Group A hears that data message and retunes to channel 2 to listen. When the call ends, the
channel goes back in the pool. The next call from Group A might land somewhere else
entirely. The control-channel data rides on the same kinds of
[digital modulation](/learn/rf-sdr/digital-modulation/) you met in the RF path.

## A short history

Trunking grew up as spectrum pressure became acute in the **1970s and 1980s**. **Motorola**
and **GE** were among the pioneers building commercial trunked systems, and the early ones
carried **analog voice** with **digital control** — the voice was old-fashioned FM, but the
channel assignments were already computer-managed data. That hybrid era is the subject of
the next lesson; for now, the point is that the organising idea — controller, pool,
per-call assignment — arrived well before digital voice and outlasted it.

## Why this breaks old-style scanning

Here's the consequence that matters for monitoring. An old scanner assumes a group lives on
*one* frequency, so it parks there and waits. On a trunked system that assumption is false:
the same group hops to a different voice channel nearly every call, while other groups reuse
the channel it just left. Park on one frequency and you hear fragments of unrelated
conversations — a jumble. To follow a trunked system you have to read the **control
channel** and let *it* tell you where each call went. That's exactly why
[GopherTrunk's RF-path primer](/learn/rf-sdr/what-is-trunking/) is short — it introduces the
idea — while *this* path goes much deeper into the signalling and the systems built on it.

<div class="knowledge-check" data-quiz data-correct-msg="Correct — the controller assigns a free channel per call and takes it back afterward." markdown="0">
  <p class="knowledge-check__q">Quick check: what does a trunking controller do with a voice channel when a call ends?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Keeps it permanently assigned to that group</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Releases it back into the shared pool</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Shuts the channel off until the next day</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Conventional** radio wastes spectrum: each group's permanent frequency sits idle most of the time.
- **Trunking** pools channels and a **central controller** assigns a **free one per call**, then reclaims it.
- A dedicated **control channel** carries the data that coordinates every assignment.
- Trunking emerged in the **1970s–80s**, with **Motorola** and **GE** among the pioneers, first carrying analog voice with digital control.
- Because frequencies change call to call, trunking **breaks old-style scanning** — you follow the control channel.

We've talked about why trunking matters at the level of one group; for the deeper
contrast, see [Conventional vs trunked](/learn/digital-trunking/conventional-vs-trunked/).
Next, we'll meet the first big real-world trunked systems in
[The analog trunking era](/learn/digital-trunking/analog-trunking-era/).
