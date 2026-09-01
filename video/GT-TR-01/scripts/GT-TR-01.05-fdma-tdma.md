# GT-TR-01.05 — FDMA & TDMA
slug: fdma
also_slugs: [tdma]

**[V: title "FDMA & TDMA"]**

**[V: fdmastack intro]** [CLIP c1 start]
{FDMA|F D M A} and {TDMA|T D M A} are the two answers to the oldest question in radio: how do many conversations share one band without talking over each other?

The first answer is frequency-division multiple access. Slice the band into non-overlapping channels, and give each call its own frequency for the whole duration. Call A on one channel, call B on the next, small guard bands between them so neighbors don't interfere. Capacity scales the obvious way: more conversations means more channels.

**[V: fdmastack narrowband]**
Its virtue is simplicity — no timing to coordinate, tolerant of distance and delay — which is why it carried early land-mobile radio. Its cost is that a quiet talker still ties up a whole frequency. So the industry narrowed the slices instead: twenty-five kilohertz FM became the twelve-and-a-half of {P25|P twenty-five} Phase 1, and the six-and-a-quarter of {NXDN|N X D N}. [CLIP c1 end]

**[V: tdmaslots intro]** [CLIP c2 start]
The second answer divides time instead. Time-division multiple access splits one frequency into rapid, repeating timeslots, and calls take turns. Call one bursts in slot one, call two in slot two, back to call one — many times per second, fast enough that each conversation sounds perfectly continuous. One channel now does the work of two.

**[V: tdmaslots timing]**
The price is discipline. Every radio must burst exactly inside its slot, so bursts are padded with a little guard time, and receivers lock onto a frame-synchronization pattern to learn where the slot boundaries fall. In exchange, {DMR|D M R} and {P25|P twenty-five} Phase 2 double the calls per frequency with two slots, and {TETRA|tetra} packs four calls onto one carrier. A subscriber radio even saves battery by switching its transmitter off between bursts. [CLIP c2 end]

**[V: accesscompare table]**
Real systems combine both: {FDMA|F D M A} channelizes the band, {TDMA|T D M A} time-shares each channel. And for trunking, the difference shows up in the grant itself — on {FDMA|F D M A} a grant names a frequency; on {TDMA|T D M A} it must name a frequency and a slot.

**[V: gt-tiein]**
For an SDR, the split is concrete. On {FDMA|F D M A}, a granted channel is simply a carrier: GopherTrunk down-converts that frequency from the wideband capture and demodulates it. On {TDMA|T D M A}, the same carrier holds two or four unrelated calls, so GopherTrunk must also recover burst timing, sync to the frame, and pull out just the granted slot — which is exactly what its {DMR|D M R}, {P25|P twenty-five} Phase 2, and {TETRA|tetra} decode paths do.

**[V: recap "fdma|① FDMA: one call per frequency ② TDMA: calls share a frequency in timeslots ③ TDMA grants add a slot number"]**
So: {FDMA|F D M A} gives each call its own frequency; {TDMA|T D M A} lets calls share one frequency in timeslots, doubling or quadrupling capacity at the cost of tight timing — and on {TDMA|T D M A}, every grant names a slot as well as a channel. Full write-ups linked below.
