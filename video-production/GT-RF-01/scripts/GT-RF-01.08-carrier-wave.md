# GT-RF-01.08 — Carrier wave
slug: carrier-wave · type: term · treatment: 2.1 term · target: 3:30–4:30 · words: 552

**[V: Title card — "Carrier wave"]**

**[V: a perfect, endless sine wave scrolling across the frame; a label fades in: "content: nothing"]** [CLIP c1 starts]
A carrier wave is a radio signal that carries, by itself, absolutely nothing. It's a steady radio-frequency signal at a single frequency — the same cycle, repeated forever, with no voice and no data in it. And yet it's the foundation of every transmission you'll ever tune. The carrier is the vehicle; the message rides on it.

**[V: definition card: "carrier = steady RF signal · useful only when modulated (amplitude · frequency · phase)"]**
Here's the definition. A carrier conveys no information on its own — it becomes useful only when modulation varies one of its properties, its amplitude, its frequency, or its phase, in step with a message. When you say a station is "on" some frequency, you're naming its carrier. Tuning a receiver means picking one carrier out of the crowd — which is why "the frequency of a station" and "its carrier" mean nearly the same thing. [CLIP c1 ends]

**[V: analogy — an empty delivery truck circling a route; then cargo loads in and the truck matters]**
Why bother with a wave that says nothing? Think of an empty delivery truck: it moves no cargo, but it's the thing that can make the trip. Your voice can't make the trip itself — an antenna is only efficient at a size comparable to the wavelength, and baseband audio, a few kilohertz, has wavelengths of tens of kilometres. Impractical to radiate. Shift the message up onto a high-frequency carrier, and it radiates from a reasonable antenna. And there's a second win: carriers let many stations share the spectrum, each on its own carrier frequency with a guard band between them, and a receiver picks one out just by tuning.

**[V: CORE ANIMATION — the article's two-trace figure: the flat, even "unmodulated carrier" on top; below it, the same carrier with its shape varying — "modulated (carries information)". Then cut to the frequency domain: a single thin spike blooming into sidebands either side]** [CLIP c2 starts]
Now watch what the message does to it. A pure, unmodulated carrier is a single sinusoid, so in the frequency domain it's one narrow spike — all of its energy at exactly one frequency. The moment it's modulated, that energy spreads: the message frequencies mix with the carrier and produce sidebands — mirror-image bands of energy above and below the carrier frequency. The span those sidebands occupy is essentially the signal's bandwidth, and it grows with the rate and depth of the modulation. A voice AM signal spreads a few kilohertz either side of its carrier; a wideband FM broadcast, hundreds of kilohertz. [CLIP c2 ends]

**[V: waterfall view: a thin bright vertical line; a "key-up" moment blooms it into a wide modulated band]**
You can see all of this on a waterfall. An idle transmitter is a thin bright line — a bare carrier. Key it up, and the line blooms into a modulated band. That presence-of-carrier test is also the simplest squelch there is: carrier squelch opens the audio whenever a strong enough carrier shows up, regardless of what's on it.

**[V: GopherTrunk dark-theme spectrum view: bright spike dead centre, annotated "DC spike — artifact, not a station"]**
In GopherTrunk you'll meet one famous impostor. After the SDR mixes your chosen slice of spectrum down toward baseband, the tuned carrier lands near zero frequency — and a residual carrier or DC offset sitting at exactly zero paints the familiar bright "DC spike" in the centre of the display. It's an artifact, not a real station. The real carriers matter more: GopherTrunk locks onto each channel's carrier to establish a phase and frequency reference, and the digital voice systems it decodes keep the carrier present and modulate its phase — so accurate carrier tracking is the foundation of the whole decode chain.

**[V: Recap card: "① carrier = steady signal, zero info until modulated ② modulation → sidebands = the signal's bandwidth ③ centre-screen DC spike = artifact, not a station"]**
So: a carrier is a steady wave that exists to be changed — modulation spreads it into sidebands that carry the message, tuning means choosing a carrier, and the bright spike in the middle of your SDR's screen isn't one. Full write-up linked below.

## Clip picks
- c1: "a signal that carries absolutely nothing" hook + vehicle definition (~30s)
- c2: spike-to-sidebands bloom — the core frequency-domain moment (~30s)
- c3 (optional): the DC-spike impostor beat (~20s)

## Vertical plan
Direct center crop. The two-trace carrier figure already stacks vertically (unmodulated over modulated); the spike-to-sidebands animation is symmetric about centre and crops cleanly. Re-hook first 2 s with the "content: nothing" scrolling sine.

## Assets
- Core animation: rebuild of the article's unmodulated-vs-modulated two-trace SVG, extended with a time→frequency cut (single spike blooming into sidebands)
- Definition card (carrier · modulated via amplitude/frequency/phase)
- Delivery-truck analogy graphic
- Waterfall key-up B-roll (thin line blooming into a band — capture from GopherTrunk's waterfall)
- GopherTrunk screen capture: spectrum view with DC spike annotated, dark theme
- Title + recap cards from templates

## Checklist deltas
- [ ] Analogy actually maps (empty truck = radiatable vehicle; no implication the carrier "contains" anything)
- [ ] Sideband/bandwidth wording matches the article (sidebands' span ≈ bandwidth; AM few kHz, wideband FM hundreds of kHz)
- [ ] DC spike labelled as artifact exactly as the article does — never as a signal
