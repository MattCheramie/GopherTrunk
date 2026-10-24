---
title: "From the Issue Tracker, Season 2, Part 2: A Tone at Four Times the Offset — Zero-IF Clipping on the Analog Scanner"
description: "Why a strong Kenwood on the conventional analog scanner whistled at exactly four times its carrier offset — per-axis ADC clipping squares the constellation and the discriminator turns the π/2-periodic phase error into a tone — and why the fix is a searched LO offset rather than the obvious sample_rate/4, whose third-order product aliases straight back onto the channel."
category: solution-postmortem
keywords: sdr fm whistle strong signal, zero-if clipping tone, adc clipping squares constellation, tone at 4x carrier offset, lo offset tuning analog fm, dc spur iq image fm scanner, why not fs/4 offset, scanner lo_offset_hz, rtl-sdr overload whistle, gophertrunk from the issue tracker season 2
tags: [from-the-issue-tracker-s2, analog-fm, clipping, zero-if, scanner, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker, Season 2"
series_part: 2
---

*Part 2 of **From the Issue Tracker, Season 2**, postmortems of GopherTrunk
bugs told with receipts.
[Part 1]({{ '/blog/solution-postmortem/issue-tracker-s2-01-tones-were-the-data/' | relative_url }})
followed MDC1200's silent decoder to a line code the tests had agreed on
with the decoder. This part moves to the same reporter's analog FM
channels, where the signal was decoding fine and the complaint was a
whistle — one that got worse as the radio got closer. The fingerprint was
in the constellation, the arithmetic was in the discriminator, and the
obvious fix would have reproduced the bug exactly.*

> **TL;DR:** The conventional scanner tuned the SDR's LO exactly onto each
> analog FM channel — zero-IF — so three front-end artefacts landed inside
> the audio band at frequencies set by the residual carrier offset δ: the
> DC spur beating at δ, the I/Q image at 2δ, and, dominating the reporter's
> captures, **per-axis ADC clipping at 4δ**. An overloaded 8-bit ADC clips I
> and Q independently, squaring the constellation (|IQ| ∈ [1.0, 1.414]);
> the resulting phase error is periodic in carrier phase with period π/2,
> so the discriminator emits a tone at 4·|δ|. The Kenwood NX-5000 and
> NX-300 captures carry it 18–25 dB above the voice floor at exactly four
> times each file's offset (δ ≈ −420 Hz → 1676 Hz, δ ≈ −850 Hz → 3381 Hz);
> the unclipped Radtel at −12 dBFS does not. `convScannerFrontEnd`
> (`cmd/gophertrunk/conv_offset.go`) now tunes the LO below the channel and
> NCO-mixes back, at an offset `pickClipSafeLOOffsetHz` searches so that no
> product aliases onto the channel — **never fs/4**: 4·(fs/4) ≡ 0 mod fs,
> and the simulated SINAD at fs/4 is 0.5 dB, identical to on-channel,
> against 56 dB at the searched 518 kHz. `scanner.lo_offset_hz` exposes it
> ([#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184)).

**Key takeaways**

- **Check the rail fraction before concluding "it decodes cleanly."** The
  earlier verdict on these six captures was that all of them decoded; nobody
  had looked at |IQ| or searched the audio for a 4δ line.
- **Clipping has a frequency signature, not just a level.** Independent
  clipping of I and Q is a π/2-periodic distortion of the phase, and an FM
  discriminator renders that as a tone at four times the carrier offset.
- **The product that matters wraps mod fs.** An offset of fs/4 — the
  `dc_avoid` default elsewhere — folds the third-order clipping product
  back onto the channel. The offset must be searched against every product.
- **The mix hides the artefact; it does not fix the overload.** The front
  end still WARNs on a rail-pinned window, because a clipped ADC still
  loses weak signals.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| LO placement | tunes `offsetHz` below the channel, mixes back with an NCO | `cmd/gophertrunk/conv_offset.go` (`convScannerFrontEnd`, `mixToChannel`) |
| Offset search | best minimum clearance over DC spur, image and 4k·off products | `pickClipSafeLOOffsetHz`, `loOffsetClearanceHz` |
| Search bounds | 150 kHz … 0.35·fs, 1 kHz steps, products to k = 6 | `convOffsetMinHz`, `convOffsetMaxFrac`, `convOffsetStepHz`, `convOffsetMaxOrder` |
| Config | 0 auto, > 0 pin (≤ 35 % of rate), < 0 legacy on-channel | `scanner.lo_offset_hz` (`convScannerLOOffsetHz`) |
| Overload watch | WARN when > 0.2 % of a 1 s window is rail-pinned, once per 5 min | `observeRaw`, `convClipWarnFrac`, `siglab.CountClipped` |
| Failing-first pin | on-channel SINAD ≤ 10 dB, fs/4 ≤ 10 dB, auto offset > 40 dB | `conv_offset_test.go` (`TestConvScannerOffsetTuningRemovesClippingWhistle`) |
| Picker pin | ten common rates, never fs/4, 250 kS/s → no room | `TestPickClipSafeLOOffset` |

## In this post

- **The report, and the verdict that missed it** — six captures, "all decode cleanly."
- **The fingerprint** — a squared constellation and a tone at 4δ.
- **Three products, one cause** — spur, image, clipping, all parked by one offset.
- **Why not fs/4** — the aliasing arithmetic that makes the obvious choice the worst.
- **The failing-first simulation** — an 8-bit clipping front end in a test.
- **What is verified, and what is not** — the bench test, the WARN, the open gate.

## The report, and the verdict that missed it

The reporter's Kenwood NX-5000 and NX-300 on the conventional scanner
produced a "high-pitched noise" on an on-frequency, strong signal; a Radtel
handheld on the same channel sounded better. Six captures arrived and an
earlier pass concluded that all six decoded cleanly — which, as a statement
about the FM demodulator, was true. Nobody had checked the fraction of
samples at the ADC rail, and nobody had searched the audio for a line at a
specific frequency.
[The Analog Edge Part 4]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }})
had already made the clip ratio the authoritative verdict on overload; the
rule had not been applied here.

