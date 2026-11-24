package wmbus

import (
	"bytes"
	"encoding/hex"
	"math"
	"strings"
	"testing"
)

func mustKey(t *testing.T, s string) []byte {
	t.Helper()
	k, err := hex.DecodeString(s)
	if err != nil || len(k) != 16 {
		t.Fatalf("bad key %q", s)
	}
	return k
}

// RFC 4493 section 4 test vectors.
func TestAESCMACRFC4493(t *testing.T) {
	key := mustKey(t, "2b7e151628aed2a6abf7158809cf4f3c")
	msg, _ := hex.DecodeString("6bc1bee22e409f96e93d7e117393172aae2d8a571e03ac9c9eb76fac45af8e5130c81c46a35ce411e5fbc1191a0a52eff69f2445df4f9b17ad2b417be66c3710")
	for _, c := range []struct {
		n    int
		want string
	}{
		{0, "bb1d6929e95937287fa37d129b756746"},
		{16, "070a16b46b4d4144f79bdd9dd04a287c"},
		{40, "dfa66747de9ae63030ca32611497c827"},
		{64, "51f0bebf7e3b9d92fc49741779363cfe"},
	} {
		got, err := aesCMAC(key, msg[:c.n])
		if err != nil || hex.EncodeToString(got) != c.want {
			t.Errorf("CMAC of %d bytes = %x, want %s", c.n, got, c.want)
		}
	}
}

// recordValue returns the value of the first record with the given
// quantity and storage number.
func recordValue(tg Telegram, quantity string, storage int) (float64, bool) {
	for _, r := range tg.Records {
		if r.Quantity == quantity && r.Storage == storage && r.HasValue {
			return r.Value, true
		}
	}
	return 0, false
}

// Real encrypted telegrams from wmbusmeters' test set, with the keys it
// publishes for them, decode to the readings its drivers publish.
func TestParseTelegramDecryptsRealTelegrams(t *testing.T) {
	for _, c := range []struct {
		name, id, key, data string
		ell                 bool
		mode                int
		quantity            string
		value               float64
	}{
		// TPL mode 5, short header (wmbusmeters drivers/src/elf2.xmq).
		{"elf2", "24271170", "ACA5769E7902B8A770A7118C11D5F0F6",
			"6E44010670112724420D7A35006025B2F96D0306197B34C7E4E6F45F434FCB78BCBBE8837A2C253BEC3E958B3BBBFBE6AE465214941AF544237DB8A69E6C4CE8DFB7A1F64860200BB2E995F1A4F501E7702E0A581576F60B58CBDA39E1F150F29E04D689C38FD6C430ABD05DD789E0",
			false, 5, "volume", 17.856},
		// TPL mode 5, LONG header: the IV takes the transport layer's
		// address (aventieswm.xmq).
		{"aventieswm", "61070071", "A004EB23329A477F1DD2D7820B56EB3D",
			"76442104710007612507727100076121042507B5006005E2E95A3C2A1279A5415E6732679B43369FD5FDDDD783EEEBB48236D34E7C94AF0A18A5FDA5F7D64111EB42D4D891622139F2952F9D12A20088DFA4CF8123871123EE1F6C1DCEA414879DDB4E05E508F1826D7EFBA6964DF804C9261EA23BBF03",
			false, 5, "volume", 466.472},
		// Unencrypted ELL (CI 0x8C) in front of a mode 5 TPL (ebeco500.xmq).
		{"ebeco500", "000001E3", "FCC9B808216D9D1F5FF6E1A3CF7D3084",
			"31444514e301000001028c20807a2e002025e25dea4f2bc8ec94920260443a95c2bf3856e3e380e889cc8233ca5d9771f289",
			false, 5, "energy", 68600},
		// ELL AES-CTR (CI 0x8D), then a 0x78 application layer
		// (simulations/serial_aes.msg, kamwater).
		{"kamwater", "76348799", "28F64A24988064A079AA2C807D6102AE",
			"2A442D2C998734761B168D2091D37CAC21E1D68CDAFFCD3DC452BD802913FF7B1706CA9E355D6C2701CC24",
			true, 0, "volume", 6.408},
	} {
		frame := Frame{Mode: ModeT1, Format: FormatA, Data: mustHex(t, c.data), CRCOK: true}
		plain, err := ParseTelegram(frame)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !plain.Encrypted || plain.Decrypted || len(plain.Records) != 0 {
			t.Errorf("%s without a key: encrypted=%v decrypted=%v records=%d", c.name, plain.Encrypted, plain.Decrypted, len(plain.Records))
		}
		tg, err := ParseTelegramKeys(frame, KeyMap(map[string][]byte{c.id: mustKey(t, c.key)}))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if tg.ID != c.id || !tg.Decrypted || tg.Encrypted || tg.DecryptErr != "" || tg.SecurityMode != c.mode {
			t.Fatalf("%s: id=%s decrypted=%v encrypted=%v err=%q mode=%d", c.name, tg.ID, tg.Decrypted, tg.Encrypted, tg.DecryptErr, tg.SecurityMode)
		}
		if v, ok := recordValue(tg, c.quantity, 0); !ok || math.Abs(v-c.value) > 1e-6*math.Max(1, c.value) {
			t.Errorf("%s: %s = %v (found %v), want %v; records %v", c.name, c.quantity, v, ok, c.value, tg.Records)
		}
		// A wrong key is caught by the scheme's own check.
		bad := mustKey(t, c.key)
		bad[0] ^= 1
		w, _ := ParseTelegramKeys(frame, KeyMap(map[string][]byte{c.id: bad}))
		if w.Decrypted || !w.Encrypted || !strings.Contains(w.DecryptErr, "wrong key") || len(w.Records) != 0 {
			t.Errorf("%s wrong key: decrypted=%v err=%q records=%d", c.name, w.Decrypted, w.DecryptErr, len(w.Records))
		}
	}
}

