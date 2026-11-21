---
title: "Squelch, Part 9: Call Gates II — System Fusion, NXDN and D-STAR"
description: "Three more scan-list call gates and why each needs different evidence: System Fusion opens on one 20-dibit frame sync, NXDN needs a sync pair 192 dibits apart plus a traffic LICH because a lone sync turns up in noise every seven seconds, and D-STAR needs a CRC-valid header or two slow-data syncs, heard by two receivers half a symbol apart."
category: deep-dives
keywords: system fusion scanner squelch, ysf frame sync detection, nxdn call detection lich, nxdn frame sync false alarm rate, d-star slow data sync, d-star header crc gate, mueller muller half symbol hang, nxdn48 scan list channel, digital scan list call gate, gophertrunk ysf nxdn dstar
tags: [squelch, ysf, nxdn, dstar, scanner, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Squelch"
series_part: 9
---

*Part 9 of **Squelch**, a 14-part deep dive into GopherTrunk's conventional
scanner — a scan list with no control channel to announce its calls — and
every gate that decides when one of its channels opens.
[Part 8]({{ '/blog/deep-dives/squelch-08-call-gates-p25-dmr/' | relative_url }})
made P25 and DMR channels open on decoded frames instead of carrier power.
This part finishes the digital modes with System Fusion, NXDN and D-STAR:
the same idea three times, with three different answers to "how much
evidence is enough?" — because the three protocols hand a receiver sync
words of very different strength.*

> **TL;DR:** A `ysf`, `nxdn`, `nxdn48` or `dstar` channel used to open a
> call on any carrier above `squelch_dbfs`; each now has a `callGate` whose
> evidence rule follows from its sync word. `YSFCallDetector` opens on one
> 20-dibit frame sync (a chance match is about one in 10⁹ positions) and
> needed feed-forward timing: the first sync arrived after 0.6–1.2 s from
> three of ten start phases, 0.11 s with it. `NXDNCallDetector` wants a
> sync, a second exactly 192 dibits later and a parity-clean LICH naming a
> traffic channel — a lone 10-dibit sync turns up in noise every ~7 s at
> NXDN96 — and opens in 60–100 ms (NXDN96) or 440–770 ms (NXDN48).
> `DStarCallDetector` wants a CRC-valid header or two slow-data syncs one or
> two cycles apart, in either polarity, and runs two receivers half a symbol
> apart. NXDN is pinned on real-air slices; YSF and D-STAR have no real
> capture yet.

**Key takeaways**

- **The strength of the sync word sets the evidence rule.** Twenty dibits
  can stand alone; ten cannot; a 24-bit pattern at one error turns up in
  noise about once a minute.
- **A control channel is the protocol, but it is not a call.** The NXDN gate
  reads the LICH, so a trunked site's control frequency on a scan list never
  opens.
- **Every gate hit the same timing wall and got a different door.** YSF
  seeds its loop feed-forward, NXDN rebuilds its receiver on each retune,
  D-STAR runs two receivers.
- **Only one of the three has heard a radio.** NXDN is pinned on a
  reporter's captures; the YSF and D-STAR gates open only on signals the
  tests built.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| YSF evidence | one frame sync within 2 dibits; present for 500 ms | `ysf_gate.go` (`YSFCallDetector`, `ysfCallHold`) |
| YSF timing | feed-forward acquisition: two agreeing 96-symbol windows seed the loop | `internal/radio/ysf/receiver/receiver.go` (`ArmTimingAcquisition`) |
| NXDN evidence | a sync, a traffic LICH with good parity, the next sync exactly one frame later | `nxdn_gate.go` (`step`, `nxdnCand`) |
| NXDN windows | scan-window hold of 500 ms for NXDN96, 1 s for NXDN48 | `nxdn_gate.go` (`nxdnGateMinDwell`, `nxdn48GateMinDwell`) |
| D-STAR evidence | CRC-valid header, or slow-data syncs one or two cycles apart; the end pattern closes | `dstar_gate.go` (`onBits`, `syncCycleGap`) |
| D-STAR lanes | two receivers, the second fed half a symbol later | `dstar_gate.go` (`dstarGateLane`) |
| Real-air pins | NXDN96 and NXDN48 voice open; a control channel never does | `nxdn_gate_test.go`, `nxdn_scanner_test.go` |
| Not-a-call pin | the reporter's analog-tone capture opens no gate | `testdata/ysf_issue1261_analog_tone_448900k_48k.cs16` |

## In this post

- **The capture that opened a "ysf" call** — and the gate it led to.
- **YSF: late to its own window** — the first sync word and feed-forward timing.
- **NXDN: why one sync word is not enough** — the pair, the LICH, the measured open times.
- **D-STAR: a header, or two slow-data syncs** — both polarities, and the end pattern.
- **Two receivers, half a symbol apart** — the Mueller-Müller hang.
- **What has heard a radio** — the honest status of each gate.

## The capture that opened a "ysf" call

On 11 Nov the reporter on
[#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261) posted the
project's first on-air System Fusion capture, with the complaint that a
`ysf` scan-list channel had opened a call and recorded no sound. The
capture was not System Fusion. It was an analog FM carrier with a 1 kHz
test tone, 1.4 kHz off frequency, with 57 % of its IQ samples at the ADC
rail — the
[clipping]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }})
of a handheld beside the dongle
([Rebuilt From the Air Part 13]({{ '/blog/solution-postmortem/rebuilt-from-the-air-13-read-the-capture-first/' | relative_url }})
tells how that was read off the file). The voice chain had decoded nothing,
correctly — and logged nothing that said why. And the scanner had opened a
"ysf" call on it, because `ysf`, like `nxdn` and `dstar`, squelched on
in-channel power alone.

