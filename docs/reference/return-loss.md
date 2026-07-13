---
slug: return-loss
title: Return Loss
entry_type: term
category: rf-fundamentals
description: Return loss is the decibel measure of how much power reflects from an impedance mismatch, RL = −20·log10|Γ|; higher dB means a better match and less reflected power.
keywords: return loss, reflected power, dB, impedance match, reflection coefficient, VSWR, S11, insertion loss
aka: [RL]
autolink: true
infobox:
  - { label: Symbol, value: "RL" }
  - { label: Unit, value: "decibel (dB)" }
  - { label: Formula, value: "RL = −20·log10|Γ|" }
see_also: [reflection-coefficient, standing-wave-ratio, impedance, s-parameters, vector-network-analyzer, decibel]
cite_urls:
  - https://en.wikipedia.org/wiki/Return_loss
  - https://en.wikipedia.org/wiki/Reflection_coefficient
---

**Return loss** is the ratio, expressed in [decibels](/reference/decibel/), of the
power incident on an impedance boundary to the power reflected back from it:
**RL = −20·log₁₀|Γ|**, where Γ is the
[reflection coefficient](/reference/reflection-coefficient/).[^wiki] A large return
loss (say 20 dB or more) means very little power comes back and the match is good; a
small return loss (a few dB) means a large fraction reflects and the match is poor.
It measures exactly the same mismatch as the
[standing-wave ratio](/reference/standing-wave-ratio/), just on a logarithmic scale.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A plot of return loss in decibels versus frequency showing a deep notch where the antenna is well matched, with a horizontal line marking the 10 dB acceptable threshold." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rlar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="50" y1="20" x2="50" y2="140" stroke="currentColor" stroke-width="1" marker-end="url(#rlar)"/>
  <line x1="50" y1="140" x2="440" y2="140" stroke="currentColor" stroke-width="1" marker-end="url(#rlar)"/>
  <text x="18" y="30" font-size="9" fill="currentColor">0 dB</text>
  <text x="12" y="110" font-size="9" fill="currentColor">30 dB</text>
  <text x="420" y="155" font-size="9" fill="currentColor">freq</text>
  <line x1="50" y1="60" x2="440" y2="60" stroke="currentColor" stroke-width="1" stroke-dasharray="4 3"/>
  <text x="360" y="55" font-size="8" fill="currentColor">10 dB threshold</text>
  <path d="M50 40 Q180 45 230 125 Q245 132 260 125 Q310 45 440 42" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="205" y="130" font-size="8" text-anchor="end" fill="currentColor">well-matched notch</text>
</svg>
<figcaption>Return loss versus frequency: the deep notch marks the band where the antenna is well matched (little reflected power); values above the 10 dB line are usually considered acceptable.</figcaption>
</figure>

## How it works

When a wave meets a mismatched load, a fraction of its power reflects, set by the
reflection coefficient's magnitude: reflected power is |Γ|² of incident power.
Return loss simply expresses that fraction in decibels and flips its sign so the
number is positive and larger-is-better:

- RL = 0 dB → |Γ| = 1 → all power reflected (open or short circuit).
- RL = 10 dB → |Γ| ≈ 0.316 → 10 % of power reflected.
- RL = 20 dB → |Γ| = 0.1 → 1 % of power reflected.
- RL = ∞ dB → |Γ| = 0 → perfect match, nothing reflected.

A rough rule of thumb across RF engineering is that a return loss above 10 dB (VSWR
below about 2:1) is acceptable for most receive and many transmit uses, while
sensitive or high-power systems aim for 15–20 dB or better. Note the sign
convention: return loss is conventionally quoted as a positive number of decibels,
so "improving" the match means the number goes *up*. Some instruments instead report
the negative *S11* value; the two describe the same thing and only the sign differs.

## In practice

Return loss is a frequency-dependent curve, not a single number. An antenna presents
its lowest reflection near resonance and worsens toward the band edges, so a sweep
reveals the usable bandwidth as the span where the curve stays above the chosen
threshold. Connectors, adaptors, and cable faults each add their own small
reflections, and their contributions can add or partly cancel depending on phase and
spacing — which is why a chain of individually decent parts can still show a
disappointing aggregate return loss.

Return loss should not be confused with **insertion loss**, which is power *lost*
passing *through* a component (heat, radiation) rather than power *reflected* back.
A good filter has high return loss in its passband and low insertion loss; a bad
match shows the opposite.

## Relevance to SDR

For a receive-only SDR, return loss quantifies how much of the antenna-captured
signal is turned away at the input connector instead of reaching the low-noise
amplifier and ADC. It is measured with a [vector network
analyzer](/reference/vector-network-analyzer/) or a simpler antenna analyzer as the
magnitude of the input [S-parameter](/reference/s-parameters/) *S11*. Keeping return
loss high across the band of interest maximises the delivered signal and therefore
the [signal-to-noise ratio](/reference/signal-to-noise-ratio/) the decoder receives.

GopherTrunk itself measures nothing in the analog domain — it consumes IQ samples
after the front end — so return loss is an antenna-and-cabling property that shapes
the quality of the samples GopherTrunk is handed, not something the software reports.
For operators chasing a marginal control channel, checking the antenna's return loss
on the target frequency is a practical first step before blaming the decoder.

## Sources

[^wiki]: [Return loss](https://en.wikipedia.org/wiki/Return_loss) — Wikipedia, the RL = −20·log10|Γ| definition, sign conventions, and relation to reflected power.
