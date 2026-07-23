---
slug: mixing-and-downconversion
title: Mixing & downconversion
description: Multiplying by a local oscillator to shift a signal to baseband — the digital downconverter (DDC) that centres a channel at zero before it is decoded.
keywords: mixing, downconversion, digital downconverter, ddc, local oscillator, nco, frequency shift, baseband, dsp mixing
level: intermediate
status: full
prereq:
  - complex-signals-and-iq
faq:
  - q: What does mixing do in DSP?
    a: Mixing multiplies your signal by a complex sinusoid from a numerically controlled oscillator. Multiplying two sinusoids shifts frequencies, so this moves your chosen channel up or down the spectrum. Mixing a channel down so it sits at zero frequency is called downconversion, and it centres the channel before filtering and demodulation.
  - q: What is a numerically controlled oscillator?
    a: A numerically controlled oscillator (NCO) is software that generates a complex sine wave one sample at a time at a frequency you set — the digital equivalent of a local oscillator. You multiply the incoming I/Q by the NCO's output to shift the spectrum by exactly that frequency.
---

# Mixing & downconversion

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Mixing** multiplies the signal by a complex sinusoid from a **numerically
controlled oscillator (NCO)**, which **shifts** the whole spectrum in frequency.
**Downconversion** mixes your chosen channel down to **zero** so it sits at baseband,
ready to be [filtered](/learn/dsp/fir-filters/) and
[demodulated](/learn/dsp/demodulation/). Mix, filter, and resample together make a
**digital downconverter (DDC)**.
</div>

Your channel is somewhere off-centre in a wide capture. This lesson is how you slide
it to the middle — the first step of turning a channel into decoded data.

## Multiplying shifts frequency

Here's the key fact: **multiplying two sinusoids shifts frequencies**. Multiply your
signal by a complex sine wave of frequency f and every component of the signal moves by
f. Pick f to be *minus* your channel's offset from centre, and the channel lands
exactly at zero.

The complex sine wave comes from a **numerically controlled oscillator (NCO)** — software
that produces a rotating I/Q value one sample at a time at whatever frequency you dial
in. It's the digital version of the **local oscillator** in an analog radio.

```text
input channel at +200 kHz  ×  NCO at -200 kHz  =  channel now at 0 Hz
```

## Why move it to zero?

Downconverting to zero (**baseband**) makes everything after it simpler:

- The [low-pass filter](/learn/dsp/fir-filters/) that isolates the channel is a plain
  filter around zero — no need to design a band-pass at some arbitrary frequency.
- [Decimation](/learn/dsp/decimation-and-resampling/) can then shed the now-unneeded
  bandwidth.
- The [demodulator](/learn/dsp/demodulation/) works on a channel centred at zero,
  where phase and frequency changes are measured relative to nothing but the channel
  itself.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 120" role="img" aria-label="A spectrum with a channel off to the right of centre; after mixing, the same channel is centred at zero frequency." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="70" x2="230" y2="70" stroke="currentColor" stroke-opacity="0.3"/>
  <line x1="125" y1="70" x2="125" y2="60" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="125" y="88" text-anchor="middle" font-size="8" fill="currentColor">0</text>
  <line x1="185" y1="70" x2="185" y2="30" stroke="currentColor" stroke-width="2"/>
  <text x="185" y="24" text-anchor="middle" font-size="8" fill="currentColor">channel</text>
  <text x="270" y="70" text-anchor="middle" font-size="14" fill="currentColor">&#8594;</text>
  <text x="270" y="55" text-anchor="middle" font-size="8" fill="currentColor">mix</text>
  <line x1="300" y1="70" x2="510" y2="70" stroke="currentColor" stroke-opacity="0.3"/>
  <line x1="405" y1="70" x2="405" y2="60" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="405" y="88" text-anchor="middle" font-size="8" fill="currentColor">0</text>
  <line x1="405" y1="70" x2="405" y2="30" stroke="currentColor" stroke-width="2"/>
  <text x="405" y="24" text-anchor="middle" font-size="8" fill="currentColor">now centred</text>
</svg>
<figcaption>Mixing by the negative of the channel's offset slides it to zero, ready for filtering and decoding.</figcaption>
</figure>

## The digital downconverter (DDC)

Put the three operations together — **mix** to zero, **low-pass filter** to isolate,
**decimate** to the channel rate — and you have a **digital downconverter**, the
front door of nearly every software decoder. GopherTrunk has two distinct DDC
implementations for different jobs: a single-channel `Downconverter` used by the
replay path, and a multi-tap wideband `DDCBank` that extracts many channels from one
capture at once. They're separate code paths that happen to do the same conceptual
thing — a distinction worth remembering when you read the
[applied lesson](/learn/dsp/dsp-in-gophertrunk/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — mixing by the channel's negative offset shifts it to zero." markdown="0">
  <p class="knowledge-check__q">Quick check: to bring a channel sitting at +200 kHz down to zero, you mix it with an NCO at…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">+200 kHz</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">-200 kHz</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">+400 kHz</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Mixing** multiplies by an **NCO**'s complex sinusoid, **shifting** the spectrum.
- **Downconversion** mixes a channel to **zero** (baseband) for simpler downstream DSP.
- Mix + filter + decimate = a **digital downconverter (DDC)**.
- GopherTrunk has two DDCs — a single-channel one and a wideband bank — for different
  jobs.

Next up: recovering the actual message — demodulation.