// Security mode 7 (ephemeral key from AES-CMAC over the authentication
// layer's message counter). wmbusmeters publishes no keyed mode 7 telegram,
// so this one was built with an independent encoder (Python cryptography's
// CMAC and AES-CBC) in the shape of a real Techem heat cost allocator's
// telegram (simulation_rtlwmbus_errors.txt): AFL 900F002C25 B3290900 + MAC,
// then 7A B3 00, configuration 0x0720 (mode 7, 2 blocks), extension 0x10
// (KDF 1). The plaintext is 2F2F 0C13 78563412 02FD17 0000 2F…
func TestParseTelegramDecryptsMode7(t *testing.T) {
	data := "40446850677572037408900f002c25b3290900aa7c765a7dfed6737ab30020071075f3b34c80e6f503f3f458ad6dce7e81c45a1eda0ed66db10efdab59cace0bbb"
	key := mustKey(t, "000102030405060708090A0B0C0D0E0F")
	k, _ := mode7Key(key, []byte{0xb3, 0x29, 0x09, 0x00}, []byte{0x67, 0x75, 0x72, 0x03})
	if hex.EncodeToString(k) != "621434523c467e5e689adabcdbb75f08" {
		t.Errorf("Kenc = %x", k)
	}
	frame := Frame{Mode: ModeC1, Format: FormatA, Data: mustHex(t, data), CRCOK: true}
	tg, err := ParseTelegramKeys(frame, KeyMap(map[string][]byte{"03727567": key}))
	if err != nil {
		t.Fatal(err)
	}
	if !tg.HasAFLCounter || tg.AFLCounter != 0x000929b3 || tg.KDF != 1 || tg.SecurityMode != 7 || !tg.Decrypted {
		t.Fatalf("%+v", tg)
	}
	if v, ok := recordValue(tg, "volume", 0); !ok || math.Abs(v-12345.678) > 1e-6 {
		t.Errorf("volume = %v (%v); records %v", v, ok, tg.Records)
	}
	if !bytes.HasPrefix(tg.Payload, []byte{0x2F, 0x2F}) {
		t.Errorf("payload %x", tg.Payload)
	}
	// Without a key the extension byte is still consumed: the payload is
	// the ciphertext alone.
	plain, _ := ParseTelegram(frame)
	if !plain.Encrypted || len(plain.Payload) != 32 || plain.KDF != 1 {
		t.Errorf("no key: encrypted=%v payload %d bytes kdf=%d", plain.Encrypted, len(plain.Payload), plain.KDF)
	}
}

// The reporter's real Techem mode 7 telegram (wmbus_test.go) parses its
// authentication layer's message counter.
func TestParseTelegramReadsAFLCounterOnRealAir(t *testing.T) {
	tg, err := ParseTelegram(Frame{Mode: ModeC1, Format: FormatA, Data: mustHex(t, realAirFrames[1].data), CRCOK: true})
	if err != nil {
		t.Fatal(err)
	}
	if !tg.HasAFLCounter || tg.SecurityMode != 7 {
		t.Errorf("%+v", tg)
	}
}
