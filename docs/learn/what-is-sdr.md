---
slug: what-is-sdr
title: What is software-defined radio?
description: A beginner's guide to software-defined radio (SDR) — what it is, how it differs from a traditional radio, why one cheap dongle can decode dozens of systems, and how GopherTrunk turns IQ samples into decoded calls.
keywords: software defined radio, what is SDR, SDR explained, RTL-SDR, IQ samples, SDR vs traditional radio, SDR for beginners, how does SDR work
level: beginner
status: full
prereq:
  - radio-waves
faq:
  - q: What is software-defined radio in simple terms?
    a: Software-defined radio (SDR) is a radio where the parts that used to be fixed hardware — the tuning, filtering, and demodulation — are done in software instead. The hardware just converts a slice of the radio spectrum into a stream of digital samples, and a program decides what to do with them. That means one device can receive AM, FM, digital voice, pagers, aircraft transponders, and more, by changing software rather than swapping parts.
  - q: How is an SDR different from a normal radio?
    a: A traditional radio has dedicated circuits built for one job — an FM broadcast radio can only do FM broadcast. An SDR digitises the raw signal early and leaves the actual decoding to software, so the same hardware can be reconfigured for completely different modes. The flexibility comes from moving the radio's "brains" into code.
  - q: Why can one cheap SDR dongle decode so many different systems?
    a: Because the dongle's only job is to hand the computer a chunk of spectrum as IQ samples. Everything that distinguishes one system from another — the modulation, the protocol, the error correction — is handled in software like GopherTrunk. Add support for a new protocol in code and the same hardware can suddenly decode it, no new equipment required.
  - q: Do I need an expensive SDR to get started?
    a: No. An RTL-SDR dongle costing around $30 is enough to follow most of this learning path and to run GopherTrunk on many trunked systems. More expensive radios like HackRF or Airspy add wider bandwidth, better sensitivity, or transmit capability, but they aren't required to start learning.
gophertrunk_links:
  - title: What is GopherTrunk?
    url: /getting-started.html
    note: see how GopherTrunk fits the "software" half of software-defined radio.
  - title: Hardware guide
    url: /hardware.html
    note: the dongles GopherTrunk drives, from RTL-SDR to Airspy.
  - title: Get started
    url: /getting-started-setup.html
    note: go from a dongle to your first decoded call.
---

# What is software-defined radio?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Software-defined radio (SDR)** moves the "brains" of a radio out of fixed
circuits and into software. The hardware does just one thing: it converts a slice
of the radio spectrum into a stream of digital numbers — **IQ samples**. A program
then tunes, filters, and demodulates those samples to recover whatever's on the
air. That's why a single ~$30 dongle can decode FM, digital voice, pagers, and
aircraft transponders: change the software, not the hardware. **GopherTrunk is the
software half** — it turns IQ samples from your dongle into decoded trunked-radio
calls.
</div>

You now know what a radio wave is and how digital data rides on it. This lesson
introduces the device that catches those waves for a computer — and the single
idea that makes the whole hobby possible.

## What does "software-defined" actually mean?

In a **traditional radio**, every function is a dedicated piece of hardware. There's
a circuit that tunes to a frequency, a circuit that filters out everything else, and
a circuit that demodulates the result into sound. Those circuits are built for one
job — an FM broadcast receiver can *only* receive FM broadcast, forever.

A **software-defined radio** keeps the hardware to a bare minimum: just enough to
grab a chunk of spectrum and turn it into digital samples. Everything after that —
the tuning fine-tune, the filtering, the demodulation, the protocol decoding — is
done by a *program* running on a computer. Want to receive a different mode? You
don't rewire anything; you run different software.

That's the whole revolution in one sentence: **the radio becomes code.**

## How does an SDR turn radio waves into numbers?

