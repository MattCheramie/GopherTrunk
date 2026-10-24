---
title: "The Legacy Family End to End, Part 14: The Verification Ladder — Where Each Protocol Stands and How to Move It"
description: "The series' closing table — eight legacy protocols placed on four verification rungs, each with the tests that pin it, the skip-gated harness waiting for a file, and the exact capture that would move it — followed by the contributor path: gophertrunk capture, the .metadata.json sidecar, gophertrunk test, and the samples/ drop that makes a dormant real-air test run."
category: deep-dives
keywords: gophertrunk verification ladder, on-air verified vs capture pinned, how to contribute an iq capture, gophertrunk capture command, metadata.json sidecar sample_rate_hz center_freq_hz, gophertrunk test acceptance criteria, samples nxdn metadata, skip-gated real air test, smartnet edacs ltr mpt1327 verification status, nxdn dpmr dstar ysf capture needed, gophertrunk legacy family
tags: [legacy-family-end-to-end, verification, captures, testing, smartnet, nxdn, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Legacy Family End to End"
series_part: 14
---

*Part 14 of **The Legacy Family End to End**, a 14-part deep dive through
the protocols the P25, DMR and TETRA series left out, with each part naming
the verification rung its protocol stands on.
[Part 13]({{ '/blog/deep-dives/legacy-family-13-ambe-chains-with-placeholders/' | relative_url }})
closed the per-protocol reading with the three AMBE chains and their
labelled placeholders. This last part does what
[Part 1]({{ '/blog/deep-dives/legacy-family-01-what-legacy-means/' | relative_url }})
promised: every protocol on one ladder, sourced line by line from
`docs/status.md`, `docs/decoder-capture-needs.md`,
`docs/protocol-feature-parity.md`, the `samples/*/README.md` files and the
packages — then the path a contributor takes to move a rung: record,
describe, grade, drop into `samples/`.*

> **TL;DR:** Four rungs: **placeholder** (a stand-in the code labels as
> such), **reference-pinned** (literal constants and vectors from an
> independent decoder or the spec, with tests that catch drift),
> **capture-pinned** (a real-air recording replays through the production
> path under a test or harness), **on-air verified** (a live run on a
> real rig confirms it — the #764/#771 bar). None of the eight legacy
> protocols is on-air verified. SmartNet, EDACS and the NXDN control
> channel are reference-pinned; LTR and dPMR control are spec-derived;
> MPT 1327 decodes two real audio samples through a manual harness; NXDN
> has a skip-gated real-air test (`TestDaemonCCDecodesNXDNRealAir`) and a
> replay harness (`TestReplayNXDNRealCapture`) waiting for a `.cfile` +
> `.metadata.json`; NXDN, dPMR and D-STAR voice are placeholders; YSF's
> FICH codec is reference-pinned but unwired and its voice absent. Moving
> any rung is the same four steps: `gophertrunk capture` (which writes
> the sidecar and reports offset, clipping and drops), `gophertrunk test`
> against the sidecar's `expected`, and a `samples/<proto>/` drop that
> wakes the dormant harness.

**Key takeaways**

- **A rung states evidence, not quality.** Reference-pinned code can be
  correct and still unproven; on-air verified means a human on a real rig
  said so.
- **The harnesses exist before the captures do.** NXDN's real-air test
  skips with a message naming the file it wants.
- **The sidecar is the capture.** A raw `.cfile` carries no rate or
  centre; `sample_rate_hz` and `center_freq_hz` let anything replay it.
- **The two eras' gaps differ in kind.** The trunking generation needs
  control-channel IQ; the AMBE modes need clear voice, because the unknown
  is a 72-bit table.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The rungs | synthetic-green / capture-verified / on-air-verified as claim levels | [From Spec to Shipping 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }}), `CLAUDE.md` ("Issue-closing policy") |
