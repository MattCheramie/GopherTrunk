# GT-RF-01.02 — Frequency
slug: frequency
also_slugs: [wavelength]
type: term
target: ~3:50

**[V: title | "Frequency" | GT-RF-01.02]**

**[V: dial — a tuning readout ticking to 162.550 MHz, a wave pulsing behind it]** [CLIP c1 start]
One hundred sixty-two point five five zero megahertz. Tune there, almost anywhere in the United States, and you'll hear the robotic voice of a weather station. That number — the one thing everybody knows about a radio signal — is its frequency: how many times the wave cycles each second. Get it right and the signal is yours. Get it wrong by a hair and you hear nothing at all.

**[V: definition-card | "Frequency f — cycles per second, in hertz (Hz)" | f = 1/T]**
Frequency is measured in hertz — one hertz is one cycle per second, named for Heinrich Hertz, who first generated and detected radio waves in the eighteen-eighties. It's the inverse of the cycle's duration: a signal at one hundred megahertz repeats a hundred million times a second, so each cycle lasts just ten nanoseconds. [CLIP c1 end]

**[V: two-waves — the article's figure: slow wave on top, fast wave below, cycle counters running]**
Picture two waves side by side for one second. The low-frequency wave fits a handful of cycles into that second; the high-frequency wave packs in thousands of times more. Same speed of travel — always the speed of light — just a faster wiggle. And since radio numbers get big fast, we scale the unit: kilohertz for thousands, megahertz for millions, gigahertz for billions of cycles per second.

**[V: seesaw — λ = c / f as a seesaw: frequency up, wavelength down; examples slide in: 100 MHz ↔ 3 m, 460 MHz ↔ 65 cm]** [CLIP c2 start]
Frequency has a twin: wavelength — the physical distance one cycle covers in flight. Because every radio wave travels at the same speed, the two are locked on a seesaw: wavelength equals the speed of light divided by frequency. Double the frequency and the wavelength halves. An FM broadcast station at one hundred megahertz has waves three meters long; a UHF public-safety channel near four hundred sixty megahertz, about sixty-five centimeters. That's why antennas come in different sizes — an antenna wants to match its wave, and we'll spend a whole chapter on that later. [CLIP c2 end]

**[V: spectrum-axis — a frequency axis with band labels HF / VHF / UHF sliding in, tick marks at real services]**
Lay every possible frequency along one axis and you get the radio spectrum — and its neighborhoods have names. HF, the shortwave bands that bounce off the upper atmosphere and cross oceans. VHF, home of FM broadcast, aircraft, and marine radio. UHF, where most trunked public-safety systems, cell phones, and Wi-Fi live. A frequency isn't just a number — it's an address, with physics and regulations attached.

**[V: gt-tie-in — GopherTrunk spectrum view; tuning readout; a PPM-correction annotation appears]**
For an SDR, frequency is the first thing you set: it decides what slice of spectrum lands in the capture. But here's the catch — cheap tuner hardware runs slightly off its dial frequency, by a few parts per million. GopherTrunk, like any serious decoder, applies a correction and then *continuously* tracks the residual error while it decodes, because at these precisions, oscillators drift with temperature. Getting the frequency right, and keeping it right, is the first requirement for any decode.

**[V: recap | "Frequency" | ① Cycles per second, in hertz | ② λ = c / f — frequency up, wavelength down | ③ Bands are neighborhoods: HF, VHF, UHF]**
So: frequency is cycles per second, it's chained to wavelength through the speed of light, and it places every signal into a named neighborhood of the spectrum. Full write-up linked below.
