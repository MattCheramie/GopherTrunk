# GT-RF-01.22 — Shannon capacity
slug: shannon-capacity · type: concept · treatment: 2.2 concept (pillar finale) · target: 3:30–4:30 · words: 633

**[V: Title card — "Shannon capacity"]**

**[V: a road sign–styled card: "SPEED LIMIT" over a channel; beneath it, "no exceptions — ever"]** [CLIP c1 starts]
Shannon capacity is the speed limit — the one number no radio, no protocol, and no clever engineer will ever beat. Before 1948, engineers assumed noise simply meant errors: push data faster, get more of them, forever. Then Claude Shannon proved something nobody expected. Every noisy channel has an exact capacity — a rate below which communication can be made as error-free as you like, and above which it cannot. Not difficult above the line. Impossible. [CLIP c1 ends]

**[V: a channel drawn as a pipe: its width labeled "bandwidth B", the signal's height above a noise haze labeled "SNR"]**
The whole result comes from two resources. First, bandwidth — how wide the channel is, in hertz — sets how many independent symbols per second it can carry. Second, signal-to-noise ratio — how far the signal stands above the noise — sets how many distinguishable levels each symbol can reliably hold, and therefore how many bits each one encodes. Width, times bits per symbol. That is everything a channel has to give.

**[V: formula card: "C = B · log₂(1 + SNR)  bits/s", each term lighting as it's spoken]**
Written down, the ceiling is one line: capacity C equals bandwidth B, times the log base two of one plus the signal-to-noise ratio — bits per second, from two measurable numbers. Divide through by bandwidth and you get spectral efficiency: the ceiling on how many bits each hertz of spectrum can ever deliver.

**[V: article's capacity-vs-SNR curve animates; annotations appear: "near 0 dB: ~1 bit/s/Hz" and "+3 dB SNR ≈ +1 bit/s/Hz"]** [CLIP c2 starts]
The shape of that curve explains half of radio engineering. At high signal-to-noise, capacity grows only logarithmically: each extra three dB — a doubling of signal power — buys roughly one more bit per second per hertz. Brute-force power hits brutally diminishing returns. At low signal-to-noise, the curve is nearly linear — so you can trade bandwidth for power, spreading a weak signal across more hertz and still getting the bits through. That trade is the principle behind spread-spectrum and deep-space links. And there is an absolute floor: below about minus one point six dB of energy-per-bit against noise density, reliable communication is impossible at *any* bandwidth. [CLIP c2 ends]

**[V: card sequence: "1948 — capacity proved achievable… with no recipe" → "today: turbo and LDPC codes, within a fraction of a dB of the limit"]**
Here's the strangest part of the story. Shannon proved capacity is *achievable* — codes exist that approach it with vanishing error — but his proof was non-constructive: it offered no recipe for building them. The gap between the proven limit and working hardware is what decades of error-correction research have been closing, and today's best codes — turbo codes, LDPC — operate within a fraction of a decibel of the line Shannon drew in 1948.

**[V: the capacity curve with several dots placed on it, each labeled generically "modulation + code rate"]** [CLIP c3 starts]
Why this matters to a listener: every modulation-and-coding scheme in every modern standard is, in effect, a chosen point on Shannon's curve — a trade of spectral efficiency against robustness. And the theorem cuts both ways. If the signal-to-noise at your antenna sits below what a waveform's chosen point requires, no receiver cleverness will recover the data — the transmitter already sent at a rate your channel cannot sustain at your noise level. The only honest lever is the ratio itself: a better antenna, a quieter front end, less noise. [CLIP c3 ends]

**[V: GopherTrunk web UI, dark theme: live decode with the SNR readout beside a climbing decode counter; then a weaker channel where the counter stalls]**
GopherTrunk never computes capacity, but its entire decode chain lives under this theorem. When the SNR readout sits above what a waveform's coding and modulation require, frames decode and the counters climb. When it sits below, they don't — and no setting changes that, because the boundary is exactly the line Shannon's formula draws. Watch the SNR number next to a live decode, and you are watching the theorem enforce itself.

**[V: Recap card: "① C = B·log₂(1+SNR) — a hard ceiling ② high SNR: +3 dB ≈ +1 bit/s/Hz ③ below the required SNR, no decoder can save you"]**
So this is where every radio conversation ends: bandwidth and signal-to-noise, multiplied into a hard ceiling that no code or modulation exceeds. Capacity equals bandwidth times log two of one plus SNR; at high SNR each three dB buys about one more bit per hertz; and below the SNR a signal requires, no decoder can save it. That ceiling is the wall every protocol in this course is climbing toward — and everything a radio does well is a way of getting closer to it. Full write-up linked below.

## Clip picks
- **c1** (~30 s): the speed-limit hook — "not difficult. Impossible." The strongest cold open in the pillar.
- **c2** (~40 s): the curve's two regimes + the −1.6 dB floor — the numbers beat, dense but self-contained.
- **c3** (~30 s): "no receiver cleverness will recover it" — the contrarian, myth-busting beat. Shorts title: "Why a better decoder can't fix a weak signal".

## Vertical plan
Direct center crop for cards and pipe diagram. The capacity-vs-SNR curve is wide — prepare a tall re-layout (SNR axis vertical, capacity horizontal, annotations restacked). The dots-on-the-curve visual reuses the same tall layout. Re-hook: burned text "The speed limit no radio can break" over the road-sign card.

## Assets
- Title + recap cards (templates)
- **Core animation:** rebuilt from the article's inline SVG — the capacity-vs-SNR curve with its two annotations ("near 0 dB: ~1 bit/s/Hz", "+3 dB ≈ +1 bit/s/Hz"); extended with the points-on-the-curve overlay
- Pipe diagram (bandwidth = width, SNR = height above noise haze)
- Cards: the C = B·log₂(1+SNR) formula card · 1948/no-recipe → turbo/LDPC pair · −1.6 dB Eb/N0 floor (one at a time)
- GopherTrunk screen capture: SNR readout + decode counter, healthy channel then stalling channel (10–20 s, dark theme)
- VO track; music bed with a finale arc (this segment closes the pillar's Act III)

## Checklist deltas
- [ ] Problem stated before the solution (pre-1948 "noise means errors forever" assumption) and the term spoken in the first sentence
- [ ] Formula card matches article notation exactly: C = B·log₂(1 + SNR), SNR linear — and it is the only formula on screen at any moment
- [ ] All numbers article-sourced: +3 dB ≈ +1 bit/s/Hz at high SNR, ~1 bit/s/Hz near 0 dB, Eb/N0 ≈ −1.6 dB, "fraction of a dB" for turbo/LDPC, 1948
- [ ] Arc-closing line ("the wall every protocol in this course is climbing toward") names no other segment or chapter
- [ ] Achievability nuance kept honest: proved achievable, proof non-constructive — no claim that Shannon designed the codes
- [ ] GT tie-in claims only what the article does: GopherTrunk does not compute capacity; the SNR-vs-decode boundary is the illustration
