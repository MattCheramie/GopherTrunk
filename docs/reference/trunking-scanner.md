---
slug: trunking-scanner
title: Trunking scanner
entry_type: concept
category: consumer-scanners
description: "A trunking scanner reads a trunked system's control channel and follows a talkgroup as it hops across voice channels — Motorola Type II, P25, SmartNet, EDACS, LTR. A non-trunking scanner misses most."
keywords: trunking scanner, trunk tracking scanner, what is a trunking scanner, control channel scanner, talkgroup scanner, Motorola Type II, P25 trunking, SmartNet, EDACS, LTR, follow talkgroup
aka: [trunk-tracking scanner, trunk tracker]
autolink: true
infobox:
  - { label: What it is, value: "A scanner that follows trunked-radio talkgroups" }
  - { label: How, value: "Reads the control channel, follows channel grants" }
  - { label: System types, value: "Motorola Type II, P25, SmartNet, EDACS, LTR" }
  - { label: Without it, value: "You miss most modern police/fire traffic" }
  - { label: Cannot do, value: "Decode encrypted (AES) talkgroups" }
  - { label: Free alternative, value: "SDR + GopherTrunk" }
see_also: [police-scanner, trunked-radio, control-channel, talkgroup, channel-grant, motorola-type-ii, edacs, ltr, uniden-sds200]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
cite_urls:
  - https://wiki.radioreference.com/index.php/Trunking_Basics
  - https://en.wikipedia.org/wiki/Trunked_radio_system
faq:
  - q: "What is a trunking scanner?"
    a: "A trunking scanner is a scanner that can follow a trunked-radio system. Instead of parking on one frequency, it reads the system's control channel, learns which voice channel each talkgroup has been assigned, and jumps there automatically — so a conversation that hops across many channels sounds continuous."
  - q: "Do I need a trunking scanner?"
    a: "If your local police, fire, or EMS use a trunked system — most metro agencies do — then yes. A non-trunking scanner can only catch scattered, disconnected fragments of a trunked conversation. Check your county on RadioReference: if it lists a Motorola, P25, EDACS, or LTR system, you need trunk-tracking."
  - q: "Is a trunking scanner the same as a digital scanner?"
    a: "No — they are separate capabilities. Trunking is about following talkgroups across channels; digital is about decoding P25/DMR/NXDN voice. A system can be analog-trunked (needs trunking but not digital), or digital-trunked (needs both). Match the scanner to your system's combination."
  - q: "Which scanners are trunk-tracking?"
    a: "Uniden's SDS100/SDS200, BCD436HP/BCD536HP, BCD325P2/BCD996P2, and the analog-only BearTracker BCT15X all trunk-track, as do Whistler's TRX-1 and TRX-2. The cheap analog handhelds (SR30C, BC125AT, BC75XLT) do not."
  - q: "Can a trunking scanner hear encrypted talkgroups?"
    a: "No. A trunking scanner will read the control channel, identify the encrypted talkgroup, and follow it to the voice channel — but it plays silence because it lacks the AES key. No scanner and no SDR, including GopherTrunk, can decode encryption."
  - q: "What is the cheapest way to follow a trunked system?"
    a: "A ~$30 RTL-SDR plus free, open-source GopherTrunk trunk-tracks P25, DMR, and NXDN systems on a PC, decoding and logging every talkgroup. A dedicated trunking scanner is more portable and turnkey but starts around $380 for digital."
---
**A trunking scanner** is a scanner that can **follow a
[trunked-radio](/reference/trunked-radio/) system** — it reads the system's
[control channel](/reference/control-channel/), learns which voice channel each
[talkgroup](/reference/talkgroup/) has just been assigned, and jumps there
automatically.[^rr] Without that ability, a scanner catches only disconnected
scraps of the conversation, which is why an ordinary
[scanner](/reference/police-scanner/) misses most modern public-safety traffic.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Trunking scanners follow talkgroups across channels.** They decode the
[control channel](/reference/control-channel/), read each
[channel grant](/reference/channel-grant/), and hop to the assigned
[voice channel](/reference/voice-channel/) so a call sounds continuous. **Most
modern police/fire are trunked** — a non-trunking scanner hears only fragments.
**Trunking ≠ digital**: a system can be analog-trunked or digital-trunked. **No
scanner decodes [encryption](/police-scanner-encryption/).** The **free path** is an
[SDR + GopherTrunk](/police-scanner-vs-sdr/).
</div>

## What makes a scanner a *trunking* scanner

On a conventional system, an agency owns a fixed frequency and you just tune to it.
On a **trunked** system, a pool of voice channels is **shared** among many
talkgroups, and a computer hands out channels on demand. When a unit keys up, the
system picks any free channel and announces the assignment on a dedicated
[control channel](/reference/control-channel/).

