---
title: "Squelch, Part 4: Offset Tuning — The Tone a Clipped ADC Adds"
description: "An overloaded SDR tuned exactly onto an analog FM channel adds a tone at four times the carrier offset: per-axis ADC clipping squares the constellation and the discriminator hears it. GopherTrunk's conventional scanner now tunes its LO below the channel at a searched offset — never sample_rate/4 — and mixes back. Simulated, not yet confirmed on air."
category: deep-dives
keywords: sdr offset tuning, lo offset tuning rtl-sdr, adc clipping fm audio tone, zero-if dc spur fm, whistle on fm audio sdr, sample rate over four offset, iq imbalance image fm, clipped adc discriminator tone, scanner lo offset hz, gophertrunk offset tuning
tags: [squelch, conventional, offset-tuning, clipping, dsp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Squelch"
series_part: 4
---

*Part 4 of **Squelch**, a 14-part deep dive into GopherTrunk's conventional
scanner — the scan list with no control channel to tell it a call exists —
and every gate that decides when a channel opens.
[Part 3]({{ '/blog/deep-dives/squelch-03-noise-squelch/' | relative_url }})
added a noise squelch to the level squelch. Both of those, and every tone
gate after them, measure "the channel" — and assume the tuner has put
nothing of its own there. This part is the one that is not a gate: where
the scanner points its local oscillator, and what
[#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184) found
sitting in the channel when it pointed straight at it.*

> **TL;DR:** A reporter's analog FM recordings carried "noise/tone" on a
> strong, on-frequency signal. The cause was zero-IF tuning of an
> overloaded ADC: per-axis clipping squares the constellation
> (|IQ| ∈ [1.0, 1.414]), and the discriminator then emits a tone at
> **4× the residual carrier offset** — 18–25 dB above the voice floor on the
> Kenwood captures, absent on the unclipped Radtel. `convScannerFrontEnd`
> (`cmd/gophertrunk/conv_offset.go`) now tunes the LO *below* the channel
> and NCO-mixes back, for the scanner and the FM chain alike, at an offset
> searched so the DC spur, the I/Q image and the clipping products at
> 4k·offset (mod fs) all miss the channel: 518 kHz at 2.4 MS/s. Never fs/4:
> 4·(fs/4) ≡ 0 mod fs, simulated SINAD 0.5 dB against 56 dB. Still
> on-air-gated.

**Key takeaways**

- **Tuned exactly on channel, a tuner's flaws land in the audio.** The DC
  spur, the I/Q image and clipping products all sit within a few multiples
  of the carrier offset — a few hundred hertz — which is the voice band.
- **Clipping has a fingerprint.** Samples pinned between 1.0 and 1.414 in
  magnitude, and an audio line at exactly four times the carrier offset.
  Look for both before any other theory.
- **The obvious offset is the worst one.** At a quarter of the sample rate
  the third-order clipping product aliases exactly back onto the channel.
  The offset has to be searched, not assumed.
- **Offset tuning hides the artefact, not the overload.** A clipped front
  end still loses weak signals; the scanner watches the raw samples and says
  so.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Offset tuning | LO placed below the channel; the stream is mixed back with an NCO | `convScannerFrontEnd` in `cmd/gophertrunk/conv_offset.go` |
| Offset search | scores every 1 kHz step from 150 kHz to 35 % of the rate, keeps the clearest | `pickClipSafeLOOffsetHz`, `loOffsetClearanceHz` |
| Products modelled | DC spur, I/Q image, clipping harmonics to k = 6, each with its deviation spread | `convOffsetMaxOrder`, `convOffsetDevHz`, `convOffsetChanHalfHz` |
| Voice path | the FM chain's broker subscription is mixed back by the same offset | `convScanVoiceSource`, `mixToChannel` |
| Config | zero searches, a positive value pins, a negative value disables | `scanner.lo_offset_hz` |
| Overload watch | WARN when over 0.2 % of raw samples sit at the rail in a one-second window | `observeRaw`, `convClipWarnFrac` |
| Regression | a simulated clipped 8-bit front end: on-channel, fs/4 and the searched offset | `TestConvScannerOffsetTuningRemovesClippingWhistle` |

## In this post

- **A tone that tracked the transmitter** — the report, the fingerprint, and why a first look missed it.
- **Why a square constellation sings at 4δ** — the arithmetic of per-axis clipping.
- **Three products, one cure** — the DC spur, the image, the clipping harmonics, and the wrapper that moves all three.
- **Never fs/4** — why the offset is searched, and what the search scores.
- **The pin and the WARN** — the simulated front end, and what the scanner tells the operator.
- **What is verified** — and the same exposure left in `dc_avoid`.

## A tone that tracked the transmitter

The report on #1184, dated 23 Sep in the project's engineering notes, was
analog FM with "noise/tone on an on-frequency signal": Kenwood radios keyed
on 447.100 MHz came out of the recorder with a high-pitched tone under the
voice. The same channel in SDR# sounded clean. A cheaper handheld on the
same frequency sounded *better* than the Kenwoods.

An earlier pass over the reporter's six captures had concluded that all of
them decoded cleanly, and that conclusion was wrong for a specific reason:
nobody had checked how many samples sat at the ADC rail, and nobody had
searched the demodulated audio for a line at a multiple of the carrier
offset. Both checks take minutes. Run them and the captures sort themselves:

| Capture | Raw IQ | Carrier offset δ | Line in the audio |
|---|---|---|---|
| Kenwood NX-5000 | pinned to the rail | ≈ −420 Hz | 1676 Hz |
| Kenwood NX-300 | pinned to the rail | ≈ −850 Hz | 3381 Hz |
| Radtel handheld | unclipped, −12 dBFS | — | none |

Every Kenwood file has |IQ| between 1.0 and 1.414, and carries a tone
18–25 dB above the voice floor at four times that file's carrier offset. The
Radtel file, recorded at a level the converter could hold, has no such line —
"which is why it 'sounded better'", as the comment at the top of
`conv_offset.go` puts it. The operator's own bench test pinned the same
thing from the other side: an unmodulated −60 dBm carrier exactly on channel
was noisy, and the same carrier 3.25 kHz off either way was clean.

## Why a square constellation sings at 4δ

An FM carrier is a point moving round a circle in the I/Q plane. An
overloaded converter clips I and Q *independently*, each at its own full
scale, so the circle becomes a square: magnitude 1.0 where the point crosses
an axis, √2 ≈ 1.414 in a corner. That is the fingerprint in the raw samples
([The Analog Edge Part 4]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }})
covers what overload does more generally).

