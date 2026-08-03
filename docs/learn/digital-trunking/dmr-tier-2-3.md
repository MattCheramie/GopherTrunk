---
slug: dmr-tier-2-3
title: "DMR: Tier II & Tier III"
description: DMR explained in depth — ETSI TS 102 361, two-slot TDMA in 12.5 kHz, 4FSK and AMBE+2, the three tiers, color codes, CSBK signaling, and the proprietary Capacity/Connect variants.
keywords: DMR, ETSI TS 102 361, Tier II, Tier III, two-slot TDMA, 4FSK, AMBE+2, color code, CSBK, Capacity Plus, Connect Plus, Capacity Max, Hytera XPT, DMR trunking
level: intermediate
status: full
prereq:
  - tdma-vs-fdma
  - control-channel-signaling
faq:
  - q: What is DMR?
    a: DMR (Digital Mobile Radio) is an open ETSI standard, TS 102 361, widely used in business, utility and amateur radio. It fits two-slot TDMA into a 12.5 kHz channel — about 6.25 kHz equivalent per call — using 4FSK at 4800 symbols per second and the AMBE+2 vocoder. Its low cost made it the dominant digital standard outside public safety.
  - q: What are the three DMR tiers?
    a: Tier I is license-free, low-power simplex for personal and light commercial use. Tier II is conventional licensed operation with two-slot TDMA repeaters — the most common business DMR. Tier III is trunked DMR, with a dedicated control channel and CSBK signaling that assigns calls across a channel pool.
  - q: What is a DMR color code?
    a: A color code is a value carried in DMR bursts that works like a NAC or CTCSS tone — radios ignore traffic whose color code does not match. It lets neighboring repeaters share a frequency without interfering. GopherTrunk reads the color code to confirm it is decoding the right repeater.
  - q: How do proprietary DMR trunking systems differ from Tier III?
    a: Vendors built their own trunked schemes on the DMR physical layer — Motorola Capacity Plus, Connect Plus and Capacity Max, and Hytera XPT — before or alongside open Tier III. They use vendor-specific control signaling rather than standard Tier III CSBKs, so a decoder needs system-specific support to follow them, while conventional Tier II DMR is well covered by the open standard.
gophertrunk_links:
  - title: DMR encryption
    url: /dmr-encryption.html
    note: configure known keys for DMR systems with RC4 Enhanced Privacy.
  - title: CC Activity
    url: /cc-activity.html
    note: watch CSBK signaling on a Tier III control channel.
  - title: Vocoders
    url: /vocoders.html
    note: see how GopherTrunk renders DMR's AMBE+2 voice frames.
---

# DMR: Tier II & Tier III

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**DMR** (Digital Mobile Radio) is the open **ETSI TS 102 361** standard that dominates
business, utility and amateur digital radio. It packs **two-slot TDMA** into a **12.5 kHz**
channel — **≈6.25 kHz equivalent** per call — using **4FSK at 4800 symbols/s (9600 bps)**
and the **AMBE+2** vocoder. It comes in **three tiers**: **Tier I** (license-free
simplex), **Tier II** (conventional two-slot repeaters), and **Tier III** (trunked, with a
dedicated control channel and **CSBK** signaling). A **color code** keeps repeaters apart
like a NAC. Vendors also built proprietary trunked variants — **Capacity Plus, Connect
Plus, Capacity Max, Hytera XPT** — that differ from open Tier III.
</div>

Where [P25](/learn/digital-trunking/p25-phase-1/) rules North-American public safety, **DMR** rules almost
everything else digital: taxis, warehouses, utilities, security, and a vast amateur
network. Its appeal is simple — it is an open standard with cheap radios that doubles
capacity on existing 12.5 kHz channels. This lesson unpacks how it works and the tiers
you will meet.

## The physical layer: two slots in 12.5 kHz

