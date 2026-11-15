package dstar

import (
	"bufio"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// mmdvmHeaderBursts reads the header bursts the MMDVM firmware's own
// encoder produced (framing's testdata): the header bytes, and the on-air
// bits from the frame sync 0xEA 0xA6 onwards.
func mmdvmHeaderBursts(t *testing.T) (headers [][]byte, bursts [][]uint8) {
	t.Helper()
	f, err := os.Open("../framing/testdata/dstar_header_mmdvm.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 4096), 4096)
	for sc.Scan() {
		fl := strings.Fields(sc.Text())
		if len(fl) != 3 || strings.HasPrefix(fl[0], "#") {
			continue
		}
		h, err := hex.DecodeString(fl[1])
		if err != nil {
			t.Fatal(err)
		}
		var b []uint8
		for _, c := range fl[2] {
			b = append(b, uint8(c-'0'))
		}
		headers, bursts = append(headers, h), append(bursts, b)
	}
	return headers, bursts
}

// A header MMDVM transmitted, after its 1010… preamble, decodes through
// Process to the header MMDVM was given, in either polarity. Before
// #1261 no D-STAR transmission could: the frame sync, the FEC chain and
// the CRC were all invented.
func TestProcessDecodesMMDVMHeaderBursts(t *testing.T) {
	headers, bursts := mmdvmHeaderBursts(t)
	for i, burst := range bursts {
		want, err := ParseHeader(headers[i])
		if err != nil {
			t.Fatal(err)
		}
		for _, inv := range []bool{false, true} {
			stream := append(BitSyncBits(200), burst...)
			stream = append(stream, BitSyncBits(40)...)
			if inv {
				stream = invertBits(stream)
			}
			var got []Header
			cc := New(Options{Bus: events.NewBus(8), FrequencyHz: 145_670_000, Now: time.Now})
			cc.ingestHook = func(h Header) { got = append(got, h) }
			// Feed in uneven chunks so the sync and the header straddle
			// Process calls.
			pos := 0
			for _, n := range []int{37, 101, 7, 333, 1000} {
				end := min(pos+n, len(stream))
				pos = cc.Process(stream[pos:end], pos)
			}
			if len(got) != 1 || got[0] != want {
				t.Errorf("burst %d inverted=%v: decoded %+v, want %+v", i, inv, got, want)
			}
		}
	}
}
