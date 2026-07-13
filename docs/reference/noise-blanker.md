---
slug: noise-blanker
title: Noise blanker
entry_type: term
category: sdr-dsp
description: "A noise blanker detects short impulse-noise spikes and gates them out of the signal before demodulation, reducing clicks from ignition and switching noise."
keywords: noise blanker, noise blanking, impulse noise, ignition noise, pulse noise, blanking gate, click removal, interference mitigation, SDR noise reduction
aka: [noise blanking, impulse blanker, pulse blanker]
autolink: true
infobox:
  - { label: Type, value: Interference-mitigation stage }
  - { label: Targets, value: Short impulse/pulse noise }
  - { label: Action, value: Detect spike, mute the sample }
see_also: [noise-floor, energy-detection, noise-figure, thermal-noise, adaptive-filter]
cite_urls:
  - https://en.wikipedia.org/wiki/Noise_blanker
---

**A noise blanker** detects brief, high-amplitude impulse-noise spikes and momentarily gates
them out of the receive path so they never reach the demodulator, removing the sharp clicks that
impulsive interference would otherwise produce.[^wiki] It works in the time domain and targets
noise that is *short* compared with a symbol — ignition sparks, power-line arcing, electric
fences, and switching supplies — rather than the steady hiss of the
[noise floor](/reference/noise-floor/). Because an impulse concentrates its energy in time, it is
easier to reject where it is concentrated: with a fast switch that blanks the offending sample.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A signal waveform interrupted by a tall narrow noise spike; a detector flags the spike above a threshold and the blanker replaces that interval with a muted gap." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="90" x2="215" y2="90" stroke="currentColor" stroke-width="1"/>
  <path d="M30 90 Q 70 74 110 90 T 190 90" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <line x1="150" y1="90" x2="150" y2="28" stroke="currentColor" stroke-width="2"/>
  <text x="150" y="22" font-size="8.5" fill="currentColor" text-anchor="middle">impulse</text>
  <line x1="30" y1="50" x2="215" y2="50" stroke="currentColor" stroke-width="0.9" stroke-dasharray="4 3"/>
  <text x="34" y="46" font-size="8" fill="currentColor">threshold</text>
  <text x="120" y="120" font-size="9" fill="currentColor">input</text>
  <line x1="225" y1="90" x2="255" y2="90" stroke="currentColor" stroke-width="1.2" marker-end="url(#nbar)"/>
  <text x="240" y="82" font-size="7.5" fill="currentColor" text-anchor="middle">blank</text>
  <line x1="265" y1="90" x2="360" y2="90" stroke="currentColor" stroke-width="1"/>
  <path d="M265 90 Q 300 74 335 90" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <line x1="343" y1="90" x2="360" y2="90" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <path d="M368 90 Q 405 74 440 90" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <rect x="343" y="82" width="17" height="16" fill="none" stroke="currentColor" stroke-width="0.8" stroke-dasharray="2 2"/>
  <text x="365" y="120" font-size="9" fill="currentColor">output (gap)</text>
  <defs><marker id="nbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An impulse that exceeds the detector threshold is replaced by a short muted gap, so the click never reaches the demodulator.</figcaption>
</figure>

## How it works

A noise blanker has two parts: a **detector** and a **gate**. The detector forms a fast estimate
of the signal envelope and compares it to a threshold set a few dB above the running average
level. Because a genuine impulse rises far above the ordinary signal-plus-noise envelope, it
crosses the threshold cleanly. When it does, the gate blanks a short window — typically it holds
the sample at zero (a hard blank) or, better, replaces the interval with the surrounding average
so the sudden zero does not itself create a spectral splatter. The window is deliberately narrow,
just long enough to cover the pulse plus its filter ringing.

The subtlety is timing. By the time the envelope detector fires, the impulse has already entered
any narrow filter ahead of it, and a filter smears a sharp pulse into a longer ring. Effective
blankers therefore tap the signal at a **wide bandwidth** — ahead of the channel filter, where
the impulse is still short — so the blank window can be brief. This is why a blanker built into a
wideband SDR front end can outperform one buried after a narrow IF filter.

The limits are important. A blanker helps only when noise is impulsive and sparse; it does
nothing for continuous interference or for raising the noise floor. Set too sensitively, it
blanks on strong wanted signals or on modulation peaks, punching holes that *degrade* reception
and can create intermodulation-like artefacts from the switching itself. It is a scalpel for
clicks, not a general noise reducer.

## Relevance to SDR

Noise blankers are standard in HF and VHF communications receivers and are a common feature in
SDR software (SDR#, SDRangel, GQRX, and ham transceivers), where mobile installations plagued by
ignition noise benefit most. In digital land-mobile decoding the payoff is indirect: an
un-blanked impulse can flip symbols and break [frame sync](/reference/frame-synchronization/),
so removing it lowers the bit-error rate on marginal signals. GopherTrunk does not implement a
dedicated noise blanker in its DSP chain — its focus is channelization and digital demodulation —
so impulse mitigation, where needed, is best handled upstream at the SDR/front-end stage before
the [I/Q](/reference/iq-data/) reaches the decoder.

## Sources

[^wiki]: [Noise blanker](https://en.wikipedia.org/wiki/Noise_blanker) — Wikipedia, on time-domain gating of impulse noise ahead of demodulation.