Two things changed. The chain's ended line now carries `sync_hits` and
`adc_rail_pct`, with a WARN when a call had no frame sync at all. And the
channel got a gate in the shape Part 8 gave DMR:

```go
// internal/scanner/conventional/ysf_gate.go
func (d *YSFCallDetector) onDibits(dibits []uint8, _ int) {
    d.hits, d.pos = d.det.Process(d.hits[:0], dibits, d.pos)
    if len(d.hits) > 0 {
        d.syncs += len(d.hits)
        d.lastSync = d.samples
    }
}
```

That is the whole rule. The production
[System Fusion]({{ '/reference/system-fusion-ysf/' | relative_url }})
receiver — carrier correction and channel filter on, as the voice chain
runs it — turns the channel's IQ into dibits; a sync detector hunts the
20-dibit frame sync word at the voice chain's tolerance of 2; and any hit
keeps the gate present for `ysfCallHold` = 500 ms. Every System Fusion
frame is 480 dibits, 100 ms, and opens with that word, so the hold rides
out four lost frames. One hit is enough because a chance match at that
tolerance is about one in 10⁹ dibit positions.

`TestScannerYSFAnalogCarrierOpensNoCall` is the failing-first pin: an
analog tone at the scanner's 2.4 MS/s opened a call before the gate and
opens none with it. `TestYSFCallDetectorIgnoresRealAirAnalogTone` runs a
committed one-second slice of the reporter's capture itself, and
`TestYSFCallDetectorIgnoresAnalogCarrier` adds an on-frequency tone, an
unmodulated carrier and a wide 300 Hz tone.

## YSF: late to its own window

Building the gate exposed the problem Part 8 met on DMR. Started on a
transmission at ten sub-sample phases, the receiver found its first frame
sync after 0.11 s at the best of them and after **0.6–1.2 s at three** — a
[Mueller-Müller]({{ '/reference/mueller-muller-timing-recovery/' | relative_url }})
loop
([SDR Internals Part 7]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }}))
pulls a bad start phase in at gain times error per symbol, and near a
half-symbol offset the error is nearly nothing. The gate has
`ysfGateMinDwell` = 500 ms to open in.

