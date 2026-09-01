# GT-RF-01.19 — Phase Noise
slug: phase-noise · type: term · treatment: 2.1 term · target: 3:30–4:30 · words: 622

**[V: Title card — "Phase Noise"]**

**[V: an oscillator's output: a perfect spectral spike morphing into a spike with broad sloping skirts]** [CLIP c1 starts]
Phase noise is how an oscillator lies to you: its frequency is right on average, but every individual cycle arrives a tiny bit early or a tiny bit late. That rapid, random wobble in phase smears what should be an infinitely thin spectral line into a mound with sloping "skirts" — and inside a receiver, those skirts are the mechanism that lets a strong signal next door raise the noise on the channel you're actually trying to hear. [CLIP c1 ends]

**[V: two metronomes side by side: one ticking with machine precision, one keeping the same tempo on average while individual ticks land visibly early/late]**
Picture two metronomes set to the same tempo. One ticks with machine precision. The other keeps perfect time *on average* — count its ticks over a minute and it's spot-on — but each individual tick lands slightly early or slightly late, at random. That second metronome has phase noise. And notice it's a different flaw from drift: an oscillator can hold its average frequency beautifully over months and still jitter from cycle to cycle — or be rock-steady tick-to-tick while slowly wandering off frequency. Short-term jitter and long-term stability are separate report cards.

**[V: article's figure animates: the ideal single line at f₀ beside the real carrier with skirts falling off on both sides]**
The spectrum view makes the wobble visible. An oscillator's output is a cosine at its frequency plus a small random phase term that wanders continuously. That wander takes energy which belongs in a single line and spreads it into a continuous pedestal on either side of the carrier — the skirts. They're worst close in and fall away as you move out, eventually flattening into the oscillator's broadband noise.

**[V: measurement card: "−110 dBc/Hz @ 10 kHz offset" with each part highlighted as spoken]**
The standard measure reads like this: minus one hundred ten dBc per hertz, at ten kilohertz offset. Translation: ten kilohertz away from the carrier, each hertz of bandwidth holds a hundred and ten decibels less power than the carrier itself. More negative is better. And one design rule worth keeping: multiply an oscillator up to N times its frequency and its phase noise rises by twenty log N decibels — which is why higher bands are inherently harder on spectral purity.

**[V: mixer diagram: local oscillator with skirts multiplying a strong off-channel carrier; noise energy lands inside the highlighted passband]** [CLIP c2 starts]
Now the receive-side trap: reciprocal mixing. Your receiver's local oscillator multiplies with *everything* coming in. If that oscillator has skirts, a strong off-channel signal — a pager, a broadcast tower — mixes with the skirts, and its energy lands inside your passband as raised noise, even though the strong signal itself is nominally out of band. The symptom reads as "this receiver is deaf near that tower," when the real fault is the oscillator's purity. A cleaner oscillator — or a band-pass filter that keeps the strong neighbour out entirely — is the fix. [CLIP c2 ends]

**[V: a constellation of symbols rotating slightly from symbol to symbol; the error vectors grow]** [CLIP c3 starts]
On transmit, the same wobble corrupts the signal directly: it rotates the constellation a little from symbol to symbol, inflating the error vector magnitude and capping how dense a modulation the link can carry. Phase noise is a ceiling on both ends of a link — and no amount of signal strength raises it. [CLIP c3 ends]

**[V: GopherTrunk web UI, dark theme: waterfall showing a strong carrier, SNR readout healthy, but frame-error indicators stubborn; no issue numbers anywhere on screen]**
For GopherTrunk, phase noise arrives baked into the samples: whatever the SDR's front end contributed before digitization, no software afterwards can remove. What the concept gives you is a diagnosis. A signal that looks strong on the waterfall yet decodes with a stubbornly high error rate near a powerful neighbour is the classic reciprocal-mixing signature — it points at the receiver's oscillator, not at the decoder. GopherTrunk's own issue investigations have met exactly this fingerprint: a capture whose carrier measured clean while the modulation on it was degraded. Carrier-clean but modulation-degraded is the signature of front-end phase noise.

**[V: Recap card: "① cycle-to-cycle jitter → carrier skirts, in dBc/Hz at an offset ② reciprocal mixing: a strong neighbour rides the skirts into your channel ③ strong-but-undecodable near a big signal → suspect the oscillator"]**
So: phase noise is cycle-to-cycle jitter that smears a carrier into skirts, measured in dBc per hertz at an offset; reciprocal mixing lets a strong neighbour ride those skirts into your channel; and a signal that's strong but won't decode near a big transmitter is your cue to suspect the oscillator. Full write-up linked below.

## Clip picks
- **c1** (~30 s): "how an oscillator lies to you" hook + the spike-to-skirts morph.
- **c2** (~35 s): reciprocal mixing — the deaf-near-a-tower beat. Shorts title: "Why your radio goes deaf near a strong transmitter".
- **c3** (~20 s): the rotating constellation / "a ceiling no signal strength raises" — punchy closer clip.

## Vertical plan
Direct center crop. The ideal-vs-real dual spectrum is the widest visual — vertical re-layout stacks the two spectra (ideal above, skirted below). Metronomes, constellation, and all cards center-safe. Re-hook: burned text "Your oscillator is lying to you" over the spike-to-skirts morph.

## Assets
- Title + recap cards (templates)
- **Core animation:** rebuilt from the article's inline SVG — ideal thin line vs real carrier with skirts; extended with the spike→skirts morph for the hook
- Two-metronome jitter animation
- Reciprocal-mixing diagram (LO with skirts × strong off-channel carrier → noise in passband)
- Rotating-constellation/EVM animation (reusable for the EVM entry later)
- Cards: "−110 dBc/Hz @ 10 kHz offset" measurement card · 20·log₁₀(N) rule card (show one at a time)
- GopherTrunk screen capture: waterfall + SNR readout + error indicators (10–20 s, dark theme)
- VO track; calm/technical music bed

## Checklist deltas
- [ ] Analogy check: metronome maps jitter-vs-drift exactly (article's short-term vs long-term stability distinction) — keep the "separate report cards" line
- [ ] Only one formula/measurement card on screen at a time (dBc/Hz card and 20·log N card are separate beats)
- [ ] GopherTrunk investigation reference held to 2 sentences, no issue numbers spoken or on screen (per plan note)
- [ ] dBc/Hz translation matches article wording: noise power in 1 Hz bandwidth at a given offset, relative to total carrier power
- [ ] Reciprocal-mixing beat never claims software mitigation — fix is hardware (cleaner LO / preselect filter), matching the article
