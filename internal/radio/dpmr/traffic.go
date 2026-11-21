package dpmr

// TrafficChannel frames a dPMR traffic channel in the air format of
// airframe.go and hands out the vocoder frames of a voice transmission.
//
// It locks onto the superframe grid at an FS2 and then follows the
// transmission two payload frames at a time (an FS2 frame and the
// channel-code frame after it, 384 dibits), until an end frame (FS3) or
// until a pair no longer looks like dPMR.
//
// A sync word alone proves little: FS2 is 12 symbols, and a detector that
// tolerates one wrong symbol matches noise about once every three minutes
// in either polarity. So a first FS2 only locks the framer when the frames
// behind it agree: a control channel that passes its CRC7 with the frame
// number that position must carry, or a clean channel code followed by the
// next sync word 384 dibits on. Until then nothing is handed out.
//
// TCH blocks are decoded only once a control channel has said the
// transmission carries voice; a data transmission is framed and counted
// but its payload is left alone.
//
// The dibit stream and baseIdx contract match ControlChannel.Process
// (continuous stream, monotonic baseIdx). Not safe for concurrent use.
//
// The format is pinned to the standard and to two decoders (see
// airframe.go and its tests); no dPMR transmission has been received
// through it yet.
type TrafficChannel struct {
	sink func(frames []TCHFrame)

	// OnEnd, when set, is called for each end frame received while the
	// framer is locked: the transmitter released the channel.
	OnEnd func()

	buf      []uint8 // done consumed dibits kept for a look back, then the rest
	done     int
	pair     []uint8 // scratch: one polarity-corrected frame pair + next sync
	locked   bool
	inverted bool // the stream's sign bits are flipped

	modeKnown bool // a control channel of this lock has named the mode
	voice     bool
	codeKnown bool   // this lock has seen a clean channel code
	codeSign  uint16 // its twelve sign bits
	idHalf    [4]uint16
	idHave    [4]bool

	stats TrafficStats
}

// TrafficStats counts what a TrafficChannel has seen.
type TrafficStats struct {
	// Locks is how often the framer locked onto a superframe grid, Drops
	// how often it gave a lock up because a frame pair could not be
	// trusted, EndFrames how many end frames (FS3) it received.
	Locks, Drops, EndFrames uint64
	// Frames is the number of payload frames accepted while locked.
	Frames uint64
	// CCHValid and CCHInvalid count their control channels by checksum.
	CCHValid, CCHInvalid uint64
	// VoiceFrames is the number of vocoder frames handed out. DataFrames
	// counts payload frames of a data transmission, UnknownFrames those
	// accepted before any control channel had named the mode.
	VoiceFrames, DataFrames, UnknownFrames uint64
	// Inverted reports that the last lock was on an inverted spectrum.
	Inverted bool
	// ChannelCode is the last cleanly received 24-bit channel code.
	ChannelCode     uint32
	HaveChannelCode bool
	// CalledID and CallingID are the 24-bit addresses assembled from the
	// control channels of one superframe.
	CalledID, CallingID     uint32
	HaveCalled, HaveCalling bool
}

// How strict the framer is. See the TrafficChannel comment for why a first
// FS2 needs corroboration.
const (
	// airHuntSyncTolerance is how many of FS2's 12 dibits may be wrong
	// when hunting.
	airHuntSyncTolerance = 1
	// airLockCorrectedMax is how many bits the Hamming decode may have
	// changed in a control channel that is to confirm a lock. A block of
	// noise is "correctable" 13 times in 16, so the checksum alone would
	// pass one noise field in 128.
	airLockCorrectedMax = 2
	// airTrackSignTolerance is how many of FS2's 12 sign bits may be wrong
	// once locked, where the sync word's position is already known.
	airTrackSignTolerance = 2
	// airTrackOuterMin is how many of the channel code's 12 symbols must
	// be outer ones for the field to count as a channel code once locked.
	airTrackOuterMin = 9
	// airCleanOuterMin is the same count for a channel code that is to
	// confirm a lock or be reported.
	airCleanOuterMin = 11
	// airBacktrackDibits is how far back the framer looks for the grid
	// when a pair is not where it should be. A stream that lost samples
	// brings the next sync word early, by the length of the gap: 48 dibits
	// are 20 ms, more than the gaps a dropped USB transfer or datagram
	// leaves.
	airBacktrackDibits = 48
)

