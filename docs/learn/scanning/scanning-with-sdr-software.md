---
slug: scanning-with-sdr-software
title: Scanning with SDR software
description: The software side of SDR scanning — the tools that turn a raw software-defined radio into a trunk-tracking scanner, from waterfall viewers to full decoders, and where GopherTrunk fits among them.
keywords: SDR software, trunk tracking software, SDRTrunk, DSDplus, OP25, GopherTrunk, SDR scanner, software defined radio scanning, decoder software, wideband capture
level: intermediate
status: full
prereq:
  - scanners-vs-sdr
faq:
  - q: Why do you need software to scan with an SDR?
    a: "An SDR is just a receiver — it hands your computer a raw stream of radio samples and does nothing with it on its own. All the scanning intelligence lives in software: tuning, demodulating, decoding a control channel, following grants, recording, and logging. With a hardware scanner that logic is baked into the box; with an SDR you choose the software, which is both the freedom and the work."
  - q: What kinds of SDR scanning software are there?
    a: A rough ladder. General SDR viewers show you the spectrum and demodulate one signal. Digital-voice decoders add the ability to decode specific protocols. Full trunk-trackers read the control channel and follow calls across a whole system automatically. GopherTrunk sits at that top rung — a multi-protocol trunk-tracking decoder.
  - q: What's the advantage of software over a hardware scanner?
    a: Flexibility and reach. Software can be updated to handle new protocols, can decode several channels or systems at once from one wideband capture, exposes everything for logging and integration, and turns a cheap receiver into a capable scanner. The trade is that you assemble and maintain the setup yourself instead of switching on a finished appliance.
---

# Scanning with SDR software

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An [SDR](/learn/rf-sdr/what-is-sdr/) is just a receiver — it hands your computer raw
radio samples and nothing more. **All the scanning intelligence lives in software**:
tuning, demodulating, decoding the control channel, following calls, recording. The
tools form a ladder — from **spectrum viewers**, to **digital-voice decoders**, up to
full **trunk-trackers** that follow a whole system automatically. **GopherTrunk** sits
at that top rung: a multi-protocol trunk-tracking decoder. Software is the freedom of
SDR — and the work.
</div>

A hardware scanner is an appliance: the tuning, the trunk-tracking, the recording are
all baked into the box, and you switch it on. An SDR is the opposite — a bare
receiver that streams raw samples to your computer and does *nothing* with them by
itself. Everything that makes it a scanner is software you choose and run. That's the
trade at the heart of [scanners vs. SDR](/learn/scanning/scanners-vs-sdr/): more
power and flexibility, in exchange for assembling the setup yourself. This lesson maps
the software landscape and shows where GopherTrunk fits.

## The SDR is just the front door

It's worth being blunt about the division of labour. If you've read
[what an SDR is](/learn/rf-sdr/what-is-sdr/), you know the hardware's whole job is to
tune to a chunk of spectrum and digitise it — turn radio into a stream of numbers.
From there, **everything is software**:

- **Demodulation** — pulling the signal out of the samples.
- **Decoding** — turning the demodulated symbols into bits, then into voice or data.
- **Trunk-tracking** — reading the control channel and following grants to voice
  channels.
- **Recording, logging, alerting** — everything you built up over Unit 5.

With a hardware scanner all of that is invisible, inside the case. With an SDR it's
laid bare, which is exactly why SDR scanning is where the interesting work — and this
whole path — lives.

## A ladder of tools

SDR scanning software isn't one thing; it's a range, and it helps to see it as rungs
on a ladder from "look at the spectrum" to "follow the whole system":

1. **Spectrum viewers / general SDR apps.** These show you the
   [waterfall](/learn/scanning/identifying-unknown-signals/), let you tune around,
   and demodulate one signal at a time. Superb for exploring a band and
   [identifying signals](/learn/scanning/identifying-unknown-signals/), but they don't
   trunk-track — they're the eyes, not the brain.
2. **Digital-voice decoders.** Add the ability to decode specific digital protocols,
   so you hear a P25 or DMR conversation instead of a buzz. A step up, but often still
   channel-at-a-time rather than following a trunked system.
3. **Trunk-trackers.** The top rung: software that reads the
   [control channel](/learn/scanning/programming-a-trunked-system/), follows grants
   automatically, and decodes the resulting voice — the full behaviour a hardware
   trunking scanner gives you, in software.

Several well-known projects live on that top rung, and GopherTrunk is one of them.
The point of the ladder isn't to rank them but to know *what job you're asking the
software to do* before you pick a tool.

## What software buys you over a box

Why go through the assembly at all? Because software reaches places a sealed scanner
can't:

- **New protocols by update.** A scanner's decoding is fixed at manufacture; software
  gains new modes with a new release.
- **Many channels at once.** One wideband SDR capture can contain a whole system's
  worth of channels, and software can decode several in parallel — a single receiver
  covering what would take several scanners.
- **Everything is exposed.** Logs, metadata, recordings, and live status are all
  available to script, store, and integrate, which is what made Unit 5's
  [logging](/learn/scanning/logging-and-recording/) and
  [alerting](/learn/scanning/alerting-on-calls/) so natural.
- **Cheap hardware, capable scanner.** An inexpensive SDR plus good software rivals a
  far pricier appliance — and grows instead of ageing out.

The cost is real and worth naming: you build and maintain the setup, keep the
software current, and own the troubleshooting. For many people that's not a downside
at all — it's the fun.

## Where GopherTrunk fits

GopherTrunk is a **multi-protocol trunk-tracking decoder** on that top rung — it takes
the raw samples from an SDR, locks a system's control channel, follows every grant,
and decodes the voice, across P25, DMR, NXDN, TETRA, and more. It's built to be the
brain behind the receiver: the piece that turns a bare SDR into a scanner that follows
whole systems, records per call, and runs unattended. It's open source, so you can see
and shape exactly how it decodes, which is the theme this module's final unit is all
about.

The next lesson puts it to work on a real system, and the one after assembles it into
a complete, worked monitoring setup end to end.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an SDR only digitises the spectrum; demodulation, decoding, and trunk-tracking all happen in software." markdown="0">
  <p class="knowledge-check__q">Quick check: with an SDR, where does the trunk-tracking and decoding actually happen?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Inside the SDR hardware, like a scanner in a box</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">In software on your computer — the SDR only digitises the spectrum</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">On the trunked system's control channel, which does it for you</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **SDR is just a receiver** — it digitises the spectrum; all scanning
  intelligence lives in **software** on your computer.
- SDR scanning tools form a ladder: **spectrum viewers**, **digital-voice decoders**,
  and full **trunk-trackers** that follow a whole system.
- Software buys you **new protocols by update**, **many channels from one wideband
  capture**, **everything exposed** for logging and integration, and a capable
  scanner from cheap hardware.
- The cost is that you **assemble and maintain** the setup yourself.
- **GopherTrunk** is a multi-protocol **trunk-tracker** on the top rung — the brain
  that turns a bare SDR into a system-following scanner.

Next up: [GopherTrunk as a scanner](/learn/scanning/gophertrunk-as-a-scanner/).
