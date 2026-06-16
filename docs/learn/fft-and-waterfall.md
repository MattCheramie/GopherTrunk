---
slug: fft-and-waterfall
title: The FFT & reading a waterfall
description: The Fast Fourier Transform explained without the math — how an FFT turns IQ samples into a spectrum, what bins and resolution mean, and how to read GopherTrunk's spectrum and waterfall to spot, identify, and find control channels.
keywords: FFT, fast fourier transform, spectrum analyzer, waterfall display, reading a waterfall, FFT bins, frequency resolution, find control channel, SDR spectrum
level: intermediate
status: full
prereq:
  - iq-data
  - signal-anatomy
faq:
  - q: What does an FFT do in an SDR?
    a: An FFT (Fast Fourier Transform) converts a block of time-domain IQ samples into a frequency-domain spectrum — it measures how much energy is present at each frequency across the captured band. That spectrum is what an SDR draws as the peaks-and-valleys display and, stacked over time, as the waterfall. It's how raw samples become a picture you can read.
  - q: What are FFT bins and resolution?
    a: An FFT splits the captured bandwidth into a number of equal slices called bins; each bin reports the energy at one narrow frequency range. More bins (a larger FFT) give finer frequency resolution but update less often and cost more CPU. The resolution is roughly the sample rate divided by the FFT size.
  - q: How do I read a waterfall display?
    a: Frequency runs across the horizontal axis and time scrolls vertically; brightness or colour shows signal strength. A bright vertical stripe is a signal present at that frequency; how it behaves over time — steady, bursty, or patterned — tells you what kind of signal it is. A steady, fixed-width stripe is often a control channel.
  - q: How do I find a control channel on the waterfall?
    a: Look for a continuous, fixed-width digital signal that's always on, usually in the band your target system uses. Voice channels flicker on and off as calls come and go, but the control channel transmits constantly. Its steady presence is its signature, and tools like GopherTrunk's Hunt automate the search.
gophertrunk_links:
  - title: Plots (signal scopes)
    url: /plots.html
    note: GopherTrunk's live spectrum and waterfall.
  - title: Hunt (discover systems)
    url: /hunt.html
    note: automatically scan a band for steady control channels.
---

# The FFT & reading a waterfall

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The **FFT (Fast Fourier Transform)** turns a block of [IQ samples](/learn/iq-data/) into
a **spectrum** — energy at each frequency across the band. Stack spectra over time and
you get the **waterfall**. The FFT splits the band into **bins**; more bins = finer
**resolution** but slower updates and more CPU. You read a waterfall by **frequency
(across), time (scrolling), strength (brightness)** — and a **steady, fixed-width
stripe** is the tell-tale of a [control channel](/learn/what-is-trunking/). No calculus
required.
</div>

The FFT is the engine behind every spectrum display you'll ever use, and the waterfall
is how you'll actually *find* signals. You can use both expertly without touching the
underlying math — here's the working understanding.

## What the FFT does (without the math)

Your SDR produces [IQ samples](/learn/iq-data/) over *time*. But to *see* signals you
want them organised by *frequency*. The **FFT** does exactly that conversion: feed it a
block of samples and it tells you **how much energy sits at each frequency** across the
captured [bandwidth](/learn/sample-rate-nyquist/). 

Think of it as a prism for radio: a jumble of samples goes in, and out comes a tidy
breakdown of "this much at 851.0 MHz, this much at 851.2 MHz…". Draw that as height
versus frequency and you have the [spectrum view](/learn/signal-anatomy/); the FFT runs
many times a second to keep it live.

## Bins, resolution, and frame rate

The FFT divides the band into a fixed number of equal slices called **bins**. Each bin
reports the energy in one narrow frequency range. The number of bins is the **FFT
size** (512, 1024, 4096…), and it sets a trade-off:

