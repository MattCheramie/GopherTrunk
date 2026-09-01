# GT-RF-01.17 — Resonance
slug: resonance · type: term · treatment: 2.1 term · target: 3:30–4:30 · words: 632

**[V: Title card — "Resonance"]**

**[V: a child's swing pushed with two fingers, arc growing; hard cut to a dipole antenna with a wavelength ruler beside it]** [CLIP c1 starts]
Resonance is why a child on a swing can be sent soaring with tiny, well-timed pushes — and it's why an antenna has a natural size. Resonance is the condition where a system stores and exchanges energy most readily at one particular frequency, so a small periodic drive produces a large response. Tuning a radio, the sharpness of a filter, the length of an antenna — all of it is this one idea.

**[V: swing animation: pushes timed to the swing's rhythm build the arc; mistimed pushes fight it and the arc dies]** [CLIP c1 ends]
Stay on the swing a moment. Push at exactly the swing's own rhythm and every push adds energy — the arc grows and grows. Push at any other rhythm and you fight the swing as often as you help it, and nothing builds. The swing has a natural frequency, and only a drive at that frequency accumulates. Every resonant system — a struck bell, a plucked string, a tuned circuit — is that swing: energy sloshing between two stores at a rate the system itself chooses.

**[V: article's LC-tank figure builds; then a graph: XL = 2πfL rising line, XC = 1/(2πfC) falling line, crossing at one frequency]** [CLIP c2 starts]
In a circuit, the two stores are an inductor's magnetic field and a capacitor's electric field. The inductor's reactance rises with frequency; the capacitor's falls. Plot both, and at exactly one frequency the lines cross — equal in size, opposite in sign, so they cancel. That crossover is resonance. The reactive parts of the impedance vanish, the circuit looks purely resistive, voltage and current fall back into phase, and the system exchanges energy with its drive as efficiently as it ever can. [CLIP c2 ends]

**[V: formula card: "f₀ = 1 / (2π√(LC))"; then a split card: series LC = minimum impedance at f₀ · parallel tank = maximum impedance at f₀]**
The crossover has a formula: f-zero equals one over two pi, root L C. Bigger inductance or bigger capacitance means a lower natural frequency — a heavier swing, swinging slower. And the same two parts give you mirror-image tools. In series, L and C present *minimum* impedance at resonance — a path that passes one frequency and blocks the rest. In parallel — a tank — they present *maximum* impedance there, a wall that selects one frequency out of a series path. Oscillators and filters are built from those two moves.

**[V: half-wave dipole with a standing wave on it; the wavelength ruler shows the element length matching half the wavelength]** [CLIP c3 starts]
And here's the payoff for antennas. An antenna is itself a resonant structure: a half-wave dipole resonates where its physical length matches half the signal's wavelength. That's why antennas are cut for a target band — the metal has a natural frequency exactly the way the swing has a natural rhythm, and a wave at that frequency builds a large response in it. The wrong length isn't just a smaller antenna. It's a swing being pushed off the beat. [CLIP c3 ends]

**[V: two response curves on one axis: tall narrow peak labeled "high Q — crystal, cavity" vs broad hump labeled "low Q"]**
How sharp the peak is depends on loss, captured by the Q factor. High-Q resonators — quartz crystals, coaxial cavities — give a tall, narrow peak that selects one frequency tightly; a lossy circuit gives a broad hump that responds over a wide span. Radio keeps a whole zoo of resonators — lumped LC circuits, crystals, cavities, dielectric pucks, mechanical structures — each chosen for the stability and bandwidth a job needs.

**[V: GopherTrunk web UI, dark theme: wideband spectrum, then the channelizer isolating one narrow channel from it; decode indicators live]**
Around any SDR, resonance is the analog scaffolding. Preselector and RF filter stages use resonant elements to pass the wanted band and keep strong out-of-band energy from overloading the front end, and the local oscillator the receiver mixes against is steadied by a high-Q resonant reference. GopherTrunk does its frequency selection downstream, in software: its digital channelizer carves the wanted channel out of the wideband stream, playing the role a tuned circuit plays in an analog radio. But the physical resonances upstream — the antenna, the preselector, the reference — still bound the quality of everything that reaches the decoder.

**[V: Recap card: "① resonance = reactances cancel, response peaks ② f₀ = 1/(2π√(LC)) ③ an antenna is a resonator — cut it for the band"]**
So: resonance is the frequency where a system's reactances cancel and its response peaks; the formula is f-zero equals one over two pi root L C; and an antenna is a resonator — cut it to the band, like timing your push to the swing. Full write-up linked below.

## Clip picks
- **c1** (~35 s): swing hook + definition + the timed-vs-mistimed push beat — visual, loopable, zero context needed.
- **c2** (~30 s): the two reactance lines crossing and canceling — the mechanism in one picture.
- **c3** (~25 s): "why antennas have a size" — the dipole/half-wavelength beat. Shorts title: "Why your antenna has to be THIS long".

## Vertical plan
Direct center crop. The XL/XC crossing graph and the response-curve pair are the only wide visuals — prepare tall re-layouts (frequency axis running vertically works for both). Swing and dipole animations composed center-safe from the start. Re-hook: burned text "Why antennas have a natural size" over the swing shot.

## Assets
- Title + recap cards (templates)
- **Core animation:** rebuilt from the article's inline SVG — the LC tank next to the response curve peaking at f₀; extend with the XL/XC crossing-lines build
- Swing animation (timed vs mistimed pushes)
- Half-wave dipole + wavelength-ruler animation
- Cards: f₀ formula · series-vs-parallel mirror card · high-Q vs low-Q curves
- GopherTrunk screen capture: wideband spectrum → channelizer isolating a channel (10–20 s, dark theme)
- VO track; calm/technical music bed

## Checklist deltas
- [ ] Analogy check: the swing maps drive-at-natural-frequency → accumulating amplitude; do not let it imply resonance needs repeated discrete "pushes" (a continuous sine drive is the circuit case)
- [ ] Formula card matches article notation exactly: *f₀ = 1/(2π√(LC))* — the only formula on screen
- [ ] Series = minimum impedance / parallel = maximum impedance stated the right way around (easy to flip; article is explicit)
- [ ] Q factor mentioned but not defined in depth (its own entry; no cross-segment reference on screen)
- [ ] GT tie-in wording matches article: channelizer plays the tuned-circuit role; GopherTrunk does not implement analog resonance
