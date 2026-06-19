---
slug: identifying-the-system
title: Identifying what you're hearing
description: Tell trunked systems apart before you decode — read channel bandwidth on the waterfall, symbol rate, constellation shape, and audio cadence to recognise P25, DMR, NXDN and TETRA at a glance.
keywords: identify trunked system, channel bandwidth, 6.25 kHz, 12.5 kHz, 25 kHz, symbol rate, constellation shape, waterfall, P25 vs DMR, NXDN, TETRA, system recognition
level: intermediate
status: full
prereq:
  - digital-modulation-for-trunking
  - finding-the-control-channel
faq:
  - q: Can I tell which system I'm hearing before decoding it?
    a: Often yes. Channel bandwidth on the waterfall, the symbol rate, the shape of the constellation, and the cadence of the audio all give the system away. A 12.5 kHz carrier with a four-point constellation is likely P25 Phase 1 or DMR; a 6.25 kHz carrier points to NXDN or P25 Phase 2; a wider 25 kHz channel with a phase ring suggests TETRA. These clues narrow the field before you commit to a decoder.
  - q: What does channel bandwidth tell me about a system?
    a: Bandwidth reflects how much spectrum each carrier occupies, which is set by the standard. P25 Phase 1 and DMR use 12.5 kHz channels; NXDN and P25 Phase 2 squeeze into 6.25 kHz; older analog and some wideband systems use 25 kHz; TETRA uses 25 kHz shared by four time slots. Measuring the carrier width on the waterfall is the fastest first cut.
  - q: How does the constellation help me identify a system?
    a: Different modulations draw different shapes. C4FM and 4FSK show four horizontal levels or a four-point pattern; π/4-DQPSK draws a ring of eight phase points; CQPSK shows a tighter four-corner QPSK pattern. The constellation, eye diagram and histogram together confirm the modulation family, which maps onto a short list of candidate systems.
  - q: Why bother identifying before decoding?
    a: Picking the wrong decoder wastes time and can make a perfectly good signal look broken. A quick visual identification — bandwidth, symbol rate, constellation, cadence — tells you which decoder to try first, so you lock faster and avoid chasing a non-existent fault. It is the difference between a confident guess and trial-and-error.
gophertrunk_links:
  - title: Constellation
    url: /constellation.html
    note: read the symbol pattern to spot the modulation family.
  - title: Eye diagram
    url: /eye-diagram.html
    note: count symbol levels and judge timing on an unknown signal.
  - title: Histogram
    url: /histogram.html
    note: see how received levels cluster to confirm 2-level vs 4-level.
---

# Identifying what you're hearing

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
You can often name a system *before* decoding it by reading four clues. **Channel
bandwidth** on the [waterfall](/learn/rf-sdr/fft-and-waterfall/) — **6.25, 12.5, or
25 kHz** — is the fastest first cut. **Symbol rate** and the **constellation shape**
narrow the modulation family: a four-level pattern means **C4FM or 4FSK** (P25, DMR),
a phase ring means **π/4-DQPSK** (TETRA). The **audio cadence** — bursty TDMA versus a
continuous stream — confirms it. GopherTrunk's [constellation](/constellation.html),
[eye diagram](/eye-diagram.html) and [histogram](/histogram.html) make every one of
these visible, so identification becomes a glance rather than a guess.
</div>

Before GopherTrunk can follow a system, it needs to know *which* system. Sometimes a
database tells you; often you're staring at an unknown carrier on the waterfall and
have to work it out. The good news is that digital trunking uses only a [handful of
modulations](digital-modulation-for-trunking/), and each leaves fingerprints you can
read with the scopes.

## Clue 1 — channel bandwidth on the waterfall

The width of a carrier on the [FFT and waterfall](/learn/rf-sdr/fft-and-waterfall/) is
fixed by the standard, so it's the quickest first cut. Zoom in until a single carrier
fills the display and estimate how many kilohertz of spectrum it occupies:

- **6.25 kHz** — a narrow carrier. Think **NXDN** or **P25 Phase 2** (which packs two
  voice channels into this width using TDMA).
- **12.5 kHz** — the workhorse width. Think **P25 Phase 1** or **DMR** (DMR fits two
  TDMA slots here).
- **25 kHz** — wide. Older analog/wideband channels, or **TETRA** (one 25 kHz carrier
  shared by four time slots).

Bandwidth alone rarely pins the exact system, but it eliminates most candidates in a
second. A 6.25 kHz carrier is never P25 Phase 1; a 25 kHz carrier is never DMR.

## Clue 2 — symbol rate and constellation shape

Now look at the recovered symbols on the [constellation](/constellation.html) and
[eye diagram](/eye-diagram.html). The *symbol rate* sets how fast the eye opens and
closes, and the *shape* reveals the modulation family:

- **Four horizontal levels** (or a four-point cluster) — **4-level FSK**: C4FM (P25
  Phase 1) or 4FSK (DMR, NXDN). The [histogram](/histogram.html) shows four distinct
  peaks.
- **A ring of phase points** — **π/4-DQPSK** (TETRA): the constellation rotates in
  quarter-pi steps and traces a rosette rather than fixed levels.
