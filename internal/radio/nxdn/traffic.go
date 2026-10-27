package nxdn

// TrafficChannel decodes NXDN voice from the raw dibit stream of a traffic
// channel, in the real air format (airframe.go): it finds the 20-bit FSW,
// collects the 182 dibits that follow, descrambles them, checks the LICH
// and decodes the AMBE+2 voice frames the LICH announces, delivering them
// to a sink callback. NXDN48 and NXDN96 share the frame, so one decoder
// serves both; the receiver supplies the symbol rate.
//
// The dibit stream + baseIdx contract matches ControlChannel.Process
// (continuous stream, monotonic baseIdx).
type TrafficChannel struct {
	framer airFramer
	sink   func(frames []VCHFrame)

	// OnFrame, when set, receives every LICH-valid frame — voice or not —
	// with its LICH, its 60 SACCH channel bits and the voice frames it
	// carried. The voice chain uses it to follow the SACCH superframe
	// (VCALL names the cipher and key) and to keep a scrambler's keystream
	// aligned across frames that carried no voice.
	OnFrame func(TrafficFrame)

	// Frames counts FSW-framed frames, LICHOK those whose LICH parity held.
	Frames, LICHOK uint64
}

// TrafficFrame is one framed, descrambled, LICH-valid traffic-channel
// frame as TrafficChannel.OnFrame delivers it.
type TrafficFrame struct {
	// Index counts LICH-valid frames since the channel was built or Reset.
	Index uint64
	LICH  AirLICH
	// SACCH is the frame's 60 SACCH channel bits (0/1, on-air order),
	// ready for DecodeSACCH.
	SACCH []byte
	// Voice is the frames DecodeAirVoice recovered (Slot set), nil for a
	// frame with no voice.
	Voice []VCHFrame
}

// airFramer finds the real-air FSW in a dibit stream and hands each frame's
// 182 post-FSW dibits to onFrame, which descrambles and decodes them and
// reports whether the frame was good (its LICH parity held). It runs the
// flywheel every NXDN decoder runs once it has frame lock: a frame whose FSW
// ends exactly one frame after the last good one is accepted with more symbol
// errors than a cold search. Shared by the traffic and control channels.
type airFramer struct {
	hist      [AirFSWDibits]uint8
	histN     int
	histPos   int
	remaining int
	frame     []uint8
	// sinceFrame counts dibits since the last accepted FSW ended, so the
	// next frame's FSW (ending exactly AirFrameDibits later) can be accepted with more
	// symbol errors than a cold search.
	sinceFrame int
	locked     bool
	onFrame    func(post []uint8) bool
}

func (f *airFramer) process(dibits []uint8) {
	for _, d := range dibits {
		f.hist[f.histPos] = d
		f.histPos = (f.histPos + 1) % AirFSWDibits
		if f.histN < AirFSWDibits {
			f.histN++
		}
		f.sinceFrame++
		if f.remaining > 0 {
			f.frame = append(f.frame, d)
			f.remaining--
			if f.remaining == 0 {
				f.locked = f.onFrame(f.frame)
				f.frame = f.frame[:0]
			}
			continue
		}
		if f.histN < AirFSWDibits {
			continue
		}
		tol := airFSWColdTolerance
		if f.locked && f.sinceFrame == AirFrameDibits {
			tol = airFSWLockTolerance
		}
		if f.fswErrors() <= tol {
			f.remaining = airPostFSWDibits
			f.frame = f.frame[:0]
			f.sinceFrame = 0
		} else if f.sinceFrame > AirFrameDibits {
			f.locked = false
		}
	}
}

func (f *airFramer) fswErrors() int {
	errs := 0
	idx := f.histPos
	for k := 0; k < AirFSWDibits; k++ {
		if f.hist[idx] != AirFSW[k] {
			errs++
		}
		idx = (idx + 1) % AirFSWDibits
	}
	return errs
}

func (f *airFramer) reset() {
	f.histN, f.histPos, f.remaining, f.sinceFrame = 0, 0, 0, 0
	f.locked = false
	f.frame = f.frame[:0]
}

const (
	airFSWColdTolerance = 1 // symbol errors allowed when searching
	airFSWLockTolerance = 3 // ... at the expected position when locked
)

// NewTrafficChannel builds a traffic-frame decoder. sink receives the
// decoded voice frames of each voice frame (never called with an empty
// slice).
func NewTrafficChannel(sink func(frames []VCHFrame)) *TrafficChannel {
	t := &TrafficChannel{sink: sink}
	t.framer = airFramer{frame: make([]uint8, 0, airPostFSWDibits), onFrame: t.handleFrame}
	return t
}

// Process consumes a window of raw dibits and returns baseIdx+len(dibits).
func (t *TrafficChannel) Process(dibits []uint8, baseIdx int) int {
	t.framer.process(dibits)
	return baseIdx + len(dibits)
}

func (t *TrafficChannel) handleFrame(post []uint8) bool {
	t.Frames++
	DescrambleAir(post)
	lich, frames := DecodeAirVoice(post)
	if !lich.ParityOK {
		return false
	}
	t.LICHOK++
	if t.OnFrame != nil {
		t.OnFrame(TrafficFrame{Index: t.LICHOK - 1, LICH: lich, SACCH: AirSACCHBits(post), Voice: frames})
	}
	if len(frames) > 0 && t.sink != nil {
		t.sink(frames)
	}
	return true
}

// AirSACCHBits returns the 60 SACCH channel bits of one descrambled
// post-FSW frame, two per dibit MSB-first, in on-air order.
func AirSACCHBits(post []uint8) []byte {
	if len(post) < AirLICHDibits+AirSACCHDibits {
		return nil
	}
	out := make([]byte, 2*AirSACCHDibits)
	for k, d := range post[AirLICHDibits : AirLICHDibits+AirSACCHDibits] {
		out[2*k] = d >> 1 & 1
		out[2*k+1] = d & 1
	}
	return out
}

// Reset clears the sync and partial-frame state; call on retune.
func (t *TrafficChannel) Reset() { t.framer.reset() }
