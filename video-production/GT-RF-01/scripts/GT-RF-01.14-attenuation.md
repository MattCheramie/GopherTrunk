# GT-RF-01.14 — Attenuation
slug: attenuation · type: term · treatment: 2.1 term · target: 3:30–4:30 · words: 565

**[V: Title card — "Attenuation"]**

**[V: the article's shrinking-sine figure: a wave leaves "transmitter" tall and arrives at "weaker at receiver" tiny; a running dB counter ticks down along the way]** [CLIP c1 starts]
Attenuation is where your signal goes to die — a fraction of a dB at a time. It's the reduction of signal strength as energy passes through a medium, a cable, a connector, or an obstacle, and it's expressed in decibels because losses in dB simply subtract from a power budget. The numbers bite fast: a loss of just six dB means only a quarter of the power survives.

**[V: split card: "spreading — same power, bigger area" (ripples widening) · "absorption — RF becomes heat" (wave fading inside a wall)]**
There are only two mechanisms. Spreading: a fixed amount of power gets diluted over a larger area, like ripples widening on a pond. And absorption: the medium converts RF energy into heat, like a shout muffled inside a wall. Everything that eats your signal is one of those two — and because each loss is logged in dB, the total is just a running sum down the chain. [CLIP c1 ends]

**[V: CORE ANIMATION — a receive chain builds left to right: antenna → coax → connectors → filter → receiver; each element stamps its loss: "coax: several dB per 10 m @ 400 MHz" · "each connector: fractions of a dB" · a wall icon and a rain cloud above]**
Walk the losses in a real receive path. Cable first: coax loss rises with both frequency and length — a cheap RG-58 run can lose several dB per ten metres at four hundred megahertz, which is why low-loss cable and short runs matter. Then every connector, filter, splitter, and switch adds its own insertion loss, a fraction of a dB to a few dB each — and they accumulate. Out in the world, walls, foliage, and terrain absorb and scatter energy, and loss through material generally worsens at higher frequencies — the reason UHF penetrates buildings less well than VHF. And way up above ten gigahertz, the atmosphere itself joins in: oxygen and water vapour absorb RF and rain adds "rain fade" — though below UHF that's negligible.

**[V: card: "the recurring theme: higher frequency → more attenuation"; a second card: "loss BEFORE the first amplifier counts twice"]** [CLIP c2]
Two rules fall out of that list. First, the recurring theme: for most of these mechanisms, higher frequencies attenuate more — that single fact shapes band choice, cable choice, and site planning. Second, the sneaky one: a lossy cable ahead of the first amplifier is doubly harmful, because it doesn't just weaken the signal — it degrades the noise figure of the whole receiver. Loss before the first amp counts twice. That's why the preamp belongs at the antenna, not at the desk.

**[V: an "attenuator" block deliberately inserted before a receiver; a nearby-transmitter spike shrinks below a "full scale" ceiling line]**
Now the twist: attenuation is not always the enemy. A deliberate attenuator protects a receiver from strong signals that would otherwise overload the front end and drive the converter past full scale, spraying intermodulation products across the band. Trading a few dB of wanted signal for headroom against a nearby transmitter is often a net win — and filters attenuate out-of-band energy for the same protective reason.

**[V: GopherTrunk dark-theme UI: dBFS signal meter; annotation: "what survives the cable is all GT ever sees"]**
One scope note: free-space spreading over the air is a kind of attenuation important enough to have its own name — path loss — while the feedline losses we've just walked are the part of the budget you actually control. And for GopherTrunk, the arithmetic is unforgiving: it sees only what survives the path and the cable. Attenuation upstream of the SDR is loss the software cannot recover — no DSP setting refunds dB spent in the coax. In a link budget, every one of these losses is a negative term, and knowing where the dB are going tells you where to spend effort: usually a shorter cable run, and a preamp at the antenna.

**[V: Recap card: "① loss in dB, it all adds: −3 = ½, −6 = ¼ ② loss before the first amp counts twice ③ deliberate attenuation = headroom"]**
So: attenuation is loss in decibels — spreading and absorption, adding up connector by connector — it hurts most before the first amplifier, and used deliberately, it's how you buy headroom. Full write-up linked below.

## Clip picks
- c1: the shrinking-wave hook + "only two mechanisms" (~35s)
- c2: "loss before the first amp counts twice" — the install-changing rule (~25s)
- c3 (optional): the attenuator-as-a-tool twist (~25s)

## Vertical plan
The receive-chain animation is wide — prepare a tall re-layout (chain runs top-to-bottom for 9:16). Split cards stack vertically. Re-hook first 2 s with the ticking dB counter over the shrinking wave.

## Assets
- Core animation: rebuild of the article's shrinking-sine SVG (with its −6 dB = ×¼ · −3 dB = ×½ footer), extended into the loss-stamped receive chain
- Spreading-vs-absorption split card
- Two rules cards (frequency theme; first-amp rule)
- Attenuator/headroom animation with full-scale ceiling line
- GopherTrunk screen capture: dBFS signal meter, dark theme
- Title + recap cards from templates

## Checklist deltas
- [ ] Analogy actually maps (ripples = spreading, muffled shout = absorption; neither implies the energy "runs out" abruptly)
- [ ] Loss figures match the article exactly (−6 dB = ¼ power; RG-58 several dB per 10 m at 400 MHz; atmospheric absorption above ~10 GHz, negligible below UHF)
- [ ] Deliberate-attenuator framing mirrors the article (protection/headroom — not a workaround for gain staging)