| Status of record | what ships, what is bypassed, what is capture-gated | `docs/status.md`, `docs/decoder-capture-needs.md`, `docs/protocol-feature-parity.md` |
| Recording | live SDR → `.cfile`/`cs16`/`wav`/`flac` + `.metadata.json` | `cmd/gophertrunk/capture.go` (`runCapture`, `siglab.WriteMetadata`) |
| Sidecar schema | `protocol`, `sample_rate_hz`, `center_freq_hz`, `expected` | `internal/siglab/metadata.go` (`Metadata`), `verdict.go` (`Acceptance`) |
| Grading | production decode → pass/fail verdict, exit 1 on fail | `cmd/gophertrunk/test.go` (`runSiglabTest`, `siglab.Run`) |
| Dormant harness | one `.cfile` + sidecar in `samples/nxdn/` runs it | `cmd/gophertrunk/integration_cc_nxdn_realair_test.go` |
| Replay instrument | `GT_NXDN_IQ`, `_IQ_RATE`, `_SOFT`, `_AFC`, `_ALLOW_EMPTY` | `cmd/gophertrunk/nxdn_realcapture_test.go` |
| What is kept | binaries ignored, `*.metadata.json` and READMEs kept | `samples/.gitignore` |

## In this post

- **Four rungs, defined** — what each claims and what evidence earns it.
- **The table** — eight protocols, sourced line by line.
- **Reading the table** — the two families' different gaps.
- **Recording a capture** — `gophertrunk capture` and what it tells you.
- **Describing, grading and contributing** — the sidecar, `gophertrunk test`, the `samples/` drop.
- **What verified means, and where the series lands** — the policy and the related series.

## Four rungs, defined

The rule under every part of this series is CLAUDE.md's: a green
synthetic is never proof of an on-air fix, because an encoder and decoder
sharing a wrong constant agree with each other perfectly.
[From Spec to Shipping Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})
turned that into three claim levels; this series adds a bottom rung and
names the middle precisely, because the legacy family mostly lives there:

- **Placeholder.** The code says so. `nxdnAMBEDeinterleave` is "a direct
  sequential split … Replace with the real NXDN VCH interleave table once
  a capture is available".
- **Reference-pinned.** Constants and layouts come from an independent
  decoder or the published spec, and a test holds them as literals so
  drift fails: SmartNet's sync `0xAC`, interleave and XOR masks from OP25
  and trunk-recorder; YSF's puncture `{0, 1, 102, 103}` from MMDVMHost.
- **Capture-pinned.** A real-air recording replays through the production
  path under a test or harness that would fail on regression — the DMR
  and P25 series' `cmd/gophertrunk/testdata/*.cfile` pattern.
- **On-air verified.** A live run on a real rig, confirmed by the
  operator, as the issue-closing policy requires.

A protocol can sit on different rungs for control and voice; the table
keeps them apart.

## The table

