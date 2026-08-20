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
	softBuf []complex64 // per-symbol differentials, strictly parallel to buf (or empty)
	bufBase int
	pending []int
	onSlot  func(bkn1, aach1, aach2, bkn2 []uint8, bkn1Diffs, bkn2Diffs []complex64)
}

func newDownlinkNCDB(onSlot func(bkn1, aach1, aach2, bkn2 []uint8, bkn1Diffs, bkn2Diffs []complex64)) *downlinkNCDB {
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
func (d *downlinkNCDB) process(dibits []uint8, diffs []complex64, baseIdx int) {
	if len(d.buf) == 0 {
		d.bufBase = baseIdx
	}
	d.buf = append(d.buf, dibits...)
	// Keep the soft buffer strictly parallel to buf (same discipline as
	// processSB): append only while soft data is supplied every call, else drop
	// the soft path rather than misalign. Empty softBuf ⇒ hard-only decode.
	if diffs != nil && len(diffs) == len(dibits) && len(d.softBuf) == len(d.buf)-len(dibits) {
		d.softBuf = append(d.softBuf, diffs...)
	} else if len(d.softBuf) != len(d.buf) {
		d.softBuf = d.softBuf[:0]
	}

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
		if len(d.softBuf) == len(d.buf) {
			d.softBuf = append(d.softBuf[:0], d.softBuf[drop:]...)
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
	// softSlice returns the per-symbol differentials for the same span, or nil
	// when soft data is not available/aligned (hard-only decode downstream).
	softSlice := func(off, n int) []complex64 {
		if len(d.softBuf) != len(d.buf) {
			return nil
		}
		s := L + off - d.bufBase
		if s < 0 || s+n > len(d.softBuf) {
			return nil
		}
		return d.softBuf[s : s+n]
	}
	bkn1 := slice(ndbBKN1Start, ndbBlockDibits)
	aach1 := slice(ndbAACH1Start, ndbAACH1Len)
	aach2 := slice(ndbAACH2Start, ndbAACH2Len)
	bkn2 := slice(ndbBKN2Start, ndbBlockDibits)
	if bkn1 == nil || aach1 == nil || aach2 == nil || bkn2 == nil {
		return
	}
	d.onSlot(bkn1, aach1, aach2, bkn2, softSlice(ndbBKN1Start, ndbBlockDibits), softSlice(ndbBKN2Start, ndbBlockDibits))
}

// decodeDownlinkSlot classifies and decodes one NCDB. It reads the AACH to find
// the constellation rotation the slot decodes under, then recovers the slot's
// signalling — SCH/F over the full slot (BKN1+BKN2) and SCH/HD over each half —
// and hands each CRC-clean block to the MAC demux. CRC gates every decode, so
// trying all three channel shapes never double-counts: a full-slot SCH/F slot
// fails the half-slot SCH/HD CRC and vice versa.
func (c *ControlChannel) decodeDownlinkSlot(bkn1, aach1, aach2, bkn2 []uint8, bkn1Diffs, bkn2Diffs []complex64) {
	c.mu.Lock()
	colour := c.colourCode
	c.mu.Unlock()

	derot := func(d []uint8, rot uint8) []byte {
		return TetraDibitsToBits(rotateDibits(d, (4-rot)&3))
	}

	// The AACH is present in every downlink slot and decodes ~100% on a locked
	// carrier, so it pins the rotation cheaply; fall back to trying all four
	// only when it does not decode (e.g. a synthesised slot with no AACH). The
	// parsed ACCESS-ASSIGN also classifies the slot (control vs traffic), which
	// gates the SCHPDUsFail counter below.
	rots := []uint8{0, 1, 2, 3}
	var aa AccessAssign
	var aachOK bool
	aachDi := append(append([]uint8{}, aach1...), aach2...)
	for r := uint8(0); r < 4; r++ {
		rec, errs := DecodeAACH(derot(aachDi, r), colour)
		if errs >= 0 {
			if parsed, ok := ParseAccessAssign(rec); ok {
				aa, aachOK = parsed, true
				rots = []uint8{r}
				break
			}
		}
	}

	// Soft data (the per-symbol differentials) lets the longer SCH/F and SCH/HD
	// blocks decode ~1.5–2 dB deeper on a marginal constellation — the same soft
	// path the BSCH/BNCH already use. Try soft first, fall back to the hard slice.
	// softType5FromDiffs(diffs, (4-r)&3) hard-slices to derot(dibits, r), so soft
	// and hard use the identical rotation. CRC gates both, so no double-count.
	haveF := bkn1Diffs != nil && bkn2Diffs != nil && len(bkn1Diffs) == len(bkn1) && len(bkn2Diffs) == len(bkn2)

	recovered := 0
	for _, r := range rots {
		full := append(append([]uint8{}, bkn1...), bkn2...)
		if haveF {
			fullDiffs := append(append([]complex64{}, bkn1Diffs...), bkn2Diffs...)
			if rec, ok := DecodeSCHFSoft(softType5FromDiffs(fullDiffs, (4-r)&3), colour); ok {
				c.ingestMAC(rec)
				recovered++
			} else if rec, ok := DecodeSCHF(derot(full, r), colour); ok {
				c.ingestMAC(rec)
				recovered++
			}
		} else if rec, ok := DecodeSCHF(derot(full, r), colour); ok {
			c.ingestMAC(rec)
			recovered++
		}
		if c.decodeDownlinkHalf(bkn1, bkn1Diffs, r, derot, colour) {
			recovered++
		}
		if c.decodeDownlinkHalf(bkn2, bkn2Diffs, r, derot, colour) {
			recovered++
		}
	}

	// Decode-health telemetry (debug-gated). SCHPDUs counts CRC-clean signalling
	// blocks off the real NCDB slot — one for a full-slot SCH/F, up to two for a
	// two-half SCH/HD slot — the metric that stayed 0 on air when it was bolted to
	// the legacy fixed-slice path. SCHPDUsFail counts a slot the AACH confirms is
	// carrying control that yielded no CRC-clean block: a real signalling-payload
	// decode failure. The AACH gate keeps same-carrier traffic (TCH) slots and
	// correlator noise (AACH itself fails to decode) out of the failure count.
	if recovered > 0 {
		c.addStat(&c.stats.SCHPDUs, int64(recovered))
		// Payload heartbeat: a CRC-clean SCH block is what distinguishes a
		// genuinely decoding channel from the BSCH-only "locked but deaf" state
		// (wrong AFC alias latch) the payload-drought check escapes.
		c.notePayloadActivity()
	} else if aachOK && aa.IsControlChannel() {
		c.addStat(&c.stats.SCHPDUsFail, 1)
	}
}

// decodeDownlinkHalf decodes one SCH/HD half-slot under rotation r, soft-first
// (when the per-symbol differentials are available and aligned) then hard. It
// reports whether a CRC-clean block was recovered so the caller can count it.
func (c *ControlChannel) decodeDownlinkHalf(bkn []uint8, diffs []complex64, r uint8, derot func([]uint8, uint8) []byte, colour uint32) bool {
	if diffs != nil && len(diffs) == len(bkn) {
		if rec, ok := DecodeSCHHDSoft(softType5FromDiffs(diffs, (4-r)&3), colour); ok {
			c.ingestMAC(rec)
			return true
		}
	}
	if rec, ok := DecodeSCHHD(derot(bkn, r), colour); ok {
		c.ingestMAC(rec)
		return true
	}
	return false
}

// ingestMAC demultiplexes a recovered logical-channel block (type-1 bits) at the
// MAC layer: a MAC-RESOURCE PDU's channel-allocation element becomes a voice
// grant, and its TM-SDU is handed to the L3 parser. A TM-SDU too large for one
// block arrives as a start-fragment MAC-RESOURCE plus following MAC-FRAG /
// MAC-END PDUs, reassembled here. Broadcast (SYSINFO) frequency parameters are
// learned; SYSINFO identity already reaches Ingest via the sync-burst lock path.
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
		if m.StartFragment {
			// The TM-SDU is truncated here and continues in following
			// MAC-FRAG/MAC-END PDUs. The physical resource (carrier/timeslot/GSSI)
			// is fully known from this MAC header, so publish the grant now — no
			// latency cost, and robust to a lost MAC-END — and stash the partial
			// TM-SDU so the MAC-END can recover the complete L3 CMCE PDU (the
			// call-id→group mapping / release / talker a single block can't carry).
			if m.ChanAlloc != nil {
				c.publishGrantFromMAC(m, CMCEMessage{}, false)
			}
			if sdu := m.tmSDU(recovered); sdu != nil {
				c.stashFragment(m, sdu)
			}
			return
		}
		c.ingestResourceTMSDU(m, m.tmSDU(recovered))
	case MACPDUFragEnd:
		// Continuation of a start-fragment TM-SDU (§21.4.3.3/.4). MAC-FRAG appends;
		// MAC-END completes the reassembly and runs the same L3 handling as a
		// single-block MAC-RESOURCE, keyed by the stashed start-fragment context.
		sub, ok := macFragmentSubtype(recovered)
		if !ok {
			return
		}
		payload := macFragmentPayload(recovered)
		switch sub {
		case macFrag:
			c.appendFragment(payload)
		case macEnd:
			if m, sdu, ok := c.takeFragment(payload); ok {
				c.ingestResourceTMSDU(m, sdu)
			}
		}
	case MACPDUBroadcast:
		// Learn the cell's full SYSINFO frequency parameters (main carrier +
		// frequency band + offset + duplex spacing + reverse operation) so grant
		// carrier numbers resolve to their absolute Hz *including the offset
		// field* (see carrierFrequency), and the true downlink/uplink carrier can
		// be reported. SYSINFO identity (MCC/MNC/LA) still reaches the lock via the
		// sync-burst path.
		if si, ok := ParseSysInfo(recovered); ok {
			c.learnSysInfo(si)
		}
	}
}

