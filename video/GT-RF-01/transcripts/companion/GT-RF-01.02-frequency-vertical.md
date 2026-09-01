# Frequency — transcript (GT-RF-01.02-frequency-vertical.mp4)

_9:16 re-edit · 2:57 · 1080×1920 · title card replaced by a 2 s burned hook_

**[0:00] dial**
One hundred sixty-two point five five zero megahertz. Tune there, almost anywhere in the United States, and you'll hear the robotic voice of a weather station. That number — the one thing everybody knows about a radio signal — is its frequency: how many times the wave cycles each second. Get it right and the signal is yours. Get it wrong by a hair and you hear nothing at all.

**[0:22] definition-card**
Frequency is measured in hertz — one hertz is one cycle per second, named for Heinrich Hertz, who first generated and detected radio waves in the eighteen-eighties. It's the inverse of the cycle's duration: a signal at one hundred megahertz repeats a hundred million times a second, so each cycle lasts just ten nanoseconds.

**[0:42] two-waves**
Picture two waves side by side for one second. The low-frequency wave fits a handful of cycles into that second; the high-frequency wave packs in thousands of times more. Same speed of travel — always the speed of light — just a faster wiggle. And since radio numbers get big fast, we scale the unit: kilohertz for thousands, megahertz for millions, gigahertz for billions of cycles per second.

**[1:08] seesaw**
Frequency has a twin: wavelength — the physical distance one cycle covers in flight. Because every radio wave travels at the same speed, the two are locked on a seesaw: wavelength equals the speed of light divided by frequency. Double the frequency and the wavelength halves. An FM broadcast station at one hundred megahertz has waves three meters long; a UHF public-safety channel near four hundred sixty megahertz, about sixty-five centimeters. That's why antennas come in different sizes — an antenna wants to match its wave, and we'll spend a whole chapter on that later.

**[1:45] spectrum-axis**
Lay every possible frequency along one axis and you get the radio spectrum — and its neighborhoods have names. HF, the shortwave bands that bounce off the upper atmosphere and cross oceans. VHF, home of FM broadcast, aircraft, and marine radio. UHF, where most trunked public-safety systems, cell phones, and Wi-Fi live. A frequency isn't just a number — it's an address, with physics and regulations attached.

**[2:12] gt-tie-in**
For an SDR, frequency is the first thing you set: it decides what slice of spectrum lands in the capture. But here's the catch — cheap tuner hardware runs slightly off its dial frequency, by a few parts per million. GopherTrunk, like any serious decoder, applies a correction and then continuously tracks the residual error while it decodes, because at these precisions, oscillators drift with temperature. Getting the frequency right, and keeping it right, is the first requirement for any decode.

**[2:42] Recap card**
So: frequency is cycles per second, it's chained to wavelength through the speed of light, and it places every signal into a named neighborhood of the spectrum. Full write-up linked below.
