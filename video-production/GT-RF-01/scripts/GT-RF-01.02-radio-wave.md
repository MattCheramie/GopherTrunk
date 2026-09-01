# GT-RF-01.02 — Radio wave
slug: radio-wave · type: term · treatment: 2.1 term · target: 3:30–4:30 · words: 600

**[V: Title card — "Radio wave"]**

**[V: a voice waveform dissolving into a sine wave that races away from an antenna into black space]** [CLIP c1 starts]
A radio wave is how a message crosses empty space with nothing to carry it. No wire, no air required — a police call, a broadcast, a satellite picture all ride the same physical vehicle. Formally: a radio wave is electromagnetic radiation in the radio-frequency range — about three kilohertz to three hundred gigahertz — travelling at the speed of light, and it carries information when its amplitude, frequency, or phase is varied. That varying has a name — modulation — and it's the whole trick of radio. [CLIP c1 ends]

**[V: analogy — a stone dropped in a pond, ripples spreading; then the pond fades away and the ripples keep going]**
The everyday anchor is a ripple on a pond: a disturbance spreading outward in rings from where the energy went in. But here's where the analogy deliberately breaks — a pond ripple needs water. A radio wave needs no medium at all. It carries its own support with it, which is exactly why radio crosses the vacuum of space.

**[V: CORE ANIMATION — the article's sine-wave figure builds: centre line, then the wave, then "wavelength (λ)" bracket between crests, then the "amplitude" height marker]**
How does it carry itself? A transmitter drives an alternating current into an antenna. Those accelerating charges launch a self-propagating disturbance: an oscillating electric field regenerates a magnetic field at right angles to it, and that magnetic field in turn regenerates the electric field — so the pair detaches from the antenna and radiates outward at the speed of light. Each field continually rebuilds the other. That self-sustaining handshake is the wave.

**[V: three labels light up on the wave in turn: AMPLITUDE (height), FREQUENCY (cycles/sec counter), PHASE (position marker sliding along one cycle)]** [CLIP c2 starts]
And a simple radio wave is fully described by just three properties — the only things a transmitter can manipulate. Amplitude: its strength. Frequency: its cycles per second. Phase: its position within the cycle. Every modulation scheme ever invented is a way of wiggling one or more of those three in step with the information being sent. Three knobs. That's the entire vocabulary. [CLIP c2 ends]

**[V: number cards: "speed: ~299,792,458 m/s" · "λ = c / f" · "received: a few microvolts"]**
The numbers worth keeping: in a vacuum the wave moves at about 299,792,458 metres per second — the speed of light — and its wavelength follows from its frequency by lambda equals c over f. And at the far end? A distant antenna, immersed in the passing field, develops a tiny induced current — often just a few microvolts — which the receiver amplifies, filters, and decodes. Radio is the art of doing something useful with almost nothing. One practical footnote rides along: the wave's polarization — the orientation of that electric field — must usually match between the transmit and receive antennas for good reception, which is why antenna orientation isn't cosmetic.

**[V: split card: low-frequency wave bending around a hill vs high-frequency wave bouncing off buildings; a noise haze rising from below]**
Why care about the physics? Because the journey is hostile. Even in free space the wave's power density falls with the square of distance. Lower-frequency waves diffract around hills and buildings; higher frequencies travel more like light and get blocked or reflected, producing multipath. And the wave always arrives buried in noise, thermal and man-made — the ratio of wanted signal to that floor is the signal-to-noise ratio, and it bounds how reliably the message can be decoded at all. Every receiver ever built is fighting that same three-front war: spreading, obstacles, noise.

**[V: GopherTrunk dark-theme UI: spectrum/waterfall view with a live carrier, then the signal meter in dBFS beside it]**
By the time GopherTrunk sees a radio wave, its front end has mixed it down and the converter has turned it into IQ samples — complex numbers capturing the amplitude and phase of the wave at each instant. Watch the spectrum and waterfall view and the signal meter in dBFS: that's the physical wave, become numbers. Every property you just met survives the conversion, and every one must be tracked to recover the message.

**[V: Recap card: "① EM radiation, 3 kHz–300 GHz, needs no medium ② two fields regenerating each other at light speed ③ amplitude, frequency, phase — the only three knobs"]**
So: a radio wave is two fields carrying each other through nothing, at light speed, with exactly three properties a transmitter can vary — and everything else in radio is built on those three. Full write-up linked below.

## Clip picks
- c1: "crosses empty space with nothing to carry it" hook + definition (~35s)
- c2: "three knobs — the entire vocabulary" beat (~25s)
- c3 (optional): "a few microvolts — something useful from almost nothing" (~20s)

## Vertical plan
Direct center crop — sine animation and cards built center-safe. Re-hook first 2 s with the c1 line burned over the antenna-launch visual.

## Assets
- Core animation: rebuild of the article's sine-wave SVG (wavelength bracket + amplitude marker), extended with the three-property label pass
- Pond-ripple analogy graphic (ripples persist as pond fades)
- Field-handshake animation (E and B fields regenerating at right angles, detaching from the antenna)
- Number cards (speed of light, λ = c/f, microvolts)
- GopherTrunk screen capture: spectrum/waterfall view + signal meter (dBFS), dark theme
- Title + recap cards from templates

## Checklist deltas
- [ ] Analogy actually maps (pond ripple used, with the no-medium break stated explicitly so it can't teach the wrong model)
- [ ] Formula card matches the article's notation exactly (λ = c / f; one formula on screen)
