# GT-RF-01.04 — Bandwidth
slug: bandwidth
type: term
target: ~3:55

**[V: title | "Bandwidth" | GT-RF-01.04]**

**[V: lanes — a spectrum axis as a highway; signals as vehicles of different widths: a narrow scooter (12.5 kHz), a car (200 kHz), a truck (20 MHz)]** [CLIP c1 start]
Radio spectrum is real estate — and bandwidth is how much of it a signal occupies. A narrowband voice channel takes about twelve and a half kilohertz of road. An FM broadcast station, two hundred kilohertz — sixteen times wider. A Wi-Fi channel, tens of *megahertz*. Bandwidth decides how fast information can flow, how many signals fit side by side, and how much spectrum a receiver has to swallow. It may be the most consequential number in radio.

**[V: definition-card | the article's figure: a spectrum bump with the "bandwidth" span arrows]**
Formally: bandwidth is the width, in hertz, of the frequency range a signal occupies. [CLIP c1 end]

**[V: sidebands — a zero-width carrier spike; modulation switches on and the spike spreads into a bump; faster message = wider bump]**
Why does a signal have width at all? An unmodulated carrier is a single spectral line — zero width. The moment you modulate it, energy spreads into sidebands on either side, and the *faster* the message changes, the *wider* they spread. Information takes room. That's not an engineering flaw; it's physics. Regulators then draw lane markings: each service gets a channel of a defined width, with guard bands of empty spectrum between lanes to keep neighbors from sideswiping each other.

**[V: shannon — card: C = B · log₂(1 + SNR); B highlighted, sliding B doubles C; sliding SNR barely moves it]** [CLIP c2 start]
And here's the law that makes bandwidth precious. Shannon's theorem sets the ceiling on any channel's data rate: capacity equals bandwidth times the log of one plus the signal-to-noise ratio. Look at the shape of that formula. Capacity grows *linearly* with bandwidth — double the width, double the ceiling. But only *logarithmically* with signal power — doubling your SNR buys you a mere fraction more. If you want more data, more bandwidth is the straight road, which is exactly why spectrum auctions raise billions. [CLIP c2 end]

**[V: noise-cut — a wide receiver filter admitting a wide slab of noise; the filter narrows to hug the signal; noise slab shrinks]**
Bandwidth cuts the other way too. Noise power grows with every hertz you listen to — open your receiver's filter wider than the signal and all you admit is more hiss. The craft on the receive side is matching the filter to the signal: wide enough to pass it undistorted, narrow enough to shut out everything else. Keep that idea — it becomes the star of the signal-to-noise chapter.

**[V: gt-tie-in — GopherTrunk wideband spectrum with a whole trunked system in view; DDC boxes carve out narrow channels]**
GopherTrunk lives on both sides of this trade at once. Its SDR captures a slice several megahertz wide — wide enough to watch a trunked system's control channel and all its voice channels simultaneously. Then digital down-converters carve out each narrow twelve-and-a-half-kilohertz channel and filter it down to just the bandwidth the decoder needs. Wide capture to see everything; narrow channels to decode cleanly. That split *is* trunk-tracking.

**[V: recap | "Bandwidth" | ① Width in Hz — a signal's spectrum footprint | ② Capacity: C = B·log₂(1+SNR), linear in B | ③ Wider filter = more noise; match it to the signal]**
So: bandwidth is a signal's footprint on the spectrum, the linear lever in Shannon's speed limit, and — on receive — a filter you keep just as narrow as you can. Full write-up linked below.