At a high level, an SDR receiver does three things, which we'll unpack across
[Module 3](/learn/#sdr):

1. **Tune** — an oscillator shifts the band you care about down to a low frequency
   the rest of the chain can handle.
2. **Digitise** — an **analog-to-digital converter (ADC)** measures the signal
   millions of times per second, turning the smooth wave into a stream of numbers.
3. **Deliver IQ** — those numbers come out in pairs called **[IQ samples](/learn/iq-data/)**,
   which together capture both the amplitude *and* the phase of the signal — exactly
   the two things you need to reconstruct any modulation.

The output isn't audio. It's a firehose of IQ samples — a faithful digital copy of a
whole slice of the airwaves. What you *do* with that firehose is up to software.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 130" role="img" aria-label="A signal chain: antenna into SDR hardware which outputs IQ samples into software which outputs decoded audio and data." xmlns="http://www.w3.org/2000/svg">
  <g font-size="12" fill="currentColor" text-anchor="middle">
    <path d="M40 80 v-30 m-10 0 l10 -14 l10 14" fill="none" stroke="currentColor" stroke-width="2"/>
    <text x="40" y="100">antenna</text>
    <rect x="90" y="40" width="110" height="44" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="145" y="60">SDR hardware</text>
    <text x="145" y="76" font-size="10">tune · digitise</text>
    <rect x="270" y="40" width="120" height="44" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="330" y="60">software</text>
    <text x="330" y="76" font-size="10">(GopherTrunk)</text>
    <rect x="455" y="40" width="70" height="44" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="490" y="60">audio</text>
    <text x="490" y="76">&amp; data</text>
    <line x1="52" y1="62" x2="88" y2="62" stroke="currentColor" stroke-width="1.5" marker-end="url(#a)"/>
    <line x1="200" y1="62" x2="268" y2="62" stroke="currentColor" stroke-width="1.5" marker-end="url(#a)"/>
    <text x="234" y="34" font-size="10">IQ samples</text>
    <line x1="390" y1="62" x2="453" y2="62" stroke="currentColor" stroke-width="1.5" marker-end="url(#a)"/>
  </g>
  <defs><marker id="a" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The SDR pipeline: hardware grabs spectrum and emits IQ samples; software does the rest. The hardware boundary is where flexibility comes from.</figcaption>
</figure>

## Why can one cheap dongle decode dozens of systems?

Because the dongle doesn't *care* what it's receiving. Its only job is to deliver
IQ samples for a slice of spectrum. The thing that distinguishes a police P25 system
from a pager network from an aircraft transponder is the **modulation and protocol**
— and all of that lives in software.

So when GopherTrunk adds support for a new protocol, your *existing* hardware can
suddenly decode it. The same RTL-SDR that follows a trunked system can, with
different software, watch aircraft, plot ships, or read pager traffic. This is the
payoff of "the radio becomes code": capabilities arrive as updates, not as new
equipment.

It's also why this learning path spends its early modules on *fundamentals* rather
than any one device. The RF concepts — frequency, modulation, IQ, SNR — are what you
actually configure. The hardware is almost interchangeable.

## Where does GopherTrunk fit?

GopherTrunk is the **software half** of a software-defined radio, specialised for
**digital trunked-radio** systems. It takes the IQ samples from your dongle and runs
the full chain you'll learn in this path: it tunes to a control channel, filters and
demodulates the [4FSK signal](/learn/digital-modulation/), recovers the symbols, decodes
the trunking protocol to learn which voice channel a call jumped to, follows it,
decodes the [vocoded voice](/learn/vocoders/), and writes you a WAV file — all in real time,
across many channels at once.

You don't need to understand every stage to *run* it. But you'll operate it far
better once you do — which is exactly what the rest of this path builds toward, and
what the [From antenna to audio](/learn/antenna-to-audio/) lesson ties together at the end.

## What hardware do I need to start?

Not much. A basic **RTL-SDR** dongle (around $30) is enough to follow this path and
run GopherTrunk on many systems. Step-up radios add capability you may grow into:

| Class | Examples | What it adds |
|-------|----------|--------------|
| Entry | RTL-SDR (RTL2832U) | Cheap, ~2.4 MHz bandwidth, receive only — perfect to learn on |
| Wideband | Airspy R2 / Mini | More bandwidth and sensitivity |
| HF-capable | Airspy HF+ | Strong performance on lower bands |
| Transmit-capable | HackRF One | Wide range and TX (not needed for scanning) |

The [Hardware guide](/hardware.html) covers exactly which radios GopherTrunk drives
and how to pick one. The [SDR hardware lesson](/learn/sdr-hardware/) later in this module
goes deeper on the trade-offs.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the hardware just delivers IQ samples; software does the decoding." markdown="0">
  <p class="knowledge-check__q">Quick check: in an SDR, what decides whether you're decoding FM, P25, or pagers?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A switch on the dongle</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The software processing the IQ samples</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A different antenna for each mode</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **SDR** moves tuning, filtering, and demodulation from fixed hardware into
  software.
- The hardware's only job is to digitise a slice of spectrum into **IQ samples**.
- One dongle decodes many systems because the differences live in *code*, not parts.
- **GopherTrunk** is the software that turns those samples into decoded trunked calls.
- An ~$30 RTL-SDR is enough to begin.

Next, we'll open up the receiver and trace how those IQ samples are actually made.
