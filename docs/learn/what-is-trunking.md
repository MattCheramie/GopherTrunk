---
slug: what-is-trunking
title: What is trunked radio?
description: Trunked radio explained for beginners — control channels, voice channels, talkgroups, and affiliation. Understand how systems like P25 and DMR share a few frequencies across a whole fleet, and why GopherTrunk follows the control channel.
keywords: trunked radio, trunking explained, control channel, voice channel, talkgroup, affiliation, P25 trunking, DMR trunking, what is a control channel, conventional vs trunked
level: intermediate
status: full
prereq:
  - radio-waves
  - digital-modulation
faq:
  - q: What is trunked radio?
    a: Trunked radio is a system where many user groups share a small pool of radio channels automatically. Instead of each group having its own permanent frequency, a computer assigns a free channel for the duration of each call, then releases it. A dedicated control channel coordinates everything, telling radios where to go for each conversation. This packs far more users onto the same spectrum than giving everyone a fixed frequency.
  - q: What is a control channel?
    a: A control channel is one frequency in a trunked system that continuously carries data rather than voice. It manages the system — handling radios registering, requesting calls, and being told which voice channel to switch to. To monitor a trunked system you decode the control channel first, because it's the map that tells you where every call goes.
  - q: What is a talkgroup?
    a: A talkgroup is a virtual channel — a label that identifies a group of users, like "Police Dispatch" or "Fire Ground 2." Members of a talkgroup hear each other regardless of which physical frequency the system assigns to a given call. In a trunked system you follow talkgroups, not frequencies, because the frequency changes call to call.
  - q: What is the difference between conventional and trunked radio?
    a: In conventional radio, each group has a fixed frequency it always uses. In trunked radio, frequencies are shared and assigned on demand by a control channel. Conventional is simple to scan — just listen to the frequency. Trunked needs you to decode the control channel to follow conversations as they hop between voice channels.
  - q: Why does GopherTrunk follow the control channel?
    a: Because the control channel is where the system announces every call and the voice channel it's assigned to. By decoding it in real time, GopherTrunk knows the instant a talkgroup starts a call and which frequency to tune a receiver to in order to capture the audio — then it follows the next call, and the next, across the whole system.
gophertrunk_links:
  - title: CC Activity
    url: /cc-activity.html
    note: watch the raw control-channel messages GopherTrunk decodes in real time.
  - title: Radio IDs
    url: /radio-ids.html
    note: see which radios are affiliating and transmitting on the system.
  - title: Hunt (discover systems)
    url: /hunt.html
    note: find a control channel when you don't know the system's frequencies yet.
---

# What is trunked radio?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Trunked radio** lets many user groups share a small pool of frequencies by
assigning a free channel to each call on demand, then reclaiming it. A dedicated
**control channel** carries data — not voice — and orchestrates the whole system,
announcing every call and the **voice channel** it's been assigned. Users belong to
**talkgroups** (virtual channels), so you follow *who's talking*, not a fixed
frequency. This is why **GopherTrunk decodes the control channel first**: it's the
map that tells the software where each call goes so it can follow them across the
system.
</div>

This is the idea GopherTrunk was built around. Once trunking clicks, the whole
design of the software — control channels, talkgroups, the activity panels — turns
from jargon into something obvious.

## What problem does trunking solve?

Radio spectrum is scarce and expensive. Imagine a county with 40 agencies — police,
fire, public works, transit, schools. Give each its own permanent frequency and
you've burned 40 channels, most sitting *silent* most of the time while a few are
congested. That's **conventional** radio: simple, but wasteful.

**Trunking** fixes the waste by sharing. The agencies pool, say, 10 frequencies, and
a central computer hands out a free one *per call*, just for that call's duration,
then takes it back. Because real conversations are short and bursty, 10 shared
channels can serve all 40 groups with almost no waiting. It's the same logic as a
bank with one queue feeding several tellers instead of a separate line per teller.

## How does a trunked call actually work?

The magic is coordination, and it happens on the **control channel** — one frequency
that carries a constant stream of *data* (never voice). Here's a single call,
step by step:

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 210" role="img" aria-label="A diagram showing a control channel at top sending assignment messages, and a pool of voice channels below being assigned to different talkgroups." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="20" width="460" height="40" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.5"/>
  <text x="270" y="38" text-anchor="middle" font-size="13" fill="currentColor" font-weight="600">Control channel (data only)</text>
  <text x="270" y="53" text-anchor="middle" font-size="10" fill="currentColor">“Talkgroup 101 → go to channel 3”</text>
  <g font-size="11" fill="currentColor" text-anchor="middle">
    <rect x="40" y="110" width="90" height="36" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="85" y="132">Voice 1</text>
    <rect x="150" y="110" width="90" height="36" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="195" y="132">Voice 2</text>
    <rect x="260" y="110" width="90" height="36" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.5"/>
    <text x="305" y="127">Voice 3</text>
    <text x="305" y="140" font-size="9">TG 101 active</text>
    <rect x="370" y="110" width="90" height="36" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="415" y="132">Voice 4</text>
  </g>
  <line x1="270" y1="60" x2="305" y2="108" stroke="currentColor" stroke-width="1.5" stroke-dasharray="4 3" marker-end="url(#tg)"/>
  <text x="270" y="190" text-anchor="middle" font-size="11" fill="currentColor">A free voice channel is assigned per call, then released.</text>
  <defs><marker id="tg" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The control channel assigns a free voice channel for each call. Follow the control channel and you know where every conversation is.</figcaption>
