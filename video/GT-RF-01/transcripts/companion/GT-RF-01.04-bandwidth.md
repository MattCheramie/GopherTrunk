# Bandwidth — transcript (GT-RF-01.04-bandwidth.mp4)

_GT-RF-01 segment · 2:42 · 1920×1080 · timestamps match the video_

**[0:00] Title card**

**[0:02] lanes**
Radio spectrum is real estate — and bandwidth is how much of it a signal occupies. A narrowband voice channel takes about twelve and a half kilohertz of road. An FM broadcast station, two hundred kilohertz — sixteen times wider. A Wi-Fi channel, tens of megahertz. Bandwidth decides how fast information can flow, how many signals fit side by side, and how much spectrum a receiver has to swallow. It may be the most consequential number in radio.

**[0:31] definition-card**
Formally: bandwidth is the width, in hertz, of the frequency range a signal occupies.

**[0:37] sidebands**
Why does a signal have width at all? An unmodulated carrier is a single spectral line — zero width. The moment you modulate it, energy spreads into sidebands on either side, and the faster the message changes, the wider they spread. Information takes room. That's not an engineering flaw; it's physics. Regulators then draw lane markings: each service gets a channel of a defined width, with guard bands of empty spectrum between lanes to keep neighbors from sideswiping each other.

**[1:06] shannon**
And here's the law that makes bandwidth precious. Shannon's theorem sets the ceiling on any channel's data rate: capacity equals bandwidth times the log of one plus the signal-to-noise ratio. Look at the shape of that formula. Capacity grows linearly with bandwidth — double the width, double the ceiling. But only logarithmically with signal power — doubling your SNR buys you a mere fraction more. If you want more data, more bandwidth is the straight road, which is exactly why spectrum auctions raise billions.

**[1:37] noise-cut**
Bandwidth cuts the other way too. Noise power grows with every hertz you listen to — open your receiver's filter wider than the signal and all you admit is more hiss. The craft on the receive side is matching the filter to the signal: wide enough to pass it undistorted, narrow enough to shut out everything else. Keep that idea — it becomes the star of the signal-to-noise chapter.

**[1:59] gt-tie-in**
GopherTrunk lives on both sides of this trade at once. Its SDR captures a slice several megahertz wide — wide enough to watch a trunked system's control channel and all its voice channels simultaneously. Then digital down-converters carve out each narrow twelve-and-a-half-kilohertz channel and filter it down to just the bandwidth the decoder needs. Wide capture to see everything; narrow channels to decode cleanly. That split is trunk-tracking.

**[2:27] Recap card**
So: bandwidth is a signal's footprint on the spectrum, the linear lever in Shannon's speed limit, and — on receive — a filter you keep just as narrow as you can. Full write-up linked below.
