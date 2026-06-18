---
slug: sample-rate-nyquist
redirect_from: /learn/sample-rate-nyquist/
title: Sample rate, bandwidth & Nyquist
description: Sample rate, bandwidth, and the Nyquist theorem made simple — how much spectrum an SDR captures at once, why sample rate equals IQ bandwidth, what aliasing is, the bandwidth-versus-CPU trade-off, and choosing a rate for trunk-tracking.
keywords: sample rate, Nyquist theorem, bandwidth, aliasing, IQ bandwidth, SDR sample rate, decimation, sample rate vs CPU, RTL-SDR sample rate
level: intermediate
status: full
prereq:
  - iq-data
faq:
  - q: How much spectrum can an SDR capture at once?
    a: With IQ (complex) sampling, the captured bandwidth roughly equals the sample rate. An RTL-SDR running at 2.4 million samples per second captures about 2.4 MHz of spectrum at once. To see or decode signals across a wider span you either raise the sample rate (if the hardware allows) or retune to a different centre frequency.
  - q: What is the Nyquist theorem in simple terms?
    a: The Nyquist theorem says you must sample at least twice as fast as the highest frequency you want to represent, or you lose information. For SDRs using IQ sampling, the practical version is that the usable bandwidth is about equal to the sample rate. Sample too slowly for the bandwidth you're capturing and signals fold back as aliases.
  - q: What is aliasing?
    a: Aliasing is when a signal outside the bandwidth your sample rate can represent gets "folded" back into your captured spectrum, appearing at the wrong frequency. It looks like a real signal but isn't where it seems. Anti-aliasing filtering and choosing an adequate sample rate prevent it.
  - q: Does a higher sample rate always help?
    a: No. A higher sample rate captures more spectrum but produces more data, so it uses more USB bandwidth and more CPU to process. For trunk-tracking you only need enough sample rate to cover the channels you're following. Capturing a huge span you don't use just wastes CPU and can stress the hardware.
gophertrunk_links:
  - title: Architecture
    url: /architecture.html
    note: how GopherTrunk channelises a wideband IQ stream.
  - title: Hardware guide
    url: /hardware.html
    note: typical sample rates for each supported radio.
---

# Sample rate, bandwidth & Nyquist

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An SDR's **sample rate** (samples per second) sets how much spectrum it captures at
once: with [IQ sampling](/learn/rf-sdr/iq-data/), **captured bandwidth ≈ sample rate**. The
**Nyquist theorem** is the rule behind this — sample too slowly for the bandwidth and
signals **alias** (fold back to the wrong frequency). Higher sample rate = more
spectrum but **more data, more CPU, more USB load**. For trunk-tracking you want *just
enough* rate to cover the channels you follow — not the widest possible span.
</div>

You know the SDR streams [IQ samples](/learn/rf-sdr/iq-data/). How wide a slice of spectrum do
those samples represent, and how fast must they come? That's this lesson — and it
directly sets one of the few numbers you'll configure.

## Sample rate and what it buys you

The **sample rate** is how many [IQ samples](/learn/rf-sdr/iq-data/) per second the ADC
produces, in samples/second (Sa/s) or, loosely, "MHz." Its headline effect is simple:

> With complex (IQ) sampling, the **captured bandwidth is approximately equal to the
> sample rate.**

So an RTL-SDR at **2.4 MSa/s** shows you about **2.4 MHz** of spectrum at once, centred
on whatever frequency the [LO](/learn/rf-sdr/sdr-receiver/) is tuned to. Want to see more than
that span? Raise the rate (within hardware limits) or **retune** the centre frequency
to a different chunk.

## The Nyquist theorem, plainly

The **Nyquist theorem** is the law underneath that rule of thumb. In its general form:
*to represent a signal faithfully you must sample at least twice its highest
frequency.* For SDRs doing IQ sampling, the practical takeaway is the one above —
**usable bandwidth ≈ sample rate**. Try to capture more bandwidth than your sample rate
supports and the information doesn't just vanish; it corrupts your data through
aliasing.

## What is aliasing?

