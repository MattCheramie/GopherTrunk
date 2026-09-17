---
title: "DMR End to End, Part 10: Wideband DMR — Bin Edges & the Deaf Heal"
description: "Two IPSC repeaters on one X310 channelizer, one tap deaf for minutes at a time — how the polyphase bin edge folded half a DMR channel back on itself, why the fix was a 2x-oversampled channelizer, and how the wideband engine now heals a receiver deaf at its own decoding level and rejects a coarse offset decode evidence contradicts."
category: deep-dives
keywords: wideband dmr channelizer, polyphase channelizer bin edge, oversampled polyphase filter bank, dmr tap deaf sync_hits zero, ipsc wideband decode, coarse carrier offset reject, tuner_strategy polyphase, harris m/2 channelizer, gophertrunk dmr
tags: [dmr-end-to-end, dmr, wideband, channelizer, dsp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "DMR End to End"
series_part: 10
---

*Part 10 of **DMR End to End**, a 14-part deep dive that follows the world's
most widely deployed digital PMR protocol through GopherTrunk — from a 4FSK
carrier to two simultaneous recorded calls, direct-mode handhelds, and
decrypted Enhanced Privacy voice.
[Part 9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
put one handheld on one dongle. This part goes to the other end of the
scale: an X310 carrying two IPSC repeaters through a polyphase channelizer —
and one tap going deaf for minutes while its neighbour decoded everything.
Two decode paths, two ways for a tap to go quiet: the series' running thread
again.*

> **TL;DR:** The wideband tap for a 442.3875 MHz IPSC repeater logged
> `sync_hits=0` for minutes on a 30 dB SNR carrier a DDC on the same IQ
> decoded continuously. Cause one: the repeater sat **0.48 bins** off a
> channelizer bin centre, where a critically-sampled polyphase bin is −6 dB
> and folds the half of a 12.5 kHz channel past ±binRate/2 back onto itself.
> `channelizer.Oversampled` (Harris M/2, cutoff 0.75·Fs/M, bins at 2·Fs/M)
> makes the whole bin flat. Cause two, still open: the live
> receiver chain latches deaf in ~3-minute stretches at a steady −51 dBFS,
> so `healDeafTier2` resets the Tier II receiver after three windows without
> sync **within 6 dB of the level it last synced at** — never an absolute
> dBFS. On 15 Sep that instrument named a third thing: the coarse acquirer
> freezing on a −20.1 kHz neighbour in every idle gap;
> `Receiver.RejectCoarseCarrierOffset` now rules out any engage the channel
> never synced under. The emitter is unidentified.

**Key takeaways**

- **A bin edge is a channel boundary only if the prototype says so.** 2x
  oversampling moves the Nyquist edge to ±Fs/M and the cutoff to 0.75·Fs/M,
  so a tap at residual 0.48 decodes like one at 0.
- **Yield is the verdict, a tone is the pin.** A synthetic carrier at
  residual 0.48 still *granted* through the old bank; the regression is tone
  flatness plus the capture harness.
- **Gate health on the channel's own level, never on dBFS.** The deaf tap sat
  at −51 dBFS, below every absolute hint floor, so no WARN ever fired.
- **Decode evidence settles what frequency cannot.** A 12.5 kHz neighbour and
  a 45 ppm tuner error look identical to a discriminator mean; only an offset
  the tap decoded under is confirmed.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| 2x-oversampled channelizer | Harris M/2 polyphase: M/2 inputs per step, IDFT, per-bin rotation | `internal/dsp/channelizer/oversampled.go` (`Oversampled`) |
| Bank on top | fine-tune NCO + resampler per tap from 2·binRate | `internal/dsp/tuner/channelizerbank.go` (`ChannelizerBank`) |
| Bin-edge pin | tones at 0.45..0.51 bins leave the tap within [−1, +0.5] dB | `TestChannelizerBankBinEdgeChannelIsFlat` |
| Capture A/B | `ChannelizerBank` vs `DDCBank`, one tap each, same stream | `cmd/gophertrunk/dmr_ipsc_wideband_replay_test.go` (`GT_DMR_WB=1`) |
| Deaf-tap heal | 3 windows no sync within 6 dB of last-synced level ⇒ `Reset` + `ResyncReset` | `internal/scanner/widebandt2/engine.go` (`healDeafTier2`) |
| Coarse-offset reject | revert + ignore an engage the channel never synced under | `dmrrx.Receiver.RejectCoarseCarrierOffset`, `coarse_reject_test.go` |
| Counters | `dibits`, `deaf_heals` in the activity line | `widebandt2: channel decode activity` |

## In this post

- **Two taps, one channelizer, one deaf** — the field logs.
- **The bin edge** — −6 dB at the edge and a fold, measured on the capture.
- **Oversampling the channelizer** — Harris M/2 and what it costs.
- **The deaf heal** — resetting a receiver at its own decoding level.
- **The −20 kHz emitter and the reject** — decode evidence versus frequency.

## Two taps, one channelizer, one deaf

The rig is the one
[P25 End to End Part 11]({{ '/blog/deep-dives/p25-end-to-end-11-wideband/' | relative_url }})
described from the P25 side: `role: wideband`, an X310 at 6.25 MS/s,
`tuner_strategy: polyphase`, one `channels:` entry per carrier — here two
IPSC repeaters, Fire and Fire2, each driving its own `dmrrx.Receiver` and
[Part 6]({{ '/blog/deep-dives/dmr-end-to-end-06-conventional-ipsc-two-calls/' | relative_url }})'s
`tier2.ConventionalChannel` inside `widebandt2.Engine`. The bank is 32 bins
of 195.3125 kHz; Fire2 at 442.3875 MHz sits +687.5 kHz from centre —
**3.52 bins, 0.48 of a bin from bin 4's centre**.

The operator's 1200 s Signal Lab flac decodes 25 001 idle beacons and 14
grants through a DDC, and the 20-minute 12 Sep capture decodes every one of
its 26 transmissions (all 26 865 data bursts colour code 12 — not two colour
codes). The live tap logged `sync_hits=0` for minutes at a time at its
normal −51 dBFS while Fire on the same channelizer decoded throughout; ten of
the 26 transmissions fell in those stretches.

## The bin edge: −6 dB and a fold

`TestDMRIPSCWidebandReplay` re-creates the field geometry from a narrowband
capture: it interpolates the 25 kS/s slice to a wideband rate, mixes it to
the field offset, and feeds the same stream to a `ChannelizerBank` and a
`DDCBank` with one tap each. Over 300 s the old polyphase arm scored **1991
idle beacons and 2 grants against the DDC's 6103 and 7**, deaf in the same
stretches the live log showed. The mechanism is the prototype: a
critically-sampled bank emits each bin at Fs/M with a lowpass whose −6 dB
point sits *on* the bin edge, so
a 12.5 kHz channel centred 0.48 bins out is tilted by the roll-off and the
part past ±binRate/2 **folds back across baseband**:

| Tone residual (bins) | 0.45 | 0.48 | 0.50 | 0.51 |
|---|---|---|---|---|
| Old bank | −18.6 dB | −9.3 dB | −6.2 dB | −5.2 dB |

`TestChannelizerBankBinEdgeChannelIsFlat` now requires each within
[−1, +0.5] dB and unfolded, and fails on the old bank. What does **not**
work as a regression: a clean synthetic C4FM carrier at residual 0.48 still
*granted* through the old bank, because 200 header repeats need only a few
good bursts — a grant count is too forgiving to catch a 6 dB tilt, the
[metrics-that-lie lesson]({{ '/blog/deep-dives/weak-signal-engineering-02-metrics-that-lie/' | relative_url }})
inverted.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Two channelizer bin responses over frequency in bins. Left, a critically sampled bin rolls off to minus 6 decibels at the half-bin edge, and a DMR channel centred at 0.48 bins straddles it with the part beyond folding back across baseband. Right, the same bin two times oversampled: the Nyquist edge moves to one bin, the cutoff sits at 0.75 bins, and the channel lies inside the flat passband with no fold.">
  <text x="170" y="16" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">critically sampled: emit at Fs/M</text>
  <text x="510" y="16" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">Oversampled: emit at 2·Fs/M</text>
  <line x1="30" y1="150" x2="310" y2="150" stroke="var(--fg-muted)"/>
  <line x1="370" y1="150" x2="650" y2="150" stroke="var(--fg-muted)"/>
  <path d="M40 148 Q60 60 100 40 L240 40 Q280 60 300 148" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <line x1="240" y1="40" x2="240" y2="150" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="240" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+0.5 bin = Nyquist edge</text>
  <text x="170" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="8">0</text>
  <text x="262" y="60" fill="var(--fg-muted)" font-size="8">−6 dB</text>
  <rect x="232" y="100" width="24" height="50" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="244" y="94" text-anchor="middle" fill="var(--accent)" font-size="8">channel @ 0.48</text>
  <path d="M256 125 Q275 110 258 100" fill="none" stroke="var(--accent)" stroke-dasharray="3 2"/>
  <polygon points="260,98 254,104 262,106" fill="var(--accent)"/>
  <text x="290" y="112" text-anchor="end" fill="var(--accent)" font-size="8">folds back</text>
  <text x="170" y="184" text-anchor="middle" fill="var(--fg-muted)" font-size="8">−18.6 / −9.3 / −6.2 / −5.2 dB across 0.45..0.51</text>
  <path d="M380 148 Q400 46 430 40 L590 40 Q620 46 640 148" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <line x1="510" y1="40" x2="510" y2="150" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="510" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="8">0</text>
  <line x1="580" y1="40" x2="580" y2="150" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="580" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+0.5 bin</text>
  <text x="640" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+1 Nyquist</text>
  <text x="612" y="60" fill="var(--fg-muted)" font-size="8">cutoff 0.75</text>
  <rect x="572" y="100" width="24" height="50" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="584" y="94" text-anchor="middle" fill="var(--accent)" font-size="8">channel @ 0.48</text>
  <text x="510" y="184" text-anchor="middle" fill="var(--fg-muted)" font-size="8">whole ±0.5 bin flat within [−1, +0.5] dB — no fold</text>
</svg>
<figcaption>Same bin spacing, different Nyquist edge: oversampling by two moves the fold a whole bin away and puts the prototype's cutoff past every channel a tap can own.</figcaption>
</figure>

## Oversampling the channelizer

`channelizer.Oversampled` is the Harris M/2 polyphase structure: every
D = M/2 input samples it computes one output per bin,

```go
// internal/dsp/channelizer/oversampled.go (shape)
// y_k[s] = e^{-j2πk·t_s/M} · Σ_r e^{+j2πkr/M} · u_r,   u_r = Σ_p h[r+pM]·x[t_s−r−pM]
// proto: Kaiser prototype, cutoff 0.75·Fs/M;  rot[q] = e^{-j2πq/M}
```

The polyphase partial sums and the M-point inverse DFT are the
critically-sampled bank's. The new piece is the per-bin rotation
`rot[(k·tmod) % M]`: advancing output time by M/2 rather than M samples per
step leaves a phase the critically-sampled structure never has to correct.
Each bin emits at
2·Fs/M, its Nyquist edge moves to ±Fs/M, and with the cutoff at 0.75·Fs/M
everything within ±0.5 bin passes flat (`TestOversampledBinEdgeToneIsFlat`).

`ChannelizerBank` uses it unconditionally; its per-tap fine-tune NCO and
resampler run from `binOutRateHz = 2·binRateHz`, doubling per-tap cost: the
dense-71 bench moved from 1.5 to 2.8 ms per chunk against the `DDCBank`'s
11.8 ms — still cheaper, which is why
[SDR Internals Part 5]({{ '/blog/deep-dives/sdr-internals-05-tuning-channelization/' | relative_url }})
picks polyphase above six taps. Post-fix the polyphase arm matches the DDC
arm window for window, and the old "dense plan crowds taps onto bin edges"
WARN is gone (the
[polyphase filter bank]({{ '/reference/polyphase-filter-bank/' | relative_url }})
and [channelizer]({{ '/reference/channelizer/' | relative_url }}) pages have
the textbook version).

## The deaf heal

Fixing the bin edge did not explain the 12 Sep log, because that tap was not
weak — it was **deaf**. Fire2 logged `sync_hits=0` for six stretches of
180–210 s at −51 dBFS while offline replays of the same IQ — DDC,
channelizer, live-sized chunks via `GT_DMR_WB_CHUNK=500` — decoded
everything. Nothing in the daemon runs on a ~3-minute timer and the coarse
acquirer never engaged on the capture; state accumulates inside the live
receiver chain, and the root cause of that latch is **still open**.

What the engine can do without knowing the cause is what a fresh offline
receiver does by construction — start clean. `healDeafTier2` runs inline on
the pump goroutine at every diagnostics window: a channel that has synced
before and now sees no sync, FEC pass or beacon for `tier2DeafHealWindows`
= 3 consecutive windows (~15 s) at a power within `tier2DeafHealMarginDb`
= 6 dB of the level it last synced at gets `Receiver.Reset()` plus
`tier2.ResyncReset()`. The gate is the channel's **own** decoding level,
never an absolute dBFS — the rule that stopped the "channel iq power very
low" WARN firing on a Tier III control channel decoding every C_ALOHA at
−56 dBFS. The deaf tap sat at −51 dBFS, *below* the −45 dBFS floor of the
"strong signal but no sync" hint, which is why no WARN had ever fired; an
unkeyed repeater drops well below its decoding level, so idle silence never
trips the heal. The WARN — `widebandt2: conventional DMR tap deaf at its
decoding level — resetting the receiver` — carries `coarse_offset_hz`,
`agc_level`, `mm_mu` and `mm_sps`: the receiver internals are the instrument
for the next log.

Two smaller defects fell out: `dmrrx.Receiver.Reset` left the timing loop
and demod history in place — a reset that was not a reset — and
`Counters.Dibits` was never incremented, so every activity line read
`dibits=0`. `TestDeafTier2TapIsResetAtItsDecodingLevel` fails against the
old engine, which never reset a Tier II receiver at all.

## The −20 kHz emitter and the reject

The heal shipped, and the 15 Sep log showed it firing every ~15 s — 21 times
in 6 minutes — with `coarse_offset_hz≈−20100` on a GPSDO-locked X310 that
decodes at 0 Hz. The repeater beacons in
~10 s idle-burst trains
([Part 7]({{ '/blog/deep-dives/dmr-end-to-end-07-idle-beacon-csbk/' | relative_url }}))
with ~5–9 s gaps; in the gap the tap's power stays at −49 dBFS (Fire's drops
5 dB, so it is not the repeater), and the coarse acquirer from Part 9 — a
discriminator mean with no bound and no decode check — froze on whatever
dominated the tap at −20.1 kHz, deafening the channel until the heal reset
it — in every gap.

By frequency alone the acquirer cannot tell a neighbour from a 45 ppm tuner
error — the case #836 built it for — so decode evidence decides.
`healDeafTier2` **confirms** an offset in force for a whole window in which
the channel synced (an engage mid-window is not confirmed by a decoding
train's tail) and **rejects** one the channel never synced under:
`Receiver.RejectCoarseCarrierOffset` reverts a live correction within
`coarseAcqRejectHz` = 500 Hz and ignores future candidates there, across
`Reset`. Pinned by `TestCoarseAcquirerRejectedOffsetNeverReengages` and the
`TestDeafHeal*` family. The WARN is now once per 10 minutes per channel.

What the −20 kHz emitter at ≈442.3675 MHz **is** remains unknown. The 15 Sep
capture does not cover it: the siglab grab was carved at the tuner centre
(441.7 MHz) rather than the typed 442.3875 MHz — a client-side coercion now
replaced by a strict parse plus `requested_center_hz` in the start line — so
it contains neither repeater. Geometry rules out the channelizer (the bin-4
image of Fire lands at +68.75 kHz). A ±100 kHz capture centred on
442.3875 MHz over one full idle cycle will name it. The reject path makes the
tap immune either way; do **not** add a
frequency bound to the acquirer — a 12.5 kHz adjacent channel sits inside any
bound that still serves #836. It is the two-pipelines discipline of
[From the Issue Tracker Part 22]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }})
again: only an A/B on one stream could say which arm to fix.

