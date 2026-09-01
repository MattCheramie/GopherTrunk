# GT-RF-01.11 — Decibel (dB)
slug: decibel · type: term · treatment: 2.1 term (playbook §2.1 sample, verified) · also_slugs: [dbm, dbfs] · target: 3:30–4:30 · words: 601

**[V: Title card — "Decibel (dB)"]**

**[V: numbers flying past: 0.00000000002 W … 2 W]** [CLIP c1 starts]
Decibels exist because radio numbers are ridiculous. The signal your antenna picks up from a repeater ten miles away can be a hundred *billion* times weaker than the signal leaving that repeater's transmitter. Nobody wants to do math with eleven zeros. The decibel is how radio engineers make those numbers small, friendly — and addable.

**[V: definition card: "dB = 10 · log₁₀(P₁/P₂) — a ratio, not an amount"]**
A decibel is a logarithmic way of writing a **ratio** between two power levels. Not an amount — a ratio. Ten decibels means "ten times the power." Twenty decibels means a hundred times. Thirty means a thousand. Every ten dB you add multiplies the power by another ten. [CLIP c1 ends]

**[V: analogy — piano keyboard: each octave = same step, doubling pitch]**
The everyday anchor is a piano. Each octave is the same-sized step to your ear, but each one *doubles* the frequency. Decibels do that for power: equal steps that stand for equal *multiplications*. Your ear and your radio both live on that kind of scale, which is why dB feels natural once it clicks.

**[V: animated card stack: +3 dB ≈ ×2 · −3 dB ≈ ½ · +10 dB = ×10]** [CLIP c2]
Three numbers are worth memorizing. Plus three dB: double the power. Minus three dB: half. Plus ten: ten times. That's it — every other value is a combo. Sixteen dB? That's 10 + 3 + 3: ten times two times two — forty times the power. You just did logarithms in your head without noticing.

**[V: card: "power: 10·log₁₀ · voltage: 20·log₁₀ — because P ∝ V²"]**
One footnote for the meter-readers: those rules are for *power*. When the quantity is a voltage, a current, or a field strength, power goes as the *square* of the amplitude, so the same ratio is written with twenty log instead of ten. It's not a different decibel — it falls straight out of the squaring — but it's a classic source of two-times errors: a voltage that doubles is plus *six* dB, not plus three.

**[V: link-budget chain: TX +37 dBm → feedline −3 dB → antenna +6 dB → path −120 dB → RX]**
And here's the superpower: on a log scale, multiplication becomes **addition**. Follow a signal from transmitter to receiver — every cable loses a few dB, every antenna adds a few, the path between them takes away a hundred or more. Instead of multiplying raw numbers twelve orders of magnitude apart, you just add and subtract small ones down the chain. That running sum is called a link budget, and it's how every radio link on Earth is designed.

**[V: card: "dB = ratio · dBm = power vs 1 mW · dBFS = level vs digital full scale"]** [CLIP c3 starts]
One trap. A plain dB is always relative — a ratio between two things. The moment you see a letter after it, it's become an absolute number, measured against a fixed reference: dB-m means compared to one milliwatt; dB-F-S means compared to the loudest sample your SDR can represent. Same math, different anchor.

**[V: split card: dBm scale (0 = 1 mW, received signals negative, "−70 beats −90") · dBFS scale (0 at the TOP, "0 = clipping")]**
And each anchor has its own quirk. Received radio signals are tiny fractions of a milliwatt, so they land as *negative* dBm — and less negative means stronger: minus seventy beats minus ninety by twenty dB, which is a hundred times the power. On the dBFS scale, zero is the *ceiling* — the largest value the converter can represent — so real samples are always negative, and driving the input to zero dBFS means clipping. [CLIP c3 ends]

**[V: GopherTrunk web UI: spectrum display spanning a −120 dBm floor to a −20 dBm signal; signal meter in dBFS; SNR readout in dB]**
Decibels are also what let one screen hold that whole absurd range: a spectrum display marked in dB shows a noise floor near minus one-twenty and a strong signal at minus twenty on the same plot — a linear scale could never resolve both. And you'll see all three flavors in GopherTrunk's dashboard: signal level in dBFS, signal-to-noise ratio in plain dB, and hardware specs quoted in dBm. Now they're not three mysteries — they're one idea with three anchors.

**[V: Recap card: "① dB = a ratio, log scale ② +3 ×2 · +10 ×10 ③ suffix = absolute (dBm, dBFS)"]**
So: decibels turn absurd ratios into small numbers, multiplication into addition, and three memorized steps into fluent mental math. The full write-up, with the formulas, is linked below.

## Clip picks
- c1: hook + "ratio not amount" definition (~35s)
- c2: the mental-math trick (+3/+10 stack) (~30s)
- c3: the trap (dB vs dBm vs dBFS, negative dBm, 0 dBFS ceiling) (~35s)

## Vertical plan
Direct center crop. Cards already center-safe → only work is re-hooked first 2 s (the flying-numbers visual) + burned captions. The split dBm/dBFS card stacks vertically for 9:16.

## Assets
- Core animation: rebuild of the article's dB-ladder + gain/loss-chain SVG (the +10 dB = ×10 ladder and the LNA/coax/amp running sum)
- Flying-numbers hook graphic (0.00000000002 W … 2 W)
- Definition card — notation per the article: dB = 10·log₁₀(P₁/P₂)
- Piano-octave analogy graphic
- +3/−3/+10 card stack (article rules: +3 dB ≈ ×2, +10 dB = ×10)
- Link-budget chain graphic; split dBm/dBFS scale card (dBm ticks per the dbm article: +30 = 1 W, 0 = 1 mW, −80 strong RX, −120 in the noise; dBFS ceiling per the dbfs article)
- GopherTrunk screen capture: dashboard signal meter (dBFS) + SNR readout (dB), dark theme
- Title + recap cards from templates

## Checklist deltas
- [ ] Formula card matches the article's notation exactly — dB = 10·log₁₀(P₁/P₂) (sample's P₂/P₁ corrected)
- [ ] Analogy actually maps (octave = equal step for equal multiplication; no pitch/power conflation)
- [ ] also_slugs [dbm, dbfs] recorded for `videos.yml` so this segment embeds on all three reference pages
- [ ] dBm/dBFS claims verified against their articles (negative received dBm, −70 vs −90 = 20 dB = ×100; 0 dBFS = ADC max, clipping at full scale)
