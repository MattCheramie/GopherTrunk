package motorola

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestProcessDecodesGroupVoiceGrantAfterSync builds a bit stream
// containing 30 bits of padding (so the SyncDetector primes), the
// 24-bit outbound sync, and a 32-bit OSW carrying a Group Voice
// Channel Grant. Process must publish a KindGrant on the bus.
func TestProcessDecodesGroupVoiceGrantAfterSync(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 851_012_500,
	})

	// Command = (opcode << 4) | LCN. Group Voice Channel Grant on
	// LCN 5.
	cmd := (uint16(OpGroupVoiceChannelGrant) << 4) | 0x5
	osw := OSW{Address: 0xCAFE, Command: cmd}
	oswBits := OSWBits(osw)

	stream := make([]byte, 30)
	stream = append(stream, OutboundSyncBits()...)
	stream = append(stream, oswBits...)

	cc.Process(stream, 0)

	var sawGrant bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				g, ok := ev.Payload.(trunking.Grant)
				if !ok {
					t.Fatalf("Grant payload type = %T, want trunking.Grant", ev.Payload)
				}
				if g.System != "Sys" {
					t.Errorf("Grant.System = %q, want Sys", g.System)
				}
				if g.Protocol != "motorola" {
					t.Errorf("Grant.Protocol = %q, want motorola", g.Protocol)
				}
				if g.GroupID != 0xCAFE {
					t.Errorf("Grant.GroupID = %#x, want 0xCAFE", g.GroupID)
				}
				sawGrant = true
			}
		default:
			if !sawGrant {
				t.Errorf("Process did not publish a KindGrant for a valid OSW")
			}
			return
		}
	}
}

func TestProcessIgnoresGarbageWithoutSync(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})

	garbage := make([]byte, 1000)
	for i := range garbage {
		garbage[i] = byte(i & 1)
	}
	cc.Process(garbage, 0)

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event from garbage stream: %v", ev.Kind)
	default:
	}
}

func TestProcessHandlesSyncSpanningCalls(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})

	cmd := (uint16(OpGroupVoiceChannelGrant) << 4) | 0x3
	osw := OSW{Address: 0xBEEF, Command: cmd}
	oswBits := OSWBits(osw)

	chunk1 := make([]byte, 30)
	chunk1 = append(chunk1, OutboundSyncBits()...)
	cc.Process(chunk1, 0)
	cc.Process(oswBits, len(chunk1))

	var sawGrant bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				sawGrant = true
			}
		default:
			if !sawGrant {
				t.Errorf("Process did not publish a KindGrant across the chunk boundary")
			}
			return
		}
	}
}

// TestProcessBCHModeDecodesEncodedOSW: in BCHOn mode the adapter
// must read two 64-bit BCH(64,16,11) codewords after sync and
// recover the underlying 32-bit OSW.
func TestProcessBCHModeDecodesEncodedOSW(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus: bus, Log: slog.Default(), SystemName: "Sys",
		FrequencyHz: 851_012_500,
	})
	cc.SetBCHMode(BCHOn)

	cmd := (uint16(OpGroupVoiceChannelGrant) << 4) | 0x4
	osw := OSW{Address: 0xC0DE, Command: cmd}

	// Encode the OSW's two 16-bit halves through BCH(64,16,11)
	// and pack into a 128-bit channel stream.
	cw1 := framingBCHEncode64_16(osw.Address)
	cw2 := framingBCHEncode64_16(osw.Command)
	encoded := make([]byte, 128)
	for i := 0; i < 64; i++ {
		if cw1&(uint64(1)<<uint(63-i)) != 0 {
			encoded[i] = 1
		}
	}
	for i := 0; i < 64; i++ {
		if cw2&(uint64(1)<<uint(63-i)) != 0 {
			encoded[64+i] = 1
		}
	}

	stream := make([]byte, 30)
	stream = append(stream, OutboundSyncBits()...)
	stream = append(stream, encoded...)

	cc.Process(stream, 0)

	var sawGrant bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				g, _ := ev.Payload.(trunking.Grant)
				if g.GroupID == 0xC0DE {
					sawGrant = true
				}
			}
		default:
			if !sawGrant {
				t.Errorf("BCHOn mode did not publish a KindGrant for the encoded OSW")
			}
			return
		}
	}
}

// framingBCHEncode64_16 is a tiny shim so this test file doesn't
// import framing directly (which would require widening the
// existing test file's import set).
func framingBCHEncode64_16(data uint16) uint64 {
	// Reproduce framing.BCHEncode64_16 inline. Keeping this local
	// avoids a wider import for one test.
	const generator uint64 = 0xF391E2F34B99
	info := uint64(data) & 0xFFFF
	rem := info << 47
	for i := 62; i >= 47; i-- {
		if rem&(uint64(1)<<uint(i)) != 0 {
			rem ^= generator << uint(i-47)
		}
	}
	cw63 := (info << 47) | (rem & ((uint64(1) << 47) - 1))
	// Append overall-even parity bit at bit 0.
	parity := uint64(0)
	for b := 0; b < 63; b++ {
		parity ^= (cw63 >> uint(b)) & 1
	}
	return (cw63 << 1) | parity
}

func TestSyncDetectorReset(t *testing.T) {
	det := NewSyncDetector(OutboundSyncBits(), 0)
	junk := make([]byte, 100)
	det.Process(nil, junk, 0)
	det.Reset()
	if det.primed != 0 {
		t.Errorf("post-Reset primed = %d, want 0", det.primed)
	}
	if det.pos != 0 {
		t.Errorf("post-Reset pos = %d, want 0", det.pos)
	}
}

// TestSetBCHModeDefault confirms the zero-value ControlChannel
// uses BCHOff, the accessor mirrors the setter, and the
// SetBCHMode toggle round-trips both ways.
func TestSetBCHModeDefault(t *testing.T) {
	cc := New(Options{})
	if cc.bchMode != BCHOff {
		t.Errorf("default bchMode = %v, want BCHOff", cc.bchMode)
	}
	if got := cc.BCHMode(); got != BCHOff {
		t.Errorf("BCHMode() = %v, want BCHOff", got)
	}
	cc.SetBCHMode(BCHOn)
	if cc.bchMode != BCHOn {
		t.Errorf("SetBCHMode(BCHOn) did not take effect")
	}
	if got := cc.BCHMode(); got != BCHOn {
		t.Errorf("BCHMode() = %v, want BCHOn", got)
	}
	cc.SetBCHMode(BCHOff)
	if cc.bchMode != BCHOff {
		t.Errorf("SetBCHMode(BCHOff) did not take effect")
	}
}

// TestParseBCHMode covers the config-string → BCHMode mapping the
// ccdecoder connector uses to translate the `motorola_bch_mode`
// YAML field into a SetBCHMode call.
func TestParseBCHMode(t *testing.T) {
	cases := []struct {
		in   string
		want BCHMode
		ok   bool
	}{
		{"", BCHOff, true},
		{"off", BCHOff, true},
		{"false", BCHOff, true},
		{"0", BCHOff, true},
		{"on", BCHOn, true},
		{"ON", BCHOn, true},
		{"true", BCHOn, true},
		{"1", BCHOn, true},
		{" on ", BCHOn, true},
		{"nonsense", BCHOff, false},
	}
	for _, tc := range cases {
		got, ok := ParseBCHMode(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ParseBCHMode(%q) = (%v, %v), want (%v, %v)",
				tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
