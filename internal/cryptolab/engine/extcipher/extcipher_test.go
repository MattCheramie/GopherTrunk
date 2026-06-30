package extcipher

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/crypto/rc4"
)

// mockSpec returns a command spec that re-execs this test binary as a mock
// external cipher (RC4 over key‖iv stands in for the real cipher).
func mockSpec() string {
	return os.Args[0] + " -test.run=TestHelperProcess --"
}

// TestHelperProcess is the mock external cipher. The Go test framework also
// runs it as an ordinary test; without a "--" marker in argv it is a no-op, so
// only the subprocess invocation (which appends "-- <verb> …") acts.
func TestHelperProcess(t *testing.T) {
	args := os.Args
	sep := -1
	for i, a := range args {
		if a == "--" {
			sep = i
			break
		}
	}
	if sep < 0 || sep+1 >= len(args) {
		return // ordinary test run — do nothing
	}
	rest := args[sep+1:]
	ks := func(key, iv []byte, n int) []byte {
		c, _ := rc4.NewCipher(append(append([]byte{}, key...), iv...))
		return c.KeyStream(n)
	}
	switch rest[0] {
	case "keystream":
		key, _ := hex.DecodeString(rest[1])
		iv, _ := hex.DecodeString(rest[2])
		n, _ := strconv.Atoi(rest[3])
		fmt.Println(hex.EncodeToString(ks(key, iv, n)))
	case "brute":
		iv, _ := hex.DecodeString(rest[1])
		known, _ := hex.DecodeString(rest[2])
		bits, _ := strconv.Atoi(rest[3])
		base, _ := hex.DecodeString(rest[4])
		found := ""
		for v := 0; v < (1 << uint(bits)); v++ {
			key := append([]byte(nil), base...)
			for b := 0; b*8 < bits; b++ {
				key[len(key)-1-b] = byte(v >> uint(8*b))
			}
			if bytes.Equal(ks(key, iv, len(known)), known) {
				found = hex.EncodeToString(key)
				break
			}
		}
		fmt.Println(found)
	}
	os.Exit(0)
}

func TestKeystream(t *testing.T) {
	cli, err := New(mockSpec())
	if err != nil {
		t.Fatal(err)
	}
	key := []byte{1, 2, 3}
	iv := []byte{9, 9}
	got, err := cli.Keystream(key, iv, 16)
	if err != nil {
		t.Fatal(err)
	}
	ref := func() []byte {
		c, _ := rc4.NewCipher(append(append([]byte{}, key...), iv...))
		return c.KeyStream(16)
	}()
	if !bytes.Equal(got, ref) {
		t.Fatalf("keystream mismatch: %x vs %x", got, ref)
	}
}

func TestBruteRecoversLowBits(t *testing.T) {
	cli, err := New(mockSpec())
	if err != nil {
		t.Fatal(err)
	}
	// True key: zeros except low 12 bits = 0xABC.
	key := []byte{0, 0, 0, 0x0A, 0xBC}
	iv := []byte{7, 7}
	pt := []byte("KNOWN PLAINTEXT FRAME HERE")
	c, _ := rc4.NewCipher(append(append([]byte{}, key...), iv...))
	known := c.KeyStream(len(pt))

	base := make([]byte, len(key))
	got, found, err := cli.Brute(iv, known, base, 13)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("brute did not find the key")
	}
	if !bytes.Equal(got, key) {
		t.Fatalf("recovered key %x, want %x", got, key)
	}
}

func TestNewRejectsEmpty(t *testing.T) {
	if _, err := New("   "); err == nil {
		t.Fatal("expected error for empty command")
	}
}
