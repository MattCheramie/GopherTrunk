# GT-RF-01.12 — Noise floor
slug: noise-floor · type: term · treatment: 2.1 term + waterfall demo beat · target: 3:30–4:30 · words: 596

**[V: Title card — "Noise floor"]**

**[V: a silent, empty spectrum display... that isn't silent: a fuzzy, wandering baseline hisses along the bottom]** [CLIP c1 starts]
The noise floor is what your receiver hears when there's nothing to hear. Tune to an empty channel and the display doesn't go to zero — there's a constant background level of random energy, thermal noise in the electronics plus environmental RF picked up by the antenna. That level is the noise floor. It's measured in dBm, and it sets the bar every signal must clear to be usable.

**[V: analogy — a party room: a murmur meter pinned at "crowd hum"; a whisper fails to register, a raised voice pops above it]**
The everyday anchor is a crowded room. The room is never silent — there's always a hum of conversation — and whether you hear someone depends not on how loudly they speak in absolute terms, but on whether they rise above that hum. Radio is the same: the floor is the hum, and "can I decode it" means "does it stand above the floor." [CLIP c1 ends]

**[V: card builds: "kTB" → "−174 dBm per hertz at room temperature" → "+10·log₁₀(B)" → "12.5 kHz channel ≈ −121 dBm"]**
The floor has a hard physical bottom. Any resistor — including an antenna's radiation resistance — at temperature T generates thermal noise power k-T-B: Boltzmann's constant, times temperature, times bandwidth. At room temperature that works out to about minus one hundred seventy-four dBm per hertz. Multiply by your bandwidth — add ten log B in dB — and you get the floor for a perfect receiver: roughly minus one twenty-one dBm in a twelve-and-a-half kilohertz channel, minus one fourteen in a hundred-kilohertz span. No receiver can hear below this. Physics says so.

**[V: CORE ANIMATION — the article's wandering-baseline figure; two layers stack on top of it: "+ noise figure" lifts the dashed line, "+ environment" lifts it again with icons: power brick, USB, LED bulb, solar inverter]**
Two things push the real floor above that ideal. First, the receiver adds its own noise — its noise figure: a five-dB noise figure raises the floor five dB above the thermal limit, which is why a low-noise amplifier placed first in the chain matters so much — it sets the noise figure of everything after it. Second, the environment contributes: atmospheric and galactic noise, and above all man-made interference — switching power supplies, USB buses, LED lighting, Ethernet, solar inverters. In a noisy urban install, the environment, not the receiver, dominates the floor.

**[V: a filter bracket narrows around a thin signal; the floor inside the bracket steps down 3 dB as the width halves]**
And here's the part people miss: the floor moves with the measurement. Thermal noise scales with bandwidth, so halving the receive bandwidth lowers the noise power by three dB while a narrowband signal inside stays intact. That's exactly why narrow channel filters and matched filtering improve signal-to-noise — and why a wideband capture always shows a higher floor: it's integrating noise over more spectrum.

**[V: DEMO — GopherTrunk dark-theme waterfall, ~25 s: the speckled "grass" along the bottom annotated "this is the noise floor"; a signal stands above it; then the antenna is pulled and the grass level visibly drops several dB]** [CLIP c2]
You've been looking at the noise floor all along. In GopherTrunk's waterfall, it's the grass — that speckled, ever-present texture across the bottom of the display. Every signal you'll ever decode is a shape standing above the grass. And the grass hands you a free diagnostic: disconnect the antenna. If the displayed floor drops several dB, environmental noise coming in through the antenna is dominating — and the fix is at the install, not in the software.

**[V: checklist card: "LNA at the antenna · chokes + shielded cable · SDR away from computers · filter strong out-of-band · match the bandwidth"]**
That makes lowering the floor the cheapest way to hear more. Put a low-noise preamp at the antenna, before cable loss degrades the noise figure. Cut environmental pickup — ferrite chokes, quality shielded cable, moving the SDR away from computers and chargers can drop the floor ten dB or more. Filter out strong out-of-band signals. And match the receive bandwidth to the signal, so noise isn't admitted needlessly. GopherTrunk cannot decode a signal sitting at or below the floor — getting the floor down is the highest-leverage move an operator has before touching any DSP setting.

**[V: Recap card: "① floor = thermal kTB + noise figure + environment ② ~−121 dBm in a 12.5 kHz channel ③ the grass on the waterfall — lower it at the install"]**
So: the noise floor is the ever-present hum — thermal physics at the bottom, receiver and environment stacked on top — it's the grass on your waterfall, and it's fixable at the antenna, not in software. Full write-up linked below.

## Clip picks
- c1: "what your receiver hears when there's nothing to hear" hook + party-hum analogy (~35s)
- c2: the waterfall grass demo + pull-the-antenna diagnostic — the shareable trick (~30s)
- c3 (optional): "−174 dBm per hertz — physics says so" numbers beat (~25s)

## Vertical plan
Direct center crop for cards. The waterfall demo needs a zoom-crop pan for 9:16 (grass band + annotation must stay readable at phone size). Re-hook first 2 s with the "empty channel that isn't silent" visual.

## Assets
- Core animation: rebuild of the article's wandering-baseline SVG, extended with stacking "+noise figure" / "+environment" layers
- kTB numbers card (−174 dBm/Hz → −121 dBm @ 12.5 kHz → −114 dBm @ 100 kHz)
- Party-room analogy graphic; bandwidth-halving/3 dB filter animation
- Install checklist card
- GopherTrunk screen capture, ~25 s: waterfall with grass annotated + antenna-disconnect floor drop, dark theme (capture the drop live — never fake it)
- Title + recap cards from templates

## Checklist deltas
- [ ] Analogy actually maps (crowd hum = additive background; whisper audibility = margin above it)
- [ ] Numbers card matches the article exactly (−174 dBm/Hz, ≈−121 dBm @ 12.5 kHz, −114 dBm @ 100 kHz, 3 dB per halving, 10 dB install win)
- [ ] Demo beat is a real capture: the antenna-disconnect floor drop recorded off real hardware, 20–30 s in the edit
