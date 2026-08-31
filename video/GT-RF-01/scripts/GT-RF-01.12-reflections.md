# GT-RF-01.12 — Reflection Coefficient (Γ)
slug: reflection-coefficient
also_slugs: [return-loss]
type: term
target: ~3:55
note: pairs Γ with return loss, the decibel repackaging of |Γ|; both article figures animated.

**[V: title | "Reflection Coefficient (Γ)" | GT-RF-01.12]**

**[V: bounce — the reflection-coefficient article's SVG animated: the incident wave runs down the line toward the ZL box; at the boundary a smaller wave peels off and travels back, labelled "reflected = Γ · incident"]** [CLIP c1 start]
The reflection coefficient measures radio's most quietly expensive problem: the part of your signal that hits a boundary and bounces back the way it came. When a travelling wave reaches a point where the impedance changes — a cable meeting an antenna, an antenna port meeting a receiver — it cannot simply continue unchanged. The boundary conditions on voltage and current must be satisfied on both sides, and the only way to satisfy them is for part of the wave to reflect. The reflection coefficient — written gamma — is the complex ratio of that reflected wave to the incident one.

**[V: definition-card | "Γ = (ZL − Z0) / (ZL + Z0) · |Γ|: 0 (matched) → 1 (total reflection)"]**
The formula is beautifully compact: gamma equals the load impedance minus the line's characteristic impedance, over their sum. Its magnitude runs from zero — a perfect match — to one — total reflection. Its phase records where in the wave's cycle the bounce happens. [CLIP c1 end]

**[V: rope — a pulse on a rope, three ways: spliced to an identical rope (passes clean), tied to a wall (returns inverted), free end (returns upright)]**
It's the same physics as a pulse on a rope. Splice the rope to an identical rope, and the pulse passes through cleanly — the matched case. Tie the end to a wall, and the pulse comes back inverted. Leave the end free, and it comes back upright. Radio has exactly those three landmarks.

**[V: three-cases — three cards animate in: ZL = Z0 → Γ = 0 · open circuit → Γ = +1, voltage doubles · short circuit → Γ = −1, voltage forced to zero; then a "|Γ|² = reflected power" stamp]** [CLIP c2 start]
Matched load, Z-L equal to Z-zero: the numerator vanishes, gamma is zero, nothing reflects, all the power enters the load. That's the design goal. Open circuit: gamma goes to plus one — full reflection, in phase, and the voltage doubles at the open end. Short circuit: gamma is minus one — full reflection, inverted, the voltage forced to zero at the short. For any passive load, gamma's magnitude never exceeds one, because a passive termination can't reflect more power than it received. And here's the twist worth remembering: reflected *power* goes as gamma magnitude squared. A gamma of zero point one sends back a tenth of the voltage — but only one percent of the power. [CLIP c2 end]

**[V: rl-ladder — the conversion card RL = −20·log₁₀|Γ|, then a ladder: 0 dB = everything reflected · 10 dB = 10% back · 20 dB = 1% back · ∞ = perfect match]** [CLIP c3 start]
Because engineers live in decibels, the mismatch is usually quoted as return loss: minus twenty log of gamma's magnitude — a scale where larger is better. Zero decibels of return loss means everything reflects: an open or a short. Ten decibels: ten percent of the power comes back. Twenty decibels: just one percent. The rule of thumb across RF engineering is that return loss above ten decibels — a standing-wave ratio below about two to one — is acceptable for most receive and many transmit uses, while sensitive or high-power systems aim for fifteen to twenty or better. Watch the sign, though: some instruments report the negative S-one-one value instead — same quantity, only the sign differs. [CLIP c3 end]

**[V: sweep — the return-loss article's SVG animated: RL versus frequency, the deep well-matched notch at resonance, the 10 dB threshold line; the usable band highlights where the curve clears it]**
Return loss is a curve, not a single number. An antenna reflects least near resonance and worsens toward the band edges, so a sweep across frequency reveals the usable bandwidth — the span where the curve stays above your threshold. A vector network analyzer or a simpler antenna analyzer measures it directly, as the magnitude of the input S-parameter, S-one-one. One last distinction: don't confuse return loss with insertion loss. Insertion loss is power *lost passing through* a component; return loss is power *turned back*. A good filter has high return loss in its passband and low insertion loss.

**[V: gt-tie-in — antenna → feedline → SDR chain; a slice of a weak control-channel signal reflects away at the input; a checklist card: "chasing a marginal channel? check the antenna's return loss first"]**
At an SDR's antenna port, the reflection coefficient decides how much of the captured signal actually crosses into the receiver instead of bouncing back up the feedline. A feedpoint impedance that drifts away from fifty ohms as you tune across a band quietly costs signal — and for a weak trunking control channel, that can be the difference between a lock and dropped frames. GopherTrunk never sees gamma: the reflection has already happened in the antenna and cabling by the time IQ samples reach the decoder. So for operators chasing a marginal channel, checking the antenna's return loss on the target frequency is a practical first step — before blaming the software.

**[V: recap | "Reflection Coefficient (Γ)" | ① Γ = (ZL − Z0)/(ZL + Z0), from 0 to 1 | ② Reflected power = |Γ|² | ③ Return loss = −20·log₁₀|Γ| — above 10 dB is acceptable]**
So: gamma is the fraction of the wave a mismatch turns back, its magnitude squared is the power you lose, and return loss is the same story in friendly decibels — keep it above ten, and your signal makes it into the radio. Full write-ups on both are linked below.