// ingestResourceTMSDU runs the L3 handling for a MAC-RESOURCE whose TM-SDU is
// complete — either delivered in a single block or reassembled from a
// start-fragment + MAC-FRAG/MAC-END chain. sdu is the TM-SDU bits (nil for a
// bare grant with no TM-SDU). It decodes the CMCE PDU (to enrich the grant with
// its emergency flag + source SSI and to drive call-state), publishes a voice
// grant for any channel allocation — unless the CMCE is a teardown (D-RELEASE /
// D-DISCONNECT) whose allocation is the resource being reclaimed, which would
// otherwise spawn a zero-byte ghost call — and acts on the call-control PDU.
func (c *ControlChannel) ingestResourceTMSDU(m MACResource, sdu []byte) {
	var (
		msg      CMCEMessage
		haveCMCE bool
	)
	if sdu != nil {
		// The MAC TM-SDU is an LLC PDU (§21.2): strip the basic-link header (and
		// any FCS trailer) to reach the TL-SDU — the MLE PDU — before the 3-bit MLE
		// discriminator + CMCE fields line up. Feeding the raw TM-SDU into
		// ParseCMCE misframes every L3 address (corrupts ISSI/GSSI).
		if tl, ok := ParseLLC(sdu); ok {
			msg, haveCMCE = ParseCMCE(tl)
			// Keep the broadcast/SYSINFO L3 path (Ingest no longer emits grants).
			if pdu, err := PDUFromBits(tl); err == nil {
				c.Ingest(pdu)
			}
		}
	}
	if m.ChanAlloc != nil && !(haveCMCE && isCMCETeardown(msg.Type)) {
		c.publishGrantFromMAC(m, msg, haveCMCE)
	}
	if haveCMCE {
		c.handleCMCE(m, msg)
	}
}

