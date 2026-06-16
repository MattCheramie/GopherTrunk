---
slug: calibration-troubleshooting
title: Calibration & troubleshooting
description: Calibrate and troubleshoot your SDR setup — PPM frequency-error correction, voice calibration for clean audio, and a stage-by-stage checklist for the usual reasons a control channel won't decode.
keywords: PPM correction, SDR calibration, frequency offset, voice calibration, troubleshoot SDR, control channel won't decode, no audio, SDR not working, scanner troubleshooting
level: advanced
status: full
prereq:
  - antenna-to-audio
  - gain-and-agc
faq:
  - q: What is PPM correction on an SDR?
    a: PPM (parts per million) correction compensates for the small frequency error in an SDR's reference oscillator. Cheap dongles can be off by tens of parts per million, which at UHF means the signal lands a few kHz away from where you tuned. Setting the right PPM value shifts the radio so signals appear at their true frequency, which is essential for a clean digital lock.
  - q: How do I find the right PPM value?
    a: Tune to a known, stable reference signal and adjust PPM until it sits exactly on its correct frequency, or use a tool that measures the offset against a known signal. Once set, the value is roughly constant for that dongle (it can drift slightly with temperature, so let the dongle warm up first).
  - q: My control channel won't decode — what should I check?
    a: Work the signal path in order — antenna and placement, gain (too low or clipping), frequency/PPM offset, then system parameters. Confirm there's actually signal above the noise floor, that gain isn't clipping the ADC, that PPM is correct so the signal is on frequency, and that you have the right control-channel frequency and system type. The scopes pinpoint where it breaks.
  - q: Calls are listed but there's no audio — is that a hardware fault?
    a: Usually not. Listed calls with no audio most often means the talkgroup is encrypted, or the wrong vocoder/voice settings, rather than a hardware problem. Check whether other talkgroups produce audio; if they do, the system and hardware are fine and the silent one is likely encrypted.
gophertrunk_links:
  - title: Voice calibration
    url: /voice-calibration.html
    note: tune decoded audio level and clarity.
  - title: Tuning (receiver meters)
    url: /tuning.html
    note: the meters you watch while calibrating.
  - title: Hardware guide
    url: /hardware.html
    note: rule out a hardware fault when config and signal check out.
---

# Calibration & troubleshooting

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Two calibrations matter most: **PPM correction** (compensating the dongle's small
frequency error so signals land on their true frequency) and **[voice
calibration](/voice-calibration.html)** (clean, correctly-levelled audio). When a control
channel won't decode, **walk the [signal path](/learn/antenna-to-audio/) in order** —
antenna/placement, gain (too low or clipping), PPM/frequency, then system parameters —
using the [scopes](/learn/tuning-with-scopes/) to pinpoint the broken stage. And remember:
**calls listed but silent is usually [encryption](/learn/encryption/), not a fault.**
</div>

This is the practical payoff of the whole path. Because you understand every
[stage](/learn/antenna-to-audio/), you can troubleshoot by reasoning instead of guessing.
We start with the one calibration newcomers most often miss.

## PPM — correcting frequency error

Every SDR has a reference oscillator, and cheap ones aren't perfectly accurate. The
error is measured in **PPM (parts per million)**. It sounds tiny, but at UHF a 30 PPM
error at 460 MHz is about **14 kHz** — more than a channel's width, so your signal lands
in the wrong place and won't lock.

**PPM correction** tells GopherTrunk to shift by that amount so signals appear at their
**true frequency**. To find your value: tune to a **known, stable reference** and adjust
PPM until it sits exactly where it should, or use a measurement tool. The value is
roughly constant per dongle, though it **drifts with temperature** — let the dongle warm
up for a few minutes before calibrating. A correct PPM is exactly what cures the
*rotating constellation* from the [tuning lesson](/learn/tuning-with-scopes/).

## Voice calibration for clean audio

Once a system locks, [voice calibration](/voice-calibration.html) dials in the **decoded
audio** — levels and clarity — so recordings and live playback sound right. This is
separate from getting a lock: a perfectly locked system can still need its audio tuned.
Follow the [Voice calibration](/voice-calibration.html) guide for the specifics.

## The troubleshooting checklist (by pipeline stage)

When something's wrong, march down the [signal path](/learn/antenna-to-audio/) — the
failure almost always lives at one identifiable stage:

| Symptom | Stage | Check |
|---------|-------|-------|
| Flat spectrum, nothing | Antenna / gain | Antenna connected? Up and clear? [Gain](/learn/gain-and-agc/) high enough? Signal above [noise floor](/learn/decibels/)? |
| Smeared/distorted spectrum, ghosts | Gain | ADC **clipping** — reduce gain |
| Signal present, constellation rotating | Frequency | Set/correct **PPM** |
| Constellation fuzzy despite strong meter | SNR | Improve [antenna/placement](/learn/propagation/); check for nearby overload |
| Control channel won't lock | Demod/params | Right control-channel frequency and [system type](/learn/protocol-landscape/)? |
| Locks but no calls | Decode/params | Correct system parameters; right talkgroups not all locked out |
| Calls listed but silent | Vocoder/encryption | Other talkgroups OK? If yes, likely **[encrypted](/learn/encryption/)** |

The [scopes](/learn/tuning-with-scopes/) turn most of these from guesswork into a glance.

## Common failure modes and fixes

- **Wrong PPM** → rotating constellation, no lock → calibrate PPM.
- **Gain too high** → clipping, ghost signals → reduce gain.
- **Gain too low / poor antenna** → buried in noise → improve SNR.
- **Wrong control-channel frequency or system type** → no lock → re-check the
  [system details](/learn/finding-systems/).
- **Encrypted talkgroup** → silent calls with visible metadata → expected, not fixable.
- **USB issues at high sample rate** → dropped samples, glitches → lower
  [sample rate](/learn/sample-rate-nyquist/) or use a better USB port/cable.

## When to suspect hardware vs. config

Most problems are **configuration or signal**, not hardware. Suspect **hardware** only
after the basics check out: try a **different USB port/cable** (power and data issues are
common), let the dongle **warm up** (drift), test a **known-good system** to prove the
chain, and watch for **overheating** on long runs. If a known-good system decodes
elsewhere but not on this dongle, *then* consider the hardware. Otherwise, it's almost
always antenna, gain, PPM, or parameters — in that order.

<div class="knowledge-check" data-quiz data-correct-msg="Right — PPM corrects the dongle's frequency error so signals land on frequency." markdown="0">
  <p class="knowledge-check__q">Quick check: a strong signal shows a slowly rotating constellation and won't lock. The fix?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Increase the gain</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Set the correct PPM frequency correction</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Buy a bigger antenna</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **PPM correction** aligns the dongle to true frequency — cures rotating constellations.
- **Voice calibration** dials in clean decoded audio, separate from getting a lock.
- Troubleshoot by **walking the signal path**: antenna → gain → PPM → parameters.
- **Calls listed but silent** is usually [encryption](/learn/encryption/), not a fault.
- Suspect **hardware last**, after antenna, gain, PPM, and config check out.

Last lesson: the rules and etiquette that make all of this responsible.