The YSF receiver therefore grew the DMR receiver's feed-forward acquisition
([DMR End to End Part 9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})),
as an opt-in `ArmTimingAcquisition()`. It skips ten symbols of
matched-filter transient, measures 96-symbol windows with
`sync.EstimateSymbolPhase`, and seeds the loop once **two consecutive
windows agree within one sample** — one window is not evidence, because a
keyup transient can pass the estimator a couple of samples off. After four
disagreeing windows it gives up and the loop pulls in as before. Unarmed,
the receiver is byte-identical to its old self. Armed — by the gate after
`New` and every `Reset`, and by the voice chain at a grant — the first sync
arrives after **0.11 s at every phase**, and
`TestYSFCallDetectorAcquiresAtAnySymbolPhase` requires the gate to open
inside its window from all ten.

## NXDN: why one sync word is not enough

An NXDN frame is 192 dibits: a 10-dibit frame sync, an 8-dibit LICH, the
SACCH and the information field
([Rebuilt From the Air Part 7]({{ '/blog/solution-postmortem/rebuilt-from-the-air-07-nxdn-frame-layer/' | relative_url }})
has the layout, and why the old one was wrong). Ten dibits is a short word.
At the framer's cold-search tolerance of one wrong dibit, 31 of the 4¹⁰
possible windows match — about 3·10⁻⁵ per position. At NXDN96's 4800 dibits
a second that is one false sync roughly every 7 s on any carrier that
produces random-looking dibits: often enough to open on an analog carrier.

So the evidence is deliberately two-part, and a frame counts only when both
hold:

- a second sync follows **exactly one frame later** — where a chance match
  almost never lands, which lets that position accept three wrong dibits;
- the frame's descrambled LICH has good parity and names a **traffic**
  channel: RTCH, RDCH or the composite type, not the control channel.

```go
// internal/scanner/conventional/nxdn_gate.go (shape)
if errs <= nxdnGatePairTolerance {
    for _, c := range d.cand {
        if d.n == c.end+nxdn.AirFrameDibits && c.lichRead && c.traffic {
            confirmed = true // a sync exactly one frame after a candidate
        }
    }
}
if errs <= nxdnGateColdTolerance || confirmed {
    d.cand = append(d.cand, nxdnCand{end: d.n})
}
```

Each confirmed frame starts the candidate for the next, so a transmission
keeps confirming frame by frame, and the gate stays present for 500 ms
(`nxdnCallHold`) after the last one — about twelve NXDN96 frames or six
NXDN48.

The LICH test is what makes a control channel safe to leave on a list. An
RCCH is perfectly good NXDN — syncs every 192 dibits, LICH parity fine — but
its LICH says "control", so nothing confirms. The same construction settles
the wrong-width case: a gate built for 2400 symbols a second never frames
NXDN96, so an `nxdn48` channel carrying NXDN96 audio, or the reverse, now
never opens instead of recording silence.

The scan-window holds were measured, not chosen. On the reporter's real-air
slices, at 48 kHz and at 2.4 MS/s, from ten start offsets, **NXDN96
confirmed its first frame after 60–100 ms and NXDN48 after 440–770 ms** —
its frames are 80 ms long, and the carrier tracker was pulling in that
capture's roughly 400 Hz offset against NXDN48's ±350 Hz inner deviation.
Hence `nxdnGateMinDwell` = 500 ms and `nxdn48GateMinDwell` = 1 s.

One detail is specific to this receiver. `Reset` builds a **fresh**
receiver instead of resetting the old one, because `nxdnrx.Receiver.Reset`
keeps the timing loop and the filter histories — the next channel would
start from the previous channel's symbol phase.

## D-STAR: a header, or two slow-data syncs

