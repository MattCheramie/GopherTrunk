---
redirect_from: /learn/the-control-channel/
slug: the-control-channel
title: "The control channel: the system's heartbeat"
description: A dedicated control channel carries only data, continuously, and is the map of a whole trunked system — broadcasting identity, channel grants, and neighbors. Why you decode it first.
keywords: control channel, trunking control channel, channel grant, system identity, neighbor sites, distributed control, sub-audible signaling, LTR, P25 control channel, DMR control channel, GopherTrunk
level: intermediate
status: full
prereq:
  - tdma-vs-fdma
  - what-is-trunking
faq:
  - q: What is a control channel?
    a: A control channel is one frequency in a trunked system that carries a continuous stream of data rather than voice. It announces the system's identity and parameters, grants voice channels to calls as they happen, and lists neighboring sites. To follow a trunked system you decode the control channel first, because it tells you where every call goes.
  - q: Why do you decode the control channel before anything else?
    a: Because it is the map of the whole system. Without it you would have a list of frequencies but no way to know which one carries which call at any moment. The control channel announces each channel grant in real time, so decoding it is what lets a scanner follow conversations as they hop across voice channels.
  - q: Do all trunked systems have a dedicated control channel?
    a: No. Systems like P25, DMR Tier 3, and Motorola's networks use a dedicated control channel that does nothing but signaling. Older or simpler systems such as LTR use distributed or sub-audible control, embedding signaling within the voice channels themselves rather than dedicating a frequency to it.
  - q: What kinds of messages does a control channel carry?
    a: It carries system-identity and parameter broadcasts, registration and affiliation messages from radios, channel grants that assign a call to a voice frequency and slot, neighbor-site lists for roaming, and status or paging messages. A monitor reads the channel grants to follow calls and the identity broadcasts to recognize the system.
gophertrunk_links:
  - title: CC Activity
    url: /cc-activity.html
    note: watch the raw control-channel messages GopherTrunk decodes in real time.
  - title: Radio IDs
    url: /radio-ids.html
    note: see which radios are registering and transmitting on the system.
---

# The control channel: the system's heartbeat

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **control channel** is one frequency that carries only **data**, **continuously** —
never voice. It is the **map of the whole system**: it broadcasts the system's
**identity** and parameters, hands out **channel grants** that send each call to a
voice frequency (and slot), and lists **neighbor sites** for roaming. You **decode it
first** because without it you have frequencies but no idea which carries which call.
Not every system has a dedicated one — **LTR** uses distributed, sub-audible control
instead. This is why **GopherTrunk locks the control channel before it can follow a
single call**.
</div>

You've now seen the pieces of a digital trunked signal — vocoders, modulation, framing,
and the choice between FDMA and TDMA. This lesson ties Module 2 together by introducing
the one channel that **coordinates** all of it. If trunking has a single beating heart,
this is it. (The [RF & SDR what-is-trunking lesson](/learn/rf-sdr/what-is-trunking/)
covers the trunking idea broadly; here we focus on the control channel itself.)

## A channel that carries only data

On a trunked system, one frequency is set aside to carry a **constant stream of data**
and **no voice at all**. While voice channels sit idle between calls, the control
channel never stops talking — it is the always-on signaling bus that every radio on the
system listens to. That continuous data is what lets the system assign and reclaim
voice channels [on demand](/learn/digital-trunking/conventional-vs-trunked/), and what lets a monitor follow along.

Because it's always transmitting, the control channel is also the **easiest part of the
system to find and lock**: it's a steady digital carrier on a known frequency, not a
bursty voice channel that comes and goes. Find it once and the whole system opens up.

## The map of the whole system

The control channel earns the title "map" because it broadcasts everything you need to
understand and follow the system:

| Message type | What it tells you |
|--------------|-------------------|
| System identity / parameters | which system this is, its ID, and how to interpret its channels |
| Registration / affiliation | which radios are on the system and active |
| Channel grant | a call is starting — go to this voice frequency (and slot) |
| Neighbor-site list | the adjacent sites a radio can roam to |
| Status / paging | individual calls, alerts, and short data |

