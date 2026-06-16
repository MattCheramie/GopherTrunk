---
slug: demodulation-pipeline
title: The demodulation pipeline
description: The SDR demodulation pipeline stage by stage — digital tuning, channel filtering and decimation, demodulation of FM/FSK/PSK, symbol recovery, and framing and decoding — the chain that turns IQ samples into usable bits and audio.
keywords: demodulation pipeline, SDR signal chain, digital tuning, channel filter, demodulation, symbol recovery, framing, FEC decoding, DSP pipeline
level: advanced
status: full
prereq:
  - filtering-decimation
  - digital-modulation
faq:
  - q: What are the stages of an SDR demodulation pipeline?
    a: A typical pipeline has five stages — digital tuning to shift the wanted channel to centre, channel filtering and decimation to isolate it at a low sample rate, demodulation to recover the modulating information (FM, FSK, or PSK), symbol recovery to time and slice that into symbols, and framing and decoding to turn symbols into error-corrected bits and finally usable data or audio.
  - q: What is the difference between demodulation and decoding?
    a: Demodulation recovers the raw modulating signal from the carrier — for example turning frequency shifts back into a stream of levels. Decoding is the later step that interprets the resulting symbols and bits — finding frame boundaries, applying error correction, and extracting the actual message. Demodulation deals with the waveform; decoding deals with the data.
  - q: Why does each channel need its own pipeline?
    a: Each signal you follow needs to be individually tuned, filtered, demodulated, and decoded. Because the SDR captures a wide block of spectrum as shared IQ, software can run a separate pipeline per channel from that one stream — which is how a single radio can decode a control channel and several voice channels simultaneously.
  - q: Where in the pipeline do most decode failures happen?
    a: Usually at demodulation and symbol recovery, when the signal is too weak or smeared for the symbols to be told apart cleanly, or at decoding when error correction can no longer fix the resulting bit errors. The constellation and eye-diagram views show problems at the demod/symbol stage before the failure reaches the decoded output.
gophertrunk_links:
  - title: Architecture
    url: /architecture.html
    note: GopherTrunk's real pipeline, end to end.
  - title: Status
    url: /status.html
    note: the decoders this pipeline feeds, and their coverage.
---

# The demodulation pipeline

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Turning [IQ samples](/learn/iq-data/) into usable bits is a five-stage **pipeline**:
**(1) digital tuning** shifts your channel to centre; **(2) filter + decimate**
isolates it cheaply; **(3) demodulation** recovers the modulating signal (FM/FSK/PSK);
**(4) symbol recovery** times and slices it into [symbols](/learn/digital-modulation/);
**(5) framing + decoding** applies error correction and extracts the message.
*Demodulation* handles the waveform; *decoding* handles the data. One shared IQ stream
feeds **one pipeline per channel**, which is how GopherTrunk tracks many calls at once.
</div>

You've met the pieces — [IQ](/learn/iq-data/), [modulation](/learn/digital-modulation/),
[filtering](/learn/filtering-decimation/). This lesson assembles them into the chain that
every signal passes through inside an SDR decoder. It's the detailed version of the
[antenna-to-audio](/learn/antenna-to-audio/) story.

## The pipeline overview

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 90" role="img" aria-label="Five boxes left to right: digital tuning, filter and decimate, demodulate, symbol recovery, frame and decode, with IQ in on the left and bits out on the right." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="22" y="48">IQ →</text>
    <rect x="44" y="30" width="78" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="83" y="44">1 · digital</text><text x="83" y="55">tuning</text>
    <rect x="134" y="30" width="86" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="177" y="44">2 · filter +</text><text x="177" y="55">decimate</text>
    <rect x="232" y="30" width="80" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="272" y="50">3 · demod</text>
    <rect x="324" y="30" width="86" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="367" y="44">4 · symbol</text><text x="367" y="55">recovery</text>
    <rect x="422" y="30" width="82" height="34" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="463" y="44">5 · frame +</text><text x="463" y="55">decode</text>
    <text x="520" y="48">→ bits</text>
    <g stroke="currentColor" stroke-width="1.1">
      <line x1="122" y1="47" x2="133" y2="47"/><line x1="220" y1="47" x2="231" y2="47"/><line x1="312" y1="47" x2="323" y2="47"/><line x1="410" y1="47" x2="421" y2="47"/>
    </g>
  </g>
