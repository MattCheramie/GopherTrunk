---
slug: front-end-and-overload
title: Front-end filters, LNAs & overload
description: How the analog front end — preselector filters, low-noise amplifiers, mixer and ADC — decides what reaches your decoder, and why overload, intermodulation and reciprocal mixing can ruin a clean channel even when your gain is set correctly.
keywords: preselector filter, bandpass filter, SAW filter, LNA low noise amplifier, front-end overload, intermodulation, IMD, reciprocal mixing, phase noise, attenuator, out of band, FM broadcast filter
level: advanced
status: full
prereq:
  - gain-and-agc
  - sdr-receiver
faq:
  - q: Why do I need a filter if I already have a gain control?
    a: Gain decides how strong everything is when it reaches the ADC, but it can't tell your target apart from a strong out-of-band transmitter — it amplifies both equally. A filter rejects the unwanted energy *before* it can overload the front end, so the parts of the chain that follow only see the band you care about. Gain and filtering solve different problems; a busy RF environment usually needs both.
  - q: What is intermodulation?
    a: When two or more strong signals hit a stage that isn't perfectly linear (a mixer or an overdriven amplifier), the stage mixes them and produces false products at new frequencies — combinations like 2f1−f2 and 2f2−f1. These land on real channels and look like genuine signals, but they're artifacts of overload. The tell is that they appear and disappear as you change the input level with an attenuator.
  - q: Do I put the filter or the LNA first?
    a: It's a trade-off. A filter before the LNA protects the amplifier from strong out-of-band signals that would overload it, at the cost of the filter's insertion loss adding to the noise figure. An LNA before the filter gives the best noise figure — the first amplifier dominates system noise — but leaves that amplifier exposed to everything. In a clean location, LNA-first; in a crowded RF environment, filter-first.
  - q: Why does a strong FM or pager signal ruin a channel far away from it?
    a: A very strong nearby transmitter can push the mixer or ADC into overload even though it's nowhere near your frequency, spraying intermodulation products and a raised noise floor across the band. It can also degrade your channel through reciprocal mixing — the strong signal mixes with the local oscillator's phase noise and smears broadband noise onto your weak channel. The cure is to reject the offender with a filter, not to add gain.
gophertrunk_links:
  - title: Hardware (front-end notes)
    url: /hardware.html
    note: supported SDRs and notes on front-end filtering and overload behaviour.
  - title: Tuning (receiver meters)
    url: /tuning.html
    note: watch level and SNR as you add a filter or attenuator — a good change lifts SNR.
---

# Front-end filters, LNAs & overload

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Gain is not the only thing that decides a clean decode — **what you let into the
front end matters just as much**. A **preselector/filter** rejects out-of-band
energy before it can do harm; an **LNA** lifts weak signals with minimal added
noise; but a strong nearby transmitter can still cause **overload**,
**intermodulation** (false ghost signals), or **reciprocal mixing** (a
clean-looking carrier that decodes badly). The fixes are often *less* gain, an
attenuator, or a filter — not more amplification. Builds on
[gain & AGC](/learn/rf-sdr/gain-and-agc/) and the
[SDR receiver](/learn/rf-sdr/sdr-receiver/).
</div>

The [gain lesson](/learn/rf-sdr/gain-and-agc/) got your signals positioned correctly
in the ADC's range. But gain treats everything in the band the same way — it can't
distinguish your target from the megawatt FM station down the road. This lesson is
about the analog stages *before* digitising, and the subtle ways a strong signal you
aren't even trying to hear can wreck the one you are.

## The front end, in order

The [SDR receiver lesson](/learn/rf-sdr/sdr-receiver/) walked the whole chain; here we
zoom in on the analog stages that come before the samples exist:

**antenna → (filter) → LNA → mixer → ADC**

The antenna collects *everything* in its range. An optional **filter** rejects
frequencies you don't want. The **LNA** (low-noise amplifier) boosts what's left. The
**mixer** shifts your band down to a rate the converter can handle, and the **ADC**
digitises it into [IQ samples](/learn/rf-sdr/iq-data/). Every stage before the ADC is
analog and *shared* — whatever energy reaches a stage affects everything passing
through it, not just your channel.

