---
slug: anatomy-of-a-call
title: "Anatomy of a trunked call: request, grant, release"
description: One trunked call end to end — a radio requests a channel on the control channel, the system grants and announces a voice channel, affiliated radios retune, the call happens, and the channel is released after hang-time.
keywords: trunked call, channel request, channel grant, grant update, voice channel, hang time, channel release, late entry, control channel, retune, P25 grant
level: intermediate
status: full
prereq:
  - talkgroups-ids-affiliation
  - what-is-trunking
faq:
  - q: What happens when someone keys up on a trunked system?
    a: Their radio sends a channel request on the control channel. The system finds a free voice channel and broadcasts a grant naming the talkgroup and the channel. Every radio affiliated to that talkgroup retunes to the voice channel, the conversation happens there, and when it ends the channel is released back to the pool after a short hang-time.
  - q: What is hang-time?
    a: Hang-time is a short window the system holds the voice channel open after a transmission ends, in case someone replies. It keeps a back-and-forth conversation on the same channel instead of re-requesting one for every over. When hang-time expires with no further traffic, the channel is released to the pool.
  - q: What is late entry?
    a: Late entry is joining a call already in progress. If a radio — or a monitor — affiliates or tunes in after the initial grant, it can still pick up the call because the control channel periodically repeats the grant as a grant update. That repeated announcement lets latecomers find the active voice channel.
  - q: What is a grant update?
    a: A grant update is a control-channel message that re-announces an in-progress call's talkgroup and voice channel. The system sends it periodically while a call is active so radios that missed or arrived after the first grant can still find and follow the call. It is how late entry and channel-following stay reliable.
gophertrunk_links:
  - title: CC Activity
    url: /cc-activity.html
    note: watch grant and release messages for live calls scroll past in real time.
  - title: Radio IDs
    url: /radio-ids.html
    note: see which radio ID requested each call as it is granted.
---

# Anatomy of a trunked call: request, grant, release

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Every trunked call follows the same arc. A radio keys up and sends a **channel request**
on the [control channel](/learn/digital-trunking/the-control-channel/). The system finds a free voice channel and
broadcasts a **grant** — "talkgroup 101, go to channel 3" — which **affiliated** radios
hear and **retune** to. The conversation happens on that voice channel, with periodic
**grant updates** so latecomers can join (**late entry**). When traffic stops, the channel
is held briefly for **hang-time**, then **released** back to the pool. Decoding the grants
is exactly how a follower knows where every call goes.

</div>

You now know the [identities](/learn/digital-trunking/talkgroups-ids-affiliation/) a trunked system tracks. This
lesson stitches them together into a single call, from the moment a user presses the
push-to-talk to the moment the channel returns to the pool.

## The four phases

A trunked call moves through four phases, all coordinated on the control channel:

1. **Request.** A user keys up. Their radio sends a **channel request** on the
   [control channel](/learn/digital-trunking/the-control-channel/), naming its talkgroup (and carrying its
   [radio ID](/learn/digital-trunking/talkgroups-ids-affiliation/)).
2. **Grant.** The system computer finds a **free voice channel** and broadcasts a **grant**
   on the control channel: "talkgroup 101 → channel 3." Every radio
   [affiliated](/learn/digital-trunking/talkgroups-ids-affiliation/) to talkgroup 101 hears that data message.
3. **Conversation.** Those radios **retune** to channel 3 and the call happens there. While
   it's active, the system periodically re-sends a **grant update** so radios that arrive
   late can still find the channel.
4. **Release.** When traffic stops, the system holds the channel for a short **hang-time**
   in case of a reply. If none comes, the channel is **released** back to the pool for the
   next call — possibly a completely different talkgroup.

The voice channel is never owned by a talkgroup; it's borrowed for the call and handed
back. The next call from the same group may land somewhere else entirely.

