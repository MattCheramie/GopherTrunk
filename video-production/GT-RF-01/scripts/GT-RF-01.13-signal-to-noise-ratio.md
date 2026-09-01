# GT-RF-01.13 — Signal-to-noise ratio (SNR)
slug: signal-to-noise-ratio · type: concept · treatment: 2.2 concept + 2.4 demo beat (pillar's ~50% demo) · target: 3:30–4:30 · words: 607

**[V: Title card — "Signal-to-noise ratio (SNR)"]**

**[V: a spectrum with a modest signal; a "gain" slider drags up — signal AND grass rise together; a decode indicator stays red]** [CLIP c1 starts]
Signal-to-noise ratio is the answer to one of radio's most maddening puzzles: why turning up the gain doesn't help. Here's the scene. A distant system won't decode, so you raise the gain. The signal climbs on the meter — taller, brighter — and still: nothing. Because the noise floor climbed by exactly as much. The receiver never cared how strong the signal was. It cares about one thing only: the *gap*.

**[V: the article's figure: noisy baseline, one tall peak, and a vertical bracket between peak and floor labelled "SNR"]**
That gap is the signal-to-noise ratio: the difference, in decibels, between the signal's power and the noise floor. A signal at minus eighty-five dBm over a minus one-oh-five dBm floor has twenty dB of SNR. It is the single best predictor of whether a signal will decode — enough SNR and the bits come out clean; too little and errors overwhelm the demodulator. And because it's a *difference* of two readings, any constant calibration offset in the receiver cancels out — which is why SNR is meaningful even on cheap SDR hardware that can't report true absolute power. [CLIP c1 ends]

**[V: card: "noise power scales with bandwidth (~3 dB per doubling)" — a filter bracket narrows and the SNR bracket grows]**
One honest footnote before the payoff: SNR depends on bandwidth. Noise power scales with the width you integrate it over — about three dB per doubling — so an SNR figure is only well defined against a stated bandwidth, and comparing across systems means normalising to a common one. The practical upside: narrowing the receive filter to the signal's occupied bandwidth improves SNR directly. It admits less noise while keeping all the signal.

**[V: the "waterfall curve": bit error rate vs SNR — flat and negligible above a threshold, collapsing within a few dB below it; a card slides in: "FM ~10–12 dB · P25-class digital ~15–20 dB"]** [CLIP c2]
Now the concept that makes digital radio feel binary: the threshold. Every digital mode has a threshold SNR below which the error rate climbs steeply — the curve is flat above it, then collapses within just a few dB below. Analog FM voice becomes readable around ten to twelve dB. C4FM and CQPSK digital modes like P25 want roughly fifteen to twenty at the demodulator for a clean lock, with forward error correction buying some margin below that. Three dB over threshold is usable but fragile. Ten over is comfortable. That cliff is why digital signals seem either perfect or gone.

**[V: DEMO, ~60 s — GopherTrunk replaying two captures of the same channel, dark theme, panels zoomed. Capture A: demod SNR readout in the high teens, EVM low → sync locks, frames tick, voice decodes. Capture B: SNR readout ~10 dB, EVM high → no lock. Freeze-frames with accent-color callouts on the SNR/EVM readouts]** [CLIP c3]
Let's make that real. GopherTrunk reports per-channel demodulator SNR and EVM — error vector magnitude — so watch two replays of the same control channel. First capture: the SNR readout sits in the high teens — right in that fifteen-to-twenty clean-lock zone — EVM is low, sync locks, and the decode counters just tick. Second capture, same channel, weaker: the readout hovers around ten dB. The signal is plainly *there* on the waterfall — and the decoder gets nothing. Nothing else changed. Same software, same channel — five or six dB of gap is the entire difference between a working receiver and a dead one. That readout also tells you *what kind* of problem you have: a failed decode with healthy SNR isn't a weak signal — go look for overload or wrong tuning instead.

**[V: card: "improve the gap: antenna · height/site · correct gain · LNA · matched filter"; a small footer: "C = B·log₂(1+SNR)"]**
So how do you win dB you can actually keep? Not the gain slider — the gap. A better antenna. A higher or clearer site. Correct gain staging, a low-noise amplifier, matched filtering. That's what moves a marginal signal from un-decodable to clean. And there's a ceiling on what SNR can ever buy: channel capacity grows with the log of one plus SNR, so every doubling of SNR yields diminishing returns — which is why real systems just aim to sit a few dB above their threshold, reliably.

**[V: Recap card: "① SNR = signal − noise floor, in dB — gain lifts both ② every mode has a threshold: FM ~10–12, P25-class ~15–20 ③ win the gap: antenna, site, filter"]**
So: signal-to-noise ratio is the gap, not the height — every mode lives or dies at its threshold, and the way past a marginal signal is a better gap, not a bigger number on the gain slider. Full write-up linked below.

## Clip picks
- c1: the gain-slider problem-story + "the gap" reveal (~40s)
- c2: the threshold cliff — "why digital is either perfect or gone" (~30s)
- c3: the two-capture GopherTrunk demo, SNR readout on screen — the pillar's flagship short (~45s)

## Vertical plan
Peak-plus-bracket figure and threshold curve get tall re-layouts (both are wide). The demo needs zoom-crop pans onto the SNR/EVM readouts — verify readable at phone size. Re-hook first 2 s with the gain-slider-futility visual.

## Assets
- Core animation: rebuild of the article's signal-over-floor SVG with the vertical SNR bracket
- Gain-slider open (signal + floor rising together)
- Bandwidth/filter animation; threshold "waterfall curve" animation with mode-threshold card
- GopherTrunk screen captures: two replays of the same channel (strong + marginal) with demod SNR/EVM readouts visible; freeze-frame callout overlays (accent color)
- Improvement checklist card; recap/title cards from templates

## Checklist deltas
- [ ] Problem is stated before the solution (gain futility before "the gap")
- [ ] The concept's name is spoken within the first 20 s (it's in sentence one)
- [ ] Demo narration uses only article-sourced thresholds (10–12 dB FM, 15–20 dB C4FM/CQPSK, 3 dB fragile / 10 dB comfortable); on-screen readouts are real replay output, never staged
- [ ] Pillar placement note: this is the ~50% mark demo — the midpoint recap follows two segments later; keep this segment self-contained
