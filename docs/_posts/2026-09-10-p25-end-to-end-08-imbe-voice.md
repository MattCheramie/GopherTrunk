---
title: "P25 End to End, Part 8: IMBE Voice — From LDU to WAV"
description: How P25 Phase 1 voice travels — the 1728-bit LDU carrying nine IMBE frames woven between link-control words, the Golay and Hamming FEC that turns 144 channel bits into 88, and the composer chain in GopherTrunk that gates, decodes and records a call.
category: deep-dives
keywords: p25 imbe decoder, p25 ldu1 ldu2 structure, imbe 144 channel bits, p25 voice frame extraction, golay hamming imbe fec, p25 link control talkgroup, pure go imbe vocoder, p25 voice recording pipeline, gophertrunk p25
tags: [p25-end-to-end, p25, imbe, vocoder, voice, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "P25 End to End"
series_part: 8
---

*Part 8 of **P25 End to End**, a 14-part deep dive that follows North
America's dominant trunking protocol through GopherTrunk — from a raw C4FM
carrier to recorded, named, multi-site voice.
[Part 7]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }})
split a Phase 2 carrier between two AMBE+2 slots; this part returns to the
voice most P25 systems actually carry. A Phase 1 traffic channel streams
**LDUs** — 1728-bit units carrying nine IMBE frames each, with the call's
own metadata woven between them — and this part follows one from frame
sync to WAV, including the layout bug that decoded exactly two of nine
voice frames while every test stayed green.*

> **TL;DR:** Phase 1 voice arrives as alternating **LDU1/LDU2** units:
> 1728 bits = FS + NID + **9 IMBE frames × 144 channel bits** + 240 bits
> of Link Control (LDU1) or Encryption Sync (LDU2) + 32 bits of low-speed
> data + 24 status symbols interleaved every 70 bits
> (`internal/radio/p25/phase1/ldu.go`). Each 144-bit frame deinterleaves,
> descrambles, and FEC-decodes — Golay(23,12) ×4, Hamming(15,11) ×3, one
> unprotected vector — to **88 information bits**
> (`internal/voice/imbe`), which the pure-Go IMBE vocoder renders as
> 160 PCM samples at 8 kHz per 20 ms frame. The composer chain
> (`internal/voice/composer/p25p1_voice.go`) gates audio by the LC
> talkgroup, splits files on terminators, and only trusts a call's AFC
> measurement past `minAutotuneLDUs = 5`.

**Key takeaways**

- **An LDU is a superframe, not a frame.** Nine 20 ms voice frames — 180 ms
  of audio — travel with the call's own signalling threaded between them,
  so a late-tuning receiver learns who is talking from the voice channel
  itself.
