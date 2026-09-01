# GT-RF-01.10 — Bandwidth
slug: bandwidth · type: term · treatment: 2.1 term · target: 3:30–4:30 · words: 574

**[V: Title card — "Bandwidth"]**

**[V: three spectrum bumps side by side, to scale: a sliver labelled "voice ~12.5 kHz", a wider one "FM broadcast ~200 kHz", a huge one "Wi-Fi — tens of MHz"]** [CLIP c1 starts]
Bandwidth is how much room a signal takes up — the width, in hertz, of the frequency range it occupies, or that a receiver captures. A narrowband voice channel is about twelve and a half kilohertz wide. An FM broadcast station, around two hundred kilohertz. A Wi-Fi channel, tens of megahertz. And that width is one of the most consequential numbers in radio: it caps how fast information can flow, it decides how much spectrum a signal consumes, and on the receive side it sets how much of the spectrum an SDR must digitise at once. [CLIP c1 ends]

**[V: analogy — a street of plots along a "frequency" road; each signal owns a plot; wider plots hold bigger buildings, and the street runs out]**
Think of the spectrum as a long street of land, with frequency as the address. Every signal occupies a plot, and bandwidth is the width of its frontage. A wider plot lets you build more — carry more data — but the street has a fixed length, and every hertz you take is a hertz nobody else gets. That's why regulators parcel it out: each service is allocated a channel of a given width, emissions must stay inside it, and a guard band of empty spectrum sits between neighbours to prevent interference.

**[V: CORE ANIMATION — the article's spectrum-bump figure: a flat baseline, a bump rises, and the double-headed "bandwidth" arrow spans it; a steady carrier beside it stays a zero-width line, then spreads as modulation speeds up]**
Where does the width come from? Information cannot be sent instantaneously. A steady, unchanging carrier is a single spectral line of zero width; modulating it to carry a message spreads its energy into sidebands, and the faster the modulation, the wider those sidebands get. The exact width a signal genuinely uses is its occupied bandwidth — usually defined as the span containing ninety-nine percent of the transmitted power.

**[V: formula card: "C = B · log₂(1 + SNR)" — B highlighted, then a second highlight sweeps to SNR]** [CLIP c2 starts]
Now the hard link to data rate. The Shannon–Hartley theorem gives a channel's theoretical ceiling: capacity equals bandwidth times the log of one plus the signal-to-noise ratio. Look at the shape of that formula. Capacity scales linearly with bandwidth, but only logarithmically with SNR — so more bandwidth is the most direct way to send data faster, and crowded spectrum is such a valuable resource. But bandwidth cuts the other way too: noise power is proportional to bandwidth, so a wider receiver passband admits more noise. Opening the filter beyond what the signal needs only lowers your SNR. [CLIP c2 ends]

**[V: card stack: "DMR — 2 TDMA slots in 12.5 kHz · P25 Phase 1 — 12.5 kHz · NXDN — 6.25 kHz mode"]**
In practice, channel width sets the standard. Land-mobile digital voice squeezes into narrow channels — DMR packs two time-division slots into twelve and a half kilohertz, P25 Phase 1 uses twelve and a half, NXDN offers a six-and-a-quarter kilohertz mode — and that width drives the choice of modulation and symbol rate. The receiver's rule of thumb: match the channel filter to the occupied bandwidth — wide enough to pass the signal undistorted, narrow enough to reject neighbours and noise.

**[V: GopherTrunk dark-theme spectrum: a several-MHz capture; highlight boxes carve out individual 12.5 kHz channels, each funnelling down to "48 kHz"]**
GopherTrunk lives on both ends of this. An SDR's capture bandwidth is roughly its sample rate — a complex IQ stream sampled at some rate represents a band that wide. So GopherTrunk captures a wide slice, several megahertz, then uses filtering, decimation, and a digital down-converter to carve out each narrow channel at its own centre frequency, reducing it to the per-channel rate the decoder needs — forty-eight kilohertz for the 4800-baud C4FM family. The wide capture is what lets one receiver follow a control channel and its voice grants at once; the narrow per-channel bandwidth is what keeps noise out of each individual decode.

**[V: Recap card: "① width in Hz — more width = more data, more spectrum ② C = B·log₂(1+SNR): linear in B ③ filter no wider than the signal"]**
So: bandwidth is the width a signal occupies — it buys data rate linearly, it costs spectrum and admits noise, and the receiver's job is to capture wide but filter narrow. Full write-up linked below.

## Clip picks
- c1: the three-bumps-to-scale hook (~30s)
- c2: the Shannon formula shape — "linear in B, logarithmic in SNR" (~30s)
- c3 (optional): the capture-wide/filter-narrow GopherTrunk carve-out (~25s)

## Vertical plan
The bandwidth-arrow bump figure is wide — prepare a tall re-layout (bump plus arrow re-framed square, axis label moved under). Cards and formula are centre-safe. Re-hook first 2 s with the three-bumps comparison.

## Assets
- Core animation: rebuild of the article's spectrum-bump-with-arrow SVG, extended with the zero-width-carrier-spreads beat
- Three-bumps-to-scale comparison graphic (12.5 kHz / 200 kHz / tens of MHz)
- Street-of-plots analogy graphic
- Formula card (C = B·log₂(1 + SNR)) with term highlights — one formula on screen at a time
- Channel-widths card stack (DMR/P25/NXDN)
- GopherTrunk screen capture: wide spectrum with per-channel DDC carve-out annotation, dark theme
- Title + recap cards from templates

## Checklist deltas
- [ ] Analogy actually maps (plot width = spectrum consumed; no implication that frequency position matters to capacity)
- [ ] Formula card matches the article's notation exactly (C = B · log₂(1 + SNR))
- [ ] Channel-width numbers verified against the article (12.5 kHz DMR two-slot, 12.5 kHz P25 Phase 1, 6.25 kHz NXDN mode, 48 kHz C4FM-family channel rate)
