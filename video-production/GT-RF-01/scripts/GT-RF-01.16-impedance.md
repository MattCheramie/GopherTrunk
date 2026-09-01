# GT-RF-01.16 — Impedance (Z)
slug: impedance · type: term · treatment: 2.1 term · target: 3:30–4:30 · words: 609

**[V: Title card — "Impedance (Z)"]**

**[V: montage: SMA connector spec, coax jacket printing, SDR antenna port — "50 Ω" stamped on each in turn]** [CLIP c1 starts]
Impedance is why almost every connector, cable, and radio you will ever plug together has the same number printed on it: fifty ohms. It's the total opposition a circuit, a component, or a transmission line presents to an alternating current — and radio standardizes on one shared value so that power flows from source to antenna without bouncing back.

**[V: definition card: "Z = R + jX — resistance plus reactance, in ohms"]** [CLIP c1 ends]
Written out, impedance is a complex number: Z equals R plus j X. The R part is plain resistance. The X part is reactance — the opposition from capacitors and inductors, which store energy and hand it back, shifting the current's phase against the voltage. One number, two personalities: how much the signal is held back, and how much it's twisted in time.

**[V: analogy — a thin rope knotted to a thick rope; a wave pulse hits the knot, part continues on, part reflects back]** [CLIP c2 starts]
Here's the everyday picture. Tie a thin rope to a thick one and flick a wave down it. At the knot, part of the wave keeps going — and part reflects straight back at you. That knot is an impedance mismatch. A radio signal traveling down a cable does exactly the same thing anywhere the impedance changes: some power continues forward, and some bounces home. [CLIP c2 ends]

**[V: coax cross-section; card: "Z0 = √(L/C) — set by geometry and dielectric"; an ohmmeter on disconnected coax reads open]**
A cable like coax has a characteristic impedance — fifty ohms for most RF coax — fixed purely by its geometry and its insulation. It's the ratio of voltage to current for a wave traveling along the line. And here's the trap: it is not a resistance you can measure with an ohmmeter. Put a meter across a disconnected coax and you'll read an open circuit. Fifty ohms is what the cable looks like *to a signal propagating down it*.

**[V: chain: 50 Ω source → 50 Ω line → 50 Ω load, wave flows cleanly; load changes value, a reflected wave animates back and standing waves build on the line]**
Maximum power transfers — and nothing reflects — only when the impedances agree. Match a fifty-ohm source through fifty-ohm cable into a fifty-ohm load and the wave flows through cleanly. Mismatch the far end and part of the wave comes back, piling up against the forward wave into standing waves along the line. Every watt in that reflection is a watt that never did its job.

**[V: number line: 30 Ω "peak power" — 50 Ω — 77 Ω "lowest loss"; a second marker at 75 Ω labeled "TV/video"]** [CLIP c3 starts]
So why fifty, of all numbers? Pure historical compromise. Coaxial cable handles peak power best near thirty ohms and has its lowest loss near seventy-seven. Fifty sits usefully between the two, with convenient physical dimensions — so it became the RF standard. Video and broadcast gear went the other way and standardized on seventy-five ohms, which is why TV coax and radio coax are not the same cable. [CLIP c3 ends]

**[V: antenna feedpoint impedance wandering off 50 Ω across a band; a small matching network of L and C nudges it back to center]**
In the real world, an antenna's feedpoint rarely lands exactly on fifty ohms across a whole band, so matching networks — series and shunt inductors and capacitors, tuners, transformers — cancel the leftover reactance and rotate the resistance back toward the target. On receive, a bad mismatch mostly costs signal rather than hardware: part of what the antenna caught reflects away instead of ever reaching the receiver.

**[V: GopherTrunk web UI, dark theme: spectrum view with the dBFS signal meter; a weak control channel, then the same channel stronger after an antenna fix]**
Every SDR front end GopherTrunk listens through presents a nominal fifty-ohm input, and the amplifier, filters, and mixer behind that connector are all designed around it. GopherTrunk itself works on the digital samples *after* the front end, so it cannot fix a mismatch in software — a control channel that arrives weak because half the signal reflected off a bad match stays weak on the dBFS meter, and no amount of DSP recovers it. A resonant, reasonably matched antenna beats a random length of wire, and the signal-to-noise readout will show you the difference.

**[V: Recap card: "① Z = R + jX — opposition with a phase twist ② mismatch = reflection; 50 Ω keeps power flowing forward ③ a good antenna match buys SNR no software can"]**
So: impedance is R plus j X — opposition with a phase twist; fifty ohms is the shared reference that keeps power moving forward instead of reflecting; and a good match at the antenna is worth more than any setting in the software. Full write-up linked below.

## Clip picks
- **c1** (~30 s): hook — "why everything says 50 Ω" + the Z = R + jX definition card. Opens cold, no context needed.
- **c2** (~25 s): the rope-knot reflection analogy — the most visual, loopable moment.
- **c3** (~30 s): "why fifty, of all numbers?" — the 30/50/77 compromise + the 75 Ω TV-coax aside. Curiosity-gap title: "Why is everything in radio 50 ohms?"

## Vertical plan
Direct center crop. All cards (definition, Z0, number line, recap) built center-safe. The source→line→load chain is the only wide visual — prepare a stacked vertical re-layout (source on top, line middle, load bottom, reflection arrow running vertically). Re-hook first 2 s with burned text "Why is EVERYTHING 50 ohms?" over the connector montage.

## Assets
- Title + recap cards (templates)
- **Core animation:** rebuilt from the article's inline SVG — the source → 50 Ω line → load chain plus the R/jX phasor diagram; extend it with the reflected-wave/standing-wave beat
- Rope-knot reflection animation (2 ropes, one pulse; reusable for the reflection-coefficient entry later)
- Cards: "Z = R + jX" definition · "Z0 = √(L/C)" · 30/50/77 number line
- GopherTrunk screen capture: spectrum + dBFS meter + SNR readout, dark theme (10–20 s)
- VO track; calm/technical music bed

## Checklist deltas
- [ ] Analogy check: rope-knot maps to reflection at an impedance discontinuity — do NOT let it imply thick rope = high impedance as a taught fact (keep the knot generic)
- [ ] Formula cards match article notation exactly: *Z = R + jX*, *Z0 = √(L/C)* — one formula on screen at a time
- [ ] The ohmmeter-trap beat states clearly that Z0 is not DC-measurable (article's emphasis)
- [ ] 30 Ω / 77 Ω / 75 Ω figures verified against article (historical-compromise paragraph)
- [ ] GT tie-in shows real UI only — no implication that GopherTrunk measures or corrects impedance
