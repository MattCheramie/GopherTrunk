---
slug: digital-modulation
title: Digital modulation & constellations
description: How digital radio puts bits on a carrier — FSK, PSK, QAM, and the C4FM/CQPSK modes behind P25 and DMR. Learn to read a constellation diagram and an eye diagram the way GopherTrunk shows them.
keywords: digital modulation, FSK, PSK, QPSK, QAM, C4FM, CQPSK, constellation diagram, eye diagram, symbols, 4FSK, P25 modulation, DMR modulation
level: intermediate
status: full
prereq:
  - radio-waves
faq:
  - q: What is digital modulation?
    a: Digital modulation is the process of putting digital data — ones and zeros — onto a radio carrier by switching the carrier between a fixed set of states. Each state, called a symbol, stands for one or more bits. The receiver measures which state arrived and maps it back to bits. Common families are FSK (switching frequency), PSK (switching phase), and QAM (switching both amplitude and phase).
  - q: What is a constellation diagram?
    a: A constellation diagram is a plot of a digital signal's symbols on a 2-D plane, where each point's position encodes the carrier's phase (angle) and amplitude (distance from centre). A clean signal shows tight, well-separated dots at the expected symbol positions; noise, mistuning, or distortion smears the dots, and when they blur together the decoder starts making errors.
  - q: What is the difference between a symbol and a bit?
    a: A bit is a single 0 or 1. A symbol is one transmitted state of the carrier, which can carry several bits at once. For example, 4-level FSK (used by P25 and DMR) has four symbols, so each symbol carries 2 bits. That's why a 4800-symbol-per-second signal moves 9600 bits per second.
  - q: What modulation do P25 and DMR use?
    a: "P25 Phase 1 uses C4FM, a four-level frequency-shift keying (4FSK) where the carrier sits at one of four frequency deviations per symbol. DMR uses 4FSK as well. P25 Phase 2 uses H-DQPSK, a phase-shift variant. GopherTrunk's constellation, symbol-scope, and eye-diagram panels let you see these symbols directly."
gophertrunk_links:
  - title: Constellation
    url: /constellation.html
    note: see the live IQ constellation of the signal you're decoding.
  - title: Symbol scope
    url: /symbol-scope.html
    note: watch the recovered symbol levels stream past in real time.
  - title: Eye diagram
    url: /eye-diagram.html
    note: judge timing and noise margin at a glance.
---

# Digital modulation & constellations

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Digital modulation** sends data by switching a carrier between a fixed set of
states called **symbols**, each carrying one or more **bits**. The three families
are **FSK** (switch the frequency), **PSK** (switch the phase), and **QAM** (switch
both phase and amplitude). A **constellation diagram** plots those symbols on a
2-D plane so you can *see* signal quality: tight, separated dots mean a clean
signal; smeared dots mean errors are coming. P25 and DMR use four-level FSK
(**C4FM**), which GopherTrunk visualises live in its constellation, symbol-scope,
and eye-diagram panels.
</div>

In [lesson 1](/learn/radio-waves/) you saw that information rides on a wave by changing
its amplitude, frequency, or phase. Analog modes vary those smoothly. **Digital**
modes do something cleaner: they snap the carrier to a few *predefined* states, so
the receiver only has to decide *which* state arrived — far more robust against
noise. This lesson is where the scopes in GopherTrunk finally make sense.

## What is a symbol, and how is it different from a bit?

A **bit** is a single 0 or 1. A **symbol** is one state the transmitter actually
puts on the air, and a symbol can stand for more than one bit at a time.

If a scheme has only two states, each symbol carries one bit. If it has four states,
each symbol carries two bits (00, 01, 10, 11). Eight states → three bits, and so on.
This is why the *symbol rate* (symbols per second, called **baud**) and the *bit
rate* (bits per second) are different numbers:

> **bit rate = symbol rate × bits per symbol**