The **channel grant** is the one a monitor lives on: it's the message that says
*"talkgroup 101 is now on voice channel 3"* (and, on a [TDMA](/learn/digital-trunking/tdma-vs-fdma/) system,
*which slot*). Read those in real time and you always know where every conversation is.
The **identity** broadcasts let you recognise the system, and the **neighbor list** is
how multi-site roaming works, both topics later in the path.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 200" role="img" aria-label="A diagram with a control channel at top broadcasting a continuous data stream, and arrows pointing to system identity, channel grants, and a neighbor-site list, with a grant directing a call to one voice channel in a pool below." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="18" width="460" height="40" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.5"/>
  <text x="270" y="35" text-anchor="middle" font-size="13" fill="currentColor" font-weight="600">Control channel (data only, continuous)</text>
  <text x="270" y="51" text-anchor="middle" font-size="10" fill="currentColor">identity · grants · neighbors · affiliation</text>
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <rect x="40" y="100" width="90" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="85" y="121">Voice 1</text>
    <rect x="150" y="100" width="90" height="34" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.5"/>
    <text x="195" y="116">Voice 2</text>
    <text x="195" y="129" font-size="9">TG 101 active</text>
    <rect x="260" y="100" width="90" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="305" y="121">Voice 3</text>
    <rect x="370" y="100" width="90" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="415" y="121">Voice 4</text>
  </g>
  <line x1="200" y1="58" x2="195" y2="98" stroke="currentColor" stroke-width="1.5" stroke-dasharray="4 3" marker-end="url(#cc)"/>
  <text x="270" y="178" text-anchor="middle" font-size="11" fill="currentColor">A grant directs each call to a free voice channel; the CC keeps broadcasting.</text>
  <defs><marker id="cc" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The control channel continuously broadcasts identity, grants, and neighbor lists. A channel grant points a call at a free voice channel — follow the grants and you follow the whole system.</figcaption>
</figure>

## Why you decode it first

Everything routes through the control channel, so it is the **single most important
thing to decode**. Lock onto it and read its stream, and you learn the instant any
talkgroup starts a call, which voice channel (and slot) it was assigned, which
[radio ID](/radio-ids.html) is transmitting, and the system's identity. With that map,
a follower can point a receiver at the right voice channel at the right moment, capture
the call, and be ready for the next grant — across the whole channel pool.

Lose the control channel and you're **blind**: the voice channels still carry audio,
but you have no way to know which is which or who's on it. This is why so much of
operating a trunk-tracker comes down to getting a clean, solid lock on the control
channel — watch the raw stream in GopherTrunk's
**[CC Activity panel](/cc-activity.html)**.

## Not every system has a dedicated one

The dedicated control channel is the model used by P25, DMR Tier 3, TETRA, and the
big Motorola networks — but it isn't universal. Simpler or older systems use
**distributed** or **sub-audible** control: instead of dedicating a frequency, they
embed the signaling *within the voice channels themselves*, often below the audible
range. **LTR** is the classic example — it has **no dedicated control channel** at all;
each repeater carries its own low-rate signaling alongside the voice.

For a monitor, the difference is fundamental. On a dedicated-control system you find
*one* frequency and the whole system unfolds. On a distributed-control system there's
no single map to lock — you read the signaling spread across the channels. Knowing
which kind you're facing is the first step in identifying any system, which is exactly
where Module 3 and beyond pick up.

<div class="knowledge-check" data-quiz data-correct-msg="Correct — the control channel is the map, so you decode it first to know where calls go." markdown="0">
  <p class="knowledge-check__q">Quick check: why does a trunk-tracker decode the control channel before the voice channels?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Because it carries the clearest audio</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Because it's the map that says which voice channel each call is on</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because it uses a simpler modulation</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **control channel** carries only **data**, **continuously** — never voice.
- It's the **map** of the system: identity, **channel grants**, neighbor sites, and
  affiliation.
- You **decode it first** because without it you can't tell which voice channel carries
  which call.
- Its always-on data makes it the **easiest part of the system to find and lock**.
- Not all systems have one — **LTR** uses distributed, sub-audible control instead.

That closes Module 2. The next module opens with the distinction that started it all:
[conventional vs. trunked](/learn/digital-trunking/conventional-vs-trunked/).
