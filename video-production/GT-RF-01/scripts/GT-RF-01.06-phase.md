# GT-RF-01.06 — Phase
slug: phase · type: term · treatment: 2.1 term · target: 3:30–4:30 · words: 554

**[V: Title card — "Phase"]**

**[V: two identical sine waves scrolling, one slightly shifted behind the other; the gap between their crests pulses]** [CLIP c1 starts]
Phase is the subtlest of a wave's three properties — and the one modern digital radio leans on hardest. Phase is the position of a point within the cycle of a wave, expressed in degrees, zero to 360, or radians, zero to two pi. Two waves of the same frequency can differ in phase, meaning one is shifted in time relative to the other: a quarter-cycle lag is a 90-degree phase difference. It doesn't change a wave's strength or its pitch — just where it is in its cycle. And that's enough to send bits. [CLIP c1 ends]

**[V: analogy — a clock face; the hand sweeps steadily; a second clock beside it runs at the same speed but its hand points 90° behind]**
The everyday anchor is a clock face. A wave cycling is a hand sweeping around a dial — one lap per cycle, and the speed of the sweep is the frequency. Phase is where the hand points right now. Two clocks can tick at exactly the same rate and still disagree: one reads twelve while the other reads three. Same frequency, ninety degrees of phase apart. Phase is only meaningful relative to a reference — another wave, or a notional clock ticking at the carrier frequency.

**[V: CORE ANIMATION — the article's figure builds: solid sine draws; the dashed identical sine draws offset behind it; the bracket beneath labels the gap "phase difference"; then the whole picture morphs into a rotating point on the IQ plane, angle arc highlighted]**
That clock isn't just a metaphor — it's literally how radios represent signals. On the IQ plane, a sample's angle measured from the positive I axis is its phase, and its distance from the origin is its amplitude. A rotating point traces out the wave: constant frequency is steady rotation, and a sudden change of angle is a phase shift. Amplitude and phase — the two independent quantities of a bandpass signal — become the polar coordinates of one complex number.

**[V: constellation card: BPSK's two points at 0° and 180°; then QPSK's four points spaced 90° apart, each tagged with a bit pattern]** [CLIP c2 starts]
Here's why that matters: phase can be changed instantly and read out precisely, which makes it prime real estate for carrying data. Phase-shift keying assigns bit patterns to discrete phase positions. BPSK uses two — zero and 180 degrees. QPSK uses four, spaced 90 degrees apart — two bits per symbol. The receiver just measures the angle of each symbol and decides which was sent. And differential schemes like pi-over-four D-QPSK go one cleverer: they encode data in the *change* of phase between successive symbols, sidestepping the need to know the absolute phase reference at all. [CLIP c2 ends]

**[V: a constellation slowly rotating off its ideal points; a loop symbol locks it back; then the points smear into fuzzy clouds]**
The catch: the receiver's idea of "zero phase" must be locked to the transmitter's, or the whole constellation rotates. A Costas loop or a phase-locked loop estimates and removes that offset continuously — this is carrier recovery. And real oscillators jitter, smearing each symbol's angle into a fuzzy cloud called phase noise; too much of it collapses the margin between adjacent points and forces errors. Symmetric constellations even allow locking 90 or 180 degrees off — differential coding or a known sync pattern resolves which rotation is correct.

**[V: GopherTrunk dark-theme UI: live decode panel running; annotation on the carrier-recovery step; an EVM readout highlighted]**
The digital voice signals GopherTrunk decodes all live in the angle of the IQ samples, so its carrier-recovery loop estimates the incoming phase and rotates each symbol back onto the ideal constellation before slicing it to bits. Watch the live decode panel on a marginal signal: residual phase error directly raises the error-vector magnitude, and past a point, it breaks the decode. When symbols garble, phase is usually where the story is.

**[V: Recap card: "① phase = position in the cycle (0–360°) ② on the IQ plane: phase = angle, amplitude = distance ③ jumping phase between fixed values = digital bits"]**
So: phase is where a wave sits in its cycle, it lives as the angle of every IQ sample, and deliberately jumping it between fixed positions is how digital radios send bits. Full write-up linked below.

## Clip picks
- c1: "the subtlest property — and the one digital radio leans on hardest" hook + definition (~35s)
- c2: BPSK → QPSK constellation beat ("measure the angle, read the bits") (~30s)
- c3 (optional): the fuzzy-cloud phase-noise beat (~20s)

## Vertical plan
Direct center crop; clock, IQ plane, and constellations are naturally square/centered. Re-hook first 2 s with the two-offset-waves visual.

## Assets
- Core animation: rebuild of the article's offset-sines SVG with "phase difference" bracket, morphing into the rotating IQ-plane point
- Twin clock-face analogy animation (same rate, 90° apart)
- Constellation cards: BPSK (0°/180°) and QPSK (four points, 90° spacing)
- Rotating/smearing constellation card (carrier recovery + phase noise)
- GopherTrunk screen capture: live decode panel with EVM readout on a marginal signal, dark theme
- Title + recap cards from templates

## Checklist deltas
- [ ] Analogy actually maps (clock = rotating phasor; sweep rate = frequency, hand position = phase — same picture the IQ plane formalizes)
- [ ] Formula card matches the article's notation exactly (degrees 0–360° / radians 0–2π; no formula card otherwise — constellations carry the math)