A square has four-fold symmetry, so the error it introduces in the phase
repeats every quarter turn of the carrier: it is periodic in the carrier
phase with period π/2. Write the carrier's phase as θ and the fully clipped
signal as sign(cos θ) + j·sign(sin θ); its Fourier series contains the
carrier at θ and harmonics at −3θ, +5θ, −7θ and so on, which sit at −4θ,
+4θ, −8θ … *relative to the carrier*. If the transmitter is δ hertz away
from where the tuner's LO sits, those terms are 4δ, 8δ … from the carrier,
and an FM discriminator, which reports how the phase is changing, turns them
into a tone at 4|δ| and its harmonics.

With the LO on the channel, δ is whatever is left between the transmitter's
frequency and the dongle's idea of it: on a real rig, a few hundred hertz.
Four times a few hundred hertz is one to three kilohertz. −420 Hz gives a
tone near 1.68 kHz; −850 Hz gives one near 3.4 kHz. The measured lines are
1676 Hz and 3381 Hz.

## Three products, one cure

Clipping dominated these captures, but it is one of three things a
direct-conversion ([zero-IF]({{ '/reference/zero-if/' | relative_url }}))
tuner places at multiples of δ when it is tuned straight onto a carrier:

- the **DC spur** (LO leakage, converter offset) sits at the LO and beats
  with the carrier at δ;
