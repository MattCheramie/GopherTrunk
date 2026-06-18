---
slug: sdr-receiver
redirect_from: /learn/sdr-receiver/
title: How an SDR receiver works
description: Inside an SDR receiver — the signal chain from antenna through low-noise amplifier, mixer and local oscillator, to the analog-to-digital converter that turns radio waves into IQ samples, plus direct vs quadrature sampling.
keywords: SDR receiver, how SDR works, LNA, mixer, local oscillator, ADC, downconversion, direct sampling, quadrature sampling, RTL-SDR architecture
level: intermediate
status: full
prereq:
  - what-is-sdr
faq:
  - q: What are the main parts of an SDR receiver?
    a: An SDR receiver chains a few stages — front-end filtering and a low-noise amplifier (LNA) to boost the faint antenna signal, a mixer driven by a local oscillator that shifts the band of interest down to a lower frequency, and an analog-to-digital converter (ADC) that samples it into numbers. The output is a stream of IQ samples for software to process.
  - q: What does the mixer and local oscillator do?
    a: The mixer combines the incoming signal with a pure tone from the local oscillator (LO) to shift the frequency you care about down to a low "intermediate" frequency or to baseband, where the ADC can handle it. Tuning the SDR is really just changing the LO frequency so a different part of the spectrum lands in the ADC's range.
  - q: What is the difference between direct sampling and quadrature sampling?
    a: Direct sampling digitises the radio signal straight away with a fast ADC, no mixer — simple but it needs a very fast converter and is usually limited to lower frequencies. Quadrature sampling first mixes the signal down to baseband as two channels (I and Q), so a slower ADC can capture a chosen slice of spectrum at any tuned frequency. Most VHF/UHF SDRs use quadrature sampling.
  - q: Why does the front end need a low-noise amplifier?
    a: Signals at the antenna can be extremely weak, and every stage adds some noise. A low-noise amplifier boosts the signal early, before later stages can swamp it with their own noise, setting the receiver's sensitivity. Its noise figure largely determines how weak a signal the whole receiver can usefully detect.
gophertrunk_links:
  - title: Hardware guide
    url: /hardware.html
    note: how GopherTrunk's supported radios implement this chain.
  - title: Tuning (receiver meters)
    url: /tuning.html
    note: the live receiver telemetry from this signal chain.
---

# How an SDR receiver works

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An [SDR](/learn/rf-sdr/what-is-sdr/) receiver is a short chain: **front-end filter + low-noise
amplifier (LNA)** boost the faint antenna signal; a **mixer** driven by a **local
oscillator (LO)** shifts your chosen band down to a low frequency; and an
**analog-to-digital converter (ADC)** samples it into numbers — emitting
[IQ samples](/learn/rf-sdr/iq-data/). **Tuning is just changing the LO.** Most VHF/UHF SDRs
use **quadrature sampling** (mix to baseband, then digitise), while some use **direct
sampling** (digitise the RF straight away).
</div>

[Last lesson](/learn/rf-sdr/what-is-sdr/) said the SDR's job is to turn a slice of spectrum
into samples. Now let's open the box and see the stages that do it — because knowing
them makes [gain](/learn/rf-sdr/gain-and-agc/), [sample rate](/learn/rf-sdr/sample-rate-nyquist/), and
troubleshooting far more intuitive.

## The receive chain at a glance

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 110" role="img" aria-label="Signal chain: antenna, filter, low-noise amplifier, mixer fed by a local oscillator, then ADC, then IQ samples out." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <path d="M30 70 v-26 m-9 0 l9 -12 l9 12" fill="none" stroke="currentColor" stroke-width="2"/>
    <text x="30" y="88">antenna</text>
    <rect x="70" y="42" width="60" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="100" y="62">filter</text>
    <rect x="146" y="42" width="60" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="176" y="62">LNA</text>
    <rect x="222" y="42" width="60" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="252" y="62">mixer</text>
    <rect x="222" y="92" width="60" height="0" /><text x="252" y="100" font-size="9">LO</text>
    <circle cx="252" cy="92" r="3" fill="currentColor"/>
    <line x1="252" y1="76" x2="252" y2="89" stroke="currentColor" stroke-width="1.2"/>
    <rect x="320" y="42" width="60" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="350" y="62">ADC</text>
    <rect x="408" y="42" width="84" height="34" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="450" y="62">IQ samples</text>
    <g stroke="currentColor" stroke-width="1.2">
      <line x1="40" y1="59" x2="69" y2="59"/><line x1="130" y1="59" x2="145" y2="59"/><line x1="206" y1="59" x2="221" y2="59"/><line x1="282" y1="59" x2="319" y2="59"/><line x1="380" y1="59" x2="407" y2="59"/>
    </g>
  </g>
