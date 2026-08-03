---
slug: nxdn
title: "NXDN"
description: NXDN explained — the narrowband 4FSK FDMA standard from Icom and Kenwood, with NXDN48 at 6.25 kHz and NXDN96 at 12.5 kHz, AMBE+2 voice, and Type-C trunking.
keywords: NXDN, NXDN48, NXDN96, NEXEDGE, IDAS, 4FSK FDMA, 6.25 kHz, narrowband digital, AMBE+2, Type-C trunking, Icom Kenwood, common air interface
level: intermediate
status: full
prereq:
  - digital-modulation-for-trunking
  - identifying-the-system
faq:
  - q: What is NXDN?
    a: NXDN is a narrowband digital land-mobile-radio standard developed jointly by Icom and Kenwood. It uses 4FSK FDMA in very narrow channels — 6.25 kHz (NXDN48) or 12.5 kHz (NXDN96) — with the AMBE+2 vocoder. It comes in conventional and trunked forms and is branded NEXEDGE by Kenwood and IDAS by Icom.
  - q: What is the difference between NXDN48 and NXDN96?
    a: They are the two channel options of the same standard. NXDN48 runs at 2400 symbols/s in a 6.25 kHz channel for maximum spectrum efficiency. NXDN96 runs at 4800 symbols/s in a 12.5 kHz channel for higher throughput. Both use the same 4FSK modulation and AMBE+2 voice; they differ in width and bit rate.
  - q: Who uses NXDN?
    a: NXDN is common in business and industrial radio — utilities, transport, security, and similar — and appears in some public-safety and government deployments. Because it is very narrow, it suits operators who want to fit many channels into limited spectrum. You'll meet it both conventionally and as Type-C trunked systems.
  - q: How do I recognise NXDN on the air?
    a: Its defining trait is how narrow it is, especially NXDN48 at 6.25 kHz — a thin sliver on the waterfall, narrower than DMR or P25. It is 4FSK like those systems, so the surest identification comes from letting a decoder read the signaling once it locks the channel.
gophertrunk_links:
  - title: Status
    url: /status.html
    note: which protocols GopherTrunk decodes and how completely.
  - title: Symbol scope
    url: /symbol-scope.html
    note: inspect the narrow 4FSK symbols of an NXDN channel.
---

# NXDN

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**NXDN** is a **narrowband 4FSK FDMA** standard developed by **Icom and Kenwood**. It comes
in two widths: **NXDN48** (**6.25 kHz**, 2400 symbols/s) and **NXDN96** (**12.5 kHz**, 4800
symbols/s), both using the **AMBE+2** vocoder. It supports **conventional** and **trunked**
(Type-C) operation and is sold as **NEXEDGE** by Kenwood and **IDAS** by Icom. It's common
in **business and industrial** radio, with some public-safety use. Its signature is being
**very narrow** — NXDN48 is one of the thinnest signals you'll see on the waterfall. Its
close ETSI cousin **[dPMR](/learn/digital-trunking/dpmr-dstar-ysf/)** works almost identically.
</div>

After the 25 kHz bulk of [TETRA](/learn/digital-trunking/tetra/), NXDN is the opposite extreme: one of the
narrowest digital voice standards in regular use. Where TETRA spread its calls wide, NXDN
squeezes them into a sliver of spectrum.

## What NXDN is

NXDN grew out of a joint effort by **Icom** and **Kenwood** to define an open *common air
interface* for narrowband digital radio. The result is **FDMA** — each call gets its own
frequency, no time slots — using **4FSK**, the same four-level frequency modulation you met
in [digital modulation for trunking](/learn/digital-trunking/digital-modulation-for-trunking/) and that P25 and DMR
also use. Voice rides on the **[AMBE+2](/learn/rf-sdr/vocoders/)** vocoder.

Because the two vendors sell it under their own names, you'll see NXDN behind several
brands:

| Brand | Vendor | Notes |
|-------|--------|-------|
| **NEXEDGE** | Kenwood | Kenwood's NXDN product line |
| **IDAS** | Icom | Icom's NXDN product line |
| **NXDN** | (the standard) | The common air interface both implement |

It's the same protocol underneath — a NEXEDGE radio and an IDAS radio speak NXDN to each
other.

## NXDN48 and NXDN96

The standard defines **two channel options**, and the names come from the bit rate:

| Variant | Channel width | Symbol rate | Spirit |
|---------|---------------|-------------|--------|
| **NXDN48** | **6.25 kHz** | **2400 symbols/s** | maximum spectrum efficiency |
| **NXDN96** | **12.5 kHz** | **4800 symbols/s** | higher throughput |