// publishGrantFromMAC turns a MAC-RESOURCE channel allocation into a voice
// grant. The physical resource (carrier + timeslot) and the addressed group SSI
// come from the MAC layer (reliable, bit-accurate). When the same MAC-RESOURCE's
// CMCE TM-SDU decoded (haveCMCE), the grant is enriched with the correctly
// decoded emergency flag and — when present — the calling party's source SSI.
func (c *ControlChannel) publishGrantFromMAC(m MACResource, msg CMCEMessage, haveCMCE bool) {
	// A grant on the CC is itself enough to declare the channel locked.
	c.maybeLock(LockState{FrequencyHz: c.freqHz})
	g := VoiceGrant{
		DestSSI:       m.Address.SSI,
		CarrierNumber: m.ChanAlloc.CarrierNumber,
		Timeslot:      m.ChanAlloc.Timeslot & 0x3,
		// The MAC address usage marker (present when the party is addressed by
		// SSI+usage, §21.4.3.1) is the call's downlink usage marker — the reliable
		// key the voice chain uses to demultiplex concurrent same-carrier calls by
		// matching it against each traffic slot's AACH marker. Absent (0) on a
		// plain-SSI grant, which falls the voice chain back to CRC-gated isolation.
		UsageMarker: m.Address.UsageMarker,
		Encrypted:   m.Encrypted,
	}
	if haveCMCE {
		g.Emergency = msg.Emergency
		if msg.HasPriority {
			g.Priority = msg.Priority
		}
	}
	c.classifyParties(&g, m, msg, haveCMCE)
	c.publishGrant(g)
}

