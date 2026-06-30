// Package extcipher bridges to an external, operator-supplied cipher program so
// the toolkit can decrypt and brute-force ciphers it does not (and should not)
// bundle itself — most importantly TETRA TEA1, whose only public
// implementations are licence-incompatible (AGPL) with this Apache-2.0 project
// and which must not be re-authored unverified inside a security-test tool.
//
// The operator points the harness at a program (e.g. MidnightBlue's TEA1 tool
// or sq5bpf/teatime) that implements a tiny, shell-free line protocol:
//
//	<prog> [fixed args…] keystream <key_hex> <iv_hex> <n>
//	    → prints <keystream_hex> (n bytes) on stdout
//	<prog> [fixed args…] brute <iv_hex> <known_keystream_hex> <bits> <base_key_hex>
//	    → prints the recovered <key_hex> (or an empty line if none) on stdout
//
// The `brute` verb keeps the heavy 2^32-style key search in the native
// external cipher; the toolkit orchestrates it, verifies the result, and
// decrypts the corpus with `keystream`. The program is invoked with exec (no
// shell), and key/IV values are passed as separate argv entries, so the cipher
// values cannot inject a command. This capability is CLI-only by design — the
// web console refuses recipes that shell out (see the webserver).
package extcipher

import (
	"context"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Client runs an external cipher program.
type Client struct {
	prog string
	args []string // fixed leading args before the verb
	// KeystreamTimeout / BruteTimeout bound each call.
	KeystreamTimeout time.Duration
	BruteTimeout     time.Duration
}

// New parses a command spec ("prog --flag value") into a Client. The first
// field is the program; the rest are fixed leading arguments. It does not run a
// shell, so the spec must not rely on shell features.
func New(spec string) (*Client, error) {
	fields := strings.Fields(spec)
	if len(fields) == 0 {
		return nil, fmt.Errorf("extcipher: empty command")
	}
	return &Client{
		prog:             fields[0],
		args:             fields[1:],
		KeystreamTimeout: 30 * time.Second,
		BruteTimeout:     30 * time.Minute,
	}, nil
}

// Keystream asks the external cipher for n keystream bytes under (key, iv).
func (c *Client) Keystream(key, iv []byte, n int) ([]byte, error) {
	out, err := c.run(c.KeystreamTimeout, "keystream", hex.EncodeToString(key), hex.EncodeToString(iv), fmt.Sprintf("%d", n))
	if err != nil {
		return nil, err
	}
	ks, err := decodeHexLine(out)
	if err != nil {
		return nil, fmt.Errorf("extcipher: keystream output: %w", err)
	}
	if len(ks) < n {
		return nil, fmt.Errorf("extcipher: keystream returned %d bytes, want %d", len(ks), n)
	}
	return ks[:n], nil
}

// Brute asks the external cipher to search the low `bits` of the key against a
// known keystream (knownKS), with the high bits fixed to baseKey. It returns
// the recovered key and whether one was found.
func (c *Client) Brute(iv, knownKS, baseKey []byte, bits int) ([]byte, bool, error) {
	out, err := c.run(c.BruteTimeout, "brute", hex.EncodeToString(iv), hex.EncodeToString(knownKS), fmt.Sprintf("%d", bits), hex.EncodeToString(baseKey))
	if err != nil {
		return nil, false, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, false, nil
	}
	key, err := decodeHexLine(out)
	if err != nil {
		return nil, false, fmt.Errorf("extcipher: brute output: %w", err)
	}
	return key, true, nil
}

func (c *Client) run(timeout time.Duration, verb string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	full := append(append([]string(nil), c.args...), append([]string{verb}, args...)...)
	cmd := exec.CommandContext(ctx, c.prog, full...)
	out, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("extcipher: %q %s timed out after %s", c.prog, verb, timeout)
	}
	if err != nil {
		return "", fmt.Errorf("extcipher: %q %s failed: %w", c.prog, verb, err)
	}
	return string(out), nil
}

// decodeHexLine decodes the first whitespace-trimmed line of s as hex.
func decodeHexLine(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return hex.DecodeString(s)
}
