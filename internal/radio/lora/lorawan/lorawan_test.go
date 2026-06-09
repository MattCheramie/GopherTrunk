package lorawan

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"testing"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("hex %q: %v", s, err)
	}
	return b
}

func key16(t *testing.T, s string) [16]byte {
	t.Helper()
	b := mustHex(t, s)
	if len(b) != 16 {
		t.Fatalf("want 16-byte key, got %d", len(b))
	}
	var k [16]byte
	copy(k[:], b)
	return k
}

// TestAESCMACVectors checks the AES-CMAC implementation against the
// authoritative RFC 4493 §4 test vectors (key 2b7e1516...).
func TestAESCMACVectors(t *testing.T) {
	key := key16(t, "2b7e151628aed2a6abf7158809cf4f3c")
	cases := []struct {
		msg, want string
	}{
		{"", "bb1d6929e95937287fa37d129b756746"},
		{"6bc1bee22e409f96e93d7e117393172a", "070a16b46b4d4144f79bdd9dd04a287c"},
		{
			"6bc1bee22e409f96e93d7e117393172aae2d8a571e03ac9c9eb76fac45af8e51" +
				"30c81c46a35ce411",
			"dfa66747de9ae63030ca32611497c827",
		},
	}
	for i, c := range cases {
		got := aesCMAC(key, mustHex(t, c.msg))
		if hex.EncodeToString(got[:]) != c.want {
			t.Errorf("case %d: got %x want %s", i, got, c.want)
		}
	}
}

// buildUplink assembles an unconfirmed-uplink PHY payload for round-trip
// testing: it encrypts the payload with AppSKey, computes the MIC with
// NwkSKey, and returns the full PHYPayload bytes.
func buildUplink(devAddr uint32, fcnt uint16, fport uint8, payload []byte, nwk, app [16]byte) []byte {
	phy := []byte{byte(UnconfirmedDataUp) << 5}
	var fhdr [7]byte
	binary.LittleEndian.PutUint32(fhdr[0:4], devAddr)
	fhdr[4] = 0x00 // FCtrl: no FOpts
	binary.LittleEndian.PutUint16(fhdr[5:7], fcnt)
	phy = append(phy, fhdr[:]...)
	phy = append(phy, fport)

	// Encrypt payload via the same symmetric routine.
	enc := (&Message{
		MType:      UnconfirmedDataUp,
		DevAddr:    devAddr,
		FCnt:       fcnt,
		FRMPayload: payload,
	}).DecryptFRMPayload(app)
	phy = append(phy, enc...)

	// MIC over MHDR|FHDR|FPort|FRMPayload.
	m := &Message{MType: UnconfirmedDataUp, DevAddr: devAddr, FCnt: fcnt, mac: phy}
	mic := m.ComputeMIC(nwk)
	return append(phy, mic[:]...)
}

func TestLoRaWANRoundTrip(t *testing.T) {
	nwk := key16(t, "0102030405060708090a0b0c0d0e0f10")
	app := key16(t, "a0a1a2a3a4a5a6a7a8a9aaabacadaeaf")
	devAddr := uint32(0x26011BDA)
	payload := []byte("temp=21.5C")

	phy := buildUplink(devAddr, 42, 1, payload, nwk, app)

	ks := NewKeyStore()
	ks.Add(devAddr, SessionKeys{NwkSKey: nwk, AppSKey: app})

	d, err := ks.Decode(phy)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if d.MType != UnconfirmedDataUp {
		t.Errorf("MType=%s", d.MType)
	}
	if d.DevAddr != devAddr {
		t.Errorf("DevAddr=%08x want %08x", d.DevAddr, devAddr)
	}
	if d.FCnt != 42 || d.FPort != 1 {
		t.Errorf("FCnt=%d FPort=%d", d.FCnt, d.FPort)
	}
	if !d.HasKeys || !d.MICOK {
		t.Errorf("HasKeys=%v MICOK=%v", d.HasKeys, d.MICOK)
	}
	if !d.Decrypted || !bytes.Equal(d.Plaintext, payload) {
		t.Errorf("decrypt: got %q want %q", d.Plaintext, payload)
	}

	// A wrong NwkSKey must fail the MIC.
	bad := nwk
	bad[0] ^= 0xFF
	ks2 := NewKeyStore()
	ks2.Add(devAddr, SessionKeys{NwkSKey: bad, AppSKey: app})
	d2, _ := ks2.Decode(phy)
	if d2.MICOK {
		t.Error("MIC verified under wrong key")
	}
}

func TestParseJoinRequest(t *testing.T) {
	phy := []byte{byte(JoinRequest) << 5}
	phy = append(phy, make([]byte, 18)...) // JoinEUI+DevEUI+DevNonce
	binary.LittleEndian.PutUint16(phy[17:19], 0xBEEF)
	phy = append(phy, 1, 2, 3, 4) // MIC
	m, err := Parse(phy)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if m.MType != JoinRequest {
		t.Fatalf("MType=%s", m.MType)
	}
	if m.DevNonce != 0xBEEF {
		t.Errorf("DevNonce=%04x", m.DevNonce)
	}
}

func TestParseTooShort(t *testing.T) {
	if _, err := Parse([]byte{0x40, 0x00}); err == nil {
		t.Fatal("expected error for short payload")
	}
}
