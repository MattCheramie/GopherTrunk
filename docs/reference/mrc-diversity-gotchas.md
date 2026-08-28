---
slug: mrc-diversity-gotchas
title: MRC diversity gotchas
entry_type: term
category: fn-diagnostics
description: "MRC diversity gotchas are the calibration truths that decide whether two-antenna combining helps or hurts: gate on coherence not dBFS, remove DC before correlating, align inter-branch delay, and read branch_phase_deg to learn your hardware class."
keywords: mrc, diversity combining, coherence gate, dbfs gate, dc offset correlation, inter-branch skew, branch alignment, branch_phase_deg, tracking calibrator, pre-combine capture, x310, twinrx
aka: [diversity combining gotchas, MRC calibration gotchas]
infobox:
  - { label: Type, value: DSP + calibration facts }
  - { label: Key rule, value: "Gate on coherence, never on absolute dBFS" }
  - { label: Trap, value: A common DC offset fakes perfect coherence }
  - { label: Field instrument, value: branch_phase_deg — constant vs walking }
see_also: [maximal-ratio-combining, coherence, fractional-delay-filter, interference-rejection-combining, channel-estimation, dc-offset, usrp-soapyremote-notes, signal-signatures]
related_reading:
  - { title: "The Analog Edge, Part 11: Two Antennas — Diversity & MRC From the Operator's Seat", url: /blog/tutorials/analog-edge-11-diversity-mrc/ }
  - { title: "Weak-Signal Engineering, Part 10: MRC Calibration", url: /blog/deep-dives/weak-signal-engineering-10-mrc-calibration/ }
  - { title: "Weak-Signal Engineering, Part 11: Tracking MRC", url: /blog/deep-dives/weak-signal-engineering-11-tracking-mrc/ }
  - { title: "The Analog Edge, Part 13: Coherence, Not dBFS", url: /blog/tutorials/analog-edge-13-coherence-not-dbfs/ }
cite_urls:
  - https://github.com/MattCheramie/GopherTrunk/issues/1062
---

**MRC diversity gotchas** are the calibration truths GopherTrunk learned wiring
[maximal-ratio combining](/reference/maximal-ratio-combining/) into a real two-antenna rig
(an X310 with TwinRX daughterboards, over three weeks of operator field logs). Each one was
paid for with a session where the combiner confidently did the wrong thing — and each now
has a regression test and a diagnostic signature.

## Gate on coherence, never on absolute level

The first calibration gate was "reference branch above −40 [dBFS](/reference/dbfs/)" — and
an operator raised front-end gain from 65 dB to 82 dB purely to push a number past a
software constant (landing at −39.8 dBFS, clearing it by 0.2 dB). Any absolute-power gate
is a gain-staging trap that re-fires on the next front end. The scale-invariant question is
measured [coherence](/reference/coherence/): `|ρ| = γ/(1+γ)` reads directly as per-branch
SNR, and the thing calibration actually needs bounded — the channel estimate's phase
error — is computable from ρ and the window length as `√((1−ρ²)/(2Nρ²))`. GopherTrunk's
gates now bound that projected error (defaults 0.10/0.16 rad), which also cured a subtler
bandwidth trap: wideband ρ is diluted by every hertz of noise-only bandwidth
(`ρ_wb ≈ ρ_ch·√(f₀f₁)`), so a fixed 0.50 threshold had made "coherent enough" depend on the
configured capture bandwidth. On one capture the control channel decoded 1,425 CRC-clean
BSCH from a branch while wideband ρ sat at ~0.16 and the old gate refused to calibrate —
with a WARN whose advice ("raising RF gain will not help") was exactly wrong in that regime.

## DC removal in the correlator is load-bearing

Both receivers of one front end share [LO leakage](/reference/dc-offset/). An uncentred
correlator fed two branches of *independent noise* plus a common DC term reports
`|ρ| → 1` and freezes the branch gain at the ratio of the DC offsets — confidently
calibrating on nothing, in exactly the weak-signal regime the gate exists to protect.
Subtract each branch's mean before every cross-correlation
(pinned by `TestCrossStatsRejectsCommonDCOffset`).

## A delay is not a gain

