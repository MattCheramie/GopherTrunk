---
slug: control-channel-signaling
title: "Control-channel signaling: what the data says"
description: The message types on a trunked control channel — voice grants and updates, registration and affiliation, system status, neighbor-site broadcasts, and the identifier updates that map channel numbers to real frequencies.
keywords: control channel signaling, TSBK, CSBK, channel grant, grant update, affiliation, identifier update, IDEN, adjacent site, network status, channel map, P25, DMR
level: advanced
status: full
prereq:
  - anatomy-of-a-call
  - what-is-trunking
faq:
  - q: What kinds of messages are on a trunking control channel?
    a: A control channel carries channel grants and grant updates for calls, registration and affiliation messages from radios, system service and status broadcasts, adjacent-site (neighbor) announcements, identifier or frequency-band updates that map channel numbers to real frequencies, and network status. Together these let a decoder build a complete live picture of the system.
  - q: How does a channel number become an actual frequency?
    a: Grants reference a channel number, not a frequency. Identifier or frequency-band update messages provide the formula — a base frequency, channel spacing, and other parameters — that converts a channel number into a real transmit frequency. A trunk-tracker must collect these before it can tune to a granted voice channel.
  - q: What are TSBK and CSBK?
    a: TSBK (Trunking Signaling Block) is the message unit P25 uses to carry control-channel signaling on its dedicated control channel. CSBK (Control Signaling Block) is the equivalent in DMR. Both are short, structured data blocks whose type field says whether they carry a grant, an affiliation, a status broadcast, and so on.
  - q: Why does a trunk-tracker need neighbor-site messages?
    a: Adjacent-site broadcasts list the control-channel frequencies of neighboring sites in a multi-site system. A tracker uses them to know where a radio might roam and which other control channels exist, which is useful both for following roaming users and for finding a stronger site to monitor.
gophertrunk_links:
  - title: CC Activity
    url: /cc-activity.html
    note: see decoded control-channel messages by type as GopherTrunk reads them.
---

# Control-channel signaling: what the data says

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The control channel is a constant stream of short, typed data messages. **Grants** and
**grant updates** announce calls and their voice channels; **registration/affiliation**
messages track which radios and talkgroups are active; **system status** and **network
status** describe the system; **adjacent-site** broadcasts list neighbor control channels;
and crucially, **identifier/frequency-band updates** carry the formula that turns a
**channel number** into a real **frequency**. In P25 these ride in **TSBKs**; in DMR, in
**CSBKs**. A trunk-tracker assembles all of this into a **live channel map**.

</div>

You've watched a [single call](/learn/digital-trunking/anatomy-of-a-call/) flow through the control channel. Now we
catalogue the *full vocabulary* — the message types that, taken together, let a decoder
reconstruct the entire system. This is the most data-dense part of trunking, and it's where
a "trunk-tracker" earns its name.

## The carrier: TSBK and CSBK

Control-channel messages aren't free-form; each is a short, structured **block** with a
**type** field. Two concrete carriers dominate the digital world:

- **P25** packs its signaling into **TSBKs** — *Trunking Signaling Blocks*. Each TSBK is a
  fixed-size block whose opcode says what kind of message it is.
- **DMR** Tier III uses **CSBKs** — *Control Signaling Blocks* — the analogous unit on the
  DMR control channel.

The idea is the same in both: a stream of small, typed packets. A decoder reads the type,
then parses the fields that type defines. Everything below is just *which types matter* and
*what they tell you*.

## The message families

| Message family | What it carries | Why a tracker needs it |
|----------------|-----------------|------------------------|
| Group voice channel grant | Talkgroup + assigned voice channel + source radio ID | The cue to tune a receiver to a new call |
| Grant update | Re-announcement of an active call's channel | Late entry; staying locked to ongoing calls |
| Unit/group registration & affiliation | A radio registering or declaring its talkgroup | Live roster of active radios and groups |
| System service / status | System capabilities and operating parameters | Confirms identity and features of the system |
| Adjacent (neighbor) site broadcast | Control-channel frequencies of nearby sites | Following roaming; finding a stronger site |
| Identifier / frequency-band update | Base frequency, spacing — the channel-number formula | Converting channel numbers into real frequencies |
| Network status | Network/system identifiers (e.g. system ID, WACN) | Pinning down exactly which system this is |

