---
slug: trunking-flavors
title: "Trunking flavors: dedicated vs. distributed, message vs. transmission"
description: The two axes that classify trunked systems — dedicated vs. distributed control channels, and message vs. transmission trunking — and how each choice changes what you monitor and how.
keywords: dedicated control channel, distributed control, LTR, Passport, message trunking, transmission trunking, hang time, P25, DMR Tier III, EDACS, SmartNet, trunking types
level: intermediate
status: full
prereq:
  - control-channel-signaling
  - conventional-vs-trunked
faq:
  - q: What is the difference between dedicated and distributed control?
    a: A dedicated control channel is one frequency reserved full-time to carry trunking data, used by P25, DMR Tier III, EDACS, and SmartNet. Distributed control instead spreads the trunking data across every repeater, with no separate control frequency, as in LTR and Passport. Dedicated systems give a single channel to lock onto; distributed systems require watching the sub-audible data on the active repeaters.
  - q: What is message trunking?
    a: Message trunking holds a voice channel for an entire conversation, including the pauses between transmissions, releasing it only after a hang-time once the exchange truly ends. It keeps a back-and-forth on one channel, which is simpler to follow but ties up a channel slightly longer.
  - q: What is transmission trunking?
    a: Transmission trunking releases the voice channel after each individual transmission, so the next over in the same conversation may be granted a different channel. It packs more conversations into the pool but means a single exchange can hop channels, which the control channel announces with each new grant.
  - q: How do these choices affect monitoring?
    a: With a dedicated control channel you lock one frequency and read every grant. With distributed control there's no single channel, so a tracker watches the data riding on the repeaters themselves. Transmission trunking makes individual conversations hop more, so following relies even more on reading every grant in real time.
gophertrunk_links:
  - title: CC Activity
    url: /cc-activity.html
    note: see how often grants appear, which reveals message vs. transmission trunking behavior.
  - title: Status
    url: /status.html
    note: check which trunking types GopherTrunk decodes.
---

# Trunking flavors: dedicated vs. distributed, message vs. transmission

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Trunked systems split along **two independent axes**. First, *where the control data
lives*: a **dedicated control channel** (P25, DMR Tier III, EDACS, SmartNet) reserves one
frequency for data, while **distributed control** (LTR, Passport) rides the data on each
repeater with **no separate control channel**. Second, *when the channel is freed*:
**message trunking** holds a channel for a whole conversation (pauses included);
**transmission trunking** releases it after *each transmission*, so a single exchange can
hop channels. These choices change **what you lock onto** and **how a call moves**.

</div>

By now you can read a [control channel](control-channel-signaling/) and follow a
[call](anatomy-of-a-call/). But not all trunked systems are built the same way. Two simple
axes classify nearly all of them, and together they explain a lot about how a given system
behaves on the air — and how you monitor it.

## Axis 1 — dedicated vs. distributed control

The first question is *where the trunking data lives*.

A **dedicated control channel** sets aside **one frequency** to carry signaling full-time,
never voice. This is the model you've studied so far, and it's used by **P25**, **DMR Tier
III**, **EDACS**, and Motorola **SmartNet/Type II**. The appeal for a monitor is obvious:
there's a *single channel to lock onto*, and it announces everything.

**Distributed control** takes the opposite approach: there is **no separate control
frequency**. Instead, the trunking data is spread across **every repeater** in the system,
typically as a low-rate sub-audible data stream alongside (or between) the voice. **LTR**
(Logic Trunked Radio) and **Passport** work this way. Each repeater advertises its own
status, and radios piece the system together from whichever repeaters they can hear.

For monitoring, the difference is real: a dedicated system gives you one obvious target,
whereas a distributed system has no control channel to park on — a tracker must watch the
data riding on the *active repeaters* to know what's happening.

## Axis 2 — message vs. transmission trunking

The second question is *when the voice channel is released*.

In **message trunking**, the system holds the granted voice channel for an **entire
conversation** — through the natural pauses between overs — releasing it only after a
**hang-time** once the exchange truly ends. A back-and-forth stays on one channel from
start to finish. This is simpler to follow: one grant, one channel, until release.

