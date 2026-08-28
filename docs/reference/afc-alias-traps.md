---
slug: afc-alias-traps
title: AFC alias traps
entry_type: term
category: fn-diagnostics
description: "AFC alias traps are the ways a frequency estimator's alias structure silently breaks a lock: a wrong alias bucket that rejects every correct estimate, a transient that re-primes an established track, and a WARN that fires on one bad sample."
keywords: AFC, automatic frequency control, alias bucket, fourth-power estimator, re-prime, locked but deaf, payload heartbeat, resync, carrier offset warn, frequency spike, TETRA AFC
aka: [AFC alias bucket traps, frequency estimator aliasing]
infobox:
  - { label: Type, value: DSP + diagnostics facts }
  - { label: Key rule, value: "A 4th-power estimator is only unambiguous within ±f_sym/8" }
  - { label: Signature, value: "Sync ~100%, payload 0% — locked but deaf" }
  - { label: Tell, value: "A reported offset error that is an exact multiple of f_sym/4" }
see_also: [automatic-frequency-control, tetra-lock-facts, carrier-offset-adjacent-lock, frequency-locked-loop, signal-signatures, pi-4-dqpsk]
related_reading:
  - { title: "From the Issue Tracker, Part 18: The Stall That Wasn't", url: /blog/solution-postmortem/from-the-issue-tracker-18-the-stall-that-wasnt/ }
cite_urls:
  - https://github.com/MattCheramie/GopherTrunk/issues/815
  - https://github.com/MattCheramie/GopherTrunk/issues/940
---

**AFC alias traps** are the failure modes that follow from one structural fact of
[automatic frequency control](/reference/automatic-frequency-control/) on PSK signals: the
fine estimator is only unambiguous within a fraction of the symbol rate, so every estimate
it produces is really "the true offset, modulo an alias spacing". For
[π/4-DQPSK](/reference/pi-4-dqpsk/), raising the per-symbol differential phase to the 4th
power collapses the data — but wraps at 2π, leaving the offset recoverable only within
±f_sym/8 (±2,250 Hz at TETRA's 18,000 sym/s) and aliased in steps of f_sym/4 (4,500 Hz).
Everything below is a consequence of what happens when the tracking loop lands in the wrong
alias bucket — and each mode carries a distinct signature GopherTrunk now checks for.

## The wrong-bucket latch: "locked but deaf"

An EMA-tracked AFC guards itself by rejecting per-block estimates far from its current
track. That guard is also a trap: if the track is ever primed from one bad block — say, a
decode-drought resync re-seeding from a weak burst — the loop latches a bucket ±4,500 Hz
off and then *rejects every correct estimate forever*, because they all disagree with the
track by exactly the alias spacing. The cruel part is what stays working: TETRA's BSCH
survives the resulting ~17°/symbol residual (its sync burst gets per-burst frequency
correction and heavy FEC), while every SCH payload fails. The signature is unmistakable
once named: **`bsch_ok` ~100%, `sch_pdus` = 0** — a receiver that is provably synchronised
and decodes nothing. One field log showed 37 minutes of this across a 10.5-hour session,
unrecoverable because every decoded BSCH stamped the only activity heartbeat that both
resync mechanisms fed on.

Two fixes, both required: a clustered-reject escape (three consecutive rejected estimates
that *agree with each other* re-prime the track) and an independent **payload heartbeat** —
a liveness timestamp stamped only by CRC-clean *payload* decodes, driving a forced resync
after 12 s of signal with a live lock and no payload, escalating to a full re-hunt after
three fruitless resyncs. Sync proving itself is not evidence the receiver works; only
payload is.

## The transient re-prime: the same signature, younger

The clustered-reject escape created its own trap. Any transient bias larger than the
unambiguous range — a neighbouring carrier's skirts leaking through the channel-filter edge
while the wanted signal fades — lands the raw estimate exactly one alias bucket off, and
three such blocks (~170 ms) tripped the escape, slamming a long-corroborated track into the
wrong bucket. An operator filmed the mixer panel's carrier readout spiking to ~−5,004 Hz on
a carrier really ~500 Hz off: **5,004 = 504 + 4,500** — the reported "spike" was the alias
spacing itself, the tell that this is an estimator artifact and not a carrier move. The
resolution distinguishes track *age*: a fresh track keeps the fast 3-block escape (a wrong
seed must be escapable quickly), while an established track (≥16 accepted blocks) demands
an 18-block (~1 s) streak before an alias-class cluster may re-prime it. A wrongly-held
transient self-heals; a wrongly-accepted one costs minutes.

## One bad sample is not a condition

The companion diagnostic trap: a wrong-site WARN that fired whenever a single per-chunk
offset sample crossed a threshold, so every sub-second estimator blip produced a scary
"carrier offset 5 kHz — check your site" report. A warning that diagnoses a *persistent*
condition must observe persistence: the offset now has to sit over threshold continuously
for 10 s, with the excursion clock reset on dips. The general rule travels well beyond AFC:
alarm on integrated evidence, never on one sample of a noisy estimator.

## Symptom table

| Symptom | Looks like | Actually | Fix / check |
|---|---|---|---|
| Sync ~100%, payload 0%, forever | Encryption, dead channel | AFC latched a wrong alias bucket; correct estimates rejected | Clustered-reject re-prime + payload heartbeat ([#940](https://github.com/MattCheramie/GopherTrunk/issues/940)) |
| Reported offset error ≈ n × f_sym/4 | Oscillator jump | Alias-class estimator artifact | Recognise the spacing; check track age before re-priming |
| Carrier readout spikes ~±4.5 kHz then recovers | RF instability | Transient coarse bias through the filter edge | Established-track hold (~1 s streak) before re-prime ([#815](https://github.com/MattCheramie/GopherTrunk/issues/815)) |
| Wrong-site WARNs during normal operation | Neighbour interference | Threshold on one instantaneous sample | Require 10 s persistence before warning |
| Resyncs fire but never help | CPU starvation | Resyncs re-seed from the same wrong regime | Escalate to full re-hunt after N fruitless resyncs |

## Provenance

- [#940](https://github.com/MattCheramie/GopherTrunk/issues/940) — the 4th-power estimator's ±2,250 Hz ambiguity, the coarse-acquisition stage, and the pre-matched-filter rule (see also [TETRA lock facts](/reference/tetra-lock-facts/)).
- [#815](https://github.com/MattCheramie/GopherTrunk/issues/815) — the carrier-offset WARN and its false triggers; the −5,004 Hz alias spike diagnosis.
- 19 Aug field log — the 37-minute "locked but deaf" stretches, the payload heartbeat, and the track-age re-prime rules, each pinned by a failing-first regression.
