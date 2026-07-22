package tetra

// Downlink control-channel slot decode with correct Normal Continuous Downlink
// Burst (NCDB) geometry + AACH-steered logical-channel selection + lower-MAC
// demux. This is what lets voice-call grants decode on real air.
//
// The legacy normal-burst path in Process slices a fixed count of dibits
// *after* the normal training sequence and feeds the bits straight to the L3
// parser. That is doubly wrong for a real downlink slot: (1) an NCDB carries its
// data in BKN1 *before* the training sequence and BKN2 *after* it (the fixed
// forward slice misses BKN1 and starts mid-AACH), and (2) the recovered bits are
// a MAC PDU whose header must be stripped to reach the L3 TM-SDU. downlinkNCDB
// fixes the geometry (mirroring TrafficExtractor); decodeDownlinkSlot fixes the
// channel/MAC demux.
//
// It runs additively alongside the legacy path: real bursts (with the full
// [L-115, L+127) span) are decoded here, while the synthetic fixtures the
// in-package tests feed — which place a single block contiguously after the
// training sequence, with no BKN1 look-back — are ignored here and still handled
// by the legacy path.

// AACH half-block geometry relative to the normal training sequence leading
// dibit L (ETSI EN 300 392-2 §9.4.4.3.2, matching traffic.go's BKN layout): the
// 30-bit access-assignment is split either side of the training sequence.
const (
	ndbAACH1Start = -7 // AACH half 1: [L-7, L)   — 7 dibits
	ndbAACH1Len   = 7
	ndbAACH2Start = 11 // AACH half 2: [L+11, L+19) — 8 dibits
	ndbAACH2Len   = 8
)

// downlinkNCDB scans the control-channel dibit stream for Normal Continuous
// Downlink Bursts and hands each burst's BKN1, AACH halves, and BKN2 to onSlot.
// A rolling buffer with look-back to BKN1 and look-ahead to BKN2 around each
// detected normal training sequence — the same shape as TrafficExtractor, but
// it also surfaces the AACH so the caller can classify the slot.
type downlinkNCDB struct {
	dets    []*SyncDetector
	scratch []int
	buf     []uint8
	bufBase int
	pending []int
	onSlot  func(bkn1, aach1, aach2, bkn2 []uint8)
}

func newDownlinkNCDB(onSlot func(bkn1, aach1, aach2, bkn2 []uint8)) *downlinkNCDB {
	d := &downlinkNCDB{onSlot: onSlot}
	// The π/4-DQPSK stream carries a residual 0..3 dibit rotation, so correlate
	// the normal training sequence (NTS1 + NTS2) under all four rotations.
	for _, base := range [][]uint8{NormalSyncDibits(), NormalSyncDibits2()} {
		for r := uint8(0); r < 4; r++ {
			d.dets = append(d.dets, NewSyncDetector(rotateDibits(base, r), 2))
		}
	}
	return d
}

// process consumes a window of dibits (baseIdx = absolute index of dibits[0],
// monotonically non-decreasing) and emits each fully-buffered NCDB.
func (d *downlinkNCDB) process(dibits []uint8, baseIdx int) {
	if len(d.buf) == 0 {
		d.bufBase = baseIdx
	}
	d.buf = append(d.buf, dibits...)

	ntsLen := len(NormalSyncDibits())
	for _, det := range d.dets {
		hits, _ := det.Process(d.scratch[:0], dibits, baseIdx)
		for _, trailing := range hits {
			L := trailing - (ntsLen - 1)
			dup := false
			for _, q := range d.pending {
				if q == L {
					dup = true
					break
				}
			}
			if !dup {
				d.pending = append(d.pending, L)
			}
		}
	}

	bufEnd := d.bufBase + len(d.buf)
	kept := d.pending[:0:0]
	for _, L := range d.pending {
		needStart := L + ndbBKN1Start
		needEnd := L + ndbBKN2Start + ndbBlockDibits
		if needStart < d.bufBase {
			continue // look-back trimmed away; give up on this hit
		}
		if needEnd > bufEnd {
			kept = append(kept, L) // not enough look-ahead yet
			continue
		}
		d.emit(L)
	}
	d.pending = kept

	// Trim, keeping the trailing margin plus any unresolved hit's look-back.
	keepFrom := bufEnd - ndbTrimMargin
	for _, L := range d.pending {
		if ns := L + ndbBKN1Start; ns < keepFrom {
			keepFrom = ns
		}
	}
	if keepFrom > d.bufBase {
		drop := keepFrom - d.bufBase
		if drop > len(d.buf) {
			drop = len(d.buf)
		}
		d.buf = append(d.buf[:0], d.buf[drop:]...)
		d.bufBase += drop
	}
}