A trunking scanner is defined by one behavior: it **monitors that control channel**.
It continuously decodes the stream of [channel grants](/reference/channel-grant/) —
"talkgroup 1234 is now on voice channel 7" — and when it sees a grant for a
talkgroup you want, it **retunes to that voice channel** in time to catch the
transmission, then returns to the control channel to await the next grant. That
tracking is what makes a hopping conversation sound like one continuous call.

A plain scanner can't do this. Tuned to a single voice channel, it would hear a
few seconds of one talkgroup, then a fragment of an unrelated one, then silence —
never a coherent conversation.

## The systems a trunking scanner follows

- **[Motorola Type II](/reference/motorola-type-ii/) / SmartNet / SmartZone** — the
  most common US public-safety trunking family, analog and digital.
- **[APCO Project 25 (P25)](/reference/project-25/)** — digital trunking,
  [Phase I and II](/reference/p25-phase-2/), now dominant for police and fire.
- **[EDACS](/reference/edacs/)** — an older GE/Ericsson trunking standard, still in
  use in places.
- **[LTR](/reference/ltr/)** — a lightweight trunking scheme common on business and
  some public systems.

Each encodes its control channel differently, so a trunking scanner must support the
specific format your agency runs. Check your system type on
[RadioReference](/reference/radioreference/).

## Trunking vs digital — two separate axes

It is easy to confuse "trunking" with "digital," but they are independent:

| | Conventional | Trunked |
|---|---|---|
| **Analog FM** | Any scanner | Needs **trunk-tracking** (e.g. [BearTracker BCT15X](/reference/uniden-bearcat-bct15x/)) |
| **Digital (P25/DMR/NXDN)** | Needs a **digital** scanner | Needs a **digital trunk-tracking** scanner |

- **Trunking** = follows talkgroups across channels (control-channel tracking).
- **Digital** = decodes [P25](/reference/project-25/)/[DMR](/reference/dmr/)/[NXDN](/reference/nxdn/)
  voice.

A system can need one, the other, or both. The [BCT15X](/reference/uniden-bearcat-bct15x/)
trunk-tracks but is analog-only; the [SDS200](/reference/uniden-sds200/) does both.

## Which of these models trunk-track

Among the scanners in our field guide:

- **Trunk-tracking (digital):** [Uniden SDS200](/reference/uniden-sds200/),
  [Whistler TRX-1](/reference/whistler-trx-1/),
  [Whistler TRX-2](/reference/whistler-trx-2/),
  [Whistler WS1065](/reference/whistler-ws1065/) (legacy P25).
- **Trunk-tracking (analog only):** [BearTracker BCT15X](/reference/uniden-bearcat-bct15x/).
- **Not trunk-tracking:** the analog handhelds
  [Uniden SR30C](/reference/uniden-sr30c/),
  [BC125AT](/reference/uniden-bc125at/), and
  [BC75XLT](/reference/uniden-bc75xlt/) — fine for
  [conventional](/reference/conventional-radio/) analog, air, marine, rail, and
  racing, but they cannot follow a trunked system.

See the full [best police scanners](/best-police-scanners/) ranking to match a model
to your area.

## What a trunking scanner cannot do

Following a talkgroup is not the same as **hearing** it. When a talkgroup is
[encrypted](/police-scanner-encryption/), a trunking scanner still reads the control
channel, identifies the talkgroup, and hops to the voice channel — then plays
silence, because it has no key. **No scanner and no SDR — including
[GopherTrunk](/police-scanner-vs-sdr/) — can decode AES encryption.** Trunk-tracking
gets you to the conversation; it can't break the cipher.

## The free alternative

A [software-defined radio](/reference/software-defined-radio/) — a ~$30
[RTL-SDR](/reference/rtl-sdr/) — running **GopherTrunk** is a full trunking receiver:
it decodes the [control channel](/reference/control-channel/), follows
[talkgroups](/reference/talkgroup/) across [P25](/reference/project-25/),
[DMR](/reference/dmr/), and [NXDN](/reference/nxdn/) systems, and additionally
**records, logs, and timestamps every call** across unlimited talkgroups at once —
something no single scanner does. It trades portability and a turnkey experience for
zero cost and far more data. The honest head-to-head is in
[police scanner vs GopherTrunk](/police-scanner-vs-sdr/), and you can
[download GopherTrunk](/downloads.html) to try it with a cheap dongle.

## Bottom line

A trunking scanner is what lets one radio follow a shared, channel-hopping
[trunked system](/reference/trunked-radio/) as if it were a single channel. If your
agencies are trunked — most are — you need trunk-tracking, plus **digital** decode if
they run [P25/DMR/NXDN](/reference/p25-phase-2/). Encrypted talkgroups stay off-limits
to everyone, and a dongle plus [GopherTrunk](/downloads.html) does the same tracking
and decoding for free if you have a PC to spare.

## Sources

[^rr]: [Trunking Basics](https://wiki.radioreference.com/index.php/Trunking_Basics) — RadioReference Wiki, on control channels, channel grants, and how trunk-tracking scanners follow talkgroups across a trunked system.
