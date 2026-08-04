package voice

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

// TestDecoderSurfacesGPS builds one superframe whose embedded LC is a GPS
// Info PDU and confirms the decoder surfaces the decoded fix on
// VoiceSuperframe rather than misreading it as a group-voice LC.
func TestDecoderSurfacesGPS(t *testing.T) {
	const wantLat, wantLon = -37.8136, 144.9631
	info := dmr.AssembleGPSInfo(dmr.GPSInfo{Latitude: wantLat, Longitude: wantLon})
	gpsFrags := aliasFragmentsFromInfo(t, info) // same raw-info → 4-fragment helper

	var frames [][]byte
	for f := 0; f < FramesPerSuperframe; f++ {
		frames = append(frames, mkFrame(f))
	}
	stream := make([]uint8, 200)
	for b := 0; b < BurstsPerSuperframe; b++ {
		sync := dmr.BSData.Dibits
		switch {
		case b == 0:
			sync = dmr.BSVoice.Dibits
		case b >= 1 && b <= 4:
			sync = embeddedSync(dmr.EMB{ColorCode: 1, LCSS: burstLCSS(b)}, gpsFrags[b-1])
		}
		base := b * FramesPerBurst
		stream = append(stream, makeVoiceBurst(frames[base:base+FramesPerBurst], sync)...)
	}
	stream = append(stream, make([]uint8, interleaveTail)...)

	d := NewDecoder()
	sfs := d.Process(stream, 0)
	if len(sfs) == 0 {
		t.Fatal("no superframe decoded")
	}
	sf := sfs[0]
	if !sf.HasGPS {
		t.Fatal("HasGPS not set on a GPS-carrying superframe")
	}
	const tol = 360.0 / (1 << 25)
	if math.Abs(sf.GPS.Latitude-wantLat) > tol || math.Abs(sf.GPS.Longitude-wantLon) > tol {
		t.Fatalf("surfaced fix wrong: got %.4f,%.4f want %.4f,%.4f",
			sf.GPS.Latitude, sf.GPS.Longitude, wantLat, wantLon)
	}
}
