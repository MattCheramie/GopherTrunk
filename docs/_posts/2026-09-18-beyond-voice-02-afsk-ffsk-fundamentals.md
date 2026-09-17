---
title: "Beyond Voice, Part 2: AFSK & FFSK — Two Tones, One Bit"
description: "How GopherTrunk turns an FM voice channel into a bit stream for MDC1200, FleetSync and APRS — the FM discriminator, the resample to 9600 Hz, demod.FFSK's mix-filter-discriminate tone detector, Mueller-Müller timing at eight samples per bit, and the slicer lesson that made FleetSync slice at a fixed zero while APRS tracks DC."
category: deep-dives
keywords: afsk demodulation sdr, ffsk 1200 1800 hz, bell 202 1200 2200 hz, fm discriminator audio fsk, mueller muller timing recovery, afsk slicer threshold, nrzi decode aprs, mdc1200 fleetsync dsp, tone discriminator go, gophertrunk afsk
tags: [beyond-voice, afsk, ffsk, dsp, timing-recovery, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Beyond Voice"
series_part: 2
---

*Part 2 of **Beyond Voice**, a 14-part deep dive into everything GopherTrunk
decodes that is not a trunked voice call, and the one eleven-place wiring
pattern that carries each of them from a burst on the air to a row in the web
console.
[Part 1]({{ '/blog/deep-dives/beyond-voice-01-eleven-place-pattern/' | relative_url }})
fixed the pattern and noted that the three AFSK front ends run an identical
DSP chain, differing only in tone pair, line code and slicer. This part is
that chain — the physics every in-band signalling decoder shares before a
protocol bit exists — and the place a "harmless" slicer refinement cost a
round.*

> **TL;DR:** MDC1200, FleetSync, MPT 1327 and APRS all ride as **two audio
> tones inside an FM voice channel**. GopherTrunk's front ends
> (`internal/radio/{aprs,mdc1200,fleetsync}/afsk`) run one chain: `demod.FM`
> → `dsp.RealResampler` to **9600 Hz** (1200 baud × `Oversample` 8) →
> `demod.FFSK`, which mixes the audio down by the tone midpoint, low-passes
> with a 60 dB Kaiser filter and FM-discriminates the complex baseband so the
> output is positive for mark → `sync.MuellerMuller` at 8 sps, gain 0.05 → a
> slicer. CCIR FFSK's 1200/1800 Hz tones hold exactly 1 and 1.5 cycles per
> bit; Bell 202's 2200 Hz does not. The slicer is where the front ends part:
> APRS and MDC1200 track DC with a 1/64 EMA, while FleetSync slices at
> **zero** — a 1/512 bias tracker drifted toward a 68-symbol run of space tone
> and flipped 23 bits of a 260-bit FS-II frame.

**Key takeaways**

- **In-band data is audio first.** Two tones modulate an ordinary FM
  transmitter; the receiver's first stage is the voice discriminator.
- **FFSK is AFSK with integer-cycle tones.** At 1200 baud a 1200 Hz mark is
  one cycle per bit and an 1800 Hz space is 1.5; phase stays continuous at
  every bit edge. Bell 202's 2200 Hz breaks that.
- **The tone detector is itself an FM demodulator.** `demod.FFSK` treats the
  two tones as a ±300 Hz (or ±500 Hz) deviation around a midpoint and
  discriminates the complex baseband, removing carrier offset as a side effect.
- **A slicer threshold is a protocol decision, not a DSP one.** FleetSync
  II's long space runs made a DC tracker harmful where AX.25's flags keep it
  benign.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| FM discriminator | IQ → phase derivative in rad/sample | `internal/dsp/demod/fm.go` (`demod.FM.Process`) |
| Audio-rate resampler | polyphase L/M to `baud × Oversample` | `internal/dsp/resampler_real.go` (`dsp.NewRealResampler`) |
| Tone discriminator | mix by midpoint → Kaiser LPF → FM discriminate | `internal/dsp/demod/ffsk.go` (`demod.NewFFSK`, `Discriminate`) |
| Symbol timing | closed-loop MM at 8 sps, chunk-safe | `internal/dsp/sync/clock.go` (`sync.NewMuellerMuller`) |
| DC-tracking slicer | 1/64 EMA threshold, then NRZI or NRZ | `aprs/afsk/receiver.go`, `mdc1200/afsk/receiver.go` (`feedSymbol`) |
| Zero-threshold slicer | `s > 0` → mark, polarity left to the framer | `fleetsync/afsk/receiver.go` (`feedSymbol`) |

## In this post

- **Two tones, one bit** — AFSK, FFSK and the integer-cycle property.
- **From IQ to 9600 Hz audio** — discriminator and resampler.
- **The tone discriminator** — how `demod.FFSK` mixes, filters and decides.
- **Timing at eight samples per bit** — Mueller-Müller and chunk boundaries.
- **The slicer that cost a round** — DC tracking, zero thresholds, line codes.

## Two tones, one bit

Every protocol here shifts an *audio* tone between two pitches and feeds it
into a stock FM transmitter's microphone path; the RF stage never knows. The
[AFSK reference]({{ '/reference/afsk/' | relative_url }}) covers the family,
the [FFSK page]({{ '/reference/ffsk/' | relative_url }}) the coherent cousin,
and the distinction is arithmetic.

At 1200 baud a bit lasts 1/1200 s. A 1200 Hz tone completes exactly one
cycle in that period; an 1800 Hz tone exactly 1.5. Both cross zero at every
bit boundary, so the transmitter switches tones with **no phase
discontinuity** — a continuous-phase waveform a saturated FM channel passes
cleanly. That is *fast* FSK, the CCIR pair MDC1200, FleetSync and MPT 1327
use. Bell 202, which APRS inherits through AX.25, keys 1200 and 2200 Hz: the
space completes 1.833 cycles per bit, so bit-edge phase depends on history.
The front ends encode the choice as two constants:

| Front end | Mark / space | Baud | Line code | Slicer |
|---|---|---|---|---|
| `aprs/afsk` (Bell 202) | 1200 / 2200 Hz | 1200 | NRZI | DC-tracking |
| `mdc1200/afsk` (CCIR) | 1200 / 1800 Hz | 1200 | NRZ | DC-tracking |
| `fleetsync/afsk` (CCIR) | 1200 / 1800 Hz | 1200 (2400 accepted) | NRZ | fixed zero |
| `mpt1327/receiver`, `dsc/ffsk` | 1200 / 1800, 1300 / 2100 Hz | 1200 | — | control-channel paths |

Mark is binary 1 in every one of them — the CCIR convention `demod.FFSK`
bakes into its output sign; `invertSlice` keeps positive meaning mark
whichever tone is higher.

## From IQ to 9600 Hz audio

The first two stages are the ones every FM voice path in the tree uses
([SDR Internals Part 6]({{ '/blog/deep-dives/sdr-internals-06-demodulation/' | relative_url }})).
`demod.FM` is the quadrature discriminator — `arg(z[n]·conj(z[n−1]))` in
radians per sample. Its output is the transmitter's audio plus a DC term for
any tuning error, which matters shortly. That audio is resampled to the rate
the tone detector wants, defined identically in all three front ends:

```go
// internal/radio/mdc1200/afsk/receiver.go (shape) — identical in aprs/afsk
const Oversample  = 8                     // discriminator samples per bit
const mmGain      = 0.05
const AudioRateHz = BaudHz * Oversample   // 1200 × 8 = 9600
g := gcd(uint32(AudioRateHz), opts.InputRateHz)
rsmp := dsp.NewRealResampler(int(AudioRateHz/g), int(opts.InputRateHz/g), 16, 7.0)
```

9600 Hz clears the Nyquist rate of the higher tone and is, the APRS doc
notes, "a round divisor of most common SDR sample rates."
`dsp.NewRealResampler` is the polyphase L/M design of the
[resampler reference]({{ '/reference/resampler/' | relative_url }}), a Kaiser
prototype at cutoff `0.5/max(L, M)` so images and aliases are rejected on
both legs. FleetSync departs from the fixed 16 taps in one respect: because
it may be fed a 48 kHz or 250 kHz stream directly, it sizes `tapsPerBranch`
as `12·M/L`, floored at 16 and capped at 512, so a large decimation still
rejects the out-of-band energy that would fold onto the tones. The
[48 kHz lesson]({{ '/blog/deep-dives/p25-end-to-end-01-c4fm-carrier/' | relative_url }})
of the C4FM family applies here at 9600.

## The tone discriminator

`demod.FFSK` turns audio into a soft bit by treating the two tones as an
**FM signal in their own right**: a mark at 1200 Hz and a space at 1800 Hz
are a 1500 Hz carrier deviated ±300 Hz. So mix the audio down by the
midpoint, low-pass to keep only the deviation, and run the same
`arg(z·conj(last))` discriminator again on the complex baseband:

```go
// internal/dsp/demod/ffsk.go (shape)
func NewFFSK(sampleRate, markHz, spaceHz float64) *FFSK {
    centerHz := (markHz + spaceHz) / 2      // 1500 (CCIR) or 1700 (Bell 202)
    deltaHz := math.Abs(spaceHz-markHz) / 2  // 300 or 500
    cutoff := deltaHz * 1.5 / sampleRate     // passband ends at 1.5 × half-spacing
    // Kaiser LPF for ≥ 60 dB rejection of the mirror at 2·centerHz (≥ 21 taps)
    return &FFSK{
        radPerSample: -2 * math.Pi * centerHz / sampleRate,
        lpf:          filter.NewFIR(filter.LowpassKaiser(n, cutoff, beta)),
        invertSlice:  markHz < spaceHz,
    }
}
```

`Discriminate` mixes each sample by `e^{−j·2π·centerHz·t}`, runs the FIR,
then emits `atan2` of the product with the previous filtered sample. The
output is a signed frequency: positive for mark, negative for space, near
zero between bursts.

Two properties fall out free. **Carrier offset vanishes.** A tuner a few
hundred hertz off puts a DC term on the first discriminator's output — the
slicer-bias problem of
[P25 End to End Part 1]({{ '/blog/deep-dives/p25-end-to-end-01-c4fm-carrier/' | relative_url }}).
Here that DC is an audio-band constant, and a frequency-*derivative* of a
constant is nothing; `TestReceiverToleratesCarrierOffset` shifts a FleetSync
burst by −600 and +400 Hz and decodes both. **The mirror cannot beat.** The
mix also produces an image near `2·centerHz`; the 60 dB stopband keeps it out
of the second discriminator, where it would appear as an inter-tone beat.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="The FFSK tone discriminator chain: 9600 hertz audio holding 1200 or 1800 hertz tones is mixed by the 1500 hertz midpoint so mark sits at minus 300 and space at plus 300 hertz, a 450 hertz Kaiser low-pass keeps that deviation and rejects the 3 kilohertz mirror, and a second FM discriminator yields a signed soft bit, positive for mark. A spectrum sketch shows the two tones straddling zero inside the passband.">
  <rect x="6" y="40" width="96" height="44" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="54" y="58" text-anchor="middle" fill="currentColor" font-size="10">audio @ 9600</text>
  <line x1="102" y1="62" x2="126" y2="62" stroke="currentColor"/>
  <rect x="126" y="40" width="110" height="44" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="181" y="58" text-anchor="middle" fill="var(--accent)" font-size="10">mix by 1500 Hz</text>
  <text x="181" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">mark −300, space +300</text>
  <line x1="236" y1="62" x2="260" y2="62" stroke="currentColor"/>
  <rect x="260" y="40" width="110" height="44" rx="5" fill="none" stroke="currentColor"/>
  <text x="315" y="58" text-anchor="middle" fill="currentColor" font-size="10">Kaiser LPF</text>
  <text x="315" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">cutoff 450 Hz · 60 dB</text>
  <line x1="370" y1="62" x2="394" y2="62" stroke="currentColor"/>
  <rect x="394" y="40" width="130" height="44" rx="5" fill="none" stroke="currentColor"/>
  <text x="459" y="58" text-anchor="middle" fill="currentColor" font-size="10">arg(z · conj(last))</text>
  <line x1="524" y1="62" x2="548" y2="62" stroke="currentColor"/>
  <rect x="548" y="40" width="124" height="44" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="610" y="58" text-anchor="middle" fill="var(--accent)" font-size="10">soft bit, + = mark</text>
  <line x1="60" y1="200" x2="620" y2="200" stroke="var(--fg-muted)"/>
  <line x1="256" y1="200" x2="256" y2="145" stroke="var(--accent)" stroke-width="3"/>
  <text x="256" y="138" text-anchor="middle" fill="var(--accent)" font-size="9">mark −300</text>
  <line x1="424" y1="200" x2="424" y2="145" stroke="currentColor" stroke-width="3"/>
  <text x="424" y="138" text-anchor="middle" fill="currentColor" font-size="9">space +300</text>
  <path d="M 200 198 L 200 155 L 480 155 L 480 198" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="340" y="220" text-anchor="middle" fill="var(--fg-muted)" font-size="8">passband ±450 Hz around 0 Hz (was 1500); tuning-error DC sits at 0, removed by the derivative</text>
</svg>
<figcaption>The tone pair is an FM signal around its own midpoint: mix it to baseband, keep the ±300 Hz deviation, discriminate. A tuning-error constant becomes a frequency the derivative ignores; the mirror never reaches the second discriminator.</figcaption>
</figure>

## Timing at eight samples per bit

The discriminator emits 8 soft samples per bit; the framer wants one.
Bridging them is `sync.MuellerMuller`
([SDR Internals Part 7]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }}),
[reference]({{ '/reference/mueller-muller-timing-recovery/' | relative_url }})):
a sub-sample clock `mu` advanced by `sps` per symbol and nudged by
`gain × error`, the error `|s[n] − sgn(s[n−1])·s[mid]|` — zero when the
sampling instant sits at the eye's centre. `NewMuellerMuller(8, 0.05)` is
what every AFSK front end constructs; the APRS doc records why 0.05: "fast
enough to settle inside the AX.25 preamble (~30 flags = 240 bits), slow
enough that random payload bytes don't yank the symbol clock around." A
closed loop rather than POCSAG's open-loop integrator because Bell 202 audio
"is messier than POCSAG direct-FSK at the slicer" — variable AGC, a clock a
few hundred ppm off.

