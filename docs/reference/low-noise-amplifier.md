---
slug: low-noise-amplifier
title: Low-noise amplifier (LNA)
entry_type: hardware
category: hardware
description: A low-noise amplifier boosts a weak antenna signal early in the receive chain with minimal added noise, setting much of a receiver's sensitivity.
keywords: LNA, low noise amplifier, noise figure, sensitivity, front end, preamp
aka: [low-noise amplifier, LNA]
autolink: true
infobox:
  - { label: Type, value: RF amplifier }
  - { label: Placed, value: Early in receive chain (near antenna) }
  - { label: Key spec, value: Noise figure }
see_also: [noise-floor, signal-to-noise-ratio, superheterodyne-receiver, bias-tee, antenna]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/rf-sdr/sdr-receiver/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Low-noise_amplifier
---

A **low-noise amplifier** (**LNA**) boosts a weak [antenna](/reference/antenna/) signal
**early** in the [receive chain](/reference/superheterodyne-receiver/), adding as little
noise as possible.[^wiki] Because later stages add their own noise, amplifying first preserves
[SNR](/reference/signal-to-noise-ratio/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="Antenna into a low-noise amplifier early in the chain, boosting the weak signal before later stages add noise." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="35" y="55">antenna</text>
    <path d="M120 38 L120 68 L160 53 Z" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="135" y="84" font-size="8">LNA</text>
    <rect x="200" y="38" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="240" y="57">receiver</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="62" y1="53" x2="119" y2="53"/><line x1="160" y1="53" x2="199" y2="53"/></g>
    <text x="330" y="49" font-size="8">boost early =</text><text x="330" y="61" font-size="8">best sensitivity</text>
  </g>
</svg>
<figcaption>A low-noise amplifier boosts the faint antenna signal early, setting the receiver's sensitivity.</figcaption>
</figure>

## How it works

An LNA's *noise figure* largely determines how weak a signal the whole receiver can
detect — its sensitivity. It is best mounted at the antenna, often powered through the
coax by a [bias tee](/reference/bias-tee/).

## Relevance to SDR

An antenna-mounted LNA can meaningfully improve reception of weak signals, especially
with lossy cable runs — but watch for overload from strong nearby transmitters.

## Sources

[^wiki]: [Low-noise amplifier](https://en.wikipedia.org/wiki/Low-noise_amplifier) — Wikipedia, on LNAs, noise figure, and their role in setting receiver sensitivity.
