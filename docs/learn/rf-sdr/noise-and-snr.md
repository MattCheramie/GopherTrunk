---
slug: noise-and-snr
title: Noise, SNR & sensitivity
description: Why signal-to-noise ratio, not raw signal strength, decides whether a transmission decodes — the receiver noise floor, thermal noise (kTB), SNR in dB, receiver sensitivity and MDS, noise figure, and why more gain isn't more SNR.
keywords: SNR, signal to noise ratio, noise floor, receiver sensitivity, MDS, minimum discernible signal, noise figure, thermal noise, kTB, dBm, EVM, weak signal decoding
level: intermediate
status: full
prereq:
  - decibels
  - sdr-receiver
faq:
  - q: What is a good SNR for decoding?
    a: There's no single number, but digital voice systems typically need roughly 10–20 dB of signal-to-noise ratio to lock reliably, with the exact threshold depending on the protocol and how demanding its error correction is. What actually matters at the demodulator is a clean constellation and low symbol error (low EVM); SNR is the practical proxy you can read off a meter. Below the threshold you'll see intermittent locks and dropped voice; well above it, more SNR buys you little.
  - q: What is the noise floor?
    a: The noise floor is the background level of random energy present at the receiver even with no signal of interest — it sets the bottom of what you can hear. It comes from unavoidable thermal noise, man-made noise picked up by the antenna, and the receiver's own electronics. A signal has to rise above this floor by enough margin (its SNR) to decode.
  - q: Does more gain improve SNR?
    a: Only up to a point. Gain amplifies the signal and the noise together, so once a weak signal is already clear of the receiver's internal noise, adding gain doesn't change the ratio between them — it just moves both up toward the ADC ceiling. Past that point extra gain only risks overload. See the [gain lesson](/learn/rf-sdr/gain-and-agc/) for the full trade-off.
  - q: What is receiver sensitivity, or MDS?
    a: Sensitivity describes the weakest signal a receiver can usefully recover, often quoted as the minimum discernible signal (MDS) — the input power, in dBm, that produces a signal just detectable above the noise floor. It's set mainly by the receiver's bandwidth and its noise figure. A lower (more negative) MDS means a more sensitive receiver.
gophertrunk_links:
  - title: Tuning (SNR & level meters)
    url: /tuning.html
    note: read live SNR and level while you evaluate a signal.
  - title: Constellation
    url: /constellation.html
    note: a tight constellation (low EVM) is what good SNR buys you.
---

# Noise, SNR & sensitivity

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Every receiver sits on top of a **noise floor** — a background hiss from thermal
noise, the environment, and its own electronics. A signal decodes not because it's
strong in absolute terms but because it clears that floor by enough margin: its
**SNR** ([signal-to-noise ratio](/learn/rf-sdr/decibels/), measured in dB). **Sensitivity**
is how weak a signal a receiver can still pull out of the noise. Crucially, more
gain doesn't add SNR — see [gain & AGC](/learn/rf-sdr/gain-and-agc/).
</div>

Beginners naturally reach for "signal strength" as the thing that matters. It isn't. A
booming signal buried in an equally booming noise floor won't decode, while a faint one
sitting in silence will. This lesson explains the quantity that actually decides a
decode — SNR — and the receiver properties that set the floor beneath it.

## The noise floor

Point an antenna at empty spectrum and the receiver still shows a level. That's the
**noise floor**, and it has three sources.

The first is **thermal noise** — the random jostling of electrons in any conductor above
absolute zero. It's unavoidable and its power depends only on temperature and bandwidth,
captured by the expression **kTB** (Boltzmann's constant × temperature × bandwidth). At
room temperature this works out to roughly **−174 dBm/Hz** — the noise power in a single
hertz of bandwidth. That figure is the floor beneath the floor: no receiver can do
better.

The second is **man-made noise** the antenna picks up — switching power supplies, LED
lighting, chargers, motors, and general urban electrical hash. In a noisy environment
this often dominates the thermal contribution entirely.

The third is the **receiver's own noise**, added by its amplifiers and mixers (more on
that below).

Notice the bandwidth term in kTB: **the wider your receive bandwidth, the more noise
power you collect.** Doubling bandwidth adds about 3 dB of noise. This is why narrow,
matched filtering ahead of the [demodulator](/learn/rf-sdr/demodulation-pipeline/) helps —
you want the noise only from the channel you care about, not the whole band.

## Signal-to-noise ratio (SNR)

**SNR is the gap, in dB, between your signal and the noise floor.** If a signal peaks at
−80 dBm and the floor sits at −100 dBm, the SNR is 20 dB. That difference — not the −80
dBm by itself — is what the demodulator has to work with.

This is why absolute [dBm](/learn/rf-sdr/decibels/) is a poor predictor of success. A −40 dBm
signal sounds strong, but if local noise or overload has pushed the floor up to −45 dBm,
you have only 5 dB of SNR and it won't decode. Meanwhile a −95 dBm signal over a quiet
−115 dBm floor has a healthy 20 dB and locks fine. **The ratio wins, every time.**

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 150" role="img" aria-label="A spectrum trace with a low wavy noise floor line and a tall narrow peak rising above it. A vertical bracket between the noise line and the top of the peak is labelled SNR." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none">
    <path d="M20 110 q20 -6 40 0 q20 6 40 -2 q20 -4 40 2 q20 4 40 -3 q20 -3 40 2 q20 3 40 -1 q20 -4 40 2 q20 3 40 -1" stroke-width="1.2" stroke-opacity="0.7"/>
    <path d="M210 108 l6 -78 l6 78" stroke-width="1.4"/>
    <line x1="150" y1="112" x2="150" y2="30" stroke-dasharray="3 3" stroke-opacity="0.6"/>
    <line x1="146" y1="112" x2="154" y2="112" stroke-opacity="0.6"/>
    <line x1="146" y1="30" x2="154" y2="30" stroke-opacity="0.6"/>
  </g>
  <g font-size="10" fill="currentColor">
    <text x="126" y="74" text-anchor="middle" transform="rotate(-90 126 74)">SNR</text>
    <text x="300" y="126" font-size="10">noise floor</text>
    <text x="222" y="26" text-anchor="middle" font-size="10">signal</text>
  </g>