**Aliasing** is when energy outside the bandwidth your sample rate can represent gets
**folded** back into your captured spectrum, showing up at a *wrong* frequency. It can
look exactly like a genuine signal — a phantom that isn't really there (or is really
somewhere else). 

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 140" role="img" aria-label="A captured band between two dashed Nyquist edges. A real signal sits inside; an out-of-band signal beyond the edge is shown folding back to appear inside as an alias." xmlns="http://www.w3.org/2000/svg">
  <line x1="60" y1="110" x2="420" y2="110" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="110" y1="20" x2="110" y2="120" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.6"/>
  <line x1="330" y1="20" x2="330" y2="120" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.6"/>
  <text x="220" y="134" text-anchor="middle" font-size="10" fill="currentColor">captured bandwidth (≈ sample rate)</text>
  <path d="M170 110 L180 60 L190 110 Z" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
  <text x="180" y="52" text-anchor="middle" font-size="9" fill="currentColor">real</text>
  <path d="M360 110 L370 75 L380 110 Z" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="370" y="68" text-anchor="middle" font-size="9" fill="currentColor">out of band</text>
  <path d="M260 110 L270 80 L280 110 Z" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-dasharray="3 2"/>
  <text x="270" y="73" text-anchor="middle" font-size="9" fill="currentColor">alias!</text>
  <path d="M370 78 q-50 -30 -100 0" fill="none" stroke="currentColor" stroke-opacity="0.5" stroke-dasharray="2 2"/>
</svg>
<figcaption>A signal beyond the captured bandwidth folds back ("aliases") to appear at a false position inside it. Anti-alias filtering and adequate sample rate prevent this.</figcaption>
</figure>

SDR front ends include **anti-aliasing filtering**, and choosing an adequate sample
rate keeps real signals safely inside the usable window. The edges of that window can be
less clean, so it's common to keep your target a little away from the very edge.

## The bandwidth vs. CPU trade-off

It's tempting to crank the sample rate to "see everything," but more bandwidth has
costs:

| Higher sample rate | Consequence |
|--------------------|-------------|
| More spectrum captured | Good — more channels at once |
| More samples/second | More **USB bandwidth** (can cause dropped samples) |
| More data to process | More **CPU** for FFTs, filtering, demod |
| Cheap-tuner edge quality | Wider spans expose front-end limits |

GopherTrunk takes a wide IQ stream and **channelises** it — digitally
[filtering and decimating](/learn/rf-sdr/filtering-decimation/) out each narrow channel it
needs. So the question isn't "how wide can I go" but "how wide do I *need*."

**Worked example.** Suppose a system's control channel is at 851.0 MHz and its voice
channels span up to 853.4 MHz — a 2.4 MHz spread. Centre an RTL-SDR at 852.2 MHz and
run it at **2.4 MSa/s**: the whole system fits in one capture, with no retuning.
Each 12.5 kHz channel is then filtered out and decimated down to perhaps **24 kSa/s**
— a **100×** reduction — before demodulation. That's the magic: you sample the *band*
fast, but each *channel* slowly, so a modest CPU can follow many at once.

## Choosing a sample rate for trunk-tracking

A trunked system's [control channel](/learn/rf-sdr/what-is-trunking/) and its voice channels
may be spread across some span of a band. You want a sample rate (and centre frequency)
that **covers the channels you actually follow**, with a little margin, and no more:

- If all your channels fit within ~2 MHz, an RTL-SDR's 2.4 MSa/s is plenty.
- If they're spread wider, you may need a wider-bandwidth radio (Airspy) or a second
  dongle covering another chunk.
- Avoid maxing the rate "just in case" — it risks dropped samples and wastes CPU you
  could spend [decoding more calls](/learn/rf-sdr/antenna-to-audio/).

The [Hardware guide](/hardware.html) lists practical rates per radio.

<div class="knowledge-check" data-quiz data-correct-msg="Right — with IQ sampling, captured bandwidth ≈ sample rate." markdown="0">
  <p class="knowledge-check__q">Quick check: an RTL-SDR runs at 2.4 MSa/s. Roughly how much spectrum does it capture at once?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">About 1.2 kHz</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">About 2.4 MHz</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The entire VHF band</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Sample rate ≈ captured bandwidth** for IQ sampling.
- **Nyquist** is the rule behind it; under-sampling causes **aliasing** (false signals).
- Higher rate = more spectrum but **more USB/CPU load** and possible dropped samples.
- For trunk-tracking, pick a rate that **just covers your channels**, with margin.

Next: the setting people get wrong most — gain — and how to avoid overloading the ADC.
