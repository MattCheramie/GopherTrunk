---
title: "Rebuilt From the Air, Part 8: NXDN II — The Control Channel on Real Air, 214 of 214"
description: "An NXDN trunked control channel never locked, and voice that decoded for ten minutes died until the ppm setting moved by one. One capture explained both: a timing loop that saw the carrier offset, fixed by a variance-gated frequency loop, and a CAC decoder on the real frame — all 214 captured frames pass their CRC."
category: solution-postmortem
keywords: nxdn control channel decoder, nxdn cac decode, nxdn rcch site_info, nxdn carrier drift ppm, frequency locked loop fm discriminator, nxdn band plan config, nxdn trunking sdr, rtl-sdr drift nxdn48, nxdn ran site code, gophertrunk nxdn control channel
tags: [rebuilt-from-the-air, nxdn, control-channel, carrier-tracking, self-consistent-trap, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Rebuilt From the Air"
series_part: 8
---

*Part 8 of **Rebuilt From the Air**, a 14-part postmortem series on the
GopherTrunk decoders that passed every test and never matched a real radio —
each one diagnosed from a reference implementation or a reporter's capture,
rebuilt against independent decoders, and pinned to real air where a capture
exists.
[Part 7]({{ '/blog/solution-postmortem/rebuilt-from-the-air-07-nxdn-frame-layer/' | relative_url }})
replaced the invented NXDN frame with the real one and decoded two
conventional voice captures. This part takes the two things the same
reporter still could not do: keep that voice decoding as a dongle warmed up,
and lock a trunked control channel. One eight-second capture answered both.*

> **TL;DR:** Two symptoms on
> [#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261). "Works
> 5–10 minutes, then fails until I change ppm by 1" was the NXDN receiver's
> timing loop seeing the carrier offset: its AFC ran after the clock, so
> NXDN48 decoded only within about ±700 Hz, and the reporter's dongle walked
> −645 → −679 Hz in 8.7 s. `EnableCarrierTracking` — a frequency-locked loop
> on the discriminator mean, ahead of the filter and clock, held whenever the
> discriminator variance says "noise" — now decodes across ±3 kHz. The
> control channel never locked because `ControlChannel.Process` hunted an
> invented frame; `DecodeAirCAC` (`internal/radio/nxdn/air_cac.go`) reads the
> real one: 150 dibits → 12×25 deinterleave → 12-of-14 depuncture →
> `framing.ViterbiK5` → 171 bits, CRC seeded `0xC3EE`. All 214 frames of the
> capture pass. Trunked voice was on-air verified on the reporter's
> ten-channel system on 21 October; the `VCALL_ASSGN` field layout is still
> reference-pinned.

**Key takeaways**

- **A timing loop fed an uncorrected discriminator fails in a band, not past
  a threshold.** NXDN48 decoded 0–12 of 56 frames at roughly ±750–1250 Hz and
  recovered further out, which is why "ppm ±1 fixes it" was true and useless.
- **A frequency loop needs a measured reason to hold.** Without its variance
  gate the tracker random-walked to about 1.1 kHz in five seconds of noise.
- **The transcribed half was right; the invented half hid it.** The old
  spec-mode CAC coder had the real code, puncture and interleave, and never
  saw a real frame because the sync word in front of it does not exist.
- **Two fields that check each other settle a layout.** SITE_INFO's location
  `0x400AA1` splits to site 1, and every CAC's SR carries RAN 1 — the site
  code mod 64.

## Cheat sheet

| | |
|---|---|
| Issue | [#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261) — NXDN trunked control channel never locks; conventional voice dies after minutes |
| Symptom | no `cc.locked` on a real NXDN96 control channel; voice "works for 5–10 minutes, then fails until I change ppm by 1 and restart" |
| Wrong theories | the ppm calibration is off; the channel filter is too narrow or too wide; NXDN cannot run carrier correction because the CAC has long constant-dibit runs |
| Real cause | the AFC ran after the Mueller-Müller clock, which still saw the raw offset; and the control-channel path framed an invented FSW and LICH with no descrambler |
| Fix | `EnableCarrierTracking` (`receiver/carrier_track.go`); `DecodeAirCAC` behind the shared `airFramer` in `ControlChannel.Process` |
| Diagnostic | an offset sweep of the committed real-air slice; frame spacing at the real FSW; SITE_INFO and RAN cross-checking |
| Status | lock, SITE_INFO and SRV_INFO capture-verified (214 of 214 CRC-valid); trunked voice on-air verified 21 Oct; `VCALL_ASSGN` reference-pinned |

## In this post

- **Two symptoms, one capture** — what was reported and what 8.7 seconds of control channel showed.
- **Why "ppm ±1" worked** — the offset band where the timing loop fails.
- **A tracker that knows when to hold** — the loop, the variance gate, the sub-blocks.
- **The CAC on the real frame** — 150 dibits to 171 bits, and what the old path had right.
- **The cross-check, the lock and the config crash** — two fields agreeing, the live run, a TUI panic.
- **What this overturns, and what we keep** — Protocol Decoders 6, and the rules.

## Two symptoms, one capture

When Part 7 ended, NXDN voice decoded from two committed real-air slices and
the comment at the top of `airframe.go` said what was left: "Only the voice
path uses it so far. The control-channel (CAC) path still runs on the
invented layout and needs the same rebuild against a capture."

The reporter had both halves of that sentence on their bench. On a
conventional 447.100 MHz channel, voice decoded — for a while. The test that
now pins the fix quotes the report: decoding "works for 5-10 minutes, then
fails until I change ppm by 1 and restart." And on their trunked system, ten
channels around 453 MHz, the control channel never locked at all.

The capture that settled both was the scanner's own `voice_iq_debug`
recording of the control channel: 453.775 MHz, NXDN96, 2.4 MS/s, 8.7 seconds
(the project's notes are dated 20 October; the committed slice's file name
dates the recording to the 19th). Run through the frame layer Part 7 had just
built, it framed 213 times at the real 20-bit sync word, each frame exactly
192 dibits after the last, every LICH with good parity and RF channel type 0
— a control channel. One second of it is committed as
`testdata/nxdn96_rcch_cc_453775_19oct_48k.cs16`, and
`TestAirFramerFramesRealNXDNControlChannel` holds every frame in it to that
spacing and that LICH.

So the frame layer was already right for the control channel; nothing above
it was. And the same recording carried a second measurement nobody had asked
for: across those 8.7 seconds, taken just after a ppm calibration, the
carrier walked from −645 Hz to −679 Hz.

## Why "ppm ±1" worked

The NXDN receiver is the C4FM chain every 4-level protocol in the tree
shares: FM discriminator, matched filter, Mueller-Müller clock, slicer
([SDR Internals Part 7]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }})).
Part 7 had switched on `EnableAFC` for voice because both captures sat about
400 Hz off frequency. That AFC subtracts the offset's DC bias from the
*recovered symbols* — after the timing loop has already used the biased
signal to decide where the symbols are.

