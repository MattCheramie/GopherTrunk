---
slug: what-is-a-signal
title: What is a signal?
description: Amplitude, phase, and the sine wave that underlies everything — the vocabulary of signals before we turn them into numbers a computer can process.
keywords: what is a signal, sine wave, amplitude phase frequency, phasor, signal basics dsp, sinusoid, signal vocabulary
level: beginner
status: full
prereq:
  - what-is-dsp
faq:
  - q: Why is the sine wave so important in signal processing?
    a: "The sine wave is the one signal that keeps its shape through any linear filter — it comes out a sine of the same frequency, only scaled and delayed. That property makes sines the natural building blocks of every other signal, which is exactly why the Fourier transform describes any signal as a sum of them."
  - q: What is the difference between amplitude, frequency, and phase?
    a: "Amplitude is how tall the wave is (its strength), frequency is how many cycles it completes each second (how fast it wiggles), and phase is where in its cycle it starts (its timing offset). Those three numbers fully describe a pure sine wave, and modulation works by deliberately changing one of them."
---

# What is a signal?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **signal** is a value that changes over time. The **sine wave** is its fundamental
building block, described by three numbers: **amplitude** (height/strength),
**frequency** (cycles per second), and **phase** (timing offset). A rotating arrow —
a **phasor** — captures all three at once, which is why signals map so neatly onto the
complex plane you'll meet next.
</div>

Before we turn signals into numbers, we need the vocabulary for talking about them.
This lesson is that vocabulary. It builds directly on
[what DSP is](/learn/dsp/what-is-dsp/) and sets up
[sampling](/learn/dsp/sampling-and-quantization/), where the wave becomes data.

## What counts as a signal?

A **signal** is simply any quantity that varies — usually over time. The voltage on an
antenna, air pressure at a microphone, the brightness of a pixel down a scan line: all
signals. In radio the signal we care about is the strength of an electromagnetic
[radio wave](/learn/rf-sdr/radio-waves/) measured at the receiver, moment to moment.

What makes a signal *useful* is that its variation carries information. A flat, unchanging
value says nothing; the changes are the message.

## The sine wave: the atom of signals

The most important signal is the **sinusoid** — a smooth, repeating wave. It matters
because it is the one shape a linear filter can never distort: a sine goes in, a sine of
the *same frequency* comes out, only scaled and shifted. Every other signal can be built
by adding sines together, an idea the [Fourier transform](/learn/dsp/the-fourier-transform/)
makes exact.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 160" role="img" aria-label="A sine wave with its amplitude marked as height from the centre line and one full period marked along the axis." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="80" x2="500" y2="80" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 80 C 70 0, 130 0, 180 80 S 290 160, 340 80 S 450 0, 500 80" fill="none" stroke="currentColor" stroke-width="2.5"/>
  <line x1="100" y1="80" x2="100" y2="22" stroke="currentColor" stroke-width="1.5" stroke-dasharray="4 3"/>
  <text x="108" y="48" font-size="12" fill="currentColor">amplitude</text>
  <line x1="180" y1="120" x2="340" y2="120" stroke="currentColor" stroke-width="1.5" stroke-dasharray="4 3"/>
  <text x="260" y="138" text-anchor="middle" font-size="12" fill="currentColor">one period (1 / frequency)</text>
</svg>
<figcaption>A sine wave: amplitude is its height, one period is how long a cycle takes, and frequency is how many periods pass each second.</figcaption>
</figure>

## The three numbers that describe it

A pure sine is fully pinned down by three quantities:

| Property | What it means | Change it and… |
|----------|---------------|----------------|
| **Amplitude** | height of the wave — its strength | the signal gets louder or weaker (**AM**) |
| **Frequency** | cycles per second, in hertz | the pitch/tuning shifts (**FM/FSK**) |
| **Phase** | where in the cycle it starts (its timing) | the wave slides left or right (**PSK**) |

Those three are exactly the properties a transmitter varies to send information —
[modulation](/learn/rf-sdr/analog-modulation/) is nothing more than deliberately changing
one of them in step with a message.

## The phasor: one arrow for all three

Here is the idea that makes everything later click. Picture the sine wave not as a
squiggle but as an arrow spinning around a circle. The arrow's **length** is the
amplitude, how fast it **spins** is the frequency, and the **angle** it starts at is the
phase. That spinning arrow is called a **phasor**, and its shadow on a horizontal line
traces out the sine wave.

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 160" role="img" aria-label="A circle with a rotating arrow from the centre; its length is amplitude and its angle is phase, and it projects onto a sine wave to the right." xmlns="http://www.w3.org/2000/svg">
  <circle cx="80" cy="80" r="55" fill="none" stroke="currentColor" stroke-opacity="0.35"/>
  <line x1="80" y1="80" x2="126" y2="48" stroke="currentColor" stroke-width="2"/>
  <path d="M110 80 A 30 30 0 0 0 122 57" fill="none" stroke="currentColor" stroke-opacity="0.7"/>
  <text x="112" y="74" font-size="9" fill="currentColor">phase</text>
  <text x="92" y="56" font-size="9" fill="currentColor">length = amp</text>
  <path d="M150 80 C 175 30, 205 30, 230 80 S 275 130, 290 80" fill="none" stroke="currentColor" stroke-width="1.8" stroke-opacity="0.8"/>
  <line x1="126" y1="48" x2="150" y2="48" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="3 3"/>
</svg>
<figcaption>A phasor: a spinning arrow whose length, spin rate, and starting angle are the amplitude, frequency, and phase. Its projection is the sine wave.</figcaption>
</figure>

This is the seed of complex I/Q. When the next unit represents a sample as a point on a
plane, it is just recording where that arrow points right now — a picture that becomes
[complex signals & I/Q](/learn/dsp/complex-signals-and-iq/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — amplitude, frequency, and phase fully describe a pure sine wave." markdown="0">
  <p class="knowledge-check__q">Quick check: which three numbers fully describe a pure sine wave?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Bit depth, sample rate, and gain</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Amplitude, frequency, and phase</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Voltage, current, and resistance</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **signal** is a value that changes over time; its changes carry the information.
- The **sine wave** is the fundamental building block — the one shape filters don't distort.
- **Amplitude**, **frequency**, and **phase** fully describe a sine; modulation varies one of them.
- A **phasor** — a spinning arrow — captures all three, and previews the complex plane.

Next up: how a continuous wave becomes a stream of numbers a computer can hold.
