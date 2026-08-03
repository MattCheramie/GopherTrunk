---
slug: digital-modulation-for-trunking
title: "Digital modulation for trunking: C4FM, π/4-DQPSK & CQPSK"
description: The handful of modulations digital trunking uses — C4FM and CQPSK for P25, π/4-DQPSK for TETRA, 4FSK for DMR and NXDN — and how each looks on a constellation and eye diagram.
keywords: C4FM, CQPSK, LSM, pi/4-DQPSK, 4FSK, P25 modulation, DMR modulation, NXDN modulation, TETRA modulation, constellation, eye diagram, simulcast modulation
level: intermediate
status: full
prereq:
  - digital-modulation
  - symbols-and-baud
faq:
  - q: What modulation does P25 Phase 1 use?
    a: P25 Phase 1 uses C4FM, a four-level frequency-shift keying. It runs at 4800 symbols per second, and because four levels carry two bits each, that yields 9600 bits per second. The same signal can also be produced with CQPSK, a linear-modulation cousin that fits the same constellation.
  - q: Why does P25 specify both C4FM and CQPSK?
    a: They produce a compatible on-air signal but suit different transmitters. C4FM uses a constant-envelope FM-style path that works with cheap, efficient nonlinear amplifiers in handhelds. CQPSK (also called LSM, Linear Simulcast Modulation) is a linear modulation that some simulcast systems use so overlapping copies combine more cleanly. But it is a choice, not a requirement — many simulcast systems transmit plain C4FM, so you cannot infer the modulation from the fact that a system is simulcast — you confirm it by which demodulator actually locks.
  - q: What is the difference between 4FSK and π/4-DQPSK?
    a: 4FSK shifts the carrier among four frequencies, one per symbol, and is used by DMR and NXDN as well as P25 Phase 1's C4FM. π/4-DQPSK is a phase-shift modulation used by TETRA that rotates the carrier phase in quarter-pi steps, encoding bits in the change of phase rather than its absolute value.
  - q: How can I see these modulations in GopherTrunk?
    a: The Constellation panel plots each received symbol in the IQ plane, so you can see the four points of C4FM or the ring of π/4-DQPSK, and the Eye diagram shows the symbol levels against time. Tight clusters and a wide-open eye mean a clean lock; smearing means errors are near.
gophertrunk_links:
  - title: Constellation
    url: /constellation.html
    note: see the live symbol constellation of the system you're decoding.
  - title: Eye diagram
    url: /eye-diagram.html
    note: judge timing and noise margin on the recovered symbols.
---

# Digital modulation for trunking: C4FM, π/4-DQPSK & CQPSK

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Digital trunking uses only a **handful of modulations**. **C4FM** is four-level FSK —
P25 Phase 1 runs it at **4800 symbols/s × 2 bits = 9600 bps**. **CQPSK** (also called
**LSM**) is a *linear* cousin that produces a compatible constellation and is used by
**simulcast** transmitters. **π/4-DQPSK** is the phase-shift modulation of **TETRA**,
and plain **4FSK** carries **DMR** and **NXDN**. Each has a recognisable
**constellation** and **eye diagram**, which GopherTrunk plots live so you can judge a
lock at a glance. Understanding these shapes is how you tell systems apart and tune a
clean signal.

</div>

You met the [three families of digital modulation](/learn/rf-sdr/digital-modulation/)
already — FSK, PSK, QAM — and the relationship between
[symbols and baud](/learn/rf-sdr/symbols-and-baud/). This lesson narrows that to the
*specific* modulations digital trunked systems actually use, because once you can name
the shape on the scope, you're halfway to identifying the system.

## C4FM — the P25 Phase 1 workhorse

**C4FM** (Continuous 4-level FM) is a **four-level FSK**: the carrier sits at one of
four small frequency deviations per symbol. Four levels means each symbol carries
**two bits** (a *dibit*). P25 Phase 1 runs C4FM at **4800 symbols per second**, so:

> 4800 symbols/s × 2 bits/symbol = **9600 bits per second**

The four deviations map to fixed dibits, the same scheme DMR's 4FSK uses:

| Deviation | Symbol level | Dibit |
|-----------|--------------|-------|
| +1800 Hz | +3 | 01 |
| +600 Hz | +1 | 00 |
| −600 Hz | −1 | 10 |
| −1800 Hz | −3 | 11 |

C4FM's appeal is that it is **constant-envelope** — the amplitude never changes, only
the frequency — so it works with the cheap, efficient *nonlinear* amplifiers in
handheld radios without distorting. On a [symbol scope](/learn/rf-sdr/digital-modulation/)
you see four stacked levels; on a constellation you see four points.

## CQPSK / LSM — the linear twin for simulcast

Here is the clever part of P25: it also defines **CQPSK** (Compatible QPSK), often
called **LSM** (Linear Simulcast Modulation). CQPSK is a *linear* modulation — it
shifts **phase**, not frequency — yet it is engineered to produce a signal that a
C4FM receiver can decode and that lands on the **same constellation**. A radio doesn't
care which one transmitted it.

