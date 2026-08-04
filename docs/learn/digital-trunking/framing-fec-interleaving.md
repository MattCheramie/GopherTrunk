---
slug: framing-fec-interleaving
title: "Framing, error correction & interleaving"
description: Raw bits are never sent bare. Sync patterns, forward error correction, interleaving, and CRC are what let digital voice survive fading that would only hiss in analog.
keywords: forward error correction, FEC, interleaving, framing, sync pattern, Golay code, Hamming code, BCH, trellis code, Viterbi, Reed-Solomon, CRC, burst errors, digital trunking
level: advanced
status: full
prereq:
  - digital-modulation-for-trunking
  - voice-to-bits-vocoders
faq:
  - q: What is forward error correction?
    a: Forward error correction adds carefully structured redundant bits to the data before transmission so the receiver can detect and fix a limited number of errors without asking for a retransmission. Digital radio relies on it because there is no time to request retries in a live voice stream. Codes like Hamming, Golay, BCH, trellis, and Reed-Solomon each fix errors in different ways.
  - q: What is interleaving and why is it used?
    a: Interleaving scrambles the order of bits before transmission and unscrambles them at the receiver. A fade or noise burst on the air corrupts a run of consecutive bits, but after de-interleaving those errors are spread thinly across many code words, where the error correction can fix them. Without interleaving a single burst could overwhelm the FEC in one spot.
  - q: What is the difference between FEC and a CRC?
    a: Forward error correction repairs bit errors so the data can be used. A cyclic redundancy check (CRC) only detects whether errors remain — it cannot fix them. Systems use FEC to recover the data and a CRC afterward to verify integrity, discarding or flagging a frame that still fails the check.
  - q: Why can digital survive fading that just makes analog hiss?
    a: In analog the noise rides straight into the audio. In digital, framing, error correction, and interleaving work together so the receiver can reconstruct the exact original bits even after some are corrupted. As long as the damage stays under the code's limit, the recovered voice is perfect, where analog would already be noisy.
---

# Framing, error correction & interleaving

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Raw bits are **never sent bare**. They're packed into **frames** and **bursts** marked
by **sync patterns**, protected by **forward error correction (FEC)** — Hamming, Golay,
BCH, trellis/Viterbi, Reed–Solomon — that lets the decoder *fix* bit errors, and
**interleaved** so a fade spreads its damage thinly enough for the FEC to recover.
A **CRC** then checks integrity. Together these are exactly what let digital voice
**survive fading** that would leave analog hissing. This structure is also why a
demodulated bitstream needs a whole [decode pipeline](/learn/rf-sdr/demodulation-pipeline/)
before it becomes audio.
</div>

You now have bits from the vocoder and a modulation to carry them. But if you simply
threw those bits on the air, the first fade would corrupt them and the vocoder would
produce garbage. Every digital system instead wraps its bits in protective structure.
Understanding that structure explains both how digital beats fading and why decoding
is more than just "read the symbols."

## Framing: structure and sync

Bits are organised into **frames** (and, in time-slotted systems,
[bursts](/learn/digital-trunking/tdma-vs-fdma/)) — fixed-length packages with defined fields for voice, for
signalling, and for housekeeping. The receiver has to know *where* each frame begins,
and it finds out from a **sync pattern**: a known, fixed sequence of symbols placed at
the frame boundary. The decoder slides along the incoming symbols looking for that
pattern; when it matches, frame alignment is locked and every field afterward sits at a
known offset.

Sync patterns do double duty as a fingerprint. Because each system uses its own sync
words, recognising the pattern is one of the first ways to **identify** what you're
hearing — a theme we return to in the system-identification lessons.

## Forward error correction: fixing, not just finding

A live voice stream can't pause to ask for a retransmission, so digital radio repairs
errors **in place** with **forward error correction**. FEC adds structured redundant
bits so the decoder can work out what the original bits *must* have been, even when
some arrived wrong. Different jobs call for different codes:

| Code | What it's good at |
|------|-------------------|
| Hamming | fixes a single bit error in a small block — cheap, common |
| Golay | corrects several errors in a short word — used on critical control fields |
| BCH | configurable multi-error correction on a block |
| Trellis / Viterbi | corrects a continuous bitstream by tracking the most likely sequence |
| Reed–Solomon | corrects whole corrupted *symbols*, strong against bursts |

Systems mix these. A P25 voice frame, for instance, protects different fields with
different codes — the most important bits (the ones that say *which channel a call went
to*) get the strongest protection, because losing them loses the call.