The loop carries a scar from the P25 work: its `prevTail` field holds the
last sample of the previous chunk so a symbol boundary landing on `src[0]`
interpolates against the correct predecessor. Without it the walk skipped
`src[0]` every call and lost a sample of clock phase per chunk — invisible on
long synthetic buffers, fatal on RTL-sized ~19-symbol chunks (issue #275).
`TestReceiverIsChunkInvariant` pins it here: the same FS-II burst fed in
chunks of 1, 37, 500, 4096 and whole-stream samples must yield the identical
ANI.

## The slicer that cost a round

After the loop one soft sample per bit remains, and here the three front
ends disagree — a protocol fact wearing DSP clothes. APRS and MDC1200 slice
against a **slow mean**:

```go
// internal/radio/aprs/afsk/receiver.go (shape) — mdc1200/afsk is identical
func (r *Receiver) feedSymbol(s float32) {
    r.meanEMA += (s - r.meanEMA) * (1.0 / 64.0) // ~50 ms memory at 1200 baud
    var raw byte
    if s > r.meanEMA { raw = 1 }
    r.inner.Push(r.nrzi.Decode(raw)) // MDC1200: r.inner.Push(raw) — plain NRZ
}
```

The tracker absorbs residual DC from imperfect tuning, and for AX.25's
`0x7E` flag preamble — alternating enough to keep the mean honest — it is
benign. FleetSync tried it and measured the opposite.
`fleetsync/afsk`'s `feedSymbol` slices at a **fixed zero**, and its comment is
the post-mortem: a FleetSync II frame whose `word1` nibbles are small "opens
with ~68 symbols of continuous space tone"; a 1/512-per-symbol bias tracker
"drifted ~12% of a symbol toward it," and the ISI-attenuated *isolated* bits
that follow — the weakest samples in the frame — flipped. On one synthesised
260-bit frame: **23 errors with the tracker, zero with the fixed threshold.**
The reference decoder slices on the sign of the accumulated deviation, and
after the previous section zero is also *correct*: the discriminator output
is DC-free by construction, so a tracker has nothing to remove and everything
to break. Polarity inversion is handled a layer up by the framer locking on
the complemented sync word.

The last step is the line code: "Unlike APRS the line code is plain NRZ,
not NRZI," the FleetSync package doc says. AX.25 transmitters NRZI-encode — a
0 is a tone *transition*, a 1 no transition — so `aprs/afsk.NRZIDecoder`
emits 1 when the raw bit matches the previous one and 0 when it differs,
making AX.25 indifferent to which tone is "high"
([NRZI reference]({{ '/reference/nrzi/' | relative_url }})). MDC1200 and
FleetSync carry the bit on the tone directly, so tone-sense ambiguity is
resolved by sync-word complement instead.

### How the tone pair shaped the Go code

- **One demodulator, parameterised by two frequencies.** `demod.NewFFSK`
  derives midpoint, deviation, cutoff and filter length from
  `markHz`/`spaceHz`; its five users differ by two constants.
- **Rates are named constants, not arguments.** `AudioRateHz = BaudHz ×
  Oversample`, the resampler ratio reduced by GCD from whatever the SDR
  delivers — 9600 Hz is as fixed a design point as 48 kHz is for C4FM.
- **The slicer is per protocol.** Each `feedSymbol` lives in the protocol's
  own `afsk` package, because the right threshold is a property of the frame's
  bit statistics.
- **Every stage is chunk-safe and resettable.** `prevTail`, `last`, LPF
  history and mixer phase persist across `Process` calls.

## Where this goes next

A stream of sliced bits is where the shared chain ends and a protocol begins.
[Part 3]({{ '/blog/deep-dives/beyond-voice-03-mdc1200-template-decoder/' | relative_url }})
takes the MDC1200 path from here: the 40-bit sync word and its five-error
tolerance, the 16×7 interleave, the reflected CRC-16, the opcode table, and
the wiring that made it the template later signalling decoders were cloned
from.

## FAQ

**What is the difference between AFSK and FFSK?**
Both send bits as two audio tones through an FM transmitter. FFSK (fast FSK)
chooses tones that are integer or half-integer multiples of the bit rate —
1200 and 1800 Hz at 1200 baud hold exactly 1 and 1.5 cycles per bit — so the
phase is continuous at bit edges. Bell 202's 2200 Hz space is not.

**Why does GopherTrunk demodulate the tones with a second FM discriminator?**
Because a mark/space pair is an FM signal around its own midpoint: 1200/1800 Hz
is a 1500 Hz carrier deviated ±300 Hz. Mixing by the midpoint, low-passing and
discriminating gives a signed frequency, positive for mark, and removes
tuning-error DC as a side effect.

**Why 9600 Hz and eight samples per bit?**
1200 baud × 8 is above the Nyquist rate of the highest tone (2200 Hz), gives
the Mueller-Müller loop enough sub-sample resolution to track the clock, and
divides evenly into most SDR sample rates so the resampler stays small.

**Why does FleetSync slice at zero when APRS tracks DC?**
FleetSync II frames can open with about 68 symbols of continuous space tone; a
1/512 bias tracker drifted toward that run and flipped the weak isolated bits
that followed — 23 errors in one synthesised 260-bit frame, zero with a fixed
threshold. The discriminator output is already DC-free, so the sign slicer is
simpler and correct.

**How does the receiver cope with an inverted FM discriminator?**
Per line code. APRS is NRZI, where a 0 is a transition, so tone sense is
irrelevant by construction. MDC1200 and FleetSync are plain NRZ, so their
framers hunt the sync word and its bitwise complement and invert the payload
when they lock on the complement.

## Series navigation

**Part 2 of 14** · ←
[Part 1: Why a Scanner Decodes Data — The Eleven-Place Pattern]({{ '/blog/deep-dives/beyond-voice-01-eleven-place-pattern/' | relative_url }})
· Next →
[Part 3: MDC1200 — The Template Decoder]({{ '/blog/deep-dives/beyond-voice-03-mdc1200-template-decoder/' | relative_url }})
