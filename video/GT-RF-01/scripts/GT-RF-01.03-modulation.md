# GT-RF-01.03 — Modulation
slug: modulation
also_slugs: [carrier-wave]
type: term
target: ~4:05

**[V: title | "Modulation" | GT-RF-01.03]**

**[V: bare-carrier — a perfect, endless sine wave scrolling; a "no information" stamp]** [CLIP c1 start]
Modulation is the answer to an embarrassing question: a perfect radio wave says absolutely nothing. A pure, steady carrier is just a hum in the spectrum — endlessly repeating, perfectly predictable, carrying zero information. To make a wave *say* something, you have to change it. Modulation is the art of changing a carrier wave in step with a message — and every radio scheme in history is a variation on that one move.

**[V: three-knobs — the carrier with three labeled knobs: AMPLITUDE, FREQUENCY, PHASE]**
Remember the three properties from the last chapter's write-up — a sine wave has an amplitude, a frequency, and a phase, and those are the only things a transmitter can touch. So there are only three families of modulation. Everything from nineteen-twenties broadcasting to your phone is built from these three knobs. [CLIP c1 end]

**[V: am-fm — the article's figure animates: message wave on top; AM trace grows its envelope; FM trace squeezes and stretches its cycles]** [CLIP c2 start]
Turn the amplitude knob and you get AM: the message is written into the carrier's height — its envelope literally traces the audio. Simple, but noise also lives in amplitude, which is why AM crackles in a thunderstorm. Turn the frequency knob and you get FM: the message speeds up and slows down the carrier's wiggle while the height stays constant — so amplitude noise mostly wipes right off. That constant envelope is why FM sounds cleaner than AM, and the third knob, phase, nudges the carrier's timing — closely related to FM, and the foundation of almost everything digital. [CLIP c2 end]

**[V: symbols — continuous wave snaps to discrete states; four phase states appear as dots on an IQ plane, each labeled with 2 bits]**
Analog modulation varies the carrier continuously. Digital modulation makes one crucial change: it switches the carrier among a *finite menu* of states, called symbols, each standing for one or more bits. Shift between discrete frequencies and it's FSK — frequency-shift keying. Discrete phases: PSK. Combine amplitude and phase levels: QAM. Draw the menu as dots on a plane — amplitude out from the center, phase around it — and you get the constellation diagram, the map every digital receiver uses to decide which symbol just arrived.

**[V: trade — constellation grows 4 → 16 → 64 points; dots crowd together; a noise cloud blurs them]**
And here is the trade that shapes all of digital radio. More dots on the map means more bits per symbol — faster data in the same bandwidth. But more dots also means the dots sit closer together, so it takes less noise to smudge one into its neighbor. Fast-but-fragile, or slow-but-tough. Wi-Fi at close range runs hundreds of points; a police radio fighting fades across a whole city runs just four.

**[V: gt-tie-in — GopherTrunk decoding: four-level FSK eye and π/4-DQPSK constellation side by side, live]**
That's exactly what GopherTrunk spends its life demodulating. The land-mobile systems it decodes chose rugged, low-order schemes on purpose: four-level FSK — four frequency steps, two bits per symbol — for DMR and NXDN, and a rotating four-phase scheme called pi-over-four DQPSK for TETRA and P25's simulcast flavor. Once the signal is digitized, demodulation is just measuring amplitude, frequency, and phase — the same three knobs — and mapping them back to bits.

**[V: recap | "Modulation" | ① A bare carrier says nothing — change it to speak | ② Three knobs: amplitude (AM), frequency (FM), phase | ③ Digital = a finite menu of symbols; more symbols = faster but fragile]**
So: modulation turns a mute carrier into a messenger, there are exactly three knobs to turn, and digital radio is a menu of symbol states traded off between speed and toughness. Full write-up linked below.