## Interleaving: defeating burst errors

There's a catch. FEC codes are usually strongest against errors that are **spread
out**, and weakest when many errors land **together** — which is exactly what a fade or
a noise burst produces. **Interleaving** solves this by shuffling the bit order before
transmission and reversing the shuffle at the receiver.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 170" role="img" aria-label="A diagram showing a burst of errors hitting consecutive transmitted bits, then de-interleaving spreading those errors thinly across several code words so error correction can fix each one." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="30" font-size="11" fill="currentColor">on the air — a fade hits consecutive bits:</text>
  <g>
    <rect x="20" y="40" width="180" height="22" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <rect x="92" y="40" width="44" height="22" fill="currentColor" fill-opacity="0.35"/>
    <text x="114" y="56" text-anchor="middle" font-size="10" fill="currentColor">burst</text>
  </g>
  <text x="20" y="100" font-size="11" fill="currentColor">after de-interleaving — errors spread across code words:</text>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="112" width="110" height="24" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <rect x="40" y="112" width="14" height="24" fill="currentColor" fill-opacity="0.35"/>
    <text x="75" y="128">word 1</text>
    <rect x="140" y="112" width="110" height="24" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <rect x="180" y="112" width="14" height="24" fill="currentColor" fill-opacity="0.35"/>
    <text x="195" y="128">word 2</text>
    <rect x="260" y="112" width="110" height="24" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <rect x="300" y="112" width="14" height="24" fill="currentColor" fill-opacity="0.35"/>
    <text x="315" y="128">word 3</text>
    <rect x="380" y="112" width="110" height="24" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <rect x="420" y="112" width="14" height="24" fill="currentColor" fill-opacity="0.35"/>
    <text x="435" y="128">word 4</text>
  </g>
  <line x1="114" y1="62" x2="114" y2="110" stroke="currentColor" stroke-width="1" stroke-dasharray="3 3"/>
</svg>
<figcaption>A fade corrupts a run of consecutive bits. Because the bits were interleaved, de-interleaving scatters that damage thinly across several code words — few enough errors per word that the FEC fixes each one.</figcaption>
</figure>

After de-interleaving, a single on-air burst that wiped out twenty consecutive bits
becomes one or two errors in each of many code words — well within what the FEC can
repair. Interleaving and FEC are a team: neither alone survives the bursty,
fading-prone mobile channel, but together they're formidable.

## CRC: the final integrity check

FEC *fixes* errors; a **CRC** (cyclic redundancy check) only *detects* whether any
remain. After error correction, the decoder recomputes the CRC over the recovered bits
and compares it to the transmitted check value. A match means the frame is trustworthy;
a mismatch means damage got through, and the system flags, repeats, or discards the
frame rather than acting on bad data — important when the frame is a channel-grant that
would send a receiver to the wrong frequency.

## Why this is the whole point

Stack it up and the picture is clear. Where analog lets noise leak straight into the
audio, digital wraps voice in **sync, FEC, interleaving, and CRC** so the receiver can
reconstruct the **exact original bits** even after the channel mangles them. As long as
the damage stays under the codes' limits, the recovered voice is *perfect* — and the
moment it exceeds them, you hit the [digital cliff](/learn/digital-trunking/analog-vs-digital-voice/) from two
lessons back. All of this is unwound inside the
[demodulation pipeline](/learn/rf-sdr/demodulation-pipeline/), downstream of the
[clock recovery](/learn/rf-sdr/clock-recovery/) that finds the symbols in the first
place.

<div class="knowledge-check" data-quiz data-correct-msg="Right — interleaving spreads a burst across many code words so FEC can fix each one." markdown="0">
  <p class="knowledge-check__q">Quick check: why do digital systems interleave bits before transmission?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To encrypt the data</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">To spread a burst of errors thinly so error correction can fix them</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To make the signal use less bandwidth</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Bits are packed into **frames/bursts** marked by **sync patterns** that the decoder
  locks onto (and that fingerprint the system).
- **FEC** — Hamming, Golay, BCH, trellis/Viterbi, Reed–Solomon — **fixes** bit errors
  in place, with the most important fields protected most strongly.
- **Interleaving** scatters a burst across many code words so the FEC can recover it.
- A **CRC** detects any errors that survived, so the system never acts on bad data.
- Together these let digital **survive fading** where analog would only hiss — until
  the cliff.

Next, we see how a single channel can carry more than one call at once:
[TDMA vs. FDMA](/learn/digital-trunking/tdma-vs-fdma/).