## Preselector & bandpass filters

A **preselector** is a filter placed early in the chain that passes the band you care
about and rejects the rest. Why bother, when you're going to
[filter digitally](/learn/rf-sdr/filtering-decimation/) later anyway? Because the analog
front end has to survive the *total* power hitting it, and much of that power is out of
band: **FM broadcast (88–108 MHz)**, pagers, cellular, and TV transmitters are often
tens of dB stronger than the trunked control channel you want.

- A **bandpass filter** passes a range and attenuates everything outside it.
- A **notch filter** does the opposite — it kills one troublesome band (an
  **FM-broadcast reject** notch is the classic first purchase).
- **SAW filters** (surface acoustic wave) are compact fixed-frequency bandpass parts
  common in ready-made SDR filter modules.

A filter **can't add signal** — it only takes energy away, and it costs a little of
your wanted signal too (insertion loss). What it buys you is protection: everything
downstream now sees a much smaller total power, so the LNA, mixer and ADC are far less
likely to overload.

## Low-noise amplifiers (LNAs)

An **LNA** amplifies weak signals while adding as little noise of its own as possible.
It matters *where* it sits because **the first amplifier dominates the system
[noise figure](/learn/rf-sdr/noise-and-snr/)** — noise added at the front is amplified by
everything after it, so a quiet first stage sets the floor for the whole receiver. This
is why a **masthead LNA** (mounted at the antenna) can help so much: it lifts the signal
*before* the loss of a long feedline drags the [SNR](/learn/rf-sdr/noise-and-snr/) down.

That leads straight to the classic ordering trade-off:

- **Filter before LNA** — the filter protects the amplifier from strong out-of-band
  signals that would otherwise overload it. The price is the filter's insertion loss,
  which adds directly to the noise figure.
- **LNA before filter** — the best possible noise figure, because nothing lossy sits
  ahead of the amplifier. The price is that the LNA is exposed to the full band and can
  itself be driven into overload.

In a quiet rural location, LNA-first wins. In a crowded urban RF environment with
strong nearby transmitters, filter-first is usually the safer choice — a slightly higher
noise figure beats an amplifier that's constantly overloading.

## Overload: too much total power

The mixer and ADC don't see your channel in isolation — they see the **sum of
everything in the band at once**. A strong nearby transmitter can drive them into
compression (the stage stops responding linearly) even when your target is weak and far
away in frequency. Once a stage is compressed, it distorts *every* signal passing
through it, including yours.

This looks similar to the too-much-gain problem from the
[gain lesson](/learn/rf-sdr/gain-and-agc/), but the cause is different: with clipping you
turned the gain up too far, and turning it down fixes it. With front-end overload the
*input* is too hot — the culprit is an external signal, and the fix is to reject that
signal or attenuate the whole input, not just to trim gain. Watch for it whenever a
channel decodes fine at some times and falls apart when a strong local transmitter keys
up.

## Intermodulation (IMD)

When a non-linear stage is fed two strong signals at f1 and f2, it doesn't just distort
them — it **mixes** them and produces new **intermodulation products** at frequencies
like **2f1−f2** and **2f2−f1**. These third-order products fall *close to* the original
signals, so they can land right on top of a real channel you're trying to decode, and
they look exactly like genuine signals on the [waterfall](/learn/rf-sdr/fft-and-waterfall/).

The giveaway is how they behave with level. A real signal barely changes when you add a
few dB of attenuation. An intermodulation product changes *much faster* — a third-order
product drops about **3 dB for every 1 dB** you attenuate the input. So if a suspected
ghost signal vanishes when you insert 10 dB of attenuation while your real signals only
dip slightly, you've found IMD, not a station.

## Reciprocal mixing & phase noise

This is the subtle one. A real local oscillator isn't a perfect single tone — it has
**phase noise**, a skirt of noise spread either side of its frequency. In the mixer,
your wanted channel isn't the only thing that beats against the oscillator: a **strong
nearby signal** does too. When that strong signal mixes with the oscillator's phase
noise, it drags a copy of that broadband noise skirt right onto your weak channel. This
is **reciprocal mixing**.