The committed NXDN48 slice made the consequence measurable. It is natively
401 Hz off; shifted to other absolute offsets and replayed through the voice
receiver, it decoded only within about ±700 Hz. From roughly ±750 to
±1250 Hz it produced 0–12 of its 56 voice frames — and then decoded again
further out. NXDN96, with twice the deviation margin per hertz of error, held
to about ±1.5 kHz. The collapse did not move when the channel filter was
changed.

That non-monotonic band is the whole "ppm ±1" story. At 447 MHz one ppm is
447 Hz. A dongle that starts a session inside the good region drifts a ppm
or two as it warms and lands in the dead band; nudging the
[ppm correction]({{ '/reference/ppm-frequency-correction/' | relative_url }})
by one moves the carrier 447 Hz, out the other side, and decoding returns
until the next drift. The workaround was real. It was also a description of
the defect.

Three theories went out with that sweep. The calibration was not wrong — the
capture was taken right after one and was still moving. The filter was not
involved. And the belief written into the receiver's own `EnableAFC` comment,
that NXDN alone in the C4FM family cannot run carrier correction because a
zero-heavy CAC encodes to long constant-dibit runs, came from the invented
layout: on real air every dibit after the sync word is PN9-scrambled, so the
symbol stream is balanced. The DMR receiver was swept the same way on its own
real-air keyup and holds sync over ±2 kHz, because its coarse acquirer
already sits ahead of the discriminator — DMR drift is not this bug.