| Protocol | Rung today | Evidence in the tree | Real-air harness | Capture needed (source) |
|---|---|---|---|---|
| Motorola Type II | reference-pinned (control); FM voice via the composer | OP25 `rx_smartnet` + trunk-recorder `SmartnetParser` literals; `TestProcessDecodesRealAirFormat`; `TestDaemonCCDecodesMotorola` | none; the #1143 854.5625 MHz Airspy R2 cfile is the named gate | any SmartNet CC IQ (`CLAUDE.md` #1143) |
| EDACS | reference-pinned (control, per `lwvmobile/edacs-fm`); ProVoice **bypassed** | `edacs_bch_mode` on, BCH(40,28,2) in `framing/bch_edacs.go`; `strict_test.go`; `TestDaemonCCDecodesEDACS` | none | 9600-baud CC IQ, none on file (`status.md`: ProVoice "followed and logged but not yet turned into PCM") |
| LTR | spec-derived, synthetic-green (control); FM voice | 41-bit status word (`statusBits`), `ltr_fcs_mode` FCSOn, `ltr_manchester_mode` ManchesterSoft; `strict_test.go`; `TestDaemonCCDecodesLTR` | none | subaudible-data repeater IQ, none on file (`opt-in-features.md`, package docs name "the most-cited public reference") |
| MPT 1327 | real audio decoded, manually; IQ not on file | `samples/mpt1327/MPT1327_423.6_{1,2}.mp3` through `samples/cmd/audio_smoketest`; CWSC tolerance 2, BCH(64,48,2); `process_cwsc_test.go`, `minconfirm_test.go`; `TestDaemonCCDecodesMPT1327` | manual only | optional: ≥ 60 s IQ or 8 kHz audio for the false-positive rate (`decoder-capture-needs.md` Tier 3) |
| NXDN | reference-pinned control (`ViterbiSpec` chain); voice **placeholder** | `nxdn_soft_decision` 26/200 → 151/200 at σ=0.7 (synthetic, `protocol-feature-parity.md`); `TestDaemonCCDecodesNXDN` | `TestDaemonCCDecodesNXDNRealAir` (skip-gated), `TestReplayNXDNRealCapture` (`GT_NXDN_IQ`) | outbound RCCH IQ ≥ 5 s at 48 kHz: ≥ 80 % CAC CRC, SystemID/SiteID/RAN match, lock < 3 s (`samples/nxdn/README.md`); clear voice for the table |
| dPMR Mode 3 | spec-derived control, CSBK FEC absent; voice **placeholder** | FS3 → 80-bit CSBK → `LinearBandPlan`; `TestStrictValidationDropsUnknownMessageType`; `TestDaemonCCDecodesDPMR`; `TestTCHFrameRoundTrip` | none | ≥ 10 s IQ at 48 kHz, clear voice (`decoder-capture-needs.md` item 6) |
| D-STAR | reference-matched polynomials/CRC; shell self-consistent; `FECOff` default; voice **placeholder** | `TestComputeCRCKnownVector` (`0x29B1`); `TestDaemonCCDecodesDStar`, `…FECOn`; `TestDVVoiceBitsRoundTrip` | none; no `samples/dstar/` | ≥ 10 s IQ at 48 kHz, clear voice, replay with `dstar_fec_mode: "on"` (`decoder-capture-needs.md` item 6) |
| YSF | synthetic lock; FICH codec reference-pinned but **unwired**; voice **absent** | `TestFICHOnAirRecoversFromSingleBitFlip`; `TestDaemonCCDecodesYSF` (zero-filled FICH) | none | DN-mode IQ ≥ 10 s, pass 100 % FICH CRC, metric ≤ 4/100 bits at ≥ 12 dB (`samples/ysf/README.md`; audio-only removed) |

Two cells deserve a note. `docs/decoder-capture-needs.md` lists "EDACS,
LTR, Motorola Type II, dPMR control" under "not capture-blocked" because
their FEC is on by default with no outstanding capture *request*; the
rung column records evidence, not requests, and `CLAUDE.md` is explicit
that the SmartNet rebuild's on-air verification is still pending. And
`samples/nxdn/` already holds two IQ WAVs (`NXDN48 IQ.wav`,
`NXDN96 IQ.wav`) without sidecars; the README's only recorded result from
them is a bimodal dibit distribution — 3 / 50 / 3 / 44 % through the
production pipeline on the NXDN96 file — which is why `nxdn_deviation_hz`
exists and why the row still says capture-gated.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="Four horizontal rungs labelled from bottom to top placeholder, reference-pinned, capture-pinned, on-air verified. On the placeholder rung sit NXDN voice, dPMR voice and D-STAR voice. On reference-pinned sit SmartNet, EDACS, NXDN control, YSF FICH and D-STAR polynomials, with LTR and dPMR control marked spec-derived beside it. On capture-pinned sits MPT 1327 audio, marked manual. The on-air verified rung is empty for this family, with P25, DMR and TETRA named there from the other series. Arrows from each lower rung carry the capture that moves it.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">the ladder: where the legacy family stands</text>
  <line x1="60" y1="46" x2="640" y2="46" stroke="currentColor" stroke-width="1.5"/>
  <text x="56" y="49" text-anchor="end" fill="currentColor" font-size="9">on-air verified</text>
  <text x="350" y="40" text-anchor="middle" fill="var(--fg-muted)" font-size="8">none of the eight · (P25 / DMR / TETRA in their own series)</text>
  <line x1="60" y1="96" x2="640" y2="96" stroke="currentColor"/>
  <text x="56" y="99" text-anchor="end" fill="currentColor" font-size="9">capture-pinned</text>
  <text x="350" y="90" text-anchor="middle" fill="currentColor" font-size="8">MPT 1327 (two real audio samples, manual harness — not CI)</text>
  <line x1="60" y1="146" x2="640" y2="146" stroke="currentColor"/>
  <text x="56" y="149" text-anchor="end" fill="currentColor" font-size="9">reference-pinned</text>
  <text x="350" y="130" text-anchor="middle" fill="currentColor" font-size="8">SmartNet · EDACS · NXDN control · YSF FICH codec (unwired) · D-STAR polynomials + CRC</text>
  <text x="350" y="141" text-anchor="middle" fill="var(--fg-muted)" font-size="8">spec-derived beside it: LTR · dPMR control (CSBK FEC absent)</text>
  <line x1="60" y1="196" x2="640" y2="196" stroke="currentColor"/>
  <text x="56" y="199" text-anchor="end" fill="currentColor" font-size="9">placeholder</text>
  <text x="350" y="190" text-anchor="middle" fill="var(--accent)" font-size="8">NXDN voice · dPMR voice · D-STAR voice (sequential-split deinterleave) · YSF voice: absent</text>
  <text x="350" y="212" text-anchor="middle" fill="var(--fg-muted)" font-size="8">what moves a rung: control-channel IQ for the FM era · clear-voice IQ for the AMBE era · a live run for the top</text>