Why bother with two ways to make the same signal? Because of **simulcast**: large
systems broadcast the same channel from many transmitters at once over an overlapping
area. A **linear** modulation gives the transmitters tighter control so those
overlapping signals combine cleanly rather than smear, so **some** simulcast operators
choose **CQPSK/LSM** while the constellation a receiver sees stays the same. It is not
mandatory, though — plenty of simulcast systems still run C4FM — so simulcast alone
doesn't tell you the modulation; the demodulator that locks does. (This is exactly why
simulcast distortion is its own decoding challenge, covered later in the path.)

## π/4-DQPSK and 4FSK — the rest of the field

Two more shapes round out digital trunking:

- **π/4-DQPSK** — used by **TETRA**. This is a *differential* phase-shift keying that
  rotates the carrier phase in steps of a quarter of pi, encoding bits in the **change**
  of phase between symbols rather than its absolute value. On a constellation it traces
  an eight-point ring; differential coding makes it forgiving of phase ambiguity.
- **4FSK** — the same four-level FSK family as C4FM, used by **DMR** and **NXDN**.
  DMR and P25 Phase 1 share the underlying 4FSK idea, which is why their constellations
  and eye diagrams look so alike even though the systems differ in every other respect.

<figure class="figure" markdown="0">
<svg viewBox="0 0 500 200" role="img" aria-label="Two constellations side by side. The left shows four points in a vertical column labelled C4FM and 4FSK. The right shows eight points arranged in a ring labelled pi over four DQPSK." xmlns="http://www.w3.org/2000/svg">
  <!-- 4FSK / C4FM -->
  <g>
    <line x1="30" y1="100" x2="200" y2="100" stroke="currentColor" stroke-opacity="0.3"/>
    <line x1="115" y1="20" x2="115" y2="180" stroke="currentColor" stroke-opacity="0.3"/>
    <g fill="currentColor">
      <circle cx="115" cy="40" r="4"/><text x="135" y="44" font-size="9">+3</text>
      <circle cx="115" cy="80" r="4"/><text x="135" y="84" font-size="9">+1</text>
      <circle cx="115" cy="120" r="4"/><text x="135" y="124" font-size="9">−1</text>
      <circle cx="115" cy="160" r="4"/><text x="135" y="164" font-size="9">−3</text>
    </g>
    <text x="115" y="196" text-anchor="middle" font-size="11" fill="currentColor">C4FM / 4FSK — 4 levels</text>
  </g>
  <!-- pi/4 DQPSK -->
  <g>
    <line x1="290" y1="100" x2="460" y2="100" stroke="currentColor" stroke-opacity="0.3"/>
    <line x1="375" y1="20" x2="375" y2="180" stroke="currentColor" stroke-opacity="0.3"/>
    <g fill="currentColor">
      <circle cx="375" cy="48" r="4"/><circle cx="412" cy="63" r="4"/><circle cx="427" cy="100" r="4"/><circle cx="412" cy="137" r="4"/>
      <circle cx="375" cy="152" r="4"/><circle cx="338" cy="137" r="4"/><circle cx="323" cy="100" r="4"/><circle cx="338" cy="63" r="4"/>
    </g>
    <text x="375" y="196" text-anchor="middle" font-size="11" fill="currentColor">π/4-DQPSK — 8-point ring</text>
  </g>
</svg>
<figcaption>C4FM and 4FSK stack four symbol levels (a vertical column on a symbol scope, four points on a constellation); π/4-DQPSK rotates phase around an eight-point ring. CQPSK is engineered to land on the same four-point pattern as C4FM.</figcaption>
</figure>

## Reading them on the scopes

Every one of these modulations is just a pattern of symbols, and GopherTrunk draws
those symbols live. On the **[Constellation panel](/constellation.html)** you'll see
the four points of C4FM/4FSK or the ring of π/4-DQPSK; tight, well-separated clusters
mean a clean signal, while smearing toward the centre signals low
[SNR](/learn/rf-sdr/decibels/), mistuning, or a clock problem. The
**[Eye diagram](/eye-diagram.html)** shows the same symbols against time — for the
four-level modes you'll see **three stacked eyes**, and the wider they open, the more
margin the decoder has.

These views are how you both **identify** a system and **tune** it: a P25 simulcast
signal that won't lock often shows a recognisable distortion on the constellation, and
the only way to spot that is to look. Later lessons on
[tuning with scopes](/learn/rf-sdr/tuning-with-scopes/) lean directly on what you've
just learned to read.

<div class="knowledge-check" data-quiz data-correct-msg="Correct — 4 levels carry 2 bits each, so 4800 × 2 = 9600 bps." markdown="0">
  <p class="knowledge-check__q">Quick check: P25 Phase 1 C4FM runs at 4800 symbols/s. What is its bit rate?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">4800 bps</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">9600 bps</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">19200 bps</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **C4FM** is four-level FSK; P25 Phase 1 runs it at **4800 symbols/s = 9600 bps**.
- **CQPSK / LSM** is a linear cousin that lands on the same constellation and is used
  by **simulcast** transmitters that need phase precision.
- **π/4-DQPSK** carries **TETRA**; plain **4FSK** carries **DMR** and **NXDN**.
- Each has a recognisable **constellation** and **eye diagram** that GopherTrunk plots
  live.
- Reading those shapes is how you **identify** a system and **tune** a clean lock.

Next, we look inside the bitstream itself: [framing, error correction and
interleaving](/learn/digital-trunking/framing-fec-interleaving/) — the structure that lets digital survive
fading.
