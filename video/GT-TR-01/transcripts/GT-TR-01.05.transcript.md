# GT-TR-01.05 — FDMA & TDMA

FDMA and TDMA are the two answers to the oldest question in radio: how do many conversations share one band without talking over each other?

The first answer is frequency-division multiple access. Slice the band into non-overlapping channels, and give each call its own frequency for the whole duration. Call A on one channel, call B on the next, small guard bands between them so neighbors don't interfere. Capacity scales the obvious way: more conversations means more channels.

Its virtue is simplicity — no timing to coordinate, tolerant of distance and delay — which is why it carried early land-mobile radio. Its cost is that a quiet talker still ties up a whole frequency. So the industry narrowed the slices instead: twenty-five kilohertz FM became the twelve-and-a-half of P25 Phase 1, and the six-and-a-quarter of NXDN.

The second answer divides time instead. Time-division multiple access splits one frequency into rapid, repeating timeslots, and calls take turns. Call one bursts in slot one, call two in slot two, back to call one — many times per second, fast enough that each conversation sounds perfectly continuous. One channel now does the work of two.

The price is discipline. Every radio must burst exactly inside its slot, so bursts are padded with a little guard time, and receivers lock onto a frame-synchronization pattern to learn where the slot boundaries fall. In exchange, DMR and P25 Phase 2 double the calls per frequency with two slots, and TETRA packs four calls onto one carrier. A subscriber radio even saves battery by switching its transmitter off between bursts.

Real systems combine both: FDMA channelizes the band, TDMA time-shares each channel. And for trunking, the difference shows up in the grant itself — on FDMA a grant names a frequency; on TDMA it must name a frequency and a slot.

For an SDR, the split is concrete. On FDMA, a granted channel is simply a carrier: GopherTrunk down-converts that frequency from the wideband capture and demodulates it. On TDMA, the same carrier holds two or four unrelated calls, so GopherTrunk must also recover burst timing, sync to the frame, and pull out just the granted slot — which is exactly what its DMR, P25 Phase 2, and TETRA decode paths do.

So: FDMA gives each call its own frequency; TDMA lets calls share one frequency in timeslots, doubling or quadrupling capacity at the cost of tight timing — and on TDMA, every grant names a slot as well as a channel. Full write-ups linked below.
