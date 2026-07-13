---
slug: link-budget
title: Link budget
entry_type: term
category: rf-fundamentals
description: A link budget is the accounting of every gain and loss from transmitter to receiver, in decibels, that predicts received signal power and the margin above the required threshold.
keywords: link budget, RF link budget, received power, margin, gains and losses, path loss, fade margin, receiver sensitivity, decibel accounting
aka: [link budget, RF link budget, power budget]
autolink: true
infobox:
  - { label: Type, value: Power accounting (dB) }
  - { label: Core relation, value: "P_rx = EIRP − losses + G_rx" }
  - { label: Goal, value: "Margin = P_rx − sensitivity > 0" }
see_also: [path-loss, free-space-path-loss, friis-transmission-equation, fade-margin, erp-eirp, receiver-sensitivity]
cite_urls:
  - https://en.wikipedia.org/wiki/Link_budget
---

A **link budget** is the bookkeeping of every gain and loss a signal encounters on its way from
transmitter to receiver, tallied in [decibels](/reference/decibel/) so the terms simply add and
subtract.[^wiki] The output is the predicted received power, which is then compared against the
receiver's required threshold; the difference is the **link margin**. If the margin is comfortably
positive the link closes; if it is negative or thin, the link is unreliable.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A staircase of decibel terms: transmitter power plus antenna gains minus feedline and path losses lands at received power, which sits a positive margin above the receiver sensitivity floor." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" stroke="none">
    <line x1="40" y1="20" x2="40" y2="160" stroke="currentColor" stroke-opacity="0.4"/>
    <line x1="40" y1="160" x2="440" y2="160" stroke="currentColor" stroke-opacity="0.4"/>
    <rect x="55" y="30" width="40" height="20" fill="none" stroke="currentColor"/>
    <text x="58" y="44">P_tx</text>
    <rect x="105" y="24" width="40" height="26" fill="none" stroke="currentColor"/>
    <text x="108" y="40">+G_tx</text>
    <rect x="155" y="24" width="55" height="70" fill="none" stroke="currentColor" stroke-dasharray="3 2"/>
    <text x="160" y="62">− path</text>
    <text x="160" y="74">loss</text>
    <rect x="220" y="94" width="40" height="26" fill="none" stroke="currentColor"/>
    <text x="223" y="110">+G_rx</text>
    <line x1="270" y1="118" x2="430" y2="118" stroke="currentColor" stroke-dasharray="4 3"/>
    <text x="360" y="112">P_rx</text>
    <line x1="270" y1="150" x2="430" y2="150" stroke="currentColor"/>
    <text x="300" y="146">sensitivity floor</text>
    <line x1="410" y1="118" x2="410" y2="150" stroke="currentColor"/>
    <text x="416" y="138">margin</text>
  </g>
</svg>
<figcaption>A link budget adds transmit power and gains, subtracts feedline and path losses, and checks that the received power clears the receiver's sensitivity floor by a positive margin.</figcaption>
</figure>

## How it works

The canonical form starts from the transmitter's [EIRP](/reference/erp-eirp/) (transmitter power
plus transmit [antenna gain](/reference/antenna-gain/), less transmit feedline loss), subtracts the
propagation losses, and adds the receive gains:

**P_rx (dBm) = EIRP − L_path − L_misc + G_rx**

- **EIRP** — effective isotropic radiated power at the transmit antenna.
- **L_path** — [path loss](/reference/path-loss/); over a clear line of sight this is the
  [free-space path loss](/reference/free-space-path-loss/) computed from the
  [Friis transmission equation](/reference/friis-transmission-equation/).
- **L_misc** — everything else: feedline and connector loss, polarization mismatch,
  atmospheric and rain attenuation, body/foliage blockage, pointing error.
- **G_rx** — receive antenna gain, less receive feedline loss.

Compare **P_rx** with the receiver's [sensitivity](/reference/receiver-sensitivity/) — the minimum
input power for an acceptable [bit error rate](/reference/bit-error-rate/) or
[SNR](/reference/signal-to-noise-ratio/). The surplus is the **link margin**. A robust design does not
aim for margin = 0; it reserves a **[fade margin](/reference/fade-margin/)** on top so that fading,
[multipath](/reference/multipath-propagation/), and weather do not drop the link below threshold.

## In practice

Every term is a straight addition once expressed in dB, so a link budget fits on one line of a
spreadsheet. Engineers run it in both directions: forward to predict coverage from known equipment,
and backward to solve for a missing requirement — for instance, "how much [antenna gain](/reference/antenna-gain/)
or transmit power do I need for 10 dB of margin at this range?" On fading channels the required
margin is set statistically (e.g. enough to keep the outage probability below a target), which is
where [Rayleigh](/reference/rayleigh-fading/) and [Rician](/reference/rician-fading/) fade models
feed into the budget.

## Relevance to SDR

For a receive-only [SDR](/reference/software-defined-radio/) the link budget explains what you can
and cannot hear. Starting from a transmitter's published [ERP/EIRP](/reference/erp-eirp/) and the
distance, subtract free-space (and terrain) path loss, add your receive antenna gain and any
[LNA](/reference/low-noise-amplifier/), and compare the result against your effective sensitivity —
the [noise floor](/reference/noise-floor/) plus the SNR the [demodulator](/reference/demodulation/)
needs. A negative margin is a quantitative diagnosis: the fix is more gain, a lower-loss feedline, a
quieter front end, or a better location, and the budget tells you exactly how many dB short you are.

**GopherTrunk** does not compute link budgets internally — it decodes whatever reaches its input —
but the framework is the right way to reason about why a given trunking site or channel does or does
not decode at your location, and what change would close the gap.

## Sources

[^wiki]: [Link budget](https://en.wikipedia.org/wiki/Link_budget) — Wikipedia, the gains-and-losses accounting from transmitter to receiver and the definition of link margin.
