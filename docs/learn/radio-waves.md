---
slug: radio-waves
title: What is a radio wave?
description: A plain-language introduction to radio waves — frequency, wavelength, amplitude, and the electromagnetic spectrum. The one foundational idea every SDR and scanner skill builds on.
keywords: radio wave, what is a radio wave, frequency, wavelength, amplitude, electromagnetic spectrum, hertz, RF basics
level: beginner
status: full
faq:
  - q: What is a radio wave in simple terms?
    a: A radio wave is an invisible ripple of electric and magnetic energy that travels through space at the speed of light. A transmitter shakes electrons in an antenna, and that motion radiates outward as a wave that a distant antenna can pick up. Radio waves carry information — voice, data, video — by varying the wave's height (amplitude), rate (frequency), or timing (phase).
  - q: What is the difference between frequency and wavelength?
    a: Frequency is how many times per second a wave cycles, measured in hertz (Hz). Wavelength is the physical distance the wave covers in one cycle, measured in metres. They are inversely related — higher frequency means shorter wavelength. For radio you can convert between them with wavelength (m) = 300 / frequency (MHz).
  - q: How fast do radio waves travel?
    a: Radio waves travel at the speed of light — about 299,792,458 metres per second (roughly 300,000 km/s) in a vacuum, and very slightly slower in air. This is why you can think of "the speed of light" and "the speed of radio" as the same thing.
  - q: Why do I need to understand radio waves to use an SDR?
    a: Every setting on a software-defined radio — the frequency you tune to, the bandwidth you capture, the antenna you choose — is a direct consequence of how radio waves behave. Once frequency, wavelength, and amplitude click, the rest of SDR stops feeling like magic and starts feeling like cause and effect.
gophertrunk_links:
  - title: Hardware guide
    url: /hardware.html
    note: pick a dongle whose frequency range covers the waves you want to receive.
  - title: Tuning (receiver meters)
    url: /tuning.html
    note: watch live signal levels — amplitude — as you tune across a band.
---

# What is a radio wave?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **radio wave** is energy that travels through space as a vibrating electric and
magnetic field, moving at the speed of light. It is described by three numbers:
**frequency** (how fast it cycles, in hertz), **wavelength** (how long one cycle
is, in metres), and **amplitude** (how strong it is). Radio is just the slow,
long-wavelength part of the same electromagnetic spectrum that includes light.
Get these three numbers straight and every other SDR idea follows.
</div>

This is lesson 1 of the path, and it's the one everything else leans on. By the
end you'll be able to look at "460.025 MHz" and know roughly how long that wave
is, why it behaves the way it does, and how it connects to the dials you'll turn
in GopherTrunk.

## What exactly is a radio wave?

A radio wave is a ripple of **electromagnetic energy**. When a transmitter pushes
electrical current back and forth in an antenna, it creates a changing electric
field, which creates a changing magnetic field, which creates a changing electric
field — and that self-sustaining pattern races away from the antenna at the speed
of light. No wires, no medium required; radio waves cross empty space happily.

It's the same physics as visible light, microwaves, and X-rays. The only thing
that separates "radio" from "light" is how fast the field vibrates. Radio waves
vibrate slowly enough (thousands to billions of times per second) that we can
build electronics to generate and detect them directly — which is exactly what
your SDR does.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 160" role="img" aria-label="A sine wave showing one wavelength from crest to crest and amplitude as height from the centre line." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="80" x2="500" y2="80" stroke="currentColor" stroke-opacity="0.3" stroke-width="1"/>
  <path d="M20 80 C 70 0, 130 0, 180 80 S 290 160, 340 80 S 450 0, 500 80" fill="none" stroke="currentColor" stroke-width="2.5"/>
  <line x1="180" y1="30" x2="340" y2="30" stroke="currentColor" stroke-width="1.5" stroke-dasharray="4 3"/>
  <text x="260" y="22" text-anchor="middle" font-size="13" fill="currentColor">one wavelength (λ)</text>
  <line x1="100" y1="80" x2="100" y2="22" stroke="currentColor" stroke-width="1.5" stroke-dasharray="4 3"/>
  <text x="108" y="50" font-size="13" fill="currentColor">amplitude</text>
</svg>
<figcaption>One cycle of a wave: <strong>wavelength</strong> is the distance between repeats; <strong>amplitude</strong> is its height; <strong>frequency</strong> is how many cycles pass each second.</figcaption>
</figure>

## What is frequency, and why is it measured in hertz?

**Frequency** is how many complete cycles the wave makes every second. The unit is
the **hertz (Hz)** — one cycle per second. Radio frequencies are big numbers, so we
use prefixes:

