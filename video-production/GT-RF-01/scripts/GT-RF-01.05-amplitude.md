# GT-RF-01.05 — Amplitude
slug: amplitude · type: term · treatment: 2.1 term · target: 3:30–4:30 · words: 586

**[V: Title card — "Amplitude"]**

**[V: two sine waves on one centre line — a tall one and a faint low one — the tall wave pulsing bright]** [CLIP c1 starts]
Amplitude is the loudness of a radio wave. Formally, it's the magnitude — the height — of the wave: the peak departure of its oscillation from the resting level. For radio it corresponds to signal strength, the thing a receiver reports as a power level. And it's one of exactly three properties — with frequency and phase — that fully describe a wave, and that a transmitter can vary to carry information. [CLIP c1 ends]

**[V: analogy — two people saying the same word, one whispering, one shouting; identical waveform shapes, different heights]**
The everyday anchor: a whisper and a shout. Same words, same pitch, same pace — the only difference is how hard the air gets pushed. That's amplitude. Nothing about the message changes when a signal is stronger; what changes is how much energy arrives to carry it above everything else that's shouting too.

**[V: CORE ANIMATION — the article's figure builds: centre line, the large-amplitude wave, then the smaller wave overlaid on the same line; the dashed height marker rises with the label "larger amplitude = stronger signal"]** [CLIP c2 starts]
Now the one relationship worth engraving: amplitude and power are linked by a square. The power a wave carries is proportional to the square of its amplitude — so doubling the amplitude quadruples the power. That square is why engineers stop talking about amplitude the moment a signal reaches a receiver and switch to power instead, and why levels are quoted logarithmically in decibels: a scale that turns those wide multiplicative ranges into manageable additions. [CLIP c2 ends]

**[V: card: "peak vs RMS" — a jagged noisy waveform with its peak marked and a lower RMS line through it]**
A refinement: a single clean sinusoid can be summarised by its peak amplitude, but real signals are messy, so engineers more often use the RMS — root-mean-square — amplitude, which relates directly to average power. The gap between peak and average, the crest factor, is what decides how much headroom an amplifier needs.

**[V: envelope card: a fast carrier with a slow outline traced over its peaks; a diode-detector icon recovering the outline]**
And amplitude was radio's first mailbox. The slowly varying outline traced by a modulated carrier's amplitude is its envelope, and amplitude modulation — AM — writes the message directly into that envelope, so simply that a lone diode detector can recover it. Varying amplitude is still one axis of modern QAM.

**[V: a wave leaving a transmitter and visibly shrinking with distance; a noise haze at the bottom of frame it gradually sinks toward]**
Amplitude is also the property the journey eats. As a wave spreads out, its amplitude falls — in free space with the square of distance — and obstacles, absorption, and destructive multipath take more. That steady weakening is why a distant station is faint and why link budgets exist. At the receiver, what's left competes with the ever-present noise floor, and that ratio — signal to noise — ultimately bounds whether the message can be recovered.

**[V: gain-staging card: a signal bar sliding between "quantisation noise" at the bottom and "clipping" at the top, with a sweet-spot zone marked]**
Why you care day to day: gain staging. Your front-end amplifiers and attenuators set where the signal's amplitude lands relative to the converter's range. Too little, and it sinks into quantisation noise. Too much, and it clips — generating intermodulation products that trash the whole band. Automatic gain control exists to hold it in the sweet spot, because amplitude isn't static: multipath and motion make it fade and flutter, and receivers must ride that out.

**[V: GopherTrunk dark-theme UI: signal meter in dBFS beside the spectrum/waterfall view; IQ-plane inset showing magnitude √(I²+Q²)]**
Inside GopherTrunk, the wave is IQ samples, and a sample's amplitude is its distance from the origin of the IQ plane — the magnitude, root of I-squared plus Q-squared. The signal meter in dBFS is that number, reported against the converter's full scale. GopherTrunk reads it to judge whether a channel is active, to drive automatic gain control, and to compute the SNR estimates that gate weak decodes. The four-level and π/4-DQPSK signals it decodes are largely constant-envelope, so there amplitude serves as a health and gain-staging indicator rather than the information itself.

**[V: Recap card: "① amplitude = the wave's height = strength ② power ∝ amplitude² — double it, 4× power ③ in an SDR: √(I²+Q²), shown in dBFS"]**
So: amplitude is the height of the wave, power grows with its square, and in an SDR it becomes the magnitude of every IQ sample — the number your signal meter is showing you. Full write-up linked below.

## Clip picks
- c1: "the loudness of a radio wave" hook + definition (~30s)
- c2: "double the amplitude, quadruple the power" square-law beat (~30s)
- c3 (optional): gain-staging trap — quantisation noise vs clipping (~25s)

## Vertical plan
Direct center crop; two-wave animation and slider cards center-safe. Re-hook first 2 s with the tall-vs-faint wave visual.

## Assets
- Core animation: rebuild of the article's two-amplitude SVG (shared centre line, dashed height marker, "larger amplitude = stronger signal" label)
- Whisper/shout analogy graphic
- Peak-vs-RMS card; envelope/diode-detector card; gain-staging slider card
- Shrinking-wave-with-distance animation over a noise haze
- GopherTrunk screen capture: signal meter in dBFS + spectrum/waterfall view, IQ-magnitude inset, dark theme
- Title + recap cards from templates

## Checklist deltas
- [ ] Analogy actually maps (whisper/shout changes only energy, not content — stated explicitly)
- [ ] Formula card matches the article's notation exactly (power ∝ amplitude²; √(I² + Q²); one formula on screen at a time)
