---
slug: sampling-and-quantization
title: Sampling & quantization
description: Turning a continuous wave into samples — sample rate, the Nyquist limit, bit depth, and the aliasing and quantization noise that come with digitizing a signal.
keywords: sampling, quantization, sample rate, nyquist theorem, aliasing, bit depth, analog to digital, adc, dsp sampling
level: beginner
status: full
prereq:
  - what-is-dsp
faq:
  - q: What is the Nyquist theorem?
    a: The Nyquist–Shannon sampling theorem says you must sample at more than twice the highest frequency present in a signal to capture it faithfully. Sample slower and higher frequencies fold down and masquerade as lower ones — a corruption called aliasing. So to capture a 1 MHz-wide signal you need a sample rate above 2 MHz.
  - q: What is the difference between sample rate and bit depth?
    a: Sample rate is how often you measure the signal (samples per second), which sets the range of frequencies you can capture. Bit depth is how precisely you record each measurement (bits per sample), which sets how finely you can distinguish amplitudes and therefore the quantization noise floor. One is about time, the other about amplitude.
---

# Sampling & quantization

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Sampling** measures a wave at a fixed **sample rate**; the **Nyquist** limit says
that rate must exceed **twice** the highest frequency, or higher components **alias**
into false low ones. **Quantization** rounds each sample to a fixed **bit depth**,
adding a little **quantization noise**. Together they turn a continuous wave into the
number stream DSP works on.
</div>

Lesson 1 said DSP works on a stream of numbers. This lesson is how a real,
continuous wave *becomes* that stream — and the two ways digitizing can go wrong.

## Sampling: measuring on a clock

An analog-to-digital converter measures the signal at a steady tick — the **sample
rate**, in samples per second (often written S/s, or MS/s for millions). An SDR
dongle might run at 2.4 MS/s: 2.4 million measurements every second.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 130" role="img" aria-label="A smooth sine wave with evenly spaced sample points marked on it as dots." xmlns="http://www.w3.org/2000/svg">
  <line x1="10" y1="65" x2="510" y2="65" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M10 65 C 50 5, 90 5, 130 65 S 210 125, 250 65 S 330 5, 370 65 S 450 125, 490 65" fill="none" stroke="currentColor" stroke-width="1.5" stroke-opacity="0.6"/>
  <g fill="currentColor">
    <circle cx="10" cy="65" r="3"/><circle cx="50" cy="25" r="3"/><circle cx="90" cy="25" r="3"/><circle cx="130" cy="65" r="3"/><circle cx="170" cy="105" r="3"/><circle cx="210" cy="105" r="3"/><circle cx="250" cy="65" r="3"/><circle cx="290" cy="25" r="3"/><circle cx="330" cy="25" r="3"/><circle cx="370" cy="65" r="3"/><circle cx="410" cy="105" r="3"/><circle cx="450" cy="105" r="3"/><circle cx="490" cy="65" r="3"/>
  </g>
</svg>
<figcaption>Each dot is one sample. Take enough of them and the dots capture the wave; take too few and you lose it.</figcaption>
</figure>

## Nyquist: the speed limit

How fast is fast enough? The **Nyquist–Shannon theorem** gives the rule:

> Sample at **more than twice** the highest frequency in the signal.

Sample a 1 MHz-wide chunk of spectrum and you need above 2 MS/s. Sample too slowly
and a disaster called **aliasing** happens: frequencies above half the sample rate
"fold" back down and appear as *false* lower frequencies, indistinguishable from real
ones. It's the same effect that makes wagon wheels seem to spin backwards in film —
too few frames per second. This is why an SDR always [filters](/learn/dsp/fir-filters/)
before reducing its rate, a theme you'll meet again in
[decimation](/learn/dsp/decimation-and-resampling/). The
[sample rate & Nyquist](/learn/rf-sdr/sample-rate-nyquist/) lesson in the RF path
covers the same limit from the radio side.

## Quantization: rounding the amplitude

Sampling handles *when* you measure; **quantization** handles *how precisely*. Each
sample is rounded to one of a fixed number of levels set by the **bit depth**:

| Bit depth | Levels | Typical use |
|-----------|--------|-------------|
| 8-bit | 256 | RTL-SDR dongles |
| 12-bit | 4,096 | Airspy, better SDRs |
| 16-bit | 65,536 | Audio, high-end receivers |

More bits mean finer amplitude resolution and a lower noise floor. The rounding error
you can't avoid shows up as **quantization noise** — a faint hiss set by the bit
depth. It's one contributor to the noise floor you met in
[noise & SNR](/learn/rf-sdr/noise-and-snr/).

## The two knobs, together

Digitizing a signal is these two choices: **sample rate** (sets the frequency span
you can see) and **bit depth** (sets how finely you resolve amplitude). Get either
wrong and information is lost before DSP even begins — which is why understanding them
is the foundation for everything that follows.

<div class="knowledge-check" data-quiz data-correct-msg="Right — Nyquist requires sampling above twice the highest frequency, or aliasing occurs." markdown="0">
  <p class="knowledge-check__q">Quick check: to capture a signal containing frequencies up to 1 MHz, your sample rate must be…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">above 500 kHz</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">above 2 MHz</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">exactly 1 MHz</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Sampling** measures the wave at a fixed **sample rate**.
- **Nyquist**: sample above **twice** the highest frequency, or suffer **aliasing**.
- **Quantization** rounds each sample to a **bit depth**, adding **quantization
  noise**.
- Sample rate sets the frequency span; bit depth sets amplitude precision.

Next up: why SDR samples come in pairs — the complex I/Q representation.
