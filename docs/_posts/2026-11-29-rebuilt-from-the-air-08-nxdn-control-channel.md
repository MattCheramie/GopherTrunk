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
> [#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261). Voice
> that "works for 5-10 minutes, then fails until I change ppm by 1" was the
> NXDN timing loop seeing the carrier offset: the AFC ran after the clock, so
> NXDN48 decoded only within about ±700 Hz, and the reporter's dongle walked
> −645 → −679 Hz in 8.7 s. `EnableCarrierTracking` — a frequency-locked loop on the
> discriminator mean, ahead of filter and clock, held whenever the
> discriminator variance says "noise" — decodes across ±3 kHz. The control
> channel never locked because `ControlChannel.Process` hunted an invented
> frame; `DecodeAirCAC` (`internal/radio/nxdn/air_cac.go`) reads the real
> one: 150 dibits → 12×25 deinterleave → 12-of-14 depuncture →
> `framing.ViterbiK5` → 171 bits, CRC seeded `0xC3EE`. All 214 captured
> frames pass. Trunked voice was on-air verified on 21 October; the
> `VCALL_ASSGN` layout is still reference-pinned.

**Key takeaways**

- **A timing loop fed an uncorrected discriminator fails in a band, not past
  a threshold.** NXDN48 decoded 0–12 of 56 frames at roughly ±750–1250 Hz and
  recovered further out, so "ppm ±1 fixes it" was true and useless.
- **A frequency loop needs a measured reason to hold.** Ungated, the tracker
  random-walked to about 1.1 kHz in five seconds of noise.
- **The transcribed half was right; the invented half hid it.** The old
  spec-mode CAC coder had the real code, puncture and interleave, and never
  saw a real frame behind a sync word that does not exist.
- **Two fields that check each other settle a layout.** SITE_INFO's location
  splits to site 1; every CAC's SR carries RAN 1, the site code mod 64.

## Cheat sheet

| | |
|---|---|
| Issue | [#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261) — NXDN control channel never locks; conventional voice dies after minutes |
| Symptom | no `cc.locked` on a real NXDN96 control channel; voice "works for 5-10 minutes, then fails until I change ppm by 1 and restart" |
| Wrong theories | the ppm calibration is off; the channel filter; NXDN cannot run carrier correction because the CAC has long constant-dibit runs |
| Real cause | the AFC ran after the Mueller-Müller clock, which still saw the raw offset; the control path framed an invented FSW and LICH with no descrambler |
| Fix | `EnableCarrierTracking` (`receiver/carrier_track.go`); `DecodeAirCAC` behind the shared `airFramer` in `ControlChannel.Process` |
| Diagnostic | an offset sweep of the committed real-air slice; frame spacing at the real FSW; SITE_INFO and RAN cross-checking |
| Status | lock and site broadcasts capture-verified (214 of 214 CRC-valid); trunked voice on-air verified 21 Oct; `VCALL_ASSGN` reference-pinned |

## In this post

- **Two symptoms, one capture** — what 8.7 seconds of control channel showed.
- **Why "ppm ±1" worked** — the offset band where the timing loop fails.
- **A tracker that knows when to hold** — the loop, the gate, the sub-blocks.
- **The CAC on the real frame** — 150 dibits to 171 bits.
- **The cross-check, the lock and the config crash** — the live run and a TUI panic.
- **What this overturns, and what we keep** — Protocol Decoders 6 and the rules.

## Two symptoms, one capture

When Part 7 ended, NXDN voice decoded from two committed real-air slices,
and the control-channel path still ran on the invented layout.

The reporter had both problems on the bench. On a conventional 447.100 MHz
channel voice decoded — for a while; the report, quoted in the test that now
pins the fix, is that decoding "works for 5-10 minutes, then fails until I
change ppm by 1 and restart." And on their trunked system, ten channels
around 453 MHz, the control channel never locked.

The capture that settled both was the scanner's own `voice_iq_debug`
recording of that control channel: 453.775 MHz, NXDN96, 2.4 MS/s, 8.7
seconds, dated 19 October by the committed slice's file name. Through the
frame layer Part 7 had just built it framed 213 times at the real 20-bit
sync word, each frame exactly 192 dibits after the last, every LICH with
good parity and RF channel type 0 — a control channel. One second of it is
committed as `testdata/nxdn96_rcch_cc_453775_19oct_48k.cs16`, and
`TestAirFramerFramesRealNXDNControlChannel` holds it to that spacing.

So the frame layer was already right; nothing above it was. And the same
recording, taken just after a ppm calibration, showed its carrier walking
from −645 Hz to −679 Hz in those 8.7 seconds.

## Why "ppm ±1" worked

The NXDN receiver is the C4FM chain every 4-level protocol in the tree
shares: FM discriminator, matched filter, Mueller-Müller clock, slicer
([SDR Internals Part 7]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }})).
Part 7 switched on `EnableAFC` for voice because both captures sat about
400 Hz off. That AFC subtracts the offset's DC bias from the *recovered
symbols* — after the timing loop has already used the biased signal to
decide where the symbols are.

