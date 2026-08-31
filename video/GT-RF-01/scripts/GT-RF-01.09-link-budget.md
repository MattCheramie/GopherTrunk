# GT-RF-01.09 — Link budget
slug: link-budget
also_slugs: [friis-transmission-equation]
type: term
target: ~3:55
note: pairs the link-budget accounting with the Friis transmission equation that supplies its free-space path-loss term; both article figures animated.

**[V: title | "Link budget" | GT-RF-01.09]**

**[V: ledger — a bank-ledger card: deposits (TX power, antenna gains) and withdrawals (feedline, connectors, path loss); a running balance ticks along in dBm]** [CLIP c1 start]
A link budget is radio's bank statement. Every signal starts with a deposit — the transmitter's power — and then the world makes withdrawals: feedline loss, connector loss, and the enormous toll of the path itself. Antennas at both ends pay a little back in. A link budget is the bookkeeping of every one of those gains and losses on the way from transmitter to receiver, tallied in decibels so the terms simply add and subtract. The bottom line is the predicted received power — and whether a link will work is a question the budget answers before anything gets built.

**[V: definition-card | "P_rx = EIRP − L_path − L_misc + G_rx · margin = P_rx − sensitivity"]**
The canonical form: received power equals the transmitter's effective isotropic radiated power, minus path loss, minus miscellaneous losses — polarization mismatch, rain, pointing error — plus the receive antenna's gain. Compare that against the receiver's sensitivity, the minimum power it needs, and the surplus is the link margin. Comfortably positive, the link closes. Negative or thin, it's unreliable. [CLIP c1 end]

**[V: staircase — the article's SVG animated: P_tx steps up, +G_tx, the long dashed fall of path loss, +G_rx steps back up, landing at P_rx; the margin bracket opens down to the sensitivity floor]**
Watch it as a staircase. Start at the transmit power. Step up by the transmit antenna's gain. Then take the long fall of path loss — the single largest term. Step back up by the receive antenna's gain. Where you land is the received power, and the gap between that landing and the sensitivity floor is your margin. A robust design doesn't aim for zero margin — it reserves a fade margin on top, so fading, multipath, and weather can't drop the link below threshold.

**[V: friis — the Friis article's SVG animated: TX antenna, arcs crossing "distance d" to the RX antenna; the formula P_r/P_t = G_t·G_r·(λ/4πd)² assembles term by term]** [CLIP c2 start]
So where does the path-loss number come from? Over a clear line of sight, from a formula published by Harald Friis in nineteen forty-six. The Friis transmission equation: received power over transmitted power equals both antenna gains, times wavelength over four pi d, squared. Two consequences deserve emphasis. The inverse square in distance means doubling the range costs six decibels. And the wavelength-squared factor means that for antennas of *fixed gain*, a longer wavelength — a lower frequency — delivers more received power. The low band travels "better" not because space treats it differently, but because a fixed-gain antenna has a larger physical aperture at longer wavelengths. [CLIP c2 end]

**[V: ideal-vs-real — the Friis prediction drawn as an optimistic ceiling line; correction terms stack beneath it: feedline, polarization, atmosphere, fading]**
Friis is an idealization — it assumes a clear line of sight, far-field distances, matched polarization, lossless antennas, no obstruction or multipath. Real links add every correction as extra decibels in the budget. But the bare equation still sets the ceiling: no terrestrial path beats free space, so Friis is the optimistic bound every real measurement gets compared against.

**[V: two-directions — the same budget run forward ("what coverage do I get?") and backward ("how much gain for 10 dB of margin at this range?"); the whole thing fits one spreadsheet row]**
Because every term is a straight addition once it's in decibels, a whole link budget fits on one line of a spreadsheet — and engineers run it in both directions. Forward, to predict coverage from known equipment. Backward, to solve for a missing requirement: how much antenna gain or transmit power do I need for ten decibels of margin at this range? That one line of arithmetic is how every radio link on Earth gets designed.

**[V: gt-tie-in — a receive-side budget card: published site ERP → −path loss → +RX antenna gain → +LNA → compared against noise floor + required SNR; a "how many dB short?" readout]** [CLIP c3 start]
For a receive-only SDR running GopherTrunk, the link budget explains what you can and cannot hear. Start from a transmitter's published radiated power and the distance, subtract free-space and terrain path loss, add your receive antenna gain and any low-noise amplifier, and compare against your effective sensitivity — the noise floor plus the SNR the demodulator needs. GopherTrunk doesn't compute link budgets internally; it decodes whatever reaches its input. But a negative margin is a quantitative diagnosis: it tells you exactly how many decibels short you are, and whether the fix is more gain, a lower-loss feedline, a quieter front end, or a better location. [CLIP c3 end]

**[V: recap | "Link budget" | ① Add gains, subtract losses — all in dB | ② Margin = P_rx − sensitivity; keep it positive, plus fade margin | ③ Friis sets the free-space ceiling: −6 dB per doubling of range]**
So: a link budget adds every gain and subtracts every loss in decibels, the margin above sensitivity decides whether the link lives, and the Friis equation sets the free-space ceiling no real path can beat. Full write-ups on both are linked below.