The D-STAR gate could not exist until the format behind it was rebuilt.
GopherTrunk's old D-STAR sync words matched nothing a radio sends
([Rebuilt From the Air Part 10]({{ '/blog/solution-postmortem/rebuilt-from-the-air-10-dstar/' | relative_url }})),
so a gate on them would have blocked every real transmission.

[D-STAR]({{ '/reference/d-star/' | relative_url }}) is two-level GMSK at
4800 bit/s, so this gate counts bits, and it accepts either of two kinds of
evidence, in either polarity:

- **A header whose CRC matches.** A frame sync (two errors allowed) starts
  collecting the 660-bit header section; if `dstar.DecodeHeaderBurst`
  returns a valid header, a transmission has just begun.
- **Two slow-data syncs at the right spacing.** A voice stream carries a
  slow-data sync every 21 frames — 2016 bits, 420 ms. Two of them one or two
  cycles apart, within two bits of slip and in the same polarity, mean the
  scanner has joined a transmission already under way.

The pair is there for the same reason as NXDN's. At the search tolerance of
one error a lone slow-data sync matches 25 of 2²⁴ windows per polarity,
which in noise is about once a minute; a second landing in the ten bit
positions the rule accepts after it is about one in 10⁵ of those. Allowing
*two* cycles means a sync lost while the receiver settles costs one sync,
not the call. Both polarities are searched because the reference firmware
reads a negative sample as a one and GopherTrunk's receiver slices the
other way.

D-STAR also gives the gate something no other mode does: the
end-of-transmission pattern. Hearing it clears the evidence at once, in
both lanes, instead of waiting out `dstarCallHold` (1 s, one lost sync's
worth). The scan-window hold is `dstarGateMinDwell` = 1.2 s — joining
mid-transmission needs two syncs 420 ms apart after the receiver settles,
and on synthetic transmissions from twenty start offsets at 2.4 MS/s, in
both polarities, the gate opened within 0.9 s.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Three rows comparing the evidence each call gate needs. System Fusion: one frame sync at the head of any 100 millisecond frame. NXDN: a frame sync, a LICH with good parity naming a traffic channel, and another frame sync exactly 192 dibits later. D-STAR: either a header whose CRC matches, or two slow-data syncs 2016 bits apart.">
  <text x="10" y="42" fill="currentColor" font-size="9" font-weight="bold">System Fusion</text>
  <rect x="110" y="26" width="130" height="24" fill="none" stroke="var(--fg-muted)"/>
  <rect x="240" y="26" width="130" height="24" fill="none" stroke="var(--fg-muted)"/>
  <rect x="370" y="26" width="130" height="24" fill="none" stroke="var(--fg-muted)"/>
  <rect x="500" y="26" width="130" height="24" fill="none" stroke="var(--fg-muted)"/>
  <rect x="110" y="26" width="14" height="24" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <rect x="240" y="26" width="14" height="24" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <rect x="370" y="26" width="14" height="24" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <rect x="500" y="26" width="14" height="24" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="182" y="41" text-anchor="middle" fill="var(--fg-muted)" font-size="8">frame = 100 ms</text>
  <text x="110" y="64" fill="var(--accent)" font-size="8">any one 20-dibit frame sync → present for 500 ms</text>
  <line x1="10" y1="78" x2="670" y2="78" stroke="var(--fg-muted)" stroke-dasharray="2 4"/>
  <text x="10" y="116" fill="currentColor" font-size="9" font-weight="bold">NXDN</text>
  <rect x="110" y="100" width="290" height="24" fill="none" stroke="var(--fg-muted)"/>
  <rect x="110" y="100" width="15" height="24" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <rect x="125" y="100" width="12" height="24" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <rect x="400" y="100" width="230" height="24" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <rect x="400" y="100" width="15" height="24" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="268" y="115" text-anchor="middle" fill="var(--fg-muted)" font-size="8">SACCH + information — frame = 192 dibits</text>
  <text x="110" y="94" fill="var(--accent)" font-size="8">sync, at most 1 wrong dibit</text>
  <text x="125" y="138" fill="currentColor" font-size="8">LICH: parity good, traffic channel</text>
  <text x="400" y="138" fill="var(--accent)" font-size="8">sync again, exactly one frame later → confirmed</text>
  <line x1="10" y1="152" x2="670" y2="152" stroke="var(--fg-muted)" stroke-dasharray="2 4"/>
  <text x="10" y="190" fill="currentColor" font-size="9" font-weight="bold">D-STAR</text>
  <line x1="110" y1="186" x2="630" y2="186" stroke="var(--fg-muted)"/>
  <rect x="110" y="174" width="90" height="24" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="155" y="189" text-anchor="middle" fill="currentColor" font-size="8">header, 660 bits</text>
  <line x1="290" y1="172" x2="290" y2="200" stroke="var(--accent)" stroke-width="2"/>
  <line x1="500" y1="172" x2="500" y2="200" stroke="var(--accent)" stroke-width="2"/>
  <line x1="294" y1="180" x2="496" y2="180" stroke="var(--fg-muted)"/>
  <text x="395" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="8">2016 bits = 420 ms</text>
  <text x="110" y="214" fill="currentColor" font-size="8">CRC matches → confirmed</text>
  <text x="290" y="214" fill="var(--accent)" font-size="8">two slow-data syncs, one or two cycles apart → confirmed</text>
  <text x="110" y="236" fill="var(--fg-muted)" font-size="8">either polarity · the end pattern closes the gate at once</text>