</svg>
<figcaption>The receive chain. Each stage has a job — and each is a place a signal can be lost or distorted, which is why this map helps troubleshooting.</figcaption>
</figure>

## Front-end filtering and the low-noise amplifier

The antenna delivers a faint, messy mix of everything in the band. A **front-end
filter** trims the worst out-of-band energy so it can't overload later stages. Then a
**low-noise amplifier (LNA)** boosts the wanted signal **early** — this matters because
every later stage adds noise, and you want the signal strong before that happens. The
LNA's quality (its *noise figure*) largely sets how weak a signal the whole receiver
can detect — its **sensitivity**.

## Mixing down with a local oscillator

The ADC can't directly digitise, say, an 850 MHz signal cheaply. So a **mixer**
combines the incoming signal with a pure tone from a **local oscillator (LO)**,
shifting your chosen band **down** to a low frequency the ADC can handle (baseband or a
low intermediate frequency). 

The beautiful part: **tuning the radio is just changing the LO frequency.** Want a
different part of the spectrum? Move the LO so that part lands in the ADC's window.
There are no mechanical dials — just a number.

**Worked example.** To receive an 851.0 MHz control channel, the tuner sets its LO so
that 851.0 MHz mixes down to baseband (0 Hz at the centre of the capture). To jump to
770.0 MHz instead, it simply moves the LO down by 81 MHz — the same hardware, a
different number. Whatever sits within ±half the [sample rate](/learn/rf-sdr/sample-rate-nyquist/)
of the LO appears in your captured band, ready for software to [channelise](/learn/rf-sdr/filtering-decimation/).

## The analog-to-digital converter (ADC)

The **ADC** measures the down-shifted signal millions of times a second, turning the
continuous wave into a stream of numbers. Two ADC properties shape everything
downstream:

- **Sample rate** — how fast it measures, which sets how much
  [bandwidth](/learn/rf-sdr/sample-rate-nyquist/) you capture.
- **Bit depth / range** — how finely and over what span it can measure, which is why
  [gain](/learn/rf-sdr/gain-and-agc/) must be set so strong signals don't exceed the ADC's
  ceiling (**clipping** at 0 dBFS).

## Direct vs. quadrature sampling

Two ways to get the signal into the ADC:

| Approach | How | Trade-off |
|----------|-----|-----------|
| **Quadrature sampling** | Mix down to baseband as two channels — **I and Q** — then digitise | Works at any tuned frequency with a modest ADC; the norm for VHF/UHF SDRs |
| **Direct sampling** | Digitise the RF immediately with a very fast ADC, no mixer | Simple, but needs a fast/expensive ADC and is usually limited to lower bands |

Quadrature sampling is why SDR output comes as **pairs** of numbers — the I and Q that
the [next lesson](/learn/rf-sdr/iq-data/) is all about.

## How this maps to real hardware

- An **RTL-SDR** uses a tuner chip (the mixer + LO) feeding the RTL2832U's ADC,
  outputting IQ — classic quadrature sampling across ~24 MHz–1.7 GHz. (Some support a
  direct-sampling mode for HF.)
- **Airspy** radios use higher-quality front ends and faster ADCs for better
  sensitivity and bandwidth; the **Airspy HF+** is optimised for the lower bands.
- **HackRF** is a wideband quadrature transceiver (it can transmit too).

The [Hardware guide](/hardware.html) and the [SDR hardware lesson](/learn/rf-sdr/sdr-hardware/)
cover which to choose; here, the point is that they all implement this same chain.

<div class="knowledge-check" data-quiz data-correct-msg="Right — tuning an SDR just changes the local oscillator frequency." markdown="0">
  <p class="knowledge-check__q">Quick check: what actually changes when you "tune" an SDR to a new frequency?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The antenna length</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The local oscillator frequency feeding the mixer</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The ADC's bit depth</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The chain is **filter → LNA → mixer (LO) → ADC → IQ samples**.
- The **LNA** sets sensitivity; **tuning = changing the LO**.
- The **ADC** sets capture bandwidth (sample rate) and the clipping ceiling.
- **Quadrature sampling** (mix to I/Q) is standard; **direct sampling** digitises RF
  directly.

Next: those pairs of numbers — what I and Q actually mean.