The committed NXDN48 slice made the cost measurable. Shifted to other
absolute offsets and replayed, it decoded only within about ±700 Hz. From
roughly ±750 to ±1250 Hz it produced 0–12 of its 56 voice frames — and then
decoded again further out. NXDN96 held to about ±1.5 kHz. Changing the
channel filter did not move the collapse.

That non-monotonic band is the whole story. At 447 MHz one ppm is 447 Hz. A
dongle that starts inside the good region drifts a ppm or two as it warms
and lands in the dead band; moving the
[ppm correction]({{ '/reference/ppm-frequency-correction/' | relative_url }})
by one shifts the carrier 447 Hz out the other side, until the next drift.

Three theories left with that sweep. The calibration was not wrong — the
capture was taken right after one and was still moving. The filter was not
involved. And the warning in the receiver's own `EnableAFC` comment, that a
zero-heavy CAC encodes to long constant-dibit runs a carrier tracker would
walk onto, came from the invented layout: on real air every dibit after the
sync word is PN9-scrambled, so the stream is balanced.

## A tracker that knows when to hold

The fix moves the correction to the front. `carrierTracker`
(`internal/radio/nxdn/receiver/carrier_track.go`) is a first-order
[frequency-locked loop]({{ '/reference/frequency-locked-loop/' | relative_url }})
on the IQ: an NCO mixes each block down by the current estimate, and the
estimate moves toward the mean of that block's discriminator output — for a
balanced symbol stream, the residual offset.

```go
// internal/radio/nxdn/receiver/carrier_track.go (shape)
mean := sum / float64(len(disc))
if sum2/float64(len(disc))-mean*mean > t.varLimit {
    return // no carrier in this block: hold
}
residualHz := mean * t.fs / (2 * math.Pi)
t.offHz += (1 - math.Exp(-float64(len(disc))/t.tc)) * residualHz
t.offHz = math.Max(-carrierTrackMaxHz, math.Min(carrierTrackMaxHz, t.offHz))
```

**The time constant is in symbols.** `carrierTrackSymbols` is 128 symbol
periods, about 53 ms at NXDN48 and 27 ms at NXDN96: long enough for the
modulation to average out, short enough to pull 1 kHz in within a call's
first few frames.

**The gate is a variance.** On noise the loop has no restoring force — every
block reads zero-mean *in its own mixed frame* — and ungated it walked to
about 1.1 kHz in five seconds. So a block may move the estimate only when
its discriminator variance is under `carrierTrackNoiseFraction` (0.25) of
what white noise produces through the same channel-filter taps, a figure
measured at construction on fixed-seed noise (`noiseDiscVariance`). On the
captures a carrier block sits at 0.07× (NXDN96) and 0.05× (NXDN48). It is the
gain-blind statistic behind the carrier gate in
[DMR End to End Part 9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }}).

**The loop runs on its own block size.** The estimate corrects the *next*
block, so `Receiver.Process` cuts its input into `tc/4` pieces; fed whole, an
85 ms chunk would decode at the uncorrected offset and the result would
depend on the caller's chunking.

`TestNXDNVoiceReceiverToleratesTunerDrift` replays the slice at the offsets
that used to fail: on the current tree it decodes 52 of the 56 frames at
−1250 Hz and all 56 at ±1000 and +1250 Hz, with the tracker settled within
3 Hz of the true offset. `TestNXDNVoiceCarrierTrackerHoldsOnNoise` keeps it
within 300 Hz after five seconds of noise.

One limit is open. On the live system a call that decoded 406 frames
cleanly ended with the tracker at its ±3 kHz bound: an unmodulated carrier
passes the gate and pulls the loop — shown synthetically, unconfirmed on
air. The end-of-call line now carries `carrier_offset_at_voice_hz` to
diagnose it.