</figure>

1. A user keys up. Their radio sends a request on the **control channel**.
2. The system computer finds a **free voice channel** and broadcasts an *assignment*
   on the control channel: "Talkgroup 101, go to voice channel 3."
3. Every radio affiliated with talkgroup 101 hears that data message and instantly
   retunes to voice channel 3 to listen.
4. The call happens on channel 3. When it ends, the channel is released back into the
   pool for the next call — maybe a completely different talkgroup.

The next call from the same talkgroup might land on channel 7, then channel 2. The
*frequency is never fixed* — which is exactly why you can't scan a trunked system the
old-fashioned way.

### Watching the pool churn

To see why this defeats an old scanner, follow ten seconds on a busy system —
this is exactly the stream GopherTrunk reads on the control channel:

| Time | Control-channel message | What a follower does |
|------|-------------------------|----------------------|
| 0.0 s | TG 101 (Dispatch) → channel 3 | Tune a receiver to ch 3, record |
| 1.2 s | TG 250 (Fire) → channel 7 | Tune a *second* receiver to ch 7 |
| 3.8 s | TG 101 ends; ch 3 released | Free ch 3 back to the pool |
| 4.1 s | TG 101 (Dispatch) → channel 2 | The *same* talkgroup, a *new* channel |
| 5.5 s | TG 412 (Transit) → channel 3 | Reuse ch 3, now a different group |

An old scanner parked on "channel 3" would hear Dispatch, then silence, then Transit
— a meaningless jumble. A trunk-tracker following the *control channel* knows who's on
each channel at every instant, which is the whole point of decoding it first.

## What is a talkgroup?

Because the physical frequency keeps changing, trunked systems give users a stable
*virtual* identity: the **talkgroup**. A talkgroup is just a number with a label —
"Police Dispatch," "Fire Ground 2," "City Transit." Everyone in a talkgroup hears
each other's calls no matter which voice channel the system happens to assign.

So when you monitor a trunked system, **you follow talkgroups, not frequencies**.
In GopherTrunk you'll lock, prioritise, or mute *talkgroups* — and the software
handles the frequency-hopping underneath. Each transmitting radio also has its own
**[radio ID](/radio-ids.html)**, so you can see individual units, not just groups.

## What is affiliation?

**Affiliation** is how the system keeps track of who's listening to what. When a
radio is turned on or switches to a talkgroup, it *registers* (affiliates) with the
system over the control channel. That lets the system route calls efficiently and,
helpfully for monitoring, it means the control channel is a constant source of
information about which radios and talkgroups are active — visible in GopherTrunk's
**[Radio IDs](/radio-ids.html)** and **[CC Activity](/cc-activity.html)** panels.

## Why does GopherTrunk decode the control channel first?

Everything above runs through the control channel, so it is the **single most
important thing to decode**. By locking onto it and reading its data stream in real
time, GopherTrunk learns:

- the instant any talkgroup starts a call,
- which voice channel that call was assigned to,
- which radio ID is transmitting, and
- the system's identity and parameters.

With that map, GopherTrunk can point a receiver at the right voice channel at the
right moment, capture the audio, and immediately be ready for the next assignment —
following dozens of conversations as they scatter across the channel pool. Lose the
control channel and you're blind; that's why so much of operating GopherTrunk comes
down to getting a [clean, solid lock](/learn/tuning-with-scopes/) on it.

The control-channel data itself is sent using the [digital
modulation](/learn/digital-modulation/) you met earlier — for P25 and DMR, that's the 4FSK
whose symbols you can watch in the constellation and eye-diagram panels. Different
systems package this differently, which is the subject of [the protocol
landscape](/learn/protocol-landscape/).

<div class="knowledge-check" data-quiz data-correct-msg="Correct — you follow the talkgroup; the system moves it across voice channels." markdown="0">
  <p class="knowledge-check__q">Quick check: on a trunked system, what do you follow to hear a particular group of users?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A single fixed frequency</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Their talkgroup</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The control channel's audio</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Trunking** shares a small pool of frequencies across many groups by assigning a
  channel per call.
- The **control channel** carries data and orchestrates everything — it's the map.
- **Talkgroups** are virtual channels; you follow them, not frequencies.
- **Affiliation** registers radios so the system (and you) know who's active.
- **GopherTrunk decodes the control channel first** so it can follow every call.

Next, we'll survey the actual protocols — P25, DMR, NXDN and friends — and how to
tell them apart. Or see the whole chain assembled in [From antenna to
audio](/learn/antenna-to-audio/).