</svg>
<figcaption>The five stages every channel passes through. The first two are the front-end DSP; the last three turn a waveform into a message.</figcaption>
</figure>

## Stage 1 — Digital tuning

The captured [IQ](/learn/iq-data/) covers a whole band, but your channel sits somewhere
off-centre. **Digital tuning** multiplies the IQ by a rotating tone to **shift your
channel down to zero frequency** (baseband), so the next stage's simple low-pass filter
can isolate it. It's the software equivalent of fine-tuning the dial — and, unlike
hardware, you can do it for many channels at once from the same samples.

## Stage 2 — Channel filtering and decimation

With the channel centred, a **channel filter** keeps just its narrow bandwidth and
rejects neighbours and noise; **decimation** then drops the sample rate to suit that
narrow signal. This is the [filtering & decimation](/learn/filtering-decimation/) step,
and it's what makes the rest cheap enough to run in parallel.

## Stage 3 — Demodulation

Now the channel is isolated at a manageable rate, **demodulation** recovers the
*modulating signal* from the carrier, according to the
[modulation type](/learn/digital-modulation/):

- **FM/FSK** — track the instantaneous frequency; for 4FSK (P25/DMR) the output is a
  stream of four levels.
- **PSK** — track the phase angle of each [IQ](/learn/iq-data/) sample.

The output isn't symbols yet — it's a continuous, noisy stream that *contains* the
symbols. This is the stage the [constellation](/constellation.html) visualises.

## Stage 4 — Symbol recovery

The demodulated stream has to be sliced into discrete [symbols](/learn/digital-modulation/)
at exactly the right instants. That requires knowing the signal's rhythm —
**[clock recovery](/learn/clock-recovery/)** — so the decoder samples each symbol at its
centre, where the [eye diagram](/learn/digital-modulation/) is most open. Get the timing
right and the four levels resolve into clean symbols; get it wrong and symbols smear
into their neighbours.

## Stage 5 — Framing and decoding

Symbols become **bits**, but raw bits aren't the message. **Framing** finds where
packets begin (using known sync patterns), and **decoding** applies **forward error
correction (FEC)** to fix the inevitable bit errors, then extracts the payload — a
[control-channel message](/learn/what-is-trunking/), or [vocoder](/learn/vocoders/) frames
for voice. *Demodulation gave us a waveform; decoding gives us meaning.*

## Demodulation vs. decoding, and where failures live

It's worth nailing the distinction: **demodulation** works on the *waveform* (stages
3–4), **decoding** works on the *data* (stage 5). Most failures show up at the boundary
— a signal too weak or smeared for symbol recovery produces bit errors that FEC can't
fix. Because the [constellation and eye diagram](/learn/tuning-with-scopes/) watch stage
3–4, they warn you *before* the decoded output fails, which is what makes them such good
[tuning tools](/learn/tuning-with-scopes/).

## How GopherTrunk runs many pipelines at once

Since every stage is arithmetic on the shared [IQ stream](/learn/iq-data/), GopherTrunk
instantiates **one pipeline per channel** — the [control channel](/learn/what-is-trunking/)
plus each active [voice channel](/learn/antenna-to-audio/) — and runs them concurrently.
One radio, many simultaneous decodes. The [Architecture](/architecture.html) page shows
how this is wired in practice.

<div class="knowledge-check" data-quiz data-correct-msg="Right — demodulation recovers the waveform; decoding interprets the resulting bits." markdown="0">
  <p class="knowledge-check__q">Quick check: which stage applies forward error correction and extracts the message?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Digital tuning</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Demodulation</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Framing and decoding</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The pipeline: **tune → filter/decimate → demodulate → recover symbols → frame/decode**.
- **Demodulation** recovers the waveform; **decoding** (with FEC) recovers the message.
- Symbol recovery needs **clock recovery** to sample at the right instant.
- Failures cluster at demod/symbol-recovery — which the scopes reveal early.
- One IQ stream feeds **many parallel pipelines**.

Next: the timing trick at the heart of stage 4 — clock recovery.