### How the bin edge shaped the Go code

- **No critically-sampled mode survives.** `ChannelizerBank` builds
  `channelizer.NewOversampled` unconditionally and reads `binOutRateHz`.
- **Pins are tones and captures, never grants.** The flatness test asserts
  frequency and level per residual.
- **Health thresholds are relative and per channel.** `decodeDbFS` is learned
  from the channel's own syncs, `acqConfirmedHz` from a synced window.
- **The receiver exposes evidence hooks, not policy.** `tier2Diag` and
  `coarseRejecter` are small interfaces the engine type-asserts.

## Where this goes next

Everything so far has produced dibits, grants and superframes. The next part
follows a DMR superframe into the vocoder — and to a bug that made every
utterance after a pause start 24 dB too quiet.
[Part 11]({{ '/blog/deep-dives/dmr-end-to-end-11-ambe2-silence-frames/' | relative_url }})
diffs the AMBE+2 3600×2450 decoder frame by frame against two mbelib
lineages and calibrates the unvoiced band.

## FAQ

**Why did one DMR tap on a wideband channelizer decode and its neighbour go deaf?**
Two causes. The repeater sat 0.48 bins off a bin centre, where the
critically-sampled prototype is −6 dB and folds the part of the channel past
the edge back onto itself. Separately, the live receiver chain latches deaf
for ~3-minute stretches; that cause is still open, and the engine now heals
it by reset.