P25 Phase 1 runs at **4800 baud** with **4 states (2 bits each)**, giving
**9600 bits per second**. That distinction — explored further in
[Symbols, baud & bitrate](/learn/symbols-and-baud/) — matters when you size capture
buffers and reason about how much SNR a mode needs.

## What are the three families of digital modulation?

Every digital mode you'll meet varies one of the wave's three properties — exactly
the three from lesson 1.

### FSK — frequency-shift keying

The carrier jumps between a set of **frequencies**, one per symbol. Two frequencies
gives **2FSK**; four gives **4FSK**. This is the workhorse of land-mobile digital
voice: **P25 Phase 1 (C4FM)** and **DMR** both use 4FSK, where the carrier sits at
one of four small frequency offsets (deviations) for each symbol. FSK is popular
because it tolerates the cheap, efficient amplifiers used in handhelds.

Concretely, C4FM's four symbols are four exact frequency **deviations** from the
channel centre — and each maps to a fixed 2-bit pair (a *dibit*):

| Deviation | Symbol level | Bits (dibit) |
|-----------|--------------|--------------|
| +1800 Hz | +3 | 01 |
| +600 Hz | +1 | 00 |
| −600 Hz | −1 | 10 |
| −1800 Hz | −3 | 11 |

When you watch the [symbol scope](/symbol-scope.html), those four levels are
literally these deviations; on the [constellation](/learn/iq-data/) they're four
points. A clean signal lands every symbol on one of the four — anything in between
is the receiver guessing.

### PSK — phase-shift keying

The carrier's **phase** (its timing offset) jumps between fixed angles while
amplitude stays constant. Two phases is **BPSK**; four is **QPSK** (2 bits/symbol).
**P25 Phase 2** uses a PSK variant. PSK is spectrally efficient and shows up
everywhere from satellites to digital broadcast.

### QAM — quadrature amplitude modulation

QAM varies **both phase and amplitude**, packing many states into the plane —
16-QAM (4 bits/symbol), 64-QAM, and higher. More bits per symbol means more data in
the same bandwidth, but the states sit closer together, so QAM needs a higher
[SNR](/learn/decibels/) to decode. You'll meet it in Wi-Fi, cable, and LTE rather than
scanner traffic, but the same constellation idea applies.

## What is a constellation diagram, and how do I read one?

A **constellation diagram** is the single most useful picture in digital radio. It
plots each received symbol as a point on a 2-D plane built from the SDR's
**[IQ data](/learn/iq-data/)**: the horizontal axis (**I**) and vertical axis (**Q**)
together encode the carrier's phase as the point's *angle* and its amplitude as the
point's *distance from the centre*.

