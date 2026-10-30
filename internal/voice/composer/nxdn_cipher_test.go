package composer

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// loadNXDNScramblerFrames reads the issue #1261 known-key fixture the nxdn
// package pins (every LICH-valid frame of a scrambled real-air call) as
// TrafficFrames, the way TrafficChannel.OnFrame delivers them.
func loadNXDNScramblerFrames(t *testing.T) []nxdn.TrafficFrame {
	t.Helper()
	raw, err := os.ReadFile("../../radio/nxdn/testdata/scrambler_key32767_frames.json")
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Frames []struct {
			Index    uint64   `json:"index"`
			SACCH    string   `json:"sacch"`
			Slots    []int    `json:"slots"`
			Payloads []string `json:"payloads"`
		} `json:"frames"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	bits := func(s string) []byte {
		out := make([]byte, len(s))
		for i := range s {
			out[i] = s[i] - '0'
		}
		return out
	}
	var frames []nxdn.TrafficFrame
	for _, f := range fx.Frames {
		tf := nxdn.TrafficFrame{Index: f.Index, SACCH: bits(f.SACCH)}
		for i, p := range f.Payloads {
			tf.Voice = append(tf.Voice, nxdn.VCHFrame{Slot: f.Slots[i], Payload: bits(p)})
		}
		frames = append(frames, tf)
	}
	return frames
}

// TestNXDNCipherStateResolvesConventionalChannelKey: a scanner call's grant
// names the scanner, not a system, so the VCALL's key id must resolve
// through the channel's frequency (ConventionalKeySystem), the call must be
// published as encrypted (cipher type 1, key id 3), and with the key the
// frames come out descrambled (the real-air fixture: 128 voice frames,
// pitch continuity off the ciphertext floor).
func TestNXDNCipherStateResolvesConventionalChannelKey(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	resolver := func(system, algorithm string, keyID uint16, _ uint32) ([]byte, bool) {
		if system == ConventionalKeySystem(447_100_000) && algorithm == "scrambler" && keyID == 3 {
			return []byte{0x7F, 0xFF}, true // 32767
		}
		return nil, false
	}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{}},
		Sink:          &recordingSink{},
		Engine:        &fakeEngine{},
		IQSampleRate:  2_400_000,
		PCMSampleRate: 8000,
		KeyResolver:   resolver,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	st := c.newNXDNCipherState("VOICE-1", "scanner", 447_100_000, 0x80000003)
	if !st.tracker.Hold {
		t.Fatal("keys configured ⇒ voice is held until the VCALL names the cipher")
	}
	var out, raw [][]byte
	for _, f := range loadNXDNScramblerFrames(t) {
		for _, v := range f.Voice {
			raw = append(raw, append([]byte(nil), v.Payload...))
		}
		for _, v := range st.tracker.Frame(f) {
			out = append(out, v.Payload)
		}
	}
	for _, v := range st.tracker.Flush() {
		out = append(out, v.Payload)
	}
	if len(out) != len(raw) {
		t.Fatalf("released %d frames, want %d", len(out), len(raw))
	}
	if !st.keyConfigured || st.tracker.Descrambled != uint64(len(raw)) {
		t.Fatalf("key_configured=%v descrambled=%d of %d", st.keyConfigured, st.tracker.Descrambled, len(raw))
	}
	changed := 0
	for i := range out {
		for j := range out[i] {
			if out[i][j] != raw[i][j] {
				changed++
				break
			}
		}
	}
	if changed < len(raw)*9/10 {
		t.Fatalf("only %d of %d frames differ from the ciphertext", changed, len(raw))
	}
	fields := st.logFields()
	if len(fields) == 0 {
		t.Fatal("no log fields")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			t.Fatal("no call.encryption event")
		case ev := <-sub.C:
			if ev.Kind != events.KindCallEncryption {
				continue
			}
			ce := ev.Payload.(trunking.CallEncryption)
			if ce.Protocol != "nxdn" || ce.AlgorithmID != uint8(nxdn.CipherScrambler) || ce.KeyID != 3 || ce.DeviceSerial != "VOICE-1" {
				t.Fatalf("call.encryption = %+v", ce)
			}
			return
		}
	}
}

// TestNXDNCipherStateWithoutKeysPassesThrough: no resolver ⇒ no hold, the
// frames are delivered as received, but the VCALL is still decoded and the
// call still published as encrypted (so mute_encrypted / the call log see it).
func TestNXDNCipherStateWithoutKeysPassesThrough(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{}},
		Sink:          &recordingSink{},
		Engine:        &fakeEngine{},
		IQSampleRate:  2_400_000,
		PCMSampleRate: 8000,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	st := c.newNXDNCipherState("VOICE-1", "scanner", 447_100_000, 1)
	if st.tracker.Hold {
		t.Fatal("no keys ⇒ no hold")
	}
	n := 0
	for _, f := range loadNXDNScramblerFrames(t) {
		got := st.tracker.Frame(f)
		if len(got) != len(f.Voice) {
			t.Fatalf("frame %d: released %d of %d voice frames immediately", f.Index, len(got), len(f.Voice))
		}
		n += len(got)
	}
	// The first superframe's voice went out before the VCALL named the
	// cipher (Unknown); everything after is counted as ciphertext.
	if st.tracker.Descrambled != 0 || st.tracker.Ciphertext+st.tracker.Unknown != uint64(n) || st.tracker.Ciphertext == 0 || st.keyConfigured {
		t.Fatalf("descrambled=%d ciphertext=%d unknown=%d of %d key_configured=%v", st.tracker.Descrambled, st.tracker.Ciphertext, st.tracker.Unknown, n, st.keyConfigured)
	}
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCallEncryption {
			t.Fatalf("first event %v, want call.encryption", ev.Kind)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no call.encryption event")
	}
}

// TestNXDNCipherStateResolvesTalkgroupScopedKey (issue #1261): a trunked
// NXDN site programs every talkgroup with key id 0 but a different key, so
// the chain must hand the resolver the call's talkgroup. On the real-air
// fixture (key id 3, key 32767), the call on the talkgroup the key is
// scoped to descrambles, and the same call on another talkgroup does not.
func TestNXDNCipherStateResolvesTalkgroupScopedKey(t *testing.T) {
	resolver := func(system, algorithm string, keyID uint16, group uint32) ([]byte, bool) {
		if system == "SJPR NXDN" && algorithm == "scrambler" && keyID == 3 && group == 21 {
			return []byte{0x7F, 0xFF}, true // 32767, configured for TG 21 only
		}
		return nil, false
	}
	bus := events.NewBus(16)
	defer bus.Close()
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{}},
		Sink:          &recordingSink{},
		Engine:        &fakeEngine{},
		IQSampleRate:  2_400_000,
		PCMSampleRate: 8000,
		KeyResolver:   resolver,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	frames := loadNXDNScramblerFrames(t)
	run := func(group uint32) *nxdnCipherState {
		st := c.newNXDNCipherState("VOICE-1", "SJPR NXDN", 453_775_000, group)
		for _, f := range frames {
			st.tracker.Frame(f)
		}
		st.tracker.Flush()
		return st
	}
	if st := run(21); !st.keyConfigured || st.tracker.Descrambled == 0 {
		t.Fatalf("tg 21: key_configured=%v descrambled=%d, want the tg 21 key applied", st.keyConfigured, st.tracker.Descrambled)
	}
	if st := run(10); st.keyConfigured || st.tracker.Descrambled != 0 {
		t.Fatalf("tg 10: key_configured=%v descrambled=%d, want no key (it is scoped to tg 21)", st.keyConfigured, st.tracker.Descrambled)
	}
}
