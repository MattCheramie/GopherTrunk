---
title: "The Analog Edge, Part 4: Clipping, Overload & Intermod"
description: What front-end overload actually looks like from the samples up — rail-pinned IQ, the clip ratio as the authoritative verdict, intermodulation products that manufacture phantom signals as gain rises, out-of-band overload from FM broadcast, and why a clean clip ratio still doesn't clear the front end.
category: tutorials
keywords: adc clipping, rail pinned samples, front end overload, intermodulation distortion, imd third order, clip ratio, fm broadcast overload, sdr desense, phantom signals, gophertrunk analog edge
tags: [analog-edge, overload, intermod, adc, sdr, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Analog Edge"
series_part: 4
---

*Part 4 of **The Analog Edge**, a 14-part field guide to the analog half of a
GopherTrunk installation. [Part 3]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
ended with the ladder method's quiet assumption: that you can recognize
overload when a gain sweep walks into it. This part makes overload concrete
for our reader with the marginal system — because "too much gain" doesn't
fail the way intuition says it should. It doesn't get loud; it gets
*dishonest*. The samples start containing signals that were never on the
air, and signals that were on the air stop surviving into the samples.*

> **TL;DR:** Overload has two faces. **Clipping** is samples pinned to the
> ADC rail — the [#881](https://github.com/MattCheramie/GopherTrunk/issues/881)
> "nineteen dibits" capture was **50% rail-pinned** while every symptom
> pointed at software —
> and `gophertrunk_sdr_iq_clip_ratio` is the **authoritative** verdict
> (sustained > ~0.002 = overloaded; the RMS gauge averages peaks away).
> **Intermodulation** is subtler: a nonlinear front end *multiplies* strong
> signals together, manufacturing phantom carriers (third-order products at
> 2f₁−f₂ and 2f₂−f₁) that land in-band and raise the floor for everyone.
> Both explain the up-slope of the decode-error U-curve. But the converse
> fails: **a clean clip ratio does not clear the front end** — the
> [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) captures
> peaked ≈ −48 dBFS, nowhere near the rails, and were still ruined upstream.

**Key takeaways**

- **Clipping destroys information non-locally.** A rail-pinned sample is a
  hard nonlinearity, and its distortion products spray across the whole
  capture bandwidth — one strong pager can garble every tap on a wideband
  dongle.
- **The clip ratio is the verdict; the power gauge is a hint.** RMS can read
  a survivable −5 dBFS while peaks are flat-topping — which is exactly why
  the metric pair exists (Part 2).
- **Intermod means gain manufactures signals.** Third-order products grow
  3 dB for every 1 dB of gain, so the phantom floor rises three times faster
  than your wanted carrier — the mechanism behind the U-curve's right side.
- **The overload you can't see is out-of-band.** An FM broadcast blaster
  50 MHz away never appears in your capture, but it's compressing the same
  LNA your control channel uses. Filters, not software, fix that (Part 9).

## Cheat sheet

| Concern | The test | Where it lives |
|---|---|---|
| Am I clipping? | sustained clip ratio > ~0.002 | `gophertrunk_sdr_iq_clip_ratio`, `gophertrunk_sdr_wideband_input_clip_ratio` (`internal/metrics/prom.go`) |
| Is a strong site burying weak taps? | per-tap level far below wideband input power | issue #749 guidance in `config.example.yaml`; throttled WARN in the log |
| Is this carrier real or intermod? | drop gain 10 dB — a real carrier drops 10, an IM3 product drops ~30 | [Intermodulation]({{ '/reference/intermodulation/' | relative_url }}) |
| Where does compression start? | the amp's 1 dB compression point | [P1dB]({{ '/reference/1-db-compression-point/' | relative_url }}) |
| Out-of-band blaster suspected | bandpass/broadcast-notch filter A/B | [SDR filters guide]({{ '/sdr-filters/' | relative_url }}) |
| Optional RF amp making it worse | `rf_amp` is off by default for this reason | `sdr.devices[].rf_amp` note in `config.example.yaml` |

## In this post

- **What clipping is in the samples** — rails, flat tops, and spray.
- **Reading the instruments** — clip ratio over power, always.
- **Intermod: gain manufactures signals** — the 3-for-1 slope.
- **The U-curve, explained end to end** — both slopes named.
- **The overload you can't see** — out-of-band compression.
- **Why "no clipping" is not a clean bill** — the #764 distinction.

## What clipping is in the samples

An ADC has a highest value it can write. When the waveform exceeds it, the
converter writes that maximum — again and again — until the waveform comes
back into range. In the time domain the sine peaks are sliced flat; in the
IQ constellation, samples pile up in a box at the rails; in a histogram of
sample values, two spikes grow at the extremes. GopherTrunk counts exactly
that: the fraction of samples whose I or Q component sits at the rail.

The damage is worse than "the loud parts are missing." Flat-topping is a
hard nonlinearity, and nonlinearities create energy at new frequencies —
harmonics and mixing products smeared across the entire capture bandwidth.
That's why clipping is a *wideband* catastrophe: the strong signal that
caused it usually survives well enough to look healthy, while every weaker
channel in the capture inherits a raised, structured noise floor. On a
multi-tap wideband dongle this is the issue #749 failure mode from Part 2:
one hot site, and every weaker sibling drowns.

The tracker's canonical case is the
[Nineteen Dibits postmortem]({{ '/blog/solution-postmortem/from-the-issue-tracker-08-nineteen-dibits/' | relative_url }}):
a meticulously argued software hypothesis, a chain of plausible evidence —
and a raw capture that turned out to be 50% pinned to the rails. The fix was
turning the gain down. No amount of reading the decoder would have found it;
one look at the clip ratio did.

## Reading the instruments

Part 2 introduced the metric pair; overload is where the division of labor
matters. The power gauge is RMS over ~1 s, and RMS *averages away* exactly
the peaks that clip — a high-crest wideband capture can flat-top on pager
bursts while the mean reads a merely-hot −5 dBFS. So the discipline is
mechanical: **suspicion of overload is settled by the clip ratio and only
the clip ratio.** Zero is the only comfortable reading; a sustained value
above ~0.002 (one sample in 500) is the front end telling you it's being
overdriven, and both the metric help text and the config comments give the
same instruction — reduce gain or add attenuation, and do **not** raise
gain: a hot neighbor is desensitizing the receiver, and more gain feeds the
fire. On wideband dongles, watch the per-serial
`wideband_input_clip_ratio`, which sees the whole capture *before*
channelization — a tap can read clean while the shared converter is already
in trouble.

## Intermod: gain manufactures signals

Clipping is overload's blunt face. The subtle face begins earlier, while
every sample still looks legal. Real amplifiers are only approximately
linear, and as input level approaches the amp's
[1 dB compression point]({{ '/reference/1-db-compression-point/' | relative_url }}),
the approximation fails gracefully — by *multiplying* signals together.
Feed a slightly nonlinear stage two strong carriers at f₁ and f₂ and it
emits **intermodulation products** at combinations of them. The third-order
pair, 2f₁−f₂ and 2f₂−f₁, is the killer: those land *near the originals* —
in-band, on top of whatever weak channel is unlucky enough to live there —
and no filter after the nonlinearity can help, because the products are
created inside your own front end.

The growth rate is what makes gain so dangerous here. Raise the input of a
nonlinear stage by 1 dB and its third-order products rise by **3 dB**. Turn
your gain up 5 dB and every real carrier gains 5 while the phantom floor
gains 15. This is the arithmetic behind the field observation that an
overdriven SDR band looks *busier* — ghost carriers at arithmetic spacings,
"stations" that vanish when you touch the gain. Which is also the field
test: **drop gain 10 dB; a real signal drops ~10 dB, a third-order product
drops ~30 dB.** Anything that falls off a cliff was never on the air.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="Two spectrum sketches side by side. On the left, at moderate gain, two strong carriers f1 and f2 stand over a flat noise floor with a small wanted channel between them. On the right, at high gain, the same two carriers are larger and new third-order intermod products have appeared at 2f1 minus f2 and 2f2 minus f1, flanking the originals, and the noise floor has risen, burying the wanted channel. A note states that intermod products grow 3 dB for every 1 dB of gain.">
  <line x1="30" y1="180" x2="320" y2="180" stroke="var(--fg-muted)"/>
  <text x="175" y="200" text-anchor="middle" fill="var(--fg-muted)" font-size="10">moderate gain</text>
  <polyline points="30,176 320,176" stroke="var(--fg-muted)" fill="none" stroke-dasharray="2 3"/>
  <polyline points="110,176 116,70 122,176" fill="none" stroke="currentColor"/>
  <text x="116" y="62" text-anchor="middle" fill="currentColor" font-size="9">f₁</text>
  <polyline points="230,176 236,64 242,176" fill="none" stroke="currentColor"/>
  <text x="236" y="56" text-anchor="middle" fill="currentColor" font-size="9">f₂</text>
  <polyline points="170,176 176,140 182,176" fill="none" stroke="var(--accent)"/>
  <text x="176" y="132" text-anchor="middle" fill="var(--accent)" font-size="9">your channel</text>
  <line x1="360" y1="180" x2="650" y2="180" stroke="var(--fg-muted)"/>
  <text x="505" y="200" text-anchor="middle" fill="var(--fg-muted)" font-size="10">high gain: same air, new signals</text>
  <polyline points="360,160 650,160" stroke="var(--fg-muted)" fill="none" stroke-dasharray="2 3"/>
  <text x="645" y="152" text-anchor="end" fill="var(--fg-muted)" font-size="9">floor up</text>
  <polyline points="440,160 446,42 452,160" fill="none" stroke="currentColor"/>
  <text x="446" y="34" text-anchor="middle" fill="currentColor" font-size="9">f₁</text>
  <polyline points="560,160 566,38 572,160" fill="none" stroke="currentColor"/>
  <text x="566" y="30" text-anchor="middle" fill="currentColor" font-size="9">f₂</text>
  <polyline points="380,160 386,96 392,160" fill="none" stroke="var(--accent)"/>
  <text x="386" y="88" text-anchor="middle" fill="var(--accent)" font-size="9">2f₁−f₂</text>
  <polyline points="614,160 620,94 626,160" fill="none" stroke="var(--accent)"/>
  <text x="620" y="86" text-anchor="middle" fill="var(--accent)" font-size="9">2f₂−f₁</text>
  <polyline points="500,160 506,148 512,160" fill="none" stroke="var(--fg-muted)"/>
  <text x="506" y="140" text-anchor="middle" fill="var(--fg-muted)" font-size="9">your channel, buried</text>
  <text x="340" y="216" text-anchor="middle" fill="var(--fg-muted)" font-size="10">IM3 grows 3 dB per 1 dB of gain — the phantom floor outruns the signal you were trying to help</text>
</svg>
<figcaption>Two strong carriers through a front end pushed toward compression: third-order products appear beside them and the floor rises, three times faster than the gain that caused it.</figcaption>
</figure>

## The U-curve, explained end to end

Part 3's ladder sweep produces a U-shaped curve of decode error against
gain, and both slopes now have names. The **left slope** is thermal: too
little gain and your carrier sits down in the receiver's own noise, so
errors fall as gain lifts it clear. The **flat bottom** is the plateau where
the channel's SNR is set by the air, not the knob. The **right slope** is
this post: compression begins, intermod products bloom 3-for-1, the
effective floor rises, and errors climb — often before a single sample
clips. The clip ratio only catches the far end of the right slope; the
decode error rate sees all of it, which is precisely why
[autogain]({{ '/blog/deep-dives/the-hunt-05-autogain-autotune/' | relative_url }})
scores rungs by decoding rather than by any level meter, and why its final
tie-break walks *down* the ladder.

## The overload you can't see

Everything above assumed the aggressor is in your capture where you can at
least look at it. Often it isn't. Your LNA and tuner amplify a wide swath of
spectrum *before* any narrow filtering — so a 100 kW FM broadcast
transmitter at 98 MHz, a paging blaster at 152 MHz, or a nearby cell site
can compress the front end while sitting entirely outside your capture. The
signature is desense that follows geography rather than tuning: everything
is a little worse, the noise floor a little higher, the U-curve's bottom
shifted left, and nothing visible to blame. If dropping gain helps *every*
channel at once, or an attenuator paradoxically improves decode, think
out-of-band. The remedy is analog by definition — a bandpass or
broadcast-notch filter ahead of the first amplifier — and
[Part 9]({{ '/blog/tutorials/analog-edge-09-filters-lnas/' | relative_url }})
covers choosing one (the [filters guide]({{ '/sdr-filters/' | relative_url }})
has the hardware). This is also why `rf_amp` ships **off** by default in
`config.example.yaml`: the same amplifier that lowers your noise figure on
a quiet band overloads first on a hot one.

## Why "no clipping" is not a clean bill

Now the distinction this part exists to draw. Clipping and intermod are
*level-driven* failures: back the gain off and they retreat. It is tempting
to conclude the converse — clip ratio zero, decode still bad, therefore the
front end is innocent and the software is guilty. [#764](https://github.com/MattCheramie/GopherTrunk/issues/764)
is the standing counterexample: both captures peaked around −48 dBFS,
45 dB of headroom below the rails, no clipping, no plausible intermod — and
the 10 MS/s capture was still missing ~10 dB of usable SNR that no software
could restore. The front end has failure modes that leave the level
statistics pristine, and the biggest of them — oscillator phase noise, the
actual culprit in #764 — is [Part 5]({{ '/blog/tutorials/analog-edge-05-phase-noise-reciprocal-mixing/' | relative_url }})'s
subject. Overload is the *first* front-end suspect because it's the easiest
to test, not the only one. Clearing it narrows the search; it doesn't end
it. The [front end & overload lesson]({{ '/learn/rf-sdr/front-end-and-overload/' | relative_url }})
is a gentler companion to this whole part.

## Where this goes next

With clipping and intermod ruled out by instruments you now know how to
read, our marginal reader is left with the strangest failure in the series:
a carrier that looks *clean* on every meter and still won't decode.
[Part 5]({{ '/blog/tutorials/analog-edge-05-phase-noise-reciprocal-mixing/' | relative_url }})
retells the ten-megasamples lesson for operators — phase noise, reciprocal
mixing, and the carrier-clean-but-modulation-degraded signature that fooled
the wideband FFT itself.

## FAQ

**What clip ratio is acceptable?**
Zero. The metric's own threshold for "overloaded" is a sustained ~0.002 —
one sample in 500 — but that's an alarm level, not a budget. Brief blips
during a nearby key-up are survivable; any *resting* non-zero value means
your loudest neighbor owns your headroom, and the next strong burst garbles
everything (Part 2's table applies: reduce gain, or add attenuation).

**How can one pager channel ruin a P25 decode two megahertz away?**
Two ways at once. If it clips the ADC, the distortion spray is
capture-wide. If it merely compresses the LNA, its intermod products with
other strong signals can land directly on your channel, and the compression
itself desensitizes everything. Distance in frequency is no protection
inside a nonlinear front end — only level (attenuation) or selectivity
(filters) is.

**Are those regular "carriers" every N kHz real?**
Test them: drop gain 10 dB. Real carriers drop ~10 dB; third-order intermod
drops ~30 dB and usually vanishes. Evenly spaced ghosts that appear only at
high gain are your own front end mixing two strong stations. (Regular
spikes that *don't* respond to gain at all are more likely your computer's
own switching noise — a different Part 9 problem.)

**Can GopherTrunk decode through mild clipping?**
Sometimes — FEC exists, constant-envelope modulations tolerate amplitude
abuse better than linear ones, and a barely-clipped strong signal often
still locks. But you're spending decode margin on a self-inflicted wound,
and the weaker channels in the same capture are spending far more. The
gain that stops the clipping almost always decodes better everywhere.

**My hardware scanner doesn't overload in the same spot. Why?**
Selectivity. A scanner filters the band down before most of its gain; a
wideband SDR amplifies nearly everything the antenna delivers and filters
late, in software — after the damage. That's not a defect, it's the
tradeoff that makes an SDR wideband, and it's why Part 9's filters exist:
they buy back, externally, the selectivity a scanner has built in.

## Series navigation

**Part 4 of 14** · ←
[Part 3: Gain Staging — Never Chase a Software Threshold]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
· Next →
[Part 5: Phase Noise & Reciprocal Mixing — The Ten-Megasamples Lesson]({{ '/blog/tutorials/analog-edge-05-phase-noise-reciprocal-mixing/' | relative_url }})
