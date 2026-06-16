---
slug: protocol-landscape
title: The digital protocol landscape
description: A field guide to digital radio protocols — P25 Phase 1 and 2, DMR Tier II/III, NXDN, dPMR, TETRA, and legacy systems like Motorola Type II, EDACS, LTR, and MPT 1327 — who uses each, how they differ, and how to tell them apart.
keywords: P25, DMR, NXDN, TETRA, Motorola Type II, EDACS, LTR, MPT 1327, dPMR, digital radio protocols, trunking standards, identify radio system
level: intermediate
status: full
prereq:
  - what-is-trunking
faq:
  - q: What is the difference between P25 and DMR?
    a: Both are digital land-mobile-radio standards using 4-level FSK, but P25 is a North-American public-safety standard (Phase 1 is FDMA at 12.5 kHz; Phase 2 is TDMA for double capacity) using IMBE/AMBE+2 vocoders. DMR is an ETSI standard widely used in business and amateur radio, using two-slot TDMA in a 12.5 kHz channel with the AMBE+2 vocoder. P25 dominates US public safety; DMR dominates commercial and ham digital.
  - q: How can I tell which protocol a system uses?
    a: Start with a database like RadioReference, which lists the system type for known systems. On the air, the modulation and channel width give clues, and decoder software identifies the protocol once it locks onto the control channel. GopherTrunk recognises the protocol from the control-channel signalling once it's decoding.
  - q: What is the difference between FDMA and TDMA?
    a: FDMA (frequency-division multiple access) gives each call its own frequency channel. TDMA (time-division multiple access) splits one frequency into time slots so several calls share it, alternating in rapid turns. P25 Phase 1 is FDMA; P25 Phase 2 and DMR are TDMA, which is how they fit more conversations into the same spectrum.
  - q: Are old analog trunking systems still around?
    a: Yes. Legacy systems like Motorola Type II, EDACS, LTR, and MPT 1327 still operate in places, often with analog voice but a data control channel for trunking. Newer deployments are overwhelmingly digital (P25, DMR), but a scanner enthusiast still encounters legacy systems, and GopherTrunk supports a range of them.
gophertrunk_links:
  - title: Status
    url: /status.html
    note: which protocols GopherTrunk decodes and how completely.
  - title: Hunt (discover systems)
    url: /hunt.html
    note: identify an unknown system from its control channel.
---

# The digital protocol landscape

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The digital airwaves are a patchwork of standards. **P25** dominates North-American
public safety (**Phase 1** = FDMA 12.5 kHz; **Phase 2** = TDMA, double capacity).
**DMR** rules business and amateur digital (two-slot TDMA, [AMBE+2](/learn/vocoders/)).
**NXDN/dPMR** are narrowband alternatives; **TETRA** is common in Europe. Older
**Motorola Type II, EDACS, LTR, MPT 1327** still linger. Most use
[4FSK](/learn/digital-modulation/); they differ in how they pack calls (**FDMA vs.
TDMA**) and signal their [control channel](/learn/what-is-trunking/). GopherTrunk
identifies the protocol once it locks on.
</div>

You understand [trunking](/learn/what-is-trunking/) in the abstract; this is the field
guide to the actual standards you'll meet, what's behind each name, and how they relate.
The [Status reference](/status.html) tracks GopherTrunk's coverage of them.

## FDMA vs. TDMA — the key axis

One distinction explains a lot of the family tree. **FDMA** (frequency-division) gives
each call its own frequency. **TDMA** (time-division) splits one frequency into rapid
**time slots**, so two or more calls share it by taking turns. TDMA doubles capacity in
the same spectrum, which is why newer standards adopt it. Keep this in mind as we go.

## P25 (Phase 1 and Phase 2)

**P25** is the open standard for **North-American public safety**:

- **Phase 1** — **FDMA**, 12.5 kHz channels, **[C4FM](/learn/digital-modulation/)** 4FSK,
  **[IMBE](/learn/vocoders/)** vocoder. The workhorse for police/fire.
- **Phase 2** — **TDMA**, two voice slots per channel for double capacity, a PSK-family
  modulation, **AMBE+2** vocoder.

If you're in the US tracking emergency services, P25 is what you'll see most.

## DMR (Tier II and Tier III)

**DMR** (Digital Mobile Radio) is an **ETSI** standard huge in **business and amateur**
radio. It's **two-slot TDMA** in a 12.5 kHz channel using **AMBE+2**:

- **Tier II** — conventional (non-trunked) two-slot DMR; very common commercially and on
  ham repeaters.
- **Tier III** — **trunked** DMR with a [control channel](/learn/what-is-trunking/).

DMR's low cost made it ubiquitous outside public safety.

## NXDN and dPMR

**NXDN** (used by Kenwood/Icom systems) and **dPMR** are **narrowband** standards (often
6.25 kHz) using 4FSK, found in business, transport, and utilities. They trade some audio
quality for very efficient spectrum use, and both come in conventional and trunked
flavours.

## TETRA

**TETRA** (Terrestrial Trunked Radio) is a European-rooted standard widespread in
**public safety and transport outside North America**. It uses a four-slot TDMA scheme
with π/4-DQPSK modulation and is a complete trunked system with strong feature support.

## Legacy systems

Plenty of older systems still run, frequently with **analog voice** but a **data control
channel** for trunking:

| System | Notes |
|--------|-------|
| Motorola Type II | Classic analog trunking, widespread legacy |
| EDACS | GE/Ericsson trunking |
| LTR | Logic Trunked Radio, simple business trunking |
| MPT 1327 | Analog trunking common outside the US |

They're fading as agencies migrate to P25/DMR, but a scanner enthusiast still meets
them.

## How to tell them apart

1. **Database first.** [RadioReference](/learn/finding-systems/) lists the system type for
   known systems in your area — by far the easiest route.
2. **On the air.** Channel width and [modulation](/learn/digital-modulation/) footprint on
   the [waterfall](/learn/fft-and-waterfall/) narrow it down; TDMA vs. FDMA behaviour
   (one slot vs. shared) is a clue.
3. **Let the decoder say.** Once GopherTrunk locks a [control
   channel](/learn/what-is-trunking/), the signalling identifies the protocol and system
   parameters for you.

| Standard | Access | Typical use | Vocoder |
|----------|--------|-------------|---------|
| P25 Phase 1 | FDMA | US public safety | IMBE |
| P25 Phase 2 | TDMA | US public safety (high capacity) | AMBE+2 |
| DMR | TDMA (2-slot) | Business, amateur | AMBE+2 |
| NXDN / dPMR | FDMA (narrow) | Business, utilities | AMBE+ variants |
| TETRA | TDMA (4-slot) | Public safety (EU and beyond) | ACELP |

<div class="knowledge-check" data-quiz data-correct-msg="Right — TDMA shares one frequency across time slots to carry more calls." markdown="0">
  <p class="knowledge-check__q">Quick check: P25 Phase 2 and DMR both use TDMA. What does TDMA let them do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Transmit on every frequency at once</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Fit more calls in the same spectrum via time slots</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Avoid needing a control channel</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The big axis is **FDMA** (a frequency per call) vs. **TDMA** (time slots share a frequency).
- **P25** = US public safety (Phase 1 FDMA, Phase 2 TDMA); **DMR** = business/amateur (2-slot TDMA).
- **NXDN/dPMR** are narrowband; **TETRA** is common in Europe; legacy Motorola/EDACS/LTR/MPT linger.
- Identify systems via a **database**, their **on-air footprint**, or by letting the
  decoder read the **control channel**.

Next: the reason some talkgroups stay silent no matter what — encryption.
