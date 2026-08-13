---
title: "From the Issue Tracker, Part 12: Seventy-Eight Degrees — The Phase Angle That Named the Bug"
description: "An Airspy R2 that refused to open because its opcode table was systematically shifted, then opened and decoded nothing at all. One diagnostic triplet — EVM 22.7%, phase imbalance +78.1°, image rejection 3.3 dB — read as a sentence, named the bug: a real-sampling stream misread as interleaved I/Q."
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

## Cheat sheet

| | |
|---|---|
| Issues | [#270](https://github.com/MattCheramie/GopherTrunk/issues/270) (R2 support), [#454](https://github.com/MattCheramie/GopherTrunk/issues/454) (open fails) |
| Symptom 1 | `set sample type ... device disconnected`, then `set sample rate ... protocol error` — the R2 never opens |
| Cause 1 | vendor-request opcode table systematically shifted against libairspy's `airspy_commands` enum, plus a wrong transfer direction — fixed via [PR #380](https://github.com/MattCheramie/GopherTrunk/pull/380) |
| Symptom 2 | device opens, tunes, streams — and decodes nothing; RTL-SDR locks on the same antenna |
| Diagnostic triplet | gain +0.001 dB, phase **+78.136°**, image rejection **3.3 dB** (EVM 22.7%) |
| Cause 2 | the R2 streams **real** ADC samples at 2× the IQ rate; the driver paired adjacent samples as I/Q — the host-side complex conversion was missing |
| Fix | DC blocker → Fs/4 translation → half-band Hilbert pair → decimate-by-2; phase −0.0007°, image rejection ~70.8 dB |

## In this post

- **Prologue: an opcode that didn't exist** — the Windows "device disconnected" era and a firmware update that fixed nothing.
- **Bug one: the opcode table that was almost right** — a systematically shifted enum and a backwards transfer direction.
- **Bug two: opens fine, decodes nothing** — the offline capture and the triplet that reads as a sentence.
- **The real cause: the conversion nobody remembers exists** — real-sampling receivers and the hidden host-side job.
- **The fix: a Hilbert converter on the host** — four stages, why each is there, and the numbers after.
- **What we keep** — signatures, real-vs-complex, offline captures, and shifted-table bug class.

## Prologue: an opcode that didn't exist

The R2's troubles predate [#454](https://github.com/MattCheramie/GopherTrunk/issues/454).
When preliminary Airspy support first landed
([#270](https://github.com/MattCheramie/GopherTrunk/issues/270)), a contributor with
an R2 on the bench found it detected but unusable on Windows:

```text
level=ERROR msg="open device failed" driver=airspy index=0
  err="airspy: set sample type: winusb: WinUsb_ControlTransfer OUT: usb: device disconnected"
```

That "set sample type" command is a preview of everything that follows: the driver
was sending opcode 11 for a `SET_SAMPLE_TYPE` operation that **does not exist** in
the Airspy firmware's command set at all. The device's response to the nonsense
request surfaced as a disconnect. An interim change ([#358](https://github.com/MattCheramie/GopherTrunk/pull/358))
deferred the call out of the open path — symptom relief, not diagnosis.

Another user then did everything right from the hardware side: updated the R2 to
firmware NOS v1.0.0-rc10, confirmed with `airspy_info` that it advertises its two
native rates (10 and 2.5 MSPS), and retested — and still hit
`set sample rate failed ... err="protocol error"`, with the R2 dropping out of the
pool and the daemon falling back to its RTL-SDRs. (They also tried the R2's native
10 MS/s, which config validation rejected — `sdr.sample_rate must be between
225 kHz and 3.2 MHz` — a cap tracked separately in
[#550](https://github.com/MattCheramie/GopherTrunk/issues/550).) The same
contributor said plainly what the accumulating evidence showed — "there are a
number of incorrect primitives in the existing Airspy driver" — and put up
[PR #380](https://github.com/MattCheramie/GopherTrunk/pull/380) with the corrected
wire protocol, smoke-tested on a real R2. Root-causing #454 confirmed it
wholesale.

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
| `SET_SAMPLE_TYPE` | *(doesn't exist)* | 11 | nonsense — the prologue's "disconnect" |

Two of those rows combine to produce the exact reported symptom. `GET_SAMPLERATES`
went out as `SET_FREQ`, so the supported-rate table never loaded. Then
`SET_SAMPLERATE` went out as opcode 14 (`SET_LNA_GAIN`) — and direction compounded
it: the real `SET_SAMPLERATE` is a vendor-**IN** request carrying the rate in
`wIndex` and reading one status byte back, and sending it as a data-less
vendor-OUT gets NAK'd by the firmware, which surfaces as EPROTO: `protocol error`.
A systematically shifted table is worth recognizing as a *class*: every entry is
plausible in isolation, some commands even appear to work, and the device does
something — just never what you asked.

The correction was correspondingly wholesale, not a one-line patch: opcodes
renumbered to match libairspy's enum, `SetSampleRate` switched to vendor-IN with
the rate in `wIndex`, the bogus `SET_SAMPLE_TYPE` removed, bias-tee reimplemented
as the GPIO write it actually is, a receiver reset plus transient-error retry
added to the open path, and serial matching taught the `AIRSPY SN:` prefix the
firmware reports (so a configured bare serial still matches). Opt-in on-hardware
tests (`make test-airspy-real`) now exercise the corrected wire protocol against
a real device.

## Bug two: opens fine, decodes nothing

With the table fixed, the R2 opened, tuned, and streamed. And decoded nothing — on
a control channel (Mt Anakie, 420.0875 MHz) that locked immediately with an
RTL-SDR on the same antenna and feed. The reporter did exactly the right thing:
took the live pipeline out of the equation with a 10-second offline capture and
replayed it with diagnostics on — two commands that turned "it doesn't work" into
a portable, replayable fixture:

```bash
gophertrunk capture -serial "AIRSPY SN:..." -freq 420087500 \
  -sample-rate 2500000 -seconds 10 -out airspy-420087500.cfile

gophertrunk replay -in airspy-420087500.cfile -format f32 \
  -sample-rate 2500000 -protocol p25p1 -demod c4fm -diag
```

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

Each stage earns its place. A real stream at twice the IQ rate carries its unique
spectrum between 0 Hz and the IQ rate, with the tuner parking the wanted band at
a quarter of the ADC rate — so the Fs/4 translation is what slides that band down
to 0 Hz, and it's also what makes the next stage cheap: once the band is
centered, rejecting the negative-frequency image is exactly the job a Hilbert
pair does, and a *half-band* FIR design does it with nearly half its coefficients
identically zero. The Q rail's matched pure delay keeps the two rails
time-aligned with the FIR's group delay — skip that and you have rebuilt a
miniature of the original bug, two rails that don't correspond to the same
instant. And the final decimate-by-two is information-preserving, not lossy:
2×-rate real samples and 1×-rate complex samples carry the same bandwidth, which
is why the output lands exactly on the configured IQ rate.

The converter is stateful across USB packets — the FIR history and translation
phase carry over each buffer boundary, so packet edges introduce no
discontinuities — and it was written from DSP first principles rather than
vendored, because libairspy's own converter carries an Airspy-ecosystem-only
license. On a synthetic real tone the results bracket the story neatly:

| Metric | Before | After |
|---|---|---|
| Phase imbalance | +78.136° | −0.0007° |
| [Image rejection]({{ '/reference/image-rejection/' | relative_url }}) | 3.3 dB | ~70.8 dB |

Unit tests gate phase below 1° and image rejection above 40 dB so the converter
can't quietly regress. The reporter rebuilt and confirmed both fixes on
hardware: the R2 now opens reliably, runs with `dc_avoid` active
(`lo_offset_hz=625000`), and locks the control channel — matching the RTL-SDR on
the same feed. That confirmation closed [#454](https://github.com/MattCheramie/GopherTrunk/issues/454),
and with it the original support issue [#270](https://github.com/MattCheramie/GopherTrunk/issues/270),
whose two adjacent loose ends are tracked on their own: the config cap that
blocks the R2's native 10 MS/s
([#550](https://github.com/MattCheramie/GopherTrunk/issues/550)), and the
10-vs-2.5 MS/s in-channel SNR deficit on this same R2 — which became its own
investigation, told in
[Part 5]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }}).

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

## FAQ

**How can a wrongly shifted opcode table half-work?**
Because most wrong opcodes still hit *some* valid command. The device detected,
enumerated, and even reported a gain list — while "set sample rate" was actually
setting LNA gain and "get sample rates" was actually setting frequency. The
device always did something; it just never did what was asked. That's what makes
the class dangerous: partial success reads as "nearly working" instead of "not
talking the protocol at all."

**What does "real-sampling receiver" mean, and how is it different from an RTL-SDR?**
An RTL-SDR delivers complex (I/Q) baseband: two channels in quadrature, produced
by its own hardware chain. The Airspy R2/Mini instead stream the bare output of
one ADC — real samples, 12 bits in a little-endian `uint16`, DC offset at 2048 —
at twice the configured IQ rate, and expect the *host* to construct the complex
signal. libairspy does that construction internally, which is exactly why it's
easy to port the USB protocol and forget the conversion exists.

**Why 78° and not exactly 90°?**
The measured "phase imbalance" of the mis-paired stream is `arcsin` of the
correlation between adjacent samples. Neighboring samples of an oversampled real
signal are highly correlated but not perfectly so — the residual decorrelation
comes from the signal's actual bandwidth — so the angle lands near, but not at,
90°. Anything in that neighborhood, combined with ~0 dB gain error and no image
rejection, means the two "channels" are one real signal read twice.

**Why write the converter from scratch instead of using libairspy's?**
License. libairspy's converter carries an Airspy-ecosystem-only license, and
GopherTrunk only ports permissively-licensed reference code. The chain — leaky DC
blocker, Fs/4 translation, half-band Hilbert pair, decimate-by-two — is standard
DSP, implementable from first principles and verifiable by measurement (phase
−0.0007°, image rejection ~70.8 dB on a synthetic tone).

**Does the Airspy Mini need the same treatment?**
Yes. The R2 and Mini share the real-sampling architecture, so both stream real
ADC samples at twice the IQ rate and both go through the same host-side
converter. How the 2× real rate interacts with each model's usable IQ rates is
covered in [Airspy rate selection]({{ '/reference/airspy-rate-selection/' | relative_url }}).

## Series navigation

**Part 12 of 22** · ←
[Part 11: Detected but Not Present — One Hex Code from a Fix That Already Existed]({{ '/blog/solution-postmortem/from-the-issue-tracker-11-detected-but-not-present/' | relative_url }})
· Next →
[Part 13: The SoapyRemote Handshake — Three Wrong Root Causes and a Server That Says Nothing First]({{ '/blog/solution-postmortem/from-the-issue-tracker-13-soapyremote-handshake/' | relative_url }})
