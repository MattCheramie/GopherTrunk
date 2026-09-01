# GT-RF-01.01 — Electromagnetic spectrum
slug: electromagnetic-spectrum · type: concept · treatment: 2.2 concept · target: 3:30–4:30 · words: 610

**[V: Title card — "Electromagnetic spectrum"]**

**[V: dark city skyline; dozens of faint signal traces passing through everything — broadcast, aviation, police, satellite — none colliding]** [CLIP c1 starts]
The electromagnetic spectrum is the map every wireless signal in the world lives on. Right now, passing straight through you, are broadcast radio, aviation voice, police dispatch, satellite downlinks, and the light from your screen — thousands of signals sharing the same air without colliding. How? They're all the same physical thing, at different addresses. Sorting out those addresses is the whole idea.

**[V: definition card: "The full range of electromagnetic radiation, ordered by frequency — radio → microwaves → light → X-rays → gamma"]**
The electromagnetic spectrum is the full range of electromagnetic radiation, ordered by frequency: from radio waves through microwaves and visible light to X-rays and gamma rays. One continuous whole — the named regions are human labels of convenience, not physical boundaries. [CLIP c1 ends]

**[V: CORE ANIMATION — the article's band-strip figure builds left to right: Radio (highlighted), Micro, IR, Visible, UV, X-ray, Gamma; the "lower frequency · longer wavelength → higher frequency" arrow draws underneath]**
Here's the surprising part: radio waves, the light you see, and a hospital X-ray are one phenomenon. Each is an oscillating electric and magnetic field propagating through space — a changing electric field creates a magnetic field, and vice versa, and the pair sustains itself as a wave. One law, Maxwell's equations, governs the entire strip. The only thing that changes as you slide along it is how fast the field vibrates.

**[V: on the strip: a slow wave under "Radio", a furious one under "Gamma"; card: "E = hf — photon energy rises with frequency"]** [CLIP c2 starts]
And that one variable changes everything, because the energy each cycle carries rises with frequency. At the top of the strip, gamma rays vibrate so fast their photons are ionising and dangerous. Down at the radio end, the photons are far too weak to break chemical bonds. Same wave, wildly different character — set entirely by frequency. [CLIP c2 ends]

**[V: zoom into the Radio block; card: "Radio: ~3 kHz – 300 GHz — slow enough for circuits"; strip subdivides into VLF…EHF ticks]**
Radio is the low-frequency end — conventionally about three kilohertz to three hundred gigahertz — and it's special for an engineering reason: the oscillations are slow enough that ordinary electronic circuits — oscillators, amplifiers, antennas — can generate, radiate, and detect them directly. Above radio, into infrared and light, no circuit switches fast enough, and we resort to optics instead. The radio portion is itself subdivided by the ITU into named bands — VLF up through EHF — each one a decade of frequency, a factor of ten.

**[V: three-column card: "Propagation · Antenna size · Bandwidth", one icon each; HF wave bouncing off the ionosphere, a shrinking antenna, a widening channel]**
Where a signal sits on this map decides almost everything practical about it. Propagation: HF waves, three to thirty megahertz, refract off the ionosphere and can travel worldwide, while VHF and UHF are largely line-of-sight and microwaves are blocked by terrain and rain. Antenna size: a resonant antenna is a fraction of a wavelength, so lower frequencies demand physically larger antennas. And bandwidth: higher bands have more absolute spectrum to spare — the reason Wi-Fi and 5G keep climbing. One sense of scale: the visible-light octave alone spans more frequency than the entire radio range below it. Radio is a small, crowded strip of an enormous continuum.

**[V: the radio block partitions into labelled slices — broadcasting, aviation, land-mobile — with a stamp: "allocated"]**
Crowded enough that it's governed like territory. The ITU and national regulators — the FCC, Ofcom, and others — allocate slices of the radio spectrum to services, so a given frequency legally belongs to broadcasting, aviation, land-mobile radio, and so on. A frequency isn't just a physical address; it's a legal one.

**[V: GopherTrunk web UI, dark theme: spectrum/waterfall view on a UHF control channel; frequency readout highlighted]**
GopherTrunk's spectrum and waterfall view is a window a few megahertz wide onto exactly this map, bounded by what the hardware can reach — an RTL-SDR covers roughly 24 megahertz to 1.7 gigahertz, a wideband device like the HackRF up to about 6 gigahertz. The trunking systems it decodes — P25, DMR, NXDN, TETRA — live in the VHF and UHF land-mobile bands, typically 136 to 174, 380 to 520, and 700 to 900 megahertz. Where a target sits in the spectrum dictates the antenna, how far it will be heard, and which SDR can receive it at all.

**[V: Recap card: "① one phenomenon, ordered by frequency ② radio = 3 kHz–300 GHz, the circuit-friendly end ③ position on the map sets propagation, antenna, bandwidth"]**
So: one continuous spectrum, one kind of wave, and a signal's address on it decides how it travels, what receives it, and who's allowed to use it. Full write-up linked below.

## Clip picks
- c1: "the map every wireless signal lives on" hook + definition (~30s)
- c2: "same wave, wildly different character" — gamma vs radio photon-energy beat (~25s)
- c3 (optional): "the visible-light octave spans more than all of radio" scale beat (~20s)

## Vertical plan
Band-strip animation is wide by nature — prepare a tall re-layout (strip rotated vertical, radio at bottom). All cards center-safe; re-hook first 2 s per §3.1.

## Assets
- Core animation: rebuild of the article's band-strip SVG (Radio…Gamma, radio highlighted, frequency arrow) with zoom-in to the VLF–EHF subdivision
- City-skyline signal-traces open (stylized motion graphic)
- Photon-energy card (E = hf) — the segment's one formula
- Propagation/antenna/bandwidth three-column card
- Allocation card (radio block partitioned into broadcasting/aviation/land-mobile slices)
- GopherTrunk screen capture: spectrum/waterfall view on a live UHF control channel (dark theme)
- Recap + title cards from templates

## Checklist deltas
- [ ] Problem is stated before the solution (colliding-signals tension precedes the map reveal)
- [ ] Concept's name spoken within the first 20 s (it is the first spoken phrase)
- [ ] Formula card matches the article's notation exactly (E = hf; λ = c/f only if shown)
- [ ] Analogy ("addresses on a map") actually maps — bands are labels, not physical boundaries, and the script says so
