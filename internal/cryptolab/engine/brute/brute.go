// Package brute is the toolkit's generic keyspace brute-force engine:
// pluggable Cipher, KeySpace, and Scorer interfaces driving an ordered,
// resumable sweep that keeps the top-scoring candidates. It ships the
// classical building blocks (XOR, Caesar) and scorers (printable, English
// frequency, known-plaintext crib), plus the smart single-byte and
// repeating-key XOR solvers that decompose the otherwise-astronomical
// repeating-key keyspace into per-column single-byte sweeps.
package brute

import (
	"context"
	"sort"
)

// Cipher decrypts ciphertext under a key. Implementations are pure.
type Cipher interface {
	Decrypt(key, ct []byte) []byte
}

// KeySpace is an ordered, seekable enumerator over candidate keys. Ordered
// + seekable is what makes a sweep resumable from a checkpointed position.
type KeySpace interface {
	Next() (key []byte, ok bool)
	Position() uint64
	Seek(pos uint64)
	Size() uint64 // 0 means unknown/unbounded
}

// Scorer ranks a decrypted plaintext; higher is more plausible.
type Scorer interface {
	Score(pt []byte) float64
}

// Candidate is one scored decryption.
type Candidate struct {
	Key       []byte
	Plaintext []byte
	Score     float64
	Position  uint64
}

// Options configure a Run.
type Options struct {
	TopN     int                    // keep this many best candidates (default 5)
	OnTick   func(pos, size uint64) // optional progress callback
	TickEach uint64                 // call OnTick every N keys (default 65536)
}

// Run sweeps ks, decrypting ct with c and scoring with sc, returning the
// TopN highest-scoring candidates in descending score order. It honors
// ctx cancellation between keys.
func Run(ctx context.Context, c Cipher, ks KeySpace, sc Scorer, ct []byte, opt Options) []Candidate {
	if opt.TopN <= 0 {
		opt.TopN = 5
	}
	if opt.TickEach == 0 {
		opt.TickEach = 65536
	}
	var best []Candidate
	size := ks.Size()
	for {
		select {
		case <-ctx.Done():
			sortCandidates(best)
			return best
		default:
		}
		pos := ks.Position()
		key, ok := ks.Next()
		if !ok {
			break
		}
		pt := c.Decrypt(key, ct)
		score := sc.Score(pt)
		best = insertTop(best, Candidate{Key: append([]byte(nil), key...), Plaintext: pt, Score: score, Position: pos}, opt.TopN)
		if opt.OnTick != nil && pos%opt.TickEach == 0 {
			opt.OnTick(pos, size)
		}
	}
	sortCandidates(best)
	return best
}

func insertTop(best []Candidate, c Candidate, n int) []Candidate {
	if len(best) < n {
		return append(best, c)
	}
	// Replace the current minimum if c beats it.
	minI := 0
	for i := 1; i < len(best); i++ {
		if best[i].Score < best[minI].Score {
			minI = i
		}
	}
	if c.Score > best[minI].Score {
		best[minI] = c
	}
	return best
}

func sortCandidates(c []Candidate) {
	sort.SliceStable(c, func(i, j int) bool { return c[i].Score > c[j].Score })
}

// ---- Built-in ciphers ----

// XOR repeats key over the data (single-byte key = single-byte XOR).
type XOR struct{}

func (XOR) Decrypt(key, ct []byte) []byte {
	if len(key) == 0 {
		return append([]byte(nil), ct...)
	}
	out := make([]byte, len(ct))
	for i := range ct {
		out[i] = ct[i] ^ key[i%len(key)]
	}
	return out
}

// Caesar subtracts key[0] from each byte (mod 256).
type Caesar struct{}

func (Caesar) Decrypt(key, ct []byte) []byte {
	var k byte
	if len(key) > 0 {
		k = key[0]
	}
	out := make([]byte, len(ct))
	for i := range ct {
		out[i] = ct[i] - k
	}
	return out
}

// ---- Built-in keyspace ----

// ByteKeySpace enumerates all keys of a fixed byte length as a big-endian
// odometer. Practical for 1–3 bytes; wider keys should use the smart
// per-column solvers below.
type ByteKeySpace struct {
	Len int
	pos uint64
	cur []byte
}

func (k *ByteKeySpace) Next() ([]byte, bool) {
	if k.cur == nil {
		k.cur = make([]byte, k.Len)
	}
	if k.Size() != 0 && k.pos >= k.Size() {
		return nil, false
	}
	// Render pos as the key (big-endian).
	v := k.pos
	for i := k.Len - 1; i >= 0; i-- {
		k.cur[i] = byte(v)
		v >>= 8
	}
	k.pos++
	return k.cur, true
}

func (k *ByteKeySpace) Position() uint64 { return k.pos }
func (k *ByteKeySpace) Seek(p uint64)    { k.pos = p }
func (k *ByteKeySpace) Size() uint64 {
	if k.Len <= 0 || k.Len > 7 {
		return 0
	}
	return uint64(1) << uint(8*k.Len)
}

// ---- Smart XOR solvers ----

// SolveSingleByteXOR finds the single-byte XOR key best explaining ct.
func SolveSingleByteXOR(ct []byte, sc Scorer) Candidate {
	res := Run(context.Background(), XOR{}, &ByteKeySpace{Len: 1}, sc, ct, Options{TopN: 1})
	if len(res) == 0 {
		return Candidate{}
	}
	return res[0]
}

// SolveRepeatingXOR recovers a repeating-key XOR key of the given length by
// solving each key column independently as a single-byte XOR — the standard
// decomposition that turns 256^keyLen into keyLen×256.
func SolveRepeatingXOR(ct []byte, keyLen int, sc Scorer) Candidate {
	if keyLen <= 0 {
		return Candidate{}
	}
	key := make([]byte, keyLen)
	for col := 0; col < keyLen; col++ {
		var column []byte
		for i := col; i < len(ct); i += keyLen {
			column = append(column, ct[i])
		}
		best := SolveSingleByteXOR(column, sc)
		if len(best.Key) == 1 {
			key[col] = best.Key[0]
		}
	}
	pt := XOR{}.Decrypt(key, ct)
	return Candidate{Key: key, Plaintext: pt, Score: sc.Score(pt)}
}
