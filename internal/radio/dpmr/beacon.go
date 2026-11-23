package dpmr

// BeaconFramer frames a Mode 3 beacon downlink (clause 4.2.3.7): a
// continuous stream of 264-dibit framesets, each a message slot (MI0, the
// channel code, MI1) and a SYScast slot (three SYScasts and the FS1 that
// opens the next message) or an appended-data frame in its place.
//
// It hunts for FS1 and locks when the message behind it decodes: a
// CRC8-valid copy of MI0 or MI1 with at most a few Hamming corrections.
// FS1 alone is already strong (24 symbols; at the hunt tolerance noise
// matches it about once in 10^10 positions), and the message check keeps a
// traffic-channel header frame, which also opens with FS1, from locking it
// for longer than its own frame. Locked, it follows the grid one frameset
// at a time and hands each to OnFrameset; two framesets in a row without a
// valid message, a SYScast, appended data or the FS1 where the grid puts it
// end the lock, and the hunt starts again a little before the lost grid,
// because a stream that lost samples brings the next FS1 early.
//
// The dibit stream and baseIdx contract match TrafficChannel.Process. Not
// safe for concurrent use.
type BeaconFramer struct {
	// OnFrameset receives every frameset framed while locked.
	OnFrameset func(BeaconFrameset)

	buf      []uint8
	off      int // locked: the next message slot; hunting: where to search from
	locked   bool
	inverted bool
	misses   int
	scratch  []uint8

	stats BeaconStats
}

// BeaconFrameset is one framed frameset.
type BeaconFrameset struct {
	// Message is the decoded message field, valid when MessageOK. Either
	// copy may have supplied it; MessageCorrected counts the Hamming
	// corrections in the one used.
	Message          BeaconMessage
	MessageOK        bool
	MessageCorrected int
	// SYScast holds the three SYScast fields when the second slot carried
	// a SYScast frame.
	SYScast [3]SYScastField
	// Appended is an appended-data frame (72 bits, coded like a message)
	// that took the SYScast slot, valid when AppendedOK.
	Appended   [72]uint8
	AppendedOK bool
	// NextFS1 is whether the FS1 that ends the SYScast slot was there.
	NextFS1 bool
	// ChannelCode is the 24-bit channel code between MI0 and MI1.
	ChannelCode uint32
}

// BeaconStats counts what a BeaconFramer has seen.
type BeaconStats struct {
	Locks, Drops, Framesets   uint64
	Messages, MessagesBad     uint64
	SYScastFields, SYScastBad uint64
	AppendedFrames            uint64
	Inverted                  bool
}

const (
	// beaconHuntFS1Tolerance is how many of FS1's 24 dibits may be wrong
	// in a hunt; beaconTrackFS1Tolerance where the grid expects it.
	beaconHuntFS1Tolerance  = 3
	beaconTrackFS1Tolerance = 5
	// beaconLockCorrectedMax is how many Hamming corrections a message
	// that confirms a lock may need. A noise field passes CRC8 after
	// correction about 5 times in 10 000; with this few corrections far
	// fewer.
	beaconLockCorrectedMax = 3
	// beaconMaxMisses framesets without evidence end a lock.
	beaconMaxMisses = 2
	// beaconBacktrackDibits is how far back the hunt starts after a lost
	// lock (20 ms).
	beaconBacktrackDibits = 48
)

// NewBeaconFramer builds a framer that hands each frameset to onFrameset.
func NewBeaconFramer(onFrameset func(BeaconFrameset)) *BeaconFramer {
	return &BeaconFramer{
		OnFrameset: onFrameset,
		buf:        make([]uint8, 0, 4*BeaconFramesetDibits),
		scratch:    make([]uint8, BeaconFramesetDibits),
	}
}

// Stats returns the framer's counters.
func (f *BeaconFramer) Stats() BeaconStats { return f.stats }

// Locked reports whether the framer is following a beacon.
func (f *BeaconFramer) Locked() bool { return f.locked }

// Reset drops the lock and any buffered dibits; call on a retune. The
// counters are kept.
func (f *BeaconFramer) Reset() {
	f.buf = f.buf[:0]
	f.off = 0
	f.locked = false
	f.misses = 0
}

