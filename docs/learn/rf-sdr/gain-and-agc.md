---
slug: gain-and-agc
redirect_from: /learn/gain-and-agc/
title: Gain, AGC & avoiding overload
description: How to set SDR gain correctly — the trade-off between sensitivity and overload, what ADC clipping looks like, reading dBFS headroom, manual gain vs automatic gain control (AGC), and a practical gain-setting routine for clean decoding.
keywords: SDR gain, set RTL-SDR gain, ADC clipping, overload, dBFS, headroom, AGC automatic gain control, gain too high, intermodulation, best gain setting
level: intermediate
status: full
prereq:
  - decibels
  - sdr-receiver
faq:
  - q: How do I set the gain on an SDR?
    a: Start low, raise the gain until weak signals appear and the signal-to-noise ratio stops improving, then back off slightly so strong signals don't push the ADC into clipping. The goal is the highest gain that keeps your strongest signal comfortably below 0 dBFS. Watch the dBFS level or the spectrum for distortion as you adjust.
  - q: What happens if SDR gain is too high?
    a: Too much gain drives the analog-to-digital converter past its maximum (0 dBFS), causing clipping. The waveform is flattened and distortion spreads across the spectrum as spurious signals and a raised noise floor — often making nearby channels undecodable. Strong nearby transmitters can also cause intermodulation. The fix is to reduce gain.
  - q: What happens if SDR gain is too low?
    a: Too little gain leaves weak signals buried in the receiver's own noise. The signal-to-noise ratio suffers and faint systems won't decode even though there's plenty of headroom at the ADC. The right gain is high enough to lift weak signals clear of the noise but not so high that strong ones clip.
  - q: Should I use AGC or manual gain?
    a: For decoding digital trunked systems, manual gain set once for your environment is usually best — it's predictable and avoids the AGC pumping up and down as signals come and go. AGC can be handy for casual listening across varied signals, but for a fixed monitoring setup a well-chosen manual gain gives the most consistent results.
gophertrunk_links:
  - title: Tuning (receiver meters)
    url: /tuning.html
    note: watch live level, dBFS, and SNR while setting gain.
  - title: Histogram (symbol)
    url: /histogram.html
    note: a clipped or starved signal shows in the symbol histogram too.
---

# Gain, AGC & avoiding overload

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Gain** controls how strong the signal is when it reaches the
[ADC](/learn/rf-sdr/sdr-receiver/). Too **low** and weak signals stay buried in noise; too
**high** and the ADC **clips** at 0 [dBFS](/learn/rf-sdr/decibels/), spraying distortion across
the band. The sweet spot is the **highest gain that keeps your strongest signal
comfortably below 0 dBFS**. Set it by ear-of-the-meters: raise until SNR stops
improving, then back off a touch. For a fixed monitoring rig, **manual gain usually
beats AGC**.
</div>

This is the setting beginners get wrong most, and it can make a perfectly good antenna
and signal undecodable. The good news: once you understand what gain does, getting it
right is a 30-second routine.

## What does the gain control actually do?

Gain is amplification — it boosts the signal somewhere along the
[receive chain](/learn/rf-sdr/sdr-receiver/) (front-end LNA, mixer, and/or baseband) before
the **ADC** digitises it. The ADC can only measure within a fixed range. Gain decides
**where your signals land inside that range**: turn it up and everything moves toward
the top; turn it down and everything moves toward the bottom (the noise).

The whole game is positioning your signals **high enough to clear the noise** but **not
so high they hit the ceiling**.

## Too low: weak signals lost in the noise

With gain too low, faint signals sit down near the receiver's own
[noise floor](/learn/rf-sdr/decibels/). Their [SNR](/learn/rf-sdr/decibels/) is poor, so even though
there's tons of unused ADC range above, the [demodulator](/learn/rf-sdr/demodulation-pipeline/)
can't reliably tell symbols apart. Symptom: weak/distant systems won't lock, while the
spectrum looks "quiet" with lots of headroom.

## Too high: ADC clipping and distortion

