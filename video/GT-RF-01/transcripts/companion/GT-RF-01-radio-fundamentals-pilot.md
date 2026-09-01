# Radio Fundamentals — pilot transcript

_21:24 · 1920×1080 · chapter timestamps match the video_


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


## [7:53] Transition

**[7:54] map-card**
You can now find any signal's address on the spectrum. But a wave sitting at an address, perfectly steady, says nothing at all. Making it talk is next: modulation.


## [8:05] Modulation

**[8:05] Title card**

**[8:07] bare-carrier**
Modulation is the answer to an embarrassing question: a perfect radio wave says absolutely nothing. A pure, steady carrier is just a hum in the spectrum — endlessly repeating, perfectly predictable, carrying zero information. To make a wave say something, you have to change it. Modulation is the art of changing a carrier wave in step with a message — and every radio scheme in history is a variation on that one move.

**[8:33] three-knobs**
Remember the three properties from the last chapter's write-up — a sine wave has an amplitude, a frequency, and a phase, and those are the only things a transmitter can touch. So there are only three families of modulation. Everything from nineteen-twenties broadcasting to your phone is built from these three knobs.

**[8:51] am-fm**
Turn the amplitude knob and you get AM: the message is written into the carrier's height — its envelope literally traces the audio. Simple, but noise also lives in amplitude, which is why AM crackles in a thunderstorm. Turn the frequency knob and you get FM: the message speeds up and slows down the carrier's wiggle while the height stays constant — so amplitude noise mostly wipes right off. That constant envelope is why FM sounds cleaner than AM, and the third knob, phase, nudges the carrier's timing — closely related to FM, and the foundation of almost everything digital.

**[9:29] symbols**
Analog modulation varies the carrier continuously. Digital modulation makes one crucial change: it switches the carrier among a finite menu of states, called symbols, each standing for one or more bits. Shift between discrete frequencies and it's FSK — frequency-shift keying. Discrete phases: PSK. Combine amplitude and phase levels: QAM. Draw the menu as dots on a plane — amplitude out from the center, phase around it — and you get the constellation diagram, the map every digital receiver uses to decide which symbol just arrived.

**[10:05] trade**
And here is the trade that shapes all of digital radio. More dots on the map means more bits per symbol — faster data in the same bandwidth. But more dots also means the dots sit closer together, so it takes less noise to smudge one into its neighbor. Fast-but-fragile, or slow-but-tough. Wi-Fi at close range runs hundreds of points; a police radio fighting fades across a whole city runs just four.

**[10:32] gt-tie-in**
That's exactly what GopherTrunk spends its life demodulating. The land-mobile systems it decodes chose rugged, low-order schemes on purpose: four-level FSK — four frequency steps, two bits per symbol — for DMR and NXDN, and a rotating four-phase scheme called pi-over-four DQPSK for TETRA and P25's simulcast flavor. Once the signal is digitized, demodulation is just measuring amplitude, frequency, and phase — the same three knobs — and mapping them back to bits.

**[11:04] Recap card**
So: modulation turns a mute carrier into a messenger, there are exactly three knobs to turn, and digital radio is a menu of symbol states traded off between speed and toughness. Full write-up linked below.


## [11:20] Transition

**[11:21] map-card**
Here's the catch modulation doesn't advertise: the moment a carrier starts talking, it spreads out and takes up room. How much room — and why that room is so expensive — is bandwidth.


## [11:32] Bandwidth

**[11:32] Title card**

**[11:34] lanes**
Radio spectrum is real estate — and bandwidth is how much of it a signal occupies. A narrowband voice channel takes about twelve and a half kilohertz of road. An FM broadcast station, two hundred kilohertz — sixteen times wider. A Wi-Fi channel, tens of megahertz. Bandwidth decides how fast information can flow, how many signals fit side by side, and how much spectrum a receiver has to swallow. It may be the most consequential number in radio.

**[12:03] definition-card**
Formally: bandwidth is the width, in hertz, of the frequency range a signal occupies.

