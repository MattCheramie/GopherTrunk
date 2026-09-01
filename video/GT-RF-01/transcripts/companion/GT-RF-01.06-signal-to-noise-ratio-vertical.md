# Signal-to-noise ratio (SNR) — transcript (GT-RF-01.06-signal-to-noise-ratio-vertical.mp4)

_9:16 re-edit · 2:58 · 1080×1920 · title card replaced by a 2 s burned hook_

**[0:00] party**
Signal-to-noise ratio is the number that decides everything. Every receiver on Earth is listening to a party it can't leave: below every signal hisses a floor of noise — thermal noise from physics itself, plus every switching power supply and LED billboard in town. Whether your signal decodes comes down to one question: how far does it stand above that floor? That gap is the signal-to-noise ratio — SNR — and it is the single best predictor of whether the bits come out clean.

**[0:30] definition-card**
The definition is one subtraction: signal power minus noise-floor power, both in d-B-m, giving a gap in decibels. A signal at minus eighty-five over a floor at minus one-oh-five has twenty decibels of SNR. And because it's a difference, any calibration offset in your receiver cancels out — which is why SNR means something even on twenty-five dollar hardware that has no idea what a true d-B-m is.

**[0:56] thresholds**
Every mode has a price of admission. Analog FM voice becomes readable around ten to twelve decibels of SNR. The digital voice modes GopherTrunk decodes — C4FM, CQPSK — want roughly fifteen to twenty at the demodulator for a clean lock. Below the threshold, things don't degrade gracefully…

**[1:15] waterfall-curve**
…they fall off a cliff. Digital links live on a waterfall curve: a few decibels above threshold, the error rate is negligible — the audio is perfect. A few decibels below, the link collapses entirely. That's why digital radio never gets static-y like analog; it's flawless, then gone. Forward error correction — deliberately sending redundant bits — buys the cliff a few decibels of grace, letting a mode decode below the SNR its raw modulation could survive.

**[1:45] bandwidth-link**
And here's the payoff of the bandwidth chapter's closing idea: the noise floor isn't fixed. Noise power scales with the bandwidth you admit — the floor rises about three decibels every time you double your filter width. Narrow the filter to hug the signal's occupied bandwidth and you shut noise out while keeping all the signal: free SNR, no new antenna required. It also means an SNR figure only really means something at a stated bandwidth — worth remembering when comparing numbers between systems.

**[2:15] gt-tie-in**
GopherTrunk reports live demodulator SNR and error-vector magnitude for every channel it decodes, in decibels, precisely so you can see how much margin a link has. A failed decode with healthy SNR isn't a weak signal — it's something else: overload, wrong tuning, multipath. The number tells you which problem you're hunting. When you're improving a station, you are really improving one number — and this is it.

**[2:40] Recap card**
So: SNR is the gap between your signal and the floor, digital modes live or die on a cliff around fifteen to twenty decibels, and the cheapest SNR you'll ever buy is a filter no wider than the signal. Full write-up linked below.
