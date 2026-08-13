---
title: "From the Issue Tracker, Part 12: Seventy-Eight Degrees — The Phase Angle That Named the Bug"
description: An Airspy R2 that refused to open because its opcode table was systematically shifted, then opened and decoded nothing at all. One diagnostic triplet — EVM 22.7%, phase imbalance +78.1°, image rejection 3.3 dB — read as a sentence, named the bug: a real-sampling stream misread as interleaved I/Q.
category: solution-postmortem
keywords: airspy r2, opcode table, protocol error, real sampling, hilbert transform, iq imbalance, image rejection, evm, fs/4 translation, host-side conversion
tags: [from-the-issue-tracker, airspy, usb, drivers, dsp, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 12
---

*Part 12 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 11]({{ '/blog/solution-postmortem/from-the-issue-tracker-11-detected-but-not-present/' | relative_url }})
ended with a probe line that named its bug in one hex code. This part's bug needed
three numbers — and once you learn to read them together, they form a signature you
will recognize on sight for the rest of your SDR career.*

> **TL;DR:** The Airspy R2 failed to open with `set sample rate ... protocol error`
> ([#454](https://github.com/MattCheramie/GopherTrunk/issues/454)): the driver's
> vendor-request opcode table was systematically shifted against libairspy's enum,
> so "set sample rate" went out on the wire as "set LNA gain" — with the wrong
> transfer direction on top. Fixing the table opened the device… which then decoded
> nothing. The reporter's offline capture measured EVM 22.7%, phase imbalance
> +78.1°, and image rejection 3.3 dB — the textbook signature of a **real**-sampling
> stream misread as interleaved I/Q. The Airspy streams bare ADC samples at twice
> the IQ rate; complex conversion is the *host's* job, and libairspy does it so
> internally you can forget it exists. The fix is a host-side converter — DC
> blocker, Fs/4 translation, half-band Hilbert pair, decimate-by-two — that took the
> measured phase error from +78.1° to −0.0007°.

## Bug one: the opcode table that was almost right

The R2's first failure mode was refusing to open at all, with `protocol error`
(EPROTO) on the sample-rate command. The cause was a driver opcode table that had
drifted systematically against libairspy's `airspy_commands` enum — not one wrong
entry but a whole shifted block, the fingerprint of an off-by-a-few transcription
from the reference source:

| Command | Should be | Was sent as | Which the firmware reads as |
|---|---|---|---|
| `SET_SAMPLERATE` | 12 | 14 | `SET_LNA_GAIN` |
| `SET_FREQ` | 13 | 12 | (shifted down) |
| gain / AGC family | 14–18 | 19–23 | wrong or nonexistent ops |
| `SET_RF_BIAS` | 20 | 24 | nonexistent |
| `GET_SAMPLERATES` | 25 | 13 | `SET_FREQ` |

Direction compounded it: `SET_SAMPLERATE` is a vendor-**IN** request carrying the
rate in `wIndex` and reading one status byte back. Sending it as a data-less
vendor-OUT gets NAK'd by the firmware, which surfaces as EPROTO. (The table even
carried a `SET_SAMPLE_TYPE` opcode that does not exist in the real command set.) A
systematically shifted table is worth recognizing as a *class*: every entry is
plausible in isolation, some commands even appear to work, and the device does
something — just never what you asked.

## Bug two: opens fine, decodes nothing

With the table fixed, the R2 opened, tuned, and streamed. And decoded nothing — on
a control channel that locked immediately with an RTL-SDR on the same antenna and
feed. The reporter did exactly the right thing: took the live pipeline out of the
equation with a 10-second offline capture and replayed it with diagnostics on.

```text
did NOT lock the control channel
FSW hits = 0

demod (c4fm):     EVM = 22.7%   SNR ≈ 7.4 dB
raw IQ imbalance: gain = +0.001 dB   phase = +78.136°   image_rejection = 3.3 dB
```

Read as a sentence, that triplet is unambiguous. Gain imbalance is essentially
zero — both "channels" have identical statistics. Phase imbalance is not the
half-degree of an imperfect quadrature mixer but *seventy-eight degrees* — the two
channels are nearly the same signal. And 3.3 dB of image rejection means the image
is essentially not suppressed at all. Hardware doesn't fail like this. A
quadrature front end with a real analog defect gives you a few degrees and maybe
20 dB. Two channels that are almost copies of each other with no image suppression
means the "I" and "Q" were never in quadrature to begin with — they are **adjacent
samples of one real signal**.

## The real cause: the conversion nobody remembers exists

The Airspy R2 and Mini are real-sampling receivers. The firmware streams bare ADC
samples — unpacked little-endian `uint16`, 12 bits used, DC centered at 2048 — at
**twice** the configured IQ rate. Turning that into complex baseband is a host-side
job, and libairspy performs it inside its own sample callbacks, so every
application built on it receives complex samples without ever seeing the real
stream. Port the USB protocol without porting the converter and you inherit exactly
this bug.

GopherTrunk's driver was pairing adjacent real samples as (I, Q). Two neighboring
samples of an oversampled signal are highly correlated, and the measured phase
imbalance of such a pairing lands at `arcsin(correlation)` — near 90° for strong
correlation. The +78.1° wasn't noise; it was the correlation coefficient of the
input wearing a trench coat. That's why the number *names* the bug: no other defect
produces near-zero gain error, a phase error pushing 90°, and no image rejection
simultaneously. The triplet now has an entry of its own in
[signal signatures]({{ '/reference/signal-signatures/' | relative_url }}).

## The fix: a Hilbert converter on the host

The driver now runs the real stream through a proper real-to-complex converter:

1. **Leaky DC blocker** — removes the ADC's 2048 offset without a hard notch.
2. **Fs/4 translation** — shifts the band of interest to center the spectrum.
3. **Half-band [Hilbert]({{ '/reference/hilbert-transform/' | relative_url }}) pair** —
   a symmetric FIR on one rail and a matched pure delay on the other, producing
   genuine quadrature.
4. **Decimate by two** — down to the configured IQ rate.

The converter is stateful across USB packets, so buffer boundaries introduce no
discontinuities, and it was written from DSP first principles rather than vendored
(libairspy's converter carries an Airspy-ecosystem-only license). On a synthetic
real tone the results bracket the story neatly:

| Metric | Before | After |
|---|---|---|
| Phase imbalance | +78.136° | −0.0007° |
| [Image rejection]({{ '/reference/image-rejection/' | relative_url }}) | 3.3 dB | ~70.8 dB |

Unit tests gate phase below 1° and image rejection above 40 dB so the converter
can't quietly regress. The reporter rebuilt, the R2 opened and locked, and the
issue closed on their confirmation.

## What we keep

- **EVM tells you it's broken; the imbalance triplet tells you what broke.** 22.7%
  EVM says "unusable." Zero gain error + ~78° phase + ~3 dB image rejection says
  "real stream read as I/Q" — a one-look diagnosis catalogued in
  [signal signatures]({{ '/reference/signal-signatures/' | relative_url }}).
- **Know whether your device samples real or complex before reading one byte of
  its stream.** It's the single most consequential fact in a driver bring-up, and
  the reference library may be hiding the conversion from you. Rates double, DC
  moves, and pairing rules change; see
  [Airspy rate selection]({{ '/reference/airspy-rate-selection/' | relative_url }})
  for how the 2× real rate interacts with the usable IQ rates.
- **An offline capture converts an argument into a measurement.** "Decodes on RTL,
  not on Airspy, same antenna" became a pinned, replayable fixture with numbers
  attached — the pipeline-vs-source question answered in one replay.
- **Systematically shifted tables are a bug class.** When one opcode is wrong,
  check the whole table against the reference enum — and check each request's
  *direction*, because a right opcode with the wrong direction still NAKs.

*Next: [Part 13]({{ '/blog/solution-postmortem/from-the-issue-tracker-13-soapyremote-handshake/' | relative_url }})
closes the driver cluster with a network SDR: a SoapyRemote server that segfaulted
on connect, three confident root causes that were all wrong, and a handshake with
one more phase than anyone implemented.*