**[12:09] sidebands**
Why does a signal have width at all? An unmodulated carrier is a single spectral line — zero width. The moment you modulate it, energy spreads into sidebands on either side, and the faster the message changes, the wider they spread. Information takes room. That's not an engineering flaw; it's physics. Regulators then draw lane markings: each service gets a channel of a defined width, with guard bands of empty spectrum between lanes to keep neighbors from sideswiping each other.

**[12:38] shannon**
And here's the law that makes bandwidth precious. Shannon's theorem sets the ceiling on any channel's data rate: capacity equals bandwidth times the log of one plus the signal-to-noise ratio. Look at the shape of that formula. Capacity grows linearly with bandwidth — double the width, double the ceiling. But only logarithmically with signal power — doubling your SNR buys you a mere fraction more. If you want more data, more bandwidth is the straight road, which is exactly why spectrum auctions raise billions.

**[13:09] noise-cut**
Bandwidth cuts the other way too. Noise power grows with every hertz you listen to — open your receiver's filter wider than the signal and all you admit is more hiss. The craft on the receive side is matching the filter to the signal: wide enough to pass it undistorted, narrow enough to shut out everything else. Keep that idea — it becomes the star of the signal-to-noise chapter.

**[13:31] gt-tie-in**
GopherTrunk lives on both sides of this trade at once. Its SDR captures a slice several megahertz wide — wide enough to watch a trunked system's control channel and all its voice channels simultaneously. Then digital down-converters carve out each narrow twelve-and-a-half-kilohertz channel and filter it down to just the bandwidth the decoder needs. Wide capture to see everything; narrow channels to decode cleanly. That split is trunk-tracking.

**[13:59] Recap card**
So: bandwidth is a signal's footprint on the spectrum, the linear lever in Shannon's speed limit, and — on receive — a filter you keep just as narrow as you can. Full write-up linked below.


## [14:14] Transition

**[14:15] map-card**
We just compared powers that differ by billions. Radio has its own arithmetic for exactly that — small numbers you can add in your head. Meet the decibel.


## [14:24] Decibel (dB)

**[14:24] Title card**

**[14:26] numbers-flying**
Decibels exist because radio numbers are ridiculous. The signal your antenna picks up from a repeater ten miles away can be a hundred billion times weaker than the signal leaving that repeater's transmitter. Nobody wants to do math with eleven zeros. The decibel is how radio engineers make those numbers small, friendly — and addable.

**[14:46] definition-card**
A decibel is a logarithmic way of writing a ratio between two power levels. Not an amount — a ratio. Ten decibels means ten times the power. Twenty decibels means a hundred times. Thirty means a thousand. Every ten decibels you add multiplies the power by another ten.

**[15:04] piano**
The everyday anchor is a piano. Each octave is the same-sized step to your ear, but each one doubles the frequency. Decibels do that for power: equal steps that stand for equal multiplications. Your ear and your radio both live on that kind of scale, which is why decibels feel natural once they click.

**[15:23] db-ladder**
Three numbers are worth memorizing. Plus three decibels: double the power. Minus three: half. Plus ten: ten times. That's it — every other value is a combination. Sixteen decibels? That's ten plus three plus three: ten times two times two — forty times the power. You just did logarithms in your head without noticing.

**[15:44] link-budget**
And here's the superpower: on a log scale, multiplication becomes addition. Follow a signal from transmitter to receiver — every cable loses a few decibels, every antenna adds a few, the path between them takes away a hundred or more. Instead of multiplying eleven-digit numbers, you just add and subtract small ones down the chain. That running sum is called a link budget, and it's how every radio link on Earth is designed.

**[16:11] suffix-card**
One trap. A plain decibel is always relative — a ratio between two things. The moment you see a letter after it, it's become an absolute number, measured against a fixed reference: d-B-m means compared to one milliwatt; d-B-F-S means compared to the loudest sample your SDR can represent. Same math, different anchor.

