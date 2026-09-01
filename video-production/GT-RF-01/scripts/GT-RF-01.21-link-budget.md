# GT-RF-01.21 — Link budget
slug: link-budget · type: concept · treatment: 2.2 concept + 2.4 demo beat (pillar's ~90% payoff demo) · target: 3:30–4:30 · words: 690

**[V: Title card — "Link budget"]**

**[V: a map: a transmitter tower far across the county, a small rooftop antenna at the other edge, a long uncertain line between them]** [CLIP c1 starts]
A link budget is the answer to the question every radio listener eventually asks: that transmitter is way across the county — will I actually hear it? You could guess. You could drive around with an antenna and find out the hard way. Or you could do what every radio engineer on Earth does: follow the signal from the transmitter's amplifier all the way to your receiver's input, writing down every single gain and every single loss along the way — and know the answer before you plug anything in. [CLIP c1 ends]

**[V: chain skeleton fades in: TX → feedline → antenna → path → antenna → RX, with empty dB boxes over each hop]**
The trick that makes this bookkeeping possible is the decibel — a logarithmic ratio scale on which every multiplication becomes simple addition and plus three dB means doubled power. So the whole journey — amplifiers multiplying, cables eating fractions, the path dividing the signal by billions — collapses into one running sum of small numbers. A link budget is exactly that sum, and it fits on one line of a spreadsheet.

**[V: staircase chart builds left to right: P_tx +40 dBm, step down −2 dB cable, step up +9 dB antenna, landing on "EIRP 47 dBm"]**
Let's walk one. The transmitter makes plus forty dBm — ten watts. Its feedline and connectors eat two dB on the climb up the tower. The antenna adds nine dB of gain by focusing the energy toward the horizon. Forty, minus two, plus nine: the signal leaves the site at an effective radiated power of forty-seven dBm. That launch figure has a name — EIRP — and it's the first line of every budget ever written.

**[V: the staircase plunges off a cliff labeled "− path loss", dwarfing every previous step]** [CLIP c2 starts]
Then comes the path — and the path is brutal. Spreading out through space, the signal sheds more decibels than everything else in the chain combined; over a clear line of sight that loss is computed from just distance and frequency by the Friis transmission equation, and terrain, foliage, rain, and polarization mismatch each pile their own subtractions on top. On the staircase chart, the path isn't a step. It's the cliff.

**[V: a small +G_rx step back up; a dashed line labeled "P_rx" hovers above a solid line labeled "sensitivity floor"; the gap between them highlights as "margin"]**
At your end, you claw a little back: your antenna's gain, minus your own feedline loss. Where the staircase finally lands is the predicted received power. Now comes the verdict. Compare it against your receiver's sensitivity — the minimum power it needs for an acceptable error rate — and the surplus is the link margin. Comfortably positive: the link closes. Thin or negative: unreliable. And a robust design never aims for zero — it reserves a fade margin on top, so that fading, multipath, and weather can't drag the link below threshold on a bad day. [CLIP c2 ends]

**[V: the budget line runs in reverse; a card: "need 10 dB of margin at this range — solve for antenna gain"]**
What makes the budget more than arithmetic is that it runs in both directions. Forward, it predicts coverage from equipment you already have. Backward, it solves for the missing piece: how much antenna gain, or how much transmit power, do I need for ten dB of margin at this range? Every term is a straight addition, so the design question and its answer live on the same line.

**[V: LIVE DEMO — GopherTrunk web UI, dark theme: tuned to a real control channel; annotate the dBFS meter "what survived the trip", the SNR readout "height above the noise", the decode counter ticking steadily]** [CLIP c3 starts]
And here's the bottom line of a budget, live. This is GopherTrunk locked to a real control channel. The signal on this waterfall left a transmitter at full power, lost almost all of itself to the path, and arrived here: the dBFS meter shows exactly what's left, and the SNR readout is the margin story in real time — how far the survivor stands above my noise. Because it clears what the demodulator needs, the decode counter just keeps ticking. Every one of those frames is a link budget that closed. [CLIP c3 ends]

**[V: card: a negative margin "−6 dB" beside a fixes list — receive antenna gain · lower-loss feedline · quieter front end · better location]**
And when a channel *doesn't* decode, the budget is your diagnosis. A negative margin isn't "bad reception" — it's a number: exactly how many dB short you are. Each fix has a dB value of its own — a better antenna, a lower-loss feedline, a quieter front end, a better location — so you can pick the cheapest combination that covers the gap before spending a cent.

**[V: Recap card: "① add every gain, subtract every loss, in dB ② margin = P_rx − sensitivity, plus a fade reserve ③ a failed link is a number — buy back the missing dB"]**
So: a link budget follows a signal from transmitter to receiver, adding every gain and subtracting every loss in decibels; received power minus sensitivity is your margin, with a fade reserve on top; and when a channel won't decode, the budget tells you how many dB you're short — and which fix buys them back. Full write-up linked below.

## Clip picks
- **c1** (~30 s): "will I actually hear it?" problem hook — the map + know-before-you-plug-in promise.
- **c2** (~40 s): the staircase walk from the path-loss cliff to the margin verdict — the concept in one continuous visual.
- **c3** (~35 s): the live GopherTrunk demo — "every one of those frames is a link budget that closed." The flagship short for the pillar's payoff chapter.

## Vertical plan
The staircase chart is the segment's spine and is inherently wide — build a tall variant from the start (budget terms descending top-to-bottom like a ledger, which arguably reads *better* in 9:16). Demo UI needs zoom-crop pans: meter → SNR → decode counter as three sequential close-ups. Re-hook: burned text "Will you hear that tower? Do the math first" over the map. Budget +1 h for the re-layout.

## Assets
- Title + recap cards (templates)
- **Core animation:** the dB staircase, rebuilt and extended from the article's inline SVG (P_tx / +G_tx / −path / +G_rx / P_rx over the sensitivity floor, margin bracket)
- Map/tower-to-rooftop establishing graphic
- Cards: reverse-budget "solve for gain" card · negative-margin fixes card
- **Live GopherTrunk capture** (the demo beat): real control channel decode with dBFS meter, SNR readout, decode counters — 4K, panels zoomed ≥150 %, annotation overlays in accent color
- VO track; music bed with a slight lift into the demo beat

## Checklist deltas
- [ ] Problem stated before the solution; the term "link budget" spoken in the first sentence (concept rules)
- [ ] Exactly ONE sentence restating the dB idea, ONE naming Friis — no other cross-segment context (per pillar plan; Friis folded in as one restated sentence)
- [ ] Chain numbers are article-sourced only (the 40 − 2 + 9 = 47 dBm chain; "10 dB of margin" example); path loss stays qualitative — no invented dB figure for it
- [ ] Core relation matches article: P_rx = EIRP − L_path − L_misc + G_rx; margin = P_rx − sensitivity (one formula on screen at a time)
- [ ] Demo is a real decode — never staged; capture provenance logged
- [ ] Fade margin named and distinguished from plain margin (article's robust-design point)
