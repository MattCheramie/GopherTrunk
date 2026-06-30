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
	forced := proto != trunking.ProtocolUnknown
	tuneHz := 0.0 // carrier offset the decode tunes to (resolved below)
	if !forced {
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
		// Decode at the carrier identify locked, NOT a fresh dominant-carrier
		// estimate: under auto-tune the control channel may be off-centre and
		// not the band's loudest, so re-estimating here would re-find the wrong
		// carrier and the decode would fail even though identify succeeded.
		tuneHz = idr.TuneHz
	}
	rep.Protocol = proto.String()
	rep.Confidence = conf

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		rep.Error = fmt.Sprintf("rewind: %v", err)
		return rep
	}
	cfg := siglab.Config{
		Protocol:     proto,
		FrequencyHz:  p.FrequencyHz,
		SampleRateHz: p.SampleRateHz,
		Format:       p.Format,
		AutoTune:     false,
		TuneHz:       tuneHz,
		Conjugate:    p.Conjugate,
		IQCorrect:    p.IQCorrect,
		// CollectIQDiag engages P25 Phase 1's deep path, which is what snapshots
		// the system topology (WACN/SYSID/RFSS/Site + neighbors + band plan)
		// onto Result.Topology. Without it P25 runs the generic factory pipeline
		// and the map would be NAC-only.
		CollectIQDiag: true,
		Log:           log,
	}
	var res *siglab.Result
	var err error
	if forced && p.AutoTune {
		// Forced protocol + auto-tune: identify didn't run, so resolve the
		// carrier here by trying the ranked candidates and keeping the best lock.
		res, err = siglab.RunReaderAutoTuneMulti(r, source, cfg, 0)
	} else {
		res, err = siglab.RunReader(r, source, cfg)
	}
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
	rep.IdentityNote = identityNote(res)
	if rep.IdentityNote != "" {
		log.Warn("hunt: "+rep.IdentityNote, "path", source)
	}
	rep.Encrypted, rep.EncType = encryptionFromGrants(proto.String(), res.Grants)
	rep.Talkgroups = len(sys.Talkgroups) - before
	return rep
}

// encryptionFromGrants reports whether any grant on the control channel was
// encrypted and names the algorithm. P25 Phase 1 carries the ALGID in the
// voice LDU2 rather than the CC grant, so a CC-only decode may show encrypted
// with an unknown algorithm (encType empty) — still a useful "encrypted" flag.
func encryptionFromGrants(protocol string, grants []siglab.GrantRecord) (encrypted bool, encType string) {
	for _, g := range grants {
		if !g.Encrypted {
			continue
		}
		encrypted = true
		if name := encTypeName(protocol, g.AlgorithmID); name != "" {
			encType = name // first named algorithm wins
			break
		}
	}
	return encrypted, encType
}

// identityNote explains, for a locked P25 control channel whose WACN/System ID
// are still blank, *why* — by reporting which identity broadcasts decoded. The
// Network Status Broadcast (NSB, opcode 0x3B) is the only P25 Phase 1 message
// carrying WACN, and the RFSS Status Broadcast (0x3A) is the only other source
// of System ID; if neither lands in the captured window the identity is simply
// absent, not withheld. It returns "" when identity resolved, the capture
// didn't lock, or it isn't P25 — there is nothing to explain.
func identityNote(res *siglab.Result) string {
	if res == nil || !res.Locked || res.Topology == nil {
		return ""
	}
	if res.Topology.WACN != 0 && res.Topology.SystemID != 0 {
		return "" // identity fully resolved
	}
	d, ok := res.Detail.(*siglab.P25P1Detail)
	if !ok || d.CCStats == nil {
		return "" // not the P25 deep path
	}
	cc := d.CCStats
	if res.Topology.SystemID != 0 && res.Topology.WACN == 0 {
		// System ID recovered (voted from adjacent-site broadcasts), but WACN
		// is still blank: NET_STS_BCST (0x3B) is the only P25 message that
		// carries it, and this system never transmitted it. Widening the dwell
		// won't help if the system simply doesn't emit the NSB.
		return fmt.Sprintf("System ID resolved from adjacent-site broadcasts, but WACN is "+
			"unavailable: the Network Status Broadcast (NSB, 0x3B) that carries it was never "+
			"decoded (saw NSB×%d, RFSS×%d, adjacent×%d, loc-reg×%d, TSBK×%d) — if the system never "+
			"emits the NSB, the WACN cannot be recovered",
			cc.NetStatusSeen, cc.RFSSStatusSeen, cc.AdjacentSeen, cc.LocRegSeen, cc.TSBKDecoded)
	}
	if cc.NetStatusSeen > 0 {
		// NSB decoded but yielded no WACN — a parse problem, not a capture gap.
		return fmt.Sprintf("identity unresolved despite %d Network Status Broadcast(s): "+
			"NSB decoded but WACN/System ID parsed as zero", cc.NetStatusSeen)
	}
	return fmt.Sprintf("identity unresolved: the Network Status Broadcast (NSB, 0x3B) that carries "+
		"WACN+System ID never decoded (saw RFSS×%d, adjacent×%d, TSBK×%d) — "+
		"widen --dwell-seconds or use --monitor-seconds to catch the periodic NSB",
		cc.RFSSStatusSeen, cc.AdjacentSeen, cc.TSBKDecoded)
}