## The CAC on the real frame

On air an outbound control frame is the Part 7 frame with a different
middle: the 10-dibit sync word, 8 LICH dibits, then **150 dibits of CAC** and
24 more this decoder does not read.

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

`DecodeAirCAC` is that chain and little else:

```go
// internal/radio/nxdn/air_cac.go (shape)
for i, b := range bits { // 300 CAC bits in on-air order
    coded[(i%25)*12+i/25] = b
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
([puncturing]({{ '/reference/puncturing/' | relative_url }})). The 171 bits
are an 8-bit SR — two structure bits and the six-bit
[RAN]({{ '/reference/ran-nxdn/' | relative_url }}) — a 147-bit layer-3
message and a CRC-16: a shift register (x¹⁶ + x¹² + x⁵ + 1, seeded `0xC3EE`)
run over all 171 bits must leave a remainder that complements to zero. The
message type is the low six bits of the message's first octet:
`AirMsgSiteInfo` is `0x18`, `AirMsgSrvInfo` `0x19`, `AirMsgVCallAssign`
`0x04`. The layout facts are DSD-FME's and the convolutional code is
MMDVMHost's; both are GPL and were read for protocol facts only.

Here is the uncomfortable part. Line that chain up against `cac_channel.go`,
the package's older "spec mode" coder, and the channel coding matches: the
same generator pair, the same two punctured positions, the same
`(i%25)*12 + i/25` interleave, and a CRC whose different-looking convention
(init `0xFFFF` over the 155 bits before it) yields the same sixteen check
bits. That file was transcribed from the published coding description. It
never decoded a real frame, because everything in front of it was invented:
an 8-dibit `0xC55A` sync where the air sends ten dibits, a repetition-coded
16-bit LICH where the air sends eight bits, and no descrambler at all. Above
it, `cac.go` modelled an 88-bit message with SITE_INFO at `0x3C`. The
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
does not need a whole decoder to be wrong: one fabricated layer makes every
correct layer behind it unreachable, and the round trip stays green — the
[SmartNet rebuild]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})
seen from the other side.

On the committed slice `TestDecodeAirCACRealControlChannel` decodes 20
frames, all CRC-valid, all RAN 1, carrying two distinct messages: a SRV_INFO
nineteen times and a SITE_INFO once. Across the whole capture it is 214 of
214.

## The cross-check, the lock and the config crash

A CRC proves 171 bits arrived intact, not which of them mean what. The field
positions came from one reference, so the capture was asked to check itself.

SITE_INFO's 24-bit location reads `0x400AA1`. Its top two bits name a
category that sets how the rest splits; category 1 is Local, a 17-bit system
code and a 5-bit site code: system 85, site 1. Separately, the SR at the head
of every CAC carries a RAN, and it is 1 on all of them — the site code
mod 64. Two fields in different parts of the frame, read by different code,
agree. SITE_INFO also names control channel 1 and two adjacent sites, which
is how the reporter describes the system.

`ControlChannel.Process` now runs the voice path's `airFramer` beside the
legacy paths and hands each frame to `ingestAirFrame`. From the committed
slice, `TestControlChannelLocksOnRealAirCAC` logs:

```text
INFO nxdn cc locked freq=453775000 rate=9600 site=1 sys=85
INFO nxdn: site info system="" ran=1 location_id=4197025 category=local system_code=85 site_code=1 control_channel_1=1 control_channel_2=0 adjacent_sites=2
```

The control-channel pipelines in `ccdecoder` and `widebandt2` turn the
carrier tracker on — at the capture's roughly −650 Hz they decoded zero CAC
frames without it — but not the voice chain's channel filter, which on the
Signal Lab wideband survey turned DMR leakage into a false NXDN guess. The
synthetic fixtures moved too, held dibit for dibit to an independent encoder
(`TestEncodeAirControlFrameMatchesIndependentEncoder`).

On 21 October the reporter's system did it live: lock, grant follow, calls
uploaded with talkgroup and radio ID across ten channels; conventional
NXDN48 and NXDN96 kept decoding through drift. NXDN trunked voice is
**on-air verified**.

What stood between the fix and that run was not DSP. An NXDN grant names a
channel *number*; with no `nxdn_band_plan` to turn it into hertz the daemon
warns that voice grants will be dropped (`no-bandplan`). The reporter went
to enter one and the config TUI crashed: `configtui.resolve` walked its
drill path with `FieldByName` straight onto the band-plan *pointer* (DMR's
plan, the same shape, crashed too), and the web builder had no editor for
it. The guard test `TestEveryLeafEditable` classifies field kinds and never
drives a drill path; `bandplan_drill_test.go` now presses the keys. The key
was missing from `config.example.yaml` as well. It is there now, and a plan
looks like this:

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

One thing in the tree is weaker than the live result. The committed capture
is an idle control channel with no `VCALL_ASSGN` on it, so
`TestControlChannelGrantsFromAirVCallAssign` builds its grant at the
reference's field positions inside a real captured frame. The reporter's
system followed grants; the repository cannot yet prove that layout from air
([From Spec to Shipping Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})
is why the distinction is kept).

## What this overturns

[Protocol Decoders Part 6]({{ '/blog/deep-dives/protocol-decoders-06-nxdn-dpmr/' | relative_url }})
described an 8-dibit frame sync (`0xC55A` outbound), an 8-bit LICH
"transmitted twice", a 32-dibit SACCH, and a CAC message of "88 information
bits — an 8-bit RCCH opcode, 64-bit payload, 16-bit CRC" with SITE_INFO at
`0x3C`. None of that is on the air. Part 7 corrected the frame; this part
corrects the control half: every dibit after the 20-bit sync is scrambled,
the CAC carries an SR and a 147-bit message with a six-bit type, and
SITE_INFO is `0x18`. The coding chain that post walked — 25×12 deinterleave,
puncturing, K=5 Viterbi, 16-bit CRC — was the right chain behind the wrong
frame.

## What we keep

- **Sweep the capture you have.** One real-air slice shifted in frequency is
  a drift test; the dead band was found and pinned without a second
  recording.
- **Correct the carrier ahead of anything that takes timing from the
  signal.** A post-clock AFC fixes the slicer and leaves the clock exposed.
- **Give every tracking loop a hold condition in its own statistic.**
  Variance against calibrated noise, not a power level.
- **Audit by layer.** The transcribed coder was correct, the invented frame
  was not, and both had green round trips.
- **Ask the capture to check itself.** A site code and a RAN that must
  agree outweigh a second reading of one reference.

## Where this goes next

The reporter's radios had one more feature switched on: the built-in 15-bit
voice scrambler, with keys they had programmed themselves and shared with
the captures.
[Part 9]({{ '/blog/solution-postmortem/rebuilt-from-the-air-09-nxdn-scrambler/' | relative_url }})
is how a known key became an instrument. The operator recipe is
[Cookbook Volume 2, Part 3]({{ '/blog/tutorials/cookbook-volume-2-03-nxdn/' | relative_url }}),
with a summary on the [NXDN page]({{ '/nxdn-scanner/' | relative_url }}).

## FAQ

**Why does NXDN decoding stop after a few minutes until I change the ppm setting?**
The receiver used to correct carrier offset only after its symbol-timing
loop, which failed at roughly ±750–1250 Hz on NXDN48. A drifting dongle
walks into that band, and one ppm moves it back out. The voice chain now
tracks the carrier ahead of the discriminator and decodes across ±3 kHz.

**How does GopherTrunk decode an NXDN control channel?**
It finds the 20-bit frame sync, descrambles the frame, checks the LICH, then
deinterleaves, depunctures and Viterbi-decodes the 150 CAC dibits to 171
bits under a CRC-16. SITE_INFO and SRV_INFO lock the channel; VCALL_ASSGN
publishes a voice grant.

**Is NXDN trunking verified on air in GopherTrunk?**
Lock, grant following and recording are: the reporter's ten-channel system
ran live on 21 October. Lock and the site broadcasts are also
capture-verified on a committed slice. The VCALL_ASSGN layout is
reference-pinned only; that capture has no call on it.

**Why are my NXDN voice grants dropped with a no-bandplan warning?**
An NXDN grant carries a channel number, not a frequency. Add an
`nxdn_band_plan` to the system — `linear` or a `table` — and the grants
resolve.

## Series navigation

**Part 8 of 14** · ←
[Part 7: NXDN I — The Frame Sync Was Twenty Bits, Not Sixteen]({{ '/blog/solution-postmortem/rebuilt-from-the-air-07-nxdn-frame-layer/' | relative_url }})
· Next →
[Part 9: NXDN III — The Scrambler, and the Key That Ranked First]({{ '/blog/solution-postmortem/rebuilt-from-the-air-09-nxdn-scrambler/' | relative_url }})
