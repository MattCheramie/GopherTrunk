// Package keystream implements keystream-reuse / many-time-pad analysis — the
// real-world break against additive stream ciphers (P25 DES-OFB and ADP/RC4,
// TETRA AIE) when a transmitter reuses an initialisation vector. Under OFB and
// raw stream ciphers the keystream is a deterministic function of (key, IV);
// two frames sent under the same key and the same IV therefore share an
// identical keystream, and XORing their ciphertexts cancels it:
//
//	c1 ^ c2 = (p1 ^ ks) ^ (p2 ^ ks) = p1 ^ p2
//
// leaving a classic running-key system that crib-dragging and a language
// model can peel apart — no key recovery required. For P25 the IV is the
// 72-bit Message Indicator the decoder already extracts, so this is the bridge
// from "encrypted, opaque" to "recoverable" whenever a radio's MI repeats.
//
// This package only exploits operator misuse (IV reuse) and known plaintext;
// it does not attack the cipher's key, in keeping with the toolkit's scope.
package keystream

import (
	"encoding/hex"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/lang"
)

// Frame is one captured encrypted unit: an IV (e.g. a P25 Message Indicator)
// and the ciphertext under that IV. Label identifies the source (call id,
// timestamp) for reporting.
type Frame struct {
	Label string
	IV    []byte
	CT    []byte
}

// ReuseGroup is a set of frames that share an identical IV and therefore an
// identical keystream.
type ReuseGroup struct {
	IV     []byte
	Frames []Frame
}

// XOR returns a^b over the common prefix (min length). It is the additive
// stream-cipher primitive: keystream = pt^ct, and ct1^ct2 = pt1^pt2.
func XOR(a, b []byte) []byte {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = a[i] ^ b[i]
	}
	return out
}

// FindReuse groups frames sharing an identical IV, returning only groups with
// at least two frames (the exploitable ones), largest group first. Frames
// with an empty IV are ignored — without an IV there is nothing to collide.
func FindReuse(frames []Frame) []ReuseGroup {
	byIV := make(map[string][]Frame)
	var order []string
	for _, f := range frames {
		if len(f.IV) == 0 {
			continue
		}
		k := string(f.IV)
		if _, seen := byIV[k]; !seen {
			order = append(order, k)
		}
		byIV[k] = append(byIV[k], f)
	}
	var out []ReuseGroup
	for _, k := range order {
		if len(byIV[k]) >= 2 {
			out = append(out, ReuseGroup{IV: []byte(k), Frames: byIV[k]})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if len(out[i].Frames) != len(out[j].Frames) {
			return len(out[i].Frames) > len(out[j].Frames)
		}
		return hex.EncodeToString(out[i].IV) < hex.EncodeToString(out[j].IV)
	})
	return out
}

// Decoded is one frame recovered by RecoverWithKnown.
type Decoded struct {
	Label     string
	Plaintext []byte
}

// RecoverWithKnown turns one known plaintext within a reuse group into the
// group's keystream (ks = knownPT ^ knownCT) and uses it to decrypt every
// other frame in the group. This is the decisive step once any single frame's
// content is known (a standard tone, a known data payload, a confirmed
// transcript). It returns the recovered keystream and the decrypted frames.
func RecoverWithKnown(group ReuseGroup, knownLabel string, knownPT []byte) (ks []byte, out []Decoded, ok bool) {
	var known *Frame
	for i := range group.Frames {
		if group.Frames[i].Label == knownLabel {
			known = &group.Frames[i]
			break
		}
	}
	if known == nil {
		return nil, nil, false
	}
	ks = XOR(knownPT, known.CT)
	for _, f := range group.Frames {
		out = append(out, Decoded{Label: f.Label, Plaintext: XOR(f.CT, ks)})
	}
	return ks, out, true
}

// DragHit is one crib-drag placement: positioning crib at Offset in one
// ciphertext reveals Revealed in the partner, scored by a language model.
type DragHit struct {
	CribLabel    string
	PartnerLabel string
	Offset       int
	Revealed     []byte
	Score        float64
}

// CribDragGroup slides crib across every ordered ciphertext pair in the group
// (c_i ^ c_j ^ crib) and returns the placements whose revealed partner bytes
// are fully printable, ranked by English trigram score — the highest are the
// likely true positions. With the right crib this peels plaintext out of a
// reuse group with no key and no known plaintext.
func CribDragGroup(group ReuseGroup, crib []byte, topN int) []DragHit {
	if len(crib) == 0 {
		return nil
	}
	model := lang.EnglishTrigram()
	var hits []DragHit
	for i := range group.Frames {
		for j := range group.Frames {
			if i == j {
				continue
			}
			a, b := group.Frames[i].CT, group.Frames[j].CT
			n := len(a)
			if len(b) < n {
				n = len(b)
			}
			for off := 0; off+len(crib) <= n; off++ {
				rev := make([]byte, len(crib))
				printable := true
				for k := range crib {
					rev[k] = a[off+k] ^ b[off+k] ^ crib[k]
					if rev[k] < 0x20 || rev[k] > 0x7e {
						printable = false
						break
					}
				}
				if !printable {
					continue
				}
				hits = append(hits, DragHit{
					CribLabel:    group.Frames[i].Label,
					PartnerLabel: group.Frames[j].Label,
					Offset:       off,
					Revealed:     rev,
					Score:        model.Score(rev),
				})
			}
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if topN > 0 && len(hits) > topN {
		hits = hits[:topN]
	}
	return hits
}