// Process consumes a window of dibits and returns baseIdx+len(dibits).
func (f *BeaconFramer) Process(dibits []uint8, baseIdx int) int {
	f.buf = append(f.buf, dibits...)
	for {
		if f.locked {
			if len(f.buf)-f.off < BeaconFramesetDibits {
				break
			}
			fs, evidence := f.decode(f.buf[f.off:f.off+BeaconFramesetDibits], f.inverted)
			f.stats.Framesets++
			if f.OnFrameset != nil {
				f.OnFrameset(fs)
			}
			if evidence {
				f.misses = 0
				f.off += BeaconFramesetDibits
				continue
			}
			f.misses++
			if f.misses < beaconMaxMisses {
				f.off += BeaconFramesetDibits
				continue
			}
			f.locked = false
			f.stats.Drops++
			f.off -= beaconBacktrackDibits
			if f.off < 0 {
				f.off = 0
			}
			continue
		}
		// Hunting: FS1, then the message slot behind it.
		need := BeaconFS1Dibits + BeaconSlotDibits
		found := false
		for i := f.off; i+need <= len(f.buf); i++ {
			inverted, ok := beaconFS1At(f.buf[i:])
			if !ok {
				continue
			}
			copy(f.scratch, f.buf[i+BeaconFS1Dibits:i+need])
			if inverted {
				airInvert(f.scratch[:BeaconSlotDibits])
			}
			_, corrected, ok := decodeBeaconMessageSlot(f.scratch[:BeaconSlotDibits])
			if !ok || corrected > beaconLockCorrectedMax {
				continue
			}
			f.locked, f.inverted, f.misses = true, inverted, 0
			f.stats.Locks++
			f.stats.Inverted = inverted
			f.off = i + BeaconFS1Dibits
			found = true
			break
		}
		if !found {
			// Keep what could be the start of FS1 and its message.
			if keep := need - 1; len(f.buf)-f.off > keep {
				f.off = len(f.buf) - keep
			}
			break
		}
	}
	// Keep a little of what has been consumed, for the look back.
	drop := f.off - beaconBacktrackDibits - BeaconFS1Dibits
	if drop > 0 {
		n := copy(f.buf, f.buf[drop:])
		f.buf = f.buf[:n]
		f.off -= drop
	}
	return baseIdx + len(dibits)
}

// beaconFS1At reports whether d opens with FS1, in which polarity. An
// inverted FS1 is FS4, which a beacon downlink does not send.
func beaconFS1At(d []uint8) (inverted, ok bool) {
	if beaconFS1Distance(d) <= beaconHuntFS1Tolerance {
		return false, true
	}
	n := 0
	for i := 0; i < BeaconFS1Dibits; i++ {
		if d[i]^2 != uint8(AirFS1>>uint(2*(BeaconFS1Dibits-1-i)))&3 {
			n++
		}
	}
	return true, n <= beaconHuntFS1Tolerance
}

// decodeBeaconMessageSlot decodes MI0, and MI1 if MI0 fails, from a
// polarity-corrected message slot.
func decodeBeaconMessageSlot(d []uint8) (BeaconMessage, int, bool) {
	bits, corrected, ok := DecodeBeaconMI(d[:BeaconMIDibits])
	if !ok {
		bits, corrected, ok = DecodeBeaconMI(d[BeaconMIDibits+AirSyncDibits : BeaconSlotDibits])
	}
	return BeaconMessage{Bits: bits}, corrected, ok
}

// decode reads one frameset that starts at a message slot.
func (f *BeaconFramer) decode(d []uint8, inverted bool) (fs BeaconFrameset, evidence bool) {
	copy(f.scratch, d)
	s := f.scratch[:BeaconFramesetDibits]
	if inverted {
		airInvert(s)
	}
	fs.Message, fs.MessageCorrected, fs.MessageOK = decodeBeaconMessageSlot(s[:BeaconSlotDibits])
	if fs.MessageOK {
		f.stats.Messages++
	} else {
		f.stats.MessagesBad++
	}
	fs.ChannelCode, _ = AirChannelCode(s[BeaconMIDibits:])
	slot := s[BeaconSlotDibits:]
	fs.NextFS1 = beaconFS1Distance(slot[BeaconSYScastDibits:]) <= beaconTrackFS1Tolerance
	fs.SYScast = DecodeBeaconSYScast(slot[:BeaconSYScastDibits])
	anySY := false
	for _, fd := range fs.SYScast {
		if fd.OK {
			anySY = true
			f.stats.SYScastFields++
		} else {
			f.stats.SYScastBad++
		}
	}
	if !fs.NextFS1 && !anySY {
		// No SYScast frame: the slot may hold appended data instead.
		var app BeaconMessage
		app, _, fs.AppendedOK = decodeBeaconMessageSlot(slot)
		if fs.AppendedOK {
			fs.Appended = app.Bits
			f.stats.AppendedFrames++
			fs.SYScast = [3]SYScastField{}
		}
	}
	return fs, fs.MessageOK || fs.NextFS1 || anySY || fs.AppendedOK
}