// NewTrafficChannel builds a traffic-channel decoder. sink receives the
// vocoder frames of each voice payload frame, four at a time, and is never
// called with an empty slice.
func NewTrafficChannel(sink func(frames []TCHFrame)) *TrafficChannel {
	return &TrafficChannel{
		sink: sink,
		buf:  make([]uint8, 0, 4*airPairDibits),
		pair: make([]uint8, airPairDibits+AirSyncDibits),
	}
}

// Stats returns what the channel has counted so far.
func (t *TrafficChannel) Stats() TrafficStats { return t.stats }

// Locked reports whether the framer is following a transmission.
func (t *TrafficChannel) Locked() bool { return t.locked }

// Reset drops the lock and any buffered dibits; call on a retune. The
// counters are kept.
func (t *TrafficChannel) Reset() {
	t.buf = t.buf[:0]
	t.done = 0
	t.unlock()
}

func (t *TrafficChannel) unlock() {
	t.locked = false
	t.modeKnown = false
	t.voice = false
	t.codeKnown = false
	t.idHave = [4]bool{}
}

// Process consumes a window of dibits and returns baseIdx+len(dibits).
func (t *TrafficChannel) Process(dibits []uint8, baseIdx int) int {
	t.buf = append(t.buf, dibits...)
	off := t.done
	for {
		rest := t.buf[off:]
		if t.locked {
			if len(rest) < AirSyncDibits {
				break
			}
			if t.isEndFrame(rest) {
				t.stats.EndFrames++
				t.unlock()
				off += AirSyncDibits
				if t.OnEnd != nil {
					t.OnEnd()
				}
				continue
			}
			if len(rest) < airPairDibits {
				break
			}
			p := t.decodePair(rest, t.inverted)
			switch {
			case p.trusted() && p.secondTrusted():
				t.accept(p, true)
				off += airPairDibits
			case p.trusted():
				// The first frame is where the grid says and the second
				// is not: the stream lost dibits in between. Take the
				// first and look for the grid again from the second.
				t.accept(p, false)
				t.stats.Drops++
				t.unlock()
				off += AirFrameDibits
			default:
				// Not a superframe at this position: the signal went
				// away, or the stream lost dibits just before and the
				// sync word came early. Look for it from a little back.
				t.stats.Drops++
				t.unlock()
				if off -= airBacktrackDibits; off < 0 {
					off = 0
				}
			}
			continue
		}

		i, inverted := findFS2(rest)
		if i < 0 {
			// Keep what could be the start of a sync word.
			if keep := AirSyncDibits - 1; len(rest) > keep {
				off += len(rest) - keep
			}
			break
		}
		off += i
		rest = t.buf[off:]
		if len(rest) < airPairDibits+AirSyncDibits {
			break // wait for the pair and the sync word after it
		}
		p := t.decodePair(rest, inverted)
		if !p.confirms(t.pair[airPairDibits:]) {
			off++
			continue
		}
		t.locked = true
		t.inverted = inverted
		t.stats.Locks++
		t.stats.Inverted = inverted
		t.accept(p, true)
		off += airPairDibits
	}
	// Keep a little of what has been consumed, for the look back.
	drop := off - airBacktrackDibits
	if drop < 0 {
		drop = 0
	}
	n := copy(t.buf, t.buf[drop:])
	t.buf = t.buf[:n]
	t.done = off - drop
	return baseIdx + len(dibits)
}

// findFS2 returns the first position in d where FS2 matches in either
// polarity, and which, or -1.
func findFS2(d []uint8) (pos int, inverted bool) {
	const flip = 0xAAAAAA // the sign bit of every dibit
	for i := 0; i+AirSyncDibits <= len(d); i++ {
		if airSyncDistance(d[i:], AirFS2) <= airHuntSyncTolerance {
			return i, false
		}
		if airSyncDistance(d[i:], AirFS2^flip) <= airHuntSyncTolerance {
			return i, true
		}
	}
	return -1, false
}

// isEndFrame reports whether d opens with FS3 in the lock's polarity.
func (t *TrafficChannel) isEndFrame(d []uint8) bool {
	sign := airFS3Sign
	if t.inverted {
		sign ^= 0xFFF
	}
	return airSignDistance(d, sign) <= 1 && airOuterCount(d) >= AirSyncDibits-2
}

// airPair is one FS2 frame and the channel-code frame after it.
type airPair struct {
	syncOK     bool // FS2 where it belongs
	a, b       AirCCH
	okA, okB   bool // checksum good and the frame number fits the position
	code       uint32
	codeOuter  int
	codeOK     bool    // a channel code where it belongs
	tchA, tchB []uint8 // 144 dibits each, polarity-corrected
}

