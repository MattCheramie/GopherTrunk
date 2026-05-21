package phase2

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func TestServiceOptionsBits(t *testing.T) {
	cases := []struct {
		so        ServiceOptions
		emergency bool
		encrypted bool
		priority  uint8
	}{
		{0x00, false, false, 0},
		{0x80, true, false, 0},
		{0x40, false, true, 0},
		{0xC0, true, true, 0},
		{0xC5, true, true, 5},
		{0x07, false, false, 7},
	}
	for _, c := range cases {
		if c.so.Emergency() != c.emergency || c.so.Encrypted() != c.encrypted ||
			c.so.Priority() != c.priority {
			t.Errorf("ServiceOptions(%#x): emer=%v enc=%v prio=%d, want %v/%v/%d",
				uint8(c.so), c.so.Emergency(), c.so.Encrypted(), c.so.Priority(),
				c.emergency, c.encrypted, c.priority)
		}
	}
}

func TestEncryptionSyncRoundTrip(t *testing.T) {
	in := EncryptionSync{
		AlgorithmID:      0x84, // AES-256 in the P25 algorithm registry
		KeyID:            0xBEEF,
		MessageIndicator: [9]byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
	}
	pdu := EncodeEncryptionSync(in)
	got, ok := pdu.AsEncryptionSync()
	if !ok {
		t.Fatal("AsEncryptionSync returned !ok")
	}
	if got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}

	// Survives an AssembleMACPDU → ParseMACPDU trip (the wire form).
	reparsed, err := ParseMACPDU(AssembleMACPDU(pdu))
	if err != nil {
		t.Fatalf("ParseMACPDU: %v", err)
	}
	if got2, ok := reparsed.AsEncryptionSync(); !ok || got2 != in {
		t.Errorf("wire round-trip = %+v ok=%v, want %+v", got2, ok, in)
	}
}

func TestControlChannelFlagsEncryptedGrant(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_000_000})

	es := EncryptionSync{AlgorithmID: 0x84, KeyID: 0x1234}
	cc.Ingest(EncodeEncryptionSync(es))

	g := grantPDU(0xABCD, 0x00ABCD, 0x1, 0x005)
	g.Payload[0] = 0xC0 // emergency + protected
	cc.Ingest(g)

	grants := countGrants(sub)
	if len(grants) != 1 {
		t.Fatalf("expected 1 grant, got %d", len(grants))
	}
	gr := grants[0]
	if !gr.Encrypted || !gr.Emergency {
		t.Errorf("grant Encrypted=%v Emergency=%v, want both true", gr.Encrypted, gr.Emergency)
	}
	if gr.AlgorithmID != 0x84 || gr.KeyID != 0x1234 {
		t.Errorf("grant alg/key = %#x/%#x, want 0x84/0x1234", gr.AlgorithmID, gr.KeyID)
	}
}

func TestControlChannelClearGrantNotEncrypted(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_000_000})
	cc.Ingest(grantPDU(0x0042, 0x000123, 0x1, 0x005)) // ServiceOptions byte 0x00

	grants := countGrants(sub)
	if len(grants) != 1 {
		t.Fatalf("expected 1 grant, got %d", len(grants))
	}
	if grants[0].Encrypted || grants[0].Emergency {
		t.Errorf("clear grant flagged Encrypted=%v Emergency=%v", grants[0].Encrypted, grants[0].Emergency)
	}
	if grants[0].AlgorithmID != 0 || grants[0].KeyID != 0 {
		t.Errorf("clear grant carried alg/key %#x/%#x", grants[0].AlgorithmID, grants[0].KeyID)
	}
}

// TestEncryptedGrantWithoutSyncDegrades confirms a protected grant with
// no Encryption Sync seen yet still flags Encrypted, with alg/key 0.
func TestEncryptedGrantWithoutSyncDegrades(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_000_000})
	g := grantPDU(0x0042, 0x000123, 0x1, 0x005)
	g.Payload[0] = 0x40 // protected, no emergency
	cc.Ingest(g)

	grants := countGrants(sub)
	if len(grants) != 1 {
		t.Fatalf("expected 1 grant, got %d", len(grants))
	}
	if !grants[0].Encrypted {
		t.Error("protected grant not flagged Encrypted")
	}
	if grants[0].AlgorithmID != 0 || grants[0].KeyID != 0 {
		t.Errorf("alg/key should be 0 without an Encryption Sync, got %#x/%#x",
			grants[0].AlgorithmID, grants[0].KeyID)
	}
}
