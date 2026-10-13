package conventional

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25p1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
)

// P25 NAC values a radio programs to "receive any NAC" (TIA-102.BAAC):
// 0xF7E receives every NAC, 0xF7F also repeats every NAC. As a scan-list
// filter both mean "any P25 carrier, but only a P25 one".
const (
	NACReceiveAny       = 0xF7E
	NACReceiveAnyRepeat = 0xF7F
)

// nacHold is how long a matching NID keeps the NAC gate open. A voice
// transmission carries a NID on every data unit (an LDU each 180 ms), so
// 750 ms rides out three or four frames lost to fades without letting a
// finished transmission hold the channel.
const nacHold = 750 * time.Millisecond

// nacMinDwell is the shortest scan window that lets the NAC gate open on a
// live transmission: the receiver's loops settle and the next frame sync +
// NID arrives (one LDU is 180 ms) with margin for a lost first frame.
const nacMinDwell = 500 * time.Millisecond

// nacSyncTolerance is the frame-sync mismatch (in dibits) the gate accepts,
// the voice-channel LDU assembler's default. A false sync almost never
// survives the NID's BCH(63,16) decode on top of it.
const nacSyncTolerance = 4

// ParseNAC parses a scan-list nac value: three hex digits, with or without
// a 0x prefix ("293", "0x293", "F7E"). A NAC is conventionally written in
// hex, so a bare "293" is 0x293, never decimal 293.
func ParseNAC(s string) (uint16, error) {
	t := strings.TrimSpace(s)
	t = strings.TrimPrefix(strings.TrimPrefix(t, "0x"), "0X")
	if t == "" {
		return 0, fmt.Errorf("nac: empty")
	}
	v, err := strconv.ParseUint(t, 16, 16)
	if err != nil || v > 0xFFF {
		return 0, fmt.Errorf("nac %q must be three hex digits, 000..FFF (e.g. 293)", s)
	}
	return uint16(v), nil
}

// NACDetector is the P25 analogue of the CTCSS gate (#1262): it runs the
// production P25 Phase 1 receiver on the channel's own narrowband IQ, finds
// each frame sync, decodes the NID that follows it, and reports present
// while a NID carrying the wanted NAC was decoded within nacHold. A NID with
// any other NAC closes it at once, so a different system's transmission on
// the same frequency neither opens the scanner's squelch nor holds a dwell.
//
// It satisfies toneDetector, so the scanner gates on it exactly as on a
// CTCSS / DCS tone: carrier power AND a matching NAC.
type NACDetector struct {
	fe   *toneFrontEnd
	rx   *p25p1rx.Receiver
	want uint16
	any  bool

	dibits []uint8 // received dibits not yet searched for a frame sync

	inRate      float64
	samples     int64 // input samples processed since Reset
	lastMatch   int64 // input-sample index of the last matching NID; -1 = none
	holdSamples int64

	lastNAC    uint16 // most recent NAC decoded, for the gate's log lines
	sawForeign bool   // a NID with another NAC was decoded since the last TakeForeign
}

// NewNACDetector builds a NAC gate for IQ at sampleHz. demodMode is the
// channel's P25 Phase 1 demod ("" / "c4fm" or "cqpsk"). Returns nil when
// sampleHz is not set.
func NewNACDetector(sampleHz float64, nac uint16, demodMode string) *NACDetector {
	if sampleHz <= 0 {
		return nil
	}
	mode, _ := p25p1rx.ParseDemodMode(demodMode)
	d := &NACDetector{
		fe:          newToneFrontEnd(sampleHz),
		want:        nac,
		any:         nac == NACReceiveAny || nac == NACReceiveAnyRepeat,
		inRate:      sampleHz,
		lastMatch:   -1,
		holdSamples: int64(nacHold.Seconds() * sampleHz),
	}
	d.rx = p25p1rx.New(p25p1rx.Options{
		SampleRateHz: d.fe.rate,
		DeviationHz:  1800,
		DemodMode:    mode,
		DibitSink:    func(b []uint8, _ int) { d.dibits = append(d.dibits, b...) },
	})
	return d
}

// Process feeds one IQ chunk and reports whether the wanted NAC is present.
func (d *NACDetector) Process(iq []complex64) bool {
	d.rx.Process(d.fe.channel(iq))
	d.samples += int64(len(iq))
	d.scan()
	return d.Present()
}

// scan looks for frame syncs in the buffered dibits and decodes the NID
// after each. Dibits too close to the end to hold a whole sync + NID stay
// buffered for the next chunk.
func (d *NACDetector) scan() {
	pos := 0
	for ; pos+phase1.FrameNIDDibits <= len(d.dibits); pos++ {
		if syncMismatch(d.dibits[pos:pos+len(phase1.FrameSyncWord)]) > nacSyncTolerance {
			continue
		}
		nid, err := phase1.ParseFrameNID(d.dibits[pos:])
		if err != nil {
			continue
		}
		d.lastNAC = nid.NAC
		if d.any || nid.NAC == d.want {
			d.lastMatch = d.samples
		} else {
			// Another system keyed up on this frequency: the gate shuts now,
			// not when the hold runs out.
			d.lastMatch = -1
			d.sawForeign = true
		}
		pos += phase1.FrameNIDDibits - 1
	}
	n := copy(d.dibits, d.dibits[pos:])
	d.dibits = d.dibits[:n]
}

func syncMismatch(w []uint8) int {
	n := 0
	for i, v := range w {
		if v != phase1.FrameSyncWord[i] {
			n++
		}
	}
	return n
}

// Present reports whether a matching NID was decoded within nacHold.
func (d *NACDetector) Present() bool {
	return d.lastMatch >= 0 && d.samples-d.lastMatch <= d.holdSamples
}

// Reset clears all state; the scanner calls it on every retune.
func (d *NACDetector) Reset() {
	d.fe.reset()
	d.rx.Reset()
	d.dibits = d.dibits[:0]
	d.samples = 0
	d.lastMatch = -1
	d.sawForeign = false
}

// TakeForeign reports (and clears) whether a NID with a NAC other than the
// wanted one was decoded since the last call, with that NAC.
func (d *NACDetector) TakeForeign() (uint16, bool) {
	saw := d.sawForeign
	d.sawForeign = false
	return d.lastNAC, saw
}

// Want returns the NAC the gate opens on.
func (d *NACDetector) Want() uint16 { return d.want }

// buildNACDetector returns the NAC gate for a p25 channel with a NAC set,
// or nil (no gate) otherwise.
func buildNACDetector(ch Channel, sampleHz float64, log *slog.Logger) toneDetector {
	if ch.Mode != ModeP25 || !ch.NACSet {
		return nil
	}
	if d := NewNACDetector(sampleHz, ch.NAC, ch.P25Phase1DemodMode); d != nil {
		return d
	}
	if log != nil {
		log.Warn("conv: NAC filter configured but scanner sample rate is zero; NAC gate disabled — every P25 carrier passes",
			"freq_hz", ch.FrequencyHz, "nac", fmt.Sprintf("%03X", ch.NAC))
	}
	return nil
}
