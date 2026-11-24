package wmbus

import (
	"testing"
)

type wantRec struct {
	quantity string
	storage  int
	value    float64
	text     string
}

func checkRecords(t *testing.T, name string, tg Telegram, want []wantRec) {
	t.Helper()
	for _, w := range want {
		found := false
		for _, r := range tg.Records {
			if r.Quantity != w.quantity || r.Storage != w.storage || r.Function != "instantaneous" {
				continue
			}
			if w.text != "" && r.Text == w.text || w.text == "" && r.HasValue && r.Value == w.value {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no %s[%d] = %v%s; records %v", name, w.quantity, w.storage, w.value, w.text, tg.Records)
		}
	}
}

// Qundis walk-by data sets decode to the readings wmbusmeters' qcaloric
// driver publishes for its own test telegrams (drivers/src/qcaloric.xmq).
func TestQundisWalkByWMBusmetersVectors(t *testing.T) {
	for _, c := range []struct {
		name, data string
		want       []wantRec
	}{
		{"MyElement2", "49449344939291903408780DFF5F350082180000800007B06EFFFF970000009F2C70020000BE26970000000000010018002E001F002E0023FF210008000500020000002F046D220FA227",
			[]wantRec{{"heat cost allocation", 0, 97, ""}, {"date", 1, 0, "2020-12-31"}, {"heat cost allocation", 1, 270, ""},
				{"date", 17, 0, "2021-06-30"}, {"heat cost allocation", 17, 97, ""}}},
		{"Caloric55", "49449344670450903308780dff5f350082041800f00007b06eb82b02000000fe29000000001f35000000000000000000000000000000000000000000000000000000002f046d04111836",
			[]wantRec{{"heat cost allocation", 0, 2, ""}, {"date", 1, 0, "2023-09-30"}, {"heat cost allocation", 1, 0, ""},
				{"date", 17, 0, "2024-05-31"}}},
	} {
		tg, err := ParseTelegram(Frame{Data: mustHex(t, c.data), CRCOK: true})
		if err != nil || !tg.QundisWalkBy {
			t.Fatalf("%s: walk-by=%v err=%v", c.name, tg.QundisWalkBy, err)
		}
		checkRecords(t, c.name, tg, c.want)
	}
}

// The reporter's meter 67837559 (behind a radio converter, 23 Nov walk)
// sent both forms minutes apart: a standard CI 0x72 telegram and a CI 0x78
// walk-by data set. They carry the same readings.
func TestQundisWalkByMatchesSameMetersStandardTelegram(t *testing.T) {
	std := "41449344597583674637725975836793444604ab0000200c06049308004c0684330600426c3f3ccc080644560800c2086c5f3a02fd170000326cffff046d1c0d573b"
	wb := "5844934459758367463778077959758367934446040dff5f350082ab00005f0107c006ffff049308003f3c843306005f3a445608007e103714d3169c11860eb0076e042d026c01b1013d035a0b2f02fd170000046d1d0d573b"
	want := []wantRec{{"energy", 0, 89304000, ""}, {"date", 1, 0, "2025-12-31"}, {"energy", 1, 63384000, ""},
		{"date", 17, 0, "2026-10-31"}, {"energy", 17, 85644000, ""}}
	for name, h := range map[string]string{"standard": std, "walk-by": wb} {
		tg, err := ParseTelegram(Frame{Data: mustHex(t, h), CRCOK: true})
		if err != nil {
			t.Fatal(err)
		}
		checkRecords(t, name, tg, want)
	}
}

// An encrypted walk-by data set ([4] = 0x35) decrypts with the meter's key.
// Built from two of the reporter's clear blocks with an independent encoder
// (Python cryptography AES-CBC, IV = M + A + 8 × counter): one under the link
// layer's address, one behind a radio converter, where the IV takes the
// meter's own identity from its 07 79 record. A wrong IV address corrupts the
// first block's bytes after the header, i.e. the current reading, so the
// values are what proves the address.
func TestQundisWalkByDecrypts(t *testing.T) {
	key := mustKey(t, "0F0E0D0C0B0A09080706050403020100")
	for _, c := range []struct {
		name, id, data string
		want           []wantRec
	}{
		{"hca", "38377271", "49449344717237383608780dff5f3500821d00357382becac583b50dbc4da4187430eee5d2b46f03144ccb0811b26631bc0c51b33a1aca3f813573b79ff23a6a47fcc6fa046d160d573b",
			[]wantRec{{"heat cost allocation", 0, 296, ""}, {"heat cost allocation", 1, 771, ""}, {"heat cost allocation", 17, 272, ""}}},
		{"converter", "67837559", "5844934459758367463778077959758367934446040dff5f350082ab0035ea2317479b57435fd6188591cae8804bf7d2dab5eae4badd3701f90ccbcfc55f7ada9a971bff4bc6d0b0fdfea95bf77302fd170000046d1d0d573b",
			[]wantRec{{"energy", 0, 89304000, ""}, {"energy", 1, 63384000, ""}, {"energy", 17, 85644000, ""}}},
	} {
		f := Frame{Data: mustHex(t, c.data), CRCOK: true}
		plain, _ := ParseTelegram(f)
		if plain.QundisWalkBy || !plain.Encrypted {
			t.Errorf("%s without key: walk-by=%v encrypted=%v", c.name, plain.QundisWalkBy, plain.Encrypted)
		}
		tg, err := ParseTelegramKeys(f, KeyMap(map[string][]byte{c.id: key}))
		if err != nil || !tg.QundisWalkBy || !tg.Decrypted || tg.Encrypted {
			t.Fatalf("%s: walk-by=%v decrypted=%v encrypted=%v err=%v %q", c.name, tg.QundisWalkBy, tg.Decrypted, tg.Encrypted, err, tg.DecryptErr)
		}
		checkRecords(t, c.name, tg, c.want)
		bad := append([]byte(nil), key...)
		bad[3] ^= 0x40
		w, _ := ParseTelegramKeys(f, KeyMap(map[string][]byte{c.id: bad}))
		if w.QundisWalkBy || w.DecryptErr == "" {
			t.Errorf("%s wrong key accepted: %+v", c.name, w)
		}
	}
}
