---
slug: conventional-vs-trunked-recap
title: Conventional vs. trunked, in the field
description: "The one distinction that changes how you scan — a conventional channel you can park on versus a trunked system you have to follow. Seen from the listener's chair: what each sounds like, what you program, and why trunk-tracking exists."
keywords: conventional vs trunked, trunked radio scanning, conventional channel, control channel, trunk tracking, following a call, scanning a trunked system, talkgroup
level: beginner
status: full
prereq:
  - what-is-scanning
  - history-of-scanning
faq:
  - q: What is the practical difference between conventional and trunked scanning?
    a: On a conventional system each channel lives on a fixed frequency, so you can park your scanner on it and hear everything said there. On a trunked system the frequencies are pooled and handed out per call by a control channel, so a single conversation hops between frequencies. To follow trunked traffic your scanner must decode the control channel and chase each grant — you cannot just sit on one voice frequency.
  - q: How do I tell if a system is trunked?
    a: A trunked system has a control channel — a carrier that transmits data continuously and never goes quiet — alongside voice channels that light up only during calls. In a database like RadioReference a system is labelled conventional or trunked, with the trunked entries listing a control channel and talkgroups rather than named channel frequencies. On the air, the always-on data carrier is the giveaway.
  - q: Can one scanner do both?
    a: Yes. A trunk-tracking scanner (or SDR software like GopherTrunk) handles both — it can park on conventional channels and follow trunked systems in the same scan. An older analog-only scanner can do conventional but cannot follow trunking, which is the main reason people upgrade.
---

# Conventional vs. trunked, in the field

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
This is the single distinction that changes *how you scan*. A **conventional** channel
lives on a **fixed frequency** — park on it and you hear everything. A **trunked**
system **pools frequencies and hands them out per call**, coordinated by a
**[control channel](/learn/digital-trunking/the-control-channel/)**, so one conversation
hops around the pool. To follow trunked traffic your scanner must **read the control
channel and chase the grants** — which is exactly why trunk-tracking scanners and
GopherTrunk exist. The [theory lives in the digital
module](/learn/digital-trunking/conventional-vs-trunked/); this is the view from your
chair.
</div>

You met these two families in passing in the
[first lesson](/learn/scanning/what-is-scanning/), and the
[history](/learn/scanning/history-of-scanning/) explained why trunking forced a new kind
of scanner. Now we look at the difference the way a listener experiences it — because it
decides what you program, what you hear, and why some systems seem to fall apart on an
older radio.

## Conventional: a channel you can park on

A **conventional** system is the intuitive kind. Each logical channel — dispatch,
tactical 1, fireground, public works — is a **fixed frequency**. "Dispatch" is always
on, say, 154.250 MHz, and it is always there. If you want to hear dispatch, you program
that frequency and you are done; the traffic comes and goes on that one carrier.

From the listener's chair this is wonderfully simple:

- **One channel = one frequency.** Program it once and it never moves.
- **You can park.** Sit on the frequency and you hear every transmission on it,
  start to finish.
- **What you load is what you hear.** A list of conventional frequencies is a complete,
  self-contained scan — no extra machinery required.

Plenty of the spectrum is still conventional: many fire and EMS operations, business
radios, aviation, marine, rail, amateur repeaters, and smaller agencies. For all of
these, an ordinary programmable scanner parked on the right frequencies does the whole
job.

## Trunked: a system you have to follow

A **trunked** system is designed to serve many talkgroups with only a handful of
frequencies, by refusing to tie any conversation to a fixed channel. Instead the system
keeps a **pool** of voice frequencies and a controlling computer that hands them out on
demand. When a radio keys up, the system finds a free frequency in the pool, assigns the
call to it, and tells everyone involved where to go — all over a
**[control channel](/learn/digital-trunking/the-control-channel/)**.

The consequence for you is decisive: **a single conversation is not on any fixed
frequency**. The first call of the day might be assigned frequency A; the next call on
the very same talkgroup might land on frequency C; the one after that on frequency B.
Park on frequency A and you will hear a random scrap of one call and then silence, while
the conversation you wanted continues elsewhere. This is precisely the chaos that broke
conventional scanners when trunking arrived.

## The control channel is the map

What makes a trunked system followable is that all of this assignment traffic is out in
the open on the control channel. The **control channel** transmits data **continuously**
— it never goes quiet — announcing every call: *talkgroup 1234 is now on frequency C.*
A trunk-tracking scanner decodes that stream and, the instant it sees a grant for a
talkgroup you care about, **jumps to the assigned frequency** to hear the call, then
returns to the control channel to wait for the next one.

So following a trunked system is a two-step dance the radio does for you:

1. **Lock the control channel** and read its running commentary of grants.
2. **Follow each grant** to its voice frequency, listen, and come back.

You program the *system* — its control-channel frequency and the talkgroups you want —
rather than a list of voice frequencies. The digital module details the signalling in
[the control channel](/learn/digital-trunking/the-control-channel/) and identity in
[talkgroups & affiliation](/learn/digital-trunking/talkgroups-ids-affiliation/); the
field-side steps come later in this module in
[programming a trunked system](/learn/scanning/programming-a-trunked-system/) and
[following a call](/learn/scanning/following-a-call/).

## Talkgroups replace channels

On a conventional system you think in **channels** (frequencies). On a trunked system
you think in **talkgroups** — logical groups like "City Fire Dispatch" or "PD North
Patrol" that exist independently of any frequency. A talkgroup is an *identity* the
system routes to whatever frequency is free, and it is the unit you select, prioritise,
and lock out. Learning to think in talkgroups instead of frequencies is the mental shift
trunked scanning demands, and we build scan lists out of them in
[talkgroups & scan lists](/learn/scanning/talkgroups-and-scan-lists/).

## Telling them apart

In practice you rarely have to guess. A database like
[RadioReference](/learn/scanning/radioreference-database/) labels each system as
conventional or trunked: conventional entries list named channel frequencies, while
trunked entries list a control channel, a system type (P25, DMR, and so on), and
talkgroups. On the air, the signature is the **always-on control-channel carrier** — a
data stream that never pauses — sitting among voice channels that flicker only during
calls. That constant carrier is the surest sign you are looking at a trunked system, and
[finding it](/learn/digital-trunking/the-control-channel/) is the first step to following
the system.

<div class="knowledge-check" data-quiz data-correct-msg="Right — on a trunked system a single call can be assigned to any frequency in the pool, so you must follow the control channel." markdown="0">
  <p class="knowledge-check__q">Quick check: why can't you just park on one voice frequency to follow a trunked conversation?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Because voice frequencies on trunked systems are always encrypted</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Because the system assigns each call to whatever frequency is free, so the conversation hops around the pool</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because trunked voice channels only transmit for one second at a time</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Conventional** = a channel on a **fixed frequency**; park on it and hear everything.
  Simple to program, still common across many services.
- **Trunked** = a **pool of frequencies handed out per call**, so a conversation hops
  around and you cannot park on it.
- The **control channel** announces every grant continuously; a trunk-tracking scanner
  reads it and **follows each call** to its assigned frequency.
- You program a trunked **system and its talkgroups**, not a list of voice frequencies —
  and you think in **talkgroups** instead of channels.
- Tell them apart via a **database label** or the **always-on control-channel carrier**
  on the air.

Next up: [What you can (and can't) hear today](/learn/scanning/what-you-can-hear/).
