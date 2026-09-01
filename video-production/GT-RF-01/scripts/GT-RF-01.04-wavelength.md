# GT-RF-01.04 — Wavelength
slug: wavelength · type: term · treatment: 2.1 term · target: 3:30–4:30 · words: 567

**[V: Title card — "Wavelength"]**

**[V: silhouettes to scale: a rooftop antenna about a metre tall … then a laptop with a hidden antenna inside its lid; a "why?" tag between them]** [CLIP c1 starts]
Wavelength is why some antennas are the size of a person and others hide inside a laptop lid. It's the physical distance a wave covers in one complete cycle — the gap between one crest and the next — written with the Greek letter lambda. And it's inversely proportional to frequency: higher frequency, shorter wavelength. If frequency counts cycles in time, wavelength measures one cycle laid out in space — the spatial twin of frequency's temporal count. [CLIP c1 ends]

**[V: analogy — a walker's strides: fast short steps vs slow long strides covering ground at the same speed]**
Anchor it with a walk. Two people move down a street at exactly the same speed — but one takes quick, short steps and the other slow, long strides. Steps per second is frequency; the length of each step is wavelength. Because the travel speed is fixed, more steps per second forces each step to be shorter. Radio waves all walk at one speed — the speed of light — so the same bargain holds.

**[V: CORE ANIMATION — the article's figure builds: sine wave draws, then the double-arrow bracket "λ = one cycle" between crests, then the caption line "λ (m) ≈ 300 ÷ frequency (MHz)"]**
That bargain is one clean formula: lambda equals c over f — wavelength is the speed of light divided by frequency. Plug in c, about three times ten-to-the-eighth metres per second, and you get the shortcut every radio operator memorizes: wavelength in metres is roughly 300 divided by the frequency in megahertz. One division, done in your head.

**[V: number cards stack in: "150 MHz → ~2 m" · "460 MHz → ~0.65 m" · "1 GHz → ~0.3 m" · "2.4 GHz Wi-Fi → ~12.5 cm"]** [CLIP c2 starts]
Try it. A 150-megahertz VHF signal: 300 over 150 — about two metres long. A 460-megahertz UHF signal: about sixty-five centimetres. One gigahertz: thirty centimetres. And Wi-Fi at 2.4 gigahertz is down around twelve and a half centimetres — which is exactly why its antenna fits inside a laptop lid while a two-metre-band antenna stands roughly a metre tall. [CLIP c2 ends] Zoom out and the range is staggering: as you climb the spectrum, wavelengths shrink from kilometres at the low-frequency end to millimetres at the top of the microwave range — the origin of the term "millimetre wave."

**[V: dipole graphic: full wave above, a λ/2 dipole and a λ/4 whip drawn to scale beneath it]**
Why does the antenna care? Because an efficient antenna is resonant when its length is a simple fraction of the wavelength: a half-wave dipole is lambda over two, a quarter-wave whip is lambda over four over a ground plane. Get the fraction wrong and the antenna presents a poor impedance match — reflecting power instead of radiating or capturing it. Wavelength also sets how waves negotiate the world: they bend readily around objects small compared to lambda, but cast sharp shadows behind objects much larger than it. Long-wavelength HF hugs terrain and refracts off the ionosphere; short-wavelength UHF behaves more like light — line-of-sight, with pronounced multipath.

**[V: photo-style card: a repeater antenna and a radar dish side by side, each tagged with an estimated λ]**
Here's a field skill hiding in this: the size of a repeater's antenna or a radar dish is a quick visual estimate of its operating wavelength — and therefore its band. Hardware wears its wavelength on the outside.

**[V: GopherTrunk dark-theme UI: signal meter in dBFS, healthy level; cut to the same meter starved low with a "wrong antenna" annotation]**
Wavelength never appears in GopherTrunk's sample stream — the software works in frequency and time — but it governs everything in front of the converter. The VHF and UHF land-mobile bands GopherTrunk targets run from about two metres down to thirty centimetres of wavelength, which is why discone and vertical whip antennas of that scale are the usual choice. Pick an antenna mismatched to your target and the signal meter in dBFS tells the story: a starved receiver that no amount of DSP can rescue.

**[V: Recap card: "① λ = distance of one cycle ② λ (m) ≈ 300 ÷ f (MHz) ③ antennas resonate at fractions of λ"]**
So: wavelength is one cycle measured in metres, it's 300 over the frequency in megahertz, and it's the number that physically sizes every antenna you'll ever use. Full write-up linked below.

## Clip picks
- c1: "why some antennas are the size of a person" hook + definition (~35s)
- c2: the 300 ÷ MHz mental-math stack (150 MHz → 2 m … Wi-Fi → 12.5 cm) (~30s)
- c3 (optional): "hardware wears its wavelength on the outside" field-skill beat (~20s)

## Vertical plan
Direct center crop; number-card stack and dipole graphic built center-safe. Re-hook first 2 s with the person-vs-laptop antenna visual.

## Assets
- Core animation: rebuild of the article's λ-bracket sine SVG including its "λ (m) ≈ 300 ÷ frequency (MHz)" rule-of-thumb line
- Stride-length analogy animation (two walkers, same speed)
- Number cards (150 MHz / 460 MHz / 1 GHz / 2.4 GHz conversions)
- Dipole/whip scale graphic (λ/2, λ/4)
- Antenna/dish visual-estimate card
- GopherTrunk screen capture: signal meter in dBFS (matched vs starved), dark theme
- Title + recap cards from templates

## Checklist deltas
- [ ] Analogy actually maps (fixed walking speed = fixed c; steps/second × step length = speed holds exactly)
- [ ] Formula card matches the article's notation exactly (λ = c / f; the ≈300/MHz rule as the article writes it)
