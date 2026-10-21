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

	// Frames counts FSW-framed frames, LICHOK those whose LICH parity held.
	Frames, LICHOK uint64
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
	if len(frames) > 0 && t.sink != nil {
		t.sink(frames)
	}
	return true
}

// Reset clears the sync and partial-frame state; call on retune.
func (t *TrafficChannel) Reset() { t.framer.reset() }
