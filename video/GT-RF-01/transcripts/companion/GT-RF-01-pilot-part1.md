# Pilot part 1 transcript (cold open → Frequency)

_7:53 · 1920×1080 · chapter timestamps match the video_


## [0:00] Cold open

**[0:01] cold-montage**
Right now, the air around you is carrying thousands of conversations — police dispatch, aircraft, weather satellites, your neighbor's doorbell — all at once, all invisible, all riding ripples in the electromagnetic field. In the next half hour you'll learn the physics that makes that possible: what a radio wave actually is, the three knobs every transmitter turns, why spectrum is priced like real estate, and the one number that decides whether a signal decodes or dies. This is Radio Fundamentals — how signals actually work.


## [0:34] Course intro

**[0:34] map-card**
Welcome. This course is for anyone pointing a software-defined radio at the sky — scanner listeners, new hams, tinkerers with a thirty-dollar dongle — and for anyone who's ever nodded along to "megahertz" without being totally sure what it measures. No math beyond arithmetic, no prerequisites. Just the mental models underneath every antenna, every waterfall display, and every decode.

**[0:58] map-card**
Six chapters, each one idea. First, the radio wave itself — what's actually rippling. Then frequency, the address every signal lives at. Then modulation — how a wave is made to carry a message. Then bandwidth, the width of the road it drives on. And finally the measuring tools: the decibel, radio's own arithmetic, and signal-to-noise ratio, the number that decides everything.

**[1:23] map-card**
Every chapter stands alone, and each one matches a written Field Guide article at gophertrunk dot org — linked below, chapter by chapter, if you want the formulas and the sources. Skip freely. But watched in order, each idea hands off to the next. Let's start with the ripple itself.


## [1:42] Radio wave

**[1:42] Title card**

**[1:44] ripple**
A radio wave is a ripple — not in water, but in the electromagnetic field that fills all of space. Drop a stone in a pond and rings spread outward. Feed an alternating current into an antenna, and the same thing happens to the invisible field around it: a disturbance detaches and races away at the speed of light. Every wireless thing you own — your phone, your car fob, a police radio — is throwing stones into that same pond.

**[2:11] definition-card**
Formally, a radio wave is electromagnetic radiation in the radio range of the spectrum — by convention, about three kilohertz up to three hundred gigahertz. It is the same physical phenomenon as light. Radio, microwaves, infrared, visible light: one family, different frequencies. Radio just sits at the low, long-wavelength end — the part that bends around hills and passes through walls.

**[2:36] leapfrog**
Here is the mechanism. The accelerating charges in the antenna create an oscillating electric field. A changing electric field generates a magnetic field at right angles to it. And a changing magnetic field regenerates the electric field. The two leapfrog each other, detach from the antenna, and self-propagate — no wires, no air, no medium needed at all. That is why radio crosses the vacuum of space just fine.

**[3:01] wave-anatomy**
Freeze one of those waves and it looks like this. Three properties describe it completely. Amplitude — how strong the wave is. Frequency — how many times it cycles each second. And phase — where in its cycle it is at a given instant. Hold on to those three, because they are the only three things a transmitter can manipulate. Every radio scheme ever invented — AM, FM, the digital modulation in a trunked police system — is just a different way of wiggling amplitude, frequency, or phase in step with a message.

**[3:32] journey**
Out in the world, the ripple spreads and weakens — power thins out with the square of distance. By the time it reaches a receiving antenna ten miles away, the passing field induces a current of just a few microvolts. That whisper is what a receiver amplifies, filters, and decodes. Everything in radio engineering — antennas, amplifiers, the decibel math coming later in this course — exists to protect that whisper on its trip.

**[3:59] gt-tie-in**
And this is where GopherTrunk picks up the story. A software-defined radio doesn't decode the wave directly: its front end mixes a slice of spectrum down and digitizes it into a stream of IQ samples — complex numbers that capture the wave's amplitude and phase at every instant. From that point on, the physical wave has become arithmetic. But every property you just met survives the conversion — and recovering the message means tracking all three.

**[4:27] Recap card**
So: a radio wave is a self-propagating ripple in the electromagnetic field, it lives between three kilohertz and three hundred gigahertz, and it gives a transmitter exactly three knobs to turn. The full write-up is linked below.


## [4:45] Transition

**[4:45] map-card**
So a wave is a ripple with three properties. The first one — how fast it wiggles — is so important it gets a whole chapter: frequency.


## [4:54] Frequency

**[4:54] Title card**

**[4:56] dial**
One hundred sixty-two point five five zero megahertz. Tune there, almost anywhere in the United States, and you'll hear the robotic voice of a weather station. That number — the one thing everybody knows about a radio signal — is its frequency: how many times the wave cycles each second. Get it right and the signal is yours. Get it wrong by a hair and you hear nothing at all.

**[5:19] definition-card**
Frequency is measured in hertz — one hertz is one cycle per second, named for Heinrich Hertz, who first generated and detected radio waves in the eighteen-eighties. It's the inverse of the cycle's duration: a signal at one hundred megahertz repeats a hundred million times a second, so each cycle lasts just ten nanoseconds.

**[5:39] two-waves**
Picture two waves side by side for one second. The low-frequency wave fits a handful of cycles into that second; the high-frequency wave packs in thousands of times more. Same speed of travel — always the speed of light — just a faster wiggle. And since radio numbers get big fast, we scale the unit: kilohertz for thousands, megahertz for millions, gigahertz for billions of cycles per second.

**[6:05] seesaw**
Frequency has a twin: wavelength — the physical distance one cycle covers in flight. Because every radio wave travels at the same speed, the two are locked on a seesaw: wavelength equals the speed of light divided by frequency. Double the frequency and the wavelength halves. An FM broadcast station at one hundred megahertz has waves three meters long; a UHF public-safety channel near four hundred sixty megahertz, about sixty-five centimeters. That's why antennas come in different sizes — an antenna wants to match its wave, and we'll spend a whole chapter on that later.

**[6:41] spectrum-axis**
Lay every possible frequency along one axis and you get the radio spectrum — and its neighborhoods have names. HF, the shortwave bands that bounce off the upper atmosphere and cross oceans. VHF, home of FM broadcast, aircraft, and marine radio. UHF, where most trunked public-safety systems, cell phones, and Wi-Fi live. A frequency isn't just a number — it's an address, with physics and regulations attached.

**[7:08] gt-tie-in**
For an SDR, frequency is the first thing you set: it decides what slice of spectrum lands in the capture. But here's the catch — cheap tuner hardware runs slightly off its dial frequency, by a few parts per million. GopherTrunk, like any serious decoder, applies a correction and then continuously tracks the residual error while it decodes, because at these precisions, oscillators drift with temperature. Getting the frequency right, and keeping it right, is the first requirement for any decode.

**[7:39] Recap card**
So: frequency is cycles per second, it's chained to wavelength through the speed of light, and it places every signal into a named neighborhood of the spectrum. Full write-up linked below.