**What does `channelizer.Oversampled` change?**
Each bin is emitted at 2·Fs/M instead of Fs/M, with the prototype cutoff at
0.75·Fs/M and the per-bin rotation an M/2-step structure needs. The whole
±0.5-bin span is flat within [−1, +0.5] dB, so a tap on a bin edge decodes
like one at centre, at double the per-tap fine-tune cost.

**When does the wideband engine reset a DMR receiver?**
After `tier2DeafHealWindows` = 3 consecutive diagnostics windows with no
sync, FEC pass or beacon while the channel's power is within 6 dB of the
level it last synced at. An idle repeater falls well below that level and is
never reset.

**What is `coarse_offset_rejected=true` in the deaf-tap WARN?**
The coarse carrier acquirer had frozen on an offset the channel never decoded
under — on 15 Sep a −20.1 kHz neighbour in every idle gap — so the heal
reverted the correction and told the acquirer to ignore candidates within
500 Hz of it, across future `Reset`s. An offset the channel synced under for
a whole window is confirmed instead.

**Is the −20 kHz emitter identified?**
No. The 15 Sep capture was carved at the tuner centre, not the typed
442.3875 MHz, so it contains neither repeater nor the emitter; channelizer
geometry rules the bank out. A ±100 kHz capture centred on 442.3875 MHz over
one beacon train and gap will name it.

## Series navigation

**Part 10 of 14** · ←
[Part 9: Direct Mode — The Gaps That Blinded the Receiver]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
· Next →
[Part 11: AMBE+2 Voice & the Silence-Frame Bug]({{ '/blog/deep-dives/dmr-end-to-end-11-ambe2-silence-frames/' | relative_url }})
