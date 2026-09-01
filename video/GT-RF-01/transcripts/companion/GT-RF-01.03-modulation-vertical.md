# Modulation — transcript (GT-RF-01.03-modulation-vertical.mp4)

_9:16 re-edit · 3:13 · 1080×1920 · title card replaced by a 2 s burned hook_

**[0:00] bare-carrier**
Modulation is the answer to an embarrassing question: a perfect radio wave says absolutely nothing. A pure, steady carrier is just a hum in the spectrum — endlessly repeating, perfectly predictable, carrying zero information. To make a wave say something, you have to change it. Modulation is the art of changing a carrier wave in step with a message — and every radio scheme in history is a variation on that one move.

**[0:26] three-knobs**
Remember the three properties from the last chapter's write-up — a sine wave has an amplitude, a frequency, and a phase, and those are the only things a transmitter can touch. So there are only three families of modulation. Everything from nineteen-twenties broadcasting to your phone is built from these three knobs.

**[0:45] am-fm**
Turn the amplitude knob and you get AM: the message is written into the carrier's height — its envelope literally traces the audio. Simple, but noise also lives in amplitude, which is why AM crackles in a thunderstorm. Turn the frequency knob and you get FM: the message speeds up and slows down the carrier's wiggle while the height stays constant — so amplitude noise mostly wipes right off. That constant envelope is why FM sounds cleaner than AM, and the third knob, phase, nudges the carrier's timing — closely related to FM, and the foundation of almost everything digital.

**[1:22] symbols**
Analog modulation varies the carrier continuously. Digital modulation makes one crucial change: it switches the carrier among a finite menu of states, called symbols, each standing for one or more bits. Shift between discrete frequencies and it's FSK — frequency-shift keying. Discrete phases: PSK. Combine amplitude and phase levels: QAM. Draw the menu as dots on a plane — amplitude out from the center, phase around it — and you get the constellation diagram, the map every digital receiver uses to decide which symbol just arrived.

**[1:59] trade**
And here is the trade that shapes all of digital radio. More dots on the map means more bits per symbol — faster data in the same bandwidth. But more dots also means the dots sit closer together, so it takes less noise to smudge one into its neighbor. Fast-but-fragile, or slow-but-tough. Wi-Fi at close range runs hundreds of points; a police radio fighting fades across a whole city runs just four.

**[2:25] gt-tie-in**
That's exactly what GopherTrunk spends its life demodulating. The land-mobile systems it decodes chose rugged, low-order schemes on purpose: four-level FSK — four frequency steps, two bits per symbol — for DMR and NXDN, and a rotating four-phase scheme called pi-over-four DQPSK for TETRA and P25's simulcast flavor. Once the signal is digitized, demodulation is just measuring amplitude, frequency, and phase — the same three knobs — and mapping them back to bits.

**[2:57] Recap card**
So: modulation turns a mute carrier into a messenger, there are exactly three knobs to turn, and digital radio is a menu of symbol states traded off between speed and toughness. Full write-up linked below.