In **transmission trunking**, the channel is released after **each individual
transmission**. When the other party replies, the system grants a channel again — possibly
a *different* one. This squeezes more conversations into the pool, since channels aren't
tied up during pauses, but it means a single exchange can **hop channels**, with the control
channel issuing a fresh grant for each over.

<figure class="figure" markdown="0">
<svg viewBox="0 0 560 170" role="img" aria-label="Two timelines comparing message trunking, which holds one channel across pauses, with transmission trunking, which releases the channel after each over and may grant a different one." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor">
    <text x="30" y="30" font-weight="600">Message trunking</text>
    <rect x="30" y="40" width="80" height="20" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="0.8"/><text x="70" y="54" text-anchor="middle" font-size="8">over 1</text>
    <rect x="110" y="40" width="40" height="20" fill="none" stroke="currentColor" stroke-width="0.6" stroke-dasharray="3 2"/><text x="130" y="54" text-anchor="middle" font-size="7">pause</text>
    <rect x="150" y="40" width="80" height="20" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="0.8"/><text x="190" y="54" text-anchor="middle" font-size="8">over 2</text>
    <text x="250" y="54" font-size="8">all on channel 3</text>

    <text x="30" y="100" font-weight="600">Transmission trunking</text>
    <rect x="30" y="110" width="80" height="20" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="0.8"/><text x="70" y="124" text-anchor="middle" font-size="8">over 1 (ch 3)</text>
    <rect x="110" y="110" width="40" height="20" fill="none" stroke="currentColor" stroke-width="0.6"/><text x="130" y="124" text-anchor="middle" font-size="7">freed</text>
    <rect x="150" y="110" width="80" height="20" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="0.8"/><text x="190" y="124" text-anchor="middle" font-size="8">over 2 (ch 7)</text>
    <text x="250" y="124" font-size="8">re-granted, may move</text>
  </g>
</svg>
<figcaption>Message trunking keeps a conversation on one channel through its pauses; transmission trunking releases the channel after each over, so the next over may be granted a different channel.</figcaption>
</figure>

## The two axes combined

The axes are independent, so a system has one choice on each. The matrix below shows the
combination and what it means for monitoring:

| | Dedicated control channel | Distributed control |
|---|---|---|
| **Message trunking** | One CC to lock; calls stay put on a channel (e.g. many P25/SmartNet configs) | No CC; data on repeaters, conversation stays on one repeater (e.g. LTR) |
| **Transmission trunking** | One CC to lock; follow each grant as overs may hop channels | No CC; watch repeater data; overs may move between repeaters |

Practically: **dedicated** systems give you a single control channel to find and lock —
most of the digital systems in this path. **Distributed** systems make you watch the
repeaters themselves. And wherever **transmission trunking** is in play, following a
conversation leans even harder on reading *every* grant the moment it appears, because the
channel keeps moving. Watching grant frequency in
**[CC Activity](/cc-activity.html)** is a quick way to sense which behavior a system shows.

<div class="knowledge-check" data-quiz data-correct-msg="Right — distributed control has no separate control channel; the data rides on each repeater." markdown="0">
  <p class="knowledge-check__q">Quick check: which describes a system like LTR with distributed control?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">One reserved frequency carries all trunking data</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">No separate control channel — data rides on each repeater</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It cannot be trunked at all</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Dedicated control** reserves one frequency for data (P25, DMR Tier III, EDACS,
  SmartNet) — one channel to lock.
- **Distributed control** (LTR, Passport) spreads data across repeaters with **no separate
  control channel**.
- **Message trunking** holds a channel for the whole conversation, pauses included.
- **Transmission trunking** releases the channel after each over, so an exchange can **hop
  channels**.
- The axes are independent, and each choice changes **what you lock onto** and **how a call
  moves**.

Next, we scale up to real deployments: [sites, simulcast and
roaming](sites-simulcast-roaming/) in multi-site systems.
