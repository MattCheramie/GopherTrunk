---
slug: the-eye-diagram
title: The eye diagram
description: Overlaying symbol periods into an "eye" — how to read timing margin, noise, and inter-symbol interference at a glance from a single picture.
keywords: eye diagram, eye pattern, eye opening, timing margin, isi, symbol quality, decision instant, signal integrity
level: intermediate
status: full
prereq:
  - clock-and-symbol-recovery
  - snr-evm-and-ber
faq:
  - q: What is an eye diagram?
    a: "An eye diagram is made by chopping a demodulated signal into single-symbol-wide slices and drawing them all on top of each other. Where the traces avoid each other an open space appears — the eye. A wide-open eye means clean, easy-to-decide symbols; a closing eye means noise, timing jitter, or inter-symbol interference is eating away the margin the decoder relies on."
  - q: How do you read an eye diagram?
    a: "Look at the open space in the middle. Its height is the voltage margin between symbol levels — how much noise the decoder can tolerate before mistaking one level for another. Its width is the timing margin — how much the sampling instant can drift and still land in the clear. The best moment to sample is the widest, tallest point of the opening, right at the eye's centre."
---

# The eye diagram

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **eye diagram** overlays many single-symbol slices of a demodulated signal on top of
each other. The clear region that emerges — the **eye** — shows link quality at a glance:
its **height** is the amplitude margin against noise, its **width** is the timing margin
against jitter, and the best **sampling instant** is where the eye is most open. A wide,
open eye decodes cleanly; a **closing** eye warns of noise, jitter, or ISI.
</div>

The last lesson gave numeric grades ([SNR/EVM/BER](/learn/dsp/snr-evm-and-ber/)); the eye
diagram is their *picture*. It is the single most useful visual for symbol quality and
ties directly to [clock recovery](/learn/dsp/clock-and-symbol-recovery/), which decides
*where* in the eye to sample.

## Building the eye

Take the demodulated waveform — the signal stepping between symbol levels — and cut it
into pieces exactly one (or two) symbol periods wide. Now draw every piece on the same
axes, all starting at the same left edge. The random data means each slice takes a
different path, but they all pass through the same **transition zones** and avoid the
same **open centre**. That open centre is the eye.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 170" role="img" aria-label="An eye diagram: many overlaid traces forming an open lens-shaped eye in the centre, with the eye height marked as amplitude margin, eye width as timing margin, and a vertical line at the best sampling instant." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="150" x2="500" y2="150" stroke="currentColor" stroke-opacity="0.2"/>
  <g fill="none" stroke="currentColor" stroke-opacity="0.55" stroke-width="1.3">
    <path d="M40 40 C 140 40, 200 130, 265 130 C 330 130, 390 40, 490 40"/>
    <path d="M40 130 C 140 130, 200 40, 265 40 C 330 40, 390 130, 490 130"/>
    <path d="M40 40 C 150 45, 210 120, 265 120 C 320 120, 380 45, 490 40"/>
    <path d="M40 130 C 150 125, 210 50, 265 50 C 320 50, 380 125, 490 130"/>
  </g>
  <line x1="265" y1="52" x2="265" y2="118" stroke="currentColor" stroke-width="1.5" stroke-dasharray="3 3"/>
  <text x="278" y="88" font-size="9" fill="currentColor">amplitude margin (height)</text>
  <line x1="205" y1="140" x2="325" y2="140" stroke="currentColor" stroke-width="1.5" stroke-dasharray="3 3"/>
  <text x="265" y="165" text-anchor="middle" font-size="9" fill="currentColor">timing margin (width)</text>
  <line x1="265" y1="30" x2="265" y2="150" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="265" y="26" text-anchor="middle" font-size="8" fill="currentColor">best sample here</text>
</svg>
<figcaption>An eye diagram: overlaid symbol slices leave an open eye. Its height is noise margin, its width is timing margin, and the centre is the ideal sampling instant.</figcaption>
</figure>

## Reading the two margins

The eye's opening is a direct picture of decoding headroom:

- **Height (vertical opening)** — the gap between symbol levels at the decision instant.
  This is the **amplitude margin**: how much noise can push a sample before it crosses
  into the wrong level. Noise (low [SNR](/learn/dsp/snr-evm-and-ber/)) thickens the traces
  and shrinks the height.
- **Width (horizontal opening)** — how long the eye stays open. This is the **timing
  margin**: how far the sampling instant can drift and still land in the clear. Timing
  **jitter** narrows it; [inter-symbol interference](/learn/dsp/equalization/) pulls the
  crossings inward from both sides.

The decoder wants to sample at the **widest, tallest** point — the centre of the eye —
which is exactly the instant [clock recovery](/learn/dsp/clock-and-symbol-recovery/) hunts
for. A well-locked timing loop parks the sampler dead-centre in the eye.

## A closing eye names the problem

Because each impairment attacks the eye in its own way, a glance often diagnoses the
fault:

| Symptom in the eye | Likely cause |
|--------------------|--------------|
| Traces thick, eye short | noise / low SNR |
| Crossings blurred left–right | timing jitter |
| Opening squeezed, levels pulled in | ISI / multipath |
| Eye fully closed | any of the above, too severe to decode |

More symbol levels means more, smaller eyes stacked vertically (four-level
[C4FM](/learn/dsp/demodulation/) shows three eyes), and each must stay open. When the eye
is shut, no sampling instant is safe and the decode fails — the visual companion to a high
BER.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the eye's height is the amplitude (noise) margin and its width is the timing margin." markdown="0">
  <p class="knowledge-check__q">Quick check: the vertical opening (height) of the eye represents…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">the carrier frequency offset</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">the amplitude margin against noise before a level is misread</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">the number of taps in the channel filter</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **eye diagram** overlays single-symbol slices; the clear **eye** shows quality at a glance.
- **Height** = amplitude margin against noise; **width** = timing margin against jitter.
- The ideal **sampling instant** is the eye's centre — where clock recovery aims.
- A **closing** eye names the fault: thick traces (noise), blurred crossings (jitter), squeezed opening (ISI).

Next up: turning raw, error-prone symbols into trustworthy data — framing and error correction.