</svg>
<figcaption>Evidence, not quality: reference-pinned code may well be right. The top rung is empty for this family because no reporter has yet run any of these decoders on a real rig and said so.</figcaption>
</figure>

## Reading the table

The two generations fail in different ways. The FM-era trunking
protocols — SmartNet, EDACS, LTR, MPT 1327 — have control channels whose
framing is cheap to pin from references, and all four now ship with
their on-air FEC on by default
([opt-in-features]({{ '/opt-in-features.html' | relative_url }})). What
they lack is IQ: not one of them has a `samples/` directory, a committed
`.cfile`, or a skip-gated real-air test. MPT 1327 is the exception that
proves the shape — its FFSK rides the audio band, so two sigidwiki MP3s
decode "end-to-end today", but through a manual `audio_smoketest`
harness, not CI. SmartNet's case is the sharpest: the decoder was rebuilt
from OP25 and trunk-recorder after the original framing proved fabricated
([From Spec to Shipping 8]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})),
and the one capture that could confirm the rebuild was unreachable from
the development environment.

The AMBE-era modes have the opposite profile. Their physical layers are
the C4FM family's (NXDN, dPMR, YSF) or a close cousin (D-STAR), and the
unknowns are small and specific: a 72-bit interleave table, a frame
carve, a codebook, a FICH schedule. That is why their capture requests
say "clear voice", and why NXDN — the one protocol with a Tier-1 harness
ready — tops `decoder-capture-needs.md`'s priority list.

## Recording a capture

`gophertrunk capture` opens an SDR directly, outside the daemon's pool,
and records raw IQ plus the sidecar the tooling reads:

```text
gophertrunk capture -freq 851062500 -sample-rate 2400000 -seconds 30 \
  -center 851062500 -bandwidth 50000 -format cs16 -protocol nxdn \
  -source "RCCH @ <site>, MMDVMHost log" -out nxdn-cc.raw
```

