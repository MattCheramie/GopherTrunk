---
title: "Squelch, Part 12: One Front End for Every Narrowband Decoder"
description: "Why every APRS frame on a 2.4 MS/s dongle decoded as garbage — eight single-channel decoders ran their FM discriminator on the SDR's whole span — and the shared narrowband.FrontEnd that fixed them: an LO-offset mix, decimation to 48 kHz and a ±8 kHz channel filter. Plus the paging group that decoded only while the Spectrum view was open."
category: deep-dives
keywords: narrowband front end sdr, fm discriminator noise bandwidth, aprs decoder garbage frames rtl-sdr, channel filter before discriminator, pocsag flex paging single dongle, ads-b false aircraft icao, iq broker primary stream, dedicated sdr role, gophertrunk narrowband decoders
tags: [squelch, narrowband, front-end, dsp, decoders, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Squelch"
series_part: 12
---

*Part 12 of **Squelch**, a 14-part deep dive into GopherTrunk's conventional
scanner — the scan list with no control channel to tell it a call exists —
and every gate that decides when a channel opens.
[Part 11]({{ '/blog/deep-dives/squelch-11-data-on-the-scan-list/' | relative_url }})
hung MDC1200, FleetSync and ACARS decoders off scan-list channels, behind a
front end the scanner owns. This part follows that front end off the scan
list, to the decoders that pin a whole dongle to one frequency — where the
same missing channel filter turned every frame into garbage, and where a
second defect left a decoder waiting on a stream nobody was running.*

> **TL;DR:** Eight single-channel decoders ran their FM discriminator on the
> SDR's full-rate IQ with no channel filter
> ([#1316](https://github.com/MattCheramie/GopherTrunk/issues/1316)). A
> discriminator's output SNR follows the carrier-to-noise ratio across its
> whole input bandwidth, so 2.4 MHz around a 12.5 kHz channel is about 23 dB
> of needless noise: every APRS frame on the reporter's 144.39 MHz dongle
> failed its FCS. All eight now sit behind `narrowband.FrontEnd` — LO-offset
> mix, anti-aliased integer decimation to ~48 kHz, a ±8 kHz Kaiser filter —
> with each receiver built for `fe.OutRate()` and the dongle tuned 518 kHz
> below the channel at 2.4 MS/s. Failing-first:
> `TestReceiverDecodesAtSDRRateThroughFrontEnd` (raw 0 packets, front end 1).
> The same pass reserved pinned dongles as `sdr.RoleDedicated` and stopped
> ADS-B inventing ~440 "aircraft" a second; a later one made the wideband
> paging group drive its own stream instead of waiting on a subscription.
> All of it is synthetic-verified; none has been run on the reporter's rig.

**Key takeaways**

- **A channel filter belongs before the discriminator, not after it.**
  FM demodulation is nonlinear: noise admitted ahead of it cannot be
  filtered out of the audio afterwards.
- **A receiver is built for the rate it is fed, not the rate the SDR runs
  at.** Every decoder takes `fe.OutRate()`; `sdr.sample_rate` stops at the
  front end.
- **A fixture's units decide what it can test.** The FFSK modulator's
  default deviation was in radians per sample, so at 2.4 MS/s it was a
  ±191 kHz signal no channel filter could pass.
- **Subscribing to a broker is not streaming it.** A decoder that must run
  all day has to start the SDR's stream itself, and give it back when the
  stream dies.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Channel front end | LO-offset mix, integer decimation to ~48 kHz, ±8 kHz Kaiser lowpass (63 taps) | `internal/dsp/narrowband/frontend.go` (`FrontEnd`, `PickDecimation`) |
| Daemon wiring | one front end per decoder at the dongle's own rate; LO tuned below the channel | `cmd/gophertrunk/single_channel.go` (`newSingleChannelFrontEnd`, `singleChannelLOHz`) |
| Retry loop | tune, open, decode, and again, with a 500 ms to 30 s backoff | `runSingleChannelDecoder` in `cmd/gophertrunk/daemon.go` |
| Stream ownership | first decoder on a serial drives the stream, later ones subscribe, cleanup releases the claim | `openSingleChannelIQ`, `cmd/gophertrunk/viewer_iq.go` |
| Reserved dongles | a serial a decoder section pins is never a control or voice candidate | `sdr.RoleDedicated`, `markDedicatedHints` |
| ADS-B address gate | address/parity frames accepted only for an ICAO proved within 60 s | `internal/radio/adsb/report.go` (`ProcessFrame`), `Tracker.KnownICAO` |
| Failing-first pins | raw full-rate receiver decodes nothing; the front-ended one decodes the packet | `internal/radio/aprs/afsk/frontend_test.go`, `cmd/gophertrunk/paging_wideband_test.go` |

## In this post

- **The discriminator that listened to 2.4 MHz** — the report, and why noise outside the channel still lands in the audio.
- **Mix, decimate, filter** — `narrowband.FrontEnd` stage by stage, and how the daemon wires it.
- **Two fixtures that could not fail** — a deviation that scaled with the sample rate, and a flag that never left the filter.
- **A dongle that belongs to its decoder** — `sdr.RoleDedicated`, and the ADS-B frames that were "CRC valid" by construction.
- **Subscribing is not streaming** — the paging group that decoded only while the Spectrum view was open.

## The discriminator that listened to 2.4 MHz

The scan list is not the only place GopherTrunk demodulates a narrow FM
channel. Eight decoder sections each pin an SDR to one frequency for the
daemon's lifetime: `aprs.channels`, `ais.channels`, `dsc.channels`,
`paging.pocsag`, `paging.flex`, `m17.channels`, `mdc1200.channels` and
`fleetsync.channels`
([Beyond Voice]({{ '/blog/series/beyond-voice/' | relative_url }}) walks
each one). Until #1316 every one was built with `sdr.sample_rate` as its
input rate and handed the dongle's raw stream.

The reporter's rig put [APRS]({{ '/reference/aprs/' | relative_url }}) on a
dongle at 144.39 MHz. Packets arrived — the HDLC framer found flags — and
every one of them was published with `fcs_ok=false`. Not weak-signal
losses: all of them.

The cause is arithmetic. An FM discriminator computes the phase step
between consecutive samples of whatever is in the stream, and its output
SNR follows the carrier-to-noise ratio across the **whole bandwidth it is
given**, not across the channel the operator cares about. At 2.4 MS/s the
stream is 2.4 MHz wide; an FM data channel is 12.5 kHz. The ratio is 192,
about 23 dB. A signal comfortably above the noise in its own channel is
below it in the span, and below the FM threshold the discriminator stops
following the modulation. A low-pass on the *audio* cannot repair that:
the damage is done in the nonlinear step.

A hardware receiver's IF filter comes first, so it never meets this. The
series already had, twice:
[Part 2]({{ '/blog/deep-dives/squelch-02-channel-power/' | relative_url }})
found the squelch measuring the whole span and
[Part 5]({{ '/blog/deep-dives/squelch-05-ctcss/' | relative_url }}) found
the tone detector discriminating it. Part 11's `dataFrontEnd` had even
measured it on the FleetSync real-air slices: raw IQ, 0 of 2 bursts;
filtered, 2 of 2. The dedicated-SDR decoders were the last place still
running raw.

## Mix, decimate, filter

`internal/dsp/narrowband` is that filter, shared: each decoder keeps its
own demod chain and runs it at a lower rate:

```go
// internal/dsp/narrowband/frontend.go (shape)
func New(inRateHz, offsetHz, cutoffHz float64) *FrontEnd {
    m := PickDecimation(inRateHz)
    f := &FrontEnd{outRate: inRateHz / float64(m), offset: offsetHz, decim: m}
    if offsetHz != 0 && inRateHz > 0 {
        f.nco = dsp.NewNCO(offsetHz, inRateHz)
    }
    if m > 1 {
        f.pre = dsp.NewResampler(1, m, m*8+1, kaiserBeta)
    }
    if fc := cutoffHz / f.outRate; fc < 0.45 {
        f.chanFilter = filter.NewFIR(filter.LowpassKaiser(channelFilterTaps, fc, kaiserBeta))
    }
    return f
}
```

Three stages, each optional:

- **The mix.** When the LO was tuned below the channel, the channel sits at
  `+offsetHz` in the stream and an NCO shifts it back to 0 Hz.
- **The decimator.** `PickDecimation` returns the largest integer that
  divides the input rate exactly and leaves the output at or above
  `ChannelRateHz` = 48 kHz: 50 at 2.4 MS/s (48 kHz), 40 at 2.048 MS/s
  (51.2 kHz), 60 at 3 MS/s (50 kHz). An exact divisor keeps the decoders'
  own audio resamplers at small L/M ratios
  ([SDR in Pure Go Part 5]({{ '/blog/deep-dives/sdr-internals-05-tuning-channelization/' | relative_url }})
  has the polyphase machinery).
- **The channel filter.** A 63-tap Kaiser lowpass (β 8.6) at
  `DefaultCutoffHz` = ±8 kHz: about a 2 kHz transition and more than 50 dB
  of stopband at 48 kHz, the figure the scanner's tone and data front ends
  use.

`TestFrontEndPassesChannelRejectsNeighbour` pins the point: a tone 2 kHz
off centre comes through at unity, a carrier 20 kHz away is down by more
than 50 dB.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Two block chains. Before: an SDR at 2.4 megasamples per second feeds an FM discriminator running at the full rate, then the decoder, with 2.4 megahertz of noise around a 12.5 kilohertz channel. After: the SDR is tuned 518 kilohertz below the channel, then an NCO mix, a divide-by-50 decimator, a plus or minus 8 kilohertz Kaiser filter, a discriminator at 48 kilohertz and a decoder built for the front end's output rate.">
  <text x="10" y="16" fill="var(--fg-muted)" font-size="9" font-weight="bold">before #1316</text>
  <g fill="none" stroke="var(--fg-muted)">
    <rect x="10" y="26" width="130" height="40" rx="5"/>
    <rect x="180" y="26" width="170" height="40" rx="5"/>
    <rect x="390" y="26" width="130" height="40" rx="5"/>
    <line x1="140" y1="46" x2="180" y2="46"/>
    <line x1="350" y1="46" x2="390" y2="46"/>
  </g>
  <g text-anchor="middle" font-size="9" fill="var(--fg-muted)">
    <text x="75" y="43">SDR 2.4 MS/s</text>
    <text x="75" y="56">LO on the channel</text>
    <text x="265" y="43">FM discriminator</text>
    <text x="265" y="56">at 2.4 MS/s, no channel filter</text>
    <text x="455" y="43">decoder built for</text>
    <text x="455" y="56">sdr.sample_rate</text>
  </g>
  <text x="10" y="86" fill="var(--fg-muted)" font-size="8">noise bandwidth 2.4 MHz around a 12.5 kHz channel: about 23 dB more noise into the discriminator</text>
  <line x1="10" y1="102" x2="670" y2="102" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="10" y="122" fill="var(--accent)" font-size="9" font-weight="bold">after: narrowband.FrontEnd</text>
  <g fill="none" stroke="currentColor">
    <rect x="10" y="134" width="100" height="40" rx="5"/>
    <rect x="458" y="134" width="100" height="40" rx="5"/>
    <rect x="570" y="134" width="100" height="40" rx="5"/>
    <line x1="110" y1="154" x2="122" y2="154"/>
    <line x1="222" y1="154" x2="234" y2="154"/>
    <line x1="324" y1="154" x2="336" y2="154"/>
    <line x1="446" y1="154" x2="458" y2="154"/>
    <line x1="558" y1="154" x2="570" y2="154"/>
  </g>
  <g fill="none" stroke="var(--accent)" stroke-width="1.5">
    <rect x="122" y="134" width="100" height="40" rx="5"/>
    <rect x="234" y="134" width="90" height="40" rx="5"/>
    <rect x="336" y="134" width="110" height="40" rx="5"/>
  </g>
  <g text-anchor="middle" font-size="8" fill="currentColor">
    <text x="60" y="151">SDR 2.4 MS/s</text>
    <text x="60" y="163">LO 518 kHz low</text>
    <text x="172" y="151">NCO mix</text>
    <text x="172" y="163">+518 kHz to 0 Hz</text>
    <text x="279" y="151">decimate ÷50</text>
    <text x="279" y="163">anti-alias FIR</text>
    <text x="391" y="151">±8 kHz Kaiser</text>
    <text x="391" y="163">63 taps at 48 kHz</text>
    <text x="508" y="151">FM discriminator</text>
    <text x="508" y="163">at 48 kHz</text>
    <text x="620" y="151">decoder built for</text>
    <text x="620" y="163">fe.OutRate()</text>
  </g>
  <text x="10" y="196" fill="currentColor" font-size="8">DC spur, I/Q image and clipping products land outside the ±8 kHz channel; the discriminator sees one channel of noise</text>
  <text x="10" y="212" fill="var(--fg-muted)" font-size="8">ADS-B and LoRa pass nil for the front end: wideband signals that channelize themselves</text>
</svg>
<figcaption>The decoders did not change; what changed is that the discriminator now sees one channel instead of the tuner's whole span.</figcaption>
</figure>

The daemon side is two small functions in
`cmd/gophertrunk/single_channel.go`. `newSingleChannelFrontEnd(rateHz)`
calls `narrowband.New` with the offset `pickClipSafeLOOffsetHz` returns for
that rate — the search from
[Part 4]({{ '/blog/deep-dives/squelch-04-offset-tuning/' | relative_url }}),
which parks the DC spur, the I/Q image and the clipping products outside
the channel. At 2.4 MS/s it picks 518 kHz. `singleChannelLOHz` programs the
tuner that far **below** the channel, and `runSingleChannelDecoder` says so
once at start-up — `tuning the SDR off-channel so its DC spur misses the
channel` — with `channel_hz`, `lo_hz`, `offset_hz`, `channel_rate_hz` and
`decimation`. A rate with no room for an offset, such as 250 kS/s, tunes
on-channel as before. `TestSingleChannelFrontEndAtRTLRate` pins it for the
common [RTL-SDR]({{ '/reference/rtl-sdr/' | relative_url }}) rate: 50:1,
48 kHz, LO equal to the channel minus the offset.

Every receiver constructor then takes `InputRateHz: spec.fe.OutRate()`,
never the SDR's rate; `TestDaemonBuildsSingleChannelDecodersBehindFrontEnd`
builds a daemon with APRS, AIS, DSC, MDC1200, FleetSync and POCSAG sections
and checks each has a front end at 48 kHz. The run loop changed in the
same pass: a single-channel decoder used to give up for the daemon's
lifetime on one failed `SetCenterFreq`
([Part 10]({{ '/blog/deep-dives/squelch-10-bursty-carriers/' | relative_url }})),
and `runSingleChannelDecoder` now retries the tune, the stream open and a
decoder that stops — for all of them, ADS-B and LoRa included.

## Two fixtures that could not fail

Putting a filter in front of the receivers broke two tests, and both
breakages were the tests' fault.

**The deviation that scaled with the sample rate.** `demod.FFSKModulator`
FM-modulates its audio tones with a default amplitude of 0.5 — radians per
*sample*. That is a deviation only once a sample rate is attached: ±7.6 kHz
at 96 kHz, ±191 kHz at 2.4 MS/s. A receiver that demodulates the full
stream never notices, and every round trip passes; a ±8 kHz channel filter
cannot pass a ±191 kHz signal. `SetDeviationHz` and
`ModulateFFSKDeviation` now state a real deviation (the APRS fixture uses
±3 kHz). The Signal Lab's MPT 1327 fixture still uses the default at the
SDR rate — check before putting a filter in front of it. It is Part 5's
radians-per-sample trap again, on the transmit side.

**The flag that never left the filter.** A skipped APRS synthetic test had
been failing because its closing HDLC flag ended on the very last sample
and never cleared the resampler's and timing loop's group delay. A real
transmitter keeps sending flags; `aprsTestIQ` now appends eight.

With honest fixtures, `TestReceiverDecodesAtSDRRateThroughFrontEnd` is the
regression. It modulates one packet at 2.4 MS/s under white noise across
the whole span, at a level that leaves the 16 kHz channel at about +15 dB —
about −7 dB against the full-band noise. The receiver on the raw stream,
the daemon's old wiring, decodes 0 packets; the same receiver behind the
front end decodes 1. The test also fails if the raw arm ever decodes, so
the fixture cannot quietly stop failing first — the discipline of the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}).

## A dongle that belongs to its decoder

The reporter's box had four sticks on `role: auto`, with APRS and ADS-B
pinned by serial. The pool hands the first `auto` device the control role
and the rest voice, so the ADS-B stick became *the* control SDR — fair game
for the control-channel hunter and `rigctld` — and the others were spare
voice SDRs the manual-tune scanner and a live hunt could take.

`sdr.RoleDedicated` closes that. `Config.SingleChannelDecoderSerials`
collects every serial a decoder section pins; `markDedicatedHints` sets
`Hint.Dedicated` on those; the pool opens them as `RoleDedicated` whatever
the operator wrote, and no by-role lookup returns them
(`TestPoolDedicatedHintIsNeverControlOrVoice`). A dedicated dongle may also
run its own `sample_rate`, which is why `decoderSDRRate` asks the pool for
the device's rate rather than reading the shared one.

The ADS-B half of the report was louder. A
[Mode S]({{ '/reference/mode-s/' | relative_url }}) frame's last 24 bits
are a CRC for DF 11, 17 and 18. For DF 0, 4, 5, 16, 20 and 21 they are the
CRC XORed with the aircraft's address, so the "ICAO" is whatever the XOR
yields — and the old decoder marked those `CRCValid = true`, by
construction, with no check at all. On an unfiltered dongle every noise
burst that passed the preamble test became an aircraft: 78 801 reports in
three minutes on the reporter's rig, about 440 a second, with ICAOs in
unallocated blocks. `ProcessFrame` now accepts an address/parity frame only
for an ICAO that a CRC-checked frame proved within `ICAORecentWindow`
(60 s) — `Tracker.KnownICAO`, the rule dump1090 and readsb use — and a
DF 11 reply with a non-zero interrogator ID in its parity is accepted
rather than dropped.

One obvious fix was measured and **rejected**: a noise-floor margin on the
preamble detector. The shape test already selects local maxima, so a 6 dB
margin removed under 1 % of roughly 2 600 noise preambles a second while
costing weak real frames. The better statistic was "has anyone vouched
for this address".

## Subscribing is not streaming

The last defect is not about bandwidth. An operator ran a paging-only rig —
one dedicated RTL-SDR, one `paging.wideband` group of 929 MHz
[FLEX]({{ '/reference/flex/' | relative_url }}) channels, no trunking
system — and pages decoded only while the web console's Spectrum view was
open.

Every SDR in the pool is wrapped in an `iqtap.Broker`, and a broker fans IQ
out to its subscribers **only while a primary `StreamIQ` session runs**
([From the Issue Tracker Part 16]({{ '/blog/solution-postmortem/from-the-issue-tracker-16-conventional-fm-broker/' | relative_url }})
introduced the broker for the scanner). A control-channel decoder or a
wideband engine provides that session. The paging group only called
`Subscribe()`, so with no trunking system on the dongle it waited on a
channel nothing fed.

What fed it, sometimes, was the fix for an earlier report — a waterfall
that worked on the example config and went blank on one with no systems.
`Daemon.acquireViewerIQ` (`viewer_iq.go`) drives `StreamIQ` on a live
view's behalf while one is open on an SDR with no primary, and the pager's
only IQ was that viewer pump. The signature is this pair in the log, on a
dongle that *has* a decoder:

```text
sdr: no decoder is streaming this SDR — streaming it for the live view
sdr: last live view closed — stopping its IQ stream
```

The group now runs through `runSingleChannelDecoder` like every other
decoder that pins an SDR, which opens its stream with
`openSingleChannelIQ`: the first caller on a serial claims it in
`iqPrimary` and drives `StreamIQ`; later callers subscribe; a viewer pump
already running is preempted and the view keeps receiving the decoder's
fan-out. Two more defects sat on the same path, both pinned failing-first:

- **The claim was never released.** After a stream died, the restart found
  the serial still "owned", only subscribed, and stayed deaf behind a
  `decoder stopped — restarting` line that read as a recovery: two decode
  passes, one `StreamIQ` session. The primary's cleanup now releases the
  claim (`TestSingleChannelDecoderReopensItsStreamAfterItEnds`).
- **The bank was built for the wrong rate.** The group's DDC bank used
  `sdr.sample_rate`, not the dongle's own — every tap at the wrong offset,
  with no warning (`TestWidebandPagingBankFollowsItsSDRsOwnRate`).

The test SDR is the lasting part: `pagingAirFake` delivers IQ only while a
session runs, refuses a second session, and places its FLEX transmitter
relative to the *programmed* centre. A fake that streams unasked cannot see
this class of bug.

### How the front end shaped the Go code

- **One package for a lesson learned three times.** `narrowband.FrontEnd`
  is the scanner's tone and data front-end design, shared.
- **Rates are passed, never assumed.** A receiver's `InputRateHz` comes
  from `fe.OutRate()`; a dongle's from `decoderSDRRate`.
- **Fixtures carry physical units.** `ModulateFFSKDeviation` takes hertz.
- **Always-on consumers go through `openSingleChannelIQ`.** A new decoder
  that skips it inherits the paging bug.

What is **not** verified: all of #1316 is on-air-gated — nothing here has
run on the reporter's dongles — and the paging fix has not been run on a
real dongle either. Two things were not built: single-channel decoders
still do not reacquire a dongle after an unplug (the control-channel
decoder and wideband engines do), and a wideband paging group whose channel
sits at the group centre — an odd count of evenly spaced channels
auto-centres on the middle one — puts that channel on the tuner's DC spike.

## Where this goes next

Twelve parts in, the gates are built. What remains is what the operator
turns: which channels are sampled more often, which are skipped, and why a
P25 channel is louder than the NFM repeater next to it.
[Part 13]({{ '/blog/deep-dives/squelch-13-operator-controls/' | relative_url }})
covers priority scan, lockouts that persist by frequency, and per-channel
loudness.

## FAQ

**Why were all my APRS packets marked fcs_ok=false on a 2.4 MS/s dongle?**
Tuning sets where the centre of the span is, not how wide it is. Before the
#1316 fix the APRS receiver discriminated the full-rate IQ, and a
discriminator's output SNR follows the carrier-to-noise ratio across its
whole input bandwidth — about 23 dB of extra noise. It now runs behind
`narrowband.FrontEnd` at 48 kHz; the fix is synthetic-verified, not yet
confirmed on the reporter's rig.

**Why did GopherTrunk only decode pages while the Spectrum view was open?**
The wideband paging group subscribed to its SDR's IQ broker but never
started the stream, and a broker fans IQ out only while a primary stream
session runs. Opening a live view started one on the view's behalf. The
group now drives the stream itself through `runSingleChannelDecoder`.

**What is role: dedicated and do I have to set it?**
No. A dongle that an aprs, ais, dsc, adsb, paging, m17, mdc1200, fleetsync
or lora section pins by serial is opened as `sdr.RoleDedicated`
automatically, so trunking, the scanner, a hunt and `rigctld` never retune
it. Writing `role: dedicated` on a dongle no decoder names is rejected at
validation.

**Why did ADS-B show hundreds of aircraft a second with strange ICAO codes?**
Six Mode S downlink formats carry the address XORed into the parity, and
the decoder treated the recovered address as checked, so every noise burst
that passed the preamble shape test became an aircraft. Those frames are
now accepted only for an address a CRC-checked DF 11, 17 or 18 frame proved
within the last 60 seconds.

## Series navigation

**Part 12 of 14** · ←
[Part 11: Data on the Scan List — MDC1200, FleetSync and ACARS Taps]({{ '/blog/deep-dives/squelch-11-data-on-the-scan-list/' | relative_url }})
· Next →
[Part 13: Operator Controls — Priority, Lockouts and Loudness]({{ '/blog/deep-dives/squelch-13-operator-controls/' | relative_url }})
