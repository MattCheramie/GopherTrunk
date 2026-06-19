---
slug: analog-trunking-era
title: "The analog trunking era: SmartNet, EDACS, LTR & MPT-1327"
description: The first big trunked systems carried analog voice with digital control — Motorola Type I/II, EDACS, LTR, and MPT-1327 — and the concepts they invented that digital systems inherited.
keywords: analog trunking, Motorola SmartNet, SmartZone, Motorola Type II, EDACS, LTR, Logic Trunked Radio, MPT-1327, message trunking, transmission trunking, distributed control
level: intermediate
status: full
prereq:
  - birth-of-trunking
faq:
  - q: What is the difference between analog and digital trunking?
    a: Analog trunking carries the voice as ordinary FM but uses digital signalling to assign channels — the control is digital, the audio is analog. Digital trunking carries the voice itself as bits through a vocoder. Many ideas, like dedicated control channels and talkgroups, were invented in the analog era and carried straight into digital systems.
  - q: What is the difference between message and transmission trunking?
    a: In transmission trunking, a channel is assigned for a single transmission (one press of the key) and released the instant the user lets go. In message trunking, the channel is held through a short hang time so a back-and-forth conversation stays on one channel. Message trunking feels smoother; transmission trunking returns channels to the pool faster.
  - q: What was LTR and why was it different?
    a: LTR (Logic Trunked Radio) used distributed control with no single dedicated control channel. Each repeater carried sub-audible data alongside the voice, and the radios worked out channel assignments collectively. That made LTR cheap and simple for business systems, but it signals very differently from dedicated-control systems like SmartNet or EDACS.
  - q: Are analog trunked systems still on the air?
    a: Yes. Motorola Type II, EDACS, LTR, and MPT-1327 systems still operate in places, even as many agencies migrate to P25 and DMR. GopherTrunk decodes a range of these legacy systems, so understanding them is still useful for a scanner enthusiast.
gophertrunk_links:
  - title: Status
    url: /status.html
    note: which legacy trunking systems GopherTrunk decodes.
  - title: CC Activity
    url: /cc-activity.html
    note: watch dedicated-control signalling on a legacy system.
---

# The analog trunking era: SmartNet, EDACS, LTR & MPT-1327

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The first big trunked systems carried **analog voice** but **digital control**. The major
families were **Motorola Type I/II/IIi** (branded **SmartNet/SmartZone**), **GE/Ericsson
EDACS**, **LTR** (Logic Trunked Radio — *distributed* control, sub-audible data, **no
dedicated control channel**), and **MPT-1327** (the British/European standard). These
architectures invented the vocabulary digital systems inherited: **dedicated vs
distributed control**, and **message vs transmission trunking**. Many are **still on the
air**, and **GopherTrunk decodes them**.
</div>

Trunking arrived years before digital voice. For a long while the state of the art was a
clever hybrid — your voice still went out as ordinary FM, but a computer chose the channel
and announced it as data. This lesson surveys those analog trunked systems, because the
concepts they pioneered shape every digital standard that came after.

## The hybrid idea: analog voice, digital control

Picture the trunking machinery from the last lesson — a controller, a channel pool, a
control channel handing out assignments — but the audio on each voice channel is plain
[analog FM](/learn/rf-sdr/analog-modulation/). That's the whole analog trunking era in one
sentence. The *organising* layer was digital and modern; the *voice* layer was the same
FM that had been around for decades. This split is why a single system could enjoy
trunking's spectrum efficiency without anyone needing a vocoder.

These systems also settled two design questions that still matter:

- **Dedicated vs distributed control.** A *dedicated*-control system reserves one channel
  full-time as the control channel (SmartNet, EDACS). A *distributed*-control system has no
  single control channel; the signalling rides along with the voice on each repeater (LTR).
- **Message vs transmission trunking.** *Transmission* trunking releases a channel the
  instant a user unkeys; *message* trunking holds it through a short hang time so a
  conversation stays put. The trade-off is faster channel reuse versus smoother
  back-and-forth.

