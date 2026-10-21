package composer

import (
	"context"
	"encoding/binary"
	"log/slog"
	"math"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// The end-of-call line of the NXDN voice chain must tell apart the cases the
// #1261 field log could not:
//
//   - a call that decoded and THEN let the carrier tracker run to its
//     ±3 kHz bound (one call ended at carrier_offset_hz=-3000 after 406
//     clean voice frames): carrier_offset_at_voice_hz keeps the offset as of
//     the last voice frame;
//   - a chain that received no IQ at all versus one that heard no NXDN (a
//     tg 100 call ended with every counter 0): iq_samples.
//
// The reporter's real-air NXDN96 slice is followed here by an unmodulated
// carrier 5 kHz off, which the tracker accepts (no modulation, so no
// variance) and follows to the bound after the speech is over.
func TestComposerNXDNVoiceChainSummary(t *testing.T) {
	const rate = 48_000.0
	voice := readCS16Fixture(t, "../../radio/nxdn/testdata/nxdn96_rdch_voice_447100_15oct_48k.cs16")
	tail := make([]complex64, int(1.5*rate))
	for i := range tail {
		ph := -2 * math.Pi * 5000 * float64(i) / rate
		tail[i] = complex(float32(0.3*math.Cos(ph)), float32(0.3*math.Sin(ph)))
	}
	iq := append(append([]complex64{}, voice...), tail...)

	buf := &syncBuffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	src := newFakeSource()
	bus := events.NewBus(8)
	c, err := New(Options{
		Bus:           bus,
		Log:           logger,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          &recordingSink{},
		Engine:        &fakeEngine{},
		IQSampleRate:  uint32(rate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant:        trunking.Grant{System: "SJPR NXDN", Protocol: "nxdn", GroupID: 21, FrequencyHz: 447_100_000},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)
	time.Sleep(1500 * time.Millisecond)
	publishEnd(bus, "VOICE-1")

	re := regexp.MustCompile(`composer: nxdn voice chain ended.* voice_frames=(\d+) .*carrier_offset_hz=(-?\d+) carrier_offset_at_voice_hz=(-?\d+) iq_samples=(\d+)`)
	waitFor(t, 6*time.Second, func() bool { return re.MatchString(buf.String()) })
	m := re.FindStringSubmatch(buf.String())
	frames, _ := strconv.Atoi(m[1])
	end, _ := strconv.Atoi(m[2])
	atVoice, _ := strconv.Atoi(m[3])
	samples, _ := strconv.Atoi(m[4])
	if frames == 0 {
		t.Fatalf("no voice frames decoded from the real-air slice:\n%s", buf.String())
	}
	if end != -3000 {
		t.Errorf("carrier_offset_hz = %d, want the -3000 bound after the off-channel carrier", end)
	}
	// The slice sits a few hundred Hz off (realair_test.go); at the last
	// voice frame the tracker is there, nowhere near the bound.
	if atVoice < -1500 || atVoice > 1500 {
		t.Errorf("carrier_offset_at_voice_hz = %d, want the voice's own offset (|x| < 1.5 kHz)", atVoice)
	}
	if samples != len(iq) {
		t.Errorf("iq_samples = %d, want %d", samples, len(iq))
	}
}

func readCS16Fixture(t *testing.T, path string) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(raw)/4)
	for i := range iq {
		re := int16(binary.LittleEndian.Uint16(raw[4*i:]))
		im := int16(binary.LittleEndian.Uint16(raw[4*i+2:]))
		iq[i] = complex(float32(re)/32768, float32(im)/32768)
	}
	return iq
}
