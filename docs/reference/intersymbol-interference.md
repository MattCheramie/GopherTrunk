---
slug: intersymbol-interference
title: Intersymbol interference (ISI)
entry_type: term
category: modulation
description: "Intersymbol interference (ISI) is the smearing of one digital symbol into its neighbors from filtering or multipath, closing the eye diagram and causing bit errors."
keywords: intersymbol interference, ISI, Nyquist criterion, eye closure, pulse tails, multipath ISI, matched filter, equalization
aka: [intersymbol interference, ISI]
autolink: true
infobox:
  - { label: Symbol, value: "ISI" }
  - { label: Cause, value: "Bandlimiting or multipath spreading pulses" }
  - { label: Cure, value: "Nyquist pulse shaping, equalization" }
see_also: [pulse-shaping, root-raised-cosine-filter, eye-diagram, decision-feedback-equalizer, symbol-rate]
cite_urls:
  - https://en.wikipedia.org/wiki/Intersymbol_interference
  - https://en.wikipedia.org/wiki/Nyquist_ISI_criterion
---

**Intersymbol interference** (**ISI**) is the corruption of a digital symbol by energy that has
smeared in from adjacent symbols, so that the value sampled at one symbol instant no longer depends
only on the symbol sent then.[^wiki] It arises whenever the channel or filtering spreads each pulse in
time — from bandlimiting, from imperfect [pulse shaping](/reference/pulse-shaping/), or from
[multipath](/reference/multipath-propagation/) echoes — and it is a leading cause of bit errors in a
receiver that is otherwise seeing plenty of signal.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A Nyquist pulse whose tails cross zero at the neighboring symbol instants versus a spread pulse whose non-zero tails overlap neighboring symbols, closing the eye." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="80" x2="230" y2="80" stroke="currentColor" stroke-opacity="0.35"/>
  <path d="M40 80 Q 70 80 90 60 Q 110 40 130 60 Q 150 80 170 80 Q 190 80 210 80" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <circle cx="90" cy="80" r="2.5" fill="currentColor"/><circle cx="170" cy="80" r="2.5" fill="currentColor"/>
  <text x="130" y="24" text-anchor="middle" font-size="8" fill="currentColor">Nyquist: tails zero at</text>
  <text x="130" y="34" text-anchor="middle" font-size="8" fill="currentColor">neighbor instants</text>
  <line x1="90" y1="88" x2="90" y2="100" stroke="currentColor" stroke-opacity="0.4"/><line x1="170" y1="88" x2="170" y2="100" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="250" y1="80" x2="450" y2="80" stroke="currentColor" stroke-opacity="0.35"/>
  <path d="M260 80 Q 290 76 310 58 Q 330 42 350 62 Q 370 78 390 72 Q 410 68 430 76" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <circle cx="310" cy="72" r="2.5" fill="currentColor"/><circle cx="390" cy="72" r="2.5" fill="currentColor"/>
  <text x="350" y="24" text-anchor="middle" font-size="8" fill="currentColor">spread: tails leak into</text>
  <text x="350" y="34" text-anchor="middle" font-size="8" fill="currentColor">neighbors → ISI</text>
  <line x1="310" y1="80" x2="310" y2="100" stroke="currentColor" stroke-opacity="0.4"/><line x1="390" y1="80" x2="390" y2="100" stroke="currentColor" stroke-opacity="0.4"/>
</svg>
<figcaption>A Nyquist pulse's tails cross zero at every other symbol instant; a spread pulse leaves residual energy there, the definition of ISI.</figcaption>
</figure>

## How it works

Every symbol is sent as a pulse, and a real pulse is never a perfect spike — it has tails that extend
before and after its peak. If those tails are still non-zero at the instant a neighboring symbol is
sampled, the neighbor's decision is biased by this symbol, and vice versa: the samples become a weighted
sum of many transmitted symbols rather than one. Harry Nyquist showed the condition that avoids this: a
pulse whose overall (transmit-times-receive) response passes through **zero at every other symbol
instant** contributes nothing at neighboring sampling times, even though it is non-zero in between. The
[raised-cosine](/reference/pulse-shaping/) family satisfies this, which is why digital systems split it as
a [root-raised-cosine](/reference/root-raised-cosine-filter/) filter across transmitter and receiver so
the *combined* response is Nyquist and ISI-free at the sampler.

ISI shows up unmistakably on an [eye diagram](/reference/eye-diagram/): a clean Nyquist channel leaves a
wide-open eye, while ISI blurs the traces and narrows — or entirely closes — the opening, shrinking the
margin against noise. Two things reintroduce ISI even with good pulse shaping: sampling at the wrong
instant (a timing error moves you off the zero-crossings), and a dispersive channel such as multipath,
whose echoes are literally delayed copies of past symbols.

It is worth separating the two sources because the cures differ. ISI from filtering and pulse shaping
is fully under the system designer's control and is eliminated by choosing a Nyquist pulse and sampling
correctly — no channel knowledge needed. ISI from the propagation channel is not known in advance and
varies as the radio or reflectors move, so it cannot be designed away; it must be *measured and undone*
at the receiver by an adaptive equalizer that learns the channel's impulse response and subtracts the
interfering tails. A useful mental model is that a Nyquist-shaped symbol stream arriving through a
multipath channel is the clean stream convolved with the channel, and equalization is the deconvolution
that restores it.

## Variants

Not all ISI is unwanted. **Partial-response** signalling (such as duobinary and the Gaussian-filtered
[GMSK](/reference/gmsk/) used in GSM) deliberately introduces a controlled, known amount of ISI to
shrink bandwidth or smooth the phase trajectory, then removes its effect with a matched detector that
expects it. In that light the Nyquist criterion is not "no pulse overlap" but "no *uncontrolled* pulse
overlap at the decision instants" — overlap is fine as long as the receiver knows exactly what it is.

## Relevance to SDR

Controlling ISI is a central job of any digital demodulator, GopherTrunk's included. Its receivers apply a
root-raised-cosine matched filter so the composite pulse is Nyquist and ISI is nulled at the correct
sampling instant, and a timing-recovery loop keeps the sampler on those instants; drift off them and ISI
reappears as a closing eye and rising error rate. For the C4FM and π/4-DQPSK carriers in P25 and DMR the
symbol rate and RRC roll-off are specified precisely so transmitter and receiver agree on an ISI-free
composite. When the channel itself is dispersive — multipath in a mobile or simulcast environment — pulse
shaping alone is not enough and the residual ISI must be removed by an
[equalizer](/reference/decision-feedback-equalizer/) that estimates and subtracts the interfering symbol
tails.

## In practice

The amount of tail energy, and thus sensitivity to timing error, is governed by the pulse's
[roll-off factor](/reference/roll-off-factor/): a sharper (low-alpha) filter saves bandwidth but has longer,
larger tails that make the system less forgiving of timing jitter, a direct bandwidth-versus-robustness
trade.

## Sources

[^wiki]: [Intersymbol interference](https://en.wikipedia.org/wiki/Intersymbol_interference) — Wikipedia, for the definition and causes; Nyquist ISI criterion for the zero-crossing condition.
