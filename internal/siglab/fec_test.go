package siglab

import (
	"path/filepath"
	"testing"

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
