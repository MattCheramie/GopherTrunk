---
slug: decibels
title: Decibels & signal power
description: Understand decibels for radio — dB, dBm, and dBFS explained simply. Learn gain, path loss, the noise floor, signal-to-noise ratio, and why every SDR measurement lives in logarithms.
keywords: decibels, dB, dBm, dBFS, signal power, noise floor, SNR, signal to noise ratio, gain, path loss, RF power
level: beginner
status: full
prereq:
  - radio-waves
faq:
  - q: What is the difference between dB, dBm, and dBFS?
    a: "dB is a ratio between two powers — it expresses a change, like \"6 dB of gain.\" dBm is an absolute power referenced to one milliwatt, so it names an actual signal strength, like \"-95 dBm.\" dBFS is decibels relative to digital full scale, used inside the SDR to describe how close a sample is to clipping the analog-to-digital converter; 0 dBFS is the maximum and real signals sit below it as negative numbers."
  - q: Why does radio use decibels instead of plain numbers?
    a: Radio signals span an enormous range — a strong local signal can be a trillion times more powerful than the faint one you're straining to hear. Decibels compress that range with a logarithm, so instead of writing 0.000000000001 watts you write a tidy -90 dBm. Logarithms also turn the multiplication of gains and losses along a signal path into simple addition.
  - q: What is the noise floor?
    a: The noise floor is the background level of random RF energy and receiver noise that's always present, measured in dBm. A signal must rise above the noise floor to be usable. The gap between your signal and the noise floor is the signal-to-noise ratio (SNR), and digital modes need a minimum SNR to decode cleanly.
  - q: Is a signal of -80 dBm stronger or weaker than -100 dBm?
    a: -80 dBm is stronger. Because these are negative numbers, the one closer to zero is larger. -80 dBm is 100 times (20 dB) more powerful than -100 dBm. A useful rule — every 10 dB is a 10x change in power, and every 3 dB is roughly a doubling or halving.
gophertrunk_links:
  - title: Tuning (receiver meters)
    url: /tuning.html
    note: read live power, noise floor, and SNR while you tune a control channel.
  - title: Gain, AGC & avoiding overload
    url: /learn/gain-and-agc/
    note: set gain by watching dBFS so you don't clip the ADC.
---

# Decibels & signal power

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **decibel (dB)** is a logarithmic way to express a ratio of power. **dBm** pins
that ratio to a real reference (1 milliwatt) so it names an absolute signal
strength; **dBFS** does the same against a digital maximum inside your SDR. Radio
uses decibels because signals span a trillion-to-one range and logarithms make
that manageable — and because gains and losses along a path simply **add up**.
The numbers that matter most: every **10 dB = 10× power**, every **3 dB ≈ 2×**,
and a signal is only usable when it sits well above the **noise floor**.
</div>

Decibels trip up almost every newcomer, because they're negative, logarithmic, and
come in confusing flavours. Spend ten minutes here and they'll click — after which
GopherTrunk's signal meters turn from noise into a clear story about whether a
signal is good enough to decode.

## Why does radio measure everything in decibels?

The signals you care about cover an absurd dynamic range. A handheld radio across
the street might deliver a billion times more power to your antenna than a repeater
on a distant hilltop. Writing those values in watts means juggling numbers like
`0.000000000002` — error-prone and unreadable.

A **decibel** solves this by taking the *logarithm* of the ratio between two powers:

> **dB = 10 × log₁₀(P₁ ÷ P₂)**

The logarithm squashes that trillion-to-one span into a friendly range of about
120 units. Two consequences fall out of this and they're the whole reason engineers
love decibels:

1. **Multiplication becomes addition.** An amplifier that multiplies power by 4 and
   a cable that cuts it in half become "+6 dB" and "-3 dB" — you just add them:
   net +3 dB. Tracing a signal through a chain of gains and losses is arithmetic you
   can do in your head.
2. **It matches how we perceive change.** A 10× jump in power "feels" like one step,
   whether you're going from weak to medium or medium to strong.

## What do the magic numbers 3 dB and 10 dB mean?

You don't need a calculator for most day-to-day decibel thinking — just two
anchors:

| Change in dB | Change in power | Mnemonic |
|--------------|-----------------|----------|
| +3 dB | ×2 (double) | "3 to double" |
| −3 dB | ÷2 (half) | |
| +10 dB | ×10 | "10 for ten-fold" |
| +20 dB | ×100 | |
| +30 dB | ×1000 | |

Combine them: +13 dB is ×10 then ×2 = ×20. −6 dB is half and half again = ¼. This
mental arithmetic is enough to reason about almost any antenna, amplifier, or cable
spec you'll meet.

## What is dBm, and how is it different from dB?

Plain **dB** is only a *ratio* — it tells you about a change, not an actual level.
"This amplifier gives 15 dB of gain" says nothing about how strong the output is
until you know the input.

**dBm** fixes a reference: it's decibels relative to **1 milliwatt**. That turns it
into an absolute unit of power, so it can name a real signal strength on its own:

| dBm | Power | Typical meaning |
|-----|-------|-----------------|
| 0 dBm | 1 mW | A reference point |
| −30 dBm | 1 µW | A very strong received signal |
| −80 dBm | 10 pW | A solid, easy-to-decode signal |
| −100 dBm | 0.1 pW | A weak signal, near many noise floors |
| −120 dBm | 1 fW | Often buried in the noise |

