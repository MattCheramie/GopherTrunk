---
slug: clock-recovery
title: Clock recovery & symbol timing
description: Symbol timing and clock recovery explained — how a receiver finds the rhythm of a digital signal so it samples each symbol at the right instant, why timing error closes the eye diagram, and what loss of symbol lock looks like in GopherTrunk.
keywords: clock recovery, symbol timing, symbol synchronization, timing error, eye diagram, symbol clock, timing recovery loop, symbol lock, digital demodulation
level: advanced
status: full
prereq:
  - demodulation-pipeline
faq:
  - q: What is clock recovery in a digital receiver?
    a: Clock recovery is the process of working out the exact timing of a digital signal's symbols from the signal itself, since the transmitter's clock isn't shared with the receiver. Once recovered, the receiver knows when each symbol starts and can sample it at its centre, where the value is clearest. Without it, the receiver wouldn't know where one symbol ends and the next begins.
  - q: Why does symbol timing matter so much?
    a: A digital symbol is only a clean, distinct value for a brief instant at its centre. Sample too early or too late and you catch the transition between symbols, where the value is ambiguous, causing errors. Good symbol timing samples right at the centre of each symbol, where the eye diagram is most open and the decision is easiest.
  - q: What does the eye diagram have to do with clock recovery?
    a: The eye diagram shows the demodulated signal overlaid over one symbol period; the open part of the "eye" is where sampling gives a clear value. Clock recovery aims to sample at the widest point of the eye. If timing drifts, the sampling point moves off-centre and the eye appears to close, which is a visible sign of timing trouble.
  - q: What happens when clock recovery loses lock?
    a: When the receiver can't track the symbol timing — usually because the signal is too weak, noisy, or smeared by multipath — symbols are sampled at the wrong moments and bit errors climb past what error correction can fix. In practice the decode breaks up or drops, and GopherTrunk's symbol-based scopes show a collapsing, unstable pattern.
gophertrunk_links:
  - title: Eye diagram
    url: /eye-diagram.html
    note: watch the eye open or close as timing locks or drifts.
  - title: Symbol scope
    url: /symbol-scope.html
    note: see recovered symbols — steady when locked, jittery when not.
---

# Clock recovery & symbol timing

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The transmitter never sends its clock, so the receiver must **recover the symbol
timing from the signal itself** — that's **clock recovery**. It matters because a
[symbol](/learn/digital-modulation/) is only a clean value for an instant at its
**centre**; sampling off-centre catches the ambiguous transitions and causes errors.
Clock recovery aims to sample at the **widest point of the [eye
diagram](/learn/digital-modulation/)**. When the signal is too weak or smeared, timing
loses lock, the eye **closes**, and the decode breaks up — visible in GopherTrunk's
symbol scope and eye diagram.
</div>

This is the timing trick at the heart of stage 4 of the
[demodulation pipeline](/learn/demodulation-pipeline/). It's why a signal with plenty of
strength can still fail to decode — and why the scopes are such powerful diagnostic
tools.

## Why symbol timing matters

A digital signal carries one [symbol](/learn/digital-modulation/) after another at a
fixed [symbol rate](/learn/symbols-and-baud/). But each symbol is only a clean, distinct
value for a **brief instant at its centre**. Between symbols, the signal *transitions*
from one value to the next, passing through ambiguous in-between levels.

The receiver therefore has to sample **right at the centre** of every symbol. Sample a
little early or late and you catch a transition instead of a symbol, reading a value
that's neither one thing nor the other — an error. At 4800 [baud](/learn/symbols-and-baud/),
those instants are about 200 microseconds apart, and the receiver must hit each one.

## The clock-recovery problem

Here's the catch: the transmitter and receiver run on **separate clocks**, and the
transmitter doesn't send a "tick here" signal. The receiver only has the incoming
waveform. So it must **deduce the symbol timing from the signal itself** — figuring out
the rhythm purely from where the signal's transitions fall.

Worse, the two clocks drift slightly relative to each other, so the timing isn't a
one-time guess — it has to be **continuously tracked**.

