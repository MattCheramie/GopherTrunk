package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/radio/wmbus"
)

// wmbusKeyFlags collects repeated -key ID:HEX flags.
type wmbusKeyFlags []string

func (k *wmbusKeyFlags) String() string     { return strings.Join(*k, ",") }
func (k *wmbusKeyFlags) Set(v string) error { *k = append(*k, v); return nil }

// parseWMBusKey parses one "ID:KEY" (or "ID KEY", "ID,KEY", "ID=KEY") entry:
// the meter's 8-digit identification number as the decoder prints it and its
// 32-hex-digit AES-128 key.
func parseWMBusKey(s string) (id string, key []byte, err error) {
	f := strings.FieldsFunc(s, func(r rune) bool {
		return r == ':' || r == ',' || r == '=' || r == ' ' || r == '\t' || r == ';'
	})
	if len(f) != 2 {
		return "", nil, fmt.Errorf("%q: want ID:KEY (8-digit meter ID, 32 hex digits of AES-128 key)", s)
	}
	id = strings.ToUpper(f[0])
	if len(id) != 8 {
		return "", nil, fmt.Errorf("%q: meter ID %q must be the 8 digits the decoder prints", s, f[0])
	}
	key, err = hex.DecodeString(f[1])
	if err != nil || len(key) != 16 {
		return "", nil, fmt.Errorf("%q: key must be 32 hex digits (AES-128)", s)
	}
	return id, key, nil
}

// loadWMBusKeys builds the key lookup from -key flags and a -keys file (one
// ID:KEY per line; blank lines and lines starting with # are ignored).
// It returns nil when no key was given.
func loadWMBusKeys(flags []string, file string) (wmbus.KeyLookup, int, error) {
	m := map[string][]byte{}
	add := func(src, s string) error {
		id, key, err := parseWMBusKey(s)
		if err != nil {
			return fmt.Errorf("%s: %w", src, err)
		}
		m[id] = key
		return nil
	}
	if file != "" {
		fh, err := os.Open(file)
		if err != nil {
			return nil, 0, err
		}
		defer fh.Close()
		sc := bufio.NewScanner(fh)
		for n := 1; sc.Scan(); n++ {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if err := add(fmt.Sprintf("%s line %d", file, n), line); err != nil {
				return nil, 0, err
			}
		}
		if err := sc.Err(); err != nil {
			return nil, 0, err
		}
	}
	for _, s := range flags {
		if err := add("-key", s); err != nil {
			return nil, 0, err
		}
	}
	if len(m) == 0 {
		return nil, 0, nil
	}
	return wmbus.KeyMap(m), len(m), nil
}