</svg>
<figcaption>One sync word, a confirmed pair with a field check, or a checksum: each gate asks for exactly as much as its protocol's sync word cannot prove alone.</figcaption>
</figure>

## Two receivers, half a symbol apart

D-STAR met the timing wall too, in its own form. From the start phase half
a symbol off the Mueller-Müller loop's stable point — one start in ten at
ten samples per symbol — the loop hung for one or two sync cycles before it
pulled in, and the gate took **1.15–1.57 s** to open. The answer here is
not an estimator but a second receiver:

```go
// internal/scanner/conventional/dstar_gate.go (shape)
halfSymbol := int(d.fe.rate / dstarrx.SymbolRate / 2)
for l := range d.lanes {
    d.lanes[l] = &dstarGateLane{d: d, delay: l * halfSymbol}
}
// reset(): a fresh receiver, primed with the lane's delay
if ln.delay > 0 {
    ln.rx.Process(make([]complex64, ln.delay))
}
```

Both lanes see the same channel IQ, the second one half a symbol later, so
whichever phase the scanner lands on, one of them starts near its best.
Either lane's evidence confirms, and at 4800 bit/s the second receiver
costs little. The voice chain had the same hang — over a second of a joined
call lost from that one phase — and now runs the same pair
(`dstarVoiceLanes`), with a rule for which lane reaches the recorder.

### How three evidence rules shaped the Go code

- **One interface, three state machines.** Each detector adds `gateName()`
  and `minDwell()`; the scanner never learns which protocol it is holding a
  window for.
- **Tolerances come with their arithmetic.** The comments beside
  `nxdnGateColdTolerance` and `syncCycleGap` state the false-alarm rate that
  justifies the rule, so the next change argues with a number.
- **Reset is defined by what the next channel must not inherit.** YSF
  re-arms its acquisition; NXDN and D-STAR build fresh receivers because a
  reset would carry the symbol phase across the retune.
- **Negative fixtures are first-class.** One real-air slice — the capture
  that was not System Fusion — is a must-not-open case for all three gates.

## What has heard a radio

