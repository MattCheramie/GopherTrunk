package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/p25"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25p1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/voice"
	"github.com/MattCheramie/GopherTrunk/internal/voice/imbe"
)

// TestP25ADPReplay is the issue #1187 known-key harness for P25 Phase 1
// ADP (RC4, ALGID 0xAA): it replays a capture of an encrypted call through
// the production Phase 1 receiver and LDU assembler, decodes every LDU2's
// Encryption Sync, applies the in-process ADP descramble (phase1.ADP*) under
// the production MI schedule — the ES names the NEXT superframe's MI, the
// first superframe's is RewindMI of the first ES — and renders the result
// through the IMBE vocoder, so a reporter's "ADP audio with a known key"
// either decodes to voice or shows exactly which stage did not. Skipped
// unless an input is given.
//
// Inputs (one of):
//
//	GT_P25_ADP_IQ=<file>      IQ capture: cs16 (default), f32, or a wav/flac
//	                          container (sniffed; carries its own rate)
//	GT_P25_ADP_AUDIO=<file>   16-bit mono discriminator-audio WAV/FLAC (the
//	                          DSD-FME input style), FM re-modulated to IQ at
//	                          the file's rate; GT_P25_ADP_AUDIO_DEV is the
//	                          full-scale deviation in Hz (default 4800 — the
//	                          receiver's AGC makes the exact value uncritical)
//
// Knobs: GT_P25_ADP_RATE / GT_P25_ADP_FORMAT (raw IQ), GT_P25_ADP_KEY=<hex>
// (the 40-bit operator key; omit for an instrument-only run that reports the
// ES chain without descrambling), GT_P25_ADP_KEYID=<n> (apply the key only
// when the ES names this key id; default any), GT_P25_ADP_OUT=<wav> (write
// the decoded 8 kHz audio), GT_P25_ADP_ALLOW_EMPTY=1.
//
// Reproduce (the #1187 reporter's file):
//
//	GT_P25_ADP_AUDIO=P25_Enc_ADP_knownkey_1234567890.wav GT_P25_ADP_KEY=1234567890 \
//	  go test ./cmd/gophertrunk -run TestP25ADPReplay -v
func TestP25ADPReplay(t *testing.T) {
	iqPath := os.Getenv("GT_P25_ADP_IQ")
	audioPath := os.Getenv("GT_P25_ADP_AUDIO")
	if iqPath == "" && audioPath == "" {
		t.Skip("set GT_P25_ADP_IQ (IQ capture) or GT_P25_ADP_AUDIO (discriminator audio) to run the P25 ADP replay")
	}
	var key []byte
	if k := os.Getenv("GT_P25_ADP_KEY"); k != "" {
		b, err := hex.DecodeString(strings.TrimPrefix(strings.ReplaceAll(k, " ", ""), "0x"))
		if err != nil || len(b) != phase1.ADPKeyBytes {
			t.Fatalf("GT_P25_ADP_KEY must be %d bytes of hex: %v", phase1.ADPKeyBytes, err)
		}
		key = b
	}
	wantKeyID := -1
	if v := os.Getenv("GT_P25_ADP_KEYID"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			t.Fatalf("GT_P25_ADP_KEYID: %v", err)
		}
		wantKeyID = n
	}

	var (
		iq     []complex64
		inRate = 48000.0
	)
	if v := os.Getenv("GT_P25_ADP_RATE"); v != "" {
		r, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatalf("GT_P25_ADP_RATE: %v", err)
		}
		inRate = r
	}
	var audioSanity string
	switch {
	case audioPath != "":
		samples, rate, err := voice.ReadAudioSamples(audioPath)
		if err != nil {
			t.Fatalf("read discriminator audio: %v (16-bit mono WAV/FLAC expected)", err)
		}
		dev := 4800.0
		if v := os.Getenv("GT_P25_ADP_AUDIO_DEV"); v != "" {
			dev, _ = strconv.ParseFloat(v, 64)
		}
		inRate = float64(rate)
		iq = remodulateDiscriminatorAudio(samples, inRate, dev)
		audioSanity = discriminatorAudioSanity(samples, inRate, "C4FM")
		t.Logf("discriminator audio: %d samples at %d Hz, re-modulated at ±%.0f Hz full scale; %s", len(samples), rate, dev, audioSanity)
	default:
		raw, err := os.ReadFile(iqPath)
		if err != nil {
			t.Fatal(err)
		}
		format := strings.ToLower(os.Getenv("GT_P25_ADP_FORMAT"))
		if format == "" {
			format = "cs16"
		}
		if _, isContainer := siglab.SniffContainer(raw); isContainer {
			samples, rate, err := siglab.DecodeContainerFile(iqPath)
			if err != nil {
				t.Fatal(err)
			}
			iq = samples
			if rate > 0 && os.Getenv("GT_P25_ADP_RATE") == "" {
				inRate = float64(rate)
			}
		} else {
			switch format {
			case "cs16", "sc16":
				iq = make([]complex64, len(raw)/4)
				for i := range iq {
					re := int16(binary.LittleEndian.Uint16(raw[i*4:]))
					im := int16(binary.LittleEndian.Uint16(raw[i*4+2:]))
					iq[i] = complex(float32(re)/32768, float32(im)/32768)
				}
			case "f32", "cf32", "fc32":
				iq = make([]complex64, len(raw)/8)
				for i := range iq {
					re := math.Float32frombits(binary.LittleEndian.Uint32(raw[i*8:]))
					im := math.Float32frombits(binary.LittleEndian.Uint32(raw[i*8+4:]))
					iq[i] = complex(re, im)
				}
			default:
				t.Fatalf("unknown GT_P25_ADP_FORMAT %q (want cs16 or f32)", format)
			}
		}
	}

	type lduRec struct {
		duid   phase1.DUID
		es     phase1.EncryptionSync
		esOK   bool
		esErr  error
		frames [phase1.LDUVoiceSubframeCount][]byte
		errs   [phase1.LDUVoiceSubframeCount]int
		bad    bool
	}
	var ldus []lduRec
	ddc := ccdecoder.NewDownconverter(inRate, 48000)
	rx := p25p1rx.New(p25p1rx.Options{
		SampleRateHz: ddc.OutRateHz(),
		DeviationHz:  1800,
		Sink: func(ldu []byte) {
			duid, derr := phase1.LDUDuid(ldu)
			if derr != nil {
				return
			}
			r := lduRec{duid: duid}
			if duid == phase1.DUIDLogicalLink1 || duid == phase1.DUIDLogicalLink2 {
				if duid == phase1.DUIDLogicalLink2 {
					if b, berr := phase1.ExtractLCESBlocks(ldu); berr == nil {
						r.es, _, r.esErr = phase1.ParseEncryptionSync(b)
						r.esOK = r.esErr == nil
					}
				}
				var ferr error
				r.frames, r.errs, _, ferr = phase1.ExtractVoiceFramesDetailed(ldu)
				r.bad = ferr != nil
			}
			ldus = append(ldus, r)
		},
	})
	const chunk = 65536
	var scratch []complex64
	for i := 0; i < len(iq); i += chunk {
		e := i + chunk
		if e > len(iq) {
			e = len(iq)
		}
		scratch = ddc.Process(scratch[:0], iq[i:e])
		rx.Process(scratch)
	}

	// Walk the LDUs exactly as the composer's p25ADPState does: the ES
	// names the next superframe's MI, the first ES is rewound for the
	// superframe it belongs to, and an LDU2 with no decodable ES advances
	// the chain by the LFSR prediction.
	var (
		cur, next          [9]byte
		haveCur, haveNext  bool
		activeKey          []byte
		keyApplied         bool
		ldu1, ldu2, others int
		badLDUs, esFails   int
		esLines            []string
		descrambled        int
		mismatches         int
		rawFrames, written [][]byte
		firstAlg           uint8
		firstKID           uint16
		haveAlg            bool
	)
	var held [][]byte // frames of the superframe before the first ES
	flushHeld := func() {
		if held == nil {
			return
		}
		written = append(written, held...)
		held = nil
	}
	for i, l := range ldus {
		if l.duid != phase1.DUIDLogicalLink1 && l.duid != phase1.DUIDLogicalLink2 {
			others++
			esLines = append(esLines, fmt.Sprintf("  #%02d %v", i, l.duid))
			continue
		}
		if l.bad {
			badLDUs++
		}
		for _, f := range l.frames {
			if f != nil {
				rawFrames = append(rawFrames, append([]byte(nil), f...))
			}
		}
		if l.duid == phase1.DUIDLogicalLink1 {
			ldu1++
		} else {
			ldu2++
			if l.esOK && l.es.Encrypted() {
				if !haveAlg {
					firstAlg, firstKID, haveAlg = l.es.AlgorithmID, l.es.KeyID, true
				}
				esLines = append(esLines, fmt.Sprintf("  #%02d LDU2 ES alg=%s(0x%02X) key_id=%d mi=%s", i,
					p25.FormatAlgorithm(l.es.AlgorithmID), l.es.AlgorithmID, l.es.KeyID, hex.EncodeToString(l.es.MessageIndicator[:])))
				if l.es.AlgorithmID == p25.AlgorithmADP {
					if !haveCur {
						cur, haveCur = phase1.RewindMI(l.es.MessageIndicator), true
					} else if haveNext && next != l.es.MessageIndicator {
						mismatches++
					}
					next, haveNext = l.es.MessageIndicator, true
					if key != nil {
						keyApplied = wantKeyID < 0 || int(l.es.KeyID) == wantKeyID
						if keyApplied {
							activeKey = key
						} else {
							activeKey = nil
						}
					}
				}
			} else {
				if !l.esOK {
					esFails++
				}
				esLines = append(esLines, fmt.Sprintf("  #%02d LDU2 ES not usable (ok=%v alg=0x%02X err=%v)", i, l.esOK, l.es.AlgorithmID, l.esErr))
			}
		}
		frames := l.frames
		if activeKey != nil && haveCur {
			ks, err := phase1.ADPSuperframeKeystream(activeKey, cur)
			if err != nil {
				t.Fatal(err)
			}
			// Frames held before the first ES belong to this superframe too.
			if held != nil {
				var hf [phase1.LDUVoiceSubframeCount][]byte
				copy(hf[:], held)
				if n, err := phase1.ADPDescrambleVoiceFrames(ks, phase1.DUIDLogicalLink1, &hf); err == nil {
					descrambled += n
				}
			}
			n, err := phase1.ADPDescrambleVoiceFrames(ks, l.duid, &frames)
			if err != nil {
				t.Fatal(err)
			}
			descrambled += n
		}
		if !haveCur && l.duid == phase1.DUIDLogicalLink1 && key != nil {
			held = append(held[:0], frames[:]...)
		} else {
			flushHeld()
			written = append(written, frames[:]...)
		}
		if l.duid == phase1.DUIDLogicalLink2 && haveCur {
			if haveNext {
				cur = next
			} else {
				cur = phase1.AdvanceMI(cur)
			}
			haveNext = false
		}
	}
	flushHeld()

	// Render + verdict. Pitch continuity (the IMBE fundamental b0 between
	// consecutive frames) separates speech (≳ 0.5) from ciphertext (≈ 0.1);
	// output loudness cannot — random vocoder parameters are LOUD.
	dec := imbe.New()
	var pcm []int16
	for _, f := range written {
		if f == nil {
			continue
		}
		if out, err := dec.Decode(f); err == nil {
			pcm = append(pcm, out...)
		}
	}
	rawC, rawPairs := p25PitchContinuity(rawFrames)
	outC, outPairs := p25PitchContinuity(written)
	t.Logf("in=%.0fHz out=%.0fHz samples=%d dur=%.1fs ldus=%d (ldu1=%d ldu2=%d other=%d fec_bad=%d es_fail=%d) key=%v key_id_filter=%d",
		inRate, ddc.OutRateHz(), len(iq), float64(len(iq))/inRate, len(ldus), ldu1, ldu2, others, badLDUs, esFails, key != nil, wantKeyID)
	for _, l := range esLines {
		t.Log(l)
	}
	t.Logf("descrambled_frames=%d mi_mismatches=%d raw_pitch_continuity=%.2f (%d pairs) decoded_pitch_continuity=%.2f (%d pairs; speech ≳0.5, ciphertext ≈0.15) pcm_seconds=%.1f",
		descrambled, mismatches, rawC, rawPairs, outC, outPairs, float64(len(pcm))/8000)
	if out := os.Getenv("GT_P25_ADP_OUT"); out != "" && len(pcm) > 0 {
		w, err := voice.NewWavFile(out, 8000)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.WriteSamples(pcm); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", out)
	}

	switch {
	case ldu1+ldu2 == 0:
		msg := fmt.Sprintf("no LDUs decoded — check the rate (%v), tuning, or that this is a P25 Phase 1 capture", inRate)
		if audioSanity != "" {
			msg += "; " + audioSanity
		}
		if os.Getenv("GT_P25_ADP_ALLOW_EMPTY") == "1" {
			t.Logf("WARNING: %s", msg)
		} else {
			t.Fatalf("%s", msg)
		}
	case !haveAlg:
		t.Logf("VERDICT: no encrypted Encryption Sync decoded — the call is clear (or every ES failed RS); nothing to descramble")
	case key == nil:
		t.Logf("VERDICT: instrument-only run (no GT_P25_ADP_KEY) — alg=%s key_id=%d, %d LDU2s; re-run with the key to descramble",
			p25.FormatAlgorithm(firstAlg), firstKID, ldu2)
	case firstAlg != p25.AlgorithmADP:
		t.Logf("VERDICT: key NOT applied — the call uses %s, and only ADP (0xAA) is decrypted in-process", p25.FormatAlgorithm(firstAlg))
	case !keyApplied:
		t.Logf("VERDICT: key NOT applied — the ES names key id %d, not GT_P25_ADP_KEYID=%d", firstKID, wantKeyID)
	case descrambled == 0:
		t.Logf("VERDICT: nothing descrambled — no Message Indicator was established (every ES failed?)")
	case outC >= pitchContinuityVoice:
		t.Logf("VERDICT: PASS — %d frames descrambled and the decoded frames are speech-like (pitch continuity %.2f vs %.2f raw); listen to the WAV to confirm",
			descrambled, outC, rawC)
	default:
		t.Logf("VERDICT: FAIL — %d frames descrambled but the result is still ciphertext-like (pitch continuity %.2f, raw %.2f): wrong key, wrong key id, or a keystream layout this capture does not use",
			descrambled, outC, rawC)
	}
}

// p25PitchContinuity is the IMBE flavour of the DMR harness's verdict:
// the fraction of consecutive frame pairs whose fundamental parameter b0
// (info bits 0..5 + 85, 86) moves by at most 10 steps. Silence-window and
// out-of-range b0 values break the chain.
func p25PitchContinuity(frames [][]byte) (float64, int) {
	var pairs, close int
	prev := -1
	for _, f := range frames {
		if len(f) != imbe.FrameBytes {
			prev = -1
			continue
		}
		b := int(f[0]>>2)<<2 | int((f[10]>>2)&1)<<1 | int((f[10]>>1)&1)
		if b > 207 {
			prev = -1
			continue
		}
		if prev >= 0 {
			pairs++
			if d := b - prev; d >= -10 && d <= 10 {
				close++
			}
		}
		prev = b
	}
	if pairs == 0 {
		return 0, 0
	}
	return float64(close) / float64(pairs), pairs
}
