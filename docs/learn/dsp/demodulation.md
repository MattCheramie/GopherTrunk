---
slug: demodulation
title: Demodulation in code
description: Recovering the message from a carrier — AM, FM, and phase demodulation as arithmetic on I/Q samples, and how digital C4FM voice starts to appear.
keywords: demodulation, fm demodulation, am demodulation, phase demodulation, c4fm, fsk, iq demodulation, dsp demod, discriminator
level: intermediate
status: full
prereq:
  - mixing-and-downconversion
faq:
  - q: How do you demodulate FM from I/Q samples?
    a: FM carries information in the rate the phase changes. With complex I/Q samples the phase is the angle of each sample, so FM demodulation is measuring how much the angle rotates from one sample to the next — the change in angle per sample is proportional to the instantaneous frequency, which is the recovered signal. It's a few operations per sample.
  - q: What is C4FM?
    a: C4FM (compatible four-level FM) is the four-level frequency-shift modulation used by P25 Phase 1 and related systems. The transmitter shifts the frequency to one of four levels, each representing two bits, 4800 times a second. Demodulating it is FM demodulation followed by deciding which of the four levels each symbol landed on.
---

# Demodulation in code

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Demodulation** recovers the message riding on a carrier. With **I/Q** samples it's
arithmetic: **AM** is the sample's magnitude, **FM** is the rate its **phase** rotates
between samples, and **phase** demodulation reads the angle directly. The digital
voice GopherTrunk decodes — like **C4FM** — is FM demodulation followed by deciding
which symbol level each moment lands on.
</div>

You've isolated a channel at baseband. Now recover what's written on it. This lesson
turns the modulation types from the RF path into the actual math a decoder runs.

## Modulation, in reverse

Recall that a transmitter encodes information by varying a carrier's
[amplitude, frequency, or phase](/learn/rf-sdr/analog-modulation/). Demodulation
undoes each one — and because a baseband sample is a complex I/Q value (an arrow with
length and angle), each is a small calculation:

| Modulation | Information is in | Demodulate by |
|------------|-------------------|---------------|
| **AM** | amplitude | taking each sample's **magnitude** (arrow length) |
| **FM / FSK** | frequency | measuring **phase change** between samples |
| **PM / PSK** | phase | reading each sample's **angle** |

## FM demodulation: the phase-change trick

FM (and its digital cousin FSK) is the workhorse for voice and much digital radio, so
it's worth seeing concretely. Frequency *is* the rate phase changes. Since each I/Q
sample's angle is its phase, the recovered signal is simply **how much the angle turned
since the last sample**:

```text
sample n-1:  angle = 30 deg
sample n:    angle = 55 deg
-> phase advanced 25 deg -> proportional to instantaneous frequency
```

Do that every sample and the sequence of angle-changes *is* the demodulated FM — a few
multiplies and an arctangent per sample. This block is often called a **discriminator**.

## From FM to digital symbols

Digital voice like **C4FM** (used by P25 Phase 1) is FM with a twist: instead of a
continuously varying voice tone, the transmitter shifts the frequency to one of
**four discrete levels**, 4800 times a second, each level carrying two bits. So the
decoder:

1. **FM-demodulates** to get the instantaneous frequency (as above).
2. Watches it settle near one of four levels each symbol period.
3. **Decides** which level — that's two bits recovered.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 120" role="img" aria-label="A demodulated signal that steps between four horizontal levels, with dashed lines marking the four decision levels the decoder chooses among." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-opacity="0.3" stroke-dasharray="3 3"><line x1="20" y1="25" x2="500" y2="25"/><line x1="20" y1="50" x2="500" y2="50"/><line x1="20" y1="75" x2="500" y2="75"/><line x1="20" y1="100" x2="500" y2="100"/></g>
  <text x="508" y="28" font-size="8" fill="currentColor">+3</text><text x="508" y="53" font-size="8" fill="currentColor">+1</text><text x="508" y="78" font-size="8" fill="currentColor">-1</text><text x="508" y="103" font-size="8" fill="currentColor">-3</text>
  <path d="M20 50 H70 V25 H120 V100 H170 V75 H220 V25 H270 V50 H320 V100 H370 V50 H420 V75 H470" fill="none" stroke="currentColor" stroke-width="1.5"/>
</svg>
<figcaption>C4FM demodulated: the frequency steps between four levels; the decoder decides which level each symbol period holds, recovering two bits each.</figcaption>
</figure>

But there's a catch: to read "each symbol period," the decoder has to know *where* each
symbol begins — and it has no shared clock with the transmitter. Finding those symbol
boundaries is the next lesson, [clock & symbol recovery](/learn/dsp/clock-and-symbol-recovery/).
The RF path's [demodulation pipeline](/learn/rf-sdr/demodulation-pipeline/) walks the
same chain from the radio side.

<div class="knowledge-check" data-quiz data-correct-msg="Right — FM information is the rate of phase change between I/Q samples." markdown="0">
  <p class="knowledge-check__q">Quick check: to FM-demodulate an I/Q stream, you measure…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">the magnitude of each sample</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">how much the phase (angle) changes between samples</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">the bit depth of each sample</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Demodulation** undoes modulation: **AM** = magnitude, **FM** = phase-change rate,
  **PM** = angle.
- On **I/Q** samples each is a few operations — FM demod is a **discriminator**.
- Digital voice like **C4FM** is FM demod plus deciding which of four **symbol levels**
  each period holds.
- Reading symbols needs their boundaries — the job of clock recovery, next.

Next up: finding where each symbol begins without a shared clock.