- the **I/Q-imbalance image** of the carrier sits at −δ and beats at 2δ;
- the **clipping products** beat at 4δ and its harmonics.

All three are in the channel, and so in everything Parts 2 and 3 measure and
everything the FM chain demodulates. They share a cure, and it is the one
SDR# and SDR++ apply by default, which is why the same RF sounded clean
there: do not put the LO on the channel. `convScannerFrontEnd` wraps the
scanner SDR's IQ broker and does two things.

```go
// cmd/gophertrunk/conv_offset.go (shape)
func (f *convScannerFrontEnd) SetCenterFreq(hz uint32) error {
    f.centerHz = hz // the channel the scanner asked for, not the LO
    return f.inner.SetCenterFreq(hz - uint32(f.offsetHz))
}

func (f *convScannerFrontEnd) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
    in, err := f.inner.StreamIQ(ctx)
    if err != nil {
        return nil, err
    }
    return mixToChannel(ctx, in, f.offsetHz, f.rateHz, f.observeRaw), nil
}
```

The hardware is tuned `offsetHz` below the channel, so the channel arrives
at +offset in the raw stream; `mixToChannel` runs a `dsp.NCO`
([numerically controlled oscillator]({{ '/reference/numerically-controlled-oscillator/' | relative_url }}))
that shifts it back to the centre. The scanner — squelch, tone detectors,
data decoders — never knows. Neither does the voice path: the FM chain's
`convScanVoiceSource` subscribes to the same broker and passes its
subscription through the same `mixToChannel` with the same offset.

After the mix the three products are no longer near the carrier. The DC spur
is at −offset. The image is at −2·offset. The clipping products are at
∓4k·offset, folded into the sampled band. All that remains is to choose an
offset for which none of those folds back onto the channel.

## Never fs/4

The offset everyone reaches for is a quarter of the sample rate: it centres
the channel in one half of the band, and it is the `dc_avoid` default
elsewhere in GopherTrunk. For a clipped signal it is the worst possible
choice. The first clipping product sits at 4·offset, and
4·(fs/4) = fs ≡ 0 (mod fs). It aliases exactly back onto the channel, and so
does every higher one.

