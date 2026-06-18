---
slug: john-costas
title: John P. Costas
entry_type: person
category: people
description: John P. Costas was an American engineer who invented the Costas loop, a carrier-recovery technique key to demodulating PSK and suppressed-carrier signals.
keywords: John Costas, Costas loop, carrier recovery, PSK, SSB, engineer
aka: ["John Costas", "John P. Costas"]
autolink: true
see_also: [costas-loop, phase-shift-keying, single-sideband, demodulation]
related_lessons:
  - { title: "Clock recovery & symbol timing", url: /learn/rf-sdr/clock-recovery/ }
external:
  - { title: "John P. Costas (Wikipedia)", url: https://en.wikipedia.org/wiki/John_P._Costas_(engineer) }
---

**John P. Costas** was an American engineer best known for the **[Costas loop](/reference/costas-loop/)**,
a phase-locked carrier-recovery circuit he devised in the 1950s. It made coherent
reception of suppressed-carrier signals (such as [SSB](/reference/single-sideband/) and
later [PSK](/reference/phase-shift-keying/)) practical.

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

## Contribution

The Costas loop remains a standard building block in digital receivers, including the
carrier recovery used when demodulating phase-modulated digital-voice systems.
