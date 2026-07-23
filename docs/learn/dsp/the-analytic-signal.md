---
slug: the-analytic-signal
title: The analytic signal & Hilbert transform
description: How a real signal becomes complex I/Q — the Hilbert transform, the analytic signal, and why it removes the negative-frequency mirror image.
keywords: analytic signal, hilbert transform, quadrature, negative frequency, complex baseband, real to complex, 90 degree phase shift
level: advanced
status: full
prereq:
  - complex-signals-and-iq
  - the-fourier-transform
faq:
  - q: What does the Hilbert transform actually do?
    a: "The Hilbert transform shifts every frequency component of a real signal by exactly 90 degrees. Used as the imaginary (quadrature) part alongside the original real signal, it produces the analytic signal — a complex signal whose spectrum has no negative-frequency half, so it can be represented and processed as clean I/Q."
  - q: Why do we want to get rid of negative frequencies?
    a: "A real signal's spectrum is always mirror-symmetric: every positive frequency has an identical negative-frequency twin carrying no new information. That mirror is what makes a real signal ambiguous about which side of a carrier it sits on. Removing it with the analytic signal leaves one clean copy, which is exactly what lets an SDR tell a signal above centre from one below."
---

# The analytic signal & Hilbert transform

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **real** signal's spectrum is always **mirror-symmetric** — every positive frequency
has a redundant **negative-frequency** twin. The **Hilbert transform** phase-shifts the
signal by 90° to make the **quadrature** part; pairing it with the original gives the
**analytic signal**, a complex signal whose negative-frequency half is **cancelled**.
That is how a one-wire real signal becomes clean [I/Q](/learn/dsp/complex-signals-and-iq/).
</div>

The I/Q lesson took two-number samples for granted. This one answers where the second
number *comes from* when a signal starts life as a single real stream. It builds on
[complex signals & I/Q](/learn/dsp/complex-signals-and-iq/) and the
[Fourier transform](/learn/dsp/the-fourier-transform/).

## The mirror in every real spectrum

Take the Fourier transform of any purely **real** signal and you always find the same
thing: the spectrum is symmetric about zero. Whatever sits at +5 kHz has an identical
mirror image at −5 kHz. That twin carries no extra information — it is a mathematical
consequence of the signal being real — but it is what makes a lone real stream unable to
say *which side* of a frequency a component is on.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 130" role="img" aria-label="Two spectra: a real signal with mirror-image peaks on both sides of zero, and the analytic signal with only the positive-frequency peak remaining." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="60" x2="230" y2="60" stroke="currentColor" stroke-opacity="0.3"/>
  <line x1="125" y1="60" x2="125" y2="18" stroke="currentColor" stroke-width="2"/>
  <line x1="70" y1="60" x2="70" y2="35" stroke="currentColor" stroke-width="2"/>
  <line x1="180" y1="60" x2="180" y2="35" stroke="currentColor" stroke-width="2"/>
  <text x="125" y="75" text-anchor="middle" font-size="8" fill="currentColor">0</text>
  <text x="60" y="30" font-size="8" fill="currentColor">mirror</text>
  <text x="125" y="92" text-anchor="middle" font-size="9" fill="currentColor">real signal</text>
  <text x="270" y="60" text-anchor="middle" font-size="15" fill="currentColor">&#8594;</text>
  <line x1="310" y1="60" x2="500" y2="60" stroke="currentColor" stroke-opacity="0.3"/>
  <line x1="405" y1="60" x2="405" y2="18" stroke="currentColor" stroke-width="2"/>
  <line x1="460" y1="60" x2="460" y2="35" stroke="currentColor" stroke-width="2"/>
  <text x="405" y="75" text-anchor="middle" font-size="8" fill="currentColor">0</text>
  <text x="405" y="92" text-anchor="middle" font-size="9" fill="currentColor">analytic signal (mirror removed)</text>
</svg>
<figcaption>A real signal carries a redundant negative-frequency mirror; the analytic signal keeps only the positive half.</figcaption>
</figure>

## The Hilbert transform: a 90° phase shifter

The tool that removes the mirror is the **Hilbert transform**. Think of it as a filter
that leaves every frequency's *amplitude* untouched but shifts its *phase* by exactly
**90°** — a quarter cycle. Feed a cosine in and a sine comes out; that is precisely the
[quadrature](/learn/dsp/complex-signals-and-iq/) relationship between I and Q.

## Building the analytic signal

Now assemble the complex signal:

```text
I (in-phase)  = the original real signal
Q (quadrature)= its Hilbert transform (90 deg shifted)
analytic signal = I + jQ
```

Because of how the 90°-shifted copy adds and subtracts across the spectrum, the two
mirror halves **cancel on the negative side and reinforce on the positive side**. What
remains is a complex signal with a one-sided spectrum — the **analytic signal**. Its
magnitude at each instant is the signal's **envelope** and its angle is the
instantaneous **phase**, the very quantities [demodulation](/learn/dsp/demodulation/)
reads.

## Where this happens in practice

Most SDRs never run a literal Hilbert transform on a real stream — their hardware
produces I/Q directly by mixing the antenna signal against two oscillators 90° apart, a
**quadrature downconversion** that achieves the same one-sided result in analog. The
Hilbert view still matters: it is the theory that explains *why* two channels 90° apart
capture everything a real signal holds, and it is exactly how you'd convert a real
recording (say, an audio file) into I/Q in software. The result feeds the same
[complex baseband](/learn/dsp/mixing-and-downconversion/) pipeline either way.

<div class="knowledge-check" data-quiz data-correct-msg="Right — pairing a signal with its 90°-shifted Hilbert transform cancels the negative-frequency mirror." markdown="0">
  <p class="knowledge-check__q">Quick check: what does forming the analytic signal accomplish?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It doubles the sample rate</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It removes the redundant negative-frequency mirror, giving clean I/Q</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It eliminates all noise from the signal</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **real** signal's spectrum has a redundant **negative-frequency mirror**.
- The **Hilbert transform** shifts every frequency by **90°** to form the quadrature part.
- Pairing original + Hilbert gives the **analytic signal**, whose mirror half **cancels**.
- SDR hardware reaches the same one-sided I/Q by **quadrature downconversion**.

Next up: the operation at the heart of every filter — convolution and the impulse response.
