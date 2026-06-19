---
slug: motorola-smartnet
title: "Motorola SmartNet / SmartZone & Type II"
description: Motorola SmartNet, SmartZone, and Type II explained — a 3600 bps digital control channel coordinating analog FM voice, Type I/II/IIi addressing, multi-site networking.
keywords: Motorola SmartNet, SmartZone, Type II, Type I, Type IIi, 3600 bps control channel, analog trunking, fleet subfleet, talkgroup ID, proprietary trunking, multi-site
level: intermediate
status: full
prereq:
  - analog-trunking-era
  - control-channel-signaling
  - talkgroups-ids-affiliation
faq:
  - q: What is Motorola SmartNet?
    a: SmartNet is Motorola's proprietary analog trunking system. It uses a digital control channel running at 3600 bps to coordinate calls, but the voice itself is ordinary analog FM. SmartZone is the multi-site extension that networks several SmartNet sites together. Both remain very widespread despite being decades old.
  - q: What is the difference between Type I, Type II, and Type IIi?
    a: They are addressing schemes. Type I organises radios by fleet, subfleet, and a size code — a hierarchical scheme. Type II uses flat numeric talkgroup IDs with no fleet structure, which is simpler and more common. Type IIi is a hybrid that supports both Type I and Type II radios on the same system during transitions.
  - q: Is SmartNet digital or analog?
    a: It is a mix. The control channel is digital — a 3600 bps data stream that announces calls and assigns voice channels. The voice traffic itself is analog FM, the same as a conventional analog radio. So it is analog trunking coordinated by digital signaling, not a fully digital system like P25 or DMR.
  - q: Can GopherTrunk follow SmartNet systems?
    a: Yes. GopherTrunk decodes the 3600 bps control channel to learn which talkgroup is being granted which voice channel, then follows the analog voice. Because the control channel is the map, the same follow-the-control-channel approach used for digital systems applies, even though the voice is analog FM.
gophertrunk_links:
  - title: CC Activity
    url: /cc-activity.html
    note: watch the 3600 bps SmartNet control-channel messages decode in real time.
  - title: Radio IDs
    url: /radio-ids.html
    note: see which radios and talkgroups are active on a SmartNet system.
---

# Motorola SmartNet / SmartZone & Type II

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**SmartNet** is Motorola's **proprietary analog trunking** family, still very widespread.
A **3600 bps digital control channel** coordinates the system, but the **voice is plain
analog FM** — digital signaling, analog audio. Radios are addressed by **Type I**
(fleet/subfleet/size-code), **Type II** (flat talkgroup IDs), or **Type IIi** (a hybrid).
**SmartZone** extends it across **multiple sites**. It is *not* an open standard, but it's
everywhere, and **GopherTrunk decodes the 3600 bps control channel** to follow calls the
same way it follows a digital system.

</div>

You met the era of [analog trunking](analog-trunking-era/) earlier. SmartNet is its most
successful survivor: a Motorola design from the 1980s–90s that, despite being decades old
and proprietary, still carries an enormous amount of traffic across North America.

## Digital control, analog voice

The defining idea of SmartNet is the split between *how it's coordinated* and *what it
carries*:

- The **control channel** is **digital** — a continuous **3600 bps** data stream. It does
  exactly what a control channel does in any [trunked system](conventional-vs-trunked/):
  registers radios, takes call requests, and broadcasts grants telling a talkgroup which
  voice channel to use.
