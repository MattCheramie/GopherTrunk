---
slug: mbe-adaptive-smoothing
title: MBE adaptive smoothing
entry_type: term
category: voice-coding
description: "MBE adaptive smoothing is the IMBE decoder's soft-error handling: a running channel error-rate estimate caps amplitude spikes, reclaims obviously-voiced harmonics, and mutes hopeless frames, while leaving clean audio untouched."
keywords: MBE adaptive smoothing, IMBE soft error, error rate recursion, amplitude cap, voicing cleanup, frame mute, adaptive smoothing, TIA-102.BABA, multi-band excitation, forward error correction
aka: ["adaptive smoothing", "soft-error handling", "amplitude smoothing"]
autolink: true
infobox:
  - { label: Role, value: Soft-error cleanup of model params }
  - { label: Driver, value: Running channel error-rate estimate }
  - { label: Mute threshold, value: "error rate > 0.0875" }
  - { label: Clean frame, value: No-op (bit-identical) }
see_also: [multi-band-excitation, imbe, forward-error-correction, mbe-spectral-enhancement, mbe-voiced-synthesis]
cite_urls:
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Forward_error_correction
---

**MBE adaptive smoothing** is the [IMBE](/reference/imbe/) decoder's soft-error handling stage:
using a running estimate of the channel error rate, it tames the amplitude spikes and voicing
glitches that survive [forward error correction](/reference/forward-error-correction/) on a weak
signal, and mutes frames the channel has corrupted beyond rescue.[^mbe] Its defining property is
restraint — on a clean channel it changes nothing at all, only advancing its trackers, so
well-decoded audio is bit-identical with and without the stage.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 130" role="img" aria-label="A running error rate rises with each frame's corrected-bit count; below a clean threshold the model parameters pass through untouched, in a middle band amplitude spikes are capped and clearly-voiced harmonics reclaimed, and above the mute threshold the frame is silenced." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-opacity="0.4"><line x1="30" y1="105" x2="440" y2="105"/><line x1="30" y1="20" x2="30" y2="105"/></g>
  <line x1="30" y1="88" x2="440" y2="88" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.7"/>
  <line x1="30" y1="40" x2="440" y2="40" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.7"/>
  <text x="436" y="99" text-anchor="end" font-size="7" fill="currentColor">clean → pass through</text>
  <text x="436" y="60" text-anchor="end" font-size="7" fill="currentColor">cap spikes · reclaim voicing</text>
  <text x="436" y="34" text-anchor="end" font-size="7" fill="currentColor">mute (er &gt; 0.0875)</text>
  <path d="M30 100 L110 96 L180 80 L250 66 L320 48 L400 30" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="34" y="18" font-size="7.5" fill="currentColor">error rate</text>
</svg>
<figcaption>The running error-rate estimate selects the response: pass-through when clean, cap and reclaim in the middle band, mute when the channel is hopeless.</figcaption>
</figure>

## The error-rate recursion

The stage keeps a single scalar, a leaky-integrator estimate of the channel error rate driven by
how many bits the FEC had to correct each frame:

    er = 0.95·er_prev + 0.000365·correctedBits

This is the JMBE/OP25 recursion exactly. A frame is declared **hopeless and muted** when
`er > SmoothMuteErrorRate = 0.0875`; at the IMBE FEC budget (~15 correctable bits per frame) that
threshold requires a sustained ~12+ corrected bits per frame — a channel so bad the audio would be
unintelligible anyway. `UpdateErrorRate` in `internal/voice/mbe/smoothing.go` advances this
estimate on *every* frame, on every code path, so the running rate is uniform regardless of what
else the decoder does.

## The clean-channel guarantee

Before touching any model parameter, `Smooth` checks a clean-frame test: if `er ≤ 0.005` **and** the
current frame's corrected-bit count is `≤ 6`, it advances only its local-energy tracker and returns,
leaving the amplitudes `M` and the voicing decisions `Vl` exactly as decoded. This is the
guarantee that adaptive smoothing never regresses good audio — a clean call decodes identically
whether or not the stage is present, which also makes the stage safe to run unconditionally.

## Scale-free thresholds

The reference decoders express their amplitude and voicing thresholds as fixed absolute constants
tied to their internal amplitude scale. GopherTrunk's linear amplitudes live on a different scale,
so the two parameter-cleanup rules are expressed **relative to a running local-energy estimate**
instead, which tracks the recent speech level:

    le = 0.95·le_prev + 0.05·RM0     (floored at 1.0)

The error-rate recursion and the mute threshold, by contrast, are scale-free and match the
reference exactly. Two cleanups then run, but only once the clean-frame test has failed:

- **Amplitude cap.** If the frame energy `RM0` spikes above `2.0 × localEnergy` — a hallmark of
  bit-error corruption — every amplitude is scaled by `sqrt(cap / RM0)` to pull the frame back to a
  plausible level.
- **Voicing cleanup.** A harmonic whose amplitude exceeds `2.0 × RMS` (the per-harmonic RMS of the
  smoothed local energy) is almost certainly voiced, so if a corrupted voicing bit marked it
  unvoiced, the stage reclaims it by forcing `Vl[l] = 1`.

## Where it sits

Adaptive smoothing runs on the decoded model parameters after
[spectral enhancement](/reference/mbe-spectral-enhancement/) and before the
[voiced](/reference/mbe-voiced-synthesis/) and [unvoiced](/reference/mbe-unvoiced-synthesis/)
synthesis stages consume them. It is the counterpart to enhancement: enhancement is a perceptual
polish applied to *good* parameters, while smoothing is damage control applied to *corrupted* ones.
The two never fight, because smoothing is inert whenever the channel is clean.

## Relevance to SDR

For a scanner working real, marginal RF, adaptive smoothing is what makes a fading
[P25 Phase 1](/reference/imbe/) signal degrade into a muffled-but-intelligible warble and then a
clean mute, rather than erupting into loud bit-error squawks. It converts the FEC's corrected-bit
count — information the decoder already has — into a graceful-degradation policy, so the listener
hears the signal fade instead of the decoder failing.

## Sources

[^mbe]: [Multi-Band Excitation](https://en.wikipedia.org/wiki/Multi-Band_Excitation) — Wikipedia, on the IMBE model parameters that adaptive smoothing cleans up.
[^fec]: [Forward error correction](https://en.wikipedia.org/wiki/Forward_error_correction) — Wikipedia, on the corrected-bit count that drives the error-rate estimate.
