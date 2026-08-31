# GT-RF-01.18 — Duty Cycle
slug: duty-cycle
type: term
target: ~3:50

**[V: title | "Duty Cycle" | GT-RF-01.18]**

**[V: keyed-mic — a PTT mic keys; a timeline paints solid while keyed, empty in the gaps; a percentage counter tallies on-time]** [CLIP c1 start]
Duty cycle is the answer to a deceptively simple question: how much of the time is your transmitter actually on? Not how strong the signal is — how often it exists at all. Take the time the transmitter is keyed, divide by the whole cycle — on-time plus off-time — and you get a fraction between zero and one hundred percent. That one number quietly decides how hot a radio runs, how big its heatsink has to be, and how fast a portable drains its battery.

**[V: definition-card | "D = t_on / (t_on + t_off) — average power = D × peak power"]**
It does all that through one clean relationship: average power equals peak power scaled by duty cycle. A fifty-watt transmitter keyed ten percent of the time is delivering just five watts on average. Same peak, one tenth of the heat. [CLIP c1 end]

**[V: sprinter — split screen: sprinter in short explosive bursts vs marathoner at steady pace; a shared "heat" gauge fills on average effort]**
Think of a sprinter. A sprinter puts out enormous power — for ten seconds. A marathon runner puts out far less, but holds it for hours. Neither is cheating; they're spending the same thermal budget, because the body overheats on *average* effort, not peak effort. A transmitter's final amplifier is exactly that athlete. Heating in the output stage tracks average dissipated power, so a low duty cycle lets a small amplifier and a modest heatsink survive bursts that would cook them if sustained continuously.

**[V: pulse-trains — the article's figure animates: two pulse trains, same peak height; the low-duty (~25%) train flashes brief pulses with long gaps, the high-duty (~75%) train stays lit; an average-power bar fills beside each]** [CLIP c2 start]
Picture two pulse trains with the same peak height. The first fires short bursts with long gaps — roughly twenty-five percent duty. The second stays keyed most of the time — around seventy-five percent. On a peak-reading meter they look identical, but the second dissipates three times the average power. And real radio spans the whole range. At one hundred percent: analog FM broadcast, an unmodulated carrier, and data modes that hold the key down for the entire message. Up high: a busy repeater, or a trunking control channel that must broadcast signalling almost continuously so subscribers can find it. Down low: push-to-talk voice, keyed only while someone actually speaks, and TDMA systems where each radio transmits in its assigned time slots and rests in between. [CLIP c2 end]

**[V: datasheet — a dummy-load spec card: "100 W continuous / 300 W intermittent"; a duty-cycle slider moves the allowed peak up and down]**
Datasheets bake this in. A dummy load rated one hundred watts continuous but three hundred watts intermittent survives the bigger peaks only if the average — after applying the duty cycle — stays inside its thermal limit. It's also why operators running near-one-hundred-percent modes like FT8, RTTY, or digital voice routinely turn their power down below the rig's sideband rating: the finals were never meant for constant key-down at full output.

**[V: boundary-card | "Duty cycle = time on vs off · Crest factor = amplitude spread while on"]**
One boundary worth drawing: duty cycle is about time on versus time off. It says nothing about how the amplitude behaves *while* the transmitter is on — that's a separate statistic called crest factor. A signal can be keyed one hundred percent of the time with a wildly varying envelope, or keyed ten percent of the time with a perfectly constant one.

**[V: gt-tie-in — GopherTrunk spectrogram: a slotted TDMA carrier beside a continuous control-channel smear; callouts "≈50% duty" and "near-continuous"]** [CLIP c3 start]
GopherTrunk is a receiver — it never keys a transmitter, so it has no duty-cycle rating of its own. But the concept is a field mark. Open a spectrogram of a TDMA signal like DMR or P25 Phase Two and the slotted, bursty structure you see is duty cycle drawn directly on screen: each logical channel runs at roughly fifty percent because two conversations alternate slots on one carrier — and the portable's average power and battery drain fall with it. A near-continuous emission, by contrast, says control channel. You can often tell which protocol and channel type you're looking at before decoding a single bit. [CLIP c3 end]

**[V: recap | "Duty Cycle" | ① D = on-time ÷ full cycle | ② Average power = D × peak — heat follows the average | ③ A spectrogram's slots are duty cycle made visible]**
So: duty cycle is the on-time fraction, average power is peak power scaled by that fraction, and heat always follows the average. The full write-up is linked below.
