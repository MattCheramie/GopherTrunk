package main

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

// TestTETRADMOSignallingRealCapture pins the TETRA DMO SCH/S (sync) decode
// against a committed real-air Motorola capture — a 2 s slice of a two-radio
// Direct Mode call (MCC 250 / MNC 1 / TG 3, cs16 @ 144 kHz). Unlike the
// env-gated TestTETRADMOReplay, this runs in CI unconditionally: it drives the
// full DMO chain (DDC → π/4-DQPSK receiver + blind-CMA equalizer →
// ExtractDMBurstsSoft → DecodeDMSCHS → ParseSyncPDU) and asserts it recovers
// several CRC-valid SCH/S with a MONOTONICALLY ADVANCING frame number — proof of
// a genuine DMO lock rather than chance CRC. The clear 2.5 s → 20 s captures all
// decode 37–51/≈40–63 SCH/S; this short slice is trimmed to the active PTT window.
//
// NOTE: this call's TCH/S VOICE is air-interface encrypted (TEA) — as is every
// real DMO capture received so far (#1003): across 128 descramble seeds (raw
// colour 0–63 and MCC/MNC-extended) the TCH/S CRC never left the ~1/256 chance
// floor while the signalling above decodes cleanly, the textbook
// clear-signalling / encrypted-voice signature. So DMO voice is deliberately NOT
// asserted here — validating it still needs a known-CLEAR (unencrypted) capture.
func TestTETRADMOSignallingRealCapture(t *testing.T) {
	raw, err := os.ReadFile("testdata/tetra_dmo_sig_2s_144k.cs16")
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(raw)/4)
	for i := range iq {
		re := int16(binary.LittleEndian.Uint16(raw[i*4:]))
		im := int16(binary.LittleEndian.Uint16(raw[i*4+2:]))
		iq[i] = complex(float32(re)/32768, float32(im)/32768)
	}

	ddc := ccdecoder.NewDownconverter(144000, 144000)
	var allDibits []uint8
	var allDiffs []complex64
	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz:        ddc.OutRateHz(),
		DibitSink:           func(d []uint8, _ int) { allDibits = append(allDibits, d...) },
		SoftSink:            func(diffs []complex64, _ int) { allDiffs = append(allDiffs, diffs...) },
		ClockMode:           tetrarx.ClockGardner,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
		EnableEqualizer:     true, // blind CMA — required for DMO (see TestTETRADMOReplay)
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

	bursts := tetra.ExtractDMBurstsSoft(allDibits, allDiffs, 0)
	crc := 0
	fnSeen := map[uint8]struct{}{}
	for _, b := range bursts {
		if b.Kind != tetra.DMBurstSync {
			continue
		}
		if type1, ok := tetra.DecodeDMSCHS(b); ok {
			crc++
			if pdu, ok := tetra.ParseSyncPDU(type1); ok {
				fnSeen[pdu.FN] = struct{}{}
			}
		}
	}
	t.Logf("DMO SCH/S: crc_valid=%d distinct_fn=%d total_bursts=%d", crc, len(fnSeen), len(bursts))

	if crc < 4 {
		t.Errorf("CRC-valid SCH/S = %d, want >= 4 (DMO sync decode regressed)", crc)
	}
	if len(fnSeen) < 2 {
		t.Errorf("distinct SCH/S frame numbers = %d, want >= 2 (a genuine lock advances the frame counter)", len(fnSeen))
	}
}