**NXDN** is the one with air behind it. The gate's fixtures are 48 kS/s
slices of the reporter's Kenwood captures, shared with the `nxdn` package's
own tests: an NXDN96 and an NXDN48 conventional voice call on 447.1 MHz,
about 400 Hz off as captured, and an idle NXDN96 control channel on
453.775 MHz. `TestNXDNCallDetectorOpensOnRealAirVoice` interpolates them to
2.4 MS/s and requires an open inside the mode's window from five start
offsets; `TestScannerNXDNControlChannelOpensNoCall` and
`TestScannerNXDNAnalogCarrierOpensNoCall` run the whole scanner, and both
open a call with the gate unwired. That is capture-verified; a live
scan-list run of the gate is not recorded. The operator-side setup is in
[the NXDN guide]({{ '/nxdn-scanner/' | relative_url }}) and
[Cookbook Volume 2 Part 3]({{ '/blog/tutorials/cookbook-volume-2-03-nxdn/' | relative_url }}).

**System Fusion** is half-pinned. The closed side is real air: the capture
that started this. The open side is synthetic — frames assembled around
[literal]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
FICH vectors from the reference encoder
([Rebuilt From the Air Part 6]({{ '/blog/solution-postmortem/rebuilt-from-the-air-06-ysf-fich/' | relative_url }})).
A real DN-mode capture is still the open item, and the gate opens on any
System Fusion frame while the voice chain decodes V/D mode 2 only.

**D-STAR** is synthetic-verified only. Its format is reference-pinned to
the MMDVM firmware, but no D-STAR capture exists in the project, and
`config.example.yaml` calls voice decode on both modes experimental and
unverified on air. The earlier tour of these modes,
[Beyond Voice Part 13]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }}),
predates both rebuilds.

## Where this goes next

Every gate so far assumes two things it has not earned: that a call's
carrier is continuous, and that the IQ stream keeps arriving.
[Part 10]({{ '/blog/deep-dives/squelch-10-bursty-carriers/' | relative_url }})
is about the day each assumption failed — a DMR handheld that is off for
half of every frame, a stream that stopped without closing, and decoders
that gave up on the first failed tune.

## FAQ

**Why did my System Fusion scan-list channel record silence?**
Before the frame-sync gate, a `ysf` channel opened a call on any carrier
above its squelch, and the voice chain decoded nothing from a carrier that
was not System Fusion. The channel now opens only on a System Fusion frame
sync, and a call with no sync at all logs a WARN naming `adc_rail_pct`.

**Why does the NXDN call gate need two frame syncs instead of one?**
The NXDN frame sync is only 10 dibits. At one wrong dibit of tolerance it
matches about 3·10⁻⁵ of random positions — one false sync roughly every
seven seconds at NXDN96. A second sync exactly 192 dibits later, plus a
parity-clean LICH naming a traffic channel, is what chance does not
produce.

**Will an NXDN control channel on my scan list open calls?**
No. A control channel frames as NXDN, but its LICH names the control
channel, and the gate confirms only traffic frames. The reporter's real
453.775 MHz control-channel capture opens no call through the whole
scanner.

**Why does the D-STAR gate run two receivers?**
From the start phase half a symbol off its stable point, the
Mueller-Müller timing loop hung for one or two slow-data sync cycles and
the gate took 1.15–1.57 s to open. The second receiver is fed half a symbol
later, so one of the two always starts near its best phase.

**Are the YSF, NXDN and D-STAR call gates verified on air?**
NXDN is capture-verified on real-air slices of two voice calls and a
control channel. The YSF gate is pinned on a real capture it must ignore and
on synthetic frames it must open on; the D-STAR gate is synthetic-verified
only. None has a live scan-list run on record.

## Series navigation

**Part 9 of 14** · ←
[Part 8: Call Gates I — An Idle Repeater Is Not a Call]({{ '/blog/deep-dives/squelch-08-call-gates-p25-dmr/' | relative_url }})
· Next →
[Part 10: Bursty Carriers — Direct Mode, Stalled Streams and Decoders That Give Up]({{ '/blog/deep-dives/squelch-10-bursty-carriers/' | relative_url }})
