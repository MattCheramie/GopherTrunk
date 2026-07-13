---
slug: heinrich-hertz
title: Heinrich Hertz
entry_type: person
category: people
description: Heinrich Hertz (1857–1894) was a German physicist who first demonstrated electromagnetic waves, confirming Maxwell's theory; the hertz is named after him.
keywords: Heinrich Hertz, electromagnetic waves, hertz unit, radio history, Maxwell, spark gap, resonance, dipole
aka: [Heinrich Hertz]
autolink: true
infobox:
  - { label: Lived, value: "1857–1894" }
  - { label: Field, value: Physics }
  - { label: Known for, value: Proving EM waves exist }
  - { label: Legacy, value: The hertz (Hz) unit }
see_also: [james-clerk-maxwell, guglielmo-marconi, oliver-lodge, oliver-heaviside, nikola-tesla, radio-wave, frequency, electromagnetic-spectrum]
related_lessons:
  - { title: "What is a radio wave?", url: /learn/rf-sdr/radio-waves/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Heinrich_Hertz
  - https://www.britannica.com/biography/Heinrich-Hertz
---

**Heinrich Hertz** (1857–1894) was a German physicist who first conclusively
**demonstrated electromagnetic waves**, experimentally confirming
[James Clerk Maxwell](/reference/james-clerk-maxwell/)'s theory and turning it from
mathematics into observable, reproducible fact.[^wiki] Every reference to a signal's
[frequency](/reference/frequency/) "in hertz" honours the man who proved the
[radio wave](/reference/radio-wave/) exists.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A spark gap on the left radiating waves to a loop receiver on the right, illustrating Hertz's proof of radio waves." xmlns="http://www.w3.org/2000/svg">
  <line x1="50" y1="40" x2="50" y2="62" stroke="currentColor" stroke-width="2"/><line x1="50" y1="72" x2="50" y2="94" stroke="currentColor" stroke-width="2"/>
  <text x="50" y="110" text-anchor="middle" font-size="8" fill="currentColor">spark gap</text>
  <g fill="none" stroke="currentColor" stroke-opacity="0.5"><path d="M70 67 A 30 30 0 0 1 100 67"/><path d="M70 67 A 60 60 0 0 1 130 67"/><path d="M70 67 A 90 90 0 0 1 160 67"/></g>
  <circle cx="400" cy="67" r="22" fill="none" stroke="currentColor" stroke-width="2"/><text x="400" y="110" text-anchor="middle" font-size="8" fill="currentColor">loop receiver</text>
</svg>
<figcaption>Hertz experimentally proved electromagnetic waves exist; the unit of frequency, the hertz, is named for him.</figcaption>
</figure>

## Life and work

Hertz was born in Hamburg in 1857 and studied under Hermann von Helmholtz and Gustav
Kirchhoff at the University of Berlin, completing his doctorate in 1880. Helmholtz, who
recognised his student's exceptional gift for combining theory and precise experiment,
steered him toward an open prize problem: whether Maxwell's predicted electromagnetic
effects could actually be detected. Hertz took up a professorship at the Karlsruhe
Polytechnic in 1885, and it was there, between 1886 and 1889, that he built the apparatus
that made his name.

His transmitter was an induction coil driving a spark gap between two brass spheres
attached to short rods — in effect an early [dipole antenna](/reference/dipole-antenna/)
tuned by its own geometry to a specific resonant frequency. Each spark launched a burst of
oscillating current, and the radiated energy induced a faint answering spark across a much
smaller gap in a nearby loop of wire, his detector. By moving the detector around a
darkened room, Hertz mapped where the invisible field was strong and where it vanished,
demonstrating standing waves and thereby measuring their [wavelength](/reference/wavelength/)
directly.

## Contribution

Hertz did not merely show that *something* crossed the room; he proved the something was
light in every respect but wavelength. He reflected the waves off a zinc sheet to form
standing-wave patterns, refracted them through a large prism cast from pitch, and polarised
them with a grid of parallel wires — the classic optical experiments, repeated with waves
about a metre long instead of a fraction of a micrometre. Multiplying his measured
wavelength by the drive frequency yielded a propagation speed indistinguishable from the
speed of light, exactly as Maxwell's equations demanded. This closed the loop between
[James Clerk Maxwell](/reference/james-clerk-maxwell/)'s prediction and physical reality,
and it established [resonance](/reference/resonance/) as the practical key to both
generating and receiving radio energy.[^brit]

Characteristically, Hertz saw only the physics. Asked about applications, he is reported
to have dismissed the waves as having "no use whatsoever" — a scientist's honesty about
pure discovery. Within a decade [Oliver Lodge](/reference/oliver-lodge/),
[Guglielmo Marconi](/reference/guglielmo-marconi/), and others would prove him spectacularly
wrong by turning his spark-gap apparatus into a communication system.

## Legacy

Hertz died in 1894 at just 36 from granulomatosis with polyangiitis, before the wireless
industry he made possible had truly begun. His influence nonetheless runs through the whole
of radio: the SI unit of frequency, the **hertz (Hz)**, was adopted in his honour in 1930
and is now spoken millions of times a day whenever anyone tunes a radio, cites a Wi-Fi band,
or reads a clock speed. His experimental confirmation also cleared the ground on which
[Oliver Heaviside](/reference/oliver-heaviside/) recast Maxwell's twenty equations into the
compact vector form engineers still use, and on which
[Hendrik Lorentz](/reference/hendrik-lorentz/) built the electron theory linking fields to
matter. The dipole he built to radiate his waves remains, in refined form, one of the most
common [antenna](/reference/antenna/) designs in existence, and the concept of a tuned
resonant circuit at both ends of a link is fundamental to every receiver — including the
front end of any [software-defined radio](/reference/software-defined-radio/) that
GopherTrunk runs on.

## Sources

[^wiki]: [Heinrich Hertz](https://en.wikipedia.org/wiki/Heinrich_Hertz) — Wikipedia, for biography and his proof of electromagnetic waves.
[^brit]: [Heinrich Hertz](https://www.britannica.com/biography/Heinrich-Hertz) — Encyclopædia Britannica, for his Karlsruhe experiments demonstrating reflection, refraction, and polarisation of radio waves.