How much drift? A cheap dongle's clock might be off by **20 ppm** — 20 parts per
million. At 4800 [baud](/learn/symbols-and-baud/) that's only about 0.1 symbol of slip
per second, which sounds negligible. But a digital frame is thousands of symbols long,
so left uncorrected the sampling point would wander right out of the symbol within a
fraction of a second and the decode would collapse. That's why clock recovery is a
*loop* that nudges constantly, not a one-time measurement.

## How a receiver locks to the symbol rate

A **timing-recovery loop** solves it. In essence the receiver:

1. Makes an initial estimate of the symbol period (it knows the nominal
   [baud](/learn/symbols-and-baud/)).
2. Watches where the signal's **transitions** actually occur — transitions should fall
   *between* symbols, so their position reveals whether the current sampling point is
   early or late.
3. **Nudges** the sampling instant to keep it centred, repeating forever to track drift.

When this loop has settled onto the right rhythm, the receiver has **symbol lock** —
it's sampling each symbol at its centre, and the symbols come out clean.

## Timing error and the eye diagram

The [eye diagram](/learn/digital-modulation/) makes timing visible. It overlays many
symbol periods so the open "eyes" between levels stack up. The **centre of the eye is
the widest, clearest place to sample**:

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 150" role="img" aria-label="An eye diagram with an open eye shape. A vertical dashed line at the centre marks the ideal sampling instant where the eye is widest; arrows show that sampling off to the side hits the narrow, closing part." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2" stroke-opacity="0.8">
    <path d="M40 30 C120 30 120 120 200 120 C280 120 280 30 360 30"/>
    <path d="M40 120 C120 120 120 30 200 30 C280 30 280 120 360 120"/>
    <path d="M40 30 C120 30 120 120 200 120"/>
    <path d="M200 30 C280 30 280 120 360 120"/>
  </g>
  <line x1="200" y1="20" x2="200" y2="130" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="200" y="146" text-anchor="middle" font-size="10" fill="currentColor">sample here (eye widest)</text>
  <text x="70" y="18" font-size="10" fill="currentColor">off-centre = errors →</text>
</svg>
<figcaption>Clock recovery aims the sampling instant at the centre of the eye. If timing drifts off-centre, the usable opening shrinks and bit errors rise.</figcaption>
</figure>

If timing drifts off-centre, the effective opening shrinks — the eye appears to
**close** — and decisions get error-prone even though the signal strength hasn't
changed. A closing eye is therefore a direct, visual symptom of a timing (or noise)
problem.

## What loss of lock looks like in GopherTrunk

When the signal is too weak, too noisy, or smeared by [multipath](/learn/propagation/),
the timing loop can't keep up and **loses lock**. Symbols get sampled at the wrong
moments, bit errors outrun [error correction](/learn/demodulation-pipeline/), and the
decode breaks up or drops out. On the scopes you'll see it clearly:

- The **[eye diagram](/eye-diagram.html)** closes and blurs.
- The **[symbol scope](/symbol-scope.html)** goes from steady, well-separated levels to
  a jittery, collapsing mess.
- The **[constellation](/constellation.html)** smears or rotates.

Recognising these is the core skill of [tuning for a clean
lock](/learn/tuning-with-scopes/) — often the fix is more SNR (better antenna/placement,
right [gain](/learn/gain-and-agc/)) so the timing loop has a clean signal to track.

<div class="knowledge-check" data-quiz data-correct-msg="Right — sample at the centre of the eye, where the symbol value is clearest." markdown="0">
  <p class="knowledge-check__q">Quick check: where should the receiver sample each symbol?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">At the transition between symbols</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">At the centre of the symbol, where the eye is widest</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It doesn't matter when</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The transmitter's clock isn't shared, so the receiver must **recover symbol timing**
  from the signal.
- Sample at the **symbol centre** (widest eye); off-centre sampling causes errors.
- A **timing-recovery loop** tracks the rhythm and follows clock drift.
- **Losing lock** closes the eye and breaks the decode — visible on GopherTrunk's scopes.
- The usual cure is more **SNR** so the loop has a clean signal.

That completes Module 4. Next module reaches the systems GopherTrunk was built for —
starting with why voice went digital.
