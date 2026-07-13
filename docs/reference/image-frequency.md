---
slug: image-frequency
title: Image frequency
entry_type: term
category: sdr-dsp
description: "The image frequency is the second input a mixer folds onto the same IF as the wanted signal, sitting one IF on the far side of the local oscillator."
keywords: image frequency, mixer image, image response, spurious response, superheterodyne, twice the IF, half-IF spur, image interference
aka: [mixer image, image response]
autolink: true
infobox:
  - { label: Type, value: Unwanted mixer response }
  - { label: Location, value: "LO ± IF (the far side of LO)" }
  - { label: Separation, value: "2 × IF from the wanted signal" }
see_also: [image-rejection, superheterodyne-receiver, local-oscillator, intermediate-frequency, mixer-rf, aliasing]
cite_urls:
  - https://en.wikipedia.org/wiki/Image_response
  - https://en.wikipedia.org/wiki/Heterodyne
---

The **image frequency** is the unwanted input frequency that a
[mixer](/reference/mixer-rf/) converts to the *same*
[intermediate frequency](/reference/intermediate-frequency/) as the wanted signal.[^wiki]
Because a mixer responds to the absolute value of the difference between an input and the
[local oscillator](/reference/local-oscillator/), two input frequencies — one above the
oscillator and one below, both offset by the IF — produce identical outputs. One is the
signal you want; the other is its **image**, and any energy there lands on top of your
signal unless it is filtered out first.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A frequency axis showing the local oscillator, the wanted signal one IF above it, and the image one IF below it, both mapping down to the same intermediate frequency, with the wanted-to-image gap equal to twice the IF." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="ifqar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="30" y1="95" x2="440" y2="95" stroke="currentColor"/>
  <line x1="120" y1="95" x2="120" y2="55" stroke="currentColor"/><text x="100" y="48" font-size="8" fill="currentColor">image</text>
  <line x1="235" y1="95" x2="235" y2="65" stroke="currentColor" stroke-dasharray="3 3"/><text x="222" y="58" font-size="8" fill="currentColor">LO</text>
  <line x1="350" y1="95" x2="350" y2="55" stroke="currentColor"/><text x="332" y="48" font-size="8" fill="currentColor">wanted</text>
  <line x1="120" y1="112" x2="235" y2="112" stroke="currentColor" marker-end="url(#ifqar)"/>
  <line x1="350" y1="112" x2="235" y2="112" stroke="currentColor" marker-end="url(#ifqar)"/>
  <text x="150" y="128" font-size="8" fill="currentColor">IF</text>
  <text x="285" y="128" font-size="8" fill="currentColor">IF</text>
  <path d="M120 40 Q235 12 350 40" fill="none" stroke="currentColor" stroke-opacity="0.5" stroke-dasharray="2 2"/>
  <text x="195" y="24" font-size="8" fill="currentColor">gap = 2 × IF</text>
</svg>
<figcaption>The wanted signal and its image sit one IF above and one IF below the local oscillator; both mix to the same IF, so they are separated by twice the IF.</figcaption>
</figure>

## How it works

Suppose the receiver wants a signal at frequency *f*<sub>RF</sub> and sets its oscillator
to *f*<sub>LO</sub>, producing an IF of *f*<sub>IF</sub> = |*f*<sub>RF</sub> −
*f*<sub>LO</sub>|. If the wanted signal is above the oscillator (*f*<sub>RF</sub> =
*f*<sub>LO</sub> + *f*<sub>IF</sub>), then a signal at *f*<sub>LO</sub> −
*f*<sub>IF</sub> — the same distance below — also mixes down to exactly *f*<sub>IF</sub>.
That lower frequency is the image. The two are always separated by **twice the IF**, and
after mixing they are indistinguishable: no amount of IF filtering can separate them,
because they occupy the identical IF band.

The only cure is to act *before* the mixer. Attenuate the image while it is still at RF —
either with a preselector filter or by using a quadrature
[image-reject](/reference/image-rejection/) architecture that cancels it. This is why the
choice of IF is a design trade-off: a **high** IF pushes the image far from the wanted
signal, where a modest RF filter easily rejects it, but demands a higher-frequency IF
stage; a **low** IF is easier to process but places the image close by, where only a very
selective filter can help.

## Variants

A related hazard is the **half-IF spur**: an interferer halfway between the wanted signal
and the oscillator, whose second harmonic mixes with the oscillator's second harmonic to
land in the IF. Mixers also produce other spurious responses at combinations
*m*·*f*<sub>LO</sub> ± *n*·*f*<sub>RF</sub>, but the primary image is the dominant one and
the reason preselection exists.

## Relevance to SDR

In zero-IF and low-IF SDR front ends the IF is 0 (or nearly 0), so a signal's image is its
own mirror around the tuned frequency — the faint reflected copies seen across the center
of an SDR [waterfall](/reference/waterfall-display/) when the tuner's
[IQ imbalance](/reference/iq-imbalance/) is imperfect. In IF-sampling and
[superheterodyne](/reference/superheterodyne-receiver/) designs the image is a distinct
band twice-the-IF away that the preselector must reject. Recognising an image — it moves in
the *opposite* direction to real signals as you retune, and mirrors around the LO or the
tuning center — is a basic SDR troubleshooting skill.

GopherTrunk decodes the IQ stream after the front end, so it inherits whatever image
suppression the device achieved; a strong out-of-plan signal appearing mirrored across the
tuned frequency is an image artefact of the receiver, not of GopherTrunk's DSP.

## Sources

[^wiki]: [Image response](https://en.wikipedia.org/wiki/Image_response) — Wikipedia, on the mixer image frequency, its 2×IF separation, and why it must be filtered before mixing.
