---
slug: squelch
title: Squelch
entry_type: term
category: sdr-dsp
description: "Squelch mutes a receiver's audio or output until a signal meeting a set criterion — carrier level, noise, or a subaudible tone — is present."
keywords: squelch, carrier squelch, noise squelch, CTCSS squelch, tone squelch, DCS squelch, mute, signal presence, receiver gating, scanner squelch
aka: [muting, carrier-operated relay, COR, tone squelch]
autolink: true
infobox:
  - { label: Type, value: Receiver gating stage }
  - { label: Criterion, value: Carrier / noise / tone / code }
  - { label: Output, value: Open (pass) or closed (mute) }
see_also: [ctcss, dcs, signal-to-noise-ratio, noise-floor, carrier-to-noise-ratio, energy-detection]
cite_urls:
  - https://en.wikipedia.org/wiki/Squelch
---

**Squelch** is the circuit or algorithm that mutes a receiver's output until an incoming signal
meets a chosen presence criterion, so the listener hears active traffic instead of the
open-channel hiss of the [noise floor](/reference/noise-floor/).[^wiki] It decides, moment by
moment, whether the channel is busy or idle and gates the audio (or the downstream decoder)
accordingly. Squelch is what lets a scanner sweep many channels and stop only on the ones
actually carrying a transmission.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A signal level rising above and falling below a squelch threshold, with the receiver output muted below the threshold and passed above it, plus hysteresis between open and close points." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="140" x2="440" y2="140" stroke="currentColor" stroke-width="1.1"/>
  <line x1="30" y1="30" x2="30" y2="140" stroke="currentColor" stroke-width="1.1"/>
  <text x="18" y="34" font-size="8.5" fill="currentColor" text-anchor="end">level</text>
  <line x1="30" y1="80" x2="440" y2="80" stroke="currentColor" stroke-width="1" stroke-dasharray="4 3"/>
  <text x="435" y="74" font-size="8.5" fill="currentColor" text-anchor="end">threshold</text>
  <path d="M30 128 L90 126 L140 60 L210 55 L260 118 L320 122 L360 50 L410 52 L440 120" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <rect x="140" y="150" width="112" height="10" fill="currentColor" opacity="0.85"/>
  <rect x="360" y="150" width="52" height="10" fill="currentColor" opacity="0.85"/>
  <text x="196" y="167" font-size="8" fill="currentColor" text-anchor="middle">output open</text>
  <text x="120" y="167" font-size="8" fill="currentColor" text-anchor="middle">muted</text>
</svg>
<figcaption>The output is muted while the measured level sits below the threshold and opens once it rises above; hysteresis keeps it from chattering near the edge.</figcaption>
</figure>

## How it works

A squelch continuously measures some proxy for signal presence and compares it to a threshold.
The classic implementations differ in *what* they measure:

- **Carrier (level) squelch** compares received signal strength — the RSSI or, in an SDR, the
  in-channel power estimate — against a set level. Simple and fast, but it opens on any carrier,
  including interference.
- **Noise squelch** looks instead at energy in a band *above* the audio passband (typically
  several kHz up). A strong FM signal quiets that band as the discriminator captures; the drop
  in high-frequency noise is a reliable "signal present" cue that is largely independent of
  absolute level, which is why it is the traditional choice for analog FM.
- **Tone squelch ([CTCSS](/reference/ctcss/))** additionally requires a specific subaudible tone
  (67–250 Hz) to be present, so the receiver stays muted for other users sharing the same RF
  channel.
- **Digital code squelch ([DCS](/reference/dcs/))** does the same with a continuous low-rate
  digital codeword instead of a tone.

Two refinements matter in practice. **Hysteresis** uses a slightly higher level to open than to
close, so a signal hovering near threshold does not rapidly toggle ("squelch chatter"). A
**hang time** keeps the gate open for a short interval after the signal drops, so brief fades
mid-transmission do not clip the audio.

A distinct control often confused with squelch is the **attack/decay** of the level estimator.
The presence measurement is a smoothed average of the channel power; a fast estimator reacts
quickly to a signal appearing but is jittery, while a slow one is steady but sluggish to open and
close. Tuning that time constant against the hang time is what separates a squelch that snaps
cleanly onto short transmissions from one that either clips their start or lingers on noise after
they end.

## In practice

Setting squelch is a trade-off. Too tight (threshold too high) and weak but wanted signals are
rejected; too loose and the channel opens on noise. The useful reference is the
[signal-to-noise ratio](/reference/signal-to-noise-ratio/): a good squelch opens a few dB above
the noise floor so that anything it passes is actually intelligible. Digital voice systems fold
this decision into the modem — a P25 or DMR receiver "opens" only when it detects a valid
[frame sync](/reference/frame-synchronization/) pattern and produces good symbols, which is a far
more selective test than raw energy.

## Relevance to SDR

Every scanner and SDR receiver applies squelch of some kind. Analog FM applications (marine VHF,
amateur, business band) still use carrier or CTCSS/DCS tone squelch. In a software receiver the
measurement is just a power or sync estimate computed from the [I/Q](/reference/iq-data/) stream,
so the same logic scales to many channels at once. GopherTrunk's channel-activity logic is a
squelch in spirit: on a trunked system it does not gate on audio energy but on control-channel
decodes and, on a granted voice channel, on valid digital sync — it passes traffic only when the
decoder confirms a live transmission, and treats CTCSS/DCS as identifying sub-channels on the
conventional analog channels it can also monitor.

## Sources

[^wiki]: [Squelch](https://en.wikipedia.org/wiki/Squelch) — Wikipedia, on carrier, noise, and tone squelch and their use in receivers and scanners.