The trap is what it looks like. On a wideband [FFT](/learn/rf-sdr/fft-and-waterfall/) the
strong carrier looks perfectly clean, and your channel looks fine — there's no obvious
spur, no clipping. But the *demodulated* signal is degraded: the effective noise on the
channel is raised and the [EVM](/learn/rf-sdr/symbols-and-baud/) climbs, so it decodes
poorly despite looking healthy on the spectrum. The strength of the offender and the
quality of the oscillator set how bad it gets.

More gain does **not** help here — it amplifies the smeared noise right along with the
signal. The fixes are a cleaner front end (a better, lower-phase-noise oscillator) or
**knocking the strong offender down with a filter** so it never reaches the mixer at full
strength.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 150" role="img" aria-label="Two spectra. Left: a tall strong carrier and a short weak channel both sitting on a low flat noise floor, labelled looks clean. Right: the same two signals, but a noise skirt from the strong carrier has raised the floor under the weak channel, labelled reciprocal mixing." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <path d="M20 110 L60 110 L64 40 L68 110 L150 110 L154 80 L158 110 L200 110"/>
    <path d="M240 110 Q260 108 270 96 L274 40 L278 96 Q300 104 320 100 L344 90 L348 80 L352 90 Q380 104 420 108"/>
  </g>
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <text x="64" y="30">strong</text>
    <text x="154" y="72">weak</text>
    <text x="110" y="130">looks clean on FFT</text>
    <text x="330" y="130">reciprocal mixing</text>
  </g>
</svg>
<figcaption>The strong carrier still looks clean, but its phase-noise skirt has raised the floor right under the weak channel — the decode suffers even though the spectrum looks fine.</figcaption>
</figure>

## Fixes: attenuators & filters

The counter-intuitive part of front-end trouble is that **adding an attenuator** — or
lowering gain, or fitting a bandpass or FM-reject filter — can *improve* your decode.
You're trading a little signal for a large reduction in the total power abusing the
front end, and clearing overload or IMD wins that trade easily. A practical checklist
when a channel decodes worse than its raw strength suggests it should:

1. Check the [gain](/learn/rf-sdr/gain-and-agc/) first — rule out plain ADC clipping.
2. Insert **10 dB of attenuation**. If SNR *improves* or ghost signals vanish, you were
   overloaded — the front end, not the channel, was the problem.
3. Identify the strongest offender on a wide [spectrum](/learn/rf-sdr/fft-and-waterfall/)
   (often FM broadcast or a pager); fit a **notch or bandpass filter** to reject it.
4. In a crowded location, move the **filter ahead of the LNA**.
5. Re-check level and SNR on the [tuning meters](/tuning.html) — a good change shows as
   a higher, steadier SNR, not just a lower level.

<div class="knowledge-check" data-quiz data-correct-msg="Right — ghost signals that die faster than real ones under attenuation are intermodulation from an overloaded front end." markdown="0">
  <p class="knowledge-check__q">Quick check: ghost signals appear across the band and vanish when you add 10 dB of attenuation, while your real signals only dip slightly. What is it?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The antenna is too long</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Front-end overload / intermodulation</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The sample rate is too high</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The analog front end — **antenna → filter → LNA → mixer → ADC** — decides what even
  reaches your decoder, and every pre-ADC stage is shared by the whole band.
- A **preselector/filter** can't add signal, but it rejects out-of-band energy (FM,
  pagers, cellular) and protects everything downstream from overload.
- The **first amplifier dominates the noise figure**: LNA-first for best sensitivity,
  filter-first to survive a crowded RF environment.
- **Overload** (too much total input power) and **intermodulation** (false products like
  2f1−f2) are caused by strong signals, not by your gain — an attenuator often cures them.
- **Reciprocal mixing** raises the effective noise on a weak channel via oscillator
  phase noise; the carrier looks clean on the FFT yet decodes badly, and more gain won't
  help — a cleaner front end or a filter will.

Next up: the dongles all of this runs on — [SDR hardware: RTL-SDR, HackRF & Airspy](/learn/rf-sdr/sdr-hardware/).
