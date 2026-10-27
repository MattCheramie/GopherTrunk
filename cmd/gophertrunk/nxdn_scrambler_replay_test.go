package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

// TestNXDNScramblerReplay is the NXDN scrambler diagnostic harness (issue
// #1261): it runs a voice-channel IQ capture through the production NXDN
// voice receiver + TrafficChannel + VoiceCipherTracker and prints what the
// daemon's voice chain would see — the SACCH VCALL (source, destination,
// cipher, key ID), SACCH CRC yield, and the AMBE+2 pitch continuity of the
// voice frames as received and, with GT_NXDN_SCR_KEY set, descrambled.
// Speech sits ≳ 0.45; ciphertext / a wrong key ≈ 0.16. Skip-guarded:
//
//	GT_NXDN_SCR_IQ=<cs16 IQ> [GT_NXDN_SCR_RATE=2400000] [GT_NXDN_SCR_KEY=32767]
//	[GT_NXDN_SCR_SYMBOLS=2400] [GT_NXDN_SCR_BRUTE=1]
//	go test ./cmd/gophertrunk -run TestNXDNScramblerReplay -v
//
// GT_NXDN_SCR_BRUTE=1 sweeps all 32767 keys by pitch continuity and ranks
// the configured one — the instrument that settled the register and its
// schedule on the reporter's two known-key captures. The capture is the
// composer's voice_iq_debug .cs16 (its name carries the channel and rate)
// or a `gophertrunk capture` of the channel.
func TestNXDNScramblerReplay(t *testing.T) {
	path := os.Getenv("GT_NXDN_SCR_IQ")
	if path == "" {
		t.Skip("set GT_NXDN_SCR_IQ (cs16 IQ of an NXDN voice channel) [+ GT_NXDN_SCR_RATE, GT_NXDN_SCR_KEY] to run")
	}
	inRate := 2_400_000.0
	if v := os.Getenv("GT_NXDN_SCR_RATE"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatalf("bad GT_NXDN_SCR_RATE: %v", err)
		}
		inRate = f
	}
	symRate := nxdnrx.SymbolRate
	if v := os.Getenv("GT_NXDN_SCR_SYMBOLS"); v == "2400" {
		symRate = nxdnrx.SymbolRateNXDN48
	}
	var key uint16
	if v := os.Getenv("GT_NXDN_SCR_KEY"); v != "" {
		k, err := strconv.ParseUint(v, 0, 16)
		if err != nil || k == 0 || k > nxdn.ScramblerKeyMax {
			t.Fatalf("bad GT_NXDN_SCR_KEY %q: want 1..32767", v)
		}
		key = uint16(k)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(raw)/4)
	for i := range iq {
		re := int16(binary.LittleEndian.Uint16(raw[i*4:]))
		im := int16(binary.LittleEndian.Uint16(raw[i*4+2:]))
		iq[i] = complex(float32(re)/32768, float32(im)/32768)
	}

	// The production voice front end + receiver, as runNXDNVoiceChain.
	ddc := ccdecoder.NewDownconverter(inRate, 48000)
	var frames []nxdn.TrafficFrame
	tc := nxdn.NewTrafficChannel(nil)
	tc.OnFrame = func(f nxdn.TrafficFrame) {
		cp := f
		cp.SACCH = append([]byte(nil), f.SACCH...)
		cp.Voice = make([]nxdn.VCHFrame, len(f.Voice))
		for i, v := range f.Voice {
			cp.Voice[i] = v
			cp.Voice[i].Payload = append([]byte(nil), v.Payload...)
		}
		frames = append(frames, cp)
	}
	rx := nxdnrx.New(nxdnrx.Options{SampleRateHz: ddc.OutRateHz(), SymbolRateHz: symRate, DeviationHz: 1800,
		EnableAFC: true, EnableCarrierTracking: true, EnableChannelFilter: true,
		DibitSink: func(d []uint8, base int) { tc.Process(d, base) }})
	const chunk = 65536
	for off := 0; off < len(iq); off += chunk {
		end := off + chunk
		if end > len(iq) {
			end = len(iq)
		}
		rx.Process(ddc.Process(nil, iq[off:end]))
	}
	voice := 0
	var asIs [][]byte
	for _, f := range frames {
		voice += len(f.Voice)
		for _, v := range f.Voice {
			asIs = append(asIs, v.Payload)
		}
	}
	t.Logf("%s: %.1f s at %.0f S/s, symbols %.0f/s: frames=%d lich_ok=%d voice_frames=%d carrier_offset_hz=%d",
		path, float64(len(iq))/inRate, inRate, symRate, tc.Frames, tc.LICHOK, voice, int(rx.CarrierOffsetHz()))
	if voice == 0 {
		t.Fatal("no voice frames decoded — not an NXDN voice channel at this rate / symbol rate, or no signal")
	}

	run := func(k uint16) (*nxdn.VoiceCipherTracker, [][]byte) {
		tr := &nxdn.VoiceCipherTracker{Hold: k != 0}
		if k != 0 {
			tr.OnVCALL = func(v nxdn.VCALLMessage) { tr.SetKey(k) }
		}
		var out [][]byte
		for _, f := range frames {
			cp := f
			cp.Voice = make([]nxdn.VCHFrame, len(f.Voice))
			for i, v := range f.Voice {
				cp.Voice[i] = v
				cp.Voice[i].Payload = append([]byte(nil), v.Payload...)
			}
			for _, v := range tr.Frame(cp) {
				out = append(out, v.Payload)
			}
		}
		for _, v := range tr.Flush() {
			out = append(out, v.Payload)
		}
		return tr, out
	}
	tr, _ := run(0)
	t.Logf("SACCH: crc_ok=%d crc_fail=%d superframes=%d vcalls=%d", tr.SACCHOK, tr.SACCHFail, tr.Superframes, tr.VCALLs)
	if tr.HaveVCALL {
		v := tr.VCALL
		t.Logf("VCALL: src=%d dst=%d call_type=%d voice_option=%d cipher=%s (type %d) key_id=%d",
			v.Source, v.Destination, v.CallType, v.VoiceCallOption, v.Cipher, uint8(v.Cipher), v.KeyID)
	} else {
		t.Logf("VCALL: none decoded (SACCH too weak, or the call is shorter than one superframe)")
	}
	cont, pairs := pitchContinuity(asIs)
	t.Logf("voice as received: pitch continuity %.2f over %d pairs (%s)", cont, pairs, continuityLabel(cont))
	if key != 0 {
		trk, dec := run(key)
		kc, _ := pitchContinuity(dec)
		t.Logf("descrambled with key %d: continuity %.2f (%s); descrambled=%d ciphertext=%d unaligned=%d pre_vcall=%d",
			key, kc, continuityLabel(kc), trk.Descrambled, trk.Ciphertext, trk.Unaligned, trk.Unknown)
		if trk.HaveVCALL && trk.VCALL.Cipher != nxdn.CipherScrambler {
			t.Logf("NOTE: the VCALL says cipher %s, so the key was not applied", trk.VCALL.Cipher)
		}
	}
	if os.Getenv("GT_NXDN_SCR_BRUTE") == "1" {
		type hit struct {
			key  uint16
			cont float64
		}
		best := make([]hit, 0, 5)
		rank := 0
		var keyCont float64
		for k := 1; k <= nxdn.ScramblerKeyMax; k++ {
			_, dec := run(uint16(k))
			c, _ := pitchContinuity(dec)
			if uint16(k) == key {
				keyCont = c
			}
			for i := 0; i <= len(best); i++ {
				if i == len(best) {
					if len(best) < 5 {
						best = append(best, hit{uint16(k), c})
					}
					break
				}
				if c > best[i].cont {
					best = append(best[:i], append([]hit{{uint16(k), c}}, best[i:]...)...)
					if len(best) > 5 {
						best = best[:5]
					}
					break
				}
			}
		}
		if key != 0 {
			for k := 1; k <= nxdn.ScramblerKeyMax; k++ {
				_, dec := run(uint16(k))
				if c, _ := pitchContinuity(dec); c > keyCont {
					rank++
				}
			}
			t.Logf("brute force: %d keys score above key %d (%.2f)", rank, key, keyCont)
		}
		s := "brute force top 5:"
		for _, h := range best {
			s += fmt.Sprintf(" %d@%.2f", h.key, h.cont)
		}
		t.Log(s)
	}
}