The reporter's bench test pinned the mechanism from the other side: an
unmodulated −60 dBm carrier placed exactly on-channel was noisy, the same
carrier ±3.25 kHz off was clean, and SDR# and SDR++, which tune their LO
away from the VFO, never showed the problem. A strong hint — the artefact
depends on where the carrier sits relative to the LO — but it does not
name the product.

## The fingerprint

Plotting the Kenwood captures' IQ gave the first half of the answer: the
constellation is not a ring. Every sample sits with |IQ| between 1.0 and
1.414 — a **square**. An 8-bit ADC that is overdriven does not clip the
magnitude; it clips I and Q independently at their rails, and a unit circle
clipped per axis becomes a square inscribed in the rails' corners.

The second half is what a discriminator does with a square. The phase error
introduced by per-axis clipping depends on where the carrier is around the
circle, and it repeats every quarter turn — period π/2 in carrier phase. A
carrier offset δ sweeps the phase at δ turns per second, so the error
repeats at 4δ per second, and an FM discriminator — the derivative of phase
— renders a π/2-periodic phase ripple as a tone at **4·|δ|** plus its
harmonics. On a real rig δ is a few hundred hertz, so 4δ lands in the audio
band.

The captures confirm it line by line. The NX-5000 file sits at δ ≈ −420 Hz
and carries a tone at 1676 Hz; the NX-300 at δ ≈ −850 Hz carries one at
3381 Hz — in both cases 18–25 dB above the voice floor, at four times the
measured offset. The unclipped Radtel file at −12 dBFS shows no such line,
which is why it "sounded better". The `conv_offset.go` header comment keeps
these numbers next to the code they justify.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Left: a unit-circle constellation clipped per axis into a square, the magnitude ranging from 1.0 at the axes to 1.414 at the corners, with the phase error marked as repeating four times per turn. Right: a baseband spectrum after on-channel tuning, showing the wanted FM channel around zero with the DC spur beating at delta, the I/Q image at two delta and the clipping product at four delta, all inside the audio band for a few-hundred-hertz offset.">
  <text x="170" y="16" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">per-axis clipping: ring → square</text>
  <circle cx="170" cy="120" r="60" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <rect x="110" y="60" width="120" height="120" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <line x1="100" y1="120" x2="240" y2="120" stroke="var(--fg-muted)"/>
  <line x1="170" y1="50" x2="170" y2="190" stroke="var(--fg-muted)"/>
  <text x="236" y="66" fill="currentColor" font-size="8">|IQ| = 1.414</text>
  <text x="236" y="124" fill="var(--fg-muted)" font-size="8">|IQ| = 1.0</text>
  <path d="M170 120 L222 86" stroke="var(--accent)" stroke-width="1.2"/>
  <text x="170" y="206" text-anchor="middle" fill="var(--accent)" font-size="8">phase error repeats every π/2 → 4 cycles per carrier turn</text>
  <text x="500" y="16" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">on-channel (zero-IF) baseband</text>
  <line x1="330" y1="170" x2="670" y2="170" stroke="var(--fg-muted)"/>
  <text x="400" y="186" text-anchor="middle" fill="var(--fg-muted)" font-size="8">0 Hz</text>
  <path d="M340 168 Q400 70 460 168" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="400" y="62" text-anchor="middle" fill="currentColor" font-size="8">FM channel ±12.5 kHz</text>
  <line x1="408" y1="170" x2="408" y2="120" stroke="var(--fg-muted)"/>
  <text x="408" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="8">spur @ δ</text>
  <line x1="420" y1="170" x2="420" y2="135" stroke="var(--fg-muted)"/>
  <text x="432" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="8">image @ 2δ</text>
  <line x1="448" y1="170" x2="448" y2="86" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="470" y="82" text-anchor="middle" fill="var(--accent)" font-size="8">clip @ 4δ, +18–25 dB</text>
  <text x="560" y="186" text-anchor="middle" fill="var(--fg-muted)" font-size="8">all inside the audio band</text>