With gain too high, the strongest signal (or the *sum* of all signals in your
[bandwidth](/learn/rf-sdr/sample-rate-nyquist/)) reaches **0 dBFS** — the ADC's maximum — and
**clips**. The waveform's peaks are flattened, which mathematically generates harmonics
and **spurious signals** that spread across the spectrum and **raise the apparent noise
floor**. A strong nearby transmitter can also cause **intermodulation** — false signals
born from overload. Symptom: the band looks messy, ghost signals appear, and channels
that should decode don't.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 130" role="img" aria-label="Three small waveforms. Left: a small wave low near a noise line, labelled too low. Middle: a healthy wave well within the ceiling, labelled good. Right: a wave with flattened tops hitting the ceiling, labelled clipping." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none">
    <line x1="20" y1="20" x2="140" y2="20" stroke-dasharray="3 3" stroke-opacity="0.5"/>
    <line x1="160" y1="20" x2="280" y2="20" stroke-dasharray="3 3" stroke-opacity="0.5"/>
    <line x1="300" y1="20" x2="420" y2="20" stroke-dasharray="3 3" stroke-opacity="0.5"/>
    <path d="M20 95 q15 -10 30 0 q15 -10 30 0 q15 -10 30 0 q15 -10 30 0" stroke-width="1.3"/>
    <path d="M160 95 q15 -55 30 0 q15 -55 30 0 q15 -55 30 0 q15 -55 30 0" stroke-width="1.3"/>
    <path d="M300 95 q15 -80 22 -75 l16 0 q14 -5 22 75 q15 -80 22 -75 l16 0 q14 -5 22 75" stroke-width="1.3"/>
  </g>
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <text x="80" y="120">too low (noise)</text>
    <text x="220" y="120">good headroom</text>
    <text x="360" y="120">clipping (0 dBFS)</text>
    <text x="380" y="14">ceiling</text>
  </g>
</svg>
<figcaption>Gain positions the signal in the ADC's range. Aim for the middle: clear of the noise, well under the 0 dBFS ceiling.</figcaption>
</figure>

## Reading dBFS and keeping headroom

[dBFS](/learn/rf-sdr/decibels/) is the scale inside the SDR where **0 dBFS is the ceiling** and
real samples are negative. The practical target: set gain so your **strongest** signal
peaks somewhere comfortably below 0 — leaving **headroom** for bursts and the loudest
local transmitters. GopherTrunk's [tuning meters](/tuning.html) show level and SNR live,
so you can watch this directly rather than guess.

**Worked example.** Say your strongest local signal currently peaks at **−18 dBFS**.
That's 18 dB of headroom — plenty, and you could raise gain to pull up weaker signals.
Add 12 dB of gain and it now peaks at **−6 dBFS**: still safe, but close. Add another
10 dB and it hits **+4 dBFS** — impossible, so it **clips**, and the band fills with
[spurious signals](/learn/rf-sdr/sample-rate-nyquist/). The sweet spot here was the **−6 dBFS**
setting: weak signals lifted as far as possible while the strong one keeps a few dB of
margin below the ceiling.

## Manual gain vs. AGC

**AGC (automatic gain control)** adjusts gain on its own to chase the current signal.
That sounds ideal but, for decoding a fixed trunked system, AGC can **pump** — ramping
up during quiet moments (lifting the noise) and clamping down when a strong call hits —
which destabilises the very levels the decoder relies on.

For a stationary monitoring rig, **manual gain set once for your environment** is
usually the better choice: predictable and consistent. AGC earns its keep more for
casual hand-tuning across wildly varied signals.

## A practical gain-setting routine

1. Tune to your [control channel](/learn/rf-sdr/what-is-trunking/) (or strongest target).
2. Start with **low gain**.
3. Raise it step by step, watching **SNR** climb on the [tuning meters](/tuning.html).
4. Keep going until SNR **stops improving** — past this, you're mostly amplifying noise.
5. Check the spectrum for **distortion/ghost signals** and the level for **clipping**.
6. **Back off** a step or two to leave headroom. Lock it in.

If a strong local transmitter is overloading you, *lower* gain even if your target is
weak — clearing the overload often helps more than the lost gain costs.

<div class="knowledge-check" data-quiz data-correct-msg="Right — clipping at 0 dBFS sprays distortion across the band." markdown="0">
  <p class="knowledge-check__q">Quick check: the band suddenly shows ghost signals and a raised noise floor after you turned gain up. What happened?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The antenna got better</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The ADC is clipping — reduce gain</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The sample rate is too low</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Gain positions your signals in the **ADC's range**.
- **Too low** → buried in noise; **too high** → **clipping** and spurious signals.
- Aim for the **highest gain that keeps the strongest signal below 0 dBFS**.
- **Manual gain** is usually best for a fixed decoding setup; AGC can pump.
- Routine: raise until SNR plateaus, check for clipping, back off for headroom.

Next: choosing the hardware that all of this runs on.
