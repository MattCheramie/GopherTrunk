---
layout: page
title: DMR voice quality
description: Why DMR (AMBE+2) voice — especially female voices — sounds rough, what was ruled out, and how to verify a fix
nav_group: Reference
---

# DMR voice quality (AMBE+2)

Conventional DMR now decodes end to end, but the decoded voice — **female
voices especially** — can sound rough. This page records what the decode chain
does, a mbelib reconciliation review (what was ruled out and the one genuine
divergence found), the operator levers you can pull today, and the
calibration-gated path to a *verified* fix.

The short version: GopherTrunk's default DMR decoder
([`internal/voice/ambe2`](https://github.com/MattCheramie/GopherTrunk/tree/main/internal/voice/ambe2))
is a clean-room Go re-implementation of **mbelib**'s `ambe3600x2450` path. mbelib
is a reverse-engineered, *approximate* AMBE+2 decoder — it is known to sound
worse than DVSI silicon, and worst on high-pitched (female) voices. So a large
part of what you hear is inherent to the algorithm being mirrored, not a discrete
bug. That said, one genuine divergence from the reference was found (below), and
it is the prime suspect for the female-voice-specific roughness.

## Why female voices are the hard case

A female voice has a higher fundamental ω₀, hence **fewer harmonics** L
(`internal/voice/ambe2/tables2450.go`: L ranges 9…56, and the low-L rows are the
high-pitch ones). Everything in the model that is indexed by harmonic position
`l` relative to `L` therefore behaves differently for small L — most of the
band sits in the "high band" (`8l > L`) where the spectral-amplitude enhancement
acts, and the voiced/unvoiced and phase-dispersion decisions cover proportionally
more of the voice. So a mis-tuned or mis-transcribed high-band step hits female
voices hardest.

## Reconciliation review (mbelib `ambe3600x2450` + TIA-102.BABA §6.2)

### Ruled OUT — these faithfully match the reference

- **Pitch / L / w₀ tables** (`tables2450.go`) — generated from mbelib's
  `ambe3600x2450_const.h`; the b0→(w₀, L) lookup matches.
- **Voiced/unvoiced decode** (`params2450.go:60-72`). The `jl = int(l·16·f0)`
  index and the `dmrVuv[b1][jl]` lookup match mbelib. The clamp to `[0,7]` is a
  defined-behaviour guard; across the real f0 range `jl` spans the full 0…7 of
  the table (e.g. at the highest f0≈0.05, `jl = 0.8·l` reaches 7 only at l≈9), so
  there is **no pathological V/UV collapse** for female frames.
- **§6.3 voiced-phase de-buzz** (`decoder.go:synthFrame`, ~627-664). mbelib does
  `PHIl = PSIl` for `l ≤ L/4` and `PHIl = PSIl + numUv·rand_phase()/L` above it,
  with `rand_phase() ∈ [-π, π]`. GT draws `(-1..1)·π · (numUv/L)` for `l > L/4`
  voiced harmonics — the **same** distribution and scaling. Faithful.

### Found — the one genuine divergence (§6.2 spectral-amplitude enhancement)

[`internal/voice/mbe/enhance.go`](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/voice/mbe/enhance.go)
`EnhanceAmplitudes` is on the **default** decode path (`ambe2/decoder.go:485`,
and `imbe/decoder.go:552` for P25). Its per-harmonic weight ξ drops a factor that
is present in **both** references the code cites (TIA-102.BABA §6.2 *and*
mbelib's `mbe_spectralAmpEnhance`):

```
mbelib / spec:   ξ = 0.96·π·num / ( ω0 · R_M0 · (R_M0² − R_M1²) )
GopherTrunk:     ξ = 0.96·num   / (       R_M0 · (R_M0² − R_M1²) )
                                   └─ missing the  π / ω0  factor ─┘
        num = R_M0² + R_M1² − 2·R_M0·R_M1·cos(ω0·l),   W_l = ξ^0.25, clamped [0.5, 1.2]
```

For DMR, ω₀ ∈ [~0.05, ~0.31] rad, so `π/ω0 ∈ [~10, ~63]`. GT's ξ is therefore
10–63× **smaller** than the spec's; after the `^0.25`, GT's high-band weights are
~1.8–2.8× lower, so they land on the **0.5 (attenuate)** clamp where the spec
would push toward the **1.2 (boost)** clamp. Because the frame is then
energy-renormalised (γ = √(R_M0/Σ)), the *relative* spectral tilt shifts energy
out of the high band into the low band — a duller timbre, and, since small-L
(female) frames have most of their harmonics in that high band, the effect is
**strongest on female voices**. This matches the reported symptom.

`enhance.go`'s own header comment flags this as unfinished: *"spec-tuning to
bit-match mbelib output is part of step 5c (gain calibration)."* So it is a known
TODO, not a deliberate design choice.

The one-line correction (restore the spec/mbelib factor) is:

```go
// internal/voice/mbe/enhance.go, inside EnhanceAmplitudes:
xi := 0.96 * math.Pi * num / (p.W0 * den)   // was: xi := 0.96 * num / den
```

**Why this is not shipped as a default change yet.** It is deliberately left for
the calibration A/B, per this repo's #764/#771 discipline:

1. It cannot be verified to *improve* audio without a reference decode of the
   same frames — mbelib is itself approximate, and matching its (or the spec's)
   number is not proof of better perceived quality.
2. `EnhanceAmplitudes` is shared with the **P25 IMBE** path, so a blind change has
   blast radius beyond DMR.

The calibrate harness below is the gate: with a real reference WAV in place, the
`TestCompareAMBE2` cross-correlation will *rise* if this correction helps and
*fall* if it hurts — decide from that, not from the formula.

> A secondary, lower-confidence note: mbelib additionally multiplies the weight by
> `√M_l` (`Wl = sqrtf(Ml[l]) · powf(...)`), which the spec closed form does not.
> GT (correctly, per spec) omits it. Leave it omitted; it is a mbelib artefact,
> not part of the spec.

## Levers you can pull today (no code change)

These are perceptual, not spec-exact, but cost nothing to try and can meaningfully
warm up / brighten DMR audio while the calibration above is pending:

- **`recordings.enhance`** — the opt-in output chain
  ([`EnhanceConfig`](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/config/config.go)).
  A high-shelf and a louder AGC target help intelligibility:

  ```yaml
  recordings:
    write_raw: true          # also needed for calibration below
    enhance:
      enabled: true
      hpf_hz: 200            # trim rumble
      lpf_hz: 3400           # telephone band
      shelf_hz: 1600         # gentle presence
      shelf_db: 2            # dB of cut above shelf_hz (positive; <=0 disables)
      agc_target: 22000      # louder playback
  ```

- **`ambe2-dmr-warm`** — the warm DMR vocoder variant (a high-shelf `ToneTilt`),
  selected by `recordings.warm_dmr_audio: true` (`WarmDMRAudio`). Softens the
  synthetic "edge" without touching the model decode.

## Verified-fix path (calibration-first)

You already have the input side: every DMR call the daemon records with
`recordings.write_raw: true` writes a `<UTC>_<id>.raw` sidecar of the packed
AMBE+2 frames (7 bytes/frame) next to the `.wav`. Pick a **female-voice** call —
the hard case — and:

1. **Reference-decode the same `.raw`** through an independent AMBE+2 decoder:

   ```sh
   dsd-fme -r <call>.raw -o reference.wav      # 8 kHz mono 16-bit PCM
   ```

2. **Drop both files in and run the A/B** (the test skips until the reference
   lands, so CI stays green today):

   ```sh
   cp <call>.raw    internal/voice/ambe2/testdata/dmr-voice.raw
   cp reference.wav internal/voice/ambe2/testdata/dmr-voice-dsdfme.wav
   go test ./internal/voice/calibrate/ -v -run TestCompareAMBE2
   ```

   Acceptance: `|RMSRatioDb| < 3.0` (loudness) and `PeakXcorr > 0.85` (waveform
   similarity). A clean RMS but low xcorr is the **synthesis** signature — i.e.
   the §6.2 enhancement above, not a gain knob.

3. **A/B the §6.2 correction.** Note the baseline `PeakXcorr`, apply the one-line
   `xi` change, re-run. If xcorr rises (and DMR + P25 both hold or improve), ship
   it; if not, leave the default and keep the correction documented here.

See [Voice calibration](voice-calibration.md) for the full harness reference
(the `cmd/voice-calibrate` CLI, the IMBE path, and the AGC `TargetPeak` knob for
the loudness half).

> **Do not commit real over-the-air public-safety recordings as fixtures.** The
> calibrate testdata dirs are for **local**, operator-provided captures — the
> tests `t.Skip` when the files are absent, which is the intended state in the
> repo.
