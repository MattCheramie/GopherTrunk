package siglab

import (
	"bytes"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestCollectPDUs synthesizes a P25 Phase 1 control channel, replays it with
// CollectPDUs set, and asserts the per-signaling-block dissection is populated
// with named opcodes and monotonically-increasing offsets.
func TestCollectPDUs(t *testing.T) {
	synth := SynthOptions{Protocol: trunking.ProtocolP25, Format: FormatF32, Modulation: "c4fm"}
	res, _, err := SynthesizeAndAnalyze(synth, Config{CollectPDUs: true}, nil)
	if err != nil {
		t.Fatalf("SynthesizeAndAnalyze: %v", err)
	}
	if !res.Locked {
		t.Fatalf("synthetic P25 CC did not lock; cannot exercise PDU dissection")
	}
	if len(res.PDUs) == 0 {
		t.Fatal("CollectPDUs set but Result.PDUs is empty")
	}

	var lastOff float64 = -1
	namedOpcodes := 0
	for i, p := range res.PDUs {
		if p.Seq != i {
			t.Errorf("PDU %d: Seq = %d, want %d", i, p.Seq, i)
		}
		if p.Protocol != "p25p1" {
			t.Errorf("PDU %d: protocol = %q, want p25p1", i, p.Protocol)
		}
		if p.OpcodeName == "" {
			t.Errorf("PDU %d: empty opcode name", i)
		}
		if p.OpcodeName != "" && p.RawHex != "" {
			namedOpcodes++
		}
		if p.OffsetSec < lastOff {
			t.Errorf("PDU %d: offset %.4f went backwards from %.4f", i, p.OffsetSec, lastOff)
		}
		lastOff = p.OffsetSec
	}
	if namedOpcodes == 0 {
		t.Error("no PDU carried an opcode name + raw payload")
	}
	t.Logf("dissected %d signaling blocks (e.g. %s, crc_ok=%v)", len(res.PDUs), res.PDUs[0].OpcodeName, res.PDUs[0].CRCOK)

	// csv-pdus export round-trips the header + one row per PDU.
	var buf bytes.Buffer
	if err := WriteResult(&buf, res, FormatCSVPDUs); err != nil {
		t.Fatalf("WriteResult csv-pdus: %v", err)
	}
	lines := strings.Count(strings.TrimRight(buf.String(), "\n"), "\n") + 1
	if !strings.HasPrefix(buf.String(), "seq,offset_sec,protocol,opcode,opcode_name") {
		t.Errorf("csv-pdus header missing; got %q", strings.SplitN(buf.String(), "\n", 2)[0])
	}
	if lines != len(res.PDUs)+1 {
		t.Errorf("csv-pdus rows = %d, want %d (header + %d PDUs)", lines, len(res.PDUs)+1, len(res.PDUs))
	}

	// Without the flag, no PDUs are collected (default-off, zero hot-path cost).
	off, _, err := SynthesizeAndAnalyze(synth, Config{}, nil)
	if err != nil {
		t.Fatalf("control run: %v", err)
	}
	if len(off.PDUs) != 0 {
		t.Errorf("PDUs collected without CollectPDUs: %d", len(off.PDUs))
	}
}
