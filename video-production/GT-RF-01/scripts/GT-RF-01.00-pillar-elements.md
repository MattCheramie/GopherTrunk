# GT-RF-01.00 — Pillar-only elements
slug: (pillar) · Cold open · Course intro · 21 transition beats · Midpoint recap · Outro

Per playbook §4.2. These are the ONLY places "as we just saw / up next" language
is allowed. Batch-record all transition beats in ONE session after the running
order is locked (it is — see `../GT-RF-01-plan.md`). Every beat plays over the
**map card** (`_brand/gen_cards.py map_card`, node states per beat).

---

## Cold open (45–60 s) — before everything

**[V: GopherTrunk waterfall, a carrier spike standing tall out of the noise
grass; SNR bracket appears — pulled from GT-RF-01.13's demo beat]**
Somewhere above the noise on this screen, there's a voice.

**[V: link-budget chain animating: +37 dBm … −120 dB … a signal arriving —
pulled from GT-RF-01.21]**
In the next hour and a half you'll follow a radio signal from the transmitter
that made it, across miles of empty air where it loses ninety-nine point
nine-nine-nine — and eleven more nines — percent of its power, into an antenna,
down a cable, and out of a speaker.

**[V: quick montage: sine wave → spectrum → dB ladder → Shannon formula card]**
You'll learn the six numbers that describe every radio wave ever transmitted,
the one unit engineers use to tame impossible ratios, and the hard mathematical
ceiling that every radio ever built lives under.

**[V: pillar title card — "Radio Fundamentals: How Signals Actually Work"]**
When you turn a dial and a voice comes out — this is what actually happened.

## Course intro (2–3 min) — VO over map card + b-roll

**[V: presenter to camera, or GopherTrunk dashboard b-roll]**
Welcome. This course is for anyone who's pointed a scanner or a
thirty-dollar SDR dongle at the sky and wondered what's really going on:
new scanner listeners, new hams, tinkerers, and anyone who wants the
vocabulary under software-defined radio to stop feeling like a wall.

**[V: montage of what you'll be able to DO: read a waterfall, point at the
noise floor, follow a dB chain]**
By the end you'll be able to read a spectrum display the way you read a map,
measure signals the way engineers do — in decibels, not zeros — and follow a
complete link budget from a transmitter to your antenna. No math beyond
arithmetic, and every claim is backed by a written Field Guide article linked
below.

**[V: map card walk — Act I nodes light in turn, then Act II, then Act III]**
Three acts. Act one: what a radio wave actually *is* — the spectrum, frequency,
wavelength, amplitude, and phase. Act two: how a wave carries information and
how we measure it — modulation, bandwidth, decibels, noise, and the
signal-to-noise ratio that decides everything. Act three: the real world —
impedance, resonance, the imperfections of real hardware, and the two big
payoffs: the link budget and Shannon's limit.

**[V: chapters UI callout + gophertrunk.org/reference URL card]**
Every chapter is three to five minutes, self-contained, and listed below —
skip freely. Each one has a full written article on gophertrunk dot org with
the figures, formulas, and sources. Let's start with the map everything else
sits on.

---

## Transition beats (10–20 s each, over map card; node N lights, N+1 pulses)

**T1 → .02 radio-wave:** So that's the whole spectrum — and radio owns the
low-frequency end of it. Time to zoom in on the thing itself: the radio wave.

**T2 → .03 frequency:** Every radio wave is described by a handful of numbers,
and the first one is the one you literally tune to. Frequency.

**T3 → .04 wavelength:** Frequency has a twin — measure the same wave in space
instead of time and you get its wavelength. And wavelength is why antennas
have a size.

**T4 → .05 amplitude:** Time and space covered. The next number is about
strength: amplitude.

**T5 → .06 phase:** There's one more number, and it's the subtle one — the one
digital radio leans on hardest. Phase.

**T6 → .07 frequency-bands:** Frequency, wavelength, amplitude, phase — now
you can describe any wave. Here's how the world carves all those frequencies
into named neighborhoods.

**T7 → .08 carrier-wave (Act II open):** That's Act one: you can describe a
radio wave. Act two is about making one *carry* something — and it starts with
the wave that does the carrying.

**T8 → .09 modulation:** A carrier on its own says nothing. Changing it —
that's the message. This is the load-bearing idea of the whole course:
modulation.

**T9 → .10 bandwidth:** The moment you modulate a carrier, it stops being one
thin line on the spectrum and starts taking up room. That room has a name:
bandwidth.

**T10 → .11 decibel:** Before we can talk about signal strength honestly, we
need the unit radio actually speaks. Not watts — decibels.

**T11 → .12 noise-floor:** Now that you can count in dB, meet the enemy every
signal fights: the noise floor.

**T12 → .13 signal-to-noise-ratio:** Signal above, noise below — the only
question that matters is how far apart they are. That number is SNR, and this
chapter is the heart of the course.

**T13 → .14 attenuation:** SNR is won or lost on the way. First, the losses
you can touch: attenuation.

**T14 → .15 path-loss:** Cables lose you decibels — but the air itself takes
the biggest bite of all. Path loss.

**T15 → .16 impedance (Act III open, after midpoint recap):** Act three:
the real world. It starts with the number printed on every connector you'll
ever buy: impedance.

**T16 → .17 resonance:** Impedance explains matching. Resonance explains why a
piece of wire has a favorite frequency.

**T17 → .18 harmonics:** Real transmitters don't just emit the frequency you
asked for. Meet the uninvited multiples: harmonics.

**T18 → .19 phase-noise:** Harmonics are a transmitter's imperfection. Your
receiver has one too — its own clock jitters. That's phase noise.

**T19 → .20 erp-eirp:** Enough imperfections. Time to put numbers on a real
transmitter — what does "fifty thousand watts" actually mean? ERP and EIRP.

**T20 → .21 link-budget:** You now hold every piece: powers in dBm, gains and
losses in dB, a noise floor to beat. Let's add it all up, end to end. The
link budget.

**T21 → .22 shannon-capacity:** One question left. If you have this much
bandwidth and this much SNR — how fast can you *ever* send data? In 1948, one
man answered it permanently.

---

## Midpoint recap (60–90 s) — after GT-RF-01.15, before Act III

**[V: map card, Acts I & II fully lit; each named item's node flashes]**
Quick breather — look how much is behind you. You can describe any radio wave:
frequency, wavelength, amplitude, phase. You know how a carrier is modulated
to carry a message, why that costs bandwidth, and how to count power the way
engineers do — in decibels. And you know the fight every signal is in: staying
above the noise floor, keeping its SNR.

**[V: Act III nodes pulse one by one]**
Everything left is payoff. Real cables and connectors, real antennas with real
resonances, real hardware with real flaws — and then we put the whole course
into two closing ideas: the link budget that predicts whether a signal
arrives, and the Shannon limit that says how much it can ever carry. Act
three.

## Outro (60–90 s) + end screen (20 s)

**[V: waterfall from the cold open returns — now fully annotated: carrier,
noise floor, SNR bracket, bandwidth]**
Back to the question we opened with: you turn a dial, and a voice comes out.
Now you know what happened. A transmitter modulated a carrier; the wave spread
and faded by exactly the amount the link budget predicted; your antenna caught
a whisper of it, your receiver held it above the noise floor — and because its
SNR cleared the bar Shannon drew in 1948, the message survived.

**[V: card: gophertrunk.org/reference — "every chapter has a full write-up"]**
Every chapter of this course is a written Field Guide article at gophertrunk
dot org, with the figures, the formulas, and the sources — links below.

**[V: next-pillar card: "SDR from Zero" + subscribe elements]**
Next course: SDR from Zero — pick your first software-defined radio, plug it
in, and put everything you just learned on a real screen. Subscribe so you
don't miss it — and I'll see you there.

**[V: end screen — composed, music only, 20 s]**

---

## Production notes
- Map card states: `python3 ../../_brand/gen_cards.py` → `map_card(GT_RF_01_NODES,
  done_idx=N-1, current_idx=N, ...)` per beat; render the pulse as a 2-frame
  outline blink or a scale keyframe in Resolve.
- Record all 21 beats + midpoint in one VO session (same mic distance/EQ as
  segment VO sessions); cold open + intro + outro in the talking-head session.
- Word counts: cold open ~140 · intro ~300 · beats 25–45 each · midpoint ~150 ·
  outro ~180 — all inside the playbook's timing windows at 150 wpm.
