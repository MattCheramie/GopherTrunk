---
slug: antenna-to-audio
redirect_from: /learn/antenna-to-audio/
title: From antenna to audio — the GopherTrunk signal path
description: Follow one trunked-radio call end to end through GopherTrunk — antenna, SDR, IQ samples, demodulation, symbol recovery, trunking logic, vocoder, and a WAV on disk. The capstone that ties the whole RF and SDR learning path together.
keywords: SDR signal path, how GopherTrunk works, IQ to audio, demodulation pipeline, trunked radio decoding, vocoder, control channel following, end to end SDR
level: intermediate
status: full
prereq:
  - what-is-sdr
  - digital-modulation
  - what-is-trunking
faq:
  - q: How does GopherTrunk turn a radio signal into an audio recording?
    a: "GopherTrunk runs a pipeline: the antenna feeds an SDR that digitises spectrum into IQ samples; software filters and tunes to the control channel, demodulates its 4FSK into symbols, and decodes the trunking protocol; when a call starts, it tunes a receiver to the assigned voice channel, demodulates and decodes that, runs the vocoder frames through a speech decoder to produce audio, and writes the result to a WAV file with metadata."
  - q: What is the role of the vocoder in the signal path?
    a: Digital voice systems don't send raw audio — they compress speech into tiny data frames using a vocoder such as IMBE (P25) or AMBE+2 (DMR). At the end of the receive chain, GopherTrunk feeds those decoded frames into a matching vocoder to reconstruct an audible waveform you can listen to and save.
  - q: Can GopherTrunk follow several calls at once?
    a: Yes. By continuously decoding the control channel, GopherTrunk knows every voice-channel assignment as it happens and can task receivers to capture multiple simultaneous calls across the system, then mix or queue them for playback and recording according to your priorities.
  - q: Where can a call be lost in the signal path?
    a: A call can fail at any stage — too little signal above the noise floor, gain set so high the ADC clips, a tuning or clock error smearing the constellation, the wrong protocol or parameters, or encryption you can't decode. Most troubleshooting is about finding which stage in the chain broke.
gophertrunk_links:
  - title: Architecture
    url: /architecture.html
    note: the engineering view of the same pipeline described here.
  - title: Web console
    url: /web.html
    note: where you watch each stage of the path happen live.
  - title: Tuning for a clean lock
    url: /learn/rf-sdr/tuning-with-scopes/
    note: diagnose which stage broke when a call doesn't decode.
---

# From antenna to audio

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
This is the capstone: one trunked call traced end to end through GopherTrunk.
The **antenna** feeds an **SDR** that digitises spectrum into **IQ samples**.
Software **tunes and filters** to the **control channel**, **demodulates** its
4FSK into **symbols**, and **decodes** the trunking protocol. When a call starts,
GopherTrunk tunes a receiver to the assigned **voice channel**, recovers its
symbols, feeds the **vocoder** frames to a speech decoder, and writes an **audio
file**. Every concept from the earlier lessons is one link in this chain — and
troubleshooting is just finding which link broke.
</div>

You've met every piece on its own. Now let's assemble them. We'll follow a single
police dispatch call from radio waves in the air to a `.wav` on your disk, naming
the lesson behind each step so you can see how the whole path fits together.

## Stage 1 — Antenna: catching the wave

It starts with a **[radio wave](/learn/rf-sdr/radio-waves/)** arriving at your antenna, which
converts the wave's electric field back into a tiny electrical signal — typically a
faint **[−80 to −110 dBm](/learn/rf-sdr/decibels/)**. Antenna choice and placement set the ceiling
for everything downstream: if the signal doesn't arrive with enough strength above
the **noise floor**, no amount of clever software can recover it. This is why "where
you put the antenna matters more than the radio."

## Stage 2 — SDR hardware: spectrum becomes IQ

The signal travels down the coax into your **[SDR](/learn/rf-sdr/what-is-sdr/)**. There it's
amplified, shifted down in frequency, and handed to an **analog-to-digital
converter** that samples it millions of times a second. The output is a stream of
**[IQ samples](/learn/rf-sdr/iq-data/)** — pairs of numbers capturing the amplitude *and* phase of
a whole slice of spectrum at once.

Two settings rule this stage. **[Sample rate](/learn/rf-sdr/sample-rate-nyquist/)** sets how much
spectrum you capture in one go (and how hard the CPU works).
**[Gain](/learn/rf-sdr/gain-and-agc/)** sets how strong the signal is when it hits the ADC — too
low and weak signals vanish into the noise; too high and the ADC **clips** at
0 dBFS, spraying distortion everywhere. Getting these right is most of "setting up"
an SDR.

## Stage 3 — Tune and filter: zoom in on the control channel

Now we're in software. The IQ firehose covers far more spectrum than one channel, so
GopherTrunk **digitally tunes** to the **[control channel](/learn/rf-sdr/what-is-trunking/)** and
**[filters](/learn/rf-sdr/filtering-decimation/)** away everything else, keeping just that narrow
slice. This is the SDR equivalent of turning the dial — except it happens in code and
can be done for several channels in parallel from the same sample stream.

## Stage 4 — Demodulate: recover the symbols

The isolated control-channel signal is still a modulated waveform. GopherTrunk
**[demodulates](/learn/rf-sdr/demodulation-pipeline/)** it — for P25 and DMR, reading the
**[4FSK](/learn/rf-sdr/digital-modulation/)** frequency shifts — and runs **[clock
recovery](/learn/rf-sdr/clock-recovery/)** to sample each **symbol** at exactly the right instant.
This is the stage you can *watch*: the **[constellation, eye diagram, and symbol
scope](/learn/rf-sdr/tuning-with-scopes/)** all visualise these recovered symbols, so a smeared
constellation here tells you the demodulator is struggling before any data is lost.