## The major families

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 150" role="img" aria-label="Four labelled boxes representing Motorola, EDACS, LTR, and MPT-1327, with Motorola, EDACS, and MPT-1327 grouped as dedicated control and LTR set apart as distributed control." xmlns="http://www.w3.org/2000/svg">
  <text x="260" y="18" text-anchor="middle" font-size="11" fill="currentColor" font-weight="600">Analog trunking families</text>
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <rect x="20" y="40" width="110" height="46" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/>
    <text x="75" y="60">Motorola</text>
    <text x="75" y="74" font-size="8">Type I/II/IIi</text>
    <rect x="145" y="40" width="110" height="46" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/>
    <text x="200" y="60">EDACS</text>
    <text x="200" y="74" font-size="8">GE / Ericsson</text>
    <rect x="395" y="40" width="110" height="46" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/>
    <text x="450" y="60">MPT-1327</text>
    <text x="450" y="74" font-size="8">UK / Europe</text>
    <rect x="270" y="40" width="110" height="46" rx="5" fill="none" stroke="currentColor" stroke-width="1.3" stroke-dasharray="4 3"/>
    <text x="325" y="60">LTR</text>
    <text x="325" y="74" font-size="8">distributed</text>
  </g>
  <text x="167" y="112" text-anchor="middle" font-size="9" fill="currentColor">dedicated control channel</text>
  <text x="325" y="112" text-anchor="middle" font-size="9" fill="currentColor">no dedicated control</text>
  <line x1="20" y1="98" x2="315" y2="98" stroke="currentColor" stroke-width="1" stroke-opacity="0.5"/>
  <line x1="270" y1="92" x2="270" y2="130" stroke="currentColor" stroke-width="1" stroke-opacity="0.5" stroke-dasharray="3 3"/>
</svg>
<figcaption>Motorola, EDACS, and MPT-1327 use a dedicated control channel; LTR is the odd one out, distributing its signalling across every repeater.</figcaption>
</figure>

**Motorola Type I/II/IIi** is the most widespread legacy family, sold under the
**SmartNet** and **SmartZone** brand names. Type I, Type II, and the hybrid Type IIi differ
mainly in how they encode talkgroup and radio identity, and SmartZone added multi-site
coverage. Type II in particular blanketed North American public safety and business for
years.

**EDACS** (Enhanced Digital Access Communications System) came from **GE**, later
**Ericsson**. It used a dedicated control channel with very fast channel assignment and a
distinctive signalling scheme, and was common in public-safety and utility deployments.

**LTR (Logic Trunked Radio)** is the conceptual outlier. It has **no dedicated control
channel**: each repeater carries **sub-audible data** beneath the voice, and the radios
coordinate channel use collectively — *distributed* control. That made LTR inexpensive and
simple, which is why it was popular for smaller business systems.

**MPT-1327** is the British/European standard, an open specification widely deployed
outside North America for business and government trunking. Like the Motorola and EDACS
families, it uses a dedicated control channel.

## How they compare

| System | Origin | Control | Typical use |
|--------|--------|---------|-------------|
| Motorola Type I/II/IIi (SmartNet/SmartZone) | Motorola | Dedicated | US public safety & business |
| EDACS | GE / Ericsson | Dedicated | Public safety, utilities |
| LTR | E.F. Johnson | **Distributed** (sub-audible) | Small business systems |
| MPT-1327 | UK / open standard | Dedicated | Business & government outside US |

The shared thread is *analog voice, digital control*. Where they diverge — dedicated vs
distributed control, and how they encode IDs — is exactly the kind of difference that
decides how a monitor follows them.

## What digital inherited

When the digital standards arrived, they didn't reinvent trunking — they reused it.
Dedicated control channels, talkgroups, affiliation, and the message/transmission trunking
distinction all came straight out of this era. P25, DMR Tier III, and TETRA are, at the
trunking layer, refinements of ideas that SmartNet and EDACS proved in the field. The big
addition was digital *voice* — vocoders, error correction, and built-in encryption — on top
of a trunking model that already worked.

These systems haven't vanished. Many are **still on the air**, and **GopherTrunk decodes a
range of them**, so they're not just history. The [Status reference](/status.html) tracks
which it handles.

## Recap

- The first big trunked systems carried **analog voice** with **digital control**.
- **Motorola Type I/II/IIi (SmartNet/SmartZone)** and **EDACS** used **dedicated** control channels.
- **LTR** was different — **distributed** control with **sub-audible data** and no dedicated control channel.
- **MPT-1327** was the open British/European standard, dedicated-control, used worldwide outside North America.
- They invented **dedicated vs distributed control** and **message vs transmission trunking**, which digital systems inherited; many are **still on the air** and **GopherTrunk decodes them**.

We dig into these per-system in [Motorola SmartNet](/learn/digital-trunking/motorola-smartnet/)
and [EDACS, LTR & MPT-1327](/learn/digital-trunking/edacs-ltr-mpt1327/), and compare every
flavour in [Trunking flavours](/learn/digital-trunking/trunking-flavors/). Next, we cross the
line into digital voice in [The digital leap](/learn/digital-trunking/the-digital-leap/).
