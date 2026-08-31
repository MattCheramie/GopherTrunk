# GT-RF-01.05 — Decibel (dB)
slug: decibel
also_slugs: [dbm, dbfs]
type: term
target: ~3:50
note: adapted from the production playbook's sample script; formula notation matched to the article (10·log₁₀(P₁/P₂)).

**[V: title | "Decibel (dB)" | GT-RF-01.05]**

**[V: numbers-flying — absurd numbers stream past: 0.00000000002 W … 2 W; eleven zeros highlighted]** [CLIP c1 start]
Decibels exist because radio numbers are ridiculous. The signal your antenna picks up from a repeater ten miles away can be a hundred *billion* times weaker than the signal leaving that repeater's transmitter. Nobody wants to do math with eleven zeros. The decibel is how radio engineers make those numbers small, friendly — and addable.

**[V: definition-card | "dB = 10 · log₁₀(P₁/P₂) — a ratio, not an amount"]**
A decibel is a logarithmic way of writing a **ratio** between two power levels. Not an amount — a ratio. Ten decibels means ten times the power. Twenty decibels means a hundred times. Thirty means a thousand. Every ten decibels you add multiplies the power by another ten. [CLIP c1 end]

**[V: piano — keyboard; octave brackets, each octave = same step, frequency doubling]**
The everyday anchor is a piano. Each octave is the same-sized step to your ear, but each one *doubles* the frequency. Decibels do that for power: equal steps that stand for equal *multiplications*. Your ear and your radio both live on that kind of scale, which is why decibels feel natural once they click.

**[V: db-ladder — animated card stack from the article's figure: +3 dB → ×2 · −3 dB → ½ · +10 dB → ×10; then 16 dB decomposes into 10+3+3]** [CLIP c2 start]
Three numbers are worth memorizing. Plus three decibels: double the power. Minus three: half. Plus ten: ten times. That's it — every other value is a combination. Sixteen decibels? That's ten plus three plus three: ten times two times two — forty times the power. You just did logarithms in your head without noticing. [CLIP c2 end]

**[V: link-budget — chain builds left to right: TX +37 dBm → feedline −3 dB → antenna +6 dB → path −120 dB → RX; running sum ticks along]**
And here's the superpower: on a log scale, multiplication becomes **addition**. Follow a signal from transmitter to receiver — every cable loses a few decibels, every antenna adds a few, the path between them takes away a hundred or more. Instead of multiplying eleven-digit numbers, you just add and subtract small ones down the chain. That running sum is called a link budget, and it's how every radio link on Earth is designed.

**[V: suffix-card | "dB = ratio · dBm = power vs 1 mW · dBFS = level vs digital full scale"]** [CLIP c3 start]
One trap. A plain decibel is always relative — a ratio between two things. The moment you see a letter after it, it's become an absolute number, measured against a fixed reference: d-B-m means compared to one milliwatt; d-B-F-S means compared to the loudest sample your SDR can represent. Same math, different anchor. [CLIP c3 end]

**[V: gt-tie-in — GopherTrunk dashboard mock: signal meter in dBFS, SNR readout in dB, hardware spec in dBm — three callouts]**
You'll see all three flavors in GopherTrunk's dashboard: signal level in d-B-F-S, signal-to-noise ratio in plain decibels, and hardware specs quoted in d-B-m. Now they're not three mysteries — they're one idea with three anchors.

**[V: recap | "Decibel (dB)" | ① dB = a ratio, on a log scale | ② +3 dB = ×2 · +10 dB = ×10 | ③ A suffix makes it absolute: dBm, dBFS]**
So: decibels turn absurd ratios into small numbers, multiplication into addition, and three memorized steps into fluent mental math. The full write-up, with the formulas, is linked below.
