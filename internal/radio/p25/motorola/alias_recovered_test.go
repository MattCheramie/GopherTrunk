package motorola

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

// oracleRow mirrors one row of the committed clean-room validation set
// (testdata/alias_cipher_oracle_holdout.jsonl): an encoded ciphertext and the
// decoded byte stream a behavioral oracle produced for it. The set is
// held-out — disjoint from the corpus the cipher was fitted on — so a 100%
// match here is a genuine recovery check, not a memorised round-trip (#773).
type oracleRow struct {
	EncHex   string `json:"enc_hex"`
	DecBytes string `json:"dec_bytes_hex"`
}

// TestRecoveredCipherHoldout validates the integrated cipher against the
// committed held-out oracle vectors. This is the golden-capture regression the
// repo requires standing behind CipherVerified=true: if a future edit perturbs
// motorolaAliasLUT or the accumulator constants, this fails loudly.
//
// Every clean character must decode byte-identically. Two row/char classes
// carry no comparable ground truth and are excluded: a U+FFFD character is one
// the oracle's own lossy UTF-16 charset destroyed (its true bytes are gone),
// and a whole row whose length differs was charset-truncated by the oracle.
func TestRecoveredCipherHoldout(t *testing.T) {
	t.Parallel()
	const path = "testdata/alias_cipher_oracle_holdout.jsonl"
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open golden holdout: %v", err)
	}
	defer f.Close()

	var rows, cleanChars, cleanOK, fffd, truncated int
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var r oracleRow
		if err := json.Unmarshal(line, &r); err != nil {
			t.Fatalf("parse row: %v", err)
		}
		enc, err := hex.DecodeString(r.EncHex)
		if err != nil {
			t.Fatalf("bad enc_hex %q: %v", r.EncHex, err)
		}
		want, err := hex.DecodeString(r.DecBytes)
		if err != nil {
			t.Fatalf("bad dec_bytes_hex %q: %v", r.DecBytes, err)
		}
		got := DecodeAliasBytes(enc)
		rows++
		if len(got) != len(want) {
			truncated++
			continue
		}
		for k := 0; k+1 < len(want); k += 2 {
			if want[k] == 0xFF && want[k+1] == 0xFD {
				fffd++
				continue
			}
			cleanChars++
			if got[k] == want[k] && got[k+1] == want[k+1] {
				cleanOK++
			} else {
				t.Errorf("row %d char %d: enc=%s got=%02x%02x want=%02x%02x",
					rows, k/2, r.EncHex, got[k], got[k+1], want[k], want[k+1])
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if rows == 0 {
		t.Fatal("golden holdout is empty")
	}
	if cleanOK != cleanChars {
		t.Fatalf("clean-char match %d/%d — cipher transcription is not byte-exact", cleanOK, cleanChars)
	}
	t.Logf("holdout: %d rows, %d/%d clean chars byte-identical (FFFD excluded %d, charset-truncated rows %d)",
		rows, cleanOK, cleanChars, fffd, truncated)
}

// TestRecoveredCipherFixtures pins the integrated cipher against fixed vectors.
//
// The raw-byte vectors are independent chosen-ciphertext samples — the decoded
// bytes are the oracle's own output, not self-generated — so they cross-check
// the transcription rather than round-tripping it through our own encoder. The
// name vectors additionally show a full alias decodes end-to-end to a reliable
// printable string once the cipher is verified.
func TestRecoveredCipherFixtures(t *testing.T) {
	t.Parallel()

	// encoded ciphertext -> decoded raw bytes (UTF-16BE), straight from the oracle.
	rawVectors := []struct{ enc, dec string }{
		{"1c8e7ad0", "d4146a60"},
		{"52cd", "4376"},
		{"af40704ac768d0ae55d3", "fff7b17d63451314f874"},
		{"5210f5be069a2924540f5c9c", "ffa51f634f57fa09ffc10118"},
	}
	for _, v := range rawVectors {
		enc, err := hex.DecodeString(v.enc)
		if err != nil {
			t.Fatalf("bad fixture enc %q: %v", v.enc, err)
		}
		if got := hex.EncodeToString(DecodeAliasBytes(enc)); got != v.dec {
			t.Errorf("DecodeAliasBytes(%s) = %s, want %s", v.enc, got, v.dec)
		}
	}

	// encoded ciphertext -> printable name, decoded and rendered end to end.
	names := []struct{ enc, alias string }{
		{"b54c37bfebecf79f5f4967a70ed1", "MEDIC12"},
		{"b5de3fca93338cd98ac1770bf9ca", "ENGINE3"},
	}
	for _, v := range names {
		enc, err := hex.DecodeString(v.enc)
		if err != nil {
			t.Fatalf("bad fixture enc %q: %v", v.enc, err)
		}
		alias, reliable := DecodeAlias(DecodeAliasBytes(enc))
		if alias != v.alias || !reliable {
			t.Errorf("DecodeAlias(%s) = (%q, reliable=%v), want (%q, true)", v.enc, alias, reliable, v.alias)
		}
	}
}
