package hunt

import (
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// decodeParams carries everything decodeAndAccumulate needs to identify and
// decode one capture, independent of whether the IQ comes from a file (offline
// Discover) or a live capture buffer (LiveHunter).
type decodeParams struct {
	// Protocol forces a decoder; trunking.ProtocolUnknown auto-identifies.
	Protocol     trunking.Protocol
	Format       siglab.SampleFormat
	SampleRateHz float64
	// FrequencyHz is the capture's nominal center (recorded as the control
	// channel when the lock doesn't carry an absolute frequency).
	FrequencyHz uint32
	AutoTune    bool
	Conjugate   bool
	IQCorrect   bool
	// IdentifyMaxSamples bounds the prefix the identifier scans (0 ⇒ default).
	IdentifyMaxSamples int64
	// MinConfidence gates auto-identified captures (0 ⇒ 0.40).
	MinConfidence float64
	Log           *slog.Logger
}

// decodeAndAccumulate identifies (unless a protocol is forced), decodes, and
// folds one capture read from r into sys, returning a report of the outcome. r
// must be seekable: identification rewinds it per candidate, and the decode
// rewinds it once more. This is the single body shared by the offline Discover
// (r is an *os.File) and the live hunter (r is a bytes.Reader over a captured
// IQ buffer).
func decodeAndAccumulate(sys *DiscoveredSystem, r io.ReadSeeker, source string, p decodeParams) CaptureReport {
	log := p.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	minConf := p.MinConfidence
	if minConf <= 0 {
		minConf = 0.40
	}
	rep := CaptureReport{Path: source}

	proto := p.Protocol
	conf := 1.0 // operator-forced protocol ⇒ full confidence
	if proto == trunking.ProtocolUnknown {
		idr, err := siglab.IdentifyReader(r, source, siglab.IdentifyConfig{
			SampleRateHz: p.SampleRateHz,
			Format:       p.Format,
			AutoTune:     p.AutoTune,
			Conjugate:    p.Conjugate,
			IQCorrect:    p.IQCorrect,
			MaxSamples:   p.IdentifyMaxSamples,
			Log:          log,
		})
		if err != nil {
			rep.Error = fmt.Sprintf("identify: %v", err)
			return rep
		}
		conf = idr.Confidence
		if idr.Inconclusive || idr.Confidence < minConf {
			rep.Protocol = idr.Winner
			rep.Confidence = idr.Confidence
			rep.Skipped = true
			rep.SkipReason = fmt.Sprintf("no trunked control channel above confidence %.2f (best: %s @ %.2f)",
				minConf, idr.Winner, idr.Confidence)
			return rep
		}
		wp, err := idr.WinnerProtocol()
		if err != nil {
			rep.Error = fmt.Sprintf("winner protocol: %v", err)
			return rep
		}
		proto = wp
	}
	rep.Protocol = proto.String()
	rep.Confidence = conf

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		rep.Error = fmt.Sprintf("rewind: %v", err)
		return rep
	}
	res, err := siglab.RunReader(r, source, siglab.Config{
		Protocol:     proto,
		FrequencyHz:  p.FrequencyHz,
		SampleRateHz: p.SampleRateHz,
		Format:       p.Format,
		AutoTune:     p.AutoTune,
		Conjugate:    p.Conjugate,
		IQCorrect:    p.IQCorrect,
		// CollectIQDiag engages P25 Phase 1's deep path, which is what snapshots
		// the system topology (WACN/SYSID/RFSS/Site + neighbors + band plan)
		// onto Result.Topology. Without it P25 runs the generic factory pipeline
		// and the map would be NAC-only.
		CollectIQDiag: true,
		Log:           log,
	})
	if err != nil {
		rep.Error = fmt.Sprintf("decode: %v", err)
		return rep
	}

	before := len(sys.Talkgroups)
	Accumulate(sys, Observation{
		Protocol:       proto.String(),
		Confidence:     conf,
		Result:         res,
		FallbackFreqHz: p.FrequencyHz,
		At:             time.Now(),
	})
	rep.Locked = res.Locked
	if res.Lock != nil {
		rep.ControlHz = res.Lock.FrequencyHz
	}
	if res.Signal != nil {
		rep.ErrorRate = res.Signal.DecodeErrorRate
	}
	rep.Talkgroups = len(sys.Talkgroups) - before
	return rep
}