## A tracker that knows when to hold

The fix moves the correction to the front. `carrierTracker`
(`internal/radio/nxdn/receiver/carrier_track.go`) is a first-order
[frequency-locked loop]({{ '/reference/frequency-locked-loop/' | relative_url }})
in the IQ domain: an NCO mixes each block down by the current estimate, the
discriminator runs on the mixed block, and the estimate moves toward the
discriminator's mean — which, for a balanced symbol stream, is the residual
offset in radians per sample.

```go
// internal/radio/nxdn/receiver/carrier_track.go (shape)
mean := sum / float64(len(disc))
if sum2/float64(len(disc))-mean*mean > t.varLimit {
    return // no carrier in this block: hold
}
residualHz := mean * t.fs / (2 * math.Pi)
t.offHz += (1 - math.Exp(-float64(len(disc))/t.tc)) * residualHz
t.offHz = math.Max(-carrierTrackMaxHz, math.Min(carrierTrackMaxHz, t.offHz))
t.nco.SetOffset(t.offHz, t.fs)
```

Three details carry the design.

**The time constant is in symbols.** `carrierTrackSymbols` is 128 symbol
periods — about 53 ms at NXDN48, 27 ms at NXDN96 — long enough for the data
modulation to average out, short enough to pull a 1 kHz offset in within the
first few frames of a call. The post-clock AFC stays on behind it and removes
the loop's own few tens of hertz of wander.

**The gate is a variance, measured against noise through the same filter.**
The second line is the hold condition. On noise the loop has no restoring
force: every block reads zero-mean *in its own mixed frame*, so the estimate
just accumulates. Ungated, it walked to about 1.1 kHz in five seconds of
noise. `carrierTrackNoiseFraction` is 0.25: a block is trusted only when its
discriminator variance is under a quarter of what white noise produces
through the same channel-filter taps — a figure measured once at
construction on fixed-seed noise (`noiseDiscVariance`), because the wrapped
variance of narrowband noise has no handy closed form. On the captures a
carrier block sits at 0.07× (NXDN96) and 0.05× (NXDN48) of that. It is the
same statistic, used the same way, as the carrier gate in
[DMR End to End Part 9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }}):
a phase increment is blind to gain, so no dBFS appears anywhere.

**The loop runs on its own block size.** The estimate corrects the *next*
block, so `Receiver.Process` cuts its input into `tc/4` pieces. Fed whole, an
85 ms chunk would decode entirely at the uncorrected offset, and the result
would depend on how the caller happened to chunk the stream.

`TestNXDNVoiceReceiverToleratesTunerDrift` replays the slice at the six
offsets that used to fail. Its log on the current tree:

```text
offset -1250 Hz: 52 voice frames, tracked -1247 Hz
offset -1000 Hz: 56 voice frames, tracked -997 Hz
offset  +750 Hz: 56 voice frames, tracked 753 Hz
offset +1250 Hz: 56 voice frames, tracked 1253 Hz
```

`TestNXDNVoiceCarrierTrackerHoldsOnNoise` holds the tracker within 300 Hz
after five seconds of noise at both symbol rates. Both captures now decode
across ±3 kHz, the loop's bound (`carrierTrackMaxHz`).

One limit showed up on the live system and is not closed: a call that had
decoded 406 frames cleanly ended with the tracker parked at that ±3 kHz
bound. An unmodulated carrier passes a variance gate — it is quieter than any
modulation — and pulls the loop toward wherever it sits. That mechanism is
shown synthetically and unconfirmed on air; the chain's end-of-call line now
carries `carrier_offset_at_voice_hz` and `iq_samples` so the next log can
tell a tracker that drifted after the speech from one that was off during
it.

## The CAC on the real frame

With frames arriving, the control channel needed its payload. On air an
outbound control frame is the Part 7 frame with a different middle: the
10-dibit sync word, 8 LICH dibits, then **150 dibits of CAC** and 24 more
this decoder does not read — everything after the sync word scrambled.

