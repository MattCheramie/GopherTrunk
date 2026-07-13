---
slug: radio-propagation
title: Radio propagation
entry_type: term
category: propagation
description: Radio propagation is the behaviour of radio waves as they travel from transmitter to receiver — line-of-sight, reflection, diffraction, and atmospheric effects.
keywords: radio propagation, line of sight, reflection, diffraction, fading, coverage, ground wave, sky wave, path loss
aka: [radio propagation, propagation]
autolink: true
infobox:
  - { label: Type, value: Wave-travel behaviour }
  - { label: VHF/UHF, value: Mostly line-of-sight }
  - { label: HF, value: Ionospheric skip possible }
see_also: [multipath-propagation, radio-horizon, ionospheric-propagation, ground-wave, sky-wave, free-space-path-loss, path-loss, frequency-bands]
related_lessons:
  - { title: "How signals travel", url: /learn/rf-sdr/propagation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Radio_propagation
  - https://www.itu.int/rec/R-REC-P.341/en
---

**Radio propagation** describes how [radio waves](/reference/radio-wave/) travel from
transmitter to receiver, including line-of-sight travel, reflection, diffraction, and
atmospheric effects.[^wiki] Which of these dominates depends mostly on **frequency**: the
same terrain that is transparent to one band is a wall to another, so the propagation
picture shifts completely as you tune from HF up to microwave.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A curved earth with a tall transmitter tower and a receiver, a straight line-of-sight path, and an obstacle blocking a lower path." xmlns="http://www.w3.org/2000/svg">
  <path d="M10 140 Q230 95 450 140" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-width="1.4"/>
  <line x1="80" y1="122" x2="80" y2="55" stroke="currentColor" stroke-width="2"/><text x="62" y="48" font-size="9" fill="currentColor">TX</text>
  <line x1="380" y1="120" x2="380" y2="88" stroke="currentColor" stroke-width="2"/><text x="368" y="80" font-size="9" fill="currentColor">RX</text>
  <line x1="80" y1="55" x2="380" y2="88" stroke="currentColor" stroke-width="1.4" stroke-dasharray="5 3"/><text x="200" y="58" font-size="9" fill="currentColor">line of sight</text>
  <rect x="225" y="100" width="14" height="22" fill="currentColor" fill-opacity="0.3"/>
</svg>
<figcaption>At VHF/UHF, propagation is line-of-sight; height and a clear path matter more than raw distance.</figcaption>
</figure>

## How it works

A radio wave spreading from a transmitter weakens with distance even in empty space, as its
power spreads over an ever-larger sphere; this baseline is
[free-space path loss](/reference/free-space-path-loss/), and everything else adds to it.
On top of that, several distinct **propagation modes** carry signals in the real world:

- **Line-of-sight** — the direct ray, dominant at VHF, UHF, and above. It is bounded by the
  [radio horizon](/reference/radio-horizon/), so antenna height and a clear path matter more
  than raw transmitter power.
- **[Ground wave](/reference/ground-wave/)** — at low frequencies the wave follows the
  Earth's conducting surface and can reach well beyond the horizon; this is how AM broadcast
  and marine MF work by day.
- **[Sky wave](/reference/sky-wave/) / [ionospheric](/reference/ionospheric-propagation/)** —
  at HF the upper atmosphere refracts waves back to earth, allowing "skip" over thousands of
  kilometres.
- **Reflection and [multipath](/reference/multipath-propagation/)** — waves bounce off
  terrain and buildings, arriving by several paths that interfere and cause fading.
- **Diffraction** — waves bend slightly around edges and over hills
  ([knife-edge diffraction](/reference/knife-edge-diffraction/)), filling in some shadow
  behind obstacles.
- **Tropospheric effects** — ducting and scatter in the lower atmosphere occasionally
  stretch VHF/UHF ranges far past the normal horizon.

## In practice

Frequency sets the recipe. Below a few MHz, ground wave and sky wave rule and signals can
travel far. In the HF range (roughly 3–30 MHz), the [ionosphere](/reference/ionospheric-propagation/)
opens and closes paths with the time of day, season, and solar cycle. From VHF upward the
sky goes transparent and coverage collapses to line-of-sight plus reflections, which is why
land-mobile systems lean on tall repeater sites rather than raw power. Total signal at the
receiver is the transmit power plus antenna gains minus the accumulated
[path loss](/reference/path-loss/) — the [link budget](/reference/link-budget/) that decides
whether a decode is even possible.

## Relevance to SDR

Understanding propagation explains why antenna height and a clear path often matter more
than the radio, and why a distant hilltop system can beat a closer obstructed one. It also
tells the operator which antenna and band strategy fits a target: a line-of-sight UHF
trunked system rewards a high, clear vertical, while chasing HF utility stations means
working with a fickle ionosphere. GopherTrunk decodes whatever survives the path; it cannot
conjure a signal the propagation channel never delivered, so reading the channel correctly
is the first step in a successful monitoring setup.

## Sources

[^wiki]: [Radio propagation](https://en.wikipedia.org/wiki/Radio_propagation) — Wikipedia, on line-of-sight, ground wave, sky wave, reflection, diffraction, and atmospheric propagation.
[^itu]: [Recommendation ITU-R P.341: The concept of transmission loss for radio links](https://www.itu.int/rec/R-REC-P.341/en) — ITU-R, for the formal framework of propagation and transmission loss.