- The **voice channels** are **analog FM** — the same modulation as a conventional analog
  radio. Once a radio is granted a channel, the audio there is ordinary FM you could tune by
  hand.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 180" role="img" aria-label="A SmartNet system: a digital 3600 bps control channel at top issuing a grant, pointing to one of several analog FM voice channels below." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="20" width="460" height="40" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.5"/>
  <text x="270" y="38" text-anchor="middle" font-size="13" fill="currentColor" font-weight="600">Control channel — digital, 3600 bps</text>
  <text x="270" y="53" text-anchor="middle" font-size="10" fill="currentColor">“Talkgroup 1808 → voice channel 5”</text>
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <rect x="40" y="110" width="100" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="90" y="130">Voice 4</text>
    <text x="90" y="143" font-size="8">analog FM</text>
    <rect x="160" y="110" width="100" height="40" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.5"/>
    <text x="210" y="130">Voice 5</text>
    <text x="210" y="143" font-size="8">analog FM — active</text>
    <rect x="280" y="110" width="100" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="330" y="130">Voice 6</text>
    <text x="330" y="143" font-size="8">analog FM</text>
    <rect x="400" y="110" width="100" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="450" y="130">Voice 7</text>
    <text x="450" y="143" font-size="8">analog FM</text>
  </g>
  <line x1="270" y1="60" x2="210" y2="108" stroke="currentColor" stroke-width="1.5" stroke-dasharray="4 3" marker-end="url(#sn)"/>
  <defs><marker id="sn" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>SmartNet coordinates with a digital 3600 bps control channel but carries ordinary analog FM voice — so you decode the data to follow the audio.</figcaption>
</figure>

This is why SmartNet sits in this *digital* learning path even though its voice is analog:
the *signaling* is digital, and following the control channel is exactly the skill the rest
of the path teaches.

## Type I, Type II, and Type IIi addressing

How a SmartNet system labels its users evolved over time, giving three addressing schemes
you'll encounter:

| Scheme | How radios are addressed | Character |
|--------|--------------------------|-----------|
| **Type I** | **Fleet / subfleet / size-code** hierarchy | Older, structured; size code partitions the ID space into fleets and subfleets |
| **Type II** | **Flat numeric talkgroup IDs** | Simpler and most common; just a talkgroup number, no fleet structure |
| **Type IIi** | **Hybrid** of Type I and Type II | Supports both radio types on one system during a migration |

**Type II** is the one you'll meet most. Its flat [talkgroup
IDs](talkgroups-ids-affiliation/) map cleanly onto the talkgroup model you already know —
a number with a label, like "1808 — County Fire Dispatch." Type I's fleet/subfleet/size-code
scheme is a holdover that takes a little decoding to map back to a meaningful unit, which is
why you sometimes see Type I IDs written in a fleet-subfleet-ID notation.

## SmartZone — going multi-site

A single SmartNet site covers one area. **SmartZone** is Motorola's extension that
**networks several sites together** into one logical system, so a radio can roam between
sites and stay reachable on the same talkgroup. This is the same multi-site idea you'll see
again in [sites, simulcast & roaming](sites-simulcast-roaming/) — a wide-area network built
from many sites, each with its own control channel, stitched together by the back-end
network.

## Proprietary, but everywhere

It's worth being clear: SmartNet/SmartZone is a **proprietary Motorola system**, not an open
standard like [P25](p25-phase-1/) or the [ETSI](standards-and-bodies/) standards. There's no
public specification you can simply read; understanding it came largely from the
scanner-enthusiast community reverse-engineering the [control-channel
signaling](control-channel-signaling/).

So why does so much of it remain? Cost and inertia. These systems were expensive to build,
they work reliably, and many agencies kept them running for decades — sometimes as a
stepping stone before migrating to P25. The result is that SmartNet remains one of the most
commonly encountered trunking systems in North America, and **GopherTrunk decodes its
3600 bps control channel** so you can follow it like any other.

<div class="knowledge-check" data-quiz data-correct-msg="Right — SmartNet's control channel is digital, but the voice it grants is analog FM." markdown="0">
  <p class="knowledge-check__q">Quick check: on a Motorola SmartNet system, what kind of signal carries the actual voice?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Digital 4FSK like P25</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Analog FM</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">TDMA time slots</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **SmartNet** is Motorola's **proprietary analog trunking** family, still very widespread.
- A **3600 bps digital control channel** coordinates calls, but the **voice is analog FM**.
- Addressing comes in **Type I** (fleet/subfleet/size-code), **Type II** (flat talkgroup
  IDs), and **Type IIi** (hybrid); Type II is most common.
- **SmartZone** networks **multiple sites** into one roaming-capable system.
- It is **not an open standard**, but **GopherTrunk decodes the control channel** to follow
  it like any trunked system.

Next, three more legacy families you'll still run into: [EDACS, LTR &
MPT-1327](edacs-ltr-mpt1327/).