<figure class="lab-figure">
<svg viewBox="0 0 680 236" width="680" height="236" role="img" aria-label="Top row: an NXDN outbound control frame of 192 dibits drawn as four boxes, a 10-dibit frame sync word, an 8-dibit LICH, a 150-dibit CAC and 24 unread dibits, with a bracket showing that everything after the sync word is PN9-scrambled. Bottom row: the receive chain as five boxes joined by arrows, 300 CAC bits, a 12 by 25 deinterleave, a 12-of-14 depuncture to 350 bits, a K equals 5 Viterbi decode over 175 stages, and 171 bits checked by a CRC-16 seeded with C3EE.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">NXDN outbound control frame — 192 dibits, 40 ms at 4800 symbols/s</text>
  <rect x="20" y="26" width="70" height="40" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="55" y="43" text-anchor="middle" fill="var(--accent)" font-size="9">FSW</text>
  <text x="55" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">10 dibits</text>
  <rect x="90" y="26" width="70" height="40" fill="none" stroke="currentColor"/>
  <text x="125" y="43" text-anchor="middle" fill="currentColor" font-size="9">LICH</text>
  <text x="125" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">8 dibits</text>
  <rect x="160" y="26" width="380" height="40" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="350" y="43" text-anchor="middle" fill="var(--accent)" font-size="9">CAC</text>
  <text x="350" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">150 dibits = 300 bits</text>
  <rect x="540" y="26" width="120" height="40" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="600" y="43" text-anchor="middle" fill="var(--fg-muted)" font-size="9">not read</text>
  <text x="600" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">24 dibits</text>
  <path d="M90 76 L90 82 L660 82 L660 76" fill="none" stroke="var(--fg-muted)"/>
  <text x="375" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="8">every dibit after the FSW is PN9-scrambled (x⁹ + x⁵ + 1, seed 0xE4)</text>
  <line x1="350" y1="104" x2="350" y2="122" stroke="currentColor"/>
  <polygon points="346,122 354,122 350,130" fill="currentColor"/>
  <rect x="12" y="134" width="112" height="46" fill="none" stroke="currentColor"/>
  <text x="68" y="153" text-anchor="middle" fill="currentColor" font-size="9">300 CAC bits</text>
  <text x="68" y="167" text-anchor="middle" fill="var(--fg-muted)" font-size="8">descrambled</text>
  <line x1="124" y1="157" x2="140" y2="157" stroke="currentColor"/>
  <polygon points="140,153 148,157 140,161" fill="currentColor"/>
  <rect x="148" y="134" width="112" height="46" fill="none" stroke="currentColor"/>
  <text x="204" y="153" text-anchor="middle" fill="currentColor" font-size="9">deinterleave</text>
  <text x="204" y="167" text-anchor="middle" fill="var(--fg-muted)" font-size="8">12 × 25 block</text>
  <line x1="260" y1="157" x2="276" y2="157" stroke="currentColor"/>
  <polygon points="276,153 284,157 276,161" fill="currentColor"/>
  <rect x="284" y="134" width="112" height="46" fill="none" stroke="currentColor"/>
  <text x="340" y="153" text-anchor="middle" fill="currentColor" font-size="9">depuncture</text>
  <text x="340" y="167" text-anchor="middle" fill="var(--fg-muted)" font-size="8">12 of 14 sent → 350</text>
  <line x1="396" y1="157" x2="412" y2="157" stroke="currentColor"/>
  <polygon points="412,153 420,157 412,161" fill="currentColor"/>
  <rect x="420" y="134" width="112" height="46" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="476" y="153" text-anchor="middle" fill="var(--accent)" font-size="9">ViterbiK5</text>
  <text x="476" y="167" text-anchor="middle" fill="var(--fg-muted)" font-size="8">175 stages, rate ½</text>
  <line x1="532" y1="157" x2="548" y2="157" stroke="currentColor"/>
  <polygon points="548,153 556,157 548,161" fill="currentColor"/>
  <rect x="556" y="134" width="112" height="46" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="612" y="153" text-anchor="middle" fill="var(--accent)" font-size="9">171 bits + CRC</text>
  <text x="612" y="167" text-anchor="middle" fill="var(--fg-muted)" font-size="8">seed 0xC3EE</text>
  <text x="340" y="204" text-anchor="middle" fill="currentColor" font-size="9">171 bits = SR (structure 2 + RAN 6) · layer-3 message 147 · CRC-16</text>
  <text x="340" y="222" text-anchor="middle" fill="var(--fg-muted)" font-size="8">reporter's capture: 214 of 214 frames pass the CRC, every SR carries RAN 1</text>
