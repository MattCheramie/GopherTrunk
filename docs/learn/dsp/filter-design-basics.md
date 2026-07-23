---
slug: filter-design-basics
title: Filter design basics
description: Passband, stopband, transition width, and ripple — the specification you hand a filter designer, and how those choices trade off against each other.
keywords: filter design, passband stopband, transition band, ripple, filter order, filter specification, cutoff frequency, stopband attenuation
level: intermediate
status: full
prereq:
  - convolution-and-impulse-response
faq:
  - q: What is a filter specification?
    a: "A filter specification is the shape you want a filter's frequency response to have, written as a handful of numbers: where the passband ends, where the stopband begins, how flat the passband must be (ripple), and how deeply the stopband must attenuate. A design tool turns that spec into actual filter coefficients."
  - q: Why can't a filter just cut sharply at one frequency?
    a: "A perfectly sharp cutoff would require an infinitely long filter, which is impossible to build and adds infinite delay. Every real filter needs a transition band of finite width between passband and stopband. Making that transition narrower, or the stopband deeper, always costs more taps, more computation, and more delay — that is the fundamental tradeoff."
---

# Filter design basics

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Before you build a filter you **specify** it: the **passband** (frequencies to keep),
the **stopband** (frequencies to reject), the **transition band** between them, the
**ripple** allowed in the passband, and the **stopband attenuation** in dB. Tightening
any of those raises the filter's **order** (its length and cost). Design is choosing a
sensible point in that tradeoff — the spec you hand to a tool that computes the taps.
</div>

The last lesson showed that a filter *is* its [impulse response](/learn/dsp/convolution-and-impulse-response/).
This one is about deciding what response you want *before* you compute the taps for an
[FIR](/learn/dsp/fir-filters/) or [IIR](/learn/dsp/iir-filters/) filter. It is the
vocabulary of the filter specification.

## The anatomy of a filter response

Every frequency-selective filter has the same regions. Picture a low-pass response — keep
the lows, reject the highs:

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 170" role="img" aria-label="A low-pass filter response showing a flat passband with small ripple, a sloping transition band, and a low stopband, with the transition width and stopband attenuation marked." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="140" x2="500" y2="140" stroke="currentColor" stroke-opacity="0.3"/>
  <line x1="40" y1="20" x2="40" y2="140" stroke="currentColor" stroke-opacity="0.3"/>
  <text x="30" y="80" font-size="9" fill="currentColor" transform="rotate(-90 30 80)">gain</text>
  <text x="490" y="158" text-anchor="end" font-size="10" fill="currentColor">frequency &#8594;</text>
  <path d="M40 40 L180 40 L300 128 L500 128" fill="none" stroke="currentColor" stroke-width="2"/>
  <path d="M40 40 q 20 -6 40 0 q 20 6 40 0 q 20 -6 40 0" fill="none" stroke="currentColor" stroke-width="1" stroke-opacity="0.6"/>
  <text x="105" y="32" text-anchor="middle" font-size="9" fill="currentColor">passband (ripple)</text>
  <line x1="180" y1="45" x2="300" y2="45" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.6"/>
  <text x="240" y="60" text-anchor="middle" font-size="9" fill="currentColor">transition</text>
  <text x="400" y="120" text-anchor="middle" font-size="9" fill="currentColor">stopband</text>
  <line x1="470" y1="40" x2="470" y2="128" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.6"/>
  <text x="482" y="88" font-size="9" fill="currentColor">attenuation (dB)</text>
</svg>
<figcaption>A filter spec in one picture: a flat passband (with bounded ripple), a finite transition band, and a stopband at some attenuation depth.</figcaption>
</figure>

## The five numbers you hand a designer

| Term | What it sets |
|------|--------------|
| **Passband edge** | the highest frequency kept at (near) full gain |
| **Stopband edge** | the lowest frequency that must be fully rejected |
| **Transition width** | the gap between those two edges — narrower is harder |
| **Passband ripple** | how much the gain may wobble in the passband (dB) |
| **Stopband attenuation** | how far down rejected frequencies must sit (dB) |

Hand those to a design method — a windowed-sinc, Parks–McClellan, or an
[IIR](/learn/dsp/iir-filters/) prototype — and it returns the coefficients. You rarely
choose taps by hand; you choose the *spec*.

## The tradeoff you can't escape

These numbers pull against each other, and the currency they all spend is **filter
order** — the number of taps (FIR) or poles (IIR), which is compute cost and delay:

- A **narrower transition** costs more order.
- A **deeper stopband** costs more order.
- **Less ripple** costs more order.

You cannot have a razor-sharp cutoff, a 90 dB stopband, and a tiny cheap filter all at
once. Good design is picking the *loosest* spec that still does the job. For a scanner
channel filter, "flat across the channel, 60 dB down at the neighbour's edge, a modest
transition" is plenty — over-specifying just burns CPU that
[real-time processing](/learn/dsp/real-time-and-buffering/) can't spare.

## FIR or IIR for this spec?

The spec also steers the *type*. Need **linear phase** so a digital signal's shape
survives? That points to [FIR](/learn/dsp/fir-filters/). Need a steep cutoff with the
fewest operations and can tolerate nonlinear phase? An [IIR](/learn/dsp/iir-filters/)
does it with far fewer coefficients. The specification comes first; the choice of
structure follows from it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a narrower transition band demands a higher-order, costlier filter." markdown="0">
  <p class="knowledge-check__q">Quick check: making the transition band narrower requires…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">fewer taps and less computation</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">a higher filter order — more taps and more computation</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">a lower sample rate</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A filter is **specified** by passband, stopband, transition width, ripple, and attenuation.
- Those numbers are what you hand a design tool; it computes the **taps**.
- Every tightening — sharper transition, deeper stopband, less ripple — raises the **order** (cost).
- The spec also guides **FIR vs IIR**: linear phase points to FIR, minimal cost to IIR.

Next up: the FIR filter — the linear-phase workhorse that carves one channel out of a capture.
