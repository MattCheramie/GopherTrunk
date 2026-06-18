---
slug: tuning-with-scopes
redirect_from: /learn/tuning-with-scopes/
title: Tuning for a clean lock
description: Use GopherTrunk's constellation, eye diagram, symbol scope, and histogram to dial in a marginal signal — reading each scope to diagnose SNR, tuning error, and clock problems, with a step-by-step routine for a clean lock.
keywords: tune SDR, clean lock, constellation diagram, eye diagram, symbol scope, symbol histogram, diagnose signal, SNR tuning, frequency offset, P25 DMR tuning
level: advanced
status: full
prereq:
  - digital-modulation
  - antenna-to-audio
faq:
  - q: How do I get a clean lock on a control channel?
    a: Maximise SNR first — a good antenna, placement, and correct gain — then use the scopes to fine-tune. A clean lock shows tight, well-separated clusters on the constellation, a wide-open eye diagram, and steady, distinct levels on the symbol scope. If those look smeared or rotating, work the cause (low SNR, frequency offset, or clipping) rather than guessing.
  - q: What does a smeared constellation mean?
    a: Smearing means the symbols aren't landing cleanly on their ideal positions, so the decoder is starting to make errors. Common causes are low signal-to-noise ratio (the whole pattern fuzzes outward), a frequency/tuning offset (the pattern rotates or spins), or ADC clipping from too much gain (distortion). The shape of the smear points to the cause.
  - q: Why is my signal strong but still not decoding?
    a: Strength isn't everything. A strong signal can still fail from a frequency offset (needs PPM correction), too much gain causing clipping, multipath smearing the symbols, or simply being the wrong system/parameters. The scopes separate these — a rotating constellation suggests frequency offset; distortion suggests clipping; general fuzz suggests low SNR despite the strong meter.
  - q: What's the difference between the constellation, eye diagram, and symbol scope?
    a: They show the same recovered symbols three ways. The constellation plots symbols on the IQ plane (amplitude and phase). The eye diagram plots them against time to show timing margin. The symbol scope streams the recovered symbol levels so you can watch stability. Together they let you tell an SNR problem from a timing problem from a tuning problem.
gophertrunk_links:
  - title: Constellation
    url: /constellation.html
    note: the IQ-plane view for SNR and frequency offset.
  - title: Eye diagram
    url: /eye-diagram.html
    note: timing margin at a glance.
  - title: Symbol scope
    url: /symbol-scope.html
    note: live recovered symbol levels.
  - title: Symbol histogram
    url: /histogram.html
    note: confirm symbol levels are crisp and centred.
---

# Tuning for a clean lock

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
When a control channel is marginal, GopherTrunk's scopes tell you *why*. The
**[constellation](/constellation.html)** shows SNR and frequency offset (tight clusters =
good; fuzzy = low SNR; rotating = tuning offset). The
**[eye diagram](/eye-diagram.html)** shows timing margin (open eye = good). The
**[symbol scope](/symbol-scope.html)** and **[histogram](/histogram.html)** show whether
recovered levels are crisp and centred. The routine: **maximise SNR first**
([antenna](/learn/rf-sdr/antennas/), [placement](/learn/rf-sdr/propagation/),
[gain](/learn/rf-sdr/gain-and-agc/)), then read the scopes to fix what's left.
</div>

You've assembled the whole [signal path](/learn/rf-sdr/antenna-to-audio/). This lesson is the
applied skill that separates frustration from success: using the scopes you met in
[digital modulation](/learn/rf-sdr/digital-modulation/) to diagnose and fix a signal that won't
lock cleanly.

## What a good lock looks like

A healthy decode has a recognisable signature across the scopes:

- **Constellation** — a few **tight, well-separated clusters** at the expected symbol
  positions (four for [4FSK/C4FM](/learn/rf-sdr/digital-modulation/)).
- **Eye diagram** — **wide-open eyes** with clear gaps between levels.
- **Symbol scope** — steady, **distinct levels** that don't wander.
- **Histogram** — sharp **peaks centred** on each symbol level.

