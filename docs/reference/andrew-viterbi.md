---
slug: andrew-viterbi
title: Andrew Viterbi
entry_type: person
category: people
description: Andrew Viterbi (b. 1935) is an American engineer who devised the Viterbi algorithm for decoding convolutional codes and co-founded Qualcomm.
keywords: Andrew Viterbi, Viterbi algorithm, convolutional codes, Qualcomm, CDMA, maximum likelihood, trellis
aka: [Andrew Viterbi]
autolink: true
infobox:
  - { label: Born, value: "1935" }
  - { label: Field, value: Electrical engineering }
  - { label: Known for, value: Viterbi algorithm; co-founding Qualcomm }
see_also: [viterbi-algorithm, convolutional-code, forward-error-correction, irwin-jacobs, robert-gallager, cdma]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Andrew_Viterbi
  - https://ethw.org/Andrew_J._Viterbi
---

**Andrew Viterbi** (born 1935) is an American electrical engineer who devised the
**[Viterbi algorithm](/reference/viterbi-algorithm/)** for maximum-likelihood decoding of
[convolutional codes](/reference/convolutional-code/), and co-founded Qualcomm, the company
that put CDMA into the world's cellphones.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A trellis with one highlighted most-likely path, the Viterbi algorithm." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor"><circle cx="40" cy="40" r="3"/><circle cx="40" cy="95" r="3"/><circle cx="160" cy="40" r="3"/><circle cx="160" cy="95" r="3"/><circle cx="280" cy="40" r="3"/><circle cx="280" cy="95" r="3"/><circle cx="400" cy="40" r="3"/><circle cx="400" cy="95" r="3"/></g>
  <g stroke="currentColor" stroke-opacity="0.3"><line x1="40" y1="40" x2="160" y2="40"/><line x1="40" y1="40" x2="160" y2="95"/><line x1="40" y1="95" x2="160" y2="40"/><line x1="40" y1="95" x2="160" y2="95"/><line x1="160" y1="40" x2="280" y2="40"/><line x1="160" y1="95" x2="280" y2="40"/><line x1="160" y1="95" x2="280" y2="95"/><line x1="280" y1="40" x2="400" y2="95"/><line x1="280" y1="95" x2="400" y2="95"/></g>
  <polyline points="40,95 160,40 280,40 400,95" fill="none" stroke="currentColor" stroke-width="2.4"/>
  <text x="220" y="122" text-anchor="middle" font-size="9" fill="currentColor">the Viterbi algorithm</text>
</svg>
<figcaption>Viterbi devised the maximum-likelihood decoding algorithm now used to decode convolutional codes everywhere.</figcaption>
</figure>

## Life and work

Andrew James Viterbi was born Andrea Viterbi in Bergamo, Italy, in 1935; his Jewish family
fled Mussolini's racial laws in 1939 and settled in Boston, where he became a naturalised
American. He studied electrical engineering at MIT, taking bachelor's and master's degrees,
worked on telemetry and phase-locked loops at the Jet Propulsion Laboratory during the early
space age, and earned a PhD from the University of Southern California in 1962. He then
joined the faculty at UCLA, where he taught digital communications and co-wrote an
influential textbook on the subject. It was as a professor, working through the theory of
coding for deep-space links, that he found the algorithm that bears his name.

Viterbi was also a builder of companies. With his colleague [Irwin Jacobs](/reference/irwin-jacobs/)
he co-founded Linkabit in 1968 and then, in 1985, Qualcomm — firms that turned academic
coding and spread-spectrum theory into the silicon that runs modern cellular networks.

## Contribution

Viterbi introduced his algorithm in a 1967 paper as a way to decode
[convolutional codes](/reference/convolutional-code/). A convolutional encoder has memory,
so its output can be pictured as a path through a **trellis** of possible encoder states over
time. The decoder's job is to find the path that best matches the noisy received sequence —
the maximum-likelihood estimate — but the number of possible paths grows exponentially with
message length, so brute force is hopeless. Viterbi's insight was that at each stage only the
single best surviving path into each state need be kept; all others can be discarded because
they can never become part of the eventual best path. This dynamic-programming pruning makes
the cost grow only linearly with message length, turning an intractable search into a
practical, low-cost computation.[^ethw] It also accepts
[soft-decision](/reference/soft-decision/) inputs, using the demodulator's confidence in
each symbol rather than hard 0/1 bits, which buys a further couple of decibels of coding
gain.

The same trellis method reaches beyond error correction: it is the standard tool for
maximum-likelihood sequence estimation in equalisers, for hidden-Markov-model decoding in
speech recognition and bioinformatics, and for demodulating signals with memory such as
continuous-phase modulation.

## Legacy

The Viterbi algorithm became one of the most widely deployed algorithms in engineering. It
decodes the convolutional codes in GSM, in satellite and deep-space links, in Wi-Fi, in
digital video broadcasting, and in countless two-way radio systems, letting receivers pull
correct data out of signals buried near the noise. It sits alongside the broader coding
theory of Viterbi's mentor [Robert Gallager](/reference/robert-gallager/) and reaches back
toward the limit [Claude Shannon](/reference/claude-shannon/) defined. For land-mobile radio
the connection is direct: P25 and many digital-voice modes protect their traffic with
convolutional or trellis-coded schemes, so a decoder such as GopherTrunk relies on Viterbi's
method — the same maximum-likelihood trellis search — to recover frames cleanly at low SNR.
Viterbi's fortune from Qualcomm later funded, among much else, the USC Viterbi School of
Engineering, which bears his name.

## Sources

[^wiki]: [Andrew Viterbi](https://en.wikipedia.org/wiki/Andrew_Viterbi) — Wikipedia, for biography and his algorithm for decoding convolutional codes.
[^ethw]: [Andrew J. Viterbi](https://ethw.org/Andrew_J._Viterbi) — Engineering and Technology History Wiki (IEEE), for the Viterbi algorithm and the founding of Qualcomm.