`-format` takes `u8`, `f32` (a GNU Radio cfile), `cs16`, `wav` or `flac`;
`-center`/`-bandwidth` carve a narrowband slice through the same
`ccdecoder.Downconverter` the daemon uses, so a 50 kHz slice of a
2.4 MS/s grab is a small, shareable file; `-centers` records several
sample-synchronous slices. The command then reports what a replay would discover too late: `captureEffectiveRate` stamps the sidecar with the rate the
hardware actually delivered; the carrier-offset consensus prints the
measured offset and warns above `carrierOffsetWarnHz` (2000 Hz);
`formatClipWarning` flags ADC-rail clipping; a dropped-chunk count warns
that the recording has time gaps; and `captureSampleRateHint` notes when
a rate above 4 MS/s buys nothing for a narrowband protocol. Each line exists because a reporter's capture once failed for that
reason — [Field Notebook 9]({{ '/blog/tutorials/field-notebook-09-capture-lines/' | relative_url }})
reads them one by one.

## Describing, grading and contributing

The sidecar is `siglab.Metadata`, written next to the file as
`<stem>.metadata.json`. Two fields are required because the file cannot
carry them — `sample_rate_hz` and, for the harnesses, `center_freq_hz` —
and `protocol` is what lets `gophertrunk test` build a pipeline:

```json
{
  "protocol": "nxdn",
  "source": "RCCH @ <site>, MMDVMHost log",
  "tool_cross_check": "DSDcc 1.9.5",
  "sample_rate_hz": 50000,
  "center_freq_hz": 851062500,
  "format": "cs16",
  "expected": { "lock": true, "lock_latency_max_sec": 3,
                "lock_fields": { "SystemID": "0x1234", "SiteID": "0x01" } }
}
```

`expected` is `siglab.Acceptance`: `lock`, `lock_latency_max_sec`,
`lock_fields` (hex-tolerant, matched as a subset), `min_grants`,
`max_decode_error_rate`, `max_evm_pct`, `min_snr_db`. `gophertrunk test -capture nxdn-cc.raw` discovers the
sidecar, runs `siglab.Run` through the production pipeline, prints a
verdict, and exits 1 on failure — so a corpus of captures is a CI gate.

The `samples/` drop uses a sibling schema. `samples/nxdn/README.md`
documents `expected.system_id`, `expected.site_id`, `expected.ran` and a
`messages` list, and `integration_cc_nxdn_realair_test.go` reads exactly
that through its own `realairCaptureMetadata` (top-level
`sample_rate_hz` and `center_freq_hz` are shared with siglab's; the
`expected` shapes are not). The test is dormant by design:

```go
// cmd/gophertrunk/integration_cc_nxdn_realair_test.go (shape)
cfilePath, metaPath := findRealairCapture(t, "nxdn")
if cfilePath == "" {
    t.Skipf("samples/nxdn/: no *.cfile present — drop a capture + metadata pair to run this test")
}
// mock SDR at meta.SampleRateHz, control_channels [meta.CenterFreqHz],
// nxdn_viterbi_mode: spec → wait ≤ 3 s for events.KindCCLocked →
// assert LockState.SystemID / SiteID / FrequencyHz against the sidecar
```

Exactly one `.cfile` pair is supported (two is a test error) — so record
`-format f32` for this drop, while `TestReplayNXDNRealCapture` takes
`cs16` — `ran` is parsed but not yet asserted, and the ≥ 80 % CAC CRC rate
is deferred until the control channel surfaces a histogram. `samples/.gitignore` ignores
every capture format and keeps `**/README.md` and `**/*.metadata.json`,
so the committed record of a contribution is the sidecar — the pattern
`samples/p25/p25-450875-cc.metadata.json` already follows. For a baseline
before the daemon test, `TestReplayNXDNRealCapture` takes the same file
as `GT_NXDN_IQ` (cs16; `GT_NXDN_IQ_RATE` default 48000) and prints FSW
hits, CAC totals and CRC yields, with `GT_NXDN_SOFT=1` adding the soft
column, `GT_NXDN_AFC=1` the opt-in AFC, and `GT_NXDN_ALLOW_EMPTY=1`
turning a zero-FSW weak capture into a logged baseline rather than a
failure.

## What verified means, and where the series lands