DMR uses **4FSK** — the same four-level FSK family as P25's
[C4FM](/learn/digital-trunking/digital-modulation-for-trunking/) — running at **4800 symbols/s**, which (two bits
per symbol) gives **9600 bps**. The clever part is **two-slot TDMA**: a single 12.5 kHz
channel is split in *time* into two alternating slots, so it carries **two simultaneous
calls**. That is the same **6.25 kHz equivalent** efficiency idea as P25 Phase 2, but DMR
built it in from the start across the whole standard. Voice rides in the **AMBE+2**
[vocoder](/learn/rf-sdr/vocoders/) — specifically the 3600×2450 variant, distinct from the
3600×2400 variant P25 Phase 2 and NXDN use.

A **color code** (0–15) is carried in the bursts and works like a P25 NAC or an analog
CTCSS tone: a radio ignores traffic whose color code does not match, so neighbouring
repeaters can reuse a frequency without interfering. GopherTrunk reads it to confirm it is
locked to the intended repeater.

## The three tiers

DMR is defined as three tiers, escalating in capability:

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 200" role="img" aria-label="Three stacked tiers of DMR. Tier I license-free simplex at the bottom, Tier II conventional two-slot repeater in the middle, Tier III trunked with a control channel at the top." xmlns="http://www.w3.org/2000/svg">
  <g font-size="11" fill="currentColor">
    <rect x="60" y="140" width="420" height="44" rx="5" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.4"/>
    <text x="270" y="158" text-anchor="middle" font-weight="600">Tier I — license-free, low-power simplex</text>
    <text x="270" y="174" text-anchor="middle" font-size="9">no repeater, no infrastructure</text>
    <rect x="60" y="86" width="420" height="44" rx="5" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1.4"/>
    <text x="270" y="104" text-anchor="middle" font-weight="600">Tier II — conventional licensed, two-slot repeater</text>
    <text x="270" y="120" text-anchor="middle" font-size="9">fixed channel, 2 calls per frequency, color code</text>
    <rect x="60" y="32" width="420" height="44" rx="5" fill="currentColor" fill-opacity="0.26" stroke="currentColor" stroke-width="1.6"/>
    <text x="270" y="50" text-anchor="middle" font-weight="600">Tier III — trunked, dedicated control channel</text>
    <text x="270" y="66" text-anchor="middle" font-size="9">CSBK signaling assigns calls across a channel pool</text>
  </g>
</svg>
<figcaption>The three DMR tiers escalate from license-free simplex (Tier I) through conventional two-slot repeaters (Tier II) to fully trunked operation with a control channel (Tier III).</figcaption>
</figure>

- **Tier I** — **license-free**, low-power **simplex** (radio to radio, no repeater). Think
  consumer and very light commercial use. There is no infrastructure to follow.
- **Tier II** — **conventional, licensed** operation through **two-slot repeaters**. Each
  repeater sits on a fixed pair of frequencies and carries two slots. This is the
  workhorse of commercial DMR and amateur DMR repeaters. There is *no* control channel —
  you monitor it much like a conventional channel, decoding whichever slot is active.
- **Tier III** — **trunked** DMR. A **dedicated control channel** carries **CSBK**
  (Control Signaling Block) messages that grant calls across a pool of channels, exactly
  the trunking pattern you know from P25 but with DMR's signaling vocabulary.

## Conventional Tier II vs. trunked Tier III

The practical difference is whether there is a control channel to follow.

| | DMR Tier II | DMR Tier III |
|---|-------------|--------------|
| Operation | Conventional | Trunked |
| Control channel | None | Dedicated, CSBK signaling |
| Channel assignment | Fixed per repeater | Assigned per call from a pool |
| Signaling | Embedded link control | CSBK grants on control channel |
| How you monitor | Watch the repeater's two slots | Decode control channel, follow grants |
| GopherTrunk grant | `dmr-tier2` | `dmr-tier3` |