// classifyParties fills a grant's GroupID (DestSSI), SourceID (SourceSSI) and
// Individual flag from the MAC address and the decoded CMCE, so a unit-to-unit
// call's subscriber ISSI is not surfaced as a talkgroup.
//
// The reliable on-air signal (confirmed against real captures): a CMCE
// calling/transmitting party SSI (msg.PartySSI) is always an *individual* radio,
// and a grant that carries one is addressed under the group's GSSI (the MAC
// address) with that party as the source. Any SSI ever seen as a party is
// therefore an individual; a later grant addressed to it (e.g. a D-CONNECT to the
// calling party, which has no party field of its own) is an individual-addressed
// PDU, not a talkgroup grant.
func (c *ControlChannel) classifyParties(g *VoiceGrant, m MACResource, msg CMCEMessage, haveCMCE bool) {
	addr := m.Address.SSI
	g.DestSSI = addr
	g.SourceSSI = 0
	g.Individual = false

	if haveCMCE && msg.PartySSI != 0 {
		// Group call addressed under the GSSI, with the calling/transmitting party
		// as the source. Definitive — no history needed. Learn the party as an
		// individual so a later PDU addressed to it is classified correctly.
		g.SourceSSI = msg.PartySSI
		c.noteIndividual(msg.PartySSI)
		// Party == addressed SSI means the PDU is addressed to the transmitting
		// radio itself: this is an INDIVIDUAL (point-to-point) call, not a group —
		// you can't call yourself. Flag it and drop the impossible self-source so
		// the call is not surfaced as "group X, source X" (the tg==src the operator
		// reported). The counterparty isn't in this PDU, so the source is unknown.
		if g.SourceSSI == g.DestSSI {
			g.Individual = true
			g.SourceSSI = 0
		}
		return
	}

	// No party field on this PDU. If the addressed SSI is one we have seen act as
	// a party, it is an individual radio: mark the grant Individual so discovery /
	// the UI do not treat the subscriber ISSI as a talkgroup. Fold onto the real
	// group when a prior setup mapped this call to one.
	if c.isIndividual(addr) {
		if resolved := c.resolveGroup(msg.CallIdentifier, addr); resolved != 0 && resolved != addr && !c.isIndividual(resolved) {
			g.DestSSI = resolved
			g.SourceSSI = addr
			return
		}
		g.Individual = true
	}
	// This PDU carried no calling/transmitting party, so the source is still
	// blank — restore it from the call's learned source (bound at D-SETUP /
	// D-TX-GRANTED). ETSI strips the SSIs from compressed mid-call signalling, so
	// without this a D-CONNECT / follow-on grant loses the caller after the setup
	// burst. Guarded against re-stamping the destination as its own source.
	c.backfillCallSource(g, msg.CallIdentifier)
}