**[16:34] gt-tie-in**
You'll see all three flavors in GopherTrunk's dashboard: signal level in d-B-F-S, signal-to-noise ratio in plain decibels, and hardware specs quoted in d-B-m. Now they're not three mysteries — they're one idea with three anchors.

**[16:51] Recap card**
So: decibels turn absurd ratios into small numbers, multiplication into addition, and three memorized steps into fluent mental math. The full write-up, with the formulas, is linked below.


## [17:07] Transition

**[17:07] map-card**
Armed with decibels, you can finally ask radio's bottom-line question: how far above the noise is my signal? One last chapter: signal-to-noise ratio.


## [17:17] Signal-to-noise ratio (SNR)

**[17:17] Title card**

**[17:19] party**
Signal-to-noise ratio is the number that decides everything. Every receiver on Earth is listening to a party it can't leave: below every signal hisses a floor of noise — thermal noise from physics itself, plus every switching power supply and LED billboard in town. Whether your signal decodes comes down to one question: how far does it stand above that floor? That gap is the signal-to-noise ratio — SNR — and it is the single best predictor of whether the bits come out clean.

**[17:49] definition-card**
The definition is one subtraction: signal power minus noise-floor power, both in d-B-m, giving a gap in decibels. A signal at minus eighty-five over a floor at minus one-oh-five has twenty decibels of SNR. And because it's a difference, any calibration offset in your receiver cancels out — which is why SNR means something even on twenty-five dollar hardware that has no idea what a true d-B-m is.

**[18:15] thresholds**
Every mode has a price of admission. Analog FM voice becomes readable around ten to twelve decibels of SNR. The digital voice modes GopherTrunk decodes — C4FM, CQPSK — want roughly fifteen to twenty at the demodulator for a clean lock. Below the threshold, things don't degrade gracefully…

**[18:34] waterfall-curve**
…they fall off a cliff. Digital links live on a waterfall curve: a few decibels above threshold, the error rate is negligible — the audio is perfect. A few decibels below, the link collapses entirely. That's why digital radio never gets static-y like analog; it's flawless, then gone. Forward error correction — deliberately sending redundant bits — buys the cliff a few decibels of grace, letting a mode decode below the SNR its raw modulation could survive.

**[19:04] bandwidth-link**
And here's the payoff of the bandwidth chapter's closing idea: the noise floor isn't fixed. Noise power scales with the bandwidth you admit — the floor rises about three decibels every time you double your filter width. Narrow the filter to hug the signal's occupied bandwidth and you shut noise out while keeping all the signal: free SNR, no new antenna required. It also means an SNR figure only really means something at a stated bandwidth — worth remembering when comparing numbers between systems.

**[19:35] gt-tie-in**
GopherTrunk reports live demodulator SNR and error-vector magnitude for every channel it decodes, in decibels, precisely so you can see how much margin a link has. A failed decode with healthy SNR isn't a weak signal — it's something else: overload, wrong tuning, multipath. The number tells you which problem you're hunting. When you're improving a station, you are really improving one number — and this is it.

**[20:00] Recap card**
So: SNR is the gap between your signal and the floor, digital modes live or die on a cliff around fifteen to twenty decibels, and the cheapest SNR you'll ever buy is a filter no wider than the signal. Full write-up linked below.


## [20:18] Outro

**[20:18] map-card**
And that's the foundation. Here's the whole course in one breath: a radio wave is a self-propagating ripple with three adjustable properties; frequency is its address and wavelength its size; modulation wiggles those properties to carry a message; the message takes up bandwidth; decibels let you do the power math in your head; and signal-to-noise ratio decides whether any of it decodes. Every one of those ideas has a full written article — with the formulas and sources — in the GopherTrunk Field Guide, linked below.

**[20:49] next-pointer**
From here, the next course takes these fundamentals and puts real hardware on your desk: SDR from Zero — your first software-defined radio. If this one earned it, subscribing is the easiest way to catch that release.


## [21:04] End slate (music only)