<figure class="figure" markdown="0">
<svg viewBox="0 0 560 180" role="img" aria-label="A horizontal timeline of a trunked call with four labelled stages: request, grant, conversation on a voice channel, and release after hang-time." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="100" x2="530" y2="100" stroke="currentColor" stroke-opacity="0.4" stroke-width="1.5"/>
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <circle cx="80" cy="100" r="6" fill="currentColor"/>
    <text x="80" y="80" font-weight="600">Request</text>
    <text x="80" y="128" font-size="8">radio keys up (CC)</text>
    <circle cx="200" cy="100" r="6" fill="currentColor"/>
    <text x="200" y="80" font-weight="600">Grant</text>
    <text x="200" y="128" font-size="8">"TG 101 → ch 3"</text>
    <rect x="280" y="86" width="160" height="28" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/>
    <text x="360" y="104" font-size="9">conversation on ch 3</text>
    <text x="360" y="74" font-size="8">grant updates repeat →</text>
    <circle cx="490" cy="100" r="6" fill="currentColor"/>
    <text x="490" y="80" font-weight="600">Release</text>
    <text x="490" y="128" font-size="8">after hang-time</text>
  </g>
  <text x="280" y="160" text-anchor="middle" font-size="11" fill="currentColor">The voice channel is borrowed for the call, then returned to the pool.</text>
</svg>
<figcaption>One call end to end: request on the control channel, grant of a voice channel, the conversation (with repeated grant updates for late entry), then release after hang-time.</figcaption>
</figure>

## Watching it on the control channel

Because every phase is announced as data, a follower sees the whole call as a sequence of
control-channel messages — this is the stream GopherTrunk reads:

| Time | Control-channel message | What a follower does |
|------|-------------------------|----------------------|
| 0.0 s | Radio 1147 requests TG 101 | Note the request |
| 0.1 s | Grant: TG 101 → channel 3 | Tune a receiver to ch 3, record |
| 0.1–6.0 s | Grant update: TG 101 still on ch 3 | Keep following; late joiners can enter |
| 6.0 s | (traffic stops) | Hold ch 3 through hang-time |
| 7.5 s | TG 101 released; ch 3 freed | Return ch 3 to the pool, ready for next |

The first **grant** is the cue to point a receiver at the voice channel. The **grant
updates** are insurance: if a radio (or the follower) was busy at 0.1 s, it can still latch
onto the call at 3 s and not miss the rest. When the **release** arrives, the follower frees
the channel and is immediately ready for the next grant.

## Hang-time and late entry

Two details make trunking feel seamless.

**Hang-time** is the short pause the system keeps a channel open after a transmission ends.
Conversations are a back-and-forth, and re-requesting a channel for every single over would
add latency and churn. Holding the channel briefly lets the reply land on the *same*
channel, so a normal exchange stays put until the talking actually stops.

**Late entry** is joining a call already underway. Thanks to the repeated **grant
updates**, a radio that powers on or switches talkgroups mid-call — or a monitor that tunes
in late — can discover the active voice channel and pick up the conversation in progress.
Without grant updates, you'd only ever hear calls you happened to catch at the instant of
the first grant.

<div class="knowledge-check" data-quiz data-correct-msg="Right — grant updates re-announce an active call so late joiners can find the voice channel." markdown="0">
  <p class="knowledge-check__q">Quick check: what lets a radio join a trunked call that's already in progress?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The initial channel request</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Periodic grant updates on the control channel</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Parking on the voice frequency</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A call has four phases: **request**, **grant**, **conversation**, **release**.
- The **request** rides the control channel; the **grant** names the talkgroup and voice
  channel.
- Affiliated radios **retune** to the granted channel; **grant updates** repeat during the
  call.
- **Late entry** lets radios join a call in progress thanks to those updates.
- After traffic stops, **hang-time** holds the channel briefly, then it's **released**.

Next, we open up the control channel itself and read [what the data
says](/learn/digital-trunking/control-channel-signaling/) — the full vocabulary of trunking messages.
