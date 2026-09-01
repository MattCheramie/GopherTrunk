# GT-RF-01.15 — Path loss
slug: path-loss · type: term · treatment: 2.1 term · target: 3:30–4:30 · words: 578

**[V: Title card — "Path loss"]**

**[V: a link-budget column of small ±dB entries; then one giant entry slams in at the bottom: "path: −100 dB and more"]** [CLIP c1 starts]
Path loss is the biggest number in radio. It's the attenuation a signal suffers just travelling from transmitter to receiver — dominated by the spreading of energy over distance, plus whatever terrain, buildings, and foliage take on top — and it's the single largest term in almost every link budget. Cables cost you a few dB. Connectors, fractions. The path? It can exceed a hundred dB over a few kilometres.

**[V: CORE ANIMATION — the expanding sphere: a transmitter at centre, a translucent sphere inflating; a fixed small square (the receive antenna) intercepts a shrinking sliver of its surface; counter: "distance ×10 → −20 dB"]** [CLIP c2 starts]
Here's the physical picture, and it's worth owning. In empty space, a transmitter radiates its power over an expanding sphere — like paint on an inflating balloon, the same energy stretched over ever more surface. Your antenna is a fixed-size patch on that sphere, so the power crossing it falls as the square of the distance: the inverse-square law. Double the range, quarter the power. In decibel terms, free-space loss rises twenty dB for every tenfold increase in range — and, for a fixed-size antenna, another twenty dB for every tenfold increase in frequency. The Friis transmission equation wraps this idealised free-space case into one formula. [CLIP c2 ends]

**[V: card: "received power ∝ 1/dⁿ — n ≈ 2 free space · 2.7–4 urban/indoor"; behind it, a skyline grows and the falloff curve steepens]**
But you don't live in empty space. The catch-all model writes received power as falling with distance raised to a path-loss exponent, n. Free space is n equals two. Cluttered urban and indoor settings run two-point-seven to four — and heavy obstruction pushes it higher still. On top of that trend sit two extra effects: shadowing, the slow variation as terrain and buildings block the path, typically modelled as a log-normal spread of a few dB — and multipath fading, the fast fluctuation as reflected copies of the signal arrive out of phase. A link budget reserves a fade margin to survive those dips.

**[V: a hilltop scene: the direct ray, a diffracted ray bending over a rooftop, a ground reflection; an ellipse around the direct line labelled "Fresnel zone"; the antenna mast telescopes up a few metres and the received level jumps]** [CLIP c3]
And not all of the loss is bulk absorption. Diffraction over rooftops and hills, reflection off the ground and buildings, and blockage of the Fresnel zone — the region around the direct line the wave actually needs — all shape how much energy arrives. Which leads to the most cost-effective fact in this whole segment: raising an antenna even a few metres, clearing obstacles and opening that Fresnel zone, can buy more than a large increase in transmit power. Geometry beats wattage.

**[V: card: "path ≈ 100+ dB · cable/preamp ≈ a few dB"; then a coverage map fading smoothly outward into speckle, no hard edge]**
Because the path so often totals a hundred dB or more, it dwarfs the handful of dB you can claw back with better cable or a preamp — the big wins are height, a clear line of sight, and a band that suits the range. Lower frequencies generally carry farther for the same power — less free-space loss and better diffraction around obstacles — one reason VHF public-safety systems reach farther per site than UHF. And path loss explains the shape of coverage: signal strength doesn't fall off a cliff at some range, it decays smoothly with the logarithm of distance, fading gradually into the noise floor rather than stopping sharply.

**[V: GopherTrunk dark-theme waterfall: one strong local system, one faint distant one barely above the grass]**
On your own waterfall, path loss is the whole story of that faint, distant system sitting barely above the noise with hardly enough SNR to decode. GopherTrunk receives whatever the path delivers — so when a known nearby system won't decode, path loss is usually the first suspect: an obstruction, a low antenna — checked before anything in the DSP chain.

**[V: Recap card: "① inverse square: −20 dB per decade of distance (and of frequency) ② real world: n = 2.7–4 + shadowing + fading ③ geometry beats wattage — height and line of sight"]**
So: path loss is the expanding sphere doing its arithmetic — twenty dB per decade in free space, worse in clutter, softened or saved by geometry — and it's why antenna height is the cheapest hundred dB fix that exists. Full write-up linked below.

## Clip picks
- c1: "the biggest number in radio" hook over the link-budget column (~25s)
- c2: the expanding-sphere / inverse-square animation — the core visual (~35s)
- c3: "geometry beats wattage" antenna-height beat (~25s)

## Vertical plan
The expanding sphere is centre-framed — crops as-is. The hilltop diffraction scene is wide — prepare a tall re-layout (stack rays vertically). The article's falloff-curve figure re-frames square. Re-hook first 2 s with the giant −100 dB entry slamming in.

## Assets
- Core animation: expanding-sphere/inverse-square build (the article's falloff-curve SVG is the companion graphic: power-vs-distance curve with its "−20 dB per decade of distance, per decade of frequency" footer)
- Link-budget column hook graphic
- Path-loss-exponent card with skyline (n ≈ 2 / 2.7–4)
- Hilltop diffraction/Fresnel-zone scene with telescoping mast
- Coverage-map fade graphic
- GopherTrunk screen capture: waterfall with strong local + faint distant system, dark theme
- Title + recap cards from templates

## Checklist deltas
- [ ] Analogy actually maps (balloon paint = fixed power over growing surface; no implication the wave weakens "per bounce")
- [ ] Numbers match the article exactly (−20 dB/decade of distance and of frequency; n ≈ 2 vs 2.7–4; 100+ dB over a few km; log-normal shadowing of a few dB)
- [ ] Friis restated in exactly ONE sentence, no other segment referenced (per plan curation note)