</svg>
<figcaption>Independent I and Q clipping squares the constellation; the phase error has period π/2, and the discriminator renders it as a tone at four times the carrier offset — in the audio band when the LO sits on the channel.</figcaption>
</figure>

## Three products, one cause

The clipping tone was the loudest of three artefacts zero-IF tuning puts in
the channel: the **DC spur** (LO leakage and ADC offset) beats with the
carrier at δ; the **I/Q-imbalance image** sits at −δ and beats at 2δ;
**per-axis clipping** products sit at ±4k·δ. All three depend only on where
the carrier is relative to the LO. Move the LO, and every one moves out of
the audio band together.

That is the fix. `convScannerFrontEnd` wraps the scanner SDR's broker:
`SetCenterFreq(hz)` tunes the hardware to `hz − offsetHz`, and `StreamIQ`
returns the stream mixed back up by `offsetHz` through `dsp.NCO`, so the
scanner's squelch and CTCSS/DCS detectors — and the FM voice chain, through
`convScanVoiceSource` — see an on-channel stream with the DC spur at
−offset, the image at −2·offset and the clipping products at ±4k·offset,
all far outside the channel filter.

```go
// cmd/gophertrunk/conv_offset.go (shape)
func (f *convScannerFrontEnd) SetCenterFreq(hz uint32) error {
    f.centerHz = hz                                   // the channel, not the LO
    return f.inner.SetCenterFreq(hz - uint32(f.offsetHz))
}
func (f *convScannerFrontEnd) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
    in, err := f.inner.StreamIQ(ctx)
    …
    return mixToChannel(ctx, in, f.offsetHz, f.rateHz, f.observeRaw), nil
}
```

`mixToChannel` mixes into a fresh buffer per chunk because the broker's
fan-out may share the chunk with other subscribers, and it passes the raw
pre-mix samples to `observeRaw` first — the overload watch described below.
`TestConvScannerFrontEndTunesBelowChannel` pins the direction: a 200 kHz
offset on 447.100 MHz puts the LO at 446.900 MHz.

## Why not fs/4

The scanner already had a precedent for LO offsetting. Control and voice
SDRs use `dc_avoid`, whose default offset is a quarter of the sample rate —
sensible for a DC spur, which lands at −fs/4, as far from the channel as
an offset can be. For a clipped front end it is the single worst choice.
Every product wraps modulo the sample rate, and the first clipping product
sits at 4·offset:

```text
4 · (fs/4) = fs ≡ 0  (mod fs)
```

The third-order clipping term aliases exactly onto the channel. In the
failing-first simulation the audio SINAD with the LO parked at fs/4 is
**0.5 dB — the same as on-channel**. Offsetting the LO did nothing at all,
and a fix that copied `dc_avoid` would have shipped the bug under a new
name.

