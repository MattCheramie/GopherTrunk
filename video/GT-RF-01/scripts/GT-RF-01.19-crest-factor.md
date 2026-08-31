# GT-RF-01.19 — Crest Factor & PAPR
slug: crest-factor-papr
type: term
target: ~3:50

**[V: title | "Crest Factor & PAPR" | GT-RF-01.19]**

**[V: calm-then-spike — a waveform envelope idling near its average; suddenly one tall peak shoots toward a red clipping ceiling]** [CLIP c1 start]
Crest factor is the measure of a waveform's temperament: how far its tallest peaks tower over its everyday level. Formally, it's the ratio of a waveform's peak amplitude to its RMS value. Write the same idea in power terms and you get PAPR — the peak-to-average power ratio, quoted in decibels as ten times the log of peak power over average power. And it matters because an amplifier doesn't get to average anything: it has to stay linear all the way up to the tallest peak, however rarely that peak arrives. [CLIP c1 end]

**[V: stadium — a crowd clapping at random; a level meter shows a steady wash, then thousands of claps align into one huge crack that pins the meter]**
Picture a stadium crowd clapping at random. Most of the time the applause is a steady wash — that's the average. But every so often, purely by chance, thousands of claps line up in the same instant and produce one enormous crack. Any sound system carrying that crowd must be sized for the crack, not the wash. A high-PAPR signal is that crowd: it spends most of its life near the average and occasionally throws a peak toward the clipping ceiling.

**[V: envelope-figure — the article's figure animates: fluctuating envelope hugging the P_avg dashed line, rare peaks reaching the P_peak/clipping line; PAPR bracket between the two lines]** [CLIP c2 start]
So where do the peaks come from? A constant-envelope signal — an FM carrier, or a GMSK waveform — has a PAPR near zero decibels: its amplitude never moves, so the amplifier can be driven right up into saturation, its most efficient operating point, without distorting anything. The moment a modulation varies its amplitude, peaks appear. Filtered QAM and root-raised-cosine pulse shaping add a few decibels of PAPR, because the filter overshoots between symbols. The extreme case is OFDM, which sums many subcarriers onto one signal: when a large number of them happen to align in phase, their voltages add coherently into a brief peak far above the average. Ten to thirteen decibels of PAPR is typical for a wideband OFDM signal. [CLIP c2 end]

**[V: backoff — amplifier transfer curve; operating point slides down 8 dB from the 1 dB compression point; an efficiency gauge sags below 20%]** [CLIP c3 start]
The design response is called backoff: operate the amplifier several decibels below its one-decibel compression point, so even the rare peaks stay in the linear region. And backoff is expensive. A power amplifier backed off eight decibels for an OFDM signal may run at well under twenty percent efficiency — the rest becomes battery drain and heat. Skip the backoff and the peaks clip, and clipping regrows spurious energy into adjacent channels and inflates the error vector magnitude. So engineers also fight the peaks themselves: clipping-and-filtering shaves the rare ones at a small cost in distortion, tone reservation and selective mapping reshape the OFDM symbol to lower its crest, and digital predistortion inverts the amplifier's own curve so it can run closer to saturation. [CLIP c3 end]

**[V: boundary-card | "PAPR = amplitude spread while transmitting · Duty cycle = fraction of time transmitting"]**
Don't confuse this with duty cycle. Duty cycle is the fraction of time a transmitter is on at all; PAPR describes the amplitude statistics while it's transmitting.

**[V: air-interfaces — three cards: P25 C4FM / DMR 4FSK "≈ constant envelope"; LTE downlink OFDM vs uplink SC-FDMA; TETRA π/4-DQPSK "modest envelope"]**
This one number shapes real air interfaces. Single-carrier land-mobile systems — P25's C4FM, DMR's four-level FSK — deliberately use near-constant-envelope modulation so a portable radio can run an efficient saturated amplifier and stretch its battery. LTE runs high-PAPR OFDM on the downlink, where a base station can afford a big, linear, backed-off amplifier — and switches to lower-PAPR SC-FDMA on the uplink precisely to spare the handset. TETRA's pi-over-four DQPSK was chosen partly to keep its envelope variation modest.

**[V: gt-tie-in — GopherTrunk receive chain: ADC input with headroom margin above a peaky signal; a dBFS gain-staging meter with a "no clipping" checkmark]**
GopherTrunk is a receiver, so it never amplifies a transmit signal and has no PAPR budget of its own. But the same statistics govern the receive side: a high-PAPR signal needs enough headroom in the analog-to-digital converter that its peaks aren't clipped on the way in. That's part of why gain staging against digital full scale — and keeping the front end out of overload — belongs to getting a clean decode.

**[V: recap | "Crest Factor & PAPR" | ① Peak over average — 10·log₁₀(P_peak/P_avg) dB | ② Peaks force backoff; backoff burns efficiency | ③ Constant envelope ≈ 0 dB; OFDM ≈ 10–13 dB]**
So: crest factor and PAPR measure how far the peaks exceed the average, peaks force amplifier backoff and backoff burns efficiency, and constant-envelope modulation is the battery-saving escape hatch. The full write-up is linked below.
