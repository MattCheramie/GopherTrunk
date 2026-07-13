---
slug: john-costas
title: John P. Costas
entry_type: person
category: people
description: John P. Costas (1923–2008) was an American engineer who invented the Costas loop, a carrier-recovery technique key to demodulating PSK and suppressed-carrier signals.
keywords: John Costas, Costas loop, Costas array, carrier recovery, PSK, SSB, phase-locked loop, engineer
aka: ["John Costas", "John P. Costas"]
autolink: true
infobox:
  - { label: Lived, value: "1923–2008" }
  - { label: Field, value: Electrical engineering }
  - { label: Known for, value: Costas loop; Costas arrays }
see_also: [costas-loop, phase-shift-keying, single-sideband, phase-locked-loop, fred-gardner, demodulation]
related_lessons:
  - { title: "Clock recovery & symbol timing", url: /learn/rf-sdr/clock-recovery/ }
cite_urls:
  - https://en.wikipedia.org/wiki/John_P._Costas_(engineer)
  - https://ethw.org/John_P._Costas
---

**John P. Costas** (1923–2008) was an American engineer best known for the
**[Costas loop](/reference/costas-loop/)**, a phase-locked carrier-recovery circuit he
devised in the 1950s.[^wiki] It made coherent reception of suppressed-carrier signals — such
as [SSB](/reference/single-sideband/) and later [PSK](/reference/phase-shift-keying/) — 
practical, and it remains a standard building block inside digital demodulators today.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A feedback loop: phase detector, loop filter, controlled oscillator feeding back — the Costas loop." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="50" y="35" width="70" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="85" y="52">phase det</text>
    <rect x="160" y="35" width="70" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="195" y="52">loop filter</text>
    <rect x="270" y="35" width="80" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="310" y="52">oscillator</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="120" y1="49" x2="159" y2="49" marker-end="url(#jcar)"/><line x1="230" y1="49" x2="269" y2="49" marker-end="url(#jcar)"/><path d="M310 63 V 90 H 85 V 64" fill="none" marker-end="url(#jcar)"/></g>
  </g>
  <defs><marker id="jcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Costas's carrier-recovery loop feeds a phase-error estimate back to a controlled oscillator.</figcaption>
</figure>

## Life and work

John P. Costas was born in 1923 and spent the productive part of his career as an engineer
at the General Electric Company, where he worked on communications and signal-processing
problems. In the mid-1950s the pressing question in his field was how to receive
single-sideband and other suppressed-carrier transmissions, which are spectrally efficient
because they send no separate carrier tone but are correspondingly hard to demodulate: the
receiver must somehow regenerate the missing carrier, in exactly the right phase, from the
sidebands alone. Costas's answer, published in a 1956 paper in the *Proceedings of the IRE*,
was the feedback loop that now bears his name. Later in his career, working on sonar and
radar waveform design, he also studied families of frequency-hopping patterns with ideal
ambiguity properties, and the combinatorial objects he introduced are known as **Costas
arrays** — a second, quite separate contribution that keeps his name current in mathematics
as well as engineering.[^ethw]

## Contribution

The Costas loop is a variant of the phase-locked loop specialised for recovering a
suppressed carrier. It splits the incoming signal into two branches driven by a local
oscillator whose phases are ninety degrees apart — an in-phase (I) branch and a quadrature
(Q) branch. When the local oscillator is correctly locked, all the data appears on the I
branch and the Q branch carries only a phase-error term; multiplying the two branches
together produces an error signal proportional to how far the oscillator has drifted, which
a [loop filter](/reference/phase-locked-loop/) smooths and feeds back to steer the
oscillator. Crucially, because the error term depends on the *product* of the branches, the
loop is insensitive to the 180-degree phase flips of the data itself, so it locks to the
carrier rather than being pulled around by the modulation. Extended to four phases it
recovers the carrier for QPSK and π/4-DQPSK signals, which is why it appears in so many
digital receivers.

## Legacy

The Costas loop is one of the durable primitives of communications engineering, taught in
every digital-receiver course and implemented in countless hardware and software
demodulators. In software radio it is typically realised digitally, and it works hand in
hand with symbol-timing recovery methods such as the detector of
[Floyd Gardner](/reference/fred-gardner/) to lock a receiver in both phase and time. For
land-mobile digital voice the relevance is direct: recovering the carrier of a
phase-modulated waveform such as P25's C4FM/CQPSK or a π/4-DQPSK signal is precisely the job
a Costas-type loop performs, so a decoder like GopherTrunk relies on Costas's 1956 idea every
time it pulls symbols out of a suppressed-carrier signal.

## Sources

[^wiki]: [John P. Costas (engineer)](https://en.wikipedia.org/wiki/John_P._Costas_(engineer)) — Wikipedia, for biography and his invention of the Costas loop.
[^ethw]: [John P. Costas](https://ethw.org/John_P._Costas) — Engineering and Technology History Wiki (IEEE), for the Costas loop and Costas arrays.