func (d *downlinkNCDB) emit(L int) {
	slice := func(off, n int) []uint8 {
		s := L + off - d.bufBase
		if s < 0 || s+n > len(d.buf) {
			return nil
		}
		return d.buf[s : s+n]
	}
	bkn1 := slice(ndbBKN1Start, ndbBlockDibits)
	aach1 := slice(ndbAACH1Start, ndbAACH1Len)
	aach2 := slice(ndbAACH2Start, ndbAACH2Len)
	bkn2 := slice(ndbBKN2Start, ndbBlockDibits)
	if bkn1 == nil || aach1 == nil || aach2 == nil || bkn2 == nil {
		return
	}
	d.onSlot(bkn1, aach1, aach2, bkn2)
}

// decodeDownlinkSlot classifies and decodes one NCDB. It reads the AACH to find
// the constellation rotation the slot decodes under, then recovers the slot's
// signalling — SCH/F over the full slot (BKN1+BKN2) and SCH/HD over each half —
// and hands each CRC-clean block to the MAC demux. CRC gates every decode, so
// trying all three channel shapes never double-counts: a full-slot SCH/F slot
// fails the half-slot SCH/HD CRC and vice versa.
func (c *ControlChannel) decodeDownlinkSlot(bkn1, aach1, aach2, bkn2 []uint8) {
	c.mu.Lock()
	colour := c.colourCode
	c.mu.Unlock()

	derot := func(d []uint8, rot uint8) []byte {
		return TetraDibitsToBits(rotateDibits(d, (4-rot)&3))
	}

	// The AACH is present in every downlink slot and decodes ~100% on a locked
	// carrier, so it pins the rotation cheaply; fall back to trying all four
	// only when it does not decode (e.g. a synthesised slot with no AACH).
	rots := []uint8{0, 1, 2, 3}
	aachDi := append(append([]uint8{}, aach1...), aach2...)
	for r := uint8(0); r < 4; r++ {
		rec, errs := DecodeAACH(derot(aachDi, r), colour)
		if errs >= 0 {
			if _, ok := ParseAccessAssign(rec); ok {
				rots = []uint8{r}
				break
			}
		}
	}

	for _, r := range rots {
		full := append(append([]uint8{}, bkn1...), bkn2...)
		if rec, ok := DecodeSCHF(derot(full, r), colour); ok {
			c.ingestMAC(rec)
		}
		if rec, ok := DecodeSCHHD(derot(bkn1, r), colour); ok {
			c.ingestMAC(rec)
		}
		if rec, ok := DecodeSCHHD(derot(bkn2, r), colour); ok {
			c.ingestMAC(rec)
		}
	}
}

// ingestMAC demultiplexes a recovered logical-channel block (type-1 bits) at the
// MAC layer: a MAC-RESOURCE PDU's channel-allocation element becomes a voice
// grant, and its TM-SDU is handed to the L3 parser. MAC-FRAG/END reassembly and
// broadcast (SYSINFO) handling here are follow-ups — SYSINFO identity already
// reaches Ingest via the synchronisation-burst lock path.
func (c *ControlChannel) ingestMAC(recovered []byte) {
	if len(recovered) < 2 {
		return
	}
	switch MACPDUType(recovered[0]<<1 | recovered[1]) {
	case MACPDUResource:
		m, ok := ParseMACResource(recovered, false)
		if !ok || m.NullPDU {
			return
		}
		if m.ChanAlloc != nil {
			c.publishGrantFromMAC(m)
		}
		if sdu := m.tmSDU(recovered); sdu != nil {
			if pdu, err := PDUFromBits(sdu); err == nil {
				c.Ingest(pdu)
			}
		}
	case MACPDUBroadcast:
		// Learn the cell's own carrier number so grant carrier numbers resolve
		// to Hz relative to this carrier (see carrierFrequency). SYSINFO
		// identity (MCC/MNC/LA) still reaches the lock via the sync-burst path.
		if mc, ok := SysInfoMainCarrier(recovered); ok {
			c.learnMainCarrier(mc)
		}
	}
}

// publishGrantFromMAC turns a MAC-RESOURCE channel allocation into a voice
// grant. The physical resource (carrier + timeslot) comes from the MAC
// channel-allocation element; the addressed party SSI is the grant's group/dest
// identity. Source SSI and the emergency/group flags live in the CMCE TM-SDU and
// are filled once fragment reassembly surfaces the full CMCE PDU.
func (c *ControlChannel) publishGrantFromMAC(m MACResource) {
	// A grant on the CC is itself enough to declare the channel locked.
	c.maybeLock(LockState{FrequencyHz: c.freqHz})
	c.publishGrant(VoiceGrant{
		DestSSI:       m.Address.SSI,
		CarrierNumber: m.ChanAlloc.CarrierNumber,
		Timeslot:      m.ChanAlloc.Timeslot & 0x3,
		Encrypted:     m.Encrypted,
	})
}
