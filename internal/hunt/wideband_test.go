package hunt

import (
	"encoding/binary"
	"math"
	"math/cmplx"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// buildWidebandFixture plants the two committed 48 kHz DMR fixtures (the Tier
// III control channel and a voice carrier) into one synthetic 192 kHz
// wide-band capture at ±48 kHz and writes it as an f32 .cfile. Returns the path
// and the capture center frequency.
func buildWidebandFixture(t *testing.T) (path string, centerHz uint32) {
	t.Helper()
	const wideRate = 192_000.0
	centerHz = 441_000_000
	load := func(name string) []complex64 {
		raw, err := os.ReadFile(filepath.Join("..", "..", "cmd", "gophertrunk", "testdata", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		p := len(raw) / 8
		iq := make([]complex64, p)
		dec, _ := siglab.FormatF32.Decoder()
		dec(raw[:p*8], iq)
		return iq
	}
	upshift := func(in []complex64, offsetHz float64) []complex64 {
		rs := dsp.NewResampler(4, 1, 16, 9.0)
		up := rs.Process(nil, in)
		step := cmplx.Exp(complex(0, 2*math.Pi*offsetHz/wideRate))
		ph := complex(1, 0)
		out := make([]complex64, len(up))
		for i, s := range up {
			out[i] = complex64(complex128(s) * ph)
			ph *= step
		}
		return out
	}
	ctrl := upshift(load("dmr-t3-cc.cfile"), +48_000)
	vox := upshift(load("dmr-voice-term.cfile"), -48_000)
	n := len(ctrl)
	if len(vox) < n {
		n = len(vox)
	}
	buf := make([]byte, n*8)
	for i := 0; i < n; i++ {
		s := ctrl[i] + vox[i]
		binary.LittleEndian.PutUint32(buf[i*8:], math.Float32bits(real(s)))
		binary.LittleEndian.PutUint32(buf[i*8+4:], math.Float32bits(imag(s)))
	}
	path = filepath.Join(t.TempDir(), "dmr-t3-wideband.cfile")
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path, centerHz
}

// TestDiscoverWidebandGroupsTier3System asserts the offline wideband-hunt bridge
// surveys a wideband capture, recognizes the DMR Tier III control channel, and
// folds it into one DiscoveredSystem carrying the real SystemID.
func TestDiscoverWidebandGroupsTier3System(t *testing.T) {
	path, center := buildWidebandFixture(t)

	sys, reports, err := DiscoverWideband([]CaptureInput{{
		Path:         path,
		Format:       siglab.FormatF32,
		SampleRateHz: 192_000,
		FrequencyHz:  center,
	}}, DiscoverConfig{Name: "wb-test"})
	if err != nil {
		t.Fatalf("DiscoverWideband: %v", err)
	}

	if len(reports) != 1 {
		t.Fatalf("got %d reports, want 1", len(reports))
	}
	rep := reports[0]
	t.Logf("report: proto=%s locked=%v controlHz=%d tgs=%d verdict=%q",
		rep.Protocol, rep.Locked, rep.ControlHz, rep.Talkgroups, rep.Verdict)
	if rep.Error != "" {
		t.Fatalf("report error: %s", rep.Error)
	}
	if !rep.Locked {
		t.Errorf("report not locked; want the Tier III control channel locked")
	}
	if rep.ControlHz == 0 {
		t.Errorf("report has no control-channel frequency")
	}
	if !strings.Contains(rep.Verdict, "DMR Tier III trunked system") {
		t.Errorf("verdict = %q, want it to name the Tier III system", rep.Verdict)
	}

	if sys.Protocol != "dmr" {
		t.Errorf("system protocol = %q, want dmr", sys.Protocol)
	}
	if sys.SystemID != 61760 {
		t.Errorf("system SystemID = %d, want 61760", sys.SystemID)
	}
	if len(sys.Sites) == 0 {
		t.Errorf("system has no sites; want at least the camped control-channel site")
	}
}