The last rung is not a test. CLAUDE.md's issue-closing policy, the
subject of
[From Spec to Shipping Part 14]({{ '/blog/deep-dives/from-spec-to-shipping-14-definition-of-verified/' | relative_url }}),
says a close is a claim the problem is gone, earned when a failing-first
regression passes *and* the reporter confirms, and that PRs say `Refs #N`
until then. Every row above is a `Refs`, including those whose code is
almost certainly right. The three sibling series show the top rung reached — the
[P25 playbook]({{ '/blog/deep-dives/p25-end-to-end-14-playbook/' | relative_url }}),
the [DMR playbook]({{ '/blog/deep-dives/dmr-end-to-end-14-playbook/' | relative_url }})
and [TETRA's open questions]({{ '/blog/deep-dives/tetra-end-to-end-14-testing-open-questions/' | relative_url }})
end on captures reporters sent and runs they confirmed — and
[Protocol Decoders 12]({{ '/blog/deep-dives/protocol-decoders-12-testing-decoders-without-radios/' | relative_url }})
describes the synthetic layer every rung stands on. The legacy family's
contribution is the ladder with its lower rungs occupied and the labels
honest: eight protocols, one table, and for each a file that does not
exist yet.

### How the ladder shaped the Go code

- **Harnesses skip loudly.** `t.Skipf` names the pair it wants; CI stays
  green and the ask stays visible.
- **Placeholders are functions, not comments.** One named function per
  unknown, inverse in lock-step, so the capture-driven swap is minimal.
- **Instruments before captures.** `GT_NXDN_*` knobs, offset and clip
  lines on `capture`, per-frame Golay counts — built so the first file
  answers a question on arrival.
- **Sidecars over filenames.** `sample_rate_hz` and `center_freq_hz`
  travel with the file; a wrong rate replays with a shifted symbol clock.

## FAQ

**Which legacy protocols are verified on air in GopherTrunk?**
None of the eight. SmartNet, EDACS and the NXDN control channel are
reference-pinned; LTR and dPMR control are spec-derived; MPT 1327 decodes
two real audio samples through a manual harness; NXDN, dPMR and D-STAR
voice are labelled placeholders; YSF locks on its sync word and decodes
no FICH or voice live. The top rung needs a reporter's run on a real rig.

**How do I record an IQ capture GopherTrunk can replay?**
Use `gophertrunk capture -freq <hz> -seconds <n> -out <file> -protocol
<name>`, with `-center`/`-bandwidth` for a small narrowband slice and
`-format cs16` or `flac`. It writes `<stem>.metadata.json` with the
actual sample rate and centre, and prints the measured carrier offset,
any ADC clipping and any dropped chunks before you leave the site.

**What must the metadata.json sidecar contain?**
`sample_rate_hz` always — a raw `.cfile` carries no rate — plus
`center_freq_hz` for the harnesses and `protocol` for `gophertrunk test`.
`expected` holds the acceptance contract: for siglab, `lock`,
`lock_fields`, `min_grants`, `max_evm_pct` and the like; for a
`samples/<proto>/` drop, the per-protocol shape in that README, such as
NXDN's `system_id`, `site_id` and `ran`.

**What happens when a capture lands in samples/nxdn/?**
`TestDaemonCCDecodesNXDNRealAir` stops skipping: under `go test -tags
integration` it mounts the `.cfile` on a mock SDR, boots the daemon with
`nxdn_viterbi_mode: spec` on the sidecar's centre, waits up to 3 s for
`cc.locked`, and asserts SystemID, SiteID and frequency against the
sidecar. The binary stays git-ignored; the sidecar is committed.

**Why is NXDN the top capture priority?**
Because its harness is ready and nothing has run through it.
`docs/decoder-capture-needs.md` ranks an outbound RCCH capture first; the
pass bar is ≥ 80 % CAC CRC, SystemID/SiteID/RAN byte-match and lock within
3 s, and the same file baselines `nxdn_soft_decision` and `nxdn_afc`
through `TestReplayNXDNRealCapture`.

## Series navigation

**Part 14 of 14** · ←
[Part 13: Three AMBE Chains With Honest Placeholders — What a Capture Would Pin]({{ '/blog/deep-dives/legacy-family-13-ambe-chains-with-placeholders/' | relative_url }})