So the offset is searched. `pickClipSafeLOOffsetHz` walks every candidate
from `convOffsetMinHz` (150 kHz — enough to put the DC spur into the FM
chain's decimating FIR stopband, whose transition band at 2.4 MS/s is
~100 kHz) up to `convOffsetMaxFrac`·fs (35 %, keeping the channel clear of
the tuner's band-edge roll-off) in `convOffsetStepHz` (1 kHz) steps, and
scores each with `loOffsetClearanceHz`:

```go
// cmd/gophertrunk/conv_offset.go
func loOffsetClearanceHz(off, fs float64) float64 {
    clear := func(f, spread float64) float64 {
        return math.Abs(wrapHz(f, fs)) - convOffsetChanHalfHz - spread
    }
    score := clear(off, 0)                              // DC spur at −off
    score = math.Min(score, clear(2*off, convOffsetDevHz)) // image at −2·off
    for k := 1; k <= convOffsetMaxOrder; k++ {
        spread := float64(4*k+1) * convOffsetDevHz      // (4k±1)th-order term
        score = math.Min(score, clear(float64(4*k)*off, spread))
    }
    return score
}
```

The score is the minimum, over every product, of how far its modulated
extent stays outside a ±12.5 kHz channel after wrapping; the (4k±1)th-order
term carries (4k+1)× the deviation, so each product is widened by
`convOffsetDevHz` (5 kHz, covering 25 kHz wideband FM) times its order.
Negative means at least one product overlaps the channel. The best score
wins, ties going to the smaller offset so the result is deterministic. At
2.4 MS/s the search lands on 518 kHz, and the simulated SINAD there is
56 dB. `TestPickClipSafeLOOffset` runs ten common rates from 960 kS/s to
10 MS/s, asserts every pick clears the channel, asserts none of them is
fs/4, asserts fs/4 at 2.4 MS/s scores negative, and asserts that 250 kS/s
has no room at all (the scanner then tunes on-channel, as before).

`convScannerLOOffsetHz` turns the picker into config: `scanner.lo_offset_hz`
at 0 (the default) picks automatically, a positive value pins it (refused
above 35 % of the rate), a negative value restores on-channel tuning. The
daemon logs the chosen `lo_offset_hz` and its `mode` at startup.

## The failing-first simulation

Reproducing a clipped Kenwood in a unit test needed a transmitter and a
dishonest ADC. `clippingFrontEnd` in `conv_offset_test.go` synthesises an
FM carrier at `channelHz + carrierOffHz` relative to wherever
`SetCenterFreq` put the LO, modulated by a 1 kHz tone at 2.5 kHz deviation,
with `overdrive` 3 (three times full scale) and a DC spur of 0.02 − 0.015j,
then quantises **I and Q independently** to an 8-bit grid with each axis
clamped at ±1 — the square. `demodChannelAudio` runs the same stages the
composer's FM chain runs ahead of its audio filters: a 15 kHz Kaiser
low-pass, decimation to 48 kHz, quadrature discriminator.

`TestConvScannerOffsetTuningRemovesClippingWhistle` then measures the tone
at 4·|δ| relative to the wanted 1 kHz tone, and the SINAD, under three
configurations: `lo_offset_hz` −1 (on-channel), 0 (auto), and 600 000
(fs/4 at 2.4 MS/s). The assertions are written so the test cannot pass
vacuously: on-channel must show a spur above −30 dB and SINAD under 10 dB
(otherwise "the fixture no longer reproduces the on-channel clipping damage;
the test cannot fail first"), auto must bring the spur under −45 dB and
SINAD over 40 dB, and fs/4 must stay under 10 dB — the aliased product is
expected to wreck it, and that expectation is the reason the picker exists.

The carrier offset in the fixture is −420 Hz, the NX-5000's. The spur
frequency it searches for is therefore 1680 Hz, the line the real capture
carries at 1676 Hz.

## What is verified, and what is not

The mix parks the artefacts; it does not un-clip the ADC. A rail-pinned
front end still loses weak signals, so `observeRaw` counts clipped samples
(`siglab.CountClipped`) over one-second windows and WARNs when more than
`convClipWarnFrac` (0.2 %, the ccdecoder and widebandt2 overload threshold
from #402 and #749) of a window is at the rail, at most once per
`convClipWarnGap` (5 min):

```text
WRN conv: front end overloaded — IQ pinned to the ADC rail; a strong nearby transmitter is clipping the SDR, which distorts analog FM audio (a whistle/tone at 4x the carrier offset when tuned on-channel). Reduce gain or add attenuation (do NOT raise gain). issue #1184 serial=… channel_hz=… clipped_fraction=… lo_offset_hz=…
```

`TestConvScannerFrontEndWarnsOnOverload` pins that a clean stream never
warns and a railed one warns exactly once across six windows. The WARN says
"do NOT raise gain" because the
[gain-staging]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
reflex on a noisy channel is to add gain, which is precisely backwards here.

What is verified: the mechanism, on the reporter's captures (the 4δ line at
18–25 dB, the squared constellation, the clean unclipped Radtel) and on
their bench (on-channel noisy, ±3.25 kHz clean); and the fix, in simulation
against a modelled 8-bit clipping ADC. What is **not** verified is the
offset-tuned scanner on the reporter's own rig — the daemon path, with the
real broker, the real tuner's band edge and the tone detectors downstream
of the mix. By the repository's standing rule (#764/#771) a simulated 56 dB
SINAD is not an on-air pass, so this part of #1184 remains on-air-gated.
The tone gates behind that mix had three defects of their own, and those
the reporter's live run did pin — the next two parts.

## Where this goes next

The offset mix delivers an on-channel stream to the CTCSS and DCS
detectors. On that same rig the CTCSS gate never opened on any tone — and
the cause was not RF at all but a threshold calibrated in radians per
sample at 48 kHz and fed 2.4 million samples per second.
[Part 3]({{ '/blog/solution-postmortem/issue-tracker-s2-03-radians-per-sample/' | relative_url }})
follows that number through a 50× scale error, a Goertzel bin that rounded
162.2 Hz to 160, and a magnitude floor no narrowband radio could reach.

## FAQ

**Why does a strong FM signal whistle on an SDR scanner?**
If the SDR's LO sits exactly on the channel and the front end is
overloaded, the ADC clips I and Q independently, squaring the constellation.
The phase error repeats every π/2 of carrier phase, so the discriminator
emits a tone at four times the carrier offset — in the audio band for a
few-hundred-hertz offset. GopherTrunk now tunes the LO off-channel
(`convScannerFrontEnd`) so the product lands outside the channel filter.

**What does scanner.lo_offset_hz do?**
It sets how far below each conventional channel the scanner tunes its SDR
before mixing the channel back to baseband. 0 (default) lets
`pickClipSafeLOOffsetHz` choose an offset whose DC spur, I/Q image and
clipping products all clear the channel; a positive value pins it (at most
35 % of `sdr.sample_rate`); a negative value restores legacy on-channel
tuning.

**Why not just use sample_rate/4 like dc_avoid?**
Because the first clipping product sits at 4·offset, and 4·(fs/4) is the
sample rate — it aliases exactly onto the channel. In the failing-first
simulation fs/4 gives 0.5 dB SINAD, identical to on-channel; the searched
518 kHz at 2.4 MS/s gives 56 dB. `TestPickClipSafeLOOffset` asserts fs/4 is
never chosen.

**Does the LO offset fix front-end overload?**
No. It moves the overload's audible products out of the channel; the ADC is
still clipping and still loses weak signals. `observeRaw` watches the raw
pre-mix samples and WARNs (`conv: front end overloaded`) when more than
0.2 % of a one-second window is rail-pinned. The remedy is less gain or an
attenuator, never more gain.

**Is the offset-tuning fix verified on air?**
The mechanism is: the reporter's captures carry the 4δ tone and a squared
constellation, and their bench test showed on-channel noisy, 3.25 kHz off
clean. The fix is verified in simulation (`TestConvScannerOffsetTuningRemovesClippingWhistle`).
A live run of the offset-tuned scanner on the reporter's rig is still the
open gate on this part of #1184.

## Series navigation

**Part 2 of 14** · ←
[Part 1: The Tones Were the Data — MDC1200's XOR-Precoded Line Code]({{ '/blog/solution-postmortem/issue-tracker-s2-01-tones-were-the-data/' | relative_url }})
· Next →
[Part 3: Radians per Sample — A CTCSS Gate Calibrated at 48 kHz and Fed 2.4 MS/s]({{ '/blog/solution-postmortem/issue-tracker-s2-03-radians-per-sample/' | relative_url }})