</svg>
<figcaption>The control channel is the voice frame with a different middle: 150 scrambled dibits that deinterleave, depuncture and Viterbi-decode to 171 CRC-protected bits.</figcaption>
</figure>

`DecodeAirCAC` in `air_cac.go` is that chain and little else:

```go
// internal/radio/nxdn/air_cac.go (shape)
for i, b := range bits { // 300 CAC bits in on-air order
    coded[(i%25)*12+i/25] = b // written into 25 columns, read by 12 rows
}
for p := 0; p < airCACMotherBits; p++ { // 350
    if airCACPuncture[p%len(airCACPuncture)] { // 12 of every 14 were sent
        mother = append(mother, coded[k])
        k++
    } else {
        mother = append(mother, framing.DepunctureMark)
    }
}
out, metric := framing.ViterbiK5(mother, airCACStages) // 175 stages
if airCACCRC(out[:airCACMessageBits]) != 0 {           // 171 bits
    return AirCAC{}, false
}
```

The two unsent positions of every 14 are 3 and 11
([puncturing]({{ '/reference/puncturing/' | relative_url }})). The code is
the rate-½, K=5 code the SACCH and the System Fusion FICH also use. The 171
bits are an 8-bit SR — two structure bits and the six-bit
[RAN]({{ '/reference/ran-nxdn/' | relative_url }}) — a 147-bit layer-3
message and a CRC-16, checked by running a shift register
(x¹⁶ + x¹² + x⁵ + 1, seeded `0xC3EE`) over all 171 bits and requiring the
remainder, complemented, to be zero. The message type is the low six bits of
the message's first octet: `AirMsgSiteInfo` is `0x18`, `AirMsgSrvInfo`
`0x19`, `AirMsgVCallAssign` `0x04`. The layout facts are DSD-FME's and the
convolutional code is MMDVMHost's; both are GPL and were read for protocol
facts only — nothing is ported.

Here is the uncomfortable part. Line that chain up against
`cac_channel.go`, the package's older "spec mode" coder, and the channel
coding matches: the same generator pair, the same two punctured positions,
the same `(i%25)*12 + i/25` interleave, and a CRC whose different-looking
convention (init `0xFFFF` over the 155 bits before it) yields the same
sixteen check bits. That file had been transcribed from the published coding
description. It never decoded one real frame, because everything in front of
it was invented: an 8-dibit `0xC55A` sync word where the air sends ten
dibits, a repetition-coded 16-bit LICH where the air sends eight bits, no
descrambler, and so a CAC window that started two dibits early on
still-scrambled data. Above it, `cac.go` modelled an 88-bit message with an
8-bit opcode and SITE_INFO at `0x3C`. The
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
does not need the whole decoder to be wrong. One fabricated layer upstream
makes every correct layer behind it unreachable, and the round trip stays
green either way — the pattern the
[SmartNet rebuild]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})
met from the other direction.

On the committed slice `TestDecodeAirCACRealControlChannel` decodes 20
frames, all CRC-valid, all RAN 1, carrying exactly two distinct messages: a
SRV_INFO nineteen times and a SITE_INFO once. Across the whole capture it is
214 of 214. `TestDecodeAirCACRejectsCorruption` inverts 40 dibits of a real
frame and requires the CRC to refuse it.

## The cross-check, the lock and the config crash

A CRC proves the 171 bits arrived intact. It says nothing about which of
them mean what. The field positions came from one reference decoder, and a
single reference is one opinion — so the capture was asked to check itself.

SITE_INFO's 24-bit location reads `0x400AA1`. Its top two bits name the
category, which sets how the rest splits; category 1 is Local, a 17-bit
system code and a 5-bit site code: system 85, site 1. Independently, the SR
at the head of every CAC carries a RAN — and it is 1 on all of them. The RAN
is the site code mod 64. Two fields, in different parts of the frame, parsed
by different code, agree. SITE_INFO also names control channel 1 and two
adjacent sites, which is how the reporter describes the system. A shifted
bit offset would have had to corrupt all of that consistently.