A perfect signal places every symbol exactly on its ideal spot, so you see a few
tight clusters. Real signals scatter around those spots, and *how much* they
scatter is a direct read on quality:

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="Two constellation diagrams side by side. The left shows four tight clusters of dots labelled clean signal. The right shows four smeared clouds of dots overlapping, labelled noisy signal." xmlns="http://www.w3.org/2000/svg">
  <!-- clean -->
  <g>
    <line x1="20" y1="100" x2="180" y2="100" stroke="currentColor" stroke-opacity="0.3"/>
    <line x1="100" y1="20" x2="100" y2="180" stroke="currentColor" stroke-opacity="0.3"/>
    <g fill="currentColor">
      <circle cx="60" cy="60" r="2.5"/><circle cx="58" cy="62" r="2.5"/><circle cx="62" cy="59" r="2.5"/>
      <circle cx="140" cy="60" r="2.5"/><circle cx="138" cy="62" r="2.5"/><circle cx="141" cy="58" r="2.5"/>
      <circle cx="60" cy="140" r="2.5"/><circle cx="62" cy="138" r="2.5"/><circle cx="59" cy="141" r="2.5"/>
      <circle cx="140" cy="140" r="2.5"/><circle cx="138" cy="139" r="2.5"/><circle cx="142" cy="141" r="2.5"/>
    </g>
    <text x="100" y="196" text-anchor="middle" font-size="12" fill="currentColor">clean — easy to decode</text>
  </g>
  <!-- noisy -->
  <g>
    <line x1="280" y1="100" x2="440" y2="100" stroke="currentColor" stroke-opacity="0.3"/>
    <line x1="360" y1="20" x2="360" y2="180" stroke="currentColor" stroke-opacity="0.3"/>
    <g fill="currentColor" fill-opacity="0.7">
      <circle cx="320" cy="62" r="2.5"/><circle cx="312" cy="70" r="2.5"/><circle cx="328" cy="54" r="2.5"/><circle cx="318" cy="78" r="2.5"/><circle cx="332" cy="66" r="2.5"/>
      <circle cx="398" cy="60" r="2.5"/><circle cx="404" cy="72" r="2.5"/><circle cx="392" cy="52" r="2.5"/><circle cx="410" cy="64" r="2.5"/>
      <circle cx="322" cy="138" r="2.5"/><circle cx="314" cy="146" r="2.5"/><circle cx="330" cy="132" r="2.5"/><circle cx="318" cy="150" r="2.5"/>
      <circle cx="400" cy="140" r="2.5"/><circle cx="392" cy="148" r="2.5"/><circle cx="408" cy="132" r="2.5"/><circle cx="404" cy="150" r="2.5"/>
    </g>
    <text x="360" y="196" text-anchor="middle" font-size="12" fill="currentColor">noisy — errors creep in</text>
  </g>
</svg>
<figcaption>A 4-state (QPSK-like) constellation. Tight, separated clusters decode reliably; as noise, mistuning, or distortion smear them toward each other, the decoder starts guessing wrong.</figcaption>
</figure>

In GopherTrunk, the **[Constellation panel](/constellation.html)** draws this live.
When you're chasing a marginal signal, a constellation that's collapsing toward the
centre or rotating tells you exactly what's wrong — too little SNR, a tuning error,
or a clock problem — long before you'd guess it from the audio.

## What is an eye diagram, and what does it tell me?

Where the constellation shows symbols in the IQ *plane*, an **eye diagram** shows
them against *time*. It overlays many short snippets of the recovered signal so they
stack up, and the result has open "eye" shapes between the symbol levels. The
**wider and taller the eye opening**, the more margin the decoder has to sample each
symbol correctly. A closing eye means noise or timing jitter is eating that margin.

For the 4-level FSK in P25 and DMR you'll see **three stacked eyes** (four levels →
three gaps). GopherTrunk's **[Eye diagram](/eye-diagram.html)** and
**[Symbol scope](/symbol-scope.html)** panels let you watch this directly — together
with the constellation, they're your toolkit for [tuning a clean
lock](/learn/tuning-with-scopes/) later in the path.

<div class="knowledge-check" data-quiz data-correct-msg="Correct — 4 states means 2 bits per symbol, so 4800 × 2 = 9600 bps." markdown="0">
  <p class="knowledge-check__q">Quick check: a 4-level (4FSK) signal at 4800 baud carries how many bits per second?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">2400 bps</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">4800 bps</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">9600 bps</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Digital modulation switches the carrier between fixed **symbols**; each symbol
  carries one or more **bits**.
- **FSK** switches frequency (P25 C4FM, DMR), **PSK** switches phase (P25 Phase 2),
  **QAM** switches both for higher data rates.
- A **constellation** plots symbols in the IQ plane — tight clusters = clean,
  smeared = errors.
- An **eye diagram** plots them against time — a wide-open eye means healthy margin.
- GopherTrunk shows all three views live, which is how you diagnose a shaky signal.

Next in this module: how symbol rate and bit rate relate, and why it matters for
capture. Or jump ahead to [what software-defined radio actually
is](/learn/what-is-sdr/) to see where these symbols come from.
