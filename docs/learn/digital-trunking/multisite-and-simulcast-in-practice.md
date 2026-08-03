---
slug: multisite-and-simulcast-in-practice
title: Multi-site & simulcast in practice
description: Handle multi-site and simulcast systems with GopherTrunk — pick the closest site's control channel, recognise and mitigate simulcast distortion with a better antenna, site bias and careful gain, and know the limits of coverage.
keywords: multi-site trunking, simulcast distortion, simulcast decoding, choosing a site, control channel selection, antenna for simulcast, gain for simulcast, coverage limits, site roaming
level: advanced
status: full
prereq:
  - sites-simulcast-roaming
  - following-a-system-end-to-end
faq:
  - q: Which site's control channel should I monitor on a multi-site system?
    a: The closest or strongest site you can hear cleanly. Each site has its own control channel and its own coverage area, so monitoring the site nearest you gives the strongest signal and the traffic most relevant to your location. If you can hear several, pick the one with the cleanest constellation and most reliable lock, not necessarily the loudest.
  - q: What is simulcast distortion and why does it hurt decoding?
    a: Simulcast systems transmit the same signal from several towers at once on the same frequency. Where their coverage overlaps, the signals arrive at slightly different times and combine, smearing the waveform much like multipath. This distortion fuzzes the constellation and raises the error rate even when the raw signal is strong, which is why simulcast is one of the hardest cases to decode.
  - q: How do I mitigate simulcast distortion?
    a: Bias your reception toward a single transmitter. A directional antenna aimed at the nearest tower, careful siting to favour one site, and conservative gain to avoid overload all reduce the blend of competing copies. You can't remove simulcast distortion entirely, but making one transmitter dominate cleans up the constellation enough to lock.
  - q: What if a system covers more area than I can hear?
    a: You simply can't follow what doesn't reach your antenna. A wide-area system may have sites whose control channels are below your noise floor; you'll only follow the traffic on sites you can receive. Improving the antenna and its placement extends your reach, but geography and the system's design set a hard ceiling on coverage.
gophertrunk_links:
  - title: Tuning (receiver meters)
    url: /tuning.html
    note: judge lock quality per site while choosing which to monitor.
  - title: Demod calibration
    url: /demod-calibration.html
    note: tune the demodulator to wring a lock out of a smeared simulcast signal.
---

# Multi-site & simulcast in practice

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Big systems span **multiple sites**, each with its own control channel and coverage
area — so monitor the **closest, cleanest site** you can hear, not necessarily the
loudest. **Simulcast** sites transmit the same frequency from several towers at once;
where coverage overlaps the copies blend and **smear the constellation**, much like
[multipath](/learn/rf-sdr/propagation/). You can't erase that distortion, but you can
**bias reception toward one transmitter** with a directional antenna, careful siting, and
conservative gain. And accept the hard limit: a system that **covers more area than you
can hear** will only ever give you the sites that reach your antenna.
</div>

The [sites, simulcast and roaming](/learn/digital-trunking/sites-simulcast-roaming/) lesson explained the
architecture; this one is about *operating* against it. Multi-site and simulcast are the
two situations where a perfectly configured system still fights you, so it's worth
knowing the practical moves.

## Choosing which site to monitor

A multi-site system has **one control channel per site**, each covering its own area. You
can only lock one control channel per receiver, so the question is *which*. The answer is
the **closest, cleanest site you can hear** — usually the one nearest you, because it
arrives strongest and carries the traffic most relevant to your location.

"Cleanest" matters as much as "loudest." Tune each candidate site's control channel and
compare lock quality on the [tuning meters](/tuning.html) and
[constellation](/constellation.html): a slightly weaker site with a tight, stable
constellation will decode more reliably than a louder one that's smeared by simulcast or
overload. Pick the one that *locks best*, then let GopherTrunk follow that site's calls.

## What simulcast distortion looks like

