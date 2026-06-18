---
slug: signal-anatomy
redirect_from: /learn/signal-anatomy/
title: Anatomy of a signal
description: Learn the parts of a radio signal — carrier and sidebands, bandwidth, and the difference between the spectrum (frequency) view and the waterfall (frequency-over-time) display — so you can recognise signals on an SDR before you decode them.
keywords: radio signal, carrier, sidebands, bandwidth, spectrum analyzer, waterfall display, reading a waterfall, signal shape, SDR display
level: beginner
status: full
prereq:
  - radio-waves
faq:
  - q: What is the bandwidth of a signal?
    a: Bandwidth is how wide a signal is in frequency — the span of spectrum it occupies, measured in hertz. A narrow FM voice channel might be ~12.5 kHz wide; an FM broadcast station ~200 kHz; Wi-Fi tens of megahertz. Wider signals can carry more data but take up more spectrum and need more receiver bandwidth to capture.
  - q: What is the difference between a spectrum view and a waterfall?
    a: A spectrum view plots signal strength against frequency right now — a snapshot, like a row of peaks. A waterfall plots the same information over time, scrolling, so each horizontal line is one moment and brightness shows strength. The spectrum shows the instant; the waterfall shows history, which makes intermittent signals and patterns easy to spot.
  - q: What are sidebands?
    a: When you modulate a carrier to add information, the energy spreads out into sidebands on either side of the carrier frequency. The width of those sidebands is essentially the signal's bandwidth. A bare unmodulated carrier is a single spike; a modulated signal is a wider shape because of its sidebands.
  - q: How do I recognise a signal type on a waterfall?
    a: By its shape and behaviour — width, whether it's continuous or bursty, and any structure. A control channel is usually a steady, fixed-width digital signal; voice calls come and go; FM broadcast is wide and constant. With practice the waterfall becomes a visual fingerprint of what's on the air.
gophertrunk_links:
  - title: Plots (signal scopes)
    url: /plots.html
    note: GopherTrunk's spectrum and waterfall views.
  - title: Web console
    url: /web.html
    note: where the live spectrum and waterfall appear.
---

# Anatomy of a signal

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A real signal is a **carrier** plus the **sidebands** that modulation spreads around
it; together they occupy a width of spectrum called **bandwidth**. An SDR shows
signals two ways: the **spectrum view** plots strength vs. frequency *right now*,
while the **waterfall** plots the same thing *over time*, scrolling, with brightness
for strength. Learning to read these — by a signal's width, shape, and whether it's
steady or bursty — lets you *recognise* what's on the air before decoding it.
</div>

Before we get into modulation, it helps to picture what a signal actually looks like
on an SDR screen — because that's how you'll find and identify everything for the rest
of the path.

## Carrier and sidebands

A bare **carrier** — a steady wave at one frequency — appears as a single tall spike.
But a carrier with no changes carries no information. The moment you
[modulate](/learn/rf-sdr/digital-modulation/) it (vary its amplitude, frequency, or phase),
the energy spreads into **sidebands** on either side of the carrier. The richer or
faster the information, the wider those sidebands — and the wider the signal.

So on screen, an idle carrier is a thin spike; an active, modulated signal is a wider
shape. That width has a name.

## What is bandwidth?

**Bandwidth** is how much spectrum a signal occupies, in hertz. It's one of the most
important numbers about any signal because it determines how much receiver bandwidth
you need to capture the whole thing (a theme that returns in
[sample rate & Nyquist](/learn/rf-sdr/sample-rate-nyquist/)).

| Signal | Typical bandwidth |
|--------|-------------------|
| Narrowband FM voice | ~12.5 kHz |
| P25 / DMR digital voice | ~12.5 kHz |
| FM broadcast station | ~200 kHz |
| Wi-Fi channel | 20–80 MHz |

Wider isn't "better" — it's a trade. More bandwidth carries more data but uses more
spectrum and demands more of your receiver and CPU.

