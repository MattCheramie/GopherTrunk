---
title: "TETRA End to End, Part 5: TCH/S — From Traffic Burst to Speech Frame"
description: Decoding TETRA's full-rate speech channel — two 137-bit frames per slot, the class-0/1/2 sensitivity split with its unequal error protection, the AACH usage marker that routes concurrent same-carrier calls, and the replay harness that correlated decoded timeslots against the control channel's grants on real air.
category: deep-dives
keywords: tetra tch/s decode, tetra speech traffic channel, 137 bit speech frame, tetra traffic extractor, aach usage marker, tetra class 2 bits, same carrier voice, tetra multislot replay, tetra voice chain composer, gophertrunk tetra
tags: [tetra-end-to-end, tetra, tchs, voice, traffic, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "TETRA End to End"
series_part: 5
---

*Part 5 of **TETRA End to End**, a 14-part deep dive into how GopherTrunk turns
one real 25 kHz TETRA carrier into clear recorded voice.
[Part 4]({{ '/blog/deep-dives/tetra-end-to-end-04-scrambling-colour-codes/' | relative_url }})
finished the plumbing: with the scrambler seeded by the learned extended
colour code, every channel on the cell descrambles. Now we cash the cheque.
The control channel has issued a grant, a voice tap has retuned to the traffic
carrier, and π/4-DQPSK bursts are streaming in at 18000 dibits a second — this
part turns them into 137-bit speech frames, the currency the ACELP vocoder of
Part 6 spends. Everything here is the hard-decision path; where it runs out of
steam on a marginal signal is exactly where Part 8 picks up.*

> **TL;DR:** One TCH/S slot carries **two 137-bit speech frames** coded
> together into 432 type-5 bits (EN 300 395-2 §5.5). The 274 speech bits are
> reordered into three sensitivity classes — class 0 (102, uncoded), class 1
> (112, rate 8/12), class 2 (60 + CRC-8 + tail, rate 8/18) — through one
> continuous K=5 mother stream, then 24×18 interleaved
> (`internal/radio/tetra/tch.go`). `TrafficExtractor`
> (`internal/radio/tetra/traffic.go`) finds each burst, descrambles it, tags
> it with its TDMA slot and its **AACH downlink usage marker** — the reliable
> per-call demux key on real air (`DLUsageTraffic`) — and the class-2 **CRC
> gate** (~1/256 false-pass) isolates real speech from everything else. The
> composer's `runTETRAVoiceChain` wires it to the recorder, which maps
> `tetra` → `tetra-acelp`; `TestTETRAMultiSlotReplay` validated the whole
> demux on real air by correlating decoded slots against the control channel's
> grant timeslots.

**Key takeaways**

- **Speech bits are not created equal.** The 137-bit frame splits into three
  classes by perceptual sensitivity: class 2 gets a CRC and the strongest
  code, class 0 flies uncoded. The vocoder can survive class-0 errors; class-2
  errors wreck the frame, so the CRC gates on exactly those.
- **The CRC is the call's bouncer.** A non-speech burst descrambles to noise
  and passes the 8-bit class-2 check only ~1/256 of the time — on a
  single-call carrier that alone isolates the granted call's speech.
- **Route by usage marker, not by slot number.** The AACH decodes in every
  downlink slot and carries the call's usage marker; the slot number derived
  from the SB anchor jitters on real air and is kept as telemetry only.
- **The demux was verified against grants.** The multi-slot replay harness
  prints a per-slot and per-marker activity timeline from a real capture and
  cross-checks it against the control channel's granted timeslots — the
  anti-self-consistency discipline applied to routing.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| TCH/S decode | 432 type-5 bits → 2 × 137-bit frames + CRC flag | `internal/radio/tetra/tch.go` (`DecodeTCHS`, `TCHSpeechFrames`) |
| Class reorder | Table 5 type-2 order: class0 ‖ class1 ‖ class2 | `internal/radio/tetra/tch_tables.go` (`tchType2SpeechOrder`) |
| Burst extraction | rolling buffer, NTS×4 rotations, BKN1+BKN2 | `internal/radio/tetra/traffic.go` (`TrafficExtractor`) |
| Slot + marker tagging | SB-anchored slot, AACH usage marker | `traffic.go` (`slotOf`, `usageOf`), `aach.go` (`DLUsageTraffic`) |
| Solo-tap voice chain | front end → receiver → extractor → recorder | `internal/voice/composer/tetra_voice.go` (`runTETRAVoiceChain`) |
| Same-carrier demux | one tap, four slots, route by marker | `composer.go` (`followTETRASameCarrier`), `tetra_voice.go` (`tetraSlotDemux`) |
| Real-air validation | per-slot timeline vs granted timeslots | `cmd/gophertrunk/tetra_multislot_replay_test.go` |

## In this post

- **Two frames per slot** — the TCH/S coding chain and the sensitivity classes.
- **The CRC gate** — how 8 bits isolate a call.
- **Extracting the burst** — the traffic extractor's rolling buffer, revisited with payload.
- **Routing concurrent calls** — usage markers over slot numbers.
- **From extractor to recorder** — the composer wiring, and the harness that proved it.

## Two frames per slot

The TCH/S encode chain (EN 300 395-2 §5.5) packs two 30 ms speech frames into
each transmission slot. Its first move is the interesting one: the 274 speech
bits (2 × 137) are *reordered* by `tchType2SpeechOrder` — a generated,
spec-verbatim table — into three contiguous sensitivity classes. Class 0 is
the 102 bits the ACELP decoder can shrug off; class 1 is 112 bits of middling
sensitivity; class 2 is the 60 bits that must not be wrong — LSP indices, gain
fields, the parameters whose corruption makes synthesis screech rather than
hiss. Then unequal error protection, all through one continuous K=5 rate-1/3
mother stream (Part 3's speech code):

```go
// internal/radio/tetra/tch.go (shape) — EncodeTCHS
type2 := speechToType2(frameA, frameB)          // class0(102) ‖ class1(112) ‖ class2(60)
conv := append(class1, class2...)
conv = append(conv, crcTCHClass2(class2)...)    // CRC-8 over class 2 (the TAB_CRC matrix)
conv = append(conv, make([]byte, tchTailBits)...) // 4-bit flush
mother := framing.EncodeRCPCTetraMother(conv)   // 3 × 184 = 552

c1 := framing.PunctureRCPCTetra(mother[:336], framing.RCPCTetraPeriod23,  framing.RCPCTetraPuncture23,  168) // rate 8/12
c2 := framing.PunctureRCPCTetra(mother[336:], framing.RCPCTetraPeriod818, framing.RCPCTetraPuncture818, 162) // rate 8/18
type3 := append(append(append(make([]byte, 0, 432), class0...), c1...), c2...) // 102+168+162
return framing.PackBitsMSB(tchInterleave(type3)) // 24×18 matrix interleave
```

Class 0 is transmitted *uncoded* — straight into type-3 — while class 2 plus
its CRC gets rate 8/18, more than twice the redundancy of class 1's 8/12. The
budget goes where the ear needs it. `DecodeTCHS` is the exact mirror, ending
with the class-2 CRC check and returning the two frames, the CRC verdict, and
the Viterbi path metric. Full field-level detail lives in the
[TCH/S speech-coding reference]({{ '/reference/tetra-tchs-speech-coding/' | relative_url }}).

## The CRC gate

`TCHSpeechFrames` wraps the decode with the policy: no CRC, no frames.

```go
// internal/radio/tetra/tch.go (shape)
func TCHSpeechFrames(traffic []byte) [][]byte {
    frameA, frameB, crcOK, _, ok := DecodeTCHS(traffic)
    if !ok || !crcOK {
        return nil // non-TCH/S burst, or too corrupted to trust
    }
    return [][]byte{framing.PackBitsMSB(frameA), framing.PackBitsMSB(frameB)}
}
```

The gate does double duty. Its stated job is quality control — don't hand the
vocoder a frame whose critical bits are wrong. Its second job is *isolation*:
a voice carrier's other slots carry other calls' bursts and signalling, and any
burst that isn't this chain's TCH/S speech descrambles to effectively random
class-2 bits, passing an 8-bit check with probability ~1/256. On a solo tap
(one call on the carrier) the CRC alone keeps foreign bursts out of the
recording. The same 1/256 figure has a dark side, though — it is also the
**chance floor**: a decoder that is systematically wrong (Part 3's CRC bug,
Part 12's descramble skip) still "recovers" ~0.4% of bursts by luck, which
looks exactly like a very weak signal. Knowing the floor is what lets a
harness distinguish "marginal RF" from "broken decode."

## Extracting the burst

Part 2 introduced `TrafficExtractor` as geometry; now it earns its keep. Per
detected normal-training-sequence hit it slices BKN1 + BKN2 (216 dibits → 432
bits), descrambles with the cell's extended colour code, and emits a 54-byte
frame — but the emission signature is where the routing story lives:

```go
// internal/radio/tetra/traffic.go (shape)
func NewTrafficExtractor(colourCode uint32,
    onBurst func(frame []byte, softType5 []float32, slot, usage uint8)) *TrafficExtractor
```

Four things per burst: the hard frame; a soft-decision LLR stream (`nil` on
the hard-only path — Part 8's hook, already present in the signature); the
SB-anchored TDMA slot; and the **AACH downlink usage marker**. The extractor
also watches for NTS2 midambles — stolen half-slots (STCH) carrying urgent
signalling in place of speech — and counts them for the voice-path summary.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="The TETRA voice pipeline from grant to recording: the voice tap IQ passes through the 144 kHz front end into the receiver, the traffic extractor slices and descrambles each burst and tags it with slot and usage marker, the TCH/S decoder gates frames on the class-2 CRC, and CRC-valid 137-bit speech frames flow to the ACELP vocoder and recorder.">
  <rect x="6" y="70" width="88" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="50" y="89" text-anchor="middle" fill="currentColor" font-size="10">voice tap IQ</text>
  <text x="50" y="103" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ 144 kHz</text>
  <line x1="94" y1="93" x2="114" y2="93" stroke="currentColor"/><polygon points="114,89 123,93 114,97" fill="currentColor"/>
  <rect x="123" y="70" width="96" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="171" y="89" text-anchor="middle" fill="currentColor" font-size="10">receiver</text>
  <text x="171" y="103" text-anchor="middle" fill="var(--fg-muted)" font-size="9">dibits @18k/s</text>
  <line x1="219" y1="93" x2="239" y2="93" stroke="currentColor"/><polygon points="239,89 248,93 239,97" fill="currentColor"/>
  <rect x="248" y="70" width="130" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="313" y="89" text-anchor="middle" fill="var(--accent)" font-size="10">TrafficExtractor</text>
  <text x="313" y="103" text-anchor="middle" fill="var(--fg-muted)" font-size="9">slice · descramble · tag</text>
  <line x1="378" y1="93" x2="398" y2="93" stroke="currentColor"/><polygon points="398,89 407,93 398,97" fill="currentColor"/>
  <rect x="407" y="70" width="110" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="462" y="89" text-anchor="middle" fill="var(--accent)" font-size="10">DecodeTCHS</text>
  <text x="462" y="103" text-anchor="middle" fill="var(--fg-muted)" font-size="9">class-2 CRC gate</text>
  <line x1="517" y1="93" x2="537" y2="93" stroke="currentColor"/><polygon points="537,89 546,93 537,97" fill="currentColor"/>
  <rect x="546" y="70" width="128" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="610" y="89" text-anchor="middle" fill="currentColor" font-size="10">2 × 137-bit frames</text>
  <text x="610" y="103" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ ACELP → WAV + .raw</text>
  <text x="313" y="146" text-anchor="middle" fill="var(--fg-muted)" font-size="9">per burst: slot (telemetry) + AACH usage marker (routing)</text>
  <line x1="313" y1="118" x2="313" y2="136" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="462" y="146" text-anchor="middle" fill="var(--fg-muted)" font-size="9">CRC fail ⇒ dropped (~1/256 false pass)</text>
  <line x1="462" y1="118" x2="462" y2="136" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="340" y="182" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the hard-decision path — the softType5 hook in the onBurst signature is already waiting for Part 8</text>
</svg>
<figcaption>Grant to recording on the hard path: extract, descramble, tag, CRC-gate, then hand 137-bit frames to the vocoder.</figcaption>
</figure>

## Routing concurrent calls: markers over slots

A TETRA carrier is four TDMA slots, and each can hold an independent call —
so the extractor emits *every* NCDB it sees and something downstream must
demultiplex. The obvious key is the slot number. It is also the wrong one, and
the doc comment in `traffic.go` says so bluntly: on real air the SB anchor's
intra-slot rounding jitters a call's bursts across adjacent slot numbers, and
the grant's channel-allocation timeslot field doesn't map cleanly to the
physical slot anyway. The reliable key is the **AACH**: it's present in every
downlink slot, its 30-bit access-assignment names what the slot carries, and a
call's usage marker matches the marker in its grant. `usageOf` recovers it per
burst — the two AACH halves either side of the midamble, RM(30,14)-decoded
under the cell colour, values ≥ `DLUsageTraffic` (4) meaning traffic — with a
distance-gated soft fallback (`usageOfSoft`, `aachSoftMaxDist = 6`) so a
marginal AACH doesn't drop a routable burst or, worse, invent a marker that
leaks another call's speech into the recording.

The composer routes accordingly. `onTETRATrafficBurst` keeps bursts whose
marker matches the granted call's, and when either side lacks a marker it
falls back to CRC-gated isolation rather than guessing. Concurrent
same-carrier calls ride `followTETRASameCarrier` and a shared per-carrier
`tetraSlotDemux`; slot numbers appear in logs and the replay harness's
timeline, but no recording decision hangs on them.

## From extractor to recorder — and the harness that proved it

`runTETRAVoiceChain` (`internal/voice/composer/tetra_voice.go`) assembles the
solo-tap chain: a channel-select + decimation front end to 144 kHz, the shared
receiver with AFC, channel filter, Gardner timing (plus the equalizer and DC
block that later parts justify), the extractor, and the boundary tracker that
ends the call on hangtime. CRC-valid frames go to the recorder, which maps
protocol `tetra` to the `tetra-acelp` vocoder and writes both the decoded WAV
and a `.raw` sidecar of post-FEC frames — the same shape as the DMR and P25
paths, told in full in the
[composer deep dive]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }}).
On teardown the chain drains the IQ still buffered in its channel
(`drainTETRAIQ`) so a call's final bursts land in the recording instead of
dying in a queue — the fix for recordings that ended a beat early.

The validation is `TestTETRAMultiSlotReplay`
(`cmd/gophertrunk/tetra_multislot_replay_test.go`), a skip-guarded harness:
point `GT_TETRA_IQ` at a cs16 capture, give `GT_TETRA_IQ_RATE` and the cell's
colour via `GT_TETRA_COLOUR` (the committed default, 262144876, is our running
MCC 250 / MNC 13 cell), and it runs the real receiver + extractor, then prints
per-slot and per-usage-marker activity timelines — CRC-valid bursts, speech
seconds, active-second lists — to cross-check against the control channel's
grants. That cross-check is the point: slot tagging and marker routing were
accepted only when the decoded activity lined up with what the control channel
*said* was granted, on the operator's own capture. It's also the harness that
exposed the Part 3 CRC bug (everything at the 1/256 floor) and later carried
the soft-decision and equalizer A/Bs — one instrument, reused every time the
voice path changed.

## Where this goes next

We now hold 137-bit speech frames that passed their CRC. They are still just
bits. [Part 6]({{ '/blog/deep-dives/tetra-end-to-end-06-clean-room-acelp/' | relative_url }})
opens the black box at the end of the chain: the clean-room ACELP vocoder in
pure Go — LSP dequantisation, adaptive and algebraic codebooks, fixed-point
synthesis — that turns each frame into 240 samples of 8 kHz audio, bit-exact
against the ETSI reference.

## FAQ

**Why two speech frames per slot instead of one?**
The vocoder produces a 137-bit frame every 30 ms, but a call's TDMA slot comes
around once per 56.67 ms frame. Coding two speech frames into each slot makes
the arithmetic work: one slot per frame period carries the two speech frames
generated in that period.

**What happens to a burst whose CRC fails — is it silence in the recording?**
It's simply not emitted on the hard path, and the vocoder's erased-frame
concealment (Part 6) papers over isolated gaps. On a marginal signal where
*most* bursts fail, recordings come out short and garbled — that symptom, and
the ~70% of bursts the hard decoder was discarding on one reporter's captures,
is exactly what Part 8's soft decision recovers.

**Can encrypted calls leak into recordings?**
No — TEA-encrypted traffic fails the class-2 CRC just like noise (the
plaintext structure the CRC checks isn't there), so no decoded audio is
produced. The raw bursts still exist upstream for out-of-band analysis.

**Why does the extractor descramble but not TCH/S-decode?**
Separation of trust: the extractor knows the carrier's colour code (one fact,
learned once), while TCH/S decoding is per-call policy — CRC gating, marker
matching, soft/hard selection — that belongs to the composer chain. The
`.raw` sidecar gets descrambled type-5, the input any external TCH/S tool
expects.

**How do I run the replay harness on my own capture?**
`GT_TETRA_IQ=<file.cs16> GT_TETRA_IQ_RATE=<hz> GT_TETRA_COLOUR=<extended>
go test ./cmd/gophertrunk -run TestTETRAMultiSlotReplay -v` — plus
`GT_TETRA_OUT=<dir>` for per-slot WAVs. The colour override matters: with the
wrong extended colour the descramble is wrong and every count sits at the
1/256 chance floor, indistinguishable from a dead signal.

## Series navigation

**Part 5 of 14** · ←
[Part 4: Scrambling & Colour Codes — Why Colour 0 Is Not a No-Op]({{ '/blog/deep-dives/tetra-end-to-end-04-scrambling-colour-codes/' | relative_url }})
· Next →
[Part 6: A Clean-Room ACELP Vocoder in Pure Go]({{ '/blog/deep-dives/tetra-end-to-end-06-clean-room-acelp/' | relative_url }})