If you see that, you're done — leave it alone. When you don't, the *way* it's wrong is
the clue.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 180" role="img" aria-label="Healthy versus degraded scopes. On the left, a constellation with four tight clusters and a wide-open eye diagram. On the right, smeared constellation clusters and a closing eye diagram." xmlns="http://www.w3.org/2000/svg">
  <text x="130" y="16" text-anchor="middle" font-size="11" fill="currentColor" font-weight="600">healthy lock</text>
  <text x="390" y="16" text-anchor="middle" font-size="11" fill="currentColor" font-weight="600">degraded</text>
  <!-- healthy constellation -->
  <g fill="currentColor">
    <circle cx="60" cy="60" r="2.5"/><circle cx="58" cy="62" r="2.5"/><circle cx="62" cy="59" r="2.5"/>
    <circle cx="110" cy="60" r="2.5"/><circle cx="108" cy="62" r="2.5"/><circle cx="112" cy="58" r="2.5"/>
    <circle cx="60" cy="100" r="2.5"/><circle cx="62" cy="98" r="2.5"/><circle cx="59" cy="101" r="2.5"/>
    <circle cx="110" cy="100" r="2.5"/><circle cx="108" cy="99" r="2.5"/><circle cx="112" cy="101" r="2.5"/>
  </g>
  <text x="85" y="124" text-anchor="middle" font-size="8" fill="currentColor">tight clusters</text>
  <!-- healthy eye -->
  <g stroke="currentColor" fill="none" stroke-width="1.1" stroke-opacity="0.85">
    <path d="M160 45 C185 45 185 100 210 100 C235 100 235 45 260 45"/>
    <path d="M160 100 C185 100 185 45 210 45 C235 45 235 100 260 100"/>
  </g>
  <text x="210" y="124" text-anchor="middle" font-size="8" fill="currentColor">open eye</text>
  <!-- degraded constellation -->
  <g fill="currentColor" fill-opacity="0.7">
    <circle cx="318" cy="58" r="2.5"/><circle cx="310" cy="66" r="2.5"/><circle cx="326" cy="52" r="2.5"/><circle cx="320" cy="72" r="2.5"/>
    <circle cx="372" cy="60" r="2.5"/><circle cx="380" cy="70" r="2.5"/><circle cx="366" cy="54" r="2.5"/><circle cx="384" cy="64" r="2.5"/>
    <circle cx="320" cy="100" r="2.5"/><circle cx="312" cy="106" r="2.5"/><circle cx="328" cy="96" r="2.5"/>
    <circle cx="374" cy="100" r="2.5"/><circle cx="382" cy="106" r="2.5"/><circle cx="366" cy="96" r="2.5"/>
  </g>
  <text x="345" y="124" text-anchor="middle" font-size="8" fill="currentColor">smeared</text>
  <!-- degraded eye -->
  <g stroke="currentColor" fill="none" stroke-width="1.1" stroke-opacity="0.85">
    <path d="M420 55 C445 60 445 90 470 90 C495 90 495 60 520 55"/>
    <path d="M420 90 C445 85 445 60 470 60 C495 60 495 85 520 90"/>
  </g>
  <text x="470" y="124" text-anchor="middle" font-size="8" fill="currentColor">closing eye</text>
  <line x1="290" y1="30" x2="290" y2="135" stroke="currentColor" stroke-opacity="0.25"/>
</svg>
<figcaption>The same signal, healthy vs. degraded. Tight clusters and an open eye decode cleanly; smeared clusters and a closing eye are about to drop symbols.</figcaption>
</figure>

## Reading the constellation for SNR and tuning

The [constellation](/constellation.html) is your first stop. Two failure modes have
distinct looks:

| What you see | Likely cause | Fix |
|--------------|--------------|-----|
| Clusters fuzz outward, evenly | **Low SNR** | More signal: [antenna](/learn/rf-sdr/antennas/)/[placement](/learn/rf-sdr/propagation/), correct [gain](/learn/rf-sdr/gain-and-agc/) |
| Pattern **rotates or spins** | **Frequency/tuning offset** | [PPM correction](/learn/rf-sdr/calibration-troubleshooting/) |
| Clusters distorted, smeared unevenly + ghost signals | **ADC clipping** (too much gain) | Reduce [gain](/learn/rf-sdr/gain-and-agc/) |

So a *rotating* constellation isn't a strength problem at all — it's a tuning problem,
solved by [calibration](/learn/rf-sdr/calibration-troubleshooting/), not a better antenna.

## Reading the eye diagram for timing

The [eye diagram](/eye-diagram.html) reveals [timing](/learn/rf-sdr/clock-recovery/) and noise
margin. A **wide-open** eye means the [symbol-recovery](/learn/rf-sdr/demodulation-pipeline/)
stage has plenty of room to sample each symbol correctly. A **closing or blurred** eye
means margin is being eaten — by low SNR or a timing problem — and errors are imminent.
For 4-level signals you'll see **three stacked eyes**; all three should be open.

## Using the symbol scope and histogram

The [symbol scope](/symbol-scope.html) streams the recovered symbol levels in real time —
when locked, they sit at steady, well-separated values; when struggling, they jitter and
collapse together. The [symbol histogram](/histogram.html) is the statistical view:
**sharp, centred peaks** at each level mean clean symbols; smeared or shifted peaks
confirm SNR or offset problems. Together they corroborate what the constellation hints.

## A step-by-step tuning routine

1. **Maximise SNR first.** Most lock problems are really signal problems. Get the
   [antenna](/learn/rf-sdr/antennas/) up and clear ([placement](/learn/rf-sdr/propagation/)), and set
   [gain](/learn/rf-sdr/gain-and-agc/) by the routine in that lesson. Watch SNR on the
   [tuning meters](/tuning.html).
2. **Check the constellation.** Fuzzy → keep improving SNR. Rotating → suspect a
   **frequency offset**; apply [PPM correction](/learn/rf-sdr/calibration-troubleshooting/).
   Distorted with ghosts → **reduce gain** (clipping).
3. **Check the eye.** If SNR is good but the eye won't open, suspect
   [timing/clock](/learn/rf-sdr/clock-recovery/) — usually still cured by more SNR.
4. **Confirm with symbol scope/histogram.** Steady levels and crisp peaks = a solid lock.
5. **Lock it in** and let GopherTrunk follow calls.

Work the scopes in that order and you stop guessing — each view rules a cause in or out.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a rotating/spinning constellation points to a frequency offset." markdown="0">
  <p class="knowledge-check__q">Quick check: the constellation clusters are tight but the whole pattern is slowly rotating. What's the likely cause?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Low SNR — get a better antenna</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A frequency/tuning offset — apply PPM correction</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The system is encrypted</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The scopes diagnose *why* a signal won't lock — each view rules a cause in or out.
- **Constellation:** fuzzy = low SNR; rotating = frequency offset; distorted = clipping.
- **Eye diagram:** open = healthy timing margin; closing = trouble.
- **Symbol scope/histogram:** steady, crisp levels confirm a clean lock.
- Routine: **maximise SNR first**, then read the scopes in order and fix the specific cause.

Next: the calibration behind that frequency offset, and a full troubleshooting checklist.
