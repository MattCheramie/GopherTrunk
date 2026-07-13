---
slug: joseph-fourier
title: Joseph Fourier
entry_type: person
category: people
description: Joseph Fourier (1768–1830) was a French mathematician who showed signals decompose into sinusoids — the basis of the Fourier transform and spectrum analysis.
keywords: Joseph Fourier, Fourier series, Fourier transform, frequency analysis, mathematics, heat equation, harmonics
aka: [Joseph Fourier, Fourier]
autolink: true
infobox:
  - { label: Lived, value: "1768–1830" }
  - { label: Field, value: Mathematics / physics }
  - { label: Known for, value: Fourier series and analysis }
see_also: [fourier-transform, fast-fourier-transform, discrete-fourier-transform, ralph-hartley, frequency, spectrogram]
related_lessons:
  - { title: "The FFT & reading a waterfall", url: /learn/rf-sdr/fft-and-waterfall/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Joseph_Fourier
  - https://www.britannica.com/biography/Joseph-Baron-Fourier
---

**Joseph Fourier** (1768–1830) was a French mathematician and physicist who showed that
functions can be represented as sums of sinusoids — the insight behind the
[Fourier transform](/reference/fourier-transform/) and, with it, every spectrum display and
waterfall in software radio.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A complex waveform shown as the sum of several simple sine waves." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 40 q20 -20 40 0 t40 0 t40 0 t40 0 t40 0 t40 0" fill="none" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.6"/>
  <path d="M20 65 q10 -12 20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0" fill="none" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.6"/>
  <text x="300" y="50" font-size="14" fill="currentColor">= </text>
  <path d="M330 95 q10 -22 20 0 q10 8 20 0 q10 -16 20 0 q10 14 20 0 q10 -10 20 0" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="120" y="110" text-anchor="middle" font-size="9" fill="currentColor">simple sines</text><text x="380" y="110" text-anchor="middle" font-size="9" fill="currentColor">sum</text>
</svg>
<figcaption>Fourier showed any signal can be expressed as a sum of sinusoids — the mathematics behind the FFT and spectra.</figcaption>
</figure>

## Life and work

Jean-Baptiste Joseph Fourier was born in Auxerre in 1768, orphaned young, and educated at a
local military school run by Benedictines, where his mathematical gift emerged early. He
came of age during the French Revolution, was briefly imprisoned during the Terror, and
then taught at the newly founded École Normale and École Polytechnique alongside Lagrange
and Laplace. In 1798 he joined Napoleon's expedition to Egypt as a scientific adviser,
helping to found the Institut d'Égypte and contributing to the monumental *Description de
l'Égypte*; his experience there left him with a lifelong, almost obsessive, interest in
heat. On his return Napoleon appointed him prefect of the Isère département at Grenoble,
and it was in the hours around those administrative duties that he did his greatest
scientific work.

That work was the mathematical theory of heat conduction, presented to the Paris Academy in
1807 and published in expanded form as *Théorie analytique de la chaleur* in 1822. To solve
the heat equation he needed to represent an arbitrary temperature distribution as a sum of
sinusoidal components, and he asserted — controversially — that *any* function could be so
represented.

## Contribution

Fourier's claim that an arbitrary, even discontinuous, function could be written as an
infinite sum of sines and cosines was met with deep scepticism by Lagrange and others, who
saw it as insufficiently rigorous. They were right that the details needed care — questions
of convergence occupied mathematicians for the next century and drove the development of
modern analysis, the rigorous definition of a function, and the theory of integration. But
Fourier's core idea was correct and extraordinarily powerful: a signal in time and its
representation in [frequency](/reference/frequency/) are two views of the same object, and
one can move between them freely.[^brit] Decomposing a complicated waveform into its
constituent [harmonics](/reference/harmonics/) turns hard problems in one domain into easy
ones in the other. Beyond signal analysis, Fourier's study of heat also led him to reason
about the Earth's temperature and to describe what is now recognised as the greenhouse
effect.

## Legacy

Fourier died in Paris in 1830, and his name is now attached to one of the most-used
operations in all of engineering and science. The continuous
[Fourier transform](/reference/fourier-transform/) generalises his series to non-periodic
signals; the [discrete Fourier transform](/reference/discrete-fourier-transform/) adapts it
to sampled data; and the [fast Fourier transform](/reference/fast-fourier-transform/)
computes that discrete version efficiently enough to run in real time. That chain is the
beating heart of software-defined radio: every [spectrogram](/reference/spectrogram/) and
waterfall, every channelised FFT filter bank, and the tone- and carrier-detection stages in
decoders all rest on Fourier's insight. Related information-theoretic work by figures such
as [Ralph Hartley](/reference/ralph-hartley/) built on the same frequency-domain thinking.
When GopherTrunk displays a band's spectrum or measures a signal's occupied bandwidth, it is
performing, digitally and thousands of times a second, the transformation Fourier first
wrote down to describe the flow of heat.

## Sources

[^wiki]: [Joseph Fourier](https://en.wikipedia.org/wiki/Joseph_Fourier) — Wikipedia, for biography and his work on Fourier series and analysis.
[^brit]: [Joseph, Baron Fourier](https://www.britannica.com/biography/Joseph-Baron-Fourier) — Encyclopædia Britannica, for his theory of heat and the introduction of Fourier series.
