# Radio wave — transcript (GT-RF-01.01-radio-wave-vertical.mp4)

_9:16 re-edit · 3:01 · 1080×1920 · title card replaced by a 2 s burned hook_

**[0:00] ripple**
A radio wave is a ripple — not in water, but in the electromagnetic field that fills all of space. Drop a stone in a pond and rings spread outward. Feed an alternating current into an antenna, and the same thing happens to the invisible field around it: a disturbance detaches and races away at the speed of light. Every wireless thing you own — your phone, your car fob, a police radio — is throwing stones into that same pond.

**[0:27] definition-card**
Formally, a radio wave is electromagnetic radiation in the radio range of the spectrum — by convention, about three kilohertz up to three hundred gigahertz. It is the same physical phenomenon as light. Radio, microwaves, infrared, visible light: one family, different frequencies. Radio just sits at the low, long-wavelength end — the part that bends around hills and passes through walls.

**[0:53] leapfrog**
Here is the mechanism. The accelerating charges in the antenna create an oscillating electric field. A changing electric field generates a magnetic field at right angles to it. And a changing magnetic field regenerates the electric field. The two leapfrog each other, detach from the antenna, and self-propagate — no wires, no air, no medium needed at all. That is why radio crosses the vacuum of space just fine.

**[1:17] wave-anatomy**
Freeze one of those waves and it looks like this. Three properties describe it completely. Amplitude — how strong the wave is. Frequency — how many times it cycles each second. And phase — where in its cycle it is at a given instant. Hold on to those three, because they are the only three things a transmitter can manipulate. Every radio scheme ever invented — AM, FM, the digital modulation in a trunked police system — is just a different way of wiggling amplitude, frequency, or phase in step with a message.

**[1:49] journey**
Out in the world, the ripple spreads and weakens — power thins out with the square of distance. By the time it reaches a receiving antenna ten miles away, the passing field induces a current of just a few microvolts. That whisper is what a receiver amplifies, filters, and decodes. Everything in radio engineering — antennas, amplifiers, the decibel math coming later in this course — exists to protect that whisper on its trip.

**[2:15] gt-tie-in**
And this is where GopherTrunk picks up the story. A software-defined radio doesn't decode the wave directly: its front end mixes a slice of spectrum down and digitizes it into a stream of IQ samples — complex numbers that capture the wave's amplitude and phase at every instant. From that point on, the physical wave has become arithmetic. But every property you just met survives the conversion — and recovering the message means tracking all three.

**[2:43] Recap card**
So: a radio wave is a self-propagating ripple in the electromagnetic field, it lives between three kilohertz and three hundred gigahertz, and it gives a transmitter exactly three knobs to turn. The full write-up is linked below.