// decodePair reads the pair at the start of d, which must hold at least
// airPairDibits dibits, into t.pair with the polarity undone. Up to
// AirSyncDibits further dibits are copied behind it for confirms.
func (t *TrafficChannel) decodePair(d []uint8, inverted bool) airPair {
	n := copy(t.pair, d)
	for i := n; i < len(t.pair); i++ {
		t.pair[i] = 0
	}
	if inverted {
		airInvert(t.pair[:n])
	}
	fa, fb := t.pair[:AirFrameDibits], t.pair[AirFrameDibits:airPairDibits]
	var p airPair
	p.syncOK = airSignDistance(fa, airFS2Sign) <= airTrackSignTolerance
	p.a, p.okA = DecodeAirCCH(fa[AirSyncDibits : AirSyncDibits+AirCCHDibits])
	p.okA = p.okA && p.a.FrameNumber%2 == 0 // FS2 opens frames 1 and 3
	p.tchA = fa[AirSyncDibits+AirCCHDibits:]
	p.code, p.codeOuter = AirChannelCode(fb)
	// Outer symbols alone say little after a small slip, when most of the
	// field still overlaps the real channel code. Once the lock knows the
	// code, the field has to carry it.
	p.codeOK = p.codeOuter >= airTrackOuterMin &&
		(!t.codeKnown || airSignDistance(fb, t.codeSign) <= airTrackSignTolerance)
	p.b, p.okB = DecodeAirCCH(fb[AirSyncDibits : AirSyncDibits+AirCCHDibits])
	p.okB = p.okB && p.b.FrameNumber%2 == 1 // the channel code opens frames 2 and 4
	p.tchB = fb[AirSyncDibits+AirCCHDibits:]
	return p
}

// trusted reports whether a pair on a locked grid still is one.
func (p airPair) trusted() bool {
	return p.okA || p.okB || (p.syncOK && p.codeOK)
}

// secondTrusted reports whether the second frame of a trusted pair is
// where the grid puts it.
func (p airPair) secondTrusted() bool {
	return p.okB || p.codeOK
}

// confirms reports whether a pair found by its sync word alone is one;
// next is the 12 dibits after it.
func (p airPair) confirms(next []uint8) bool {
	if p.okA && p.a.Corrected <= airLockCorrectedMax {
		return true
	}
	if p.okB && p.b.Corrected <= airLockCorrectedMax {
		return true
	}
	if p.codeOuter < airCleanOuterMin {
		return false
	}
	return airSyncDistance(next, AirFS2) <= airHuntSyncTolerance ||
		airSyncDistance(next, AirFS3) <= airHuntSyncTolerance
}

// accept takes the first frame of a trusted pair, and the second with it
// unless the caller found it out of place.
func (t *TrafficChannel) accept(p airPair, second bool) {
	t.frame(p.a, p.okA, p.tchA)
	if !second {
		return
	}
	if p.codeOuter >= airCleanOuterMin {
		t.stats.ChannelCode, t.stats.HaveChannelCode = p.code, true
		if !t.codeKnown {
			t.codeKnown = true
			t.codeSign = airSignWord(t.pair[AirFrameDibits:])
		}
	}
	t.frame(p.b, p.okB, p.tchB)
}

func (t *TrafficChannel) frame(cch AirCCH, ok bool, tch []uint8) {
	t.stats.Frames++
	if ok {
		t.stats.CCHValid++
		t.modeKnown, t.voice = true, cch.Voice()
		t.address(cch)
	} else {
		t.stats.CCHInvalid++
	}
	switch {
	case !t.modeKnown:
		t.stats.UnknownFrames++
	case !t.voice:
		t.stats.DataFrames++
	default:
		frames := ExtractTCHFrames(tch)
		t.stats.VoiceFrames += uint64(len(frames))
		if len(frames) > 0 && t.sink != nil {
			t.sink(frames)
		}
	}
}

// address collects the two halves of the called ID (frames 1 and 2) and of
// the calling ID (frames 3 and 4). A half only pairs with the other half
// of the same superframe.
func (t *TrafficChannel) address(cch AirCCH) {
	fn := cch.FrameNumber & 3
	if fn == 0 {
		t.idHave = [4]bool{}
	}
	t.idHalf[fn], t.idHave[fn] = cch.IDHalf&0xFFF, true
	switch {
	case fn == 1 && t.idHave[0]:
		t.stats.CalledID = uint32(t.idHalf[0])<<12 | uint32(t.idHalf[1])
		t.stats.HaveCalled = true
	case fn == 3 && t.idHave[2]:
		t.stats.CallingID = uint32(t.idHalf[2])<<12 | uint32(t.idHalf[3])
		t.stats.HaveCalling = true
	}
}
