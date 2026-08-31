# GT-RF-01.22 — Shannon capacity
slug: shannon-capacity
type: term
target: ~3:50
note: course finale — closing line sweeps the whole GT-RF-01 arc in one sentence.

**[V: title | "Shannon capacity" | GT-RF-01.22]**

**[V: speed-limit — a road-sign-style card stamps "C = B·log₂(1 + SNR)" over a rising data-rate meter that hits a hard ceiling and stops]** [CLIP c1 start]
Shannon capacity is the speed limit of the universe for communication. In nineteen forty-eight, Claude Shannon proved that every channel has a maximum rate at which information can cross it with an arbitrarily small probability of error — and for a band-limited channel with additive Gaussian noise he wrote that maximum in one line: capacity equals bandwidth times the log, base two, of one plus the signal-to-noise ratio, in bits per second. It is not a guideline. It's a hard ceiling: no modulation and no code — none invented, none ever to be invented — can exceed it. [CLIP c1 end]

**[V: noisy-room — two people across a crowded party; two labeled dials appear: "talk faster" (bandwidth) and "talk louder" (SNR)]**
Think of shouting a message across a noisy room. Exactly two things help. You can talk faster — but only up to what the air between you can carry distinctly; that's bandwidth. Or you can speak up over the crowd — that's signal-to-noise ratio. Shannon's formula says those are the *only* two resources, and it prices both: the channel's width sets how many syllables per second arrive separate, and how far your voice stands above the din sets how much meaning each syllable can safely carry.

**[V: capacity-curve — the article's figure animates: the C-versus-SNR curve draws itself, steep at first, flattening right; the formula rides the curve; a C/B = bits-per-second-per-hertz label peels off]**
That's exactly how the formula factors. Bandwidth sets how many independent symbols per second the channel supports — the Nyquist rate. The log of one plus SNR sets how many distinguishable levels each symbol can reliably carry — the bits per symbol. Multiply them and you have bits per second. Divide capacity by bandwidth instead and you get spectral efficiency — bits per second per hertz — the ceiling on what every hertz of spectrum can deliver.

**[V: two-regimes — the curve annotated: high-SNR end, "+3 dB ≈ +1 bit/s/Hz" steps flattening; low-SNR end, a weak signal spreading wide across spectrum; a floor line at Eb/N0 ≈ −1.6 dB]** [CLIP c2 start]
Watch the curve and two behaviors fall out. At high SNR, capacity grows only logarithmically: every doubling of signal power — every three decibels — buys roughly one more bit per second per hertz. Brute-force power hits diminishing returns fast. At low SNR the curve is nearly linear, so you can trade the other way: spread a weak signal across more bandwidth and the bits still get through — the principle behind spread spectrum and deep-space links. And below it all sits a floor: the energy per bit against noise density cannot drop below about minus one point six decibels. Under that, reliable communication is impossible at any bandwidth. Period. [CLIP c2 end]

**[V: closing-gap — a timeline from 1948 to today: the Shannon limit as a fixed line, real codes (FEC → turbo → LDPC) climbing to within a fraction of a decibel of it]**
The astonishing half of the theorem is that the ceiling is *reachable*: Shannon proved codes exist that approach capacity with vanishing error — but the proof was non-constructive; it named no code. Closing that gap took decades of forward-error-correction work, and modern turbo and LDPC codes now operate within a fraction of a decibel of the Shannon limit. Every modulation-and-coding scheme in a modern standard is, in effect, a chosen point on the Shannon curve, trading spectral efficiency against robustness.

**[V: gt-tie-in — GopherTrunk decode chain over the capacity curve: a trunking waveform plotted as a point; an SNR marker slides below it (decode fails) and above it (decode locks); antenna/LNA icons push the marker up]** [CLIP c3 start]
And that is the deepest truth under every decode GopherTrunk attempts. A trunking waveform's modulation order and code rate are a point on this curve, and the SNR at your antenna decides whether your channel can support that point. Below the required SNR, no receiver cleverness recovers the data — the transmitter already sent at a rate the channel cannot sustain at your noise level. GopherTrunk never computes capacity, but Shannon's boundary is why the winning move so often lives on the RF side: a better antenna, a low-noise amplifier, a lower noise figure. Improving SNR is the lever that turns a failing decode into a working one — because it moves you back inside the curve. [CLIP c3 end]

**[V: recap | "Shannon capacity" | ① C = B·log₂(1 + SNR) — a hard ceiling | ② Bandwidth and SNR are the only two currencies | ③ +3 dB of SNR ≈ +1 bit/s/Hz; modern codes sit a fraction of a dB from the limit]**
So: Shannon capacity is the hard ceiling, bandwidth and signal-to-noise ratio are the only two currencies that buy data rate, and three decibels buys about one more bit per hertz. And that's the whole course in one arc — from a ripple in the electromagnetic field, through the knobs, filters, and oscillators that shape it, to the theorem that bounds everything that ripple can ever say — the full write-up, and the complete field guide, linked below.