This is why bandwidth is the *first* number to know about a signal: it sets how much
receiver bandwidth you must capture. To decode a 12.5 kHz P25 channel you only need to
[filter](/learn/rf-sdr/filtering-decimation/) out ~12.5 kHz around it — but to capture a whole
FM broadcast station you need ~200 kHz, sixteen times as much. Capture too little and
you clip the signal's edges and lose data; the bandwidth tells you exactly how wide to
open the window.

## The spectrum view vs. the waterfall

SDRs give you two windows onto the same data:

- The **spectrum view** (or "FFT view") plots **signal strength against frequency** at
  this instant. Peaks are signals; the wiggly baseline is the
  [noise floor](/learn/rf-sdr/decibels/). It's a live snapshot — great for *how strong* and
  *how wide* right now.
- The **waterfall** plots that same spectrum **over time**, scrolling downward (or
  up). Each horizontal line is one moment; **brightness or colour shows strength**.
  Because it keeps history, the waterfall makes intermittent signals, bursts, and
  repeating patterns jump out.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 190" role="img" aria-label="Top: a spectrum plot with a noisy baseline and two peaks. Bottom: a waterfall showing the same two signals as vertical stripes scrolling over time, one continuous and one bursty." xmlns="http://www.w3.org/2000/svg">
  <text x="6" y="14" font-size="10" fill="currentColor">spectrum (now)</text>
  <path d="M20 70 L60 66 L100 68 L130 66 L160 30 L175 66 L240 67 L290 66 L320 40 L335 66 L420 68" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <line x1="20" y1="80" x2="420" y2="80" stroke="currentColor" stroke-opacity="0.3"/>
  <text x="6" y="100" font-size="10" fill="currentColor">waterfall (over time)</text>
  <rect x="20" y="105" width="400" height="75" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-opacity="0.3"/>
  <rect x="160" y="105" width="14" height="75" fill="currentColor" fill-opacity="0.55"/>
  <rect x="316" y="105" width="14" height="22" fill="currentColor" fill-opacity="0.55"/>
  <rect x="316" y="150" width="14" height="18" fill="currentColor" fill-opacity="0.55"/>
  <text x="150" y="190" font-size="9" fill="currentColor">continuous</text>
  <text x="305" y="190" font-size="9" fill="currentColor">bursty</text>
</svg>
<figcaption>The spectrum shows the instant; the waterfall shows history. A steady control channel is a continuous stripe; voice calls come and go as bursts.</figcaption>
</figure>

## Reading signal strength and shape

On the spectrum, **height above the noise floor** is your [SNR](/learn/rf-sdr/decibels/) at a
glance — the taller a peak stands over the baseline, the better it'll decode. On the
waterfall, that's **brightness**.

Beyond strength, the *shape and behaviour* identify a signal:

- **Width** — narrow voice channel vs. wide broadcast.
- **Continuous vs. bursty** — a trunked [control channel](/learn/rf-sdr/what-is-trunking/) is
  a steady, fixed-width digital signal; individual voice calls flicker on and off.
- **Structure** — some signals have a distinctive look (evenly spaced tones, a flat
  digital "block," etc.).

With a little practice the waterfall becomes a **visual fingerprint**: you'll spot a
control channel, a pager burst, or an FM station without decoding anything — exactly
the skill you'll use in [finding systems](/learn/rf-sdr/finding-systems/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the waterfall adds the time axis, revealing bursts and patterns." markdown="0">
  <p class="knowledge-check__q">Quick check: what does a waterfall show that a plain spectrum view doesn't?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Higher frequencies</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">How signals change over time</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The exact modulation type</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A signal is a **carrier** plus **sidebands**; their span is its **bandwidth**.
- **Bandwidth** sets how much receiver bandwidth you need to capture a signal.
- The **spectrum view** shows strength vs. frequency now; the **waterfall** adds time.
- **Height/brightness** ≈ SNR; **width, burstiness, and structure** identify the type.

Next: the classic analog ways to put a voice on a carrier — AM, FM, and SSB.
