---
slug: tuning-with-scopes
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
([antenna](/learn/antennas/), [placement](/learn/propagation/),
[gain](/learn/gain-and-agc/)), then read the scopes to fix what's left.
</div>

You've assembled the whole [signal path](/learn/antenna-to-audio/). This lesson is the
applied skill that separates frustration from success: using the scopes you met in
[digital modulation](/learn/digital-modulation/) to diagnose and fix a signal that won't
lock cleanly.

## What a good lock looks like

A healthy decode has a recognisable signature across the scopes:

- **Constellation** — a few **tight, well-separated clusters** at the expected symbol
  positions (four for [4FSK/C4FM](/learn/digital-modulation/)).
- **Eye diagram** — **wide-open eyes** with clear gaps between levels.
- **Symbol scope** — steady, **distinct levels** that don't wander.
- **Histogram** — sharp **peaks centred** on each symbol level.

If you see that, you're done — leave it alone. When you don't, the *way* it's wrong is
the clue.

## Reading the constellation for SNR and tuning

The [constellation](/constellation.html) is your first stop. Two failure modes have
distinct looks:

| What you see | Likely cause | Fix |
|--------------|--------------|-----|
| Clusters fuzz outward, evenly | **Low SNR** | More signal: [antenna](/learn/antennas/)/[placement](/learn/propagation/), correct [gain](/learn/gain-and-agc/) |
| Pattern **rotates or spins** | **Frequency/tuning offset** | [PPM correction](/learn/calibration-troubleshooting/) |
| Clusters distorted, smeared unevenly + ghost signals | **ADC clipping** (too much gain) | Reduce [gain](/learn/gain-and-agc/) |

So a *rotating* constellation isn't a strength problem at all — it's a tuning problem,
solved by [calibration](/learn/calibration-troubleshooting/), not a better antenna.

## Reading the eye diagram for timing

The [eye diagram](/eye-diagram.html) reveals [timing](/learn/clock-recovery/) and noise
margin. A **wide-open** eye means the [symbol-recovery](/learn/demodulation-pipeline/)
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
   [antenna](/learn/antennas/) up and clear ([placement](/learn/propagation/)), and set
   [gain](/learn/gain-and-agc/) by the routine in that lesson. Watch SNR on the
   [tuning meters](/tuning.html).
2. **Check the constellation.** Fuzzy → keep improving SNR. Rotating → suspect a
   **frequency offset**; apply [PPM correction](/learn/calibration-troubleshooting/).
   Distorted with ghosts → **reduce gain** (clipping).
3. **Check the eye.** If SNR is good but the eye won't open, suspect
   [timing/clock](/learn/clock-recovery/) — usually still cured by more SNR.
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
