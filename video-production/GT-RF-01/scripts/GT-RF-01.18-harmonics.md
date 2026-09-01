# GT-RF-01.18 — Harmonics
slug: harmonics · type: term · treatment: 2.1 term · target: 3:30–4:30 · words: 631

**[V: Title card — "Harmonics"]**

**[V: a spectrum: one tall carrier at 150 MHz; ghost copies fade in at 300 and 450 MHz]** [CLIP c1 starts]
Harmonics are the echoes a radio signal leaves up the dial — extra copies at exactly two times, three times, four times its frequency. Transmit at one hundred fifty megahertz and there's a ghost of your signal at three hundred, and another at four fifty. Harmonics are spectral components at integer multiples of a signal's fundamental frequency, they appear whenever a device bends a sine wave, and they're one of the main things a transmitter must clean up before its signal is allowed near the antenna. [CLIP c1 ends]

**[V: a guitar string vibrating; an overtone ladder stacks above the fundamental note on a small spectrum]**
Music got here first. Pluck a guitar string and you don't hear one pure tone — you hear the fundamental plus overtones at integer multiples of it, and that stack is what makes the note sound rich. Radio harmonics sit on exactly the same ladder of whole-number multiples. The difference is where they come from: a string's overtones are its physics; a radio's harmonics are a defect — the fingerprint of distortion.

**[V: a transfer curve, straight then bending; a clean sine enters, a flattened wave exits, and new spectral lines grow at 2f₀ and 3f₀]**
Here's the mechanism. A perfectly linear device passes a sine wave through unchanged — the output is a scaled copy of the input. Real components bend: a power amplifier driven near saturation, a diode, a mixer. Write that bent response as a power series — a squared term, a cubed term — and feed a pure sine in. The squared term generates energy at twice the frequency; the cubed term, at three times. The harder you drive into the bend, the richer the harmonic content.

**[V: article's harmonic-spectrum figure animates: tall f₀, then 2f₀, 3f₀, 4f₀ appearing progressively shorter; even bars dim under a "push-pull" label]** [CLIP c2 starts]
Two useful facts fall out of that math. Even-order bending makes even harmonics and odd-order makes odd ones — so a symmetric, push-pull amplifier stage suppresses the even harmonics naturally. And harmonics always land *above* the fundamental at exact integer multiples — which makes them predictable, and predictable means filterable. Add up all the harmonic power and compare it with the fundamental and you get total harmonic distortion — THD — the standard score for how bent a device is. [CLIP c2 ends]

**[V: 150 MHz transmitter → low-pass filter block → spectrum: 150 passes, the 300 MHz line drops by "tens of dB"]**
So in practice, a transmitter kills them with a low-pass or band-pass filter after the final amplifier. A one-fifty-megahertz VHF transmitter runs a low-pass filter that passes one fifty but rejects the second harmonic at three hundred by tens of decibels. The most efficient amplifier classes — class C and switching designs — are efficient precisely *because* they run nonlinearly, so they lean hardest on that output filter. And regulators enforce all of it: type-approval rules set hard limits on harmonic radiation, because your second harmonic lands in somebody else's band.

**[V: waterfall: a strong FM broadcast carrier; a faint spur appears at double its frequency inside a monitored band; an arrow halves the frequency back to the culprit]** [CLIP c3 starts]
Harmonics bite on the receive side too. A strong local FM broadcast can present a harmonic that lands right in a band you're monitoring, and your receiver's own front end and local oscillator generate harmonics that create spurious responses of their own. So here's a working diagnostic: when you find a mystery carrier, halve its frequency and look down there. A strong real signal at exactly half is the giveaway that your "spur" is a harmonic of it — not a real transmission. [CLIP c3 ends]

**[V: GopherTrunk web UI, dark theme: waterfall with the noise floor visibly rising under front-end overload; card: "band-pass filter ahead of the SDR"]**
GopherTrunk is receive-only, so it makes no harmonics of its own — but the budget SDR front ends it often runs behind have limited selectivity, and a strong out-of-band signal or its harmonic can overload the tuner and raise the effective noise floor across the whole waterfall, degrading decode of the control channel you actually want. The standard cure is an external band-pass filter ahead of the SDR — and knowing where harmonics fall is how you work out which strong neighbour to blame.

**[V: Recap card: "① harmonics = exact integer multiples of f₀ ② born from nonlinearity — the bend in the curve ③ predictable → filterable, by law on TX, for your noise floor on RX"]**
So: harmonics live at exact integer multiples of the fundamental; they're born wherever a device bends a sine wave; and because they're predictable, a filter kills them — required by law on transmit, and worth adding on receive for the sake of your own noise floor. Full write-up linked below.

## Clip picks
- **c1** (~30 s): the ghost-copies hook — 150 → 300 → 450 MHz on a spectrum. Self-contained.
- **c2** (~30 s): even/odd + "predictable means filterable" + THD — the tidy-facts beat.
- **c3** (~25 s): the halve-the-frequency diagnostic — practical, contrarian-shareable. Shorts title: "That signal might not be real".

## Vertical plan
Direct center crop. The harmonic-spectrum ladder is naturally horizontal — prepare a tall re-layout with the frequency axis running vertically (bars become horizontal, stacking upward: f₀ at bottom, 4f₀ at top). All cards center-safe. Re-hook: burned text "Your signal has ghosts" over the 150/300/450 spectrum.

## Assets
- Title + recap cards (templates)
- **Core animation:** rebuilt from the article's inline SVG — the fundamental-plus-shrinking-harmonics spectrum; extended with the even-bars-dim push-pull beat
- Guitar-string overtone animation (short)
- Bent-transfer-curve animation (sine in → distorted out → new spectral lines)
- 150 MHz TX + low-pass filter diagram; halve-the-frequency waterfall mock
- GopherTrunk screen capture: waterfall + noise floor under overload (10–20 s, dark theme)
- VO track; calm/technical music bed

## Checklist deltas
- [ ] Analogy check: guitar overtones share only the integer-multiple *placement*; the script must (and does) flag that the cause differs — physics vs distortion
- [ ] No power-series formula on screen (spoken only: "a squared term, a cubed term") — keeps the one-formula budget for THD if shown; if a THD card is used, it's ratio-of-powers wording per article
- [ ] Even/odd mapping stated the right way: even-order terms → even harmonics, odd-order → odd (article is explicit)
- [ ] "Tens of dB" for the 300 MHz rejection quoted as in the article — no invented specific figure
- [ ] Harmonics-vs-intermodulation distinction kept to the one article point (harmonics fall above, at multiples, hence filterable) without opening an intermod digression
