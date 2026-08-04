---
slug: tetra-receiver-chain
title: TETRA receiver chain
entry_type: term
category: sdr-dsp
description: The TETRA receiver chain is the IQ-to-dibit demodulation pipeline — RRC matched filter, Gardner symbol-timing recovery, a frozen-tap CMA snapshot equalizer, then π/4-DQPSK differential decode — that turns a tuned 18000 sym/s carrier into the dibit stream the channel decoders read.
keywords: TETRA receiver, TETRA demodulator, pi/4 DQPSK receiver, RRC matched filter, Gardner timing recovery, snapshot CMA equalizer, differential decode, 18000 sym/s
aka: [TETRA demod pipeline, "TETRA IQ-to-dibit chain", "TETRA demodulator"]
autolink: true
infobox:
  - { label: Modulation, value: "π/4-DQPSK, 18000 sym/s" }
  - { label: Matched filter, value: "RRC, α = 0.35" }
  - { label: Timing, value: Gardner symbol recovery }
  - { label: Equalizer, value: frozen-tap CMA snapshot }
see_also: [pi-4-dqpsk, cma-equalizer, gardner-timing-recovery, matched-filter, differential-decoding, automatic-gain-control, tetra, tetra-burst-formats, tetra-training-sequences, tetra-logical-channels]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://en.wikipedia.org/wiki/Differential_coding
---

The **TETRA receiver chain** is the signal-processing pipeline that turns a tuned
[TETRA](/reference/tetra/) carrier into the stream of **dibits** every higher layer reads.[^tetra]
TETRA's physical layer is [π/4-DQPSK](/reference/pi-4-dqpsk/) at **18000 symbols per second**, one
dibit (two bits) per symbol, shaped by a root-raised-cosine pulse with roll-off α = 0.35. The
receiver undoes that shaping and clock, corrects the channel, and recovers each symbol's differential
phase — the information TETRA actually carries.[^diff]

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 130" role="img" aria-label="A left-to-right pipeline: complex IQ enters an RRC matched filter, then a Gardner symbol-timing loop, then a frozen-tap CMA snapshot equalizer, then a pi/4-DQPSK differential decoder, which emits a stream of dibits." xmlns="http://www.w3.org/2000/svg">
  <text x="18" y="52" font-size="9" fill="currentColor">IQ</text>
  <rect x="40" y="34" width="86" height="34" rx="3" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="83" y="49" text-anchor="middle" font-size="8" fill="currentColor">RRC matched</text>
  <text x="83" y="60" text-anchor="middle" font-size="8" fill="currentColor">filter α=0.35</text>
  <rect x="140" y="34" width="86" height="34" rx="3" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="183" y="49" text-anchor="middle" font-size="8" fill="currentColor">Gardner</text>
  <text x="183" y="60" text-anchor="middle" font-size="8" fill="currentColor">timing</text>
  <rect x="240" y="34" width="96" height="34" rx="3" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/>
  <text x="288" y="49" text-anchor="middle" font-size="8" fill="currentColor">CMA snapshot</text>
  <text x="288" y="60" text-anchor="middle" font-size="8" fill="currentColor">equalizer</text>
  <rect x="350" y="34" width="96" height="34" rx="3" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="398" y="49" text-anchor="middle" font-size="8" fill="currentColor">π/4-DQPSK</text>
  <text x="398" y="60" text-anchor="middle" font-size="8" fill="currentColor">diff decode</text>
  <text x="470" y="52" font-size="8" fill="currentColor">dibits</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <path d="M30 51 L40 51"/><path d="M126 51 L140 51"/><path d="M226 51 L240 51"/><path d="M336 51 L350 51"/><path d="M446 51 L466 51"/>
  </g>
  <text x="260" y="96" text-anchor="middle" font-size="7.5" fill="currentColor">taps adapt continuously · a FROZEN snapshot is applied per burst</text>
</svg>
<figcaption>Each stage feeds the next: the matched filter maximises symbol SNR, Gardner recovers symbol time, the snapshot equalizer inverts the channel with frozen taps, and the differential decoder reads the π/4 phase steps as dibits.</figcaption>
</figure>

## The stages

The chain runs four stages in order. A [matched filter](/reference/matched-filter/) — a root-raised-cosine
FIR designed around the same α = 0.35 pulse the transmitter uses — maximises symbol signal-to-noise and
suppresses inter-symbol interference from the pulse shaping itself. The filtered stream is still at many
samples per symbol, so [Gardner timing recovery](/reference/gardner-timing-recovery/) closes a feedback
loop that estimates the correct sampling instant and decimates to exactly one sample per symbol; it
recovers the clock without needing to know the carrier phase, which suits a differentially-coded signal.
An optional residual-carrier [AFC](/reference/automatic-frequency-control/) sits ahead of timing recovery
because a spinning constellation corrupts the Gardner metric. Then the equalizer inverts any linear
channel distortion, and finally the [differential decoder](/reference/differential-decoding/) reads each
symbol's phase step relative to the previous symbol and classifies it into a 0..3 dibit.

## Why a snapshot equalizer

Real captures — especially concurrent-load traffic carriers — suffer multipath and band-edge group delay
that smear the constellation. A blind [CMA equalizer](/reference/cma-equalizer/) inverts that linear
channel without a training sequence, by driving the output toward constant modulus. But CMA is placed
*ahead of a differential decoder*, and that placement is a trap: CMA's cost is rotation-invariant, so a
continuously-adapting equalizer's output phase wanders as its taps update. A **time-varying** phase does
not cancel in the differential product `s·conj(last)` — every dibit is corrupted. The design answer is to
adapt a tracking filter continuously but **apply a frozen snapshot** of its taps, refreshed only every few
hundred symbols (far longer than one 255-symbol burst). Each burst then sees a constant filter and a
constant phase that the differential step cancels cleanly; the single symbol straddling a snapshot
boundary is absorbed by the downstream FEC. On the reporter's captures this roughly **doubles** the
CRC-valid traffic-burst yield with no regression on already-clean captures.

## Relevance to SDR

`internal/radio/tetra/receiver/receiver.go` composes the chain: an RRC `PiOver4DQPSK` matched filter,
a `sync.Gardner` loop (selectable via `ClockMode`), a `carrierAFC`, a channel-select FIR that rejects
adjacent carriers admitted by the wide channelised passband, an `equalizer.SnapshotCMA`, and the
differential decoder that emits dibits through a `DibitSink` keyed by an absolute dibit index. A parallel
`SoftSink` can emit each symbol's complex differential as [soft](/reference/soft-decision/) information for
soft-decision channel decoding. The receiver is deliberately rate-invariant — everything downstream is sized
from the 18000 sym/s symbol rate, not the capture rate — so the same object drives both the control-channel
state machine and each per-call voice tap.

## Sources

[^tetra]: [Terrestrial Trunked Radio](https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio) — Wikipedia, on the TETRA standard and its π/4-DQPSK physical layer.
[^diff]: [Differential coding](https://en.wikipedia.org/wiki/Differential_coding) — Wikipedia, on decoding information carried in the phase change between successive symbols.
