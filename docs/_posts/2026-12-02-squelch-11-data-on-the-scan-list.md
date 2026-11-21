---
title: "Squelch, Part 11: Data on the Scan List — MDC1200, FleetSync and ACARS Taps"
description: "How a scan-list channel runs MDC1200, FleetSync and ACARS decoders on its own IQ instead of pinning a dongle to each: a data front end that decimates to about 48 kHz behind a channel filter (0 of 2 real-air bursts raw, 2 of 2 through it), a busy decoder that holds the scan, and a coherent MSK detector for ACARS."
category: deep-dives
keywords: mdc1200 on scanner channel, fleetsync ani scan list, acars decoder sdr scanner, acars msk coherent demodulator, ptt id decode conventional scanner, fm discriminator wideband noise, acars block check parity repair, scanner data decoder dwell hold, airband acars am channel, gophertrunk data decoders
tags: [squelch, mdc1200, fleetsync, acars, scanner, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Squelch"
series_part: 11
---

*Part 11 of **Squelch**, a 14-part deep dive into GopherTrunk's conventional
scanner — the scan list with no control channel behind it — and every gate
that decides when one of its channels opens.
[Part 10]({{ '/blog/deep-dives/squelch-10-bursty-carriers/' | relative_url }})
ended with a FleetSync receiver that needed a whole dongle to itself, beside
a scanner already listening to that kind of channel. This part is what the
scanner does about it: data decoders that run on a scan-list channel's own
IQ, the three things they need from the scanner that a dedicated SDR never
had to give them, and the one decoder — ACARS — whose line code punishes
the obvious demodulator.*

> **TL;DR:** `decoders: [mdc1200, fleetsync]` on an FM scan-list channel, or
> `decoders: [acars]` on a `mode: am` one, runs them on the channel while
> the scanner is on it — no dedicated SDR. The scanner owns three things: a
> `dataFrontEnd` that decimates to about 48 kHz behind a ±8 kHz filter
> (real-air FleetSync at 2.4 MS/s under band-wide noise: 0 of 2 bursts raw,
> 2 of 2 through it); the dwell, where a decoder with a locked sync (`Busy`)
> extends the scan window up to 500 ms (1 s for ACARS); and a `Reset` on
> every retune. Decoders ignore the CTCSS/DCS gate by design. ACARS reads
> data from the MSK phase, not from tone decisions — 5 of 7 messages became
> 7 of 7 on acarsdec's test audio — and accepts a repair only when it is
> unique and printable. MDC1200 is on-air verified through this path; the
> ACARS live run is pending.

**Key takeaways**

- **A scan-list channel is already a receiver.** A data burst needs IQ on
  its channel at the moment a radio keys up; the scanner has exactly that.
- **Decoders get a channel, never the SDR's span.** An FM discriminator fed
  2.4 MHz is dominated by the band's noise; the same samples behind a
  channel filter decode every burst.
- **A burst in progress outranks the scan clock — within a bound.** A
  locked sync holds the window, and a false lock on noise can delay the
  scan by no more than the same bound.
- **A line code that signals "changed" wants a coherent detector.** Deciding
  tones turns one error into a whole inverted block; reading the phase
  keeps it one bit.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Config | decoder names on a scan-list entry; ACARS only on an AM channel | `config.example.yaml` (`decoders`), `internal/config/config_validate.go` |
| Factory | builds the dedicated-SDR receivers at the channel rate, stamped with serial and frequency | `cmd/gophertrunk/conv_decoders.go` (`convDataDecoderFactory`) |
| Front end | integer decimation to about 48 kHz, then a ±8 kHz channel filter, kept as IQ | `data.go` (`dataFrontEnd`, `pickDataDecimation`) |
| Scan hold | a decoder mid-burst extends the window in 50 ms steps, to 500 ms or 1 s | `data.go` (`dataScanHoldMax`, `acarsScanHoldMax`) |
| Dwell | a busy decoder counts as activity, so hangtime cannot cut a burst | `scanner.go` (`beginDwell`) |
| ACARS demod | envelope, 19.2 kHz audio, coherent MSK: the phase at each bit boundary | `internal/radio/acars/receiver/msk.go` (`mskDemod`) |
| ACARS repair | parity-located repair, accepted only when unique and printable | `internal/radio/acars/acars.go` (`DecodeBlock`, `plausible`) |
| Pins | real-air FleetSync at the SDR rate; the hold; the reset; garbage never validates | `data_test.go`, `data_acars_test.go`, `acars_test.go` |

## In this post

- **Why hang decoders off the scanner** — the config, the factory, the same bus.
- **The data front end** — one channel at about 48 kHz, and the 0-of-2 measurement.
- **Holding the channel for a burst** — `Busy`, the bounded hold, the reset, the tone gate.
- **ACARS: the phase instead of the tones** — the coherent MSK detector.
- **The repair guard** — unique and printable.
- **What is verified** — decoder by decoder.

## Why hang decoders off the scanner

[MDC1200]({{ '/reference/mdc1200/' | relative_url }}) and FleetSync are the
short data bursts an analog radio keys at the start or end of a
transmission to say who it is
([Beyond Voice Part 3]({{ '/blog/deep-dives/beyond-voice-03-mdc1200-template-decoder/' | relative_url }}),
[Part 4]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }})).
Their first wiring, `mdc1200.channels` and `fleetsync.channels`, pins a
whole SDR to one frequency. During the lab run in Part 10 the reporter
asked for the obvious alternative: the scanner already sits on the channel
when a radio keys up, with its IQ in hand. That became
[#1220](https://github.com/MattCheramie/GopherTrunk/issues/1220), and
ACARS ([#1231](https://github.com/MattCheramie/GopherTrunk/issues/1231))
joined later on AM channels:

```yaml
scanner:
  conventional:
    - label: "Fleet channel"
      frequency_hz: 462562500
      mode: fm
      squelch_dbfs: -50
      decoders: [mdc1200, fleetsync]
    - label: "ACARS primary"
      frequency_hz: 131550000
      mode: am
      decoders: [acars]
```

An unknown or repeated name is rejected, `acars` is refused on anything
but `mode: am`, and the digital modes refuse `decoders` altogether.

Behind the config, `convDataDecoderFactory` builds the very receivers the
dedicated sections run — there is no second implementation to drift —
handing each the channel's sample rate, the scanner's SDR serial and the
channel frequency. They publish on the same bus kinds, so the rest of
[Beyond Voice Part 1]({{ '/blog/deep-dives/beyond-voice-01-eleven-place-pattern/' | relative_url }})'s
wiring — the log tables, REST endpoints and web panels of the
[MDC1200]({{ '/mdc1200.html' | relative_url }}),
[FleetSync]({{ '/fleetsync.html' | relative_url }}) and
[ACARS]({{ '/acars.html' | relative_url }}) guides — needed nothing new,
and a burst heard by a hopping scanner stays attributable by its frequency
stamp. What the scanner asks of a decoder is three methods:

```go
// internal/scanner/conventional/data.go (shape)
type DataDecoder interface {
    ProcessIQ(iq []complex64) // channel IQ, at the rate the factory was given
    Busy() bool               // locked a sync word, part-way through a burst
    Reset()                   // the scanner calls it on every retune
}
```

Each method stands for something the scanner owns so the decoder does not
have to: the sample rate, the dwell, and the retune.

## The data front end

The scanner's IQ arrives at the SDR's full rate and spans the whole tuner
band. A receiver written for a dedicated dongle discriminates whatever it
is given — and an FM discriminator run on 2.4 MHz is dominated by the
band's noise, while every radians-per-sample threshold behind it reads
fifty times off, the trap the CTCSS gate fell into
([Part 5]({{ '/blog/deep-dives/squelch-05-ctcss/' | relative_url }})).

`dataFrontEnd` puts one clean channel in front of the decoders.
`pickDataDecimation` chooses the largest integer factor that divides the
SDR rate exactly, keeps
the output at or above 48 kHz, and leaves the decoders' own resamplers —
down to their 9600 Hz tone-discriminator rate — a small ratio:

| SDR rate | Decimation | Channel rate |
|---|---|---|
| 2.4 MS/s | 50 | 48 kHz |
| 2.048 MS/s | 40 | 51.2 kHz |
| 3.2 MS/s | 64 | 50 kHz |
| 1 MS/s | 20 | 50 kHz |

An anti-alias resampler does the decimation and a 63-tap Kaiser low-pass
cuts the result to ±8 kHz — the same channel the tone detectors isolate.
The output stays complex, because each decoder brings its own demodulator:
an FM discriminator for the two FFSK protocols
([Beyond Voice Part 2]({{ '/blog/deep-dives/beyond-voice-02-afsk-ffsk-fundamentals/' | relative_url }})),
an envelope detector for ACARS.

The measurement that justifies it is
`TestDataFrontEndDecodesRealAirFleetSyncAtSDRRate`. It takes the two
committed real-air FleetSync slices — Fleet 107, Unit 1772 — interpolates
them back to the 2.4 MS/s a scanner SDR delivers, and adds white noise
across the *whole* band at 0 dB against the signal. Only a small share of
that noise falls inside the channel, so in-channel the signal is
comfortable. Fed raw to the dedicated-SDR receiver, the stream decodes
**0 of 2 bursts** on both slices; through the front end, the same samples
decode **2 of 2**.
`TestScannerDecodesRealAirFleetSyncOnChannel` then runs a slice through the
whole scanner.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="A block diagram of a scan-list data tap: the scanner's full-rate IQ passes through the data front end, which decimates by fifty to 48 kilohertz and filters to plus or minus 8 kilohertz, then feeds three decoders that publish to the events bus. A feedback path carries Busy back to the scanner, holding the scan window.">
  <rect x="10" y="70" width="104" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="62" y="87" text-anchor="middle" fill="currentColor" font-size="9" font-weight="bold">scanner visit</text>
  <text x="62" y="101" text-anchor="middle" fill="var(--fg-muted)" font-size="8">IQ at 2.4 MS/s, whole span</text>
  <line x1="114" y1="90" x2="134" y2="90" stroke="currentColor"/>
  <rect x="134" y="70" width="112" height="40" rx="5" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="190" y="87" text-anchor="middle" fill="var(--accent)" font-size="9" font-weight="bold">dataFrontEnd</text>
  <text x="190" y="101" text-anchor="middle" fill="var(--fg-muted)" font-size="8">÷50 → 48 kHz · ±8 kHz</text>
  <line x1="246" y1="90" x2="270" y2="41" stroke="currentColor"/>
  <line x1="246" y1="90" x2="270" y2="90" stroke="currentColor"/>
  <line x1="246" y1="90" x2="270" y2="139" stroke="currentColor"/>
  <rect x="270" y="24" width="190" height="34" rx="5" fill="none" stroke="currentColor"/>
  <text x="365" y="45" text-anchor="middle" fill="currentColor" font-size="8">mdc1200 — FM discriminator → FFSK</text>
  <rect x="270" y="73" width="190" height="34" rx="5" fill="none" stroke="currentColor"/>
  <text x="365" y="94" text-anchor="middle" fill="currentColor" font-size="8">fleetsync — FM discriminator → FFSK</text>
  <rect x="270" y="122" width="190" height="34" rx="5" fill="none" stroke="currentColor"/>
  <text x="365" y="143" text-anchor="middle" fill="currentColor" font-size="8">acars — envelope → coherent MSK</text>
  <line x1="460" y1="41" x2="484" y2="90" stroke="currentColor"/>
  <line x1="460" y1="90" x2="484" y2="90" stroke="currentColor"/>
  <line x1="460" y1="139" x2="484" y2="90" stroke="currentColor"/>
  <rect x="484" y="70" width="186" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="577" y="87" text-anchor="middle" fill="currentColor" font-size="9" font-weight="bold">events bus</text>
  <text x="577" y="101" text-anchor="middle" fill="var(--fg-muted)" font-size="8">log table · REST · web panel</text>
  <path d="M365 156 L365 180 L62 180 L62 110" fill="none" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <text x="213" y="174" text-anchor="middle" fill="var(--accent)" font-size="8">Busy()</text>
  <text x="62" y="198" fill="var(--accent)" font-size="8">a locked sync holds the scan window up to 500 ms (1 s with acars) and counts as dwell activity</text>
  <text x="62" y="214" fill="var(--fg-muted)" font-size="8">Reset() on every retune · fed from the level squelch, not from the CTCSS / DCS gate</text>
</svg>
<figcaption>The decoders are the dedicated-SDR receivers unchanged; what the scanner adds is a channel-rate front end in front of them and a bounded say over when it may leave.</figcaption>
</figure>

## Holding the channel for a burst

A dedicated receiver never leaves its channel. A scanner leaves after a
scan window of 100 ms by default, and the longest burst either FFSK decoder
frames is about 250 bits — some 210 ms at 1200 baud. Three rules keep a
burst whole.

**It is seen from its first bit.** In the scan window the decoders are fed
every chunk that clears the level squelch, including the chunk that opens
it — and, on an FM channel, the chunks that arrive while the noise squelch
of [Part 3]({{ '/blog/deep-dives/squelch-03-noise-squelch/' | relative_url }})
is still making its first reading. `TestDataDecoderSeesSquelchBreakingChunk`
pins it.

**A decoder mid-burst holds the window.** When the deadline arrives and a
decoder reports `Busy`, the window is pushed out instead:

```go
// internal/scanner/conventional/scanner.go (shape)
case <-deadline.C:
    if cd != nil && held < cd.holdMax && cd.busy() {
        held += dataScanHoldStep // 50 ms
        deadline.Reset(dataScanHoldStep)
        continue
    }
```

`holdMax` is `dataScanHoldMax` = 500 ms, or `acarsScanHoldMax` = 1 s on a
channel with the ACARS decoder, whose longest block runs about 0.86 s. The
bound is the point: a false sync lock on noise can delay the scan only that
long. Inside a dwell the same flag counts as channel activity, so hangtime
cannot end the call mid-burst (`TestDataDecoderBusyHoldsHangtime`).

**Every retune is a clean start.** `Reset` clears the resampler, the
discriminator, the timing loop and the framer, so a burst cut off by a hop
is abandoned rather than completed from the next channel's bits.

What the decoders deliberately do **not** wait for is the tone gate. A
data burst carries its own sync word and block check, so it is published
even on a channel whose CTCSS or DCS gate stays shut — the gate decides
whether a *call* opens, not whether an identity is logged.

## ACARS: the phase instead of the tones

[ACARS]({{ '/reference/acars/' | relative_url }}) is the VHF air-band data
link: 2400 bit/s
[MSK]({{ '/reference/minimum-shift-keying/' | relative_url }}) on audio
tones, amplitude-modulated onto the carrier. On the scan list it rides a
`mode: am` channel, opened by the carrier-to-noise squelch of
[Part 7]({{ '/blog/deep-dives/squelch-07-am/' | relative_url }}). The
receiver takes the envelope of the channel IQ — which needs no carrier
recovery, so a few kilohertz of tuning error costs nothing — resamples it
to 19.2 kHz, eight samples a bit, and demodulates.

The line code is the catch. A bit that *equals* the previous one is sent as
a cycle of 2400 Hz; a bit that *changed* is half a cycle of 1200 Hz. That
was not taken on trust: of the four possible tone and line-code mappings,
the reference decoder acarsdec decodes only this one from synthesised
audio. It is MDC1200's construction with the tones swapped
([Rebuilt From the Air Part 2]({{ '/blog/solution-postmortem/rebuilt-from-the-air-02-mdc1200-tones-as-data/' | relative_url }})).

The first front end was the obvious one, the tone discriminator the FFSK
decoders use. But a receiver that decides *tones* has to integrate them
back into data, and one wrong tone then inverts every later bit. On
acarsdec's own real-air test recording, a single tone error at byte 11
turned the remaining 80 bytes of a block into their complement, and the
tone path decoded **5 of the recording's 7 messages**.

Mixed down by the 1800 Hz centre, MSK's phase advances a quarter turn over
a "same" bit and retreats a quarter turn over a "changed" one. De-rotate
bit boundary *k* by *k* quarter turns and what is left is a two-point
constellation whose sign *is* the data bit:

```go
// internal/radio/acars/receiver/msk.go (shape)
rot := cmplx.Exp(complex(0, -(float64(d.k&3)*math.Pi/2 + d.theta)))
y := z * rot / complex(mag, 0)
d.k++
soft := real(y) // the data bit, up to one global sign the framer resolves
e := imag(y)
if soft < 0 {
    e = -e
}
d.theta += mskPhaseGain * e // decision-directed carrier phase
emit(soft)
```

Now a noise hit costs one bit, which parity and the block check can repair,
instead of a block. The coherent detector decodes **all 7**, and with white
noise added to the recording it decodes as many messages as acarsdec or
more at every level tried.

Timing is the other half. The loop is a
[Gardner]({{ '/reference/gardner-timing-recovery/' | relative_url }})
detector run on the *one-bit* phase difference; the per-sample frequency
was tried first and slipped bits under noise. And because a block opens
with a pre-key that is one steady tone — no transitions, no timing
information — the loop pulls in during the sync characters at a hunting
gain, then drops to a quarter of it once the framer locks.

## The repair guard: unique and printable

Every ACARS character carries an odd parity bit and every block ends in a
16-bit check, which invites repair. `DecodeBlock` will fix up to three
characters that fail parity (one wrong bit in each), one wrong bit in the
check itself, or two adjacent wrong bits inside one character.

The trap is arithmetic. A 16-bit check asked to arbitrate between *n*
candidate repairs validates a wrong one with probability about *n*/65536,
and three parity errors mean 512 candidates — roughly 0.8 %. So a repair is
accepted only when it is the **only** candidate in the searched space that
validates, and only when the repaired block reads as ACARS: printable
header, printable text. `TestRepairNeverValidatesGarbage` feeds 20 000
parity-valid garbage blocks through the corrector; without the guards 70 of
them "repair" into a valid block, with them none. The honest remainder: a
*real* block damaged beyond the searched model can still, rarely, repair to
a wrong printable text, so the number of bits repaired is carried to the
log and the panel.

### How scan-list taps shaped the Go code

- **Three methods are the whole contract.** `ProcessIQ`, `Busy` and `Reset`
  are all the scanner knows about a decoder.
- **The factory is the dedicated receiver, re-parameterised.** The input
  rate, the serial and the frequency are options, so a fix to a decoder
  lands in both wirings at once.
- **The rate is chosen, not inherited.** The factory is handed the front
  end's output rate, and `TestDataDecoderFactoryGetsChannelRate` fails if a
  decoder is ever built for the SDR rate.
- **Holds are sized by the longest frame.** Both hold constants cite the
  burst length they cover, which is also the most a false lock can cost.

## What is verified

**MDC1200 is on-air verified through this path.** On 28 Sep the reporter's
live run decoded eleven PTT ID bursts — key-up and unkey, unit 0x1777, on
447.100 MHz, narrow and wide FM — on a scanner channel. The decoder had to
be rebuilt first; Rebuilt From the Air Part 2 tells that story and
supersedes the line-code description in Beyond Voice Part 3.

**FleetSync** is on-air verified as a decoder (16 Sep, on a dedicated SDR)
and capture-verified as a tap: a real-air slice decodes through the whole
scanner. A live run of the tap is still the gate.

**ACARS** is checked against acarsdec in both directions — GopherTrunk
decodes the seven real-air messages of acarsdec's test recording to the
same fields, and acarsdec decodes GopherTrunk's synthesised audio — with
[literal]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
on-air blocks from that recording pinning the parser. acarsdec is
GPL and was read for protocol facts only; its recording is not committed.
What has not happened is the reporter's own capture through
`TestACARSReplay`, or a live scanner run.

Two limits are documented: an ACARS burst also opens an ordinary AM call,
because the channel's carrier is up, so short recordings appear beside the
decoded messages; and multi-block messages are logged block by block. The
operator-side recipes are
[Cookbook Volume 2 Part 1]({{ '/blog/tutorials/cookbook-volume-2-01-airband-rig/' | relative_url }})
for the air band and
[Part 4]({{ '/blog/tutorials/cookbook-volume-2-04-fleet-signalling/' | relative_url }})
for fleet signalling.

## Where this goes next

The data front end cured one disease for the scan-list taps: a
discriminator fed the whole SDR span. The dedicated-SDR decoders — APRS,
AIS, POCSAG and the rest — had the same disease and no front end.
[Part 12]({{ '/blog/deep-dives/squelch-12-narrowband-front-end/' | relative_url }})
gives all of them one.

## FAQ

**How do I decode MDC1200 or FleetSync without dedicating an SDR to it?**
Add `decoders: [mdc1200, fleetsync]` to the analog FM channel in
`scanner.conventional`. The decoders run on that channel's IQ whenever the
scanner is on it, and bursts land in the same log, REST endpoint and web
panel as the dedicated-SDR sections, tagged with the channel's frequency.

**Why do the data decoders ignore my CTCSS or DCS tone gate?**
By design. A data burst carries its own sync word and block check, so it is
decoded from the level squelch alone and published even when the channel's
tone gate stays shut. The tone gate still decides whether a call opens.

**Why does the scanner pause on a channel during a data burst?**
A decoder that has locked a sync word reports itself busy, and the scanner
extends the scan window in 50 ms steps — up to 500 ms, or 1 s on an ACARS
channel — so it does not hop away mid-frame.

**Why does ACARS open a voice call on my AM channel?**
An ACARS block is an AM carrier on the channel, so the carrier-to-noise
squelch opens and an ordinary AM call is recorded beside the decoded
message. It is a documented limit of running ACARS as a scan-list tap.

**Is ACARS decoding verified on air?**
Not yet by a live run. It is pinned against acarsdec in both directions and
by literal blocks from a real-air recording, on which the coherent detector
decodes all seven messages. The reporter's own capture and a live scanner
run are still open.

## Series navigation

**Part 11 of 14** · ←
[Part 10: Bursty Carriers — Direct Mode, Stalled Streams and Decoders That Give Up]({{ '/blog/deep-dives/squelch-10-bursty-carriers/' | relative_url }})
· Next →
[Part 12: One Front End for Every Narrowband Decoder]({{ '/blog/deep-dives/squelch-12-narrowband-front-end/' | relative_url }})