`ControlChannel.Process` now runs the same `airFramer` the voice path uses —
one symbol error allowed while hunting, three at the expected position once
locked — at the top of every call, beside the legacy paths, and hands each
frame to `ingestAirFrame`. From the committed slice,
`TestControlChannelLocksOnRealAirCAC` produces:

```text
INFO nxdn cc locked freq=453775000 rate=9600 site=1 sys=85
INFO nxdn: site info system="" ran=1 location_id=4197025 category=local system_code=85 site_code=1 control_channel_1=1 control_channel_2=0 adjacent_sites=2
```

The control-channel pipelines in `ccdecoder` and `widebandt2` turn the
carrier tracker on — at the capture's roughly −650 Hz the pipeline decoded
zero CAC frames without it — but deliberately not the voice chain's channel
filter: on the Signal Lab wideband survey the filter turned DMR leakage into
a false NXDN guess. The synthetic fixtures moved too. `EncodeAirControlFrame`
feeds the Signal Lab synthesiser and the daemon's integration test, and
`TestEncodeAirControlFrameMatchesIndependentEncoder` holds it, dibit for
dibit, to an encoder in the test file written from the reference's
convolution rather than from `DecodeAirCAC`'s inverse.

On 21 October the reporter's system did the whole thing live: lock, grant
follow, calls uploaded with talkgroup and radio ID, across a ten-channel
453 MHz system; conventional NXDN48 and NXDN96 kept decoding through drift.
By this blog's vocabulary NXDN trunked voice is **on-air verified**.

What stood between the fix and that run was not DSP. An NXDN grant names a
channel *number*; without a plan to turn it into hertz the daemon warns
`nxdn system has no nxdn_band_plan; voice grants will be dropped
(no-bandplan)`. The reporter went to enter one and the config TUI crashed:
`configtui.resolve` walked its drill path with `FieldByName` straight onto
the band-plan *pointer* (DMR's band plan, the same shape, crashed the same
way). The web builder had no NXDN band-plan editor either — the fields were
allow-listed as "editable in the TUI". The guard test, `TestEveryLeafEditable`,
classifies field kinds and never drives a drill path, so it could not see a
panic that needs keystrokes; `bandplan_drill_test.go` now sends them. The
key was also missing from `config.example.yaml`, where it now reads:

```yaml
trunking:
  systems:
    - name: "Metro-NXDN"
      protocol: nxdn
      control_channels: [453_775_000]
      nxdn_band_plan:            # give exactly one of linear / table
        table:
          - { channel: 1, freq_hz: 453_775_000 }
          - { channel: 2, freq_hz: 453_675_000 }
```

One thing in the tree is still weaker than the live result. The committed
capture is an idle control channel: no call, so no `VCALL_ASSGN`.
`ParseAirVCallAssign` reads the reference decoder's field positions, and
`TestControlChannelGrantsFromAirVCallAssign` builds its grant at those same
positions inside a real captured frame. The reporter's system followed
grants; the repository cannot yet prove the layout from air. A control-channel
capture with a call on it closes that
([From Spec to Shipping Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})
on why that distinction is kept).

## What this overturns

[Protocol Decoders Part 6]({{ '/blog/deep-dives/protocol-decoders-06-nxdn-dpmr/' | relative_url }})
described the NXDN control channel as an 8-dibit frame sync (`0xC55A`
outbound, `0x3AA5` inbound), an 8-bit LICH "transmitted twice" and decoded by
majority vote, a 32-dibit SACCH, and a CAC message of "88 information bits —
an 8-bit RCCH opcode, 64-bit payload, 16-bit CRC" with SITE_INFO at `0x3C`.
None of that is on the air. Part 7 corrected the frame; this part corrects
the control half: the sync word is 20 bits, the LICH is eight bits carried
one per dibit, every later dibit is scrambled, the CAC carries an SR and a
147-bit message whose type is six bits, and SITE_INFO is `0x18`. The coding
chain that post walked — 25×12 deinterleave, 50-of-350 puncture, K=5
Viterbi, 16-bit CRC — was the right chain. Its "honest deferrals" section
called the simple fixture path a convenience; the real deferral was the
frame itself.

