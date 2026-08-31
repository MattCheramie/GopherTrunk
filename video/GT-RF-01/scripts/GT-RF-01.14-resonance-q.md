# GT-RF-01.14 — Resonance
slug: resonance
also_slugs: [q-factor]
type: term
target: ~3:50

**[V: title | "Resonance" | GT-RF-01.14]**

**[V: bell-strike — a bell is struck once and rings at its note; then small repeated taps at that same note build a huge amplitude, while off-note taps do nothing]** [CLIP c1 start]
Resonance is the reason a radio can pick one station out of a sky full of them. It's the condition where a system stores and exchanges energy most readily at one particular frequency — so a small, repeated push at exactly that frequency builds a large response, while pushes at any other frequency barely register. Strike a bell and it rings at its natural note; drive it gently at that same note and the ringing grows and grows. Radios are built out of electrical bells.

**[V: definition-card | "f₀ = 1 / (2π√(LC)) — where XL and XC cancel"]**
In an electrical circuit the bell is an inductor and a capacitor. Resonance happens where their reactances become equal and cancel, leaving a purely resistive impedance — at the resonant frequency: one over two pi root L C. [CLIP c1 end]

**[V: energy-slosh — energy swings between the inductor's magnetic field and the capacitor's electric field, side by side with a plucked string trading kinetic and potential energy]**
Mechanically and electrically it's one picture: energy sloshing back and forth between two stores — kinetic and potential in a plucked string, magnetic and electric in an L-C circuit — at a natural frequency all its own.

**[V: tank-curve — the article's figure animated: the LC tank at left; at right, XL rises with frequency while XC falls; they cross, and the response curve erupts into a sharp peak at f₀]** [CLIP c2 start]
Here's the mechanism. An inductor's reactance rises with frequency — two pi f L. A capacitor's falls — one over two pi f C. Sweep the frequency and at exactly one point they're equal in magnitude, and because they carry opposite sign, they cancel. At that crossover the reactive parts of the impedance vanish, voltage and current fall back into phase, and the circuit exchanges energy with its drive as efficiently as it ever will. That's the peak in the response curve — and you place it wherever you like by choosing L and C. [CLIP c2 end]

**[V: series-parallel — two mini-circuits: series LC's impedance dips to minimum at f₀ and passes it; the parallel tank's impedance towers to maximum and blocks it]**
The two canonical arrangements are mirror images. In series, L and C present minimum impedance at resonance — the branch passes the resonant frequency and blocks everything else, so it can short unwanted frequencies to ground or pass a wanted one. In parallel — the "tank" — they present maximum impedance, developing a large voltage at resonance and rejecting that frequency from a series path; that's the frequency-selecting heart of oscillators and filters.

**[V: q-curves — the Q-factor article figure: two peaks on the same f₀, one tall and narrow, one low and broad; a Δf bracket marks the −3 dB width of each | "Q = f₀ / Δf"]** [CLIP c3 start]
How sharp that peak is has its own number: the Q factor — the quality factor — centre frequency divided by the bandwidth measured three decibels down. Equivalently, it's two pi times the energy stored per cycle over the energy dissipated per cycle. A lightly damped resonator loses little each cycle, so it responds only very close to resonance — a tall, narrow peak — and viewed in time, it rings for roughly Q cycles after being struck. Heavy damping gives a broad hump that dies quickly. Loss is the lever: for a series R-L-C circuit, Q is one over R times root L over C — lower the resistance, raise the Q. [CLIP c3 end]

**[V: q-ladder — a component ladder climbs: wire-wound inductor "tens" → good LC circuit "hundreds to low thousands" → coaxial cavity "tens of thousands" → quartz crystal "hundreds of thousands"]**
Across real components the range is enormous: tens for a simple wire-wound inductor, hundreds to low thousands for a good L-C circuit, tens of thousands for a coaxial cavity, hundreds of thousands for a quartz crystal.

**[V: loaded-q — a resonator alone shows its tall unloaded-Q peak; wiring in a source and load broadens it into the loaded-Q curve]**
One trap: a component's own Q — the unloaded Q, set only by its internal losses — is a ceiling. Connect a source and a load and their damping always lowers it; that loaded Q is what you actually see. Filter designers trade this deliberately: tighter coupling buys wider bandwidth and lower insertion loss, looser coupling a narrower, sharper response that costs more loss.

**[V: gt-tie-in — signal path: resonant antenna → preselector filter → high-Q reference oscillator → ADC → GopherTrunk's digital channelizer carving one channel from the wideband stream]**
Around any SDR, resonance is the analog scaffolding. Preselector filters use resonant elements to pass the wanted band and reject strong out-of-band energy before it overloads the front end; a high-Q resonant reference steadies the local oscillator the receiver mixes against; and the antenna itself is a resonant structure — a half-wave dipole resonates where its length matches half the signal's wavelength, which is why antennas are cut for a band. GopherTrunk does its frequency selection downstream in software: the digital channelizer and channel filters play the role a resonant circuit plays in an analog radio. The analog resonances upstream still bound what reaches the converter — a narrow digital channelizer is the software analogue of a high-Q resonator.

**[V: recap | "Resonance" | ① f₀ = 1/(2π√(LC)) — reactances cancel, response peaks | ② Series passes f₀ · parallel tank blocks it | ③ Q = f₀/Δf: high Q = narrow, sharp, long-ringing]**
So: resonance is the frequency where reactances cancel and the response peaks, series and parallel circuits use that peak to pass or trap a frequency, and Q tells you how sharply the bell rings. Both write-ups are linked below.