</svg>
<figcaption>What decodes a signal is its height above the noise floor (the SNR), not its absolute power.</figcaption>
</figure>

How much SNR do you need? It depends on the protocol, but as a rough guide **digital
voice systems need on the order of 10–20 dB** to lock cleanly. The real gate at the
receiver is the [constellation](/constellation.html) — the demodulated symbols have to land
close enough to their ideal positions (low error vector magnitude, EVM) for the error
correction to recover them. SNR is just the convenient meter reading that tracks it.

## Sensitivity & the minimum discernible signal (MDS)

**Sensitivity** is a receiver's ability to recover weak signals — how far down toward the
noise floor it can still hear. It's often quoted as the **minimum discernible signal
(MDS)**: the input power in dBm that produces an output just detectable above the noise,
conventionally the point where the signal equals the noise (a few dB of SNR).

MDS follows directly from the noise-floor discussion. It's roughly the thermal floor
(−174 dBm/Hz), plus 10·log₁₀ of the bandwidth in hertz, plus the receiver's noise figure,
plus whatever small SNR you define "discernible" to mean. Narrow the bandwidth or lower
the noise figure and the MDS drops (improves). A receiver specified at, say, −120 dBm MDS
in a given bandwidth is more sensitive than one at −110 dBm.

The practical upshot: sensitivity is not something you fix with the volume knob. It's set
by physics and by the front-end hardware.

## Noise figure — the receiver adds its own noise

No amplifier is silent. Every stage in the [receive chain](/learn/rf-sdr/sdr-receiver/) adds a
little noise of its own, degrading the SNR that arrives at its input. **Noise figure
(NF)** quantifies that penalty in dB — how much worse the SNR is coming out of a stage
than going in. An ideal noiseless stage has NF = 0 dB; real ones add anywhere from a
fraction of a dB (a good low-noise amplifier) to several dB.

The subtlety is that **the first stage matters most.** Once the first amplifier has set
the noise contribution, later stages add relatively little because the signal is already
amplified when it reaches them. That's why a clean, low-noise-figure first amplifier —
ideally right at the antenna — does more for weak-signal reception than anything
downstream. The [front-end lesson](/learn/rf-sdr/front-end-and-overload/) covers where that
amplifier belongs and how to avoid overloading it.

## Why more gain isn't more SNR

Here's the point that trips people up. **Gain multiplies the signal and the noise by the
same factor**, so it doesn't change the ratio between them. If a signal enters an
amplifier 12 dB above the noise, it leaves 12 dB above the noise — just louder.

SNR is essentially fixed by the time the signal reaches the amplifiers you control: it's
set by the antenna, the environment, and the first stage's noise figure. Turning gain up
past the point where the signal is already clear of the receiver's *internal* noise buys
nothing but risk — it pushes everything toward the ADC ceiling and invites
[clipping and overload](/learn/rf-sdr/gain-and-agc/). The right amount of gain lifts weak
signals clear of the ADC's own noise and then stops. Beyond that, SNR is what it is.

## Reading SNR in practice

GopherTrunk shows SNR live on its [tuning meters](/tuning.html) alongside the level in dBFS.
When you're evaluating whether a system is decodable, watch the **SNR**, not just the
level:

- **SNR climbing as you raise gain, then flattening** — normal. Once it plateaus, you've
  lifted the signal clear of internal noise; stop there (see the gain lesson).
- **High level but low SNR** — the noise floor is up with the signal. Suspect local
  interference or front-end overload, not a weak transmitter.
- **SNR comfortably in the double digits with a tight [constellation](/constellation.html)** —
  you're in good shape to decode.
- **SNR hovering at the threshold** — expect intermittent locks and dropped voice; you
  need more signal or less noise (a better antenna usually beats more gain).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a strong reading with poor SNR means the noise floor is nearly as high as the signal." markdown="0">
  <p class="knowledge-check__q">Quick check: a signal reads a strong −40 dBm but won't decode. Most likely cause?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The signal is too strong and needs attenuating to decode</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The noise floor is nearly as high — poor SNR</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">−40 dBm is below the receiver's sensitivity</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Every receiver has a **noise floor** from thermal noise (**kTB**, ~−174 dBm/Hz),
  man-made noise, and its own electronics.
- **Wider bandwidth collects more noise power** — about 3 dB per doubling.
- **SNR** — the dB gap between signal and floor — is what decides a decode, not absolute
  dBm.
- Digital voice typically needs roughly **10–20 dB SNR** (low EVM) to lock.
- **Sensitivity/MDS** is the weakest signal a receiver can hear, set by bandwidth and
  **noise figure**; the first amplifier matters most.
- **More gain doesn't add SNR** — it lifts signal and noise together. SNR is set before
  the ADC.

Next up: [antennas 101](/learn/rf-sdr/antennas/) — the first thing that decides how much signal, and how much noise, reaches your radio.
