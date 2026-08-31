# GT-RF-01.15 — Harmonics
slug: harmonics
also_slugs: [spurious-emissions]
type: term
target: ~3:50

**[V: title | "Harmonics" | GT-RF-01.15]**

**[V: overtones — a plucked guitar string vibrates; its spectrum shows the note plus a stack of lines at exact multiples; the picture morphs into a transmitter's spectrum doing the same]** [CLIP c1 start]
Harmonics are the radio world's overtones. Pluck a guitar string and you don't hear one pure frequency — you hear the note plus a stack of overtones at exactly two times, three times, four times the fundamental. Radio hardware does the same thing when it misbehaves: push a pure sine wave through any device that treats it nonlinearly, and energy appears at integer multiples of the original — the second harmonic at twice the fundamental, the third at three times, and on up.

**[V: definition-card | "fₙ = n · f₀ (n = 2, 3, 4 …) — born of nonlinearity"]**
That's the whole definition: spectral components at integer multiples of a signal's fundamental frequency. In a transmitter they're a leading source of unwanted spurious emissions — energy that has to be suppressed before the signal ever reaches the antenna. [CLIP c1 end]

**[V: bent-curve — a straight transfer line passes a sine unchanged; the line bends, the output sine distorts, and new spectral lines pop up at 2f₀ and 3f₀]**
Where do they come from? A perfectly linear device passes a sine wave through unchanged. Real components — power amplifiers driven near saturation, diodes, mixers — have a transfer curve that bends. Write that bend as a power series and the squared term generates energy at twice the input frequency, the cubed term at three times; the harder an amplifier is driven into compression, the richer the harmonic content. Two useful facts fall out. Even-order terms create even harmonics and odd-order terms create odd ones — so a symmetric, push-pull stage naturally suppresses the even harmonics. And harmonics always land above the fundamental at exact integer multiples: predictable, and therefore easy to filter — unlike intermodulation products, which land right next to the wanted signal.

**[V: spectrum — the article's figure animated: tall fundamental at f₀, shrinking spikes rising at 2f₀, 3f₀, 4f₀; a low-pass filter curve sweeps in and crushes everything above f₀]** [CLIP c2 start]
That predictability is the defence. A transmitter puts a low-pass or band-pass filter after its final amplifier: a one-hundred-fifty-megahertz VHF transmitter passes one-fifty and knocks the second harmonic at three hundred megahertz down by tens of decibels. The efficient amplifier classes — class C and switching designs — are efficient precisely because they run nonlinearly, so they lean hardest on that output filter; linear classes generate fewer harmonics at the cost of efficiency. The overall effect gets one number: total harmonic distortion — the ratio of combined harmonic power to the fundamental, as a percentage or in decibels. [CLIP c2 end]

**[V: spur-mask — the spurious-emissions article figure: a wanted channel, scattered spur spikes across the band, and a dashed regulatory limit line they must all stay under]**
Harmonics are the best-known members of a bigger family: spurious emissions — any energy a transmitter radiates outside the bandwidth it actually needs. The family includes mixer products at sums and differences of the radio's internal oscillators, parasitic oscillations, and local-oscillator leakage. Regulators cap all of it: the FCC in the United States, ETSI in Europe, both anchored to ITU-R recommendations, publish spurious-emission masks — limits quoted as attenuation below the mean transmit power, for example sixty decibels below carrier or better, or as an absolute power in a reference bandwidth. Land-mobile equipment must pass type approval against that mask before it can be sold.

**[V: rx-side — a strong FM broadcast tower; its harmonic lands inside the monitored band; an external band-pass filter drops in ahead of the SDR dongle and the spur collapses]** [CLIP c3 start]
Harmonics matter on the receive side too. A strong local FM broadcast can present a harmonic that lands squarely in a band you're monitoring, and your receiver's own front end and mixer generate harmonics of the local oscillator that create image and spurious responses of their own. Cheap SDR front ends have limited selectivity, so a strong out-of-band signal — or its harmonic — can overload the tuner and raise the effective noise floor, degrading decode of the control channel you actually want. The standard cure is an external band-pass filter ahead of the SDR. [CLIP c3 end]

**[V: gt-tie-in — GopherTrunk waterfall: a suspicious spike; a retune animation — real signals slide, the receiver spur stays put; a second spike annotated "2× the local FM broadcast"]**
GopherTrunk is a receive-only decoder, so it produces no harmonics of its own — but its waterfall shows you everyone else's, plus your receiver's. Two diagnostic habits pay off. A spike that stays put as you retune, or appears at a predictable offset, is usually a receiver-generated spur, not a real signal. And a mystery carrier at exactly twice or three times a known strong station is that station's harmonic — not a new system to chase. Knowing where harmonics fall keeps you from decoding ghosts.

**[V: recap | "Harmonics" | ① Integer multiples of f₀, created by nonlinearity | ② Predictable → filterable: low-pass after the final amplifier | ③ Spurious-emission masks (FCC/ETSI/ITU-R) cap them by law]**
So: harmonics live at integer multiples of the fundamental, nonlinearity creates them and filters kill them, and the spurious-emission mask is the rulebook that keeps everyone's spectrum clean. Both write-ups are linked below.
