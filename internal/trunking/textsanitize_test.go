package trunking

import (
	"io"
	"strings"
	"testing"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func TestSanitizeText(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain ascii", "CPL-SMITH", "CPL-SMITH"},
		{"strips cjk", "AB中CD", "ABCD"},
		{"strips private use", "POLICE", "POLICE"},
		{"strips control and nul", "A\x00B\tC\r\nD", "ABCD"},
		{"keeps printable punctuation", "Cpl. Smith #5 (K-9)", "Cpl. Smith #5 (K-9)"},
		{"all junk", "中文\x00", ""},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitizeText(tc.in); got != tc.want {
				t.Errorf("sanitizeText(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// encodeUTF16 returns s encoded with a BOM in the given endianness, mimicking
// a Windows/SDRTrunk export.
func encodeUTF16(t *testing.T, s string, endian unicode.Endianness) []byte {
	t.Helper()
	enc := unicode.UTF16(endian, unicode.UseBOM).NewEncoder()
	b, _, err := transform.Bytes(enc, []byte(s))
	if err != nil {
		t.Fatalf("encode utf16: %v", err)
	}
	return b
}

func TestDecodeReader(t *testing.T) {
	const want = "Decimal,Alias\n900007,DISPATCH\n"

	t.Run("plain ascii passthrough", func(t *testing.T) {
		got := readAll(t, decodeReader(strings.NewReader(want)))
		if got != want {
			t.Errorf("ascii: got %q, want %q", got, want)
		}
	})

	t.Run("utf8 bom stripped", func(t *testing.T) {
		in := append([]byte{0xEF, 0xBB, 0xBF}, []byte(want)...)
		got := readAll(t, decodeReader(strings.NewReader(string(in))))
		if got != want {
			t.Errorf("utf8-bom: got %q, want %q", got, want)
		}
	})

	t.Run("utf16le bom transcoded", func(t *testing.T) {
		in := encodeUTF16(t, want, unicode.LittleEndian)
		got := readAll(t, decodeReader(strings.NewReader(string(in))))
		if got != want {
			t.Errorf("utf16le: got %q, want %q", got, want)
		}
	})

	t.Run("utf16be bom transcoded", func(t *testing.T) {
		in := encodeUTF16(t, want, unicode.BigEndian)
		got := readAll(t, decodeReader(strings.NewReader(string(in))))
		if got != want {
			t.Errorf("utf16be: got %q, want %q", got, want)
		}
	})
}

func readAll(t *testing.T, r io.Reader) string {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(b)
}
