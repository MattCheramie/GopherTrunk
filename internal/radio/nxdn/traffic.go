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
	hist      [AirFSWDibits]uint8
	histN     int
	histPos   int
	remaining int
	frame     []uint8
	// sinceFrame counts dibits since the last accepted FSW ended, so the
	// next frame's FSW (ending exactly AirFrameDibits later) can be accepted with more
	// symbol errors than a cold search: the flywheel every NXDN decoder
	// runs once it has frame lock.
	sinceFrame int
	locked     bool
	sink       func(frames []VCHFrame)

	// Frames counts FSW-framed frames, LICHOK those whose LICH parity held.
	Frames, LICHOK uint64
}

const (
	airFSWColdTolerance = 1 // symbol errors allowed when searching
	airFSWLockTolerance = 3 // ... at the expected position when locked
)

// NewTrafficChannel builds a traffic-frame decoder. sink receives the
// decoded voice frames of each voice frame (never called with an empty
// slice).
func NewTrafficChannel(sink func(frames []VCHFrame)) *TrafficChannel {
	return &TrafficChannel{
		frame: make([]uint8, 0, airPostFSWDibits),
		sink:  sink,
	}
}

// Process consumes a window of raw dibits and returns baseIdx+len(dibits).
func (t *TrafficChannel) Process(dibits []uint8, baseIdx int) int {
	for _, d := range dibits {
		t.hist[t.histPos] = d
		t.histPos = (t.histPos + 1) % AirFSWDibits
		if t.histN < AirFSWDibits {
			t.histN++
		}
		t.sinceFrame++
		if t.remaining > 0 {
			t.frame = append(t.frame, d)
			t.remaining--
			if t.remaining == 0 {
				t.handleFrame(t.frame)
				t.frame = t.frame[:0]
			}
			continue
		}
		if t.histN < AirFSWDibits {
			continue
		}
		tol := airFSWColdTolerance
		if t.locked && t.sinceFrame == AirFrameDibits {
			tol = airFSWLockTolerance
		}
		if t.fswErrors() <= tol {
			t.remaining = airPostFSWDibits
			t.frame = t.frame[:0]
			t.sinceFrame = 0
		} else if t.sinceFrame > AirFrameDibits {
			t.locked = false
		}
	}
	return baseIdx + len(dibits)
}

func (t *TrafficChannel) fswErrors() int {
	errs := 0
	idx := t.histPos
	for k := 0; k < AirFSWDibits; k++ {
		if t.hist[idx] != AirFSW[k] {
			errs++
		}
		idx = (idx + 1) % AirFSWDibits
	}
	return errs
}

func (t *TrafficChannel) handleFrame(post []uint8) {
	t.Frames++
	DescrambleAir(post)
	lich, frames := DecodeAirVoice(post)
	t.locked = lich.ParityOK
	if !lich.ParityOK {
		return
	}
	t.LICHOK++
	if len(frames) > 0 && t.sink != nil {
		t.sink(frames)
	}
}

// Reset clears the sync and partial-frame state; call on retune.
func (t *TrafficChannel) Reset() {
	t.histN, t.histPos, t.remaining, t.sinceFrame = 0, 0, 0, 0
	t.locked = false
	t.frame = t.frame[:0]
}