Two things to internalise. First, **received signals are negative** — they're tiny
fractions of a milliwatt. Second, **closer to zero is stronger**: −80 dBm beats
−100 dBm by 20 dB, which is 100× the power.

<div class="calc" data-calc="db" markdown="0">
  <p class="calc__title">Power ↔ dBm converter</p>
  <div class="calc__row">
    <label for="db-w">Power (watts)</label>
    <input id="db-w" type="number" data-watts value="0.001" min="0" step="any" inputmode="decimal">
  </div>
  <div class="calc__row">
    <label for="db-dbm">Power (dBm)</label>
    <input id="db-dbm" type="number" data-dbm step="any" inputmode="decimal">
  </div>
  <p class="calc__note">Edit either field. 1 W = 30 dBm; 1 mW = 0 dBm; a 5 W handheld transmits at +37 dBm.</p>
</div>

## What is the noise floor, and why does it decide what you can hear?

No receiver sees pure silence. There's always a background hiss — thermal noise in
the electronics plus random RF energy from the environment. That background level
is the **noise floor**, and like signal strength it's measured in dBm (often around
−110 to −90 dBm for an SDR, depending on bandwidth and surroundings).

A signal is only useful if it pokes up *above* the noise floor. The gap between the
two is the **signal-to-noise ratio (SNR)**, measured in dB:

> **SNR (dB) = signal level (dBm) − noise floor (dBm)**

A signal at −85 dBm with a noise floor at −105 dBm has 20 dB of SNR — plenty.
Digital voice modes each need a minimum SNR to decode; drop below it and the
decoder starts dropping symbols and you get gaps or silence. This is why, in
GopherTrunk's [tuning meters](/tuning.html), you watch not just the raw signal but
how far it stands above the noise.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 180" role="img" aria-label="A spectrum plot with a noisy baseline labelled noise floor and a tall peak labelled signal, with the gap between them labelled SNR." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="150" x2="500" y2="150" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="40" y1="20" x2="40" y2="150" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="14" y="28" font-size="11" fill="currentColor">dBm</text>
  <path d="M40 120 L70 124 L100 116 L130 122 L160 118 L185 119 L210 121 L240 117 L300 120 L340 118 L400 121 L450 117 L500 120" fill="none" stroke="currentColor" stroke-width="1.5" stroke-opacity="0.6"/>
  <line x1="40" y1="120" x2="500" y2="120" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.5"/>
  <text x="360" y="135" font-size="11" fill="currentColor">noise floor</text>
  <path d="M250 120 L262 50 L274 120 Z" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.5"/>
  <text x="262" y="44" text-anchor="middle" font-size="11" fill="currentColor">signal</text>
  <line x1="295" y1="50" x2="295" y2="120" stroke="currentColor" stroke-width="1.5"/>
  <text x="302" y="88" font-size="12" fill="currentColor">SNR</text>
</svg>
<figcaption>SNR is the height of the signal above the noise floor. Below a mode's minimum SNR, the decoder can't recover the bits.</figcaption>
</figure>

## What is dBFS, and where does it show up?

Inside the SDR, after the antenna signal is digitised, levels are measured against
the maximum number the **analog-to-digital converter (ADC)** can represent. That
scale is **dBFS** — decibels relative to **full scale**. Here **0 dBFS is the
ceiling**, and every real sample sits below it as a negative number, like −12 dBFS.

dBFS matters for one practical reason: if a signal (or the sum of all signals in
your bandwidth) reaches 0 dBFS, the ADC **clips** — it can't go higher, so the
waveform is flattened and distortion sprays across the spectrum, wrecking nearby
decodes. The fix is setting **[gain](/learn/gain-and-agc/)** so strong signals peak
comfortably below 0 dBFS. dBm describes the world outside the radio; dBFS describes
the digital world inside it.

## How do gain and path loss add up along a signal path?

Here's where decibels pay off. Walk a signal from antenna to decoder and every
stage is just a number you add:

```
  -70 dBm   antenna picks up the signal
  +20 dB    low-noise amplifier (LNA)
   -2 dB    coax cable loss
  ----------
  -52 dBm   at the SDR input
```

No multiplying microwatts — just running addition. The same logic covers **path
loss** (the signal weakening as it spreads from the transmitter, which can be 100+
dB over a few kilometres) and **antenna gain** (a directional antenna concentrating
energy, adding several dB in its favoured direction). Master this and you can
predict, roughly, whether a system will be receivable before you ever tune to it.

<div class="knowledge-check" data-quiz data-correct-msg="Exactly — +20 dB is two 10x steps, so 100x the power." markdown="0">
  <p class="knowledge-check__q">Quick check: how much more power is −60 dBm than −80 dBm?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">2× the power</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">20× the power</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">100× the power</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **dB** is a logarithmic *ratio* of power; **+10 dB = ×10**, **+3 dB ≈ ×2**.
- **dBm** is absolute power vs. 1 mW; received signals are negative, and closer to
  zero is stronger.
- **dBFS** is the digital scale inside the SDR; 0 dBFS is the clipping ceiling.
- The **noise floor** sets the bar a signal must clear; **SNR** is how far it clears
  it, and decoders need a minimum.
- Gains and losses along a path **add**, which is the whole point of decibels.

Next, we'll look at how the spectrum is divided into bands — and then start putting
information *onto* these waves with modulation.
