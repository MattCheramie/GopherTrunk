---
slug: conventional-vs-trunked
title: "Conventional vs. trunked radio"
description: Conventional radio puts each group on a fixed frequency you can just listen to; trunked radio assigns channels on demand from a controller — why the old park-on-a-frequency scan fails.
keywords: conventional radio, trunked radio, fixed frequency, control channel, P25 conventional, digital conventional, scanning, repeater, talkgroup, channel assignment
level: beginner
status: full
prereq:
  - what-is-trunking
faq:
  - q: What is the difference between conventional and trunked radio?
    a: In conventional radio, each group has its own fixed frequency it always uses, so you tune to that frequency to listen. In trunked radio, a small pool of frequencies is shared and a controller assigns a free one to each call on demand. Conventional is simple to scan; trunked requires you to decode the control channel to follow conversations as they hop.
  - q: Can conventional radio be digital?
    a: Yes. Conventional only describes how channels are assigned — fixed, not pooled. The voice on a conventional channel can be analog FM or digital, including P25 conventional, DMR Tier II, or NXDN conventional. A digital conventional channel still lives on one frequency; it just carries a digital voice signal instead of analog.
  - q: Why can't I scan a trunked system the old way?
    a: Because a trunked call can land on any frequency in the pool, and a different call lands there a moment later. Parking a scanner on one frequency hears fragments of unrelated conversations. To follow a group you have to read the control channel, which announces which voice channel each call uses.
  - q: Is a repeater the same as trunking?
    a: No. A repeater simply rebroadcasts a single conventional channel to extend range; it is still one fixed frequency. Trunking adds a controller that manages a pool of channels across many repeaters and assigns them per call. A trunked site is built from several repeaters working together under one control channel.
gophertrunk_links:
  - title: CC Activity
    url: /cc-activity.html
    note: watch the control-channel traffic that a trunked system needs and a conventional one does not.
  - title: Hunt (discover systems)
    url: /hunt.html
    note: tell whether a frequency is a plain conventional channel or a trunked control channel.
---

# Conventional vs. trunked radio

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Conventional** radio gives each group its own **fixed frequency** — you tune to it and
listen, which is why a classic scanner works. **Trunked** radio pools a few frequencies
and a **controller** assigns a free one to each call on demand, so the frequency a group
uses *changes call to call*. Conventional can itself be **digital** (P25 conventional,
DMR Tier II) — "conventional" describes the *channel plan*, not the voice. You **cannot**
park a scanner on a trunked system: you must decode its **control channel** to follow
who's talking.

</div>

Before diving into how trunking works in detail, it's worth nailing the one distinction
everything else hangs on: how a system decides *which frequency* a conversation uses. Get
this straight and the rest of the path falls into place.

## Conventional: one group, one frequency

In a **conventional** system, each user group is assigned a **fixed frequency** it always
uses. Police dispatch is on one channel, fire ground on another, public works on a third,
and they never move. To listen, you do the obvious thing — tune your receiver to that
frequency and wait for traffic. A scanner just steps through a list of these frequencies,
stopping whenever one is active.

Conventional channels are often *repeated*: a repeater on a hilltop rebroadcasts the
channel to extend range. That's still conventional — it's one fixed frequency, just heard
further. Simplicity is the whole appeal. The cost is **spectrum**: every group needs its
own channel whether it's busy or idle, which wastes airwaves when most channels sit silent
most of the time.

## Conventional doesn't mean analog

A common trap is to assume "conventional" means old analog FM. It doesn't.
**Conventional** describes only the *channel plan* — fixed frequencies, no pooling. The
**voice** riding on a conventional channel can be:

- **Analog FM** — the classic narrowband voice scanners have always heard, or
- **Digital** — **P25 conventional**, **DMR Tier II**, **NXDN conventional**, and others.

A P25 conventional channel is still one frequency you can tune to; it just carries a
[digital voice](/learn/rf-sdr/digital-voice/) signal that needs a
[vocoder](/learn/digital-trunking/voice-to-bits-vocoders/) to turn back into audio. So "is it digital?" and "is it
trunked?" are two **independent** questions. A system can be analog-conventional,
digital-conventional, analog-trunked, or digital-trunked.

## Trunked: a shared pool, assigned on demand

In a **trunked** system there is no fixed home frequency. A small pool of channels is
shared across many groups, and a **controller** hands out a free channel for the duration
of each call, then reclaims it. Coordination rides on a dedicated
[control channel](/learn/digital-trunking/the-control-channel/) — one frequency that carries *data*, announcing where
each call goes. Members of a group hear each other through their
[talkgroup](/learn/digital-trunking/talkgroups-ids-affiliation/), regardless of which voice channel the system
picked.

This is far more spectrum-efficient: ten shared channels can serve forty groups, because
real conversations are short and bursty. But it means the **frequency a group uses is not
stable** — the next call from the same talkgroup might be on a completely different
channel.

## Why the old scan fails on trunking

This is the crux. On a conventional system, "park on the frequency" works because the
frequency *is* the group. On a trunked system, parking on a voice frequency hears whatever
call happened to be assigned there — a fragment of one conversation, then silence, then a
fragment of an unrelated one. There is no way to follow a single group that way.

To follow a trunked group you have to read the **control channel** and let it tell you
which voice channel each call uses, retuning in step. That is exactly the job
[GopherTrunk](/learn/digital-trunking/the-control-channel/) does, and it's why so much of this path is about the
control channel rather than the voice channels.

| | Conventional | Trunked |
|---|---|---|
| Channel assignment | Fixed — each group has its own frequency | On demand — controller assigns from a pool |
| Where you listen | Tune to the group's frequency | Decode the control channel, follow its grants |
| Voice can be | Analog FM **or** digital | Analog **or** digital |
| Old scanner works? | Yes — park on the frequency | No — calls hop across the pool |
| Spectrum efficiency | Low — idle channels still reserved | High — channels shared across groups |
| Coordination | None needed | Dedicated control channel carries data |

<div class="knowledge-check" data-quiz data-correct-msg="Right — conventional describes the channel plan, so it can carry digital voice too." markdown="0">
  <p class="knowledge-check__q">Quick check: can a conventional system carry digital voice?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Yes — e.g. P25 conventional on a fixed frequency</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">No — conventional always means analog FM</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only if it has a control channel</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Conventional** = each group on a **fixed frequency** you can tune to and listen.
- **Trunked** = a **shared pool** with a **controller** assigning channels per call.
- "Conventional" describes the *channel plan*, not the voice — it can be **digital**.
- A classic scanner works on conventional but **fails** on trunked, where calls hop.
- To follow a trunked group you decode the **control channel**, not a voice frequency.

Next, we look at the identities a trunked system tracks — [talkgroups, radio IDs and
affiliation](/learn/digital-trunking/talkgroups-ids-affiliation/) — the virtual channels you actually follow.
