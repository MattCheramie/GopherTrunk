# GT-RF-01.09 — Modulation
slug: modulation · type: concept · treatment: 2.2 concept (problem-story open) · target: 3:30–4:30 · words: 566

**[V: Title card — "Modulation"]**

**[V: a voice waveform on the left, a steady carrier on the right, a gap between them; the voice "pushes" at the gap and nothing crosses]** [CLIP c1 starts]
Modulation is the answer to radio's founding problem: how do you make a wave carry a voice? Because here's the bind. The wave you can actually radiate — a radio-frequency carrier — is a steady tone that says nothing. And the thing you want to send — audio, data — is a baseband signal that can't be radiated at all. Two signals, and neither one can do the job alone.

**[V: the carrier morphs three ways in sequence: taller/shorter · cycles squeezing/stretching · cycles nudging early/late — labels "amplitude", "frequency", "phase"]**
The way across the gap is this: there are only three things about a sinusoidal carrier you can change. Its amplitude — how tall it is. Its frequency — how fast it cycles. Its phase — the timing of those cycles. Modulation is varying one or more of them in step with the message, and every scheme ever invented is a way of doing exactly that. The reverse process at the receiver is demodulation. [CLIP c1 ends]

**[V: CORE ANIMATION — the article's three-trace figure builds top to bottom: "message" wave, then "AM" whose height follows the message, then "FM" whose cycle spacing follows it]**
Watch the classics. AM — amplitude modulation — writes the message into the carrier's envelope: the wave's height traces the voice. FM shifts the carrier's frequency with the message instead — and because its envelope stays constant, amplitude noise mostly bounces off, which is why FM sounds cleaner than AM. Phase modulation nudges the carrier's timing; it's closely related to FM, and it underlies most digital schemes.

**[V: the smooth message snaps into discrete steps; card: "symbol = one of a finite set of states = one or more bits"]** [CLIP c2 starts]
Analog modulation varies a property continuously. Digital modulation makes one sharp break with that: it switches the carrier among a finite set of symbols, each standing for one or more bits. FSK toggles between discrete frequencies. PSK toggles between discrete phases. QAM picks from combinations of amplitude and phase at once.

**[V: constellation diagram builds on the IQ plane: 4 points appear, then 16; noise fuzz grows around the points until neighbours blur]**
Those symbols are naturally drawn on a constellation diagram — the map of the IQ plane the receiver uses to decide which symbol arrived. And the constellation makes radio's central bargain visible. Pack more bits per symbol — more constellation points — and spectral efficiency rises. But the points crowd closer together, so noise blurs them into each other sooner: it takes more signal-to-noise to keep them distinct. That's the fundamental power-versus-rate trade, and Shannon capacity puts a hard ceiling on it. Choosing a modulation scheme is a trade among data rate, bandwidth, power, and robustness — there is no free lunch anywhere on the map. [CLIP c2 ends]

**[V: two-column card: "weak / fading link → four-level FSK · π/4-DQPSK" vs "clean channel → 16/64/256-QAM over OFDM"]**
So real systems pick their point on the trade deliberately. Land-mobile digital voice has to survive weak, fading links, so it runs low, robust orders: C4FM four-level FSK and π/4-DQPSK, precisely because they hold up at modest SNR — with pulse shaping keeping each symbol from smearing into its neighbours. High-throughput systems — LTE, Wi-Fi, DVB — climb to 16-, 64-, or 256-QAM over OFDM when the channel is good, and adapt downward when it degrades.

**[V: GopherTrunk dark-theme UI: constellation/symbol view on a live channel; caption "four-level FSK slicing (DMR/NXDN) · π/4-DQPSK tracking (P25)"]**
Recognising a signal's modulation and applying the matching demodulator is the heart of everything GopherTrunk does. Its chain identifies the trunking waveform, then runs the appropriate symbol recovery — four-level FSK slicing for DMR and NXDN, π/4-DQPSK carrier-and-symbol tracking for P25. And notice the loop closing: the three carrier properties reappear as the axes of the IQ plane, so once the samples are in software, demodulation is just measuring amplitude, frequency, and phase — and mapping them back to symbols and bits.

**[V: Recap card: "① only 3 knobs: amplitude · frequency · phase ② digital = a finite alphabet on the constellation ③ more bits per symbol costs SNR"]**
So: modulation is the bridge between message and carrier — three properties to vary, analog varies them smoothly, digital snaps them to an alphabet of symbols, and every extra bit per symbol is bought with signal-to-noise. Full write-up linked below.

## Clip picks
- c1: the problem-story open + "only three things you can change" reveal (~40s)
- c2: constellation crowding — the bits-per-symbol vs SNR trade made visible (~35s)
- c3 (optional): AM/FM three-trace build ("FM sounds cleaner because its envelope stays constant") (~25s)

## Vertical plan
The article's three-trace message/AM/FM figure stacks vertically — crops as-is. The constellation is square and centre-framed — crops as-is. The two-column trade card needs a stacked tall re-layout (flag it). Re-hook first 2 s with the voice-vs-carrier gap visual.

## Assets
- Core animation: rebuild of the article's message/AM/FM three-trace SVG, built trace by trace synced to VO
- Three-knob morph animation (amplitude/frequency/phase on one carrier)
- Constellation build animation (4 → 16 points + noise fuzz) on the IQ plane
- Trade card (robust low-order vs high-order QAM/OFDM) + symbol definition card
- GopherTrunk screen capture: constellation/symbol view on a live channel, dark theme
- Title + recap cards from templates

## Checklist deltas
- [ ] Problem is stated before the solution (voice can't radiate; carrier says nothing — then the three knobs)
- [ ] The concept's name is spoken within the first 20 s (it's word one)
- [ ] Constellation/SNR trade wording matches the article (more points → less spacing → more SNR; Shannon bounds it)
- [ ] Load-bearing chapter: transition beats in the pillar may point back here — keep this segment fully self-contained anyway