[Simulcast](/learn/digital-trunking/sites-simulcast-roaming/) sites broadcast the **same signal from several
towers simultaneously on the same frequency** — a design that extends coverage cheaply
but punishes receivers in the overlap zones. There, copies from different towers arrive
at **slightly different times** and combine, smearing the waveform the same way
[multipath](/learn/rf-sdr/propagation/) does.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 180" role="img" aria-label="Two towers each sending the same signal to a receiver in the middle. The two paths have different lengths, so the copies arrive at different times and combine into a smeared waveform." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <!-- towers -->
    <path d="M60 120 L52 60 L68 60 Z" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="60" y="138">Tower A</text>
    <path d="M480 120 L472 60 L488 60 Z" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="480" y="138">Tower B</text>
    <!-- receiver -->
    <circle cx="270" cy="95" r="8" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="1.3"/><text x="270" y="120">receiver</text>
    <!-- paths -->
    <line x1="62" y1="62" x2="262" y2="92" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3"/>
    <line x1="478" y1="62" x2="278" y2="92" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3"/>
    <text x="160" y="64" font-size="8.5">copy 1 (short path)</text>
    <text x="378" y="64" font-size="8.5">copy 2 (long path — arrives later)</text>
    <!-- smeared result -->
    <text x="270" y="158" font-size="9">copies combine → smeared symbols, fuzzy constellation</text>
  </g>
</svg>
<figcaption>In the overlap zone, the same signal arrives from two towers at slightly different times. The copies add up into a smeared waveform — the constellation fuzzes even though the raw signal is strong.</figcaption>
</figure>

The tell is a **strong signal meter but a fuzzy constellation** and a stubbornly high
error rate. Raising gain won't help — the problem isn't weakness, it's the blend of
competing copies.

## Mitigating it — bias toward one transmitter

You can't remove simulcast distortion, but you can make **one tower dominate** so its copy
overwhelms the others:

- **A directional antenna** aimed at the nearest tower is the single biggest win — it
  amplifies one copy and rejects the others.
- **Careful siting** — moving the antenna so terrain or buildings favour one site — has
  the same effect for free.
- **Conservative gain** keeps the strong, blended signal from also **overloading** the
  front end, which would pile distortion on distortion. Set gain just high enough for a
  clean lock, no more.
- **[Demod calibration](/demod-calibration.html)** can help the demodulator wring a lock
  out of a marginally smeared signal once the antenna and gain are doing their part.

The goal isn't a perfect signal — it's making the constellation tight enough to decode,
by tilting the balance toward a single transmitter. The
[tuning-with-scopes](/learn/rf-sdr/tuning-with-scopes/) techniques are exactly how you
read whether each change is helping.

## When the system is bigger than your horizon

Some systems simply **cover more area than you can hear**. A statewide network may have
dozens of sites; from one location you'll receive only the few whose control channels
clear your noise floor. There's no software fix for a control channel that doesn't reach
your antenna — you follow the sites you can hear and accept the rest are out of range.

Improving the **antenna and its placement** — higher, clearer, better matched — extends
your reach and may pull in another site or two. But geography and the system's design set
a hard ceiling. Knowing that ceiling exists stops you from chasing a "fault" that's really
just distance.

| Situation | Symptom | Practical move |
|-----------|---------|----------------|
| Several sites audible | Multiple control channels lock | Monitor the **closest, cleanest** site |
| In a simulcast overlap | Strong meter, **fuzzy constellation** | Directional antenna, favour one tower, lower gain |
| Front-end overloaded | Smearing *and* ghost signals | **Reduce gain**, add attenuation |
| Site out of range | No lock on a known control channel | Better antenna/placement, or accept it's too far |

<div class="knowledge-check" data-quiz data-correct-msg="Right — a strong meter with a fuzzy constellation is the classic simulcast overlap; bias toward one tower." markdown="0">
  <p class="knowledge-check__q">Quick check: the signal meter is strong but the constellation is fuzzy and won't lock. On a simulcast system, what's the best first move?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Increase the gain to push through the fuzz</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Use a directional antenna to favour one tower</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Switch to a different talkgroup</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A multi-site system has **one control channel per site** — monitor the **closest,
  cleanest** one you can hear.
- **Simulcast** overlap **smears the constellation** even when the signal is strong — like
  [multipath](/learn/rf-sdr/propagation/).
- Mitigate by **biasing toward one transmitter**: directional antenna, careful siting,
  conservative gain.
- **[Demod calibration](/demod-calibration.html)** can squeeze a lock out of a marginally
  smeared signal.
- A system **bigger than your horizon** gives you only the sites that reach your antenna —
  that's a hard limit, not a fault.

Last in the module: a systematic checklist for when a system still won't decode, in
[troubleshooting a digital decode](/learn/digital-trunking/troubleshooting-a-decode/).
