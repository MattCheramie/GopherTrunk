---
slug: harry-nyquist
title: Harry Nyquist
entry_type: person
category: people
description: Harry Nyquist (1889–1976) was a Swedish-American engineer at Bell Labs whose work on sampling and signalling underlies the sampling theorem central to all digital radio.
keywords: Harry Nyquist, Nyquist rate, sampling theorem, Bell Labs, signal processing, thermal noise, telegraph
aka: [Harry Nyquist, Nyquist]
autolink: true
infobox:
  - { label: Lived, value: "1889–1976" }
  - { label: Field, value: Electrical engineering }
  - { label: Known for, value: Sampling and signalling theory }
see_also: [nyquist-theorem, sample-rate, claude-shannon, ralph-hartley, aliasing, thermal-noise]
related_lessons:
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/rf-sdr/sample-rate-nyquist/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Harry_Nyquist
  - https://www.britannica.com/biography/Harry-Nyquist
---

**Harry Nyquist** (1889–1976) was a Swedish-American engineer at Bell Labs whose work on
the maximum signalling rate of a channel underlies the
[sampling theorem](/reference/nyquist-theorem/) at the heart of digital radio.[^wiki] The
rule that a signal must be sampled at more than twice its highest frequency — the boundary
every [ADC](/reference/analog-to-digital-converter/) in an SDR respects — carries his name.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A sine wave sampled at just over two points per cycle, illustrating the Nyquist sampling rate." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="60" x2="440" y2="60" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 60 C 70 20, 130 20, 180 60 S 290 100, 340 60 S 440 30, 440 60" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <g fill="currentColor"><circle cx="50" cy="40" r="3"/><circle cx="115" cy="32" r="3"/><circle cx="180" cy="60" r="3"/><circle cx="245" cy="88" r="3"/><circle cx="310" cy="80" r="3"/><circle cx="375" cy="48" r="3"/></g>
  <text x="230" y="108" text-anchor="middle" font-size="9" fill="currentColor">sample at ≥ 2× the highest frequency</text>
</svg>
<figcaption>Nyquist established the sampling limit at the heart of all digital radio; the Nyquist rate bears his name.</figcaption>
</figure>

## Life and work

Nyquist was born in 1889 in Nilsby, in the Swedish province of Värmland, one of eight
children in a poor farming family. He emigrated to the United States in 1907 at eighteen,
worked and studied his way through the University of North Dakota, earning bachelor's and
master's degrees in electrical engineering, and then completed a PhD in physics at Yale in
1917. He joined the American Telephone and Telegraph Company that year and moved to Bell
Telephone Laboratories when it was formed in 1925, remaining there until his retirement in
1954. Over that career he accumulated some 138 patents and a reputation, among colleagues,
as the quiet source of the good ideas other people developed. Shannon himself later remarked
that Nyquist and [Ralph Hartley](/reference/ralph-hartley/) were the two people whose work
most influenced his own.

## Contribution

Nyquist's two most cited papers came in the 1920s. In "Certain Factors Affecting Telegraph
Speed" (1924) and "Certain Topics in Telegraph Transmission Theory" (1928) he analysed how
fast independent pulses could be pushed through a channel of a given
[bandwidth](/reference/bandwidth/) without them smearing into one another — what is now
called intersymbol interference. He established that a channel of bandwidth *B* can carry
about *2B* independent symbols per second, and, read in reverse, that a signal containing no
frequency above *B* is fully captured by samples taken at rate *2B*. This is the sampling
theorem in embryo; [Claude Shannon](/reference/claude-shannon/) later stated and proved it
in full generality, which is why it is often called the Nyquist–Shannon theorem. Sample too
slowly and high frequencies fold down into the passband as false low-frequency artefacts —
[aliasing](/reference/aliasing/) — which is why every SDR front end pairs its ADC with an
anti-alias filter.[^brit]

In 1928 Nyquist also derived, independently of and simultaneously with John B. Johnson, the
formula for thermal (Johnson–Nyquist) noise, showing that the noise power available from a
resistor depends only on its temperature and the bandwidth. That result sets the
fundamental [thermal noise](/reference/thermal-noise/) floor of every receiver. A decade
later, in 1932, he produced the Nyquist stability criterion, a graphical test that tells
engineers whether a feedback amplifier or control loop will be stable — still taught in
every controls course and directly relevant to the phase-locked loops inside a demodulator.

## Legacy

Three quite different "Nyquist" concepts — the sampling rate, the thermal-noise floor, and
the stability criterion — each anchor a corner of modern electronics, an unusually broad
legacy for one engineer. For software radio the first is inescapable: the choice of
[sample rate](/reference/sample-rate/), the design of decimation filters, and the very
notion of a usable bandwidth all follow directly from his 1928 analysis. GopherTrunk lives
inside those limits every time it selects a capture rate wide enough to hold a trunking
system's channels while staying above twice their highest frequency, and its decode chain
is deliberately built to be well-behaved with respect to the Nyquist boundary so that no
signal energy folds back to corrupt a channel.

## Sources

[^wiki]: [Harry Nyquist](https://en.wikipedia.org/wiki/Harry_Nyquist) — Wikipedia, for biography and his work on sampling and signalling theory.
[^brit]: [Harry Nyquist](https://www.britannica.com/biography/Harry-Nyquist) — Encyclopædia Britannica, for his telegraph-transmission analysis and the sampling and thermal-noise results.
