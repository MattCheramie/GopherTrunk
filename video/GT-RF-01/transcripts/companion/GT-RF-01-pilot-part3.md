# Pilot part 3 transcript (Decibels → end slate)

_7:00 · 1920×1080 · chapter timestamps match the video_


## [0:00] Decibel (dB)

**[0:00] Title card**

**[0:02] numbers-flying**
Decibels exist because radio numbers are ridiculous. The signal your antenna picks up from a repeater ten miles away can be a hundred billion times weaker than the signal leaving that repeater's transmitter. Nobody wants to do math with eleven zeros. The decibel is how radio engineers make those numbers small, friendly — and addable.

**[0:22] definition-card**
A decibel is a logarithmic way of writing a ratio between two power levels. Not an amount — a ratio. Ten decibels means ten times the power. Twenty decibels means a hundred times. Thirty means a thousand. Every ten decibels you add multiplies the power by another ten.

**[0:40] piano**
The everyday anchor is a piano. Each octave is the same-sized step to your ear, but each one doubles the frequency. Decibels do that for power: equal steps that stand for equal multiplications. Your ear and your radio both live on that kind of scale, which is why decibels feel natural once they click.

**[0:58] db-ladder**
Three numbers are worth memorizing. Plus three decibels: double the power. Minus three: half. Plus ten: ten times. That's it — every other value is a combination. Sixteen decibels? That's ten plus three plus three: ten times two times two — forty times the power. You just did logarithms in your head without noticing.

**[1:20] link-budget**
And here's the superpower: on a log scale, multiplication becomes addition. Follow a signal from transmitter to receiver — every cable loses a few decibels, every antenna adds a few, the path between them takes away a hundred or more. Instead of multiplying eleven-digit numbers, you just add and subtract small ones down the chain. That running sum is called a link budget, and it's how every radio link on Earth is designed.

**[1:47] suffix-card**
One trap. A plain decibel is always relative — a ratio between two things. The moment you see a letter after it, it's become an absolute number, measured against a fixed reference: d-B-m means compared to one milliwatt; d-B-F-S means compared to the loudest sample your SDR can represent. Same math, different anchor.

**[2:09] gt-tie-in**
You'll see all three flavors in GopherTrunk's dashboard: signal level in d-B-F-S, signal-to-noise ratio in plain decibels, and hardware specs quoted in d-B-m. Now they're not three mysteries — they're one idea with three anchors.

**[2:26] Recap card**
So: decibels turn absurd ratios into small numbers, multiplication into addition, and three memorized steps into fluent mental math. The full write-up, with the formulas, is linked below.


## [2:42] Transition

**[2:43] map-card**
Armed with decibels, you can finally ask radio's bottom-line question: how far above the noise is my signal? One last chapter: signal-to-noise ratio.


## [2:53] Signal-to-noise ratio (SNR)

**[2:53] Title card**

**[2:55] party**
Signal-to-noise ratio is the number that decides everything. Every receiver on Earth is listening to a party it can't leave: below every signal hisses a floor of noise — thermal noise from physics itself, plus every switching power supply and LED billboard in town. Whether your signal decodes comes down to one question: how far does it stand above that floor? That gap is the signal-to-noise ratio — SNR — and it is the single best predictor of whether the bits come out clean.

**[3:25] definition-card**
The definition is one subtraction: signal power minus noise-floor power, both in d-B-m, giving a gap in decibels. A signal at minus eighty-five over a floor at minus one-oh-five has twenty decibels of SNR. And because it's a difference, any calibration offset in your receiver cancels out — which is why SNR means something even on twenty-five dollar hardware that has no idea what a true d-B-m is.

**[3:51] thresholds**
Every mode has a price of admission. Analog FM voice becomes readable around ten to twelve decibels of SNR. The digital voice modes GopherTrunk decodes — C4FM, CQPSK — want roughly fifteen to twenty at the demodulator for a clean lock. Below the threshold, things don't degrade gracefully…

**[4:10] waterfall-curve**
…they fall off a cliff. Digital links live on a waterfall curve: a few decibels above threshold, the error rate is negligible — the audio is perfect. A few decibels below, the link collapses entirely. That's why digital radio never gets static-y like analog; it's flawless, then gone. Forward error correction — deliberately sending redundant bits — buys the cliff a few decibels of grace, letting a mode decode below the SNR its raw modulation could survive.

**[4:40] bandwidth-link**
And here's the payoff of the bandwidth chapter's closing idea: the noise floor isn't fixed. Noise power scales with the bandwidth you admit — the floor rises about three decibels every time you double your filter width. Narrow the filter to hug the signal's occupied bandwidth and you shut noise out while keeping all the signal: free SNR, no new antenna required. It also means an SNR figure only really means something at a stated bandwidth — worth remembering when comparing numbers between systems.

**[5:10] gt-tie-in**
GopherTrunk reports live demodulator SNR and error-vector magnitude for every channel it decodes, in decibels, precisely so you can see how much margin a link has. A failed decode with healthy SNR isn't a weak signal — it's something else: overload, wrong tuning, multipath. The number tells you which problem you're hunting. When you're improving a station, you are really improving one number — and this is it.

**[5:35] Recap card**
So: SNR is the gap between your signal and the floor, digital modes live or die on a cliff around fifteen to twenty decibels, and the cheapest SNR you'll ever buy is a filter no wider than the signal. Full write-up linked below.


## [5:53] Outro

**[5:54] map-card**
And that's the foundation. Here's the whole course in one breath: a radio wave is a self-propagating ripple with three adjustable properties; frequency is its address and wavelength its size; modulation wiggles those properties to carry a message; the message takes up bandwidth; decibels let you do the power math in your head; and signal-to-noise ratio decides whether any of it decodes. Every one of those ideas has a full written article — with the formulas and sources — in the GopherTrunk Field Guide, linked below.

**[6:25] next-pointer**
From here, the next course takes these fundamentals and puts real hardware on your desk: SDR from Zero — your first software-defined radio. If this one earned it, subscribing is the easiest way to catch that release.


## [6:40] End slate (music only)