So the offset is searched. `pickClipSafeLOOffsetHz` walks from
`convOffsetMinHz` (150 kHz — far enough to put the DC spur in the stopband
of the FM chain's front-end decimator) to `convOffsetMaxFrac` (35 % of the
sample rate, clear of the tuner's band-edge roll-off) in 1 kHz steps, and
keeps the offset with the best score:

```go
// cmd/gophertrunk/conv_offset.go
func loOffsetClearanceHz(off, fs float64) float64 {
    clear := func(f, spread float64) float64 {
        return math.Abs(wrapHz(f, fs)) - convOffsetChanHalfHz - spread
    }
    score := clear(off, 0)
    score = math.Min(score, clear(2*off, convOffsetDevHz))
    for k := 1; k <= convOffsetMaxOrder; k++ {
        spread := float64(4*k+1) * convOffsetDevHz
        score = math.Min(score, clear(float64(4*k)*off, spread))
    }
    return score
}
```

The score is the distance, in hertz, by which the *nearest* product misses a
±12.5 kHz channel. Two details make it honest. `wrapHz` folds every product
into ±fs/2, because that is where sampling puts it. And each product is
given a width: an order-n product of an FM signal carries n times the
deviation, so the search assumes 5 kHz deviation and widens the product at
4k·offset by (4k+1)×5 kHz. Six orders are checked.

| Sample rate | Searched offset | Nearest product clears the channel by | The same score at fs/4 |
|---|---|---|---|
| 2.048 MS/s | 443 kHz | 238.5 kHz | negative: overlaps |
| 2.4 MS/s | 518 kHz | 290.5 kHz | negative: overlaps |
| 3 MS/s | 647 kHz | 374.5 kHz | negative: overlaps |

At 250 kS/s the range is empty and the scanner tunes on channel, as before,
logging the reason.

<figure class="lab-figure">
<svg viewBox="0 0 680 246" width="680" height="246" role="img" aria-label="Three spectra of a 2.4 megasample per second stream after the channel has been mixed back to the centre. With the local oscillator on the channel, the DC spur, the image and the clipping products all fall inside the channel. With the oscillator a quarter of the sample rate below, the DC spur and image move away but every clipping product aliases back onto the channel. With the searched 518 kilohertz offset, the DC spur sits at minus 518 kilohertz, the image at minus 1036 kilohertz, and the clipping products are spread across the band, the nearest at plus and minus 328 kilohertz, leaving the channel clear.">
  <text x="72" y="16" fill="currentColor" font-size="9" font-weight="bold">LO on the channel (zero-IF): DC spur, image and clipping products all within a few δ of the carrier</text>
  <line x1="72" y1="46" x2="648" y2="46" stroke="var(--fg-muted)"/>
  <rect x="352" y="24" width="16" height="22" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <line x1="356" y1="46" x2="356" y2="32" stroke="currentColor" stroke-width="1.5"/>
  <line x1="360" y1="46" x2="360" y2="28" stroke="currentColor" stroke-width="1.5"/>
  <line x1="364" y1="46" x2="364" y2="34" stroke="currentColor" stroke-width="1.5"/>
  <text x="376" y="38" fill="var(--accent)" font-size="8">channel (drawn wider than scale)</text>
  <text x="72" y="78" fill="currentColor" font-size="9" font-weight="bold">LO at fs/4 below (600 kHz): 4 × 600 kHz ≡ 0 mod 2.4 MHz, so the clipping products land back on the channel</text>
  <line x1="72" y1="118" x2="648" y2="118" stroke="var(--fg-muted)"/>
  <rect x="352" y="96" width="16" height="22" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <line x1="360" y1="118" x2="360" y2="100" stroke="currentColor" stroke-width="2"/>
  <line x1="216" y1="118" x2="216" y2="104" stroke="currentColor"/>
  <text x="216" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="8">DC spur</text>
  <line x1="73" y1="118" x2="73" y2="106" stroke="currentColor"/>
  <text x="78" y="103" fill="var(--fg-muted)" font-size="8">image (band edge)</text>
  <text x="376" y="108" fill="currentColor" font-size="8">every 4k·offset product</text>
  <text x="72" y="148" fill="currentColor" font-size="9" font-weight="bold">LO 518 kHz below (searched): nearest product at ±328 kHz, 290.5 kHz clear after its 25 kHz spread</text>
  <line x1="72" y1="190" x2="648" y2="190" stroke="var(--fg-muted)"/>
  <rect x="352" y="168" width="16" height="22" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <line x1="236" y1="190" x2="236" y2="174" stroke="currentColor"/>
  <text x="236" y="169" text-anchor="middle" fill="var(--fg-muted)" font-size="8">DC spur</text>
  <line x1="111" y1="190" x2="111" y2="176" stroke="currentColor"/>
  <text x="111" y="171" text-anchor="middle" fill="var(--fg-muted)" font-size="8">image</text>
  <line x1="281" y1="190" x2="281" y2="168" stroke="currentColor" stroke-width="1.5"/>
  <line x1="439" y1="190" x2="439" y2="168" stroke="currentColor" stroke-width="1.5"/>
  <line x1="203" y1="190" x2="203" y2="176" stroke="currentColor"/>
  <line x1="517" y1="190" x2="517" y2="176" stroke="currentColor"/>
  <line x1="124" y1="190" x2="124" y2="180" stroke="currentColor"/>
  <line x1="596" y1="190" x2="596" y2="180" stroke="currentColor"/>
  <line x1="99" y1="190" x2="99" y2="182" stroke="currentColor"/>
  <line x1="621" y1="190" x2="621" y2="182" stroke="currentColor"/>
  <line x1="178" y1="190" x2="178" y2="184" stroke="currentColor"/>
  <line x1="542" y1="190" x2="542" y2="184" stroke="currentColor"/>
  <line x1="256" y1="190" x2="256" y2="185" stroke="currentColor"/>
  <line x1="464" y1="190" x2="464" y2="185" stroke="currentColor"/>
  <text x="281" y="202" text-anchor="middle" fill="currentColor" font-size="8">−328 kHz</text>
  <text x="439" y="202" text-anchor="middle" fill="currentColor" font-size="8">+328 kHz</text>
  <text x="360" y="202" text-anchor="middle" fill="var(--accent)" font-size="8">clear</text>
  <text x="72" y="220" text-anchor="middle" fill="var(--fg-muted)" font-size="8">−1.2 MHz</text>
  <text x="360" y="220" text-anchor="middle" fill="var(--fg-muted)" font-size="8">channel</text>
  <text x="648" y="220" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+1.2 MHz</text>
  <text x="360" y="238" text-anchor="middle" fill="var(--fg-muted)" font-size="8">bottom row: DC spur at −518 kHz, image at −1036 kHz, clipping products at ∓4k·518 kHz for k = 1…6, folded into ±1.2 MHz</text>
</svg>
<figcaption>Moving the LO moves every front-end product, but only a searched offset moves them somewhere harmless; fs/4 folds the clipping products straight back onto the channel.</figcaption>
</figure>

## The pin and the WARN

`TestConvScannerOffsetTuningRemovesClippingWhistle` simulates the captures
rather than replaying them. Its `clippingFrontEnd` synthesises an FM
transmitter at 447.100 MHz, 420 Hz below the channel, carrying a 1 kHz tone
at 2.5 kHz deviation; overdrives it three times past full scale; adds a
small DC offset; clips I and Q separately and quantises to 8 bits. Crucially
it synthesises relative to wherever `SetCenterFreq` put the LO, so the same
"transmitter" can be received on channel or offset. The audio is then
demodulated and two things are measured: the level of the line at 4×420 Hz
relative to the wanted tone, and the audio
[SINAD]({{ '/reference/sinad/' | relative_url }}).

| LO placement | Line at 4δ, relative to the wanted tone | SINAD |
|---|---|---|
| on channel (a negative offset setting) | −23.9 dB | 0.5 dB |
| fs/4, 600 kHz | not the limit: the aliased product is | 0.5 dB |
| searched, 518 kHz | −108.0 dB | 56.1 dB |

The test fails first by construction: it asserts that the on-channel case is
still wrecked, so a fixture that stopped reproducing the damage could not
pass quietly. `TestPickClipSafeLOOffset` then checks ten sample rates from
960 kS/s to 10 MS/s: every picked offset scores clear, and none is fs/4.

The operator's side is small. `scanner.lo_offset_hz` is `0` by default,
which searches; a positive value pins the offset in hertz, refused above
35 % of the sample rate; a negative value restores on-channel tuning for an
A/B. What was chosen is logged at start-up as `conv: scanner LO offset
tuning`, with `serial`, `lo_offset_hz`, `mode` and `sample_rate_hz`.

And the scanner now says when the fingerprint is present. `observeRaw` sees
each chunk *before* the mix and counts samples whose I or Q is at or beyond
0.98 of full scale. Over 0.2 % in a one-second window raises a WARN, at most
once every five minutes:

```text
conv: front end overloaded — IQ pinned to the ADC rail; a strong nearby transmitter is clipping the SDR, which distorts analog FM audio (a whistle/tone at 4x the carrier offset when tuned on-channel). Reduce gain or add attenuation (do NOT raise gain). issue #1184
```

That WARN matters because the fix is cosmetic in one respect. In the type's
own comment: "the mix hides the artefacts but a clipped front end still
loses weak signals."
[DMR End to End Part 9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
found that a 36–40 % clipped capture can still decode; that is a statement
about one strong signal, not about anything weaker sharing the band with it.

## What is verified

The diagnosis is measured on the reporter's captures: the rail-pinned
samples and the 4δ lines are in the files. The fix is
**synthetic-verified** — a simulated front end, with the damage reproduced
and removed — and is **still on-air-gated**: the notes record no run by the
reporter on a build with it.

One exposure is deliberately left open. `dc_avoid`, the equivalent option
for control and voice SDRs, auto-selects `sample_rate/4` when
`dc_avoid_offset_hz` is `0`. That offset does its stated job — it moves the
channel off the DC spur and its own image — but for a clipped signal it has
exactly the alias described above, and it was not changed here. An operator
running `dc_avoid` on an overloaded front end can pin a different
`dc_avoid_offset_hz`; reducing the gain is the better answer.

### How offset tuning shaped the Go code

- **A wrapper, not a scanner feature.** `convScannerFrontEnd` satisfies the
  scanner's `Tuner` and `IQSource` interfaces; `scanner.go` has no notion of
  an LO offset, and its tests run without one.
- **Both consumers mix back.** The scanner reads through the wrapper; the
  voice chain's subscription goes through `mixToChannel` with the offset
  recorded for that serial. `mixToChannel` allocates a fresh buffer per
  chunk because the broker's fan-out may share chunks between subscribers.
- **The picker is reused.** The dedicated narrowband decoders of Part 12
  call the same `pickClipSafeLOOffsetHz` to place their dongles.
- **The regression asserts its own premise.** If the simulated on-channel
  case ever stops being bad, the test fails rather than passing vacuously.

## Where this goes next

With the channel measured in the right place, by a statistic that ignores
gain, on a stream the tuner has not decorated, the next question is whose
carrier it is.
[Part 5]({{ '/blog/deep-dives/squelch-05-ctcss/' | relative_url }})
is the CTCSS gate: sub-audible tones 2.3 Hz apart, and a detector that has
to tell them from each other in a quarter of a second.

## FAQ

**Why is there a whistle or tone on my SDR's analog FM audio?**
If the tone's frequency is four times the offset between the transmitter and
your tuned frequency, the SDR's ADC is clipping. Per-axis clipping turns the
FM signal's circular constellation into a square, and an FM discriminator
turns that into a tone at 4× the carrier offset. Reduce the gain; offset
tuning moves the tone out of the channel but does not undo the overload.

**What is offset tuning in an SDR receiver?**
Tuning the hardware's local oscillator away from the wanted channel and
shifting the channel back digitally. The tuner's own artefacts — the DC
spur, the I/Q image and clipping products — are tied to the LO, so they land
away from the channel instead of inside it. GopherTrunk's conventional
scanner does this by default.

**Why not offset by a quarter of the sample rate?**
Because clipping products appear at multiples of four times the offset, and
four times a quarter of the sample rate is the sample rate itself, which
aliases to zero: the product lands exactly on the channel. In GopherTrunk's
simulation that left the audio at 0.5 dB SINAD, the same as no offset at
all, against 56 dB at the searched 518 kHz.

**How do I change or disable the scanner's LO offset?**
Set `scanner.lo_offset_hz`: `0` searches for a clip-safe offset, a positive
value pins it in hertz (at most 35 % of the sample rate), and a negative
value disables it and tunes on channel. The start-up log line `conv: scanner
LO offset tuning` reports what is in use.

**Is offset tuning verified on air?**
Not yet. The 4× tone was measured on a reporter's real captures and the fix
removes it in a simulated clipped front end, but no on-air run on a build
with the fix has been reported, so it remains on-air-gated.

## Series navigation

**Part 4 of 14** · ←
[Part 3: Noise Squelch — When an AGC Makes dBFS Meaningless]({{ '/blog/deep-dives/squelch-03-noise-squelch/' | relative_url }})
· Next →
[Part 5: CTCSS — Separating Tones 2.3 Hz Apart]({{ '/blog/deep-dives/squelch-05-ctcss/' | relative_url }})
