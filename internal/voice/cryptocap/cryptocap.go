// Package cryptocap is the bridge from GopherTrunk's live decoders to the
// offline cryptolab toolkit. The voice composer already extracts each
// encrypted P25 call's Message Indicator (the per-superframe IV), algorithm
// id, and key id, and recovers the encrypted voice frames; this package
// captures those as a JSONL artifact that `gophertrunk cryptolab ks
// reuse/mtp` consumes directly.
//
// The captured artifact feeds `gophertrunk cryptolab assess`, the security-test
// harness that attempts to break the encryption offline by every applicable
// method (keystream reuse, known-plaintext recovery, default/weak keys against
// the real ADP/DES/AES ciphers, keystream-LFSR prediction) and grades the
// deployment's resistance. This package itself only records the encrypted
// frames and their IVs; the decryption attempts run later, offline, in the
// assess harness.
//
// The Sink interface lives here (a small leaf package) so the composer can
// depend on it without importing the optional, build-tagged cryptolab tree,
// and so the file writer is always compiled and unit-tested regardless of
// build tags.
package cryptocap

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Frame is one captured encrypted unit handed to a Sink. MI is the IV
// (P25 Message Indicator); CT is the encrypted payload (the superframe's
// voice frames). The remaining fields are provenance for the analyst.
type Frame struct {
	System   string
	Protocol string
	Serial   string
	CallID   uint64
	TG       uint32
	AlgID    uint8
	KeyID    uint16
	MI       []byte
	CT       []byte
	At       time.Time
}

// Sink receives captured encrypted frames. Implementations must be safe for
// concurrent use — the composer runs one voice chain per device in its own
// goroutine, and they share a single sink.
type Sink interface {
	WriteCryptoFrame(Frame)
}

// record is the on-disk JSON shape. The first three fields (label, iv, ct)
// are exactly what `cryptolab ks` parses; the rest are ignored by that parser
// but preserved for provenance and ad-hoc filtering.
type record struct {
	Label    string `json:"label"`
	IV       string `json:"iv"`
	CT       string `json:"ct"`
	System   string `json:"system,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	TG       uint32 `json:"tg,omitempty"`
	AlgID    uint8  `json:"algid"`
	KeyID    uint16 `json:"keyid"`
	At       string `json:"at,omitempty"`
}

// FileWriter is a goroutine-safe Sink that appends one JSON line per frame to
// a file. It is the concrete sink the daemon wires in when
// recordings.crypto_capture_path is set.
type FileWriter struct {
	mu  sync.Mutex
	f   *os.File
	w   *bufio.Writer
	seq uint64
}

// NewFileWriter opens (creating or appending to) path for capture.
func NewFileWriter(path string) (*FileWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("cryptocap: open %s: %w", path, err)
	}
	return &FileWriter{f: f, w: bufio.NewWriter(f)}, nil
}

// WriteCryptoFrame appends one record. A frame with no IV or no ciphertext is
// dropped — without both there is nothing for keystream-reuse analysis to use.
// Errors are intentionally swallowed (best-effort capture must never disturb
// the live voice path); the writer flushes on Close.
func (fw *FileWriter) WriteCryptoFrame(fr Frame) {
	if len(fr.MI) == 0 || len(fr.CT) == 0 {
		return
	}
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.seq++
	label := fr.Serial
	if label == "" {
		label = "frame"
	}
	rec := record{
		Label:    fmt.Sprintf("%s-%d-%d", label, fr.CallID, fw.seq),
		IV:       hex.EncodeToString(fr.MI),
		CT:       hex.EncodeToString(fr.CT),
		System:   fr.System,
		Protocol: fr.Protocol,
		TG:       fr.TG,
		AlgID:    fr.AlgID,
		KeyID:    fr.KeyID,
	}
	if !fr.At.IsZero() {
		rec.At = fr.At.UTC().Format(time.RFC3339Nano)
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return
	}
	fw.w.Write(b)
	fw.w.WriteByte('\n')
}

// Close flushes and closes the underlying file. Idempotent-safe to call once.
func (fw *FileWriter) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if fw.f == nil {
		return nil
	}
	err := fw.w.Flush()
	if cerr := fw.f.Close(); err == nil {
		err = cerr
	}
	fw.f = nil
	return err
}
