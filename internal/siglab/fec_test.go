package siglab

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// runSynthDetail synthesizes a clean capture for proto, decodes it with the
// analyzer on, and returns the ProtocolDetail.
func runSynthDetail(t *testing.T, proto trunking.Protocol) *ProtocolDetail {
	t.Helper()
	iq, meta, err := Synthesize(SynthOptions{Protocol: proto, Format: FormatF32})
	if err != nil {
		t.Fatalf("Synthesize(%s): %v", proto, err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, proto.String()+".cfile")
	if err := WriteCapture(path, iq, FormatF32); err != nil {
		t.Fatalf("WriteCapture: %v", err)
	}
	cfg, err := meta.Config(true) // CollectIQDiag
	if err != nil {
		t.Fatalf("meta.Config: %v", err)
	}
	res, err := Run(path, cfg)
	if err != nil {
		t.Fatalf("Run(%s): %v", proto, err)
	}
	d, ok := res.Detail.(*ProtocolDetail)
	if !ok || d == nil {
		t.Fatalf("Detail for %s is %T, want *ProtocolDetail", proto, res.Detail)
	}
	return d
}

// TestFECTallyCleanProtocols asserts the FEC tally decodes every frame on the
// (GFSK / header) protocols whose synthesized recovered stream is symbol-clean:
// no uncorrectable frames, and at least one frame found.
func TestFECTallyCleanProtocols(t *testing.T) {
	cases := []struct {
		proto trunking.Protocol
		crc   bool // graded via CRC pass rather than error count
	}{
		{trunking.ProtocolEDACS, false},
		{trunking.ProtocolMotorola, false},
		{trunking.ProtocolDStar, true},
	}
	for _, c := range cases {
		t.Run(c.proto.String(), func(t *testing.T) {
			d := runSynthDetail(t, c.proto)
			if len(d.FEC) == 0 {
				t.Fatalf("%s: no FEC tally", c.proto)
			}
			f := d.FEC[0]
			if f.Frames == 0 {
				t.Fatalf("%s: FEC found no frames", c.proto)
			}
			if c.crc {
				if f.CRCPass != f.Frames || f.CRCFail != 0 {
					t.Errorf("%s: crc_pass=%d crc_fail=%d, want all %d pass", c.proto, f.CRCPass, f.CRCFail, f.Frames)
				}
			} else if f.Uncorrectable != 0 {
				t.Errorf("%s: %d uncorrectable frames on a clean capture", c.proto, f.Uncorrectable)
			}
		})
	}
}

// TestFECTallyDMR asserts the DMR slot-type tally runs and finds frames. (DMR
// C4FM recovery introduces correctable symbol errors on the synth, so we don't
// require clean==frames — only that the tally is populated and most frames are
// recoverable.)
func TestFECTallyDMR(t *testing.T) {
	d := runSynthDetail(t, trunking.ProtocolDMR)
	if len(d.FEC) == 0 || d.FEC[0].Frames == 0 {
		t.Fatal("DMR slot-type tally empty")
	}
	f := d.FEC[0]
	if f.Clean+f.Corrected == 0 {
		t.Errorf("DMR: no recoverable slot-type frames (frames=%d uncorrectable=%d)", f.Frames, f.Uncorrectable)
	}
}

// findFEC returns the FECStat whose Stage contains sub, or nil.
func findFEC(d *ProtocolDetail, sub string) *FECStat {
	for i := range d.FEC {
		if strings.Contains(d.FEC[i].Stage, sub) {
			return &d.FEC[i]
		}
	}
	return nil
}

// TestFECTallyP25P2 asserts the P25 Phase 2 ISCH Golay and MAC trellis tallies
// decode every frame clean on the synth (a strong check that the ISCH/MAC
// offsets + dibit packing are correct).
func TestFECTallyP25P2(t *testing.T) {
	d := runSynthDetail(t, trunking.ProtocolP25Phase2)
	for _, stage := range []string{"ISCH", "MAC trellis"} {
		f := findFEC(d, stage)
		if f == nil || f.Frames == 0 {
			t.Fatalf("P25P2 %s tally missing/empty", stage)
		}
		if f.Clean != f.Frames || f.Uncorrectable != 0 {
			t.Errorf("P25P2 %s: clean=%d uncorrectable=%d, want all %d clean", stage, f.Clean, f.Uncorrectable, f.Frames)
		}
	}
}

// TestFECTallyNXDN asserts the NXDN LICH decodes clean on the synth and the CAC
// CRC passes for at least some frames (the CAC is long; C4FM symbol errors fail
// some frames, which is itself the diagnostic).
func TestFECTallyNXDN(t *testing.T) {
	d := runSynthDetail(t, trunking.ProtocolNXDN)
	lich := findFEC(d, "LICH")
	if lich == nil || lich.Frames == 0 {
		t.Fatal("NXDN LICH tally missing")
	}
	if lich.Uncorrectable != 0 {
		t.Errorf("NXDN LICH: %d uncorrectable on a clean capture", lich.Uncorrectable)
	}
	cac := findFEC(d, "CAC")
	if cac == nil || cac.Frames == 0 {
		t.Fatal("NXDN CAC tally missing")
	}
	if cac.CRCPass == 0 {
		t.Errorf("NXDN CAC: no frames passed CRC (frames=%d)", cac.Frames)
	}
}

// TestTetraFECRecoversColourCode covers issue #648: under blind identify there
// is no configured colour code, so descrambling SCH/HD with 0 fails on every
// real cell and TETRA forfeits its 30% FEC score. tetraFEC must recover the
// colour code from the BSCH of a synchronisation burst in the stream and use it.
func TestTetraFECRecoversColourCode(t *testing.T) {
	const (
		cc6        = 0x2D
		mcc uint16 = 0x0CE
		mnc uint16 = 0x1234
	)
	colour := tetra.ExtendedColourCode(mcc, mnc, cc6)

	// Synchronisation burst: BSCH (colour 0 on the wire, carrying the code) + STS.
	set := func(bits []byte, off, n int, v uint32) {
		for i := 0; i < n; i++ {
			bits[off+i] = byte((v >> uint(n-1-i)) & 1)
		}
	}
	syncInfo := make([]byte, 60)
	set(syncInfo, 4, 6, cc6)
	set(syncInfo, 31, 10, uint32(mcc))
	set(syncInfo, 41, 14, uint32(mnc))
	var sb []uint8
	sb = append(sb, tetra.TetraBitsToDibits(tetra.EncodeBSCH(syncInfo))...) // BSCH [0,60)
	sb = append(sb, tetra.SyncTrainingDibits()...)                          // STS  [60,79)
	sb = append(sb, make([]uint8, 15)...)                                   // trailing pad

	variants := []SyncVariant{
		{Name: "nts1", Pattern: tetra.NormalSyncDibits()},
		{Name: "nts2", Pattern: tetra.NormalSyncDibits2()},
		{Name: "extended", Pattern: tetra.ExtendedSyncDibits()},
		{Name: "sync", Pattern: tetra.SyncTrainingDibits()},
	}

	// SB burst + SCH/HD SYSINFO bursts scrambled with the cell's colour code.
	sch := buildTETRASCHHDStream(8, colour, 0x2B7)
	stream := append(append([]uint8{}, sb...), sch...)
	land := computeSyncLandscape(stream, variants, 4, 2)

	// Blind identify (zero configured colour code) must still tally CRC passes.
	got := tetraFEC(stream, land, trunking.System{})
	if len(got) == 0 || got[0].CRCPass == 0 {
		t.Fatalf("tetraFEC tallied no SCH/HD CRC passes after colour-code recovery: %+v", got)
	}

	// Control: with no BSCH to recover from, colour 0 can't descramble the
	// cell's SCH/HD, so nothing passes — proving the recovery is what helped.
	noSB := computeSyncLandscape(sch, variants, 4, 2)
	if s := tetraFEC(sch, noSB, trunking.System{}); len(s) > 0 && s[0].CRCPass != 0 {
		t.Errorf("expected 0 CRC passes without a BSCH to recover the colour code, got %d", s[0].CRCPass)
	}
}

// TestFECTallyTETRA asserts the TETRA SCH/HD decode passes CRC for most frames
// on the synth (the colour code from the fixture metadata seeds the
// descrambler; a couple of in-tolerance false-positive sync hits may fail).
func TestFECTallyTETRA(t *testing.T) {
	d := runSynthDetail(t, trunking.ProtocolTETRA)
	f := findFEC(d, "SCH/HD")
	if f == nil || f.Frames == 0 {
		t.Fatal("TETRA SCH/HD tally missing")
	}
	if f.CRCPass <= f.Frames/2 {
		t.Errorf("TETRA: crc_pass=%d of %d frames, want a clear majority", f.CRCPass, f.Frames)
	}
}
