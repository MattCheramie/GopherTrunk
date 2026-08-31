# GT-RF-01.08 — Attenuation
slug: attenuation
also_slugs: [path-loss]
type: term
target: ~3:50
note: double-header — attenuation (the general mechanism) plus path loss (the distance-dominated total); both article figures animated.

**[V: title | "Attenuation" | GT-RF-01.08]**

**[V: fading-wave — the attenuation article's SVG animated: a sine wave shrinking steadily left to right, "transmitter" → "weaker at receiver"; the caption "−6 dB = ×¼ power · −3 dB = ×½" stamps on]** [CLIP c1 start]
Attenuation is the toll every signal pays. From the moment a wave leaves the transmitter, everything it touches — air, cable, connectors, walls, rain — takes a cut, and what reaches your receiver is whatever's left. Attenuation is that reduction in signal strength, and it's expressed in decibels because losses in decibels simply add down the chain: a loss of six decibels means only a quarter of the power survives; three decibels, half.

**[V: definition-card | "Attenuation = spreading + absorption, in dB · Path loss = the TX→RX total"]**
Two mechanisms cover all of it. Spreading: a fixed amount of power diluted over a larger and larger area. Absorption: the medium converting radio energy into heat. And the attenuation a signal racks up over the whole journey from transmitter to receiver has its own name — path loss. [CLIP c1 end]

**[V: toll-booths — the signal travels a road through toll booths: coax (loss rises with length & frequency), connectors (fractions of a dB each), walls & foliage, rain clouds above 10 GHz]**
Walk a receive path and count the tolls. Coax loss rises with both frequency and length — a cheap run of RG-58 can lose several decibels per ten meters at four hundred megahertz — and loss ahead of the first amplifier is doubly harmful, because it degrades the noise figure too. Every connector, filter, splitter, and switch adds its fraction of a decibel, and they accumulate. Walls, foliage, and terrain absorb and scatter energy. Above roughly ten gigahertz, oxygen and water vapor absorb RF outright, and rain adds "rain fade." The recurring theme: for most of these mechanisms, higher frequencies attenuate more — that single fact shapes band choice, cable choice, and site planning.

**[V: sphere — a transmitter at the centre of an expanding sphere; a fixed receive window catches an ever-smaller share; morphs into the path-loss article's SVG: received power decaying smoothly with distance]** [CLIP c2 start]
The biggest toll of all is distance. In empty space a transmitter radiates over an expanding sphere, so the power crossing a fixed receive aperture falls as the square of distance — the inverse-square law. In decibel terms, free-space loss rises twenty decibels for every tenfold increase in range and, for a fixed-size antenna, another twenty for every tenfold increase in frequency. Real environments lose far more: the catch-all model raises distance to a path-loss exponent — about two in free space, but two point seven to four in cluttered urban and indoor settings, and higher still through heavy obstruction. [CLIP c2 end]

**[V: layers — the smooth distance curve, with slow "shadowing" undulations laid over it, then fast "multipath fading" flicker on top; a "fade margin" band is reserved above threshold]**
On top of that distance trend sit two more effects. Shadowing: the slow variation as terrain and buildings block the path. Multipath fading: the fast fluctuation as reflected copies of the signal arrive out of phase. A link budget reserves a fade margin to survive those dips. Add it all up, and path loss can exceed one hundred decibels over a few kilometers — the single largest term in almost every link budget, and the reason budgets are done in decibels at all.

**[V: levers — two levers side by side: "better coax / preamp: a few dB" vs "antenna height & clear path: the big win"; a mast rises and the Fresnel zone around the direct line clears]**
And here's the practical punchline: because path loss dwarfs everything else, the handful of decibels from better cable or a preamp is small next to what geometry gives you. Raising an antenna even a few meters — clearing obstacles, opening the Fresnel zone around the direct line — can buy more than a large increase in transmit power. Lower frequencies also carry farther for the same power, which is one reason VHF public-safety systems reach farther per site than UHF. One caveat: attenuation isn't always the enemy. A deliberate attenuator protects a receiver from strong signals that would overload the front end, trading a few decibels of wanted signal for headroom.

**[V: gt-tie-in — GopherTrunk dashboard with a channel fading toward the noise floor; a checklist overlays: antenna height? line of sight? coax run and connectors?]**
For GopherTrunk, this is the first chapter of troubleshooting. The software sees only what survives the path and the feedline — attenuation upstream of the SDR is loss it cannot recover. When a known nearby system won't decode, path loss is usually the first suspect — an obstruction, a low antenna — ahead of anything in the DSP chain. Keeping coax runs short and connectors clean preserves the signal-to-noise ratio every decoder depends on.

**[V: recap | "Attenuation" | ① Spreading + absorption, tallied in dB | ② Free space: −20 dB per decade of distance & of frequency | ③ Geometry beats gain: height and a clear path]**
So: attenuation is the toll — spreading plus absorption, added up in decibels; path loss is the distance-dominated total, easily a hundred decibels or more; and the cheapest decibels come back from antenna height and a clear path, not from the software. Full write-ups on both are linked below.
