---
slug: data-services
title: Data services — GPS, text & registration
description: How trunked radio systems carry more than voice — registration and affiliation, GPS location reporting, short data and text messaging, and packet data — and how a monitor sees it all ride on or beside the control channel.
keywords: trunked radio data, GPS location reporting, short data service, text messaging, registration, affiliation, ARS, packet data, P25 PDU, DMR data, control channel data
level: intermediate
status: full
prereq:
  - the-control-channel
  - talkgroups-ids-affiliation
faq:
  - q: Do trunked radios send more than voice?
    a: Yes — a great deal more. Every radio registers with the system, affiliates to talkgroups, and can send location reports, status codes, short text messages, and bearer packet data. Much of this is small signaling that rides the control channel; heavier data can be granted its own channel just like a voice call. A trunked system is really a data network that also carries voice.
  - q: Can I see the GPS locations of radios?
    a: Sometimes you can tell that location data is being sent — the control channel or a data call will show a radio ID transferring a short payload on a schedule. Whether you can read the coordinates depends on the system. Location reporting is often carried in vendor-specific formats, and it may be encrypted, so a scanner may see that a position update happened without being able to decode where the radio is.
  - q: What is registration and affiliation data?
    a: Registration is a radio telling the system it is present and reachable on a site; affiliation is a radio telling the system which talkgroup it is currently listening to. Both are short data messages a radio sends without any user speaking, and they let the system route calls and manage roaming. To a monitor they appear as frequent control-channel messages from many radio IDs with no voice call attached.
  - q: Where does data ride — the control channel or a separate channel?
    a: Both. Small, frequent data — registration, affiliation, status, short messages — rides the control channel alongside the call grants. Heavier or sustained data, such as bulk packet transfer, is typically assigned a traffic channel the same way a voice call is granted one, so it does not clog the control channel.
gophertrunk_links:
  - title: CC Activity
    url: /cc-activity.html
    note: watch control-channel messages, including registrations and affiliations, as GopherTrunk decodes them.
  - title: Radio IDs
    url: /radio-ids.html
    note: the unit IDs seen on the system, including those sending data rather than voice.
---

# Data services — GPS, text & registration

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A trunked system is a **data network that also carries voice**. Radios constantly send
**registration/affiliation** to say who they are and which talkgroup they want; many
also send **location/GPS** reports, **short data** and text messages, and some carry
bearer **packet data**. Small data rides the [control channel](/learn/digital-trunking/the-control-channel/)
alongside call grants; heavy data gets its own channel like a voice call. Much of it a
monitor can *see happening* even when the payload is opaque — building on
[talkgroups, IDs & affiliation](/learn/digital-trunking/talkgroups-ids-affiliation/).
</div>

It is easy to picture a trunked radio system as a telephone exchange for voice calls.
That is only half the story. Beneath the calls runs a steady flow of non-voice traffic —
the messages radios exchange with the system, and with each other, without anyone
keying up. This lesson catalogues that traffic and shows what a scanner can and can't
make of it.

## More than voice

Modern land-mobile-radio (LMR) systems move a surprising amount of non-voice data. Every
radio on the system is a networked device: it announces itself, keeps its talkgroup
membership current, may report where it is, can send and receive short messages, and on
some systems can carry general packet data. Dispatchers see much of this as map pins,
status boards, and message logs rather than as audio.

For a monitor, the important shift in mindset is this: the *absence* of voice does not
mean the system is idle. A busy control channel with no calls in progress is usually a
system doing plenty of data work.

## Registration & affiliation data

You met these in [talkgroups, IDs & affiliation](/learn/digital-trunking/talkgroups-ids-affiliation/); here they
matter as *data traffic*. When a radio powers on or roams onto a site it **registers** —
it tells the system "unit 12345 is here." When the user selects a talkgroup, the radio
**affiliates** — "unit 12345 wants group 101." Neither involves speech; both are short,
structured data messages.

This is exactly the kind of traffic a monitor sees on the control channel. In the
vocabulary of [control-channel signaling](/learn/digital-trunking/control-channel-signaling/), registration and
affiliation are their own message families, decoded right next to the call grants. Watch
a control channel for a minute and most of what scrolls past is this housekeeping data,
not voice.

## Location (GPS) reporting

Many systems support periodic **location reporting**: a radio sends its GPS position on a
schedule (say every 30 seconds, or on movement) so a dispatch console can plot it on a
map. This is the backbone of **AVL** — automatic vehicle location — used heavily by
public-safety and fleet operators.

What a scanner can infer varies. The location report is a data payload from a known radio
ID, so you can often tell *that* a unit is reporting position and *how often*. Reading the
coordinates is another matter: location data is frequently carried in **vendor-specific**
formats layered on top of the standard, and it may be **encrypted**. So it is common to
see the position updates happening — regular short data from a unit — without being able
to decode the actual latitude and longitude.

