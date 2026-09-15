package afsk

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/fleetsync"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

func loadRealAirSlice(t *testing.T, name string) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(raw)/4)
	for i := range iq {
		re := int16(binary.LittleEndian.Uint16(raw[i*4:]))
		im := int16(binary.LittleEndian.Uint16(raw[i*4+2:]))
		iq[i] = complex(float32(re)/32768, float32(im)/32768)
	}
	return iq
}

func drainFleetSync(sub *events.Subscription, wait time.Duration) []storage.FleetSyncMessage {
	var out []storage.FleetSyncMessage
	deadline := time.After(wait)
	for {
		select {
		case ev, ok := <-sub.C:
			if !ok {
				return out
			}
			if ev.Kind != events.KindFleetSyncMessage {
				continue
			}
			if m, ok := ev.Payload.(storage.FleetSyncMessage); ok {
				out = append(out, m)
			}
		case <-deadline:
			return out
		}
	}
}

// TestReceiverPublishesRealAirBurstsOnBus is the daemon-path pin: the same
// on-air FleetSync-II slice realair_test.go decodes must reach the events
// bus as KindFleetSyncMessage events carrying the storage payload the log
// writer, REST endpoint and web panel consume — Fleet 107 / Unit 1772.
func TestReceiverPublishesRealAirBurstsOnBus(t *testing.T) {
	iq := loadRealAirSlice(t, "fleetsync2_fleet107_unit1772_48k.cs16")
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	rcv, err := New(Options{InputRateHz: 48000, Bus: bus, SourceName: "test"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(iq); i += 4096 {
		rcv.ProcessIQ(iq[i:min(i+4096, len(iq))])
	}

	got := drainFleetSync(sub, 200*time.Millisecond)
	ok := 0
	for _, m := range got {
		if !m.CRCOK {
			continue
		}
		if m.Fleet != 107 || m.Unit != 1772 || !m.IsFS2 {
			t.Errorf("CRC-valid bus payload with the wrong identity: %+v", m)
			continue
		}
		if m.ReceivedAt.IsZero() || m.RawHex == "" || m.Body == "" {
			t.Errorf("bus payload missing fields: %+v", m)
		}
		ok++
	}
	if ok < 2 {
		t.Fatalf("CRC-valid bus events = %d, want ≥ 2 (all events: %+v)", ok, got)
	}
	if st := rcv.Stats(); st.BurstsPublished != uint64(len(got)) || st.BurstsDropped != 0 {
		t.Errorf("Stats = %+v, want BurstsPublished=%d BurstsDropped=0", st, len(got))
	}
}

// TestReceiverDropBadCRCKeepsFailuresOffBusOnly: DropBadCRC filters the bus,
// never the OnMessage callback, and CRC-valid bursts still publish.
func TestReceiverDropBadCRCKeepsFailuresOffBusOnly(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	var seen []fleetsync.Message
	rcv, err := New(Options{
		InputRateHz: 48000,
		Bus:         bus,
		DropBadCRC:  true,
		OnMessage:   func(m fleetsync.Message) { seen = append(seen, m) },
	})
	if err != nil {
		t.Fatal(err)
	}
	// Drive the framer callback directly with one good and one bad burst so
	// the test pins the routing, not the DSP.
	rcv.onFrame(fleetsync.Message{Fleet: 107, Unit: 1772, CRCOK: true, Body: "good"})
	rcv.onFrame(fleetsync.Message{Fleet: 1, Unit: 2, CRCOK: false, Body: "bad"})

	if len(seen) != 2 {
		t.Fatalf("OnMessage saw %d bursts, want both", len(seen))
	}
	got := drainFleetSync(sub, 100*time.Millisecond)
	if len(got) != 1 || !got[0].CRCOK || got[0].Unit != 1772 {
		t.Fatalf("bus got %+v, want only the CRC-valid burst", got)
	}
	if st := rcv.Stats(); st.BurstsPublished != 1 || st.BurstsDropped != 1 {
		t.Errorf("Stats = %+v, want 1 published / 1 dropped", st)
	}
}

func TestNewRequiresASink(t *testing.T) {
	if _, err := New(Options{InputRateHz: 48000}); err == nil {
		t.Fatal("New with neither OnMessage nor Bus should fail")
	}
	if _, err := New(Options{InputRateHz: 48000, Bus: events.NewBus(1)}); err != nil {
		t.Fatalf("New with only a Bus: %v", err)
	}
}
