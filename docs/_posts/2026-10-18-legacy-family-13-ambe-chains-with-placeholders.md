---
title: "The Legacy Family End to End, Part 13: Three AMBE Chains With Honest Placeholders — What a Capture Would Pin"
description: "NXDN, dPMR and D-STAR voice share one AMBE FEC layer in GopherTrunk — Golay(23,12) over C0 and C1, a C0-seeded C1 keystream, a 49-bit assembly — and differ only in a 72-bit deinterleave table all three ship as a labelled placeholder; what the composer chains do today, how DMR's real table compares, and the three measurements a voice capture would make."
category: deep-dives
keywords: nxdn voice decode ambe+2, dpmr voice ambe 2450, d-star ambe 2400 decode, ambe deinterleave placeholder, golay 23 12 ambe c0 c1, ambe c1 keystream descramble, voice_ambe.go gophertrunk, ambe2-dmr vs ambe2 vocoder, nxdn vch 4x72 frames, dpmr tch 144 dibits, unverified on air voice chain, gophertrunk legacy family
tags: [legacy-family-end-to-end, ambe, nxdn, dpmr, d-star, vocoder, verification, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Legacy Family End to End"
series_part: 13
---

*Part 13 of **The Legacy Family End to End**, a 14-part deep dive through
the protocols the P25, DMR and TETRA series left out, with each part naming
the verification rung its protocol stands on.
[Part 12]({{ '/blog/deep-dives/legacy-family-12-ysf/' | relative_url }})
ended on a protocol with no voice chain at all. This part turns to the
three that have one — NXDN, dPMR Mode 3 and D-STAR — and reads the one
file each shares the shape of, `voice_ambe.go`, side by side. The point is
not that the chains are wrong — nobody knows. The point is that the tree
says what it does not know, in a function marked `PLACEHOLDER`, and that
the measurements which would settle it already exist in the DMR path.*

> **TL;DR:** `internal/radio/{nxdn,dpmr,dstar}/voice_ambe.go` are three
> copies of one AMBE FEC layer: Golay(23,12) over C0 and C1, a C0-seeded
> keystream XORed onto C1 first, and the C0:12 + C1:12 + C2:11 + C3:14 =
> 49-bit `ambe_d` assembly — mbelib's 2450 and 2400 ECC paths share it,
> and DMR's `internal/radio/dmr/voice/ambefec.go` is the same code with
> the real `rW/rX/rY/rZ` schedule from szechyjs/dsd.
> The one protocol-specific piece — how 72 on-air bits map into C0..C3 —
> is `nxdnAMBEDeinterleave`, `dpmrAMBEDeinterleave` and
> `dstarAMBEDeinterleave`, each a **direct sequential split** labelled
> `PLACEHOLDER`, with an encoder inverse kept in lock-step so
> `TestVCHFrameRoundTrip` / `TestTCHFrameRoundTrip` / `TestDVVoiceBitsRoundTrip`
> pass by construction. The composer chains feed the recorder
> through `WriteRawFrameWithErrors`, rendering `"nxdn"`/`"dpmr"` via
> `ambe2-dmr` (3600×2450) and `"dstar"` via `ambe2` (3600×2400). Nothing
> has decoded real air; the first capture pins the table, the geometry
> and the codebook, in that order.

**Key takeaways**

- **The FEC is the codec's, the interleave is the protocol's.** Everything
  in the three files except one function is AMBE's, shared with the
  capture-verified DMR path.
- **A placeholder that round-trips cannot fail from inside.** The split is
  sequential and labelled, inverse in lock-step, so the swap is one
  function — and no test in the tree can fail until a capture does.
- **The composer wiring is production.** Three chains, one front-end
  builder, per-frame Golay counts into the recorder — only the table is
  provisional.
- **DMR already shows what the first capture looks like.** A Golay
  histogram, a b0 continuity score and a per-frame diff against mbelib
  apply here unchanged.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Shared FEC layer | Golay(23,12) C0/C1, C0-seeded C1 keystream, 49-bit assembly | `internal/radio/{nxdn,dpmr,dstar}/voice_ambe.go` (`DecodeVCHFrame`, `DecodeTCHFrame`, `DecodeDVVoiceBits`) |
| The placeholder | sequential C0\|C1\|C2\|C3 split of 72 bits, inverse in lock-step | `nxdnAMBEDeinterleave`, `dpmrAMBEDeinterleave`, `dstarAMBEDeinterleave` |
| The real thing, for contrast | `rW/rX/rY/rZ` from szechyjs/dsd, capture-verified | `internal/radio/dmr/voice/ambefec.go` (`DecodeAMBEFrame`) |
| Frame geometry | NXDN 4×72 in a 144-dibit Info field; dPMR 24 CCH + 144 TCH; D-STAR 72 of 96 | `{nxdn,dpmr}/voice.go`, `{nxdn,dpmr}/traffic.go`, `dstar/voice.go` |
| Composer chains | 48 kHz front end → receiver → traffic tracker → recorder | `internal/voice/composer/{nxdn,dpmr,dstar}_voice.go` |
| Vocoder map | `nxdn`/`dpmr` → `ambe2-dmr`; `dstar` → `ambe2` | `internal/voice/recorder.go` (`DefaultVocoderForProtocol`), `cmd/gophertrunk/runtime.go` |
| Error-aware sink | per-frame corrected-bit count drives the smoother | `errAwareRawSink`, `Recorder.WriteRawFrameWithErrors` |
| What a capture pins | table → geometry → codebook | `docs/decoder-capture-needs.md`, `docs/status.md` |

## In this post

- **One FEC layer, three files** — what AMBE owns and what the protocol owns.
- **The placeholder, exactly** — the sequential split and why its tests cannot fail.
- **Three frame geometries** — where the 72-bit frames sit in each burst.
- **The composer wiring** — front ends, trackers, the error-aware sink, two codebooks.
- **What the first capture pins** — table, geometry, codebook, and the DMR instruments.
- **The #764/#771 rule applied** — what "verified" would mean for each chain.

## One FEC layer, three files

Open the three `voice_ambe.go` files side by side and the diff is almost
entirely identifier prefixes. Each decodes a 72-bit on-air frame into 49 vocoder bits in the same four
steps:

```go
// internal/radio/nxdn/voice_ambe.go (shape) — DecodeVCHFrame
fr := nxdnAMBEDeinterleave(frame)                // 72 bits → C0[24] C1[23] C2[11] C3[14]
c0data, c0errs := framing.GolayDecode23_12(c0cw) // fr[0][1..23]; fr[0][0] = ext parity
ks := nxdnC1Keystream(c0data)                    // pr[i] = (173·pr[i-1] + 13849) & 0xFFFF
for j := 0; j <= 22; j++ { fr[1][j] ^= ks[23-j] }
c1data, c1errs := framing.GolayDecode23_12(c1cw)
// ambe_d = C0:12 + C1:12 + C2:11 + C3:14 = 49 bits
return out, clampErrs(c0errs) + clampErrs(c1errs), nil
```

The keystream is mbelib's `mbe_demodulateAmbe3600x2450Data` sequence —
the D-STAR copy, `dstarC1Keystream`, cites the 2400 variant and notes it
is "identical to the 3600×2450 one". The Golay primitives are the shared
`framing.GolayDecode23_12`/`GolayEncode23_12`. An uncorrectable
sub-vector's −1 is clamped to 0 so the returned count stays a
non-negative sum — the number the composer later hands the recorder.

All of that belongs to the codec. The file comments say so: the FEC "is
a property of the AMBE+2 codec, not of the radio protocol, so it is
identical to `internal/radio/dmr/voice/ambefec.go`". That DMR file is the
control group. Its `DecodeAMBEFrame` runs the same Golay, keystream and assembly,
but its first step reads

```go
// internal/radio/dmr/voice/ambefec.go (shape)
fr[rW[i]][rX[i]] = frame[2*i] & 1     // dibit i, high bit
fr[rY[i]][rZ[i]] = frame[2*i+1] & 1   // dibit i, low bit
```

with `rW`, `rX`, `rY`, `rZ` as four 36-entry tables "verbatim from
szechyjs/dsd" (`dmr_const.h`). That schedule decoded real air —
[DMR End to End Part 11]({{ '/blog/deep-dives/dmr-end-to-end-11-ambe2-silence-frames/' | relative_url }})
diffed the committed #644 clip frame by frame against mbelib — which is
what makes the siblings' one divergent function so visible.

## The placeholder, exactly

The function, identical in all three files but for its name:

```go
// internal/radio/dpmr/voice_ambe.go (shape) — PLACEHOLDER
func dpmrAMBEDeinterleave(frame []byte) [4][24]uint8 {
    var fr [4][24]uint8
    p := 0
    get := func() uint8 { b := frame[p] & 1; p++; return b }
    for j := 0; j < 24; j++ { fr[0][j] = get() } // C0: bits 0..23
    for j := 0; j < 23; j++ { fr[1][j] = get() } // C1: bits 24..46
    for j := 0; j < 11; j++ { fr[2][j] = get() } // C2: bits 47..57
    for j := 0; j < 14; j++ { fr[3][j] = get() } // C3: bits 58..71
    return fr
}
```

Its doc comment is the honest part of the design: "PLACEHOLDER:
a direct sequential split. Replace with the real dPMR TCH interleave
schedule (TS 102 658 §6) once a capture is available to verify against
(the encoder inverse below must be updated in lock-step). Isolated here
so that swap is a one-function change." D-STAR's names DSD's `dW`/`dX`
schedule as the target. Each file opens with the same warning —
"UNVERIFIED ON AIR … the first thing a capture confirms is this table" —
and cites CLAUDE.md's TETRA-CRC lesson "for why the table is not guessed".

Why not guess? Because a guessed table that round-trips is undetectable
from inside. `TestVCHFrameRoundTrip`, `TestTCHFrameRoundTrip` and
`TestDVVoiceBitsRoundTrip` push 128 seeded payloads through `Encode*` and
back; their comments say what they prove — "everything except the
(capture-unknown) on-air interleave table, which round-trips by
construction here". The `*CorrectsErrors` tests flip positions
`{1, 5, 12, 24, 30, 40}`, with a comment that gives the dependency away:
"C0 occupies on-air positions 0..23, C1 24..46 (sequential placeholder
layout)". When the real table lands, those positions move with it. This
is the [self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
made explicit on purpose: the split is not a hypothesis about the air but
scaffolding that lets the Golay, keystream and assembly be tested now,
labelled so nobody mistakes it for a claim.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="Two rows of 72 on-air bits: the shipped placeholder, four contiguous blocks C0 bits 0 to 23, C1 24 to 46, C2 47 to 57, C3 58 to 71; and DMR's real rW rX rY rZ schedule, 36 dibits scattered across C0 to C3. Both feed the same Golay, keystream and 49-bit assembly.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">72 on-air bits → C0 (24) · C1 (23) · C2 (11) · C3 (14)</text>
  <text x="8" y="54" fill="var(--fg-muted)" font-size="8">placeholder</text>
  <text x="8" y="65" fill="var(--fg-muted)" font-size="8">(nxdn/dpmr/dstar)</text>
  <rect x="100" y="40" width="190" height="26" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="195" y="57" text-anchor="middle" fill="var(--accent)" font-size="8">C0 · bits 0..23 (bit 0 = ext parity)</text>
  <rect x="290" y="40" width="182" height="26" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="381" y="57" text-anchor="middle" fill="var(--accent)" font-size="8">C1 · bits 24..46</text>
  <rect x="472" y="40" width="88" height="26" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="516" y="57" text-anchor="middle" fill="var(--accent)" font-size="8">C2 · 47..57</text>
  <rect x="560" y="40" width="110" height="26" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="615" y="57" text-anchor="middle" fill="var(--accent)" font-size="8">C3 · 58..71</text>
  <text x="385" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="8">sequential split, inverse in lock-step — round-trips by construction, labelled PLACEHOLDER</text>
  <text x="8" y="124" fill="var(--fg-muted)" font-size="8">DMR (real)</text>
  <text x="8" y="135" fill="var(--fg-muted)" font-size="8">rW/rX/rY/rZ</text>
  <rect x="100" y="110" width="570" height="26" fill="none" stroke="currentColor"/>
  <text x="385" y="127" text-anchor="middle" fill="currentColor" font-size="8">36 dibits: high bit → fr[rW[i]][rX[i]], low bit → fr[rY[i]][rZ[i]] — scattered across C0..C3 (szechyjs/dsd dmr_const.h)</text>
  <text x="385" y="164" text-anchor="middle" fill="currentColor" font-size="8">both rows → Golay(23,12) on C0, C1 → C0-seeded keystream on C1 → 12+12+11+14 = 49-bit ambe_d</text>
  <text x="385" y="186" text-anchor="middle" fill="var(--fg-muted)" font-size="8">capture-verified: the DMR row (#644 clip vs mbelib) · unverified: the placeholder row</text>
</svg>
<figcaption>Everything after the split is shared and already proven on DMR air. Only the mapping from 72 on-air bits into the four sub-vectors is protocol-specific, and in three of four packages it is a labelled stand-in.</figcaption>
</figure>

## Three frame geometries

The table is not the only unknown. Each protocol's carve of 72-bit frames
from a burst is transcribed from spec, not from a signal.

**NXDN.** `nxdn/voice.go` reads the 144-dibit Information field of a
traffic frame as 288 bits = `VCHVoiceFramesPerFrame` (4) × 72;
`ExtractVCHFrames` skips any frame whose FEC hard-fails. `nxdn/traffic.go`'s
`TrafficChannel` runs its own outbound-FSW detector, collects
`postSyncDibitsTraffic` = 8 LICH + 32 SACCH + 144 Info = 184 dibits per
match, and routes only frames with `ParityOK` and `RFCh == RFChTraffic`;
the SACCH is skipped.

**dPMR.** `dpmr/voice.go` fixes `CCHDibits` 24, `TCHSubframeDibits` 36,
`TCHFramesPerBurst` 4 and `TCHFieldDibits` 144, so `postSyncDibitsTraffic`
= 168 — the 192-dibit frame minus its sync. `dpmr/traffic.go` runs FS1
and FS2 detectors at tolerance 1 (the voice syncs the CC path never
uses), skips the CCH, and hands the TCH to `ExtractTCHFrames`. Its
comment flags "the CCH width" among the spec-transcribed unknowns.

**D-STAR.** [Part 11]({{ '/blog/deep-dives/legacy-family-11-dstar/' | relative_url }})
covered the 96-bit DV frame — 72 voice + 24 data — anchored on the Slow
Data sync every 21 frames.

Three carves, three sync anchors, zero captures; `docs/decoder-capture-needs.md`
lists the geometry beside the table as capture-confirmed.

## The composer wiring

The chains are production code. `classifyVoiceKind` in `composer.go`
maps `"nxdn"`, `"dpmr"` and `"dstar"` to `voiceKindNXDN`, `voiceKindDPMR`
and `voiceKindDSTAR`, and `handleStart` dispatches to `runNXDNVoiceChain`,
`runDPMRVoiceChain` or `runDStarVoiceChain`. Each follows one shape:

```go
// internal/voice/composer/nxdn_voice.go (shape)
fe := newNXDNVoiceFrontEnd(iqHz, c.bw)       // newVoiceFrontEnd(iqHz, bw, 48_000, 6250)
rs, _  := c.sink.(rawFrameSink)
ers, _ := c.sink.(errAwareRawSink)
tc := nxdn.NewTrafficChannel(func(frames []nxdn.VCHFrame) {
    for _, vf := range frames {
        packed := framing.PackBitsMSB(vf.Payload)  // 49 bits → 7 bytes
        bt.onVoice(0)
        werr = ers.WriteRawFrameWithErrors(serial, packed, vf.Errors)
    }
})
rx := nxdnrx.New(nxdnrx.Options{SampleRateHz: symbolHz, DeviationHz: 1800.0,
    DibitSink: func(d []uint8, base int) { tc.Process(d, base) }})
```

The front-end constants differ per protocol, each with a reason in a
comment: `nxdnChannelSelectHz` 6250 (half the 12.5 kHz NXDN96 spacing),
`dpmrChannelSelectHz` and `dstarChannelSelectHz` 3125 (half of 6.25 kHz),
`dpmrVoiceDeviationHz` 900 (half of P25/DMR), all decimating to 48 kHz.
Every chain logs `composer: … voice chain started (… decode is unverified
on air — experimental)` and ends by logging its `voice_frames` count. The
boundary tracker is built with grant talkgroup 0 — none of the three
decoders surfaces a per-frame talkgroup yet — and `bt.onVoice(0)` keeps
hangtime alive per frame.

`errAwareRawSink` is the detail that matters on the first capture:
`Recorder.WriteRawFrameWithErrors` takes the per-frame Golay corrected-bit
count and drives the vocoder's error-rate adaptive smoothing, as the DMR
chain does. And the recorder's
`DefaultVocoderForProtocol` picks the codebook: `"nxdn"` and `"dpmr"` map
to `"ambe2-dmr"` (`ambe2.NewDMR`, the 3600×2450 `unpackParams2450`
tables), `"dstar"` to `"ambe2"` (`ambe2.New`, 3600×2400). The map's comments carry the caveat — "the 2450-vs-2400 codebook choice
is one of the things a real … voice capture confirms" — and v1.1.2's
changelog records that dPMR's entry once pointed at 2400.

## What the first capture pins, and the DMR instruments that transfer

`docs/decoder-capture-needs.md` asks for "dPMR / D-STAR voice (≥ 10 s IQ,
48 kHz, clear voice)" and an NXDN voice capture beside the control-channel
one, and names the three confirmations in order — "its AMBE interleave
table … plus the frame geometry and codebook choice". The DMR path shows
what those look like as numbers, and none of the instruments is
DMR-specific.

**The Golay histogram.** `VCHFrame.Errors`, `TCHFrame.Errors` and
`DVFrame.Errors` already carry the corrected-bit counts. On the #1187 DMR
captures, bursts sliced from the wrong positions showed "the random-word
Golay signature (73 % exactly 3 corrections, 12 % exactly 2)" while
correctly sliced ones were FEC-clean — the C0/C1 histogram is "the first
thing to check when frames look scrambled". A wrong deinterleave produces
the same signature: Golay(23,12) is a perfect code, so every random
23-bit word decodes to *some* codeword, mostly three corrections away,
and a real table collapses the histogram toward zero. That answers the
table question without a WAV.

**b0 continuity.** The #1187 harnesses judge a decrypt by pitch
continuity — |Δb0| ≤ 10 between frames, "speech ≳ 0.5, random/ciphertext
≈ 0.16" — never by loudness, because random AMBE parameters are loud. The
same score separates the 2450 and 2400 codebooks: unpack the recovered
49 bits both ways and the right one tracks.

**The per-frame diff.** DMR's "computer voice" was settled by dumping
`cur_mp` per frame from two mbelib lineages over a committed `.raw` clip,
which found the real bug in silence frames (b0 124/125) that reset the
gain predictor 24 dB low. That fix lives in `unpackParams2450`, so NXDN
and dPMR inherit it through `ambe2-dmr`; D-STAR's 2400 base path has no
silence-frame indicator ("Silence is not a separate AMBE+2 indicator",
per `params.go`), and only a capture can say whether a real D-STAR stream
needs one. The recorder's `.raw` sidecar (`recordings.write_raw`) is that diff's
input.

## The #764/#771 rule applied

The rule this series keeps returning to — a green synthetic is never an
on-air pass — is written into these files more literally than anywhere
else. The chains are **wired end to end** (the composer dispatches, the
recorder renders, a WAV appears), **labelled experimental** (every start
line says so), and **gated** (`Refs`, never `Closes`, until a reporter
confirms — [From Spec to Shipping Part 14]({{ '/blog/deep-dives/from-spec-to-shipping-14-definition-of-verified/' | relative_url }})
has the policy). "Verified" for each chain would mean three things in order: a Golay
histogram at the floor under the real table, frames at the protocol's
cadence, and a b0 track that reads as speech under one codebook. Until a
capture produces them, `status.md`'s sentence stands: NXDN, dPMR and
D-STAR "are wired end-to-end through the composer but not yet verified
on air".

### How the placeholders shaped the Go code

- **One function per unknown.** A capture-driven swap touches
  `nxdnAMBEDeinterleave` and its inverse and nothing else.
- **Encoders kept in lock-step on purpose.** `EncodeVCHFrame`,
  `EncodeTCHFrame` and `EncodeDVVoiceBits` exist for tests and must move
  with the decoder.
- **Error counts clamped, never dropped.** `clampErrs`/`clampGolayErrs`
  keep an uncorrectable sub-vector from poisoning the smoother's sum.
- **Warnings at the top of the file.** Each `voice_ambe.go` opens with
  the "UNVERIFIED ON AIR" block so the caveat travels with the code.

## Where this goes next

Twelve packages later, the series has a rung for every protocol and a
named capture for every rung.
[Part 14]({{ '/blog/deep-dives/legacy-family-14-verification-ladder-playbook/' | relative_url }})
puts them in one table — protocol × rung × harness × capture — and walks
the contributor path: `gophertrunk capture`, the sidecar, `gophertrunk
test`, and the skip-gated harnesses that wake when a file lands in
`samples/`.

## FAQ

**Is the NXDN voice decoder wrong, or just unverified?**
Unverified, precisely. The Golay(23,12) coding, C1 descramble and 49-bit
assembly are the codec's and match the capture-verified DMR path; the 72-bit deinterleave (`nxdnAMBEDeinterleave`)
is a sequential split labelled `PLACEHOLDER`, and the VCH carve and
2450-vs-2400 codebook are spec-transcribed. A real NXDN voice capture
confirms those three.

**Why does dPMR use the `ambe2-dmr` vocoder?**
Because ETSI TS 102 658 specifies the AMBE+2 half-rate codec — 2450 bps
voice plus 1150 bps FEC — the 3600×2450 variant DMR and NXDN use, so
`DefaultVocoderForProtocol` maps `"dpmr"` to `"ambe2-dmr"`; v1.1.2 fixed
an earlier mapping to 2400. The map's comment still calls the choice one
of the things a capture confirms.

**How would a capture show the deinterleave table is wrong?**
Through the Golay corrected-bit counts the chains already report. A wrong
table turns C0 and C1 into random 23-bit words, which a perfect code
decodes mostly with three corrections — the random-word histogram the
#1187 DMR captures exposed. A correct table collapses the histogram
toward zero before anyone listens.

**Do the DMR AMBE+2 fixes apply to NXDN and dPMR voice?**
The silence-frame fix (b0 124/125 synthesised with the gain predictor
carried through) lives in `unpackParams2450`, which `ambe2-dmr` runs, so
NXDN and dPMR inherit it. D-STAR renders through the 3600×2400 base
decoder, which has no silence-frame indicator; whether a real D-STAR
stream needs one is a capture question.

## Series navigation

**Part 13 of 14** · ←
[Part 12: Yaesu System Fusion — FICH Trellis, Frame Types and the Voice That Is Not Yet PCM]({{ '/blog/deep-dives/legacy-family-12-ysf/' | relative_url }})
· Next →
[Part 14: The Verification Ladder — Where Each Protocol Stands and How to Move It]({{ '/blog/deep-dives/legacy-family-14-verification-ladder-playbook/' | relative_url }})
