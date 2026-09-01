# GT-RF-01.07 — Frequency bands (HF/VHF/UHF)
slug: frequency-bands · type: term · treatment: 2.1 term · target: 3:30–4:30 · words: 573

**[V: Title card — "Frequency bands (HF/VHF/UHF)"]**

**[V: the band ruler flashes past: VLF · LF · MF · HF · VHF · UHF · SHF, with VHF and UHF glowing]** [CLIP c1 starts]
Frequency bands are the neighborhoods of the radio spectrum. HF, VHF, UHF — the letters you see on every scanner listing and band plan — are the conventional decade-wide divisions of the radio spectrum: HF is 3 to 30 megahertz, VHF is 30 to 300 megahertz, UHF is 300 megahertz to 3 gigahertz, with VLF, LF, and MF below and SHF above. Each decade behaves so differently in propagation, antenna size, and available bandwidth that engineers treat it as a distinct regime. Learn the neighborhoods and you know what to expect before you ever tune in. [CLIP c1 ends]

**[V: analogy — a city map: dense old downtown, mid-rise districts, wide new suburbs — each district tinted like a band]**
A city works the same way. Districts are drawn on a map by convention, but each one genuinely feels different — how you get around, how big the buildings are, what business gets done there. The band boundaries are round numbers, not physics walls, yet cross one and the rules of the road really do change.

**[V: CORE ANIMATION — the article's band-ruler figure builds: the boxes VLF → SHF draw left to right from 3 kHz to 300 GHz; VHF and UHF fill with highlight; the annotations "sky-wave / worldwide" under HF and "line-of-sight scanning" under VHF/UHF fade in]**
Why does each decade feel different? Wavelength. Because wavelength is c over frequency, HF wavelengths are tens of metres, VHF a few metres, UHF centimetres to a metre — and that directly sets antenna size, since an efficient antenna is a meaningful fraction of a wavelength. But the biggest difference is how each band travels. HF refracts off the ionosphere: that sky-wave "skip" can span continents on modest power, with conditions swinging by the sun, the time of day, and the eleven-year solar cycle. VHF and UHF normally punch straight through the ionosphere and travel in near straight lines — range set by the radio horizon and antenna height. That predictability is exactly why land-mobile, aviation, and public-safety systems live there, though occasional tropospheric ducting and sporadic-E can fling VHF far beyond the horizon. SHF and above go strongly line-of-sight, offer huge bandwidth for radar, satellite, and 5G — and start to suffer rain fade.

**[V: trade-off card: an arrow up the ruler labelled "more path loss, less building penetration → but more bandwidth"]** [CLIP c2 starts]
There's a running trade as you climb: higher frequencies carry more path loss for a given distance and antenna, and penetrate buildings less — but they buy the wide bandwidth high-rate digital systems need. Range at the bottom of the ruler, capacity at the top. Every wireless system is parked where its trade balances — and once you see the ruler that way, an unfamiliar frequency starts telling you a story before you've heard a single transmission. [CLIP c2 ends]

**[V: three number cards: "VHF high 136–174 MHz — marine, aviation, older public safety" · "UHF 400–520 MHz — business, public safety, trunked" · "700/800 MHz — modern P25"]**
For a scanner operator, the map gets specific. Regulators — coordinated globally by the ITU, administered nationally by the FCC, Ofcom, and others — publish band plans carving each band into slices for services. Three slices to memorize: VHF high band, 136 to 174 megahertz — marine, nearby aviation voice, older public-safety and business radio. UHF, 400 to 520 — business, public safety, and many trunked systems. And 700/800 megahertz — modern public-safety trunked radio, P25, in North America.

**[V: GopherTrunk dark-theme UI: the band presets list; selecting "UHF 400–520" retunes the spectrum/waterfall view onto a busy control channel]**
This is why GopherTrunk ships band presets: pick the target band and the scanner starts looking where that traffic actually lives. But the band decides more than the tuning — it determines the antenna type and size, the expected path loss, the feedline, whether your SDR's range even covers it, and which protocols are likely to appear. Matching the band is the first hardware decision, ahead of any software configuration.

**[V: Recap card: "① decade-wide divisions: HF 3–30 MHz · VHF 30–300 MHz · UHF 300 MHz–3 GHz ② HF skips worldwide; VHF/UHF are line-of-sight ③ scanner traffic: 136–174, 400–520, 700/800 MHz"]**
So: the bands are decade-wide neighborhoods, each with its own propagation and antenna scale — HF skips off the sky, VHF and UHF run line-of-sight, and the trunked traffic you're after lives in three UHF-and-VHF slices. Full write-up linked below.

## Clip picks
- c1: "neighborhoods of the radio spectrum" hook + the three ranges (~35s)
- c2: the range-vs-capacity trade-off beat (~20s)
- c3 (optional): the three-slices-to-memorize scanner card (~25s)

## Vertical plan
Band ruler is a wide graphic — prepare a tall re-layout (ruler rotated vertical, 3 kHz at bottom, annotations beside each band). Number cards and trade-off card are center-safe.

## Assets
- Core animation: rebuild of the article's band-ruler SVG (VLF→SHF boxes, VHF/UHF highlighted, sky-wave and line-of-sight annotations, 3 kHz / 300 GHz end labels)
- City-districts analogy graphic
- Trade-off arrow card (path loss vs bandwidth)
- Three allocation number cards (136–174 / 400–520 / 700–800)
- GopherTrunk screen capture: band presets UI driving the spectrum/waterfall view, dark theme
- Title + recap cards from templates

## Checklist deltas
- [ ] Analogy actually maps (districts are drawn by convention but behave differently — mirrors "conventional divisions, distinct engineering regimes"; script says boundaries aren't physics walls)
- [ ] Formula card matches the article's notation exactly (wavelength = c / frequency, spoken only — no formula card needed)
- [ ] Band edges verified against the article's infobox (HF 3–30 MHz, VHF 30–300 MHz, UHF 300 MHz–3 GHz, SHF 3–30 GHz)