Each family answers a different question. The grants say *where calls are happening*. The
registration and affiliation messages say *who is on the system*. The status and network
messages say *what system this is*. The neighbor broadcasts say *what's around it*. And the
identifier updates — easy to overlook — are what make the grants *usable at all*.

## From channel number to frequency

Here's the subtlety that trips up newcomers: a **grant doesn't contain a frequency**. It
says "go to channel 3," where *channel 3* is an abstract number. To turn that number into a
real frequency you tune, the decoder needs the **identifier / frequency-band update**
messages (P25 calls these IDEN updates). These carry a **base frequency**, a **channel
spacing**, and related parameters — effectively a formula:

> frequency = base + (channel number × spacing) ± offset

Until a tracker has collected the relevant identifier update, a grant for "channel 3" is
meaningless — there's no way to know *which* frequency channel 3 is. This is why a freshly
locked control channel sometimes takes a moment before calls become followable: the tracker
is waiting to hear the identifier updates that complete its map.

## Assembling the live channel map

Put the families together and a trunk-tracker maintains a continuously updated model of the
whole system:

<figure class="figure" markdown="0">
<svg viewBox="0 0 560 210" role="img" aria-label="A diagram showing control-channel message types flowing into a trunk-tracker, which combines identifier updates and grants to produce a live channel map of talkgroups, frequencies, and active calls." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <rect x="20" y="30" width="150" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="95" y="45">Grants & updates</text>
    <rect x="20" y="62" width="150" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="95" y="77">Affiliation/registration</text>
    <rect x="20" y="94" width="150" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="95" y="109">Identifier updates</text>
    <rect x="20" y="126" width="150" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="95" y="141">Neighbor & status</text>
  </g>
  <rect x="220" y="60" width="120" height="60" rx="6" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.5"/>
  <text x="280" y="86" text-anchor="middle" font-size="11" fill="currentColor" font-weight="600">Trunk-</text>
  <text x="280" y="100" text-anchor="middle" font-size="11" fill="currentColor" font-weight="600">tracker</text>
  <g stroke="currentColor" stroke-width="1.2">
    <line x1="170" y1="41" x2="220" y2="72"/><line x1="170" y1="73" x2="220" y2="82"/>
    <line x1="170" y1="105" x2="220" y2="98"/><line x1="170" y1="137" x2="220" y2="108"/>
  </g>
  <rect x="390" y="40" width="150" height="100" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="465" y="60" text-anchor="middle" font-size="11" fill="currentColor" font-weight="600">Live channel map</text>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="465" y="82">channel № → frequency</text>
    <text x="465" y="98">talkgroups active now</text>
    <text x="465" y="114">radios affiliated</text>
    <text x="465" y="130">neighbor sites</text>
  </g>
  <line x1="340" y1="90" x2="390" y2="90" stroke="currentColor" stroke-width="1.5" marker-end="url(#cm)"/>
  <defs><marker id="cm" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A trunk-tracker fuses the message families — especially identifier updates with grants — into a live channel map: which numbers map to which frequencies, which talkgroups and radios are active, and what neighbor sites exist.</figcaption>
</figure>

This map is what GopherTrunk's **[CC Activity](/cc-activity.html)** panel exposes: the raw,
typed messages decoded in real time. Watching them is the clearest way to understand a
system — and to spot, for example, that you've locked a control channel but haven't yet
seen the identifier update you need.

<div class="knowledge-check" data-quiz data-correct-msg="Right — identifier/frequency-band updates carry the formula that maps channel numbers to frequencies." markdown="0">
  <p class="knowledge-check__q">Quick check: a grant says "go to channel 3." Which message tells you what frequency channel 3 is?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The affiliation message</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The identifier / frequency-band update</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The neighbor-site broadcast</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Control-channel signaling is a stream of short, **typed blocks** — **TSBK** (P25),
  **CSBK** (DMR).
- **Grants** and **grant updates** announce calls; **affiliation/registration** track
  active radios.
- **System/network status** identify the system; **adjacent-site** broadcasts list neighbor
  control channels.
- **Identifier/frequency-band updates** carry the formula that turns **channel numbers**
  into **frequencies**.
- A trunk-tracker fuses these into a **live channel map** — exactly what CC Activity shows.

Next, we step back and compare the [trunking flavors](/learn/digital-trunking/trunking-flavors/) — dedicated vs.
distributed control, and message vs. transmission trunking.
