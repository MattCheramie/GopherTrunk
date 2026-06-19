---
slug: talkgroups-ids-affiliation
title: "Talkgroups, radio IDs & affiliation"
description: A talkgroup is a virtual channel; a radio ID identifies the individual unit; affiliation registers a radio so the system knows which talkgroup it monitors — and why you follow talkgroups, not frequencies.
keywords: talkgroup, radio ID, unit ID, affiliation, registration, group call, individual call, private call, emergency, trunking identity, virtual channel
level: intermediate
status: full
prereq:
  - conventional-vs-trunked
  - what-is-trunking
faq:
  - q: What is a talkgroup?
    a: A talkgroup is a virtual channel — a number with a label like "PD Dispatch" or "Fire Ground 2" — that identifies a group of users. Everyone in a talkgroup hears each other's calls no matter which voice channel the system assigns. On a trunked system you follow talkgroups rather than frequencies, because the frequency changes from call to call.
  - q: What is a radio ID or unit ID?
    a: A radio ID, also called a unit ID or source ID, is a number that identifies an individual radio on the system. Every transmission carries the ID of the radio that sent it, so you can see which specific unit is talking, not just which group. IDs are usually mapped to aliases like a badge number or vehicle in databases.
  - q: What is affiliation?
    a: Affiliation, or registration, is how a radio tells the system which talkgroup it is monitoring. When a radio powers on or changes talkgroups, it sends an affiliation message on the control channel. This lets the system route calls only to sites where members are listening, and it gives a monitor a live picture of who is active.
  - q: What is the difference between a group call and a private call?
    a: A group call goes to a whole talkgroup — everyone affiliated to it hears the audio. A private or individual call is one radio calling another by its radio ID, heard only by those two units. Most traffic on public-safety systems is group calls; private calls and emergency declarations also appear in the control-channel data.
gophertrunk_links:
  - title: Radio IDs
    url: /radio-ids.html
    note: see which individual radios are affiliating and transmitting on the system.
  - title: CC Activity
    url: /cc-activity.html
    note: watch affiliation and call messages stream by on the control channel.
---

# Talkgroups, radio IDs & affiliation

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **talkgroup** is a *virtual channel* — a number plus a label like "PD Dispatch" — that
binds a group of users together regardless of which frequency a call lands on. A **radio
ID** (unit ID) identifies the *individual* radio behind each transmission. **Affiliation**
(registration) is the message a radio sends to tell the system *which talkgroup it's
monitoring*, which both routes calls and gives a monitor a live who's-active picture.
Calls can be **group**, **individual/private**, or flagged **emergency**. Because
frequencies change per call, **you follow talkgroups, not frequencies**.

</div>

A trunked system constantly moves conversations across its channel pool, so it needs
*stable identities* to make sense of it all. Three identities matter, and once you know
them, the [control-channel data](control-channel-signaling/) stops looking like noise.

## Talkgroups — the virtual channel

A **talkgroup** is the unit of organisation users actually think in. It's a **number**
(say, 101) paired with a **label** ("Police Dispatch"). Every member of a talkgroup hears
every call to that talkgroup, no matter which physical voice channel the system assigns.
The talkgroup is *virtual* precisely because the frequency underneath it is not fixed —
the system may grant channel 3 for one call and channel 7 for the next.

This is why, on a trunked system, **you follow talkgroups, not frequencies**. You decide
you want "Fire Dispatch" and let the software chase the frequency for you. In GopherTrunk
you lock, prioritise, or mute *talkgroups*, and the control-channel follower handles the
hopping underneath.

## Radio IDs — the individual unit

Where a talkgroup names a *group*, a **radio ID** (also **unit ID** or **source ID**)
names a *single radio*. Every transmission on a trunked system carries the ID of the radio
that keyed up, so you can see not just "Dispatch is busy" but "unit 1147 is transmitting to
Dispatch." Systems and databases map these numbers to aliases — a badge number, a vehicle,
a dispatcher console — so a stream of IDs becomes a roster of who's on the air.

