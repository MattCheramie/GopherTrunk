# GT-RF-01.01 — Radio wave
slug: radio-wave
also_slugs: [electromagnetic-spectrum]
type: term
target: ~3:55

**[V: title | "Radio wave" | GT-RF-01.01]**

**[V: ripple — a dark pond, a dropped stone, rings expanding; rings morph into RF rings around an antenna]** [CLIP c1 start]
A radio wave is a ripple — not in water, but in the electromagnetic field that fills all of space. Drop a stone in a pond and rings spread outward. Feed an alternating current into an antenna, and the same thing happens to the invisible field around it: a disturbance detaches and races away at the speed of light. Every wireless thing you own — your phone, your car fob, a police radio — is throwing stones into that same pond.

**[V: definition-card | "Radio wave — electromagnetic radiation, ~3 kHz to 300 GHz" with spectrum strip]**
Formally, a radio wave is electromagnetic radiation in the radio range of the spectrum — by convention, about three kilohertz up to three hundred gigahertz. It is the same physical phenomenon as light. Radio, microwaves, infrared, visible light: one family, different frequencies. Radio just sits at the low, long-wavelength end — the part that bends around hills and passes through walls. [CLIP c1 end]

**[V: leapfrog — E field and B field arrows regenerating each other, wave propagating off the antenna]**
Here is the mechanism. The accelerating charges in the antenna create an oscillating electric field. A changing electric field generates a magnetic field at right angles to it. And a changing magnetic field regenerates the electric field. The two leapfrog each other, detach from the antenna, and self-propagate — no wires, no air, no medium needed at all. That is why radio crosses the vacuum of space just fine.

**[V: wave-anatomy — the article's sine figure builds: centerline, the wave, wavelength bracket, amplitude arrow]** [CLIP c2 start]
Freeze one of those waves and it looks like this. Three properties describe it completely. Amplitude — how strong the wave is. Frequency — how many times it cycles each second. And phase — where in its cycle it is at a given instant. Hold on to those three, because they are the *only* three things a transmitter can manipulate. Every radio scheme ever invented — AM, FM, the digital modulation in a trunked police system — is just a different way of wiggling amplitude, frequency, or phase in step with a message. [CLIP c2 end]

**[V: journey — TX antenna, expanding rings fading with distance, tiny RX antenna, meter showing microvolts]**
Out in the world, the ripple spreads and weakens — power thins out with the square of distance. By the time it reaches a receiving antenna ten miles away, the passing field induces a current of just a few *microvolts*. That whisper is what a receiver amplifies, filters, and decodes. Everything in radio engineering — antennas, amplifiers, the decibel math coming later in this course — exists to protect that whisper on its trip.

**[V: gt-tie-in — GopherTrunk pipeline card: wave → antenna → SDR front end → IQ samples → decoded call]**
And this is where GopherTrunk picks up the story. A software-defined radio doesn't decode the wave directly: its front end mixes a slice of spectrum down and digitizes it into a stream of IQ samples — complex numbers that capture the wave's amplitude and phase at every instant. From that point on, the physical wave has become arithmetic. But every property you just met survives the conversion — and recovering the message means tracking all three.

**[V: recap | "Radio wave" | ① A self-propagating EM ripple — same family as light | ② ~3 kHz–300 GHz, at the speed of light | ③ Only 3 knobs: amplitude, frequency, phase]**
So: a radio wave is a self-propagating ripple in the electromagnetic field, it lives between three kilohertz and three hundred gigahertz, and it gives a transmitter exactly three knobs to turn. The full write-up is linked below.
