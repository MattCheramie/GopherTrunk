# GT-RF-01.03 — Frequency
slug: frequency · type: term · treatment: 2.1 term · target: 3:30–4:30 · words: 565

**[V: Title card — "Frequency"]**

**[V: a tuning dial sweeping; readout stops on 162.550 MHz; a waveform locks in]** [CLIP c1 starts]
Frequency is the number you tune to. When you dial in 162.550 megahertz, you're not choosing a channel out of a list — you're selecting how many times per second a wave vibrates, and listening for the one wave in the air that matches. Formally: frequency is the number of cycles a periodic wave completes each second, measured in hertz — one hertz being one cycle per second. It's the coordinate the whole electromagnetic spectrum is laid out along. [CLIP c1 ends]

**[V: analogy — a metronome ticking slowly, then fast; a counter tallying beats per second under each]**
The everyday anchor is a metronome. Set it slow and you get one tick a second — one hertz. Speed it up and more ticks pack into the same second. Nothing about any single tick changed; only the counting rate did. Frequency is exactly that: a counting rate for wave cycles.

**[V: CORE ANIMATION — the article's two-sine figure: the "low frequency" wave draws on top with a few lazy cycles; the "high frequency" wave draws below, cramming many cycles into the same width; a one-second bracket spans both]** [CLIP c2 starts]
Put two waves side by side over the same one second and you can see it. The top wave completes a handful of cycles; the bottom wave crams in many more. Same duration, same speed of travel — the only difference is how many cycles pass per second. And notice what higher frequency costs: each individual cycle gets shorter. More cycles per second means less time — and less distance — per cycle. That's the seed of the frequency–wavelength trade you'll use constantly. [CLIP c2 ends]

**[V: formula card: "f = 1 / T"; then a stack: kHz = 10³ · MHz = 10⁶ · GHz = 10⁹; footnote card "Hz honours Heinrich Hertz, 1880s"]**
The numbers. Frequency is the inverse of period: if one cycle takes T seconds, the frequency is one over T. A 100-megahertz FM signal repeats a hundred million times a second, so each cycle lasts just ten nanoseconds. Radio numbers run big, so they're scaled: kilohertz, thousands; megahertz, millions; gigahertz, billions of cycles per second. The unit honours Heinrich Hertz, who first generated and detected radio waves in the 1880s.

**[V: card: "λ = c / f — double f, halve λ"; then a spectrum-analyzer sketch: single spike for a pure tone, widening into a band for a modulated signal]**
Two consequences follow. First, because all radio waves travel at the speed of light, frequency and wavelength are locked together — doubling the frequency halves the wavelength. Second, frequency isn't just a dial setting; it's an axis. The frequency domain — what a spectrum analyzer or an FFT shows — plots energy against frequency, where a pure tone is a single spike and a modulated signal spreads into a band around its centre frequency.

**[V: a carrier drifting slowly off a channel-edge marker; a "tolerance" bracket around it]**
Why you care: a transmitter and receiver must agree on frequency to within a small tolerance, and real oscillators don't cooperate. They drift with temperature and age, and their short-term jitter shows up as phase noise that smears the carrier and degrades demodulation. Motion shifts the apparent frequency too — Doppler shift — significant for satellites and fast vehicles. And beyond the physics there's law: the radio spectrum is divided into frequency bands, and regulators assign slices of each to specific services. A frequency is not just a number but a legal and physical context.

**[V: GopherTrunk dark-theme UI: spectrum/waterfall view; centre-frequency readout highlighted; a small offset annotation ticking as the decoder tracks]**
In GopherTrunk, tuning sets the centre frequency the SDR's local oscillator mixes down toward baseband — the slice of air that lands in the spectrum and waterfall view. And because cheap tuners are off by a few parts per million, GopherTrunk applies a PPM correction and then continuously tracks residual frequency error with automatic frequency control, so symbols stay aligned. Getting the frequency right — and keeping it right — is the first requirement for any decode.

**[V: Recap card: "① cycles per second, in hertz ② f = 1/T · double f = half λ ③ tune it right, then track it"]**
So: frequency is a counting rate — cycles per second in hertz — it's the inverse of period, it fixes the wavelength, and holding it steady is job one for every receiver. Full write-up linked below.

## Clip picks
- c1: "the number you tune to" hook + definition (~30s)
- c2: the two-sine same-second comparison — the core visual moment (~30s)
- c3 (optional): "each cycle lasts ten nanoseconds" numbers beat (~20s)

## Vertical plan
Direct center crop. The two-sine animation stacks vertically already (low over high) — it survives the 9:16 crop as-is. Re-hook first 2 s with the 162.550 dial visual.

## Assets
- Core animation: rebuild of the article's two-sine SVG (low-frequency wave above, high-frequency below), with a shared one-second bracket added
- Tuning-dial open (162.550 MHz readout)
- Metronome analogy graphic with beats-per-second counter
- Formula card (f = 1/T) + kHz/MHz/GHz scale stack + λ = c/f card (one formula on screen at a time)
- Spectrum-analyzer sketch card (spike vs band)
- GopherTrunk screen capture: spectrum/waterfall view with centre-frequency readout, dark theme
- Title + recap cards from templates

## Checklist deltas
- [ ] Analogy actually maps (metronome = counting rate; no implication that ticks travel)
- [ ] Formula card matches the article's notation exactly (f = 1/T; λ = c/f)
- [ ] The article's two-sine SVG is the storyboard for the core animation (per plan note)