## What we keep

- **Sweep the capture you have.** One committed real-air slice, shifted in
  frequency, is a drift test, an AFC test and a tracker test. The dead band
  was found, and is now pinned, without a second recording.
- **Correct the carrier ahead of anything that takes timing from the
  signal.** A post-clock AFC fixes the slicer and leaves the clock exposed.
- **Give every tracking loop a hold condition measured in its own
  statistic.** Discriminator variance against calibrated noise, not a power
  level, decides whether a block may move the estimate.
- **Audit by layer, and trust provenance, not tests.** The transcribed CAC
  coder was correct and the invented frame was not; both had green round
  trips.
- **Ask the capture to check itself.** Two fields that must agree —
  a site code and a RAN — are worth more than a second reading of one
  reference.
- **A config editor is part of the decode path.** If the band plan cannot be
  entered, no grant is followed; an editor's test has to press keys.

## Where this goes next

The reporter's radios had one more feature switched on: the built-in 15-bit
voice scrambler, with keys they had programmed themselves and were willing to
share alongside the captures.
[Part 9]({{ '/blog/solution-postmortem/rebuilt-from-the-air-09-nxdn-scrambler/' | relative_url }})
is how a known key turned into an instrument — and what it said about the
register, the keystream schedule and a six-bit CRC. The operator recipe for
the whole NXDN setup is
[Cookbook Volume 2, Part 3]({{ '/blog/tutorials/cookbook-volume-2-03-nxdn/' | relative_url }}),
and the [NXDN operator page]({{ '/nxdn-scanner/' | relative_url }}) has the
feature summary.

## FAQ

**Why does NXDN decoding stop after a few minutes until I change the ppm setting?**
Before this fix the NXDN receiver corrected carrier offset only after its
symbol-timing loop, which failed in a band of roughly ±750–1250 Hz on NXDN48.
A dongle drifting a ppm or two at 447 MHz walks into that band; moving the
ppm by one shifts the carrier 447 Hz back out. The voice chain now runs a
pre-discriminator carrier tracker and decodes across ±3 kHz.

**How does GopherTrunk decode an NXDN control channel?**
It finds the 20-bit frame sync, descrambles the frame, checks the LICH, then
takes the 150 CAC dibits through a 12×25 deinterleave, a 12-of-14 depuncture
and a K=5 Viterbi decode to 171 bits protected by a CRC-16. SITE_INFO and
SRV_INFO lock the channel on the site's system and site codes; VCALL_ASSGN
publishes a voice grant.

**Is NXDN trunking verified on air in GopherTrunk?**
Yes for lock, grant following and recording: the reporter's ten-channel
system ran live on 21 October. In the repository, lock and the site
broadcasts are capture-verified on a committed one-second slice (all frames
CRC-valid). The VCALL_ASSGN field layout is reference-pinned only, because
the committed capture has no call on it.

**Why are my NXDN voice grants dropped with a no-bandplan warning?**
An NXDN grant carries a channel number, not a frequency. Add an
`nxdn_band_plan` to the system — either `linear` (base, spacing, offset) or a
`table` of channel-to-frequency entries — and the grants resolve. Both config
editors can enter it since the crash described above was fixed.

**Does the carrier tracker ever move on noise or an idle channel?**
It holds whenever the discriminator variance of a block exceeds a quarter of
calibrated noise, so receiver noise does not move it. An unmodulated carrier
does pass that gate and can pull it to its ±3 kHz bound; that case is shown
synthetically, not confirmed on air, and the end-of-call log line carries the
fields to diagnose it.

## Series navigation

**Part 8 of 14** · ←
[Part 7: NXDN I — The Frame Sync Was Twenty Bits, Not Sixteen]({{ '/blog/solution-postmortem/rebuilt-from-the-air-07-nxdn-frame-layer/' | relative_url }})
· Next →
[Part 9: NXDN III — The Scrambler, and the Key That Ranked First]({{ '/blog/solution-postmortem/rebuilt-from-the-air-09-nxdn-scrambler/' | relative_url }})