The biggest single finding: branch 0 lagged branch 1 by a **constant 2.60 samples**
(13 µs at 200 kS/s), and a scalar weight cannot represent a delay — combining the skewed
branches decoded **22% fewer** CRC-clean BSCH than the best branch alone (886 vs 1,142).
MRC was hurting. The signature is unmistakable once known: per-frequency coherence ~0.99
while broadband ρ dilutes to ~0.78, and no amount of gain or tracking improves it. The
driver now measures the skew per stream and delays the early branch through an
interpolating [fractional-delay filter](/reference/fractional-delay-filter/); the identical
combine after alignment scored exactly 1,142. The replay harness prints
"inter-branch delay: other lags ref by N±f samples".

## Read `branch_phase_deg` to learn your hardware class

The MRC health log's `branch_phase_deg` is the no-capture field instrument. **Constant**
phase ⇒ a shared-LO front end (single-chip AD9361 class): a one-shot calibration is
correct — use `mrc-static`. **Walking** phase ⇒ independent PLLs per branch (TwinRX class:
frequency-locked to a common reference, random relative phase per lock, drifting after):
a frozen constant decays over minutes — use tracking `mrc`. The 17 Aug capture measured a
~145° inter-branch phase walking only −0.22°/s: coherent enough that static works within
one session, drifty enough that tracking is the right default.

## Keep the anchor and the applied gain honest

Details that each produced a silent wrong state: the reference branch was once re-picked by
argmax on *every* datagram while uncalibrated, so a ~1 dB crossover between healthy
receivers swapped the phase anchor mid-stream; a control-channel retune cleared the anchor
and let it flip mid-run; and divergence bounds gated only the *proposed* gain, letting the
applied gain random-walk to −34 dB with `calibrated=true` still asserted. The rules that
came out: the anchor survives rearm/retune, every anchor change is logged, a rejected
window **holds** the previous gains (falling back to passthrough mid-stream is itself a
huge phase step), and the applied gain is floored. A calibration that locks once and then
holds every window afterwards now WARNs — one field log had `updates` frozen for eleven
minutes while the line said INFO.

## Capture pre-combine, or the capture answers nothing

The combiner lives in the driver, so every downstream tap — `baseband.auto_record`, iqtap,
scopes — records a stream with one combiner already baked in, which can never be replayed
through another. `diversity_capture` taps both branches straight after de-interleave into
one headerless [cs16](/reference/cs16-format/) file per branch plus a metadata sidecar, and
enforces the alignment invariant: a datagram missing any branch is dropped from *both*
files, never written short, because one short write silently desynchronises the pair.
Verdicts come from the replay harness's decode arms scored by CRC yield — never from EVM.

## Symptom table

| Symptom | Looks like | Actually | Fix / check |
|---|---|---|---|
| Calibration never engages on a decodable signal | Weak signal, bad antenna | Fixed ρ threshold diluted by capture bandwidth | Phase-error gates; compare narrowband vs wideband ρ ([#1062](https://github.com/MattCheramie/GopherTrunk/issues/1062)) |
| Perfect coherence, garbage gains, weak signal | Great link | Common DC offset faking ρ→1 | DC-remove in the correlator |
| MRC decodes *worse* than one branch | "MRC doesn't work" | Constant inter-branch delay | Look for per-frequency ρ ≈ 1 vs low broadband ρ; align first |
| Coherence stuck ~0.3–0.5, tracking never helps | Calibration bug | Wideband-scalar limit: antennas metres apart, per-carrier phases differ | Known limitation; per-channel combining after the DDC is the real fix |
| Operator raises gain to satisfy the combiner | Procedure | A software constant being gamed | Any absolute-dBFS gate is a bug; gate on coherence |

## Provenance

- [#1062](https://github.com/MattCheramie/GopherTrunk/issues/1062) — the IRC RFC and the blind-estimate contamination finding.
- Operator field logs, 17–19 Aug (X310 / TwinRX): the dBFS gain-staging trap, bandwidth-diluted ρ, the 2.60-sample skew, anchor flips, the frozen-`updates` WARN, and the antenna-swap test that pinned a branch deficit to feedline rather than receiver.