## Short data & text messaging

Alongside voice, radios exchange **short data**: canned **status** codes (a button that
means "en route," "on scene," "available") and **text messages**, both radio-to-radio and
between radios and the dispatch console. Status messaging is popular because it is terse
and machine-readable — a dispatcher's screen fills with unit states without a word being
spoken. Free-text messaging is slower and less common on the air but supported by most
standards.

## Packet / IP data

Some systems carry general **bearer** or **packet data** — moving arbitrary bytes rather
than a fixed message type. P25 defines a **packet data** service (data assembled into
PDUs, *protocol data units*); DMR carries data in its own frames. Typical uses are
telemetry, database queries (a mobile data terminal checking a plate or a name), and
software or configuration transfers. Sustained packet data is bandwidth-hungry, so it is
usually run on an assigned channel rather than squeezed onto the control channel.

## Control channel vs. dedicated data channel

The dividing line is size and duration:

- **Small, frequent data** — registration, affiliation, status, short messages, brief
  location reports — rides the **control channel** itself, interleaved with the call
  grants. It is cheap and needs no channel assignment.
- **Heavy or sustained data** — bulk packet transfer, large messages — is granted a
  **traffic/data channel**, assigned by the control channel exactly the way a voice call
  is. The radio tunes to the granted channel, moves its data, and releases it.

Which mechanism a system leans on depends on its design; the [trunking
flavors](/learn/digital-trunking/trunking-flavors/) lesson covers how different systems assign channels.

## What it looks like to GopherTrunk

GopherTrunk's control-channel decoder does not distinguish "voice systems" from "data
systems" — it decodes the message stream as it comes. In practice that means the log
shows **registrations**, **affiliations**, and **data-call grants** interleaved with the
voice-call grants. The [CC Activity](/cc-activity.html) panel is where this is visible;
the [Radio IDs](/radio-ids.html) view collects the unit IDs seen, including those that
only ever send data.

Some payloads are **opaque**. A data-call grant tells you a unit is moving data on an
assigned channel, but the contents may be a vendor format or **encrypted** — see
[encryption & authentication](/learn/digital-trunking/encryption-and-authentication/) and
[OTAR & key management](/learn/digital-trunking/otar-key-management/). As with encrypted voice, the *metadata*
(who, when, how often, which channel) is usually still readable even when the payload is
not.

## Per-system notes

Brief and standard-specific:

- **P25** — carries control-channel data in TSBKs and defines a full **packet data**
  service built from PDUs; registration, affiliation, and status all have defined message
  types. See [P25 Phase 1](/learn/digital-trunking/p25-phase-1/).
- **DMR** — Tier II and Tier III both support data; Tier III (trunked) uses CSBKs for
  signaling. Location and registration are often handled by **vendor protocols** (for
  example an **ARS**, *automatic registration service*, plus a GPS/telemetry protocol)
  layered on DMR's data frames. See [DMR Tier II & III](/learn/digital-trunking/dmr-tier-2-3/).
- **NXDN** and others — likewise define short data and status messaging alongside voice,
  with location and registration commonly vendor-specific. See [NXDN](/learn/digital-trunking/nxdn/).

Across all of them the pattern holds: standardized signaling for registration and
affiliation, plus a mix of standard and vendor-specific formats for location, status, and
bulk data.

<div class="knowledge-check" data-quiz data-correct-msg="Right — that constant stream of short messages from many IDs with no audio is registration, affiliation, and data traffic." markdown="0">
  <p class="knowledge-check__q">Quick check: you see frequent short messages on the control channel from many radio IDs, but no voice call is in progress. What are you most likely watching?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A single encrypted voice call</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Registration, affiliation, and data traffic</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Interference from an adjacent system</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A trunked system is a **data network that also carries voice** — non-voice traffic is
  constant.
- **Registration** and **affiliation** are short data messages that ride the control
  channel; a monitor sees them plainly.
- **Location/GPS** reporting drives dispatch mapping (AVL); you can often see it happening
  even when the coordinates are vendor-specific or encrypted.
- **Short data**, status codes, and text messaging move between radios and dispatch;
  **packet data** carries telemetry and queries.
- Small data rides the **control channel**; heavy data gets a **granted channel** like a
  voice call.
- To GopherTrunk it appears as registrations, affiliations, and data-call grants in
  [CC Activity](/cc-activity.html), with some payloads opaque or encrypted.

Next up: with the whole system understood, Module 4 takes the dominant standard in depth — [P25 Phase 1](/learn/digital-trunking/p25-phase-1/).
