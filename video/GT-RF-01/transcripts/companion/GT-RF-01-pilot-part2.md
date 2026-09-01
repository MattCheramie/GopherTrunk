# Pilot part 2 transcript (Modulation → Bandwidth)

_6:31 · 1920×1080 · chapter timestamps match the video_


## [0:00] Transition

**[0:00] map-card**
You can now find any signal's address on the spectrum. But a wave sitting at an address, perfectly steady, says nothing at all. Making it talk is next: modulation.


## [0:11] Modulation

**[0:11] Title card**

**[0:13] bare-carrier**
Modulation is the answer to an embarrassing question: a perfect radio wave says absolutely nothing. A pure, steady carrier is just a hum in the spectrum — endlessly repeating, perfectly predictable, carrying zero information. To make a wave say something, you have to change it. Modulation is the art of changing a carrier wave in step with a message — and every radio scheme in history is a variation on that one move.

**[0:39] three-knobs**
Remember the three properties from the last chapter's write-up — a sine wave has an amplitude, a frequency, and a phase, and those are the only things a transmitter can touch. So there are only three families of modulation. Everything from nineteen-twenties broadcasting to your phone is built from these three knobs.

**[0:58] am-fm**
Turn the amplitude knob and you get AM: the message is written into the carrier's height — its envelope literally traces the audio. Simple, but noise also lives in amplitude, which is why AM crackles in a thunderstorm. Turn the frequency knob and you get FM: the message speeds up and slows down the carrier's wiggle while the height stays constant — so amplitude noise mostly wipes right off. That constant envelope is why FM sounds cleaner than AM, and the third knob, phase, nudges the carrier's timing — closely related to FM, and the foundation of almost everything digital.

**[1:36] symbols**
Analog modulation varies the carrier continuously. Digital modulation makes one crucial change: it switches the carrier among a finite menu of states, called symbols, each standing for one or more bits. Shift between discrete frequencies and it's FSK — frequency-shift keying. Discrete phases: PSK. Combine amplitude and phase levels: QAM. Draw the menu as dots on a plane — amplitude out from the center, phase around it — and you get the constellation diagram, the map every digital receiver uses to decide which symbol just arrived.

**[2:12] trade**
And here is the trade that shapes all of digital radio. More dots on the map means more bits per symbol — faster data in the same bandwidth. But more dots also means the dots sit closer together, so it takes less noise to smudge one into its neighbor. Fast-but-fragile, or slow-but-tough. Wi-Fi at close range runs hundreds of points; a police radio fighting fades across a whole city runs just four.

**[2:38] gt-tie-in**
That's exactly what GopherTrunk spends its life demodulating. The land-mobile systems it decodes chose rugged, low-order schemes on purpose: four-level FSK — four frequency steps, two bits per symbol — for DMR and NXDN, and a rotating four-phase scheme called pi-over-four DQPSK for TETRA and P25's simulcast flavor. Once the signal is digitized, demodulation is just measuring amplitude, frequency, and phase — the same three knobs — and mapping them back to bits.

**[3:10] Recap card**
So: modulation turns a mute carrier into a messenger, there are exactly three knobs to turn, and digital radio is a menu of symbol states traded off between speed and toughness. Full write-up linked below.


## [3:27] Transition

**[3:27] map-card**
Here's the catch modulation doesn't advertise: the moment a carrier starts talking, it spreads out and takes up room. How much room — and why that room is so expensive — is bandwidth.


## [3:39] Bandwidth

**[3:39] Title card**

**[3:41] lanes**
Radio spectrum is real estate — and bandwidth is how much of it a signal occupies. A narrowband voice channel takes about twelve and a half kilohertz of road. An FM broadcast station, two hundred kilohertz — sixteen times wider. A Wi-Fi channel, tens of megahertz. Bandwidth decides how fast information can flow, how many signals fit side by side, and how much spectrum a receiver has to swallow. It may be the most consequential number in radio.

**[4:10] definition-card**
Formally: bandwidth is the width, in hertz, of the frequency range a signal occupies.

**[4:16] sidebands**
Why does a signal have width at all? An unmodulated carrier is a single spectral line — zero width. The moment you modulate it, energy spreads into sidebands on either side, and the faster the message changes, the wider they spread. Information takes room. That's not an engineering flaw; it's physics. Regulators then draw lane markings: each service gets a channel of a defined width, with guard bands of empty spectrum between lanes to keep neighbors from sideswiping each other.

**[4:45] shannon**
And here's the law that makes bandwidth precious. Shannon's theorem sets the ceiling on any channel's data rate: capacity equals bandwidth times the log of one plus the signal-to-noise ratio. Look at the shape of that formula. Capacity grows linearly with bandwidth — double the width, double the ceiling. But only logarithmically with signal power — doubling your SNR buys you a mere fraction more. If you want more data, more bandwidth is the straight road, which is exactly why spectrum auctions raise billions.

**[5:16] noise-cut**
Bandwidth cuts the other way too. Noise power grows with every hertz you listen to — open your receiver's filter wider than the signal and all you admit is more hiss. The craft on the receive side is matching the filter to the signal: wide enough to pass it undistorted, narrow enough to shut out everything else. Keep that idea — it becomes the star of the signal-to-noise chapter.

**[5:38] gt-tie-in**
GopherTrunk lives on both sides of this trade at once. Its SDR captures a slice several megahertz wide — wide enough to watch a trunked system's control channel and all its voice channels simultaneously. Then digital down-converters carve out each narrow twelve-and-a-half-kilohertz channel and filter it down to just the bandwidth the decoder needs. Wide capture to see everything; narrow channels to decode cleanly. That split is trunk-tracking.

**[6:06] Recap card**
So: bandwidth is a signal's footprint on the spectrum, the linear lever in Shannon's speed limit, and — on receive — a filter you keep just as narrow as you can. Full write-up linked below.


## [6:21] Transition

**[6:21] map-card**
We just compared powers that differ by billions. Radio has its own arithmetic for exactly that — small numbers you can add in your head. Meet the decibel.
