package afsk

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/fleetsync"
)

// TestFleetSyncRealAirSlices decodes two real Kenwood transmissions through
// the production front end: 1.3 s channelized (48 kHz, cs16) slices of the
// issue #1184 SDR# captures — FleetSync-I and FleetSync-II ANI bursts from
// Fleet 107 / Unit 1772 (see testdata/README.md). They are the on-air pin the
// synthetic round-trips cannot be (the #764/#771 rule): a framing or
// front-end change that still passes its own encoder must also still decode
// these.
func TestFleetSyncRealAirSlices(t *testing.T) {
	for _, tc := range []struct {
		file    string
		fs2     bool
		minOK   int
		rawWord string
	}{
		{"fleetsync1_fleet107_unit1772_48k.cs16", false, 2, ""},
		{"fleetsync2_fleet107_unit1772_48k.cs16", true, 2, ""},
	} {
		t.Run(tc.file, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", tc.file))
			if err != nil {
				t.Fatal(err)
			}
			iq := make([]complex64, len(raw)/4)
			for i := range iq {
				re := int16(binary.LittleEndian.Uint16(raw[i*4:]))
				im := int16(binary.LittleEndian.Uint16(raw[i*4+2:]))
				iq[i] = complex(float32(re)/32768, float32(im)/32768)
			}
			var msgs []fleetsync.Message
			rcv, err := New(Options{InputRateHz: 48000, BaudHz: 1200, OnMessage: func(m fleetsync.Message) {
				msgs = append(msgs, m)
			}})
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < len(iq); i += 4096 {
				rcv.ProcessIQ(iq[i:min(i+4096, len(iq))])
			}
			ok := 0
			for _, m := range msgs {
				t.Logf("burst crc_ok=%v fs2=%v fleet=%d unit=%d raw=%s", m.CRCOK, m.IsFS2, m.Fleet, m.Unit, m.RawHex)
				if !m.CRCOK {
					continue
				}
				if m.Fleet != 107 || m.Unit != 1772 || m.IsFS2 != tc.fs2 {
					t.Errorf("CRC-valid burst with the wrong identity: %+v", m)
					continue
				}
				ok++
			}
			if ok < tc.minOK {
				t.Fatalf("%d CRC-valid Fleet 107 / Unit 1772 bursts, want >= %d (%d bursts framed)", ok, tc.minOK, len(msgs))
			}
		})
	}
}
