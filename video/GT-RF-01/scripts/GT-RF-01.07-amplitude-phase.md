# GT-RF-01.07 — Amplitude
slug: amplitude
also_slugs: [phase]
type: term
target: ~3:50
note: double-header segment — covers amplitude AND phase, the two wave properties remaining after frequency; both article figures are animated.

**[V: title | "Amplitude" | GT-RF-01.07]**

**[V: two-signals — the amplitude article's SVG animated: two sine waves share a centre line; the taller one glows with the "larger amplitude = stronger signal" label; a signal meter climbs as its height grows]** [CLIP c1 start]
Amplitude is the height of the wave — the peak departure of its oscillation from the resting level. In radio, that height is signal strength: the taller the wave arriving at your antenna, the more power it delivers to the receiver, and the easier everything downstream becomes. Together with frequency, amplitude and its quieter sibling — phase — are the three properties that fully describe a wave, and the three knobs a transmitter can vary to carry information. This segment covers the two we haven't met yet.

**[V: definition-card | "Amplitude = wave height → power · Phase = position within the cycle (0–360°)"]**
Two definitions, then. Amplitude is the magnitude of the oscillation. Phase is the position of a point within the wave's cycle, measured in degrees — zero to three hundred sixty — or in radians. [CLIP c1 end]

**[V: swing — a playground swing; a bracket for how far it travels (amplitude) and a marker for where it sits in the arc right now (phase); a second identical swing runs out of step]**
Picture a playground swing. How far it swings — that's amplitude. Where it happens to be in its arc at this instant — rising, at the top, passing through the middle — that's phase. Two swings can move at exactly the same rhythm and still be out of step with each other, and that offset is a phase difference: a quarter-cycle of lag is a ninety-degree phase difference. Phase only means something relative to a reference — another wave, or a clock ticking at the carrier frequency.

**[V: power-square — the amplitude arrow doubles; a power readout jumps ×4; the scale flips to logarithmic decibels]**
Amplitude and power are tightly linked: the power a wave carries is proportional to the square of its amplitude, so doubling the height quadruples the power. That's why power is the natural currency at the receiver — reported in d-B-m, or inside an SDR in d-B-F-S — and why levels ride a logarithmic decibel scale. Amplitude never stands still, either: it falls as the wave spreads out from the transmitter, obstacles and multipath cut it further, and at the receiver it has to stand above an ever-present noise floor.

**[V: phase-offset — the phase article's SVG animated: two identical sines slide apart, the "phase difference" bracket stretching; then an IQ plane fades in — a rotating point, radius labelled amplitude, angle labelled phase]** [CLIP c2 start]
Radios make phase geometric. On the IQ plane, a signal is a rotating point: its distance from the origin is the amplitude, and its angle, measured from the I axis, is the phase. Constant frequency is steady rotation; a sudden change of angle is a phase shift. That one picture — amplitude and phase as the polar coordinates of a single complex number — is why the IQ representation is so powerful, and it's how every signal in this course actually lives inside a receiver. [CLIP c2 end]

**[V: psk — constellation builds: two dots at 0° and 180° (BPSK), then four spaced 90° apart (QPSK); bit patterns label each dot; symbols hop between positions]** [CLIP c3 start]
And because phase can be changed instantly and read out precisely, it is prime real estate for carrying data. Phase-shift keying assigns bit patterns to discrete phase positions — BPSK uses two, at zero and one hundred eighty degrees; QPSK uses four, spaced ninety apart — and the receiver decides what was sent by measuring the angle of each symbol. Differential schemes go one step slyer: they encode data in the *change* of phase between successive symbols, sidestepping the need for an absolute reference. [CLIP c3 end]

**[V: hazards — the constellation slowly rotates (unlocked carrier); dots smear into fuzzy clouds (phase noise); amplitude bars flicker (fading)]**
Each property has its failure mode. If the receiver's idea of zero phase isn't locked to the transmitter's, the whole constellation rotates — a Costas loop or phase-locked loop estimates and removes that offset continuously. Real oscillators jitter, smearing each symbol's angle into a fuzzy cloud called phase noise. And amplitude fades and flutters as reflections come and go, so front-end gain staging has to keep the signal above the quantization noise without clipping it.

**[V: gt-tie-in — GopherTrunk IQ view: magnitude √(I²+Q²) drives activity detection and AGC meters; the angle feeds the carrier-recovery loop and a live constellation]**
In GopherTrunk, both live in every IQ sample. The magnitude — the square root of I squared plus Q squared — is the amplitude, and it drives channel-activity detection, automatic gain control, and the SNR estimates that gate weak decodes. The angle is the phase, and that's where the digital voice schemes GopherTrunk decodes keep their bits: the carrier-recovery loop rotates each incoming symbol back onto the ideal constellation before slicing it. Residual phase error shows up directly as error-vector magnitude — and past a point, it breaks the decode.

**[V: recap | "Amplitude" | ① Amplitude = height; power ∝ amplitude² | ② Phase = position in the cycle — the IQ angle | ③ PSK puts the bits in the phase]**
So: amplitude is the wave's strength, and power grows as its square; phase is the wave's position in its cycle, the angle on the IQ plane; and jumping that angle between fixed values is how digital radios talk. The full write-ups on both are linked below.