NXDN48's **6.25 kHz** channel is the headline feature — it fits *two* channels into the
space a single 12.5 kHz DMR or P25 channel uses, which is why it's prized where spectrum is
tight. NXDN96 trades that efficiency for a faster channel in a conventional 12.5 kHz slot.
Both use the same 4FSK and AMBE+2; only the width and rate change.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 150" role="img" aria-label="A width comparison showing NXDN48 at 6.25 kHz as a thin block, NXDN96 and DMR/P25 at 12.5 kHz as wider blocks, and TETRA at 25 kHz as the widest." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor">
    <text x="260" y="20" text-anchor="middle" font-weight="600">Channel width, to scale</text>
    <rect x="40" y="40" width="40" height="28" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1"/>
    <text x="60" y="84" text-anchor="middle" font-size="9">NXDN48</text>
    <text x="60" y="96" text-anchor="middle" font-size="8">6.25 kHz</text>
    <rect x="120" y="40" width="80" height="28" fill="currentColor" fill-opacity="0.20" stroke="currentColor" stroke-width="1"/>
    <text x="160" y="84" text-anchor="middle" font-size="9">NXDN96</text>
    <text x="160" y="96" text-anchor="middle" font-size="8">12.5 kHz</text>
    <rect x="240" y="40" width="80" height="28" fill="currentColor" fill-opacity="0.20" stroke="currentColor" stroke-width="1"/>
    <text x="280" y="84" text-anchor="middle" font-size="9">DMR / P25 P1</text>
    <text x="280" y="96" text-anchor="middle" font-size="8">12.5 kHz</text>
    <rect x="360" y="40" width="120" height="28" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/>
    <text x="420" y="84" text-anchor="middle" font-size="9">TETRA</text>
    <text x="420" y="96" text-anchor="middle" font-size="8">25 kHz</text>
    <text x="260" y="130" text-anchor="middle" font-size="9">NXDN48 is a thin sliver — that narrowness is its signature.</text>
  </g>
</svg>
<figcaption>NXDN48 at 6.25 kHz is one of the narrowest digital voice signals you'll encounter — about half the width of a DMR or P25 channel.</figcaption>
</figure>

## Conventional and trunked

NXDN runs in both styles you met earlier in [conventional vs
trunked](/learn/digital-trunking/conventional-vs-trunked/):

- **Conventional** — a fixed channel that a group always uses, common for small business and
  industrial fleets.
- **Trunked** — NXDN's trunking is called **Type-C**, with a [control
  channel](/learn/digital-trunking/the-control-channel/) granting traffic channels just like the systems you've
  studied. GopherTrunk follows the control channel to track [talkgroups and
  IDs](/learn/digital-trunking/talkgroups-ids-affiliation/).

## Recognising it, and its cousin dPMR

The easiest tell is **width**. NXDN48's 6.25 kHz channel is one of the thinnest signals on
the [waterfall](/learn/rf-sdr/fft-and-waterfall/) — much narrower than DMR or P25. It *is* 4FSK
like those systems, so width alone won't fully distinguish it; the reliable answer comes
from letting the decoder read the [signaling](/learn/digital-trunking/identifying-the-system/) once it locks.

NXDN has a very close European relative, **dPMR**, an ETSI standard that also does 6.25 kHz
4FSK FDMA. They are separate standards but so similar in footprint that they're easy to
confuse — we cover dPMR alongside D-STAR and System Fusion in [dPMR, D-STAR & System
Fusion](/learn/digital-trunking/dpmr-dstar-ysf/).

<div class="knowledge-check" data-quiz data-correct-msg="Correct — NXDN48 is a 6.25 kHz 4FSK FDMA channel, about half a DMR channel's width." markdown="0">
  <p class="knowledge-check__q">Quick check: what is the most distinctive thing about NXDN48 on the air?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It is a wide 25 kHz channel</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It is very narrow — only 6.25 kHz</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It uses phase modulation, not 4FSK</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **NXDN** is a **narrowband 4FSK FDMA** standard from **Icom and Kenwood**, branded
  **IDAS** and **NEXEDGE**.
- **NXDN48** is **6.25 kHz** (2400 symbols/s); **NXDN96** is **12.5 kHz** (4800 symbols/s);
  both use **AMBE+2**.
- It runs **conventional** and **trunked** (**Type-C**), used mostly in business and
  industry with some public safety.
- Its signature is being **very narrow** — though confirming it usually means letting the
  decoder read the signaling.
- Its ETSI cousin **dPMR** is nearly identical in footprint.

Next, we turn to a North-American giant that's still everywhere — Motorola's analog
trunking family: [SmartNet / SmartZone & Type II](/learn/digital-trunking/motorola-smartnet/).