GopherTrunk surfaces these in the **[Radio IDs](/radio-ids.html)** panel, letting you watch
individual units appear, affiliate, and transmit across the system.

## Affiliation — registering interest

How does the system know who's listening to "Fire Dispatch" so it can route a call there?
Through **affiliation**, also called **registration**. When a radio powers on, joins a
site, or turns its channel selector to a new talkgroup, it sends an **affiliation message**
on the [control channel](/learn/digital-trunking/the-control-channel/) declaring *which talkgroup it now monitors*.

Affiliation serves the system: it can grant a voice channel only at sites where members are
actually present, saving capacity. It also serves *you*: because radios announce themselves
this way, the control channel is a steady stream of information about which units and
talkgroups are active right now — visible in GopherTrunk's
**[CC Activity](/cc-activity.html)** and **[Radio IDs](/radio-ids.html)** panels even
before any voice is keyed.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 200" role="img" aria-label="A diagram showing two radios affiliating to a talkgroup over the control channel. Each radio has its own radio ID, and both belong to talkgroup 101 labelled PD Dispatch." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="20" width="460" height="34" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.5"/>
  <text x="270" y="41" text-anchor="middle" font-size="12" fill="currentColor" font-weight="600">Control channel — affiliation messages</text>
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <rect x="60" y="110" width="110" height="44" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="115" y="130">Radio ID 1147</text>
    <text x="115" y="145" font-size="9">"monitoring TG 101"</text>
    <rect x="370" y="110" width="110" height="44" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="425" y="130">Radio ID 2093</text>
    <text x="425" y="145" font-size="9">"monitoring TG 101"</text>
    <rect x="210" y="110" width="120" height="44" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.5"/>
    <text x="270" y="130">TG 101</text>
    <text x="270" y="145" font-size="9">"PD Dispatch"</text>
  </g>
  <line x1="115" y1="108" x2="160" y2="56" stroke="currentColor" stroke-width="1.5" stroke-dasharray="4 3"/>
  <line x1="425" y1="108" x2="380" y2="56" stroke="currentColor" stroke-width="1.5" stroke-dasharray="4 3"/>
  <text x="270" y="184" text-anchor="middle" font-size="11" fill="currentColor">Each radio has its own ID; affiliation binds it to a talkgroup.</text>
</svg>
<figcaption>Two radios, each with its own radio ID, affiliate to the same talkgroup. The system now knows where to deliver TG 101's calls — and a monitor can see both units are active.</figcaption>
</figure>

## Group, private, and emergency calls

Not every call goes to a whole group. The control-channel signalling distinguishes a few
kinds:

| Call type | Goes to | Notes |
|-----------|---------|-------|
| Group call | A whole talkgroup | The common case; all affiliated members hear it |
| Individual / private call | One specific radio ID | Unit-to-unit, like a phone call between two radios |
| Emergency | A talkgroup, flagged urgent | A radio raises an emergency; the system prioritises it |

A **group call** is the everyday traffic — dispatch to a unit, units to each other within a
talkgroup. An **individual (private) call** is one radio addressing another by its radio
ID, heard only by those two. An **emergency** is a group call carrying an urgent flag,
which the system prioritises and which usually stands out clearly in the control-channel
log. All three are announced in the same data stream you decode, so you can see them coming.

<div class="knowledge-check" data-quiz data-correct-msg="Correct — you follow the talkgroup; the system moves it across voice channels." markdown="0">
  <p class="knowledge-check__q">Quick check: to hear a particular group of users on a trunked system, what do you follow?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Their radio IDs one by one</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Their talkgroup</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A single fixed frequency</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **talkgroup** is a virtual channel — a number and label that binds a group of users.
- A **radio ID** (unit ID) identifies the *individual* radio behind each transmission.
- **Affiliation** registers a radio's chosen talkgroup so the system can route calls.
- Calls can be **group**, **individual/private**, or flagged **emergency**.
- You **follow talkgroups, not frequencies**, because the frequency changes per call.

Next, we trace a single call from start to finish — the
[anatomy of a trunked call](anatomy-of-a-call/): request, grant, and release.
