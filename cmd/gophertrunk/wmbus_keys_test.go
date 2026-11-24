package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/wmbus"
)

// A real mode 5 telegram from wmbusmeters' test set (drivers/src/elf2.xmq)
// and its published key.
const (
	wmbusElf2     = "6E44010670112724420D7A35006025B2F96D0306197B34C7E4E6F45F434FCB78BCBBE8837A2C253BEC3E958B3BBBFBE6AE465214941AF544237DB8A69E6C4CE8DFB7A1F64860200BB2E995F1A4F501E7702E0A581576F60B58CBDA39E1F150F29E04D689C38FD6C430ABD05DD789E0"
	wmbusElf2ID   = "24271170"
	wmbusElf2Key  = "ACA5769E7902B8A770A7118C11D5F0F6"
	wmbusOtherKey = "00112233445566778899AABBCCDDEEFF"
)

func TestLoadWMBusKeys(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "keys.txt")
	if err := os.WriteFile(file, []byte("# my meters\n\n"+wmbusElf2ID+" "+strings.ToLower(wmbusElf2Key)+"\n0000abcd,"+wmbusOtherKey+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	keys, n, err := loadWMBusKeys([]string{"12345678:" + wmbusOtherKey}, file)
	if err != nil || n != 3 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	for _, id := range []string{wmbusElf2ID, "0000ABCD", "12345678"} {
		if _, ok := keys(id); !ok {
			t.Errorf("no key for %s", id)
		}
	}
	if k, _ := keys(wmbusElf2ID); hex.EncodeToString(k) != strings.ToLower(wmbusElf2Key) {
		t.Errorf("key %x", k)
	}
	if keys, n, err := loadWMBusKeys(nil, ""); keys != nil || n != 0 || err != nil {
		t.Errorf("no keys: %v %d %v", keys != nil, n, err)
	}
	for _, bad := range []string{"1234567:" + wmbusOtherKey, "12345678:00112233", "12345678", "12345678:zz112233445566778899AABBCCDDEEFF"} {
		if _, _, err := loadWMBusKeys([]string{bad}, ""); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

// With the meter's key, an encrypted telegram prints its readings, and its
// JSON says it was decrypted; with a wrong key the line says the key failed.
func TestWMBusTelegramDecryptedOutput(t *testing.T) {
	data, _ := hex.DecodeString(wmbusElf2)
	fr := wmbus.Frame{Mode: wmbus.ModeT1, Format: wmbus.FormatA, Data: data, CRCOK: true}
	keys, _, err := loadWMBusKeys([]string{wmbusElf2ID + ":" + wmbusElf2Key}, "")
	if err != nil {
		t.Fatal(err)
	}
	tg, err := wmbus.ParseTelegramKeys(fr, keys)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	writeWMBusTelegram(&out, "", tg, nil, fr, wmbus.FrameInfo{}, false, false)
	if s := out.String(); !strings.Contains(s, "(decrypted)") || !strings.Contains(s, "volume") || strings.Contains(s, "encrypted:") {
		t.Errorf("text:\n%s", s)
	}
	out.Reset()
	writeWMBusTelegram(&out, "", tg, nil, fr, wmbus.FrameInfo{}, false, true)
	var j wmbusJSON
	if err := json.Unmarshal(out.Bytes(), &j); err != nil {
		t.Fatal(err)
	}
	if !j.Decrypted || j.Encrypted || len(j.Records) == 0 {
		t.Errorf("json %s", out.String())
	}

	wrong, _, _ := loadWMBusKeys([]string{wmbusElf2ID + ":" + wmbusOtherKey}, "")
	tg, _ = wmbus.ParseTelegramKeys(fr, wrong)
	out.Reset()
	writeWMBusTelegram(&out, "", tg, nil, fr, wmbus.FrameInfo{}, false, false)
	if s := out.String(); !strings.Contains(s, "key did not decrypt") {
		t.Errorf("wrong key:\n%s", s)
	}
}
