package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/MattCheramie/GopherTrunk/internal/gtbundle"
	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25"
)

// cryptoFrame is the minimal shape of a cryptolab-frames record we need to fold
// a confirmed encryption observation back onto the mapping: which talkgroup was
// carrying encrypted traffic under which algorithm. It mirrors the fields the
// live decoder→cryptolab bridge writes (internal/voice/cryptocap); every other
// field is ignored.
type cryptoFrame struct {
	TG    uint32 `json:"tg"`
	AlgID uint8  `json:"algid"`
}

// enrichFromCryptoVerdict folds the bundle's crypto observations onto the
// discovered system before it is committed to config.yaml: any talkgroup seen
// carrying traffic under a non-clear algorithm is marked Encrypted. It reads the
// cryptolab-frames artifact (tg + algid per encrypted frame); the ALGID → name
// mapping is via internal/radio/p25. It is a no-op when the bundle carries no
// crypto frames. Returns the number of talkgroups newly marked encrypted.
//
// This deliberately reads the frames rather than the CryptoLab Result prose, so
// `bundle commit` stays in the always-compiled tree (no -tags cryptolab needed).
func enrichFromCryptoVerdict(r *gtbundle.Reader, sys *hunt.DiscoveredSystem) int {
	body, err := r.CryptolabFramesRaw()
	if err != nil {
		if errors.Is(err, gtbundle.ErrNotFound) {
			return 0
		}
		fmt.Fprintf(os.Stderr, "bundle: crypto frames unreadable (%v); skipping enrichment\n", err)
		return 0
	}

	// algByTG records the strongest (non-clear) algorithm seen per talkgroup.
	algByTG := map[uint32]uint8{}
	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		var fr cryptoFrame
		if json.Unmarshal(line, &fr) != nil {
			continue
		}
		if fr.TG == 0 || isClearAlg(fr.AlgID) {
			continue
		}
		algByTG[fr.TG] = fr.AlgID
	}

	marked := 0
	for i := range sys.Talkgroups {
		tg := &sys.Talkgroups[i]
		alg, ok := algByTG[tg.Dec]
		if !ok {
			continue
		}
		if !tg.Encrypted {
			tg.Encrypted = true
			marked++
		}
		fmt.Fprintf(os.Stderr, "bundle: talkgroup %d confirmed encrypted (%s)\n",
			tg.Dec, p25.FormatAlgorithm(alg))
	}
	return marked
}

// isClearAlg reports whether an ALGID means "no encryption". 0x80 is the P25
// CLEAR algorithm id; 0 is an absent/unknown id in the frame.
func isClearAlg(id uint8) bool { return id == 0x00 || id == 0x80 }
