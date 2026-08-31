# GT-RF-01.17 — Occupied bandwidth
slug: occupied-bandwidth
also_slugs: [guard-band]
type: term
target: ~3:50

**[V: title | "Occupied bandwidth" | GT-RF-01.17]**

**[V: endless-skirts — zooming into a waterfall signal; its edges never quite end, power rolling off into skirts that fade toward the noise; a cursor hunts for "the edge"]** [CLIP c1 start]
Occupied bandwidth answers a question that sounds trivial and isn't: how wide is a signal, really? Zoom into any transmission on a waterfall and its edges never actually end — the power just rolls off into skirts that fade toward the noise. So where do you draw the line? Regulators settled it long ago: draw it where ninety-nine percent of the power lives.

**[V: definition-card — the article's figure animated: the PSD curve's area fills with shading; 0.5% slivers shave off each edge; a bracket spans the middle | "OBW = the band holding 99% of total power"]**
Occupied bandwidth is the width of the band containing a specified fraction of the signal's total transmitted power — by long-standing ITU convention ninety-nine percent, with half a percent left over above and half a percent below. Integrate the power spectral density to get the total, find the edge frequencies that cut off half a percent on each side, and the span between them is the occupied bandwidth. [CLIP c1 end] Unlike the vague "bandwidth" of a signal, it's a precise, measurable number — and it captures the real spectral footprint, modulation sidebands and pulse-shaping skirts included.

**[V: parking-lot — cars in painted bays; doors and mirrors stick out past each car; the painter draws bays wider than the cars and leaves gaps between them]**
Think parking spaces. A car has a nominal width, but the doors swing and the mirrors stick out — so the lot painter draws each bay a little wider than the car, and leaves a gap between bays. Spectrum is planned exactly the same way: the car is the occupied bandwidth, the bay is the channel, and the painted gap between bays has a name.

**[V: guard-band — the guard-band article figure animated: two channel masks side by side; their skirts descend toward each other; the vacant strip between them highlights | "channel spacing = OBW + guard band"]** [CLIP c2 start]
That gap is the guard band: a deliberately unused strip of spectrum between adjacent channels, so energy from one doesn't leak into the next. It exists because every real transmitter's skirts roll off over a finite transition, and every receiver's filter has finite steepness — the guard band puts the overlap where both are already far down, turning would-be interference into negligible noise. The arithmetic is one line: channel spacing equals occupied bandwidth plus guard band. Sharp pulse shaping needs less guard; a sloppy or drifting emitter needs more. So the guard band trades a little spectral efficiency for a large reduction in adjacent-channel interference. [CLIP c2 end]

**[V: designator-card | "16K0F3E → 16.0 kHz · F = frequency modulation · 3 = analog telephony · E = telephony" — then a 25 kHz channel splits into two 12.5 kHz channels]** [CLIP c3 start]
These numbers go on the license. A close regulatory cousin, necessary bandwidth, is the minimum width an emission class theoretically needs; occupied bandwidth is what a transmitter actually uses. Both feed the ITU emission designator — a code like one-six-K-zero-F-three-E: sixteen point zero kilohertz wide, frequency modulation, analog telephony. And occupied bandwidth is the number that sets channel spacing: tighten it with sharper filtering or more efficient modulation and a whole band can be re-planned narrower. The VHF and UHF move from twenty-five-kilohertz channels to twelve and a half — "narrowbanding" — was exactly this. [CLIP c3 end]

**[V: scales — guard bands at every scale: land-mobile channel plan ticks, broadcast FM/TV allocations, cellular and Wi-Fi block edges, OFDM edge subcarriers going dark]**
Guard bands show up at every scale: between each twelve-and-a-half or twenty-five-kilohertz land-mobile channel, between broadcast stations, at the edges of cellular and Wi-Fi blocks. In multicarrier systems like OFDM, unused edge subcarriers act as an internal guard band — and a guard interval plays the same role in the *time* domain, against inter-symbol interference.

**[V: gt-tie-in — GopherTrunk channelizer: a wideband capture; a channel filter hugs one P25 channel's ~12.5 kHz footprint; too-narrow clips the sidebands, too-wide drags in the neighbour]**
For your receiver, occupied bandwidth dictates the channel filter. Too narrow clips the modulation sidebands and degrades the demodulator; too wide admits extra noise and adjacent-channel energy. Measuring it off a spectrogram is also a quick way to identify an unknown emission before decoding. GopherTrunk sizes its per-channel filters and decimation from each protocol's known channel rate and occupied bandwidth — the roughly twelve-and-a-half-kilohertz footprint of a P25 C4FM channel, for example — so the channelizer passes the full modulation without dragging in neighbours. And the guard bands in the source system's plan are part of why those channels can be cleanly split at all: GopherTrunk doesn't create guard bands, but its channelizer relies on them existing.

**[V: recap | "Occupied bandwidth" | ① OBW = span holding 99% of the power (0.5% off each edge) | ② Channel spacing = OBW + guard band | ③ Filter to the OBW: narrower clips, wider adds noise]**
So: occupied bandwidth is where the ninety-nine-percent line falls, guard bands are the bought-and-paid-for silence between neighbours, and matching your filter to the real footprint is how clean channels come out of a crowded band. Both write-ups are linked below.
