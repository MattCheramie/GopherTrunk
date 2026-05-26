package aprs

import (
	"math"
	"testing"
)

func TestDecodeUncompressedPositionNoTimestampNoMsg(t *testing.T) {
	// APRS 1.0.1 spec format: DDMM.hhH/DDDMM.hhH — two-digit
	// hundredths of a minute. Example "!4903.50N/07201.75W-Test"
	// is the spec's canonical sample.
	info := []byte("!4903.50N/07201.75W-Test 001234")
	p := Decode(info)
	if p.Type != TypePosition {
		t.Fatalf("Type = %v, want TypePosition (raw=%q)", p.Type, p.Raw)
	}
	if p.Position == nil {
		t.Fatal("Position is nil")
	}
	// 49 deg + 3.50/60 minutes ≈ 49.0583.
	if math.Abs(p.Position.Latitude-49.0583) > 1e-3 {
		t.Errorf("lat = %f, want ≈ 49.0583", p.Position.Latitude)
	}
	// -(72 deg + 1.75/60 minutes) ≈ -72.0292.
	if math.Abs(p.Position.Longitude-(-72.0292)) > 1e-3 {
		t.Errorf("lon = %f, want ≈ -72.0292", p.Position.Longitude)
	}
	if p.Position.SymbolTable != '/' || p.Position.SymbolCode != '-' {
		t.Errorf("symbol = %c%c, want /-", p.Position.SymbolTable, p.Position.SymbolCode)
	}
	if p.Position.Comment != "Test 001234" {
		t.Errorf("comment = %q", p.Position.Comment)
	}
	if p.Position.WithMessaging {
		t.Error("WithMessaging = true on '!' packet; want false")
	}
}

func TestDecodeUncompressedPositionWithMessaging(t *testing.T) {
	info := []byte("=4903.50N/07201.75W-")
	p := Decode(info)
	if p.Type != TypePosition || p.Position == nil {
		t.Fatalf("Type = %v", p.Type)
	}
	if !p.Position.WithMessaging {
		t.Error("WithMessaging = false on '=' packet; want true")
	}
}

func TestDecodeSouthernAndWesternHemisphere(t *testing.T) {
	info := []byte("!3354.40S/15110.20W>Sydney")
	p := Decode(info)
	if p.Position == nil {
		t.Fatal("Position is nil")
	}
	if p.Position.Latitude > 0 {
		t.Errorf("lat = %f, want negative (S)", p.Position.Latitude)
	}
	if p.Position.Longitude > 0 {
		t.Errorf("lon = %f, want negative (W)", p.Position.Longitude)
	}
}

func TestDecodePositionWithTimestamp(t *testing.T) {
	// "/" + 7-char timestamp + position. Timestamp is preserved
	// raw on the Position; the parser doesn't try to figure out
	// timezone.
	info := []byte("/092345z4903.50N/07201.75W-")
	p := Decode(info)
	if p.Type != TypePosition || p.Position == nil {
		t.Fatalf("Type = %v, want TypePosition (raw=%q)", p.Type, p.Raw)
	}
	if !p.Position.HasTimestamp {
		t.Error("HasTimestamp = false")
	}
	if p.Position.TimestampRaw != "092345z" {
		t.Errorf("TimestampRaw = %q, want %q", p.Position.TimestampRaw, "092345z")
	}
}

func TestDecodeMessage(t *testing.T) {
	// 9-char addressee, ":", body, optional "{seq}".
	info := []byte(":W1AW-1   :Hello from W2XYZ{42}")
	p := Decode(info)
	if p.Type != TypeMessage {
		t.Fatalf("Type = %v, want TypeMessage", p.Type)
	}
	if p.Message == nil {
		t.Fatal("Message is nil")
	}
	if p.Message.Addressee != "W1AW-1" {
		t.Errorf("Addressee = %q", p.Message.Addressee)
	}
	if p.Message.Body != "Hello from W2XYZ" {
		t.Errorf("Body = %q", p.Message.Body)
	}
	if p.Message.SeqNo != "42" {
		t.Errorf("SeqNo = %q, want 42", p.Message.SeqNo)
	}
}

func TestDecodeMessageAck(t *testing.T) {
	info := []byte(":W1AW-1   :ack42")
	p := Decode(info)
	if p.Type != TypeMessage || p.Message == nil {
		t.Fatalf("Type = %v", p.Type)
	}
	if !p.Message.Ack {
		t.Error("Ack = false, want true")
	}
	if p.Message.SeqNo != "42" {
		t.Errorf("SeqNo = %q", p.Message.SeqNo)
	}
}

func TestDecodeStatus(t *testing.T) {
	info := []byte(">Net control on 146.52 simplex")
	p := Decode(info)
	if p.Type != TypeStatus || p.Status == nil {
		t.Fatalf("Type = %v", p.Type)
	}
	if p.Status.Text != "Net control on 146.52 simplex" {
		t.Errorf("Text = %q", p.Status.Text)
	}
}

func TestDecodeBulletin(t *testing.T) {
	info := []byte(":BLN1     :Weekly net Tuesday 19:00 local")
	p := Decode(info)
	if p.Type != TypeBulletin {
		t.Fatalf("Type = %v, want TypeBulletin", p.Type)
	}
	if p.Bulletin == nil {
		t.Fatal("Bulletin is nil")
	}
	if p.Bulletin.ID != "BLN1" {
		t.Errorf("ID = %q, want BLN1", p.Bulletin.ID)
	}
}

func TestDecodeMicE(t *testing.T) {
	// First byte is 0x60 (back-tick) → Mic-E. We don't decode it
	// yet but the type must be recognised.
	info := []byte("`hello micE payload")
	p := Decode(info)
	if p.Type != TypeMicE {
		t.Errorf("Type = %v, want TypeMicE", p.Type)
	}
}

func TestDecodeUnknownLeavesRawIntact(t *testing.T) {
	info := []byte("xxxxxx unknown payload")
	p := Decode(info)
	if p.Type != TypeUnknown {
		t.Errorf("Type = %v, want TypeUnknown", p.Type)
	}
	if p.Raw != "xxxxxx unknown payload" {
		t.Errorf("Raw = %q", p.Raw)
	}
}

func TestDecodeEmptyInfoSafe(t *testing.T) {
	p := Decode(nil)
	if p.Type != TypeUnknown {
		t.Errorf("Type = %v", p.Type)
	}
	if p.Raw != "" {
		t.Errorf("Raw = %q", p.Raw)
	}
}

func TestPacketStringSwitchesOnType(t *testing.T) {
	for _, c := range []struct {
		info string
		want string
	}{
		{"!4903.50N/07201.75W-Test", "POSITION 49.0583,-72.0292 \"Test\""},
		{":W1AW-1   :Hi there", "MSG to W1AW-1: \"Hi there\""},
		{":W1AW-1   :ack9", "ACK to W1AW-1 seq=9"},
		{">on the net", "STATUS \"on the net\""},
	} {
		got := Decode([]byte(c.info)).String()
		if got != c.want {
			t.Errorf("Decode(%q).String() = %q, want %q", c.info, got, c.want)
		}
	}
}
