# GT-RF-01.11 — Impedance (Z)
slug: impedance
type: term
target: ~3:50

**[V: title | "Impedance (Z)" | GT-RF-01.11]**

**[V: fifty-ohm — a parade of RF gear: SDR input jack, coax, antenna, filter — every port stamped "50 Ω"]** [CLIP c1 start]
Impedance is the number printed on practically every piece of radio equipment you'll ever own: fifty ohms. Your SDR's antenna jack — fifty ohms. Your coax, your antenna, your filters — fifty ohms. Why does an entire industry agree on one number? Because impedance is the total opposition a circuit, component, or transmission line presents to an alternating current — and when the source, the line, and the load all present the *same* impedance, power flows between them cleanly, with nothing reflected.

**[V: definition-card | the article's SVG: source → Z0 = 50 Ω line → load, beside the phasor diagram Z = R + jX]**
Formally, impedance is a complex number: Z equals R plus j X, in ohms. The real part, R, is resistance — it dissipates energy. The imaginary part, X, is reactance — it stores energy and gives it back, through capacitance or inductance. [CLIP c1 end]

**[V: spring-friction — plumbing analogy: a narrow rough pipe (friction = resistance, same at any speed) and a springy membrane (stores & returns = reactance); a frequency knob changes how the membrane fights back]**
Think of pushing water through plumbing. A narrow, rough pipe resists flow and turns your effort into heat — that's resistance, the same at any speed. A springy membrane across the pipe pushes back, stores your push, and returns it a moment later — that's reactance, and it depends on how fast you push. An inductor's reactance rises with frequency — two pi f L. A capacitor's falls with frequency — one over two pi f C. And because reactance shifts the current's timing relative to the voltage, one plain number can't capture it: the complex Z carries the amplitude ratio in its magnitude and the phase shift in its angle.

**[V: line — a wave launched down a coax cross-section; the card Z0 = √(L/C); a stamped caption: "not measurable with an ohmmeter"]** [CLIP c2 start]
A transmission line like coax adds a twist of its own: characteristic impedance, Z-zero, set purely by the line's geometry and dielectric — for a lossless line, the square root of its per-meter inductance over its per-meter capacitance. Crucially, Z-zero is not a resistance you can measure with an ohmmeter. It's the impedance the line *looks like* to a signal travelling down it — the ratio of voltage to current for the moving wave itself. [CLIP c2 end]

**[V: matching — source Zs, line Z0, load ZL in a row; when all match, the wave flows through whole; a mismatched load sends a partial wave back and standing-wave ripple builds on the line]**
And here's why impedance dominates RF engineering: the matching condition. Maximum power transfers — and no energy reflects — only when the impedances are equal: the load matching the line, the line matching the source. Mismatch a boundary, and part of the wave is sent back toward the source, setting up standing waves. That reflected fraction gets a whole vocabulary of its own — reflection coefficient, return loss, standing-wave ratio — all describing the same mismatch.

**[V: fifty-compromise — a dial sweeping from 30 Ω ("peak power") to 77 Ω ("lowest loss"); the needle settles at 50; a side note: "75 Ω — video & broadcast"]**
So why fifty, exactly? A historical compromise. Coaxial lines carry peak power best near thirty ohms and have their lowest loss near seventy-seven; fifty sits usefully between them with convenient dimensions. Video and broadcast gear standardized on seventy-five ohms instead. Antennas, meanwhile, rarely land exactly on fifty ohms across a whole band — so antenna tuners and matching networks exist to pull them back: series and shunt inductors and capacitors, quarter-wave line sections, and transformers, chosen to cancel the load's reactance and rotate its resistance to the target. The Smith chart is the classic graphical tool that turns those matching moves into geometric steps.

**[V: gt-tie-in — SDR front-end diagram: 50 Ω input → LNA → filters → mixer → ADC; a mismatched antenna reflects a slice of a weak control channel away before the ADC]**
Every software-defined radio front end presents a nominal fifty-ohm input at its antenna connector, and the low-noise amplifier, filters, and mixer behind it are all designed around that reference. A badly mismatched antenna or feedline reflects part of the received signal before it ever reaches the ADC, degrading sensitivity — though on receive, a mismatch mainly costs signal rather than damaging hardware. GopherTrunk lives entirely downstream, on the digital IQ stream, so it can't measure or correct impedance itself. But a good match raises the signal-to-noise ratio the decoder ultimately sees — and poor matching is a common cause of weak control-channel reception that no amount of DSP can recover.

**[V: recap | "Impedance (Z)" | ① Z = R + jX — resistance plus reactance, in ohms | ② Z0 = what a line looks like to a travelling wave | ③ Match = max power, no reflection; 50 Ω is the deal]**
So: impedance is resistance plus reactance in one complex number, a transmission line has a characteristic impedance all its own, and matching everything to the fifty-ohm reference is how power gets from the antenna into the receiver without bouncing back. Full write-up linked below.