| Bigger FFT | Smaller FFT |
|------------|-------------|
| Finer frequency **resolution** | Coarser resolution |
| Slower updates (more samples per frame) | Faster, smoother updates |
| More CPU | Less CPU |

As a rule, **resolution ≈ sample rate ÷ FFT size**. So at 2.4 MSa/s a 2048-point FFT
gives bins about 1.2 kHz wide — fine enough to separate adjacent channels. You don't
usually set this by hand, but knowing it explains why a display can look "blocky" (too
few bins) or sluggish (too many).

## Reading the spectrum view

The spectrum is a snapshot: **frequency across, strength up.** Peaks are signals; the
restless baseline is the [noise floor](/learn/decibels/). The **height of a peak above
that floor is its SNR at a glance** — the taller it stands, the better it'll
[decode](/learn/antenna-to-audio/). Width tells you roughly what kind of signal it is
(narrow voice channel vs. wide broadcast), as covered in
[anatomy of a signal](/learn/signal-anatomy/).

## Reading the waterfall view

The **waterfall** plots that same spectrum **over time**: frequency across, time
scrolling, and **brightness/colour for strength**. Because it keeps history, it reveals
behaviour the snapshot can't:

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 150" role="img" aria-label="A waterfall: frequency across, time down. One bright continuous vertical stripe labelled control channel, and two short bright blocks at another frequency labelled voice calls." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="20" width="380" height="110" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-opacity="0.3"/>
  <rect x="130" y="20" width="12" height="110" fill="currentColor" fill-opacity="0.6"/>
  <text x="136" y="14" text-anchor="middle" font-size="9" fill="currentColor">control</text>
  <rect x="300" y="28" width="12" height="26" fill="currentColor" fill-opacity="0.6"/>
  <rect x="300" y="78" width="12" height="20" fill="currentColor" fill-opacity="0.6"/>
  <text x="306" y="14" text-anchor="middle" font-size="9" fill="currentColor">voice</text>
  <text x="230" y="146" text-anchor="middle" font-size="10" fill="currentColor">frequency →</text>
  <text x="30" y="75" font-size="10" fill="currentColor" transform="rotate(-90 30 75)">time ↓</text>
</svg>
<figcaption>On a waterfall, a control channel is a continuous stripe (always on); voice calls are intermittent blocks that come and go.</figcaption>
</figure>

## Identifying signals by their footprint

Each signal type has a recognisable look on the waterfall:

- **Control channel** — continuous, fixed-width, *always on*.
- **Voice calls** — intermittent blocks that appear and vanish.
- **FM broadcast** — wide, constant, with audio "texture."
- **Pagers** — periodic bursts.

This is pattern recognition, and it comes fast with practice. The waterfall becomes a
map of what's on the air before you decode a single bit.

## Tips for spotting a control channel

To bring a trunked system into GopherTrunk you first need its
[control channel](/learn/what-is-trunking/):

1. Tune to the [band](/learn/frequency-and-spectrum/) the system uses.
2. Scan the waterfall for a **steady, fixed-width digital stripe that never turns off** —
   voice channels flicker, the control channel doesn't.
3. Confirm by checking it against a [systems database](/learn/finding-systems/).
4. Or let **[Hunt](/hunt.html)** sweep the band and surface candidates automatically.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the control channel is the one that's always on." markdown="0">
  <p class="knowledge-check__q">Quick check: on a waterfall, which signal is most likely the control channel?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A block that appears for a few seconds then vanishes</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A steady, fixed-width stripe that's always present</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The widest signal on screen</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The **FFT** converts IQ samples into a **spectrum** (energy vs. frequency).
- **Bins/FFT size** set **resolution** vs. update speed vs. CPU (resolution ≈ rate ÷ size).
- Read the **spectrum** for strength/SNR and width; read the **waterfall** for behaviour over time.
- Signals have **footprints**; a **steady fixed-width stripe** flags a control channel.

Next: how software zooms in on just one of those channels — filtering and decimation.