For **Tier II**, GopherTrunk treats the repeater like a conventional digital channel:
lock the carrier, read the embedded link control to learn the talkgroup and source ID per
slot, and decode AMBE+2 for whichever slot is transmitting. For **Tier III**, it does the
full trunking dance — lock the **control channel**, read **CSBKs** on the
**[CC Activity](/cc-activity.html)** panel, and follow each grant to the assigned voice
channel and slot, exactly as it does for [P25](/learn/digital-trunking/p25-phase-1/) but with DMR framing.

## CSBKs: DMR's trunking signaling

The **CSBK** is to Tier III what the TSBK is to P25: a fixed-size control message. On a
Tier III control channel a continuous stream of CSBKs announces channel grants,
registrations, and system parameters. The grant CSBKs tell a follower which physical
channel and **time slot** a talkgroup's call was assigned. Because DMR is two-slot, a
Tier III grant — like P25 Phase 2 — references a *slot* as well as a frequency, so the
decoder must tune the channel and then lock the correct slot.

## The proprietary variants

Here is the wrinkle that makes DMR trunking messy in the field: several major **trunked
DMR systems are not open Tier III** at all. Vendors built their own trunking on top of the
DMR physical layer — often before Tier III was finalized — using **vendor-specific control
signaling** rather than standard CSBK grants:

- **Motorola Capacity Plus** — single-site (and Multi-Site) trunking that spreads calls
  across a repeater's slots without a separate dedicated control channel in the
  Tier III sense.
- **Motorola Connect Plus** — multi-site trunking using a dedicated control channel, but
  with Motorola's own signaling, not open CSBKs.
- **Motorola Capacity Max** — Motorola's later, larger multi-site trunked architecture.
- **Hytera XPT** — Hytera's distributed trunking scheme.

All sit on the same 4FSK + AMBE+2 physical layer, so they *look* like DMR on a scope, but
their control signaling differs from the open standard. A decoder therefore needs
**system-specific support** to follow each one; the open Tier III CSBK logic does not
automatically apply. Conventional Tier II, by contrast, is well covered by the open
standard and the most reliably decoded.

## Encryption on DMR

Many commercial DMR systems protect voice with **RC4/ARC4 "Enhanced Privacy."**
GopherTrunk supports **known-key** decode: an operator supplies a key they are authorized
to hold, declared per system with a `key_id`, `algorithm: rc4`, and the hex key. It reads
the encrypted flag from the link control before voice starts, and for unencrypted calls
decodes AMBE+2 end to end. The full configuration and current pipeline status — including
which descramble step is still pending — are in [DMR encryption](/dmr-encryption.html).
GopherTrunk performs **no key recovery**; only monitor systems you are legally permitted
to monitor.

<div class="knowledge-check" data-quiz data-correct-msg="Correct — Tier III is the trunked tier, with a dedicated control channel and CSBK signaling." markdown="0">
  <p class="knowledge-check__q">Quick check: which DMR tier is trunked, with a dedicated control channel?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Tier I</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Tier II</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Tier III</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **DMR** is the open **ETSI TS 102 361** standard: **two-slot TDMA in 12.5 kHz, 4FSK at
  9600 bps, AMBE+2** voice, with a **color code** that keeps repeaters apart.
- **Tier I** is license-free simplex; **Tier II** is conventional two-slot repeaters;
  **Tier III** is trunked with a control channel and **CSBK** signaling.
- GopherTrunk reads embedded link control on **Tier II** and follows **CSBK grants** on
  **Tier III**, including the per-call **slot**.
- Proprietary **Capacity Plus / Connect Plus / Capacity Max / Hytera XPT** share DMR's
  physical layer but use vendor signaling, needing system-specific support.
- DMR **RC4 Enhanced Privacy** is supported in a known-key model only.

That closes the P25 and DMR module. The next module opens with another major trunked
family rooted in Europe: [TETRA](/learn/digital-trunking/tetra/).
