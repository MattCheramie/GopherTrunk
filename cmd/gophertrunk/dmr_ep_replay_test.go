package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/cmplx"
	"os"
	"strconv"
	"strings"
	"testing"

	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/voice"
	"github.com/MattCheramie/GopherTrunk/internal/voice/ambe2"
)

// TestDMREnhancedPrivacyReplay is the issue #1187 known-key harness: it
// replays a capture of a DMR "Enhanced Privacy" (DMRA RC4) transmission
// through the production DMR receiver, the PI header detector, the voice
// superframe decoder and the Enhanced Privacy descramble, then renders the
// result through the ambe2-dmr vocoder — so a reporter's "RC4 audio with a
// known key" either decodes to voice or shows exactly which stage did not.
// Skipped unless an input is given.
//
// Inputs (one of):
//
//	GT_DMR_EP_IQ=<file>      IQ capture: cs16 (default), f32, or a wav/flac
//	                         container (sniffed; carries its own rate)
//	GT_DMR_EP_AUDIO=<file>   16-bit mono discriminator-audio WAV/FLAC (the
//	                         DSD-FME input style), FM re-modulated to IQ at
//	                         the file's rate; GT_DMR_EP_AUDIO_DEV is the
//	                         full-scale deviation in Hz (default 4800)
//
// Knobs: GT_DMR_EP_RATE / GT_DMR_EP_FORMAT (raw IQ), GT_DMR_EP_KEY=<hex> (the
// operator key; omit for an instrument-only run that reports PI headers and
// MIs without descrambling), GT_DMR_EP_KEYID=<n> (apply the key only when
// the PI header names this id; default any), GT_DMR_EP_MI=<8 hex> (seed the
// MI chain by hand when neither a PI header nor an embedded IV decodes),
// GT_DMR_EP_INTERLEAVED=1 (2-slot repeater carrier), GT_DMR_EP_OUT=<wav>
// (write the decoded 8 kHz audio), GT_DMR_EP_ALLOW_EMPTY=1.
//
// Reproduce: GT_DMR_EP_IQ=<capture> GT_DMR_EP_KEY=0123456789 \
//
//	go test ./cmd/gophertrunk -run TestDMREnhancedPrivacyReplay -v
func TestDMREnhancedPrivacyReplay(t *testing.T) {
	iqPath := os.Getenv("GT_DMR_EP_IQ")
	audioPath := os.Getenv("GT_DMR_EP_AUDIO")
	if iqPath == "" && audioPath == "" {
		t.Skip("set GT_DMR_EP_IQ (IQ capture) or GT_DMR_EP_AUDIO (discriminator audio) to run the enhanced-privacy replay")
	}
	var key []byte
	if k := os.Getenv("GT_DMR_EP_KEY"); k != "" {
		b, err := hex.DecodeString(strings.TrimPrefix(strings.ReplaceAll(k, " ", ""), "0x"))
		if err != nil || len(b) == 0 {
			t.Fatalf("GT_DMR_EP_KEY must be hex: %v", err)
		}
		key = b
	}
	wantKeyID := -1
	if v := os.Getenv("GT_DMR_EP_KEYID"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			t.Fatalf("GT_DMR_EP_KEYID: %v", err)
		}
		wantKeyID = n
	}
	var seedMI uint32
	haveSeedMI := false
	if v := os.Getenv("GT_DMR_EP_MI"); v != "" {
		n, err := strconv.ParseUint(strings.TrimPrefix(v, "0x"), 16, 32)
		if err != nil {
			t.Fatalf("GT_DMR_EP_MI must be 8 hex digits: %v", err)
		}
		seedMI, haveSeedMI = uint32(n), true
	}
	interleaved := os.Getenv("GT_DMR_EP_INTERLEAVED") == "1"

	var (
		iq          []complex64
		inRate      = 48000.0
		audioSanity string
	)
	if v := os.Getenv("GT_DMR_EP_RATE"); v != "" {
		r, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatalf("GT_DMR_EP_RATE: %v", err)
		}
		inRate = r
	}
	switch {
	case audioPath != "":
		samples, rate, err := voice.ReadAudioSamples(audioPath)
		if err != nil {
			t.Fatalf("read discriminator audio: %v (16-bit mono WAV/FLAC expected; convert with `sox in.wav -c 1 -b 16 out.wav`)", err)
		}
		dev := 4800.0
		if v := os.Getenv("GT_DMR_EP_AUDIO_DEV"); v != "" {
			dev, _ = strconv.ParseFloat(v, 64)
		}
		inRate = float64(rate)
		iq = remodulateDiscriminatorAudio(samples, inRate, dev)
		audioSanity = discriminatorAudioSanity(samples, inRate, "4FSK")
		t.Logf("discriminator audio: %d samples at %d Hz, re-modulated at ±%.0f Hz full scale; %s", len(samples), rate, dev, audioSanity)
	default:
		raw, err := os.ReadFile(iqPath)
		if err != nil {
			t.Fatal(err)
		}
		format := strings.ToLower(os.Getenv("GT_DMR_EP_FORMAT"))
		if format == "" {
			format = "cs16"
		}
		if _, isContainer := siglab.SniffContainer(raw); isContainer {
			samples, rate, err := siglab.DecodeContainerFile(iqPath)
			if err != nil {
				t.Fatal(err)
			}
			iq = samples
			if rate > 0 && os.Getenv("GT_DMR_EP_RATE") == "" {
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
				t.Fatalf("unknown GT_DMR_EP_FORMAT %q (want cs16 or f32)", format)
			}
		}
	}

	ddc := ccdecoder.NewDownconverter(inRate, 48000)
	outRate := ddc.OutRateHz()
	var allDibits []uint8
	rx := dmrrx.New(dmrrx.Options{
		SampleRateHz: outRate,
		DeviationHz:  1944.0,
		ClockGain:    0.015,
		DibitSink: func(dibits []uint8, _ int) {
			allDibits = append(allDibits, dibits...)
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

	piDet := dmrvoice.NewPIHeaderDetector()
	termDet := dmrvoice.NewTerminatorDetector()
	voiceDec := dmrvoice.NewDecoder()
	if interleaved {
		voiceDec = dmrvoice.NewInterleavedDecoder()
	}
	var tracker dmrvoice.EPTracker
	if haveSeedMI {
		tracker.SetHeaderMI(seedMI)
	}
	vocoder := ambe2.NewDMR()
	var (
		headers      []string
		activeKey    []byte
		superframes  int
		ivVerified   int
		descrambled  int
		silence      int
		noMI         int
		ambeOK       int
		ambeBad      int
		terminators  int
		pcm          []int16
		lastHeaderMI string
		keyApplied   bool
		miTimeline   []string
		written      [][]byte // the 49-bit payloads as the recorder would get them
	)
	activeKey = key // GT_DMR_EP_KEYID gates it per header below
	feed := func(dibits []uint8, baseIdx int) {
		for _, h := range piDet.Process(dibits, baseIdx) {
			line := fmt.Sprintf("%8.2fs PI header alg=%s(0x%02X) fid=0x%02X key_id=%d mi=%s dst=%d raw=%s",
				float64(baseIdx)/4800, h.AlgName(), h.AlgID, h.FID, h.KeyID, hex.EncodeToString(h.MI[:]), h.DstAddr, hex.EncodeToString(h.Raw[:]))
			headers = append(headers, line)
			tracker.SetHeaderMI(h.MI32())
			lastHeaderMI = hex.EncodeToString(h.MI[:])
			if key != nil {
				keyApplied = h.IsRC4() && (wantKeyID < 0 || int(h.KeyID) == wantKeyID)
				if keyApplied {
					activeKey = key
				} else {
					activeKey = nil
				}
			}
		}
		terminators += len(termDet.Process(dibits, baseIdx))
		for _, sf := range voiceDec.Process(dibits, baseIdx) {
			superframes++
			iv, corrected, ivOK := dmrvoice.ExtractEmbeddedIV(sf.Frames)
			if ivOK {
				ivVerified++
			}
			mi, ok := tracker.Next(iv, ivOK, corrected)
			if len(miTimeline) < 40 {
				miTimeline = append(miTimeline, fmt.Sprintf("%8.2fs sf#%d embedded_iv=%08X ok=%v corrected=%d -> mi=%08X known=%v",
					float64(sf.StartDibit)/4800, superframes, iv, ivOK, corrected, mi, ok))
			}
			infos := make([][]byte, dmrvoice.FramesPerSuperframe)
			for i := range sf.Frames {
				info, _, err := dmrvoice.DecodeAMBEFrame(sf.Frames[i])
				if err != nil {
					ambeBad++
					continue
				}
				ambeOK++
				infos[i] = info
			}
			if !ok {
				noMI++
			} else if activeKey != nil {
				n, sil, err := dmrvoice.DescrambleSuperframe(activeKey, mi, infos)
				if err != nil {
					t.Fatalf("descramble: %v", err)
				}
				descrambled += n
				silence += sil
			}
			for _, info := range infos {
				if info == nil {
					continue
				}
				written = append(written, info)
				packed := make([]byte, 7)
				for b := range info {
					if info[b]&1 != 0 {
						packed[b>>3] |= 1 << uint(7-(b&7))
					}
				}
				out, err := vocoder.Decode(packed)
				if err == nil {
					pcm = append(pcm, out...)
				}
			}
		}
	}
	const dibitChunk = 4096
	for i := 0; i < len(allDibits); i += dibitChunk {
		e := i + dibitChunk
		if e > len(allDibits) {
			e = len(allDibits)
		}
		feed(allDibits[i:e], i)
	}

	// Pitch continuity is the verdict's discriminator: the AMBE+2 fundamental
	// index (b0, the payload's first 7 bits) of real speech moves slowly
	// between consecutive 20 ms frames, while ciphertext (or the wrong key)
	// gives a uniformly random b0 every frame — output loudness cannot tell
	// the two apart (random vocoder parameters are LOUD), so RMS alone is
	// reported but never trusted.
	continuity, voicedPairs := pitchContinuity(written)
	const pcmRate = 8000
	var active []int
	for s := 0; s*pcmRate < len(pcm); s++ {
		seg := pcm[s*pcmRate:]
		if len(seg) > pcmRate {
			seg = seg[:pcmRate]
		}
		var acc float64
		for _, v := range seg {
			acc += float64(v) * float64(v)
		}
		if math.Sqrt(acc/float64(len(seg))) > 500 {
			active = append(active, s)
		}
	}

	rejects, rejectHex := piDet.Rejects()
	t.Logf("in=%.0fHz out=%.0fHz samples=%d dur=%.1fs dibits=%d interleaved=%v key=%v key_id_filter=%d",
		inRate, outRate, len(iq), float64(len(iq))/inRate, len(allDibits), interleaved, key != nil, wantKeyID)
	t.Logf("pi_headers=%d pi_crc_rejects=%d last_reject=%s terminators=%d", len(headers), rejects, rejectHex, terminators)
	for _, h := range headers {
		t.Logf("  %s", h)
	}
	t.Logf("superframes=%d iv_verified=%d mi_predicted=%d mi_mismatches=%d no_mi=%d ambe_ok=%d ambe_uncorrectable=%d",
		superframes, ivVerified, tracker.Predicted, tracker.Mismatches, noMI, ambeOK, ambeBad)
	for _, l := range miTimeline {
		t.Logf("  %s", l)
	}
	t.Logf("descrambled_frames=%d silence_frames=%d pitch_continuity=%.2f (over %d frame pairs; speech ≳0.5, ciphertext ≈0.15) pcm_seconds=%.1f loud_seconds=%v",
		descrambled, silence, continuity, voicedPairs, float64(len(pcm))/pcmRate, active)
	if out := os.Getenv("GT_DMR_EP_OUT"); out != "" && len(pcm) > 0 {
		w, err := voice.NewWavFile(out, pcmRate)
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
	case superframes == 0 && len(headers) == 0:
		msg := fmt.Sprintf("no voice superframes and no PI headers decoded — check the rate (%v), tuning, or that this is a DMR capture", inRate)
		if audioSanity != "" {
			msg += "; " + audioSanity
		}
		if os.Getenv("GT_DMR_EP_ALLOW_EMPTY") == "1" {
			t.Logf("WARNING: %s", msg)
		} else {
			t.Fatalf("%s", msg)
		}
	case key == nil:
		t.Logf("VERDICT: instrument-only run (no GT_DMR_EP_KEY) — %d PI header(s), last MI %s, pitch continuity %.2f on the recorded frames (%s); re-run with the key to descramble",
			len(headers), lastHeaderMI, continuity, continuityLabel(continuity))
	case !keyApplied && len(headers) > 0:
		t.Logf("VERDICT: key NOT applied — the PI header names a different key id / algorithm than GT_DMR_EP_KEYID (see the header lines above)")
	case descrambled == 0:
		t.Logf("VERDICT: nothing descrambled — no Message Indicator was ever known (no PI header, no embedded IV); seed one with GT_DMR_EP_MI=<8 hex> if the header is in a part of the capture not present here")
	case continuity < pitchContinuityVoice:
		t.Logf("VERDICT: descrambled %d frames but the result is still ciphertext-like (pitch continuity %.2f) — the key, key id, or the keystream construction does not match this radio; the PI header / MI lines above are the evidence to file", descrambled, continuity)
	default:
		t.Logf("VERDICT: voice decoded — %d descrambled frames, pitch continuity %.2f, loud seconds %v (listen to GT_DMR_EP_OUT to confirm intelligibility)", descrambled, continuity, active)
	}
}

// pitchContinuityVoice is the b0-continuity fraction above which decoded
// frames read as speech. Random parameters land near 21/128 ≈ 0.16 (the
// chance of |Δb0| ≤ 10 on a uniform 7-bit index); speech sits well above
// 0.5 because the fundamental moves slowly within a syllable.
const pitchContinuityVoice = 0.45

// pitchContinuity returns the fraction of consecutive non-silence frame
// pairs whose AMBE+2 fundamental index b0 (payload bits 0..6) differs by at
// most 10, and the number of pairs measured.
func pitchContinuity(frames [][]byte) (float64, int) {
	b0 := func(f []byte) int {
		v := 0
		for i := 0; i < 7; i++ {
			v = v<<1 | int(f[i]&1)
		}
		return v
	}
	var pairs, close int
	prev := -1
	for _, f := range frames {
		if len(f) < 7 {
			continue
		}
		p := b0(f)
		if p >= 120 { // silence / tone frames carry no fundamental
			prev = -1
			continue
		}
		if prev >= 0 {
			pairs++
			if d := p - prev; d <= 10 && d >= -10 {
				close++
			}
		}
		prev = p
	}
	if pairs == 0 {
		return 0, 0
	}
	return float64(close) / float64(pairs), pairs
}

func continuityLabel(c float64) string {
	if c >= pitchContinuityVoice {
		return "speech-like"
	}
	return "ciphertext-like"
}

// remodulateDiscriminatorAudio turns a discriminator-audio recording (the
// FM demodulator's output, as DSD-FME consumes) back into unit-amplitude IQ
// by treating each sample as an instantaneous frequency of ±dev Hz at full
// scale and integrating the phase, so the receiver's own discriminator
// recovers the same waveform.
func remodulateDiscriminatorAudio(samples []int16, rate, dev float64) []complex64 {
	iq := make([]complex64, len(samples))
	var phase float64
	for i, s := range samples {
		f := float64(s) / 32768 * dev
		phase += 2 * math.Pi * f / rate
		if phase > math.Pi {
			phase -= 2 * math.Pi
		} else if phase < -math.Pi {
			phase += 2 * math.Pi
		}
		iq[i] = complex64(cmplx.Rect(1, phase))
	}
	return iq
}

// discriminatorAudioSanity characterises a discriminator-audio recording
// before the receiver is blamed for not decoding it. A demodulated 4800-baud
// 4FSK / C4FM signal keeps ~all of its energy below 3 kHz (RRC-shaped at
// 4800 baud it is flat to ~2 kHz and gone by ~2.9 kHz), so an "audio" file
// whose energy sits mostly ABOVE 3 kHz is not a discriminator tap of that
// signal at all — the #1187 reporter's DMR files were 96 kHz recordings of
// white-noise bursts gated at the TDMA slot cadence with a 15.5 kHz codec
// cliff (decoded/garbled audio re-recorded, not the air signal), and every
// receiver decodes nothing from those. The fraction is measured through a
// 2nd-order Butterworth low-pass at 3 kHz over the whole file (the gaps
// are quiet, so the bursts dominate either way).
func discriminatorAudioSanity(samples []int16, rate float64, modulation string) string {
	if len(samples) < 1024 || rate <= 6000 {
		return ""
	}
	// Bilinear-transformed 2nd-order Butterworth low-pass, corner 3 kHz.
	k := math.Tan(math.Pi * 3000 / rate)
	norm := 1 / (1 + math.Sqrt2*k + k*k)
	b0 := k * k * norm
	b1 := 2 * b0
	b2 := b0
	a1 := 2 * (k*k - 1) * norm
	a2 := (1 - math.Sqrt2*k + k*k) * norm
	var x1, x2, y1, y2, eLow, eAll float64
	for _, s := range samples {
		x := float64(s) / 32768
		y := b0*x + b1*x1 + b2*x2 - a1*y1 - a2*y2
		x2, x1 = x1, x
		y2, y1 = y1, y
		eLow += y * y
		eAll += x * x
	}
	if eAll == 0 {
		return "audio is digital silence"
	}
	frac := eLow / eAll
	verdict := "consistent with a discriminator tap"
	if frac < 0.6 {
		verdict = fmt.Sprintf("NOT a demodulable %s discriminator tap — a 4800-baud %s signal keeps ~90%% of its energy below 3 kHz; this looks like decoded/garbled audio or a wideband recording, and no receiver can recover bursts from it (record IQ with `gophertrunk capture`, or the receiver's raw unsquelched discriminator output instead)", modulation, modulation)
	}
	return fmt.Sprintf("energy below 3 kHz: %.0f%% (%s)", 100*frac, verdict)
}