// backfillCallSource restores a grant's blank source from the persistent
// call-identifier → source-ISSI binding (rememberCallSource). No-op when the
// source is already known, the call identifier is absent, or the learned source
// is the grant's own destination (which would surface a "group X, source X").
func (c *ControlChannel) backfillCallSource(g *VoiceGrant, callID uint16) {
	if g.SourceSSI != 0 || callID == 0 {
		return
	}
	if src := c.resolveCallSource(callID); src != 0 && src != g.DestSSI {
		g.SourceSSI = src
	}
}

// handleCMCE acts on a decoded CMCE call-control PDU that rode in a MAC-RESOURCE
// addressed to m.Address.SSI (the GSSI for a group call). It learns the
// call-identifier → group mapping from setup PDUs so a later release/talker PDU
// resolves to the talkgroup the engine keys calls by, ends a call on D-RELEASE,
// and surfaces the transmitting party on D-TX-GRANTED.
func (c *ControlChannel) handleCMCE(m MACResource, msg CMCEMessage) {
	switch msg.Type {
	case CMCETypeDSetup, CMCETypeDConnect:
		// Map the call identifier to the group SSI — the MAC address a group setup
		// is addressed under — so a later release/talker resolves to the talkgroup
		// the engine tracks the call by. Fall back to the CMCE temporary address
		// only when the MAC address is not an SSI (0). Skip a known individual: a
		// D-CONNECT addressed to the calling party must not overwrite the mapping
		// with the radio ID (which would misattribute the release/talker).
		gssi := m.Address.SSI
		if gssi == 0 {
			gssi = msg.GroupSSI
		}
		if !c.isIndividual(gssi) {
			c.rememberCall(msg.CallIdentifier, gssi)
		}
		// Bind the call identifier to its calling party (D-SETUP carries it) so a
		// later party-less PDU on this call — a D-CONNECT, a follow-on grant — can
		// restore the source that ETSI strips from the compressed mid-call
		// signalling (the individual-call "missing SRC" symptom).
		c.rememberCallSource(msg.CallIdentifier, msg.PartySSI)
		// Surface the decoded basic-service communication type for empirical
		// confirmation of the point-to-point/group enum mapping (spec-derived,
		// capture-unconfirmed — see CMCEMessage.CommsType). On an operator's
		// known individual-vs-group capture, comms_type_raw should split cleanly
		// by call nature; only then does it earn a place in classifyParties.
		if msg.Type == CMCETypeDSetup && msg.HasBasicSvc {
			c.log.Debug("tetra: d-setup basic service",
				"system", c.systemName,
				"call_id", msg.CallIdentifier,
				"comms_type", commsTypeString(msg.CommsType),
				"comms_type_raw", msg.CommsType,
				"circuit_mode", msg.CircuitMode,
				"svc_enc", msg.BasicSvcEnc,
				"dst", m.Address.SSI,
				"src", msg.PartySSI,
				"group", msg.GroupSSI)
		}
	case CMCETypeDRelease, CMCETypeDDisconnect:
		// Both end the call: D-RELEASE from the SwMI, D-DISCONNECT the
		// infrastructure-initiated disconnect (EN 300 392-2 §14.5.1.3.3).
		gssi := c.resolveGroup(msg.CallIdentifier, m.Address.SSI)
		c.publishRelease(gssi, msg.DisconnectCause)
		c.forgetCall(msg.CallIdentifier)
	case CMCETypeDTxGranted:
		if msg.PartySSI != 0 {
			gssi := c.resolveGroup(msg.CallIdentifier, m.Address.SSI)
			c.publishTalker(gssi, msg.PartySSI)
			// The transmitting party is the current source for this call; bind it so
			// a follow-on party-less grant restores it.
			c.rememberCallSource(msg.CallIdentifier, msg.PartySSI)
		}
	case CMCETypeDTxCeased:
		// Transmission boundary within a call; no engine action today (a
		// CallSourceUpdate cannot represent "talker stopped"). Hook for a future
		// KindCallSegment emission.
	}
}