- **The field layout is sourced, not derived.** The interleaving order of
  voice, LC and LSD inside the 1680-bit payload came from an independent
  decoder (szechyjs/dsd); the previous, plausible layout round-tripped
  green while only u_0 and u_8 decoded on air (issue #489 follow-up).
- **FEC strength follows bit sensitivity.** Golay(23,12) guards the four
  most-perceptible parameter vectors, Hamming(15,11) the next three, and
  the least-sensitive seven bits ride unprotected — a budget, not an
  oversight.
- **Voice frames flow even when FEC loses.** An uncorrectable vector still
  yields a (degraded) frame plus an error count the decoder's adaptive
  smoothing consumes — and the counters (`uncorrectable_ldus`,
  `gated_ldus`) are the operator's garbled-audio instrument.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| LDU structure | 1728-bit layout constants + compile-time sum check | `internal/radio/p25/phase1/ldu.go` |
| Status symbols | strip 24 × 2 bits interleaved every 70 payload bits | `ldu.go` (`StripStatusSymbols`) |
| Voice extraction | 9 × 144-bit frames → 11-byte decoded frames | `ldu.go` (`ExtractVoiceFramesDetailed`) |
| LDU framing | dibit stream → complete 1728-bit LDUs | `ldu_assembler.go` (`LDUAssembler`) |
| IMBE channel FEC | deinterleave + descramble + Golay/Hamming, 144→88 | `internal/voice/imbe/channel.go` (`DecodeChannelToFrame`) |
| The voice chain | gating, telemetry, ES publish, autotune | `internal/voice/composer/p25p1_voice.go` (`runP25Phase1VoiceChain`) |
| Vocoder mapping | protocol `"p25"` → pure-Go `imbe`, 8 kHz PCM | `internal/voice/recorder.go` (`DefaultVocoderForProtocol`) |

## In this post

- **The LDU superframe** — nine voice frames and the metadata between them.
- **The layout that round-tripped green and decoded wrong** — issue #489's lesson.
- **From 144 bits to 88** — the IMBE channel-coding inverse, surveyed.
- **The composer chain** — gating, terminators, and the telemetry that names bad audio.
- **What LDU2 carries** — encryption sync, and the out-of-set guard.

## The LDU superframe

After [Part 2]({{ '/blog/deep-dives/p25-end-to-end-02-sync-nid-lock/' | relative_url }})'s
frame sync and NID, a voice transmission is a run of LDUs — LDU1 and LDU2
alternating, each 1728 bits, each 180 ms. The budget is fixed by
TIA-102.BAAA and pinned in `ldu.go` with a compile-time check that the
fields still sum to 1728:

| Field | Bits | Carries |
|---|---|---|
| Frame sync | 48 | the FSW from Part 2 |
| NID | 64 | NAC + DUID (BCH-protected) |
| Voice | 9 × 144 | IMBE frames u_0..u_8, 20 ms each |
| LC or ES | 240 | Link Control (LDU1) / Encryption Sync (LDU2) |
| Low-speed data | 32 | 2 cyclic codewords |
| Status symbols | 24 × 2 | trunking-layer signalling, interleaved |

The status symbols come out first: two bits after every 70 payload bits,
24 times (`StripStatusSymbols`), leaving a 1680-bit payload. What remains
is not voice-then-metadata but a deliberate weave — u_0 and u_1
back-to-back, then a 40-bit LC/ES block after each of u_1 through u_6,
both 16-bit LSD blocks together between u_7 and u_8. The weave is the
design: link control arrives in six pieces spread across 180 ms, so a
receiver that tunes in mid-call (which a trunking scanner *always* does)
assembles who-is-talking-to-whom from the voice channel itself, and a
burst error takes a slice of everything rather than all of one thing.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="The 1680-bit LDU payload after status-symbol removal: nine 144-bit IMBE voice frames with six 40-bit link-control or encryption-sync blocks woven after frames two through seven and two 16-bit low-speed-data blocks before the final frame">
  <text x="30" y="26" fill="currentColor" font-size="11" font-weight="bold">LDU payload (1680 bits, status symbols stripped)</text>
  <!-- row 1 -->
  <rect x="30" y="40" width="44" height="26" fill="none" stroke="var(--fg-muted)"/>
  <text x="52" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="9">FS+NID</text>
  <rect x="74" y="40" width="60" height="26" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="104" y="57" text-anchor="middle" fill="var(--accent)" font-size="9">u_0</text>
  <rect x="134" y="40" width="60" height="26" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="164" y="57" text-anchor="middle" fill="var(--accent)" font-size="9">u_1</text>
  <rect x="194" y="40" width="26" height="26" fill="none" stroke="currentColor"/>
  <text x="207" y="57" text-anchor="middle" fill="currentColor" font-size="8">LC1</text>
  <rect x="220" y="40" width="60" height="26" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="250" y="57" text-anchor="middle" fill="var(--accent)" font-size="9">u_2</text>
  <rect x="280" y="40" width="26" height="26" fill="none" stroke="currentColor"/>
  <text x="293" y="57" text-anchor="middle" fill="currentColor" font-size="8">LC2</text>
  <rect x="306" y="40" width="60" height="26" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="336" y="57" text-anchor="middle" fill="var(--accent)" font-size="9">u_3</text>
  <rect x="366" y="40" width="26" height="26" fill="none" stroke="currentColor"/>
  <text x="379" y="57" text-anchor="middle" fill="currentColor" font-size="8">LC3</text>
  <rect x="392" y="40" width="60" height="26" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="422" y="57" text-anchor="middle" fill="var(--accent)" font-size="9">u_4</text>
  <rect x="452" y="40" width="26" height="26" fill="none" stroke="currentColor"/>
  <text x="465" y="57" text-anchor="middle" fill="currentColor" font-size="8">LC4</text>
  <rect x="478" y="40" width="60" height="26" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="508" y="57" text-anchor="middle" fill="var(--accent)" font-size="9">u_5</text>
  <rect x="538" y="40" width="26" height="26" fill="none" stroke="currentColor"/>
  <text x="551" y="57" text-anchor="middle" fill="currentColor" font-size="8">LC5</text>
  <!-- row 2 -->
  <rect x="30" y="86" width="60" height="26" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="60" y="103" text-anchor="middle" fill="var(--accent)" font-size="9">u_6</text>
  <rect x="90" y="86" width="26" height="26" fill="none" stroke="currentColor"/>
  <text x="103" y="103" text-anchor="middle" fill="currentColor" font-size="8">LC6</text>
  <rect x="116" y="86" width="60" height="26" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="146" y="103" text-anchor="middle" fill="var(--accent)" font-size="9">u_7</text>
  <rect x="176" y="86" width="20" height="26" fill="none" stroke="var(--fg-muted)"/>
  <rect x="196" y="86" width="20" height="26" fill="none" stroke="var(--fg-muted)"/>
  <text x="196" y="103" text-anchor="middle" fill="var(--fg-muted)" font-size="8">LSD</text>
  <rect x="216" y="86" width="60" height="26" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="246" y="103" text-anchor="middle" fill="var(--accent)" font-size="9">u_8</text>
  <text x="300" y="97" fill="var(--fg-muted)" font-size="10">9 voice frames × 144 bits = 180 ms of audio;</text>
  <text x="300" y="111" fill="var(--fg-muted)" font-size="10">LC/ES split into six 40-bit blocks between them</text>
  <!-- annotations -->
  <text x="30" y="146" fill="currentColor" font-size="10" font-weight="bold">the trap: the old layout put LC1 between u_0 and u_1</text>
  <text x="30" y="162" fill="var(--fg-muted)" font-size="10">— shifting u_1..u_7 by 40 bits. Round-trip tests passed (encoder shared the</text>
  <text x="30" y="178" fill="var(--fg-muted)" font-size="10">error); on air only u_0 and u_8 decoded. Layout now pinned to szechyjs/dsd</text>
  <text x="30" y="194" fill="var(--fg-muted)" font-size="10">(process_p25_ldu1) + a real-air capture test — issue #489 follow-up.</text>
</svg>
<figcaption>The LDU weave: nine IMBE frames with link control threaded between them — and the 40-bit layout shift that once silenced seven of the nine.</figcaption>
</figure>

## The layout that round-tripped green and decoded wrong

The figure's annotation deserves its paragraph, because it is this
series' recurring villain caught in the act. The original `lduVoiceOffsets`
table placed an LC/ES block between u_0 and u_1 (and the LSD between u_6
and u_7) — plausible, tidy, and wrong: every offset from u_1 to u_7 was 40
bits off. GopherTrunk's own `InjectStatusSymbols`/encode side used the
same table, so **every round-trip test passed** while on real air only u_0
and u_8 — the two frames whose offsets happened to coincide — produced
audio. Recordings existed, contained mostly garble, and nothing failed.

The fix (issue #489 follow-up) re-sourced the layout from an independent
implementation — szechyjs/dsd's `process_p25_ldu1` read order — and the
current `ldu.go` documents the cumulative bit table field by field, with
`ldu_realair_test.go` holding it against a real capture. Same lesson as
Part 3's SCCB byte offset and Part 7's scrambler ordering: **a
self-consistent codec validates its assumptions against nothing.** Pin
layouts with literal vectors from an implementation that has decoded real
air.

## From 144 bits to 88

Each extracted 144-bit voice frame then runs the IMBE channel-coding
inverse (`internal/voice/imbe`), three layers deep — deinterleave
(§7.5), descramble (§7.4, a PRBS keyed off u_0, which itself rides
unscrambled), then per-vector FEC:

```go
// internal/voice/imbe/channel.go (shape)
// IMBE 4400 channel coding maps 88 information bits to 144 channel bits:
//   u_0..u_3  23 bits each  Golay(23,12,7)   — most sensitive parameters
//   u_4..u_6  15 bits each  Hamming(15,11,3)
//   u_7        7 bits       no FEC           — least-sensitive bits
func DecodeChannel(channel []byte) ([]byte, int, error) {
    /* … per-vector Golay/Hamming decode, summing corrected bits … */
    // ErrUncorrectable when a vector exceeds its radius — the partially
    // recovered info bits are STILL returned so callers can log +
    // frame-repeat upstream rather than drop 20 ms of audio.
}
```

The 88 recovered bits are the vocoder's model parameters — fundamental
frequency, voicing decisions, spectral amplitudes — and the pure-Go
decoder synthesizes each frame into **160 PCM samples at 8 kHz** (20 ms).
The deep math has its own series:
[what a vocoder is]({{ '/blog/deep-dives/voice-coding-01-what-is-a-vocoder/' | relative_url }}),
[the MBE model]({{ '/blog/deep-dives/voice-coding-02-the-mbe-model/' | relative_url }}),
[the IMBE decode]({{ '/blog/deep-dives/voice-coding-04-imbe-decode/' | relative_url }})
and [the FEC + deinterleave layer]({{ '/blog/deep-dives/voice-coding-05-imbe-fec-deinterleave/' | relative_url }})
own it. Two facts matter here. First, the error *count* travels with the
frame: `ExtractVoiceFramesDetailed` reports per-frame corrected bits, and
the recorder hands them to the decoder's adaptive smoothing so a noisy
channel gets gentler synthesis instead of harsh artifacts. Second, the
honest one-liner from the project's vocoder-conformance work: measured
against a reference decode of identical frames, GopherTrunk's IMBE shows a
mild high-band energy deficit — the same direction as the measured AMBE+2
3600×2450 gap, much milder — known and bounded rather than a mystery.

## The composer chain

`runP25Phase1VoiceChain` (`internal/voice/composer/p25p1_voice.go`) is
where a grant becomes a recording. IQ for the granted frequency arrives on
a channel; the chain runs a **±6.25 kHz channel-select front end** (half
the P25 channel spacing — without it, a wideband tap's ±24 kHz span lets
the FM capture effect lock onto a stronger adjacent channel during the
talker's syllable gaps, recording foreign talkgroups and garble), then the
Part 1 receiver, then `LDUAssembler`, and finally a sink that dispatches
per LDU:

- **Terminators end transmissions.** A TDU/TDULC rolls the recording for
  the next over — after harvesting the terminator's link control, since
  Motorola systems carry the talker alias there as often as in LDU1
  (issue #376).
- **The talkgroup gate decides writes.** LDU1's Link Control names the
  talkgroup; the boundary tracker compares it to the grant and drops
  non-matching audio — a shared voice frequency must never
  cross-contaminate a recording. When the LC's outer RS is uncorrectable
  the talkgroup is *untrusted*, so the tracker inherits the last decision
  rather than gating 20 ms of real audio on garbage — the dominant cause
  of fragmented recordings in one field capture.
- **Telemetry names the failure.** The rolling
  `composer: p25p1 decode quality` line carries `ldus`,
  `uncorrectable_ldus`, `gated_ldus`, `corrected_bit_errs` and the
  LC/ES RS counters — so "audio is garbled" resolves to *which layer* is
  losing, and an operator can watch the rate fall as they fix gain
  (issue #356 follow-up).

The chain also feeds the per-dongle **autotune** average — each call's AFC
residual estimates the tuner's frequency error — but only from calls that
earn trust: `minAutotuneLDUs = 5` (roughly a second of voice) plus a <10%
uncorrectable rate, because a short or noisy call yields a data-dependent
snapshot that would drag the correction toward the kilohertz. A threshold
worth knowing when a marginal system's calls run four LDUs long — that is
also [Part 12]({{ '/blog/deep-dives/p25-end-to-end-12-weak-signal-gap/' | relative_url }})'s
opening symptom. Decoded frames land through the recorder, whose default
map sends protocol `"p25"` to the pure-Go `imbe` vocoder
(`DefaultVocoderForProtocol`, `internal/voice/recorder.go`), and the
[recording layer]({{ '/blog/deep-dives/recording-streaming-03-assembling-a-call/' | relative_url }})
takes it from PCM to WAV on disk.

## What LDU2 carries

LDU2's 240-bit slot holds **Encryption Sync** instead of link control:
Message Indicator, Algorithm ID, Key ID. The chain parses it every LDU2
but publishes a bus event only when something changed — ALGID/KID/MI are
stable within a call, and re-announcing them every 180 ms is noise. One
guard is worth flagging now: an ES whose outer RS "succeeds" but yields an
Algorithm ID outside the TIA-102 set is a residual-error mis-decode, and
GopherTrunk drops it rather than surface a phantom key (issue #924) —
downstream, a fake algorithm is indistinguishable from a real one.
Everything else about the encryption story — where the flags live, the
four-layer path to the call log, and what policy does with them — is
[Part 9]({{ '/blog/deep-dives/p25-end-to-end-09-encryption/' | relative_url }})'s
whole subject.

### How the superframe design shaped the Go code

- **Constants that add up, or don't compile.** `ldu.go` closes with a
  compile-time identity over the field widths, so a field edit that breaks
  the 1728-bit budget fails at build, not on air.
- **Extraction returns partial results.** Uncorrectable vectors still
  yield frames plus error counts — the audio chain degrades instead of
  gapping, and the counts feed both smoothing and telemetry.
- **Metadata outruns the gate.** Talker aliases, source IDs and ES are
  surfaced even for gated audio, because identity is worth learning from a
  call you chose not to record.
- **Trust thresholds are named constants.** `minAutotuneLDUs`,
  `qualityLogEveryLDUs`, the RS-uncorrectable counters — every judgment
  call in the chain is a greppable number with a comment, not folklore.

## Where this goes next

LDU2 just introduced the encryption fields;
[Part 9]({{ '/blog/deep-dives/p25-end-to-end-09-encryption/' | relative_url }})
follows them the whole way: where "encrypted" actually lives on the air
across HDU, LDU2 and Phase 2's MAC, the postmortem of a flag and its
metadata separated by four layers, and what GopherTrunk's policy engine —
follow, log metadata, or skip — does with a call it cannot listen to.

## FAQ

**Why does P25 voice come in 180 ms chunks?**
The LDU is the framing unit: nine 20 ms IMBE frames plus the signalling
overhead amortised across them. GopherTrunk's receiver emits complete
LDUs (`LDUAssembler`), so voice latency through the decode path is
bounded by LDU assembly plus synthesis — well under real-time for live
listening.

**What is the difference between LDU1 and LDU2?**
Only the 240-bit metadata slot: LDU1 carries Link Control (talkgroup,
source, service options), LDU2 carries Encryption Sync (MI/ALGID/KID).
Voice layout is identical, and they alternate for the duration of a
transmission so both kinds of metadata repeat about every 360 ms.

**Why is my P25 recording missing pieces of a conversation?**
Check `gated_ldus` in the decode-quality log line. A high count means the
in-band talkgroup didn't match the grant — adjacent-channel bleed on a
wideband tap or a shared voice frequency — so the composer correctly
dropped that audio. `frames − gatedLDUs` approximates what actually
reached the WAV.

**Does GopherTrunk use mbelib or DVSI hardware for IMBE?**
Neither by default: `internal/voice/imbe` is a clean pure-Go IMBE 4400
decoder, the sole backend in default builds, mapped from protocol `"p25"`
by the recorder. The `.imb`/`.amb` sidecar option (`recordings.mbe_files`)
writes DSD-FME-compatible frame files if you want to A/B against another
decoder — the interop is frame-exact, per `docs/vocoders.md`.

**Can GopherTrunk decode encrypted P25 voice?**
No — like SDRTrunk, it identifies encryption (ALGID, KID, MI) and applies
policy, but does not decrypt. What it does with that identification —
skip, record-silent, or follow with metadata — is Part 9.

## Series navigation

**Part 8 of 14** · ←
[Part 7: Phase 2 TDMA — Two Voices per Carrier]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }})
· Next →
[Part 9: Encryption Signalling — Flags, Metadata & Policy]({{ '/blog/deep-dives/p25-end-to-end-09-encryption/' | relative_url }})
