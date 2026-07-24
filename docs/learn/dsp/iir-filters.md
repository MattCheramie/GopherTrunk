---
slug: iir-filters
title: IIR filters
description: Infinite impulse response filters — feedback, poles and zeros, far fewer taps than FIR, and the stability and phase tradeoffs that come with them.
keywords: iir filter, infinite impulse response, feedback filter, biquad, poles and zeros, filter stability, iir vs fir, dsp iir
level: advanced
status: full
prereq:
  - fir-filters
faq:
  - q: What is the difference between FIR and IIR filters?
    a: A FIR filter computes each output only from inputs, so it has no feedback and is always stable, but needs many taps for a sharp response. An IIR filter feeds past outputs back in, so it achieves a similar sharpness with far fewer coefficients — at the cost of possible instability and non-linear phase. FIR is safe and precise; IIR is cheap and compact.
  - q: When would I choose an IIR filter over FIR?
    a: Choose IIR when compute or memory is tight and you need a sharp response for few operations — for example a simple audio tone control, DC blocking, or an envelope smoother. Choose FIR when you need linear phase (to avoid distorting a digital signal's shape) or guaranteed stability, which is why channel filters before a demodulator are usually FIR.
---

# IIR filters

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **IIR (infinite impulse response)** filter feeds **past outputs** back into the
computation. That feedback lets it match a FIR filter's sharpness with **far fewer
coefficients** — cheap and compact — but it can be **unstable** if designed carelessly
and generally lacks **linear phase**. The FIR-vs-IIR choice is precision and safety
versus efficiency.
</div>

FIR filters are safe but can be expensive. IIR filters are the efficient alternative,
with strings attached. This lesson covers the tradeoff.

## Feedback: the defining difference

A [FIR filter](/learn/dsp/fir-filters/) computes each output purely from inputs. An
**IIR** filter also mixes in its own **previous outputs**:

```text
output[n] = b0*x[n] + b1*x[n-1] + ...        (feed-forward, like FIR)
          - a1*output[n-1] - a2*output[n-2]  (feedback — the IIR part)
```

That feedback loop means a single input impulse can keep echoing through the output
essentially forever — hence **infinite** impulse response. It also means an IIR filter
can achieve a steep frequency response with only a handful of coefficients, where a FIR
would need dozens or hundreds.

## Poles, zeros, and stability

IIR filters are described by **poles** and **zeros** — the feedback and feed-forward
terms, respectively. The zeros carve notches; the poles create peaks and set the
sharpness. But poles are dangerous: place one wrong and the feedback amplifies its own
output each pass, and the filter **blows up** to infinity. A FIR filter can never do
this; an IIR filter must be designed to keep its poles "inside the unit circle" to stay
**stable**.

| | FIR | IIR |
|--|-----|-----|
| Feedback | none | yes |
| Stability | always stable | must be designed carefully |
| Phase | can be linear | generally non-linear |
| Cost for a sharp cutoff | many taps | few coefficients |

## The phase cost

Because of the feedback, IIR filters delay different frequencies by different amounts —
**non-linear phase**. For audio smoothing or a DC blocker that doesn't matter. But for
a digital signal about to be [demodulated](/learn/dsp/demodulation/), non-linear phase
smears the symbol transitions that carry the data — which is why the selective channel
filter in a decoder is almost always FIR, and IIR is reserved for less shape-critical
jobs.

## Where IIR earns its place

Common IIR uses in a radio pipeline:

- **DC blocking** — a one-pole high-pass that removes a constant offset from the I/Q
  stream, cheaply.
- **Envelope / power smoothing** — averaging a signal's power for an
  [AGC](/learn/dsp/gain-and-agc/) loop.
- **Simple tone shaping** on decoded audio.

The rule of thumb: reach for **IIR when efficiency matters and phase doesn't**, and
**FIR when phase linearity or guaranteed stability matters**.

<div class="knowledge-check" data-quiz data-correct-msg="Right — feedback lets IIR filters be sharp and cheap, but risks instability." markdown="0">
  <p class="knowledge-check__q">Quick check: what does an IIR filter use that a FIR filter does not?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Feedback from its own past outputs</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A higher sample rate</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Complex I/Q samples</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **IIR** filter feeds **past outputs** back in, giving an **infinite** impulse
  response.
- Feedback buys a **sharp response for few coefficients** — but risks **instability**
  (poles) and gives **non-linear phase**.
- **FIR** is chosen for linear phase and guaranteed stability (channel filtering);
  **IIR** for cheap jobs like DC blocking and smoothing.
- The tradeoff: precision and safety (FIR) versus efficiency (IIR).

Next up: changing a signal's sample rate safely — decimation and resampling.