| Unit | Cycles per second | You'll see it for |
|------|-------------------|-------------------|
| 1 kHz (kilohertz) | 1,000 | Long-wave broadcast, navigation |
| 1 MHz (megahertz) | 1,000,000 | FM radio, public-safety, scanners |
| 1 GHz (gigahertz) | 1,000,000,000 | Wi-Fi, GPS, cellular, satellite |

When you tell GopherTrunk to listen on **851.0125 MHz**, you're telling it to look
for a wave cycling about 851 million times every second. Tune to the wrong
frequency and you simply won't hear the signal — it's like dialing the wrong phone
number.

## What is wavelength, and how do I calculate it?

**Wavelength** (the Greek letter λ, "lambda") is the physical length of one cycle —
the distance from one crest to the next. Because all radio waves travel at the same
speed, frequency and wavelength are two sides of the same coin: the faster a wave
cycles, the less distance it covers per cycle, so the **shorter** its wavelength.

The exact relationship is *wavelength = speed of light ÷ frequency*. For everyday
radio there's a shortcut worth memorising:

> **λ (metres) ≈ 300 ÷ frequency (MHz)**

So a 150 MHz signal is about 2 metres long; a 460 MHz signal is about 0.65 m; a
1.2 GHz signal is just 25 cm. This isn't trivia — wavelength decides how big your
**[antenna](/learn/antennas/)** needs to be (antennas are sized to a fraction of a
wavelength) and how the wave bends around buildings and hills.

<div class="calc" data-calc="freq" markdown="0">
  <p class="calc__title">Frequency ↔ wavelength calculator</p>
  <div class="calc__row">
    <label for="rw-freq">Frequency</label>
    <input id="rw-freq" type="number" data-freq value="460" min="0" step="any" inputmode="decimal">
    <select data-freq-unit aria-label="Frequency unit">
      <option value="1000">kHz</option>
      <option value="1000000" selected>MHz</option>
      <option value="1000000000">GHz</option>
    </select>
  </div>
  <div class="calc__row">
    <label>Wavelength</label>
    <span class="calc__out" data-wavelength>—</span>
  </div>
  <p class="calc__note">Uses the exact speed of light (299,792,458 m/s). The “300 ÷ MHz” rule is a handy approximation of the same formula.</p>
</div>

## What is amplitude, and how does it relate to signal strength?

**Amplitude** is the height of the wave — how much energy it carries. A bigger
amplitude means a stronger signal. As a radio wave spreads out from a transmitter
and travels past obstacles, its amplitude shrinks, which is why a distant repeater
is harder to hear than a nearby one.

Your SDR doesn't measure amplitude in metres of wave height; it reports it as a
**power level** in decibels (you'll meet **[dBm and the noise floor](/learn/decibels/)**
in the next-but-one lesson). For now, hold onto the intuition: *amplitude = how
loud the signal is at your antenna*, and a digital decoder needs enough amplitude
above the background noise to recover the data.

## How does information ride on a radio wave?

A plain, unchanging wave — a **carrier** — carries no information by itself. To send
something, the transmitter deliberately *changes* one of the wave's three
properties in step with the message:

- Vary the **amplitude** → that's **AM** (and the basis of many data modes).
- Vary the **frequency** → that's **FM** (and digital cousins like FSK).
- Vary the **phase** (the wave's timing) → the basis of **PSK** and the digital
  voice modes GopherTrunk decodes.

That deliberate changing is called **modulation**, and it's the whole subject of
[Module 2](/learn/#signals-modulation). For now, just know that the radio wave is the
*envelope* and modulation is the *message written on it*.

## Where do radio waves sit in the bigger picture?

Radio is the low-frequency end of the **electromagnetic spectrum** — the same
family as microwaves, infrared, visible light, and X-rays, just vibrating far more
slowly. "Radio" conventionally spans about **3 kHz to 300 GHz**. Within that range,
governments carve the spectrum into **bands** and assign them to uses — broadcast,
aviation, public safety, amateur radio, satellite. That's the subject of the next
lesson, [Frequency, bands & the spectrum](/learn/frequency-and-spectrum/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — higher frequency means a shorter wave." markdown="0">
  <p class="knowledge-check__q">Quick check: a 900 MHz signal compared with a 150 MHz signal has a…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">longer wavelength</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">shorter wavelength</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">the same wavelength</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A radio wave is electromagnetic energy travelling at the speed of light.
- **Frequency** (Hz) is how fast it cycles; **wavelength** (m) is how long one cycle
  is; they're linked by *λ ≈ 300 ÷ MHz*.
- **Amplitude** is its strength, which your SDR reports as a power level.
- **Modulation** changes the wave to carry information.
- Radio is just the slow part of the same spectrum as light.

Next up: how that spectrum is divided into the bands you'll actually tune to.