## Stage 5 — Decode the trunking protocol: read the map

The symbols become bits, and the bits become **control-channel messages**. Decoding
the [trunking protocol](/learn/rf-sdr/protocol-landscape/) is where GopherTrunk reads the system's
running commentary: registrations, requests, and — crucially — **channel
assignments**. The moment it sees *"talkgroup 101 → voice channel 3,"* it knows a
call is starting and exactly where. You can watch this raw stream in the
**[CC Activity](/cc-activity.html)** panel.

## Stage 6 — Follow the call: capture the voice channel

GopherTrunk now tasks a receiver to tune to **voice channel 3** and runs stages 3–5
again on *that* channel — filter, demodulate, recover symbols, decode. Because it's
still reading the control channel in parallel, it can follow this call *and* catch
the next assignment elsewhere, capturing **multiple simultaneous calls** across the
system. It's also reading the transmitting **[radio ID](/radio-ids.html)**, so the
recording is labelled with *who* spoke, not just which talkgroup.

## Stage 7 — Vocoder: frames become sound

The voice channel doesn't carry audio — it carries **[vocoder](/learn/rf-sdr/vocoders/)** frames,
speech compressed to a few kilobits per second by a codec like **IMBE** (P25) or
**AMBE+2** (DMR). GopherTrunk feeds those frames into a matching speech decoder,
which **reconstructs an audible waveform**. This is the step that turns abstract bits
back into a human voice — the [analog-vs-digital-voice](/learn/rf-sdr/digital-voice/) trade-off in
action.

## Stage 8 — Audio out: playback and a WAV on disk

Finally, the reconstructed audio is played live in the console and written to a
**`.wav` file**, tagged with the talkgroup, radio ID, timestamp, and system. From
here GopherTrunk can also stream it onward to services like Broadcastify or
RdioScanner. The call that began as a faint ripple at your antenna is now a labelled
recording you can replay — and the engine is already following the next one.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 110" role="img" aria-label="The full pipeline as a row of boxes: antenna, SDR, tune and filter, demodulate, decode protocol, follow voice, vocoder, audio file." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9.5" fill="currentColor" text-anchor="middle">
    <g>
      <rect x="8"   y="40" width="56" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="36" y="60">antenna</text>
      <rect x="74"  y="40" width="56" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="102" y="60">SDR</text>
      <rect x="140" y="40" width="56" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="168" y="57">tune/</text><text x="168" y="68">filter</text>
      <rect x="206" y="40" width="56" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="234" y="60">demod</text>
      <rect x="272" y="40" width="56" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="300" y="57">decode</text><text x="300" y="68">protocol</text>
      <rect x="338" y="40" width="56" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="366" y="57">follow</text><text x="366" y="68">voice</text>
      <rect x="404" y="40" width="56" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="432" y="60">vocoder</text>
      <rect x="470" y="40" width="62" height="34" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="501" y="57">audio</text><text x="501" y="68">.wav</text>
    </g>
    <g stroke="currentColor" stroke-width="1.2">
      <line x1="64" y1="57" x2="73" y2="57"/><line x1="130" y1="57" x2="139" y2="57"/><line x1="196" y1="57" x2="205" y2="57"/>
      <line x1="262" y1="57" x2="271" y2="57"/><line x1="328" y1="57" x2="337" y2="57"/><line x1="394" y1="57" x2="403" y2="57"/><line x1="460" y1="57" x2="469" y2="57"/>
    </g>
    <text x="102" y="92" font-size="9">IQ samples →</text>
    <text x="300" y="22" font-size="9">control channel reads the map ↑</text>
  </g>
</svg>
<figcaption>The complete GopherTrunk path. Stages 3–6 run continuously on the control channel and again on each voice channel a call is assigned to.</figcaption>
</figure>

## Why this map is your best troubleshooting tool

When a call *doesn't* decode, the chain tells you where to look. Walk it in order:

| Symptom | Likely stage | Where to look |
|---------|--------------|---------------|
| Nothing at all, flat spectrum | Antenna / gain | signal below noise floor, or too little gain |
| Spectrum looks "smeared" or distorted | SDR / gain | ADC clipping — reduce gain |
| Control channel won't lock | Demodulate | smeared constellation, [clock/timing](/learn/rf-sdr/clock-recovery/) |
| Locks but no calls | Decode protocol | wrong system parameters or protocol |
| Calls listed but silent | Vocoder / encryption | [encrypted](/learn/rf-sdr/encryption/) talkgroup, or wrong vocoder |

That's the real payoff of this whole learning path: you're no longer poking settings
at random. You have a mental model of every stage, so you can reason about *which*
one broke — exactly what the [calibration and
troubleshooting](/learn/rf-sdr/calibration-troubleshooting/) lesson builds on.

<div class="knowledge-check" data-quiz data-correct-msg="Right — vocoder frames are decoded into an audible waveform at the end of the chain." markdown="0">
  <p class="knowledge-check__q">Quick check: a system is locked and calls are listed, but they play back silent. Which stage is the prime suspect?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The antenna</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The digital tuner</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The vocoder — or the talkgroup is encrypted</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A call flows: **antenna → SDR → IQ → tune/filter → demodulate → decode protocol →
  follow voice → vocoder → audio**.
- The **control channel** is decoded continuously so the engine knows where every
  call goes and can follow several at once.
- Each stage maps to an earlier lesson — this is where the whole path comes together.
- The pipeline doubles as a **troubleshooting checklist**: find the stage that broke.

You've now seen the entire chain. The remaining lessons in this module make you
fluent at *operating* it — finding systems, tuning a clean lock, and calibrating —
and the [glossary](/learn/rf-sdr/glossary/) is there whenever a term needs a refresher.
