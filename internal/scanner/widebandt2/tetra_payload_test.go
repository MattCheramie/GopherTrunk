package widebandt2

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	tetra "github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
)

// buildArmingSBDibits assembles a clean TETRA sync burst (BSCH decodes, BNCH
// idle) from exported tetra symbols — enough to lock the control channel and
// stamp its ACTIVITY heartbeat without ever stamping the PAYLOAD one. Mirrors
// ccdecoder's buildArmingSBStream.
func buildArmingSBDibits() []uint8 {
	setBits := func(bits []byte, off, n int, v uint32) {
		for i := 0; i < n; i++ {
			bits[off+i] = byte((v >> uint(n-1-i)) & 1)
		}
	}
	syncInfo := make([]byte, 60)
	setBits(syncInfo, 4, 6, 0x2D)
	setBits(syncInfo, 31, 10, 3)
	setBits(syncInfo, 41, 14, 5)
	bschDibits := tetra.TetraBitsToDibits(tetra.EncodeBSCH(syncInfo))

	var s []uint8
	s = append(s, make([]uint8, 50)...)
	s = append(s, bschDibits...)
	s = append(s, tetra.SyncTrainingDibits()...)
	s = append(s, make([]uint8, 15)...)
	s = append(s, make([]uint8, 108)...)
	s = append(s, make([]uint8, 300)...)
	return s
}

// TestWidebandTETRAPayloadDroughtForcesResyncAndEscalates mirrors the
// ccdecoder payload-drought tests on the wideband tap: a locked channel whose
// sync bursts keep the activity heartbeat fresh but which decodes no SCH
// payload fires the payload resync per budget, and the third fruitless budget
// declares the lock lost (the 19 Aug "locked but deaf" escape).
func TestWidebandTETRAPayloadDroughtForcesResyncAndEscalates(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)

	cc := tetra.New(tetra.Options{
		Bus: bus, Log: logger, SystemName: "Test", FrequencyHz: 412_000_000,
	})
	cc.SetChannelCoding(tetra.ChannelCodingOn)
	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz: tetraChannelRateHz,
		ClockMode:    tetrarx.ClockGardner,
		GardnerGain:  0.005,
		DibitSink:    func(d []uint8, b int) { cc.Process(d, b) },
	})
	rcv := &tetraChannelReceiver{
		rx: rx, cc: cc, log: logger, system: "Test",
		rateHz: tetraChannelRateHz, now: time.Now,
	}

	cc.Process(buildArmingSBDibits(), 0)
	if !cc.Locked() {
		t.Fatal("control channel did not lock on the arming burst")
	}
	rcv.Process(nil)
	buf.Reset()

	budget := int64(tetraPayloadResyncTimeout.Seconds() * rcv.rateHz)
	const chunk = 100_000
	feed := func(n int64) {
		iq := make([]complex64, chunk)
		for i := range iq {
			iq[i] = complex(0.01, 0.01)
		}
		for n > 0 {
			c := n
			if c > chunk {
				c = chunk
			}
			rcv.lastSeenActivity = cc.LastActivityNano() - 1 // BSCH stays "alive"
			rcv.Process(iq[:c])
			n -= c
		}
	}

	feed(2*(budget+chunk))
	if got := strings.Count(buf.String(), "payload drought; sync bursts still decoding"); got != 2 {
		t.Fatalf("want two payload-drought resyncs, got %d\n%s", got, buf.String())
	}
	if !cc.Locked() {
		t.Fatal("lock lost before the escalation threshold")
	}
	feed(budget + chunk)
	if !strings.Contains(buf.String(), "payload drought persists across resyncs") {
		t.Fatalf("third fruitless budget did not escalate\n%s", buf.String())
	}
	if cc.Locked() {
		t.Fatal("escalation did not declare the lock lost")
	}
}