- **A tight four-corner QPSK box** — **CQPSK/LSM**, the linear cousin P25 simulcast
  transmitters use.

A clean lock draws tight clusters and a wide-open eye; smearing means you're close to
the noise floor — useful information, but don't mistake a weak-signal smear for the
wrong modulation. The [histogram](/histogram.html) is the tie-breaker: count the
peaks, and a two-versus-four-level question answers itself.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 170" role="img" aria-label="Three constellation sketches side by side: four horizontal levels labelled 4-level FSK, a ring of phase points labelled pi over 4 DQPSK, and a four-corner box labelled CQPSK." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <!-- 4FSK -->
    <text x="90" y="22" font-weight="600">4-level FSK</text>
    <text x="90" y="36" font-size="8.5">C4FM / 4FSK — P25 P1, DMR, NXDN</text>
    <line x1="40" y1="130" x2="140" y2="130" stroke="currentColor" stroke-opacity="0.4"/>
    <g fill="currentColor"><circle cx="90" cy="62" r="4"/><circle cx="90" cy="86" r="4"/><circle cx="90" cy="110" r="4"/><circle cx="90" cy="50" r="4"/></g>
    <!-- DQPSK -->
    <text x="270" y="22" font-weight="600">π/4-DQPSK</text>
    <text x="270" y="36" font-size="8.5">TETRA — phase ring</text>
    <g fill="currentColor"><circle cx="270" cy="55" r="3.5"/><circle cx="300" cy="68" r="3.5"/><circle cx="313" cy="98" r="3.5"/><circle cx="300" cy="128" r="3.5"/><circle cx="270" cy="141" r="3.5"/><circle cx="240" cy="128" r="3.5"/><circle cx="227" cy="98" r="3.5"/><circle cx="240" cy="68" r="3.5"/></g>
    <!-- CQPSK -->
    <text x="450" y="22" font-weight="600">CQPSK / LSM</text>
    <text x="450" y="36" font-size="8.5">P25 simulcast — QPSK box</text>
    <g fill="currentColor"><circle cx="420" cy="68" r="4"/><circle cx="480" cy="68" r="4"/><circle cx="420" cy="128" r="4"/><circle cx="480" cy="128" r="4"/></g>
  </g>
</svg>
<figcaption>The constellation gives away the modulation family at a glance: four levels for FSK systems, a rosette ring for TETRA's π/4-DQPSK, a four-corner box for CQPSK simulcast.</figcaption>
</figure>

## Clue 3 — audio and burst cadence

Even without decoding, the *rhythm* of a signal helps. **FDMA** systems (P25 Phase 1,
NXDN, conventional DMR carriers) put one continuous transmission on the channel for the
duration of a call. **[TDMA](tdma-vs-fdma/)** systems (P25 Phase 2, DMR, TETRA) chop the
carrier into time slots, so on the waterfall and in the symbol stream you see a
*pulsing* pattern — bursts with gaps — even during a single conversation. A steady
solid trace versus a stuttering one is a strong TDMA-versus-FDMA tell.

## Putting the clues together

No single clue is decisive, but two or three together usually name the system. Use this
recognition table as a field guide:

| System | Channel bandwidth | Modulation | Access | Telltale |
|--------|-------------------|------------|--------|----------|
| **P25 Phase 1** | 12.5 kHz | C4FM (or CQPSK simulcast) | FDMA | Four levels, continuous trace, 4800 sym/s |
| **P25 Phase 2** | 6.25 kHz equiv. | H-CPM / TDMA | TDMA (2-slot) | Narrow channel, pulsing bursts |
| **DMR** | 12.5 kHz | 4FSK | TDMA (2-slot) | Four levels *and* slotting on one 12.5 kHz carrier |
| **NXDN** | 6.25 / 12.5 kHz | 4FSK | FDMA | Very narrow carrier, four levels, continuous |
| **TETRA** | 25 kHz | π/4-DQPSK | TDMA (4-slot) | Phase-ring constellation, wide channel, strong slotting |
| **Analog / wideband** | 25 kHz | FM | — | No symbol structure; fuzzy noise-like spectrum |

Read it top to bottom: bandwidth eliminates rows, the constellation shape confirms the
modulation, and the burst cadence settles TDMA versus FDMA. For the protocol details
behind each row, the [protocol landscape](/learn/rf-sdr/protocol-landscape/) lesson is
the deeper reference.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a 12.5 kHz carrier with four levels and visible time-slot bursts is DMR." markdown="0">
  <p class="knowledge-check__q">Quick check: you see a 12.5 kHz carrier with a four-level constellation that pulses in bursts during a single call. Best guess?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">TETRA</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">DMR</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Analog wideband FM</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Channel bandwidth** (6.25 / 12.5 / 25 kHz) on the waterfall is the fastest first cut.
- The **constellation shape** names the modulation: four levels for FSK, a ring for
  π/4-DQPSK, a box for CQPSK.
- The **[histogram](/histogram.html)** confirms 2-level versus 4-level by counting peaks.
- **Burst cadence** separates pulsing **TDMA** systems from continuous **FDMA** ones.
- Two or three clues together usually name the system before you decode a single bit.

Next, we'll find the one frequency worth monitoring — locating and confirming the
[control channel](finding-the-control-channel/).
